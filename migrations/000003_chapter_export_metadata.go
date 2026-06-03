package migrations

import (
	"bookbuilder/internal/db"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(db.CollectionChapters)
		if err != nil {
			return err
		}
		if collection.Fields.GetByName("subtitle") == nil {
			collection.Fields.Add(&core.TextField{Name: "subtitle", Max: 500})
		}
		if collection.Fields.GetByName("front_matter_label") == nil {
			collection.Fields.Add(&core.TextField{Name: "front_matter_label", Max: 240})
		}
		if collection.Fields.GetByName("front_matter_blurb") == nil {
			collection.Fields.Add(&core.TextField{Name: "front_matter_blurb", Max: 20000})
		}
		return app.Save(collection)
	}, func(app core.App) error {
		return nil
	})
}
