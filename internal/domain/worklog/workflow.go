package worklog

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
)

// TicketType is what kind of SPinno ticket it is, which decides its steps:
// a spec ticket is an investigation or an integration test; a backend or web
// ticket is an implementation or a review. BlackBox tickets have no type and
// keep the six phases as their steps.
type TicketType string

const (
	TypeInvestigation  TicketType = "investigation"
	TypeIntegration    TicketType = "integration_testing"
	TypeImplementation TicketType = "implementation"
	TypeReview         TicketType = "review"
)

var typeLabels = map[TicketType]string{
	TypeInvestigation:  "Investigation",
	TypeIntegration:    "Integration Testing",
	TypeImplementation: "Implementation",
	TypeReview:         "Review",
}

func (t TicketType) Label() string { return typeLabels[t] }

// TypesFor lists the types a repo's tickets can have; none outside SPinno.
func TypesFor(repo project.Repo) []TicketType {
	switch repo {
	case project.RepoSpec:
		return []TicketType{TypeInvestigation, TypeIntegration}
	case project.RepoBackend, project.RepoWeb:
		return []TicketType{TypeImplementation, TypeReview}
	}
	return nil
}

// DefaultType is a ticket's type when none was chosen (e.g. tickets made
// before types): the first its repo offers, or "" for none.
func DefaultType(repo project.Repo) TicketType {
	if types := TypesFor(repo); len(types) > 0 {
		return types[0]
	}
	return ""
}

// ParseType checks a type fits the ticket's repo. "" picks the default.
func ParseType(repo project.Repo, s string) (TicketType, error) {
	if s == "" {
		return DefaultType(repo), nil
	}
	for _, t := range TypesFor(repo) {
		if string(t) == s {
			return t, nil
		}
	}
	if len(TypesFor(repo)) == 0 {
		return "", fmt.Errorf("%w: only SPinno tickets have a type", ErrInvalid)
	}
	return "", fmt.Errorf("%w: a %s ticket can't be %q", ErrInvalid, repo.Label(), s)
}

// StepKind is how a step is filled in.
type StepKind string

const (
	// StepWork is the user's own work, timed with the timer
	StepWork StepKind = "work"
	// StepReview is someone else's review: its time typed in (optional, it
	// depends on them) and its result, approved or changes requested
	StepReview StepKind = "review"
	// StepAction is a quick thing to do, marked done with one click, no time
	StepAction StepKind = "action"
)

// FlowStep is one step of a ticket type. Its time goes on Phase, one of the
// table's six columns (an action takes none: its Phase is only a place
// holder). Approving a review, finishing work or doing an action moves on to
// the next step, or finishes the ticket after the last.
type FlowStep struct {
	Key   string
	Label string
	Who   string // who does it: "You", the leads, Cho-san
	Kind  StepKind
	Phase Phase
	Hint  string
	// A review's other result: back to an earlier step of the same round
	// (ChangesTo), or a review-back, the next round: the type's review-back
	// steps (ChangesNewRound) or its original steps again (ChangesRestart).
	// ChangesTo with ChangesNewRound: the review-back starts at that step.
	// None: it can only be approved.
	ChangesTo       string
	ChangesNewRound bool
	ChangesRestart  bool
	// What approving is called, e.g. "Review done"; "" for "Approved"
	ApproveLabel string
	// A third result that keeps the step open, e.g. "Still waiting" on a
	// BlackBox confirmation; "" for none
	WaitLabel string
}

// Workflow is a type's steps: First for the original work, Again for a
// review-back round after Cho-san asked for changes (nil when there are
// none), and Redo for one after the leads asked for changes on the issue:
// the original steps again, from revising the issue. Redo's keys are its own
// (redo_...), so a round's step says which of the two it follows.
type Workflow struct {
	First []FlowStep
	Again []FlowStep
	Redo  []FlowStep
}

// redoPrefix starts the keys of a Redo round's steps
const redoPrefix = "redo_"

// redoOf is the original steps as a review-back after the leads asked for
// changes: the issue revised first, then everything after it as before.
func redoOf(first []FlowStep) []FlowStep {
	redo := make([]FlowStep, len(first))
	for i, s := range first {
		s.Key = redoPrefix + s.Key
		if s.ChangesTo != "" {
			s.ChangesTo = redoPrefix + s.ChangesTo
		}
		if i == 0 {
			s.Label, s.Hint = "Revise issue", "Edit the issue as the leads asked."
		}
		redo[i] = s
	}
	return redo
}

