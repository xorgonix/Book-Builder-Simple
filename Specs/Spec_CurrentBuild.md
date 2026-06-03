# Current Build Reality: Book Prototype Architect

This document records what the runnable application currently does. It is the implementation ledger, not the product vision. If this file disagrees with `Spec_Master.md`, treat this file as the current-state truth and `Spec_Master.md` as the target-state contract.

## 1. Current Status

The app is a first runnable vertical slice of the Book Prototype Architect. It supports creating projects, collecting intake, generating a Book Brief, generating outline/table-of-contents chapter shells from that approved brief, and running a chapter drafting pipeline through draft, diagnosis, rewrite, and polish passes.

Current classification: **local-first prototype with partial production architecture**.

## 2. Implemented

### Runtime Stack

- Go application embedding PocketBase.
- PocketBase collections created through Go migrations.
- HTMX-rendered server views using Tailwind and DaisyUI from CDN.
- Local LLM adapter driven by visible local config in `bookbuilder.config`.
- Environment variables remain optional overrides for runtime config:
  - `LOCAL_LLM_URL`
  - `LOCAL_LLM_MODEL`
  - `LLM_MOCK`

### Collections

- `projects`
- `intake_responses`
- `book_briefs`
- `chapters`
- `jobs`
- Existing databases are patched by a follow-up migration so `intake_responses.key` accepts the current setup keys, including `book_form`, `narrative_pov`, and `structure_model`.
- Existing databases are patched by a follow-up migration so chapters can store optional export metadata: `subtitle`, `front_matter_label`, and `front_matter_blurb`.
- Existing databases are patched by a follow-up migration so projects and chapters can store the first metadata control layer:
  - project `author_name`, `publishing_metadata_json`, `book_architecture_json`, and `global_style_contract_json`
  - chapter `chapter_metadata_json`, `arc_metadata_json`, `concept_jurisdiction_json`, `generation_directives_json`, and `media_prompts_json`

### User Flow

- New project creation.
- Project selection from the header dropdown.
- Named book creation from the Books panel.
- Project setup form includes an explicit editable book title.
- Project setup saves show an in-button spinner while saving and a success confirmation after the save completes.
- Fiction/nonfiction intake toggle.
- Intake persistence to `intake_responses`.
- Saved intake values rehydrate into the setup form when the project is reopened.
- Basic book-jacket style workspace placeholder.
- Book Brief generation as its own visible stage.
- Book Brief can be edited and saved after generation.
- Book Brief saves show an in-button spinner while saving and a success confirmation after the save completes.
- Saved Book Brief values are used by outline generation, chapter drafting, editing, polishing, and export.
- Saved metadata is included in chapter draft, diagnosis, rewrite, and polish prompt context without changing the configured local model.
- Brief, Outline, and Auto actions include the current Book Setup form values, so the chapter-count field is saved before generation starts.
- Regenerating the Book Brief returns the workspace to the Book Brief tab and clears stale empty outline shells so old chapters do not make the UI look like Outline ran.
- Outline/table-of-contents chapter shell generation after the Book Brief exists.
- Outline/table-of-contents results render as a reviewable chapter-shell list with the Book Brief still accessible in the workspace.
- If the saved chapter target changes before drafting begins, regenerating the outline replaces the old empty chapter shells with the new target count.
- If drafted chapter content already exists, the app refuses to silently replace the outline with a different chapter count.
- The center workspace uses tabs for available project artifacts: Book Brief, Outline, and Drafting.
- The center workspace includes a Metadata tab for editing book-level and selected chapter-level control metadata.
- Metadata saves show an in-button spinner while saving and a success confirmation after the save completes.
- Chapter cockpit with visible tabs for:
  - raw draft
  - editorial diagnosis
  - targeted revision
  - polished manuscript
- Manual per-chapter pipeline buttons:
  - draft
  - diagnose
  - revise
  - polish
- Autopilot route that can run the full pipeline.
- Export Book control that produces local Markdown, rendered HTML preview, EPUB, lint report, and style report files under `exports/`.
- Download/open links for the generated EPUB, HTML preview, Markdown source, lint report, and style report.

### Async Behavior

- Long-running LLM jobs are launched in goroutines.
- Job records are created in the `jobs` collection.
- The visible processing component polls while a job is active.
- Idle right-panel polling was removed to avoid screen flicker and terminal noise.
- Status polling now treats stale `running` jobs as failed and refreshes completed project stages instead of letting stale job rows mask saved outputs.
- TOC parsing tolerates common LLM formatting drift where the chapter title appears after or below the `Order:` value.

## 3. Partially Implemented

### UI State

- `projects.ui_state` exists and has a default schema.
- Full rehydration of exact tabs, scroll positions, expanded sections, and focus targets is not yet complete.

### Escape Hatches

- A working escape hatch exists for `reader_hunger`.
- The master rule says every open-ended field should have an escape hatch. That is not yet implemented.

### Human-In-The-Loop Editing

