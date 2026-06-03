package exporter

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCanonicalChapterHeading(t *testing.T) {
	chapter := Chapter{SortOrder: 3, Title: "Rewilding the Nervous System", Subtitle: "Practical Strategies"}
	got := BuildCanonicalChapterHeading(chapter)
	want := "## Chapter 3: Rewilding the Nervous System: Practical Strategies"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStripLeadingDuplicateHeadings(t *testing.T) {
	chapter := Chapter{
		ID:        "ch1",
		SortOrder: 6,
		Title:     "The Bridge of Small Things",
		Body:      "## Chapter 6: Wrong Exported Heading\n# Chapter 1: Wrong Inner Heading\n\nActual prose begins here.\n\n## A Real Section\nKeep this.",
	}
	cleaned, issues := StripLeadingDuplicateHeadings(chapter)
	if strings.Contains(cleaned, "Wrong Exported Heading") || strings.Contains(cleaned, "Wrong Inner Heading") {
		t.Fatalf("leading headings were not stripped: %q", cleaned)
	}
	if !strings.Contains(cleaned, "## A Real Section") {
		t.Fatalf("later section heading was stripped: %q", cleaned)
	}
	if len(issues) != 2 {
		t.Fatalf("got %d issues, want 2", len(issues))
	}
}

func TestStripLeadingDuplicateHeadingsKeepsSpecificOpeningHeading(t *testing.T) {
	chapter := Chapter{
		ID:        "ch1",
		SortOrder: 1,
		Title:     "The Door",
		Body:      "## The Offer Nobody Wanted\n\nActual prose begins here.",
	}
	cleaned, issues := StripLeadingDuplicateHeadings(chapter)
	if !strings.Contains(cleaned, "The Offer Nobody Wanted") {
		t.Fatalf("specific opening heading was stripped: %q", cleaned)
	}
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestFrontMatterInjectionAndNoDoubleInject(t *testing.T) {
	chapter := Chapter{SortOrder: 1, FrontMatterLabel: "The Field Note", FrontMatterBlurb: "Welcome to the exchange."}
	cleaned, issues := InjectFrontMatter("Chapter prose.", chapter)
	if !strings.Contains(cleaned, "> **The Field Note:**") {
		t.Fatalf("front matter missing: %q", cleaned)
	}
	if len(issues) != 1 || issues[0].Type != "front_matter_injected" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	again, issues := InjectFrontMatter(cleaned, chapter)
	if strings.Count(again, "The Field Note") != 1 {
		t.Fatalf("front matter double injected: %q", again)
	}
	if len(issues) != 1 || issues[0].Type != "existing_front_matter_detected" {
		t.Fatalf("unexpected issues after double inject check: %#v", issues)
	}
}

func TestNormalizeMarkdownSpacing(t *testing.T) {
	got := NormalizeMarkdownSpacing("Intro\n## Heading\nBody\n\n\n> Quote\nNext")
	if strings.Contains(got, "\n\n\n") {
		t.Fatalf("excess blank lines remain: %q", got)
	}
	if !strings.Contains(got, "Intro\n\n## Heading\n\nBody") {
		t.Fatalf("heading spacing not normalized: %q", got)
	}
	if !strings.Contains(got, "> Quote\n\nNext") {
		t.Fatalf("blockquote spacing not normalized: %q", got)
	}
}

func TestNormalizeMarkdownLists(t *testing.T) {
	input := "Step 1: Breathe out longer than you breathe in. Step 2: Sit down. Step 3: Look near, then far."
	got, issues := NormalizeMarkdownLists(input)
	for _, want := range []string{"1. Breathe out", "2. Sit down", "3. Look near"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
	unchanged, issues := NormalizeMarkdownLists("First breathe out. Then sit down.")
	if unchanged != "First breathe out. Then sit down." || len(issues) != 0 {
		t.Fatalf("uncertain prose changed: %q %#v", unchanged, issues)
	}
}

func TestNormalizeMarkdownInventoryList(t *testing.T) {
	input := "This chapter includes: the setup; the conflict; the consequence."
	got, issues := NormalizeMarkdownLists(input)
	for _, want := range []string{"This chapter includes:", "- the setup", "- the conflict", "- the consequence"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}
}

func TestValidateChapterMetadata(t *testing.T) {
	chapter := Chapter{
		ID:                  "ch1",
		SortOrder:           1,
		ChapterMetadataJSON: `{"function":"opening"}`,
		ArcMetadataJSON:     `{bad json`,
	}
	issues := ValidateChapterMetadata(chapter)
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1: %#v", len(issues), issues)
	}
	if issues[0].Type != "invalid_metadata_json" || issues[0].ChapterNumber != 1 {
		t.Fatalf("unexpected issue: %#v", issues[0])
	}
}

func TestMediaPromptPlaceholdersArePromptOnly(t *testing.T) {
	chapter := Chapter{
		ID:               "ch2",
		SortOrder:        2,
		MediaPromptsJSON: `{"images":[{"id":"img1","placement":"chapter_open","prompt":"A quiet cover-like scene","alt_text":"A quiet scene","caption":"Opening image"}],"diagrams":[]}`,
	}
	rendered, issues := RenderMediaPromptPlaceholders(chapter)
	if len(issues) != 0 {
		t.Fatalf("unexpected issues: %#v", issues)
	}
	for _, want := range []string{"<!-- MEDIA PROMPT:", "chapter: 2", "id: img1", "status: prompt_only_no_image_generation"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("missing %q in %q", want, rendered)
		}
	}
	if strings.Contains(strings.ToLower(rendered), "gpt-image") {
		t.Fatalf("image model reference leaked into placeholder: %q", rendered)
	}
}

func TestConceptJurisdictionForbiddenPhrase(t *testing.T) {
	chapter := Chapter{
		ID:                  "ch3",
		SortOrder:           3,
		Body:                "This chapter should not re-teach the pincer protocol, but it says the pincer protocol twice.",
		ConceptJurisdiction: `{"forbidden_phrases":["pincer protocol"]}`,
	}
	issues := AuditConceptJurisdiction(chapter)
	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1: %#v", len(issues), issues)
	}
	if issues[0].Type != "concept_jurisdiction_violation" {
		t.Fatalf("unexpected issue: %#v", issues[0])
	}
}

