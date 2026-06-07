package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fabula-studio/backend/internal/config"
	"github.com/fabula-studio/backend/internal/db/sqlc"
	"github.com/fabula-studio/backend/internal/observability"
	"github.com/fabula-studio/backend/internal/pipeline"
	"github.com/fabula-studio/backend/internal/repo"
	"github.com/fabula-studio/backend/internal/schema"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type generationServiceError struct {
	Status  int
	Code    int
	Message string
}

type GenerationService struct {
	config   config.Config
	eventBus *observability.EventBus
	store    *repo.Store
}

func NewGenerationService(cfg config.Config, eventBus *observability.EventBus, store *repo.Store) *GenerationService {
	return &GenerationService{config: cfg, eventBus: eventBus, store: store}
}

func (g *GenerationService) Status(ctx context.Context, userID, projectID string) (generationStatusDTO, *generationServiceError) {
	p, err := g.store.Projects.ByIDForUser(ctx, projectID, userID)
	if err != nil {
		return generationStatusDTO{}, &generationServiceError{Status: 404, Code: codeProjectMiss, Message: "项目不存在"}
	}
	job, err := g.store.GenerationJobs.GetLatestGenerationJobByProjectID(ctx, projectID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return generationStatusDTO{}, &generationServiceError{Status: 500, Code: codeDB, Message: "数据库操作失败"}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		progress := 0
		step := "未开始"
		if p.Status == "completed" {
			progress = 100
			step = "已完成"
		}
		if p.Status == "failed" {
			step = "生成失败"
		}
		return generationStatusDTO{ProjectID: projectID, ProjectStatus: p.Status, Status: p.Status, Progress: progress, CurrentStep: step, ErrorMessage: textPtr(p.ErrorMessage)}, nil
	}
	dto := generationJobToDTO(job)
	status := job.Status
	if status == "queued" || status == "running" {
		status = "generating"
	}
	return generationStatusDTO{ProjectID: projectID, JobID: job.ID, ProjectStatus: p.Status, Status: status, Progress: int(job.Progress), CurrentStep: job.CurrentStep, ErrorMessage: textPtr(job.ErrorMessage), Artifacts: dto.Artifacts, Job: &dto}, nil
}

func (g *GenerationService) Start(ctx context.Context, userID, projectID string) (generationResponse, string, *generationServiceError) {
	p, err := g.store.Projects.ByIDForUser(ctx, projectID, userID)
	if err != nil {
		return generationResponse{}, "", &generationServiceError{Status: 404, Code: codeProjectMiss, Message: "项目不存在"}
	}
	if strings.TrimSpace(p.SourceText) == "" {
		return generationResponse{}, "", &generationServiceError{Status: 400, Code: codeNoSource, Message: "项目缺少小说文本"}
	}
	u, err := g.store.Users.ByID(ctx, userID)
	if err != nil {
		return generationResponse{}, "", &generationServiceError{Status: 401, Code: codeUnauth, Message: "未登录或登录已过期"}
	}
	if u.AiPoints < generationCost {
		return generationResponse{}, "", &generationServiceError{Status: 400, Code: codeNoPoints, Message: "AI 点数不足"}
	}
	job, err := g.store.GenerationJobs.GetActiveGenerationJobByProjectID(ctx, projectID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return generationResponse{}, "", &generationServiceError{Status: 500, Code: codeDB, Message: "数据库操作失败"}
	}
	if err == nil {
		return generationResponseFromJob(projectID, job), "剧本生成任务已在运行", nil
	}
	job, err = g.store.GenerationJobs.CreateGenerationJob(ctx, repo.CreateGenerationJobParams{ID: uuid.NewString(), ProjectID: projectID, CurrentStep: "queued"})
	if err != nil {
		if active, activeErr := g.store.GenerationJobs.GetActiveGenerationJobByProjectID(ctx, projectID); activeErr == nil {
			return generationResponseFromJob(projectID, active), "剧本生成任务已在运行", nil
		}
		return generationResponse{}, "", &generationServiceError{Status: 500, Code: codeDB, Message: "数据库操作失败"}
	}
	if _, err := g.store.Queries.UpdateProjectStatus(ctx, sqlc.UpdateProjectStatusParams{ID: projectID, UserID: userID, Status: "generating"}); err != nil {
		_, _ = g.store.GenerationJobs.FailGenerationJob(ctx, repo.FailGenerationJobParams{ID: job.ID, CurrentStep: "start", ErrorMessage: err.Error(), Artifacts: nil})
		return generationResponse{}, "", &generationServiceError{Status: 500, Code: codeDB, Message: "数据库操作失败"}
	}
	g.runGenerationJob(job.ID, projectID, userID)
	return generationResponseFromJob(projectID, job), "剧本生成任务已启动", nil
}

