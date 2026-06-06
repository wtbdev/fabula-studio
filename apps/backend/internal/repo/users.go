package repo

import (
	"context"
	"strings"

	"github.com/fabula-studio/backend/internal/db/sqlc"
)

type UserRepo struct {
	q *sqlc.Queries
}

func (r *UserRepo) Create(ctx context.Context, id, email, passwordHash, nickname string) (sqlc.User, error) {
	return r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           id,
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: passwordHash,
		Nickname:     strings.TrimSpace(nickname),
	})
}

func (r *UserRepo) ByEmail(ctx context.Context, email string) (sqlc.User, error) {
	return r.q.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
}

func (r *UserRepo) ByID(ctx context.Context, id string) (sqlc.User, error) {
	return r.q.GetUserByID(ctx, id)
}
