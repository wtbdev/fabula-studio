package tool

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"

	"github.com/fabula-studio/backend/internal/graph"
)

// Context key for graph snapshot
type graphSnapshotKey struct{}

// WithGraphSnapshot injects a graph snapshot into the context.
func WithGraphSnapshot(ctx context.Context, snap *graph.GraphSnapshot) context.Context {
	return context.WithValue(ctx, graphSnapshotKey{}, snap)
}

// GraphSnapshotFromContext retrieves the graph snapshot from context.
func GraphSnapshotFromContext(ctx context.Context) *graph.GraphSnapshot {
	snap, _ := ctx.Value(graphSnapshotKey{}).(*graph.GraphSnapshot)
	return snap
}

// ---- Request/Response types ----

type AddCharacterRequest struct {
	ID             string   `json:"id" jsonschema:"description=Character ID (e.g. char_001)"`
	Name           string   `json:"name" jsonschema:"description=Character name"`
	CurrentGoal    string   `json:"current_goal" jsonschema:"description=What this character wants right now"`
	EmotionalState string   `json:"emotional_state" jsonschema:"description=Current emotional state"`
	Personality    []string `json:"personality" jsonschema:"description=Array of personality traits"`
}

type UpdateCharacterRequest struct {
	ID             string   `json:"id" jsonschema:"description=Existing character ID"`
	CurrentGoal    string   `json:"current_goal,omitempty" jsonschema:"description=New goal if changed"`
	EmotionalState string   `json:"emotional_state,omitempty" jsonschema:"description=New emotional state if changed"`
	NewKnownInfo   string   `json:"new_known_info,omitempty" jsonschema:"description=New information the character learned"`
}

type AddRelationRequest struct {
	FromID      string `json:"from_id" jsonschema:"description=First character ID"`
	ToID        string `json:"to_id" jsonschema:"description=Second character ID"`
	Type        string `json:"type" jsonschema:"description=Relationship type (e.g. ally, enemy, family)"`
	Description string `json:"description" jsonschema:"description=Description of the relationship"`
}

type AddConflictRequest struct {
	Description string   `json:"description" jsonschema:"description=What the conflict is"`
	Characters  []string `json:"characters" jsonschema:"description=Array of character IDs involved"`
}

type ResolveConflictRequest struct {
	Index      int    `json:"index" jsonschema:"description=Index of the conflict in unresolved_conflicts array"`
	Resolution string `json:"resolution" jsonschema:"description=How the conflict was resolved"`
}

// ---- Tool constructors ----

// NewGraphTools returns all graph-related tools for the GraphAnalyzer agent.
func NewGraphTools() []tool.Tool {
	return []tool.Tool{
		newAddCharacterTool(),
		newUpdateCharacterTool(),
		newAddRelationTool(),
		newAddConflictTool(),
		newResolveConflictTool(),
	}
}

func newAddCharacterTool() tool.CallableTool {
	return function.NewFunctionTool(
		addCharacter,
		function.WithName("add_character"),
		function.WithDescription("Adds a new character to the graph snapshot"),
	)
}

func newUpdateCharacterTool() tool.CallableTool {
	return function.NewFunctionTool(
		updateCharacter,
		function.WithName("update_character"),
		function.WithDescription("Updates an existing character's state in the graph snapshot"),
	)
}

func newAddRelationTool() tool.CallableTool {
	return function.NewFunctionTool(
		addRelation,
		function.WithName("add_relation"),
		function.WithDescription("Adds or updates a relationship between two characters"),
	)
}

func newAddConflictTool() tool.CallableTool {
	return function.NewFunctionTool(
		addConflict,
		function.WithName("add_conflict"),
		function.WithDescription("Adds a new unresolved conflict to the graph snapshot"),
	)
}

func newResolveConflictTool() tool.CallableTool {
	return function.NewFunctionTool(
		resolveConflict,
		function.WithName("resolve_conflict"),
		function.WithDescription("Marks an existing conflict as resolved"),
	)
}

// ---- Tool implementations ----

