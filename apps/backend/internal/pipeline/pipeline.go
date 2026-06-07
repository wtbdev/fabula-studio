package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	graphAnalyzer      *agent.GraphAnalyzerAgent
	storyBeatExtractor *agent.StoryBeatExtractorAgent
	scenePlanner       *agent.ScenePlannerAgent
	sceneWriter        *agent.SceneWriterAgent
	chiefEditor        *agent.ChiefEditorAgent
	validator          *validator.Validator
	eventBus           *observability.EventBus
	tracer             trace.Tracer

	mu          sync.RWMutex
	status      PipelineStatus
	lastResult  *PipelineResult
	runMetadata RunMetadata
}

// New creates a Pipeline with the given config and agent configuration.
func New(cfg Config, modelName, apiKey, baseURL string, eventBus *observability.EventBus) *Pipeline {
	return &Pipeline{
		config:             cfg,
		unitAggregator:     agent.NewUnitAggregatorAgent(modelName, apiKey, baseURL),
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
	p.mu.RLock()
	meta := p.runMetadata
	progress := p.status.Progress
	p.mu.RUnlock()

	if event.ProjectID == "" {
		event.ProjectID = meta.ProjectID
	}
	if event.JobID == "" {
		event.JobID = meta.JobID
	}
	if event.RunID == "" {
		event.RunID = meta.RunID
	}
	if event.TraceID == "" {
		event.TraceID = meta.TraceID
	}
	if event.Progress == nil {
		event.Progress = intPtr(progress)
	}
	if p.eventBus != nil {
		p.eventBus.Publish(event)
	}
}

func intPtr(v int) *int {
	return &v
}

func setPipelineStepAttributes(span trace.Span, step string, progress int) {
	span.SetAttributes(
		attribute.String("pipeline.step", step),
		attribute.Int("pipeline.progress", progress),
	)
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
	meta, _ := RunMetadataFromContext(ctx)
	if meta.RunID == "" {
		meta.RunID = uuid.NewString()
	}

	startTime := time.Now()

	p.mu.Lock()
	p.status = PipelineStatus{State: "running", StartedAt: startTime}
	p.lastResult = nil
	p.runMetadata = meta
	p.mu.Unlock()

	if p.eventBus != nil {
		p.eventBus.Clear()
	}

	// Start tracing span
	ctx, span := p.tracer.Start(ctx, "pipeline.convert")
	span.SetAttributes(
		attribute.String("pipeline.title", title),
		attribute.String("pipeline.author", author),
		attribute.Int("pipeline.chapters_count", len(chapters)),
		attribute.String("pipeline.run_id", meta.RunID),
		attribute.String("pipeline.step", "pipeline"),
		attribute.Int("pipeline.progress", 0),
	)
	if meta.ProjectID != "" {
		span.SetAttributes(attribute.String("project.id", meta.ProjectID))
	}
	if meta.JobID != "" {
		span.SetAttributes(attribute.String("generation.job_id", meta.JobID))
	}
	if spanContext := span.SpanContext(); spanContext.IsValid() {
		meta.TraceID = spanContext.TraceID().String()
		p.mu.Lock()
		p.runMetadata = meta
		p.mu.Unlock()
	}
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
	setPipelineStepAttributes(beatSpan, "extract_story_beats", 5)
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
	setPipelineStepAttributes(treeSpan, "aggregate_units", 15)
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventNodeAnalyzing,
		Step:    "aggregate_units",
		Message: "步骤 2/8: 聚合剧情单元...",
	})
	st, treeErr := p.buildTreeFromSentences(ctx, sourceIndex.Sentences)
	if treeErr != nil {
		treeSpan.RecordError(treeErr)
		treeSpan.End()
		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventNodeFailed,
			Step:    "aggregate_units",
			Message: "剧情单元聚合失败，继续使用故事节拍主线生成",
			Error:   treeErr.Error(),
		})
	} else {
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
	}

	p.mu.Lock()
	p.status.CurrentStep = "update_graph"
	p.status.Progress = 40
	p.mu.Unlock()

	stepStart = time.Now()
	ctx, graphSpan := p.tracer.Start(ctx, "pipeline.update_graph")
	setPipelineStepAttributes(graphSpan, "update_graph", 40)
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventGraphUpdating,
		Step:    "update_graph",
		Message: "步骤 4/8: 更新动态图...",
	})
	// Step 7: Update the dynamic graph on the same source-grounded timeline used for scene planning.
	sceneCandidates := scene.BuildSceneCandidates(storyBeats)
	if len(sceneCandidates) == 0 {
		graphSpan.End()
		p.mu.Lock()
		p.status.State = "failed"
		p.status.Error = "no scene candidates produced"
		p.mu.Unlock()
		p.publishEvent(observability.PipelineEvent{
			Type:  observability.EventError,
			Step:  "update_graph",
			Error: "no scene candidates produced",
		})
		return nil, fmt.Errorf("graph update failed: no scene candidates produced")
	}
	graphMgr, err := p.updateGraphFromCandidates(ctx, sourceIndex, sceneCandidates)
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
	if afterSnap := graphMgr.SnapshotsAfter()[sceneCandidates[len(sceneCandidates)-1].ID]; afterSnap != nil {
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
	ctx, planSpan := p.tracer.Start(ctx, "pipeline.plan_scenes")
	setPipelineStepAttributes(planSpan, "plan_scenes", 55)
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
	if err := validatePlanGrounding(plans, sceneCandidates); err != nil {
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
		return nil, fmt.Errorf("scene plan grounding failed: %w", err)
	}
	planSpan.End()
	planSummaries := make([]map[string]interface{}, len(plans))
	for i, plan := range plans {
		planSummaries[i] = map[string]interface{}{
			"id":                   plan.ID,
			"source_candidate_ids": plan.SourceCandidateIDs(),
			"scene_count":          plan.SceneCount,
			"purpose":              plan.Purpose,
			"location":             plan.Location,
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
	setPipelineStepAttributes(writeSpan, "generate_scenes", 65)
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventSceneWriting,
		Step:    "generate_scenes",
		Message: "步骤 6/8: 生成场景...",
	})
	scenes, err := p.generateScenesSequential(ctx, plans, sourceIndex, sceneCandidates, graphMgr)
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
	finalGraph := graphMgr.SnapshotsAfter()[sceneCandidates[len(sceneCandidates)-1].ID]
	genre := []string{"剧情"}
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
		Characters: collectCharactersFromGraph(finalGraph),
		Scenes:     scenes,
	}

	artifacts := generationArtifacts(sourceIndex, storyBeats, plans, finalGraph)

	// Step 12: Chief editor review
	stepStart = time.Now()
	ctx, editorSpan := p.tracer.Start(ctx, "pipeline.editor_review")
	setPipelineStepAttributes(editorSpan, "editor_review", 80)
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventEditorReviewing,
		Step:    "editor_review",
		Message: "步骤 7/8: 总编审查...",
	})
	editResult, err := p.chiefEditor.ReviewAndRevise(ctx, screenplay, artifacts)
	if err != nil {
		editorSpan.RecordError(err)
		if artifacts != nil {
			artifacts.Warnings = append(artifacts.Warnings, schema.GenerationWarning{
				Code:    "editor_review_failed",
				Message: err.Error(),
				Source:  "chief-editor",
			})
		}
		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventEditorReviewed,
			Step:    "editor_review",
			Message: fmt.Sprintf("总编审查失败，保留原剧本: %v", err),
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
	setPipelineStepAttributes(validSpan, "validation", 90)
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventValidation,
		Step:    "validation",
		Message: "步骤 8/8: 校验...",
	})
	validateResult := p.validator.Validate(screenplay, 3)
	if !validateResult.Valid {
		validationErr := fmt.Errorf("screenplay validation failed: %s", strings.Join(validateResult.Errors, "; "))
		validSpan.RecordError(validationErr)
		validSpan.End()
		p.mu.Lock()
		p.status.State = "failed"
		p.status.Error = validationErr.Error()
		p.mu.Unlock()
		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventError,
			Step:    "validation",
			Message: "校验失败",
			Error:   validationErr.Error(),
			Details: map[string]interface{}{
				"errors":   validateResult.Errors,
				"warnings": validateResult.Warnings,
			},
		})
		return nil, validationErr
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
		Artifacts:   artifacts,
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

