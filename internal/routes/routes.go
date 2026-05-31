package routes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bookbuilder/internal/db"
	"bookbuilder/internal/llm"
	"bookbuilder/internal/prompts"
	"bookbuilder/internal/views"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const jobTimeout = 5 * time.Minute

type Server struct {
	app *pocketbase.PocketBase
	llm llm.Client
}

func Register(app *pocketbase.PocketBase) {
	s := &Server{app: app, llm: llm.FromEnv()}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/", s.handleRoot)
		se.Router.GET("/app/project/{id}/render", s.handleProjectRender)
		se.Router.GET("/app/project/select", s.handleProjectSelect)
		se.Router.GET("/app/ui/fragments/intake-nonfiction", s.handleIntakeNonfiction)
		se.Router.GET("/app/ui/fragments/intake-fiction", s.handleIntakeFiction)
		se.Router.GET("/api/project/{id}/status", s.handleProjectStatus)

		se.Router.POST("/api/projects", s.handleCreateProject)
		se.Router.POST("/api/project/{id}/intake", s.handleProjectIntake)
		se.Router.POST("/api/project/{id}/escape-hatch", s.handleEscapeHatch)
		se.Router.POST("/api/project/{id}/generate/blueprint", s.handleBlueprint)
		se.Router.POST("/api/project/{id}/generate/autopilot", s.handleAutopilot)
		se.Router.POST("/api/project/{id}/chapters/{chapterId}/pipeline/{stage}", s.handlePipelineStage)

		return se.Next()
	})
}

func (s *Server) handleRoot(e *core.RequestEvent) error {
	project, err := s.latestProject()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			project, err = s.createProject("Untitled Book")
			if err != nil {
				return e.InternalServerError("failed to create initial project", err)
			}
		} else {
			return e.InternalServerError("failed to load project", err)
		}
	}
	return e.Redirect(http.StatusFound, "/app/project/"+project.Id+"/render")
}

func (s *Server) handleCreateProject(e *core.RequestEvent) error {
	title := strings.TrimSpace(e.Request.FormValue("title"))
	if title == "" {
		title = "Untitled Book"
	}
	project, err := s.createProject(title)
	if err != nil {
		return e.InternalServerError("failed to create project", err)
	}
	e.Response.Header().Set("HX-Redirect", "/app/project/"+project.Id+"/render")
	return s.renderGrid(e, http.StatusCreated, project, nil)
}

func (s *Server) handleProjectRender(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	chapter, _ := s.firstChapter(project.Id)
	return s.renderPage(e, http.StatusOK, project, chapter)
}

func (s *Server) handleProjectSelect(e *core.RequestEvent) error {
	projectID := e.Request.URL.Query().Get("project_id")
	if projectID == "" {
		projectID = e.Request.FormValue("project_id")
	}
	if projectID == "" {
		return e.BadRequestError("project_id is required", nil)
	}
	project, err := s.app.FindRecordById(db.CollectionProjects, projectID)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	chapter, _ := s.firstChapter(project.Id)
	return s.renderGrid(e, http.StatusOK, project, chapter)
}