func generationResponseFromJob(projectID string, job repo.GenerationJob) generationResponse {
	dto := generationJobToDTO(job)
	return generationResponse{ProjectID: projectID, JobID: job.ID, Status: "generating", Progress: int(job.Progress), CurrentStep: job.CurrentStep, CostPoints: generationCost, Scenes: []sceneDTO{}, Artifacts: dto.Artifacts, Job: &dto}
}

func (g *GenerationService) runGenerationJob(jobID, projectID, userID string) {
	go func() {
		ctx := context.Background()
		if _, err := g.store.GenerationJobs.UpdateGenerationJobRunning(ctx, repo.UpdateGenerationJobRunningParams{ID: jobID, Progress: 1, CurrentStep: "running"}); err != nil {
			log.Printf("generation job %s: mark running failed: %v", jobID, err)
			return
		}
		p, err := g.store.Projects.ByIDForUser(ctx, projectID, userID)
		if err != nil {
			g.failGenerationJob(ctx, jobID, projectID, userID, "load_project", err.Error(), nil)
			return
		}
		u, err := g.store.Users.ByID(ctx, userID)
		if err != nil {
			g.failGenerationJob(ctx, jobID, projectID, userID, "load_user", err.Error(), nil)
			return
		}
		if u.AiPoints < generationCost {
			g.failGenerationJob(ctx, jobID, projectID, userID, "check_points", "AI points are insufficient", nil)
			return
		}

		jobPipeline := pipeline.New(pipelineConfigFromAppConfig(g.config), g.config.ModelName, g.config.APIKey, g.config.BaseURL, g.eventBus)
		profile := adaptationProfileFromConfigJSON(p.ConfigJson)
		convertCtx := schema.WithAdaptationProfile(ctx, &profile)
		convertCtx = pipeline.WithRunMetadata(convertCtx, pipeline.RunMetadata{ProjectID: projectID, JobID: jobID})
		done := make(chan struct{})
		go g.persistGenerationProgress(ctx, jobID, jobPipeline, done)
		sp, err := jobPipeline.Convert(convertCtx, p.Title, u.Nickname, splitChapters(p.SourceText))
		close(done)
		artifacts := generationArtifactsFromPipeline(jobPipeline.Result())
		if err != nil {
			g.failGenerationJob(ctx, jobID, projectID, userID, jobPipeline.Status().CurrentStep, err.Error(), artifacts)
			return
		}
		if sp == nil || len(sp.Scenes) == 0 {
			g.failGenerationJob(ctx, jobID, projectID, userID, "empty_result", "empty generation result", artifacts)
			return
		}
		if err := g.commitGenerationResult(ctx, jobID, projectID, userID, sp, artifacts); err != nil {
			g.failGenerationJob(ctx, jobID, projectID, userID, "commit_result", err.Error(), artifacts)
		}
	}()
}

func (g *GenerationService) persistGenerationProgress(ctx context.Context, jobID string, jobPipeline *pipeline.Pipeline, done <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			status := jobPipeline.Status()
			if status.State == "completed" {
				_, _ = g.store.GenerationJobs.UpdateGenerationJobProgress(ctx, repo.UpdateGenerationJobProgressParams{ID: jobID, Status: "running", Progress: 99, CurrentStep: "commit_result", ErrorMessage: pgtype.Text{}, Artifacts: generationArtifactsBytes(generationArtifactsFromPipeline(jobPipeline.Result()))})
				return
			}
			_, _ = g.store.GenerationJobs.UpdateGenerationJobProgress(ctx, repo.UpdateGenerationJobProgressParams{ID: jobID, Status: generationJobStatus(status.State), Progress: int32(status.Progress), CurrentStep: status.CurrentStep, ErrorMessage: textValue(status.Error), Artifacts: generationArtifactsBytes(generationArtifactsFromPipeline(jobPipeline.Result()))})
			return
		case <-ticker.C:
			status := jobPipeline.Status()
			if status.State == "idle" {
				continue
			}
			_, _ = g.store.GenerationJobs.UpdateGenerationJobProgress(ctx, repo.UpdateGenerationJobProgressParams{ID: jobID, Status: generationJobStatus(status.State), Progress: int32(status.Progress), CurrentStep: status.CurrentStep, ErrorMessage: textValue(status.Error), Artifacts: nil})
		}
	}
}

func generationJobStatus(state string) string {
	if state == "failed" || state == "completed" || state == "running" {
		return state
	}
	return "running"
}

