package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bookbuilder/internal/db"
	"bookbuilder/internal/exporter"
	"bookbuilder/internal/llm"
	"bookbuilder/internal/prompts"
	"bookbuilder/internal/views"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const jobTimeout = 5 * time.Minute
const autopilotJobTimeout = 90 * time.Minute

type Server struct {
	app *pocketbase.PocketBase
	llm llm.Client
}

func Register(app *pocketbase.PocketBase) {
	s := &Server{app: app, llm: llm.FromEnv()}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/", s.handleRoot)
		se.Router.GET("/app/project/{id}/render", s.handleProjectRender)
		se.Router.GET("/app/project/{id}/chapters/{chapterId}/workspace", s.handleChapterWorkspace)
		se.Router.GET("/app/project/select", s.handleProjectSelect)
		se.Router.GET("/app/ui/fragments/intake-nonfiction", s.handleIntakeNonfiction)
		se.Router.GET("/app/ui/fragments/intake-fiction", s.handleIntakeFiction)
		se.Router.GET("/api/project/{id}/status", s.handleProjectStatus)
		se.Router.GET("/api/project/{id}/chapters/{chapterId}/status", s.handleChapterStatus)
		se.Router.GET("/exports/reports/{filename}", s.handleExportReportDownload)
		se.Router.GET("/exports/{filename}", s.handleExportDownload)

		se.Router.POST("/api/projects", s.handleCreateProject)
		se.Router.POST("/api/project/{id}/intake", s.handleProjectIntake)
		se.Router.POST("/api/project/{id}/metadata", s.handleSaveProjectMetadata)
		se.Router.POST("/api/project/{id}/escape-hatch", s.handleEscapeHatch)
		se.Router.POST("/api/project/{id}/generate/brief", s.handleBrief)
		se.Router.POST("/api/project/{id}/brief", s.handleSaveBrief)
		se.Router.POST("/api/project/{id}/generate/toc", s.handleTOC)
		se.Router.POST("/api/project/{id}/generate/blueprint", s.handleBlueprint)
		se.Router.POST("/api/project/{id}/generate/autopilot", s.handleAutopilot)
		se.Router.POST("/api/project/{id}/exports/book", s.handleExportBook)
		se.Router.POST("/api/project/{id}/chapters/{chapterId}/metadata", s.handleSaveChapterMetadata)
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

func (s *Server) handleChapterWorkspace(e *core.RequestEvent) error {
	project, chapter, err := s.projectChapterFromPath(e)
	if err != nil {
		return e.NotFoundError("chapter not found", err)
	}
	return s.renderWorkspace(e, http.StatusOK, project, chapter, "drafting")
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
	title := strings.TrimSpace(e.Request.FormValue("title"))
	if title == "" {
		return e.BadRequestError("book title is required", nil)
	}

	project.Set("title", title)
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
		"book_form",
		"narrative_pov",
		"structure_model",
	}
	for _, key := range keys {
		value := strings.TrimSpace(e.Request.FormValue(key))
		if value == "" {
			if err := s.deleteIntakeResponse(project.Id, key); err != nil {
				return e.InternalServerError("failed to clear intake response", err)
			}
			continue
		}
		if err := s.upsertIntakeResponse(project.Id, key, value); err != nil {
			return e.InternalServerError("failed to save intake response", err)
		}
	}

	return s.renderGridWithNotice(e, http.StatusOK, project, nil, "Book setup saved.")
}

