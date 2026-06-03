package exporter

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	MinDuplicateBlockChars = 180
	MinDuplicateBlockWords = 30
)

type Book struct {
	ID          string
	Title       string
	Subtitle    string
	Description string
	Author      string
	Language    string
	Chapters    []Chapter
}

type Chapter struct {
	ID                   string
	SortOrder            int
	Title                string
	Subtitle             string
	Body                 string
	Source               string
	FrontMatterLabel     string
	FrontMatterBlurb     string
	Purpose              string
	StateStart           string
	StateEnd             string
	ChapterMetadataJSON  string
	ArcMetadataJSON      string
	ConceptJurisdiction  string
	GenerationDirectives string
	MediaPromptsJSON     string
}

type Result struct {
	Slug            string
	MarkdownURL     string
	EPUBURL         string
	HTMLURL         string
	LintReportURL   string
	StyleReportURL  string
	MarkdownPath    string
	EPUBPath        string
	HTMLPath        string
	LintReportPath  string
	StyleReportPath string
	Warnings        []string
}

type LintIssue struct {
	Type             string `json:"type"`
	ChapterNumber    int    `json:"chapter_number,omitempty"`
	ChapterID        string `json:"chapter_id,omitempty"`
	Severity         string `json:"severity"`
	Message          string `json:"message"`
	OriginalText     string `json:"original_text,omitempty"`
	SuggestedAction  string `json:"suggested_action,omitempty"`
	MatchingChapter  int    `json:"matching_chapter,omitempty"`
	MatchingTextHash string `json:"matching_text_hash,omitempty"`
}

type LintReport struct {
	BookID     string      `json:"book_id"`
	Title      string      `json:"title"`
	ExportedAt string      `json:"exported_at"`
	IssueCount int         `json:"issue_count"`
	Issues     []LintIssue `json:"issues"`
}

type DuplicateConfig struct {
	MinChars int
	MinWords int
}

type ExportOptions struct {
	OutputDir                     string
	RenderMediaPromptPlaceholders bool
	WriteStyleReport              bool
	WriteLintReport               bool
	WriteHTML                     bool
	WriteEPUB                     bool
}

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

type ConceptJurisdictionRules struct {
	Owns             []string `json:"owns"`
	MayReference     []string `json:"may_reference"`
	MustNotReteach   []string `json:"must_not_reteach"`
	ForbiddenPhrases []string `json:"forbidden_phrases"`
}

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

type Builder struct {
	Dir     string
	Options ExportOptions
}

func (b Builder) ExportBook(book Book) (Result, error) {
	book = normalizeBookDefaults(book)
	options := b.resolvedOptions()
	slug := slugify(book.Title)
	if slug == "" {
		slug = "untitled-book"
	}
	dir := options.OutputDir
	reportDir := filepath.Join(dir, "reports")
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return Result{}, err
	}

	cleaned, issues := CleanBookWithOptions(book, options)
	styleMetrics := AnalyzeBookStyleMetrics(cleaned)
	for _, chapter := range cleaned.Chapters {
		issues = append(issues, DetectWatchlistPhrases(chapter, DefaultWatchlistPhrases())...)
	}
	warnings := exportWarnings(cleaned, issues)

	mdPath := uniquePath(filepath.Join(dir, slug+".md"))
	epubPath := uniquePath(filepath.Join(dir, slug+".epub"))
	htmlPath := uniquePath(filepath.Join(dir, slug+".html"))
	reportPath := uniquePath(filepath.Join(reportDir, slug+"-lint.json"))
	stylePath := uniquePath(filepath.Join(reportDir, slug+"-style.json"))

	if err := os.WriteFile(mdPath, []byte(RenderBookMarkdown(cleaned)), 0644); err != nil {
		return Result{}, err
	}
	if options.WriteHTML {
		if err := os.WriteFile(htmlPath, []byte(RenderBookHTML(cleaned, warnings)), 0644); err != nil {
			return Result{}, err
		}
	}
	if options.WriteEPUB {
		if err := writeEPUB(epubPath, cleaned, warnings); err != nil {
			return Result{}, err
		}
	}
	if options.WriteLintReport {
		if err := WriteLintReport(reportPath, LintReport{
			BookID:     cleaned.ID,
			Title:      cleaned.Title,
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
			IssueCount: len(issues),
			Issues:     issues,
		}); err != nil {
			return Result{}, err
		}
	}
	if options.WriteStyleReport {
		if err := WriteStyleReport(stylePath, styleMetrics); err != nil {
			return Result{}, err
		}
	}

	result := Result{
		Slug:         slug,
		MarkdownURL:  "/exports/" + filepath.Base(mdPath),
		MarkdownPath: mdPath,
		Warnings:     warnings,
	}
	if options.WriteEPUB {
		result.EPUBURL = "/exports/" + filepath.Base(epubPath)
		result.EPUBPath = epubPath
	}
	if options.WriteHTML {
		result.HTMLURL = "/exports/" + filepath.Base(htmlPath)
		result.HTMLPath = htmlPath
	}
	if options.WriteLintReport {
		result.LintReportURL = "/exports/reports/" + filepath.Base(reportPath)
		result.LintReportPath = reportPath
	}
	if options.WriteStyleReport {
		result.StyleReportURL = "/exports/reports/" + filepath.Base(stylePath)
		result.StyleReportPath = stylePath
	}
	return result, nil
}

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

func (b Builder) resolvedOptions() ExportOptions {
	options := DefaultExportOptions()
	if b.Options.OutputDir != "" {
		options.OutputDir = b.Options.OutputDir
	}
	if b.Dir != "" {
		options.OutputDir = b.Dir
	}
	options.RenderMediaPromptPlaceholders = b.Options.RenderMediaPromptPlaceholders
	if b.Options.WriteStyleReport {
		options.WriteStyleReport = true
	}
	if b.Options.WriteLintReport {
		options.WriteLintReport = true
	}
	if b.Options.WriteHTML {
		options.WriteHTML = true
	}
	if b.Options.WriteEPUB {
		options.WriteEPUB = true
	}
	return options
}

