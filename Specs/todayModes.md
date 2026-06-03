Change Specification
Go Book Compiler, Exporter, Metadata System, and Quality Pipeline
1. Purpose

Implement a full Go-based book compilation and export pipeline that treats a book as a structured document object stored in PocketBase.

The system must support:

Existing generated chapter prose.
Chapter-level metadata.
Book-level style contracts.
Chapter-specific generation directives.
Front matter injection.
Image and media prompts.
Concept jurisdiction rules.
Style modulation rules.
Structural Markdown normalization.
Duplicate heading removal.
Duplicate content detection.
Style drift auditing.
Controlled patch generation.
Final export to Markdown, with future support for DOCX, PDF, HTML, and other formats.

The system must not assume that a chapter is only prose.

A chapter is a structured object containing metadata, prose, generation controls, style controls, image prompts, continuity rules, export settings, and revision state.

2. Core Design Principle

The canonical source of truth is PocketBase structured data, not raw Markdown.

Raw generated prose is only one field inside a chapter record.

The exporter should render selected fields into the final reader-facing manuscript.

PocketBase Book Record
        ↓
PocketBase Chapter Records
        ↓
Go Internal Book Object
        ↓
Generation / Revision / Normalization / Linting / Export
        ↓
Final Manuscript Artifacts
3. Important Architectural Clarification

The system has two related but separate responsibilities:

3.1 Post-Generation Export Cleanup

This runs after chapter prose has already been generated.

It should:

preserve existing prose
clean headings
normalize Markdown
inject front matter
detect duplicate blocks
report style drift
export the final manuscript
3.2 Pre-Generation and Revision Guidance

This runs before new chapter prose is generated or before existing prose is revised.

It should use:

book metadata
chapter metadata
style contracts
jurisdiction rules
chapter-specific intent
prior chapter summaries
continuity rules
image prompts
required inclusions
avoid lists

The first implementation should not require full regeneration of existing chapters.

4. Storage Model

The project currently uses PocketBase collection documents.

Do not convert the project into YAML files.

However, the internal conceptual model should behave like a structured document.

The system should treat the book as if it were shaped like:

Book
  ├── metadata
  ├── style contract
  ├── audience profile
  ├── concept map
  ├── chapter order
  ├── export settings
  └── chapters
        ├── metadata
        ├── generation directives
        ├── style controls
        ├── concept jurisdiction
        ├── media prompts
        ├── prose
        ├── lint reports
        ├── revision state
        └── export state

PocketBase remains the actual persistence layer.

5. Suggested PocketBase Collections

Codex should adapt to the existing collection names and fields. Do not break current working behavior.

If fields or collections are missing, add only the minimum necessary fields.

5.1 books

Suggested fields:

id
title
subtitle
description
audience_profile
global_style_contract
concept_jurisdiction
export_settings
status
created
updated
5.2 chapters

Suggested fields:

id
book_id
chapter_number
sort_order
title
subtitle
chapter_type
purpose
reader_state
chapter_function
style_profile
concept_jurisdiction
required_elements
avoid
front_matter
image_prompts
diagram_prompts
research_notes
prose
summary
status
lint_status
export_status
created
updated
5.3 Optional chapter_revisions

Suggested fields:

id
chapter_id
revision_number
prose_snapshot
metadata_snapshot
change_reason
created
5.4 Optional lint_reports

Suggested fields:

id
book_id
chapter_id
report_type
severity
issues_json
created
6. Internal Go Data Structures

Codex should map existing PocketBase fields into internal Go structs.

