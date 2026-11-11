package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// TimerCommand starts timing one phase of a ticket.
type TimerCommand struct {
	Project string
	Repo    string // SPinno: spec, backend or web; "" for none
	Ticket  string
	Title   string
	Round   int // 0 means the original work
	Phase   string
	// Step is the workflow step of a typed (SPinno) ticket to time; its
	// time goes on the step's phase, so Phase isn't needed
	Step string
}

// StoppedTimer is a timer that was ended and the check-ins its time became.
// Under a minute logs nothing.
type StoppedTimer struct {
	Timer    worklog.Timer
	Minutes  int
	CheckIns []*worklog.CheckIn
}

// TimerService holds the use cases for timing work instead of typing it in.
type TimerService struct {
	timers   worklog.TimerRepository
	checkIns worklog.Repository
	progress worklog.ProgressRepository
	types    worklog.TypeRepository
	parents  worklog.ParentRepository
	planned  worklog.PlannedTicketRepository
}

func NewTimerService(timers worklog.TimerRepository, checkIns worklog.Repository, progress worklog.ProgressRepository, types worklog.TypeRepository, parents worklog.ParentRepository, planned worklog.PlannedTicketRepository) *TimerService {
	return &TimerService{timers: timers, checkIns: checkIns, progress: progress, types: types, parents: parents, planned: planned}
}

// Current is the user's running timer, or nil.
func (s *TimerService) Current(ctx context.Context, u *user.User) (*worklog.Timer, error) {
	t, _, err := s.find(ctx, u.ID(), time.Now())
	return t, err
}

// find is the user's running timer, or nil. One whose site was closed (and
// no page came back in time) ends first, at the close, and is returned as
// stopped.
func (s *TimerService) find(ctx context.Context, userID user.ID, now time.Time) (*worklog.Timer, *StoppedTimer, error) {
	t, err := s.timers.Find(ctx, userID)
	if err != nil || t == nil {
		return nil, nil, err
	}
	if !t.IsClosed(now) {
		return t, nil, nil
	}
	stopped, err := s.stop(ctx, userID, *t, *t.ClosingAt)
	return nil, stopped, err
}

// Closing notes that the user's last open page of the site was closed. The
// timer ends there, unless a page comes back within worklog.CloseGrace (a
// reload, or another tab that's still open).
func (s *TimerService) Closing(ctx context.Context, u *user.User) error {
	t, _, err := s.find(ctx, u.ID(), time.Now())
	if err != nil || t == nil {
		return err
	}
	t.Closing(time.Now())
	return s.timers.Save(ctx, *t)
}

// Open notes that a page of the site is open. A reload, or another tab that
// was open all along, carries the timer on. A page opened fresh after the
// site was closed (fresh) means it was really closed: the timer ends at the
// close. So does a site closed for too long. The ended timer is returned as
// stopped, so the page can say so.
func (s *TimerService) Open(ctx context.Context, u *user.User, fresh bool) (*worklog.Timer, *StoppedTimer, error) {
	t, stopped, err := s.find(ctx, u.ID(), time.Now())
	if err != nil || t == nil || t.ClosingAt == nil {
		return t, stopped, err
	}
	if fresh {
		stopped, err := s.stop(ctx, u.ID(), *t, *t.ClosingAt)
		return nil, stopped, err
	}
	t.Reopen()
	return t, nil, s.timers.Save(ctx, *t)
}

// StopClosed ends every timer whose site was closed more than
// worklog.CloseGrace ago, recording its time up to the close. It runs in the
// background, so the time shows in reports without its owner coming back.
func (s *TimerService) StopClosed(ctx context.Context, now time.Time) (int, error) {
	closed, err := s.timers.FindClosedBefore(ctx, now.Add(-worklog.CloseGrace))
	if err != nil {
		return 0, err
	}
	for _, t := range closed {
		if _, err := s.stop(ctx, t.UserID, t, *t.ClosingAt); err != nil {
			return 0, err
		}
	}
	return len(closed), nil
}

