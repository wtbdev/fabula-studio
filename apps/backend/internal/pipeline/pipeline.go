package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fabula-studio/backend/internal/agent"
	"github.com/fabula-studio/backend/internal/graph"
	"github.com/fabula-studio/backend/internal/schema"
	"github.com/fabula-studio/backend/internal/scene"
	"github.com/fabula-studio/backend/internal/tree"
	"github.com/fabula-studio/backend/internal/validator"
)

// Pipeline orchestrates the full novel-to-screenplay conversion.
type Pipeline struct {
	config         Config
	nodeAnalyzer   *agent.NodeAnalyzerAgent
	graphAnalyzer  *agent.GraphAnalyzerAgent
	scenePlanner   *agent.ScenePlannerAgent
	sceneWriter    *agent.SceneWriterAgent
	chiefEditor    *agent.ChiefEditorAgent
	validator      *validator.Validator
}

// New creates a Pipeline with the given config and agent configuration.
func New(cfg Config, modelName, apiKey, baseURL string) *Pipeline {
	return &Pipeline{
		config:         cfg,
		nodeAnalyzer:   agent.NewNodeAnalyzerAgent(modelName, apiKey, baseURL),
		graphAnalyzer:  agent.NewGraphAnalyzerAgent(modelName, apiKey, baseURL),
		scenePlanner:   agent.NewScenePlannerAgent(modelName, apiKey, baseURL),
		sceneWriter:    agent.NewSceneWriterAgent(modelName, apiKey, baseURL),
		chiefEditor:    agent.NewChiefEditorAgent(modelName, apiKey, baseURL),
		validator:      &validator.Validator{},
	}
}

// Convert executes the full conversion pipeline.
func (p *Pipeline) Convert(ctx context.Context, title, author string, chapters []string) (*schema.Screenplay, error) {
	// Step 1-3: Build story tree with hard cuts and horizontal chains
	fmt.Println("[Pipeline] Step 1: Building story tree...")
	st := p.buildTree(chapters)
	fmt.Printf("[Pipeline] Story tree built: %d nodes, %d leaves\n", len(st.Nodes), len(st.LeafNodeIDs))

	// Step 4-6: Analyze nodes recursively
	fmt.Println("[Pipeline] Step 2: Analyzing nodes...")
	if err := p.analyzeNodes(ctx, st, 0); err != nil {
		return nil, fmt.Errorf("node analysis failed: %w", err)
	}
	st.UpdateLeafIDs()
	fmt.Printf("[Pipeline] Analysis complete: %d leaf nodes\n", len(st.LeafNodeIDs))

	// Step 7: Update dynamic graph
	fmt.Println("[Pipeline] Step 3: Updating dynamic graph...")
	graphMgr, err := p.updateGraph(ctx, st)
	if err != nil {
		return nil, fmt.Errorf("graph update failed: %w", err)
	}

	// Step 8: Plan scenes
	fmt.Println("[Pipeline] Step 4: Planning scenes...")
	leaves := make([]*tree.StoryNode, 0, len(st.LeafNodeIDs))
	for _, lid := range st.LeafNodeIDs {
		if n := st.GetNode(lid); n != nil {
			leaves = append(leaves, n)
		}
	}
	plans, err := p.scenePlanner.PlanScenes(ctx, leaves)
	if err != nil {
		return nil, fmt.Errorf("scene planning failed: %w", err)
	}
	fmt.Printf("[Pipeline] %d scene plans created\n", len(plans))

	// Step 9-10: Generate scenes
	fmt.Println("[Pipeline] Step 5: Generating scenes...")
	scenes, err := p.generateScenes(ctx, plans, st, graphMgr)
	if err != nil {
		return nil, fmt.Errorf("scene generation failed: %w", err)
	}

	// Assemble screenplay
	genre := []string{}
	if len(leaves) > 0 && leaves[0].Summary != "" {
		genre = []string{"剧情"}
	}
	sourceChapters := make([]int, len(chapters))
	for i := range chapters {
		sourceChapters[i] = i + 1
	}

	screenplay := &schema.Screenplay{
		Metadata: schema.Metadata{
			Title:          title,
			Author:         author,
			Version:        "1.0",
			CreatedAt:      time.Now().Format(time.RFC3339),
			OriginalNovel:  title,
			Genre:          genre,
			SourceChapters: sourceChapters,
		},
		Characters: p.collectCharacters(leaves),
		Scenes:     scenes,
	}

	// Step 12: Chief editor review
	fmt.Println("[Pipeline] Step 6: Chief editor review...")
	editResult, err := p.chiefEditor.ReviewAndRevise(ctx, screenplay)
	if err != nil {
		fmt.Printf("[Pipeline] Editor review failed (non-fatal): %v\n", err)
	} else {
		if editResult.Screenplay != nil {
			screenplay = editResult.Screenplay
		}
		if len(editResult.Issues) > 0 {
			fmt.Printf("[Pipeline] Editor found %d issues\n", len(editResult.Issues))
		}
	}

	// Step 13: Validation
	fmt.Println("[Pipeline] Step 7: Validating...")
	result := p.validator.Validate(screenplay, 3)
	if !result.Valid {
		return nil, fmt.Errorf("validation failed: %s", strings.Join(result.Errors, "; "))
	}
	if len(result.Warnings) > 0 {
		fmt.Printf("[Pipeline] Validation warnings: %v\n", result.Warnings)
	}

	fmt.Println("[Pipeline] Done!")
	return screenplay, nil
}