func normalizeBookDefaults(book Book) Book {
	if strings.TrimSpace(book.Title) == "" {
		book.Title = "Untitled Book"
	}
	if strings.TrimSpace(book.Language) == "" {
		book.Language = "en"
	}
	if strings.TrimSpace(book.Author) == "" {
		book.Author = "Unknown Author"
	}
	return book
}

func CleanBook(book Book) (Book, []LintIssue) {
	return CleanBookWithOptions(book, DefaultExportOptions())
}

func CleanBookWithOptions(book Book, options ExportOptions) (Book, []LintIssue) {
	var issues []LintIssue
	SortChapters(book.Chapters)
	issues = append(issues, ValidateBookForExport(book)...)
	for i, chapter := range book.Chapters {
		cleaned, chapterIssues := CleanChapterWithOptions(chapter, options)
		issues = append(issues, chapterIssues...)
		book.Chapters[i] = cleaned
	}
	issues = append(issues, DetectDuplicateBlocks(book.Chapters, DefaultDuplicateConfig())...)
	issues = append(issues, AuditBookConceptJurisdiction(book)...)
	return book, issues
}

func SortChapters(chapters []Chapter) {
	sort.SliceStable(chapters, func(i, j int) bool {
		return chapters[i].SortOrder < chapters[j].SortOrder
	})
}

func ValidateBookForExport(book Book) []LintIssue {
	var issues []LintIssue
	if strings.TrimSpace(book.Title) == "" {
		issues = append(issues, LintIssue{Type: "missing_title", Severity: "high", Message: "Book title is missing."})
	}
	if strings.TrimSpace(book.Author) == "" || strings.TrimSpace(book.Author) == "Unknown Author" {
		issues = append(issues, LintIssue{Type: "missing_author", Severity: "medium", Message: "Author name is not set."})
	}
	expected := 1
	for _, chapter := range book.Chapters {
		if chapter.SortOrder <= 0 {
			issues = append(issues, chapterIssue(chapter, "missing_chapter_number", "high", "Chapter has no valid canonical sort order."))
		} else if chapter.SortOrder != expected {
			issues = append(issues, chapterIssue(chapter, "chapter_order_gap", "medium", fmt.Sprintf("Expected chapter order %d but found %d.", expected, chapter.SortOrder)))
			expected = chapter.SortOrder
		}
		expected++
		if strings.TrimSpace(chapter.Title) == "" {
			issues = append(issues, chapterIssue(chapter, "missing_chapter_title", "medium", "Chapter title is missing."))
		}
		body := strings.TrimSpace(chapter.Body)
		if body == "" {
			issues = append(issues, chapterIssue(chapter, "missing_prose", "high", "Chapter has no manuscript text."))
			continue
		}
		if countWords(body) < 250 {
			issues = append(issues, chapterIssue(chapter, "suspiciously_short_chapter", "medium", "Chapter is unusually short."))
		}
		if countWords(body) > 12000 {
			issues = append(issues, chapterIssue(chapter, "suspiciously_long_chapter", "medium", "Chapter is unusually long."))
		}
		if chapter.Source != "" && chapter.Source != "draft_content" {
			issues = append(issues, chapterIssue(chapter, "fallback_source_used", "medium", fmt.Sprintf("Chapter used %s fallback for export.", chapter.Source)))
		}
	}
	return issues
}

func CleanChapter(chapter Chapter) (Chapter, []LintIssue) {
	return CleanChapterWithOptions(chapter, DefaultExportOptions())
}

func CleanChapterWithOptions(chapter Chapter, options ExportOptions) (Chapter, []LintIssue) {
	var issues []LintIssue
	issues = append(issues, ValidateChapterMetadata(chapter)...)
	body, headingIssues := StripLeadingDuplicateHeadings(chapter)
	issues = append(issues, headingIssues...)
	body = NormalizeMarkdownSpacing(body)
	body, listIssues := NormalizeMarkdownLists(body)
	issues = append(issues, listIssues...)
	body, frontMatterIssues := InjectFrontMatter(body, chapter)
	issues = append(issues, frontMatterIssues...)
	if options.RenderMediaPromptPlaceholders {
		var mediaIssues []LintIssue
		body, mediaIssues = InsertMediaPromptPlaceholders(body, chapter)
		issues = append(issues, mediaIssues...)
	}
	chapter.Body = body
	return chapter, issues
}

func BuildCanonicalChapterHeading(chapter Chapter) string {
	title := strings.TrimSpace(chapter.Title)
	if title == "" {
		title = "Untitled"
	}
	if subtitle := strings.TrimSpace(chapter.Subtitle); subtitle != "" {
		title += ": " + subtitle
	}
	if chapter.SortOrder > 0 {
		return fmt.Sprintf("## Chapter %d: %s", chapter.SortOrder, title)
	}
	return "## " + title
}

func StripLeadingDuplicateHeadings(chapter Chapter) (string, []LintIssue) {
	normalized := normalizeNewlines(chapter.Body)
	lines := strings.Split(normalized, "\n")
	nonEmptySeen := 0
	remove := make(map[int]bool)
	var issues []LintIssue
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		nonEmptySeen++
		if nonEmptySeen > 5 {
			break
		}
		if looksLikeLeadingHeading(line) && shouldStripLeadingHeading(chapter, lines, i) {
			remove[i] = true
			issues = append(issues, chapterIssueWithOriginal(chapter, "duplicate_heading", "medium", "Removed redundant or conflicting generated heading from the start of chapter prose.", line))
			continue
		}
		if len(remove) > 0 {
			break
		}
		if !looksLikeLeadingHeading(line) {
			break
		}
	}
	if len(remove) == 0 {
		return strings.TrimSpace(normalized), nil
	}
	var kept []string
	for i, line := range lines {
		if remove[i] {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n")), issues
}

func looksLikeLeadingHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return false
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 2 {
		return false
	}
	rest := strings.TrimSpace(trimmed[level:])
	return rest != ""
}