func TestStyleMetricsAndWatchlist(t *testing.T) {
	chapter := Chapter{
		ID:        "ch4",
		SortOrder: 4,
		Body:      "It should be noted that this opens. It should be noted that this repeats.\n\n- one\n- two",
	}
	metrics := AnalyzeStyleMetrics(chapter)
	if metrics.WordCount == 0 || metrics.BulletCount != 2 || metrics.QuestionCount != 0 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
	if len(metrics.WatchlistPhraseMatches) != 1 {
		t.Fatalf("expected watchlist match: %#v", metrics)
	}
	issues := DetectWatchlistPhrases(chapter, DefaultWatchlistPhrases())
	if len(issues) != 1 || issues[0].Type != "watchlist_phrase_repeated" {
		t.Fatalf("unexpected watchlist issues: %#v", issues)
	}
}

func TestDetectDuplicateBlocks(t *testing.T) {
	paragraph := strings.Repeat("This is a long repeated paragraph with enough distinct manuscript language to trigger duplicate block detection. ", 4)
	issues := DetectDuplicateBlocks([]Chapter{
		{ID: "a", SortOrder: 1, Body: paragraph},
		{ID: "b", SortOrder: 2, Body: paragraph},
		{ID: "c", SortOrder: 3, Body: "short phrase"},
	}, DefaultDuplicateConfig())
	if len(issues) != 1 {
		t.Fatalf("got %d duplicate issues, want 1: %#v", len(issues), issues)
	}
	if issues[0].MatchingChapter != 1 {
		t.Fatalf("matching chapter = %d, want 1", issues[0].MatchingChapter)
	}
}