6.1 Book
type Book struct {
    ID                  string
    Title               string
    Subtitle            string
    Description         string
    AudienceProfile     AudienceProfile
    GlobalStyleContract StyleContract
    ConceptJurisdiction ConceptJurisdiction
    ExportSettings      ExportSettings
    Chapters            []Chapter
    Status              string
}
6.2 Chapter
type Chapter struct {
    ID                  string
    BookID              string
    Number              int
    SortOrder           int
    Title               string
    Subtitle            string
    ChapterType         string
    Purpose             string
    ReaderState         string
    ChapterFunction     string
    StyleProfile        StyleProfile
    ConceptJurisdiction ConceptJurisdiction
    RequiredElements    []string
    Avoid               []string
    FrontMatter         FrontMatter
    ImagePrompts        []ImagePrompt
    DiagramPrompts      []DiagramPrompt
    ResearchNotes       []ResearchNote
    Prose               string
    Summary             string
    Status              string
    LintStatus          string
    ExportStatus        string
}
6.3 StyleProfile
type StyleProfile struct {
    Tone              string
    HumorLevel        int
    IntensityLevel    int
    TechnicalDepth    int
    EmotionalWarmth   int
    PracticalityLevel int
    TheoryLevel       int
    Pace              string
    SentenceTexture   string
    Notes             string
}
6.4 StyleContract
type StyleContract struct {
    PrimaryVoice      []string
    AvoidVoice        []string
    ReaderStance      string
    HumorRules        HumorRules
    SentenceRules     SentenceRules
    ChapterShape      []string
    LexicalWatchlist  []string
    PreferredPhrases  map[string]string
}
6.5 ConceptJurisdiction
type ConceptJurisdiction struct {
    Owns           []string
    MayReference   []string
    MustNotReteach []string
    ForbiddenTerms []string
    Notes          string
}
6.6 FrontMatter
type FrontMatter struct {
    Label string
    Blurb string
}
6.7 ImagePrompt
type ImagePrompt struct {
    ID          string
    Placement   string
    Type        string
    Prompt      string
    AltText     string
    Caption     string
    Status      string
    OutputPath  string
}
6.8 LintIssue
type LintIssue struct {
    Type             string `json:"type"`
    BookID           string `json:"book_id,omitempty"`
    ChapterID        string `json:"chapter_id,omitempty"`
    ChapterNumber    int    `json:"chapter_number,omitempty"`
    ParagraphIndex   int    `json:"paragraph_index,omitempty"`
    Severity         string `json:"severity"`
    Message          string `json:"message"`
    OriginalText     string `json:"original_text,omitempty"`
    SuggestedAction  string `json:"suggested_action,omitempty"`
    MatchingChapter  int    `json:"matching_chapter,omitempty"`
    MatchingTextHash string `json:"matching_text_hash,omitempty"`
}
7. Pipeline Overview

The full pipeline should support this flow:

[PocketBase Book + Chapter Records]
        ↓
[Load Book Object]
        ↓
[Validate Metadata]
        ↓
[Optional Generation / Revision Prompt Build]
        ↓
[Normalize Existing Prose]
        ↓
[Normalize Headings]
        ↓
[Inject Front Matter]
        ↓
[Render Image Placeholders]
        ↓
[Normalize Markdown]
        ↓
[Detect Duplicate Blocks]
        ↓
[Audit Concept Jurisdiction]
        ↓
[Audit Style Drift]
        ↓
[Optional Controlled Patches]
        ↓
[Render Final Markdown]
        ↓
[Write Reports]
        ↓
[Update Export Status]
8. Phase 1: Load Book From PocketBase
Requirement

Load the selected book and all associated chapters from PocketBase.

Behavior
Fetch the book record.
Fetch all related chapter records.
Map PocketBase fields into internal Go structs.
Sort chapters by sort_order or chapter_number.
Validate that chapter numbers are present and sequential.
Validate that every chapter has a title and prose field.
Do not regenerate missing prose automatically.
Functions
func LoadBookFromPocketBase(bookID string) (Book, error)
func LoadChaptersForBook(bookID string) ([]Chapter, error)
func SortChapters(chapters []Chapter) []Chapter
func ValidateBookForExport(book Book) []LintIssue
9. Phase 2: Chapter Metadata System
Requirement

Each chapter must be treated as a structured object with metadata and prose.

The prose field is not the whole chapter.

Metadata Usage

Chapter metadata should be usable for:

generation prompts
revision prompts
heading rendering
front matter injection
image prompt placement
list normalization expectations
style drift auditing
concept redundancy auditing
export decisions
Required Chapter Metadata Fields

Minimum useful chapter metadata:

chapter_number
sort_order
title
subtitle
purpose
reader_state
chapter_function
style_profile
concept_jurisdiction
required_elements
avoid
front_matter
image_prompts
prose
status
Rule

The exporter must never assume chapter prose contains reliable title, subtitle, or chapter number data.

PocketBase metadata owns those fields.

10. Phase 3: Canonical Heading Normalization
Problem

Generated prose may contain duplicate or conflicting headings.

Example:

## Chapter 6: The Bridge of Small Things: Designing Micro-Habits for Administrative Re-entry
# Chapter 1: The Bridge of Small Things
Requirement

The final manuscript must contain exactly one canonical rendered heading per chapter.

The heading must be built from PocketBase metadata.

Output Format

If title only:

## Chapter 3: Rewilding the Nervous System

If title and subtitle:

