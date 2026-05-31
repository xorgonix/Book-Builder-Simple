package prompts

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"bookbuilder/internal/db"
)

type ProjectInput struct {
	ID             string
	Title          string
	BookType       string
	TargetLength   string
	TargetChapters int
	Intake         map[string]string
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

type ChapterPlan struct {
	SortOrder  int
	Title      string
	Purpose    string
	StateStart string
	StateEnd   string
}

type ChapterInput struct {
	Project      ProjectInput
	Brief        Brief
	SortOrder    int
	Title        string
	Purpose      string
	StateStart   string
	StateEnd     string
	RawDraft     string
	Diagnosis    string
	Rewrite      string
	DraftNotes   string
	DiagnoseNote string
	RewriteNotes string
}

const systemPrompt = "You are a precise book architecture engine. Follow the requested output contract exactly. Never wrap structural outputs in Markdown code fences."

func SystemPrompt() string {
	return systemPrompt
}

func BuildBriefPrompt(project ProjectInput) string {
	return fmt.Sprintf(`You are an elite structural book editor and publishing strategist. Your task is to process raw intake parameters and transform them into a comprehensive, professional Book Brief.

USER INTAKE NOTES:

- Book Vector Type: %s
- Book Form / Genre: %s
- Core System Topic: %s
- Target Reader Profile: %s
- Deep Reader Hunger/Desire: %s
- Prohibited Directions (What NOT to become): %s
- Stated Author Tone Target: %s
- Narrative Point of View: %s
- Structure / Formula Model: %s

Target Manuscript Scale:

- Total Target Word Count: %d
- Intended Chapter Count: %d

You must output your analysis matching this exact text layout below. Do not wrap the output in Markdown code blocks or include any introductory conversational comments.

TITLE: [Generate an engaging, high-impact working title]
SUBTITLE: [Generate a subtitle that clarifies the core hook]
PROMISE: [A single-sentence value proposition addressing the Deep Reader Hunger]
VOICE_TONE: [Extract 3 definitive stylistic parameters matching the Selected Tone]
WHAT_IT_IS: [A detailed overview description of the book's delivery framework]
WHAT_IT_IS_NOT: [Strict boundary constraints directly addressing the Prohibited Directions]
AI_SUGGESTIONS: [A critical structural evaluation of how to execute this concept across the requested chapter count]`,
		project.BookType,
		emptyFallback(project.Intake["book_form"]),
		project.Intake["core_topic"],
		project.Intake["target_audience"],
		project.Intake["reader_hunger"],
		project.Intake["prohibited_directions"],
		project.Intake["selected_tone"],
		emptyFallback(project.Intake["narrative_pov"]),
		emptyFallback(project.Intake["structure_model"]),
		TargetWords(project.TargetLength),
		project.TargetChapters,
	)
}

func BuildTOCPrompt(project ProjectInput, brief Brief) string {
	return fmt.Sprintf(`You are a master book architect. Your task is to generate a complete, structural Table of Contents for a book based on the provided Book Brief.

VERIFIED BOOK BRIEF:

%s

MANUSCRIPT METRICS:

- Total Targeted Chapters: %d
- Target Chapter Word Count Boundary: %d words per chapter

You must generate exactly %d individual chapter entries. Each entry must be cleanly wrapped inside the active parser markers CHAPTER_START and CHAPTER_END. Do not include any introductory commentary or markdown code blocks.

CHAPTER_START
Order: [Sequence Integer starting at 1]
Title: [A compelling, clear chapter title]
Purpose: [What the chapter must mechanically accomplish to advance the book's promise]
Reader Start: [The precise emotional or intellectual frustration of the reader on entry]
Reader End: [The transformation goal or clarity target of the reader upon exiting this chapter]
CHAPTER_END`,
		FormatBrief(brief),
		project.TargetChapters,
		ChapterTargetWords(project),
		project.TargetChapters,
	)
}

func BuildEscapeHatchPrompt(project ProjectInput, field string) string {
	return fmt.Sprintf(`Return three concise, high-quality suggestions for the intake field %q.

Use the current premise, reader, tone, and book shape. Do not return generic filler. Make each suggestion specific enough that a non-writer can choose one and keep moving.

Field-specific guidance: %s

Current project:
%s

Output only the three suggestions as plain text, one suggestion per line. Do not use markdown bullets or code fences.`, field, escapeHatchDirection(field), formatProject(project))
}

func escapeHatchDirection(field string) string {
	switch field {
	case "target_audience":
		return "Suggest a precise reader archetype or job-to-be-done based on the premise."
	case "reader_hunger":
		return "Suggest the concrete pain, curiosity, fear, or aspiration that makes this premise matter."
	case "prohibited_directions":
		return "Suggest specific things to avoid: tropes, tone, content, structure, or overused angles."
	case "book_form":
		return "Suggest a better-fit book form, genre, or format for the premise."
	case "narrative_pov":
		return "Suggest a point of view that fits the premise and reader experience."
	case "structure_model":
		return "Suggest a proven structure or formula that fits the book's promise."
	default:
		return "Suggest a practical, premise-aware answer that helps the user move forward."
	}
}

func BuildDraftPrompt(input ChapterInput) string {
	return fmt.Sprintf(`Write a complete, fully detailed book chapter based on the provided structural outline, baseline profile details, and project variables.

PROJECT CONTEXT BRIEF:

%s

CHAPTER ARCHITECTURAL OBJECTIVE:

- Title: %s
- Core Purpose: %s
- Reader Entry State: %s
- Reader Exit State: %s
- Dynamic Target Chapter Length: %d words

USER PRE-DRAFT CONSTRAINT NOTES (High Priority):

%s

Generate this as a fully developed chapter. Do not output abbreviated summaries or simple expanded bullet points.

Execution Constraints:

- Honor the structural path laid out in the outline, but ensure the narrative flows naturally.
- Use smooth transitions instead of mechanical or textbook phrasing.
- Vary sentence length and paragraph structures to create an engaging rhythm.
- The prose must feel authored by a single person, not assembled by a machine.
- Avoid relying on clean lists or bullet sections unless the context explicitly demands them.
- Do not conclude every section with an overt summary paragraph.
- Strictly avoid generic filler text like "it is important to note," "in today's fast-paced world," "this chapter explores," or "in conclusion."
- Do not invent facts, citations, quotes, or anecdotes unless explicitly outlined above.

Output: Return the complete chapter draft text only. Do not add introductory or concluding assistant commentary.`,
		FormatBrief(input.Brief),
		input.Title,
		input.Purpose,
		input.StateStart,
		input.StateEnd,
		ChapterTargetWords(input.Project),
		emptyFallback(input.DraftNotes),
	)
}

func BuildDiagnosisPrompt(input ChapterInput) string {
	return fmt.Sprintf(`Analyze the attached draft chapter from an editor's perspective. Your job is to identify structural issues and areas for improvement before making edits.

PROJECT BRIEF:

%s

UNEDITED DRAFT CHAPTER:

%s

Evaluate the text using these 10 distinct lenses:

1. Voice consistency matching the target author profile
2. Reader engagement levels and pacing issues
3. Overly generic, predictable, or AI-sounding phrasing
4. Repetitive sentence structures or boring rhythms
5. Weak or abrupt structural transitions
6. Places where the text reads like expanded outline notes rather than real book prose
7. Claims or assertions that lack adequate context or support
8. Sections that feel bogged down, bloated, or padded
9. Areas that feel rushed or overly compressed
10. Opportunities to strengthen the author's point of view

Output your analysis using these exact headings:

- CRITICAL DIAGNOSIS:
- TARGETED FIXES:
- SPECIFIC PASSAGES:
- PROTECTED ELEMENTS:`,
		FormatBrief(input.Brief),
		input.RawDraft,
	)
}

func BuildRewritePrompt(input ChapterInput) string {
	return fmt.Sprintf(`Rewrite the provided chapter text by applying the targeted editorial fixes outlined in the structural diagnosis and incorporating the user's manual revisions.

PROJECT BRIEF & TARGET INTENT:

%s

EDITORIAL DIAGNOSIS & ACTION PLAN:

%s

USER'S MANUAL CRITIQUE OVERRIDE (High Priority):

%s

ORIGINAL CHAPTER TEXT:

%s

Execution Rewrite Constraints:

- Preserve the underlying arguments, chapter sequence, and factual claims unless the diagnosis explicitly directs a change.
- Significantly improve sentence variety, rhythmic flow, and narrative clarity.
- Completely remove generic, uninspired AI phrasing patterns.
- Ensure the prose reads like a single voice with a clear perspective.
- Do not invent new stories, unverified statistics, or external quotes.
- Do not make text patterns artificially repetitive.
- Avoid concluding sections with a generic inspirational summary.

Output: Return the rewritten chapter text only. Do not add introductory or concluding commentary.`,
		FormatBrief(input.Brief),
		input.Diagnosis,
		emptyFallback(input.DiagnoseNote),
		input.RawDraft,
	)
}

func BuildPolishPrompt(input ChapterInput) string {
	return fmt.Sprintf(`Perform a structural edit on this text to remove common AI writing patterns and stylistic tells.

TARGET TEXT:

%s

Identify and eliminate these specific issues:

- Repetitive sentence setups or predictable paragraph rhythms.
- Overused "not just X, but Y" balancing setups.
- Generic transitions and abstract summary phrasing.
- Explaining concepts past the point of clarity.
- Rigid three-part structures or list formats that feel automated.
- A neutral, clinical, or corporate tone.
- Conclusion-heavy paragraphs that repeat previous points.
- Smooth sentences that lack real substance or punch.

Maintain the core arguments, factual elements, and tone of the draft. Do not add casual filler text, jokes, or synthetic slang. Improve the rhythm, voice authority, and flow of the writing.

Output: Return the polished text only.`, input.Rewrite)
}

func ParseBrief(text string) (Brief, error) {
	values := parseLabelBlock(text, []string{
		"TITLE",
		"SUBTITLE",
		"PROMISE",
		"VOICE_TONE",
		"WHAT_IT_IS",
		"WHAT_IT_IS_NOT",
		"AI_SUGGESTIONS",
	})
	brief := Brief{
		Title:         values["TITLE"],
		Subtitle:      values["SUBTITLE"],
		Promise:       values["PROMISE"],
		VoiceTone:     values["VOICE_TONE"],
		WhatItIs:      values["WHAT_IT_IS"],
		WhatItIsNot:   values["WHAT_IT_IS_NOT"],
		AISuggestions: values["AI_SUGGESTIONS"],
	}
	if brief.Title == "" || brief.Promise == "" {
		return brief, fmt.Errorf("brief output missing required TITLE or PROMISE")
	}
	return brief, nil
}

func ParseTOC(text string, expected int) ([]ChapterPlan, error) {
	parts := strings.Split(text, "CHAPTER_START")
	plans := make([]ChapterPlan, 0, expected)
	for _, part := range parts[1:] {
		entry := strings.Split(part, "CHAPTER_END")[0]
		values := parseLabelBlock(entry, []string{"Order", "Title", "Purpose", "Reader Start", "Reader End"})
		order, orderRemainder, err := parseOrder(values["Order"])
		if err != nil || order < 1 {
			return nil, fmt.Errorf("invalid chapter order in TOC entry %q", values["Order"])
		}
		if values["Title"] == "" && orderRemainder != "" {
			values["Title"] = orderRemainder
		}
		if values["Title"] == "" || values["Purpose"] == "" {
			return nil, fmt.Errorf("TOC entry %d missing title or purpose", order)
		}
		plans = append(plans, ChapterPlan{
			SortOrder:  order,
			Title:      values["Title"],
			Purpose:    values["Purpose"],
			StateStart: values["Reader Start"],
			StateEnd:   values["Reader End"],
		})
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].SortOrder < plans[j].SortOrder })
	if expected > 0 && len(plans) != expected {
		return nil, fmt.Errorf("expected %d TOC chapters, got %d", expected, len(plans))
	}
	return plans, nil
}