func addCharacter(ctx context.Context, req AddCharacterRequest) (map[string]interface{}, error) {
	snap := GraphSnapshotFromContext(ctx)
	if snap == nil {
		return nil, fmt.Errorf("no graph snapshot in context")
	}

	if _, exists := snap.Characters[req.ID]; exists {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("character %s already exists, use update_character instead", req.ID),
		}, nil
	}

	snap.Characters[req.ID] = &graph.CharacterState{
		ID:             req.ID,
		Name:           req.Name,
		CurrentGoal:    req.CurrentGoal,
		EmotionalState: req.EmotionalState,
		Personality:    req.Personality,
	}

	return map[string]interface{}{
		"success":      true,
		"character_id": req.ID,
		"total":        len(snap.Characters),
	}, nil
}

func updateCharacter(ctx context.Context, req UpdateCharacterRequest) (map[string]interface{}, error) {
	snap := GraphSnapshotFromContext(ctx)
	if snap == nil {
		return nil, fmt.Errorf("no graph snapshot in context")
	}

	cs, exists := snap.Characters[req.ID]
	if !exists {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("character %s not found, use add_character first", req.ID),
		}, nil
	}

	if req.CurrentGoal != "" {
		cs.CurrentGoal = req.CurrentGoal
	}
	if req.EmotionalState != "" {
		cs.EmotionalState = req.EmotionalState
	}
	if req.NewKnownInfo != "" {
		cs.KnownInfo = append(cs.KnownInfo, req.NewKnownInfo)
	}

	return map[string]interface{}{
		"success":      true,
		"character_id": req.ID,
	}, nil
}

func addRelation(ctx context.Context, req AddRelationRequest) (map[string]interface{}, error) {
	snap := GraphSnapshotFromContext(ctx)
	if snap == nil {
		return nil, fmt.Errorf("no graph snapshot in context")
	}

	// Check if relation already exists
	for i, r := range snap.Relations {
		if (r.CharA == req.FromID && r.CharB == req.ToID) ||
			(r.CharA == req.ToID && r.CharB == req.FromID) {
			// Update existing relation
			if req.Type != "" {
				snap.Relations[i].Type = req.Type
			}
			if req.Description != "" {
				snap.Relations[i].Description = req.Description
			}
			return map[string]interface{}{
				"success": true,
				"action":  "updated",
				"from":    req.FromID,
				"to":      req.ToID,
			}, nil
		}
	}

	// Add new relation
	snap.Relations = append(snap.Relations, graph.Relation{
		CharA:       req.FromID,
		CharB:       req.ToID,
		Type:        req.Type,
		Description: req.Description,
	})

	return map[string]interface{}{
		"success": true,
		"action":  "created",
		"from":    req.FromID,
		"to":      req.ToID,
		"total":   len(snap.Relations),
	}, nil
}

func addConflict(ctx context.Context, req AddConflictRequest) (map[string]interface{}, error) {
	snap := GraphSnapshotFromContext(ctx)
	if snap == nil {
		return nil, fmt.Errorf("no graph snapshot in context")
	}

	snap.UnresolvedConflicts = append(snap.UnresolvedConflicts, graph.Conflict{
		Description: req.Description,
		Involved:    req.Characters,
		Status:      "unresolved",
	})

	return map[string]interface{}{
		"success": true,
		"index":   len(snap.UnresolvedConflicts) - 1,
		"total":   len(snap.UnresolvedConflicts),
	}, nil
}

func resolveConflict(ctx context.Context, req ResolveConflictRequest) (map[string]interface{}, error) {
	snap := GraphSnapshotFromContext(ctx)
	if snap == nil {
		return nil, fmt.Errorf("no graph snapshot in context")
	}

	if req.Index < 0 || req.Index >= len(snap.UnresolvedConflicts) {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("invalid conflict index %d, have %d conflicts", req.Index, len(snap.UnresolvedConflicts)),
		}, nil
	}

	snap.UnresolvedConflicts[req.Index].Status = "resolved"
	snap.UnresolvedConflicts[req.Index].ResolvedIn = snap.NodeID

	return map[string]interface{}{
		"success": true,
		"index":   req.Index,
	}, nil
}
