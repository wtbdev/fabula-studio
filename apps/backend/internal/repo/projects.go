package repo

import (
	"context"
	"strings"

	"github.com/fabula-studio/backend/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProjectRepo struct {
	q *sqlc.Queries
}

type ProjectList struct {
	Items []sqlc.ListProjectsRow
	Total int32
}

func (r *ProjectRepo) Create(ctx context.Context, arg sqlc.CreateProjectParams) (sqlc.Project, error) {
	arg.Title = strings.TrimSpace(arg.Title)
	return r.q.CreateProject(ctx, arg)
}

func (r *ProjectRepo) List(ctx context.Context, userID, keyword string, page, pageSize int32) (ProjectList, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	keyword = strings.TrimSpace(keyword)
	total, err := r.q.CountProjects(ctx, sqlc.CountProjectsParams{UserID: userID, Column2: keyword})
	if err != nil {
		return ProjectList{}, err
	}
	items, err := r.q.ListProjects(ctx, sqlc.ListProjectsParams{UserID: userID, Column2: keyword, Limit: pageSize, Offset: (page - 1) * pageSize})
	if err != nil {
		return ProjectList{}, err
	}
	return ProjectList{Items: items, Total: total}, nil
}

func (r *ProjectRepo) ByIDForUser(ctx context.Context, projectID, userID string) (sqlc.Project, error) {
	return r.q.GetProjectByIDForUser(ctx, sqlc.GetProjectByIDForUserParams{ID: projectID, UserID: userID})
}

func (r *ProjectRepo) UpdateInfo(ctx context.Context, projectID, userID, title string, novelTitle pgtype.Text) (sqlc.Project, error) {
	return r.q.UpdateProjectInfo(ctx, sqlc.UpdateProjectInfoParams{ID: projectID, UserID: userID, Title: strings.TrimSpace(title), NovelTitle: novelTitle})
}

func (r *ProjectRepo) Delete(ctx context.Context, projectID, userID string) (bool, error) {
	rows, err := r.q.DeleteProject(ctx, sqlc.DeleteProjectParams{ID: projectID, UserID: userID})
	return rows > 0, err
}
