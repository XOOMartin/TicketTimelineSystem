package worklog

import (
	"context"
	"fmt"

	"crud_app/internal/domain/user"
)

// Progress is the step a user is on in a ticket: the phase they work on next
// and in which round. A finished ticket has no step.
type Progress struct {
	UserID   user.ID
	Ticket   TicketKey
	Round    int
	Step     Phase // "" once the ticket is finished
	Finished bool
	// FlowStep is the step of a typed (SPinno) ticket's workflow, whose
	// time goes on Step; "" for a ticket stepping through the phases
	FlowStep string
}

func NewProgress(userID user.ID, ticket TicketKey, round int, step Phase) (Progress, error) {
	if round < 1 || round > MaxRound {
		return Progress{}, fmt.Errorf("%w: round must be between 1 and %d", ErrInvalid, MaxRound)
	}
	if step.order() < 0 {
		return Progress{}, fmt.Errorf("%w: unknown step %q", ErrInvalid, step)
	}
	return Progress{UserID: userID, Ticket: ticket, Round: round, Step: step}, nil
}

// FinishedProgress is a ticket whose last round was approved.
func FinishedProgress(userID user.ID, ticket TicketKey, round int) Progress {
	return Progress{UserID: userID, Ticket: ticket, Round: round, Finished: true}
}

// Next is the phase after p in the workflow, or "" after the last.
func (p Phase) Next() Phase {
	phases := Phases()
	if i := p.order(); i >= 0 && i+1 < len(phases) {
		return phases[i+1]
	}
	return ""
}

// NextOrReview is the step after p's time is done: the next phase, or the
// review after the last (a review waits there for its outcome).
func (p Phase) NextOrReview() Phase {
	if next := p.Next(); next != "" {
		return next
	}
	return PhaseReview
}

type ProgressRepository interface {
	// FindByUser returns the user's progress on each ticket that has it.
	FindByUser(ctx context.Context, userID user.ID) (map[TicketKey]Progress, error)
	Save(ctx context.Context, p Progress) error
	// Delete forgets the saved step, so it is worked out from the check-ins.
	Delete(ctx context.Context, userID user.ID, ticket TicketKey) error
}