// updateGraphFromCandidates builds the dynamic graph across source-grounded scene candidates.
// Candidate IDs are the authoritative timeline keys for scene planning and context assembly.
func (p *Pipeline) updateGraphFromCandidates(ctx context.Context, sourceIndex *segment.SourceIndex, candidates []scene.SceneCandidate) (*graph.Manager, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no scene candidates to update graph")
	}
	graphMgr := graph.NewManager()
	graphMgr.SetInitialSnapshot(candidates[0].ID, graph.NewSnapshot(candidates[0].ID))

	failedUpdates := 0
	successfulUpdates := 0
	for i, candidate := range candidates {
		beforeSnap := graphMgr.SnapshotsBefore()[candidate.ID]
		if beforeSnap == nil {
			beforeSnap = graph.NewSnapshot(candidate.ID)
			graphMgr.SetInitialSnapshot(candidate.ID, beforeSnap)
		}

		graphText := sourceIndex.TextForIDs(candidate.SourceSentenceIDs)
		if strings.TrimSpace(graphText) == "" {
			graphText = candidate.Summary
		}

		graphSpanName := fmt.Sprintf("pipeline.graph_update.%s", candidate.ID)
		_, graphNodeSpan := p.tracer.Start(ctx, graphSpanName)
		graphNodeSpan.SetAttributes(
			attribute.String("scene_candidate.id", candidate.ID),
			attribute.Int("scene_candidate.index", i),
		)

		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventGraphUpdating,
			Step:    "graph_update",
			NodeID:  candidate.ID,
			Message: fmt.Sprintf("图更新: %s (%d/%d)", candidate.ID, i+1, len(candidates)),
		})

		update, err := p.graphAnalyzer.AnalyzeUpdate(ctx, graphText, beforeSnap)
		if err != nil {
			failedUpdates++
			graphNodeSpan.RecordError(err)
			graphNodeSpan.End()
			p.publishEvent(observability.PipelineEvent{
				Type:   observability.EventGraphUpdated,
				Step:   "graph_update",
				NodeID: candidate.ID,
				Error:  err.Error(),
			})
			graphMgr.SetAfterSnapshot(candidate.ID, beforeSnap)
			if i+1 < len(candidates) {
				graphMgr.ChainSnapshot(candidate.ID, candidates[i+1].ID)
			}
			continue
		}

		graphMgr.ApplyUpdate(candidate.ID, update)
		successfulUpdates++
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
			NodeID:  candidate.ID,
			Message: fmt.Sprintf("图更新完成: %s", candidate.ID),
			Details: map[string]interface{}{
				"new_characters": newCharNames,
				"new_relations":  newRelDescs,
			},
		})

		if i+1 < len(candidates) {
			graphMgr.ChainSnapshot(candidate.ID, candidates[i+1].ID)
		}
	}

	if successfulUpdates == 0 {
		return nil, fmt.Errorf("graph update failed for all %d scene candidates", len(candidates))
	}
	if finalSnap := graphMgr.SnapshotsAfter()[candidates[len(candidates)-1].ID]; finalSnap == nil || len(finalSnap.Characters) == 0 {
		return nil, fmt.Errorf("graph update produced no characters after %d candidates (%d failed)", len(candidates), failedUpdates)
	}

	return graphMgr, nil
}

