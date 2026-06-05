package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/fabula-studio/backend/internal/agent"
	"github.com/fabula-studio/backend/internal/graph"
	"github.com/fabula-studio/backend/internal/observability"
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
	eventBus       *observability.EventBus
	tracer         trace.Tracer
}

// New creates a Pipeline with the given config and agent configuration.
func New(cfg Config, modelName, apiKey, baseURL string, eventBus *observability.EventBus) *Pipeline {
	return &Pipeline{
		config:         cfg,
		nodeAnalyzer:   agent.NewNodeAnalyzerAgent(modelName, apiKey, baseURL),
		graphAnalyzer:  agent.NewGraphAnalyzerAgent(modelName, apiKey, baseURL),
		scenePlanner:   agent.NewScenePlannerAgent(modelName, apiKey, baseURL),
		sceneWriter:    agent.NewSceneWriterAgent(modelName, apiKey, baseURL),
		chiefEditor:    agent.NewChiefEditorAgent(modelName, apiKey, baseURL),
		validator:      &validator.Validator{},
		eventBus:       eventBus,
		tracer:         otel.Tracer("fabula-pipeline"),
	}
}


// publishEvent sends an event to the event bus if available.
func (p *Pipeline) publishEvent(event observability.PipelineEvent) {
	if p.eventBus != nil {
		p.eventBus.Publish(event)
	}
}

