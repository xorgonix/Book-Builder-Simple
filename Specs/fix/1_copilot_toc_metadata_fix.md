# Copilot Change Request: Fix TOC Chapter Cards Metadata Wiring

## Project
`Book-Builder-Simple`

## Goal
The generated Table of Contents / Outline cards should carry and display chapter-level metadata, not just chapter title, purpose, reader entry, and reader exit.

The app already has metadata fields in the chapter model and metadata editing UI. The missing piece is that the generated TOC chapter plans do not reliably create, parse, save, and show those metadata fields on the outline cards.

## Expected result
After generating the outline, each TOC card should include:

- Chapter title
- Chapter purpose
- Reader entry state
- Reader exit state
- Chapter Metadata JSON
- Arc Metadata JSON
- Concept Jurisdiction JSON
- Generation Directives JSON
- Media Prompts JSON

The metadata can be shown in a collapsible raw JSON panel for now. Do **not** overbuild a fancy parsed metadata UI yet.

---

# Data Flow That Must Work

```text
BuildTOCPrompt()
  -> LLM response
  -> ParseTOC()
  -> ChapterPlan
  -> saveChapters()
  -> PocketBase chapters record
  -> chapterView()
  -> views.Chapter
  -> outline_workspace template
```

If metadata is missing from any one step, the TOC cards will not show it.

---

# Files To Update

## 1. `internal/prompts/prompts.go`

### A. Extend `ChapterPlan`

Find the `ChapterPlan` struct and add these fields:

```go
ChapterMetadataJSON  string
ArcMetadataJSON      string
ConceptJurisdiction  string
GenerationDirectives string
MediaPromptsJSON     string
```

The struct should contain at least:

```go
type ChapterPlan struct {
    SortOrder            int
    Title                string
    Purpose              string
    StateStart           string
    StateEnd             string
    ChapterMetadataJSON  string
    ArcMetadataJSON      string
    ConceptJurisdiction  string
    GenerationDirectives string
    MediaPromptsJSON     string
}
```

### B. Update `BuildTOCPrompt()`

The TOC prompt must explicitly ask the LLM to output metadata fields for every chapter.

Update the required chapter block format to include:

```text
CHAPTER_START
Order: [Sequence Integer starting at 1]
Title: [A compelling, clear chapter title]
Purpose: [What the chapter must mechanically accomplish to advance the book's promise]
Reader Start: [The precise emotional or intellectual frustration of the reader on entry]
Reader End: [The transformation goal or clarity target of the reader upon exiting this chapter]
Chapter Metadata JSON: {"chapter_function":"","required_elements":[],"avoid":[]}
Arc Metadata JSON: {"arc_phase":"","pov_character":"","external_plot_movement":"","internal_shift":"","relationship_shift":"","beats":[]}
Concept Jurisdiction JSON: {"owns":[],"may_reference":[],"must_not_reteach":[],"forbidden_phrases":[]}
Generation Directives JSON: {"write_only_prose":true,"required_elements":[],"avoid":[],"revision_notes":""}
Media Prompts JSON: {"images":[],"diagrams":[]}
CHAPTER_END
```

Also add a clear instruction:

```text
Each JSON field must be valid compact JSON on a single line. Do not use markdown fences. Do not leave required JSON fields blank.
```

### C. Update `ParseTOC()`

Where `parseLabelBlock()` is called, include these labels:

```go
"Chapter Metadata JSON",
"Arc Metadata JSON",
"Concept Jurisdiction JSON",
"Generation Directives JSON",
"Media Prompts JSON",
```

Then populate the new `ChapterPlan` fields:

```go
ChapterMetadataJSON:  values["Chapter Metadata JSON"],
ArcMetadataJSON:      values["Arc Metadata JSON"],
ConceptJurisdiction:  values["Concept Jurisdiction JSON"],
GenerationDirectives: values["Generation Directives JSON"],
MediaPromptsJSON:     values["Media Prompts JSON"],
```

### D. Update `BuildTOCRepairPrompt()` if needed

If the repair prompt repeats the required output format, make sure it includes the same metadata fields and the same single-line valid JSON requirement.

---

## 2. `internal/routes/routes.go`

### A. Update `saveChapters()`

In `saveChapters()`, after setting:

```go
record.Set("purpose", plan.Purpose)
record.Set("state_start", plan.StateStart)
record.Set("state_end", plan.StateEnd)
```

also set:

```go
record.Set("chapter_metadata_json", emptyJSONFallback(plan.ChapterMetadataJSON, `{"chapter_function":"","required_elements":[],"avoid":[]}`))
record.Set("arc_metadata_json", emptyJSONFallback(plan.ArcMetadataJSON, `{"arc_phase":"","pov_character":"","external_plot_movement":"","internal_shift":"","relationship_shift":"","beats":[]}`))
record.Set("concept_jurisdiction_json", emptyJSONFallback(plan.ConceptJurisdiction, `{"owns":[],"may_reference":[],"must_not_reteach":[],"forbidden_phrases":[]}`))
record.Set("generation_directives_json", emptyJSONFallback(plan.GenerationDirectives, `{"write_only_prose":true,"required_elements":[],"avoid":[],"revision_notes":""}`))
record.Set("media_prompts_json", emptyJSONFallback(plan.MediaPromptsJSON, `{"images":[],"diagrams":[]}`))
```