func (s *Server) applyProjectSetupFromRequest(e *core.RequestEvent, project *core.Record) error {
	if err := e.Request.ParseForm(); err != nil {
		return err
	}
	if strings.TrimSpace(e.Request.FormValue("target_length")) == "" &&
		strings.TrimSpace(e.Request.FormValue("target_chapters")) == "" {
		return nil
	}

	targetLength := strings.TrimSpace(e.Request.FormValue("target_length"))
	if !db.ValidTargetLengths[targetLength] {
		return errors.New("invalid target_length")
	}
	targetChapters, err := strconv.Atoi(e.Request.FormValue("target_chapters"))
	if err != nil || targetChapters < 1 || targetChapters > 25 {
		return errors.New("target_chapters must be between 1 and 25")
	}
	bookType := strings.TrimSpace(e.Request.FormValue("book_type"))
	if !db.ValidBookTypes[bookType] {
		return errors.New("invalid book_type")
	}
	title := strings.TrimSpace(e.Request.FormValue("title"))
	if title == "" {
		return errors.New("book title is required")
	}

	project.Set("title", title)
	project.Set("type", bookType)
	project.Set("target_length", targetLength)
	project.Set("target_chapters", targetChapters)
	if project.GetString("status") == "" {
		project.Set("status", db.ProjectStatusIntake)
	}
	if err := s.app.Save(project); err != nil {
		return err
	}

	for _, key := range []string{
		"core_topic",
		"target_audience",
		"reader_hunger",
		"prohibited_directions",
		"author_intent",
		"selected_tone",
		"book_form",
		"narrative_pov",
		"structure_model",
	} {
		value := strings.TrimSpace(e.Request.FormValue(key))
		if value == "" {
			if err := s.deleteIntakeResponse(project.Id, key); err != nil {
				return err
			}
			continue
		}
		if err := s.upsertIntakeResponse(project.Id, key, value); err != nil {
			return err
		}
	}
	return nil
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

func (s *Server) handleBrief(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	if err := s.applyProjectSetupFromRequest(e, project); err != nil {
		return e.BadRequestError("failed to save book setup before generating the brief", err)
	}
	if err := s.clearEmptyChapterShells(project.Id); err != nil {
		return e.InternalServerError("failed to clear stale empty outline shells before regenerating the brief", err)
	}
	job, err := s.createJob(project.Id, "", "brief")
	if err != nil {
		return e.InternalServerError("failed to create brief job", err)
	}

	go s.runJob(project.Id, "", job.Id, "brief")

	return s.renderProcessing(e, "book brief", project.Id, "")
}

func (s *Server) handleSaveBrief(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	if err := e.Request.ParseForm(); err != nil {
		return e.BadRequestError("invalid form body", err)
	}
	brief := prompts.Brief{
		Title:         strings.TrimSpace(e.Request.FormValue("title")),
		Subtitle:      strings.TrimSpace(e.Request.FormValue("subtitle")),
		Promise:       strings.TrimSpace(e.Request.FormValue("promise")),
		VoiceTone:     strings.TrimSpace(e.Request.FormValue("voice_tone")),
		WhatItIs:      strings.TrimSpace(e.Request.FormValue("what_it_is")),
		WhatItIsNot:   strings.TrimSpace(e.Request.FormValue("what_it_is_not")),
		AISuggestions: strings.TrimSpace(e.Request.FormValue("ai_suggestions")),
	}
	if brief.Title == "" {
		return e.BadRequestError("brief title is required", nil)
	}
	if err := s.saveBrief(project.Id, brief); err != nil {
		return e.InternalServerError("failed to save book brief", err)
	}
	project.Set("title", brief.Title)
	project.Set("status", db.ProjectStatusBrief)
	if err := s.app.Save(project); err != nil {
		return e.InternalServerError("failed to update project title", err)
	}
	return s.renderWorkspaceWithNotice(e, http.StatusOK, project, nil, "brief", "Book Brief saved.")
}

func (s *Server) handleSaveProjectMetadata(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	if err := e.Request.ParseForm(); err != nil {
		return e.BadRequestError("invalid form body", err)
	}
	for _, field := range []string{
		"publishing_metadata_json",
		"book_architecture_json",
		"global_style_contract_json",
	} {
		value, err := cleanOptionalJSON(e.Request.FormValue(field))
		if err != nil {
			return e.BadRequestError(field+" must be blank or valid JSON", err)
		}
		project.Set(field, value)
	}
	project.Set("author_name", strings.TrimSpace(e.Request.FormValue("author_name")))
	if err := s.app.Save(project); err != nil {
		return e.InternalServerError("failed to save book metadata", err)
	}
	chapter, _ := s.firstChapter(project.Id)
	return s.renderWorkspaceWithNotice(e, http.StatusOK, project, chapter, "metadata", "Book metadata saved.")
}

func (s *Server) handleSaveChapterMetadata(e *core.RequestEvent) error {
	project, chapter, err := s.projectChapterFromPath(e)
	if err != nil {
		return e.NotFoundError("chapter not found", err)
	}
	if err := e.Request.ParseForm(); err != nil {
		return e.BadRequestError("invalid form body", err)
	}
	title := strings.TrimSpace(e.Request.FormValue("title"))
	if title == "" {
		return e.BadRequestError("chapter title is required", nil)
	}
	for _, field := range []string{
		"chapter_metadata_json",
		"arc_metadata_json",
		"concept_jurisdiction_json",
		"generation_directives_json",
		"media_prompts_json",
	} {
		value, err := cleanOptionalJSON(e.Request.FormValue(field))
		if err != nil {
			return e.BadRequestError(field+" must be blank or valid JSON", err)
		}
		chapter.Set(field, value)
	}
	chapter.Set("title", title)
	chapter.Set("subtitle", strings.TrimSpace(e.Request.FormValue("subtitle")))
	chapter.Set("front_matter_label", strings.TrimSpace(e.Request.FormValue("front_matter_label")))
	chapter.Set("front_matter_blurb", strings.TrimSpace(e.Request.FormValue("front_matter_blurb")))
	if err := s.app.Save(chapter); err != nil {
		return e.InternalServerError("failed to save chapter metadata", err)
	}
	return s.renderWorkspaceWithNotice(e, http.StatusOK, project, chapter, "metadata", "Chapter metadata saved.")
}

func (s *Server) handleTOC(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	if err := s.applyProjectSetupFromRequest(e, project); err != nil {
		return e.BadRequestError("failed to save book setup before generating the outline", err)
	}
	if _, err := s.briefInput(project.Id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return e.BadRequestError("generate and review the Book Brief before generating the outline", err)
		}
		return e.InternalServerError("failed to load book brief", err)
	}
	job, err := s.createJob(project.Id, "", "toc")
	if err != nil {
		return e.InternalServerError("failed to create outline job", err)
	}

	go s.runJob(project.Id, "", job.Id, "toc")

	return s.renderProcessing(e, "outline and chapter shells", project.Id, "")
}

