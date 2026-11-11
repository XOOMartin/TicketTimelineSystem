package worklog

import (
	"context"
	"fmt"

	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
)

// SpecParent is the spec ticket a SPinno ticket sits under: for a backend or
// web ticket, e.g. the spec whose testing found the bug; for a spec ticket, a
// bigger spec it's part of. A spec can't sit under itself.
func SpecParent(child TicketKey, number string) (TicketKey, error) {
	if !child.Repo.HasParent() {
		return TicketKey{}, fmt.Errorf("%w: only SPinno tickets sit under a spec ticket", ErrInvalid)
	}
	n, err := ParseTicketNumber(number)
	if err != nil {
		return TicketKey{}, fmt.Errorf("%w: the parent spec ticket must be a number like 120", ErrInvalid)
	}
	parent := TicketKey{Project: child.Project, Repo: project.RepoSpec, Number: n}
	if parent == child {
		return TicketKey{}, fmt.Errorf("%w: a spec ticket can't sit under itself", ErrInvalid)
	}
	return parent, nil
}

// Ancestors are the spec tickets above a ticket, nearest first, following
// parents; a loop (which shouldn't be stored) ends the walk.
func Ancestors(ticket TicketKey, parents map[TicketKey]TicketKey) []TicketKey {
	var up []TicketKey
	seen := map[TicketKey]bool{ticket: true}
	for p, ok := parents[ticket]; ok && !seen[p]; p, ok = parents[p] {
		up = append(up, p)
		seen[p] = true
	}
	return up
}

// ParentRepository keeps, per user, the spec ticket each backend or web
// ticket sits under.
type ParentRepository interface {
	// FindByUser returns each of the user's tickets that has a parent, by
	// ticket.
	FindByUser(ctx context.Context, userID user.ID) (map[TicketKey]TicketKey, error)
	// Set puts the ticket under parent, or under nothing when parent is nil.
	Set(ctx context.Context, userID user.ID, child TicketKey, parent *TicketKey) error
}
