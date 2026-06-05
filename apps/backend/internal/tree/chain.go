package tree

// ChainManager provides horizontal chain operations on a story tree.
// Horizontal chains connect same-level nodes in document order.
type ChainManager struct {
	tree *StoryTree
}

// NewChainManager wraps a tree for horizontal chain queries.
func NewChainManager(t *StoryTree) *ChainManager {
	return &ChainManager{tree: t}
}

// GetRightNeighbor returns the right neighbor of a node, or nil.
func (cm *ChainManager) GetRightNeighbor(nodeID string) *StoryNode {
	node := cm.tree.GetNode(nodeID)
	if node == nil || node.RightNeighbor == "" {
		return nil
	}
	return cm.tree.GetNode(node.RightNeighbor)
}

// GetLeftNeighbor finds the node whose RightNeighbor points to nodeID.
func (cm *ChainManager) GetLeftNeighbor(nodeID string) *StoryNode {
	for _, n := range cm.tree.Nodes {
		if n.RightNeighbor == nodeID {
			return n
		}
	}
	return nil
}

// GetSiblingChain returns all nodes at the same level under the same parent,
// in left-to-right order.
func (cm *ChainManager) GetSiblingChain(nodeID string) []*StoryNode {
	node := cm.tree.GetNode(nodeID)
	if node == nil {
		return nil
	}
	parent := cm.tree.GetNode(node.ParentID)
	if parent == nil {
		return nil
	}
	return cm.tree.Children(parent.ID)
}

// ShouldMergeRight checks if a node appears to be cut off and needs
// content from its right neighbor to be semantically complete.
// This is a heuristic check on the text content.
func (cm *ChainManager) ShouldMergeRight(node *StoryNode) bool {
	if node == nil || node.RightNeighbor == "" {
		return false
	}
	text := node.TextContent
	runes := []rune(text)
	if len(runes) == 0 {
		return true
	}
	// Check if the text ends with terminal punctuation first
	lastChars := string(runes[len(runes)-min(3, len(runes)):])
	for _, r := range lastChars {
		switch r {
		case '。', '！', '？', '…', '.', '!', '?', '"', '」', ')', '）':
			return false
		}
	}
	// No terminal punctuation and short → likely truncated
	if len(runes) < 10 {
		return true
	}
	return true // ends without terminal punctuation → likely truncated
}

// MergeNodes combines two adjacent nodes into one.
func (cm *ChainManager) MergeNodes(left, right *StoryNode) *StoryNode {
	merged := &StoryNode{
		ID:            left.ID + "+" + right.ID,
		ParentID:      left.ParentID,
		Level:         left.Level,
		TextContent:   left.TextContent + "\n\n" + right.TextContent,
		SourceChapter: left.SourceChapter,
		RightNeighbor: right.RightNeighbor,
	}
	return merged
}

// BuildLevelGroups groups all nodes by level.
func (cm *ChainManager) BuildLevelGroups() map[int][]*StoryNode {
	groups := make(map[int][]*StoryNode)
	for _, n := range cm.tree.Nodes {
		if n.Level >= 0 { // skip virtual root
			groups[n.Level] = append(groups[n.Level], n)
		}
	}
	return groups
}