## Chapter 3: Rewilding the Nervous System: Practical Polyvagal Strategies for Multi-Household Stress
Heading Strip Rule

Only strip generated headings from the beginning of the prose field.

Search only the first five non-empty lines.

Strip lines matching:

^#\s+Chapter\s+\d+[:.\-\s]+.*$
^##\s+Chapter\s+\d+[:.\-\s]+.*$
^#\s+.*$
^##\s+.*$

Do not remove legitimate section headings later in the chapter.

Functions
func BuildCanonicalChapterHeading(ch Chapter) string
func StripLeadingDuplicateHeadings(markdown string) (string, []LintIssue)
func NormalizeChapterHeading(ch Chapter) (Chapter, []LintIssue)
11. Phase 4: Front Matter Injection
Requirement

Render chapter front matter from chapter metadata.

Front matter appears beneath the canonical chapter heading and before the prose.

Output Format
> **The Field Note:**
> *Welcome to the custody exchange: the only place on Earth where a missing pair of dinosaur pajamas triggers the exact same evolutionary panic as a saber-toothed tiger.*

---
Behavior
If chapter.FrontMatter.Blurb is present, render it.
If no front matter is present, skip unless global defaults are enabled.
Do not double-inject front matter if prose already begins with a field note.
Field note label should default to "The Field Note" only if no label is provided and a blurb exists.
Functions
func HasExistingFrontMatter(markdown string) bool
func RenderFrontMatter(f FrontMatter) string
func InjectFrontMatter(markdown string, f FrontMatter) (string, []LintIssue)
12. Phase 5: Image and Media Prompt Metadata
Requirement

Chapter records may contain image prompts, diagram prompts, or illustration prompts.

These are metadata, not prose.

They should be available to:

generation prompts
image-generation tooling
manuscript export
HTML export
future PDF/DOCX export
Image Prompt Structure
type ImagePrompt struct {
    ID          string
    Placement   string
    Type        string
    Prompt      string
    AltText     string
    Caption     string
    Status      string
    OutputPath  string
}
Rendering Behavior

For Markdown export, render image placeholders only if enabled in export settings.

Example:

![A parent grounding themselves beside a car before a custody exchange.](images/ch03_custody_exchange.png)

*The body often reacts before the mind has language for what is happening.*

If the image has not been generated yet, render a placeholder comment only if configured:

<!-- IMAGE PROMPT img_03_01:
A stressed parent standing near a parked car during a custody exchange...
-->
Functions
func RenderImagePlaceholder(img ImagePrompt, settings ExportSettings) string
func InsertImagePlaceholders(markdown string, images []ImagePrompt, settings ExportSettings) string
13. Phase 6: Markdown Spacing Cleanup
Requirement

Normalize obvious Markdown formatting problems without changing wording.

Behavior

The exporter may:

trim trailing spaces
normalize excessive blank lines
ensure one blank line after headings
ensure one blank line before lists
ensure one blank line before and after blockquotes
preserve paragraph content

The exporter must not rewrite prose in this phase.

Function
func NormalizeMarkdownSpacing(markdown string) string
14. Phase 7: Conservative List Normalization
Problem

Generated prose may flatten protocols, inventories, and checklists into dense paragraphs.

Requirement

Convert only obvious structures into Markdown lists.

If uncertain, leave text unchanged.

Numbered List Triggers

Detect sequential markers such as:

Step 1:
Step 2:
Phase 1:
Phase 2:
First:
Second:
Third:
Action Step:
Bullet List Triggers

Detect inventory/checklist patterns such as:

Consider these signs:
Look for the following:
Audit these specific shifts:
These include:

Followed by repeated label-and-description patterns.

Output Examples

Numbered list:

1. **Find the Exhale:** Consciously extend the length of your breath out.
2. **Lower the Center of Gravity:** Drop your body into a seated or crouching position.
3. **Execute Near-Far Eye Tracking:** Anchor your focus on a distinct, close object.

Bullet list:

* **Kinesthetic Rigidity:** Your shoulders rise automatically toward your ears.
* **Visual Tunneling:** Your eyes dart rapidly across peripheral zones.
Functions
func NormalizeMarkdownLists(markdown string) (string, []LintIssue)
func DetectSequentialProtocol(block string) bool
func DetectInventoryList(block string) bool
15. Phase 8: Duplicate Block Detection
Requirement

Detect severe accidental duplication across chapters.

This is deterministic and should be implemented in Go.

Do not use an LLM for exact duplicate detection.

