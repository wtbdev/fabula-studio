// Package graph implements the dynamic character relationship graph.
package graph

// CharacterState tracks a character's current status at a point in the story.
type CharacterState struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	CurrentGoal    string   `json:"current_goal"`
	EmotionalState string   `json:"emotional_state"`
	Personality    []string `json:"personality"`
	KnownInfo      []string `json:"known_info"`
	UnknownInfo    []string `json:"unknown_info"`
}

// Relation describes a relationship between two characters.
type Relation struct {
	CharA       string `json:"char_a"`
	CharB       string `json:"char_b"`
	Type        string `json:"type"`
	Description string `json:"description"`
	ChangedBy   string `json:"changed_by"`
}

// Conflict is an unresolved tension in the story.
type Conflict struct {
	Description string   `json:"description"`
	Involved    []string `json:"involved"`
	Status      string   `json:"status"` // unresolved, resolved
	ResolvedIn  string   `json:"resolved_in,omitempty"`
}

// Foreshadow tracks a planted setup awaiting payoff.
type Foreshadow struct {
	Description  string `json:"description"`
	IntroducedIn string `json:"introduced_in"`
	ResolvedIn   string `json:"resolved_in,omitempty"`
}

// GraphSnapshot is the full story state at a specific node boundary.
type GraphSnapshot struct {
	NodeID              string                     `json:"node_id"`
	Characters          map[string]*CharacterState `json:"characters"`
	Relations           []Relation                 `json:"relations"`
	UnresolvedConflicts []Conflict                 `json:"unresolved_conflicts"`
	Foreshadows         []Foreshadow               `json:"foreshadows"`
}

// NewSnapshot creates an empty snapshot for the given node.
func NewSnapshot(nodeID string) *GraphSnapshot {
	return &GraphSnapshot{
		NodeID:     nodeID,
		Characters: make(map[string]*CharacterState),
	}
}

// Clone creates a deep copy of the snapshot.
func (s *GraphSnapshot) Clone() *GraphSnapshot {
	c := &GraphSnapshot{
		NodeID:     s.NodeID,
		Characters: make(map[string]*CharacterState, len(s.Characters)),
	}
	for k, v := range s.Characters {
		cc := *v
		cc.KnownInfo = append([]string(nil), v.KnownInfo...)
		cc.UnknownInfo = append([]string(nil), v.UnknownInfo...)
		cc.Personality = append([]string(nil), v.Personality...)
		c.Characters[k] = &cc
	}
	c.Relations = append([]Relation(nil), s.Relations...)
	c.UnresolvedConflicts = append([]Conflict(nil), s.UnresolvedConflicts...)
	c.Foreshadows = append([]Foreshadow(nil), s.Foreshadows...)
	return c
}

// GetCharacter returns a character state, creating a minimal one if absent.
func (s *GraphSnapshot) GetCharacter(id string) *CharacterState {
	if cs, ok := s.Characters[id]; ok {
		return cs
	}
	cs := &CharacterState{ID: id}
	s.Characters[id] = cs
	return cs
}

// GraphUpdateResult is the structured output from the graph analysis agent.
type GraphUpdateResult struct {
	NewCharacters      []CharacterState   `json:"new_characters"`
	UpdatedCharacters  []CharacterUpdate  `json:"updated_characters"`
	RelationChanges    []RelationChange   `json:"relation_changes"`
	NewConflicts       []Conflict         `json:"new_conflicts"`
	ResolvedConflicts  []string           `json:"resolved_conflicts"`
	NewForeshadows     []Foreshadow       `json:"new_foreshadows"`
}

// CharacterUpdate describes changes to an existing character.
type CharacterUpdate struct {
	ID            string `json:"id"`
	GoalChange    string `json:"goal_change,omitempty"`
	EmotionChange string `json:"emotion_change,omitempty"`
	NewKnownInfo  string `json:"new_known_info,omitempty"`
}

// RelationChange describes a change in the relationship between two characters.
type RelationChange struct {
	CharA          string `json:"char_a"`
	CharB          string `json:"char_b"`
	NewType        string `json:"new_type,omitempty"`
	NewDescription string `json:"new_description,omitempty"`
	TriggerEvent   string `json:"trigger_event"`
}
