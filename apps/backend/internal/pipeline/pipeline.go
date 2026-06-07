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
	"gopkg.in/yaml.v3"

	"github.com/fabula-studio/backend/internal/agent"
	"github.com/fabula-studio/backend/internal/graph"
	"github.com/fabula-studio/backend/internal/observability"
	"github.com/fabula-studio/backend/internal/scene"
	"github.com/fabula-studio/backend/internal/schema"
	"github.com/fabula-studio/backend/internal/segment"
	"github.com/fabula-studio/backend/internal/tree"
	"github.com/fabula-studio/backend/internal/validator"
)

// PipelineStatus holds the current state of the pipeline.
type PipelineStatus struct {
	State       string    `json:"state"` // idle, running, completed, failed
	CurrentStep string    `json:"current_step"`
	Progress    int       `json:"progress"` // 0-100
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
}

// PipelineResult holds the complete output of a pipeline run.
type PipelineResult struct {
	SourceIndex *segment.SourceIndex        `json:"source_index,omitempty"`
	StoryBeats  []segment.StoryBeat         `json:"story_beats,omitempty"`
	Tree        *tree.StoryTree             `json:"tree"`
	GraphMgr    *graph.Manager              `json:"-"`
	Plans       []*scene.ScenePlan          `json:"plans"`
	Artifacts   *schema.GenerationArtifacts `json:"artifacts,omitempty"`
	Screenplay  *schema.Screenplay          `json:"screenplay"`
	YAMLStr     string                      `json:"yaml"`
	Duration    time.Duration               `json:"duration"`
	CompletedAt time.Time                   `json:"completed_at"`
}

// Pipeline orchestrates the full novel-to-screenplay conversion.
type Pipeline struct {
	config             Config
	unitAggregator     *agent.UnitAggregatorAgent
	nodeAnalyzer       *agent.NodeAnalyzerAgent
	graphAnalyzer      *agent.GraphAnalyzerAgent
	storyBeatExtractor *agent.StoryBeatExtractorAgent
	scenePlanner       *agent.ScenePlannerAgent
	sceneWriter        *agent.SceneWriterAgent
	chiefEditor        *agent.ChiefEditorAgent
	validator          *validator.Validator
	eventBus           *observability.EventBus
	tracer             trace.Tracer

	mu         sync.RWMutex
	status     PipelineStatus
	lastResult *PipelineResult
}

// New creates a Pipeline with the given config and agent configuration.
func New(cfg Config, modelName, apiKey, baseURL string, eventBus *observability.EventBus) *Pipeline {
	return &Pipeline{
		config:             cfg,
		unitAggregator:     agent.NewUnitAggregatorAgent(modelName, apiKey, baseURL),
		nodeAnalyzer:       agent.NewNodeAnalyzerAgent(modelName, apiKey, baseURL),
		graphAnalyzer:      agent.NewGraphAnalyzerAgent(modelName, apiKey, baseURL),
		storyBeatExtractor: agent.NewStoryBeatExtractorAgent(modelName, apiKey, baseURL),
		scenePlanner:       agent.NewScenePlannerAgent(modelName, apiKey, baseURL),
		sceneWriter:        agent.NewSceneWriterAgent(modelName, apiKey, baseURL),
		chiefEditor:        agent.NewChiefEditorAgent(modelName, apiKey, baseURL),
		validator:          &validator.Validator{},
		eventBus:           eventBus,
		tracer:             otel.Tracer("fabula-pipeline"),
	}
}

// publishEvent sends an event to the event bus if available.
func (p *Pipeline) publishEvent(event observability.PipelineEvent) {
	if p.eventBus != nil {
		p.eventBus.Publish(event)
	}
}

// Status returns the current pipeline status (thread-safe).
func (p *Pipeline) Status() PipelineStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// Result returns the last completed pipeline result (thread-safe).
func (p *Pipeline) Result() *PipelineResult {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastResult
}