Behavior
Split chapters into paragraph/block units.
Normalize whitespace.
Lowercase for comparison.
Ignore short blocks.
Hash normalized block.
Detect repeated hashes across different chapters.
Report duplicate blocks.
Suggested Defaults
const MinDuplicateBlockChars = 180
const MinDuplicateBlockWords = 30
Ignore List

Ignore small standard structural phrases:

The Field Note
Chapter Summary
Reflection Questions
Key Takeaways

Do not ignore large repeated body sections.

Functions
func DetectDuplicateBlocks(chapters []Chapter, cfg DuplicateConfig) []LintIssue
func SplitIntoBlocks(markdown string) []string
func NormalizeBlockForHash(block string) string
func HashBlock(normalized string) string
16. Phase 9: Concept Jurisdiction Audit
Purpose

Prevent conceptual repetition and over-explanation across long books.

Some chapters own concepts.

Later chapters may reference those concepts but should not re-teach them.

Example
{
  "concept": "Polyvagal Theory",
  "owner_chapters": [1, 2, 3],
  "later_reference_rule": "May be referenced briefly but not re-explained.",
  "banned_intro_phrases": [
    "To understand the nervous system",
    "The autonomic nervous system is",
    "Polyvagal Theory teaches us"
  ]
}
Requirement

Use chapter metadata to detect likely concept repetition.

Behavior

For each chapter:

Read chapter concept jurisdiction metadata.
Identify concepts the chapter owns.
Identify concepts it may reference.
Identify concepts it must not re-teach.
Detect banned intro phrases.
Report jurisdiction violations.
Functions
func AuditConceptJurisdiction(book Book) []LintIssue
func DetectForbiddenConceptPhrases(ch Chapter, rules ConceptJurisdiction) []LintIssue
func DetectConceptReteaching(ch Chapter, rules ConceptJurisdiction) []LintIssue
Safety Rule

The Go system should report concept jurisdiction issues.

It should not automatically rewrite conceptual repetition unless a controlled patching stage is explicitly enabled.

17. Phase 10: Style Consistency and Drift Audit
Purpose

Detect whether chapters drift away from the intended book style or from their own declared chapter style profile.

The system must distinguish between accidental drift and intentional modulation.

Principle

Style consistency does not mean every chapter sounds identical.

It means every chapter varies intentionally within the declared design.

Inputs

Use:

book-level style contract
chapter-level style profile
chapter purpose
reader state
chapter function
lexical watchlist
overused phrase list
sentence metrics
paragraph metrics
Deterministic Go Metrics

Compute:

type StyleMetrics struct {
    ChapterNumber          int      `json:"chapter_number"`
    WordCount              int      `json:"word_count"`
    AvgSentenceLength      float64  `json:"avg_sentence_length"`
    AvgParagraphLength     float64  `json:"avg_paragraph_length"`
    QuestionCount          int      `json:"question_count"`
    EmDashCount            int      `json:"em_dash_count"`
    BulletCount            int      `json:"bullet_count"`
    RepeatedOpeners        []string `json:"repeated_openers"`
    WatchlistPhraseMatches []string `json:"watchlist_phrase_matches"`
}
Detect

The auditor should flag:

chapters much longer than target
chapters much shorter than target
excessive questions
excessive em dashes
repeated sentence openings
repeated rhetorical structures
repeated reassurance formulas
overused watchlist phrases
tone drift from chapter style profile
technical depth drift
humor level drift
intensity level drift
Functions
func AnalyzeStyleMetrics(ch Chapter, cfg StyleConfig) StyleMetrics
func DetectRepeatedPhrases(chapters []Chapter, cfg StyleConfig) []LintIssue
func DetectRepeatedOpenings(chapters []Chapter, cfg StyleConfig) []LintIssue
func DetectChapterLengthDrift(chapters []Chapter, cfg StyleConfig) []LintIssue
func AuditStyleDrift(book Book) ([]StyleMetrics, []LintIssue)
18. Phase 11: LLM-Assisted Style Review Packets
Purpose

Some style issues require judgment.

The Go system should build structured review packets for an LLM/editor layer.

The LLM should not be given uncontrolled permission to rewrite whole chapters.

