package sqlite

import (
	"context"
	"database/sql"

	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// ProgressRepository is a SQLite worklog.ProgressRepository.
type ProgressRepository struct {
	db *sql.DB
}

func NewProgressRepository(db *sql.DB) *ProgressRepository {
	return &ProgressRepository{db: db}
}

func (r *ProgressRepository) FindByUser(ctx context.Context, userID user.ID) (map[worklog.TicketKey]worklog.Progress, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT project, repo, ticket, round, step, finished, flow_step FROM ticket_progress WHERE user_id = ?`, string(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	progress := map[worklog.TicketKey]worklog.Progress{}
	for rows.Next() {
		var proj, repo, ticket, step, flowStep string
		var round int
		var finished bool
		if err := rows.Scan(&proj, &repo, &ticket, &round, &step, &finished, &flowStep); err != nil {
			return nil, err
		}
		key, err := storedKey(proj, repo, ticket)
		if err != nil {
			return nil, err
		}
		if finished {
			progress[key] = worklog.FinishedProgress(userID, key, round)
			continue
		}
		ph, err := worklog.ParsePhase(step)
		if err != nil {
			return nil, err
		}
		p, err := worklog.NewProgress(userID, key, round, ph)
		if err != nil {
			return nil, err
		}
		p.FlowStep = flowStep
		progress[key] = p
	}
	return progress, rows.Err()
}

func (r *ProgressRepository) Delete(ctx context.Context, userID user.ID, ticket worklog.TicketKey) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM ticket_progress WHERE user_id = ? AND project = ? AND repo = ? AND ticket = ?`,
		string(userID), ticket.Project.String(), string(ticket.Repo), ticket.Number)
	return err
}

func (r *ProgressRepository) Save(ctx context.Context, p worklog.Progress) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ticket_progress (user_id, project, repo, ticket, round, step, finished, flow_step)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id, project, repo, ticket) DO UPDATE SET
			round = excluded.round,
			step = excluded.step,
			finished = excluded.finished,
			flow_step = excluded.flow_step`,
		string(p.UserID), p.Ticket.Project.String(), string(p.Ticket.Repo), p.Ticket.Number, p.Round, string(p.Step),
		p.Finished, p.FlowStep,
	)
	return err
}
