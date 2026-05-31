````
# Module 2: Asynchronous Routing & Background Processing Engine

## 1. PocketBase v0.38 Hook Registration
All custom endpoints must be registered within the native `OnServe` event lifecycle hook using the modern core routing architecture. The Go application router must pass validated requests to the handler layer, update tracking states in SQLite immediately, and hand off heavy linguistic computations to detached background workers.

## 2. Definitive Route Map

### A. View & Configuration Endpoints
* `GET /` — Root route. Automatically detects active session parameters, reads the user's last modified book project via `ui_state`, and issues a 302 redirect to `/app/project/{id}/render`.
* `POST /api/projects` — Instantiates an empty project record shell with default layout configurations. Returns the full onboarding UI shell.
* `POST /api/project/{id}/intake` — Parses and validates configuration metrics (`core_topic`, `target_audience`, `reader_hunger`, `prohibited_directions`, `selected_tone`). Maps values to individual rows in the `intake_responses` table. Sets `target_length` and `target_chapters` on the parent project row. Returns an HTMX redirection to render the Book Brief staging screen.
* `POST /api/project/{id}/escape-hatch?field={fieldName}` — Receives an empty form target parameter. Fires a high-speed, low-token LLM call to generate three contextual, creative suggestions based on current project notes. Returns a localized Tailwind selection snippet that instantly populates the targeted input container.

### B. Dual-Speed Processing Endpoints
To support both fast prototype generation and granular step-by-step editing, the routing engine exposes both Autopilot and Manual cockpit handlers:

* `POST /api/project/{id}/generate/autopilot` — **The Fast Track Route**. Instantly creates a global project tracking record inside the `jobs` table marked as `running`. Returns a 202 status code to shift the UI to polling mode, then triggers a background loop worker that processes all approved chapters sequentially through all 4 prompt pipeline passes.
* `POST /api/project/{id}/chapters/{chapterId}/pipeline/draft` — Cockpit Stage 1. Triggers initial drafting text generation. Saves to `chapters.raw_draft`.
* `POST /api/project/{id}/chapters/{chapterId}/pipeline/diagnose` — Cockpit Stage 2. Triggers structural editor diagnostic analysis. Saves to `chapters.editorial_diagnosis`.
* `POST /api/project/{id}/chapters/{chapterId}/pipeline/rewrite` — Cockpit Stage 3. Processes rewrite execution combining raw text, diagnosis data, and user's manual override notes. Saves to `chapters.targeted_rewrite`.
* `POST /api/project/{id}/chapters/{chapterId}/pipeline/polish` — Cockpit Stage 4. Performs structural pattern cleanup. Saves final polished output to `chapters.draft_content` and marks status as `completed`.

## 3. Concurrency & Context Isolation Mechanics
Long-running LLM API updates must never block the main web server thread. The Go backend must cleanly detach the background job lifecycle from the incoming HTTP request pool. Passing an active request context (`*core.RequestEvent`) straight into a long-running background goroutine is strictly prohibited.

```go
func handlePipelineStage(app *pocketbase.PocketBase) echo.HandlerFunc {
    return func(c echo.Context) error {
        projectId := c.PathParam("id")
        chapterId := c.PathParam("chapterId")
        stageTarget := c.PathParam("stage") // "draft" | "diagnose" | "rewrite" | "polish"

        // 1. Defensively verify entity matching before initiating workers
        chapter, err := app.FindRecordById("chapters", chapterId)
        if err != nil || chapter.GetString("project_id") != projectId {
            return c.String(404, "Target entity validation match failed.")
        }

        // 2. Create tracking log entry inside the jobs table
        jobRecord := core.NewRecord(app.FindCollectionByNameOrId("jobs"))
        jobRecord.Set("project_id", projectId)
        jobRecord.Set("chapter_id", chapterId)
        jobRecord.Set("job_type", stageTarget)
        jobRecord.Set("status", "running")
        app.Save(jobRecord)

        // 3. Spawning the Detached Context Job Runner
        go func(pId string, cId string, jobId string, target string) {
            // Instantiate an isolated background context independent of client connection drops
            bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
            defer cancel()

            // Execute the specific, scoped prompt task using thread-safe app instances
            err := executeOrchestratedPipelineStep(bgCtx, app, pId, cId, target)
            
            // Re-fetch job record in thread-safe context to write completion milestones
            job, _ := app.FindRecordById("jobs", jobId)
            if err != nil {
                job.Set("status", "failed")
                job.Set("error_msg", err.Error())
                updateChapterStatus(app, cId, "failed")
            } else {
                job.Set("status", "completed")
                updateChapterStatus(app, cId, mapStageToStatus(target))
            }
            app.Save(job)
        }(projectId, chapterId, jobRecord.Id, stageTarget)

        // 4. Instantly return HTTP 202 Accepted status accompanied by the visual loader
        return c.Render(202, "chapter-cockpit-processing", map[string]any{
            "ChapterID": chapterId,
            "Stage":     stageTarget,
        })
    }
}


## 4. Adaptive High-Fidelity Polling Engine

- **Endpoint Target:** `GET /api/project/{id}/status`
    
- **Behavior Matrix:** The route checks the active status fields of the project and its child chapters:
    
    - If any active background jobs are flagged as `running`, return an **HTTP 200 OK** containing a tailored loading component that reflects the exact step in progress. The HTMX poll directive continues executing normally.
        
    - Once the worker safely writes the output and updates the status to a terminal state (`brief`, `toc`, or a chapter milestone like `completed`), the route returns an **HTTP 286 Change Target** response header coupled with an `HX-Refresh: true` command. HTMX stops polling immediately and triggers a clean layout rehydration.
        

````
