// Package scene handles scene planning and context assembly for screenplay generation.
package scene

import (
	"encoding/json"
	"fmt"

	"github.com/fabula-studio/backend/internal/graph"
)

// ScenePlan defines how a leaf node maps to screenplay scenes.
type ScenePlan struct {
	ID            string   `json:"id"`
	SourceNodeIDs []string `json:"source_node_ids"`
	SceneCount    int      `json:"scene_count"` // 0 = summary only, 1 = single scene, N = multiple
	Purpose       string   `json:"purpose"`
	Location      string   `json:"location"`
	TimeFrame     string   `json:"time_frame"`
	Characters    []string `json:"characters"`
	KeyPlotPoints []string `json:"key_plot_points"`
	OmitDetails   []string `json:"omit_details"`
	Sequence      int      `json:"sequence"`
	SummaryOnly   string   `json:"summary_only,omitempty"`
}

// UnmarshalJSON handles LLM quirks (e.g. omit_details as string instead of array,
// summary_only as a boolean instead of summary text).
func (s *ScenePlan) UnmarshalJSON(data []byte) error {
	type Alias ScenePlan
	aux := &struct {
		OmitDetails interface{} `json:"omit_details"`
		SummaryOnly interface{} `json:"summary_only"`
		*Alias
	}{Alias: (*Alias)(s)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	switch v := aux.OmitDetails.(type) {
	case []interface{}:
		s.OmitDetails = make([]string, 0, len(v))
		for _, item := range v {
			s.OmitDetails = append(s.OmitDetails, fmt.Sprintf("%v", item))
		}
	case string:
		if v != "" {
			s.OmitDetails = []string{v}
		}
	}
	switch v := aux.SummaryOnly.(type) {
	case string:
		s.SummaryOnly = v
	case bool:
		if !v {
			s.SummaryOnly = ""
		}
	default:
		if v != nil {
			s.SummaryOnly = fmt.Sprintf("%v", v)
		}
	}
	return nil
}

// CharacterInfo is a lightweight character reference for the context package.
type CharacterInfo struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	CurrentGoal    string   `json:"current_goal"`
	EmotionalState string   `json:"emotional_state"`
	Personality    []string `json:"personality"`
}

// RelationInfo is a lightweight relationship reference for the context package.
type RelationInfo struct {
	CharA       string `json:"char_a"`
	CharB       string `json:"char_b"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// SceneContext is the bounded context package given to the scene writing agent.
// It contains only information relevant to one scene — no future spoilers.
type SceneContext struct {
	ScenePlan      *ScenePlan      `json:"scene_plan"`
	SourceText     string          `json:"source_text"`
	SourceSummary  string          `json:"source_summary"`
	DramaticGoal   string          `json:"dramatic_goal"`
	Characters     []CharacterInfo `json:"characters"`
	Relations      []RelationInfo  `json:"relations"`
	KnownFacts     []string        `json:"known_facts"`
	PreviousEvents []string        `json:"previous_events"`
	Unresolved     []string        `json:"unresolved"`
	ForbiddenInfo  []string        `json:"forbidden_info"`
}

// ContextBuilder assembles SceneContext packages from scene plans and the dynamic graph.
type ContextBuilder struct {
	snapshotsBefore map[string]*graph.GraphSnapshot
	snapshotsAfter  map[string]*graph.GraphSnapshot
}

// NewContextBuilder creates a builder with access to graph snapshots.
func NewContextBuilder(before, after map[string]*graph.GraphSnapshot) *ContextBuilder {
	return &ContextBuilder{
		snapshotsBefore: before,
		snapshotsAfter:  after,
	}
}

// Build assembles a SceneContext for the given plan.
func (b *ContextBuilder) Build(plan *ScenePlan, sourceText, sourceSummary string) *SceneContext {
	if len(plan.SourceNodeIDs) == 0 {
		return &SceneContext{ScenePlan: plan, SourceText: sourceText, SourceSummary: sourceSummary}
	}
	refNodeID := plan.SourceNodeIDs[0]
	snap := b.snapshotsBefore[refNodeID]
	if snap == nil {
		return &SceneContext{ScenePlan: plan, SourceText: sourceText, SourceSummary: sourceSummary}
	}

	ctx := &SceneContext{
		ScenePlan:     plan,
		SourceText:    sourceText,
		SourceSummary: sourceSummary,
		DramaticGoal:  plan.Purpose,
	}

	// Populate character info
	charSet := make(map[string]bool, len(plan.Characters))
	for _, c := range plan.Characters {
		charSet[c] = true
	}
	for _, cid := range plan.Characters {
		if cs, ok := snap.Characters[cid]; ok {
			ctx.Characters = append(ctx.Characters, CharacterInfo{
				ID:             cs.ID,
				Name:           cs.Name,
				CurrentGoal:    cs.CurrentGoal,
				EmotionalState: cs.EmotionalState,
				Personality:    cs.Personality,
			})
		}
	}

	// Populate relations between present characters
	for _, r := range snap.Relations {
		if charSet[r.CharA] && charSet[r.CharB] {
			ctx.Relations = append(ctx.Relations, RelationInfo{
				CharA: r.CharA, CharB: r.CharB,
				Type: r.Type, Description: r.Description,
			})
		}
	}

	// Populate known facts for present characters
	for _, cid := range plan.Characters {
		if cs, ok := snap.Characters[cid]; ok {
			ctx.KnownFacts = append(ctx.KnownFacts, cs.KnownInfo...)
		}
	}

	// Unresolved conflicts involving present characters
	for _, c := range snap.UnresolvedConflicts {
		if c.Status == "resolved" {
			continue
		}
		for _, inv := range c.Involved {
			if charSet[inv] {
				ctx.Unresolved = append(ctx.Unresolved, c.Description)
				break
			}
		}
	}

	// Forbidden info: gather unknown_info from the AFTER snapshot for present characters.
	// These are facts the characters don't yet know — the writer must not reveal them.
	if afterSnap := b.snapshotsAfter[refNodeID]; afterSnap != nil {
		for _, cid := range plan.Characters {
			if cs, ok := afterSnap.Characters[cid]; ok {
				ctx.ForbiddenInfo = append(ctx.ForbiddenInfo, cs.UnknownInfo...)
			}
		}
	}

	return ctx
}
