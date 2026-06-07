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
	"github.com/fabula-studio/backend/internal/schema"
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

func TestGenerateScenesRunsConcurrently(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var mu sync.Mutex
	active := 0
	maxActive := 0

	p := &Pipeline{
		config: Config{MaxConcurrency: 2},
		tracer: otel.Tracer("test"),
		sceneWriter: &agent.SceneWriterAgent{
			CustomWriteScene: func(ctx context.Context, sceneCtx *scene.SceneContext) (*schema.Scene, error) {
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
				return &schema.Scene{
					ID:      sceneCtx.ScenePlan.ID,
					Heading: sceneCtx.ScenePlan.Location,
				}, nil
			},
		},
	}
	plans := []*scene.ScenePlan{
		{ID: "plan_001", SceneCount: 1, Location: "内景 客厅 - 日"},
		{ID: "plan_002", SceneCount: 1, Location: "外景 街道 - 夜"},
	}

	done := make(chan struct{})
	var scenes []schema.Scene
	var err error
	go func() {
		scenes, err = p.generateScenesSequential(context.Background(), plans, nil, nil, graph.NewManager())
		close(done)
	}()

	<-started
	<-started
	mu.Lock()
	observedMax := maxActive
	mu.Unlock()
	if observedMax != 2 {
		t.Fatalf("expected concurrent scene writing, got max active %d", observedMax)
	}
	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("scene generation did not finish")
	}
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scenes) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(scenes))
	}
	if scenes[0].Sequence != 1 || scenes[1].Sequence != 2 {
		t.Fatalf("expected sequence 1,2 got %d,%d", scenes[0].Sequence, scenes[1].Sequence)
	}
	if scenes[0].Heading != "内景 客厅 - 日" || scenes[1].Heading != "外景 街道 - 夜" {
		t.Fatalf("unexpected scene order: %s, %s", scenes[0].Heading, scenes[1].Heading)
	}
}
func TestClusterSceneCandidates(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		if c := clusterSceneCandidates(nil); c != nil {
			t.Fatalf("expected nil, got %#v", c)
		}
	})
	t.Run("single", func(t *testing.T) {
		c := clusterSceneCandidates([]scene.SceneCandidate{{ID: "c1", Location: "客厅", TimeFrame: "日"}})
		if len(c) != 1 || len(c[0]) != 1 || c[0][0].ID != "c1" {
			t.Fatalf("unexpected clusters: %#v", c)
		}
	})
	t.Run("same_location_time", func(t *testing.T) {
		c := clusterSceneCandidates([]scene.SceneCandidate{
			{ID: "c1", Location: "客厅", TimeFrame: "日"},
			{ID: "c2", Location: "客厅", TimeFrame: "日"},
		})
		if len(c) != 1 || len(c[0]) != 2 {
			t.Fatalf("expected one cluster of 2, got %d clusters", len(c))
		}
	})
	t.Run("different_locations", func(t *testing.T) {
		c := clusterSceneCandidates([]scene.SceneCandidate{
			{ID: "c1", Location: "客厅", TimeFrame: "日"},
			{ID: "c2", Location: "街道", TimeFrame: "日"},
		})
		if len(c) != 2 || len(c[0]) != 1 || len(c[1]) != 1 {
		t.Fatalf("expected 2 clusters of 1, got %d", len(c))
		}
	})
	t.Run("same_location_different_time", func(t *testing.T) {
		c := clusterSceneCandidates([]scene.SceneCandidate{
			{ID: "c1", Location: "客厅", TimeFrame: "日"},
			{ID: "c2", Location: "客厅", TimeFrame: "夜"},
		})
		if len(c) != 2 {
			t.Fatalf("expected 2 clusters, got %d", len(c))
		}
	})
	t.Run("mixed_sequence", func(t *testing.T) {
		c := clusterSceneCandidates([]scene.SceneCandidate{
			{ID: "c1", Location: "客厅", TimeFrame: "日"},
			{ID: "c2", Location: "客厅", TimeFrame: "日"},
			{ID: "c3", Location: "街道", TimeFrame: "日"},
			{ID: "c4", Location: "街道", TimeFrame: "夜"},
			{ID: "c5", Location: "客厅", TimeFrame: "日"},
		})
		if len(c) != 4 {
			t.Fatalf("expected 4 clusters, got %d", len(c))
		}
		if len(c[0]) != 2 || c[0][0].ID != "c1" {
			t.Fatalf("cluster 0 unexpected: %#v", c[0])
		}
		if len(c[1]) != 1 || c[1][0].ID != "c3" {
			t.Fatalf("cluster 1 unexpected: %#v", c[1])
		}
		if len(c[2]) != 1 || c[2][0].ID != "c4" {
			t.Fatalf("cluster 2 unexpected: %#v", c[2])
		}
		if len(c[3]) != 1 || c[3][0].ID != "c5" {
			t.Fatalf("cluster 3 unexpected: %#v", c[3])
		}
	})
}

