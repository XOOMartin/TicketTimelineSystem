package application

import (
	"context"
	"sort"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// ActivityState is whether someone has work on the go.
type ActivityState string

const (
	// StateBusy: at least one of their tickets is ongoing
	StateBusy ActivityState = "busy"
	// StateAvailable: none of their tickets is ongoing
	StateAvailable ActivityState = "available"
)

// MemberActivity is one person as the team sees them right now: whether
// they're busy, what they're on, and how their day is going.
type MemberActivity struct {
	User  *user.User
	State ActivityState
	// Timer is what they're timing right now, nil if no timer is on
	Timer   *worklog.Timer
	Elapsed time.Duration
	// LatestOngoing is the time they logged most recently on a ticket that's
	// still ongoing: what they're on when no timer is running. nil if none.
	LatestOngoing *worklog.CheckIn
	// LastCheckIn is the time they logged most recently, nil if none yet
	LastCheckIn *worklog.CheckIn
	// TodayHours is the time logged today, not counting a running timer
	TodayHours float64
	// Their tickets by status: being worked on (ongoing), and on hold
	// (waiting for someone, or blocked)
	OngoingTickets int
	WaitingTickets int
	BlockedTickets int
}

// TeamService answers who is busy: everyone's ongoing tickets, running
// timer and day.
type TeamService struct {
	users    user.Repository
	timers   worklog.TimerRepository
	checkIns worklog.Repository
	members  *MemberService
}

func NewTeamService(users user.Repository, timers worklog.TimerRepository, checkIns worklog.Repository, members *MemberService) *TeamService {
	return &TeamService{users: users, timers: timers, checkIns: checkIns, members: members}
}

// Activity lists every member: the busy first (a ticket ongoing), then the
// available, each group by name.
func (s *TeamService) Activity(ctx context.Context, now time.Time) ([]MemberActivity, error) {
	timers, err := s.timers.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	running := make(map[user.ID]worklog.Timer, len(timers))
	for _, t := range timers {
		// A timer whose site was closed has ended; it is only waiting to be logged
		if !t.IsClosed(now) {
			running[t.UserID] = t
		}
	}

	// Each person's tickets by status, as their timeline shows them
	timelines, err := s.members.TeamTimelines(ctx, now)
	if err != nil {
		return nil, err
	}
	counts := make(map[user.ID]map[worklog.Status]int, len(timelines))
	ongoing := make(map[user.ID]map[worklog.TicketKey]bool, len(timelines))
	for _, m := range timelines {
		byStatus := make(map[worklog.Status]int)
		keys := make(map[worklog.TicketKey]bool)
		for _, t := range m.Tickets {
			byStatus[t.Status]++
			if t.Status == worklog.StatusOngoing {
				keys[t.Ticket] = true
			}
		}
		counts[m.User.ID()], ongoing[m.User.ID()] = byStatus, keys
	}

	today := calendar.DayOf(now)
	checkIns, err := s.checkIns.FindUntil(ctx, today)
	if err != nil {
		return nil, err
	}
	last := make(map[user.ID]*worklog.CheckIn)
	latestOngoing := make(map[user.ID]*worklog.CheckIn)
	todayHours := make(map[user.ID]float64)
	for _, c := range checkIns {
		id := c.UserID()
		if prev, ok := last[id]; !ok || c.WorkedAt().After(prev.WorkedAt()) {
			last[id] = c
		}
		if ongoing[id][c.Ticket()] {
			if prev, ok := latestOngoing[id]; !ok || c.WorkedAt().After(prev.WorkedAt()) {
				latestOngoing[id] = c
			}
		}
		if c.Day() == today {
			todayHours[id] += c.Hours()
		}
	}

	users, err := s.users.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	team := make([]MemberActivity, len(users))
	for i, u := range users {
		byStatus := counts[u.ID()]
		a := MemberActivity{
			User: u, State: StateAvailable,
			LatestOngoing: latestOngoing[u.ID()], LastCheckIn: last[u.ID()], TodayHours: todayHours[u.ID()],
			OngoingTickets: byStatus[worklog.StatusOngoing],
			WaitingTickets: byStatus[worklog.StatusWaiting],
			BlockedTickets: byStatus[worklog.StatusBlocked],
		}
		if a.OngoingTickets > 0 {
			a.State = StateBusy
		}
		if t, ok := running[u.ID()]; ok {
			a.Timer = &t
			a.Elapsed = t.Elapsed(now)
		}
		team[i] = a
	}
	sort.SliceStable(team, func(i, j int) bool {
		if (team[i].State == StateBusy) != (team[j].State == StateBusy) {
			return team[i].State == StateBusy
		}
		return team[i].User.Name() < team[j].User.Name()
	})
	return team, nil
}
