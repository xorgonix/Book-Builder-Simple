````
# Change Specification

# Go Book Exporter: Tonight’s Cleanup Pass

## Purpose

Update the existing Go-based book export pipeline so it can take the already-generated 20 chapter manuscript content from PocketBase, normalize the structure, fix visible formatting problems, remove obvious duplicate headings, inject chapter front matter where available, detect severe duplicated content, and export a clean Markdown manuscript.

This change is intentionally scoped for tonight’s practical goal:

> **Do not regenerate the book. Clean and export the existing generated chapter prose.**

The earlier specification identifies the same broad pipeline problems: stacked headings, mismatched chapter numbering, flattened lists, repeated concepts, dynamic front-matter hooks, and post-generation linting before final Markdown output. This document narrows that into a safe Go implementation for the current PocketBase-based system.

---

# Core Principle

The canonical source of truth is **not the raw generated Markdown heading inside the chapter prose**.

The source of truth is:

```
PocketBase book recordPocketBase chapter recordsCanonical chapter orderChapter title/subtitle metadataExisting chapter prose field
```

The prose is valuable and should be preserved unless the exporter is performing a narrow, deterministic cleanup.

---

# Tonight’s Scope

Implement a post-generation Go cleanup/export pass.

The exporter should:

1. Load the book and its chapters from PocketBase.
2. Sort chapters by canonical chapter number/order.
3. Build chapter headings from PocketBase fields.
4. Strip duplicate or conflicting generated headings from the beginning of each chapter prose field.
5. Inject front matter if available.
6. Normalize obvious Markdown spacing and simple list structures.
7. Detect exact or near-exact duplicated blocks across chapters.
8. Write the final manuscript to Markdown.
9. Write a lint report for anything suspicious.
10. Avoid broad prose rewriting.

---

# Explicit Non-Goals for Tonight

Do **not** implement full regeneration.

Do **not** convert PocketBase data into YAML.

Do **not** redesign the whole metadata model.

Do **not** implement full fiction continuity metadata.

Do **not** perform broad LLM-based chapter rewrites.

Do **not** let the cleanup pass “improve” the authorial voice across whole chapters.

Do **not** treat style harmonization as automatic rewriting.

Tonight’s job is:

```
existing generated chapters → clean exported manuscript
```

---

# Current Conceptual Pipeline

```
[PocketBase Book + Chapter Records]        ↓[Go Exporter Loads Book Object]        ↓[Sort Chapters by Canonical Order]        ↓[Normalize Chapter Headings]        ↓[Strip Duplicate Inner Headings]        ↓[Inject Optional Front Matter]        ↓[Normalize Simple Markdown Structures]        ↓[Detect Severe Duplicate Blocks]        ↓[Write Final Markdown]        ↓[Write Lint Report]
```

---

# Data Model Assumptions

The actual PocketBase collection names may differ. Codex should adapt to the current codebase.

The exporter should assume a structure broadly equivalent to:

```
type Book struct {    ID       string    Title    string    Subtitle string    Chapters []Chapter}
```

```
type Chapter struct {    ID             string    Number         int    Title          string    Subtitle       string    Prose          string    FrontMatter    FrontMatter    Status         string}
```

```
type FrontMatter struct {    Label string    Blurb string}
```

If current PocketBase fields have different names, map them into these internal structs inside the exporter layer.

---

# Required Output Files

The exporter should write:

```
/export/final_manuscript.md/export/reports/manuscript_lint_report.json
```

If the current project already uses a different export directory, follow the existing convention.

---

# Phase 1: Load Existing Book and Chapter Records

## Requirement

Load the existing book and associated chapters from PocketBase.

The exporter should not ask the generation system to create new chapter prose.

## Required Behavior

1. Fetch the selected book record.
2. Fetch all chapter records associated with the book.
3. Sort chapters by `chapter_number`, `number`, `sort_order`, or the existing canonical ordering field.
4. Validate that all expected chapters are present.
5. Continue export even if minor metadata is missing, but report missing fields in the lint report.

## Suggested Function Signatures

```
func LoadBookFromPocketBase(bookID string) (Book, error)
```

```
func SortChapters(chapters []Chapter) []Chapter
```

```
func ValidateBookForExport(book Book) []LintIssue
```

---

# Phase 2: Canonical Chapter Heading Normalization

## Problem

Generated chapter content may contain stacked or conflicting headings such as:

```
## Chapter 6: The Bridge of Small Things: Designing Micro-Habits for Administrative Re-entry# Chapter 1: The Bridge of Small Things
```

