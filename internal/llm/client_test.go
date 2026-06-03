package llm

import (
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
