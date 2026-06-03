package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ==========================================
// CONFIGURATION & STYLING
// ==========================================

var (
	apiKey  = os.Getenv("LLM_API_KEY")
	baseURL = os.Getenv("LLM_BASE_URL")
	model   = os.Getenv("LLM_MODEL")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF79C6")).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	doneStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
	docStyle  = lipgloss.NewStyle().Margin(1, 2)
)

func init() {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
}

// ==========================================
// LLM API CLIENT PIPELINE
// ==========================================

type APIRequest struct {
	Model    string       `json:"model"`
	Messages []APIMessage `json:"messages"`
}

type APIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type APIResponse struct {
	Choices []struct {
		Message APIMessage `json:"message"`
	} `json:"choices"`
}

func callLLM(system, user string) (string, error) {
	if apiKey == "" {
		time.Sleep(1 * time.Second)
		return "[SYSTEM ERROR] No LLM_API_KEY detected in environment. Setup required.", nil
	}

	reqBody := APIRequest{
		Model: model,
		Messages: []APIMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("HTTP-Referer", "http://localhost:8080")
	req.Header.Set("X-Title", "AI Publisher TUI")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var apiResp APIResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return "", fmt.Errorf("API parse error: %s", string(bodyBytes))
	}

	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("unexpected API response: %s", string(bodyBytes))
}

// ==========================================
// BUBBLE TEA MESSAGES & COMMANDS
// ==========================================

type resultMsg struct {
	phase   string
	content string
}
type errorMsg struct{ err error }

func generatePhase1() tea.Msg {
	sys := "You are an expert commercial acquisition editor for high-performing BookTok Romantasy."
	user := "Generate a high-stakes Romantasy core concept. Include FMC, MMC, magic system, threat, and two tropes. Keep it brief and punchy."
	res, err := callLLM(sys, user)
	if err != nil {
		return errorMsg{err}
	}
	return resultMsg{phase: "plot", content: res}
}

func generatePhase2(plot string) tea.Cmd {
	return func() tea.Msg {
		sys := "You are a narrative architect."
		user := fmt.Sprintf("Based on this plot:\n%s\n\nGenerate a 5-chapter structured outline. Format as a markdown list.", plot)
		res, err := callLLM(sys, user)
		if err != nil {
			return errorMsg{err}
		}
		return resultMsg{phase: "toc", content: res}
	}
}

func generatePhase3(plot string) tea.Cmd {
	return func() tea.Msg {
		sys := "You are an AI production designer."
		user := fmt.Sprintf("Based on this plot:\n%s\n\nProvide an artistic Midjourney prompt for the climax chapter illustration, and a master marketing blurb for the back cover.", plot)
		res, err := callLLM(sys, user)
		if err != nil {
			return errorMsg{err}
		}
		return resultMsg{phase: "cover", content: res}
	}
}

// ==========================================
// BUBBLE TEA MODEL STATE
// ==========================================

type modelState int

const (
	stateGenerating modelState = iota
	stateDone
	stateError
)

type appModel struct {
	state       modelState
	spinner     spinner.Model
	viewport    viewport.Model
	status      string
	document    string // Fixed: Swapped strings.Builder for a standard string
	corePlot    string
	err         error
	windowWidth int
}

func initialModel() appModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return appModel{
		state:   stateGenerating,
		spinner: s,
		status:  "Initializing AI Engine and drafting Core Concept...",
	}
}

func (m appModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, generatePhase1)
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "s":
			if m.state == stateDone {
				err := os.WriteFile("romantasy_blueprint.md", []byte(m.document), 0644)
				if err == nil {
					m.status = "Successfully saved to romantasy_blueprint.md!"
				} else {
					m.status = "Error saving file: " + err.Error()
				}
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		headerHeight := 6
		footerHeight := 2

		if m.viewport.Width == 0 {
			m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.document)
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - headerHeight - footerHeight
		}

	case resultMsg:
		// Fixed: Simple string concatenation
		m.document += fmt.Sprintf("\n## Phase: %s\n\n%s\n\n", strings.ToUpper(msg.phase), msg.content)
		m.viewport.SetContent(m.document)
		m.viewport.GotoBottom()

		switch msg.phase {
		case "plot":
			m.corePlot = msg.content
			m.status = "Core Concept secured. Drafting Chapter framework..."
			cmds = append(cmds, generatePhase2(m.corePlot))
		case "toc":
			m.status = "Table of Contents wired. Rendering Cover Art specs and Blurb..."
			cmds = append(cmds, generatePhase3(m.corePlot))
		case "cover":
			m.status = "Pipeline Execution Complete."
			m.state = stateDone
		}

	case errorMsg:
		m.err = msg.err
		m.state = stateError
		return m, tea.Quit

	case spinner.TickMsg:
		if m.state == stateGenerating {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	if m.state == stateDone {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m appModel) View() string {
	if m.state == stateError {
		return warnStyle.Render(fmt.Sprintf("Catastrophic Pipeline Failure: %v", m.err))
	}

	header := titleStyle.Render("🔮 AI PUBLISHING ARCHITECT: ROMANTASY ENGINE")

	var body string
	if m.state == stateGenerating {
		body = fmt.Sprintf("\n\n %s %s", m.spinner.View(), infoStyle.Render(m.status))
		if len(m.document) > 0 { // Fixed: using len() instead of .Len()
			body += "\n\n" + lipgloss.NewStyle().Faint(true).Render("Scroll lock engaged while generating...")
		}
	} else {
		header += doneStyle.Render(fmt.Sprintf("\n\n %s", m.status))
		body = m.viewport.View()
	}

	footer := ""
	if m.state == stateDone {
		footer = infoStyle.Render("\n[↑/↓]: Scroll • [s]: Save to File • [q]: Quit")
	} else {
		footer = warnStyle.Render("\n[q]: Abort Sequence")
	}

	return docStyle.Render(header + "\n" + body + "\n" + footer)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting TUI: %v", err)
		os.Exit(1)
	}
}