// Start times a phase. A timer already running is ended first and its time
// logged, since a person works on one thing at a time.
func (s *TimerService) Start(ctx context.Context, u *user.User, cmd TimerCommand) (*worklog.Timer, *StoppedTimer, error) {
	key, err := worklog.ParseTicketKey(cmd.Project, cmd.Repo, cmd.Ticket)
	if err != nil {
		return nil, nil, err
	}
	// Only the user's own project, and a parent ticket takes no time of its own
	if err := ownProject(ctx, s.checkIns, s.planned, u, key); err != nil {
		return nil, nil, err
	}
	if err := noParentTime(ctx, s.parents, s.types, u.ID(), key); err != nil {
		return nil, nil, err
	}
	// A typed ticket times a step of its workflow, whose phase takes the time
	var step worklog.FlowStep
	if cmd.Step != "" {
		ticketType, err := s.typeOf(ctx, u.ID(), key)
		if err != nil {
			return nil, nil, err
		}
		var i int
		if step, i = worklog.WorkflowOf(ticketType).Find(max(cmd.Round, 1), cmd.Step); i < 0 || step.Kind != worklog.StepWork {
			return nil, nil, fmt.Errorf("%w: %q isn't a timed step of %s tickets", worklog.ErrInvalid, cmd.Step, ticketType.Label())
		}
		cmd.Phase = string(step.Phase)
	}
	phase, err := worklog.ParsePhase(cmd.Phase)
	if err != nil {
		return nil, nil, err
	}
	// Checked before ending the running timer, so a bad start changes nothing
	t, err := worklog.NewTimer(u.ID(), key, cmd.Title, cmd.Round, phase, time.Now())
	if err != nil {
		return nil, nil, err
	}
	t.FlowStep = step.Key

	running, stopped, err := s.find(ctx, u.ID(), t.StartedAt)
	if err != nil {
		return nil, nil, err
	}
	if running != nil {
		if stopped, err = s.stop(ctx, u.ID(), *running, t.StartedAt); err != nil {
			return nil, nil, err
		}
	}
	if err := s.timers.Save(ctx, t); err != nil {
		return nil, nil, err
	}
	return &t, stopped, nil
}

// Pause stops the user's timer counting until they resume it.
func (s *TimerService) Pause(ctx context.Context, u *user.User) (*worklog.Timer, error) {
	return s.update(ctx, u, func(t *worklog.Timer, now time.Time) { t.Pause(now) })
}

// Resume counts again; the pause isn't part of the time.
func (s *TimerService) Resume(ctx context.Context, u *user.User) (*worklog.Timer, error) {
	return s.update(ctx, u, func(t *worklog.Timer, now time.Time) { t.Resume(now) })
}

func (s *TimerService) update(ctx context.Context, u *user.User, change func(*worklog.Timer, time.Time)) (*worklog.Timer, error) {
	t, _, err := s.find(ctx, u.ID(), time.Now())
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, worklog.ErrNoTimer
	}
	// Pausing or resuming means a page is open
	t.Reopen()
	change(t, time.Now())
	if err := s.timers.Save(ctx, *t); err != nil {
		return nil, err
	}
	return t, nil
}

// Stop ends the user's timer and logs its time on the phase. It returns
// worklog.ErrNoTimer when none is running.
func (s *TimerService) Stop(ctx context.Context, u *user.User) (*StoppedTimer, error) {
	running, closed, err := s.find(ctx, u.ID(), time.Now())
	if err != nil {
		return nil, err
	}
	// It ended already, when the site was closed
	if closed != nil {
		return closed, nil
	}
	if running == nil {
		return nil, worklog.ErrNoTimer
	}
	return s.stop(ctx, u.ID(), *running, time.Now())
}