func TestRenderedOutputsUseCleanedBody(t *testing.T) {
	book := Book{
		ID:       "book1",
		Title:    "Rendered Book",
		Author:   "Author",
		Language: "en",
		Chapters: []Chapter{{
			ID:        "ch1",
			SortOrder: 1,
			Title:     "A Clean Start",
			Body:      "# Chapter 99: Wrong\n\nStep 1: Breathe. Step 2: Sit. Step 3: Look.",
			Source:    "draft_content",
		}},
	}
	cleaned, _ := CleanBook(book)
	md := RenderBookMarkdown(cleaned)
	html := RenderBookHTML(cleaned, nil)
	if strings.Contains(md, "Chapter 99") || strings.Contains(html, "Chapter 99") {
		t.Fatalf("rendered output contains stripped heading\nmd=%s\nhtml=%s", md, html)
	}
	if !strings.Contains(md, "## Chapter 1: A Clean Start") {
		t.Fatalf("canonical markdown heading missing: %q", md)
	}
	if !strings.Contains(html, "<ol>") {
		t.Fatalf("rendered html list missing: %q", html)
	}
}

func TestRenderedInlineMarkdown(t *testing.T) {
	got := markdownBlocksToHTML("**Action Step: The 3-Second Pause Protocol**\n\nA *small* pause and `one breath`.")
	for _, want := range []string{
		"<strong>Action Step: The 3-Second Pause Protocol</strong>",
		"<em>small</em>",
		"<code>one breath</code>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "**Action Step") {
		t.Fatalf("raw bold markdown leaked into HTML: %q", got)
	}
}

func TestWriteLintReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	report := LintReport{BookID: "book1", Title: "Book", IssueCount: 1, Issues: []LintIssue{{Type: "duplicate_block", Severity: "high", Message: "duplicate"}}}
	if err := WriteLintReport(path, report); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded LintReport
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.IssueCount != 1 || decoded.Issues[0].Type != "duplicate_block" {
		t.Fatalf("unexpected report: %#v", decoded)
	}
}

func TestExportBookWritesCleanedBundle(t *testing.T) {
	dir := t.TempDir()
	result, err := Builder{Dir: dir}.ExportBook(Book{
		ID:       "book1",
		Title:    "Bundle Book",
		Author:   "Author",
		Language: "en",
		Chapters: []Chapter{{
			ID:        "ch1",
			SortOrder: 1,
			Title:     "The Clean Chapter",
			Body:      "# Chapter 9: Wrong\n\nStep 1: Begin. Step 2: Continue. Step 3: Finish.",
			Source:    "draft_content",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{result.MarkdownPath, result.HTMLPath, result.EPUBPath, result.LintReportPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected export file %s: %v", path, err)
		}
	}
	if _, err := os.Stat(result.StyleReportPath); err != nil {
		t.Fatalf("expected style report %s: %v", result.StyleReportPath, err)
	}
	htmlBody, err := os.ReadFile(result.HTMLPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(htmlBody), "Chapter 9") || !strings.Contains(string(htmlBody), "<ol>") {
		t.Fatalf("html did not use cleaned chapter body: %s", string(htmlBody))
	}
	zr, err := zip.OpenReader(result.EPUBPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	var chapterBody string
	var opfBody string
	for _, file := range zr.File {
		if file.Name == "OEBPS/content.opf" {
			rc, err := file.Open()
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				t.Fatal(err)
			}
			opfBody = string(body)
			continue
		}
		if file.Name != "OEBPS/chapter-001.xhtml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		chapterBody = string(body)
	}
	if chapterBody == "" {
		t.Fatal("EPUB chapter was not written")
	}
	if opfBody == "" {
		t.Fatal("EPUB package file was not written")
	}
	titleIndex := strings.Index(opfBody, `<itemref idref="title"/>`)
	tocIndex := strings.Index(opfBody, `<itemref idref="toc"/>`)
	chapterIndex := strings.Index(opfBody, `<itemref idref="chapter-001"/>`)
	if titleIndex < 0 || tocIndex < 0 || chapterIndex < 0 || !(titleIndex < tocIndex && tocIndex < chapterIndex) {
		t.Fatalf("EPUB spine should put readable TOC after title and before chapter: %s", opfBody)
	}
	if strings.Contains(chapterBody, "Chapter 9") || !strings.Contains(chapterBody, "<ol>") {
		t.Fatalf("epub did not use cleaned chapter body: %s", chapterBody)
	}
}
