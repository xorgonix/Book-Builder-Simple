package views

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"
	"strings"
)

type Project struct {
	ID                      string
	Title                   string
	BookType                string
	Status                  string
	TargetLength            string
	TargetChapters          int
	Intake                  map[string]string
	AuthorName              string
	PublishingMetadataJSON  string
	BookArchitectureJSON    string
	GlobalStyleContractJSON string
}

type Chapter struct {
	ID                   string
	SortOrder            int
	Title                string
	Subtitle             string
	FrontMatterLabel     string
	FrontMatterBlurb     string
	StateStart           string
	StateEnd             string
	ChapterMetadataJSON  string
	ArcMetadataJSON      string
	ConceptJurisdiction  string
	GenerationDirectives string
	MediaPromptsJSON     string
	Status               string
	Purpose              string
	RawDraft             string
	EditorialDiagnosis   string
	TargetedRewrite      string
	DraftContent         string
	PreviousDraftContent string
}

type Brief struct {
	Title         string
	Subtitle      string
	Promise       string
	VoiceTone     string
	WhatItIs      string
	WhatItIsNot   string
	AISuggestions string
}

type PageData struct {
	Project         Project
	Brief           *Brief
	Chapter         *Chapter
	PrevChapter     *Chapter
	NextChapter     *Chapter
	Chapters        []Chapter
	AllProjects     []Project
	ActiveProjectID string
	WorkspaceTab    string
	IntakeSaved     bool
	SaveNotice      string
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
	JobType      string
	Status       string
	ChapterID    string
	ErrorMessage string
}

type ExportResultData struct {
	MarkdownURL    string
	EPUBURL        string
	HTMLURL        string
	LintReportURL  string
	StyleReportURL string
	Warnings       []string
}

type ExportWaitData struct {
	Message string
}

type ExportBlockedData struct {
	Message   string
	Warnings  []string
	ProjectID string
}

var templates = template.Must(template.New("views").Funcs(template.FuncMap{
	"eq":            func(a, b string) bool { return a == b },
	"chapterSuffix": chapterSuffix,
	"stageLabel":    stageLabel,
	"jobLabel":      jobLabel,
	"statusLabel":   statusLabel,
	"statusClass":   statusClass,
	"stepDone":      stepDone,
	"js":            jsString,
	"selector":      fieldSelector,
}).Parse(templateSource))

func RenderPage(data PageData) (string, error) {
	return render("page", data)
}

func RenderGrid(data PageData) (string, error) {
	return render("grid", data)
}

func RenderWorkspace(data PageData) (string, error) {
	return render("workspace_panel", data)
}

func RenderChapterCockpit(data PageData) (string, error) {
	return render("chapter_cockpit", data)
}

func RenderIntakeNonfiction(project Project) (string, error) {
	return render("intake_nonfiction", project)
}