func (s *Server) handleBlueprint(e *core.RequestEvent) error {
	return s.handleTOC(e)
}

func (s *Server) handleAutopilot(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	if err := s.applyProjectSetupFromRequest(e, project); err != nil {
		return e.BadRequestError("failed to save book setup before running autopilot", err)
	}
	job, err := s.createJob(project.Id, "", "autopilot")
	if err != nil {
		return e.InternalServerError("failed to create autopilot job", err)
	}

	go s.runJob(project.Id, "", job.Id, "autopilot")

	return s.renderProcessing(e, "autopilot", project.Id, "")
}

func (s *Server) handleExportBook(e *core.RequestEvent) error {
	project, err := s.projectFromPath(e)
	if err != nil {
		return e.NotFoundError("project not found", err)
	}
	running, err := s.runningJobs(project.Id)
	if err != nil {
		return e.InternalServerError("failed to inspect active jobs", err)
	}
	if len(running) > 0 {
		job := running[0]
		stage := job.GetString("job_type")
		if progress := strings.TrimSpace(job.GetString("error_msg")); progress != "" {
			stage = progress
		}
		html, renderErr := views.RenderExportWait(views.ExportWaitData{
			Message: "Export is waiting because the book is still running: " + stage,
		})
		if renderErr != nil {
			return e.InternalServerError("failed to render export wait state", renderErr)
		}
		return e.HTML(http.StatusOK, html)
	}
	book, err := s.exportBookData(project)
	if err != nil {
		return e.InternalServerError("failed to prepare export data", err)
	}
	if e.Request.URL.Query().Get("allow_incomplete") != "1" {
		blockers := incompleteExportWarnings(book)
		if len(blockers) > 0 {
			html, renderErr := views.RenderExportBlocked(views.ExportBlockedData{
				Message:   "Book is not export-ready yet.",
				Warnings:  blockers,
				ProjectID: project.Id,
			})
			if renderErr != nil {
				return e.InternalServerError("failed to render export preflight", renderErr)
			}
			return e.HTML(http.StatusOK, html)
		}
	}
	result, err := exporter.Builder{Dir: "exports"}.ExportBook(book)
	if err != nil {
		return e.InternalServerError("failed to export book", err)
	}
	html, err := views.RenderExportResult(views.ExportResultData{
		MarkdownURL:    result.MarkdownURL,
		EPUBURL:        result.EPUBURL,
		HTMLURL:        result.HTMLURL,
		LintReportURL:  result.LintReportURL,
		StyleReportURL: result.StyleReportURL,
		Warnings:       result.Warnings,
	})
	if err != nil {
		return e.InternalServerError("failed to render export result", err)
	}
	return e.HTML(http.StatusOK, html)
}

func (s *Server) handleExportDownload(e *core.RequestEvent) error {
	filename := filepath.Base(e.Request.PathValue("filename"))
	if filename == "." || filename == string(filepath.Separator) || strings.Contains(filename, "..") {
		return e.BadRequestError("invalid export filename", nil)
	}
	path := filepath.Join("exports", filename)
	if _, err := os.Stat(path); err != nil {
		return e.NotFoundError("export file not found", err)
	}
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".html":
		e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".md":
		e.Response.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	case ".epub":
		e.Response.Header().Set("Content-Type", "application/epub+zip")
	}
	return e.FileFS(os.DirFS("exports"), filename)
}

func (s *Server) handleExportReportDownload(e *core.RequestEvent) error {
	filename := filepath.Base(e.Request.PathValue("filename"))
	if filename == "." || filename == string(filepath.Separator) || strings.Contains(filename, "..") {
		return e.BadRequestError("invalid export report filename", nil)
	}
	path := filepath.Join("exports", "reports", filename)
	if _, err := os.Stat(path); err != nil {
		return e.NotFoundError("export report file not found", err)
	}
	e.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	return e.FileFS(os.DirFS(filepath.Join("exports", "reports")), filename)
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
		stage := job.GetString("job_type")
		if progress := strings.TrimSpace(job.GetString("error_msg")); progress != "" {
			stage = progress
		}
		return s.renderProcessingStatus(e, http.StatusOK, stage, project.Id, job.GetString("chapter_id"))
	}
	if project.GetString("status") == db.ProjectStatusFailedJob {
		failed, err := s.failedJobs(project.Id)
		if err != nil {
			return e.InternalServerError("failed to inspect failed jobs", err)
		}
		if len(failed) > 0 {
			job := failed[0]
			return s.renderStatus(e, http.StatusOK, job.GetString("job_type"), "failed", job.GetString("chapter_id"), job.GetString("error_msg"))
		}
	}
	latest, err := s.latestProjectJob(project.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return e.InternalServerError("failed to inspect latest project job", err)
	}
	if latest != nil && latest.GetString("status") == "completed" {
		switch latest.GetString("job_type") {
		case "brief":
			return s.renderWorkspace(e, http.StatusOK, project, nil, "brief")
		case "toc":
			return s.renderWorkspace(e, http.StatusOK, project, nil, "outline")
		}
	}
	if chapters, err := s.chapterViews(project.Id); err == nil && len(chapters) > 0 {
		if project.GetString("status") == db.ProjectStatusProcessing {
			_ = s.updateProjectStatus(project.Id, db.ProjectStatusTOC)
		}
		e.Response.Header().Set("HX-Refresh", "true")
		return s.renderStatus(e, 286, "complete", "completed", "", "")
	}
	if project.GetString("status") != db.ProjectStatusProcessing && project.GetString("status") != db.ProjectStatusFailedJob {
		e.Response.Header().Set("HX-Refresh", "true")
		return s.renderStatus(e, 286, "complete", "completed", "", "")
	}
	e.Response.Header().Set("HX-Refresh", "true")
	return s.renderStatus(e, 286, "complete", "completed", "", "")
}

