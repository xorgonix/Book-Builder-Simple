# Product Roadmap And Deferred Requirements

This document captures planned product requirements that are not fully present in the current runnable build. These items are part of the product direction unless later marked `Rejected`.

## 1. Web Production Foundation

Status: **Deferred**

The app must become a web-capable multi-user product, not only a local prototype.

Required capabilities:

- User login and session handling.
- Project ownership by user.
- Authorization checks on every project, chapter, job, research, review, and asset route.
- Safe production configuration for the LLM endpoint.
- Per-user rate limits.
- Per-user active job caps.
- Admin visibility into failed jobs.
- Clear recovery path for failed generation.

Default direction:

- Use PocketBase auth collections for users.
- Add `owner_id` or equivalent relation to project-owned collections.
- Keep local single-user mode available for development.

## 2. Job Processing And Live Updates

Status: **Partially implemented**

The current build has job records and background goroutines. The target system needs production-grade job control.

Required capabilities:

- Replace unbounded goroutine spawning with a bounded worker queue.
- Store queued, running, completed, failed, cancelled, and retryable states.
- Enforce per-user and global concurrency limits.
- Use polling only while a job is active.
- Add slower/backoff polling where appropriate.
- Prefer SSE for job status updates when moving beyond HTMX polling.
- Keep WebSockets optional unless bidirectional live collaboration becomes necessary.

Default direction:

- First upgrade to a bounded queue.
- Keep active-job HTMX polling as the simple default.
- Add SSE later if polling becomes noisy at real traffic levels.

## 3. Human-In-The-Loop Manuscript Pipeline

Status: **Partially implemented**

The product must not become a black-box book generator. Users must be able to inspect, skip, repeat, and override each major step.

Required workflow stages:

- Book brief.
- Table of contents and chapter shells.
- Draft.
- Diagnosis.
- Rewrite.
- Polish.

User controls:

- Run a stage manually.
- Skip a stage knowingly.
- Re-run a stage.
- Add notes before re-running.
- View the editor or reviewer complaint before deciding whether to act on it.
- Preserve prior outputs when a stage is re-run.

Default direction:

- Manual cockpit remains the primary trusted workflow.
- Autopilot remains a shortcut for users who knowingly want to skip review.

## 4. Versions And Reviewer Passes

Status: **Deferred**

The system should support multiple editors and reviewers reviewing the same version of a document.

Required concepts:

- `chapter_versions`: immutable snapshots of chapter text at meaningful points.
- `chapter_reviews`: one collection for all review passes, with a `reviewer_type` field.
- Reviewer types should be data values, not separate collections.

Recommended reviewer types:

- `developmental_editor`
- `continuity_editor`
- `line_editor`
- `copy_editor`
- `fact_checker`
- `market_positioning_reviewer`

Default behavior:

- Multiple reviewers should review the same selected chapter version by default.
- A rewrite creates a new chapter version.
- Reviews should remain attached to the exact version they evaluated.
- The user can choose which review recommendations influence the next rewrite.

## 5. Research Module

Status: **Not yet designed**

The app should support research requests that create source-backed material for chapters.

Required capabilities:

- Accept a user query for a single chapter or the whole book.
- Generate research tasks from the table of contents.
- Batch research across chapters to avoid duplicate searches.
- Capture source URLs, titles, retrieved excerpts or summaries, and timestamps.
- Extract claims, examples, definitions, statistics, counterarguments, and open questions.
- Assign research findings to one or more chapters.
- Mark research confidence and source quality.

Default direction:

- Store raw source records separately from synthesized research notes.
- Allow one finding to be linked to multiple chapters.
- Do not inject research into prose automatically without user visibility.

## 6. Images And Visual Assets

Status: **Not yet designed**

The app should eventually manage chapter images, cover images, and image-generation prompts.

Required capabilities:

- Store image prompts.
- Store image URLs.
- Store generated asset references.
- Associate assets with chapters, cover concepts, or marketing material.
- Support external generators such as ComfyUI, Stable Diffusion, Qwen image tools, or hosted image APIs.

Default direction:

- Begin with metadata and prompt storage.
- Add generation integrations after the asset model is stable.

## 7. Book Metadata And Publishing Package

Status: **Not yet designed**

The app should capture the non-manuscript data needed for an ebook or book package.

Required metadata:

- Title and subtitle.
- Author name or pen name.
- Blurb and short description.
- Categories and keywords.
- ISBN or publisher fields when relevant.
- Cover image or cover prompt.
- Color palette.
- Font choices.
- Formatting requirements.
- Front matter and back matter.
- Export target such as ebook, PDF, web serial, print draft, or proposal package.

Default direction:

- Treat metadata as project-level publishing data.
- Keep manuscript content, research, reviews, and assets separate but linkable.

## 8. Roadmap Governance

When a requirement is postponed, record it here rather than deleting it from the master vision.

Allowed status labels:

- `Implemented`
- `Partially implemented`
- `Deferred`
- `Not yet designed`
- `Rejected`

Rejected items must include a short reason and date.