func shouldStripLeadingHeading(chapter Chapter, lines []string, index int) bool {
	heading := strings.TrimSpace(lines[index])
	headingText := strings.TrimSpace(strings.TrimLeft(heading, "#"))
	lower := strings.ToLower(headingText)
	if strings.Contains(lower, "chapter") {
		return true
	}
	if title := strings.ToLower(strings.TrimSpace(chapter.Title)); title != "" && strings.Contains(lower, title) {
		return true
	}
	if subtitle := strings.ToLower(strings.TrimSpace(chapter.Subtitle)); subtitle != "" && strings.Contains(lower, subtitle) {
		return true
	}
	for _, line := range lines[index+1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		return looksLikeLeadingHeading(line)
	}
	return false
}

func ValidateChapterMetadata(chapter Chapter) []LintIssue {
	var issues []LintIssue
	for _, field := range []struct {
		name  string
		value string
	}{
		{"chapter_metadata_json", chapter.ChapterMetadataJSON},
		{"arc_metadata_json", chapter.ArcMetadataJSON},
		{"concept_jurisdiction_json", chapter.ConceptJurisdiction},
		{"generation_directives_json", chapter.GenerationDirectives},
		{"media_prompts_json", chapter.MediaPromptsJSON},
	} {
		issues = append(issues, validateOptionalJSONField(chapter, field.name, field.value)...)
	}
	return issues
}

func validateOptionalJSONField(chapter Chapter, fieldName string, value string) []LintIssue {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		issue := chapterIssue(chapter, "invalid_metadata_json", "medium", fieldName+" is not valid JSON.")
		issue.SuggestedAction = "Fix the JSON in the chapter metadata field."
		return []LintIssue{issue}
	}
	return nil
}

func ParseMediaPrompts(chapter Chapter) ([]MediaPrompt, []LintIssue) {
	value := strings.TrimSpace(chapter.MediaPromptsJSON)
	if value == "" {
		return nil, nil
	}
	var prompts []MediaPrompt
	if err := json.Unmarshal([]byte(value), &prompts); err == nil {
		return compactMediaPrompts(prompts), nil
	}
	var grouped struct {
		Images   []MediaPrompt `json:"images"`
		Diagrams []MediaPrompt `json:"diagrams"`
		Prompts  []MediaPrompt `json:"prompts"`
	}
	if err := json.Unmarshal([]byte(value), &grouped); err != nil {
		issue := chapterIssue(chapter, "invalid_media_prompts", "medium", "media_prompts_json could not be parsed for export placeholders.")
		issue.SuggestedAction = "Fix the media prompt JSON or leave it blank."
		return nil, []LintIssue{issue}
	}
	prompts = append(prompts, grouped.Images...)
	prompts = append(prompts, grouped.Diagrams...)
	prompts = append(prompts, grouped.Prompts...)
	return compactMediaPrompts(prompts), nil
}

func compactMediaPrompts(prompts []MediaPrompt) []MediaPrompt {
	var result []MediaPrompt
	for _, prompt := range prompts {
		if strings.TrimSpace(prompt.Prompt) == "" && strings.TrimSpace(prompt.Caption) == "" && strings.TrimSpace(prompt.AltText) == "" {
			continue
		}
		result = append(result, prompt)
	}
	return result
}

func RenderMediaPromptPlaceholders(chapter Chapter) (string, []LintIssue) {
	prompts, issues := ParseMediaPrompts(chapter)
	if len(prompts) == 0 {
		return "", issues
	}
	var b strings.Builder
	for i, prompt := range prompts {
		id := strings.TrimSpace(prompt.ID)
		if id == "" {
			id = fmt.Sprintf("chapter_%03d_media_%02d", chapter.SortOrder, i+1)
		}
		kind := strings.TrimSpace(prompt.Type)
		if kind == "" {
			kind = "image_prompt"
		}
		b.WriteString("<!-- MEDIA PROMPT:\n")
		fmt.Fprintf(&b, "chapter: %d\n", chapter.SortOrder)
		fmt.Fprintf(&b, "id: %s\n", sanitizeCommentValue(id))
		fmt.Fprintf(&b, "placement: %s\n", sanitizeCommentValue(prompt.Placement))
		fmt.Fprintf(&b, "type: %s\n", sanitizeCommentValue(kind))
		fmt.Fprintf(&b, "prompt: %s\n", sanitizeCommentValue(prompt.Prompt))
		fmt.Fprintf(&b, "alt_text: %s\n", sanitizeCommentValue(prompt.AltText))
		fmt.Fprintf(&b, "caption: %s\n", sanitizeCommentValue(prompt.Caption))
		b.WriteString("status: prompt_only_no_image_generation\n")
		b.WriteString("-->")
		if i+1 < len(prompts) {
			b.WriteString("\n\n")
		}
	}
	return b.String(), issues
}

func InsertMediaPromptPlaceholders(markdown string, chapter Chapter) (string, []LintIssue) {
	rendered, issues := RenderMediaPromptPlaceholders(chapter)
	if strings.TrimSpace(rendered) == "" {
		return markdown, issues
	}
	issues = append(issues, chapterIssue(chapter, "media_prompt_placeholder_rendered", "low", "Rendered media prompt placeholders from metadata without generating images."))
	return strings.TrimSpace(markdown) + "\n\n" + rendered, issues
}

func sanitizeCommentValue(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "-->", "-- >")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.ReplaceAll(value, "\r", " ")
}

func AuditBookConceptJurisdiction(book Book) []LintIssue {
	var issues []LintIssue
	for _, chapter := range book.Chapters {
		issues = append(issues, AuditConceptJurisdiction(chapter)...)
	}
	return issues
}

