package repo

import (
	"context"

	"github.com/fabula-studio/backend/internal/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool           *pgxpool.Pool
	Queries        *sqlc.Queries
	Users          *UserRepo
	Projects       *ProjectRepo
	Scenes         *SceneRepo
	GenerationJobs *GenerationJobRepo
}

func NewStore(pool *pgxpool.Pool) *Store {
	queries := sqlc.New(pool)
	store := &Store{Pool: pool, Queries: queries}
	store.Users = &UserRepo{q: queries}
	store.Projects = &ProjectRepo{q: queries}
	store.Scenes = &SceneRepo{q: queries}
	store.GenerationJobs = &GenerationJobRepo{db: pool}
	return store
}

func (s *Store) WithTx(ctx context.Context, fn func(*sqlc.Queries) error) error {
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(s.Queries.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