func parseOrder(value string) (int, string, error) {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return 0, "", fmt.Errorf("empty order")
	}
	fields := strings.Fields(lines[0])
	if len(fields) == 0 {
		return 0, "", fmt.Errorf("empty order")
	}
	token := strings.Trim(fields[0], ".:)#")
	order, err := strconv.Atoi(token)
	if err != nil {
		return 0, "", err
	}
	remainder := strings.TrimSpace(strings.Join(lines[1:], "\n"))
	if remainder == "" && len(fields) > 1 {
		remainder = strings.TrimSpace(strings.Join(fields[1:], " "))
	}
	return order, remainder, nil
}

func FormatBrief(brief Brief) string {
	return fmt.Sprintf("TITLE: %s\nSUBTITLE: %s\nPROMISE: %s\nVOICE_TONE: %s\nWHAT_IT_IS: %s\nWHAT_IT_IS_NOT: %s\nAI_SUGGESTIONS: %s",
		brief.Title,
		brief.Subtitle,
		brief.Promise,
		brief.VoiceTone,
		brief.WhatItIs,
		brief.WhatItIsNot,
		brief.AISuggestions,
	)
}

func TargetWords(targetLength string) int {
	switch targetLength {
	case db.TargetLengthShortGuide:
		return 5000
	case db.TargetLengthFullPrototype:
		return 40000
	default:
		return 15000
	}
}

