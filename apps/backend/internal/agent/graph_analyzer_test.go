package agent

import (
	"testing"

	"github.com/fabula-studio/backend/internal/graph"
)

func TestDiffSnapshotsPreservesExistingRelationUpdates(t *testing.T) {
	original := graph.NewSnapshot("candidate_001")
	original.Relations = []graph.Relation{{CharA: "char_001", CharB: "char_002", Type: "stranger", Description: "met briefly"}}

	modified := original.Clone()
	modified.Relations[0].Type = "ally"
	modified.Relations[0].Description = "escaped together"

	update := diffSnapshots(original, modified)
	if len(update.RelationChanges) != 1 {
		t.Fatalf("expected one relation update, got %#v", update.RelationChanges)
	}
	change := update.RelationChanges[0]
	if change.CharA != "char_001" || change.CharB != "char_002" || change.NewType != "ally" || change.NewDescription != "escaped together" {
		t.Fatalf("unexpected relation change: %#v", change)
	}
}

func TestRelationKeyIsDirectionAgnostic(t *testing.T) {
	if relationKey("char_002", "char_001") != relationKey("char_001", "char_002") {
		t.Fatalf("relation key should be direction agnostic")
	}
}
