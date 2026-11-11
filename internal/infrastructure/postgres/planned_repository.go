package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// PlannedTicketRepository is a PostgreSQL worklog.PlannedTicketRepository.
type PlannedTicketRepository struct {
	db *sql.DB
}

func NewPlannedTicketRepository(db *sql.DB) *PlannedTicketRepository {
	return &PlannedTicketRepository{db: db}
}

const plannedColumns = `project, repo, ticket, title, status, note, added_day, added_at`

func scanPlanned(userID user.ID, row scanner) (worklog.PlannedTicket, error) {
	var proj, repo, ticket, title, status, note, added, addedAt string
	if err := row.Scan(&proj, &repo, &ticket, &title, &status, &note, &added, &addedAt); err != nil {
		return worklog.PlannedTicket{}, err
	}
	key, err := storedKey(proj, repo, ticket)
	if err != nil {
		return worklog.PlannedTicket{}, err
	}
	st, err := worklog.ParseStatus(status)
	if err != nil {
		return worklog.PlannedTicket{}, err
	}
	day, err := calendar.ParseDay(added)
	if err != nil {
		return worklog.PlannedTicket{}, err
	}
	// Rows from before the time was kept: the start of the day they were added
	at, err := time.ParseInLocation(calendar.DayLayout, added, time.Local)
	if addedAt != "" {
		at, err = time.Parse(timeLayout, addedAt)
	}
	if err != nil {
		return worklog.PlannedTicket{}, err
	}
	return worklog.PlannedTicket{
		UserID:  userID,
		Ticket:  key,
		Title:   title,
		Status:  st,
		Note:    note,
		AddedOn: day,
		AddedAt: at,
	}, nil
}

func (r *PlannedTicketRepository) FindByUser(ctx context.Context, userID user.ID) (map[worklog.TicketKey]worklog.PlannedTicket, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+plannedColumns+` FROM planned_tickets WHERE user_id = $1`, string(userID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	planned := map[worklog.TicketKey]worklog.PlannedTicket{}
	for rows.Next() {
		p, err := scanPlanned(userID, rows)
		if err != nil {
			return nil, err
		}
		planned[p.Ticket] = p
	}
	return planned, rows.Err()
}

func (r *PlannedTicketRepository) Find(ctx context.Context, userID user.ID, ticket worklog.TicketKey) (worklog.PlannedTicket, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+plannedColumns+` FROM planned_tickets WHERE user_id = $1 AND project = $2 AND repo = $3 AND ticket = $4`,
		string(userID), ticket.Project.String(), string(ticket.Repo), ticket.Number)
	p, err := scanPlanned(userID, row)
	if errors.Is(err, sql.ErrNoRows) {
		return worklog.PlannedTicket{}, worklog.ErrNotFound
	}
	return p, err
}

func (r *PlannedTicketRepository) Save(ctx context.Context, p worklog.PlannedTicket) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO planned_tickets (user_id, `+plannedColumns+`)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, project, repo, ticket) DO UPDATE SET
			title = excluded.title,
			status = excluded.status,
			note = excluded.note`,
		string(p.UserID), p.Ticket.Project.String(), string(p.Ticket.Repo), p.Ticket.Number, p.Title, string(p.Status),
		p.Note,
		p.AddedOn.String(), p.AddedAt.UTC().Format(timeLayout),
	)
	return err
}