func ChapterTargetWords(project ProjectInput) int {
	if project.TargetChapters < 1 {
		return TargetWords(project.TargetLength)
	}
	return TargetWords(project.TargetLength) / project.TargetChapters
}

func parseLabelBlock(text string, labels []string) map[string]string {
	result := make(map[string]string, len(labels))
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	current := ""
	labelSet := make(map[string]bool, len(labels))
	for _, label := range labels {
		labelSet[label] = true
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		found := ""
		for _, label := range labels {
			prefix := label + ":"
			if strings.HasPrefix(trimmed, prefix) {
				found = label
				result[label] = strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
				break
			}
		}
		if found != "" {
			current = found
			continue
		}
		if current != "" && trimmed != "" && !labelSet[trimmed] {
			if result[current] != "" {
				result[current] += "\n"
			}
			result[current] += trimmed
		}
	}
	return result
}

func emptyFallback(value string) string {
	if strings.TrimSpace(value) == "" {
		return "None provided."
	}
	return value
}

func formatProject(project ProjectInput) string {
	var b strings.Builder
	b.WriteString("Title: " + project.Title + "\n")
	b.WriteString("Type: " + project.BookType + "\n")
	b.WriteString(fmt.Sprintf("Target Chapters: %d\n", project.TargetChapters))
	keys := make([]string, 0, len(project.Intake))
	for key := range project.Intake {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b.WriteString(key + ": " + project.Intake[key] + "\n")
	}
	return b.String()
}
