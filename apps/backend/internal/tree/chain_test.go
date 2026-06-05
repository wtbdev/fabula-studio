package tree

import "testing"

func buildTestTree() *StoryTree {
	t := NewTree()
	t.RootNodeID = "root"
	t.AddNode(&StoryNode{ID: "root", Level: -1, ChildrenIDs: []string{"a", "b", "c"}})
	t.AddNode(&StoryNode{ID: "a", Level: 0, ParentID: "root", TextContent: "这是完整的句子。", RightNeighbor: "b"})
	t.AddNode(&StoryNode{ID: "b", Level: 0, ParentID: "root", TextContent: "这是被截断的文本", RightNeighbor: "c"})
	t.AddNode(&StoryNode{ID: "c", Level: 0, ParentID: "root", TextContent: "后续内容。"})
	return t
}

func TestGetRightNeighbor(t *testing.T) {
	tree := buildTestTree()
	cm := NewChainManager(tree)

	right := cm.GetRightNeighbor("a")
	if right == nil || right.ID != "b" {
		t.Errorf("expected right neighbor of a to be b, got %v", right)
	}
	right = cm.GetRightNeighbor("c")
	if right != nil {
		t.Errorf("expected no right neighbor for c, got %v", right)
	}
}

func TestGetLeftNeighbor(t *testing.T) {
	tree := buildTestTree()
	cm := NewChainManager(tree)

	left := cm.GetLeftNeighbor("b")
	if left == nil || left.ID != "a" {
		t.Errorf("expected left neighbor of b to be a, got %v", left)
	}
	left = cm.GetLeftNeighbor("a")
	if left != nil {
		t.Errorf("expected no left neighbor for a, got %v", left)
	}
}

func TestShouldMergeRight(t *testing.T) {
	tree := buildTestTree()
	cm := NewChainManager(tree)

	// "这是完整的句子。" ends with 。→ should NOT merge
	if cm.ShouldMergeRight(tree.GetNode("a")) {
		t.Error("node a should not merge right (ends with 。)")
	}
	// "这是被截断的文本" no terminal punctuation → should merge
	if !cm.ShouldMergeRight(tree.GetNode("b")) {
		t.Error("node b should merge right (no terminal punctuation)")
	}
	// node c has no right neighbor → should NOT merge
	if cm.ShouldMergeRight(tree.GetNode("c")) {
		t.Error("node c should not merge right (no neighbor)")
	}
}

func TestMergeNodes(t *testing.T) {
	tree := buildTestTree()
	cm := NewChainManager(tree)

	a := tree.GetNode("a")
	b := tree.GetNode("b")
	merged := cm.MergeNodes(a, b)

	if merged.TextContent != "这是完整的句子。\n\n这是被截断的文本" {
		t.Errorf("unexpected merged text: %q", merged.TextContent)
	}
	if merged.RightNeighbor != "c" {
		t.Errorf("expected right neighbor c, got %q", merged.RightNeighbor)
	}
}

func TestBuildLevelGroups(t *testing.T) {
	tree := buildTestTree()
	cm := NewChainManager(tree)

	groups := cm.BuildLevelGroups()
	if len(groups[0]) != 3 {
		t.Errorf("expected 3 level-0 nodes, got %d", len(groups[0]))
	}
}