func TestPlanFromCandidatesRunsConcurrentlyPerCluster(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var mu sync.Mutex
	active := 0
	maxActive := 0

	p := &Pipeline{
		config: Config{MaxConcurrency: 2},
		tracer: otel.Tracer("test"),
		scenePlanner: &agent.ScenePlannerAgent{
			CustomPlanFromCandidates: func(ctx context.Context, candidates []scene.SceneCandidate) ([]*scene.ScenePlan, error) {
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
				plans := make([]*scene.ScenePlan, len(candidates))
				for i, c := range candidates {
					plans[i] = &scene.ScenePlan{
						ID:                 "plan_" + c.ID,
						SourceCandidateIDs: []string{c.ID},
						SceneCount:         1,
						Location:           c.Location,
					}
				}
				return plans, nil
			},
		},
	}

	// Two clusters: same location+time → one cluster, different → second cluster
	candidates := []scene.SceneCandidate{
		{ID: "c001", Location: "客厅", TimeFrame: "日"},
		{ID: "c002", Location: "客厅", TimeFrame: "日"},
		{ID: "c003", Location: "街道", TimeFrame: "夜"},
		{ID: "c004", Location: "街道", TimeFrame: "夜"},
	}
	clusters := clusterSceneCandidates(candidates)
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}

	done := make(chan struct{})
	var allPlans []*scene.ScenePlan
	go func() {
		allPlans = scene.ValidateAndRepairScenePlans(planFromClustersConcurrent(context.Background(), p, clusters), candidates)
		close(done)
	}()

	<-started
	<-started
	mu.Lock()
	observedMax := maxActive
	mu.Unlock()
	if observedMax != 2 {
		t.Fatalf("expected concurrent cluster planning, got max active %d", observedMax)
	}
	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cluster planning did not finish")
	}
	if len(allPlans) != 4 {
		t.Fatalf("expected 4 plans, got %d", len(allPlans))
	}
	// Verify plans in order: first cluster's plans, then second cluster's plans
	if allPlans[0].SourceCandidateIDs[0] != "c001" || allPlans[1].SourceCandidateIDs[0] != "c002" {
		t.Fatalf("first cluster plans out of order: %#v", allPlans[:2])
	}
	if allPlans[2].SourceCandidateIDs[0] != "c003" || allPlans[3].SourceCandidateIDs[0] != "c004" {
		t.Fatalf("second cluster plans out of order: %#v", allPlans[2:])
	}
}

func planFromClustersConcurrent(ctx context.Context, p *Pipeline, clusters [][]scene.SceneCandidate) []*scene.ScenePlan {
	type clusterPlans struct {
		index int
		plans []*scene.ScenePlan
		err   error
	}
	results := make([]clusterPlans, len(clusters))
	concurrency := p.config.MaxConcurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(clusters) {
		concurrency = len(clusters)
	}
	jobIdx := make(chan int, len(clusters))
	var wg sync.WaitGroup
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobIdx {
				plans, err := p.scenePlanner.PlanFromCandidates(ctx, clusters[i])
				results[i] = clusterPlans{index: i, plans: plans, err: err}
			}
		}()
	}
	for i := range clusters {
		jobIdx <- i
	}
	close(jobIdx)
	wg.Wait()

	allPlans := make([]*scene.ScenePlan, 0, len(clusters)*2)
	for _, r := range results {
		if r.err != nil {
			panic(r.err)
		}
		allPlans = append(allPlans, r.plans...)
	}
	return allPlans
}
