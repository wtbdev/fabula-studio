package pipeline

import (
	"time"

	"github.com/fabula-studio/backend/internal/graph"
	"github.com/fabula-studio/backend/internal/scene"
	"github.com/fabula-studio/backend/internal/schema"
	"github.com/fabula-studio/backend/internal/segment"
)

// RunState holds the intermediate values produced during a single pipeline run.
// It keeps Convert focused on step ordering and error handling rather than a long
// list of unrelated locals.
type RunState struct {
	Title       string
	Author      string
	Chapters    []string
	StartTime   time.Time
	Duration    time.Duration
	CompletedAt time.Time

	SourceIndex     *segment.SourceIndex
	StoryBeats      []segment.StoryBeat
	SceneCandidates []scene.SceneCandidate
	GraphMgr        *graph.Manager
	Plans           []*scene.ScenePlan
	Scenes          []schema.Scene
	Screenplay      *schema.Screenplay
	Artifacts       *schema.GenerationArtifacts
	YAMLStr         string
}

func (s *RunState) finalGraph() *graph.GraphSnapshot {
	if s == nil || s.GraphMgr == nil || len(s.SceneCandidates) == 0 {
		return nil
	}
	return s.GraphMgr.SnapshotsAfter()[s.SceneCandidates[len(s.SceneCandidates)-1].ID]
}

func (s *RunState) result() *PipelineResult {
	return &PipelineResult{
		SourceIndex: s.SourceIndex,
		StoryBeats:  s.StoryBeats,
		GraphMgr:    s.GraphMgr,
		Plans:       s.Plans,
		Artifacts:   s.Artifacts,
		Screenplay:  s.Screenplay,
		YAMLStr:     s.YAMLStr,
		Duration:    s.Duration,
		CompletedAt: s.CompletedAt,
	}
}