The generated prose may include an internal heading that conflicts with the actual chapter metadata.

## Rule

The exporter must build the final rendered chapter heading from PocketBase metadata, not from the generated prose.

## Required Behavior

For each chapter:

1. Build the canonical heading from chapter number, title, and subtitle.
2. Remove generated headings from the beginning of the prose field if they look like duplicate chapter headings.
3. Ensure the final rendered chapter begins with exactly one canonical heading.
4. Do not remove legitimate section headings later in the chapter.

## Output Format

If the chapter has only a title:

```
## Chapter 3: Rewilding the Nervous System
```

If the chapter has title and subtitle:

```
## Chapter 3: Rewilding the Nervous System: Practical Polyvagal Strategies for Multi-Household Stress
```

## Heading Stripping Rule

Only strip duplicate or conflicting headings from the first few non-empty lines of the prose field.

Suggested safety limit:

```
Only examine the first 5 non-empty lines.
```

Remove lines matching patterns like:

```
^#\s+Chapter\s+\d+[:.\-\s]+.*$^##\s+Chapter\s+\d+[:.\-\s]+.*$^#\s+.*$^##\s+.*$
```

But only at the beginning of the chapter body.

## Suggested Function Signatures

```
func BuildCanonicalChapterHeading(ch Chapter) string
```

```
func StripLeadingDuplicateHeadings(markdown string) (cleaned string, issues []LintIssue)
```

```
func NormalizeChapterHeading(ch Chapter) (Chapter, []LintIssue)
```

---

# Phase 3: Front Matter Injection

## Purpose

Some chapters may have a short “field note” or introductory blockquote beneath the chapter heading.

This should be rendered from chapter metadata if present.

## Required Behavior

If a chapter has front matter metadata, render it immediately below the canonical chapter heading.

If no front matter exists, skip injection unless the current codebase already has a default front-matter rule.

Avoid double-injecting field notes if the prose already begins with one.

## Output Format

```
> **The Field Note:**> *Welcome to the custody exchange: the only place on Earth where a missing pair of dinosaur pajamas triggers the exact same evolutionary panic as a saber-toothed tiger.*---
```

## Suggested Function Signatures

```
func HasExistingFrontMatter(markdown string) bool
```

```
func RenderFrontMatter(f FrontMatter) string
```

```
func InjectFrontMatter(markdown string, f FrontMatter) (string, []LintIssue)
```

## Safety Rule

Do not insert generic front matter unless explicitly configured.

For tonight, prefer:

```
if chapter has front_matter_blurb → injectelse → skip
```

---

# Phase 4: Simple Markdown Cleanup

## Purpose

Normalize obvious Markdown spacing problems without rewriting the prose.

## Required Behavior

The exporter should:

1. Normalize excessive blank lines.
2. Ensure one blank line after headings.
3. Ensure one blank line before and after blockquotes.
4. Ensure one blank line before lists.
5. Trim trailing spaces.
6. Preserve paragraph content.

## Suggested Function Signature

```
func NormalizeMarkdownSpacing(markdown string) string
```

## Safety Rule

This function should not change wording.

It should only change whitespace and Markdown structure.

---

# Phase 5: Conservative List Normalization

## Purpose

Generated content may flatten obvious protocols or inventories into dense paragraphs.

The exporter may normalize only obvious list structures.

## Required Behavior

Convert clear sequential blocks into numbered lists when they contain markers such as:

```
Step 1:Step 2:Phase 1:Phase 2:First:Second:Third:Action Step:
```

Convert clear inventory/checklist blocks into bullets when they contain repeated label-and-description patterns.

Example target output:

```
1. **Find the Exhale:** Consciously extend the length of your breath out.2. **Lower the Center of Gravity:** Drop your body into a seated or crouching position.3. **Execute Near-Far Eye Tracking:** Anchor your focus on a distinct, close object.
```

Example bullet output:

```
* **Kinesthetic Rigidity:** Your shoulders rise automatically toward your ears.* **Visual Tunneling:** Your eyes dart rapidly across peripheral zones.
```

## Conservative Rule

If the exporter is uncertain, leave the paragraph unchanged.

This feature must not attempt to perform broad semantic rewriting.

## Suggested Function Signatures

```
func NormalizeMarkdownLists(markdown string) (string, []LintIssue)
```

```
func DetectSequentialProtocol(block string) bool
```

```
func DetectInventoryList(block string) bool
```

