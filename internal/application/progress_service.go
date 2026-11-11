package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// Outcomes of the two steps where someone else decides what happens next.
const (
	// Confirmation by SE/MGRs
	OutcomeNoChanges        = "no_changes"        // on to planning
	OutcomeChangesRequested = "changes_requested" // confirmation: back to the spec; review: a review-back, from planning
	OutcomeWaiting          = "waiting"           // no reply yet: stay on the step
	// Review after implementation
	OutcomeApproved = "approved" // the ticket is done
)

var outcomeLabels = map[string]string{
	OutcomeNoChanges:        "No changes",
	OutcomeChangesRequested: "Changes requested",
	OutcomeWaiting:          "Waiting for confirmation",
	OutcomeApproved:         "Approved",
}

// OutcomeDone finishes a typed ticket's action (e.g. Push & open PR) or work
// step without the timer, or a BlackBox ticket's timed phase with the time
// typed in instead of timed.
const OutcomeDone = "done"

// OutcomeCommand records how a confirmation or review went.
type OutcomeCommand struct {
	// Step is the workflow step of a typed (SPinno) ticket; Phase then isn't
	// needed (the step's phase takes the time)
	Step    string
	Phase   string // confirmation or review
	Outcome string
	Round   int     // 0 means the ticket's latest round
	Hours   float64 // confirmation only: time it took; 0 logs none
	Comment string
	// For a ticket with no time yet, whose confirmation time starts it
	Title string
}

// SetStep moves the user's ticket to a step, e.g. skipping one that isn't
// needed or going back to one.
func (s *CheckInService) SetStep(ctx context.Context, u *user.User, projectName, repo, ticket string, round int, step string) error {
	key, existing, err := s.ticketWork(ctx, u, projectName, repo, ticket)
	// A ticket added without time has no check-ins yet, but has steps
	if errors.Is(err, worklog.ErrNotFound) {
		if _, plannedErr := s.planned.Find(ctx, u.ID(), key); plannedErr == nil {
			err = nil
		}
	}
	if err != nil {
		return err
	}
	round = roundOrLatest(round, existing)
	// A typed ticket moves between its type's steps
	ticketType, err := s.typeOf(ctx, u.ID(), key)
	if err != nil {
		return err
	}
	if ticketType != "" {
		fs, i := worklog.WorkflowOf(ticketType).Find(round, step)
		if i < 0 {
			return fmt.Errorf("%w: %q isn't a step of %s tickets", worklog.ErrInvalid, step, ticketType.Label())
		}
		p, err := worklog.FlowProgress(u.ID(), key, round, fs)
		if err != nil {
			return err
		}
		return s.progress.Save(ctx, p)
	}
	phase, err := worklog.ParsePhase(step)
	if err != nil {
		return err
	}
	p, err := worklog.NewProgress(u.ID(), key, round, phase)
	if err != nil {
		return err
	}
	return s.progress.Save(ctx, p)
}

