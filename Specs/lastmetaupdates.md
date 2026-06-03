Remaining Change Specification
Book Builder Simple: Go / PocketBase Metadata-Aware Exporter
Purpose

This document defines the remaining modifications for the Go-based Book Builder Simple project.

The project already has:

PocketBase-backed project and chapter records
chapter metadata fields
arc metadata fields
concept jurisdiction fields
generation directive fields
media prompt fields
front matter fields
Markdown / HTML / EPUB export
heading cleanup
front matter injection
conservative list normalization
duplicate block detection
lint report generation

The remaining work should build on the current Go implementation.

Do not redesign the system.

Do not convert the project to YAML.

Do not regenerate existing chapters unless explicitly requested.

Do not use Python.

Do not use OpenAI image generation.

Do not use gpt-image-2.

Image generation will be handled later through ComfyUI.

For now, media/image prompts are metadata only.

Core Rule

The canonical source of truth is the PocketBase structured project/book data.

A chapter is not just prose.

A chapter is a structured object containing:

title
subtitle
sort order
purpose
reader start state
reader end state
front matter
chapter metadata JSON
arc metadata JSON
concept jurisdiction JSON
generation directives JSON
media prompts JSON
prose/draft content
editorial diagnosis
targeted rewrite
manual edits
user notes
status fields

The prose is one field inside the chapter object.

The exporter must render final manuscript content from this structured object.

Stop Worrying About Image Models

The programming agent must stop focusing on this error:

The model 'gpt-image-2' does not exist.

That is irrelevant to the current implementation.

The project will not use gpt-image-2.

The project will not call OpenAI image generation during export.

The project will eventually use ComfyUI for image generation, as a separate later stage.

Book export must work with image generation completely disabled.

Image / Media Prompt Policy

For now:

Preserve media_prompts_json.
Validate media_prompts_json as JSON when possible.
Optionally render media prompt placeholders in Markdown / HTML export.
Do not call any image model.
Do not hardcode any OpenAI image model.
Do not make image generation a dependency of Markdown, HTML, or EPUB export.
If media prompt JSON is invalid, write a lint warning and continue export.

Future ComfyUI support belongs in a separate job/stage.

Remaining Modification 1: Expand Exporter Chapter Struct

The database Chapter model already contains rich metadata fields.

The exporter Chapter struct should carry enough of those fields to support metadata-aware linting and future export behavior.

Update internal/exporter/exporter.go.

Current exporter chapter struct is too small.

Expand it to include:

type Chapter struct {
	ID               string
	SortOrder        int
	Title            string
	Subtitle         string
	Body             string
	Source           string
	FrontMatterLabel string
	FrontMatterBlurb string

	Purpose              string
	StateStart           string
	StateEnd             string
	ChapterMetadataJSON  string
	ArcMetadataJSON      string
	ConceptJurisdiction  string
	GenerationDirectives string
	MediaPromptsJSON     string
}

Do not remove existing fields.

Do not break existing tests.

Remaining Modification 2: Validate Chapter Metadata JSON During Export

Add export-time validation for optional JSON metadata fields.

Fields to validate:

ChapterMetadataJSON
ArcMetadataJSON
ConceptJurisdiction
GenerationDirectives
MediaPromptsJSON

Empty fields are allowed.

Non-empty fields must be valid JSON.

Invalid JSON should not fail export by default.

Instead, add a lint issue.

Example lint issue:

{
  "type": "invalid_metadata_json",
  "chapter_number": 3,
  "severity": "medium",
  "message": "media_prompts_json is not valid JSON.",
  "suggested_action": "Fix the JSON in the chapter metadata field."
}

Suggested functions:

func ValidateChapterMetadata(chapter Chapter) []LintIssue
func validateOptionalJSONField(chapter Chapter, fieldName string, value string) []LintIssue

Call this from CleanBook() or ValidateBookForExport().

Remaining Modification 3: Render Media Prompt Placeholders

Add optional rendering of media/image prompt placeholders.

This does not generate images.

This only preserves image intent in the exported manuscript when enabled.

Suggested JSON shape for media_prompts_json:

[
  {
    "id": "img_03_01",
    "placement": "after_opening_scene",
    "type": "illustration",
    "prompt": "A stressed parent standing near a parked car during a custody exchange, dusk light, subtle emotional tension, realistic editorial illustration.",
    "alt_text": "A parent grounding themselves beside a car before a custody exchange.",
    "caption": "The body often reacts before the mind has language for what is happening.",
    "status": "planned",
    "output_path": ""
  }
]

Suggested Go struct:

type MediaPrompt struct {
	ID         string `json:"id"`
	Placement  string `json:"placement"`
	Type       string `json:"type"`
	Prompt     string `json:"prompt"`
	AltText    string `json:"alt_text"`
	Caption    string `json:"caption"`
	Status     string `json:"status"`
	OutputPath string `json:"output_path"`
}

