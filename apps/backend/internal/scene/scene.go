// Package scene handles scene planning and context assembly for screenplay generation.
package scene

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fabula-studio/backend/internal/graph"
	"github.com/fabula-studio/backend/internal/segment"
)

// ScenePlan defines one screenplay scene backed by source-grounded scene candidates.
type ScenePlan struct {
	ID                 string   `json:"id"`
	SourceCandidateIDs []string `json:"source_candidate_ids"`
	SceneCount         int      `json:"scene_count"` // Always normalized to 1: one plan writes one scene.
	Purpose            string   `json:"purpose"`
	Location           string   `json:"location"`
	TimeFrame          string   `json:"time_frame"`
	Characters         []string `json:"characters"`
	KeyPlotPoints      []string `json:"key_plot_points"`
	OmitDetails        []string `json:"omit_details"`
	Sequence           int      `json:"sequence"`
}

// CandidateIDs returns the candidate IDs this plan covers.
func (s *ScenePlan) CandidateIDs() []string {
	if s == nil {
		return nil
	}
	return s.SourceCandidateIDs
}

// SourceCandidateRefs returns the candidate IDs this plan covers.
func (s *ScenePlan) SourceCandidateRefs() []string {
	return s.CandidateIDs()
}

// SceneCandidate is the deterministic bridge from extracted story beats to LLM scene planning.
// It preserves source sentence grounding so generated scenes can be checked and graph-updated in order.
type SceneCandidate struct {
	ID                string   `json:"id"`
	SourceBeatIDs     []string `json:"source_beat_ids"`
	SourceSentenceIDs []string `json:"source_sentence_ids"`
	Summary           string   `json:"summary"`
	DramaticPurpose   string   `json:"dramatic_purpose"`
	Conflict          string   `json:"conflict"`
	Location          string   `json:"location"`
	TimeFrame         string   `json:"time_frame"`
	Characters        []string `json:"characters"`
	Sequence          int      `json:"sequence"`
}

// BuildSceneCandidates converts repaired story beats into planning candidates.
func BuildSceneCandidates(beats []segment.StoryBeat) []SceneCandidate {
	candidates := make([]SceneCandidate, 0, len(beats))
	for _, beat := range beats {
		candidate := SceneCandidate{
			ID:                fmt.Sprintf("candidate_%03d", len(candidates)+1),
			SourceBeatIDs:     []string{beat.ID},
			SourceSentenceIDs: compactStrings(beat.SourceSentenceIDs),
			Summary:           strings.TrimSpace(beat.Summary),
			DramaticPurpose:   strings.TrimSpace(beat.DramaticPurpose),
			Conflict:          strings.TrimSpace(beat.Conflict),
			Location:          strings.TrimSpace(beat.Location),
			TimeFrame:         strings.TrimSpace(beat.TimeFrame),
			Characters:        compactStrings(beat.Characters),
			Sequence:          beat.Sequence,
		}
		if len(candidate.SourceSentenceIDs) == 0 && beat.StartSentenceID != "" && beat.EndSentenceID != "" {
			candidate.SourceSentenceIDs = []string{beat.StartSentenceID, beat.EndSentenceID}
		}
		candidates = append(candidates, candidate)
	}
	return candidates
}

// ValidateAndRepairScenePlans normalizes scene plans into monotonic, usable generation input.
func ValidateAndRepairScenePlans(plans []*ScenePlan, candidates []SceneCandidate) []*ScenePlan {
	if len(candidates) == 0 {
		return compactPlans(plans)
	}
	byID := make(map[string]SceneCandidate, len(candidates))
	for _, candidate := range candidates {
		byID[candidate.ID] = candidate
	}

	repaired := make([]*ScenePlan, 0, len(plans))
	usedCandidates := make(map[string]struct{}, len(candidates))
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		fixed := repairScenePlan(plan, len(repaired)+1, byID)
		if len(fixed.CandidateIDs()) == 0 {
			continue
		}
		for _, id := range fixed.CandidateIDs() {
			usedCandidates[id] = struct{}{}
		}
		repaired = append(repaired, fixed)
	}

	for _, candidate := range candidates {
		if _, ok := usedCandidates[candidate.ID]; ok {
			continue
		}
		repaired = append(repaired, planFromCandidate(candidate, len(repaired)+1))
	}
	return compactPlans(repaired)
}

func repairScenePlan(plan *ScenePlan, sequence int, candidates map[string]SceneCandidate) *ScenePlan {
	fixed := *plan
	fixed.Sequence = sequence
	fixed.ID = strings.TrimSpace(fixed.ID)
	if fixed.ID == "" {
		fixed.ID = fmt.Sprintf("plan_%03d", sequence)
	}
	fixed.SourceCandidateIDs = compactStrings(fixed.SourceCandidateIDs)
	if len(fixed.SourceCandidateIDs) == 0 && len(candidates) == 1 {
		for id := range candidates {
			fixed.SourceCandidateIDs = []string{id}
		}
	}
	fixed.SceneCount = 1
	fixed.Purpose = strings.TrimSpace(fixed.Purpose)
	fixed.Location = strings.TrimSpace(fixed.Location)
	fixed.TimeFrame = strings.TrimSpace(fixed.TimeFrame)
	fixed.Characters = compactStrings(fixed.Characters)
	fixed.KeyPlotPoints = compactStrings(fixed.KeyPlotPoints)
	fixed.OmitDetails = compactStrings(fixed.OmitDetails)
	for _, sourceID := range fixed.CandidateIDs() {
		candidate, ok := candidates[sourceID]
		if !ok {
			continue
		}
		if fixed.Purpose == "" {
			fixed.Purpose = candidate.DramaticPurpose
		}
		if fixed.Location == "" {
			fixed.Location = candidate.Location
		}
		if fixed.TimeFrame == "" {
			fixed.TimeFrame = candidate.TimeFrame
		}
		fixed.Characters = compactStrings(append(fixed.Characters, candidate.Characters...))
		if len(fixed.KeyPlotPoints) == 0 && candidate.Summary != "" {
			fixed.KeyPlotPoints = []string{candidate.Summary}
		}
	}
	if fixed.Purpose == "" {
		fixed.Purpose = "保留源故事节拍并推进场景冲突"
	}
	return &fixed
}

