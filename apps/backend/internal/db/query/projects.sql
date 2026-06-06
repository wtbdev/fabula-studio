-- name: CreateProject :one
INSERT INTO projects (id, user_id, title, novel_title, source_text, config_json, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, 'draft', now(), now())
RETURNING id, user_id, title, novel_title, source_text, config_json, status, error_message, created_at, updated_at;

-- name: ListProjects :many
SELECT p.id, p.user_id, p.title, p.novel_title, p.source_text, p.config_json, p.status, p.error_message, p.created_at, p.updated_at,
       count(s.id)::int AS scene_count
FROM projects p
LEFT JOIN scenes s ON s.project_id = p.id
WHERE p.user_id = $1
  AND ($2::text = '' OR p.title ILIKE '%' || $2 || '%' OR COALESCE(p.novel_title, '') ILIKE '%' || $2 || '%')
GROUP BY p.id
ORDER BY p.updated_at DESC
LIMIT $3 OFFSET $4;

-- name: CountProjects :one
SELECT count(*)::int
FROM projects
WHERE user_id = $1
  AND ($2::text = '' OR title ILIKE '%' || $2 || '%' OR COALESCE(novel_title, '') ILIKE '%' || $2 || '%');

-- name: GetProjectByIDForUser :one
SELECT id, user_id, title, novel_title, source_text, config_json, status, error_message, created_at, updated_at
FROM projects
WHERE id = $1 AND user_id = $2;

-- name: UpdateProjectInfo :one
UPDATE projects
SET title = $3, novel_title = $4, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, title, novel_title, source_text, config_json, status, error_message, created_at, updated_at;

-- name: UpdateProjectStatus :one
UPDATE projects
SET status = $3, error_message = $4, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, title, novel_title, source_text, config_json, status, error_message, created_at, updated_at;

-- name: DeleteProject :execrows
DELETE FROM projects
WHERE id = $1 AND user_id = $2;
