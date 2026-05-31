package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterHooks(app *pocketbase.PocketBase) {
	app.OnRecordCreate(CollectionProjects).BindFunc(func(e *core.RecordEvent) error {
		applyProjectDefaults(e.Record)
		return e.Next()
	})

	app.OnRecordValidate(CollectionProjects).BindFunc(func(e *core.RecordEvent) error {
		if err := validateProject(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordCreate(CollectionIntakeResponses).BindFunc(func(e *core.RecordEvent) error {
		if err := validateIntakeResponse(e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordUpdate(CollectionIntakeResponses).BindFunc(func(e *core.RecordEvent) error {
		if err := validateIntakeResponse(e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordCreate(CollectionChapters).BindFunc(func(e *core.RecordEvent) error {
		applyChapterDefaults(e.App, e.Record)
		if err := validateChapter(e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordUpdate(CollectionChapters).BindFunc(func(e *core.RecordEvent) error {
		if err := validateChapter(e.Record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordAfterCreateSuccess(CollectionChapters).BindFunc(func(e *core.RecordEvent) error {
		return reconcileChapterOrder(e.App, e.Record.GetString("project_id"))
	})

	app.OnRecordAfterUpdateSuccess(CollectionChapters).BindFunc(func(e *core.RecordEvent) error {
		return reconcileChapterOrder(e.App, e.Record.GetString("project_id"))
	})

	app.OnRecordAfterDeleteSuccess(CollectionChapters).BindFunc(func(e *core.RecordEvent) error {
		return reconcileChapterOrder(e.App, e.Record.GetString("project_id"))
	})
}

func applyProjectDefaults(record *core.Record) {
	if record.GetString("title") == "" {
		record.Set("title", "Untitled Book")
	}
	if record.GetString("type") == "" {
		record.Set("type", "nonfiction")
	}
	if record.GetString("status") == "" {
		record.Set("status", ProjectStatusIntake)
	}
	if record.GetString("target_length") == "" {
		record.Set("target_length", TargetLengthPracticalEbook)
	}
	if record.GetInt("target_chapters") == 0 {
		record.Set("target_chapters", 10)
	}
	if raw := record.Get("ui_state"); raw == nil || fmt.Sprint(raw) == "" {
		record.Set("ui_state", DefaultUIState())
	}
}

func applyChapterDefaults(app core.App, record *core.Record) {
	if record.GetString("status") == "" {
		record.Set("status", ChapterStatusPending)
	}
	if record.GetInt("sort_order") <= 0 {
		next, err := nextChapterSortOrder(app, record.GetString("project_id"))
		if err == nil {
			record.Set("sort_order", next)
		}
	}
}

func validateProject(app core.App, record *core.Record) error {
	status := record.GetString("status")
	if !ValidProjectStatuses[status] {
		return fmt.Errorf("invalid project status %q", status)
	}
	bookType := record.GetString("type")
	if !ValidBookTypes[bookType] {
		return fmt.Errorf("invalid book type %q", bookType)
	}
	targetLength := record.GetString("target_length")
	if !ValidTargetLengths[targetLength] {
		return fmt.Errorf("invalid target length %q", targetLength)
	}
	if record.GetInt("target_chapters") < 1 {
		return errors.New("target_chapters must be greater than zero")
	}
	if status != ProjectStatusIntake {
		if targetLength == "" || record.GetInt("target_chapters") == 0 {
			return errors.New("project cannot leave intake without target_length and target_chapters")
		}
	}
	if status == ProjectStatusBrief || status == ProjectStatusTOC || status == ProjectStatusDrafting {
		if err := ensureBookBriefExists(app, record.Id); err != nil {
			return err
		}
	}
	return validateUIState(record.Get("ui_state"))
}

func validateIntakeResponse(record *core.Record) error {
	key := record.GetString("key")
	if !ValidIntakeKeys[key] {
		return fmt.Errorf("invalid intake response key %q", key)
	}
	if record.GetString("project_id") == "" {
		return errors.New("intake response requires project_id")
	}
	return nil
}

func validateChapter(record *core.Record) error {
	if record.GetString("project_id") == "" {
		return errors.New("chapter requires project_id")
	}
	if record.GetInt("sort_order") < 1 {
		return errors.New("chapter sort_order must be a positive integer")
	}
	status := record.GetString("status")
	if !ValidChapterStatuses[status] {
		return fmt.Errorf("invalid chapter status %q", status)
	}
	return nil
}

func validateUIState(raw any) error {
	if raw == nil {
		return errors.New("ui_state is required")
	}
	switch value := raw.(type) {
	case string:
		if value == "" {
			return errors.New("ui_state is required")
		}
		var state UIState
		if err := json.Unmarshal([]byte(value), &state); err != nil {
			return fmt.Errorf("ui_state must be valid JSON: %w", err)
		}
	case []byte:
		var state UIState
		if err := json.Unmarshal(value, &state); err != nil {
			return fmt.Errorf("ui_state must be valid JSON: %w", err)
		}
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("ui_state must be JSON serializable: %w", err)
		}
		var state UIState
		if err := json.Unmarshal(encoded, &state); err != nil {
			return fmt.Errorf("ui_state must match expected schema: %w", err)
		}
	}
	return nil
}

func ensureBookBriefExists(app core.App, projectID string) error {
	if projectID == "" {
		return errors.New("project id is required before checking book brief guardrails")
	}
	_, err := app.FindFirstRecordByFilter(
		CollectionBookBriefs,
		"project_id = {:project_id}",
		dbx.Params{"project_id": projectID},
	)
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("project cannot advance without a verified book_briefs row")
	}
	return err
}

func nextChapterSortOrder(app core.App, projectID string) (int, error) {
	if projectID == "" {
		return 1, nil
	}
	records, err := app.FindRecordsByFilter(
		CollectionChapters,
		"project_id = {:project_id}",
		"-sort_order",
		1,
		0,
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 1, nil
	}
	return records[0].GetInt("sort_order") + 1, nil
}

func reconcileChapterOrder(app core.App, projectID string) error {
	if projectID == "" {
		return nil
	}
	records, err := app.FindRecordsByFilter(
		CollectionChapters,
		"project_id = {:project_id}",
		"sort_order,created",
		0,
		0,
		dbx.Params{"project_id": projectID},
	)
	if err != nil {
		return err
	}
	for i, record := range records {
		expected := i + 1
		if record.GetInt("sort_order") == expected {
			continue
		}
		record.Set("sort_order", expected)
		if err := app.Save(record); err != nil {
			return err
		}
	}
	return nil
}