---

# Phase 6: Duplicate Block Detection

## Purpose

Detect severe accidental duplication across chapters, especially copied paragraphs or repeated closing sections.

## Required Behavior

Split chapters into paragraph/block units.

For each block:

1. Trim whitespace.
2. Normalize repeated spaces.
3. Lowercase for comparison.
4. Ignore short blocks.
5. Hash normalized block.
6. Detect duplicate hashes across different chapters.
7. Write issues to the lint report.

## Suggested Defaults

```
const MinDuplicateBlockChars = 180const MinDuplicateBlockWords = 30
```

## Ignore Common Structural Blocks

Ignore common repeated labels or small standard elements such as:

```
The Field NoteChapter SummaryReflection QuestionsKey Takeaways
```

But do not ignore large repeated body sections.

## Suggested Function Signatures

```
func DetectDuplicateBlocks(chapters []Chapter, cfg DuplicateConfig) []LintIssue
```

```
func SplitIntoBlocks(markdown string) []string
```

```
func NormalizeBlockForHash(block string) string
```

```
func HashBlock(normalized string) string
```

---

# Phase 7: Lint Report

## Purpose

The exporter must report what it found and what it changed.

## Required Behavior

Generate a JSON lint report containing:

- missing title
- missing chapter number
- duplicate heading removed
- existing front matter detected
- front matter injected
- duplicate block detected
- suspiciously short chapter
- suspiciously long chapter
- missing prose
- failed cleanup steps

## Suggested Struct

```
type LintIssue struct {    Type             string `json:"type"`    ChapterNumber    int    `json:"chapter_number,omitempty"`    ChapterID        string `json:"chapter_id,omitempty"`    Severity         string `json:"severity"`    Message          string `json:"message"`    OriginalText     string `json:"original_text,omitempty"`    SuggestedAction  string `json:"suggested_action,omitempty"`    MatchingChapter  int    `json:"matching_chapter,omitempty"`    MatchingTextHash string `json:"matching_text_hash,omitempty"`}
```

## Example Report

```
{  "issues": [    {      "type": "duplicate_heading",      "chapter_number": 6,      "severity": "medium",      "message": "Removed redundant inner chapter heading from the first five non-empty lines."    },    {      "type": "duplicate_block",      "chapter_number": 12,      "severity": "high",      "message": "Paragraph appears to duplicate content from Chapter 4.",      "matching_chapter": 4,      "matching_text_hash": "b881d9..."    }  ]}
```

## Suggested Function Signature

```
func WriteLintReport(path string, issues []LintIssue) error
```

---

# Phase 8: Final Markdown Rendering

## Purpose

Render the cleaned book object into a final Markdown manuscript.

## Required Structure

The final Markdown should contain:

1. Book title
2. Optional subtitle
3. Optional generated table of contents
4. Chapters in canonical order
5. Canonical chapter headings
6. Optional front matter
7. Cleaned prose

## Suggested Output

```
# Book Title## Table of Contents1. Chapter 1: ...2. Chapter 2: ...3. Chapter 3: ...---## Chapter 1: Title: SubtitleChapter prose...---## Chapter 2: Title: SubtitleChapter prose...
```

## Suggested Function Signatures

```
func RenderBookMarkdown(book Book) string
```

```
func RenderTableOfContents(book Book) string
```

```
func RenderChapterMarkdown(ch Chapter) string
```

```
func WriteMarkdownFile(path string, content string) error
```

---

# Phase 9: Status Updates Back to PocketBase

## Optional but Useful

If the current codebase already updates PocketBase records after export, set chapter or book status fields.

Suggested statuses:

```
draftnormalizedneeds_reviewexported
```

For tonight, only update status if the existing codebase already has this pattern.

Do not introduce risky writeback behavior if export currently only reads from PocketBase.

## Suggested Function Signature

```
func UpdateExportStatus(bookID string, status string) error
```

---

# Recommended Internal Flow

