# Book-Builder-Simple: JSON-Backed Mini-Form Metadata Editor

## Quick Start (Read First)

This document defines a metadata control system with one strict split:

- Canonical chapter/book fields are normal form fields.
- Optional control-layer fields are JSON.

Implementation shorthand:

1. Keep canonical values outside JSON.
2. Keep JSON valid, compact, and schema-shaped.
3. Let both users and generators update metadata.
4. Save through existing routes and JSON validation.
5. Keep raw JSON behind an Advanced drawer only.

If a value is core identity/state/prose, it is canonical.
If a value is optional guidance/constraints/control hints, it belongs in metadata JSON.

## What This Is For

This file defines a small metadata system for book and chapter cards that is easy for users to edit and safe for the app to store.

The system must support two truths at once:

1. Every chapter has a core set of canonical fields (title, purpose, reader entry/exit, etc.).
2. Not every chapter needs the same extra metadata, because genre and book type differ.

So the design is:

- Canonical fields stay first-class fields outside JSON.
- Metadata JSON is a control layer for optional or advanced controls.
- JSON should be plain, compact, valid JSON (vanilla JSON), not a custom format.

The system must allow metadata values to be updated from:

- user edits in the form editor
- system generation (for example TOC/outline generation)
- later regeneration/repair flows

This means metadata is not static text. It is living control data that can be generated, edited, saved, and reused through the pipeline.

---

## System Update Contract

The metadata layer must support two update sources with the same storage rules:

### User update flow

Mini-form editor -> hidden JSON textarea -> existing save route -> `cleanOptionalJSON` validation -> PocketBase JSON fields.

### System update flow

Prompt generation (for example TOC) -> parser -> chapter plan -> save route -> PocketBase JSON fields.

### Conflict rule

Canonical fields always win.

If generated JSON contains canonical duplicates (title, purpose, reader state, status, prose fields), drop those duplicates before save or reject them.

### Missing optional fields

Missing optional fields are allowed.

Do not force every chapter to carry every optional control object.

## Goal

Replace raw JSON editing as the primary user interface with reusable, DaisyUI-styled mini-form editors.

The user should edit meaningful book/chapter controls such as:

- Chapter Planning
- Concept Control
- Writing Instructions
- Story / Argument Arc
- Media Ideas
- Publishing / Cover Metadata
- Book Architecture
- Global Style Contract

The JSON fields remain the storage format in PocketBase, but users should not normally see or edit raw JSON.

Raw JSON should remain available only in an **Advanced Raw JSON** drawer for debugging and power users.

---

## Non-Negotiable Data Contract

### A. Canonical fields are always editable in normal form fields

Examples:

- chapter order (`sort_order`)
- chapter title (`title`)
- chapter subtitle (`subtitle`)
- chapter purpose (`purpose`)
- reader entry (`state_start`)
- reader exit (`state_end`)
- front matter label (`front_matter_label`)
- front matter blurb (`front_matter_blurb`)
- status (`status`)

### B. Do not duplicate canonical fields inside metadata JSON

Do not store clones like:

- `order`, `sequence`, `chapter_number`
- `title`, `subtitle`
- `reader_start`, `reader_end`, `starting_state`, `ending_state`
- workflow keys (`status`), or prose payload keys (`raw_draft`, `draft_content`)

### C. Optional fields belong in JSON control layer

Metadata JSON should hold optional controls that may exist for some chapters and not others, such as:

- required elements
- avoid lists
- concept jurisdiction
- arc beats
- media prompts
- generation directives

### D. Genre-aware optionality

Different genres need different controls. The UI should support optional sections without forcing every chapter to fill every field.

Examples:

- Some nonfiction chapters may not need media prompts.
- Some fiction chapters may need richer arc controls.
- Some book types may require stricter format constraints.

Required canonical fields remain stable; optional JSON controls vary by use case.

---

## JSON Quality Rules

All metadata JSON must be:

