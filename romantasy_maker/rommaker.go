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

var (
	baseURL = "http://localhost:1234/v1/chat/completions"
	model   = "dirty-muse-writer-v01-uncensored-erotica-nsfw-i1"

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF79C6")).Border(lipgloss.RoundedBorder()).Padding(0, 1)
	infoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	doneStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B")).Bold(true)
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C"))
	docStyle   = lipgloss.NewStyle().Margin(1, 2)
)

func callLLM(system, user string) (string, error) {
	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	jsonData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LM Studio offline: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("LM Studio returned Error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return "", fmt.Errorf("JSON parse error: %v", err)
	}

	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("unexpected API format")
}

type resultMsg struct{ phase, content string }
type errorMsg struct{ err error }

func generatePhase1() tea.Msg {
	res, err := callLLM("You are a BookTok Romantasy editor.", "Generate a high-stakes Romantasy core concept with an FMC, MMC, magic system, and 2 tropes. Keep it brief.")
	if err != nil {
		return errorMsg{err}
	}
	return resultMsg{"plot", res}
}
func generatePhase2(plot string) tea.Cmd {
	return func() tea.Msg {
		res, err := callLLM("You are a narrative architect.", fmt.Sprintf("Based on this plot:\n%s\n\nGenerate a 5-chapter outline list.", plot))
		if err != nil {
			return errorMsg{err}
		}
		return resultMsg{"toc", res}
	}
}
func generatePhase3(plot string) tea.Cmd {
	return func() tea.Msg {
		res, err := callLLM("You are a production designer.", fmt.Sprintf("Based on this plot:\n%s\n\nProvide an artistic Midjourney prompt and a back cover blurb.", plot))
		if err != nil {
			return errorMsg{err}
		}
		return resultMsg{"cover", res}
	}
}

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
	document    string
	corePlot    string
	err         error
	windowWidth int
}

func initialModel() appModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return appModel{state: stateGenerating, spinner: s, status: "Drafting Core Concept..."}
}

func (m appModel) Init() tea.Cmd { return tea.Batch(m.spinner.Tick, generatePhase1) }

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if msg.String() == "s" && m.state == stateDone {
			os.WriteFile("romantasy_blueprint.md", []byte(m.document), 0644)
			m.status = "Saved successfully to romantasy_blueprint.md!"
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		// Auto word-wrap setup
		wrappedText := lipgloss.NewStyle().Width(m.windowWidth - 4).Render(m.document)
		if m.viewport.Width == 0 {
			m.viewport = viewport.New(msg.Width, msg.Height-8)
			m.viewport.YPosition = 6
			m.viewport.SetContent(wrappedText)
		} else {
			m.viewport.Width, m.viewport.Height = msg.Width, msg.Height-8
			m.viewport.SetContent(wrappedText)
		}
	case resultMsg:
		m.document += fmt.Sprintf("\n## Phase: %s\n\n%s\n\n", strings.ToUpper(msg.phase), msg.content)
		wrappedText := lipgloss.NewStyle().Width(m.windowWidth - 4).Render(m.document)

		if m.viewport.Width > 0 {
			m.viewport.SetContent(wrappedText)
			m.viewport.GotoBottom()
		}
		switch msg.phase {
		case "plot":
			m.corePlot = msg.content
			m.status = "Plot secured. Drafting outline..."
			cmds = append(cmds, generatePhase2(m.corePlot))
		case "toc":
			m.status = "Outline wired. Generating Cover/Blurb..."
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
		return warnStyle.Render(fmt.Sprintf("\nPIPELINE ERROR:\n%v", m.err))
	}
	header := titleStyle.Render("🔮 AI PUBLISHING ARCHITECT (LM STUDIO EDITION)")
	var body string
	if m.state == stateGenerating {
		body = fmt.Sprintf("\n\n %s %s", m.spinner.View(), infoStyle.Render(m.status))
	} else {
		header += doneStyle.Render(fmt.Sprintf("\n\n %s", m.status))
		body = m.viewport.View()
	}
	footer := infoStyle.Render("\n[q]: Quit")
	if m.state == stateDone {
		footer = infoStyle.Render("\n[↑/↓]: Scroll • [s]: Save • [q]: Quit")
	}
	return docStyle.Render(header + "\n" + body + "\n" + footer)
}

func main() {
	// Restored AltScreen! It will look beautiful and clean now.
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Fatal runtime error: %v\n", err)
	}
}
