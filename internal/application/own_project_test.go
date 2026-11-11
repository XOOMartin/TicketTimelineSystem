package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// Only FindByTicket is used: the user's check-ins on these tickets
type checkInsWith struct {
	worklog.Repository
	tickets map[worklog.TicketKey]bool
}

func (r checkInsWith) FindByTicket(_ context.Context, _ user.ID, key worklog.TicketKey) ([]*worklog.CheckIn, error) {
	if r.tickets[key] {
		return []*worklog.CheckIn{{}}, nil
	}
	return nil, nil
}

// Only Find is used: the tickets the user added without time
type plannedWith struct {
	worklog.PlannedTicketRepository
	tickets map[worklog.TicketKey]bool
}

func (r plannedWith) Find(_ context.Context, _ user.ID, key worklog.TicketKey) (worklog.PlannedTicket, error) {
	if r.tickets[key] {
		return worklog.PlannedTicket{}, nil
	}
	return worklog.PlannedTicket{}, worklog.ErrNotFound
}

// A member's new tickets are in the project they chose at signup; tickets
// they already have in the other one can still be worked on.
func TestOwnProject(t *testing.T) {
	u, err := user.New("u1", "Martin", "m@example.com", "hash", project.BlackBox, user.RoleMember, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	blackBox := worklog.TicketKey{Project: project.BlackBox, Number: "3287"}
	logged := worklog.TicketKey{Project: project.SPinno, Repo: project.RepoWeb, Number: "4234"}
	added := worklog.TicketKey{Project: project.SPinno, Repo: project.RepoSpec, Number: "325"}
	fresh := worklog.TicketKey{Project: project.SPinno, Repo: project.RepoSpec, Number: "999"}
	checkIns := checkInsWith{tickets: map[worklog.TicketKey]bool{logged: true}}
	planned := plannedWith{tickets: map[worklog.TicketKey]bool{added: true}}

	for _, tc := range []struct {
		name string
		key  worklog.TicketKey
		ok   bool
	}{
		{"a ticket in their own project", blackBox, true},
		{"another project's ticket they logged time on before", logged, true},
		{"another project's ticket they added before", added, true},
		{"a new ticket in another project", fresh, false},
	} {
		err := ownProject(context.Background(), checkIns, planned, u, tc.key)
		if tc.ok && err != nil {
			t.Errorf("%s: refused: %v", tc.name, err)
		}
		if !tc.ok && !errors.Is(err, worklog.ErrInvalid) {
			t.Errorf("%s: got %v, want it refused", tc.name, err)
		}
	}
}

// Only Find is used, and it fails
type plannedFailing struct {
	worklog.PlannedTicketRepository
}

func (plannedFailing) Find(context.Context, user.ID, worklog.TicketKey) (worklog.PlannedTicket, error) {
	return worklog.PlannedTicket{}, errors.New("db down")
}

// A ticket is in the timeline with time logged on it or added without time.
func TestInTimeline(t *testing.T) {
	ctx := context.Background()
	logged := worklog.TicketKey{Project: project.BlackBox, Number: "1"}
	added := worklog.TicketKey{Project: project.BlackBox, Number: "2"}
	fresh := worklog.TicketKey{Project: project.BlackBox, Number: "3"}
	checkIns := checkInsWith{tickets: map[worklog.TicketKey]bool{logged: true}}
	planned := plannedWith{tickets: map[worklog.TicketKey]bool{added: true}}
	for key, want := range map[worklog.TicketKey]bool{logged: true, added: true, fresh: false} {
		if got, err := inTimeline(ctx, checkIns, planned, "u1", key); err != nil || got != want {
			t.Errorf("%s: got %v, %v; want %v", key, got, err, want)
		}
	}
	// A failed lookup isn't taken for "not there"
	if _, err := inTimeline(ctx, checkIns, plannedFailing{}, "u1", fresh); err == nil {
		t.Error("a failed lookup of the added tickets was ignored")
	}
}

// The form's ticket is read as ParseTicketKey reads it, its errors still bad
// requests.
func TestCheckInCommandEntryTicket(t *testing.T) {
	now := time.Now()
	cmd := CheckInCommand{Project: "SPinno", Repo: "web", Ticket: " #120 ", Title: "t", Phase: "coding", Status: "ongoing", Hours: 1, WorkedAt: now}
	e, err := cmd.entry(now)
	if err != nil {
		t.Fatal(err)
	}
	if want := (worklog.TicketKey{Project: project.SPinno, Repo: project.RepoWeb, Number: "120"}); e.Ticket != want {
		t.Errorf("ticket = %+v, want %+v", e.Ticket, want)
	}
	cmd.Repo = "mobile"
	if _, err := cmd.entry(now); !errors.Is(err, worklog.ErrInvalid) {
		t.Errorf("unknown repo: got %v, want ErrInvalid", err)
	}
}