func planFromCandidate(candidate SceneCandidate, sequence int) *ScenePlan {
	plan := &ScenePlan{
		ID:                 fmt.Sprintf("plan_%03d", sequence),
		SourceCandidateIDs: []string{candidate.ID},
		SceneCount:         1,
		Purpose:            candidate.DramaticPurpose,
		Location:           candidate.Location,
		TimeFrame:          candidate.TimeFrame,
		Characters:         compactStrings(candidate.Characters),
		KeyPlotPoints:      []string{candidate.Summary},
		Sequence:           sequence,
	}
	return repairScenePlan(plan, sequence, map[string]SceneCandidate{candidate.ID: candidate})
}

func compactPlans(plans []*ScenePlan) []*ScenePlan {
	out := make([]*ScenePlan, 0, len(plans))
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		plan.SourceCandidateIDs = compactStrings(plan.SourceCandidateIDs)
		plan.SceneCount = 1
		if strings.TrimSpace(plan.ID) == "" {
			plan.ID = fmt.Sprintf("plan_%03d", len(out)+1)
		}
		plan.Sequence = len(out) + 1
		out = append(out, plan)
	}
	return out
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func normalizeStringSlice(value interface{}) []string {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{v}
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// UnmarshalJSON handles LLM quirks and accepts legacy source_node_ids as input only.
func (s *ScenePlan) UnmarshalJSON(data []byte) error {
	type Alias ScenePlan
	aux := &struct {
		SourceCandidateIDs interface{} `json:"source_candidate_ids"`
		SourceNodeIDs      interface{} `json:"source_node_ids"`
		OmitDetails        interface{} `json:"omit_details"`
		*Alias
	}{Alias: (*Alias)(s)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	s.SceneCount = 1
	s.SourceCandidateIDs = compactStrings(normalizeStringSlice(aux.SourceCandidateIDs))
	if len(s.SourceCandidateIDs) == 0 {
		s.SourceCandidateIDs = compactStrings(normalizeStringSlice(aux.SourceNodeIDs))
	}
	s.OmitDetails = compactStrings(normalizeStringSlice(aux.OmitDetails))
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
	refNodeID := plan.ID
	if snap := b.snapshotsBefore[refNodeID]; snap != nil {
		return b.buildWithSnapshot(plan, sourceText, sourceSummary, refNodeID, snap)
	}
	if len(plan.CandidateIDs()) == 0 {
		return &SceneContext{ScenePlan: plan, SourceText: sourceText, SourceSummary: sourceSummary, DramaticGoal: plan.Purpose}
	}
	refNodeID = plan.CandidateIDs()[0]
	snap := b.snapshotsBefore[refNodeID]
	if snap == nil {
		return &SceneContext{ScenePlan: plan, SourceText: sourceText, SourceSummary: sourceSummary, DramaticGoal: plan.Purpose}
	}
	return b.buildWithSnapshot(plan, sourceText, sourceSummary, refNodeID, snap)
}

func (b *ContextBuilder) buildWithSnapshot(plan *ScenePlan, sourceText, sourceSummary, refNodeID string, snap *graph.GraphSnapshot) *SceneContext {

	ctx := &SceneContext{
		ScenePlan:     plan,
		SourceText:    sourceText,
		SourceSummary: sourceSummary,
		DramaticGoal:  plan.Purpose,
	}

	// Resolve plan character refs against the graph snapshot. Plans may carry either canonical IDs
	// or source names; the graph remains the canonical registry exposed to the writer.
	charSet := make(map[string]bool, len(plan.Characters))
	resolvedCharacterIDs := make([]string, 0, len(plan.Characters))
	for _, ref := range plan.Characters {
		cid := resolveCharacterID(snap, ref)
		if cid == "" || charSet[cid] {
			continue
		}
		charSet[cid] = true
		resolvedCharacterIDs = append(resolvedCharacterIDs, cid)
		cs := snap.Characters[cid]
		ctx.Characters = append(ctx.Characters, CharacterInfo{
			ID:             cs.ID,
			Name:           cs.Name,
			CurrentGoal:    cs.CurrentGoal,
			EmotionalState: cs.EmotionalState,
			Personality:    cs.Personality,
		})
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
	for _, cid := range resolvedCharacterIDs {
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
		for _, cid := range resolvedCharacterIDs {
			if cs, ok := afterSnap.Characters[cid]; ok {
				ctx.ForbiddenInfo = append(ctx.ForbiddenInfo, cs.UnknownInfo...)
			}
		}
	}

	return ctx
}

func resolveCharacterID(snap *graph.GraphSnapshot, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" || snap == nil {
		return ""
	}
	if _, ok := snap.Characters[ref]; ok {
		return ref
	}
	for id, cs := range snap.Characters {
		if cs != nil && strings.TrimSpace(cs.Name) == ref {
			return id
		}
	}
	return ""
}
