package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client interface {
	Generate(ctx context.Context, systemPrompt string, input string) (string, error)
}

func FromEnv() Client {
	if os.Getenv("LLM_MOCK") == "1" {
		return MockClient{}
	}
	url := strings.TrimSpace(os.Getenv("LOCAL_LLM_URL"))
	if url == "" {
		url = "http://localhost:1234/api/v1/chat"
	}
	model := strings.TrimSpace(os.Getenv("LOCAL_LLM_MODEL"))
	if model == "" {
		model = "qwen/qwen3.5-9b"
	}
	return &LocalClient{
		URL:        url,
		Model:      model,
		HTTPClient: &http.Client{Timeout: 10 * time.Minute},
	}
}

type LocalClient struct {
	URL        string
	Model      string
	HTTPClient *http.Client
}

type localRequest struct {
	Model        string `json:"model"`
	SystemPrompt string `json:"system_prompt"`
	Input        string `json:"input"`
}

type localResponse struct {
	Output []struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	} `json:"output"`
}

func (c *LocalClient) Generate(ctx context.Context, systemPrompt string, input string) (string, error) {
	body, err := json.Marshal(localRequest{
		Model:        c.Model,
		SystemPrompt: systemPrompt,
		Input:        input,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("local llm returned status %d", resp.StatusCode)
	}
	var parsed localResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	for _, item := range parsed.Output {
		if strings.TrimSpace(item.Content) != "" {
			return strings.TrimSpace(item.Content), nil
		}
	}
	return "", errors.New("local llm response did not include output content")
}

type MockClient struct{}

func (MockClient) Generate(ctx context.Context, systemPrompt string, input string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	switch {
	case strings.Contains(input, "CHAPTER_START"):
		return mockTOC(input), nil
	case strings.Contains(input, "TITLE:"):
		return mockBrief(), nil
	case strings.Contains(input, "Return three"):
		return "A reader who knows they need a book, but cannot yet name the shape of it.\nA practical guide for turning scattered expertise into a useful manuscript.\nA clear promise that avoids hype and keeps the book grounded.", nil
	case strings.Contains(input, "UNEDITED DRAFT CHAPTER"):
		return "- CRITICAL DIAGNOSIS: The chapter has a clear spine but needs sharper movement.\n- TARGETED FIXES: Strengthen transitions, vary rhythm, and add more specific claims.\n- SPECIFIC PASSAGES: Replace generic lines with concrete language.\n- PROTECTED ELEMENTS: Keep the chapter purpose and reader transformation intact.", nil
	default:
		return "This is deterministic mock manuscript text for local development. It preserves the requested chapter purpose while the real local LLM endpoint is unavailable.", nil
	}
}

func mockBrief() string {
	return `TITLE: The Practical Book Blueprint
SUBTITLE: Turn a strong idea into a structured manuscript prototype
PROMISE: Readers will understand how to turn a scattered book concept into a clear, buildable manuscript plan.
VOICE_TONE: Warm, direct, structurally rigorous
WHAT_IT_IS: A practical working guide for shaping a book from intent through chapters.
WHAT_IT_IS_NOT: It is not a vague motivational manifesto or a generic writing pep talk.
AI_SUGGESTIONS: Keep the structure concrete, preserve the reader transformation, and make each chapter earn its place.`
}

func mockTOC(input string) string {
	count := 10
	marker := "You must generate exactly "
	if idx := strings.Index(input, marker); idx >= 0 {
		rest := input[idx+len(marker):]
		fields := strings.Fields(rest)
		if len(fields) > 0 {
			if parsed, err := strconv.Atoi(fields[0]); err == nil && parsed > 0 {
				count = parsed
			}
		}
	}
	var b strings.Builder
	for i := 1; i <= count; i++ {
		fmt.Fprintf(&b, `CHAPTER_START
Order: %d
Title: Prototype Chapter %d
Purpose: Move the reader through manuscript-building step %d.
Reader Start: The reader has an unresolved planning question.
Reader End: The reader has a concrete decision for this part of the book.
CHAPTER_END

`, i, i, i)
	}
	return strings.TrimSpace(b.String())
}
