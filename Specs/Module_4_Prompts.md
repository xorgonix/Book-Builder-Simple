`````
# Module 4: LLM Prompt Chaining & Structural Prompts

## 1. Engine Parsing Guardrail Rules
* Output parsing must never assume the existence of Markdown formatting boundaries unless explicitly defined. 
* Instruct the LLM explicitly **never** to wrap structural outputs inside Markdown syntax markers (e.g., ` ```json ` or ` ```xml `) to prevent system regex processing compilation exceptions.

---

## 2. Core Book Staging Prompts

### Prompt A: The Book Brief Generator
```

You are an elite structural book editor and publishing strategist. Your task is to process raw intake parameters and transform them into a comprehensive, professional Book Brief.

USER INTAKE NOTES:

- Book Vector Type: [Book Type: Fiction/Non-Fiction]
    
- Core System Topic: [Value from intake_responses: core_topic]
    
- Target Reader Profile: [Value from intake_responses: target_audience]
    
- Deep Reader Hunger/Desire: [Value from intake_responses: reader_hunger]
    
- Prohibited Directions (What NOT to become): [Value from intake_responses: prohibited_directions]
    
- Stated Author Tone Target: [Value from intake_responses: selected_tone]
    

Target Manuscript Scale:

- Total Target Word Count: [Derived from target_length: short_guide=5k, practical_ebook=15k, full_prototype=40k]
    
- Intended Chapter Count: [Value from projects.target_chapters]
    

You must output your analysis matching this exact text layout layout below. Do not wrap the output in Markdown code blocks or include any introductory conversational comments.

TITLE: [Generate an engaging, high-impact working title]

SUBTITLE: [Generate a subtitle that clarifies the core hook]

PROMISE: [A single-sentence value proposition addressing the Deep Reader Hunger]

VOICE_TONE: [Extract 3 definitive stylistic parameters matching the Selected Tone]

WHAT_IT_IS: [A detailed overview description of the book's delivery framework]

WHAT_IT_IS_NOT: [Strict boundary constraints directly addressing the Prohibited Directions]

AI_SUGGESTIONS: [A critical structural evaluation of how to execute this concept across the requested chapter count]

```

### Prompt B: The Table of Contents (TOC) Generator
```

You are a master book architect. Your task is to generate a complete, structural Table of Contents for a book based on the provided Book Brief.

VERIFIED BOOK BRIEF:

[Pasted Content from book_briefs record]

MANUSCRIPT METRICS:

- Total Targeted Chapters: [Value from projects.target_chapters]
    
- Target Chapter Word Count Boundary: [Dynamically calculated: Target Words / Chapter Count, e.g., 1500 words per chapter]
    

You must generate exactly [Target Chapters] individual chapter entries. Each entry must be cleanly wrapped inside the active parser markers CHAPTER_START and CHAPTER_END. Do not include any introductory commentary or markdown code blocks.

CHAPTER_START

Order: [Sequence Integer starting at 1]

Title: [A compelling, clear chapter title]

Purpose: [What the chapter must mechanically accomplish to advance the book's promise]

Reader Start: [The precise emotional or intellectual frustration of the reader on entry]

Reader End: [The transformation goal or clarity target of the reader upon exiting this chapter]

CHAPTER_END

```

---

## 3. The 4-Stage Cockpit Prompt Processing Hierarchy

### Phase 1: The Chapter Draft Prompt
```

Write a complete, fully detailed book chapter based on the provided structural outline, baseline profile details, and project variables.

PROJECT CONTEXT BRIEF:

[Pasted Content from book_briefs]

CHAPTER ARCHITECTURAL OBJECTIVE:

- Title: [Active Chapter Title]
    
- Core Purpose: [Active Chapter Purpose Specification]
    
- Reader Entry State: [Reader State at Beginning]
    
- Reader Exit State: [Reader State at End]
    
- Dynamic Target Chapter Length: [Calculated Word Constraint, e.g., 1500 words]
    

USER PRE-DRAFT CONSTRAINT NOTES (High Priority):

[Pasted Content from chapters.user_draft_notes, if provided]

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
    

Output: Return the complete chapter draft text only. Do not add introductory or concluding assistant commentary.

```

### Phase 2: The Chapter Diagnosis Prompt
```

Analyze the attached draft chapter from an editor's perspective. Your job is to identify structural issues and areas for improvement before making edits.

PROJECT BRIEF:

[Pasted Content from book_briefs]

UNEDITED DRAFT CHAPTER:

[Pasted Content from chapters.raw_draft]

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

- CRITICAL DIAGNOSIS: [Comprehensive overview of the chapter's performance]
    
- TARGETED FIXES: [5 to 10 specific structural improvements required]
    
- SPECIFIC PASSAGES: [Quote 3 weak sentences and explain why they fail]
    
- PROTECTED ELEMENTS: [Clearly identify elements that work well and should not be changed]
    

```

### Phase 3: The Targeted Rewrite Prompt
```

Rewrite the provided chapter text by applying the targeted editorial fixes outlined in the structural diagnosis and incorporating the user's manual revisions.

PROJECT BRIEF & TARGET INTENT:

[Pasted Content from book_briefs]

EDITORIAL DIAGNOSIS & ACTION PLAN:

[Pasted Content from chapters.editorial_diagnosis]

USER'S MANUAL CRITIQUE OVERRIDE (High Priority):

[Pasted Content from chapters.user_diagnosis_notes]

ORIGINAL CHAPTER TEXT:

[Pasted Content from chapters.raw_draft]

Execution Rewrite Constraints:

- Preserve the underlying arguments, chapter sequence, and factual claims unless the diagnosis explicitly directs a change.
    
- Significantly improve sentence variety, rhythmic flow, and narrative clarity.
    
- Completely remove generic, uninspired AI phrasing patterns.
    
- Ensure the prose reads like a single voice with a clear perspective.
    
- Do not invent new stories, unverified statistics, or external quotes.
    
- Do not make text patterns artificially repetitive.
    
- Avoid concluding sections with a generic inspirational summary.
    

Output: Return the rewritten chapter text only. Do not add introductory or concluding commentary.

```

### Phase 4: The Pattern Cleanup Pass
```

Perform a structural edit on this text to remove common AI writing patterns and stylistic tells.

TARGET TEXT:

[Pasted Content from chapters.targeted_rewrite]

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

Output: Return the polished text only.

## 6. Verification and Compilation Instruction

When executing this project with your choice AI development tool, follow this strict sequence:

1. Initialize the project directory using Go modules (`go mod init bookbuilder`).
    
2. Implement **Section 2 (`MODULE_1_DB.md`)** inside the standard migrations folder to confirm that your fields compile correctly under the embedded PocketBase storage footprint.
    
3. Drop **Section 3 (`MODULE_2_ROUTING.md`)** into your app setup routines, ensuring that your background Goroutine tasks have independent contexts decoupled from incoming user connection channels.
    
4. Render the dynamic UI fragments specified in **Section 4 (`MODULE_3_VIEWS.md`)** through standard, native `html/template` injections.
    
5. Map your system API generation routes cleanly to the bounded prompt contexts defined in **Section 5 (`MODULE_4_PROMPTS.md`)**.
    

This comprehensive blueprint gives you a clean, reliable codebase to test out your design ideas. Feel free to let me know if you run into any local environment questions as you launch the initial build case!