// RecordOutcome finishes a confirmation or review step and moves the ticket
// on: after a confirmation to planning, back to the spec, or nowhere while
// waiting; after a review to done, or into the next round (a review-back)
// from planning, since the requested changes need planning first.
func (s *CheckInService) RecordOutcome(ctx context.Context, u *user.User, projectName, repo, ticket string, cmd OutcomeCommand) error {
	key, existing, err := s.ticketWork(ctx, u, projectName, repo, ticket)
	isNew := errors.Is(err, worklog.ErrNotFound)
	if err != nil && !isNew {
		return err
	}
	if cmd.Hours > 0 {
		if err := noParentTime(ctx, s.parents, s.types, u.ID(), key); err != nil {
			return err
		}
	}
	if cmd.Step != "" {
		ticketType, err := s.typeOf(ctx, u.ID(), key)
		if err != nil {
			return err
		}
		if ticketType != "" {
			return s.recordFlowOutcome(ctx, u, key, existing, cmd)
		}
		// The same progress form sends a BlackBox ticket's phase as its step,
		// and a confirmation's "approved" means no changes
		cmd.Phase = cmd.Step
		if cmd.Phase == string(worklog.PhaseConfirmation) && cmd.Outcome == OutcomeApproved {
			cmd.Outcome = OutcomeNoChanges
		}
	}
	phase, err := worklog.ParsePhase(cmd.Phase)
	if err != nil {
		return err
	}
	// A new ticket can only be started by time: here, the confirmation's or review's
	title := strings.TrimSpace(cmd.Title)
	if isNew && (cmd.Hours <= 0 || title == "") {
		return fmt.Errorf("%w: record some time on the ticket first", worklog.ErrInvalid)
	}
	round := roundOrLatest(cmd.Round, existing)
	note := outcomeLabels[cmd.Outcome]
	if comment := strings.TrimSpace(cmd.Comment); comment != "" {
		note += ": " + comment
	}

	status := worklog.StatusOngoing
	var next worklog.Progress
	switch {
	case phase == worklog.PhaseConfirmation && cmd.Outcome == OutcomeNoChanges:
		next, err = worklog.NewProgress(u.ID(), key, round, worklog.PhasePlanning)
	case phase == worklog.PhaseConfirmation && cmd.Outcome == OutcomeChangesRequested:
		// Sent back by a review, like the final one: a review-back, which
		// starts over from the spec
		next, err = worklog.NewProgress(u.ID(), key, round+1, worklog.PhaseInvestigation)
	case phase == worklog.PhaseConfirmation && cmd.Outcome == OutcomeWaiting:
		status = worklog.StatusWaiting
		next, err = worklog.NewProgress(u.ID(), key, round, worklog.PhaseConfirmation)
	case phase == worklog.PhaseReview && cmd.Outcome == OutcomeApproved:
		status = worklog.StatusDone
		next = worklog.FinishedProgress(u.ID(), key, round)
	case phase == worklog.PhaseReview && cmd.Outcome == OutcomeChangesRequested:
		next, err = worklog.NewProgress(u.ID(), key, round+1, worklog.PhasePlanning)
	case cmd.Outcome == OutcomeDone && phase != worklog.PhaseConfirmation && phase != worklog.PhaseReview:
		// A timed phase (spec, planning, coding, testing) with its time typed
		// in: logged on the phase, then on to the next one
		if cmd.Hours <= 0 {
			return fmt.Errorf("%w: type the time %s took, or use Done, no time", worklog.ErrInvalid, phase.Label())
		}
		next, err = worklog.NewProgress(u.ID(), key, round, phase.NextOrReview())
	default:
		return fmt.Errorf("%w: %q isn't an outcome of %s", worklog.ErrInvalid, cmd.Outcome, phase.Label())
	}
	if err != nil {
		return err
	}

	// The time a confirmation or review took (typed in, as it depends on the
	// reviewers) is a new check-in carrying the outcome. Otherwise the outcome
	// is noted on the step's own latest check-in, and the status set on the
	// ticket's latest, which is what decides it.
	now := time.Now()
	if cmd.Hours > 0 {
		if !isNew {
			title = latestOf(existing).Title()
		}
		c, err := newCheckIn(ctx, s.checkIns, u.ID(), worklog.SourceApp, worklog.Entry{
			Ticket:   key,
			Title:    title,
			Round:    round,
			Phase:    phase,
			Status:   status,
			Hours:    cmd.Hours,
			Note:     note,
			WorkedAt: now,
		}, now)
		if err != nil {
			return err
		}
		if err := saveShifting(ctx, s.checkIns, []*worklog.CheckIn{c}); err != nil {
			return err
		}
	} else {
		var ofStep []*worklog.CheckIn
		for _, c := range existing {
			if c.Phase() == phase && c.Round() == round {
				ofStep = append(ofStep, c)
			}
		}
		noted, latest := latestOf(ofStep), latestOf(existing)
		if noted == nil {
			noted = latest
		}
		e := noted.Entry()
		e.Note = addNote(e.Note, note)
		if noted == latest {
			e.Status = status
		}
		if err := noted.Revise(e); err != nil {
			return err
		}
		save := []*worklog.CheckIn{noted}
		if noted != latest {
			e := latest.Entry()
			e.Status = status
			if err := latest.Revise(e); err != nil {
				return err
			}
			save = append(save, latest)
		}
		if err := s.checkIns.SaveChanges(ctx, save, nil); err != nil {
			return err
		}
	}

	// Approval ends the ticket today, keeping the start date it had
	if status == worklog.StatusDone {
		if err := s.finishToday(ctx, u.ID(), key, now); err != nil {
			return err
		}
	}
	return s.progress.Save(ctx, next)
}