func AuditConceptJurisdiction(chapter Chapter) []LintIssue {
	value := strings.TrimSpace(chapter.ConceptJurisdiction)
	if value == "" || strings.TrimSpace(chapter.Body) == "" {
		return nil
	}
	var rules ConceptJurisdictionRules
	if err := json.Unmarshal([]byte(value), &rules); err != nil {
		issue := chapterIssue(chapter, "invalid_concept_jurisdiction", "medium", "concept_jurisdiction_json could not be parsed for export linting.")
		issue.SuggestedAction = "Fix the concept jurisdiction JSON or leave it blank."
		return []LintIssue{issue}
	}
	body := strings.ToLower(chapter.Body)
	var issues []LintIssue
	for _, phrase := range rules.ForbiddenPhrases {
		phrase = strings.TrimSpace(phrase)
		if phrase == "" {
			continue
		}
		if strings.Contains(body, strings.ToLower(phrase)) {
			issue := chapterIssue(chapter, "concept_jurisdiction_violation", "medium", fmt.Sprintf("Chapter uses forbidden phrase %q from concept jurisdiction metadata.", phrase))
			issue.OriginalText = phrase
			issue.SuggestedAction = "Revise the chapter text or update the concept jurisdiction metadata."
			issues = append(issues, issue)
		}
	}
	return issues
}

func HasExistingFrontMatter(markdown string) bool {
	for _, line := range firstNonEmptyLines(markdown, 4) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "> **") || strings.Contains(strings.ToLower(trimmed), "field note") {
			return true
		}
	}
	return false
}

func RenderFrontMatter(chapter Chapter) string {
	blurb := strings.TrimSpace(chapter.FrontMatterBlurb)
	if blurb == "" {
		return ""
	}
	label := strings.TrimSpace(chapter.FrontMatterLabel)
	if label == "" {
		label = "The Field Note"
	}
	return fmt.Sprintf("> **%s:**\n> *%s*\n\n---", label, blurb)
}

func InjectFrontMatter(markdown string, chapter Chapter) (string, []LintIssue) {
	rendered := RenderFrontMatter(chapter)
	if rendered == "" {
		return markdown, nil
	}
	if HasExistingFrontMatter(markdown) {
		return markdown, []LintIssue{chapterIssue(chapter, "existing_front_matter_detected", "low", "Chapter already appears to contain front matter; skipped metadata front matter injection.")}
	}
	return rendered + "\n\n" + strings.TrimSpace(markdown), []LintIssue{chapterIssue(chapter, "front_matter_injected", "low", "Injected chapter front matter from chapter metadata.")}
}

func NormalizeMarkdownSpacing(markdown string) string {
	lines := strings.Split(normalizeNewlines(markdown), "\n")
	var out []string
	for _, line := range lines {
		out = append(out, strings.TrimRight(line, " \t"))
	}
	text := strings.TrimSpace(strings.Join(out, "\n"))
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	text = regexp.MustCompile(`(?m)([^\n])\n(#{1,3}\s+)`).ReplaceAllString(text, "$1\n\n$2")
	text = regexp.MustCompile(`(?m)(#{1,3}[^\n]*)\n([^\n])`).ReplaceAllString(text, "$1\n\n$2")
	text = regexp.MustCompile(`(?m)([^\n])\n(>\s+)`).ReplaceAllString(text, "$1\n\n$2")
	text = regexp.MustCompile(`(?m)(>\s*[^\n]*)\n([^>\n])`).ReplaceAllString(text, "$1\n\n$2")
	text = regexp.MustCompile(`(?m)([^\n])\n((?:[-*]|\d+\.)\s+)`).ReplaceAllString(text, "$1\n\n$2")
	return strings.TrimSpace(regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n"))
}

func NormalizeMarkdownLists(markdown string) (string, []LintIssue) {
	blocks := strings.Split(markdown, "\n\n")
	var issues []LintIssue
	for i, block := range blocks {
		if DetectSequentialProtocol(block) {
			blocks[i] = renderSequentialList(block)
			issues = append(issues, LintIssue{Type: "list_normalized", Severity: "low", Message: "Converted an obvious sequential protocol into a numbered list."})
			continue
		}
		if DetectInventoryList(block) {
			blocks[i] = renderInventoryList(block)
			issues = append(issues, LintIssue{Type: "list_normalized", Severity: "low", Message: "Converted an obvious inline inventory into a bullet list."})
		}
	}
	return strings.Join(blocks, "\n\n"), issues
}

func DetectSequentialProtocol(block string) bool {
	matches := sequentialMarkerPattern.FindAllStringSubmatch(block, -1)
	if len(matches) < 3 {
		return false
	}
	expected := 1
	for _, match := range matches {
		if match[1] == "" {
			return false
		}
		value, err := strconv.Atoi(match[1])
		if err != nil || value != expected {
			return false
		}
		expected++
	}
	return true
}

func DetectInventoryList(block string) bool {
	trimmed := strings.TrimSpace(block)
	if strings.Contains(trimmed, "\n") || !strings.Contains(trimmed, ";") {
		return false
	}
	lower := strings.ToLower(trimmed)
	if !strings.Contains(lower, ":") {
		return false
	}
	prefix := strings.TrimSpace(strings.SplitN(lower, ":", 2)[0])
	if !strings.Contains(prefix, "include") && !strings.Contains(prefix, "contains") && !strings.Contains(prefix, "requires") && !strings.Contains(prefix, "needs") {
		return false
	}
	items := strings.Split(strings.SplitN(trimmed, ":", 2)[1], ";")
	if len(items) < 3 {
		return false
	}
	for _, item := range items {
		if countWords(item) > 14 || strings.TrimSpace(item) == "" {
			return false
		}
	}
	return true
}

func renderSequentialList(block string) string {
	matches := sequentialMarkerPattern.FindAllStringSubmatchIndex(block, -1)
	var items []string
	for i, match := range matches {
		contentStart := match[1]
		if i+1 < len(matches) {
			nextStart := matches[i+1][0]
			items = append(items, strings.TrimSpace(block[contentStart:nextStart]))
		} else {
			items = append(items, strings.TrimSpace(block[contentStart:]))
		}
	}
	var b strings.Builder
	for i, item := range items {
		item = strings.Trim(item, " .\n\t")
		if item == "" {
			continue
		}
		fmt.Fprintf(&b, "%d. %s\n", i+1, item)
	}
	return strings.TrimSpace(b.String())
}

func renderInventoryList(block string) string {
	parts := strings.SplitN(block, ":", 2)
	if len(parts) != 2 {
		return block
	}
	var b strings.Builder
	prefix := strings.TrimSpace(parts[0])
	if prefix != "" {
		b.WriteString(prefix)
		b.WriteString(":\n\n")
	}
	for _, item := range strings.Split(parts[1], ";") {
		item = strings.Trim(item, " .\n\t")
		if item == "" {
			continue
		}
		fmt.Fprintf(&b, "- %s\n", item)
	}
	return strings.TrimSpace(b.String())
}

func DetectDuplicateBlocks(chapters []Chapter, cfg DuplicateConfig) []LintIssue {
	if cfg.MinChars == 0 {
		cfg.MinChars = MinDuplicateBlockChars
	}
	if cfg.MinWords == 0 {
		cfg.MinWords = MinDuplicateBlockWords
	}
	seen := map[string]Chapter{}
	var issues []LintIssue
	for _, chapter := range chapters {
		for _, block := range SplitIntoBlocks(chapter.Body) {
			normalized := NormalizeBlockForHash(block)
			if shouldIgnoreDuplicateBlock(normalized, cfg) {
				continue
			}
			hash := HashBlock(normalized)
			if previous, ok := seen[hash]; ok && previous.SortOrder != chapter.SortOrder {
				issue := chapterIssue(chapter, "duplicate_block", "high", fmt.Sprintf("Large text block appears to duplicate content from Chapter %d.", previous.SortOrder))
				issue.MatchingChapter = previous.SortOrder
				issue.MatchingTextHash = hash
				issues = append(issues, issue)
				continue
			}
			seen[hash] = chapter
		}
	}
	return issues
}

func DefaultDuplicateConfig() DuplicateConfig {
	return DuplicateConfig{MinChars: MinDuplicateBlockChars, MinWords: MinDuplicateBlockWords}
}

func SplitIntoBlocks(markdown string) []string {
	return strings.Split(NormalizeMarkdownSpacing(markdown), "\n\n")
}

func NormalizeBlockForHash(block string) string {
	return strings.ToLower(strings.Join(strings.Fields(block), " "))
}

func HashBlock(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])[:16]
}

