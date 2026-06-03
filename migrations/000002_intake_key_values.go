package migrations

import (
	"fmt"

	"bookbuilder/internal/db"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

var intakeResponseKeys = []string{
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

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(db.CollectionIntakeResponses)
		if err != nil {
			return err
		}
		field, ok := collection.Fields.GetByName("key").(*core.SelectField)
		if !ok {
			return fmt.Errorf("%s.key is not a select field", db.CollectionIntakeResponses)
		}
		field.Values = mergeStringSet(field.Values, intakeResponseKeys)
		return app.Save(collection)
	}, func(app core.App) error {
		return nil
	})
}

func mergeStringSet(existing []string, required []string) []string {
	seen := make(map[string]bool, len(existing)+len(required))
	merged := make([]string, 0, len(existing)+len(required))
	for _, value := range existing {
		if seen[value] {
			continue
		}
		seen[value] = true
		merged = append(merged, value)
	}
	for _, value := range required {
		if seen[value] {
			continue
		}
		seen[value] = true
		merged = append(merged, value)
	}
	return merged
}