- valid JSON
- plain JSON (no markdown fences)
- compact enough for safe storage and prompt reuse
- free of canonical-duplicate keys

Recommended server behavior:

- reject invalid JSON
- allow blank JSON
- optionally normalize known duplicate keys out of metadata payloads

---

## Existing storage model

The project already has JSON text fields at project level:

- `publishing_metadata_json`
- `book_architecture_json`
- `global_style_contract_json`

The project already has JSON text fields at chapter level:

- `chapter_metadata_json`
- `arc_metadata_json`
- `concept_jurisdiction_json`
- `generation_directives_json`
- `media_prompts_json`

Do not replace these database fields.

Instead, create a reusable browser-side mini-form layer that:

1. Reads JSON from a hidden textarea/input.
2. Renders friendly controls based on a simple local schema.
3. Updates the hidden JSON field whenever the user changes a value.
4. Allows normal server save behavior to continue.
5. Keeps an advanced raw JSON editor/drawer as fallback.

---

## Core design rule

Do not duplicate first-class fields inside metadata JSON.

These are canonical outside JSON and should not be repeated in JSON:

### Project-level canonical fields

- `title`
- `type`
- `status`
- `target_length`
- `target_chapters`
- `author_name`

### Book brief canonical fields

- `title`
- `subtitle`
- `promise`
- `voice_tone`
- `what_it_is`
- `what_it_is_not`
- `ai_suggestions`

### Chapter-level canonical fields

- `sort_order`
- `title`
- `subtitle`
- `purpose`
- `state_start`
- `state_end`
- `front_matter_label`
- `front_matter_blurb`
- `status`
- `raw_draft`
- `draft_content`
- `previous_draft_content`
- `editorial_diagnosis`
- `targeted_rewrite`
- `manual_edit_content`

The JSON layer is a control layer, not a second copy of the form.

---

## User-facing labels

Never show database field names as primary labels.

Use these labels instead:

| Database field | User-facing label |
|---|---|
| `publishing_metadata_json` | Publishing Package |
| `book_architecture_json` | Book Architecture |
| `global_style_contract_json` | Global Style Contract |
| `chapter_metadata_json` | Chapter Planning |
| `arc_metadata_json` | Story / Argument Arc |
| `concept_jurisdiction_json` | Concept Control |
| `generation_directives_json` | Writing Instructions |
| `media_prompts_json` | Media Ideas |

---

## Data types the editor must support

Build one reusable editor that supports these field types:

### Singleton fields

One value.

Supported controls:

- `text`
- `textarea`
- `select`
- `checkbox`

Examples:

- `chapter_role`
- `arc_phase`
- `write_only_prose`
- `revision_notes`
- `cover_direction`
- `back_cover_tone`

### String arrays

A repeatable list of string values.

Supported UI:

- Add item button
- Editable list rows or chips
- Delete item button
- Optional drag handle later

Examples:

- `required_elements`
- `avoid`
- `owns`
- `may_reference`
- `must_not_reteach`
- `forbidden_phrases`
- `style_constraints`
- `format_constraints`
- `beats`
- `keywords`
- `sales_angles`

### Object arrays

A repeatable list where each item has multiple fields.

Supported UI:

- Add item button
- Mini-card per object
- Per-object text/select/textarea fields
- Delete object button

Examples:

- image prompts
- diagram prompts
- cover concepts
- back-cover bullets
- comparison titles
- sample reader objections

---

## Recommended JSON shapes

### 1. `chapter_metadata_json`

Purpose: chapter planning controls not already represented by title, purpose, reader start, or reader end.

```json
{
  "chapter_role": "",
  "required_elements": [],
  "avoid": [],
  "structural_notes": []
}
```

UI fields:

- Chapter Role: select
- Required Elements: string array
- Avoid: string array
- Structural Notes: string array or textarea

Do not include:

- title
- subtitle
- purpose
- reader_start
- reader_end
- front_matter_label
- front_matter_blurb