func shouldIgnoreDuplicateBlock(normalized string, cfg DuplicateConfig) bool {
	if len(normalized) < cfg.MinChars || countWords(normalized) < cfg.MinWords {
		return true
	}
	for _, label := range []string{"the field note", "chapter summary", "reflection questions", "key takeaways"} {
		if normalized == label {
			return true
		}
	}
	return false
}

func WriteLintReport(path string, report LintReport) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0644)
}

func AnalyzeBookStyleMetrics(book Book) []StyleMetrics {
	metrics := make([]StyleMetrics, 0, len(book.Chapters))
	for _, chapter := range book.Chapters {
		metrics = append(metrics, AnalyzeStyleMetrics(chapter))
	}
	return metrics
}

func AnalyzeStyleMetrics(chapter Chapter) StyleMetrics {
	body := strings.TrimSpace(chapter.Body)
	return StyleMetrics{
		ChapterNumber:          chapter.SortOrder,
		ChapterID:              chapter.ID,
		WordCount:              countWords(body),
		AvgSentenceLength:      averageSentenceLength(body),
		AvgParagraphLength:     averageParagraphLength(body),
		QuestionCount:          strings.Count(body, "?"),
		EmDashCount:            strings.Count(body, "—"),
		BulletCount:            countBulletLines(body),
		RepeatedOpeners:        repeatedSentenceOpeners(body),
		WatchlistPhraseMatches: watchlistMatches(body, DefaultWatchlistPhrases()),
	}
}

func WriteStyleReport(path string, metrics []StyleMetrics) error {
	body, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0644)
}

func DefaultWatchlistPhrases() []string {
	return []string{
		"it should be noted",
		"as we have seen",
		"in conclusion",
		"at the end of the day",
		"needless to say",
		"for all intents and purposes",
	}
}

func DetectWatchlistPhrases(chapter Chapter, phrases []string) []LintIssue {
	var issues []LintIssue
	for _, phrase := range watchlistMatches(chapter.Body, phrases) {
		count := strings.Count(strings.ToLower(chapter.Body), strings.ToLower(phrase))
		if count < 2 {
			continue
		}
		issue := chapterIssue(chapter, "watchlist_phrase_repeated", "low", fmt.Sprintf("Watchlist phrase %q appears %d times.", phrase, count))
		issue.OriginalText = phrase
		issue.SuggestedAction = "Consider varying or removing repeated stock phrasing during revision."
		issues = append(issues, issue)
	}
	return issues
}

func watchlistMatches(body string, phrases []string) []string {
	lower := strings.ToLower(body)
	var matches []string
	for _, phrase := range phrases {
		phrase = strings.TrimSpace(phrase)
		if phrase != "" && strings.Contains(lower, strings.ToLower(phrase)) {
			matches = append(matches, phrase)
		}
	}
	return uniqueStrings(matches)
}

func averageSentenceLength(body string) float64 {
	sentences := regexp.MustCompile(`[.!?]+`).Split(body, -1)
	total := 0
	count := 0
	for _, sentence := range sentences {
		words := countWords(sentence)
		if words == 0 {
			continue
		}
		total += words
		count++
	}
	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}