Suggested functions:

func ParseMediaPrompts(chapter Chapter) ([]MediaPrompt, []LintIssue)
func RenderMediaPromptPlaceholders(chapter Chapter) (string, []LintIssue)
func InsertMediaPromptPlaceholders(markdown string, chapter Chapter) (string, []LintIssue)

For now, simplest acceptable behavior:

append media prompt placeholders at the end of the chapter body
later, placement-aware insertion can be added

Markdown placeholder format:

<!-- MEDIA PROMPT:
chapter: 3
placement: after_opening_scene
type: illustration
prompt: A stressed parent standing near a parked car during a custody exchange, dusk light, subtle emotional tension, realistic editorial illustration.
alt_text: A parent grounding themselves beside a car before a custody exchange.
caption: The body often reacts before the mind has language for what is happening.
-->

If placeholder rendering is disabled, do not render media prompts.

Remaining Modification 4: Add Concept Jurisdiction Linting

The project already has concept_jurisdiction_json.

Use it for simple phrase-based checks first.

Do not attempt full semantic concept detection yet.

Suggested JSON shape:

{
  "owns": [
    "somatic tracking during custody exchanges"
  ],
  "may_reference": [
    "autonomic nervous system",
    "freeze response"
  ],
  "must_not_reteach": [
    "Polyvagal Theory"
  ],
  "forbidden_phrases": [
    "Polyvagal Theory is",
    "The autonomic nervous system is",
    "To understand the nervous system"
  ]
}

Required behavior:

Parse ConceptJurisdiction.
If forbidden_phrases are present, search the chapter body for them.
If found, emit lint issues.
Do not rewrite text automatically.

Suggested lint issue:

{
  "type": "concept_jurisdiction_violation",
  "chapter_number": 8,
  "severity": "medium",
  "message": "Chapter uses a forbidden re-teaching phrase: Polyvagal Theory is",
  "suggested_action": "Reference the concept briefly or move the explanation to the owning chapter."
}

Suggested Go struct:

type ConceptJurisdictionRules struct {
	Owns             []string `json:"owns"`
	MayReference     []string `json:"may_reference"`
	MustNotReteach   []string `json:"must_not_reteach"`
	ForbiddenPhrases []string `json:"forbidden_phrases"`
}

Suggested functions:

func AuditConceptJurisdiction(chapter Chapter) []LintIssue
func AuditBookConceptJurisdiction(book Book) []LintIssue

Call this during CleanBook() or immediately after CleanBook().

Remaining Modification 5: Add Basic Style Metrics Report

Do not implement full LLM style rewriting yet.

Add deterministic style metrics first.

Suggested Go struct:

type StyleMetrics struct {
	ChapterNumber          int      `json:"chapter_number"`
	ChapterID              string   `json:"chapter_id"`
	WordCount              int      `json:"word_count"`
	AvgSentenceLength      float64  `json:"avg_sentence_length"`
	AvgParagraphLength     float64  `json:"avg_paragraph_length"`
	QuestionCount          int      `json:"question_count"`
	EmDashCount            int      `json:"em_dash_count"`
	BulletCount            int      `json:"bullet_count"`
	RepeatedOpeners        []string `json:"repeated_openers"`
	WatchlistPhraseMatches []string `json:"watchlist_phrase_matches"`
}

Suggested functions:

func AnalyzeStyleMetrics(chapter Chapter) StyleMetrics
func AnalyzeBookStyleMetrics(book Book) []StyleMetrics
func WriteStyleReport(path string, metrics []StyleMetrics) error

Write report to:

exports/reports/<slug>-style.json

This should not block export.

Remaining Modification 6: Add Watchlist Phrase Detection

Use a default watchlist first.

Later, this can be pulled from global_style_contract_json.

Initial watchlist examples:

it is important to understand
as we discussed earlier
this is not about
at the end of the day
you are not broken
your nervous system is trying to protect you

If a phrase appears too often, emit a lint issue.

Suggested function:

func DetectWatchlistPhrases(chapter Chapter, phrases []string) []LintIssue

This is report-only.

No automatic rewriting.

Remaining Modification 7: Improve Heading Stripping Safety

Current heading stripping removes any # or ## heading in the first five non-empty lines.

This is acceptable for current generated drafts, but should be tightened.

Improved behavior:

Strip leading headings only if they match one of these:

Contains Chapter
Contains the chapter title
Contains the chapter subtitle
Is a top-level heading immediately followed by another heading
Appears before any real paragraph prose

Do not remove legitimate internal section headings later in the chapter.

Keep the existing tests.

Add tests for:

heading matching chapter title
heading matching chapter subtitle
first real prose before heading means do not strip
later ## section heading is preserved
Remaining Modification 8: Add Export Options

Add export options so features can be enabled gradually.

Suggested Go struct:

type ExportOptions struct {
	OutputDir                    string
	RenderMediaPromptPlaceholders bool
	WriteStyleReport              bool
	WriteLintReport               bool
	WriteHTML                     bool
	WriteEPUB                     bool
}

Default behavior should preserve current exports.

Suggested default:

func DefaultExportOptions() ExportOptions {
	return ExportOptions{
		OutputDir:                     "exports",
		RenderMediaPromptPlaceholders: false,
		WriteStyleReport:              true,
		WriteLintReport:               true,
		WriteHTML:                     true,
		WriteEPUB:                     true,
	}
}

Do not make media placeholder rendering mandatory.

Do not make ComfyUI mandatory.

Remaining Modification 9: Update Export Result

The current Result struct has Markdown, EPUB, HTML, and lint report paths/URLs.

Add optional style report fields:

StyleReportURL  string
StyleReportPath string

Only populate them if style report writing is enabled.

Remaining Modification 10: Preserve Existing Behavior

All current tests must continue passing.

Existing export behavior should remain functional:

Markdown export
HTML export
EPUB export
lint report export
heading cleanup
front matter injection
list normalization
duplicate detection

Do not break current route handlers.

Do not rename existing exported files unless necessary.

Do not remove current fields.

Do not remove current status values.

Suggested Implementation Order

Implement in this order:

Step 1

Expand exporter Chapter struct to carry metadata fields.

Step 2

Add metadata JSON validation.

Step 3

Add concept jurisdiction phrase linting.

Step 4

Add media prompt parsing and optional placeholder rendering.

Step 5

Add basic style metrics report.

Step 6

Add watchlist phrase detection.

Step 7

Improve heading stripping safety.

Step 8

Add export options only if needed by current call sites.

Acceptance Criteria

The remaining work is complete when:

The exporter chapter struct can carry metadata fields.
Optional JSON metadata fields are validated during export.
Invalid metadata JSON creates lint issues but does not stop export.
media_prompts_json is preserved and parsed.
Media prompt placeholders can be rendered when enabled.
No image model is called during export.
No OpenAI image model is hardcoded.
ComfyUI is treated as a future separate image stage.
concept_jurisdiction_json can trigger simple forbidden-phrase lint warnings.
Basic style metrics can be written to a style report.
Watchlist phrases can be detected and reported.
Heading stripping is safer.
Current exporter behavior remains intact.
Current tests continue passing.
New tests cover metadata validation, concept linting, media prompt placeholders, and style metrics.
New Tests to Add

Add tests for:

ValidateChapterMetadata
validateOptionalJSONField
ParseMediaPrompts
RenderMediaPromptPlaceholders
InsertMediaPromptPlaceholders
AuditConceptJurisdiction
AnalyzeStyleMetrics
AnalyzeBookStyleMetrics
DetectWatchlistPhrases
WriteStyleReport
Safer StripLeadingDuplicateHeadings behavior
Test: Invalid Metadata JSON

Input:

Chapter{
	SortOrder: 3,
	Title: "Test",
	ChapterMetadataJSON: "{bad json",
}

Expected:

invalid_metadata_json lint issue
export continues
Test: Valid Media Prompts

Input:

[
  {
    "id": "img_01",
    "placement": "end",
    "type": "illustration",
    "prompt": "A quiet desk with manuscript pages.",
    "alt_text": "A quiet desk with manuscript pages.",
    "caption": "The working table."
  }
]

Expected:

media prompt parsed successfully
optional placeholder rendered if enabled
Test: No Image Generation

Exporting a chapter with media prompts must not call:

OpenAI image API
gpt-image-2
ComfyUI
any external image service
Test: Concept Jurisdiction

Input:

Chapter{
	SortOrder: 8,
	ConceptJurisdiction: `{"forbidden_phrases":["Polyvagal Theory is"]}`,
	Body: "Polyvagal Theory is a framework that explains the nervous system."
}

Expected:

concept_jurisdiction_violation
Test: Style Metrics

Input chapter body with:

questions
em dashes
bullets
repeated phrase

Expected:

metrics include question count, em dash count, bullet count, watchlist matches
Final Instruction to the Programming Agent

Continue in Go.

Use the existing PocketBase structure.

Keep metadata as first-class chapter data.

Do not convert to YAML.

Do not use Python.

Do not use gpt-image-2.

Do not call any image generation service during export.

Prepare media prompts for future ComfyUI use, but do not implement ComfyUI now.

Preserve current exporter behavior.

Add metadata validation, concept jurisdiction linting, optional media prompt placeholders, and basic style metrics.

Keep it boring, incremental, and testable.