const (
	leads  = "Sir Markrey & Sir Jerwine"
	choSan = "Cho-san"
	you    = "You"
)

// The opening every type shares: draft the issue, then the leads review it.
func opening() []FlowStep {
	return []FlowStep{
		{Key: "draft", Label: "Draft issue", Who: you, Kind: StepWork, Phase: PhaseInvestigation,
			Hint: "Write the issue as a .md file."},
		// Changes asked for here are a review-back too: the issue revised,
		// then the steps after it again
		{Key: "lead_review", Label: "Leads review", Who: leads, Kind: StepReview, Phase: PhaseConfirmation,
			Hint: "The leads review the drafted issue.", ChangesRestart: true},
	}
}

// removedSteps are steps that no longer exist, by the step they followed,
// so a ticket saved on one carries on from the step that came after it.
var removedSteps = map[string]string{
	"file": "lead_review", // File ticket, after the leads review
}

// mergedSteps are a type's steps that became part of another, by the step
// they're now part of, so a ticket saved on one carries on from there.
var mergedSteps = map[TicketType]map[string]string{
	// The leads' and Cho-san's reviews of the pull request are one review
	TypeImplementation: {"lead_pr_review": "pr_review", "cho_review": "pr_review"},
}

// StepNow is where a ticket saved at a step stands today: that step, or,
// if the step was removed from its type, the step after where it was. ok
// is false if neither is a step of the round.
func StepNow(userID user.ID, ticket TicketKey, t TicketType, round int, key string) (Progress, bool) {
	w := WorkflowOf(t)
	if s, i := w.Find(round, key); i >= 0 {
		p, err := FlowProgress(userID, ticket, round, s)
		return p, err == nil
	}
	if into, ok := mergedSteps[t][key]; ok {
		if s, i := w.Find(round, into); i >= 0 {
			p, err := FlowProgress(userID, ticket, round, s)
			return p, err == nil
		}
	}
	if before, ok := removedSteps[key]; ok {
		p, err := AfterStep(userID, ticket, t, round, before)
		return p, err == nil
	}
	return Progress{}, false
}