Add this helper if it does not already exist:

```go
func emptyJSONFallback(value string, fallback string) string {
    value = strings.TrimSpace(value)
    if value == "" {
        return fallback
    }
    return value
}
```

`routes.go` already uses `strings`, so this should not require a new import if `strings` is already present.

### B. Update `chapterView()`

Make sure `chapterView()` maps these fields from the PocketBase record into `views.Chapter`:

```go
StateStart:           record.GetString("state_start"),
StateEnd:             record.GetString("state_end"),
ChapterMetadataJSON:  record.GetString("chapter_metadata_json"),
ArcMetadataJSON:      record.GetString("arc_metadata_json"),
ConceptJurisdiction:  record.GetString("concept_jurisdiction_json"),
GenerationDirectives: record.GetString("generation_directives_json"),
MediaPromptsJSON:     record.GetString("media_prompts_json"),
```

---

## 3. `internal/views/views.go`

### A. Update `views.Chapter`

Make sure the `Chapter` view struct includes these fields:

```go
StateStart           string
StateEnd             string
ChapterMetadataJSON  string
ArcMetadataJSON      string
ConceptJurisdiction  string
GenerationDirectives string
MediaPromptsJSON     string
```

The uploaded template already references `.StateStart` and `.StateEnd` in `outline_workspace`, so the view struct must include them.

### B. Update `outline_workspace`

The existing outline cards show metadata badges, but they should also show the actual metadata.

Inside each chapter card, after the metadata badge row, add a collapsible metadata panel.

Use raw JSON display for now:

```gotemplate
{{ if or .ChapterMetadataJSON .ArcMetadataJSON .ConceptJurisdiction .GenerationDirectives .MediaPromptsJSON }}
<div class="collapse collapse-arrow bg-base-200 border border-base-300 rounded-lg mt-3">
    <input type="checkbox" />
    <div class="collapse-title text-xs font-bold uppercase text-base-content/70">
        Metadata control layer
    </div>
    <div class="collapse-content space-y-3 text-xs">
        {{ if .ChapterMetadataJSON }}
        <div>
            <p class="font-bold text-base-content/70">Chapter Metadata</p>
            <pre class="whitespace-pre-wrap bg-base-100 border border-base-300 rounded p-2 overflow-x-auto">{{ .ChapterMetadataJSON }}</pre>
        </div>
        {{ end }}

        {{ if .ArcMetadataJSON }}
        <div>
            <p class="font-bold text-base-content/70">Arc Metadata</p>
            <pre class="whitespace-pre-wrap bg-base-100 border border-base-300 rounded p-2 overflow-x-auto">{{ .ArcMetadataJSON }}</pre>
        </div>
        {{ end }}

        {{ if .ConceptJurisdiction }}
        <div>
            <p class="font-bold text-base-content/70">Concept Jurisdiction</p>
            <pre class="whitespace-pre-wrap bg-base-100 border border-base-300 rounded p-2 overflow-x-auto">{{ .ConceptJurisdiction }}</pre>
        </div>
        {{ end }}

        {{ if .GenerationDirectives }}
        <div>
            <p class="font-bold text-base-content/70">Generation Directives</p>
            <pre class="whitespace-pre-wrap bg-base-100 border border-base-300 rounded p-2 overflow-x-auto">{{ .GenerationDirectives }}</pre>
        </div>
        {{ end }}

        {{ if .MediaPromptsJSON }}
        <div>
            <p class="font-bold text-base-content/70">Media Prompts</p>
            <pre class="whitespace-pre-wrap bg-base-100 border border-base-300 rounded p-2 overflow-x-auto">{{ .MediaPromptsJSON }}</pre>
        </div>
        {{ end }}
    </div>
</div>
{{ end }}
```

Do not remove the existing badges. The badges are useful as a quick status indicator.

---

# Important Constraint

Do **not** remove existing functionality, debugging helpers, HTMX behavior, metadata editor forms, export behavior, or existing badges.

This is a wiring fix, not a redesign.

---

# Acceptance Checks

After the changes:

1. Generate a new outline.
2. Confirm each generated chapter record in PocketBase has non-empty values for:
   - `chapter_metadata_json`
   - `arc_metadata_json`
   - `concept_jurisdiction_json`
   - `generation_directives_json`
   - `media_prompts_json`
3. Open the Outline tab.
4. Confirm each TOC card shows:
   - Entry
   - Exit
   - Metadata badges
   - Collapsible `Metadata control layer`
5. Expand the panel and confirm the raw JSON appears.
6. Open a chapter in the metadata workspace and confirm the same metadata is editable there.
7. Run the app and confirm no template runtime error occurs for `.StateStart`, `.StateEnd`, or metadata fields.

---

# Likely Root Cause

The metadata columns and metadata editing UI already exist, but the generated TOC chapter plan did not carry metadata all the way through:

```text
Prompt did not require metadata strongly enough
Parser did not parse metadata labels
ChapterPlan did not store metadata
saveChapters did not write metadata fields
Outline card displayed badges only, not contents
```

Fix the entire chain rather than only the visible template.