func (s *Server) handleChapterStatus(e *core.RequestEvent) error {
	project, chapter, err := s.projectChapterFromPath(e)
	if err != nil {
		return e.NotFoundError("chapter not found", err)
	}
	running, err := s.runningJobsForChapter(project.Id, chapter.Id)
	if err != nil {
		return e.InternalServerError("failed to inspect chapter job status", err)
	}
	if len(running) > 0 {
		job := running[0]
		return s.renderProcessingStatus(e, http.StatusOK, job.GetString("job_type"), project.Id, chapter.Id)
	}
	latest, err := s.latestJobForChapter(project.Id, chapter.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return e.InternalServerError("failed to inspect latest chapter job", err)
	}
	if latest != nil && latest.GetString("status") == "failed" {
		return s.renderStatus(e, http.StatusOK, latest.GetString("job_type"), "failed", chapter.Id, latest.GetString("error_msg"))
	}
	return s.renderChapterCockpit(e, http.StatusOK, project, chapter)
}

func (s *Server) handleIntakeNonfiction(e *core.RequestEvent) error {
	project, err := s.intakeProjectView(e.Request.URL.Query().Get("project_id"))
	if err != nil {
		return e.InternalServerError("failed to load intake values", err)
	}
	html, err := views.RenderIntakeNonfiction(project)
	if err != nil {
		return e.InternalServerError("failed to render intake fragment", err)
	}
	return e.HTML(http.StatusOK, html)
}