---

### 2. `concept_jurisdiction_json`

Purpose: control repetition and define what the chapter owns.

```json
{
  "owns": [],
  "may_reference": [],
  "must_not_reteach": [],
  "forbidden_phrases": []
}
```

UI fields:

- Owns: string array
- May Reference: string array
- Must Not Reteach: string array
- Forbidden Phrases: string array

Keep this field. It is genuinely useful and does not duplicate left-column fields.

---

### 3. `generation_directives_json`

Purpose: drafting behavior and output constraints.

```json
{
  "write_only_prose": true,
  "style_constraints": [],
  "format_constraints": [],
  "revision_notes": ""
}
```

UI fields:

- Write Only Prose: checkbox
- Style Constraints: string array
- Format Constraints: string array
- Revision Notes: textarea

Avoid duplicating `required_elements` and `avoid` here if those are already in `chapter_metadata_json`.

---

### 4. `arc_metadata_json`

Purpose: optional story, memoir, nonfiction argument, or transformation arc.

For nonfiction MVP:

```json
{
  "arc_phase": "",
  "beats": []
}
```

UI fields:

- Arc Phase: select
- Beats: string array

Suggested `arc_phase` options:

- setup
- escalation
- explanation
- demonstration
- complication
- synthesis
- resolution
- conclusion

For fiction/memoir later, this can expand to:

```json
{
  "arc_phase": "",
  "pov_character": "",
  "external_plot_movement": "",
  "relationship_shift": "",
  "beats": []
}
```

Do not use `internal_shift` if it merely duplicates `state_start` and `state_end`.

---

### 5. `media_prompts_json`

Purpose: image and diagram ideas.

Recommended shape:

```json
{
  "images": [
    {
      "placement": "",
      "prompt": "",
      "caption": ""
    }
  ],
  "diagrams": [
    {
      "placement": "",
      "description": "",
      "caption": ""
    }
  ]
}
```

UI fields:

- Image Ideas: object array
  - Placement: text/select
  - Prompt: textarea
  - Caption: text
- Diagram Ideas: object array
  - Placement: text/select
  - Description: textarea
  - Caption: text

---

## Book-level metadata shapes

These should use the same mini-form editor.

### 6. `publishing_metadata_json`

Purpose: front cover, back cover, sales copy, retail positioning, and publishing package.

Recommended shape:

```json
{
  "front_cover": {
    "cover_direction": "",
    "title_treatment": "",
    "visual_motifs": [],
    "avoid": []
  },
  "back_cover": {
    "headline": "",
    "blurb": "",
    "bullets": [],
    "reader_promise": "",
    "call_to_action": ""
  },
  "inside_cover": {
    "short_blurb": "",
    "author_note": "",
    "positioning_note": ""
  },
  "retail_metadata": {
    "categories": [],
    "keywords": [],
    "comparison_titles": [],
    "sales_angles": []
  }
}
```

UI sections:

#### Front Cover

- Cover Direction: textarea
- Title Treatment: text
- Visual Motifs: string array
- Avoid: string array

#### Back Cover

- Headline: text
- Blurb: textarea
- Bullets: string array
- Reader Promise: textarea
- Call To Action: text

#### Inside Cover

- Short Blurb: textarea
- Author Note: textarea
- Positioning Note: textarea

#### Retail Metadata

- Categories: string array
- Keywords: string array
- Comparison Titles: string array or object array later
- Sales Angles: string array

Do not duplicate project `title`, project `author_name`, book brief `subtitle`, or book brief `promise` unless used only as generated display text.

---

### 7. `book_architecture_json`

Purpose: whole-book structure, not individual chapter content.

Recommended shape:

```json
{
  "book_shape": "",
  "reader_journey": [],
  "major_sections": [],
  "recurring_features": [],
  "through_lines": [],
  "open_loops": []
}
```

UI fields:

- Book Shape: select or text
- Reader Journey: string array
- Major Sections: string array
- Recurring Features: string array
- Through Lines: string array
- Open Loops: string array

Suggested `book_shape` options:

- practical_guide
- argument_book
- memoir
- workbook
- essay_collection
- course_as_book
- fiction
- hybrid

---

### 8. `global_style_contract_json`

Purpose: book-wide voice, style, and formatting constraints.

Recommended shape:

```json
{
  "primary_voice": [],
  "avoid_voice": [],
  "sentence_style": [],
  "formatting_rules": [],
  "reader_relationship": "",
  "examples_policy": "",
  "jargon_policy": ""
}
```

UI fields:

- Primary Voice: string array
- Avoid Voice: string array
- Sentence Style: string array
- Formatting Rules: string array
- Reader Relationship: textarea
- Examples Policy: textarea
- Jargon Policy: textarea

---

## Required UI pattern

Every mini-form editor should have this structure:

```html
<div class="collapse collapse-arrow bg-base-200 border border-base-300">
  <input type="checkbox" />
  <div class="collapse-title font-semibold">
    Chapter Planning
    <span class="badge badge-sm">3 required</span>
  </div>
  <div class="collapse-content space-y-4">
    <!-- generated controls here -->

    <div class="collapse collapse-arrow bg-base-100 border border-base-300 mt-4">
      <input type="checkbox" />
      <div class="collapse-title text-xs font-semibold opacity-70">
        Advanced Raw JSON
      </div>
      <div class="collapse-content">
        <!-- raw JSON textarea here -->
      </div>
    </div>
  </div>
</div>
```

The collapsed title should show a short useful summary, for example:

- `3 required`
- `2 avoid`
- `2 owns`
- `1 image`
- `4 keywords`
- `cover notes`

---

## JavaScript implementation

Create a new file:

```text
internal/views/static/metadata_editor.js
```

If this project does not currently use that static path, use whatever existing static asset path is already configured. Keep the file separate from Go templates.

The JS should be framework-free.

No React.

No JSON schema library.

No generic JSON editor required for MVP.

### JS responsibilities

The JS must:

1. Find all containers with `data-metadata-editor`.
2. Read schema name from `data-schema`.
3. Read source textarea selector from `data-source`.
4. Parse the textarea JSON.
5. If JSON is blank or invalid, fall back to the schema default.
6. Render DaisyUI controls.
7. On every change, serialize the data back to the source textarea.
8. Support:
   - text
   - textarea
   - select
   - checkbox
   - list
   - objectList
9. Preserve unknown keys when possible, or show an Advanced Raw JSON drawer so the user can still access them.
10. Dispatch a `change` event on the source textarea after updates.

### Example HTML hook

```html
<textarea
  name="chapter_metadata_json"
  class="hidden"
  data-metadata-source="chapter_metadata_json">{{ .ChapterMetadataJSON }}</textarea>

<div
  data-metadata-editor
  data-schema="chapterPlanning"
  data-source="[data-metadata-source='chapter_metadata_json']">
</div>
```

For repeated chapter cards, make the source selector card-local. Do not accidentally bind all cards to the first textarea.

Recommended pattern:

```html
<div class="chapter-card" data-chapter-id="{{ .ID }}">
  <textarea
    name="chapter_metadata_json"
    class="hidden"
    data-metadata-source="chapter_metadata_json">{{ .ChapterMetadataJSON }}</textarea>

  <div
    data-metadata-editor
    data-schema="chapterPlanning"
    data-source="chapter_metadata_json">
  </div>
</div>
```

The JS should resolve `data-source="chapter_metadata_json"` by searching within the closest `.chapter-card` first.

For book-level forms, use a wrapper such as `.project-metadata-card`.

---

## Suggested JS schema registry

The JS file should contain a registry similar to this:

