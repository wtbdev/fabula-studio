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

// ApplyUpdate produces the "after" snapshot for a node by applying the update result.
func (m *Manager) ApplyUpdate(nodeID string, update *GraphUpdateResult) *GraphSnapshot {
	before := m.snapshotsBefore[nodeID]
	if before == nil {
		before = NewSnapshot(nodeID)
	}
	after := before.Clone()
	after.NodeID = nodeID

	// Apply new characters
	for _, nc := range update.NewCharacters {
		after.Characters[nc.ID] = &CharacterState{
			ID:             nc.ID,
			Name:           nc.Name,
			CurrentGoal:    nc.CurrentGoal,
			EmotionalState: nc.EmotionalState,
			Personality:    nc.Personality,
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
