package worklog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/user"
)

// ErrTicketExists is returned when adding a ticket the user already has.
var ErrTicketExists = errors.New("you already have this ticket in your timeline")

// PlannedTicket is a ticket a user added to their timeline before logging
// any time on it. It shows in the table (with no hours) from the week it was
// added until time is logged on it, when its check-ins take over.
type PlannedTicket struct {
	UserID  user.ID
	Ticket  TicketKey
	Title   string
	Status  Status
	Note    string
	AddedOn calendar.Day
	// AddedAt is when it was added, to sort the newest first; AddedOn is its
	// local day, for the weeks it shows in
	AddedAt time.Time
}

func NewPlannedTicket(userID user.ID, ticket TicketKey, title string, status Status, note string, addedAt time.Time) (PlannedTicket, error) {
	title = strings.TrimSpace(title)
	note = strings.TrimSpace(note)
	switch {
	case title == "":
		return PlannedTicket{}, fmt.Errorf("%w: title is required", ErrInvalid)
	case len([]rune(title)) > maxTitleLength:
		return PlannedTicket{}, fmt.Errorf("%w: title is longer than %d characters", ErrInvalid, maxTitleLength)
	case len([]rune(note)) > maxNoteLength:
		return PlannedTicket{}, fmt.Errorf("%w: note is longer than %d characters", ErrInvalid, maxNoteLength)
	case status == StatusBlocked && note == "":
		return PlannedTicket{}, fmt.Errorf("%w: say what is blocking the ticket in the note", ErrInvalid)
	}
	return PlannedTicket{
		UserID:  userID,
		Ticket:  ticket,
		Title:   title,
		Status:  status,
		Note:    note,
		AddedOn: calendar.DayOf(addedAt),
		AddedAt: addedAt,
	}, nil
}

type PlannedTicketRepository interface {
	// FindByUser returns the tickets the user added, by ticket.
	FindByUser(ctx context.Context, userID user.ID) (map[TicketKey]PlannedTicket, error)
	// Find returns ErrNotFound when the user didn't add the ticket.
	Find(ctx context.Context, userID user.ID, ticket TicketKey) (PlannedTicket, error)
	// Save inserts or replaces the user's ticket.
	Save(ctx context.Context, p PlannedTicket) error
}
