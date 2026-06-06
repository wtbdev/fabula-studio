package scene

import (
	"encoding/json"
	"testing"
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