func (s *Server) handleProjectIntake(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	if err := e.Request.ParseForm(); err != nil {
		return e.BadRequestError("invalid form body", err)
	}

	targetLength := strings.TrimSpace(e.Request.FormValue("target_length"))
	if !db.ValidTargetLengths[targetLength] {
		return e.BadRequestError("invalid target_length", nil)
	}
	targetChapters, err := strconv.Atoi(e.Request.FormValue("target_chapters"))
	if err != nil || targetChapters < 1 || targetChapters > 25 {
		return e.BadRequestError("target_chapters must be between 1 and 25", nil)
	}
	bookType := strings.TrimSpace(e.Request.FormValue("book_type"))
	if !db.ValidBookTypes[bookType] {
		return e.BadRequestError("invalid book_type", nil)
	}

	project.Set("type", bookType)
	project.Set("target_length", targetLength)
	project.Set("target_chapters", targetChapters)
	project.Set("status", db.ProjectStatusIntake)
	if err := s.app.Save(project); err != nil {
		return e.InternalServerError("failed to update project settings", err)
	}

	keys := []string{
		"core_topic",
		"target_audience",
		"reader_hunger",
		"prohibited_directions",
		"author_intent",
		"selected_tone",
	}
	for _, key := range keys {
		value := strings.TrimSpace(e.Request.FormValue(key))
		if value == "" {
			continue
		}
		if err := s.upsertIntakeResponse(project.Id, key, value); err != nil {
			return e.InternalServerError("failed to save intake response", err)
		}
	}

	e.Response.Header().Set("HX-Redirect", "/app/project/"+project.Id+"/render")
	return s.renderGrid(e, http.StatusOK, project, nil)
}

