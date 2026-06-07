package pipeline

import (
	"testing"

	"github.com/fabula-studio/backend/internal/graph"
	"github.com/fabula-studio/backend/internal/scene"
	"github.com/fabula-studio/backend/internal/schema"
)

func TestCollectCharactersFromGraphUsesCanonicalIDs(t *testing.T) {
	snap := graph.NewSnapshot("candidate_002")
	snap.Characters["char_002"] = &graph.CharacterState{
		ID:          "char_002",
		Name:        "Ben",
		CurrentGoal: "escape",
		Personality: []string{"cautious"},
	}
	snap.Characters["char_001"] = &graph.CharacterState{ID: "char_001", Name: "Ada"}

	characters := collectCharactersFromGraph(snap)
	byID := make(map[string]schema.Character, len(characters))
	for _, character := range characters {
		byID[character.ID] = character
	}

	if len(byID) != 2 {
		t.Fatalf("expected two characters, got %#v", characters)
	}
	if byID["char_002"].Name != "Ben" || byID["char_002"].Intro != "escape" {
		t.Fatalf("canonical graph character not preserved: %#v", byID["char_002"])
	}
	if len(byID["char_002"].Personality) != 1 || byID["char_002"].Personality[0] != "cautious" {
		t.Fatalf("personality not copied: %#v", byID["char_002"].Personality)
	}
}

func TestValidatePlanGroundingRejectsUnknownCandidate(t *testing.T) {
	plans := []*scene.ScenePlan{{ID: "plan_001", SourceCandidateIDs: []string{"candidate_999"}, SceneCount: 1}}
	candidates := []scene.SceneCandidate{{ID: "candidate_001"}}
	if err := validatePlanGrounding(plans, candidates); err == nil {
		t.Fatalf("expected unknown candidate grounding error")
	}
}

func TestValidatePlanGroundingRequiresFullCoverage(t *testing.T) {
	plans := []*scene.ScenePlan{{ID: "plan_001", SourceCandidateIDs: []string{"candidate_001"}, SceneCount: 1}}
	candidates := []scene.SceneCandidate{{ID: "candidate_001"}, {ID: "candidate_002"}}
	if err := validatePlanGrounding(plans, candidates); err == nil {
		t.Fatalf("expected uncovered candidate grounding error")
	}
}

func TestValidatePlanGroundingAcceptsCoveredCandidates(t *testing.T) {
	plans := []*scene.ScenePlan{{ID: "plan_001", SourceCandidateIDs: []string{"candidate_001", "candidate_002"}, SceneCount: 1}}
	candidates := []scene.SceneCandidate{{ID: "candidate_001"}, {ID: "candidate_002"}}
	if err := validatePlanGrounding(plans, candidates); err != nil {
		t.Fatalf("expected valid grounding, got %v", err)
	}
}

func TestRequiredSourceChapterCoverageAllowsShortInputs(t *testing.T) {
	cases := map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 8: 3}
	for chapters, want := range cases {
		if got := requiredSourceChapterCoverage(chapters); got != want {
			t.Fatalf("requiredSourceChapterCoverage(%d) = %d, want %d", chapters, got, want)
		}
	}
}