// generateScenesSequential writes scenes in plan order using source-grounded graph snapshots.
func (p *Pipeline) generateScenesSequential(ctx context.Context, plans []*scene.ScenePlan, sourceIndex *segment.SourceIndex, candidates []scene.SceneCandidate, graphMgr *graph.Manager) ([]schema.Scene, error) {
	candidateByID := make(map[string]scene.SceneCandidate, len(candidates))
	for _, candidate := range candidates {
		candidateByID[candidate.ID] = candidate
	}

	ctxBuilder := scene.NewContextBuilder(graphMgr.SnapshotsBefore(), graphMgr.SnapshotsAfter())
	result := make([]schema.Scene, 0, len(plans))

	for _, plan := range plans {
		if plan == nil || plan.SceneCount == 0 {
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
		sceneSpan.End()

		result = append(result, *sc)
		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventSceneWritten,
			Step:    "write_scene",
			NodeID:  plan.ID,
			Message: fmt.Sprintf("场景 %d 完成", sc.Sequence),
			Details: map[string]interface{}{
				"source_candidate_ids": plan.SourceCandidateIDs(),
			},
		})
	}
	return result, nil
}

func validatePlanGrounding(plans []*scene.ScenePlan, candidates []scene.SceneCandidate) error {
	if len(candidates) == 0 {
		return fmt.Errorf("no scene candidates available for planning")
	}
	candidateIDs := make(map[string]struct{}, len(candidates))
	covered := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidateIDs[candidate.ID] = struct{}{}
	}
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		if len(plan.SourceCandidateIDs()) == 0 {
			return fmt.Errorf("plan %q has no source candidate IDs", plan.ID)
		}
		for _, id := range plan.SourceCandidateIDs() {
			if _, ok := candidateIDs[id]; !ok {
				return fmt.Errorf("plan %q references unknown source candidate %q", plan.ID, id)
			}
			covered[id] = struct{}{}
		}
	}
	for _, candidate := range candidates {
		if _, ok := covered[candidate.ID]; !ok {
			return fmt.Errorf("source candidate %q is not covered by any scene plan", candidate.ID)
		}
	}
	return nil
}