func (s *Server) handleEscapeHatch(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	field := strings.TrimSpace(e.Request.URL.Query().Get("field"))
	if field == "" {
		return e.BadRequestError("field query parameter is required", nil)
	}
	projectInput, err := s.projectInput(project)
	if err != nil {
		return e.InternalServerError("failed to load project context", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	output, err := s.llm.Generate(ctx, prompts.SystemPrompt(), prompts.BuildEscapeHatchPrompt(projectInput, field))
	if err != nil {
		return e.InternalServerError("failed to generate suggestions", err)
	}
	suggestions := promptLines(output)
	if len(suggestions) == 0 {
		suggestions = []string{"Write one concrete sentence that names the reader's stuck point."}
	}
	html, err := views.RenderEscapeHatch(views.EscapeHatchData{
		ProjectID:   project.Id,
		Field:       field,
		Suggestions: suggestions,
	})
	if err != nil {
		return e.InternalServerError("failed to render escape hatch", err)
	}
	return e.HTML(http.StatusOK, html)
}

func (s *Server) handleBlueprint(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	job, err := s.createJob(project.Id, "", "toc")
	if err != nil {
		return e.InternalServerError("failed to create blueprint job", err)
	}

	go s.runJob(project.Id, "", job.Id, "toc")

	return s.renderProcessing(e, "blueprint", project.Id, "")
}

func (s *Server) handleAutopilot(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	job, err := s.createJob(project.Id, "", "autopilot")
	if err != nil {
		return e.InternalServerError("failed to create autopilot job", err)
	}

	go s.runJob(project.Id, "", job.Id, "autopilot")

	return s.renderProcessing(e, "autopilot", project.Id, "")
}

func (s *Server) handlePipelineStage(e *core.RequestEvent) error {
	projectID := e.Request.PathValue("id")
	chapterID := e.Request.PathValue("chapterId")
	stage := e.Request.PathValue("stage")
	if !validPipelineStage(stage) {
		return e.BadRequestError("invalid pipeline stage", nil)
	}

	chapter, err := s.app.FindRecordById(db.CollectionChapters, chapterID)
	if err != nil || chapter.GetString("project_id") != projectID {
		return e.NotFoundError("target chapter does not belong to project", err)
	}
	job, err := s.createJob(projectID, chapterID, stage)
	if err != nil {
		return e.InternalServerError("failed to create pipeline job", err)
	}

	if notes := strings.TrimSpace(e.Request.FormValue("user_diagnosis_notes")); notes != "" {
		chapter.Set("user_diagnosis_notes", notes)
		_ = s.app.Save(chapter)
	}

	go s.runJob(projectID, chapterID, job.Id, stage)

	return s.renderProcessing(e, stage, projectID, chapterID)
}

func (s *Server) handleProjectStatus(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	running, err := s.runningJobs(project.Id)
	if err != nil {
		return e.InternalServerError("failed to inspect job status", err)
	}
	if len(running) > 0 {
		job := running[0]
		return s.renderStatus(e, http.StatusOK, job.GetString("job_type"), "running", job.GetString("chapter_id"))
	}
	failed, err := s.failedJobs(project.Id)
	if err != nil {
		return e.InternalServerError("failed to inspect failed jobs", err)
	}
	if len(failed) > 0 {
		job := failed[0]
		return s.renderStatus(e, http.StatusOK, job.GetString("job_type"), "failed", job.GetString("chapter_id"))
	}
	e.Response.Header().Set("HX-Refresh", "true")
	return s.renderStatus(e, 286, "complete", "completed", "")
}

func (s *Server) handleIntakeNonfiction(e *core.RequestEvent) error {
	html, err := views.RenderIntakeNonfiction(e.Request.URL.Query().Get("project_id"))
	if err != nil {
		return e.InternalServerError("failed to render intake fragment", err)
	}
	return e.HTML(http.StatusOK, html)
}

func (s *Server) handleIntakeFiction(e *core.RequestEvent) error {
	html, err := views.RenderIntakeFiction(e.Request.URL.Query().Get("project_id"))
	if err != nil {
		return e.InternalServerError("failed to render intake fragment", err)
	}
	return e.HTML(http.StatusOK, html)
}

func (s *Server) createProject(title string) (*core.Record, error) {
	collection, err := s.app.FindCollectionByNameOrId(db.CollectionProjects)
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set("title", title)
	record.Set("type", "nonfiction")
	record.Set("status", db.ProjectStatusIntake)
	record.Set("target_length", db.TargetLengthPracticalEbook)
	record.Set("target_chapters", 10)
	record.Set("ui_state", db.DefaultUIState())
	if err := s.app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Server) latestProject() (*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(db.CollectionProjects, "", "-updated", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}
	return records[0], nil
}

func (s *Server) projectFromPath(e *core.RequestEvent) (*core.Record, error) {
	return s.app.FindRecordById(db.CollectionProjects, e.Request.PathValue("id"))
}

func (s *Server) firstChapter(projectID string) (*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(
		db.CollectionChapters,
		"project_id = {:project_id}",
		"sort_order",
		1,
		0,
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}
	return records[0], nil
}

func (s *Server) renderPage(e *core.RequestEvent, status int, project *core.Record, chapter *core.Record) error {
	data, err := s.viewData(project, chapter)
	if err != nil {
		return e.InternalServerError("failed to prepare view data", err)
	}
	html, err := views.RenderPage(data)
	if err != nil {
		return e.InternalServerError("failed to render page", err)
	}
	return e.HTML(status, html)
}

func (s *Server) renderGrid(e *core.RequestEvent, status int, project *core.Record, chapter *core.Record) error {
	data, err := s.viewData(project, chapter)
	if err != nil {
		return e.InternalServerError("failed to prepare view data", err)
	}
	html, err := views.RenderGrid(data)
	if err != nil {
		return e.InternalServerError("failed to render project grid", err)
	}
	return e.HTML(status, html)
}

func (s *Server) renderProcessing(e *core.RequestEvent, stage string, projectID string, chapterID string) error {
	html, err := views.RenderProcessing(views.ProcessingData{
		Stage:     stage,
		ProjectID: projectID,
		ChapterID: chapterID,
	})
	if err != nil {
		return e.InternalServerError("failed to render processing state", err)
	}
	return e.HTML(http.StatusAccepted, html)
}

func (s *Server) renderStatus(e *core.RequestEvent, statusCode int, jobType string, status string, chapterID string) error {
	html, err := views.RenderStatus(views.StatusData{
		JobType:   jobType,
		Status:    status,
		ChapterID: chapterID,
	})
	if err != nil {
		return e.InternalServerError("failed to render status", err)
	}
	return e.HTML(statusCode, html)
}

func (s *Server) viewData(project *core.Record, chapter *core.Record) (views.PageData, error) {
	projects, err := s.allProjectViews()
	if err != nil {
		return views.PageData{}, err
	}
	data := views.PageData{
		Project:         projectView(project),
		AllProjects:     projects,
		ActiveProjectID: project.Id,
	}
	data.IntakeSaved, err = s.hasIntake(project.Id)
	if err != nil {
		return views.PageData{}, err
	}
	if chapter != nil {
		view := chapterView(chapter)
		data.Chapter = &view
	}
	return data, nil
}

func (s *Server) hasIntake(projectID string) (bool, error) {
	records, err := s.app.FindRecordsByFilter(
		db.CollectionIntakeResponses,
		"project_id = {:project_id}",
		"created",
		1,
		0,
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return false, err
	}
	return len(records) > 0, nil
}

func (s *Server) allProjectViews() ([]views.Project, error) {
	records, err := s.app.FindRecordsByFilter(db.CollectionProjects, "", "-updated", 0, 0)
	if err != nil {
		return nil, err
	}
	result := make([]views.Project, 0, len(records))
	for _, record := range records {
		result = append(result, projectView(record))
	}
	return result, nil
}

func projectView(record *core.Record) views.Project {
	return views.Project{
		ID:             record.Id,
		Title:          record.GetString("title"),
		BookType:       record.GetString("type"),
		Status:         record.GetString("status"),
		TargetLength:   record.GetString("target_length"),
		TargetChapters: record.GetInt("target_chapters"),
	}
}

func chapterView(record *core.Record) views.Chapter {
	return views.Chapter{
		ID:                   record.Id,
		SortOrder:            record.GetInt("sort_order"),
		Title:                record.GetString("title"),
		Status:               record.GetString("status"),
		Purpose:              record.GetString("purpose"),
		RawDraft:             record.GetString("raw_draft"),
		EditorialDiagnosis:   record.GetString("editorial_diagnosis"),
		TargetedRewrite:      record.GetString("targeted_rewrite"),
		DraftContent:         record.GetString("draft_content"),
		PreviousDraftContent: record.GetString("previous_draft_content"),
	}
}

func (s *Server) upsertIntakeResponse(projectID, key, value string) error {
	record, err := s.app.FindFirstRecordByFilter(
		db.CollectionIntakeResponses,
		"project_id = {:project_id} && key = {:key}",
		dbx.Params{"project_id": projectID, "key": key},
	)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		collection, err := s.app.FindCollectionByNameOrId(db.CollectionIntakeResponses)
		if err != nil {
			return err
		}
		record = core.NewRecord(collection)
		record.Set("project_id", projectID)
		record.Set("key", key)
	}
	record.Set("value", value)
	return s.app.Save(record)
}

func (s *Server) createJob(projectID, chapterID, jobType string) (*core.Record, error) {
	collection, err := s.app.FindCollectionByNameOrId(db.CollectionJobs)
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set("project_id", projectID)
	if chapterID != "" {
		record.Set("chapter_id", chapterID)
	}
	record.Set("job_type", jobType)
	record.Set("status", "running")
	record.Set("started_at", time.Now().UTC().Format(time.RFC3339))
	if err := s.app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Server) runJob(projectID, chapterID, jobID, jobType string) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	err := s.executeJob(ctx, projectID, chapterID, jobType)
	job, findErr := s.app.FindRecordById(db.CollectionJobs, jobID)
	if findErr != nil {
		return
	}
	job.Set("completed_at", time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		job.Set("status", "failed")
		job.Set("error_msg", err.Error())
		_ = s.markProjectFailed(projectID)
	} else {
		job.Set("status", "completed")
	}
	_ = s.app.Save(job)
}

