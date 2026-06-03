---

## SECTION 4: MODULE_3_VIEWS.md

```html
<!DOCTYPE html>
<html lang="en" data-theme="cupcake" class="h-full">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Book Prototype Architect</title>
    <link href="https://cdn.jsdelivr.net/npm/daisyui@5.0.0-beta.x/dist/full.min.css" rel="stylesheet" type="text/css" />
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@2.0.0"></script>
</head>
<body class="bg-base-300 h-screen overflow-hidden flex flex-col font-sans select-none">
    
    <header class="navbar bg-base-100 border-b border-base-300 px-4 shrink-0 flex justify-between items-center h-16">
        <div class="flex items-center gap-3">
            <span class="text-xl font-black tracking-wider text-primary">📚 SOVEREIGN.ENGINE</span>
            <div id="project-header-badge" class="badge badge-neutral font-mono font-bold">MODE: DETERMINISTIC</div>
        </div>
        <div class="flex items-center gap-2">
            <select name="project_id" class="select select-bordered select-sm w-48 font-semibold"
                    hx-get="/app/project/select" hx-target="#main-layout-grid" hx-swap="outerHTML">
                {{ range .AllProjects }}
                <option value="{{ .Id }}" {{ if eq .Id $.ActiveProjectID }}selected{{ end }}>{{ .Title }}</option>
                {{ end }}
            </select>
            <button class="btn btn-sm btn-primary font-bold shadow" hx-post="/api/projects" hx-target="#main-layout-grid" hx-swap="outerHTML">+ New Book</button>
        </div>
    </header>

    <div class="grid grid-cols-12 flex-1 overflow-hidden h-[calc(100vh-4rem)]" id="main-layout-grid">
        
        <aside class="col-span-12 lg:col-span-3 border-r border-base-300 bg-base-100 p-4 overflow-y-auto flex flex-col gap-4 h-full">
            <form hx-post="/api/project/{{ .Project.Id }}/intake" hx-target="#main-layout-grid" hx-swap="outerHTML" class="flex flex-col gap-4">
                <ul class="steps steps-horizontal w-full text-xs font-bold mb-2">
                    <li class="step step-primary">Type</li>
                    <li class="step step-primary">Intent</li>
                    <li class="step">Blueprint</li>
                    <li class="step">Drafting</li>
                </ul>

                <div class="bg-base-200 p-3 rounded-xl border border-base-300 flex flex-col gap-3">
                    <span class="text-xs font-bold uppercase tracking-wider text-base-content/60">Manuscript Target Scale</span>
                    <div class="form-control">
                        <label class="label-text mb-1 font-semibold">Target Length</label>
                        <select name="target_length" class="select select-bordered select-sm w-full bg-base-100">
                            <option value="short_guide">Short Guide (~5k words / 5 Chapters)</option>
                            <option value="practical_ebook" selected>Practical eBook (~15k words / 10 Chapters)</option>
                            <option value="full_prototype">Full-Length Prototype (~40k words / 15 Chapters)</option>
                        </select>
                    </div>
                    <div class="form-control">
                        <label class="label-text mb-1 font-semibold">Chapter Counter Target</label>
                        <input type="number" name="target_chapters" value="{{ .Project.TargetChapters }}" min="3" max="25" class="input input-bordered input-sm bg-base-100" />
                    </div>
                </div>

                <div class="form-control w-full">
                    <label class="label-text font-bold mb-1">Project Classification Vector</label>
                    <div class="join w-full shadow-sm">
                        <input class="join-item btn btn-sm flex-1" type="radio" name="book_type" value="nonfiction" aria-label="Non-Fiction" hx-get="/app/ui/fragments/intake-nonfiction" hx-target="#dynamic-questions-wrapper" checked />
                        <input class="join-item btn btn-sm flex-1" type="radio" name="book_type" value="fiction" aria-label="Fiction" hx-get="/app/ui/fragments/intake-fiction" hx-target="#dynamic-questions-wrapper" />
                    </div>
                </div>

                <div id="dynamic-questions-wrapper" class="transition-all duration-300">
                    <div class="form-control">
                        <div class="flex justify-between items-center mb-1">
                            <label class="label-text font-semibold">Core Audience Hunger & Desires</label>
                            <button type="button" class="btn btn-xs btn-outline btn-secondary font-bold" hx-post="/api/project/{{ .Project.Id }}/escape-hatch?field=reader_hunger" hx-target="#hunger-textarea" hx-swap="outerHTML">✨ Inspire Me</button>
                        </div>
                        <textarea id="hunger-textarea" name="reader_hunger" class="textarea textarea-bordered h-24 text-sm bg-base-100" placeholder="What deep pain, question, or frustration brings this exact reader to your book?"></textarea>
                    </div>
                </div>
            </form>
        </aside>

        <main class="col-span-12 lg:col-span-6 bg-base-200 p-6 overflow-y-auto h-full flex flex-col" id="workspace-panel">
            <div class="card bg-base-100 shadow border border-base-300 flex-1 flex flex-col overflow-hidden" id="chapter-cockpit-{{ .Chapter.Id }}">
                <div class="bg-base-200 p-4 border-b border-base-300 flex justify-between items-center shrink-0">
                    <div>
                        <h3 class="font-black text-lg text-base-content">Chapter {{ .Chapter.SortOrder }}: {{ .Chapter.Title }}</h3>
                        <p class="text-xs font-semibold text-base-content/60">Objective: {{ .Chapter.Purpose }}</p>
                    </div>
                    <span class="badge badge-primary font-mono text-xs font-bold uppercase p-3">{{ .Chapter.Status }}</span>
                </div>

                <div class="tabs tabs-lifted bg-base-100 px-4 pt-2 shrink-0">
                    <input type="radio" name="cockpit_tabs_{{ .Chapter.Id }}" class="tab text-xs font-bold" aria-label="1. Raw Draft Copy" checked />
                    <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
                        <div class="prose text-sm leading-relaxed">{{ .Chapter.RawDraft }}</div>
                    </div>

                    <input type="radio" name="cockpit_tabs_{{ .Chapter.Id }}" class="tab text-xs font-bold" aria-label="2. Expert Editorial Diagnosis" />
                    <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
                        <div class="bg-warning/10 border border-warning/30 p-4 rounded-xl text-sm font-mono text-warning-content">{{ .Chapter.EditorialDiagnosis }}</div>
                        <div class="mt-4 flex gap-2">
                            <input type="text" id="manual-correction-{{ .Chapter.Id }}" name="user_diagnosis_notes" placeholder="Add manual editorial directives..." class="input input-bordered input-sm flex-1 text-sm" />
                            <button class="btn btn-sm btn-warning font-bold" hx-post="/api/project/{{ .ProjectId }}/chapters/{{ .Chapter.Id }}/pipeline/rewrite" hx-include="#manual-correction-{{ .Chapter.Id }}" hx-target="#chapter-cockpit-{{ .Chapter.Id }}" hx-swap="outerHTML">Override & Rewrite</button>
                        </div>
                    </div>

                    <input type="radio" name="cockpit_tabs_{{ .Chapter.Id }}" class="tab text-xs font-bold" aria-label="3. Targeted Revision" />
                    <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
                        <div class="prose text-sm leading-relaxed text-secondary-content/90">{{ .Chapter.TargetedRewrite }}</div>
                    </div>

                    <input type="radio" name="cockpit_tabs_{{ .Chapter.Id }}" class="tab text-xs font-bold" aria-label="4. Polished Manuscript" />
                    <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
                        <div class="prose text-base leading-relaxed font-serif p-2 bg-base-50 rounded shadow-inner border border-base-200">{{ .Chapter.DraftContent }}</div>
                    </div>
                </div>

                <div class="p-3 bg-base-200 border-t border-base-300 shrink-0 flex justify-end gap-2">
                    <button class="btn btn-xs btn-outline btn-neutral" hx-post="/api/project/{{ .ProjectId }}/chapters/{{ .Chapter.Id }}/pipeline/draft" hx-target="#chapter-cockpit-{{ .Chapter.Id }}" hx-swap="outerHTML">Step 1: Draft</button>
                    <button class="btn btn-xs btn-outline btn-warning" hx-post="/api/project/{{ .ProjectId }}/chapters/{{ .Chapter.Id }}/pipeline/diagnose" hx-target="#chapter-cockpit-{{ .Chapter.Id }}" hx-swap="outerHTML">Step 2: Diagnose</button>
                    <button class="btn btn-xs btn-outline btn-secondary" hx-post="/api/project/{{ .ProjectId }}/chapters/{{ .Chapter.Id }}/pipeline/rewrite" hx-target="#chapter-cockpit-{{ .Chapter.Id }}" hx-swap="outerHTML">Step 3: Revise</button>
                    <button class="btn btn-xs btn-success font-bold" hx-post="/api/project/{{ .ProjectId }}/chapters/{{ .Chapter.Id }}/pipeline/polish" hx-target="#chapter-cockpit-{{ .Chapter.Id }}" hx-swap="outerHTML">Step 4: Polish & Lock</button>
                </div>
            </div>
        </main>

        <aside class="col-span-12 lg:col-span-3 border-l border-base-300 bg-base-100 p-4 overflow-y-auto flex flex-col gap-4 h-full">
            <div class="card bg-neutral text-neutral-content shadow-xl border border-neutral-focus p-4">
                <div class="card-body p-2 flex flex-col gap-3">
                    <h3 class="card-title text-lg font-black tracking-tight text-primary">⚡ Fast Track Autopilot</h3>
                    <p class="text-xs text-neutral-content/80 leading-relaxed">Let the engine run the full drafting, editing, and polishing cycles automatically across all approved chapters in a single background pass.</p>
                    <button class="btn btn-primary btn-md font-bold shadow w-full mt-2" hx-post="/api/project/{{ .Project.Id }}/generate/autopilot" hx-target="#workspace-panel">🚀 Generate Full Prototype</button>
                </div>
            </div>
        </aside>

    </div>
</body>
</html>