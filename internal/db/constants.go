package db

const (
	CollectionProjects        = "projects"
	CollectionIntakeResponses = "intake_responses"
	CollectionBookBriefs      = "book_briefs"
	CollectionChapters        = "chapters"
	CollectionJobs            = "jobs"
)

const (
	ProjectStatusIntake     = "intake"
	ProjectStatusProcessing = "processing"
	ProjectStatusBrief      = "brief"
	ProjectStatusTOC        = "toc"
	ProjectStatusDrafting   = "drafting"
	ProjectStatusFailedJob  = "error_failed_job"
)

const (
	ChapterStatusPending        = "pending"
	ChapterStatusCardApproved   = "card_approved"
	ChapterStatusDraftingStage1 = "drafting_stage_1"
	ChapterStatusDraftingStage2 = "drafting_stage_2"
	ChapterStatusDraftingStage3 = "drafting_stage_3"
	ChapterStatusCompleted      = "completed"
)

const (
	TargetLengthShortGuide     = "short_guide"
	TargetLengthPracticalEbook = "practical_ebook"
	TargetLengthFullPrototype  = "full_prototype"
)

var ValidProjectStatuses = map[string]bool{
	ProjectStatusIntake:     true,
	ProjectStatusProcessing: true,
	ProjectStatusBrief:      true,
	ProjectStatusTOC:        true,
	ProjectStatusDrafting:   true,
	ProjectStatusFailedJob:  true,
}

var ValidChapterStatuses = map[string]bool{
	ChapterStatusPending:        true,
	ChapterStatusCardApproved:   true,
	ChapterStatusDraftingStage1: true,
	ChapterStatusDraftingStage2: true,
	ChapterStatusDraftingStage3: true,
	ChapterStatusCompleted:      true,
}

var ValidBookTypes = map[string]bool{
	"fiction":    true,
	"nonfiction": true,
}

var ValidTargetLengths = map[string]bool{
	TargetLengthShortGuide:     true,
	TargetLengthPracticalEbook: true,
	TargetLengthFullPrototype:  true,
}

var ValidIntakeKeys = map[string]bool{
	"core_topic":            true,
	"target_audience":       true,
	"reader_hunger":         true,
	"prohibited_directions": true,
	"author_intent":         true,
	"selected_tone":         true,
	"book_form":             true,
	"narrative_pov":         true,
	"structure_model":       true,
}
