// Package tree implements the recursive story tree for novel decomposition.
package tree

// NodeDecision represents the processing decision for a story node.
type NodeDecision string

const (
	DecisionKeep          NodeDecision = "keep"
	DecisionSplit         NodeDecision = "split"
	DecisionMergeRight    NodeDecision = "merge_right"
	DecisionSummarizeOnly NodeDecision = "summarize_only"
	DecisionDiscard       NodeDecision = "discard"
)

// Event represents a discrete event within a story node.
type Event struct {
	Description string   `json:"description"`
	Characters  []string `json:"characters"`
	Impact      string   `json:"impact"`
}

// StoryNode is a single unit in the recursive story tree.
// It holds both raw text and AI-analyzed structural information.
type StoryNode struct {
	ID            string       `json:"id"`
	ParentID      string       `json:"parent_id"`
	Level         int          `json:"level"`
	TextContent   string       `json:"text_content"`
	CharStart     int          `json:"char_start"`
	CharEnd       int          `json:"char_end"`
	SourceChapter int          `json:"source_chapter"`
	ChildrenIDs   []string     `json:"children_ids"`
	RightNeighbor string       `json:"right_neighbor"`

	// AI analysis results
	Summary      string   `json:"summary"`
	MainConflict string   `json:"main_conflict"`
	Characters   []string `json:"characters"`
	Events       []Event  `json:"events"`
	Location     string   `json:"location"`
	TimeFrame    string   `json:"time_frame"`
	IsComplete   bool     `json:"is_complete"`

	// Processing decision
	Decision     NodeDecision `json:"decision"`
	SplitReason  string       `json:"split_reason,omitempty"`
}

// StoryTree is the full recursive decomposition of a novel.
type StoryTree struct {
	RootNodeID  string                `json:"root_node_id"`
	Nodes       map[string]*StoryNode `json:"nodes"`
	LeafNodeIDs []string              `json:"leaf_node_ids"`
	MaxDepth    int                   `json:"max_depth"`
}

// NewTree creates an empty story tree.
func NewTree() *StoryTree {
	return &StoryTree{
		Nodes: make(map[string]*StoryNode),
	}
}

// GetNode returns a node by ID, or nil if not found.
func (t *StoryTree) GetNode(id string) *StoryNode {
	return t.Nodes[id]
}

// AddNode inserts a node into the tree.
func (t *StoryTree) AddNode(node *StoryNode) {
	t.Nodes[node.ID] = node
	if node.Level > t.MaxDepth {
		t.MaxDepth = node.Level
	}
}

// Children returns the child nodes of the given node.
func (t *StoryTree) Children(nodeID string) []*StoryNode {
	node := t.Nodes[nodeID]
	if node == nil {
		return nil
	}
	children := make([]*StoryNode, 0, len(node.ChildrenIDs))
	for _, cid := range node.ChildrenIDs {
		if child := t.Nodes[cid]; child != nil {
			children = append(children, child)
		}
	}
	return children
}

// CollectLeaves performs a DFS to collect all leaf nodes in order.
func (t *StoryTree) CollectLeaves() []*StoryNode {
	if t.RootNodeID == "" {
		return nil
	}
	var leaves []*StoryNode
	var walk func(id string)
	walk = func(id string) {
		node := t.Nodes[id]
		if node == nil {
			return
		}
		if len(node.ChildrenIDs) == 0 {
			leaves = append(leaves, node)
			return
		}
		for _, cid := range node.ChildrenIDs {
			walk(cid)
		}
	}
	walk(t.RootNodeID)
	return leaves
}

// UpdateLeafIDs recalculates and stores the leaf node ID list.
func (t *StoryTree) UpdateLeafIDs() {
	leaves := t.CollectLeaves()
	t.LeafNodeIDs = make([]string, len(leaves))
	for i, l := range leaves {
		t.LeafNodeIDs[i] = l.ID
	}
}
