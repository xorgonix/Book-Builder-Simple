package db

type UIState struct {
	Version            int            `json:"version"`
	ActiveView         string         `json:"active_view"`
	ActivePanel        string         `json:"active_panel"`
	ActiveTab          string         `json:"active_tab"`
	TargetBookLength   string         `json:"target_book_length"`
	TargetChapterCount int            `json:"target_chapter_count"`
	ExpandedSections   []string       `json:"expanded_sections"`
	ScrollPositions    map[string]int `json:"scroll_positions"`
	FocusTarget        string         `json:"focus_target"`
	Polling            UIStatePolling `json:"polling"`
}

type UIStatePolling struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
}

func DefaultUIState() UIState {
	return UIState{
		Version:            1,
		ActiveView:         "intake",
		ActivePanel:        "workspace",
		ActiveTab:          "brief",
		TargetBookLength:   TargetLengthPracticalEbook,
		TargetChapterCount: 10,
		ExpandedSections: []string{
			"book-variables",
			"generation-pack",
			"editorial-cockpit",
		},
		ScrollPositions: map[string]int{
			"left":   0,
			"center": 0,
			"right":  0,
		},
		FocusTarget: "working-title",
		Polling: UIStatePolling{
			Enabled:  false,
			Endpoint: "/api/project/{id}/status",
		},
	}
}