// stop logs the time worked until end, pauses left out, dated when the work
// began.
// Past the most one check-in can hold, it takes several in a row. Timing
// work on a ticket means it's under way, so the time is logged as ongoing.
func (s *TimerService) stop(ctx context.Context, userID user.ID, t worklog.Timer, end time.Time) (*StoppedTimer, error) {
	minutes := int(t.Elapsed(end).Minutes())
	result := &StoppedTimer{Timer: t, Minutes: minutes}

	// Taken off first, so a timer is logged once even if two stops meet (a
	// page opening as the background check ends it): the one that finds it
	// gone logs nothing
	removed, err := s.timers.Delete(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !removed {
		return &StoppedTimer{Timer: t}, nil
	}
	// Put back if its time can't be logged, so it isn't lost
	restore := func(err error) (*StoppedTimer, error) {
		if saveErr := s.timers.Save(ctx, t); saveErr != nil {
			return nil, errors.Join(err, saveErr)
		}
		return nil, err
	}

	if minutes >= 1 {
		at := t.StartedAt
		for left := minutes; left > 0; {
			chunk := min(left, worklog.MaxHours*60)
			c, err := newCheckIn(ctx, s.checkIns, userID, worklog.SourceApp, worklog.Entry{
				Ticket:   t.Ticket,
				Title:    t.Title,
				Round:    t.Round,
				Phase:    t.Phase,
				Status:   worklog.StatusOngoing,
				Hours:    float64(chunk) / 60,
				WorkedAt: at,
			}, end)
			if err != nil {
				return restore(err)
			}
			result.CheckIns = append(result.CheckIns, c)
			at = at.Add(time.Duration(chunk) * time.Minute)
			left -= chunk
		}
		if err := saveShifting(ctx, s.checkIns, result.CheckIns); err != nil {
			return restore(err)
		}
		// The time is logged: a failure moving the ticket on doesn't undo it
		if err := s.advance(ctx, userID, t); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// saveShifting saves new check-ins. If the user already logged the phase at
// the same minute, they move a minute later rather than the time being lost.
func saveShifting(ctx context.Context, repo worklog.Repository, checkIns []*worklog.CheckIn) error {
	for attempt := 0; ; attempt++ {
		err := repo.SaveChanges(ctx, checkIns, nil)
		if !errors.Is(err, worklog.ErrDuplicate) || attempt == 5 {
			return err
		}
		for _, c := range checkIns {
			e := c.Entry()
			e.WorkedAt = e.WorkedAt.Add(time.Minute)
			if err := c.Revise(e); err != nil {
				return err
			}
		}
	}
}

// advance moves the ticket to the step after the timed phase. A review
// stays put: what comes next depends on its outcome.
func (s *TimerService) advance(ctx context.Context, userID user.ID, t worklog.Timer) error {
	// A typed ticket's work step is done: on to its next step
	if t.FlowStep != "" {
		ticketType, err := s.typeOf(ctx, userID, t.Ticket)
		if err != nil {
			return err
		}
		p, err := worklog.AfterStep(userID, t.Ticket, ticketType, t.Round, t.FlowStep)
		if err != nil {
			return err
		}
		return s.progress.Save(ctx, p)
	}
	p, err := worklog.NewProgress(userID, t.Ticket, t.Round, t.Phase.NextOrReview())
	if err != nil {
		return err
	}
	return s.progress.Save(ctx, p)
}

// typeOf is a SPinno ticket's type: chosen, or its repo's default.
// The ticket's number is parsed already: Start's by ParseTicketKey, a
// running timer's by NewTimer.
func (s *TimerService) typeOf(ctx context.Context, userID user.ID, ticket worklog.TicketKey) (worklog.TicketType, error) {
	t, err := typeOfTicket(ctx, s.parents, s.types, userID, ticket)
	if err != nil {
		return "", err
	}
	if t == "" {
		return "", fmt.Errorf("%w: only SPinno tickets have steps of their own", worklog.ErrInvalid)
	}
	return t, nil
}