```js
const metadataSchemas = {
  chapterPlanning: {
    title: "Chapter Planning",
    defaultValue: {
      chapter_role: "",
      required_elements: [],
      avoid: [],
      structural_notes: []
    },
    fields: [
      {
        key: "chapter_role",
        label: "Chapter Role",
        type: "select",
        options: [
          "",
          "setup",
          "explanation",
          "case_study",
          "practice",
          "synthesis",
          "conclusion"
        ]
      },
      {
        key: "required_elements",
        label: "Required Elements",
        type: "list",
        placeholder: "Add something this chapter must include"
      },
      {
        key: "avoid",
        label: "Avoid",
        type: "list",
        placeholder: "Add something this chapter should avoid"
      },
      {
        key: "structural_notes",
        label: "Structural Notes",
        type: "list",
        placeholder: "Add a structural note"
      }
    ]
  }
}
```

Add equivalent schemas for:

- `conceptControl`
- `writingInstructions`
- `storyArc`
- `mediaIdeas`
- `publishingPackage`
- `bookArchitecture`
- `globalStyleContract`

---

## DaisyUI class guidance

Use DaisyUI v5 classes consistently:

### Containers

- `card bg-base-100 border border-base-300`
- `collapse collapse-arrow bg-base-200 border border-base-300`
- `collapse-content space-y-4`

### Inputs

- `input input-bordered input-sm w-full`
- `textarea textarea-bordered textarea-sm w-full`
- `select select-bordered select-sm w-full`
- `checkbox checkbox-sm`

### Buttons

- `btn btn-sm btn-outline`
- `btn btn-xs btn-ghost`
- `btn btn-xs btn-error btn-outline`

### Badges

- `badge badge-sm`
- `badge badge-outline`
- `badge badge-soft`

### Repeatable list item

```html
<div class="flex gap-2 items-center">
  <input class="input input-bordered input-sm flex-1" />
  <button type="button" class="btn btn-xs btn-ghost">✕</button>
</div>
```

### Object list item

```html
<div class="card bg-base-100 border border-base-300 p-3 space-y-2">
  <!-- object fields -->
  <button type="button" class="btn btn-xs btn-outline btn-error">Remove</button>
</div>
```

---

## Integration points

### 1. `internal/views/views.go`

Update the TOC/outline workspace template.

Where raw JSON textareas currently appear, keep the textareas but hide them and add metadata editor containers beneath the correct human label.

For chapter cards, add editors for:

- `chapter_metadata_json` → `chapterPlanning`
- `concept_jurisdiction_json` → `conceptControl`
- `generation_directives_json` → `writingInstructions`
- `arc_metadata_json` → `storyArc`
- `media_prompts_json` → `mediaIdeas`

Use collapsible panels so cards do not become visually overwhelming.

### 2. Book-level/project workspace

Where project-level metadata is edited, add editors for:

- `publishing_metadata_json` → `publishingPackage`
- `book_architecture_json` → `bookArchitecture`
- `global_style_contract_json` → `globalStyleContract`

This is where front cover, back cover, inside-cover blurb, retail keywords, book architecture, and style contract become pleasant to edit.

### 3. `internal/routes/routes.go`

Do not require major changes if forms already submit the JSON fields.

But verify that any route saving project metadata and chapter metadata accepts the hidden JSON textareas.

On save:

- continue using existing `cleanOptionalJSON`
- invalid JSON should return a visible error
- blank JSON should remain blank unless the UI writes default JSON

### 4. `internal/prompts/prompts.go`

Update prompt output contracts so LLM-generated metadata uses the simplified shapes above.

The LLM should not generate duplicated fields inside metadata JSON.

Example instruction:

```text
Do not repeat first-class chapter fields inside metadata JSON.
Title, subtitle, purpose, reader start, reader end, front matter label, and front matter blurb must remain outside JSON.
Metadata JSON should only contain control-layer information.
```

For TOC generation, prefer these metadata labels:

```text
Chapter Metadata JSON: {"chapter_role":"","required_elements":[],"avoid":[],"structural_notes":[]}
Arc Metadata JSON: {"arc_phase":"","beats":[]}
Concept Jurisdiction JSON: {"owns":[],"may_reference":[],"must_not_reteach":[],"forbidden_phrases":[]}
Generation Directives JSON: {"write_only_prose":true,"style_constraints":[],"format_constraints":[],"revision_notes":""}
Media Prompts JSON: {"images":[],"diagrams":[]}
```

### 5. Tests

Add or update tests to verify:

- the simplified metadata fields are present in generated TOC prompt contracts
- duplicated keys such as `title`, `purpose`, `reader_start`, `reader_end` are not requested inside JSON
- `cleanOptionalJSON` accepts the generated JSON shapes
- draft prompts still include metadata context
- blank metadata stays blank or becomes default only if the UI intentionally writes default JSON

---

## Required behavior

### User edits generated LLM values

After TOC generation:

1. LLM creates JSON values.
2. The UI shows them as mini-form fields.
3. User can add, remove, and edit values.
4. Hidden JSON field updates automatically.
5. User saves.
6. Server stores valid JSON.
7. Drafting pipeline uses updated JSON.

### No raw JSON by default

Raw JSON must not be the default visible editor.

Only show it under:

```text
Advanced Raw JSON
```

### Card summaries

Collapsed panels should show short summary badges.

Examples:

- Chapter Planning: `role: practice`, `3 required`, `2 avoid`
- Concept Control: `2 owns`, `1 blocked`
- Writing Instructions: `prose only`, `2 style`
- Media Ideas: `1 image`, `1 diagram`
- Publishing Package: `cover`, `back blurb`, `5 keywords`

---

## Acceptance criteria

### Chapter card acceptance

A user can open a TOC chapter card and edit:

- Chapter Role
- Required Elements
- Avoid
- Structural Notes
- Owns
- May Reference
- Must Not Reteach
- Forbidden Phrases
- Write Only Prose
- Style Constraints
- Format Constraints
- Revision Notes
- Arc Phase
- Beats
- Image Ideas
- Diagram Ideas

without seeing raw JSON.

When the card is saved, the correct JSON is stored in the existing chapter JSON fields.

### Book-level acceptance

A user can edit:

- Front Cover direction
- Title treatment
- Visual motifs
- Back Cover headline
- Back Cover blurb
- Back Cover bullets
- Inside Cover short blurb
- Retail categories
- Retail keywords
- Comparison titles
- Sales angles
- Book shape
- Reader journey
- Major sections
- Global voice
- Voice to avoid
- Formatting rules

without seeing raw JSON.

When the project is saved, the correct JSON is stored in the existing project JSON fields.

### Advanced acceptance

A user can open Advanced Raw JSON, inspect/edit the JSON, and the mini-form refreshes or at least saves valid JSON.

Invalid JSON should show an error and should not silently corrupt stored metadata.

---

## Implementation priority

### Phase 1: reusable editor component

Create `metadata_editor.js`.

Support:

- text
- textarea
- select
- checkbox
- list
- objectList

Create schema registry.

Render controls with DaisyUI classes.

### Phase 2: chapter card integration

Wire editor into TOC/chapter cards.

Hide raw JSON fields by default.

Add advanced raw JSON drawer.

### Phase 3: book-level integration

Wire editor into project/book metadata area.

Add publishing package, book architecture, and global style contract editors.

### Phase 4: prompt/schema cleanup

Update LLM output contracts to generate simplified JSON shapes.

Remove duplicated metadata keys.

### Phase 5: polish

Add summary badges.

Add better empty states.

Add optional reset-to-default button.

Add optional “regenerate this metadata section” button later.

---

## Important product note

This editor is not merely a technical convenience.

It turns the app from:

```text
a form with ugly JSON blobs
```

into:

```text
a book cockpit where the author edits the levers that shape the book
```

That is a premium feature.

The JSON is still there, but it becomes plumbing.

The user edits meaning.