var workflows = map[TicketType]Workflow{
	TypeInvestigation: {
		First: append(opening(),
			FlowStep{Key: "cho_review", Label: "Cho-san review", Who: choSan, Kind: StepReview, Phase: PhaseReview,
				Hint: "Cho-san reviews the issue. Approving finishes the ticket.", ChangesNewRound: true},
		),
		Again: []FlowStep{
			{Key: "revise_issue", Label: "Revise issue", Who: you, Kind: StepWork, Phase: PhaseInvestigation,
				Hint: "Edit the issue as Cho-san asked."},
			{Key: "cho_review", Label: "Cho-san review", Who: choSan, Kind: StepReview, Phase: PhaseReview,
				Hint: "Cho-san reviews the revised issue. Approving finishes the ticket.", ChangesNewRound: true},
		},
	},
	// Planning comes once the leads have confirmed the spec, and again first
	// thing in a review-back: changes asked for are planned before coding
	TypeImplementation: {
		First: append(opening(),
			FlowStep{Key: "plan", Label: "Plan", Who: you, Kind: StepWork, Phase: PhasePlanning,
				Hint: "Plan how you’ll change the code, now that the leads have confirmed the spec."},
			FlowStep{Key: "code", Label: "Code", Who: you, Kind: StepWork, Phase: PhaseCoding,
				Hint: "Implement the change locally."},
			FlowStep{Key: "test", Label: "Test", Who: you, Kind: StepWork, Phase: PhaseTesting,
				Hint: "Check it before pushing."},
			FlowStep{Key: "push_pr", Label: "Push & open PR", Who: you, Kind: StepAction, Phase: PhasePlanning,
				Hint: "Push to the repo and open the pull request."},
			// One review of the pull request, the leads' then Cho-san's, its
			// time (both together) on Review
			FlowStep{Key: "pr_review", Label: "PR review", Who: leads + ", then " + choSan, Kind: StepReview,
				Phase: PhaseReview, Hint: "The leads, then Cho-san, review the pull request. Type the time both took.",
				ChangesNewRound: true},
			FlowStep{Key: "close", Label: "Close PR & issue", Who: you, Kind: StepAction, Phase: PhasePlanning,
				Hint: "Everything is resolved: close the PR and the issue. This finishes the ticket."},
		),
		Again: []FlowStep{
			{Key: "plan", Label: "Plan", Who: you, Kind: StepWork, Phase: PhasePlanning,
				Hint: "Plan how you’ll change the code for what the review asked."},
			{Key: "revise_code", Label: "Revise code", Who: you, Kind: StepWork, Phase: PhaseCoding,
				Hint: "Change the code as Cho-san asked."},
			{Key: "test", Label: "Test", Who: you, Kind: StepWork, Phase: PhaseTesting,
				Hint: "Check it again before pushing."},
			{Key: "cho_review", Label: "Cho-san review", Who: choSan, Kind: StepReview, Phase: PhaseReview,
				Hint: "Cho-san reviews the changes.", ChangesNewRound: true},
			{Key: "close", Label: "Close PR & issue", Who: you, Kind: StepAction, Phase: PhasePlanning,
				Hint: "Everything is resolved: close the PR and the issue. This finishes the ticket."},
		},
	},
	TypeReview: {
		First: append(opening(),
			FlowStep{Key: "cho_review", Label: "Cho-san's review", Who: choSan, Kind: StepReview, Phase: PhaseReview,
				Hint: "Cho-san does the review. Mark it done when Cho-san has finished.", ApproveLabel: "Review done"},
		),
	},
	// Like an implementation: planned once the leads have confirmed the issue,
	// and planned again when Cho-san asks for changes
	TypeIntegration: {
		First: append(opening(),
			FlowStep{Key: "plan", Label: "Plan", Who: you, Kind: StepWork, Phase: PhasePlanning,
				Hint: "Plan what you’ll test on staging, now that the leads have confirmed the issue."},
			FlowStep{Key: "staging_test", Label: "Test on staging", Who: you, Kind: StepWork, Phase: PhaseTesting,
				Hint: "Test the changes in the staging environment."},
			FlowStep{Key: "cho_results", Label: "Cho-san reviews results", Who: choSan, Kind: StepReview,
				Phase: PhaseReview, Hint: "Cho-san reviews the test results. Approving finishes the ticket.",
				ChangesNewRound: true},
		),
		Again: []FlowStep{
			{Key: "plan", Label: "Plan", Who: you, Kind: StepWork, Phase: PhasePlanning,
				Hint: "Plan the retest for what Cho-san asked."},
			{Key: "retest", Label: "Retest on staging", Who: you, Kind: StepWork, Phase: PhaseTesting,
				Hint: "Test again on staging after the requested changes."},
			{Key: "cho_results", Label: "Cho-san reviews results", Who: choSan, Kind: StepReview, Phase: PhaseReview,
				Hint: "Cho-san reviews the new results. Approving finishes the ticket.", ChangesNewRound: true},
		},
	},
}

// BlackBoxWorkflow is a BlackBox ticket's six phases as steps, so the
// progress form is the same one typed tickets use. The timed phases are your
// own work; the confirmation and the final review are typed in with an
// outcome. These are the moves CheckInService.RecordOutcome makes: either
// review asking for changes starts a review-back, from the spec after the
// confirmation and from planning after the final review; a confirmation can
// also wait for its reply. A review-back round shows all six phases.
func BlackBoxWorkflow() Workflow {
	steps := make([]FlowStep, 0, len(Phases()))
	for _, p := range Phases() {
		s := FlowStep{Key: string(p), Label: p.ShortLabel(), Who: you, Kind: StepWork, Phase: p, Hint: p.Hint()}
		switch p {
		case PhaseConfirmation:
			s.Who, s.Kind = "SE / managers", StepReview
			s.ChangesTo, s.ChangesNewRound = string(PhaseInvestigation), true
			s.ApproveLabel, s.WaitLabel = "No changes", "Still waiting"
		case PhaseReview:
			s.Who, s.Kind = "Reviewer", StepReview
			s.ChangesTo, s.ChangesNewRound = string(PhasePlanning), true
		}
		steps = append(steps, s)
	}
	return Workflow{First: steps, Again: slices.Clone(steps)}
}

