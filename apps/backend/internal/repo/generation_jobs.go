package repo

import (
	"context"

	"github.com/fabula-studio/backend/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type GenerationJob struct {
	ID           string
	ProjectID    string
	Status       string
	Progress     int32
	CurrentStep  string
	ErrorMessage pgtype.Text
	Artifacts    []byte
	StartedAt    pgtype.Timestamptz
	CompletedAt  pgtype.Timestamptz
	CreatedAt    pgtype.Timestamptz
	UpdatedAt    pgtype.Timestamptz
}

type CreateGenerationJobParams struct {
	ID          string
	ProjectID   string
	CurrentStep string
}

type UpdateGenerationJobRunningParams struct {
	ID          string
	Progress    int32
	CurrentStep string
}

type UpdateGenerationJobProgressParams struct {
	ID           string
	Status       string
	Progress     int32
	CurrentStep  string
	ErrorMessage pgtype.Text
	Artifacts    []byte
}

type UpdateGenerationJobArtifactsParams struct {
	ID        string
	Artifacts []byte
}

type CompleteGenerationJobParams struct {
	ID          string
	CurrentStep string
	Artifacts   []byte
}

type FailGenerationJobParams struct {
	ID           string
	CurrentStep  string
	ErrorMessage string
	Artifacts    []byte
}

type GenerationJobRepo struct {
	db sqlc.DBTX
}

const generationJobColumns = `id, project_id, status, progress, current_step, error_message, artifacts, started_at, completed_at, created_at, updated_at`

func (r *GenerationJobRepo) CreateGenerationJob(ctx context.Context, arg CreateGenerationJobParams) (GenerationJob, error) {
	row := r.db.QueryRow(ctx, `INSERT INTO generation_jobs (id, project_id, status, progress, current_step, error_message, artifacts, created_at, updated_at)
VALUES ($1, $2, 'queued', 0, $3, NULL, NULL, now(), now())
RETURNING `+generationJobColumns, arg.ID, arg.ProjectID, arg.CurrentStep)
	return scanGenerationJob(row)
}

func (r *GenerationJobRepo) GetActiveGenerationJobByProjectID(ctx context.Context, projectID string) (GenerationJob, error) {
	row := r.db.QueryRow(ctx, `SELECT `+generationJobColumns+`
FROM generation_jobs
WHERE project_id = $1 AND status IN ('queued', 'running')
ORDER BY created_at DESC
LIMIT 1`, projectID)
	return scanGenerationJob(row)
}

func (r *GenerationJobRepo) GetLatestGenerationJobByProjectID(ctx context.Context, projectID string) (GenerationJob, error) {
	row := r.db.QueryRow(ctx, `SELECT `+generationJobColumns+`
FROM generation_jobs
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT 1`, projectID)
	return scanGenerationJob(row)
}

func (r *GenerationJobRepo) UpdateGenerationJobRunning(ctx context.Context, arg UpdateGenerationJobRunningParams) (GenerationJob, error) {
	row := r.db.QueryRow(ctx, `UPDATE generation_jobs
SET status = 'running',
    progress = $2,
    current_step = $3,
    error_message = NULL,
    started_at = COALESCE(started_at, now()),
    updated_at = now()
WHERE id = $1
RETURNING `+generationJobColumns, arg.ID, arg.Progress, arg.CurrentStep)
	return scanGenerationJob(row)
}

func (r *GenerationJobRepo) UpdateGenerationJobProgress(ctx context.Context, arg UpdateGenerationJobProgressParams) (GenerationJob, error) {
	row := r.db.QueryRow(ctx, `UPDATE generation_jobs
SET status = $2,
    progress = $3,
    current_step = $4,
    error_message = $5,
    artifacts = $6,
    started_at = CASE WHEN $2 = 'running' THEN COALESCE(started_at, now()) ELSE started_at END,
    updated_at = now()
WHERE id = $1
RETURNING `+generationJobColumns, arg.ID, arg.Status, arg.Progress, arg.CurrentStep, arg.ErrorMessage, arg.Artifacts)
	return scanGenerationJob(row)
}

func (r *GenerationJobRepo) UpdateGenerationJobArtifacts(ctx context.Context, arg UpdateGenerationJobArtifactsParams) (GenerationJob, error) {
	row := r.db.QueryRow(ctx, `UPDATE generation_jobs
SET artifacts = $2, updated_at = now()
WHERE id = $1
RETURNING `+generationJobColumns, arg.ID, arg.Artifacts)
	return scanGenerationJob(row)
}

func (r *GenerationJobRepo) CompleteGenerationJob(ctx context.Context, arg CompleteGenerationJobParams) (GenerationJob, error) {
	row := r.db.QueryRow(ctx, `UPDATE generation_jobs
SET status = 'completed',
    progress = 100,
    current_step = $2,
    error_message = NULL,
    artifacts = $3,
    completed_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING `+generationJobColumns, arg.ID, arg.CurrentStep, arg.Artifacts)
	return scanGenerationJob(row)
}

func (r *GenerationJobRepo) FailGenerationJob(ctx context.Context, arg FailGenerationJobParams) (GenerationJob, error) {
	row := r.db.QueryRow(ctx, `UPDATE generation_jobs
SET status = 'failed',
    current_step = $2,
    error_message = $3,
    artifacts = $4,
    completed_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING `+generationJobColumns, arg.ID, arg.CurrentStep, arg.ErrorMessage, arg.Artifacts)
	return scanGenerationJob(row)
}

func scanGenerationJob(row interface{ Scan(...any) error }) (GenerationJob, error) {
	var job GenerationJob
	err := row.Scan(
		&job.ID,
		&job.ProjectID,
		&job.Status,
		&job.Progress,
		&job.CurrentStep,
		&job.ErrorMessage,
		&job.Artifacts,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	return job, err
}
