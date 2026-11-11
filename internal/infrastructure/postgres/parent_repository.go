package postgres

import (
	"context"
	"database/sql"

	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// ParentRepository is a PostgreSQL worklog.ParentRepository.
type ParentRepository struct {
	db *sql.DB
}

func NewParentRepository(db *sql.DB) *ParentRepository {
	return &ParentRepository{db: db}
}

func (r *ParentRepository) FindByUser(ctx context.Context, userID user.ID) (map[worklog.TicketKey]worklog.TicketKey, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT project, repo, ticket, parent FROM ticket_parents WHERE user_id = $1`, string(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	parents := map[worklog.TicketKey]worklog.TicketKey{}
	for rows.Next() {
		var proj, repo, ticket, parent string
		if err := rows.Scan(&proj, &repo, &ticket, &parent); err != nil {
			return nil, err
		}
		key, err := storedKey(proj, repo, ticket)
		if err != nil {
			return nil, err
		}
		parents[key] = worklog.TicketKey{Project: key.Project, Repo: project.RepoSpec, Number: parent}
	}
	return parents, rows.Err()
}

func (r *ParentRepository) Set(ctx context.Context, userID user.ID, child worklog.TicketKey, parent *worklog.TicketKey) error {
	args := []any{string(userID), child.Project.String(), string(child.Repo), child.Number}
	if parent == nil {
		_, err := r.db.ExecContext(ctx,
			`DELETE FROM ticket_parents WHERE user_id = $1 AND project = $2 AND repo = $3 AND ticket = $4`, args...)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ticket_parents (user_id, project, repo, ticket, parent)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, project, repo, ticket) DO UPDATE SET parent = excluded.parent`,
		append(args, parent.Number)...)
	return err
}