func (s *Server) executeJob(ctx context.Context, projectID, chapterID, jobType string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	switch jobType {
	case "brief":
		_, err := s.ensureBrief(ctx, projectID)
		return err
	case "toc":
		if err := s.updateProjectStatus(projectID, db.ProjectStatusProcessing); err != nil {
			return err
		}
		if _, err := s.ensureBrief(ctx, projectID); err != nil {
			return err
		}
		if err := s.updateProjectStatus(projectID, db.ProjectStatusBrief); err != nil {
			return err
		}
		if err := s.ensureTOC(ctx, projectID); err != nil {
			return err
		}
		return s.updateProjectStatus(projectID, db.ProjectStatusTOC)
	case "autopilot":
		return s.executeAutopilot(ctx, projectID)
	}
	return s.executePipelineStage(ctx, projectID, chapterID, jobType)
}

func (s *Server) executeAutopilot(ctx context.Context, projectID string) error {
	if err := s.updateProjectStatus(projectID, db.ProjectStatusProcessing); err != nil {
		return err
	}
	if _, err := s.ensureBrief(ctx, projectID); err != nil {
		return err
	}
	if err := s.updateProjectStatus(projectID, db.ProjectStatusBrief); err != nil {
		return err
	}
	if err := s.ensureTOC(ctx, projectID); err != nil {
		return err
	}
	if err := s.updateProjectStatus(projectID, db.ProjectStatusTOC); err != nil {
		return err
	}
	chapters, err := s.app.FindRecordsByFilter(
		db.CollectionChapters,
		"project_id = {:project_id}",
		"sort_order",
		0,
		0,
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return err
	}
	if len(chapters) == 0 {
		return errors.New("autopilot requires at least one pending or approved chapter")
	}
	if err := s.updateProjectStatus(projectID, db.ProjectStatusDrafting); err != nil {
		return err
	}
	for _, chapter := range chapters {
		for _, stage := range []string{"draft", "diagnose", "rewrite", "polish"} {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			if err := s.executePipelineStage(ctx, projectID, chapter.Id, stage); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) executePipelineStage(ctx context.Context, projectID, chapterID, stage string) error {
	if chapterID == "" {
		return errors.New("chapter_id is required")
	}
	chapter, err := s.app.FindRecordById(db.CollectionChapters, chapterID)
	if err != nil {
		return err
	}
	if chapter.GetString("project_id") != projectID {
		return errors.New("chapter project mismatch")
	}
	project, err := s.projectInputByID(projectID)
	if err != nil {
		return err
	}
	brief, err := s.briefInput(projectID)
	if err != nil {
		return err
	}
	chapterInput := prompts.ChapterInput{
		Project:      project,
		Brief:        brief,
		SortOrder:    chapter.GetInt("sort_order"),
		Title:        chapter.GetString("title"),
		Purpose:      chapter.GetString("purpose"),
		StateStart:   chapter.GetString("state_start"),
		StateEnd:     chapter.GetString("state_end"),
		RawDraft:     chapter.GetString("raw_draft"),
		Diagnosis:    chapter.GetString("editorial_diagnosis"),
		Rewrite:      chapter.GetString("targeted_rewrite"),
		DraftNotes:   chapter.GetString("user_draft_notes"),
		DiagnoseNote: chapter.GetString("user_diagnosis_notes"),
		RewriteNotes: chapter.GetString("user_rewrite_notes"),
	}

	switch stage {
	case "draft":
		output, err := s.llm.Generate(ctx, prompts.SystemPrompt(), prompts.BuildDraftPrompt(chapterInput))
		if err != nil {
			return err
		}
		chapter.Set("status", db.ChapterStatusDraftingStage1)
		chapter.Set("raw_draft", output)
	case "diagnose":
		if strings.TrimSpace(chapterInput.RawDraft) == "" {
			return errors.New("draft stage must complete before diagnosis")
		}
		output, err := s.llm.Generate(ctx, prompts.SystemPrompt(), prompts.BuildDiagnosisPrompt(chapterInput))
		if err != nil {
			return err
		}
		chapter.Set("status", db.ChapterStatusDraftingStage2)
		chapter.Set("editorial_diagnosis", output)
	case "rewrite":
		if strings.TrimSpace(chapterInput.RawDraft) == "" {
			return errors.New("draft stage must complete before rewrite")
		}
		output, err := s.llm.Generate(ctx, prompts.SystemPrompt(), prompts.BuildRewritePrompt(chapterInput))
		if err != nil {
			return err
		}
		chapter.Set("status", db.ChapterStatusDraftingStage3)
		chapter.Set("targeted_rewrite", output)
	case "polish":
		if strings.TrimSpace(chapterInput.Rewrite) == "" {
			chapterInput.Rewrite = chapterInput.RawDraft
		}
		if strings.TrimSpace(chapterInput.Rewrite) == "" {
			return errors.New("draft or rewrite stage must complete before polish")
		}
		output, err := s.llm.Generate(ctx, prompts.SystemPrompt(), prompts.BuildPolishPrompt(chapterInput))
		if err != nil {
			return err
		}
		chapter.Set("previous_draft_content", chapter.GetString("draft_content"))
		chapter.Set("draft_content", output)
		chapter.Set("status", db.ChapterStatusCompleted)
	default:
		return fmt.Errorf("unsupported stage %q", stage)
	}
	if err := s.updateProjectStatus(projectID, db.ProjectStatusDrafting); err != nil {
		return err
	}
	return s.app.Save(chapter)
}

func (s *Server) ensureBrief(ctx context.Context, projectID string) (prompts.Brief, error) {
	existing, err := s.briefInput(projectID)
	if err == nil && existing.Title != "" {
		return existing, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return prompts.Brief{}, err
	}
	project, err := s.projectInputByID(projectID)
	if err != nil {
		return prompts.Brief{}, err
	}
	output, err := s.llm.Generate(ctx, prompts.SystemPrompt(), prompts.BuildBriefPrompt(project))
	if err != nil {
		return prompts.Brief{}, err
	}
	brief, err := prompts.ParseBrief(output)
	if err != nil {
		return prompts.Brief{}, err
	}
	if err := s.saveBrief(projectID, brief); err != nil {
		return prompts.Brief{}, err
	}
	projectRecord, err := s.app.FindRecordById(db.CollectionProjects, projectID)
	if err == nil && strings.TrimSpace(projectRecord.GetString("title")) == "Untitled Book" {
		projectRecord.Set("title", brief.Title)
		_ = s.app.Save(projectRecord)
	}
	return brief, nil
}

func (s *Server) ensureTOC(ctx context.Context, projectID string) error {
	existing, err := s.app.FindRecordsByFilter(
		db.CollectionChapters,
		"project_id = {:project_id}",
		"sort_order",
		1,
		0,
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}
	project, err := s.projectInputByID(projectID)
	if err != nil {
		return err
	}
	brief, err := s.ensureBrief(ctx, projectID)
	if err != nil {
		return err
	}
	output, err := s.llm.Generate(ctx, prompts.SystemPrompt(), prompts.BuildTOCPrompt(project, brief))
	if err != nil {
		return err
	}
	chapters, err := prompts.ParseTOC(output, project.TargetChapters)
	if err != nil {
		return err
	}
	return s.saveChapters(projectID, chapters)
}

func (s *Server) saveBrief(projectID string, brief prompts.Brief) error {
	record, err := s.app.FindFirstRecordByFilter(
		db.CollectionBookBriefs,
		"project_id = {:project_id}",
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		collection, err := s.app.FindCollectionByNameOrId(db.CollectionBookBriefs)
		if err != nil {
			return err
		}
		record = core.NewRecord(collection)
		record.Set("project_id", projectID)
	}
	record.Set("title", brief.Title)
	record.Set("subtitle", brief.Subtitle)
	record.Set("promise", brief.Promise)
	record.Set("voice_tone", brief.VoiceTone)
	record.Set("what_it_is", brief.WhatItIs)
	record.Set("what_it_is_not", brief.WhatItIsNot)
	record.Set("ai_suggestions", brief.AISuggestions)
	return s.app.Save(record)
}

func (s *Server) saveChapters(projectID string, plans []prompts.ChapterPlan) error {
	collection, err := s.app.FindCollectionByNameOrId(db.CollectionChapters)
	if err != nil {
		return err
	}
	for _, plan := range plans {
		record := core.NewRecord(collection)
		record.Set("project_id", projectID)
		record.Set("sort_order", plan.SortOrder)
		record.Set("title", plan.Title)
		record.Set("status", db.ChapterStatusCardApproved)
		record.Set("purpose", plan.Purpose)
		record.Set("state_start", plan.StateStart)
		record.Set("state_end", plan.StateEnd)
		if err := s.app.Save(record); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) projectInput(record *core.Record) (prompts.ProjectInput, error) {
	intake, err := s.intakeMap(record.Id)
	if err != nil {
		return prompts.ProjectInput{}, err
	}
	return prompts.ProjectInput{
		ID:             record.Id,
		Title:          record.GetString("title"),
		BookType:       record.GetString("type"),
		TargetLength:   record.GetString("target_length"),
		TargetChapters: record.GetInt("target_chapters"),
		Intake:         intake,
	}, nil
}

func (s *Server) projectInputByID(projectID string) (prompts.ProjectInput, error) {
	project, err := s.app.FindRecordById(db.CollectionProjects, projectID)
	if err != nil {
		return prompts.ProjectInput{}, err
	}
	return s.projectInput(project)
}

func (s *Server) intakeMap(projectID string) (map[string]string, error) {
	records, err := s.app.FindRecordsByFilter(
		db.CollectionIntakeResponses,
		"project_id = {:project_id}",
		"key",
		0,
		0,
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return nil, err
	}
	values := make(map[string]string, len(records))
	for _, record := range records {
		values[record.GetString("key")] = record.GetString("value")
	}
	return values, nil
}

func (s *Server) briefInput(projectID string) (prompts.Brief, error) {
	record, err := s.app.FindFirstRecordByFilter(
		db.CollectionBookBriefs,
		"project_id = {:project_id}",
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return prompts.Brief{}, err
	}
	return prompts.Brief{
		Title:         record.GetString("title"),
		Subtitle:      record.GetString("subtitle"),
		Promise:       record.GetString("promise"),
		VoiceTone:     record.GetString("voice_tone"),
		WhatItIs:      record.GetString("what_it_is"),
		WhatItIsNot:   record.GetString("what_it_is_not"),
		AISuggestions: record.GetString("ai_suggestions"),
	}, nil
}

func (s *Server) updateProjectStatus(projectID, status string) error {
	project, err := s.app.FindRecordById(db.CollectionProjects, projectID)
	if err != nil {
		return err
	}
	project.Set("status", status)
	return s.app.Save(project)
}

func (s *Server) markProjectFailed(projectID string) error {
	project, err := s.app.FindRecordById(db.CollectionProjects, projectID)
	if err != nil {
		return err
	}
	project.Set("status", db.ProjectStatusFailedJob)
	return s.app.Save(project)
}

func (s *Server) runningJobs(projectID string) ([]*core.Record, error) {
	return s.app.FindRecordsByFilter(
		db.CollectionJobs,
		"project_id = {:project_id} && status = 'running'",
		"created",
		0,
		0,
		dbx.Params{"project_id": projectID},
	)
}

func (s *Server) failedJobs(projectID string) ([]*core.Record, error) {
	return s.app.FindRecordsByFilter(
		db.CollectionJobs,
		"project_id = {:project_id} && status = 'failed'",
		"-updated",
		1,
		0,
		dbx.Params{"project_id": projectID},
	)
}

func validPipelineStage(stage string) bool {
	switch stage {
	case "draft", "diagnose", "rewrite", "polish":
		return true
	default:
		return false
	}
}

func promptLines(output string) []string {
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	result := make([]string, 0, 3)
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimLeft(line, "-*0123456789. "))
		if line != "" {
			result = append(result, line)
		}
		if len(result) == 3 {
			break
		}
	}
	return result
}
