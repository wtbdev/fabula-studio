package agent

import (
	"testing"

	"github.com/fabula-studio/backend/internal/scene"
)

func TestNormalizeScenePlansJSONAcceptsWrappedCamelCaseOutput(t *testing.T) {
	raw := `{
		"scenes": [
			{
				"id": "scene-001",
				"sequence": 1,
				"sceneCount": 1,
				"sourceBeatIds": ["candidate_001"],
				"timeFrame": "当天早上",
				"keyPlotPoints": ["主任要求我带上一双特殊眼睛", {"unexpected": true}],
				"omitDetails": ""
			}
		]
	}`

	normalized := normalizeScenePlansJSON(raw)
	plans, err := parseScenePlansJSON(normalized)
	if err != nil {
		t.Fatalf("unexpected parse error: %v\n%s", err, normalized)
	}
	if len(plans) != 1 {
		t.Fatalf("expected one plan, got %d", len(plans))
	}
	plan := plans[0]
	if plan.SceneCount != 1 {
		t.Fatalf("unexpected scene count: %d", plan.SceneCount)
	}
	if len(plan.SourceCandidateIDs) != 1 || plan.SourceCandidateIDs[0] != "candidate_001" {
		t.Fatalf("unexpected source IDs: %#v", plan.SourceCandidateIDs)
	}
	if plan.TimeFrame != "当天早上" {
		t.Fatalf("unexpected time frame: %q", plan.TimeFrame)
	}
	if len(plan.KeyPlotPoints) != 1 || plan.KeyPlotPoints[0] != "主任要求我带上一双特殊眼睛" {
		t.Fatalf("unexpected key points: %#v", plan.KeyPlotPoints)
	}
	if len(plan.OmitDetails) != 0 {
		t.Fatalf("unexpected omit details: %#v", plan.OmitDetails)
	}
}

func TestParseScenePlansJSONAcceptsSinglePlanObject(t *testing.T) {
	raw := `{
		"characters":["叙述者","她"],
		"id":"plan_020",
		"key_plot_points":["叙述者指出她在乎平凡事物","两人沉默无语"],
		"location":"从日落处返回的途中",
		"omit_details":["具体的路径描述"],
		"purpose":"揭示核心冲突",
		"scene_count":1,
		"sequence":20,
		"source_candidate_ids":["candidate_020"],
		"time_frame":"夜晚（星星出现）"
	}`

	normalized := normalizeScenePlansJSON(raw)
	plans, err := parseScenePlansJSON(normalized)
	if err != nil {
		t.Fatalf("unexpected parse error: %v\n%s", err, normalized)
	}
	if len(plans) != 1 {
		t.Fatalf("expected one plan, got %d", len(plans))
	}
	plan := plans[0]
	if plan.ID != "plan_020" {
		t.Fatalf("unexpected id: %q", plan.ID)
	}
	if plan.SceneCount != 1 {
		t.Fatalf("unexpected scene count: %d", plan.SceneCount)
	}
	if len(plan.SourceCandidateIDs) != 1 || plan.SourceCandidateIDs[0] != "candidate_020" {
		t.Fatalf("unexpected source IDs: %#v", plan.SourceCandidateIDs)
	}
	if len(plan.Characters) != 2 || plan.Characters[1] != "她" {
		t.Fatalf("unexpected characters: %#v", plan.Characters)
	}
	if len(plan.KeyPlotPoints) != 2 {
		t.Fatalf("unexpected key points: %#v", plan.KeyPlotPoints)
	}
	if len(plan.OmitDetails) != 1 {
		t.Fatalf("unexpected omit details: %#v", plan.OmitDetails)
	}
	if plan.Sequence != 20 {
		t.Fatalf("unexpected sequence: %d", plan.Sequence)
	}
	if plan.TimeFrame != "夜晚（星星出现）" {
		t.Fatalf("unexpected time frame: %q", plan.TimeFrame)
	}
}

func TestValidateAndRepairScenePlansNormalizesSmallModelOutput(t *testing.T) {
	plans := []*scene.ScenePlan{
		{
			ID:                 "scene-001",
			SourceCandidateIDs: []string{"candidate_001"},
			SceneCount:         1,
			Sequence:           99,
			KeyPlotPoints:      []string{"保留核心任务"},
		},
	}
	candidates := []scene.SceneCandidate{
		{
			ID:              "candidate_001",
			Summary:         "主任要求我带上一双特殊眼睛",
			DramaticPurpose: "建立核心科幻设定",
			Location:        "办公室",
			TimeFrame:       "当天早上",
			Characters:      []string{"我", "主任"},
		},
	}

	repaired := scene.ValidateAndRepairScenePlans(plans, candidates)
	if len(repaired) != 1 {
		t.Fatalf("expected one repaired plan, got %d", len(repaired))
	}
	if repaired[0].Sequence != 1 {
		t.Fatalf("expected sequence repair to 1, got %d", repaired[0].Sequence)
	}
	if repaired[0].Purpose != "建立核心科幻设定" {
		t.Fatalf("unexpected purpose: %q", repaired[0].Purpose)
	}
	if repaired[0].Location != "办公室" {
		t.Fatalf("unexpected location: %q", repaired[0].Location)
	}
}
