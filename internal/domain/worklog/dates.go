package worklog

import (
	"context"
	"fmt"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/user"
)

// TicketDates is when a user says they started and finished a ticket, as on
// their progress sheet. A ticket without them starts on the first day time
// was logged and ends on the day it was marked done.
type TicketDates struct {
	UserID user.ID
	Ticket TicketKey
	Start  *calendar.Day // nil: the first day time was logged
	End    *calendar.Day // nil: not finished yet
}

func NewTicketDates(userID user.ID, ticket TicketKey, start, end *calendar.Day) (TicketDates, error) {
	if start != nil && end != nil && end.Before(*start) {
		return TicketDates{}, fmt.Errorf("%w: the end date can't be before the start date", ErrInvalid)
	}
	return TicketDates{UserID: userID, Ticket: ticket, Start: start, End: end}, nil
}

type TicketDatesRepository interface {
	// FindByUser returns the user's dates for each ticket that has them.
	FindByUser(ctx context.Context, userID user.ID) (map[TicketKey]TicketDates, error)
	// Save inserts or replaces the dates for the user's ticket.
	Save(ctx context.Context, d TicketDates) error
}
