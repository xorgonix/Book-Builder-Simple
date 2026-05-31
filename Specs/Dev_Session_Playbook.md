# Dev Session Playbook

This document exists so future development sessions start with a known target and end with a visible result. It is especially important for recorded or public build sessions, where unexplained scope changes can confuse viewers and erode trust in the product direction.

## 1. Purpose

Before any development effort begins, define what the work is supposed to produce.

Every dev session should answer:

- What user-facing result should exist when this is done?
- What spec or roadmap item is being advanced?
- What is explicitly out of scope?
- What should still work after the change?
- How will we prove the work is complete?
- What should be visible or explainable to viewers if this is recorded?

## 2. Required Session Header

Create or state this header before implementation starts:

```md
## Dev Session: <short name>

Date:
Project:
Audience:
Primary goal:
User-facing result:
Spec source:
Current build baseline:
In scope:
Out of scope:
Must preserve:
Acceptance checks:
Viewer-safe explanation:
Known risks:
```

## 3. Audience Modes

### Private Development

Use this when the goal is fast iteration.

Expectations:

- Short notes are acceptable.
- Rough UI is acceptable if marked as temporary.
- Hidden implementation details can be explored freely.
- The final result still needs acceptance checks.

### Recorded Development

Use this when the work may be shown to YouTube viewers or other public audiences.

Expectations:

- Start by saying what the feature should do.
- Explain what is prototype, what is production direction, and what is deliberately deferred.
- Avoid unexplained removals of previously stated product goals.
- Avoid showing secrets, local keys, private notes, or sensitive project content.
- Keep terminal noise understandable.
- Prefer deterministic test data where possible.
- End by showing what changed and what still remains.

### Demo Project Development

Use this when building or testing with a separate sample project.

Expectations:

- The demo project should have fake but realistic content.
- The demo should not depend on private books, private research, or unpublished user material.
- The session should identify which behaviors are app features and which are sample content.

## 4. Definition Of Done

A development effort is not done just because code compiles.

Minimum done criteria:

- The intended user-facing result exists.
- Existing core flow still works.
- The current build spec is updated if behavior changed.
- The roadmap is updated if scope was deferred or newly discovered.
- Any missing feature is labeled, not silently dropped.
- The verification command or manual test path is recorded.

For UI work:

- The user can tell what to do next.
- Long-running work does not look frozen.
- The interface does not flicker while idle.
- The app does not imply production-readiness for features that are still prototype-only.

For AI pipeline work:

- The user can see important intermediate outputs.
- The user can choose whether to act on reviewer/editor feedback.
- The user can accept, edit, reject, or regenerate important generated artifacts before the next stage uses them.
- The app does not silently treat generated text as user-approved structure.
- Autopilot behavior is clearly optional.
- Stage outputs are not overwritten without a recovery or versioning plan.

For web-production work:

- Auth and ownership boundaries are explicit.
- Job limits and failure handling are considered.
- Public deployment assumptions are documented.

## 5. Change Ledger Rule

Each dev effort must leave a trail in one of these places:

- `Specs/Spec_CurrentBuild.md` for what now exists.
- `Specs/Spec_ProductRoadmap.md` for what is still planned or deferred.
- The relevant module spec when a technical contract changes.

Do not remove a requirement from the product vision during implementation. If a requirement is no longer wanted, mark it `Rejected` with a reason.

## 6. Recommended Recorded Session Flow

Use this flow for viewer-friendly builds:

1. State the feature outcome in plain language.
2. Show the current behavior briefly.
3. Name the spec or roadmap item being advanced.
4. Implement the smallest coherent slice.
5. Run the acceptance checks.
6. Show the result in the app.
7. Update the spec ledger.
8. State what remains deferred.

## 7. Example Session Header

```md
## Dev Session: Add Reviewer Pass Ledger

Date: 2026-05-31
Project: Book Prototype Architect
Audience: Recorded development / YouTube
Primary goal: Let users run multiple named editorial reviews against the same chapter version.
User-facing result: A chapter can show separate developmental, continuity, and fact-check review outputs.
Spec source: Spec_ProductRoadmap.md section 4.
Current build baseline: One generic editorial diagnosis field on chapters.
In scope: Data model, routes, minimal cockpit UI, one repeat-review action.
Out of scope: Full accept/reject recommendation workflow, visual diffing, billing.
Must preserve: Existing draft/diagnose/rewrite/polish flow.
Acceptance checks: Create a chapter, run two reviewer types, confirm both reviews remain visible and attached to the same version.
Viewer-safe explanation: This adds transparency so AI critique is inspectable instead of silently folded into rewrites.
Known risks: Requires later versioning cleanup before public multi-user launch.
```