func averageParagraphLength(body string) float64 {
	blocks := SplitIntoBlocks(body)
	total := 0
	count := 0
	for _, block := range blocks {
		if strings.HasPrefix(strings.TrimSpace(block), "<!--") {
			continue
		}
		words := countWords(block)
		if words == 0 {
			continue
		}
		total += words
		count++
	}
	if count == 0 {
		return 0
	}
	return float64(total) / float64(count)
}

func countBulletLines(body string) int {
	count := 0
	for _, line := range strings.Split(normalizeNewlines(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if regexp.MustCompile(`^(?:[-*]|\d+\.)\s+`).MatchString(trimmed) {
			count++
		}
	}
	return count
}

func repeatedSentenceOpeners(body string) []string {
	sentences := regexp.MustCompile(`[.!?]+`).Split(body, -1)
	counts := map[string]int{}
	for _, sentence := range sentences {
		words := strings.Fields(strings.TrimSpace(sentence))
		if len(words) < 3 {
			continue
		}
		opener := strings.ToLower(strings.Join(words[:3], " "))
		counts[opener]++
	}
	var repeated []string
	for opener, count := range counts {
		if count > 1 {
			repeated = append(repeated, opener)
		}
	}
	sort.Strings(repeated)
	return repeated
}

func exportWarnings(book Book, issues []LintIssue) []string {
	var warnings []string
	if strings.TrimSpace(book.Author) == "Unknown Author" {
		warnings = append(warnings, "author name is not set")
	}
	if strings.TrimSpace(book.Description) == "" {
		warnings = append(warnings, "description/blurb is not set")
	}
	for _, issue := range issues {
		switch issue.Type {
		case "missing_author":
			continue
		case "missing_prose", "fallback_source_used", "duplicate_heading", "duplicate_block", "invalid_metadata_json", "invalid_media_prompts", "invalid_concept_jurisdiction", "concept_jurisdiction_violation", "watchlist_phrase_repeated":
			warnings = append(warnings, issue.Message)
		}
	}
	return uniqueStrings(warnings)
}

func RenderBookMarkdown(book Book) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntitle: %q\nsubtitle: %q\nauthor: %q\nlanguage: %q\nexported_at: %q\n---\n\n",
		book.Title, book.Subtitle, book.Author, book.Language, time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "# %s\n\n", book.Title)
	if book.Subtitle != "" {
		fmt.Fprintf(&b, "_%s_\n\n", book.Subtitle)
	}
	if book.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", book.Description)
	}
	b.WriteString(RenderTableOfContents(book))
	for _, chapter := range book.Chapters {
		b.WriteString(RenderChapterMarkdown(chapter))
	}
	return b.String()
}

func RenderTableOfContents(book Book) string {
	var b strings.Builder
	b.WriteString("## Table of Contents\n\n")
	for _, chapter := range book.Chapters {
		fmt.Fprintf(&b, "- Chapter %d: %s\n", chapter.SortOrder, chapterDisplayTitle(chapter))
	}
	b.WriteString("\n---\n\n")
	return b.String()
}

func RenderChapterMarkdown(chapter Chapter) string {
	body := strings.TrimSpace(chapter.Body)
	if body == "" {
		body = "[Incomplete chapter]"
	}
	return BuildCanonicalChapterHeading(chapter) + "\n\n" + body + "\n\n---\n\n"
}

func RenderBookHTML(book Book, warnings []string) string {
	var chapters strings.Builder
	for _, chapter := range book.Chapters {
		chapters.WriteString(renderChapterHTML(chapter))
	}
	var warningHTML strings.Builder
	if len(warnings) > 0 {
		warningHTML.WriteString(`<aside class="export-notes"><h2>Export Notes</h2><ul>`)
		for _, warning := range warnings {
			fmt.Fprintf(&warningHTML, "<li>%s</li>", html.EscapeString(warning))
		}
		warningHTML.WriteString(`</ul></aside>`)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
  <style>%s</style>
</head>
<body>
  <main class="book">
    <section class="title-page">
      <p class="kicker">Book Prototype Architect</p>
      <h1>%s</h1>
      <p class="subtitle">%s</p>
      <p class="author">%s</p>
    </section>
    %s
    <nav class="toc"><h2>Table of Contents</h2><ol>%s</ol></nav>
    %s
  </main>
</body>
</html>`,
		html.EscapeString(book.Language),
		html.EscapeString(book.Title),
		htmlCSS(),
		html.EscapeString(book.Title),
		html.EscapeString(book.Subtitle),
		html.EscapeString(book.Author),
		warningHTML.String(),
		htmlTOCItems(book),
		chapters.String(),
	)
}

func writeEPUB(path string, book Book, warnings []string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	if err := addStoredFile(zw, "mimetype", []byte("application/epub+zip")); err != nil {
		_ = zw.Close()
		return err
	}
	files := map[string][]byte{
		"META-INF/container.xml": []byte(containerXML()),
		"OEBPS/styles.css":       []byte(epubCSS()),
		"OEBPS/title.xhtml":      []byte(titleXHTML(book, warnings)),
		"OEBPS/toc.xhtml":        []byte(tocXHTML(book)),
		"OEBPS/content.opf":      []byte(contentOPF(book)),
	}
	for _, chapter := range book.Chapters {
		files[fmt.Sprintf("OEBPS/chapter-%03d.xhtml", chapter.SortOrder)] = []byte(chapterXHTML(chapter))
	}
	for name, body := range files {
		if err := addDeflatedFile(zw, name, body); err != nil {
			_ = zw.Close()
			return err
		}
	}
	return zw.Close()
}

func addStoredFile(zw *zip.Writer, name string, body []byte) error {
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

func addDeflatedFile(zw *zip.Writer, name string, body []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

func containerXML() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
}

func contentOPF(book Book) string {
	var manifest strings.Builder
	var spine strings.Builder
	manifest.WriteString(`<item id="title" href="title.xhtml" media-type="application/xhtml+xml"/>` + "\n")
	manifest.WriteString(`<item id="toc" href="toc.xhtml" media-type="application/xhtml+xml" properties="nav"/>` + "\n")
	manifest.WriteString(`<item id="css" href="styles.css" media-type="text/css"/>` + "\n")
	spine.WriteString(`<itemref idref="title"/>` + "\n")
	spine.WriteString(`<itemref idref="toc"/>` + "\n")
	for _, chapter := range book.Chapters {
		id := fmt.Sprintf("chapter-%03d", chapter.SortOrder)
		fmt.Fprintf(&manifest, `<item id="%s" href="%s.xhtml" media-type="application/xhtml+xml"/>`+"\n", id, id)
		fmt.Fprintf(&spine, `<itemref idref="%s"/>`+"\n", id)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="bookid">urn:bookbuilder:%s</dc:identifier>
    <dc:title>%s</dc:title>
    <dc:creator>%s</dc:creator>
    <dc:language>%s</dc:language>
    <dc:description>%s</dc:description>
    <meta property="dcterms:modified">%s</meta>
  </metadata>
  <manifest>
%s  </manifest>
  <spine>
%s  </spine>
</package>`, html.EscapeString(book.ID), html.EscapeString(book.Title), html.EscapeString(book.Author), html.EscapeString(book.Language), html.EscapeString(book.Description), time.Now().UTC().Format("2006-01-02T15:04:05Z"), indent(manifest.String()), indent(spine.String()))
}

