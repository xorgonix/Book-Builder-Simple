````
# Specification Master: Book Prototype Architect
# Core Environment & Architecture Blueprint (v1.2 Target Vision)

## 0. Spec Authority And Implementation Ledger

This document describes the target product vision and architectural contracts. It is not, by itself, a claim that every listed feature exists in the current runnable application.

Current implementation truth lives in:

- `Specs/Spec_CurrentBuild.md`

Deferred and future-facing requirements live in:

- `Specs/Spec_ProductRoadmap.md`

Development-session planning and acceptance expectations live in:

- `Specs/Dev_Session_Playbook.md`

No requirement should be treated as removed unless it is explicitly marked `Rejected` in a spec document. When implementation work ships only a first runnable slice, the difference between the target vision and the implemented slice must be recorded in `Spec_CurrentBuild.md` or `Spec_ProductRoadmap.md`.

Allowed requirement status labels:

- `Implemented`
- `Partially implemented`
- `Deferred`
- `Not yet designed`
- `Rejected`

## 1. Stack Version Constraints
* **Backend Language:** Go 1.25+ (Standard library preference wherever applicable)
* **Framework Engine:** PocketBase v0.38 Core Library Embedded (SQLite)
* **Frontend Engine:** HTMX v2.x (Injected via clean CDN, no local compilation pipeline)
* **UI Layout System:** Tailwind CSS 4 + DaisyUI v5 (Injected via CDN utility web hooks)

## 2. Core UX & Psychological Directives
To maximize user momentum, eliminate choice paralysis, and cultivate deep product ownership, the AI Coding Agent must enforce these rules across all layout segments:

### A. The Goal-Gradient Momentum Pattern
* Never display a 0% completion state to the end user.
* The progress tracking bar (`.steps` utility in DaisyUI v5) must remain visible across the master viewport layout.
* Moving past Step 1 (Fiction/Nonfiction toggle) must trigger an immediate visual swap, establishing user momentum within 3 seconds of opening the application.

### B. Anti-Anxiety "Escape Hatch" Inputs
* Every open-ended text input area must be paired with an auxiliary "Inspire Me" or "Surprise Me" button.
* Clicking an Escape Hatch must execute a scoped, fast JSON query to a minor LLM route that fills the text input with high-quality contextual sample data to prevent empty-canvas block.

### C. The Endowment Effect (Visual Book Jackets & Interactive Cockpits)
* The center workspace panel is not a software form; it is a live manuscript.
* Once a Working Title is captured or generated, it must be rendered inside a visually polished card component that mimics a physical book jacket.
* **Anti-Monolithic Drafting Rule:** The compilation of long-form chapter blocks must never be a single black-box execution step. The user must be given clear visibility into the distinct text transformation phases (Drafting, Diagnosis, Targeted Revision, and Polishing).

### D. Human Override At Every Stage
* Every generated artifact must be inspectable before it becomes the basis for the next major stage.
* The user must be able to accept, edit, reject, or regenerate outputs at each stage: intake, Book Brief, outline, table of contents/chapter shells, draft, diagnosis, rewrite, polish, research, images, and publishing metadata.
* AI assistance may suggest, diagnose, rewrite, or fill gaps, but it must not silently finalize decisions that affect downstream manuscript structure.
* Regeneration must preserve prior user decisions or clearly identify what will be replaced.
* Autopilot must remain an explicit shortcut mode and must never be the default path for a project.

## 3. Dynamic UI State Rehydration
* Users must be able to switch between distinct book projects seamlessly using a navigation header dropdown.
* The application must use a dedicated `ui_state` JSON block saved directly on each `projects` row.
* When a user alters their active project, HTMX must retrieve `/app/project/{id}/render` to instantly restore the panels, focus areas, target lengths, and view tabs exactly as the user left them.

## 4. Authoritative Runtime Contracts

### A. Canonical Project Status Enum
* `intake`: Project exists; user is configuring core intent, target book length, and chapter count targets.
* `processing`: An asynchronous background generation or multi-pass revision job is running.
* `brief`: The initial project brief and core style criteria have completed successfully.
* `toc`: The structural table of contents/chapter outline has completed successfully.
* `drafting`: One or more chapters are actively being generated, editorially diagnosed, or revised.
* `error_failed_job`: A processing pass failed; the UI must present a clear, recoverable error state with a retry option.

### B. Canonical Chapter Status Enum
* `pending`: Chapter shell exists but has not been approved for generation.
* `card_approved`: Chapter card structure is accepted and ready for generation.
* `drafting_stage_1`: AI is running the initial text draft generation routine.
* `drafting_stage_2`: Chapter draft is complete; AI has attached the editorial structural diagnosis payload.
* `drafting_stage_3`: Re-evaluation pass complete; the targeted structural rewrite text is ready for inspection.
* `completed`: The pattern cleanup pass is finished; the polished final draft text is ready for user review and export.

### C. Project UI State Schema
The `ui_state` field is a JSON document stored on the owning `projects` row.
```json
{
    "version": 1,
    "active_view": "intake",
    "active_panel": "workspace",
    "active_tab": "brief",
    "target_book_length": "practical_ebook", 
    "target_chapter_count": 10,
    "expanded_sections": ["book-variables", "generation-pack", "editorial-cockpit"],
    "scroll_positions": {
        "left": 0,
        "center": 0,
        "right": 0
    },
    "focus_target": "working-title",
    "polling": {
        "enabled": false,
        "endpoint": "/api/project/{id}/status"
    }
}
```


_Valid `target_book_length` options:_ `short_guide` (~5,000 words), `practical_ebook` (~15,000 words), `full_prototype` (~40,000 words).

### D. Route Ownership Boundaries

- `/app/...` routes return full layout views or responsive HTML fragments intended for direct HTMX swaps.
    
- `/api/...` routes mutate internal database states, trigger background workers, or return status snippets for asynchronous operations.
    
- Background job routes must never block. They must instantly return an HTTP 202 Accepted acknowledgement and shift execution updates to the high-fidelity polling loop.
    

## 5. Modern PocketBase Routing Idioms

All custom HTTP endpoints must conform strictly to the modern syntax structure:

Go

```
app.OnServe().BindFunc(func(se *core.ServeEvent) error {
    se.Router.GET("/app/project/{id}", func(re *core.RequestEvent) error {
        id := re.Request.PathValue("id")
        return re.String(200, "Project Payload: " + id)
    })
    return se.Next()
})
```
````
