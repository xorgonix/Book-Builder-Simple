package prompts

import (
	"strings"
	"testing"
)

func TestDraftPromptUsesMetadataAndProseOnlyBoundary(t *testing.T) {
	input := ChapterInput{
		Project: ProjectInput{
			Title:                   "The Metadata Book",
			BookType:                "fiction",
			TargetLength:            "practical_ebook",
			TargetChapters:          10,
			AuthorName:              "A. Writer",
			BookArchitectureJSON:    `{"protagonist_arc":"trust to agency"}`,
			GlobalStyleContractJSON: `{"primary_voice":["direct"],"avoid_voice":["generic"]}`,
			Intake:                  map[string]string{},
		},
		Brief: Brief{
			Title:   "The Metadata Book",
			Promise: "A precise test promise.",
		},
		SortOrder:            2,
		Title:                "The Turn",
		Purpose:              "Move the protagonist from avoidance to action.",
		StateStart:           "Avoidant",
		StateEnd:             "Committed",
		ChapterMetadataJSON:  `{"required_elements":["one concrete choice"]}`,
		ArcMetadataJSON:      `{"arc_phase":"first threshold","beats":[]}`,
		ConceptJurisdiction:  `{"must_not_reteach":["inciting incident"]}`,
		GenerationDirectives: `{"write_only_prose":true}`,
		MediaPromptsJSON:     `{"images":[{"placement":"chapter_open","prompt":"quiet street"}]}`,
	}
	got := BuildDraftPrompt(input)
	for _, want := range []string{
		"qwen/qwen3.5-9b",
		"Write only the prose field",
		"Do not output metadata",
		"Do not invent chapter numbers",
		"Book architecture JSON",
		"Arc metadata JSON",
		"Concept jurisdiction JSON",
		"Generation directives JSON",
		"Media prompts JSON",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("draft prompt missing %q:\n%s", want, got)
		}
	}
}

func TestFormatMetadataContextOmitsBlankMetadata(t *testing.T) {
	got := FormatMetadataContext(ChapterInput{})
	if strings.Contains(got, "Publishing metadata JSON") || strings.Contains(got, "Chapter metadata JSON") {
		t.Fatalf("blank metadata should not create labeled JSON sections:\n%s", got)
	}
	if !strings.Contains(got, "qwen/qwen3.5-9b") {
		t.Fatalf("metadata context should document fixed model role policy:\n%s", got)
	}
}

func TestFormatContractDetectsCookbook(t *testing.T) {
	got := FormatContract(ChapterInput{Project: ProjectInput{Intake: map[string]string{"book_form": "cookbook"}}})
	for _, want := range []string{"cookbook / recipe guide", "Do not write this chapter as essay-only prose", "ingredients", "method steps"} {
		if !strings.Contains(got, want) {
			t.Fatalf("format contract missing %q:\n%s", want, got)
		}
	}
}

func TestParseBriefToleratesMarkdownLabels(t *testing.T) {
	brief, err := ParseBrief(`**Title:** The Test Book
**Subtitle:** A small test
- Promise: The reader gets a working brief.
Voice Tone: Direct
What It Is: A clear guide.
What It Is Not: A vague essay.
AI Suggestions: Keep it practical.`)
	if err != nil {
		t.Fatal(err)
	}
	if brief.Title != "The Test Book" || brief.Promise != "The reader gets a working brief." {
		t.Fatalf("unexpected parsed brief: %#v", brief)
	}
}