// buildTree creates the story tree from chapters (Steps 1-3).
func (p *Pipeline) buildTree(chapters []string) *tree.StoryTree {
	splitter := tree.NewSplitter(p.config.MaxChunkSize)
	return splitter.SplitChapters(chapters)
}

// analyzeNodes recursively analyzes and refines the story tree (Steps 4-6).
func (p *Pipeline) analyzeNodes(ctx context.Context, st *tree.StoryTree, depth int) error {
	if depth >= p.config.MaxRecursionDepth {
		fmt.Printf("[Pipeline] Max recursion depth %d reached\n", p.config.MaxRecursionDepth)
		return nil
	}

	cm := tree.NewChainManager(st)
	nodesToSplit := make([]*tree.StoryNode, 0)

	// Find leaf nodes that haven't been analyzed yet
	for _, node := range st.Nodes {
		if node.Level < 0 { // skip root
			continue
		}
		if len(node.ChildrenIDs) > 0 { // already split
			continue
		}
		if node.Summary != "" { // already analyzed
			continue
		}

		// Check if we should merge right first
		if cm.ShouldMergeRight(node) {
			right := cm.GetRightNeighbor(node.ID)
			if right != nil && right.Summary == "" {
				// Merge the nodes
				merged := cm.MergeNodes(node, right)
				node.TextContent = merged.TextContent
				node.RightNeighbor = merged.RightNeighbor
				fmt.Printf("[Pipeline] Merged %s + %s (truncation detected)\n", node.ID, right.ID)
			}
		}

		// Analyze the node
		leftSummary := ""
		left := cm.GetLeftNeighbor(node.ID)
		if left != nil {
			leftSummary = left.Summary
		}

		fmt.Printf("[Pipeline] Analyzing node %s (depth=%d)...\n", node.ID, depth)
		result, err := p.retryAnalyze(ctx, node, leftSummary)
		if err != nil {
			fmt.Printf("[Pipeline] Node %s analysis failed: %v — treating as summarize_only\n", node.ID, err)
			node.Decision = tree.DecisionSummarizeOnly
			continue
		}

		// Apply analysis results
		node.Summary = result.Summary
		node.MainConflict = result.MainConflict
		node.Characters = result.Characters
		node.Events = result.Events
		node.Location = result.Location
		node.TimeFrame = result.TimeFrame
		node.IsComplete = result.IsComplete
		node.SplitReason = result.SplitReason

		switch result.Decision {
		case "keep":
			node.Decision = tree.DecisionKeep
		case "split":
			node.Decision = tree.DecisionSplit
			nodesToSplit = append(nodesToSplit, node)
		case "merge_right":
			node.Decision = tree.DecisionMergeRight
			// Already handled merge above; if still merge_right, mark as keep
			if node.RightNeighbor == "" {
				node.Decision = tree.DecisionKeep
			}
		case "summarize_only":
			node.Decision = tree.DecisionSummarizeOnly
		case "discard":
			node.Decision = tree.DecisionDiscard
		default:
			node.Decision = tree.DecisionKeep
		}
	}

	// Split nodes that need further decomposition
	for _, node := range nodesToSplit {
		p.splitNode(st, node)
	}

	// Recurse if any splits happened
	if len(nodesToSplit) > 0 {
		return p.analyzeNodes(ctx, st, depth+1)
	}

	return nil
}

// splitNode divides a node into child nodes.
func (p *Pipeline) splitNode(st *tree.StoryTree, node *tree.StoryNode) {
	splitter := tree.NewSplitter(p.config.MaxChunkSize / 2) // smaller chunks for children
	chunks := splitter.SplitChapters([]string{node.TextContent})
	leaves := chunks.CollectLeaves()

	node.ChildrenIDs = make([]string, 0, len(leaves))
	for _, child := range leaves {
		child.ParentID = node.ID
		child.Level = node.Level + 1
		child.SourceChapter = node.SourceChapter
		st.AddNode(child)
		node.ChildrenIDs = append(node.ChildrenIDs, child.ID)
	}
}

