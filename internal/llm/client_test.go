package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFromEnvDefaultsToQwenModel(t *testing.T) {
	oldMock := os.Getenv("LLM_MOCK")
	oldModel := os.Getenv("LOCAL_LLM_MODEL")
	t.Cleanup(func() {
		_ = os.Setenv("LLM_MOCK", oldMock)
		_ = os.Setenv("LOCAL_LLM_MODEL", oldModel)
	})
	_ = os.Unsetenv("LLM_MOCK")
	_ = os.Unsetenv("LOCAL_LLM_MODEL")

	client, ok := FromEnv().(*LocalClient)
	if !ok {
		t.Fatalf("expected LocalClient when LLM_MOCK is unset")
	}
	if client.Model != "qwen/qwen3.5-9b" {
		t.Fatalf("default model = %q, want qwen/qwen3.5-9b", client.Model)
	}
}

func TestLocalClientUsesOpenAIChatCompletionsShape(t *testing.T) {
	var request localRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	client := &LocalClient{URL: server.URL, Model: "qwen/qwen3.5-9b", HTTPClient: server.Client()}
	got, err := client.Generate(context.Background(), "system text", "user text")
	if err != nil {
		t.Fatal(err)
	}
	if got != "ok" {
		t.Fatalf("got %q, want ok", got)
	}
	if request.Model != "qwen/qwen3.5-9b" {
		t.Fatalf("model = %q", request.Model)
	}
	if len(request.Messages) != 2 {
		t.Fatalf("messages = %#v", request.Messages)
	}
	if request.Messages[0].Role != "system" || request.Messages[0].Content != "system text" {
		t.Fatalf("bad system message: %#v", request.Messages[0])
	}
	if request.Messages[1].Role != "user" || request.Messages[1].Content != "user text" {
		t.Fatalf("bad user message: %#v", request.Messages[1])
	}
}
