package scene

import (
	"encoding/json"
	"testing"

	"github.com/fabula-studio/backend/internal/segment"
)

func TestScenePlanUnmarshalAcceptsSummaryOnlyBoolFalse(t *testing.T) {
	var plan ScenePlan
	data := []byte(`{"id":"plan_001","source_node_ids":["node_001"],"scene_count":1,"omit_details":["skip"],"summary_only":false}`)
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatalf("unmarshal scene plan: %v", err)
	}
	if plan.SummaryOnly != "" {
		t.Fatalf("expected empty summary_only for false, got %q", plan.SummaryOnly)
	}
}

func TestScenePlanUnmarshalAcceptsOmitDetailsString(t *testing.T) {
	var plan ScenePlan
	data := []byte(`{"id":"plan_001","source_node_ids":["node_001"],"scene_count":1,"omit_details":"skip this"}`)
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatalf("unmarshal scene plan: %v", err)
	}
	if len(plan.OmitDetails) != 1 || plan.OmitDetails[0] != "skip this" {
		t.Fatalf("unexpected omit_details: %#v", plan.OmitDetails)
	}
}

func TestBuildSceneCandidatesPreservesBeatGrounding(t *testing.T) {
	beats := []segment.StoryBeat{{
		ID:                "beat_001",
		Sequence:          1,
		SourceSentenceIDs: []string{"s_001", "s_002"},
		Summary:           "  finds the hatch  ",
		DramaticPurpose:   " reveal danger ",
		Characters:        []string{"Ada", "Ada", ""},
		Location:          "Lab",
	}}
	candidates := BuildSceneCandidates(beats)
	if len(candidates) != 1 {
		t.Fatalf("expected one candidate, got %d", len(candidates))
	}
	got := candidates[0]
	if got.ID != "candidate_001" || got.SourceBeatIDs[0] != "beat_001" || got.SourceSentenceIDs[1] != "s_002" {
		t.Fatalf("candidate lost grounding: %#v", got)
	}
	if got.Summary != "finds the hatch" || got.DramaticPurpose != "reveal danger" {
		t.Fatalf("candidate fields were not normalized: %#v", got)
	}
	if len(got.Characters) != 1 || got.Characters[0] != "Ada" {
		t.Fatalf("characters not compacted: %#v", got.Characters)
	}
}

func TestValidateAndRepairScenePlansAddsMissingCandidates(t *testing.T) {
	candidates := []SceneCandidate{
		{ID: "candidate_001", Summary: "first beat", DramaticPurpose: "start", Characters: []string{"Ada"}, Location: "Lab"},
		{ID: "candidate_002", Summary: "second beat", DramaticPurpose: "escape", Characters: []string{"Ben"}, Location: "Tunnel"},
	}
	plans := []*ScenePlan{{
		ID:            "",
		SourceNodeIDs: []string{"candidate_001"},
		SceneCount:    -1,
	}}
	repaired := ValidateAndRepairScenePlans(plans, candidates)
	if len(repaired) != 2 {
		t.Fatalf("expected repaired plan plus missing candidate fallback, got %d", len(repaired))
	}
	if repaired[0].ID != "plan_001" || repaired[0].SceneCount != 1 || repaired[0].Purpose != "start" {
		t.Fatalf("first plan not repaired from candidate: %#v", repaired[0])
	}
	if repaired[1].SourceNodeIDs[0] != "candidate_002" || repaired[1].Purpose != "escape" {
		t.Fatalf("missing candidate not converted to plan: %#v", repaired[1])
	}
}