func titleXHTML(book Book, warnings []string) string {
	var warningHTML strings.Builder
	if len(warnings) > 0 {
		warningHTML.WriteString(`<section class="warnings"><h2>Export Notes</h2><ul>`)
		for _, warning := range warnings {
			fmt.Fprintf(&warningHTML, "<li>%s</li>", html.EscapeString(warning))
		}
		warningHTML.WriteString(`</ul></section>`)
	}
	return xhtmlPage(book.Title, fmt.Sprintf(`<section class="title-page"><h1>%s</h1><p class="subtitle">%s</p><p class="author">%s</p></section>%s`,
		html.EscapeString(book.Title), html.EscapeString(book.Subtitle), html.EscapeString(book.Author), warningHTML.String()))
}

func tocXHTML(book Book) string {
	return xhtmlPage("Table of Contents", fmt.Sprintf(`<nav epub:type="toc" id="toc"><h1>Table of Contents</h1><ol>%s</ol></nav>`, epubTOCItems(book)))
}

func chapterXHTML(chapter Chapter) string {
	title := fmt.Sprintf("Chapter %d: %s", chapter.SortOrder, chapterDisplayTitle(chapter))
	return xhtmlPage(title, fmt.Sprintf(`<section epub:type="chapter"><h1>%s</h1>%s</section>`, html.EscapeString(title), markdownBlocksToHTML(chapter.Body)))
}

func xhtmlPage(title string, body string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="en">
<head>
  <title>%s</title>
  <link rel="stylesheet" type="text/css" href="styles.css"/>
</head>
<body>
%s
</body>
</html>`, html.EscapeString(title), body)
}

func renderChapterHTML(chapter Chapter) string {
	title := fmt.Sprintf("Chapter %d: %s", chapter.SortOrder, chapterDisplayTitle(chapter))
	return fmt.Sprintf(`<article class="chapter" id="chapter-%03d"><h2>%s</h2>%s</article>`, chapter.SortOrder, html.EscapeString(title), markdownBlocksToHTML(chapter.Body))
}

func htmlTOCItems(book Book) string {
	var items strings.Builder
	for _, chapter := range book.Chapters {
		fmt.Fprintf(&items, `<li><a href="#chapter-%03d">Chapter %d: %s</a></li>`+"\n", chapter.SortOrder, chapter.SortOrder, html.EscapeString(chapterDisplayTitle(chapter)))
	}
	return items.String()
}

func epubTOCItems(book Book) string {
	var items strings.Builder
	for _, chapter := range book.Chapters {
		fmt.Fprintf(&items, `<li><a href="chapter-%03d.xhtml">Chapter %d: %s</a></li>`+"\n", chapter.SortOrder, chapter.SortOrder, html.EscapeString(chapterDisplayTitle(chapter)))
	}
	return items.String()
}

func markdownBlocksToHTML(markdown string) string {
	body := strings.TrimSpace(markdown)
	if body == "" {
		body = "[Incomplete chapter]"
	}
	blocks := strings.Split(NormalizeMarkdownSpacing(body), "\n\n")
	var out strings.Builder
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" || block == "---" {
			continue
		}
		switch {
		case strings.HasPrefix(block, "### "):
			fmt.Fprintf(&out, "<h3>%s</h3>\n", renderInlineMarkdown(strings.TrimSpace(block[4:])))
		case strings.HasPrefix(block, "## "):
			fmt.Fprintf(&out, "<h2>%s</h2>\n", renderInlineMarkdown(strings.TrimSpace(block[3:])))
		case strings.HasPrefix(block, "# "):
			fmt.Fprintf(&out, "<h2>%s</h2>\n", renderInlineMarkdown(strings.TrimSpace(block[2:])))
		case strings.HasPrefix(block, "<!--") && strings.HasSuffix(block, "-->"):
			fmt.Fprintf(&out, "%s\n", block)
		case isBlockquote(block):
			fmt.Fprintf(&out, "<blockquote>%s</blockquote>\n", renderBlockquote(block))
		case isOrderedList(block):
			fmt.Fprintf(&out, "<ol>%s</ol>\n", renderListItems(block))
		case isBulletList(block):
			fmt.Fprintf(&out, "<ul>%s</ul>\n", renderListItems(block))
		default:
			fmt.Fprintf(&out, "<p>%s</p>\n", renderInlineMarkdown(strings.Join(strings.Fields(block), " ")))
		}
	}
	return out.String()
}

func renderBlockquote(block string) string {
	var parts []string
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), ">"))
		if line != "" {
			parts = append(parts, renderInlineMarkdown(line))
		}
	}
	return "<p>" + strings.Join(parts, "<br/>") + "</p>"
}

func renderListItems(block string) string {
	var items strings.Builder
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = regexp.MustCompile(`^(?:[-*]|\d+\.)\s+`).ReplaceAllString(line, "")
		fmt.Fprintf(&items, "<li>%s</li>", renderInlineMarkdown(line))
	}
	return items.String()
}

func renderInlineMarkdown(value string) string {
	var out strings.Builder
	for i := 0; i < len(value); {
		switch {
		case strings.HasPrefix(value[i:], "`"):
			end := strings.Index(value[i+1:], "`")
			if end < 0 {
				out.WriteString("&#96;")
				i++
				continue
			}
			content := value[i+1 : i+1+end]
			fmt.Fprintf(&out, "<code>%s</code>", html.EscapeString(content))
			i += end + 2
		case strings.HasPrefix(value[i:], "**"):
			end := strings.Index(value[i+2:], "**")
			if end < 0 {
				out.WriteString("**")
				i += 2
				continue
			}
			content := value[i+2 : i+2+end]
			fmt.Fprintf(&out, "<strong>%s</strong>", renderInlineMarkdown(content))
			i += end + 4
		case value[i] == '*':
			end := strings.Index(value[i+1:], "*")
			if end < 0 {
				out.WriteString("*")
				i++
				continue
			}
			content := value[i+1 : i+1+end]
			fmt.Fprintf(&out, "<em>%s</em>", renderInlineMarkdown(content))
			i += end + 2
		default:
			next := nextInlineMarker(value[i:])
			if next <= 0 {
				out.WriteString(html.EscapeString(value[i:]))
				i = len(value)
			} else {
				out.WriteString(html.EscapeString(value[i : i+next]))
				i += next
			}
		}
	}
	return out.String()
}