func sourceContextForPlan(sourceIndex *segment.SourceIndex, candidates map[string]scene.SceneCandidate, plan *scene.ScenePlan) (string, string) {
	var sentenceIDs []string
	summaries := make([]string, 0, len(plan.SourceCandidateIDs()))
	for _, sourceID := range plan.SourceCandidateIDs() {
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

func collectCharactersFromGraph(snap *graph.GraphSnapshot) []schema.Character {
	if snap == nil || len(snap.Characters) == 0 {
		return nil
	}
	result := make([]schema.Character, 0, len(snap.Characters))
	for _, state := range snap.Characters {
		if state == nil || strings.TrimSpace(state.ID) == "" {
			continue
		}
		name := strings.TrimSpace(state.Name)
		if name == "" {
			name = state.ID
		}
		result = append(result, schema.Character{
			ID:          state.ID,
			Name:        name,
			Intro:       strings.TrimSpace(state.CurrentGoal),
			Personality: append([]string(nil), state.Personality...),
		})
	}
	return result
}

func generationArtifacts(sourceIndex *segment.SourceIndex, storyBeats []segment.StoryBeat, plans []*scene.ScenePlan, finalGraph *graph.GraphSnapshot) *schema.GenerationArtifacts {
	artifacts := &schema.GenerationArtifacts{
		SourceIndex:   schemaSourceIndex(sourceIndex),
		StoryBeats:    schemaStoryBeats(sourceIndex, storyBeats),
		ScenePlan:     schemaScenePlan(plans, sourceIndex, storyBeats),
		GraphSnapshot: finalGraph,
		Warnings:      generationWarnings(storyBeats),
	}
	if artifacts.SourceIndex == nil && len(artifacts.StoryBeats) == 0 && artifacts.ScenePlan == nil && artifacts.GraphSnapshot == nil {
		return nil
	}
	return artifacts
}

func generationWarnings(storyBeats []segment.StoryBeat) []schema.GenerationWarning {
	warnings := make([]schema.GenerationWarning, 0)
	for _, beat := range storyBeats {
		if strings.Contains(beat.BoundaryReason, "回退") || strings.Contains(beat.BoundaryReason, "补齐") {
			warnings = append(warnings, schema.GenerationWarning{
				Code:    "story_beat_fallback",
				Message: beat.BoundaryReason,
				Source:  beat.ID,
			})
		}
	}
	return warnings
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

func schemaScenePlan(plans []*scene.ScenePlan, sourceIndex *segment.SourceIndex, storyBeats []segment.StoryBeat) *schema.ScenePlan {
	if len(plans) == 0 {
		return nil
	}
	beatIDsByCandidate := sourceBeatIDsByCandidate(storyBeats)
	root := &schema.ScenePlan{ID: "scene_plan", Scenes: make([]schema.ScenePlan, 0, len(plans))}
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		root.Scenes = append(root.Scenes, schema.ScenePlan{
			ID:            plan.ID,
			Sequence:      plan.Sequence,
			Purpose:       plan.Purpose,
			SourceBeatIDs: sourceBeatIDsForPlan(plan, beatIDsByCandidate),
			SourceRefs:    sourceRefsForPlan(plan, sourceIndex, storyBeats),
			Location:      plan.Location,
			TimeFrame:     plan.TimeFrame,
			Characters:    plan.Characters,
			KeyPlotPoints: plan.KeyPlotPoints,
		})
	}
	if len(root.Scenes) == 0 {
		return nil
	}
	return root
}

func sourceBeatIDsByCandidate(storyBeats []segment.StoryBeat) map[string][]string {
	out := make(map[string][]string, len(storyBeats))
	for i, beat := range storyBeats {
		candidateID := fmt.Sprintf("candidate_%03d", i+1)
		out[candidateID] = []string{beat.ID}
	}
	return out
}

func sourceBeatIDsForPlan(plan *scene.ScenePlan, beatIDsByCandidate map[string][]string) []string {
	out := make([]string, 0, len(plan.SourceCandidateIDs()))
	for _, candidateID := range plan.SourceCandidateIDs() {
		out = append(out, beatIDsByCandidate[candidateID]...)
	}
	return out
}

func sourceRefsForPlan(plan *scene.ScenePlan, idx *segment.SourceIndex, storyBeats []segment.StoryBeat) []schema.SourceSentenceRef {
	if idx == nil {
		return nil
	}
	beatByCandidate := make(map[string]segment.StoryBeat, len(storyBeats))
	for i, beat := range storyBeats {
		beatByCandidate[fmt.Sprintf("candidate_%03d", i+1)] = beat
	}
	out := make([]schema.SourceSentenceRef, 0)
	for _, candidateID := range plan.SourceCandidateIDs() {
		beat, ok := beatByCandidate[candidateID]
		if !ok {
			continue
		}
		out = append(out, schemaSourceRefs(idx, beat.SourceSentenceIDs)...)
	}
	return out
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
