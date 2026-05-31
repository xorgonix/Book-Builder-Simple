package views

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"
)

type Project struct {
	ID             string
	Title          string
	BookType       string
	Status         string
	TargetLength   string
	TargetChapters int
}

type Chapter struct {
	ID                   string
	SortOrder            int
	Title                string
	Status               string
	Purpose              string
	RawDraft             string
	EditorialDiagnosis   string
	TargetedRewrite      string
	DraftContent         string
	PreviousDraftContent string
}

type PageData struct {
	Project         Project
	Chapter         *Chapter
	AllProjects     []Project
	ActiveProjectID string
	IntakeSaved     bool
}

type EscapeHatchData struct {
	ProjectID   string
	Field       string
	Suggestions []string
}

type ProcessingData struct {
	Stage     string
	ProjectID string
	ChapterID string
}

type StatusData struct {
	JobType   string
	Status    string
	ChapterID string
}

var templates = template.Must(template.New("views").Funcs(template.FuncMap{
	"eq":            func(a, b string) bool { return a == b },
	"chapterSuffix": chapterSuffix,
	"statusClass":   statusClass,
	"js":            jsString,
	"selector":      fieldSelector,
}).Parse(templateSource))

func RenderPage(data PageData) (string, error) {
	return render("page", data)
}

func RenderGrid(data PageData) (string, error) {
	return render("grid", data)
}

func RenderIntakeNonfiction(projectID string) (string, error) {
	return render("intake_nonfiction", Project{ID: projectID})
}

func RenderIntakeFiction(projectID string) (string, error) {
	return render("intake_fiction", Project{ID: projectID})
}

func RenderEscapeHatch(data EscapeHatchData) (string, error) {
	return render("escape_hatch", data)
}

func RenderProcessing(data ProcessingData) (string, error) {
	return render("processing", data)
}

func RenderStatus(data StatusData) (string, error) {
	return render("status", data)
}