func nextInlineMarker(value string) int {
	next := -1
	for _, marker := range []string{"`", "*"} {
		if idx := strings.Index(value, marker); idx >= 0 && (next < 0 || idx < next) {
			next = idx
		}
	}
	return next
}

func isBlockquote(block string) bool {
	for _, line := range strings.Split(block, "\n") {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), ">") {
			return false
		}
	}
	return strings.TrimSpace(block) != ""
}

func isOrderedList(block string) bool {
	return regexp.MustCompile(`(?m)^\d+\.\s+`).MatchString(block)
}

func isBulletList(block string) bool {
	return regexp.MustCompile(`(?m)^[-*]\s+`).MatchString(block)
}

func htmlCSS() string {
	return `:root{color:#1f1a17;background:#f8f5ef}body{margin:0;font-family:Georgia,serif;line-height:1.7}.book{max-width:760px;margin:0 auto;padding:48px 24px 96px}.title-page{text-align:center;min-height:70vh;display:flex;flex-direction:column;align-items:center;justify-content:center;border-bottom:1px solid #ded6c9}.kicker{font:700 12px/1.2 system-ui,sans-serif;letter-spacing:.12em;text-transform:uppercase;color:#7b6b5b}h1{font-size:clamp(2.4rem,7vw,4.8rem);line-height:1.05;margin:.2em 0}h2{font-size:2rem;line-height:1.2;margin:2.5em 0 1em}.subtitle,.author{color:#66594d;font-style:italic}.toc,.export-notes{background:#fff;border:1px solid #e5ded3;padding:24px;margin:32px 0}.toc a{color:#3d342d;text-decoration:none}.chapter{padding-top:24px;margin-top:40px;border-top:1px solid #ded6c9}.chapter p{font-size:1.08rem}blockquote{border-left:4px solid #b99b6b;margin:1.5em 0;padding:.4em 1.2em;background:#fffaf0;color:#4a3e35}li{margin:.35em 0}@media(max-width:640px){.book{padding:28px 18px 72px}.chapter p{font-size:1rem}}`
}

func epubCSS() string {
	return `body{font-family:serif;line-height:1.5;margin:5%;}h1{font-size:1.7em;line-height:1.2;margin-bottom:1em;}h2{font-size:1.3em;margin-top:1.8em;}p{margin:0 0 1em 0;}blockquote{border-left:0.2em solid #999;margin:1em 0;padding-left:1em;}.title-page{text-align:center;margin-top:25%;}.subtitle,.author{font-style:italic;}.warnings{margin-top:3em;font-size:0.9em;}`
}

func indent(value string) string {
	var b strings.Builder
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		b.WriteString("    ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-v%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func chapterDisplayTitle(chapter Chapter) string {
	title := strings.TrimSpace(chapter.Title)
	if title == "" {
		title = "Untitled"
	}
	if subtitle := strings.TrimSpace(chapter.Subtitle); subtitle != "" {
		title += ": " + subtitle
	}
	return title
}

func chapterIssue(chapter Chapter, issueType string, severity string, message string) LintIssue {
	return LintIssue{Type: issueType, ChapterNumber: chapter.SortOrder, ChapterID: chapter.ID, Severity: severity, Message: message}
}

func chapterIssueWithOriginal(chapter Chapter, issueType string, severity string, message string, original string) LintIssue {
	issue := chapterIssue(chapter, issueType, severity, message)
	issue.OriginalText = strings.TrimSpace(original)
	return issue
}

func firstNonEmptyLines(markdown string, limit int) []string {
	var lines []string
	for _, line := range strings.Split(normalizeNewlines(markdown), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) == limit {
			break
		}
	}
	return lines
}

func normalizeNewlines(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func countWords(value string) int {
	return len(strings.Fields(value))
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	var result []string
	for _, value := range values {
		if strings.TrimSpace(value) == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

var (
	slugPattern             = regexp.MustCompile(`[^a-z0-9]+`)
	sequentialMarkerPattern = regexp.MustCompile(`(?i)(?:^|\s)(?:step|phase)\s+(\d+)[:.)-]\s*`)
)

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugPattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}