Review Packet Structure
type StyleReviewPacket struct {
    BookID              string        `json:"book_id"`
    ChapterID           string        `json:"chapter_id"`
    ChapterNumber       int           `json:"chapter_number"`
    ChapterTitle        string        `json:"chapter_title"`
    ChapterPurpose      string        `json:"chapter_purpose"`
    ReaderState         string        `json:"reader_state"`
    BookStyleContract   StyleContract `json:"book_style_contract"`
    ChapterStyleProfile StyleProfile  `json:"chapter_style_profile"`
    Issues              []LintIssue   `json:"issues"`
    Excerpts            []string      `json:"excerpts"`
}
Function
func BuildStyleReviewPacket(ch Chapter, issues []LintIssue, cfg StyleConfig) StyleReviewPacket
LLM Review Instruction

The LLM should return structured findings.

It should not rewrite the whole chapter by default.

Expected response shape:

{
  "chapter_number": 9,
  "issues": [
    {
      "type": "style_drift",
      "severity": "medium",
      "location_hint": "middle section",
      "message": "This section drifts toward generic therapeutic reassurance.",
      "suggested_action": "Tighten into concrete, practical language."
    }
  ]
}
19. Phase 12: Controlled Style Harmonization
Purpose

Patch only flagged passages.

Do not rewrite whole chapters unless explicitly requested.

Rule

Patching must be minimally invasive.

The patcher must preserve:

facts
meaning
sequence
structure
chapter purpose
reader-facing practical value
authorial voice
Patch Request Structure
type PatchRequest struct {
    ChapterID           string        `json:"chapter_id"`
    ChapterNumber       int           `json:"chapter_number"`
    Issue               LintIssue     `json:"issue"`
    OriginalText         string        `json:"original_text"`
    BookStyleContract    StyleContract `json:"book_style_contract"`
    ChapterStyleProfile  StyleProfile  `json:"chapter_style_profile"`
}
Patch Response Structure
type PatchResponse struct {
    ChapterID     string `json:"chapter_id"`
    OriginalText  string `json:"original_text"`
    RevisedText   string `json:"revised_text"`
    ChangeSummary string `json:"change_summary"`
    Confidence    string `json:"confidence"`
}
Patch Prompt Rules

The editor model must be instructed:

Revise only the provided passage.

Do not add new concepts.
Do not remove concrete instructions.
Do not soften the voice into generic therapy language.
Do not make the passage more academic.
Do not make the passage more mystical.
Do not change the factual meaning.
Preserve the author's grounded, direct, practical voice.
Return only the revised passage and a short change summary.
Functions
func BuildPatchRequest(ch Chapter, issue LintIssue) PatchRequest
func ApplyPatch(ch Chapter, patch PatchResponse) (Chapter, error)
func SaveChapterRevision(ch Chapter, reason string) error
Safety Rule

Before applying an LLM patch:

Save a revision snapshot.
Verify the original text exists exactly once.
Replace only that passage.
Log the change.
Mark chapter as patched or needs_review.
20. Phase 13: Generation Prompt Builder
Purpose

For future chapter generation or regeneration, build prompts from structured metadata.

Requirement

The prompt builder should assemble:

book title
audience profile
global style contract
chapter number
chapter title/subtitle
chapter purpose
reader state
chapter function
style profile
concept jurisdiction
required elements
avoid list
image prompts
previous chapter summaries
next chapter preview if available
Function
func BuildChapterGenerationPrompt(book Book, ch Chapter, context GenerationContext) string
Prompt Requirements

The generated prompt must instruct the model to produce only the chapter prose field.

The model must not generate metadata unless explicitly requested.

The model must not invent new chapter numbers or titles.

Prompt Skeleton
You are writing the prose field for Chapter {number} of the book "{book_title}".

Do not output metadata.
Do not output a chapter heading unless explicitly requested.
Do not include YAML, JSON, or front matter.
Write only the chapter prose.

BOOK STYLE CONTRACT:
{global_style_contract}

AUDIENCE PROFILE:
{audience_profile}

CHAPTER METADATA:
Title: {title}
Subtitle: {subtitle}
Purpose: {purpose}
Reader state: {reader_state}
Chapter function: {chapter_function}

CHAPTER STYLE PROFILE:
Tone: {tone}
Humor level: {humor_level}
Intensity level: {intensity_level}
Technical depth: {technical_depth}
Emotional warmth: {emotional_warmth}
Pace: {pace}

CONCEPT JURISDICTION:
Owns: {owns}
May reference: {may_reference}
Must not re-teach: {must_not_reteach}

REQUIRED ELEMENTS:
{required_elements}

AVOID:
{avoid}

Write the chapter prose now.
21. Phase 14: Revision Prompt Builder
Purpose

For revising existing chapters, use metadata and existing prose.

Function
func BuildChapterRevisionPrompt(book Book, ch Chapter, request RevisionRequest) string
Rule

