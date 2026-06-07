package util

import (
	"encoding/json"
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

func TestPrepareJSONRepairsMalformedLLMArrayField(t *testing.T) {
	raw := `{
		"scenes": [
			{
				"id": "scene-001",
				"sequence": 1,
				"keyPlotPoints": ["我被告知任务的异常性",
				"expectedChanges": {
					"characterChanges": ["我从普通人转为任务人"]
				}
			}
		],
		"warnings": []
	}`

	repaired, err := PrepareJSON(raw, "scene planner output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid([]byte(repaired)) {
		t.Fatalf("repaired payload is not valid JSON: %s", repaired)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(repaired), &payload); err != nil {
		t.Fatalf("failed to unmarshal repaired payload: %v\n%s", err, repaired)
	}
	if _, ok := payload["scenes"].([]interface{}); !ok {
		t.Fatalf("unexpected repaired payload: %+v", payload)
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

func TestPrepareYAMLRepairsBareHeading(t *testing.T) {
	raw := "外景 草原 - 日间\nsetting:\n  location: 草原"
	got, err := PrepareYAML(raw, "scene writer output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, `heading: "外景 草原 - 日间"`) {
		t.Fatalf("expected heading prefix, got: %q", got)
	}
}

func TestPrepareYAMLDoesNotDoubleWrapValidKey(t *testing.T) {
	raw := "id: scene_001\nheading: 外景 客厅 - 日"
	got, err := PrepareYAML(raw, "scene writer output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(got, "id:") {
		t.Fatalf("expected unchanged prefix, got: %q", got)
	}
}