// retryAnalyze attempts node analysis with retries.
func (p *Pipeline) retryAnalyze(ctx context.Context, node *tree.StoryNode, leftSummary string) (*agent.NodeAnalysisResult, error) {
	var lastErr error
	for i := 0; i < p.config.MaxRetries; i++ {
		result, err := p.nodeAnalyzer.Analyze(ctx, node, leftSummary)
		if err == nil {
			return result, nil
		}
		lastErr = err
		fmt.Printf("[Pipeline] Retry %d/%d for node %s: %v\n", i+1, p.config.MaxRetries, node.ID, err)
	}
	return nil, lastErr
}

// updateGraph builds the dynamic graph across all leaf nodes (Step 7).
func (p *Pipeline) updateGraph(ctx context.Context, st *tree.StoryTree) (*graph.Manager, error) {
	graphMgr := graph.NewManager()
	graphMgr.SetInitialSnapshot(st.LeafNodeIDs[0], graph.NewSnapshot(st.LeafNodeIDs[0]))

	for i, leafID := range st.LeafNodeIDs {
		node := st.GetNode(leafID)
		if node == nil {
			continue
		}

		beforeSnap := graphMgr.SnapshotsBefore()[leafID]
		if beforeSnap == nil {
			beforeSnap = graph.NewSnapshot(leafID)
		}

		fmt.Printf("[Pipeline] Graph update for node %s (%d/%d)...\n", leafID, i+1, len(st.LeafNodeIDs))
		update, err := p.graphAnalyzer.AnalyzeUpdate(ctx, node.TextContent, beforeSnap)
		if err != nil {
			fmt.Printf("[Pipeline] Graph analysis failed for %s: %v (skipping)\n", leafID, err)
			// Chain the before snapshot through
			if i+1 < len(st.LeafNodeIDs) {
				graphMgr.SetInitialSnapshot(st.LeafNodeIDs[i+1], beforeSnap.Clone())
			}
			continue
		}

		graphMgr.ApplyUpdate(leafID, update)

		// Chain to next node
		if i+1 < len(st.LeafNodeIDs) {
			graphMgr.ChainSnapshot(leafID, st.LeafNodeIDs[i+1])
		}
	}

	return graphMgr, nil
}

// generateScenes generates YAML scenes from plans (Steps 9-10).
func (p *Pipeline) generateScenes(ctx context.Context, plans []*scene.ScenePlan, st *tree.StoryTree, graphMgr *graph.Manager) ([]schema.Scene, error) {
	// Build context builder
	ctxBuilder := scene.NewContextBuilder(graphMgr.SnapshotsBefore(), graphMgr.SnapshotsAfter())

	// Prepare context packages
	type sceneJob struct {
		plan *scene.ScenePlan
		ctx  *scene.SceneContext
	}

	var jobs []sceneJob
	for _, plan := range plans {
		if plan.SceneCount == 0 {
			continue // summary-only plans don't generate scenes
		}

		// Gather source text
		var sourceText, sourceSummary string
		for _, nodeID := range plan.SourceNodeIDs {
			if n := st.GetNode(nodeID); n != nil {
				sourceText += n.TextContent + "\n\n"
				sourceSummary += n.Summary + " "
			}
		}

		sceneCtx := ctxBuilder.Build(plan, sourceText, sourceSummary)
		jobs = append(jobs, sceneJob{plan: plan, ctx: sceneCtx})
	}

	// Write scenes with goroutine pool
	allScenes := make([]schema.Scene, len(jobs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.config.MaxConcurrency)
	var mu sync.Mutex
	var firstErr error

	for i, job := range jobs {
		wg.Add(1)
		go func(idx int, j sceneJob) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fmt.Printf("[Pipeline] Writing scene %d (plan %s)...\n", idx+1, j.plan.ID)
			sc, err := p.sceneWriter.WriteScene(ctx, j.ctx)
			if err != nil {
				fmt.Printf("[Pipeline] Scene %d write failed: %v\n", idx+1, err)
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("scene %d: %w", idx+1, err)
				}
				mu.Unlock()
				return
			}
			sc.Sequence = idx + 1
			allScenes[idx] = *sc
		}(i, job)
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	// Filter out any zero-value scenes (from failed writes)
	result := make([]schema.Scene, 0, len(allScenes))
	for _, sc := range allScenes {
		if sc.ID != "" {
			result = append(result, sc)
		}
	}
	return result, nil
}

// collectCharacters builds the character list from leaf node analysis.
func (p *Pipeline) collectCharacters(leaves []*tree.StoryNode) []schema.Character {
	charMap := make(map[string]*schema.Character)
	for _, leaf := range leaves {
		for _, name := range leaf.Characters {
			if _, exists := charMap[name]; !exists {
				charMap[name] = &schema.Character{
					ID:   fmt.Sprintf("char_%03d", len(charMap)+1),
					Name: name,
				}
			}
		}
	}
	result := make([]schema.Character, 0, len(charMap))
	for _, ch := range charMap {
		result = append(result, *ch)
	}
	return result
}