Revision prompts must specify whether the model may:

lightly edit
patch specific issues
restructure
expand
shorten
regenerate

Default mode should be conservative.

Conservative Revision Instruction
Revise only the specified issue.
Preserve the chapter's existing structure and meaning.
Do not rewrite the entire chapter.
Do not add new concepts.
Do not remove examples, protocols, lists, or concrete instructions unless explicitly asked.
22. Phase 15: Final Markdown Rendering
Requirement

Render final manuscript Markdown from the structured Book object.

Final Markdown Should Include
Book title
Optional subtitle
Optional table of contents
Chapters in canonical order
Canonical chapter headings
Front matter
Image placeholders or rendered image links if enabled
Cleaned prose
Functions
func RenderBookMarkdown(book Book) string
func RenderTableOfContents(book Book) string
func RenderChapterMarkdown(ch Chapter, settings ExportSettings) string
func WriteMarkdownFile(path string, content string) error
Example Output
# Book Title

## Table of Contents

1. Chapter 1: Title
2. Chapter 2: Title
3. Chapter 3: Title

---

## Chapter 1: Title: Subtitle

> **The Field Note:**
> *Optional front matter blurb.*

---

Chapter prose begins here.
23. Phase 16: Reports
Required Reports

Write reports to:

/export/reports/manuscript_lint_report.json
/export/reports/style_metrics_report.json
/export/reports/concept_jurisdiction_report.json
/export/reports/export_summary.json

If current project conventions differ, adapt paths accordingly.

Functions
func WriteLintReport(path string, issues []LintIssue) error
func WriteStyleReport(path string, metrics []StyleMetrics, issues []LintIssue) error
func WriteExportSummary(path string, summary ExportSummary) error
Export Summary
type ExportSummary struct {
    BookID              string `json:"book_id"`
    Title               string `json:"title"`
    ChapterCount        int    `json:"chapter_count"`
    OutputPath          string `json:"output_path"`
    LintIssueCount      int    `json:"lint_issue_count"`
    HighSeverityIssues  int    `json:"high_severity_issues"`
    MediumSeverityIssues int   `json:"medium_severity_issues"`
    LowSeverityIssues   int    `json:"low_severity_issues"`
}
24. Phase 17: PocketBase Status Updates
Requirement

If existing code supports writeback, update export status.

Suggested statuses:

draft
generated
normalized
linted
needs_review
patched
exported
Functions
func UpdateBookExportStatus(bookID string, status string) error
func UpdateChapterExportStatus(chapterID string, status string) error
func SaveLintReportToPocketBase(report LintReport) error
Safety Rule

Do not add risky PocketBase writeback behavior if the exporter currently only reads data.

Prefer file reports first.

25. Full Export Flow
func ExportBook(bookID string, options ExportOptions) error {
    book, err := LoadBookFromPocketBase(bookID)
    if err != nil {
        return err
    }

    var allIssues []LintIssue
    var allMetrics []StyleMetrics

    validationIssues := ValidateBookForExport(book)
    allIssues = append(allIssues, validationIssues...)

    book.Chapters = SortChapters(book.Chapters)

    for i, ch := range book.Chapters {
        normalizedChapter, headingIssues := NormalizeChapterHeading(ch)
        allIssues = append(allIssues, headingIssues...)

        cleanedProse := NormalizeMarkdownSpacing(normalizedChapter.Prose)

        cleanedProse, listIssues := NormalizeMarkdownLists(cleanedProse)
        allIssues = append(allIssues, listIssues...)

        if normalizedChapter.FrontMatter.Blurb != "" {
            cleanedProse, fmIssues := InjectFrontMatter(cleanedProse, normalizedChapter.FrontMatter)
            allIssues = append(allIssues, fmIssues...)
        }

        cleanedProse = InsertImagePlaceholders(
            cleanedProse,
            normalizedChapter.ImagePrompts,
            book.ExportSettings,
        )

        normalizedChapter.Prose = cleanedProse
        book.Chapters[i] = normalizedChapter
    }

    duplicateIssues := DetectDuplicateBlocks(book.Chapters, DefaultDuplicateConfig())
    allIssues = append(allIssues, duplicateIssues...)

    conceptIssues := AuditConceptJurisdiction(book)
    allIssues = append(allIssues, conceptIssues...)

    styleMetrics, styleIssues := AuditStyleDrift(book)
    allMetrics = append(allMetrics, styleMetrics...)
    allIssues = append(allIssues, styleIssues...)

    finalMarkdown := RenderBookMarkdown(book)

    if err := WriteMarkdownFile(options.OutputMarkdownPath, finalMarkdown); err != nil {
        return err
    }

    if err := WriteLintReport(options.LintReportPath, allIssues); err != nil {
        return err
    }

    if err := WriteStyleReport(options.StyleReportPath, allMetrics, styleIssues); err != nil {
        return err
    }

    summary := BuildExportSummary(book, options.OutputMarkdownPath, allIssues)
    if err := WriteExportSummary(options.ExportSummaryPath, summary); err != nil {
        return err
    }

    if options.UpdatePocketBaseStatus {
        _ = UpdateBookExportStatus(book.ID, "exported")
    }

    return nil
}