// Types lists every type, for the options the forms offer.
func Types() []TicketType {
	return []TicketType{TypeInvestigation, TypeIntegration, TypeImplementation, TypeReview}
}

// WorkflowOf is a type's steps.
func WorkflowOf(t TicketType) Workflow {
	w := workflows[t]
	w.Redo = redoOf(w.First)
	return w
}

// Steps are the steps of a round: the original work's, or a review-back's
// (Cho-san's kind; a round at a redo_ step follows Redo, see StepsAt).
func (w Workflow) Steps(round int) []FlowStep {
	if round > 1 {
		return w.Again
	}
	return w.First
}

// StepsAt are the steps of the round that a step with that key is in: a
// review-back the leads started (Redo) or Cho-san did (Again).
func (w Workflow) StepsAt(round int, key string) []FlowStep {
	if round > 1 && strings.HasPrefix(key, redoPrefix) {
		return w.Redo
	}
	return w.Steps(round)
}

// Find is the step with that key in the round, and its place; -1 if none.
func (w Workflow) Find(round int, key string) (FlowStep, int) {
	for i, s := range w.StepsAt(round, key) {
		if s.Key == key {
			return s, i
		}
	}
	return FlowStep{}, -1
}

// FlowProgress is the user at a step of a typed ticket's workflow.
func FlowProgress(userID user.ID, ticket TicketKey, round int, s FlowStep) (Progress, error) {
	p, err := NewProgress(userID, ticket, round, s.Phase)
	if err != nil {
		return Progress{}, err
	}
	p.FlowStep = s.Key
	return p, nil
}

// AfterStep is where a typed ticket goes once a step is done (work finished,
// a review approved, an action done): the next step of the round, or
// finished after the last.
func AfterStep(userID user.ID, ticket TicketKey, t TicketType, round int, key string) (Progress, error) {
	w := WorkflowOf(t)
	_, i := w.Find(round, key)
	if i < 0 {
		return Progress{}, fmt.Errorf("%w: %q isn't a step of %s tickets", ErrInvalid, key, t.Label())
	}
	steps := w.StepsAt(round, key)
	if i+1 < len(steps) {
		return FlowProgress(userID, ticket, round, steps[i+1])
	}
	return FinishedProgress(userID, ticket, round), nil
}

// TypeRepository keeps, per user, the type chosen for each SPinno ticket.
type TypeRepository interface {
	// FindByUser returns the type of each of the user's tickets that has one.
	// A child spec made a parent by dragging has none while it's a parent.
	FindByUser(ctx context.Context, userID user.ID) (map[TicketKey]TicketType, error)
	Set(ctx context.Context, userID user.ID, ticket TicketKey, t TicketType) error
	// FindMadeParents returns the tickets made parents by dragging: they
	// keep their type, for when they're dragged back to being children.
	FindMadeParents(ctx context.Context, userID user.ID) (map[TicketKey]bool, error)
	SetMadeParent(ctx context.Context, userID user.ID, ticket TicketKey, made bool) error
}

// IsParentLevel reports whether the ticket is a parent ticket, which has no
// type: a spec with tickets under it, or one under no spec that was added
// as a parent (with no type). A child spec added before its parent spec is
// known has its type, so it isn't one.
func IsParentLevel(ticket TicketKey, parents map[TicketKey]TicketKey, types map[TicketKey]TicketType) bool {
	if ticket.Repo != project.RepoSpec {
		return false
	}
	for _, above := range parents {
		if above == ticket {
			return true
		}
	}
	_, under := parents[ticket]
	_, typed := types[ticket]
	return !under && !typed
}

// TypeOf is a ticket's type: the one chosen, or its repo's default. A parent
// ticket has none, so it keeps the six phases as its steps (and gets one if
// it's put under a spec).
func TypeOf(types map[TicketKey]TicketType, parents map[TicketKey]TicketKey, ticket TicketKey) TicketType {
	if IsParentLevel(ticket, parents, types) {
		return ""
	}
	if t, ok := types[ticket]; ok {
		return t
	}
	return DefaultType(ticket.Repo)
}
