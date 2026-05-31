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
- Local LLM adapter driven by environment variables:
  - `LOCAL_LLM_URL`
  - `LOCAL_LLM_MODEL`
  - `LLM_MOCK`

### Collections

- `projects`
- `intake_responses`
- `book_briefs`
- `chapters`
- `jobs`

### User Flow

- New project creation.
- Project selection from the header dropdown.
- Fiction/nonfiction intake toggle.
- Intake persistence to `intake_responses`.
- Basic book-jacket style workspace placeholder.
- Book Brief generation as its own visible stage.
- Outline/table-of-contents chapter shell generation after the Book Brief exists.
- Outline/table-of-contents results render as a reviewable chapter-shell list with the Book Brief still accessible in the workspace.
- The center workspace uses tabs for available project artifacts: Book Brief, Outline, and Drafting.
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

### Book Metadata And Export System

- Ebook metadata.
- Cover metadata.
- Blurbs.
- Keywords and categories.
- Formatting requirements.
- Typography and color system.
- Export profiles.
- ISBN/publisher fields.
- Front matter and back matter management.

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
