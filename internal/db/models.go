package db

type Project struct {
	Id             string `json:"id"`
	Title          string `json:"title"`
	BookType       string `json:"type"`
	Status         string `json:"status"`
	TargetLength   string `json:"target_length"`
	TargetChapters int    `json:"target_chapters"`
	UIState        string `json:"ui_state"`
	VoiceCard      string `json:"voice_card"`
	Created        string `json:"created"`
	Updated        string `json:"updated"`
}

type IntakeResponse struct {
	Id        string `json:"id"`
	ProjectId string `json:"project_id"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

type BookBrief struct {
	Id            string `json:"id"`
	ProjectId     string `json:"project_id"`
	Title         string `json:"title"`
	Subtitle      string `json:"subtitle"`
	Promise       string `json:"promise"`
	VoiceTone     string `json:"voice_tone"`
	WhatItIs      string `json:"what_it_is"`
	WhatItIsNot   string `json:"what_it_is_not"`
	AISuggestions string `json:"ai_suggestions"`
}

type Chapter struct {
	Id                   string `json:"id"`
	ProjectId            string `json:"project_id"`
	SortOrder            int    `json:"sort_order"`
	Title                string `json:"title"`
	Status               string `json:"status"`
	Purpose              string `json:"purpose"`
	StateStart           string `json:"state_start"`
	StateEnd             string `json:"state_end"`
	RawDraft             string `json:"raw_draft"`
	EditorialDiagnosis   string `json:"editorial_diagnosis"`
	TargetedRewrite      string `json:"targeted_rewrite"`
	DraftContent         string `json:"draft_content"`
	PreviousDraftContent string `json:"previous_draft_content"`
	UserDraftNotes       string `json:"user_draft_notes"`
	UserDiagnosisNotes   string `json:"user_diagnosis_notes"`
	UserRewriteNotes     string `json:"user_rewrite_notes"`
	ManualEditContent    string `json:"manual_edit_content"`
}

type Job struct {
	Id           string `json:"id"`
	ProjectId    string `json:"project_id"`
	ChapterId    string `json:"chapter_id"`
	JobType      string `json:"job_type"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_msg"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at"`
}
