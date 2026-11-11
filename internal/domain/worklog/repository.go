package worklog

import (
	"context"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/user"
)

// Repository is defined by the domain and implemented by infrastructure.
type Repository interface {
	NextID(ctx context.Context) (ID, error)
	// FindByID returns ErrNotFound when no check-in has the id.
	FindByID(ctx context.Context, id ID) (*CheckIn, error)
	// FindByUser returns the user's check-ins in the period, newest first.
	FindByUser(ctx context.Context, userID user.ID, period calendar.Period) ([]*CheckIn, error)
	// FindByTicket returns the user's check-ins on the ticket, oldest first.
	FindByTicket(ctx context.Context, userID user.ID, ticket TicketKey) ([]*CheckIn, error)
	// FindUntil returns every check-in on or before the day, oldest first.
	FindUntil(ctx context.Context, last calendar.Day) ([]*CheckIn, error)
	// Save inserts or updates the check-in. It returns ErrDuplicate when the
	// user already has a check-in for the same ticket, phase and time.
	Save(ctx context.Context, c *CheckIn) error
	// SaveChanges deletes and saves (inserts or updates) check-ins together:
	// either all changes apply or, on ErrDuplicate or any other error, none.
	SaveChanges(ctx context.Context, save []*CheckIn, remove []ID) error
	// SaveNew inserts the check-ins, skipping duplicates, and returns how
	// many were added.
	SaveNew(ctx context.Context, checkIns []*CheckIn) (int, error)
	Delete(ctx context.Context, id ID) error
	// DeleteTicket removes all of the user's check-ins on the ticket, in every
	// round, and the dates and progress they had on it, together, along with
	// the ticket if they added it without time. It returns how many rows went
	// (check-ins, plus one for an added ticket), so 0 means it wasn't there.
	// A spec ticket's backend and web tickets go back to having no spec.
	DeleteTicket(ctx context.Context, userID user.ID, ticket TicketKey) (int, error)
}

// Import records that an admin loaded a user's spreadsheet for a week.
type Import struct {
	UserID     user.ID
	Week       calendar.Day // the Monday
	Filename   string
	Added      int
	Skipped    int
	ImportedBy user.ID
	ImportedAt time.Time
}

type ImportRepository interface {
	Save(ctx context.Context, imp Import) error
	// LatestForWeek returns each user's most recent import for the week.
	LatestForWeek(ctx context.Context, week calendar.Day) (map[user.ID]Import, error)
}