func RenderIntakeFiction(project Project) (string, error) {
	return render("intake_fiction", project)
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

func RenderExportResult(data ExportResultData) (string, error) {
	return render("export_result", data)
}

func RenderExportWait(data ExportWaitData) (string, error) {
	return render("export_wait", data)
}

func RenderExportBlocked(data ExportBlockedData) (string, error) {
	return render("export_blocked", data)
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

func stageLabel(stage string) string {
	switch strings.TrimSpace(strings.ToLower(stage)) {
	case "book brief", "brief":
		return "brief"
	case "outline and chapter shells", "outline", "toc":
		return "outline"
	case "draft", "drafting":
		return "draft"
	case "diagnose", "feedback":
		return "feedback"
	case "rewrite":
		return "rewrite"
	case "polish", "finish":
		return "finish"
	case "autopilot", "auto":
		return "auto"
	default:
		return strings.TrimSpace(stage)
	}
}

func jobLabel(jobType string) string {
	return stageLabel(jobType)
}

func statusLabel(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "running":
		return "working"
	case "completed":
		return "done"
	case "failed":
		return "stopped"
	default:
		return strings.TrimSpace(status)
	}
}

func stepDone(status string, step string) bool {
	switch step {
	case "intent":
		return status != ""
	case "brief":
		return status == "brief" || status == "toc" || status == "drafting"
	case "toc":
		return status == "toc" || status == "drafting"
	case "drafting":
		return status == "drafting"
	default:
		return false
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
    <style>
        .htmx-indicator { display: none; }
        .htmx-request .htmx-indicator,
        .htmx-request.htmx-indicator { display: inline-flex; }
    </style>
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
    </div>
</header>
{{- end }}

{{ define "grid" -}}
<div class="grid grid-cols-12 flex-1 overflow-hidden h-[calc(100vh-4rem)]" id="main-layout-grid">
    <aside class="col-span-12 lg:col-span-3 border-r border-base-300 bg-base-100 p-4 overflow-y-auto flex flex-col gap-4 h-full">
        {{ template "intake_form" . }}
    </aside>

    <main class="col-span-12 lg:col-span-6 bg-base-200 p-6 overflow-y-auto h-full flex flex-col" id="workspace-panel">
        {{ template "workspace_panel" . }}
    </main>

    <aside class="col-span-12 lg:col-span-3 border-l border-base-300 bg-base-100 p-4 overflow-y-auto flex flex-col gap-4 h-full">
        {{ template "project_list" . }}
        {{ template "autopilot" . }}
        {{ template "exports_panel" . }}
    </aside>
</div>
{{- end }}

{{ define "workspace_panel" -}}
{{ template "workspace_tabs" . }}
{{- end }}

{{ define "intake_form" -}}
<form id="book-setup-form" hx-post="/api/project/{{ .Project.ID }}/intake" hx-target="#main-layout-grid" hx-swap="outerHTML" hx-indicator="#setup-save-indicator" class="flex flex-col gap-4">
    <ul class="steps steps-horizontal w-full text-xs font-bold mb-2">
        <li class="step step-primary">Type</li>
        <li class="step {{ if .IntakeSaved }}step-primary{{ end }}">Intent</li>
        <li class="step {{ if stepDone .Project.Status "brief" }}step-primary{{ end }}">Brief</li>
        <li class="step {{ if stepDone .Project.Status "toc" }}step-primary{{ end }}">Outline</li>
        <li class="step {{ if stepDone .Project.Status "drafting" }}step-primary{{ end }}">Drafting</li>
    </ul>

    <div class="alert alert-info py-2 px-3 text-xs leading-relaxed">
        <span><span class="font-bold">Active book:</span> {{ .Project.Title }}. The setup fields below are the saved foundation for this project.</span>
    </div>

    <div class="bg-base-200 p-3 rounded-lg border border-base-300 flex flex-col gap-3">
        <span class="text-xs font-bold uppercase tracking-wider text-base-content/60">Book Setup</span>
        <div class="form-control">
            <div class="flex items-center gap-2 mb-1">
                <label class="label-text font-semibold">Book Title</label>
                <div class="tooltip tooltip-right" data-tip="This is the working project name and the title used by the brief unless the brief generates a replacement.">
                    <span class="badge badge-ghost badge-sm">?</span>
                </div>
            </div>
            <input class="input input-bordered input-sm bg-base-100" name="title" value="{{ .Project.Title }}" placeholder="Name this book before saving" required />
            <p class="mt-1 text-[11px] text-base-content/60">This makes it clear which book you are editing before you add premise, audience, and structure.</p>
        </div>
        <div class="form-control">
            <div class="flex items-center gap-2 mb-1">
                <label class="label-text font-semibold">Book Length</label>
                <div class="tooltip tooltip-right" data-tip="Controls the approximate book size and how much material the outline and chapter shells should generate.">
                    <span class="badge badge-ghost badge-sm">?</span>
                </div>
            </div>
            <select name="target_length" class="select select-bordered select-sm w-full bg-base-100">
                <option value="short_guide" {{ if eq .Project.TargetLength "short_guide" }}selected{{ end }}>Short Guide (~5k words)</option>
                <option value="practical_ebook" {{ if eq .Project.TargetLength "practical_ebook" }}selected{{ end }}>Practical eBook (~15k words)</option>
                <option value="full_prototype" {{ if eq .Project.TargetLength "full_prototype" }}selected{{ end }}>Full-Length Prototype (~40k words)</option>
            </select>
            <p class="mt-1 text-[11px] text-base-content/60">This controls approximate total word count. The chapter count below controls how many chapter shells are generated.</p>
        </div>
        <div class="form-control">
            <div class="flex items-center gap-2 mb-1">
                <label class="label-text font-semibold">How Many Chapters?</label>
                <div class="tooltip tooltip-right" data-tip="Sets how many chapter shells the outline should produce.">
                    <span class="badge badge-ghost badge-sm">?</span>
                </div>
            </div>
            <input type="number" name="target_chapters" value="{{ .Project.TargetChapters }}" min="1" max="25" class="input input-bordered input-sm bg-base-100" />
            <p class="mt-1 text-[11px] text-base-content/60">Use this if you want to change the default chapter count.</p>
        </div>
    </div>

    <div class="form-control w-full">
        <div class="flex items-center gap-2 mb-1">
            <label class="label-text font-bold">What Kind of Book?</label>
            <div class="tooltip tooltip-right" data-tip="First pick fiction or nonfiction. Then the form below helps narrow it down.">
                <span class="badge badge-ghost badge-sm">?</span>
            </div>
        </div>
        <div class="join w-full shadow-sm">
            <input class="join-item btn btn-sm flex-1" type="radio" name="book_type" value="nonfiction" aria-label="Non-Fiction"
                   hx-get="/app/ui/fragments/intake-nonfiction?project_id={{ .Project.ID }}" hx-target="#dynamic-questions-wrapper" {{ if eq .Project.BookType "nonfiction" }}checked{{ end }} />
            <input class="join-item btn btn-sm flex-1" type="radio" name="book_type" value="fiction" aria-label="Fiction"
                   hx-get="/app/ui/fragments/intake-fiction?project_id={{ .Project.ID }}" hx-target="#dynamic-questions-wrapper" {{ if eq .Project.BookType "fiction" }}checked{{ end }} />
        </div>
        <p class="mt-1 text-[11px] text-base-content/60">This is only the first split. The next fields let you choose the specific form, point of view, and story pattern.</p>
    </div>

    <div id="dynamic-questions-wrapper" class="transition-all duration-300 flex flex-col gap-3">
        {{ if eq .Project.BookType "fiction" }}
            {{ template "intake_fiction" .Project }}
        {{ else }}
            {{ template "intake_nonfiction" .Project }}
        {{ end }}
    </div>

    {{ if .SaveNotice }}
    <div class="alert alert-success py-2 px-3 text-xs leading-relaxed">
        <span class="font-bold">{{ .SaveNotice }}</span>
    </div>
    {{ end }}

    <div class="alert alert-success py-2 px-3 text-xs leading-relaxed">
        {{ if .IntakeSaved }}
        <span><span class="font-bold">Saved setup loaded.</span> Changes here are not committed until you press Save Book Setup.</span>
        {{ else }}
        <span><span class="font-bold">New book setup.</span> Save the title and characteristics before generating the Book Brief.</span>
        {{ end }}
    </div>

    <button class="btn btn-sm btn-primary font-bold" title="Save the title and intake values so the Book Brief can be generated from them.">
        <span id="setup-save-indicator" class="loading loading-spinner loading-xs htmx-indicator"></span>
        Save Book Setup
    </button>
</form>
{{- end }}

{{ define "intake_nonfiction" -}}
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Nonfiction Style</label>
        <div class="tooltip tooltip-right" data-tip="Pick the nonfiction lane that feels closest. This helps the app speak in the right book style.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <select name="book_form" class="select select-bordered select-sm w-full bg-base-100">
        <option value="" {{ if eq (index .Intake "book_form") "" }}selected{{ end }}>Choose a style</option>
        <option value="how_to" {{ if eq (index .Intake "book_form") "how_to" }}selected{{ end }}>How-To / Step-by-Step Guide</option>
        <option value="self_help" {{ if eq (index .Intake "book_form") "self_help" }}selected{{ end }}>Self-Help / Personal Growth</option>
        <option value="business" {{ if eq (index .Intake "book_form") "business" }}selected{{ end }}>Business / Strategy Playbook</option>
        <option value="memoir" {{ if eq (index .Intake "book_form") "memoir" }}selected{{ end }}>Memoir / Personal Story</option>
        <option value="history" {{ if eq (index .Intake "book_form") "history" }}selected{{ end }}>History / Explanation</option>
        <option value="biography" {{ if eq (index .Intake "book_form") "biography" }}selected{{ end }}>Biography / Profile</option>
        <option value="argument" {{ if eq (index .Intake "book_form") "argument" }}selected{{ end }}>Argument / Research / Essay</option>
        <option value="workbook" {{ if eq (index .Intake "book_form") "workbook" }}selected{{ end }}>Workbook / Exercises / Templates</option>
        <option value="thought_leadership" {{ if eq (index .Intake "book_form") "thought_leadership" }}selected{{ end }}>Thought Leadership / Framework</option>
    </select>
    <p class="text-[11px] text-base-content/60">Examples: a how-to book, memoir, business book, workbook, or thought-leadership title.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">What Is It About?</label>
        <div class="tooltip tooltip-right" data-tip="Write the main idea in one plain sentence. This becomes part of the Book Brief and outline focus.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <textarea class="textarea textarea-bordered text-sm bg-base-100" name="core_topic" placeholder="In one sentence, what is this book about?">{{ index .Intake "core_topic" }}</textarea>
    <p class="text-[11px] text-base-content/60">Example: the book teaches, explains, or helps the reader do something specific.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Who Is It For?</label>
        <div class="tooltip tooltip-right" data-tip="Name the reader this book is meant to help. A specific person is better than a broad audience.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <textarea id="target_audience" class="textarea textarea-bordered text-sm bg-base-100" name="target_audience" placeholder="For example: first-time managers, exhausted parents, or romance readers">{{ index .Intake "target_audience" }}</textarea>
    <div class="mt-1 flex flex-wrap gap-2">
        <button type="button" class="btn btn-xs btn-outline btn-secondary font-bold" title="Suggest a specific reader profile based on the premise." hx-post="/api/project/{{ .ID }}/escape-hatch?field=target_audience" hx-target="#target_audience" hx-swap="outerHTML">Suggest Reader</button>
    </div>
    <p class="text-[11px] text-base-content/60">This helps the app aim the tone, examples, and chapter order. If you are stuck, let the app suggest a reader profile.</p>
</div>
<div class="form-control">
    <div class="flex justify-between items-center mb-1 gap-2">
        <div class="flex items-center gap-2">
            <label class="label-text font-semibold">Why Do They Need It?</label>
            <div class="tooltip tooltip-right" data-tip="What problem, question, or wish makes this reader pick up the book?">
                <span class="badge badge-ghost badge-sm">?</span>
            </div>
        </div>
        <button type="button" class="btn btn-xs btn-outline btn-secondary font-bold" title="Generate premise-aware wording to help fill this field." hx-post="/api/project/{{ .ID }}/escape-hatch?field=reader_hunger" hx-target="#reader_hunger" hx-swap="outerHTML">Show a Hint</button>
    </div>
    <textarea id="reader_hunger" name="reader_hunger" class="textarea textarea-bordered h-24 text-sm bg-base-100" placeholder="What keeps this reader up at night, or what do they want solved?">{{ index .Intake "reader_hunger" }}</textarea>
    <p class="text-[11px] text-base-content/60">Make this premise-based. The hint should be concrete, not generic motivation.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Story Shape</label>
        <div class="tooltip tooltip-right" data-tip="Choose the story or book shape you want to follow, so the outline has a clear engine.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <select name="structure_model" class="select select-bordered select-sm w-full bg-base-100">
        <option value="" {{ if eq (index .Intake "structure_model") "" }}selected{{ end }}>Choose a shape</option>
        <option value="how_to_ladder" {{ if eq (index .Intake "structure_model") "how_to_ladder" }}selected{{ end }}>How-To Ladder</option>
        <option value="framework" {{ if eq (index .Intake "structure_model") "framework" }}selected{{ end }}>Framework / 3-Part Framework</option>
        <option value="case_study" {{ if eq (index .Intake "structure_model") "case_study" }}selected{{ end }}>Case Study / Example Driven</option>
        <option value="argument" {{ if eq (index .Intake "structure_model") "argument" }}selected{{ end }}>Argument / Thesis / Proof</option>
        <option value="memoir_arc" {{ if eq (index .Intake "structure_model") "memoir_arc" }}selected{{ end }}>Memoir Arc</option>
        <option value="workbook" {{ if eq (index .Intake "structure_model") "workbook" }}selected{{ end }}>Workbook / Exercises</option>
        <option value="hero_journey" {{ if eq (index .Intake "structure_model") "hero_journey" }}selected{{ end }}>Hero's Journey</option>
        <option value="romance_beats" {{ if eq (index .Intake "structure_model") "romance_beats" }}selected{{ end }}>Romance Beat Sheet</option>
        <option value="mystery_trail" {{ if eq (index .Intake "structure_model") "mystery_trail" }}selected{{ end }}>Mystery / Clue Trail</option>
        <option value="three_act" {{ if eq (index .Intake "structure_model") "three_act" }}selected{{ end }}>Three-Act Structure</option>
    </select>
    <p class="text-[11px] text-base-content/60">Examples: Hero's Journey, Three-Act, romance beats, a mystery clue trail, or a nonfiction framework.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">What To Avoid</label>
        <div class="tooltip tooltip-right" data-tip="Tells the app what to avoid so the brief does not drift into unwanted territory.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <textarea id="prohibited_directions" class="textarea textarea-bordered text-sm bg-base-100" name="prohibited_directions" placeholder="What should this book absolutely not become?">{{ index .Intake "prohibited_directions" }}</textarea>
    <div class="mt-1 flex flex-wrap gap-2">
        <button type="button" class="btn btn-xs btn-outline btn-secondary font-bold" title="Suggest specific things this book should avoid." hx-post="/api/project/{{ .ID }}/escape-hatch?field=prohibited_directions" hx-target="#prohibited_directions" hx-swap="outerHTML">Show Avoid Examples</button>
    </div>
    <p class="text-[11px] text-base-content/60">Think of this as a guardrail list: wrong tropes, wrong tone, wrong pace, or the wrong amount of detail.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Ending Effect</label>
        <div class="tooltip tooltip-right" data-tip="What the reader should know, do, or feel after reading.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <textarea class="textarea textarea-bordered text-sm bg-base-100" name="author_intent" placeholder="What should the reader be able to do, understand, or feel after reading?">{{ index .Intake "author_intent" }}</textarea>
    <p class="text-[11px] text-base-content/60">This is the ending effect. Use it to tell the app what the book should leave behind.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Selected Tone</label>
        <div class="tooltip tooltip-right" data-tip="A short style direction for the writing voice.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <input class="input input-bordered input-sm bg-base-100" name="selected_tone" value="{{ index .Intake "selected_tone" }}" placeholder="Tone, e.g. Warm, Direct, Credible" />
    <p class="text-[11px] text-base-content/60">Example tones: warm, practical, authoritative, compassionate, brisk, analytical.</p>
</div>
{{- end }}

{{ define "intake_fiction" -}}
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Fiction Type</label>
        <div class="tooltip tooltip-right" data-tip="Pick the fiction lane that fits best. This helps the brief use the right story language and expectations.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <select name="book_form" class="select select-bordered select-sm w-full bg-base-100">
        <option value="" {{ if eq (index .Intake "book_form") "" }}selected{{ end }}>Choose a genre</option>
        <option value="romance" {{ if eq (index .Intake "book_form") "romance" }}selected{{ end }}>Romance / Category Romance</option>
        <option value="mystery" {{ if eq (index .Intake "book_form") "mystery" }}selected{{ end }}>Mystery / Cozy Mystery</option>
        <option value="thriller" {{ if eq (index .Intake "book_form") "thriller" }}selected{{ end }}>Thriller / Suspense</option>
        <option value="fantasy" {{ if eq (index .Intake "book_form") "fantasy" }}selected{{ end }}>Fantasy</option>
        <option value="scifi" {{ if eq (index .Intake "book_form") "scifi" }}selected{{ end }}>Science Fiction</option>
        <option value="literary" {{ if eq (index .Intake "book_form") "literary" }}selected{{ end }}>Literary / Book Club</option>
        <option value="ya" {{ if eq (index .Intake "book_form") "ya" }}selected{{ end }}>Young Adult</option>
        <option value="childrens" {{ if eq (index .Intake "book_form") "childrens" }}selected{{ end }}>Children's</option>
        <option value="horror" {{ if eq (index .Intake "book_form") "horror" }}selected{{ end }}>Horror</option>
        <option value="adventure" {{ if eq (index .Intake "book_form") "adventure" }}selected{{ end }}>Adventure / Quest</option>
    </select>
    <p class="text-[11px] text-base-content/60">Examples: category romance, mystery, thriller, fantasy, sci-fi, book club fiction, YA, or children's fiction.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Story Premise</label>
        <div class="tooltip tooltip-right" data-tip="Write the core dramatic setup in one sentence. This drives the Book Brief and chapter direction.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <textarea class="textarea textarea-bordered text-sm bg-base-100" name="core_topic" placeholder="In one sentence, what happens in this story?">{{ index .Intake "core_topic" }}</textarea>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Who Is It For?</label>
        <div class="tooltip tooltip-right" data-tip="Name the reader most likely to love this story. Specific is better than broad.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <textarea id="target_audience" class="textarea textarea-bordered text-sm bg-base-100" name="target_audience" placeholder="For example: readers who love slow-burn romance, twisty thrillers, or epic fantasy">{{ index .Intake "target_audience" }}</textarea>
    <div class="mt-1 flex flex-wrap gap-2">
        <button type="button" class="btn btn-xs btn-outline btn-secondary font-bold" title="Suggest a reader profile that fits this premise." hx-post="/api/project/{{ .ID }}/escape-hatch?field=target_audience" hx-target="#target_audience" hx-swap="outerHTML">Suggest Reader</button>
    </div>
    <p class="text-[11px] text-base-content/60">If you don’t know, let the app suggest a reader profile based on the premise.</p>
</div>
<div class="form-control">
    <div class="flex justify-between items-center mb-1 gap-2">
        <div class="flex items-center gap-2">
            <label class="label-text font-semibold">Why Do They Care?</label>
            <div class="tooltip tooltip-right" data-tip="What feeling or promise keeps the reader turning pages?">
                <span class="badge badge-ghost badge-sm">?</span>
            </div>
        </div>
        <button type="button" class="btn btn-xs btn-outline btn-secondary font-bold" title="Generate premise-aware wording to help fill this field." hx-post="/api/project/{{ .ID }}/escape-hatch?field=reader_hunger" hx-target="#reader_hunger" hx-swap="outerHTML">Show a Hint</button>
    </div>
    <textarea id="reader_hunger" name="reader_hunger" class="textarea textarea-bordered h-24 text-sm bg-base-100" placeholder="What feeling, promise, or problem keeps the reader hooked?">{{ index .Intake "reader_hunger" }}</textarea>
    <p class="text-[11px] text-base-content/60">This button gives you starter ideas; nothing is saved until you save the intake.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Point of View</label>
        <div class="tooltip tooltip-right" data-tip="Choose how the story is narrated. First person feels intimate; third person is more flexible.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <select name="narrative_pov" class="select select-bordered select-sm w-full bg-base-100">
        <option value="" {{ if eq (index .Intake "narrative_pov") "" }}selected{{ end }}>Choose a POV</option>
        <option value="first_person" {{ if eq (index .Intake "narrative_pov") "first_person" }}selected{{ end }}>First Person - "I" / immediate voice</option>
        <option value="third_limited" {{ if eq (index .Intake "narrative_pov") "third_limited" }}selected{{ end }}>Third Person Limited - one character close-up</option>
        <option value="third_omniscient" {{ if eq (index .Intake "narrative_pov") "third_omniscient" }}selected{{ end }}>Third Person Omniscient - broad overview</option>
        <option value="dual_pov" {{ if eq (index .Intake "narrative_pov") "dual_pov" }}selected{{ end }}>Dual POV - two main voices</option>
        <option value="second_person" {{ if eq (index .Intake "narrative_pov") "second_person" }}selected{{ end }}>Second Person - "you" / experimental</option>
    </select>
    <p class="text-[11px] text-base-content/60">Examples: "I saw the door open" for first person or "She saw the door open" for third person limited.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Story Pattern</label>
        <div class="tooltip tooltip-right" data-tip="Pick the kind of story engine you want. Some genres work best with a known formula.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <select name="structure_model" class="select select-bordered select-sm w-full bg-base-100">
        <option value="" {{ if eq (index .Intake "structure_model") "" }}selected{{ end }}>Choose a pattern</option>
        <option value="hero_journey" {{ if eq (index .Intake "structure_model") "hero_journey" }}selected{{ end }}>Hero's Journey</option>
        <option value="three_act" {{ if eq (index .Intake "structure_model") "three_act" }}selected{{ end }}>Three-Act Structure</option>
        <option value="save_the_cat" {{ if eq (index .Intake "structure_model") "save_the_cat" }}selected{{ end }}>Save the Cat / Beat Sheet</option>
        <option value="romance_beats" {{ if eq (index .Intake "structure_model") "romance_beats" }}selected{{ end }}>Romance Beat Sheet</option>
        <option value="mystery_trail" {{ if eq (index .Intake "structure_model") "mystery_trail" }}selected{{ end }}>Mystery / Clue Trail</option>
        <option value="quest_arc" {{ if eq (index .Intake "structure_model") "quest_arc" }}selected{{ end }}>Quest / Adventure Arc</option>
        <option value="character_arc" {{ if eq (index .Intake "structure_model") "character_arc" }}selected{{ end }}>Character-Driven Arc</option>
    </select>
    <p class="text-[11px] text-base-content/60">Examples: Hero's Journey, Three-Act, romance beats, or a mystery clue trail.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Avoid This</label>
        <div class="tooltip tooltip-right" data-tip="Use this to block tropes, twists, or directions you do not want.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <textarea id="avoid-textarea" class="textarea textarea-bordered text-sm bg-base-100" name="prohibited_directions" placeholder="What genre traps or story directions should be avoided?">{{ index .Intake "prohibited_directions" }}</textarea>
    <div class="mt-1 flex flex-wrap gap-2">
        <button type="button" class="btn btn-xs btn-outline btn-secondary font-bold" title="Suggest specific things this story should avoid." hx-post="/api/project/{{ .ID }}/escape-hatch?field=prohibited_directions" hx-target="#avoid-textarea" hx-swap="outerHTML">Show Avoid Examples</button>
    </div>
    <p class="text-[11px] text-base-content/60">Think of this as a guardrail list: wrong tropes, wrong tone, wrong pace, or the wrong amount of detail.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Ending Effect</label>
        <div class="tooltip tooltip-right" data-tip="What should stay with the reader after the final page.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <textarea class="textarea textarea-bordered text-sm bg-base-100" name="author_intent" placeholder="What should linger after the final page?">{{ index .Intake "author_intent" }}</textarea>
    <p class="text-[11px] text-base-content/60">This is the ending effect. Use it to tell the app what the story should leave behind.</p>
</div>
<div class="form-control gap-1">
    <div class="flex items-center gap-2">
        <label class="label-text font-semibold">Tone</label>
        <div class="tooltip tooltip-right" data-tip="A short style label for the voice of the story.">
            <span class="badge badge-ghost badge-sm">?</span>
        </div>
    </div>
    <input class="input input-bordered input-sm bg-base-100" name="selected_tone" value="{{ index .Intake "selected_tone" }}" placeholder="Tone, e.g. Lyrical, Tense, Wry" />
    <p class="text-[11px] text-base-content/60">Example tones: lyrical, tense, witty, intimate, ominous, hopeful.</p>
</div>
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
        <div class="alert alert-success mt-6">Intake saved. Next: generate the Book Brief, review it, then move to the outline and chapter shells.</div>
        {{ else }}
        <div class="alert alert-info mt-6">Fill the intake fields, save them, then generate the Book Brief.</div>
        {{ end }}
    </div>
</div>
{{- end }}

{{ define "brief_workspace" -}}
<div class="card bg-base-100 shadow border border-base-300 flex-1 overflow-hidden">
    <div class="card-body overflow-y-auto">
        {{ if .SaveNotice }}
        <div class="alert alert-success py-2 px-3 text-xs leading-relaxed mb-3">
            <span class="font-bold">{{ .SaveNotice }}</span>
        </div>
        {{ end }}
        <div class="alert alert-info py-2 px-3 text-xs leading-relaxed mb-3">
            <span><span class="font-bold">Book Brief:</span> this is the saved contract used for the outline, chapter drafting, editing, polishing, and export. Edit it here, then save before building the next stage.</span>
        </div>
        <div class="mx-auto w-52 max-w-full rounded-lg bg-primary text-primary-content shadow-xl p-5 aspect-[2/3] flex flex-col justify-between">
            <div>
                <div class="text-xs font-black uppercase opacity-80">Working Title</div>
                <h2 class="text-2xl font-black leading-tight mt-3">{{ .Brief.Title }}</h2>
                <p class="text-xs font-semibold mt-3 opacity-80">{{ .Brief.Subtitle }}</p>
            </div>
            <div class="text-xs font-semibold opacity-80">Book Brief</div>
        </div>

        <div class="divider">Book Brief</div>
        <form class="space-y-3 text-sm" hx-post="/api/project/{{ .Project.ID }}/brief" hx-target="#workspace-panel" hx-indicator="#brief-save-indicator">
            <div class="grid gap-3 md:grid-cols-2">
                <div class="form-control gap-1">
                    <label class="label-text font-black uppercase text-xs text-base-content/60">Title</label>
                    <input class="input input-bordered input-sm bg-base-100" name="title" value="{{ .Brief.Title }}" required />
                </div>
                <div class="form-control gap-1">
                    <label class="label-text font-black uppercase text-xs text-base-content/60">Subtitle</label>
                    <input class="input input-bordered input-sm bg-base-100" name="subtitle" value="{{ .Brief.Subtitle }}" />
                </div>
            </div>
            <div class="form-control gap-1">
                <label class="label-text font-black uppercase text-xs text-base-content/60">Promise</label>
                <textarea class="textarea textarea-bordered text-sm bg-base-100" name="promise">{{ .Brief.Promise }}</textarea>
            </div>
            <div class="form-control gap-1">
                <label class="label-text font-black uppercase text-xs text-base-content/60">Voice Tone</label>
                <textarea class="textarea textarea-bordered text-sm bg-base-100" name="voice_tone">{{ .Brief.VoiceTone }}</textarea>
            </div>
            <div class="form-control gap-1">
                <label class="label-text font-black uppercase text-xs text-base-content/60">What It Is</label>
                <textarea class="textarea textarea-bordered text-sm bg-base-100 min-h-28" name="what_it_is">{{ .Brief.WhatItIs }}</textarea>
            </div>
            <div class="form-control gap-1">
                <label class="label-text font-black uppercase text-xs text-base-content/60">What It Is Not</label>
                <textarea class="textarea textarea-bordered text-sm bg-base-100 min-h-28" name="what_it_is_not">{{ .Brief.WhatItIsNot }}</textarea>
            </div>
            <div class="form-control gap-1 bg-base-200 border border-base-300 rounded-lg p-3">
                <label class="label-text font-black uppercase text-xs text-base-content/60">Structural Suggestions</label>
                <textarea class="textarea textarea-bordered text-sm bg-base-100 min-h-28" name="ai_suggestions">{{ .Brief.AISuggestions }}</textarea>
            </div>
            <div class="flex flex-wrap items-center justify-between gap-2">
                <p class="text-xs text-base-content/60">Saved changes affect newly generated outlines and all later chapter work.</p>
                <button class="btn btn-sm btn-primary font-bold">
                    <span id="brief-save-indicator" class="loading loading-spinner loading-xs htmx-indicator"></span>
                    Save Brief
                </button>
            </div>
        </form>
    </div>
</div>
{{- end }}

{{ define "workspace_tabs" -}}
<div class="tabs tabs-lift bg-base-100 flex-1 overflow-hidden">
    <input type="radio" name="workspace_tabs_{{ .Project.ID }}" class="tab text-xs font-bold" aria-label="Metadata" {{ if eq .WorkspaceTab "metadata" }}checked{{ end }} />
    <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto">
        {{ template "metadata_workspace" . }}
    </div>

    {{ if .Brief }}
    <input type="radio" name="workspace_tabs_{{ .Project.ID }}" class="tab text-xs font-bold" aria-label="Book Brief" {{ if eq .WorkspaceTab "brief" }}checked{{ end }} />
    <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto">
        {{ template "brief_workspace" . }}
    </div>
    {{ end }}

    {{ if .Chapters }}
    <input type="radio" name="workspace_tabs_{{ .Project.ID }}" class="tab text-xs font-bold" aria-label="Outline" {{ if eq .WorkspaceTab "outline" }}checked{{ end }} />
    <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto">
        {{ template "outline_workspace" . }}
    </div>
    {{ end }}

    {{ if .Chapter }}
    <input type="radio" name="workspace_tabs_{{ .Project.ID }}" class="tab text-xs font-bold" aria-label="Drafting" {{ if eq .WorkspaceTab "drafting" }}checked{{ end }} />
    <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto">
        {{ template "chapter_cockpit" . }}
    </div>
    {{ end }}
</div>
{{- end }}

{{ define "metadata_workspace" -}}
<div class="card bg-base-100 shadow border border-base-300 flex-1 overflow-hidden">
    <div class="card-body overflow-y-auto">
        {{ if .SaveNotice }}
        <div class="alert alert-success py-2 px-3 text-xs leading-relaxed mb-3">
            <span class="font-bold">{{ .SaveNotice }}</span>
        </div>
        {{ end }}
        <div class="alert alert-info py-2 px-3 text-xs leading-relaxed mb-4">
            <span><span class="font-bold">Metadata workspace:</span> prose is only one field in the chapter. This is the control layer for publishing, architecture, style, chapter function, arcs, concept ownership, media prompts, and future beat tracking.</span>
        </div>

        <div class="grid gap-4">
            <form class="border border-base-300 bg-base-100 rounded-lg p-4 space-y-3" hx-post="/api/project/{{ .Project.ID }}/metadata" hx-target="#workspace-panel" hx-indicator="#book-metadata-save-indicator">
                <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                        <h2 class="font-black text-lg">Book Architecture</h2>
                        <p class="text-xs text-base-content/70 leading-relaxed">Saved project-level metadata used by export now and prompt guidance later.</p>
                    </div>
                    <button class="btn btn-sm btn-primary font-bold">
                        <span id="book-metadata-save-indicator" class="loading loading-spinner loading-xs htmx-indicator"></span>
                        Save Book Metadata
                    </button>
                </div>

                <div class="grid gap-3 md:grid-cols-2">
                    <div class="form-control gap-1">
                        <label class="label-text font-bold text-xs uppercase text-base-content/60">Author / Pen Name</label>
                        <input class="input input-bordered input-sm bg-base-100" name="author_name" value="{{ .Project.AuthorName }}" placeholder="Name shown in EPUB, HTML, and Markdown exports" />
                    </div>
                    <div class="form-control gap-1">
                        <label class="label-text font-bold text-xs uppercase text-base-content/60">Architecture Mode</label>
                        <input class="input input-bordered input-sm bg-base-100" name="architecture_mode_hint" value="{{ .Project.BookType }}" disabled />
                    </div>
                </div>

                <div class="form-control gap-1">
                    <label class="label-text font-bold text-xs uppercase text-base-content/60">Publishing Metadata JSON</label>
                    <textarea class="textarea textarea-bordered bg-base-100 font-mono text-xs min-h-32" name="publishing_metadata_json" placeholder='{"subtitle":"","blurb":"","categories":[],"keywords":[],"cover_prompt":""}'>{{ .Project.PublishingMetadataJSON }}</textarea>
                </div>
                <div class="form-control gap-1">
                    <label class="label-text font-bold text-xs uppercase text-base-content/60">Book Architecture JSON</label>
                    <textarea class="textarea textarea-bordered bg-base-100 font-mono text-xs min-h-40" name="book_architecture_json" placeholder='{"protagonist_arc":"","reader_transformation":"","major_turning_points":[],"ending_transformation":""}'>{{ .Project.BookArchitectureJSON }}</textarea>
                </div>
                <div class="form-control gap-1">
                    <label class="label-text font-bold text-xs uppercase text-base-content/60">Global Style Contract JSON</label>
                    <textarea class="textarea textarea-bordered bg-base-100 font-mono text-xs min-h-40" name="global_style_contract_json" placeholder='{"primary_voice":[],"avoid_voice":[],"tone":"","pacing":"","watchlist_phrases":[]}'>{{ .Project.GlobalStyleContractJSON }}</textarea>
                </div>
                <p class="text-xs text-base-content/60">Blank is allowed. If present, each JSON field must be valid JSON.</p>
            </form>

            {{ if .Chapter }}
            <form class="border border-base-300 bg-base-100 rounded-lg p-4 space-y-3" hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/metadata" hx-target="#workspace-panel" hx-indicator="#chapter-metadata-save-indicator">
                <div class="flex flex-wrap items-start justify-between gap-3">
                    <div>
                        <h2 class="font-black text-lg">Chapter {{ .Chapter.SortOrder }} Control Layer</h2>
                        <p class="text-xs text-base-content/70 leading-relaxed">Metadata for chapter function, arc movement, concept jurisdiction, generation directives, and media prompts.</p>
                    </div>
                    <button class="btn btn-sm btn-primary font-bold">
                        <span id="chapter-metadata-save-indicator" class="loading loading-spinner loading-xs htmx-indicator"></span>
                        Save Chapter Metadata
                    </button>
                </div>

                <div class="grid gap-3 md:grid-cols-2">
                    <div class="form-control gap-1">
                        <label class="label-text font-bold text-xs uppercase text-base-content/60">Chapter Title</label>
                        <input class="input input-bordered input-sm bg-base-100" name="title" value="{{ .Chapter.Title }}" required />
                    </div>
                    <div class="form-control gap-1">
                        <label class="label-text font-bold text-xs uppercase text-base-content/60">Subtitle</label>
                        <input class="input input-bordered input-sm bg-base-100" name="subtitle" value="{{ .Chapter.Subtitle }}" />
                    </div>
                </div>
                <div class="grid gap-3 md:grid-cols-2">
                    <div class="form-control gap-1">
                        <label class="label-text font-bold text-xs uppercase text-base-content/60">Front Matter Label</label>
                        <input class="input input-bordered input-sm bg-base-100" name="front_matter_label" value="{{ .Chapter.FrontMatterLabel }}" placeholder="The Field Note" />
                    </div>
                    <div class="form-control gap-1">
                        <label class="label-text font-bold text-xs uppercase text-base-content/60">Chapter Function</label>
                        <input class="input input-bordered input-sm bg-base-100" name="chapter_function_hint" value="{{ .Chapter.Purpose }}" disabled />
                    </div>
                </div>
                <div class="form-control gap-1">
                    <label class="label-text font-bold text-xs uppercase text-base-content/60">Front Matter Blurb</label>
                    <textarea class="textarea textarea-bordered bg-base-100 text-sm min-h-20" name="front_matter_blurb">{{ .Chapter.FrontMatterBlurb }}</textarea>
                </div>
                <div class="form-control gap-1">
                    <label class="label-text font-bold text-xs uppercase text-base-content/60">Chapter Metadata JSON</label>
                    <textarea class="textarea textarea-bordered bg-base-100 font-mono text-xs min-h-32" name="chapter_metadata_json" placeholder='{"chapter_function":"","required_elements":[],"avoid":[]}'>{{ .Chapter.ChapterMetadataJSON }}</textarea>
                </div>
                <div class="form-control gap-1">
                    <label class="label-text font-bold text-xs uppercase text-base-content/60">Arc Metadata JSON</label>
                    <textarea class="textarea textarea-bordered bg-base-100 font-mono text-xs min-h-36" name="arc_metadata_json" placeholder='{"arc_phase":"","pov_character":"","external_plot_movement":"","internal_shift":"","relationship_shift":"","beats":[]}'>{{ .Chapter.ArcMetadataJSON }}</textarea>
                </div>
                <div class="form-control gap-1">
                    <label class="label-text font-bold text-xs uppercase text-base-content/60">Concept Jurisdiction JSON</label>
                    <textarea class="textarea textarea-bordered bg-base-100 font-mono text-xs min-h-36" name="concept_jurisdiction_json" placeholder='{"owns":[],"may_reference":[],"must_not_reteach":[],"forbidden_phrases":[]}'>{{ .Chapter.ConceptJurisdiction }}</textarea>
                </div>
                <div class="form-control gap-1">
                    <label class="label-text font-bold text-xs uppercase text-base-content/60">Generation Directives JSON</label>
                    <textarea class="textarea textarea-bordered bg-base-100 font-mono text-xs min-h-36" name="generation_directives_json" placeholder='{"write_only_prose":true,"required_elements":[],"avoid":[],"revision_notes":""}'>{{ .Chapter.GenerationDirectives }}</textarea>
                </div>
                <div class="form-control gap-1">
                    <label class="label-text font-bold text-xs uppercase text-base-content/60">Media Prompts JSON</label>
                    <textarea class="textarea textarea-bordered bg-base-100 font-mono text-xs min-h-36" name="media_prompts_json" placeholder='{"images":[{"placement":"chapter_open","prompt":"","alt_text":"","caption":""}],"diagrams":[]}'>{{ .Chapter.MediaPromptsJSON }}</textarea>
                </div>
                <p class="text-xs text-base-content/60">Beat-level editing is deferred, but arc metadata can already reserve a <span class="font-mono">beats</span> array.</p>
            </form>
            {{ else }}
            <div class="alert alert-info text-sm">
                <span>Generate an outline or open a chapter to edit chapter-level metadata. Book-level metadata can be saved now.</span>
            </div>
            {{ end }}
        </div>
    </div>
</div>
{{- end }}

{{ define "outline_workspace" -}}
<div class="card bg-base-100 shadow border border-base-300 flex-1 overflow-hidden">
    <div class="card-body overflow-y-auto">
        <div class="alert alert-info py-2 px-3 text-xs leading-relaxed mb-3">
            <span><span class="font-bold">Outline view:</span> these chapter cards are the current table of contents. They show what the app plans to draft next.</span>
        </div>
        <div class="flex items-start justify-between gap-4">
            <div>
                <h2 class="card-title text-xl font-black">Outline</h2>
                <p class="text-sm text-base-content/70">Review the chapter plan before drafting. These cards are the working table of contents.</p>
            </div>
            <span class="badge badge-primary font-mono uppercase">{{ .Project.Status }}</span>
        </div>

        {{ if .Brief }}
        <div class="collapse collapse-arrow bg-base-200 border border-base-300 rounded-lg mt-4">
            <input type="checkbox" />
            <div class="collapse-title font-bold">Brief: {{ .Brief.Title }}</div>
            <div class="collapse-content text-sm space-y-2">
                <p><span class="font-bold">Promise:</span> {{ .Brief.Promise }}</p>
                <p><span class="font-bold">Voice:</span> {{ .Brief.VoiceTone }}</p>
                <p><span class="font-bold">Boundary:</span> {{ .Brief.WhatItIsNot }}</p>
            </div>
        </div>
        {{ end }}

        <div class="grid gap-3 mt-4">
            {{ range .Chapters }}
            <div class="border border-base-300 bg-base-100 rounded-lg p-4">
                <div class="flex items-start justify-between gap-3">
                    <div class="min-w-0">
                        <h3 class="font-black">Chapter {{ .SortOrder }}: {{ .Title }}</h3>
                        <p class="text-sm mt-1 leading-relaxed">{{ .Purpose }}</p>
                        <div class="grid gap-1 mt-3 text-xs text-base-content/70">
                            <p><span class="font-bold">Entry:</span> {{ .StateStart }}</p>
                            <p><span class="font-bold">Exit:</span> {{ .StateEnd }}</p>
                        </div>
                        <div class="flex flex-wrap gap-1 mt-3">
                            {{ if .ChapterMetadataJSON }}<span class="badge badge-success badge-sm">Chapter metadata</span>{{ else }}<span class="badge badge-warning badge-sm">Needs chapter metadata</span>{{ end }}
                            {{ if .ArcMetadataJSON }}<span class="badge badge-success badge-sm">Arc</span>{{ else }}<span class="badge badge-ghost badge-sm">No arc</span>{{ end }}
                            {{ if .ConceptJurisdiction }}<span class="badge badge-success badge-sm">Concepts</span>{{ else }}<span class="badge badge-ghost badge-sm">No concepts</span>{{ end }}
                            {{ if .GenerationDirectives }}<span class="badge badge-success badge-sm">Directives</span>{{ else }}<span class="badge badge-warning badge-sm">Needs directives</span>{{ end }}
                            {{ if .MediaPromptsJSON }}<span class="badge badge-info badge-sm">Media prompts</span>{{ else }}<span class="badge badge-ghost badge-sm">No media</span>{{ end }}
                        </div>
                        {{ if or .ChapterMetadataJSON .ArcMetadataJSON .ConceptJurisdiction .GenerationDirectives .MediaPromptsJSON }}
                        <div class="collapse collapse-arrow bg-base-200 border border-base-300 rounded-lg mt-3">
                            <input type="checkbox" />
                            <div class="collapse-title text-xs font-bold uppercase text-base-content/70">Metadata control layer</div>
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
                    </div>
                    <div class="flex flex-col items-end gap-2">
                        <span class="badge badge-outline font-mono text-[10px] uppercase">{{ .Status }}</span>
                        <button class="btn btn-xs btn-primary" title="Open this chapter in the drafting cockpit." hx-get="/app/project/{{ $.Project.ID }}/chapters/{{ .ID }}/workspace" hx-target="#workspace-panel">Open</button>
                        <button class="btn btn-xs btn-outline" title="Edit metadata for this chapter." hx-get="/app/project/{{ $.Project.ID }}/chapters/{{ .ID }}/workspace?tab=metadata" hx-target="#workspace-panel">Metadata</button>
                    </div>
                </div>
            </div>
            {{ end }}
        </div>

        <div class="alert alert-info mt-4">
            <span>Next step: chapter cards still need approve/edit/regenerate controls before drafting uses them as approved structure.</span>
        </div>
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

    <div class="bg-base-100 border-b border-base-300 px-4 py-2 flex flex-wrap items-center justify-between gap-2">
        <div class="flex flex-wrap gap-1">
            {{ range .Chapters }}
            <button class="btn btn-xs {{ if eq .ID $.Chapter.ID }}btn-primary{{ else }}btn-outline{{ end }}" title="Open chapter {{ .SortOrder }}." hx-get="/app/project/{{ $.Project.ID }}/chapters/{{ .ID }}/workspace" hx-target="#workspace-panel">{{ .SortOrder }}</button>
            {{ end }}
        </div>
        <div class="flex gap-2">
            {{ if .PrevChapter }}
            <button class="btn btn-xs btn-outline" title="Open the previous chapter." hx-get="/app/project/{{ .Project.ID }}/chapters/{{ .PrevChapter.ID }}/workspace" hx-target="#workspace-panel">Previous</button>
            {{ end }}
            {{ if .NextChapter }}
            <button class="btn btn-xs btn-outline" title="Open the next chapter." hx-get="/app/project/{{ .Project.ID }}/chapters/{{ .NextChapter.ID }}/workspace" hx-target="#workspace-panel">Next</button>
            {{ end }}
        </div>
    </div>

    <div class="px-4 pt-3 flex flex-wrap gap-2 text-[11px] font-bold uppercase tracking-wider">
        <span class="badge {{ if .Chapter.RawDraft }}badge-success{{ else }}badge-ghost{{ end }}">Draft {{ if .Chapter.RawDraft }}done{{ else }}pending{{ end }}</span>
        <span class="badge {{ if .Chapter.EditorialDiagnosis }}badge-success{{ else }}badge-ghost{{ end }}">Feedback {{ if .Chapter.EditorialDiagnosis }}done{{ else }}pending{{ end }}</span>
        <span class="badge {{ if .Chapter.TargetedRewrite }}badge-success{{ else }}badge-ghost{{ end }}">Rewrite {{ if .Chapter.TargetedRewrite }}done{{ else }}pending{{ end }}</span>
        <span class="badge {{ if .Chapter.DraftContent }}badge-success{{ else }}badge-ghost{{ end }}">Finish {{ if .Chapter.DraftContent }}done{{ else }}pending{{ end }}</span>
    </div>

    <div class="alert alert-info py-2 px-3 text-xs leading-relaxed mx-4 mt-4">
        <span><span class="font-bold">How this works:</span> the top tabs show the outputs you have already generated. The bottom buttons run the next stage and then refresh this same screen.</span>
    </div>

    <div class="tabs tabs-lifted bg-base-100 px-4 pt-2 shrink-0">
        <input type="radio" name="cockpit_tabs_{{ .Chapter.ID }}" class="tab text-xs font-bold" aria-label="1. Draft" checked />
        <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
            <div class="prose max-w-none text-sm leading-relaxed whitespace-pre-wrap">{{ .Chapter.RawDraft }}</div>
        </div>

        <input type="radio" name="cockpit_tabs_{{ .Chapter.ID }}" class="tab text-xs font-bold" aria-label="2. Editor Feedback" />
        <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
            <div class="mb-3 text-xs text-base-content/70 leading-relaxed">
                This is the editor's critique. It points out weak structure, pacing, voice problems, and places that sound too generic.
            </div>
            <div class="bg-warning/10 border border-warning/30 p-4 rounded-lg text-sm font-mono text-warning-content whitespace-pre-wrap">{{ .Chapter.EditorialDiagnosis }}</div>
            <div class="mt-3 text-xs text-base-content/70 leading-relaxed">
                Use the box below if you want to add notes before rewriting.
            </div>
            <div class="mt-4 flex gap-2">
                <input type="text" id="manual-correction-{{ .Chapter.ID }}" name="user_diagnosis_notes" placeholder="Add your notes before rewriting..." class="input input-bordered input-sm flex-1 text-sm" title="Add your own correction notes before re-running the rewrite step." />
                <button class="btn btn-sm btn-warning font-bold" title="Apply your notes and regenerate the rewritten chapter." hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/rewrite" hx-include="#manual-correction-{{ .Chapter.ID }}" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Apply Notes + Rewrite</button>
            </div>
            <p class="mt-2 text-xs text-base-content/60 leading-relaxed">This button does two things: it uses the diagnosis plus your notes, then it replaces this cockpit with the rewritten result.</p>
        </div>

        <input type="radio" name="cockpit_tabs_{{ .Chapter.ID }}" class="tab text-xs font-bold" aria-label="3. Rewrite" />
        <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
            <div class="prose max-w-none text-sm leading-relaxed whitespace-pre-wrap">{{ .Chapter.TargetedRewrite }}</div>
        </div>

        <input type="radio" name="cockpit_tabs_{{ .Chapter.ID }}" class="tab text-xs font-bold" aria-label="4. Final Draft" />
        <div class="tab-content bg-base-100 border-base-300 p-4 overflow-y-auto max-h-[400px]">
            <div class="prose max-w-none text-base leading-relaxed font-serif p-2 bg-base-100 rounded-lg border border-base-200 whitespace-pre-wrap">{{ .Chapter.DraftContent }}</div>
        </div>
    </div>

    <div class="border-t border-base-300 bg-base-100 px-4 py-2 text-[11px] text-base-content/60 font-semibold uppercase tracking-wider flex items-center justify-between">
        <span>Run controls</span>
        <span>These buttons create or replace the tabs above</span>
    </div>

    <div class="p-3 bg-base-200 border-t border-base-300 shrink-0 flex flex-wrap justify-end gap-2">
        <button class="btn btn-xs btn-accent font-bold" title="Run draft, feedback, rewrite, and finish in one pass for this chapter." hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/full" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Run All</button>
        <button class="btn btn-xs btn-outline btn-neutral" title="Create the first chapter draft. After this runs, tab 1 changes." hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/draft" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Draft</button>
        <button class="btn btn-xs btn-outline btn-warning" title="Generate the editor feedback. After this runs, tab 2 changes." hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/diagnose" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Feedback</button>
        <button class="btn btn-xs btn-outline btn-secondary" title="Rewrite the chapter using the feedback and your notes. After this runs, tab 3 changes." hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/rewrite" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Rewrite</button>
        <button class="btn btn-xs btn-success font-bold" title="Do the final cleanup pass. After this runs, tab 4 changes." hx-post="/api/project/{{ .Project.ID }}/chapters/{{ .Chapter.ID }}/pipeline/polish" hx-target="#chapter-cockpit-{{ .Chapter.ID }}" hx-swap="outerHTML">Finish</button>
    </div>
</div>
{{- end }}

{{ define "autopilot" -}}
<div class="card bg-neutral text-neutral-content shadow-xl border border-neutral p-4">
    <div class="card-body p-2 flex flex-col gap-3">
        <h3 class="card-title text-lg font-black tracking-tight text-primary">Build</h3>
        <p class="text-xs text-neutral-content/80 leading-relaxed">These are the main run buttons. Use them one at a time if you want to inspect each result.</p>
        <button class="btn btn-primary btn-md font-bold shadow w-full mt-2" title="Generate the reviewable brief from the intake." hx-post="/api/project/{{ .Project.ID }}/generate/brief" hx-include="#book-setup-form" hx-target="#workspace-panel">Brief</button>
        <button class="btn btn-secondary btn-md font-bold shadow w-full" title="Generate the outline after the brief is ready." hx-post="/api/project/{{ .Project.ID }}/generate/toc" hx-include="#book-setup-form" hx-target="#workspace-panel">Outline</button>
        <div class="divider my-1 text-neutral-content/50">OR</div>
        <p class="text-xs text-neutral-content/70 leading-relaxed">Auto mode skips the checkpoints and runs the whole pipeline.</p>
        <button class="btn btn-outline btn-sm border-neutral-content/40 text-neutral-content" title="Run the full pipeline automatically without stopping to review each stage." hx-post="/api/project/{{ .Project.ID }}/generate/autopilot" hx-include="#book-setup-form" hx-target="#workspace-panel">Auto</button>
        <button class="btn btn-sm btn-error text-error-content" title="Force rebuild: regenerate brief, outline, and all chapter stages from scratch." hx-confirm="Force rebuild this entire book from scratch? This will replace the current chapter cards and generated chapter text." hx-post="/api/project/{{ .Project.ID }}/generate/autopilot?force=1" hx-include="#book-setup-form" hx-target="#workspace-panel">Force Rebuild</button>
        <div id="project-status"></div>
    </div>
</div>
{{- end }}

{{ define "project_list" -}}
<div class="card bg-base-100 shadow border border-base-300 p-4">
    <div class="card-body p-2 flex flex-col gap-3">
        <div class="flex items-center justify-between gap-2">
            <h3 class="card-title text-base font-black">Books</h3>
            <span class="badge badge-ghost">{{ len .AllProjects }}</span>
        </div>
        <form class="flex flex-col gap-2" hx-post="/api/projects" hx-target="#main-layout-grid" hx-swap="outerHTML">
            <label class="text-xs font-bold uppercase text-base-content/60">Create New Book</label>
            <div class="join w-full">
                <input class="input input-sm input-bordered join-item min-w-0 flex-1" name="title" placeholder="New book title" required />
                <button class="btn btn-sm btn-primary join-item" title="Create a new named book project.">Create</button>
            </div>
        </form>
        <div class="flex flex-col gap-2">
            {{ range .AllProjects }}
            <button class="btn btn-sm h-auto min-h-10 justify-start text-left {{ if eq .ID $.ActiveProjectID }}btn-primary{{ else }}btn-outline{{ end }}" title="Open {{ .Title }}." hx-get="/app/project/select?project_id={{ .ID }}" hx-target="#main-layout-grid" hx-swap="outerHTML">
                <span class="flex flex-col items-start min-w-0">
                    <span class="font-bold truncate max-w-full">{{ .Title }}</span>
                    <span class="text-[10px] opacity-70 uppercase">{{ .Status }} - {{ .BookType }}</span>
                </span>
            </button>
            {{ end }}
        </div>
    </div>
</div>
{{- end }}

{{ define "exports_panel" -}}
<div class="card bg-base-100 shadow border border-base-300 p-4">
    <div class="card-body p-2 flex flex-col gap-3">
        <h3 class="card-title text-base font-black">Exports</h3>
        <p class="text-xs text-base-content/70 leading-relaxed">Create a cleaned EPUB, rendered HTML preview, Markdown source file, and lint report from the current manuscript text.</p>
        <button class="btn btn-sm btn-primary font-bold" title="Export cleaned reader files and report." hx-post="/api/project/{{ .Project.ID }}/exports/book" hx-target="#export-status">Export Book</button>
        <div id="export-status"></div>
    </div>
</div>
{{- end }}

{{ define "export_result" -}}
<div class="alert alert-success flex flex-col items-start gap-2">
    <span class="font-bold">Export complete</span>
    <div class="flex flex-wrap gap-2">
        <a class="btn btn-xs btn-primary" href="{{ .EPUBURL }}" download>EPUB</a>
        <a class="btn btn-xs btn-outline" href="{{ .HTMLURL }}" target="_blank" rel="noopener">HTML Preview</a>
        <a class="btn btn-xs btn-outline" href="{{ .MarkdownURL }}" download>Markdown Source</a>
        <a class="btn btn-xs btn-outline" href="{{ .LintReportURL }}" download>Lint Report</a>
        {{ if .StyleReportURL }}
        <a class="btn btn-xs btn-outline" href="{{ .StyleReportURL }}" download>Style Report</a>
        {{ end }}
    </div>
    {{ if .Warnings }}
    <div class="text-xs leading-relaxed">
        <div class="font-bold">Notes</div>
        <ul class="list-disc pl-4">
            {{ range .Warnings }}
            <li>{{ . }}</li>
            {{ end }}
        </ul>
    </div>
    {{ end }}
</div>
{{- end }}

{{ define "export_wait" -}}
<div class="alert alert-warning flex items-start gap-2">
    <span class="loading loading-spinner loading-sm"></span>
    <span>{{ .Message }}</span>
</div>
{{- end }}

{{ define "export_blocked" -}}
<div class="alert alert-warning flex flex-col items-start gap-2">
    <span class="font-bold">{{ .Message }}</span>
    <ul class="list-disc pl-4 text-xs leading-relaxed">
        {{ range .Warnings }}
        <li>{{ . }}</li>
        {{ end }}
    </ul>
    <div class="flex flex-wrap gap-2">
        <button class="btn btn-xs btn-primary" hx-post="/api/project/{{ .ProjectID }}/generate/autopilot" hx-target="#workspace-panel">Run Auto</button>
        <button class="btn btn-xs btn-outline" hx-post="/api/project/{{ .ProjectID }}/exports/book?allow_incomplete=1" hx-target="#export-status">Export Draft Anyway</button>
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
<div class="alert alert-info" {{ if .ChapterID }}hx-get="/api/project/{{ .ProjectID }}/chapters/{{ .ChapterID }}/status"{{ else }}hx-get="/api/project/{{ .ProjectID }}/status"{{ end }} hx-trigger="every 2s" hx-swap="outerHTML">
    <span class="loading loading-spinner loading-sm"></span>
    <span>Working on {{ stageLabel .Stage }}{{ chapterSuffix .ChapterID }}.</span>
</div>
{{- end }}

{{ define "status" -}}
<div id="project-status" class="alert {{ statusClass .Status }}">
    {{ if eq .Status "running" }}<span class="loading loading-spinner loading-sm"></span>{{ end }}
    <span>{{ jobLabel .JobType }}: {{ statusLabel .Status }}{{ chapterSuffix .ChapterID }}{{ if .ErrorMessage }} - {{ .ErrorMessage }}{{ end }}</span>
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