- The Book Brief now appears as a reviewable workspace stage before chapter shells are generated.
- The Outline & Chapter Shells action requires an existing Book Brief.
- The generated chapter shells now appear as an outline review screen instead of immediately replacing the workspace with a single chapter cockpit.
- Users can inspect stage outputs and provide rewrite notes from the diagnosis tab.
- Full accept/edit/reject/regenerate controls do not yet exist at every stage.
- The app does not yet distinguish AI-generated artifacts from user-approved artifacts.
- Users cannot yet maintain a durable editorial decision log.
- Users cannot yet choose among multiple reviewer types.
- Users cannot yet ask the same reviewer to re-review the same manuscript version as a stored separate pass.

### Background Processing

- Jobs are tracked in the database.
- Execution is not yet handled by a bounded worker queue.
- There are no production-grade per-user caps, backoff rules, cancellation controls, or durable retry policies.

### Exports

- First-pass book export exists.
- Markdown, rendered HTML preview, EPUB, JSON lint report, and JSON style report files are generated synchronously from the current chapter records.
- Export cleanup is deterministic and does not regenerate prose.
- Export cleanup does not write cleaned manuscript text back into PocketBase.
- Export cleanup uses PocketBase chapter order and chapter title fields as canonical truth.
- Export cleanup strips duplicate generated chapter headings only from the beginning of chapter prose.
- Export cleanup normalizes Markdown spacing and obvious simple list formatting.
- Export cleanup detects large duplicated blocks across chapters and writes them to the lint report.
- Export linting validates optional chapter metadata JSON fields without failing the export.
- Export linting reports concept-jurisdiction forbidden phrase matches when chapter metadata defines them.
- Export linting reports repeated stock/watchlist phrases.
- Export style reporting records basic per-chapter metrics including word count, sentence length, paragraph length, question count, em dash count, bullet count, repeated sentence openers, and watchlist phrase matches.
- Chapter body fallback order is:
  - `draft_content`
  - `targeted_rewrite`
  - `raw_draft`
- Export files are written to local disk under `exports/`.
- Lint report files are written under `exports/reports/`.
- Style report files are written under `exports/reports/`.
- Export uses saved `author_name` before falling back to `book_author` in `bookbuilder.config`.
- `BOOK_AUTHOR` remains an optional environment override for the config fallback.
- Export Book refuses to run while a project job is still active and shows the current running job/progress instead.
- Export Book blocks by default if any chapter has no manuscript text.
- The user can explicitly choose to export an incomplete draft anyway.
- Export history is not yet stored in an `exports` collection.
- EPUB output is intentionally simple and reflowable.
- HTML preview output is intended as the primary browser reading surface; Markdown remains the clean source/archive output.
- Optional chapter metadata fields can inject front matter into exports when present.
- Image/media prompt metadata can be stored. Export can render prompt-only media placeholders when that option is enabled, but no image generation backend is implemented in this build.
- The default local model remains `qwen/qwen3.5-9b`; model roles are prompt roles, not provider/model switches.
- Cover images, ISBNs, publisher metadata, DOCX, and PDF are not yet implemented.

## 4. Not Implemented Yet

### Web Production Requirements

- User login.
- Project ownership by authenticated user.
- Multi-user authorization boundaries.
- Production deployment configuration.
- Per-user rate limits and job caps.
- Durable worker queue.
- SSE or WebSocket update stream.

### Editorial System

- Multiple reviewer roles:
  - developmental editor
  - continuity editor
  - line editor
  - copy editor
  - fact checker
  - sensitivity or market-positioning reviewer
- Repeatable review passes against the same manuscript version.
- Review comparison view.
- Accept/reject workflow for reviewer recommendations.
- Durable version snapshots.

### Research System

- Chapter or book-level research queries.
- Batch research across all chapters.
- Search de-duplication across overlapping chapter needs.
- Source capture, citation tracking, confidence flags, and claim extraction.
- Research-to-chapter assignment workflow.

### Image And Asset System

- Image prompt storage.
- Image URL storage.
- Generated asset records.
- Cover image prompts and generated covers.
- Chapter image planning.
- Integration with image backends such as ComfyUI, Stable Diffusion, Qwen image tooling, or remote image APIs.

### Book Metadata

- Ebook metadata.
- Cover metadata.
- Blurbs.
- Keywords and categories.
- Formatting requirements.
- Typography and color system.
- ISBN/publisher fields.
- Full front matter and back matter management.

## 5. Current User Guidance

The current app is ready for exploratory local testing of the core generation pipeline. It is not yet ready as a public web application.

Recommended test path:

1. Create or select a project.
2. Choose fiction or nonfiction.
3. Fill the intake fields.
4. Save intake.
5. Generate the Book Brief.
6. Review the Book Brief.
7. Generate the outline and chapter shells.
8. Use the chapter cockpit to run one chapter manually through draft, diagnose, revise, and polish.
9. Use autopilot only when intentionally skipping intermediate review.

## 6. Spec Drift Rule

No feature from `Spec_Master.md` should be considered removed unless a spec document explicitly labels it `Rejected`.

Use these labels in future spec work:

- `Implemented`
- `Partially implemented`
- `Deferred`
- `Not yet designed`
- `Rejected`
