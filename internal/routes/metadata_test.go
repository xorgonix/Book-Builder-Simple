package routes

import "testing"

func TestCleanOptionalJSON(t *testing.T) {
	got, err := cleanOptionalJSON(`{"b":2,"a":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if got != "{\n  \"a\": 1,\n  \"b\": 2\n}" {
		t.Fatalf("unexpected normalized JSON:\n%s", got)
	}
	got, err = cleanOptionalJSON("   ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("blank JSON should remain blank, got %q", got)
	}
	if _, err := cleanOptionalJSON(`{"nope":`); err == nil {
		t.Fatal("invalid JSON should fail")
	}
}