// GraphSnapshots returns the before/after graph snapshots from the last run.
func (p *Pipeline) GraphSnapshots() (before, after map[string]*graph.GraphSnapshot) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.lastResult == nil || p.lastResult.GraphMgr == nil {
		return nil, nil
	}
	return p.lastResult.GraphMgr.SnapshotsBefore(), p.lastResult.GraphMgr.SnapshotsAfter()
}

// Convert executes the full conversion pipeline.
func (p *Pipeline) Convert(ctx context.Context, title, author string, chapters []string) (*schema.Screenplay, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "未命名小说"
	}
	author = strings.TrimSpace(author)
	if author == "" {
		author = "未知作者"
	}
	profile, hasProfile := schema.AdaptationProfileFromContext(ctx)

	startTime := time.Now()

	p.mu.Lock()
	p.status = PipelineStatus{State: "running", StartedAt: startTime}
	p.lastResult = nil
	p.mu.Unlock()

	if p.eventBus != nil {
		p.eventBus.Clear()
	}

	// Start tracing span
	ctx, span := p.tracer.Start(ctx, "pipeline.convert")
	defer span.End()

	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventPipelineStart,
		Step:    "pipeline",
		Message: fmt.Sprintf("开始转换: %s (共 %d 章)", title, len(chapters)),
		Details: map[string]interface{}{
			"title":                  title,
			"author":                 author,
			"chapters":               len(chapters),
			"adaptation_profile":     profile,
			"has_adaptation_profile": hasProfile,
		},
	})

	p.mu.Lock()
	p.status.CurrentStep = "extract_story_beats"
	p.status.Progress = 5
	p.mu.Unlock()

	// Step 1: Build the source index and extract adaptation story beats.
	stepStart := time.Now()
	ctx, beatSpan := p.tracer.Start(ctx, "pipeline.extract_story_beats")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventNodeAnalyzing,
		Step:    "extract_story_beats",
		Message: "步骤 1/8: 提取故事节拍...",
	})
	sourceIndex := segment.BuildSourceIndex(chapters)
	if len(sourceIndex.Sentences) == 0 {
		beatSpan.End()
		p.mu.Lock()
		p.status.State = "failed"
		p.status.Error = "no usable sentences found"
		p.mu.Unlock()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "extract_story_beats",
			Error: "no usable sentences found",
		})
		return nil, fmt.Errorf("story beat extraction failed: no usable sentences found")
	}
	storyBeats, err := p.storyBeatExtractor.Extract(ctx, sourceIndex)
	if err != nil {
		beatSpan.RecordError(err)
		beatSpan.End()
		p.mu.Lock()
		p.status.State = "failed"
		p.status.Error = err.Error()
		p.mu.Unlock()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "extract_story_beats",
			Error: err.Error(),
		})
		return nil, fmt.Errorf("story beat extraction failed: %w", err)
	}
	beatSpan.End()
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventNodeAnalyzed,
		Step:     "extract_story_beats",
		Message:  fmt.Sprintf("故事节拍提取完成: %d 个节拍", len(storyBeats)),
		Duration: durationPtr(time.Since(stepStart)),
		Details: map[string]interface{}{
			"source_index": sourceIndex,
			"story_beats":  storyBeats,
			"beats_count":  len(storyBeats),
		},
	})

	p.mu.Lock()
	p.status.CurrentStep = "aggregate_units"
	p.status.Progress = 15
	p.mu.Unlock()

	// Step 2-3: Aggregate the same sentence stream into the legacy story tree.
	stepStart = time.Now()
	ctx, treeSpan := p.tracer.Start(ctx, "pipeline.aggregate_units")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventNodeAnalyzing,
		Step:    "aggregate_units",
		Message: "步骤 2/8: 聚合剧情单元...",
	})
	st, err := p.buildTreeFromSentences(ctx, sourceIndex.Sentences)
	if err != nil {
		treeSpan.RecordError(err)
		treeSpan.End()
		p.mu.Lock()
		p.status.State = "failed"
		p.status.Error = err.Error()
		p.mu.Unlock()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "aggregate_units",
			Error: err.Error(),
		})
		return nil, fmt.Errorf("unit aggregation failed: %w", err)
	}
	treeSpan.End()
	st.UpdateLeafIDs()

	// Emit the complete aggregated tree so the frontend can render it immediately.
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventTreeSnapshot,
		Step:     "aggregate_units",
		Message:  fmt.Sprintf("剧情单元聚合完成: %d 节点, %d 叶子", len(st.Nodes), len(st.LeafNodeIDs)),
		Duration: durationPtr(time.Since(stepStart)),
		Details: map[string]interface{}{
			"tree":        st,
			"total_nodes": len(st.Nodes),
			"leaf_nodes":  len(st.LeafNodeIDs),
		},
	})

	p.mu.Lock()
	p.status.CurrentStep = "update_graph"
	p.status.Progress = 40
	p.mu.Unlock()

	// Step 7: Update dynamic graph
	stepStart = time.Now()
	ctx, graphSpan := p.tracer.Start(ctx, "pipeline.update_graph")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventGraphUpdating,
		Step:    "update_graph",
		Message: "步骤 4/8: 更新动态图...",
	})
	graphMgr, err := p.updateGraph(ctx, st)
	if err != nil {
		graphSpan.RecordError(err)
		graphSpan.End()
		p.mu.Lock()
		p.status.State = "failed"
		p.status.Error = err.Error()
		p.mu.Unlock()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "update_graph",
			Error: err.Error(),
		})
		return nil, fmt.Errorf("graph update failed: %w", err)
	}
	graphSpan.End()
	graphChars := 0
	if afterSnap := graphMgr.SnapshotsAfter()[st.LeafNodeIDs[len(st.LeafNodeIDs)-1]]; afterSnap != nil {
		graphChars = len(afterSnap.Characters)
	}
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventGraphUpdated,
		Step:     "update_graph",
		Message:  "动态图更新完成",
		Duration: durationPtr(time.Since(stepStart)),
		Details: map[string]interface{}{
			"characters_count": graphChars,
		},
	})

	p.mu.Lock()
	p.status.CurrentStep = "plan_scenes"
	p.status.Progress = 55
	p.mu.Unlock()

	// Step 8: Plan scenes
	stepStart = time.Now()
	leaves := make([]*tree.StoryNode, 0, len(st.LeafNodeIDs))
	for _, lid := range st.LeafNodeIDs {
		if n := st.GetNode(lid); n != nil {
			leaves = append(leaves, n)
		}
	}
	sceneCandidates := scene.BuildSceneCandidates(storyBeats)
	ctx, planSpan := p.tracer.Start(ctx, "pipeline.plan_scenes")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventScenePlanning,
		Step:    "plan_scenes",
		Message: fmt.Sprintf("步骤 5/8: 规划场景 (%d 故事节拍)...", len(sceneCandidates)),
	})
	plans, err := p.scenePlanner.PlanFromCandidates(ctx, sceneCandidates)
	if err != nil {
		planSpan.RecordError(err)
		planSpan.End()
		p.mu.Lock()
		p.status.State = "failed"
		p.status.Error = err.Error()
		p.mu.Unlock()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "plan_scenes",
			Error: err.Error(),
		})
		return nil, fmt.Errorf("scene planning failed: %w", err)
	}
	planSpan.End()
	planSummaries := make([]map[string]interface{}, len(plans))
	for i, plan := range plans {
		planSummaries[i] = map[string]interface{}{
			"id":              plan.ID,
			"source_node_ids": plan.SourceNodeIDs,
			"scene_count":     plan.SceneCount,
			"purpose":         plan.Purpose,
			"location":        plan.Location,
		}
	}
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventScenePlanned,
		Step:     "plan_scenes",
		Message:  fmt.Sprintf("场景规划完成: %d 场景计划", len(plans)),
		Duration: durationPtr(time.Since(stepStart)),
		Details: map[string]interface{}{
			"plans_count": len(plans),
			"plans":       planSummaries,
		},
	})

	p.mu.Lock()
	p.status.CurrentStep = "generate_scenes"
	p.status.Progress = 65
	p.mu.Unlock()

	// Step 9-10: Generate scenes
	stepStart = time.Now()
	ctx, writeSpan := p.tracer.Start(ctx, "pipeline.generate_scenes")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventSceneWriting,
		Step:    "generate_scenes",
		Message: "步骤 6/8: 生成场景...",
	})
	scenes, err := p.generateScenesSequential(ctx, plans, sourceIndex, sceneCandidates, st, graphMgr)
	if err != nil {
		writeSpan.RecordError(err)
		writeSpan.End()
		p.mu.Lock()
		p.status.State = "failed"
		p.status.Error = err.Error()
		p.mu.Unlock()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "generate_scenes",
			Error: err.Error(),
		})
		return nil, fmt.Errorf("scene generation failed: %w", err)
	}
	writeSpan.End()
	sceneHeadings := make([]string, 0, len(scenes))
	for _, sc := range scenes {
		sceneHeadings = append(sceneHeadings, sc.Heading)
	}
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventSceneWritten,
		Step:     "generate_scenes",
		Message:  fmt.Sprintf("场景生成完成: %d 场景", len(scenes)),
		Duration: durationPtr(time.Since(stepStart)),
		Details: map[string]interface{}{
			"scenes_count": len(scenes),
			"scenes":       sceneHeadings,
		},
	})

	p.mu.Lock()
	p.status.CurrentStep = "editor_review"
	p.status.Progress = 80
	p.mu.Unlock()

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
		Message: "步骤 7/8: 总编审查...",
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
				"issues":        editResult.Issues,
				"changes":       editResult.Changes,
			},
		})
	}
	editorSpan.End()

	p.mu.Lock()
	p.status.CurrentStep = "validation"
	p.status.Progress = 90
	p.mu.Unlock()

	// Step 13: Validation
	stepStart = time.Now()
	ctx, validSpan := p.tracer.Start(ctx, "pipeline.validation")
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventValidation,
		Step:    "validation",
		Message: "步骤 8/8: 校验...",
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

	p.mu.Lock()
	p.status.CurrentStep = "completed"
	p.status.Progress = 100
	p.mu.Unlock()

	totalDuration := time.Since(startTime)
	yamlBytes, _ := yaml.Marshal(screenplay)

	p.mu.Lock()
	p.lastResult = &PipelineResult{
		SourceIndex: sourceIndex,
		StoryBeats:  storyBeats,
		Tree:        st,
		GraphMgr:    graphMgr,
		Plans:       plans,
		Artifacts:   generationArtifacts(sourceIndex, storyBeats, plans, graphMgr),
		Screenplay:  screenplay,
		YAMLStr:     string(yamlBytes),
		Duration:    totalDuration,
		CompletedAt: time.Now(),
	}
	p.status.State = "completed"
	p.mu.Unlock()

	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventPipelineEnd,
		Step:     "pipeline",
		Message:  fmt.Sprintf("转换完成! 耗时: %v", totalDuration),
		Duration: &totalDuration,
		Details: map[string]interface{}{
			"scenes_count":     len(screenplay.Scenes),
			"characters_count": len(screenplay.Characters),
			"screenplay":       screenplay,
			"yaml":             string(yamlBytes),
		},
	})

	return screenplay, nil
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

