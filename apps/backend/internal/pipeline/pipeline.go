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
	state := &RunState{
		Title:     strings.TrimSpace(title),
		Author:    strings.TrimSpace(author),
		Chapters:  chapters,
		StartTime: time.Now(),
	}
	if state.Title == "" {
		state.Title = "未命名小说"
	}
	if state.Author == "" {
		state.Author = "未知作者"
	}
	profile, hasProfile := schema.AdaptationProfileFromContext(ctx)
	meta, _ := RunMetadataFromContext(ctx)
	if meta.RunID == "" {
		meta.RunID = uuid.NewString()
	}
	p.mu.Lock()
	p.status = PipelineStatus{State: "running", StartedAt: state.StartTime}
	p.lastResult = nil
	p.runMetadata = meta
	p.mu.Unlock()

	if p.eventBus != nil {
		p.eventBus.Clear()
	}

	// Start tracing span
	ctx, span := p.tracer.Start(ctx, "pipeline.convert")
	span.SetAttributes(
		attribute.String("pipeline.title", state.Title),
		attribute.String("pipeline.author", state.Author),
		attribute.Int("pipeline.chapters_count", len(state.Chapters)),
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
		Message: fmt.Sprintf("开始转换: %s (共 %d 章)", state.Title, len(state.Chapters)),
		Details: map[string]interface{}{
			"title":                  state.Title,
			"author":                 state.Author,
			"chapters":               len(state.Chapters),
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
		Message: "步骤 1/6: 提取故事节拍...",
	})
	state.SourceIndex = segment.BuildSourceIndex(state.Chapters)
	if len(state.SourceIndex.Sentences) == 0 {
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
	var err error
	state.StoryBeats, err = p.storyBeatExtractor.ExtractWithConcurrency(ctx, state.SourceIndex, p.config.MaxConcurrency)
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
		Message:  fmt.Sprintf("故事节拍提取完成: %d 个节拍", len(state.StoryBeats)),
		Duration: durationPtr(time.Since(stepStart)),
		Details: map[string]interface{}{
			"source_index": state.SourceIndex,
			"story_beats":  state.StoryBeats,
			"beats_count":  len(state.StoryBeats),
		},
	})

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
		Message: "步骤 2/6: 更新动态图...",
	})
	// Step 2: Update the dynamic graph on the same source-grounded timeline used for scene planning.
	state.SceneCandidates = scene.BuildSceneCandidates(state.StoryBeats)
	if len(state.SceneCandidates) == 0 {
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
	state.GraphMgr, err = p.updateGraphFromCandidates(ctx, state.SourceIndex, state.SceneCandidates)
	if err != nil {
		graphSpan.RecordError(err)
		state.GraphMgr = emptyGraphManager(state.SceneCandidates)
		if state.Artifacts == nil {
			state.Artifacts = &schema.GenerationArtifacts{}
		}
		state.Artifacts.Warnings = append(state.Artifacts.Warnings, schema.GenerationWarning{
			Code:    "source_graph_failed",
			Message: err.Error(),
			Source:  "graph-analyzer",
		})
		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventGraphUpdated,
			Step:    "update_graph",
			Message: "源文本图谱更新失败，降级为空图继续生成",
			Error:   err.Error(),
		})
	}
	graphSpan.End()
	graphChars := 0
	if afterSnap := state.finalGraph(); afterSnap != nil {
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
		Message: fmt.Sprintf("步骤 3/6: 规划场景 (%d 故事节拍)...", len(state.SceneCandidates)),
	})
	state.Plans, err = p.scenePlanner.PlanFromCandidates(ctx, state.SceneCandidates)
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
	if err := validatePlanGrounding(state.Plans, state.SceneCandidates); err != nil {
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
	planSummaries := make([]map[string]interface{}, len(state.Plans))
	for i, plan := range state.Plans {
		planSummaries[i] = map[string]interface{}{
			"id":                   plan.ID,
			"source_candidate_ids": plan.CandidateIDs(),
			"scene_count":          plan.SceneCount,
			"purpose":              plan.Purpose,
			"location":             plan.Location,
		}
	}
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventScenePlanned,
		Step:     "plan_scenes",
		Message:  fmt.Sprintf("场景规划完成: %d 场景计划", len(state.Plans)),
		Duration: durationPtr(time.Since(stepStart)),
		Details: map[string]interface{}{
			"plans_count": len(state.Plans),
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
		Message: "步骤 4/6: 生成场景...",
	})
	state.Scenes, err = p.generateScenesSequential(ctx, state.Plans, state.SourceIndex, state.SceneCandidates, state.GraphMgr)
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
	sceneHeadings := make([]string, 0, len(state.Scenes))
	for _, sc := range state.Scenes {
		sceneHeadings = append(sceneHeadings, sc.Heading)
	}
	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventSceneWritten,
		Step:     "generate_scenes",
		Message:  fmt.Sprintf("场景生成完成: %d 场景", len(state.Scenes)),
		Duration: durationPtr(time.Since(stepStart)),
		Details: map[string]interface{}{
			"scenes_count": len(state.Scenes),
			"scenes":       sceneHeadings,
		},
	})

	p.mu.Lock()
	p.status.CurrentStep = "editor_review"
	p.status.Progress = 80
	p.mu.Unlock()

	// Assemble screenplay
	finalGraph := state.finalGraph()
	genre := []string{"剧情"}
	sourceChapters := make([]int, len(state.Chapters))
	for i := range state.Chapters {
		sourceChapters[i] = i + 1
	}

	state.Screenplay = &schema.Screenplay{
		Metadata: schema.Metadata{
			Title:          state.Title,
			Author:         state.Author,
			Version:        "1.0",
			CreatedAt:      time.Now().Format(time.RFC3339),
			OriginalNovel:  state.Title,
			Genre:          genre,
			SourceChapters: sourceChapters,
		},
		Characters: collectCharactersFromGraph(finalGraph),
		Scenes:     state.Scenes,
	}

	var priorWarnings []schema.GenerationWarning
	if state.Artifacts != nil {
		priorWarnings = append([]schema.GenerationWarning(nil), state.Artifacts.Warnings...)
	}
	state.Artifacts = generationArtifacts(state.SourceIndex, state.StoryBeats, state.SceneCandidates, state.Plans, finalGraph)
	if len(priorWarnings) > 0 {
		if state.Artifacts == nil {
			state.Artifacts = &schema.GenerationArtifacts{}
		}
		state.Artifacts.Warnings = append(priorWarnings, state.Artifacts.Warnings...)
	}

	// Step 12: Chief editor review
	stepStart = time.Now()
	ctx, editorSpan := p.tracer.Start(ctx, "pipeline.editor_review")
	setPipelineStepAttributes(editorSpan, "editor_review", 80)
	p.publishEvent(observability.PipelineEvent{
		Type:    observability.EventEditorReviewing,
		Step:    "editor_review",
		Message: "步骤 5/6: 总编审查...",
	})
	editResult, err := p.chiefEditor.ReviewAndRevise(ctx, state.Screenplay, state.Artifacts)
	if err != nil {
		editorSpan.RecordError(err)
		if state.Artifacts != nil {
			state.Artifacts.Warnings = append(state.Artifacts.Warnings, schema.GenerationWarning{
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
		Message: "步骤 6/6: 校验...",
	})
	validateResult := p.validator.Validate(state.Screenplay, requiredSourceChapterCoverage(len(state.Chapters)))
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

	state.Duration = time.Since(state.StartTime)
	yamlBytes, _ := yaml.Marshal(state.Screenplay)
	state.YAMLStr = string(yamlBytes)
	state.CompletedAt = time.Now()

	p.mu.Lock()
	p.lastResult = state.result()
	p.status.State = "completed"
	p.mu.Unlock()

	p.publishEvent(observability.PipelineEvent{
		Type:     observability.EventPipelineEnd,
		Step:     "pipeline",
		Message:  fmt.Sprintf("转换完成! 耗时: %v", state.Duration),
		Duration: &state.Duration,
		Details: map[string]interface{}{
			"scenes_count":     len(state.Screenplay.Scenes),
			"characters_count": len(state.Screenplay.Characters),
			"screenplay":       state.Screenplay,
			"yaml":             state.YAMLStr,
		},
	})

	return state.Screenplay, nil
}

func requiredSourceChapterCoverage(chapterCount int) int {
	if chapterCount < 3 {
		return chapterCount
	}
	return 3
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

type graphUpdateExtraction struct {
	candidate scene.SceneCandidate
	text      string
	update    *graph.GraphUpdateResult
	err       error
}

// updateGraphFromCandidates builds the dynamic graph across source-grounded scene candidates.
// Candidate IDs are the authoritative timeline keys for scene planning and context assembly.
func (p *Pipeline) updateGraphFromCandidates(ctx context.Context, sourceIndex *segment.SourceIndex, candidates []scene.SceneCandidate) (*graph.Manager, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no scene candidates to update graph")
	}
	extractions := make([]graphUpdateExtraction, len(candidates))
	for i, candidate := range candidates {
		graphText := sourceIndex.TextForIDs(candidate.SourceSentenceIDs)
		if strings.TrimSpace(graphText) == "" {
			graphText = candidate.Summary
		}
		extractions[i] = graphUpdateExtraction{candidate: candidate, text: graphText}
	}

	concurrency := p.config.MaxConcurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(extractions) {
		concurrency = len(extractions)
	}
	jobs := make(chan int)
	var wg sync.WaitGroup
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				extractions[i].update, extractions[i].err = p.graphAnalyzer.ExtractUpdateInstructions(ctx, extractions[i].text)
			}
		}()
	}
	for i := range extractions {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	graphMgr := graph.NewManager()
	graphMgr.SetInitialSnapshot(candidates[0].ID, graph.NewSnapshot(candidates[0].ID))

	failedUpdates := 0
	successfulUpdates := 0
	for i := range extractions {
		extraction := &extractions[i]
		candidate := extraction.candidate
		beforeSnap := graphMgr.SnapshotsBefore()[candidate.ID]
		if beforeSnap == nil {
			beforeSnap = graph.NewSnapshot(candidate.ID)
			graphMgr.SetInitialSnapshot(candidate.ID, beforeSnap)
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

		if extraction.err != nil {
			failedUpdates++
			graphNodeSpan.RecordError(extraction.err)
			graphNodeSpan.End()
			p.publishEvent(observability.PipelineEvent{
				Type:   observability.EventGraphUpdated,
				Step:   "graph_update",
				NodeID: candidate.ID,
				Error:  extraction.err.Error(),
			})
			graphMgr.SetAfterSnapshot(candidate.ID, beforeSnap)
			if i+1 < len(candidates) {
				graphMgr.ChainSnapshot(candidate.ID, candidates[i+1].ID)
			}
			continue
		}

		applied := graphMgr.ApplyUpdate(candidate.ID, extraction.update)
		successfulUpdates++
		graphNodeSpan.End()

		newCharNames := graphUpdateNewCharacterNames(beforeSnap, applied)
		newRelDescs := make([]string, len(extraction.update.RelationChanges))
		for j, rc := range extraction.update.RelationChanges {
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
		return nil, fmt.Errorf("source graph update failed for all %d scene candidates", len(candidates))
	}
	if finalSnap := graphMgr.SnapshotsAfter()[candidates[len(candidates)-1].ID]; finalSnap == nil || len(finalSnap.Characters) == 0 {
		return nil, fmt.Errorf("source graph update produced no characters after %d candidates (%d failed)", len(candidates), failedUpdates)
	}

	return graphMgr, nil
}


func graphUpdateNewCharacterNames(beforeSnap, afterSnap *graph.GraphSnapshot) []string {
	if afterSnap == nil {
		return nil
	}
	names := make([]string, 0, len(afterSnap.Characters))
	for id, character := range afterSnap.Characters {
		if beforeSnap != nil {
			if _, exists := beforeSnap.Characters[id]; exists {
				continue
			}
		}
		names = append(names, character.Name)
	}
	return names
}

func emptyGraphManager(candidates []scene.SceneCandidate) *graph.Manager {
	graphMgr := graph.NewManager()
	if len(candidates) == 0 {
		return graphMgr
	}
	graphMgr.SetInitialSnapshot(candidates[0].ID, graph.NewSnapshot(candidates[0].ID))
	for i, candidate := range candidates {
		beforeSnap := graphMgr.SnapshotsBefore()[candidate.ID]
		if beforeSnap == nil {
			beforeSnap = graph.NewSnapshot(candidate.ID)
			graphMgr.SetInitialSnapshot(candidate.ID, beforeSnap)
		}
		graphMgr.SetAfterSnapshot(candidate.ID, beforeSnap)
		if i+1 < len(candidates) {
			graphMgr.ChainSnapshot(candidate.ID, candidates[i+1].ID)
		}
	}
	return graphMgr
}

// generateScenesSequential writes scenes in plan order with concurrent WriteScene calls.
func (p *Pipeline) generateScenesSequential(ctx context.Context, plans []*scene.ScenePlan, sourceIndex *segment.SourceIndex, candidates []scene.SceneCandidate, graphMgr *graph.Manager) ([]schema.Scene, error) {
	candidateByID := make(map[string]scene.SceneCandidate, len(candidates))
	for _, candidate := range candidates {
		candidateByID[candidate.ID] = candidate
	}

	ctxBuilder := scene.NewContextBuilder(graphMgr.SnapshotsBefore(), graphMgr.SnapshotsAfter())

	// Pre-build source context per plan (cheap, no LLM)
	type planJob struct {
		plan          *scene.ScenePlan
		sourceText    string
		sourceSummary string
	}
	jobs := make([]planJob, 0, len(plans))
	for _, plan := range plans {
		if plan == nil || plan.SceneCount == 0 {
			continue
		}
		sourceText, sourceSummary := sourceContextForPlan(sourceIndex, candidateByID, plan)
		jobs = append(jobs, planJob{plan: plan, sourceText: sourceText, sourceSummary: sourceSummary})
	}

	if len(jobs) == 0 {
		return nil, nil
	}
	// Emit writing events before dispatching
	for _, job := range jobs {
		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventSceneWriting,
			Step:    "write_scene",
			NodeID:  job.plan.ID,
			Message: fmt.Sprintf("写场景 (计划 %s)...", job.plan.ID),
		})
	}

	// Concurrently write scenes
	type sceneResult struct {
		index int
		scene *schema.Scene
		err   error
	}
	results := make([]sceneResult, len(jobs))

	concurrency := p.config.MaxConcurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > len(jobs) {
		concurrency = len(jobs)
	}
	jobIdx := make(chan int, len(jobs))
	var wg sync.WaitGroup
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobIdx {
				job := jobs[i]
				sceneCtx := ctxBuilder.Build(job.plan, job.sourceText, job.sourceSummary)

				sceneSpanName := fmt.Sprintf("pipeline.write_scene.%s", job.plan.ID)
				_, sceneSpan := p.tracer.Start(ctx, sceneSpanName)
				sceneSpan.SetAttributes(
					attribute.String("scene.plan_id", job.plan.ID),
					attribute.Int("scene.job_index", i),
				)

				sc, err := p.sceneWriter.WriteScene(ctx, sceneCtx)
				if err != nil {
					sceneSpan.RecordError(err)
					sceneSpan.End()
					results[i] = sceneResult{index: i, err: err}
					continue
				}
				sceneSpan.End()
				results[i] = sceneResult{index: i, scene: sc}
			}
		}()
	}
	for i := range jobs {
		jobIdx <- i
	}
	close(jobIdx)
	wg.Wait()

	// Collect in order, assign sequence numbers
	out := make([]schema.Scene, 0, len(jobs))
	for i, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("scene %d (plan %s): %w", i+1, jobs[i].plan.ID, r.err)
		}
		sc := r.scene
		sc.Sequence = len(out) + 1
		if sc.ID == "" {
			sc.ID = fmt.Sprintf("scene_%03d", sc.Sequence)
		}

		out = append(out, *sc)
		p.publishEvent(observability.PipelineEvent{
			Type:    observability.EventSceneWritten,
			Step:    "write_scene",
			NodeID:  jobs[i].plan.ID,
			Message: fmt.Sprintf("场景 %d 完成", sc.Sequence),
			Details: map[string]interface{}{
				"source_candidate_ids": jobs[i].plan.CandidateIDs(),
			},
		})
	}
	return out, nil
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
		if len(plan.CandidateIDs()) == 0 {
			return fmt.Errorf("plan %q has no source candidate IDs", plan.ID)
		}
		for _, id := range plan.CandidateIDs() {
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
	summaries := make([]string, 0, len(plan.CandidateIDs()))
	for _, sourceID := range plan.CandidateIDs() {
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

func generationArtifacts(sourceIndex *segment.SourceIndex, storyBeats []segment.StoryBeat, sceneCandidates []scene.SceneCandidate, plans []*scene.ScenePlan, finalGraph *graph.GraphSnapshot) *schema.GenerationArtifacts {
	artifacts := &schema.GenerationArtifacts{
		SourceIndex:   schemaSourceIndex(sourceIndex),
		StoryBeats:    schemaStoryBeats(sourceIndex, storyBeats),
		ScenePlan:     schemaScenePlan(plans, sourceIndex, sceneCandidates),
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

func schemaScenePlan(plans []*scene.ScenePlan, sourceIndex *segment.SourceIndex, sceneCandidates []scene.SceneCandidate) *schema.ScenePlan {
	if len(plans) == 0 {
		return nil
	}
	candidatesByID := sceneCandidatesByID(sceneCandidates)
	root := &schema.ScenePlan{ID: "scene_plan", Scenes: make([]schema.ScenePlan, 0, len(plans))}
	for _, plan := range plans {
		if plan == nil {
			continue
		}
		root.Scenes = append(root.Scenes, schema.ScenePlan{
			ID:            plan.ID,
			Sequence:      plan.Sequence,
			Purpose:       plan.Purpose,
			SourceBeatIDs: sourceBeatIDsForPlan(plan, candidatesByID),
			SourceRefs:    sourceRefsForPlan(plan, sourceIndex, candidatesByID),
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


func sourceBeatIDsForPlan(plan *scene.ScenePlan, candidatesByID map[string]scene.SceneCandidate) []string {
	out := make([]string, 0, len(plan.CandidateIDs()))
	for _, candidateID := range plan.CandidateIDs() {
		if candidate, ok := candidatesByID[candidateID]; ok {
			out = append(out, candidate.SourceBeatIDs...)
		}
	}
	return out
}

func sourceRefsForPlan(plan *scene.ScenePlan, idx *segment.SourceIndex, candidatesByID map[string]scene.SceneCandidate) []schema.SourceSentenceRef {
	if idx == nil {
		return nil
	}
	out := make([]schema.SourceSentenceRef, 0)
	for _, candidateID := range plan.CandidateIDs() {
		candidate, ok := candidatesByID[candidateID]
		if !ok {
			continue
		}
		out = append(out, schemaSourceRefs(idx, candidate.SourceSentenceIDs)...)
	}
	return out
}

func sceneCandidatesByID(candidates []scene.SceneCandidate) map[string]scene.SceneCandidate {
	out := make(map[string]scene.SceneCandidate, len(candidates))
	for _, candidate := range candidates {
		out[candidate.ID] = candidate
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
