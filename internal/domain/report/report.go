package report

import (
	"errors"
	"fmt"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

var (
	ErrNotFound = errors.New("report not found")
	ErrInvalid  = errors.New("invalid report")
)

// ID uniquely identifies a Report.
type ID string

type Kind string

const (
	KindDaily  Kind = "daily"
	KindWeekly Kind = "weekly"
)

func ParseKind(s string) (Kind, error) {
	switch k := Kind(s); k {
	case KindDaily, KindWeekly:
		return k, nil
	}
	return "", fmt.Errorf("%w: kind must be daily or weekly", ErrInvalid)
}

// PeriodFor is the period a report of this kind covers for the given day.
func (k Kind) PeriodFor(d calendar.Day) calendar.Period {
	if k == KindWeekly {
		return calendar.WeekOf(d)
	}
	return calendar.SingleDay(d)
}

// GeneratedBySchedule marks reports the scheduler made rather than a person.
const GeneratedBySchedule = "Scheduled"

// Report is a saved snapshot: later edits to check-ins don't change it until
// it is generated again.
type Report struct {
	id          ID
	kind        Kind
	period      calendar.Period
	data        Data
	generatedBy string
	generatedAt time.Time
}

func New(id ID, kind Kind, period calendar.Period, data Data, generatedBy string, generatedAt time.Time) *Report {
	return &Report{id: id, kind: kind, period: period, data: data, generatedBy: generatedBy, generatedAt: generatedAt}
}

func (r *Report) ID() ID                  { return r.id }
func (r *Report) Kind() Kind              { return r.kind }
func (r *Report) Period() calendar.Period { return r.period }
func (r *Report) Data() Data              { return r.data }
func (r *Report) GeneratedBy() string     { return r.generatedBy }
func (r *Report) GeneratedAt() time.Time  { return r.generatedAt }

// Data is the content of a report: one table per sheet.
type Data struct {
	Tickets  []TicketRow
	Projects []ProjectRow
	People   []PersonRow
}

// TicketRow is one ticket worked on during the period. Phase, status and
// totals are as of the end of the period.
type TicketRow struct {
	Ticket      worklog.TicketKey
	Title       string
	Assignees   []string
	Phase       worklog.Phase
	Status      worklog.Status
	PeriodHours float64
	TotalHours  float64
	PhaseHours  map[worklog.Phase]float64 // all time, the ticket's timeline
	// ReviewBacks is how many times a review sent the ticket back: the
	// highest round minus one.
	ReviewBacks int
	// Rounds breaks the timeline down by round, original work first. Their
	// hours add up to PhaseHours and TotalHours.
	Rounds []RoundRow
	// Step is the phase the user works on next in the progress steps, "" once
	// the ticket is finished, and StepRound the round it is in. Only set on a
	// user's own week.
	Step      worklog.Phase
	StepRound int
	// Type is a SPinno ticket's type, and FlowStep the step of that type's
	// workflow the user is on ("" once done); both empty for BlackBox. Only
	// set on a user's own week.
	Type     worklog.TicketType
	FlowStep string
	// MadeParent: a child spec made a parent by dragging, which can be
	// dragged back to being a child (one added as a parent can't)
	MadeParent bool
	// StartDate is the first day anyone worked on the ticket. EndDate is the
	// day it was marked done, or nil while it is still open.
	StartDate  calendar.Day
	EndDate    *calendar.Day
	LastUpdate time.Time
	Note       string
	// AddedAt is when the ticket first went into the user's timeline: added
	// without time, or its first time logged. Only set on a user's own week
	// (left out of saved reports).
	AddedAt time.Time `json:",omitzero"`
	// Parent is the number of the spec ticket a SPinno backend or web ticket
	// sits under, "" for none. Only set on a user's own week.
	Parent string `json:",omitempty"`
	// ChildHours is all the time on the backend and web tickets under a spec
	// ticket, on top of its own TotalHours. Only set on a user's own week.
	ChildHours float64 `json:",omitzero"`
	// ChildPhaseHours splits ChildHours by phase, and ChildCount is how many
	// tickets sit under the spec (with time or not). Only on a user's week.
	ChildPhaseHours map[worklog.Phase]float64 `json:",omitempty"`
	ChildCount      int                       `json:",omitzero"`
	// ChildReviewBacks adds up the review-backs of the tickets under a parent
	// (each one's, from all of its rounds). Only on a user's week.
	ChildReviewBacks int `json:",omitzero"`
	// StartedAt is when work on the ticket began: its first time logged (a
	// spec ticket's, the first on the tickets under it). Only on a user's week.
	StartedAt time.Time `json:",omitzero"`
}

// RoundRow is one pass through the phases: the original work, or the rework
// after a review sent the ticket back.
type RoundRow struct {
	Round      int
	PhaseHours map[worklog.Phase]float64
	Hours      float64
	FirstDay   calendar.Day // first and last day anyone worked in the round
	LastDay    calendar.Day
	// StartedAt is when the round's first time was logged. Only set on a
	// user's own week.
	StartedAt time.Time `json:",omitzero"`
}

// ProjectRow sums up one project's tickets for the period.
type ProjectRow struct {
	Project    project.Project
	Tickets    int
	ByStatus   map[worklog.Status]int
	PhaseHours map[worklog.Phase]float64 // this period only
	Hours      float64
}

// PersonRow shows how much each person logged and whether they kept up with
// their check-ins.
type PersonRow struct {
	UserID        user.ID
	Name          string
	Team          project.Project
	CheckIns      int
	Hours         float64
	ExpectedHours float64
	// MissedDays are past weekdays in the period with no check-ins.
	MissedDays []calendar.Day
}