// buildTree creates a flat unit tree from sentence-stream aggregation.
func (p *Pipeline) buildTree(ctx context.Context, chapters []string) (*tree.StoryTree, error) {
	return p.buildTreeFromSentences(ctx, segment.SplitChapters(chapters))
}

func (p *Pipeline) buildTreeFromSentences(ctx context.Context, sentences []segment.Sentence) (*tree.StoryTree, error) {
	if len(sentences) == 0 {
		return nil, fmt.Errorf("no usable sentences found")
	}
	units := make([]segment.UnitResult, 0)
	for start := 0; start < len(sentences); {
		state := segment.NewAggregationState(sentences, start, segment.DefaultBatchSize)
		result, err := p.unitAggregator.Aggregate(ctx, state)
		if err != nil {
			return nil, fmt.Errorf("aggregate from %s: %w", sentences[start].ID, err)
		}
		end, err := state.ValidateFinish(result)
		if err != nil {
			return nil, fmt.Errorf("validate aggregation from %s: %w", sentences[start].ID, err)
		}
		units = append(units, *result)
		start = end + 1
	}
	return segment.BuildTree(sentences, units)
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

		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventNodeAnalyzed,
			Step:    "analyze_node",
			NodeID:  node.ID,
			Message: fmt.Sprintf("节点 %s 分析完成: %s", node.ID, node.Decision),
			Details: map[string]interface{}{
				"node": node,
			},
		})
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