```
func ExportBook(bookID string) error {    book, err := LoadBookFromPocketBase(bookID)    if err != nil {        return err    }    var allIssues []LintIssue    validationIssues := ValidateBookForExport(book)    allIssues = append(allIssues, validationIssues...)    book.Chapters = SortChapters(book.Chapters)    for i, ch := range book.Chapters {        normalizedChapter, issues := NormalizeChapterHeading(ch)        allIssues = append(allIssues, issues...)        cleanedProse := NormalizeMarkdownSpacing(normalizedChapter.Prose)        cleanedProse, listIssues := NormalizeMarkdownLists(cleanedProse)        allIssues = append(allIssues, listIssues...)        if normalizedChapter.FrontMatter.Blurb != "" {            cleanedProse, fmIssues := InjectFrontMatter(cleanedProse, normalizedChapter.FrontMatter)            allIssues = append(allIssues, fmIssues...)        }        normalizedChapter.Prose = cleanedProse        book.Chapters[i] = normalizedChapter    }    duplicateIssues := DetectDuplicateBlocks(book.Chapters, DefaultDuplicateConfig())    allIssues = append(allIssues, duplicateIssues...)    finalMarkdown := RenderBookMarkdown(book)    if err := WriteMarkdownFile("/export/final_manuscript.md", finalMarkdown); err != nil {        return err    }    if err := WriteLintReport("/export/reports/manuscript_lint_report.json", allIssues); err != nil {        return err    }    return nil}
```

This pseudocode is illustrative. Codex should adapt it to the existing project structure.

---

# Acceptance Criteria

The change is complete when:

1. The exporter loads the existing generated chapters from PocketBase.
2. Chapters are exported in canonical chapter order.
3. The final manuscript contains exactly one canonical chapter heading per chapter.
4. Duplicate or conflicting generated chapter headings at the beginning of prose are removed.
5. Chapter numbering comes from PocketBase metadata/order, not generated prose.
6. Front matter is injected when available.
7. Front matter is not double-injected.
8. Obvious Markdown spacing issues are cleaned.
9. Obvious sequential protocols may be converted into numbered lists.
10. Obvious inventories/checklists may be converted into bullet lists.
11. Exact duplicated large blocks across chapters are detected.
12. A JSON lint report is written.
13. The existing chapter prose is not broadly regenerated or rewritten.
14. The final Markdown manuscript is written to the export folder.

---

# Unit Tests to Add

Add tests for:

```
BuildCanonicalChapterHeadingStripLeadingDuplicateHeadingsNormalizeChapterHeadingHasExistingFrontMatterRenderFrontMatterInjectFrontMatterNormalizeMarkdownSpacingNormalizeMarkdownListsDetectDuplicateBlocksRenderBookMarkdownWriteLintReport
```

## Required Test Cases

### Heading Tests

Input:

```
## Chapter 6: Wrong Exported Heading# Chapter 1: Wrong Inner HeadingActual prose begins here.
```

Expected:

```
Actual prose begins here.
```

when stripping body headings, with the canonical heading rendered separately.

---

### Front Matter Tests

Input chapter has front matter metadata.

Expected output contains:

```
> **The Field Note:**> *...*---
```

Input prose already has a field note.

Expected result: no duplicate field note.

---

### Duplicate Detection Tests

Two chapters contain the same long paragraph.

Expected: high-severity duplicate block issue.

Two chapters contain the same short phrase.

Expected: no issue.

---

### Markdown List Tests

Input:

```
Step 1: Breathe out longer than you breathe in. Step 2: Sit down. Step 3: Look near, then far.
```

Expected: numbered list if detection is reliable.

If detection is uncertain, leave unchanged.

---

# Safety Rules

The exporter must be conservative.

It may automatically fix:

```
duplicate leading headingswrong rendered chapter numbersfront matter placementMarkdown spacingobvious duplicated generated headingsobvious exact duplicate blocks when configured
```

It must not automatically perform:

```
full chapter rewritesmajor tone editsconceptual rewritinglarge content deletions unless exact duplicate detection is clearnew prose generationnew chapter generationmetadata redesign
```

---

# Tonight’s Working Philosophy

The goal is not perfection.

The goal is to prevent the manuscript from looking mechanically broken.

Tonight’s export should produce:

```
clean chapter structurecorrect numberingreadable Markdownno obvious stacked headingsno obvious duplicate front matterno obvious giant repeated blocksa report showing what still needs attention
```

Future improvements can add:

```
richer chapter metadatastyle drift auditingcontrolled style harmonizationimage prompt exportfiction continuity trackingchapter-level generation directivesYAML-style conceptual document views
```

Those belong in tomorrow’s change document, not tonight’s.

---

# Final Instruction to Codex

Implement the smallest safe post-generation export cleanup that gets the existing 20 PocketBase chapters into a clean Markdown manuscript.

Preserve the generated prose.

Fix structure.

Report problems.

Do not redesign the whole system.

Do not regenerate the book.

Make the export work.
````