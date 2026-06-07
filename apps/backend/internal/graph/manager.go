package graph

// Manager applies GraphUpdateResults to produce new snapshots.
type Manager struct {
	snapshotsBefore map[string]*GraphSnapshot // node_id → snapshot before processing
	snapshotsAfter  map[string]*GraphSnapshot // node_id → snapshot after processing
}

// NewManager creates a new graph manager.
func NewManager() *Manager {
	return &Manager{
		snapshotsBefore: make(map[string]*GraphSnapshot),
		snapshotsAfter:  make(map[string]*GraphSnapshot),
	}
}

// SetInitialSnapshot sets the "before" snapshot for the first node.
func (m *Manager) SetInitialSnapshot(nodeID string, snap *GraphSnapshot) {
	m.snapshotsBefore[nodeID] = snap
}

// SetAfterSnapshot stores the graph snapshot after a node, plan, or generated scene boundary.
func (m *Manager) SetAfterSnapshot(nodeID string, snap *GraphSnapshot) {
	if snap == nil {
		return
	}
	stored := snap.Clone()
	stored.NodeID = nodeID
	m.snapshotsAfter[nodeID] = stored
}

// ApplyUpdate produces the "after" snapshot for a node by applying the update result.
func (m *Manager) ApplyUpdate(nodeID string, update *GraphUpdateResult) *GraphSnapshot {
	before := m.snapshotsBefore[nodeID]
	if before == nil {
		before = NewSnapshot(nodeID)
	}
	if update == nil {
		m.snapshotsAfter[nodeID] = before.Clone()
		m.snapshotsAfter[nodeID].NodeID = nodeID
		return m.snapshotsAfter[nodeID]
	}
	after := before.Clone()
	after.NodeID = nodeID

	// Apply new characters
	for _, nc := range update.NewCharacters {
		if existing := after.GetCharacter(nc.ID); existing != nil && existing.Name != "" {
			mergeCharacterState(existing, nc)
			continue
		}
		after.Characters[nc.ID] = &CharacterState{
			ID:             nc.ID,
			Name:           nc.Name,
			CurrentGoal:    nc.CurrentGoal,
			EmotionalState: nc.EmotionalState,
			Personality:    append([]string(nil), nc.Personality...),
			KnownInfo:      append([]string(nil), nc.KnownInfo...),
			UnknownInfo:    append([]string(nil), nc.UnknownInfo...),
		}
	}

	// Apply character updates
	for _, uc := range update.UpdatedCharacters {
		cs := after.GetCharacter(uc.ID)
		if uc.GoalChange != "" {
			cs.CurrentGoal = uc.GoalChange
		}
		if uc.EmotionChange != "" {
			cs.EmotionalState = uc.EmotionChange
		}
		if uc.NewKnownInfo != "" {
			cs.KnownInfo = append(cs.KnownInfo, uc.NewKnownInfo)
		}
	}

	// Apply relation changes
	for _, rc := range update.RelationChanges {
		found := false
		for i, r := range after.Relations {
			if (r.CharA == rc.CharA && r.CharB == rc.CharB) || (r.CharA == rc.CharB && r.CharB == rc.CharA) {
				if rc.NewType != "" {
					after.Relations[i].Type = rc.NewType
				}
				if rc.NewDescription != "" {
					after.Relations[i].Description = rc.NewDescription
				}
				after.Relations[i].ChangedBy = nodeID
				found = true
				break
			}
		}
		if !found {
			after.Relations = append(after.Relations, Relation{
				CharA: rc.CharA, CharB: rc.CharB,
				Type: rc.NewType, Description: rc.NewDescription,
				ChangedBy: nodeID,
			})
		}
	}

	// Apply new conflicts
	for _, nc := range update.NewConflicts {
		after.UnresolvedConflicts = append(after.UnresolvedConflicts, nc)
	}

	// Resolve conflicts
	for _, desc := range update.ResolvedConflicts {
		for i, c := range after.UnresolvedConflicts {
			if c.Description == desc {
				after.UnresolvedConflicts[i].Status = "resolved"
				after.UnresolvedConflicts[i].ResolvedIn = nodeID
				break
			}
		}
	}

	// Apply new foreshadows
	for _, fs := range update.NewForeshadows {
		fs.IntroducedIn = nodeID
		after.Foreshadows = append(after.Foreshadows, fs)
	}

	m.snapshotsAfter[nodeID] = after
	return after
}


func mergeCharacterState(existing *CharacterState, incoming CharacterState) {
	if existing.Name == "" {
		existing.Name = incoming.Name
	}
	if incoming.CurrentGoal != "" {
		existing.CurrentGoal = incoming.CurrentGoal
	}
	if incoming.EmotionalState != "" {
		existing.EmotionalState = incoming.EmotionalState
	}
	if len(incoming.Personality) > 0 {
		existing.Personality = append([]string(nil), incoming.Personality...)
	}
	if len(incoming.KnownInfo) > 0 {
		existing.KnownInfo = append(existing.KnownInfo, incoming.KnownInfo...)
	}
	if len(incoming.UnknownInfo) > 0 {
		existing.UnknownInfo = append(existing.UnknownInfo, incoming.UnknownInfo...)
	}
}
// ChainSnapshot sets the "before" snapshot of nextNodeID to the "after" snapshot of currentNodeID.
func (m *Manager) ChainSnapshot(currentNodeID, nextNodeID string) {
	if after := m.snapshotsAfter[currentNodeID]; after != nil {
		m.snapshotsBefore[nextNodeID] = after.Clone()
		m.snapshotsBefore[nextNodeID].NodeID = nextNodeID
	}
}

// SnapshotsBefore returns all before-snapshots.
func (m *Manager) SnapshotsBefore() map[string]*GraphSnapshot {
	return m.snapshotsBefore
}

// SnapshotsAfter returns all after-snapshots.
func (m *Manager) SnapshotsAfter() map[string]*GraphSnapshot {
	return m.snapshotsAfter
}