// splitNode divides a node into child nodes and emits events for each.
func (p *Pipeline) splitNode(st *tree.StoryTree, node *tree.StoryNode) {
	splitter := tree.NewSplitter(p.config.MaxChunkSize / 2)
	chunks := splitter.SplitChapters([]string{node.TextContent})
	leaves := chunks.CollectLeaves()

	// Re-ID to avoid collisions: parentID_c001, parentID_c002, ...
	oldToNew := make(map[string]string, len(leaves))
	for i, child := range leaves {
		oldID := child.ID
		newID := fmt.Sprintf("%s_c%03d", node.ID, i+1)
		delete(st.Nodes, oldID)
		child.ID = newID
		oldToNew[oldID] = newID
	}
	// Fix RightNeighbor references
	for _, child := range leaves {
		if child.RightNeighbor != "" {
			if newRN, ok := oldToNew[child.RightNeighbor]; ok {
				child.RightNeighbor = newRN
			}
		}
	}

	node.ChildrenIDs = make([]string, 0, len(leaves))
	for _, child := range leaves {
		child.ParentID = node.ID
		child.Level = node.Level + 1
		child.SourceChapter = node.SourceChapter
		st.AddNode(child)
		node.ChildrenIDs = append(node.ChildrenIDs, child.ID)

		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventTreeNodeAdded,
			Step:    "analyze_node",
			NodeID:  child.ID,
			Message: fmt.Sprintf("节点拆分: %s → %s", node.ID, child.ID),
			Details: map[string]interface{}{
				"node": child,
			},
		})
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

		newCharNames := make([]string, len(update.NewCharacters))
		for j, nc := range update.NewCharacters {
			newCharNames[j] = nc.Name
		}
		newRelDescs := make([]string, len(update.RelationChanges))
		for j, rc := range update.RelationChanges {
			newRelDescs[j] = fmt.Sprintf("%s-%s", rc.CharA, rc.CharB)
		}

		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventGraphUpdated,
			Step:    "graph_update",
			NodeID:  leafID,
			Message: fmt.Sprintf("图更新完成: %s", leafID),
			Details: map[string]interface{}{
				"new_characters": newCharNames,
				"new_relations":  newRelDescs,
			},
		})

		// Chain to next node
		if i+1 < len(st.LeafNodeIDs) {
			graphMgr.ChainSnapshot(leafID, st.LeafNodeIDs[i+1])
		}
	}

	return graphMgr, nil
}

