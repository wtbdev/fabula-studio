-- name: DeleteScenesByProjectID :exec
DELETE FROM scenes
WHERE project_id = $1;

-- name: CreateScene :one
INSERT INTO scenes (id, project_id, scene_no, title, location, time_text, summary, content, raw_json, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), now())
RETURNING id, project_id, scene_no, title, location, time_text, summary, content, raw_json, created_at, updated_at;

-- name: ListScenesForUserProject :many
SELECT s.id, s.project_id, s.scene_no, s.title, s.location, s.time_text, s.summary, s.content, s.raw_json, s.created_at, s.updated_at
FROM scenes s
JOIN projects p ON p.id = s.project_id
WHERE s.project_id = $1 AND p.user_id = $2
ORDER BY s.scene_no ASC;

-- name: GetSceneForUser :one
SELECT s.id, s.project_id, s.scene_no, s.title, s.location, s.time_text, s.summary, s.content, s.raw_json, s.created_at, s.updated_at
FROM scenes s
JOIN projects p ON p.id = s.project_id
WHERE s.id = $1 AND p.user_id = $2;

-- name: UpdateSceneForUser :one
UPDATE scenes s
SET title = $3,
    location = $4,
    time_text = $5,
    summary = $6,
    content = $7,
    updated_at = now()
FROM projects p
WHERE s.id = $1 AND s.project_id = p.id AND p.user_id = $2
RETURNING s.id, s.project_id, s.scene_no, s.title, s.location, s.time_text, s.summary, s.content, s.raw_json, s.created_at, s.updated_at;
