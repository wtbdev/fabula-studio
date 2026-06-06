-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, nickname, ai_points, created_at, updated_at)
VALUES ($1, $2, $3, $4, 1000, now(), now())
RETURNING id, email, password_hash, nickname, ai_points, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, nickname, ai_points, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, nickname, ai_points, created_at, updated_at
FROM users
WHERE id = $1;

-- name: DecrementUserAIPoints :one
UPDATE users
SET ai_points = ai_points - $2, updated_at = now()
WHERE id = $1 AND ai_points >= $2
RETURNING id, email, password_hash, nickname, ai_points, created_at, updated_at;