// finishToday ends the ticket today, keeping the start date it had.
func (s *CheckInService) finishToday(ctx context.Context, userID user.ID, key worklog.TicketKey, now time.Time) error {
	dates, err := s.dates.FindByUser(ctx, userID)
	if err != nil {
		return err
	}
	today := calendar.DayOf(now)
	d, err := worklog.NewTicketDates(userID, key, dates[key].Start, &today)
	if err != nil {
		return err
	}
	return s.dates.Save(ctx, d)
}

// clearEndUnlessDone takes the end date off a ticket that isn't done (e.g.
// set back from Done to Ongoing), keeping its start date: an open ticket
// hasn't ended. The ticket's status is its latest check-in's (a planned
// ticket's own), so time logged on an earlier day doesn't reopen it. A done
// ticket's dates stay as they are.
func (s *CheckInService) clearEndUnlessDone(ctx context.Context, userID user.ID, key worklog.TicketKey) error {
	checkIns, err := s.checkIns.FindByTicket(ctx, userID, key)
	if err != nil {
		return err
	}
	var status worklog.Status
	if latest := latestOf(checkIns); latest != nil {
		status = latest.Status()
	} else {
		p, err := s.planned.Find(ctx, userID, key)
		if errors.Is(err, worklog.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		status = p.Status
	}
	if status == worklog.StatusDone {
		return nil
	}
	dates, err := s.dates.FindByUser(ctx, userID)
	if err != nil {
		return err
	}
	d, ok := dates[key]
	if !ok || d.End == nil {
		return nil
	}
	d.End = nil
	return s.dates.Save(ctx, d)
}

// ticketWork loads the user's check-ins on a ticket, which must have some.
func (s *CheckInService) ticketWork(ctx context.Context, u *user.User, projectName, repo, ticket string) (worklog.TicketKey, []*worklog.CheckIn, error) {
	key, err := worklog.ParseTicketKey(projectName, repo, ticket)
	if err != nil {
		return worklog.TicketKey{}, nil, err
	}
	existing, err := s.checkIns.FindByTicket(ctx, u.ID(), key)
	if err != nil {
		return key, nil, err
	}
	if len(existing) == 0 {
		return key, nil, worklog.ErrNotFound
	}
	return key, existing, nil
}

func latestOf(checkIns []*worklog.CheckIn) *worklog.CheckIn {
	var latest *worklog.CheckIn
	for _, c := range checkIns {
		if latest == nil || latest.Before(c) {
			latest = c
		}
	}
	return latest
}

func latestRoundOf(checkIns []*worklog.CheckIn) int {
	round := 1
	for _, c := range checkIns {
		round = max(round, c.Round())
	}
	return round
}

// roundOrLatest is the round asked for, or the ticket's latest when it's 0.
func roundOrLatest(round int, checkIns []*worklog.CheckIn) int {
	if round == 0 {
		return latestRoundOf(checkIns)
	}
	return round
}

// addNote keeps earlier outcomes on a check-in, e.g. "Changes requested: use
// v2 · No changes", unless that would grow past what a note holds.
func addNote(existing, note string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" || existing == note || len(existing)+len(note) > 900 {
		return note
	}
	return existing + " · " + note
}

