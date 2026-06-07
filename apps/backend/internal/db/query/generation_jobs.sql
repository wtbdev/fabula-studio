-- name: CreateGenerationJob :one
INSERT INTO generation_jobs (id, project_id, status, progress, current_step, error_message, artifacts, created_at, updated_at)
VALUES ($1, $2, 'queued', 0, $3, NULL, NULL, now(), now())
RETURNING id, project_id, status, progress, current_step, error_message, artifacts, started_at, completed_at, created_at, updated_at;

-- name: GetActiveGenerationJobByProjectID :one
SELECT id, project_id, status, progress, current_step, error_message, artifacts, started_at, completed_at, created_at, updated_at
FROM generation_jobs
WHERE project_id = $1 AND status IN ('queued', 'running')
ORDER BY created_at DESC
LIMIT 1;

-- name: GetLatestGenerationJobByProjectID :one
SELECT id, project_id, status, progress, current_step, error_message, artifacts, started_at, completed_at, created_at, updated_at
FROM generation_jobs
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateGenerationJobRunning :one
UPDATE generation_jobs
SET status = 'running',
    progress = $2,
    current_step = $3,
    error_message = NULL,
    started_at = COALESCE(started_at, now()),
    updated_at = now()
WHERE id = $1
RETURNING id, project_id, status, progress, current_step, error_message, artifacts, started_at, completed_at, created_at, updated_at;

-- name: UpdateGenerationJobProgress :one
UPDATE generation_jobs
SET status = $2,
    progress = $3,
    current_step = $4,
    error_message = $5,
    artifacts = $6,
    started_at = CASE WHEN $2 = 'running' THEN COALESCE(started_at, now()) ELSE started_at END,
    updated_at = now()
WHERE id = $1
RETURNING id, project_id, status, progress, current_step, error_message, artifacts, started_at, completed_at, created_at, updated_at;

-- name: UpdateGenerationJobArtifacts :one
UPDATE generation_jobs
SET artifacts = $2, updated_at = now()
WHERE id = $1
RETURNING id, project_id, status, progress, current_step, error_message, artifacts, started_at, completed_at, created_at, updated_at;

-- name: CompleteGenerationJob :one
UPDATE generation_jobs
SET status = 'completed',
    progress = 100,
    current_step = $2,
    error_message = NULL,
    artifacts = $3,
    completed_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING id, project_id, status, progress, current_step, error_message, artifacts, started_at, completed_at, created_at, updated_at;

-- name: FailGenerationJob :one
UPDATE generation_jobs
SET status = 'failed',
    current_step = $2,
    error_message = $3,
    artifacts = $4,
    completed_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING id, project_id, status, progress, current_step, error_message, artifacts, started_at, completed_at, created_at, updated_at;
