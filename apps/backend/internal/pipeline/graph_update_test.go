package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"
	"go.opentelemetry.io/otel"

	"github.com/fabula-studio/backend/internal/agent"
	"github.com/fabula-studio/backend/internal/graph"
	"github.com/fabula-studio/backend/internal/scene"
	"github.com/fabula-studio/backend/internal/segment"
)

func TestUpdateGraphFromCandidatesExtractsConcurrentlyAppliesSequentially(t *testing.T) {
	idx := segment.NewSourceIndex([]segment.Sentence{
		{ID: "sent_000001", Chapter: 1, Index: 1, ChapterIndex: 1, Text: "Ada arrives."},
		{ID: "sent_000002", Chapter: 1, Index: 2, ChapterIndex: 2, Text: "Ben helps."},
	})

	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var mu sync.Mutex
	active := 0
	maxActive := 0

	p := &Pipeline{
		config: Config{MaxConcurrency: 2},
		tracer: otel.Tracer("test"),
		graphAnalyzer: &agent.GraphAnalyzerAgent{
			CustomExtractUpdateInstructions: func(ctx context.Context, nodeText string) (*graph.GraphUpdateResult, error) {
				mu.Lock()
				active++
				if active > maxActive {
					maxActive = active
				}
				mu.Unlock()
				started <- struct{}{}
				<-release
				mu.Lock()
				active--
				mu.Unlock()

				switch nodeText {
				case "Ada arrives.":
					return &graph.GraphUpdateResult{NewCharacters: []graph.CharacterState{{ID: "char_001", Name: "Ada"}}}, nil
				case "Ben helps.":
					return &graph.GraphUpdateResult{NewCharacters: []graph.CharacterState{{ID: "char_002", Name: "Ben"}}}, nil
				default:
					t.Fatalf("unexpected graph text %q", nodeText)
					return nil, nil
				}
			},
		},
	}
	candidates := []scene.SceneCandidate{
		{ID: "candidate_001", SourceSentenceIDs: []string{"sent_000001"}},
		{ID: "candidate_002", SourceSentenceIDs: []string{"sent_000002"}},
	}

	done := make(chan struct{})
	var mgr *graph.Manager
	var err error
	go func() {
		mgr, err = p.updateGraphFromCandidates(context.Background(), idx, candidates)
		close(done)
	}()

	<-started
	<-started
	mu.Lock()
	observedMax := maxActive
	mu.Unlock()
	if observedMax != 2 {
		t.Fatalf("expected concurrent extraction, got max active %d", observedMax)
	}
	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("graph update did not finish")
	}
	if err != nil {
		t.Fatalf("unexpected graph update error: %v", err)
	}
	firstAfter := mgr.SnapshotsAfter()["candidate_001"]
	if _, exists := firstAfter.Characters["char_002"]; exists {
		t.Fatalf("candidate_001 snapshot should not include later candidate character: %#v", firstAfter.Characters)
	}
	secondBefore := mgr.SnapshotsBefore()["candidate_002"]
	if _, exists := secondBefore.Characters["char_001"]; !exists {
		t.Fatalf("candidate_002 before snapshot should include candidate_001 character: %#v", secondBefore.Characters)
	}
	secondAfter := mgr.SnapshotsAfter()["candidate_002"]
	if len(secondAfter.Characters) != 2 {
		t.Fatalf("expected final sequential snapshot to include two characters, got %#v", secondAfter.Characters)
	}
}
