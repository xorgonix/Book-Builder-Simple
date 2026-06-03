package migrations

import (
	"bookbuilder/internal/db"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		projects, err := app.FindCollectionByNameOrId(db.CollectionProjects)
		if err != nil {
			return err
		}
		if projects.Fields.GetByName("author_name") == nil {
			projects.Fields.Add(&core.TextField{Name: "author_name", Max: 240})
		}
		if projects.Fields.GetByName("publishing_metadata_json") == nil {
			projects.Fields.Add(&core.TextField{Name: "publishing_metadata_json", Max: 100000})
		}
		if projects.Fields.GetByName("book_architecture_json") == nil {
			projects.Fields.Add(&core.TextField{Name: "book_architecture_json", Max: 100000})
		}
		if projects.Fields.GetByName("global_style_contract_json") == nil {
			projects.Fields.Add(&core.TextField{Name: "global_style_contract_json", Max: 100000})
		}
		if err := app.Save(projects); err != nil {
			return err
		}

		chapters, err := app.FindCollectionByNameOrId(db.CollectionChapters)
		if err != nil {
			return err
		}
		if chapters.Fields.GetByName("chapter_metadata_json") == nil {
			chapters.Fields.Add(&core.TextField{Name: "chapter_metadata_json", Max: 100000})
		}
		if chapters.Fields.GetByName("arc_metadata_json") == nil {
			chapters.Fields.Add(&core.TextField{Name: "arc_metadata_json", Max: 100000})
		}
		if chapters.Fields.GetByName("concept_jurisdiction_json") == nil {
			chapters.Fields.Add(&core.TextField{Name: "concept_jurisdiction_json", Max: 100000})
		}
		if chapters.Fields.GetByName("generation_directives_json") == nil {
			chapters.Fields.Add(&core.TextField{Name: "generation_directives_json", Max: 100000})
		}
		if chapters.Fields.GetByName("media_prompts_json") == nil {
			chapters.Fields.Add(&core.TextField{Name: "media_prompts_json", Max: 100000})
		}
		return app.Save(chapters)
	}, func(app core.App) error {
		return nil
	})
}