Codex should adapt this pseudocode to the existing project structure.

26. Acceptance Criteria

The implementation is complete when:

The exporter loads book and chapter records from PocketBase.
Chapters are sorted by canonical order.
Chapter metadata is mapped into internal Go structs.
Final headings are rendered from metadata, not prose.
Duplicate generated headings at the beginning of prose are removed.
Existing prose is preserved unless narrow cleanup is performed.
Front matter is rendered from chapter metadata.
Front matter is not double-injected.
Markdown spacing is normalized.
Obvious list structures may be normalized conservatively.
Image prompts can be rendered as placeholders or skipped based on export settings.
Duplicate body blocks across chapters are detected.
Concept jurisdiction violations are reported.
Style drift metrics are generated.
Style drift issues are reported.
LLM review packets can be built for flagged style issues.
Controlled patch requests can be built for individual flagged passages.
Final Markdown is exported.
Lint, style, concept, and export summary reports are written.
The system does not require conversion to YAML.
The system does not require full regeneration of existing chapters.
The system does not perform uncontrolled whole-chapter rewrites.
27. Unit Tests

Add tests for:

LoadBookFromPocketBase mapping
SortChapters
ValidateBookForExport
BuildCanonicalChapterHeading
StripLeadingDuplicateHeadings
NormalizeChapterHeading
HasExistingFrontMatter
RenderFrontMatter
InjectFrontMatter
NormalizeMarkdownSpacing
NormalizeMarkdownLists
DetectDuplicateBlocks
AuditConceptJurisdiction
AnalyzeStyleMetrics
DetectRepeatedPhrases
DetectRepeatedOpenings
DetectChapterLengthDrift
RenderImagePlaceholder
RenderBookMarkdown
WriteLintReport
WriteStyleReport
BuildChapterGenerationPrompt
BuildChapterRevisionPrompt
BuildPatchRequest
ApplyPatch
28. Required Test Cases
28.1 Duplicate Heading

Input prose:

## Chapter 6: Wrong Exported Heading
# Chapter 1: Wrong Inner Heading

Actual prose begins here.

Expected cleaned prose:

Actual prose begins here.

Canonical heading is rendered separately from metadata.

28.2 Front Matter Injection

Input:

FrontMatter{
    Label: "The Field Note",
    Blurb: "This is the field note.",
}

Expected:

> **The Field Note:**
> *This is the field note.*

---
28.3 Existing Front Matter

Input prose already starts with:

> **The Field Note:**
> *Existing note.*

---

Expected:

Do not inject duplicate front matter.
28.4 Duplicate Block Detection

Two chapters contain the same long paragraph.

Expected:

High-severity duplicate_block issue.

Two chapters contain the same short phrase.

Expected:

No issue.
28.5 Concept Jurisdiction

Chapter 8 has must_not_reteach: ["Polyvagal Theory"].

Chapter 8 prose includes:

Polyvagal Theory is a framework that explains...

Expected:

concept_jurisdiction_violation
28.6 Style Drift

Chapter style profile:

humor_level: 1
technical_depth: 3
intensity_level: 5

Chapter prose contains excessive jokes, high theory density, and long academic explanations.

Expected:

style_drift issue
28.7 Conservative List Normalization

Input:

Step 1: Breathe out longer than you breathe in. Step 2: Sit down. Step 3: Look near, then far.

Expected:

1. **Breathe out longer than you breathe in.**
2. **Sit down.**
3. **Look near, then far.**

Only if detection is reliable.

If uncertain, leave unchanged.

29. Safety Rules

The system may automatically fix:

duplicate leading headings
wrong rendered chapter numbers
front matter placement
Markdown spacing
obvious duplicate generated headings
obvious exact duplicate blocks when configured
image placeholder placement

The system must not automatically perform:

full chapter rewrites
major tone edits
conceptual rewriting
large content deletion unless exact duplication is clear
new prose generation
new chapter generation
metadata redesign during export
unreviewed LLM patching
30. Recommended Implementation Order

Implement in this order:

Step 1: Book Loading and Mapping
Load book.
Load chapters.
Map PocketBase records to structs.
Sort chapters.
Step 2: Structural Export
Build canonical headings.
Strip duplicate prose headings.
Render Markdown.
Step 3: Front Matter and Image Metadata
Inject front matter.
Render image placeholders if enabled.
Step 4: Markdown Cleanup
Normalize spacing.
Conservative list normalization.
Step 5: Reports
Lint report.
Export summary.
Step 6: Duplicate Detection
Exact duplicate block detection.
Report only at first.
Optional auto-removal later.
Step 7: Concept and Style Audits
Concept jurisdiction report.
Style metrics report.
Style drift report.
Step 8: Controlled Patching
Build review packets.
Build patch requests.
Save revisions before applying patches.
Replace only exact flagged passages.
Step 9: Generation Prompt Builder
Use metadata for future chapter generation.
Generate only prose field.
Do not regenerate existing book unless explicitly requested.
31. Final Instruction to Codex

Implement the book pipeline as a structured PocketBase-driven Go system.

Do not convert the project to YAML.

Do not treat Markdown as the source of truth.

Do not treat chapter prose as the whole chapter.

Use PocketBase chapter metadata as the control layer for generation, linting, style auditing, media prompts, and export.

Preserve existing generated prose unless a narrow deterministic cleanup or reviewed patch is explicitly applied.

Build the system in small, testable functions.

Export a clean final Markdown manuscript.

Produce useful reports.

Prepare the system for richer metadata-driven generation and revision later.

we wer tzkig abou tincorporating metadata,still a thing?
Thought for a couple of seconds

Yes — absolutely still a thing.

In fact, metadata should be one of the central ideas in the full spec. The correction I’d make is that the current full document mentions metadata, but it may not emphasize it enough as the control layer for the whole book system.

The model should be:

PocketBase book record
  ├── book-level metadata
  ├── global style contract
  ├── audience profile
  ├── concept rules
  └── export settings

PocketBase chapter record
  ├── chapter metadata
  ├── chapter purpose
  ├── reader state
  ├── style profile
  ├── concept jurisdiction
  ├── required elements
  ├── avoid list
  ├── image prompts
  ├── front matter
  └── prose field

The key sentence is:

The prose is a field inside the chapter object. It is not the chapter object.

That should definitely be in the spec.

What metadata does

Chapter metadata should guide:

Stage	Metadata role
Generation	Builds the prompt for that chapter
Revision	Tells the editor what the chapter is supposed to do
Export	Supplies canonical title, subtitle, order, front matter
Style audit	Distinguishes intentional variation from accidental drift
Deduplication	Knows which concepts are allowed where
Image generation	Stores image/diagram prompts and placement
Fiction support	Tracks arcs, POV, tension, reveals, continuity
Nonfiction support	Tracks concept ownership, exercises, reader state, practical outcomes

This lines up with the original engineering spec’s idea of jurisdiction rules, front-matter hooks, and post-generation linting, but broadens it into a proper structured chapter model.

The part I would add to the full Codex document

Add a stronger section called:

Chapter Metadata as the Control Layer

With language like this:

Each chapter must be treated as a structured object, not as a Markdown file.

The chapter prose is one field inside that object.

Chapter metadata controls generation, revision, linting, style auditing, media prompt placement, concept jurisdiction, and final export rendering.

The exporter must render reader-facing manuscript content from the structured chapter object. It must not assume that raw prose contains reliable chapter title, subtitle, chapter number, front matter, or media placement information.
For PocketBase specifically

You do not need YAML. PocketBase can hold this structure just fine.

You can have fields such as:

chapter_number
sort_order
title
subtitle
purpose
reader_state
chapter_function
style_profile_json
concept_jurisdiction_json
required_elements_json
avoid_json
front_matter_json
image_prompts_json
prose
summary
status

That gives you the YAML-like mental model without changing storage.

For tonight/today’s full spec

Yes, metadata remains part of it. More than that: metadata is the thing that keeps a 20-chapter book from becoming a pile of loosely related generated prose.

The revised architectural sentence should be:

PocketBase stores the structured book object. Each chapter contains metadata plus prose. The compiler uses metadata to generate, revise, audit, normalize, and export the manuscript.