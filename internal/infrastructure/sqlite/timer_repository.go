package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// TimerRepository is a SQLite worklog.TimerRepository.
type TimerRepository struct {
	db *sql.DB
}

func NewTimerRepository(db *sql.DB) *TimerRepository {
	return &TimerRepository{db: db}
}

const timerColumns = `user_id, project, repo, ticket, title, round, phase, started_at, paused_at, paused_seconds, closing_at, flow_step`

func (r *TimerRepository) Find(ctx context.Context, userID user.ID) (*worklog.Timer, error) {
	t, err := scanTimer(r.db.QueryRowContext(ctx, `SELECT `+timerColumns+` FROM timers WHERE user_id = ?`, string(userID)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TimerRepository) FindClosedBefore(ctx context.Context, at time.Time) ([]worklog.Timer, error) {
	// Times are stored in one fixed UTC layout, so they compare as text
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+timerColumns+` FROM timers WHERE closing_at IS NOT NULL AND closing_at <= ?`, formatTime(at))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var timers []worklog.Timer
	for rows.Next() {
		t, err := scanTimer(rows)
		if err != nil {
			return nil, err
		}
		timers = append(timers, t)
	}
	return timers, rows.Err()
}

func scanTimer(row scanner) (worklog.Timer, error) {
	var userID, proj, repo, ticket, title, phase, startedAt, flowStep string
	var pausedAt, closingAt sql.NullString
	var round, pausedSeconds int
	err := row.Scan(&userID, &proj, &repo, &ticket, &title, &round, &phase, &startedAt, &pausedAt, &pausedSeconds, &closingAt,
		&flowStep)
	if err != nil {
		return worklog.Timer{}, err
	}
	key, err := storedKey(proj, repo, ticket)
	if err != nil {
		return worklog.Timer{}, err
	}
	ph, err := worklog.ParsePhase(phase)
	if err != nil {
		return worklog.Timer{}, err
	}
	started, err := parseTime(startedAt)
	if err != nil {
		return worklog.Timer{}, err
	}
	t, err := worklog.NewTimer(user.ID(userID), key, title, round, ph, started)
	if err != nil {
		return worklog.Timer{}, err
	}
	t.Paused = time.Duration(pausedSeconds) * time.Second
	t.FlowStep = flowStep
	if t.PausedAt, err = nullTime(pausedAt); err != nil {
		return worklog.Timer{}, err
	}
	if t.ClosingAt, err = nullTime(closingAt); err != nil {
		return worklog.Timer{}, err
	}
	return t, nil
}

func nullTime(s sql.NullString) (*time.Time, error) {
	if !s.Valid {
		return nil, nil
	}
	at, err := parseTime(s.String)
	if err != nil {
		return nil, err
	}
	return &at, nil
}

func (r *TimerRepository) Save(ctx context.Context, t worklog.Timer) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO timers (`+timerColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET
			project = excluded.project,
			repo = excluded.repo,
			ticket = excluded.ticket,
			title = excluded.title,
			round = excluded.round,
			phase = excluded.phase,
			started_at = excluded.started_at,
			paused_at = excluded.paused_at,
			paused_seconds = excluded.paused_seconds,
			closing_at = excluded.closing_at,
			flow_step = excluded.flow_step`,
		string(t.UserID), t.Ticket.Project.String(), string(t.Ticket.Repo), t.Ticket.Number, t.Title, t.Round,
		string(t.Phase),
		formatTime(t.StartedAt), pausedAtOrNull(t.PausedAt), int(t.Paused/time.Second), pausedAtOrNull(t.ClosingAt),
		t.FlowStep,
	)
	return err
}

func (r *TimerRepository) Delete(ctx context.Context, userID user.ID) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM timers WHERE user_id = ?`, string(userID))
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// pausedAtOrNull stores an optional time: NULL for none
func pausedAtOrNull(t *time.Time) any {
	if t == nil {
		return nil
	}
	return formatTime(*t)
}