func render(name string, data any) (string, error) {
	var b bytes.Buffer
	if err := templates.ExecuteTemplate(&b, name, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func chapterSuffix(chapterID string) string {
	if chapterID == "" {
		return ""
	}
	return " for chapter " + chapterID
}

func statusClass(status string) string {
	switch status {
	case "failed":
		return "alert-error"
	case "completed":
		return "alert-success"
	default:
		return "alert-info"
	}
}

func jsString(value string) template.JS {
	return template.JS(strconv.Quote(value))
}

func fieldSelector(field string) string {
	return `[name="` + field + `"]`
}

const templateSource = `
{{ define "page" -}}
<!DOCTYPE html>
<html lang="en" data-theme="cupcake" class="h-full">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Book Prototype Architect</title>
    <link href="https://cdn.jsdelivr.net/npm/daisyui@5" rel="stylesheet" type="text/css" />
    <script src="https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4"></script>
    <script src="https://unpkg.com/htmx.org@2.0.0"></script>
</head>
<body class="bg-base-300 h-screen overflow-hidden flex flex-col font-sans">
    {{ template "header" . }}
    {{ template "grid" . }}
</body>
</html>
{{- end }}

{{ define "header" -}}
<header class="navbar bg-base-100 border-b border-base-300 px-4 shrink-0 flex justify-between items-center h-16">
    <div class="flex items-center gap-3 min-w-0">
        <span class="text-xl font-black tracking-wider text-primary truncate">SOVEREIGN.ENGINE</span>
        <div id="project-header-badge" class="badge badge-neutral font-mono font-bold">MODE: DETERMINISTIC</div>
    </div>
    <div class="flex items-center gap-2">
        <select name="project_id" class="select select-bordered select-sm w-48 font-semibold"
                hx-get="/app/project/select" hx-target="#main-layout-grid" hx-swap="outerHTML">
            {{ range .AllProjects }}
            <option value="{{ .ID }}" {{ if eq .ID $.ActiveProjectID }}selected{{ end }}>{{ .Title }}</option>
            {{ end }}
        </select>
        <button class="btn btn-sm btn-primary font-bold shadow" hx-post="/api/projects" hx-target="#main-layout-grid" hx-swap="outerHTML">+ New Book</button>
    </div>
</header>
{{- end }}

{{ define "grid" -}}
<div class="grid grid-cols-12 flex-1 overflow-hidden h-[calc(100vh-4rem)]" id="main-layout-grid">
    <aside class="col-span-12 lg:col-span-3 border-r border-base-300 bg-base-100 p-4 overflow-y-auto flex flex-col gap-4 h-full">
        {{ template "intake_form" . }}
    </aside>

    <main class="col-span-12 lg:col-span-6 bg-base-200 p-6 overflow-y-auto h-full flex flex-col" id="workspace-panel">
        {{ if .Chapter }}
            {{ template "chapter_cockpit" . }}
        {{ else }}
            {{ template "empty_workspace" . }}
        {{ end }}
    </main>

    <aside class="col-span-12 lg:col-span-3 border-l border-base-300 bg-base-100 p-4 overflow-y-auto flex flex-col gap-4 h-full">
        {{ template "autopilot" . }}
    </aside>
</div>
{{- end }}

{{ define "intake_form" -}}
<form hx-post="/api/project/{{ .Project.ID }}/intake" hx-target="#main-layout-grid" hx-swap="outerHTML" class="flex flex-col gap-4">
    <ul class="steps steps-horizontal w-full text-xs font-bold mb-2">
        <li class="step step-primary">Type</li>
        <li class="step {{ if .IntakeSaved }}step-primary{{ end }}">Intent</li>
        <li class="step">Blueprint</li>
        <li class="step">Drafting</li>
    </ul>

    <div class="bg-base-200 p-3 rounded-lg border border-base-300 flex flex-col gap-3">
        <span class="text-xs font-bold uppercase tracking-wider text-base-content/60">Manuscript Target Scale</span>
        <div class="form-control">
            <label class="label-text mb-1 font-semibold">Target Length</label>
            <select name="target_length" class="select select-bordered select-sm w-full bg-base-100">
                <option value="short_guide" {{ if eq .Project.TargetLength "short_guide" }}selected{{ end }}>Short Guide (~5k words / 5 Chapters)</option>
                <option value="practical_ebook" {{ if eq .Project.TargetLength "practical_ebook" }}selected{{ end }}>Practical eBook (~15k words / 10 Chapters)</option>
                <option value="full_prototype" {{ if eq .Project.TargetLength "full_prototype" }}selected{{ end }}>Full-Length Prototype (~40k words / 15 Chapters)</option>
            </select>
        </div>
        <div class="form-control">
            <label class="label-text mb-1 font-semibold">Chapter Counter Target</label>
            <input type="number" name="target_chapters" value="{{ .Project.TargetChapters }}" min="1" max="25" class="input input-bordered input-sm bg-base-100" />
        </div>
    </div>

    <div class="form-control w-full">
        <label class="label-text font-bold mb-1">Project Classification Vector</label>
        <div class="join w-full shadow-sm">
            <input class="join-item btn btn-sm flex-1" type="radio" name="book_type" value="nonfiction" aria-label="Non-Fiction"
                   hx-get="/app/ui/fragments/intake-nonfiction?project_id={{ .Project.ID }}" hx-target="#dynamic-questions-wrapper" {{ if eq .Project.BookType "nonfiction" }}checked{{ end }} />
            <input class="join-item btn btn-sm flex-1" type="radio" name="book_type" value="fiction" aria-label="Fiction"
                   hx-get="/app/ui/fragments/intake-fiction?project_id={{ .Project.ID }}" hx-target="#dynamic-questions-wrapper" {{ if eq .Project.BookType "fiction" }}checked{{ end }} />
        </div>
    </div>

    <div id="dynamic-questions-wrapper" class="transition-all duration-300 flex flex-col gap-3">
        {{ if eq .Project.BookType "fiction" }}
            {{ template "intake_fiction" .Project }}
        {{ else }}
            {{ template "intake_nonfiction" .Project }}
        {{ end }}
    </div>

    <button class="btn btn-sm btn-primary font-bold">Save Intake</button>
</form>
{{- end }}

{{ define "intake_nonfiction" -}}
<textarea class="textarea textarea-bordered text-sm bg-base-100" name="core_topic" placeholder="What is the book explicitly about?"></textarea>
<textarea class="textarea textarea-bordered text-sm bg-base-100" name="target_audience" placeholder="Who is this book written for?"></textarea>
<div class="form-control">
    <div class="flex justify-between items-center mb-1 gap-2">
        <label class="label-text font-semibold">Core Audience Hunger & Desires</label>
        <button type="button" class="btn btn-xs btn-outline btn-secondary font-bold" hx-post="/api/project/{{ .ID }}/escape-hatch?field=reader_hunger" hx-target="#hunger-textarea" hx-swap="outerHTML">Inspire Me</button>
    </div>
    <textarea id="hunger-textarea" name="reader_hunger" class="textarea textarea-bordered h-24 text-sm bg-base-100" placeholder="What deep pain, question, or frustration brings this exact reader to your book?"></textarea>
</div>
<textarea class="textarea textarea-bordered text-sm bg-base-100" name="prohibited_directions" placeholder="What should this book absolutely not become?"></textarea>
<textarea class="textarea textarea-bordered text-sm bg-base-100" name="author_intent" placeholder="What should the reader be able to do, understand, or feel after reading?"></textarea>
<input class="input input-bordered input-sm bg-base-100" name="selected_tone" placeholder="Tone, e.g. Warm, Direct, Credible" />
{{- end }}

{{ define "intake_fiction" -}}
<textarea class="textarea textarea-bordered text-sm bg-base-100" name="core_topic" placeholder="What is the story premise?"></textarea>
<textarea class="textarea textarea-bordered text-sm bg-base-100" name="target_audience" placeholder="Who is the ideal reader?"></textarea>
<div class="form-control">
    <div class="flex justify-between items-center mb-1 gap-2">
        <label class="label-text font-semibold">Reader Emotional Pull</label>
        <button type="button" class="btn btn-xs btn-outline btn-secondary font-bold" hx-post="/api/project/{{ .ID }}/escape-hatch?field=reader_hunger" hx-target="#hunger-textarea" hx-swap="outerHTML">Surprise Me</button>
    </div>
    <textarea id="hunger-textarea" name="reader_hunger" class="textarea textarea-bordered h-24 text-sm bg-base-100" placeholder="What emotional experience should pull the reader through?"></textarea>
</div>
<textarea class="textarea textarea-bordered text-sm bg-base-100" name="prohibited_directions" placeholder="What genre traps or story directions should be avoided?"></textarea>
<textarea class="textarea textarea-bordered text-sm bg-base-100" name="author_intent" placeholder="What should linger after the final page?"></textarea>
<input class="input input-bordered input-sm bg-base-100" name="selected_tone" placeholder="Tone, e.g. Lyrical, Tense, Wry" />
{{- end }}

{{ define "empty_workspace" -}}
<div class="card bg-base-100 shadow border border-base-300 flex-1">
    <div class="card-body">
        <div class="mx-auto w-52 max-w-full rounded-lg bg-primary text-primary-content shadow-xl p-5 aspect-[2/3] flex flex-col justify-between">
            <div>
                <div class="text-xs font-black uppercase opacity-80">Working Title</div>
                <h2 class="text-2xl font-black leading-tight mt-3">{{ .Project.Title }}</h2>
            </div>
            <div class="text-xs font-semibold opacity-80">Book Prototype Architect</div>
        </div>
        {{ if .IntakeSaved }}
        <div class="alert alert-success mt-6">Intake saved. Next: use Fast Track Autopilot to generate the blueprint and chapter shells.</div>
        {{ else }}
        <div class="alert alert-info mt-6">Fill the intake fields, save them, then generate the blueprint and chapter shells.</div>
        {{ end }}
    </div>
</div>
{{- end }}

{{ define "chapter_cockpit" -}}
<div class="card bg-base-100 shadow border border-base-300 flex-1 flex flex-col overflow-hidden" id="chapter-cockpit-{{ .Chapter.ID }}">
    <div class="bg-base-200 p-4 border-b border-base-300 flex justify-between items-center shrink-0 gap-3">
        <div class="min-w-0">
            <h3 class="font-black text-lg text-base-content truncate">Chapter {{ .Chapter.SortOrder }}: {{ .Chapter.Title }}</h3>
            <p class="text-xs font-semibold text-base-content/60">Objective: {{ .Chapter.Purpose }}</p>
        </div>
        <span class="badge badge-primary font-mono text-xs font-bold uppercase p-3">{{ .Chapter.Status }}</span>
    </div>

    <div class="tabs tabs-lifted bg-base-100 px-4 pt-2 shrink-0">
        <input type="radio" name="cockpit_tabs_{{ .Chapter.ID }}" class="tab text-xs font-bold" aria-label="1. Raw Draft Copy" checked />
        <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
            <div class="prose max-w-none text-sm leading-relaxed whitespace-pre-wrap">{{ .Chapter.RawDraft }}</div>
        </div>

        <input type="radio" name="cockpit_tabs_{{ .Chapter.ID }}" class="tab text-xs font-bold" aria-label="2. Expert Editorial Diagnosis" />
        <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
            <div class="bg-warning/10 border border-warning/30 p-4 rounded-lg text-sm font-mono text-warning-content whitespace-pre-wrap">{{ .Chapter.EditorialDiagnosis }}</div>
            <div class="mt-4 flex gap-2">
                <input type="text" id="manual-correction-{{ .Chapter.ID }}" name="user_diagnosis_notes" placeholder="Add manual editorial directives..." class="input input-bordered input-sm flex-1 text-sm" />
                <button class="btn btn-sm btn-warning font-bold" hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/rewrite" hx-include="#manual-correction-{{ .Chapter.ID }}" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Override & Rewrite</button>
            </div>
        </div>

        <input type="radio" name="cockpit_tabs_{{ .Chapter.ID }}" class="tab text-xs font-bold" aria-label="3. Targeted Revision" />
        <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
            <div class="prose max-w-none text-sm leading-relaxed whitespace-pre-wrap">{{ .Chapter.TargetedRewrite }}</div>
        </div>

        <input type="radio" name="cockpit_tabs_{{ .Chapter.ID }}" class="tab text-xs font-bold" aria-label="4. Polished Manuscript" />
        <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
            <div class="prose max-w-none text-base leading-relaxed font-serif p-2 bg-base-100 rounded-lg border border-base-200 whitespace-pre-wrap">{{ .Chapter.DraftContent }}</div>
        </div>
    </div>

    <div class="p-3 bg-base-200 border-t border-base-300 shrink-0 flex flex-wrap justify-end gap-2">
        <button class="btn btn-xs btn-outline btn-neutral" hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/draft" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Step 1: Draft</button>
        <button class="btn btn-xs btn-outline btn-warning" hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/diagnose" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Step 2: Diagnose</button>
        <button class="btn btn-xs btn-outline btn-secondary" hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/rewrite" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Step 3: Revise</button>
        <button class="btn btn-xs btn-success font-bold" hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/polish" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Step 4: Polish & Lock</button>
    </div>
</div>
{{- end }}

{{ define "autopilot" -}}
<div class="card bg-neutral text-neutral-content shadow-xl border border-neutral p-4">
    <div class="card-body p-2 flex flex-col gap-3">
        <h3 class="card-title text-lg font-black tracking-tight text-primary">Blueprint & Drafting</h3>
        <p class="text-xs text-neutral-content/80 leading-relaxed">Start with the blueprint, then use the chapter cockpit to draft, diagnose, revise, or polish one step at a time.</p>
        <button class="btn btn-primary btn-md font-bold shadow w-full mt-2" hx-post="/api/project/{{ .Project.ID }}/generate/blueprint" hx-target="#workspace-panel">Generate Blueprint</button>
        <div class="divider my-1 text-neutral-content/50">OR</div>
        <p class="text-xs text-neutral-content/70 leading-relaxed">Skip review and run every stage across all chapters automatically.</p>
        <button class="btn btn-outline btn-sm border-neutral-content/40 text-neutral-content" hx-post="/api/project/{{ .Project.ID }}/generate/autopilot" hx-target="#workspace-panel">Full Autopilot</button>
        <div id="project-status"></div>
    </div>
</div>
{{- end }}

{{ define "escape_hatch" -}}
<div class="flex flex-col gap-2">
    {{ range .Suggestions }}
    <button type="button" class="btn btn-xs btn-outline justify-start h-auto min-h-8 text-left whitespace-normal" onclick='document.querySelector({{ js (selector $.Field) }}).value = {{ js . }}'>{{ . }}</button>
    {{ end }}
    <textarea id="{{ .Field }}" name="{{ .Field }}" class="textarea textarea-bordered w-full">{{ index .Suggestions 0 }}</textarea>
    <input type="hidden" name="project_id" value="{{ .ProjectID }}">
</div>
{{- end }}

{{ define "processing" -}}
<div class="alert alert-info" hx-get="/api/project/{{ .ProjectID }}/status" hx-trigger="every 2s" hx-swap="outerHTML">
    <span class="loading loading-spinner loading-sm"></span>
    <span>{{ .Stage }} is running{{ chapterSuffix .ChapterID }}.</span>
</div>
{{- end }}

{{ define "status" -}}
<div id="project-status" class="alert {{ statusClass .Status }}">
    <span>{{ .JobType }}: {{ .Status }}{{ chapterSuffix .ChapterID }}</span>
</div>
{{- end }}
`

func MustRender(name string, data any) string {
	result, err := render(name, data)
	if err != nil {
		return fmt.Sprintf("template error: %s", err)
	}
	return result
}
