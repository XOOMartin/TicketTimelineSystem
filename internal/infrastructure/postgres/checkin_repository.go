package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// CheckInRepository is a PostgreSQL worklog.Repository.
type CheckInRepository struct {
	db *sql.DB
}

func NewCheckInRepository(db *sql.DB) *CheckInRepository {
	return &CheckInRepository{db: db}
}

const checkInColumns = `id, user_id, source, project, ticket, title, round, phase, status, hours, note, worked_at, day, created_at, repo`

func (r *CheckInRepository) NextID(ctx context.Context) (worklog.ID, error) {
	return worklog.ID(uuid.NewString()), nil
}

func (r *CheckInRepository) FindByID(ctx context.Context, id worklog.ID) (*worklog.CheckIn, error) {
	c, err := scanCheckIn(r.db.QueryRowContext(ctx, `SELECT `+checkInColumns+` FROM checkins WHERE id = $1`, string(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, worklog.ErrNotFound
	}
	return c, err
}

func (r *CheckInRepository) FindByUser(ctx context.Context, userID user.ID, period calendar.Period) ([]*worklog.CheckIn, error) {
	return r.query(ctx, `
		SELECT `+checkInColumns+` FROM checkins
		WHERE user_id = $1 AND day BETWEEN $2 AND $3
		ORDER BY worked_at DESC, created_at DESC`,
		string(userID), period.First().String(), period.Last().String(),
	)
}

func (r *CheckInRepository) FindByTicket(ctx context.Context, userID user.ID, ticket worklog.TicketKey) ([]*worklog.CheckIn, error) {
	return r.query(ctx, `
		SELECT `+checkInColumns+` FROM checkins
		WHERE user_id = $1 AND project = $2 AND repo = $3 AND ticket = $4
		ORDER BY worked_at, created_at`,
		string(userID), ticket.Project.String(), string(ticket.Repo), ticket.Number,
	)
}

func (r *CheckInRepository) FindUntil(ctx context.Context, last calendar.Day) ([]*worklog.CheckIn, error) {
	return r.query(ctx, `
		SELECT `+checkInColumns+` FROM checkins
		WHERE day <= $1
		ORDER BY worked_at, created_at`,
		last.String(),
	)
}

const upsertCheckIn = `
	INSERT INTO checkins (` + checkInColumns + `)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	ON CONFLICT (id) DO UPDATE SET
		project = excluded.project,
		repo = excluded.repo,
		ticket = excluded.ticket,
		title = excluded.title,
		round = excluded.round,
		phase = excluded.phase,
		status = excluded.status,
		hours = excluded.hours,
		note = excluded.note,
		worked_at = excluded.worked_at,
		day = excluded.day`

func (r *CheckInRepository) Save(ctx context.Context, c *worklog.CheckIn) error {
	_, err := r.db.ExecContext(ctx, upsertCheckIn, checkInArgs(c)...)
	if isUniqueViolation(err) {
		return worklog.ErrDuplicate
	}
	return err
}

func (r *CheckInRepository) SaveChanges(ctx context.Context, save []*worklog.CheckIn, remove []worklog.ID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete first, so a saved row can take the time slot of a removed one.
	for _, id := range remove {
		if _, err := tx.ExecContext(ctx, `DELETE FROM checkins WHERE id = $1`, string(id)); err != nil {
			return err
		}
	}
	for _, c := range save {
		if _, err := tx.ExecContext(ctx, upsertCheckIn, checkInArgs(c)...); err != nil {
			if isUniqueViolation(err) {
				return worklog.ErrDuplicate
			}
			return err
		}
	}
	return tx.Commit()
}

func (r *CheckInRepository) SaveNew(ctx context.Context, checkIns []*worklog.CheckIn) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// DO NOTHING skips rows that hit the UNIQUE key, i.e. were imported before.
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO checkins (`+checkInColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT DO NOTHING`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	added := 0
	for _, c := range checkIns {
		res, err := stmt.ExecContext(ctx, checkInArgs(c)...)
		if err != nil {
			return 0, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		added += int(n)
	}
	return added, tx.Commit()
}

func (r *CheckInRepository) Delete(ctx context.Context, id worklog.ID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM checkins WHERE id = $1`, string(id))
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return worklog.ErrNotFound
	}
	return err
}

func (r *CheckInRepository) DeleteTicket(ctx context.Context, userID user.ID, ticket worklog.TicketKey) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	args := []any{string(userID), ticket.Project.String(), string(ticket.Repo), ticket.Number}
	const where = ` WHERE user_id = $1 AND project = $2 AND repo = $3 AND ticket = $4`
	res, err := tx.ExecContext(ctx, `DELETE FROM checkins`+where, args...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	for _, table := range []string{"ticket_dates", "ticket_progress", "ticket_parents", "ticket_types"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+where, args...); err != nil {
			return 0, err
		}
	}
	// A spec ticket's backend and web tickets go back to no spec, rather
	// than sit under one that's gone
	if ticket.Repo == project.RepoSpec {
		_, err := tx.ExecContext(ctx, `DELETE FROM ticket_parents WHERE user_id = $1 AND project = $2 AND parent = $3`,
			string(userID), ticket.Project.String(), ticket.Number)
		if err != nil {
			return 0, err
		}
	}
	// A ticket added without time counts as one row gone, so deleting it
	// isn't "not found"
	res, err = tx.ExecContext(ctx, `DELETE FROM planned_tickets`+where, args...)
	if err != nil {
		return 0, err
	}
	planned, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(n + planned), tx.Commit()
}

func (r *CheckInRepository) query(ctx context.Context, q string, args ...any) ([]*worklog.CheckIn, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checkIns []*worklog.CheckIn
	for rows.Next() {
		c, err := scanCheckIn(rows)
		if err != nil {
			return nil, err
		}
		checkIns = append(checkIns, c)
	}
	return checkIns, rows.Err()
}

func checkInArgs(c *worklog.CheckIn) []any {
	return []any{
		string(c.ID()), string(c.UserID()), string(c.Source()),
		c.Ticket().Project.String(), c.Ticket().Number, c.Title(), c.Round(),
		string(c.Phase()), string(c.Status()), c.Hours(), c.Note(),
		formatTime(c.WorkedAt()), c.Day().String(), formatTime(c.CreatedAt()), string(c.Ticket().Repo),
	}
}

func scanCheckIn(s scanner) (*worklog.CheckIn, error) {
	var id, userID, source, proj, ticket, title, phase, status, note, workedAt, day, createdAt, repo string
	var hours float64
	var round int
	err := s.Scan(&id, &userID, &source, &proj, &ticket, &title, &round, &phase, &status, &hours, &note, &workedAt, &day, &createdAt, &repo)
	if err != nil {
		return nil, err
	}
	key, err := storedKey(proj, repo, ticket)
	if err != nil {
		return nil, err
	}
	ph, err := worklog.ParsePhase(phase)
	if err != nil {
		return nil, err
	}
	st, err := worklog.ParseStatus(status)
	if err != nil {
		return nil, err
	}
	worked, err := parseTime(workedAt)
	if err != nil {
		return nil, err
	}
	created, err := parseTime(createdAt)
	if err != nil {
		return nil, err
	}
	// Rebuilt as saved: a limit tightened since (e.g. MaxHours) must not
	// stop older entries from loading
	return worklog.Restore(worklog.ID(id), user.ID(userID), worklog.Source(source), worklog.Entry{
		Ticket:   key,
		Title:    title,
		Round:    round,
		Phase:    ph,
		Status:   st,
		Hours:    hours,
		Note:     note,
		WorkedAt: worked,
	}, created), nil
}

// ImportRepository is a PostgreSQL worklog.ImportRepository.
type ImportRepository struct {
	db *sql.DB
}

func NewImportRepository(db *sql.DB) *ImportRepository {
	return &ImportRepository{db: db}
}

func (r *ImportRepository) Save(ctx context.Context, imp worklog.Import) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO imports (user_id, week, filename, added, skipped, imported_by, imported_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		string(imp.UserID), imp.Week.String(), imp.Filename, imp.Added, imp.Skipped,
		string(imp.ImportedBy), formatTime(imp.ImportedAt),
	)
	return err
}

func (r *ImportRepository) LatestForWeek(ctx context.Context, week calendar.Day) (map[user.ID]worklog.Import, error) {
	// Ordered oldest first, so later imports overwrite earlier ones in the map.
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id, filename, added, skipped, imported_by, imported_at
		FROM imports WHERE week = $1 ORDER BY imported_at`, week.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	latest := map[user.ID]worklog.Import{}
	for rows.Next() {
		var userID, filename, importedBy, importedAt string
		var added, skipped int
		if err := rows.Scan(&userID, &filename, &added, &skipped, &importedBy, &importedAt); err != nil {
			return nil, err
		}
		at, err := parseTime(importedAt)
		if err != nil {
			return nil, err
		}
		latest[user.ID(userID)] = worklog.Import{
			UserID:     user.ID(userID),
			Week:       week,
			Filename:   filename,
			Added:      added,
			Skipped:    skipped,
			ImportedBy: user.ID(importedBy),
			ImportedAt: at,
		}
	}
	return latest, rows.Err()
}

// TicketDatesRepository is a PostgreSQL worklog.TicketDatesRepository.
type TicketDatesRepository struct {
	db *sql.DB
}

func NewTicketDatesRepository(db *sql.DB) *TicketDatesRepository {
	return &TicketDatesRepository{db: db}
}

func (r *TicketDatesRepository) FindByUser(ctx context.Context, userID user.ID) (map[worklog.TicketKey]worklog.TicketDates, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT project, repo, ticket, start_day, end_day FROM ticket_dates WHERE user_id = $1`, string(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dates := map[worklog.TicketKey]worklog.TicketDates{}
	for rows.Next() {
		var proj, repo, ticket string
		var start, end sql.NullString
		if err := rows.Scan(&proj, &repo, &ticket, &start, &end); err != nil {
			return nil, err
		}
		key, err := storedKey(proj, repo, ticket)
		if err != nil {
			return nil, err
		}
		d := worklog.TicketDates{UserID: userID, Ticket: key}
		if d.Start, err = nullableDay(start); err != nil {
			return nil, err
		}
		if d.End, err = nullableDay(end); err != nil {
			return nil, err
		}
		dates[key] = d
	}
	return dates, rows.Err()
}

func (r *TicketDatesRepository) Save(ctx context.Context, d worklog.TicketDates) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ticket_dates (user_id, project, repo, ticket, start_day, end_day)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, project, repo, ticket) DO UPDATE SET
			start_day = excluded.start_day,
			end_day = excluded.end_day`,
		string(d.UserID), d.Ticket.Project.String(), string(d.Ticket.Repo), d.Ticket.Number,
		dayOrNull(d.Start), dayOrNull(d.End),
	)
	return err
}

// storedKey reads a ticket's key from its stored project, repo and number.
func storedKey(proj, repo, ticket string) (worklog.TicketKey, error) {
	p, err := project.Parse(proj)
	if err != nil {
		return worklog.TicketKey{}, err
	}
	r, err := p.ParseRepo(repo)
	if err != nil {
		return worklog.TicketKey{}, err
	}
	return worklog.TicketKey{Project: p, Repo: r, Number: ticket}, nil
}

func nullableDay(s sql.NullString) (*calendar.Day, error) {
	if !s.Valid {
		return nil, nil
	}
	d, err := calendar.ParseDay(s.String)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func dayOrNull(d *calendar.Day) any {
	if d == nil {
		return nil
	}
	return d.String()
}
