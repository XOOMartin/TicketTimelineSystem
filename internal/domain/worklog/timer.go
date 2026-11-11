package worklog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"crud_app/internal/domain/user"
)

// Timer counts time on one phase of a ticket while someone works on it.
// Ending it turns the time into check-ins. A user has at most one running,
// as they work on one thing at a time. While paused it doesn't count.
type Timer struct {
	UserID    user.ID
	Ticket    TicketKey
	Title     string
	Round     int
	Phase     Phase
	StartedAt time.Time
	// PausedAt is when the current pause began, nil while counting. Paused
	// is the time spent in earlier pauses.
	PausedAt *time.Time
	Paused   time.Duration
	// FlowStep is the step of a typed ticket's workflow being timed (its
	// time goes on Phase); "" for a phase of an untyped ticket
	FlowStep string
	// ClosingAt is when the last open page of the site was closed, nil while
	// it's open. The timer belongs to the open site: if no page comes back
	// within CloseGrace (as one does on a reload), it ends at ClosingAt.
	ClosingAt *time.Time
}

// CloseGrace is how long a closed site's timer waits for a page to come back
// before it ends. A reload, or another tab still open, comes back within a
// few seconds; the time recorded stops at the close either way.
const CloseGrace = 30 * time.Second

// Closing notes that the site was closed; a second close keeps the first.
func (t *Timer) Closing(now time.Time) {
	if t.ClosingAt == nil {
		t.ClosingAt = &now
	}
}

// Reopen notes that a page of the site is open again, so the timer carries on.
func (t *Timer) Reopen() { t.ClosingAt = nil }

// IsClosed reports whether the site was closed long enough ago, with no page
// back since, that the timer has ended (at ClosingAt).
func (t Timer) IsClosed(now time.Time) bool {
	return t.ClosingAt != nil && now.Sub(*t.ClosingAt) >= CloseGrace
}

// Elapsed is the time worked so far, leaving out pauses.
func (t Timer) Elapsed(now time.Time) time.Duration {
	if t.PausedAt != nil {
		now = *t.PausedAt
	}
	return max(0, now.Sub(t.StartedAt)-t.Paused)
}

func (t Timer) IsPaused() bool { return t.PausedAt != nil }

// Pause stops the timer counting; pausing a paused timer changes nothing.
func (t *Timer) Pause(now time.Time) {
	if t.PausedAt == nil {
		t.PausedAt = &now
	}
}

// Resume counts again, leaving the pause out of the time.
func (t *Timer) Resume(now time.Time) {
	if t.PausedAt != nil {
		t.Paused += now.Sub(*t.PausedAt)
		t.PausedAt = nil
	}
}

func NewTimer(userID user.ID, ticket TicketKey, title string, round int, phase Phase, startedAt time.Time) (Timer, error) {
	number, err := ParseTicketNumber(ticket.Number)
	if err != nil {
		return Timer{}, err
	}
	ticket.Number = number
	title = strings.TrimSpace(title)
	if round == 0 {
		round = 1
	}
	switch {
	case ticket.Project == "":
		return Timer{}, fmt.Errorf("%w: project is required", ErrInvalid)
	case title == "":
		return Timer{}, fmt.Errorf("%w: add the ticket's title before starting the timer", ErrInvalid)
	case round < 1 || round > MaxRound:
		return Timer{}, fmt.Errorf("%w: round must be between 1 and %d", ErrInvalid, MaxRound)
	case phase.order() < 0:
		return Timer{}, fmt.Errorf("%w: phase is required", ErrInvalid)
	}
	return Timer{UserID: userID, Ticket: ticket, Title: title, Round: round, Phase: phase, StartedAt: startedAt}, nil
}

type TimerRepository interface {
	// Find returns the user's running timer, or nil if none is running.
	Find(ctx context.Context, userID user.ID) (*Timer, error)
	// Save starts the timer, replacing any the user had.
	Save(ctx context.Context, t Timer) error
	// Delete removes the user's timer, reporting whether there was one to
	// remove; of two stops at once (e.g. a page opening as the background
	// check ends it), only the one that removed it logs its time.
	Delete(ctx context.Context, userID user.ID) (bool, error)
	// FindClosedBefore returns every user's timer whose site was closed at
	// or before the time given.
	FindClosedBefore(ctx context.Context, at time.Time) ([]Timer, error)
	// FindAll returns every user's running timer, for who is busy right now.
	FindAll(ctx context.Context) ([]Timer, error)
}
