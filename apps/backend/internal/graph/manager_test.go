package graph

import "testing"

func TestApplyUpdate_NewCharacters(t *testing.T) {
	mgr := NewManager()
	mgr.SetInitialSnapshot("node_001", NewSnapshot("node_001"))

	update := &GraphUpdateResult{
		NewCharacters: []CharacterState{
			{ID: "char_001", Name: "林浩", CurrentGoal: "破案", EmotionalState: "焦虑"},
		},
	}
	after := mgr.ApplyUpdate("node_001", update)

	if len(after.Characters) != 1 {
		t.Fatalf("expected 1 character, got %d", len(after.Characters))
	}
	cs := after.Characters["char_001"]
	if cs.Name != "林浩" {
		t.Errorf("expected name 林浩, got %s", cs.Name)
	}
}

func TestApplyUpdate_CharacterUpdate(t *testing.T) {
	mgr := NewManager()
	snap := NewSnapshot("node_001")
	snap.Characters["char_001"] = &CharacterState{
		ID: "char_001", Name: "林浩", CurrentGoal: "破案", EmotionalState: "焦虑",
	}
	mgr.SetInitialSnapshot("node_001", snap)

	update := &GraphUpdateResult{
		UpdatedCharacters: []CharacterUpdate{
			{ID: "char_001", EmotionChange: "愤怒", NewKnownInfo: "发现关键线索"},
		},
	}
	after := mgr.ApplyUpdate("node_001", update)

	cs := after.Characters["char_001"]
	if cs.EmotionalState != "愤怒" {
		t.Errorf("expected emotion 愤怒, got %s", cs.EmotionalState)
	}
	if len(cs.KnownInfo) != 1 || cs.KnownInfo[0] != "发现关键线索" {
		t.Errorf("unexpected known info: %v", cs.KnownInfo)
	}
}

func TestApplyUpdate_Relations(t *testing.T) {
	mgr := NewManager()
	mgr.SetInitialSnapshot("node_001", NewSnapshot("node_001"))

	update := &GraphUpdateResult{
		RelationChanges: []RelationChange{
			{CharA: "char_001", CharB: "char_002", NewType: "搭档", TriggerEvent: "首次合作"},
		},
	}
	after := mgr.ApplyUpdate("node_001", update)

	if len(after.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(after.Relations))
	}
	if after.Relations[0].Type != "搭档" {
		t.Errorf("expected relation type 搭档, got %s", after.Relations[0].Type)
	}
}

func TestApplyUpdate_Conflicts(t *testing.T) {
	mgr := NewManager()
	snap := NewSnapshot("node_001")
	snap.UnresolvedConflicts = []Conflict{
		{Description: "连环失踪案", Status: "unresolved"},
	}
	mgr.SetInitialSnapshot("node_001", snap)

	update := &GraphUpdateResult{
		ResolvedConflicts: []string{"连环失踪案"},
	}
	after := mgr.ApplyUpdate("node_001", update)

	if after.UnresolvedConflicts[0].Status != "resolved" {
		t.Errorf("expected conflict to be resolved")
	}
	if after.UnresolvedConflicts[0].ResolvedIn != "node_001" {
		t.Errorf("expected resolved in node_001, got %s", after.UnresolvedConflicts[0].ResolvedIn)
	}
}

func TestChainSnapshot(t *testing.T) {
	mgr := NewManager()
	mgr.SetInitialSnapshot("node_001", NewSnapshot("node_001"))

	update := &GraphUpdateResult{
		NewCharacters: []CharacterState{
			{ID: "char_001", Name: "林浩"},
		},
	}
	mgr.ApplyUpdate("node_001", update)
	mgr.ChainSnapshot("node_001", "node_002")

	before2 := mgr.SnapshotsBefore()["node_002"]
	if before2 == nil {
		t.Fatal("expected before snapshot for node_002")
	}
	if len(before2.Characters) != 1 {
		t.Errorf("expected 1 character in chained snapshot, got %d", len(before2.Characters))
	}
	if before2.NodeID != "node_002" {
		t.Errorf("expected node_002, got %s", before2.NodeID)
	}
}

func TestClone(t *testing.T) {
	snap := NewSnapshot("test")
	snap.Characters["c1"] = &CharacterState{
		ID: "c1", KnownInfo: []string{"fact1"}, Personality: []string{"brave"},
	}
	snap.Relations = []Relation{{CharA: "c1", CharB: "c2"}}

	clone := snap.Clone()
	clone.Characters["c1"].KnownInfo = append(clone.Characters["c1"].KnownInfo, "fact2")

	if len(snap.Characters["c1"].KnownInfo) != 1 {
		t.Error("original should not be affected by clone mutation")
	}
}
