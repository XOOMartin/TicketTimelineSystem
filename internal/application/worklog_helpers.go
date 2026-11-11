package application

import (
	"context"
	"errors"
	"time"

	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// newCheckIn makes a check-in with the next free id. It isn't saved, so a
// batch can be checked whole before any of it is.
func newCheckIn(ctx context.Context, repo worklog.Repository, userID user.ID, source worklog.Source, entry worklog.Entry, now time.Time) (*worklog.CheckIn, error) {
	id, err := repo.NextID(ctx)
	if err != nil {
		return nil, err
	}
	return worklog.New(id, userID, source, entry, now)
}

// inTimeline reports whether the ticket is in the user's timeline already:
// time logged on it, or added without time.
func inTimeline(ctx context.Context, checkIns worklog.Repository, planned worklog.PlannedTicketRepository, userID user.ID, key worklog.TicketKey) (bool, error) {
	existing, err := checkIns.FindByTicket(ctx, userID, key)
	if err != nil {
		return false, err
	}
	if len(existing) > 0 {
		return true, nil
	}
	if _, err := planned.Find(ctx, userID, key); err == nil {
		return true, nil
	} else if !errors.Is(err, worklog.ErrNotFound) {
		return false, err
	}
	return false, nil
}

// hierarchy loads the user's parent links and chosen types together, as
// working out a ticket's level or type needs both.
func hierarchy(ctx context.Context, parents worklog.ParentRepository, types worklog.TypeRepository, userID user.ID) (map[worklog.TicketKey]worklog.TicketKey, map[worklog.TicketKey]worklog.TicketType, error) {
	under, err := parents.FindByUser(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	chosen, err := types.FindByUser(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	return under, chosen, nil
}

// typeOfTicket is a ticket's type: chosen, or its repo's default; "" for a
// parent ticket and outside SPinno.
func typeOfTicket(ctx context.Context, parents worklog.ParentRepository, types worklog.TypeRepository, userID user.ID, key worklog.TicketKey) (worklog.TicketType, error) {
	under, chosen, err := hierarchy(ctx, parents, types, userID)
	if err != nil {
		return "", err
	}
	return worklog.TypeOf(chosen, under, key), nil
}