// Convert executes the full conversion pipeline.
func (p *Pipeline) Convert(ctx context.Context, title, author string, chapters []string) (*schema.Screenplay, error) {
	startTime := time.Now()
	
	// Start tracing span
	ctx, span := p.tracer.Start(ctx, "pipeline.convert")
	defer span.End()

	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventPipelineStart,
		Step:    "pipeline",
		Message: fmt.Sprintf("开始转换: %s (共 %d 章)", title, len(chapters)),
		Details: map[string]interface{}{
			"title":      title,
			"author":     author,
			"chapters":   len(chapters),
		},
	})

	// Step 1-3: Build story tree with hard cuts and horizontal chains
	stepStart := time.Now()
	ctx, treeSpan := p.tracer.Start(ctx, "pipeline.build_tree")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventNodeAnalyzing,
		Step:    "build_tree",
		Message: "步骤 1/7: 构建故事树...",
	})
	st := p.buildTree(chapters)
	treeSpan.End()
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventNodeAnalyzed,
		Step:     "build_tree",
		Message:  fmt.Sprintf("故事树构建完成: %d 节点, %d 叶子", len(st.Nodes), len(st.LeafNodeIDs)),
		Duration: durationPtr(time.Since(stepStart)),
		Details: map[string]interface{}{
			"total_nodes": len(st.Nodes),
			"leaf_nodes":  len(st.LeafNodeIDs),
		},
	})

	// Step 4-6: Analyze nodes recursively
	stepStart = time.Now()
	ctx, analyzeSpan := p.tracer.Start(ctx, "pipeline.analyze_nodes")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventNodeAnalyzing,
		Step:    "analyze_nodes",
		Message: "步骤 2/7: 分析节点...",
	})
	if err := p.analyzeNodes(ctx, st, 0); err != nil {
		analyzeSpan.RecordError(err)
		analyzeSpan.End()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "analyze_nodes",
			Error: err.Error(),
		})
		return nil, fmt.Errorf("node analysis failed: %w", err)
	}
	analyzeSpan.End()
	st.UpdateLeafIDs()
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventNodeAnalyzed,
		Step:     "analyze_nodes",
		Message:  fmt.Sprintf("节点分析完成: %d 叶子节点", len(st.LeafNodeIDs)),
		Duration: durationPtr(time.Since(stepStart)),
	})

	// Step 7: Update dynamic graph
	stepStart = time.Now()
	ctx, graphSpan := p.tracer.Start(ctx, "pipeline.update_graph")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventGraphUpdating,
		Step:    "update_graph",
		Message: "步骤 3/7: 更新动态图...",
	})
	graphMgr, err := p.updateGraph(ctx, st)
	if err != nil {
		graphSpan.RecordError(err)
		graphSpan.End()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "update_graph",
			Error: err.Error(),
		})
		return nil, fmt.Errorf("graph update failed: %w", err)
	}
	graphSpan.End()
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventGraphUpdated,
		Step:     "update_graph",
		Message:  "动态图更新完成",
		Duration: durationPtr(time.Since(stepStart)),
	})

	// Step 8: Plan scenes
	stepStart = time.Now()
	leaves := make([]*tree.StoryNode, 0, len(st.LeafNodeIDs))
	for _, lid := range st.LeafNodeIDs {
		if n := st.GetNode(lid); n != nil {
			leaves = append(leaves, n)
		}
	}
	ctx, planSpan := p.tracer.Start(ctx, "pipeline.plan_scenes")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventScenePlanning,
		Step:    "plan_scenes",
		Message: fmt.Sprintf("步骤 4/7: 规划场景 (%d 叶子节点)...", len(leaves)),
	})
	plans, err := p.scenePlanner.PlanScenes(ctx, leaves)
	if err != nil {
		planSpan.RecordError(err)
		planSpan.End()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "plan_scenes",
			Error: err.Error(),
		})
		return nil, fmt.Errorf("scene planning failed: %w", err)
	}
	planSpan.End()
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventScenePlanned,
		Step:     "plan_scenes",
		Message:  fmt.Sprintf("场景规划完成: %d 场景计划", len(plans)),
		Duration: durationPtr(time.Since(stepStart)),
		Details: map[string]interface{}{
			"plans_count": len(plans),
		},
	})

	// Step 9-10: Generate scenes
	stepStart = time.Now()
	ctx, writeSpan := p.tracer.Start(ctx, "pipeline.generate_scenes")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventSceneWriting,
		Step:    "generate_scenes",
		Message: "步骤 5/7: 生成场景...",
	})
	scenes, err := p.generateScenes(ctx, plans, st, graphMgr)
	if err != nil {
		writeSpan.RecordError(err)
		writeSpan.End()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "generate_scenes",
			Error: err.Error(),
		})
		return nil, fmt.Errorf("scene generation failed: %w", err)
	}
	writeSpan.End()
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventSceneWritten,
		Step:     "generate_scenes",
		Message:  fmt.Sprintf("场景生成完成: %d 场景", len(scenes)),
		Duration: durationPtr(time.Since(stepStart)),
	})

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
	stepStart = time.Now()
	ctx, editorSpan := p.tracer.Start(ctx, "pipeline.editor_review")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventEditorReviewing,
		Step:    "editor_review",
		Message: "步骤 6/7: 总编审查...",
	})
	editResult, err := p.chiefEditor.ReviewAndRevise(ctx, screenplay)
	if err != nil {
		editorSpan.RecordError(err)
		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventEditorReviewed,
			Step:    "editor_review",
			Message: fmt.Sprintf("总编审查失败 (非致命): %v", err),
			Error:   err.Error(),
		})
	} else {
		if editResult.Screenplay != nil {
			screenplay = editResult.Screenplay
		}
		p.publishEvent(observability.PipelineEvent{
			Type:     observability.EventEditorReviewed,
			Step:     "editor_review",
			Message:  fmt.Sprintf("总编审查完成: 发现 %d 个问题", len(editResult.Issues)),
			Duration: durationPtr(time.Since(stepStart)),
			Details: map[string]interface{}{
				"issues_count":  len(editResult.Issues),
				"changes_count": len(editResult.Changes),
			},
		})
	}
	editorSpan.End()

	// Step 13: Validation
	stepStart = time.Now()
	ctx, validSpan := p.tracer.Start(ctx, "pipeline.validation")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventValidation,
		Step:    "validation",
		Message: "步骤 7/7: 校验...",
	})
	validateResult := p.validator.Validate(screenplay, 3)
	if !validateResult.Valid {
		// Log validation errors but continue - return result with warnings
		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventValidation,
			Step:    "validation",
			Message: fmt.Sprintf("校验警告: %s", strings.Join(validateResult.Errors, "; ")),
		})
		// Add errors as warnings instead of failing
		validateResult.Warnings = append(validateResult.Warnings, validateResult.Errors...)
	}
	validSpan.End()
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventValidation,
		Step:     "validation",
		Message:  fmt.Sprintf("校验完成: %d 警告", len(validateResult.Warnings)),
		Duration: durationPtr(time.Since(stepStart)),
	})

	totalDuration := time.Since(startTime)
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventPipelineEnd,
		Step:     "pipeline",
		Message:  fmt.Sprintf("转换完成! 耗时: %v", totalDuration),
		Duration: &totalDuration,
		Details: map[string]interface{}{
			"scenes_count":     len(screenplay.Scenes),
			"characters_count": len(screenplay.Characters),
		},
	})


	return screenplay, nil
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
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

		// Analyze the node with tracing
		leftSummary := ""
		left := cm.GetLeftNeighbor(node.ID)
		if left != nil {
			leftSummary = left.Summary
		}

		// Start span for this node
		nodeSpanName := fmt.Sprintf("pipeline.analyze_node.%s", node.ID)
		_, nodeSpan := p.tracer.Start(ctx, nodeSpanName)
		nodeSpan.SetAttributes(
			attribute.String("node.id", node.ID),
			attribute.Int("node.level", node.Level),
			attribute.Int("node.text_length", len(node.TextContent)),
		)

		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventNodeAnalyzing,
			Step:    "analyze_node",
			NodeID:  node.ID,
			Message: fmt.Sprintf("分析节点 %s...", node.ID),
		})

		result, err := p.retryAnalyze(ctx, node, leftSummary)
		if err != nil {
			nodeSpan.RecordError(err)
			nodeSpan.End()
			p.publishEvent(observability.PipelineEvent{
				Type:   observability.EventNodeFailed,
				Step:   "analyze_node",
				NodeID: node.ID,
				Error:  err.Error(),
			})
			node.Decision = tree.DecisionSummarizeOnly
			continue
		}
		nodeSpan.End()

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

		// Skip nodes that don't need graph tracking
		if node.Decision == tree.DecisionSummarizeOnly || node.Decision == tree.DecisionDiscard {
			fmt.Printf("[Pipeline] Skipping graph update for %s (decision: %s)\n", leafID, node.Decision)
			if i+1 < len(st.LeafNodeIDs) {
				beforeSnap := graphMgr.SnapshotsBefore()[leafID]
				if beforeSnap == nil {
					beforeSnap = graph.NewSnapshot(leafID)
				}
				graphMgr.SetInitialSnapshot(st.LeafNodeIDs[i+1], beforeSnap.Clone())
			}
			continue
		}

		beforeSnap := graphMgr.SnapshotsBefore()[leafID]
		if beforeSnap == nil {
			beforeSnap = graph.NewSnapshot(leafID)
		}

		// Use summary instead of full text to avoid token overflow
		graphText := node.Summary
		if graphText == "" {
			graphText = node.TextContent
			// Truncate if still too long
			runes := []rune(graphText)
			if len(runes) > 1500 {
				graphText = string(runes[:1500]) + "..."
			}
		}

		// Start span for this node's graph update
		graphSpanName := fmt.Sprintf("pipeline.graph_update.%s", leafID)
		_, graphNodeSpan := p.tracer.Start(ctx, graphSpanName)
		graphNodeSpan.SetAttributes(
			attribute.String("node.id", leafID),
			attribute.Int("node.index", i),
		)

		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventGraphUpdating,
			Step:    "graph_update",
			NodeID:  leafID,
			Message: fmt.Sprintf("图更新: %s (%d/%d)", leafID, i+1, len(st.LeafNodeIDs)),
		})

		update, err := p.graphAnalyzer.AnalyzeUpdate(ctx, graphText, beforeSnap)
		if err != nil {
			graphNodeSpan.RecordError(err)
			graphNodeSpan.End()
			p.publishEvent(observability.PipelineEvent{
				Type:   observability.EventGraphUpdated,
				Step:   "graph_update",
				NodeID: leafID,
				Error:  err.Error(),
			})
			if i+1 < len(st.LeafNodeIDs) {
				graphMgr.SetInitialSnapshot(st.LeafNodeIDs[i+1], beforeSnap.Clone())
			}
			continue
		}

		graphMgr.ApplyUpdate(leafID, update)
		graphNodeSpan.End()

		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventGraphUpdated,
			Step:    "graph_update",
			NodeID:  leafID,
			Message: fmt.Sprintf("图更新完成: %s", leafID),
		})

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

			// Start span for this scene
			sceneSpanName := fmt.Sprintf("pipeline.write_scene.%s", j.plan.ID)
			_, sceneSpan := p.tracer.Start(ctx, sceneSpanName)
			sceneSpan.SetAttributes(
				attribute.String("scene.plan_id", j.plan.ID),
				attribute.Int("scene.index", idx),
			)

			p.publishEvent(observability.PipelineEvent{
				Type:    observability.EventSceneWriting,
				Step:    "write_scene",
				NodeID:  j.plan.ID,
				Message: fmt.Sprintf("写场景 %d (计划 %s)...", idx+1, j.plan.ID),
			})

			sc, err := p.sceneWriter.WriteScene(ctx, j.ctx)
			if err != nil {
				sceneSpan.RecordError(err)
				sceneSpan.End()
				p.publishEvent(observability.PipelineEvent{
					Type:   observability.EventSceneWritten,
					Step:   "write_scene",
					NodeID: j.plan.ID,
					Error:  err.Error(),
				})
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("scene %d: %w", idx+1, err)
				}
				mu.Unlock()
				return
			}
			sceneSpan.End()
			sc.Sequence = idx + 1
			allScenes[idx] = *sc

			p.publishEvent(observability.PipelineEvent{
				Type:    observability.EventSceneWritten,
				Step:    "write_scene",
				NodeID:  j.plan.ID,
				Message: fmt.Sprintf("场景 %d 完成", idx+1),
			})
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
		for _, rawName := range leaf.Characters {
			name := strings.TrimSpace(rawName)
			// Remove common quotation marks and brackets that AI sometimes includes
			name = strings.Trim(name, "\"'「」『』\u201c\u201d")
			if name == "" {
				continue
			}
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