// generateScenesSequential writes scenes in plan order and updates the graph from each generated scene.
func (p *Pipeline) generateScenesSequential(ctx context.Context, plans []*scene.ScenePlan, sourceIndex *segment.SourceIndex, candidates []scene.SceneCandidate, st *tree.StoryTree, graphMgr *graph.Manager) ([]schema.Scene, error) {
	candidateByID := make(map[string]scene.SceneCandidate, len(candidates))
	for _, candidate := range candidates {
		candidateByID[candidate.ID] = candidate
	}

	current := finalGraphSnapshot(graphMgr, st)
	if current == nil {
		current = graph.NewSnapshot("scene_generation_start")
	}
	ctxBuilder := scene.NewContextBuilder(graphMgr.SnapshotsBefore(), graphMgr.SnapshotsAfter())
	result := make([]schema.Scene, 0, len(plans))

	for _, plan := range plans {
		if plan == nil {
			continue
		}
		before := current.Clone()
		before.NodeID = plan.ID
		graphMgr.SetInitialSnapshot(plan.ID, before)
		if plan.SceneCount == 0 {
			graphMgr.SetAfterSnapshot(plan.ID, before)
			current = before
			continue
		}

		sourceText, sourceSummary := sourceContextForPlan(sourceIndex, candidateByID, plan)
		sceneCtx := ctxBuilder.Build(plan, sourceText, sourceSummary)

		sceneSpanName := fmt.Sprintf("pipeline.write_scene.%s", plan.ID)
		_, sceneSpan := p.tracer.Start(ctx, sceneSpanName)
		sceneSpan.SetAttributes(
			attribute.String("scene.plan_id", plan.ID),
			attribute.Int("scene.index", len(result)),
		)

		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventSceneWriting,
			Step:    "write_scene",
			NodeID:  plan.ID,
			Message: fmt.Sprintf("写场景 %d (计划 %s)...", len(result)+1, plan.ID),
		})

		sc, err := p.sceneWriter.WriteScene(ctx, sceneCtx)
		if err != nil {
			sceneSpan.RecordError(err)
			sceneSpan.End()
			p.publishEvent(observability.PipelineEvent{Type: observability.EventSceneWritten, Step: "write_scene", NodeID: plan.ID, Error: err.Error()})
			return nil, fmt.Errorf("scene %d: %w", len(result)+1, err)
		}
		sc.Sequence = len(result) + 1
		if sc.ID == "" {
			sc.ID = fmt.Sprintf("scene_%03d", sc.Sequence)
		}

		graphMgr.SetInitialSnapshot(sc.ID, before)
		update, err := p.graphAnalyzer.AnalyzeSceneUpdate(ctx, sc, before)
		if err != nil {
			sceneSpan.RecordError(err)
			sceneSpan.End()
			p.publishEvent(observability.PipelineEvent{Type: observability.EventGraphUpdated, Step: "graph_hook", NodeID: sc.ID, Error: err.Error()})
			return nil, fmt.Errorf("scene %d graph update: %w", sc.Sequence, err)
		}
		after := graphMgr.ApplyUpdate(sc.ID, update)
		graphMgr.SetAfterSnapshot(plan.ID, after)
		current = after
		sceneSpan.End()

		result = append(result, *sc)
		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventSceneWritten,
			Step:    "write_scene",
			NodeID:  plan.ID,
			Message: fmt.Sprintf("场景 %d 完成", sc.Sequence),
			Details: map[string]interface{}{
				"graph_before": plan.ID,
				"graph_after":  sc.ID,
			},
		})
	}
	return result, nil
}