func (s *Server) handleIntakeFiction(e *core.RequestEvent) error {
	project, err := s.intakeProjectView(e.Request.URL.Query().Get("project_id"))
	if err != nil {
		return e.InternalServerError("failed to load intake values", err)
	}
	html, err := views.RenderIntakeFiction(project)
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

func (s *Server) projectChapterFromPath(e *core.RequestEvent) (*core.Record, *core.Record, error) {
	project, err := s.projectFromPath(e)
	if err != nil {
		return nil, nil, err
	}
	chapter, err := s.app.FindRecordById(db.CollectionChapters, e.Request.PathValue("chapterId"))
	if err != nil {
		return nil, nil, err
	}
	if chapter.GetString("project_id") != project.Id {
		return nil, nil, sql.ErrNoRows
	}
	return project, chapter, nil
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

func (s *Server) renderGridWithNotice(e *core.RequestEvent, status int, project *core.Record, chapter *core.Record, notice string) error {
	data, err := s.viewData(project, chapter)
	if err != nil {
		return e.InternalServerError("failed to prepare view data", err)
	}
	data.SaveNotice = notice
	html, err := views.RenderGrid(data)
	if err != nil {
		return e.InternalServerError("failed to render project grid", err)
	}
	return e.HTML(status, html)
}

func (s *Server) renderWorkspace(e *core.RequestEvent, status int, project *core.Record, chapter *core.Record, workspaceTab string) error {
	data, err := s.viewData(project, chapter)
	if err != nil {
		return e.InternalServerError("failed to prepare workspace data", err)
	}
	if workspaceTab != "" {
		data.WorkspaceTab = workspaceTab
	}
	html, err := views.RenderWorkspace(data)
	if err != nil {
		return e.InternalServerError("failed to render workspace", err)
	}
	return e.HTML(status, html)
}

func (s *Server) renderWorkspaceWithNotice(e *core.RequestEvent, status int, project *core.Record, chapter *core.Record, workspaceTab string, notice string) error {
	data, err := s.viewData(project, chapter)
	if err != nil {
		return e.InternalServerError("failed to prepare workspace data", err)
	}
	if workspaceTab != "" {
		data.WorkspaceTab = workspaceTab
	}
	data.SaveNotice = notice
	html, err := views.RenderWorkspace(data)
	if err != nil {
		return e.InternalServerError("failed to render workspace", err)
	}
	return e.HTML(status, html)
}

func (s *Server) renderChapterCockpit(e *core.RequestEvent, status int, project *core.Record, chapter *core.Record) error {
	data, err := s.viewData(project, chapter)
	if err != nil {
		return e.InternalServerError("failed to prepare chapter data", err)
	}
	data.WorkspaceTab = "drafting"
	html, err := views.RenderChapterCockpit(data)
	if err != nil {
		return e.InternalServerError("failed to render chapter cockpit", err)
	}
	return e.HTML(status, html)
}

func (s *Server) renderProcessing(e *core.RequestEvent, stage string, projectID string, chapterID string) error {
	return s.renderProcessingStatus(e, http.StatusAccepted, stage, projectID, chapterID)
}

func (s *Server) renderProcessingStatus(e *core.RequestEvent, status int, stage string, projectID string, chapterID string) error {
	html, err := views.RenderProcessing(views.ProcessingData{
		Stage:     stage,
		ProjectID: projectID,
		ChapterID: chapterID,
	})
	if err != nil {
		return e.InternalServerError("failed to render processing state", err)
	}
	return e.HTML(status, html)
}

func (s *Server) renderStatus(e *core.RequestEvent, statusCode int, jobType string, status string, chapterID string, errorMessage string) error {
	html, err := views.RenderStatus(views.StatusData{
		JobType:      jobType,
		Status:       status,
		ChapterID:    chapterID,
		ErrorMessage: errorMessage,
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
	chapters, err := s.chapterViews(project.Id)
	if err != nil {
		return views.PageData{}, err
	}
	data := views.PageData{
		Project:         projectView(project),
		AllProjects:     projects,
		ActiveProjectID: project.Id,
	}
	data.Project.Intake, err = s.intakeMap(project.Id)
	if err != nil {
		return views.PageData{}, err
	}
	data.IntakeSaved, err = s.hasIntake(project.Id)
	if err != nil {
		return views.PageData{}, err
	}
	if chapter != nil {
		view := chapterView(chapter)
		data.Chapter = &view
	}
	data.Chapters = chapters
	if data.Chapter != nil {
		data.PrevChapter, data.NextChapter = adjacentChapters(chapters, data.Chapter.ID)
	}
	if brief, err := s.briefInput(project.Id); err == nil && brief.Title != "" {
		data.Brief = &views.Brief{
			Title:         brief.Title,
			Subtitle:      brief.Subtitle,
			Promise:       brief.Promise,
			VoiceTone:     brief.VoiceTone,
			WhatItIs:      brief.WhatItIs,
			WhatItIsNot:   brief.WhatItIsNot,
			AISuggestions: brief.AISuggestions,
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return views.PageData{}, err
	}
	data.WorkspaceTab = workspaceTab(data)
	return data, nil
}

func workspaceTab(data views.PageData) string {
	if data.Chapter != nil && data.Project.Status == db.ProjectStatusDrafting {
		return "drafting"
	}
	if len(data.Chapters) > 0 {
		return "outline"
	}
	if data.Brief != nil {
		return "brief"
	}
	return "metadata"
}

func adjacentChapters(chapters []views.Chapter, chapterID string) (*views.Chapter, *views.Chapter) {
	for index, chapter := range chapters {
		if chapter.ID != chapterID {
			continue
		}
		var prev *views.Chapter
		var next *views.Chapter
		if index > 0 {
			value := chapters[index-1]
			prev = &value
		}
		if index+1 < len(chapters) {
			value := chapters[index+1]
			next = &value
		}
		return prev, next
	}
	return nil, nil
}

func (s *Server) chapterViews(projectID string) ([]views.Chapter, error) {
	records, err := s.app.FindRecordsByFilter(
		db.CollectionChapters,
		"project_id = {:project_id}",
		"sort_order",
		0,
		0,
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return nil, err
	}
	result := make([]views.Chapter, 0, len(records))
	for _, record := range records {
		result = append(result, chapterView(record))
	}
	return result, nil
}

func (s *Server) exportBookData(project *core.Record) (exporter.Book, error) {
	brief, err := s.briefInput(project.Id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return exporter.Book{}, err
	}
	title := strings.TrimSpace(brief.Title)
	if title == "" {
		title = strings.TrimSpace(project.GetString("title"))
	}
	author := strings.TrimSpace(project.GetString("author_name"))
	if author == "" {
		author = strings.TrimSpace(os.Getenv("BOOK_AUTHOR"))
	}
	book := exporter.Book{
		ID:          project.Id,
		Title:       title,
		Subtitle:    brief.Subtitle,
		Description: brief.Promise,
		Author:      author,
		Language:    exportLanguage(),
	}
	records, err := s.app.FindRecordsByFilter(
		db.CollectionChapters,
		"project_id = {:project_id}",
		"sort_order",
		0,
		0,
		dbx.Params{"project_id": project.Id},
	)
	if err != nil {
		return exporter.Book{}, err
	}
	if len(records) == 0 {
		return exporter.Book{}, errors.New("export requires at least one chapter")
	}
	for _, record := range records {
		body, source := chapterExportBody(record)
		book.Chapters = append(book.Chapters, exporter.Chapter{
			ID:                   record.Id,
			SortOrder:            record.GetInt("sort_order"),
			Title:                record.GetString("title"),
			Subtitle:             record.GetString("subtitle"),
			Body:                 body,
			Source:               source,
			FrontMatterLabel:     record.GetString("front_matter_label"),
			FrontMatterBlurb:     record.GetString("front_matter_blurb"),
			Purpose:              record.GetString("purpose"),
			StateStart:           record.GetString("state_start"),
			StateEnd:             record.GetString("state_end"),
			ChapterMetadataJSON:  record.GetString("chapter_metadata_json"),
			ArcMetadataJSON:      record.GetString("arc_metadata_json"),
			ConceptJurisdiction:  record.GetString("concept_jurisdiction_json"),
			GenerationDirectives: record.GetString("generation_directives_json"),
			MediaPromptsJSON:     record.GetString("media_prompts_json"),
		})
	}
	return book, nil
}

func chapterExportBody(record *core.Record) (string, string) {
	for _, field := range []string{"draft_content", "targeted_rewrite", "raw_draft"} {
		value := strings.TrimSpace(record.GetString(field))
		if value != "" {
			return value, field
		}
	}
	return "", "missing"
}

func incompleteExportWarnings(book exporter.Book) []string {
	var warnings []string
	for _, chapter := range book.Chapters {
		if strings.TrimSpace(chapter.Body) == "" {
			warnings = append(warnings, fmt.Sprintf("chapter %d has no manuscript text", chapter.SortOrder))
		}
	}
	return warnings
}

func exportLanguage() string {
	language := strings.TrimSpace(os.Getenv("BOOK_LANGUAGE"))
	if language == "" {
		return "en"
	}
	return language
}

func cleanOptionalJSON(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return "", err
	}
	encoded, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded), nil
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
		ID:                      record.Id,
		Title:                   record.GetString("title"),
		BookType:                record.GetString("type"),
		Status:                  record.GetString("status"),
		TargetLength:            record.GetString("target_length"),
		TargetChapters:          record.GetInt("target_chapters"),
		AuthorName:              record.GetString("author_name"),
		PublishingMetadataJSON:  record.GetString("publishing_metadata_json"),
		BookArchitectureJSON:    record.GetString("book_architecture_json"),
		GlobalStyleContractJSON: record.GetString("global_style_contract_json"),
	}
}

func (s *Server) intakeProjectView(projectID string) (views.Project, error) {
	if projectID == "" {
		return views.Project{Intake: map[string]string{}}, nil
	}
	project, err := s.app.FindRecordById(db.CollectionProjects, projectID)
	if err != nil {
		return views.Project{}, err
	}
	view := projectView(project)
	view.Intake, err = s.intakeMap(projectID)
	if err != nil {
		return views.Project{}, err
	}
	return view, nil
}

func chapterView(record *core.Record) views.Chapter {
	return views.Chapter{
		ID:                   record.Id,
		SortOrder:            record.GetInt("sort_order"),
		Title:                record.GetString("title"),
		Subtitle:             record.GetString("subtitle"),
		FrontMatterLabel:     record.GetString("front_matter_label"),
		FrontMatterBlurb:     record.GetString("front_matter_blurb"),
		ChapterMetadataJSON:  record.GetString("chapter_metadata_json"),
		ArcMetadataJSON:      record.GetString("arc_metadata_json"),
		ConceptJurisdiction:  record.GetString("concept_jurisdiction_json"),
		GenerationDirectives: record.GetString("generation_directives_json"),
		MediaPromptsJSON:     record.GetString("media_prompts_json"),
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

func (s *Server) deleteIntakeResponse(projectID, key string) error {
	record, err := s.app.FindFirstRecordByFilter(
		db.CollectionIntakeResponses,
		"project_id = {:project_id} && key = {:key}",
		dbx.Params{"project_id": projectID, "key": key},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return s.app.Delete(record)
}

func (s *Server) createJob(projectID, chapterID, jobType string) (*core.Record, error) {
	if err := s.supersedeRunningJobs(projectID, chapterID, jobType); err != nil {
		return nil, err
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), timeoutForJob(jobType))
	defer cancel()

	err := s.executeJob(ctx, projectID, chapterID, jobID, jobType)
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
		job.Set("error_msg", "")
	}
	_ = s.app.Save(job)
}

func (s *Server) executeJob(ctx context.Context, projectID, chapterID, jobID, jobType string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	switch jobType {
	case "brief":
		if err := s.updateProjectStatus(projectID, db.ProjectStatusProcessing); err != nil {
			return err
		}
		if _, err := s.generateBrief(ctx, projectID); err != nil {
			return err
		}
		return s.updateProjectStatus(projectID, db.ProjectStatusBrief)
	case "toc":
		if err := s.updateProjectStatus(projectID, db.ProjectStatusProcessing); err != nil {
			return err
		}
		if _, err := s.briefInput(projectID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("book brief must be generated before outline and chapter shells")
			}
			return err
		}
		if err := s.ensureTOC(ctx, projectID); err != nil {
			return err
		}
		return s.updateProjectStatus(projectID, db.ProjectStatusTOC)
	case "autopilot":
		return s.executeAutopilot(ctx, projectID, jobID)
	}
	return s.executePipelineStage(ctx, projectID, chapterID, jobType)
}

func (s *Server) executeAutopilot(ctx context.Context, projectID string, jobID string) error {
	_ = s.updateJobProgress(jobID, "Auto: preparing Book Brief")
	if err := s.updateProjectStatus(projectID, db.ProjectStatusProcessing); err != nil {
		return err
	}
	if _, err := s.ensureBrief(ctx, projectID); err != nil {
		return err
	}
	_ = s.updateJobProgress(jobID, "Auto: preparing outline")
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
	for chapterIndex, chapter := range chapters {
		if strings.TrimSpace(chapter.GetString("draft_content")) != "" && chapter.GetString("status") == db.ChapterStatusCompleted {
			continue
		}
		chapterLabel := fmt.Sprintf("chapter %d/%d: %s", chapterIndex+1, len(chapters), chapter.GetString("title"))
		for _, stage := range []string{"draft", "diagnose", "rewrite", "polish"} {
			latestChapter, err := s.app.FindRecordById(db.CollectionChapters, chapter.Id)
			if err != nil {
				return err
			}
			chapter = latestChapter
			if stageAlreadyDone(chapter, stage) {
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			_ = s.updateJobProgress(jobID, fmt.Sprintf("Auto: %s - %s", chapterLabel, stageLabelForProgress(stage)))
			if err := s.executePipelineStage(ctx, projectID, chapter.Id, stage); err != nil {
				return err
			}
		}
	}
	_ = s.updateJobProgress(jobID, "Auto: complete")
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
		Project:              project,
		Brief:                brief,
		SortOrder:            chapter.GetInt("sort_order"),
		Title:                chapter.GetString("title"),
		Purpose:              chapter.GetString("purpose"),
		StateStart:           chapter.GetString("state_start"),
		StateEnd:             chapter.GetString("state_end"),
		ChapterMetadataJSON:  chapter.GetString("chapter_metadata_json"),
		ArcMetadataJSON:      chapter.GetString("arc_metadata_json"),
		ConceptJurisdiction:  chapter.GetString("concept_jurisdiction_json"),
		GenerationDirectives: chapter.GetString("generation_directives_json"),
		MediaPromptsJSON:     chapter.GetString("media_prompts_json"),
		RawDraft:             chapter.GetString("raw_draft"),
		Diagnosis:            chapter.GetString("editorial_diagnosis"),
		Rewrite:              chapter.GetString("targeted_rewrite"),
		DraftNotes:           chapter.GetString("user_draft_notes"),
		DiagnoseNote:         chapter.GetString("user_diagnosis_notes"),
		RewriteNotes:         chapter.GetString("user_rewrite_notes"),
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
	return s.generateBrief(ctx, projectID)
}

func (s *Server) generateBrief(ctx context.Context, projectID string) (prompts.Brief, error) {
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

func (s *Server) clearEmptyChapterShells(projectID string) error {
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
	for _, chapter := range chapters {
		if chapterHasManuscript(chapter) {
			return nil
		}
	}
	for _, chapter := range chapters {
		if err := s.app.Delete(chapter); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) ensureTOC(ctx context.Context, projectID string) error {
	project, err := s.projectInputByID(projectID)
	if err != nil {
		return err
	}
	existing, err := s.app.FindRecordsByFilter(
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
	if len(existing) > 0 {
		if len(existing) == project.TargetChapters {
			return nil
		}
		for _, chapter := range existing {
			if chapterHasManuscript(chapter) {
				return fmt.Errorf("outline already has %d chapters but the project target is %d; review or export existing drafted chapters before regenerating the outline", len(existing), project.TargetChapters)
			}
		}
		for _, chapter := range existing {
			if err := s.app.Delete(chapter); err != nil {
				return err
			}
		}
	}
	brief, err := s.briefInput(projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("book brief must be generated before outline and chapter shells")
		}
		return err
	}
	chapters, err := s.generateTOCChapters(ctx, project, brief)
	if err != nil {
		return err
	}
	return s.saveChapters(projectID, chapters)
}

func (s *Server) generateTOCChapters(ctx context.Context, project prompts.ProjectInput, brief prompts.Brief) ([]prompts.ChapterPlan, error) {
	output, err := s.llm.Generate(ctx, prompts.SystemPrompt(), prompts.BuildTOCPrompt(project, brief))
	if err != nil {
		return nil, err
	}
	chapters, parseErr := prompts.ParseTOC(output, project.TargetChapters)
	if parseErr == nil {
		return chapters, nil
	}
	for attempt := 1; attempt <= 2; attempt++ {
		output, err = s.llm.Generate(ctx, prompts.SystemPrompt(), prompts.BuildTOCRepairPrompt(project, brief, output, parseErr))
		if err != nil {
			return nil, err
		}
		chapters, parseErr = prompts.ParseTOC(output, project.TargetChapters)
		if parseErr == nil {
			return chapters, nil
		}
	}
	return nil, parseErr
}

func chapterHasManuscript(chapter *core.Record) bool {
	for _, field := range []string{"raw_draft", "editorial_diagnosis", "targeted_rewrite", "draft_content"} {
		if strings.TrimSpace(chapter.GetString(field)) != "" {
			return true
		}
	}
	return false
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
		ID:                      record.Id,
		Title:                   record.GetString("title"),
		BookType:                record.GetString("type"),
		TargetLength:            record.GetString("target_length"),
		TargetChapters:          record.GetInt("target_chapters"),
		Intake:                  intake,
		AuthorName:              record.GetString("author_name"),
		PublishingMetadataJSON:  record.GetString("publishing_metadata_json"),
		BookArchitectureJSON:    record.GetString("book_architecture_json"),
		GlobalStyleContractJSON: record.GetString("global_style_contract_json"),
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

func (s *Server) updateJobProgress(jobID string, progress string) error {
	job, err := s.app.FindRecordById(db.CollectionJobs, jobID)
	if err != nil {
		return err
	}
	job.Set("error_msg", strings.TrimSpace(progress))
	return s.app.Save(job)
}

func (s *Server) supersedeRunningJobs(projectID, chapterID, jobType string) error {
	filter := "project_id = {:project_id} && job_type = {:job_type} && status = 'running'"
	params := dbx.Params{"project_id": projectID, "job_type": jobType}
	if chapterID != "" {
		filter += " && chapter_id = {:chapter_id}"
		params["chapter_id"] = chapterID
	}
	records, err := s.app.FindRecordsByFilter(db.CollectionJobs, filter, "created", 0, 0, params)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, record := range records {
		record.Set("status", "failed")
		record.Set("error_msg", "superseded by a newer job")
		record.Set("completed_at", now)
		if err := s.app.Save(record); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) runningJobs(projectID string) ([]*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(
		db.CollectionJobs,
		"project_id = {:project_id} && status = 'running'",
		"created",
		0,
		0,
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	active := make([]*core.Record, 0, len(records))
	for _, record := range records {
		if jobTimedOut(record, now) {
			record.Set("status", "failed")
			record.Set("error_msg", "job timed out before reporting completion")
			record.Set("completed_at", now.Format(time.RFC3339))
			if err := s.app.Save(record); err != nil {
				return nil, err
			}
			continue
		}
		active = append(active, record)
	}
	return active, nil
}

func (s *Server) runningJobsForChapter(projectID, chapterID string) ([]*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(
		db.CollectionJobs,
		"project_id = {:project_id} && chapter_id = {:chapter_id} && status = 'running'",
		"created",
		0,
		0,
		dbx.Params{"project_id": projectID, "chapter_id": chapterID},
	)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	active := make([]*core.Record, 0, len(records))
	for _, record := range records {
		if jobTimedOut(record, now) {
			record.Set("status", "failed")
			record.Set("error_msg", "job timed out before reporting completion")
			record.Set("completed_at", now.Format(time.RFC3339))
			if err := s.app.Save(record); err != nil {
				return nil, err
			}
			continue
		}
		active = append(active, record)
	}
	return active, nil
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

func (s *Server) failedJobsForChapter(projectID, chapterID string) ([]*core.Record, error) {
	return s.app.FindRecordsByFilter(
		db.CollectionJobs,
		"project_id = {:project_id} && chapter_id = {:chapter_id} && status = 'failed'",
		"-updated",
		1,
		0,
		dbx.Params{"project_id": projectID, "chapter_id": chapterID},
	)
}

func (s *Server) latestProjectJob(projectID string) (*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(
		db.CollectionJobs,
		"project_id = {:project_id}",
		"-created",
		5,
		0,
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if strings.TrimSpace(record.GetString("chapter_id")) == "" {
			return record, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (s *Server) latestJobForChapter(projectID, chapterID string) (*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(
		db.CollectionJobs,
		"project_id = {:project_id} && chapter_id = {:chapter_id}",
		"-created",
		1,
		0,
		dbx.Params{"project_id": projectID, "chapter_id": chapterID},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, sql.ErrNoRows
	}
	return records[0], nil
}

func jobTimedOut(record *core.Record, now time.Time) bool {
	startedAt := strings.TrimSpace(record.GetString("started_at"))
	if startedAt == "" {
		return false
	}
	started, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return false
	}
	return now.Sub(started) > timeoutForJob(record.GetString("job_type"))
}

func runningJobMessage(record *core.Record) string {
	startedAt := strings.TrimSpace(record.GetString("started_at"))
	if startedAt == "" {
		return "waiting for job completion"
	}
	started, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return "waiting for job completion"
	}
	age := time.Since(started).Round(time.Second)
	return fmt.Sprintf("running for %s", age)
}

func stageLabelForProgress(stage string) string {
	switch stage {
	case "draft":
		return "drafting"
	case "diagnose":
		return "editor feedback"
	case "rewrite":
		return "rewriting"
	case "polish":
		return "finishing"
	default:
		return stage
	}
}

func stageAlreadyDone(chapter *core.Record, stage string) bool {
	switch stage {
	case "draft":
		return strings.TrimSpace(chapter.GetString("raw_draft")) != ""
	case "diagnose":
		return strings.TrimSpace(chapter.GetString("editorial_diagnosis")) != ""
	case "rewrite":
		return strings.TrimSpace(chapter.GetString("targeted_rewrite")) != ""
	case "polish":
		return strings.TrimSpace(chapter.GetString("draft_content")) != ""
	default:
		return false
	}
}

func timeoutForJob(jobType string) time.Duration {
	if jobType == "autopilot" {
		return autopilotJobTimeout
	}
	return jobTimeout
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
