package repo

import (
	"context"

	"github.com/fabula-studio/backend/internal/db/sqlc"
)

type SceneRepo struct {
	q *sqlc.Queries
}

func (r *SceneRepo) ListForProject(ctx context.Context, projectID, userID string) ([]sqlc.Scene, error) {
	return r.q.ListScenesForUserProject(ctx, sqlc.ListScenesForUserProjectParams{ProjectID: projectID, UserID: userID})
}

func (r *SceneRepo) ByIDForUser(ctx context.Context, sceneID, userID string) (sqlc.Scene, error) {
	return r.q.GetSceneForUser(ctx, sqlc.GetSceneForUserParams{ID: sceneID, UserID: userID})
}

func (r *SceneRepo) UpdateForUser(ctx context.Context, arg sqlc.UpdateSceneForUserParams) (sqlc.Scene, error) {
	return r.q.UpdateSceneForUser(ctx, arg)
}
