package scene

import (
	"testing"

	"github.com/fabula-studio/backend/internal/graph"
)

func TestContextBuilderUsesCandidateBeforeSnapshot(t *testing.T) {
	before := graph.NewSnapshot("candidate_001")
	before.Characters["char_001"] = &graph.CharacterState{
		ID:             "char_001",
		Name:           "Ada",
		CurrentGoal:    "find the hatch",
		EmotionalState: "alert",
		KnownInfo:      []string{"the lab is locked"},
	}
	before.Characters["char_002"] = &graph.CharacterState{ID: "char_002", Name: "Ben"}
	before.Relations = []graph.Relation{{CharA: "char_001", CharB: "char_002", Type: "ally", Description: "trapped together"}}
	before.UnresolvedConflicts = []graph.Conflict{{Description: "escape the lab", Involved: []string{"char_001"}, Status: "unresolved"}}

	after := before.Clone()
	after.Characters["char_001"].UnknownInfo = []string{"Ben caused the alarm"}

	builder := NewContextBuilder(
		map[string]*graph.GraphSnapshot{"candidate_001": before},
		map[string]*graph.GraphSnapshot{"candidate_001": after},
	)
	plan := &ScenePlan{
		ID:                 "plan_001",
		SourceCandidateIDs: []string{"candidate_001"},
		Purpose:            "force Ada and Ben to cooperate",
		Characters:         []string{"Ada", "char_002"},
	}

	ctx := builder.Build(plan, "source", "summary")
	if len(ctx.Characters) != 2 {
		t.Fatalf("expected two resolved characters, got %#v", ctx.Characters)
	}
	if ctx.Characters[0].ID != "char_001" || ctx.Characters[1].ID != "char_002" {
		t.Fatalf("unexpected character resolution: %#v", ctx.Characters)
	}
	if len(ctx.Relations) != 1 || ctx.Relations[0].Type != "ally" {
		t.Fatalf("expected relation between resolved characters, got %#v", ctx.Relations)
	}
	if len(ctx.KnownFacts) != 1 || ctx.KnownFacts[0] != "the lab is locked" {
		t.Fatalf("expected known fact from before snapshot, got %#v", ctx.KnownFacts)
	}
	if len(ctx.Unresolved) != 1 || ctx.Unresolved[0] != "escape the lab" {
		t.Fatalf("expected unresolved conflict from before snapshot, got %#v", ctx.Unresolved)
	}
	if len(ctx.ForbiddenInfo) != 1 || ctx.ForbiddenInfo[0] != "Ben caused the alarm" {
		t.Fatalf("expected forbidden info from candidate after snapshot, got %#v", ctx.ForbiddenInfo)
	}
}
