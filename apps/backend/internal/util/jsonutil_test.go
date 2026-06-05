package util

import (
	"strings"
	"testing"
)

func TestPrepareJSONRejectsEmptyResponse(t *testing.T) {
	_, err := PrepareJSON("   ", "scene planner output")
	if err == nil {
		t.Fatal("expected empty response error")
	}
	if !strings.Contains(err.Error(), "empty model response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareJSONRejectsNonJSONResponse(t *testing.T) {
	_, err := PrepareJSON("模型没有返回结构化内容", "node analysis output")
	if err == nil {
		t.Fatal("expected no JSON payload error")
	}
	if !strings.Contains(err.Error(), "no JSON payload") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareJSONExtractsFencedPayload(t *testing.T) {
	got, err := PrepareJSON("prefix\n```json\n{\"ok\":true}\n```\nsuffix", "test output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != `{"ok":true}` {
		t.Fatalf("unexpected payload: %s", got)
	}
}

func TestPrepareYAMLRejectsEmptyResponse(t *testing.T) {
	_, err := PrepareYAML("\n\t ", "scene writer output")
	if err == nil {
		t.Fatal("expected empty response error")
	}
	if !strings.Contains(err.Error(), "empty model response") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareYAMLStripsFence(t *testing.T) {
	got, err := PrepareYAML("```yaml\nid: s1\n```", "scene writer output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "id: s1" {
		t.Fatalf("unexpected YAML: %q", got)
	}
}