func (g *GenerationService) commitGenerationResult(ctx context.Context, jobID, projectID, userID string, sp *schema.Screenplay, artifacts *schema.GenerationArtifacts) error {
	var remaining int32
	if err := g.store.WithTx(ctx, func(q *sqlc.Queries) error {
		updatedUser, err := q.DecrementUserAIPoints(ctx, sqlc.DecrementUserAIPointsParams{ID: userID, AiPoints: generationCost})
		if err != nil {
			return err
		}
		remaining = updatedUser.AiPoints
		if err := q.DeleteScenesByProjectID(ctx, projectID); err != nil {
			return err
		}
		sourceMap := buildSceneSourceMap(artifacts)
		for _, scene := range sp.Scenes {
			if _, err := q.CreateScene(ctx, screenplaySceneToCreate(projectID, scene, sourceMap[scene.Sequence])); err != nil {
				return err
			}
		}
		_, err = q.UpdateProjectStatus(ctx, sqlc.UpdateProjectStatusParams{ID: projectID, UserID: userID, Status: "completed"})
		return err
	}); err != nil {
		return err
	}
	if _, err := g.store.GenerationJobs.CompleteGenerationJob(ctx, repo.CompleteGenerationJobParams{ID: jobID, CurrentStep: "completed", Artifacts: generationArtifactsBytes(artifacts)}); err != nil {
		_, _ = g.store.GenerationJobs.FailGenerationJob(ctx, repo.FailGenerationJobParams{ID: jobID, CurrentStep: "completed", ErrorMessage: err.Error(), Artifacts: generationArtifactsBytes(artifacts)})
		log.Printf("generation job %s completed project %s but failed to mark job completed: %v", jobID, projectID, err)
		return nil
	}
	log.Printf("generation job %s completed for project %s, remaining points: %d", jobID, projectID, remaining)
	return nil
}

func (g *GenerationService) failGenerationJob(ctx context.Context, jobID, projectID, userID, step, msg string, artifacts *schema.GenerationArtifacts) {
	if strings.TrimSpace(step) == "" {
		step = "failed"
	}
	_, _ = g.store.GenerationJobs.FailGenerationJob(ctx, repo.FailGenerationJobParams{ID: jobID, CurrentStep: step, ErrorMessage: msg, Artifacts: generationArtifactsBytes(artifacts)})
	g.markGenerationFailed(ctx, projectID, userID, msg)
}

func (g *GenerationService) markGenerationFailed(ctx context.Context, projectID, userID, msg string) {
	_, _ = g.store.Queries.UpdateProjectStatus(ctx, sqlc.UpdateProjectStatusParams{ID: projectID, UserID: userID, Status: "failed", ErrorMessage: textValue(msg)})
}

// buildSceneSourceMap builds a per-scene source evidence map from generation
// artifacts. Each scene gets its chapter references and summary from the
// ScenePlan's source refs and the SourceIndex's sentence data.
func buildSceneSourceMap(artifacts *schema.GenerationArtifacts) map[int]*SceneSourceInfo {
	if artifacts == nil || artifacts.ScenePlan == nil || artifacts.SourceIndex == nil {
		return nil
	}

	// index sentences by ID for fast lookup
	sentenceByID := make(map[string]schema.SourceSentence, len(artifacts.SourceIndex.Sentences))
	for _, s := range artifacts.SourceIndex.Sentences {
		sentenceByID[s.ID] = s
	}

	// collect chapter names from a plan's source refs
	collectChapters := func(refs []schema.SourceSentenceRef) []string {
		seen := make(map[int]bool)
		var chapters []string
		for _, ref := range refs {
			s, ok := sentenceByID[ref.SentenceID]
			if !ok || seen[s.Chapter] {
				continue
			}
			seen[s.Chapter] = true
			chapters = append(chapters, fmt.Sprintf("第%d章", s.Chapter))
		}
		return chapters
	}

	plans := artifacts.ScenePlan.Scenes
	if len(plans) == 0 {
		// single-level plan without nested scenes
		if chapters := collectChapters(artifacts.ScenePlan.SourceRefs); len(chapters) > 0 {
			return map[int]*SceneSourceInfo{
				artifacts.ScenePlan.Sequence: {Chapters: chapters, Summary: artifacts.ScenePlan.Purpose},
			}
		}
		return nil
	}

	result := make(map[int]*SceneSourceInfo, len(plans))
	for _, plan := range plans {
		chapters := collectChapters(plan.SourceRefs)
		if len(chapters) == 0 && plan.Purpose == "" {
			continue
		}
		info := &SceneSourceInfo{Chapters: chapters, Summary: plan.Purpose}
		result[plan.Sequence] = info
	}
	return result
}
