package postgres

import (
	"context"
	"database/sql"

	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// TypeRepository is a PostgreSQL worklog.TypeRepository.
type TypeRepository struct {
	db *sql.DB
}

func NewTypeRepository(db *sql.DB) *TypeRepository {
	return &TypeRepository{db: db}
}

func (r *TypeRepository) FindByUser(ctx context.Context, userID user.ID) (map[worklog.TicketKey]worklog.TicketType, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT project, repo, ticket, type FROM ticket_types WHERE user_id = $1 AND NOT made_parent`, string(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	types := map[worklog.TicketKey]worklog.TicketType{}
	for rows.Next() {
		var proj, repo, ticket, t string
		if err := rows.Scan(&proj, &repo, &ticket, &t); err != nil {
			return nil, err
		}
		key, err := storedKey(proj, repo, ticket)
		if err != nil {
			return nil, err
		}
		types[key] = worklog.TicketType(t)
	}
	return types, rows.Err()
}

func (r *TypeRepository) Set(ctx context.Context, userID user.ID, ticket worklog.TicketKey, t worklog.TicketType) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ticket_types (user_id, project, repo, ticket, type)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, project, repo, ticket) DO UPDATE SET type = excluded.type`,
		string(userID), ticket.Project.String(), string(ticket.Repo), ticket.Number, string(t))
	return err
}

// FindMadeParents returns the user's tickets made parents by dragging.
func (r *TypeRepository) FindMadeParents(ctx context.Context, userID user.ID) (map[worklog.TicketKey]bool, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT project, repo, ticket FROM ticket_types WHERE user_id = $1 AND made_parent`, string(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	made := map[worklog.TicketKey]bool{}
	for rows.Next() {
		var proj, repo, ticket string
		if err := rows.Scan(&proj, &repo, &ticket); err != nil {
			return nil, err
		}
		key, err := storedKey(proj, repo, ticket)
		if err != nil {
			return nil, err
		}
		made[key] = true
	}
	return made, rows.Err()
}

func (r *TypeRepository) SetMadeParent(ctx context.Context, userID user.ID, ticket worklog.TicketKey, made bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ticket_types SET made_parent = $1 WHERE user_id = $2 AND project = $3 AND repo = $4 AND ticket = $5`,
		made, string(userID), ticket.Project.String(), string(ticket.Repo), ticket.Number)
	return err
}
