package migrations

import (
	"bookbuilder/internal/db"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		projects, err := createProjectsCollection(app)
		if err != nil {
			return err
		}
		if err := createIntakeResponsesCollection(app, projects); err != nil {
			return err
		}
		if err := createBookBriefsCollection(app, projects); err != nil {
			return err
		}
		chapters, err := createChaptersCollection(app, projects)
		if err != nil {
			return err
		}
		return createJobsCollection(app, projects, chapters)
	}, func(app core.App) error {
		for _, name := range []string{
			db.CollectionJobs,
			db.CollectionChapters,
			db.CollectionBookBriefs,
			db.CollectionIntakeResponses,
			db.CollectionProjects,
		} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue
			}
			if err := app.Delete(collection); err != nil {
				return err
			}
		}
		return nil
	})
}

func createProjectsCollection(app core.App) (*core.Collection, error) {
	collection := core.NewBaseCollection(db.CollectionProjects)
	collection.Fields.Add(
		&core.TextField{Name: "title", Required: true, Max: 240, Presentable: true},
		&core.SelectField{Name: "type", Required: true, Values: []string{"fiction", "nonfiction"}},
		&core.SelectField{Name: "status", Required: true, Values: []string{
			db.ProjectStatusIntake,
			db.ProjectStatusProcessing,
			db.ProjectStatusBrief,
			db.ProjectStatusTOC,
			db.ProjectStatusDrafting,
			db.ProjectStatusFailedJob,
		}},
		&core.SelectField{Name: "target_length", Required: true, Values: []string{
			db.TargetLengthShortGuide,
			db.TargetLengthPracticalEbook,
			db.TargetLengthFullPrototype,
		}},
		&core.NumberField{Name: "target_chapters", Required: true, OnlyInt: true, Min: floatPtr(1), Max: floatPtr(25)},
		&core.JSONField{Name: "ui_state", Required: true, MaxSize: 65536},
		&core.TextField{Name: "voice_card", Max: 20000},
		&core.TextField{Name: "author_name", Max: 240},
		&core.TextField{Name: "publishing_metadata_json", Max: 100000},
		&core.TextField{Name: "book_architecture_json", Max: 100000},
		&core.TextField{Name: "global_style_contract_json", Max: 100000},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	collection.AddIndex("idx_projects_updated", false, "updated", "")
	return collection, app.Save(collection)
}

func createIntakeResponsesCollection(app core.App, projects *core.Collection) error {
	collection := core.NewBaseCollection(db.CollectionIntakeResponses)
	collection.Fields.Add(
		&core.RelationField{Name: "project_id", Required: true, CollectionId: projects.Id, CascadeDelete: true},
		&core.SelectField{Name: "key", Required: true, Values: []string{
			"core_topic",
			"target_audience",
			"reader_hunger",
			"prohibited_directions",
			"author_intent",
			"selected_tone",
			"book_form",
			"narrative_pov",
			"structure_model",
		}},
		&core.TextField{Name: "value", Required: true, Max: 20000},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	collection.AddIndex("idx_intake_project_key", true, "project_id, key", "")
	return app.Save(collection)
}

func createBookBriefsCollection(app core.App, projects *core.Collection) error {
	collection := core.NewBaseCollection(db.CollectionBookBriefs)
	collection.Fields.Add(
		&core.RelationField{Name: "project_id", Required: true, CollectionId: projects.Id, CascadeDelete: true},
		&core.TextField{Name: "title", Required: true, Max: 240, Presentable: true},
		&core.TextField{Name: "subtitle", Max: 500},
		&core.TextField{Name: "promise", Max: 1000},
		&core.TextField{Name: "voice_tone", Max: 1000},
		&core.TextField{Name: "what_it_is", Max: 20000},
		&core.TextField{Name: "what_it_is_not", Max: 20000},
		&core.TextField{Name: "ai_suggestions", Max: 20000},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	collection.AddIndex("idx_book_briefs_project", true, "project_id", "")
	return app.Save(collection)
}

func createChaptersCollection(app core.App, projects *core.Collection) (*core.Collection, error) {
	collection := core.NewBaseCollection(db.CollectionChapters)
	collection.Fields.Add(
		&core.RelationField{Name: "project_id", Required: true, CollectionId: projects.Id, CascadeDelete: true},
		&core.NumberField{Name: "sort_order", Required: true, OnlyInt: true, Min: floatPtr(1)},
		&core.TextField{Name: "title", Required: true, Max: 240, Presentable: true},
		&core.TextField{Name: "subtitle", Max: 500},
		&core.TextField{Name: "front_matter_label", Max: 240},
		&core.TextField{Name: "front_matter_blurb", Max: 20000},
		&core.TextField{Name: "chapter_metadata_json", Max: 100000},
		&core.TextField{Name: "arc_metadata_json", Max: 100000},
		&core.TextField{Name: "concept_jurisdiction_json", Max: 100000},
		&core.TextField{Name: "generation_directives_json", Max: 100000},
		&core.TextField{Name: "media_prompts_json", Max: 100000},
		&core.SelectField{Name: "status", Required: true, Values: []string{
			db.ChapterStatusPending,
			db.ChapterStatusCardApproved,
			db.ChapterStatusDraftingStage1,
			db.ChapterStatusDraftingStage2,
			db.ChapterStatusDraftingStage3,
			db.ChapterStatusCompleted,
		}},
		&core.TextField{Name: "purpose", Max: 10000},
		&core.TextField{Name: "state_start", Max: 10000},
		&core.TextField{Name: "state_end", Max: 10000},
		&core.TextField{Name: "raw_draft", Max: 300000},
		&core.TextField{Name: "editorial_diagnosis", Max: 120000},
		&core.TextField{Name: "targeted_rewrite", Max: 300000},
		&core.TextField{Name: "draft_content", Max: 300000},
		&core.TextField{Name: "previous_draft_content", Max: 300000},
		&core.TextField{Name: "user_draft_notes", Max: 20000},
		&core.TextField{Name: "user_diagnosis_notes", Max: 20000},
		&core.TextField{Name: "user_rewrite_notes", Max: 20000},
		&core.TextField{Name: "manual_edit_content", Max: 300000},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	collection.AddIndex("idx_chapters_project_sort", true, "project_id, sort_order", "")
	return collection, app.Save(collection)
}

func createJobsCollection(app core.App, projects *core.Collection, chapters *core.Collection) error {
	collection := core.NewBaseCollection(db.CollectionJobs)
	collection.Fields.Add(
		&core.RelationField{Name: "project_id", Required: true, CollectionId: projects.Id, CascadeDelete: true},
		&core.RelationField{Name: "chapter_id", CollectionId: chapters.Id, CascadeDelete: true},
		&core.SelectField{Name: "job_type", Required: true, Values: []string{
			"brief",
			"toc",
			"autopilot",
			"draft",
			"diagnose",
			"rewrite",
			"polish",
		}},
		&core.SelectField{Name: "status", Required: true, Values: []string{
			"queued",
			"running",
			"completed",
			"failed",
		}},
		&core.TextField{Name: "error_msg", Max: 20000},
		&core.TextField{Name: "started_at", Max: 64},
		&core.TextField{Name: "completed_at", Max: 64},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	collection.AddIndex("idx_jobs_project_status", false, "project_id, status", "")
	collection.AddIndex("idx_jobs_chapter_status", false, "chapter_id, status", "")
	return app.Save(collection)
}

func floatPtr(value float64) *float64 {
	return &value
}