// recordFlowOutcome finishes a step of a typed (SPinno) ticket and moves it
// on: a review approved, or changes requested (back to an earlier step, or a
// review-back round); an action done; or work done without the timer. Time
// typed in goes on the step's phase as a check-in noting the result. Past
// the last step, the ticket is done today.
func (s *CheckInService) recordFlowOutcome(ctx context.Context, u *user.User, key worklog.TicketKey, existing []*worklog.CheckIn, cmd OutcomeCommand) error {
	ticketType, err := s.typeOf(ctx, u.ID(), key)
	if err != nil {
		return err
	}
	if ticketType == "" {
		return fmt.Errorf("%w: only SPinno tickets have steps of their own", worklog.ErrInvalid)
	}
	round := roundOrLatest(cmd.Round, existing)
	w := worklog.WorkflowOf(ticketType)
	step, i := w.Find(round, cmd.Step)
	if i < 0 {
		return fmt.Errorf("%w: %q isn't a step of %s tickets", worklog.ErrInvalid, cmd.Step, ticketType.Label())
	}
	if cmd.Hours < 0 || (cmd.Hours > 0 && step.Kind == worklog.StepAction) {
		return fmt.Errorf("%w: %s takes no time", worklog.ErrInvalid, step.Label)
	}

	var next worklog.Progress
	result := ""
	switch {
	case step.Kind == worklog.StepReview && cmd.Outcome == OutcomeApproved:
		result = step.ApproveLabel
		if result == "" {
			result = "Approved"
		}
		next, err = worklog.AfterStep(u.ID(), key, ticketType, round, step.Key)
	case step.Kind == worklog.StepReview && cmd.Outcome == OutcomeChangesRequested && step.ChangesTo != "":
		result = "Changes requested"
		back, _ := w.Find(round, step.ChangesTo)
		next, err = worklog.FlowProgress(u.ID(), key, round, back)
	case step.Kind == worklog.StepReview && cmd.Outcome == OutcomeChangesRequested && step.ChangesNewRound:
		result = "Changes requested"
		next, err = worklog.FlowProgress(u.ID(), key, round+1, w.Again[0])
	case step.Kind == worklog.StepReview && cmd.Outcome == OutcomeChangesRequested && step.ChangesRestart:
		// A review-back too, redoing the original steps from the issue
		result = "Changes requested"
		next, err = worklog.FlowProgress(u.ID(), key, round+1, w.Redo[0])
	case step.Kind != worklog.StepReview && cmd.Outcome == OutcomeDone:
		result = "Done"
		next, err = worklog.AfterStep(u.ID(), key, ticketType, round, step.Key)
	default:
		return fmt.Errorf("%w: %q isn't a result of %s", worklog.ErrInvalid, cmd.Outcome, step.Label)
	}
	if err != nil {
		return err
	}
	// Only what's worth reading later goes in the ticket's note: changes
	// asked for, and comments; approvals and steps done are in the steps
	note := ""
	if cmd.Outcome == OutcomeChangesRequested {
		note = step.Label + ": " + result
	}
	if comment := strings.TrimSpace(cmd.Comment); comment != "" {
		if note == "" {
			note = step.Label
		}
		note += ": " + comment
	}
	status := worklog.StatusOngoing
	if next.Finished {
		status = worklog.StatusDone
	}

	now := time.Now()
	switch {
	case cmd.Hours > 0:
		// The time typed in, noting the result
		title := strings.TrimSpace(cmd.Title)
		if latest := latestOf(existing); latest != nil {
			title = latest.Title()
		} else if planned, err := s.planned.Find(ctx, u.ID(), key); err == nil {
			title = planned.Title
		}
		c, err := newCheckIn(ctx, s.checkIns, u.ID(), worklog.SourceApp, worklog.Entry{
			Ticket: key, Title: title, Round: round, Phase: step.Phase, Status: status,
			Hours: cmd.Hours, Note: note, WorkedAt: now,
		}, now)
		if err != nil {
			return err
		}
		if err := saveShifting(ctx, s.checkIns, []*worklog.CheckIn{c}); err != nil {
			return err
		}
	case len(existing) > 0:
		// No time: the result is noted on the ticket's latest check-in, which
		// carries its status
		latest := latestOf(existing)
		e := latest.Entry()
		changed := false
		if note != "" {
			e.Note, changed = addNote(e.Note, note), true
		}
		if status == worklog.StatusDone || e.Status == worklog.StatusNotStarted {
			e.Status, changed = status, true
		}
		if changed {
			if err := latest.Revise(e); err != nil {
				return err
			}
			if err := s.checkIns.SaveChanges(ctx, []*worklog.CheckIn{latest}, nil); err != nil {
				return err
			}
		}
	default:
		// No time on it yet: a ticket added to the table keeps its own status
		planned, err := s.planned.Find(ctx, u.ID(), key)
		if errors.Is(err, worklog.ErrNotFound) {
			return fmt.Errorf("%w: add the ticket to the table first", worklog.ErrInvalid)
		}
		if err != nil {
			return err
		}
		if status == worklog.StatusDone || planned.Status == worklog.StatusNotStarted {
			planned.Status = status
			if err := s.planned.Save(ctx, planned); err != nil {
				return err
			}
		}
	}

	if next.Finished {
		if err := s.finishToday(ctx, u.ID(), key, now); err != nil {
			return err
		}
	}
	return s.progress.Save(ctx, next)
}