func sourceContextForPlan(sourceIndex *segment.SourceIndex, candidates map[string]scene.SceneCandidate, plan *scene.ScenePlan) (string, string) {
	var sentenceIDs []string
	summaries := make([]string, 0, len(plan.SourceNodeIDs))
	for _, sourceID := range plan.SourceNodeIDs {
		candidate, ok := candidates[sourceID]
		if !ok {
			continue
		}
		sentenceIDs = append(sentenceIDs, candidate.SourceSentenceIDs...)
		if candidate.Summary != "" {
			summaries = append(summaries, candidate.Summary)
		}
	}
	return sourceIndex.TextForIDs(sentenceIDs), strings.Join(summaries, " ")
}

func finalGraphSnapshot(graphMgr *graph.Manager, st *tree.StoryTree) *graph.GraphSnapshot {
	if st != nil {
		for i := len(st.LeafNodeIDs) - 1; i >= 0; i-- {
			if snap := graphMgr.SnapshotsAfter()[st.LeafNodeIDs[i]]; snap != nil {
				return snap
			}
			if snap := graphMgr.SnapshotsBefore()[st.LeafNodeIDs[i]]; snap != nil {
				return snap
			}
		}
	}
	for _, snap := range graphMgr.SnapshotsAfter() {
		return snap
	}
	return nil
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

func generationArtifacts(sourceIndex *segment.SourceIndex, storyBeats []segment.StoryBeat, plans []*scene.ScenePlan, graphMgr *graph.Manager) *schema.GenerationArtifacts {
	artifacts := &schema.GenerationArtifacts{
		SourceIndex:   schemaSourceIndex(sourceIndex),
		StoryBeats:    schemaStoryBeats(sourceIndex, storyBeats),
		ScenePlan:     schemaScenePlan(plans),
		GraphSnapshot: latestGraphSnapshot(graphMgr),
	}
	if artifacts.SourceIndex == nil && len(artifacts.StoryBeats) == 0 && artifacts.ScenePlan == nil && artifacts.GraphSnapshot == nil {
		return nil
	}
	return artifacts
}

func schemaSourceIndex(idx *segment.SourceIndex) *schema.SourceIndex {
	if idx == nil {
		return nil
	}
	out := &schema.SourceIndex{Sentences: make([]schema.SourceSentence, 0, len(idx.Sentences))}
	for _, sentence := range idx.Sentences {
		out.Sentences = append(out.Sentences, schema.SourceSentence{ID: sentence.ID, Index: sentence.Index, Chapter: sentence.Chapter, ChapterIndex: sentence.ChapterIndex, Text: sentence.Text})
	}
	return out
}

func schemaStoryBeats(idx *segment.SourceIndex, beats []segment.StoryBeat) []schema.StoryBeat {
	if len(beats) == 0 {
		return nil
	}
	out := make([]schema.StoryBeat, 0, len(beats))
	for _, beat := range beats {
		locations := []string(nil)
		if beat.Location != "" {
			locations = []string{beat.Location}
		}
		out = append(out, schema.StoryBeat{ID: beat.ID, Sequence: beat.Sequence, Summary: beat.Summary, Purpose: beat.DramaticPurpose, Characters: beat.Characters, Locations: locations, SourceRefs: schemaSourceRefs(idx, beat.SourceSentenceIDs)})
	}
	return out
}

func schemaScenePlan(plans []*scene.ScenePlan) *schema.ScenePlan {
	if len(plans) == 0 {
		return nil
	}
	root := &schema.ScenePlan{ID: "scene_plan", Scenes: make([]schema.ScenePlan, 0, len(plans))}
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		root.Scenes = append(root.Scenes, schema.ScenePlan{ID: plan.ID, Sequence: plan.Sequence, Purpose: plan.Purpose, Location: plan.Location, TimeFrame: plan.TimeFrame, Characters: plan.Characters, KeyPlotPoints: plan.KeyPlotPoints})
	}
	if len(root.Scenes) == 0 {
		return nil
	}
	return root
}

func schemaSourceRefs(idx *segment.SourceIndex, ids []string) []schema.SourceSentenceRef {
	if idx == nil || len(ids) == 0 {
		return nil
	}
	out := make([]schema.SourceSentenceRef, 0, len(ids))
	for _, id := range ids {
		if ref, ok := idx.RefFor(id); ok {
			out = append(out, schema.SourceSentenceRef{SentenceID: ref.ID, StartIndex: ref.Index, EndIndex: ref.Index})
		}
	}
	return out
}

func latestGraphSnapshot(graphMgr *graph.Manager) any {
	if graphMgr == nil {
		return nil
	}
	var latest *graph.GraphSnapshot
	for _, snap := range graphMgr.SnapshotsAfter() {
		if snap != nil {
			latest = snap
		}
	}
	return latest
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
