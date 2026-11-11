package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/report"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// clockSkew allows for a browser clock that runs slightly ahead.
const clockSkew = 5 * time.Minute

// CheckInCommand is a check-in as typed in the form or a spreadsheet row.
type CheckInCommand struct {
	Project  string
	Repo     string // SPinno: spec, backend or web; "" for none
	Ticket   string
	Title    string
	Round    int // 0 means the original work
	Phase    string
	Status   string
	Hours    float64
	Note     string
	WorkedAt time.Time
}

func (cmd CheckInCommand) entry(now time.Time) (worklog.Entry, error) {
	key, err := worklog.ParseTicketKey(cmd.Project, cmd.Repo, cmd.Ticket)
	if err != nil {
		return worklog.Entry{}, err
	}
	phase, err := worklog.ParsePhase(cmd.Phase)
	if err != nil {
		return worklog.Entry{}, err
	}
	status, err := worklog.ParseStatus(cmd.Status)
	if err != nil {
		return worklog.Entry{}, err
	}
	if cmd.WorkedAt.After(now.Add(clockSkew)) {
		return worklog.Entry{}, fmt.Errorf("%w: date and time can't be in the future", worklog.ErrInvalid)
	}
	return worklog.Entry{
		Ticket:   key,
		Title:    cmd.Title,
		Round:    cmd.Round,
		Phase:    phase,
		Status:   status,
		Hours:    cmd.Hours,
		Note:     cmd.Note,
		WorkedAt: cmd.WorkedAt,
	}, nil
}

// TicketWorkCommand logs time on several phases of one ticket at once, like
// a row of the team's progress sheet. Phases with no hours are skipped, since
// some aren't done yet.
type TicketWorkCommand struct {
	Project string
	Repo    string // SPinno: spec, backend or web; "" for none
	Ticket  string
	Title   string
	Round   int // which pass through the phases; 0 means the original work
	Status  string
	// YYYY-MM-DD, as on the progress sheet. Both blank leaves the ticket's
	// dates as they are; a blank end means not finished.
	StartDate  string
	EndDate    string
	Note       string
	WorkedAt   time.Time
	PhaseHours map[string]float64 // keyed by phase, e.g. "coding"
}

// checkIn is the single-phase command for one phase of the work.
func (cmd TicketWorkCommand) checkIn(phase worklog.Phase, hours float64) CheckInCommand {
	return CheckInCommand{
		Project:  cmd.Project,
		Repo:     cmd.Repo,
		Ticket:   cmd.Ticket,
		Title:    cmd.Title,
		Round:    cmd.Round,
		Phase:    string(phase),
		Status:   cmd.Status,
		Hours:    hours,
		Note:     cmd.Note,
		WorkedAt: cmd.WorkedAt,
	}
}

// ticketDates reads the start and end dates given with the work. The bool is
// false when neither was given.
func (cmd TicketWorkCommand) ticketDates(userID user.ID) (worklog.TicketDates, bool, error) {
	if cmd.StartDate == "" && cmd.EndDate == "" {
		return worklog.TicketDates{}, false, nil
	}
	key, err := worklog.ParseTicketKey(cmd.Project, cmd.Repo, cmd.Ticket)
	if err != nil {
		return worklog.TicketDates{}, false, err
	}
	start, err := optionalDay(cmd.StartDate)
	if err != nil {
		return worklog.TicketDates{}, false, err
	}
	end, err := optionalDay(cmd.EndDate)
	if err != nil {
		return worklog.TicketDates{}, false, err
	}
	dates, err := worklog.NewTicketDates(userID, key, start, end)
	return dates, err == nil, err
}

// workedOutStep is where a ticket stands in the steps when none was saved:
// the phase after the latest one, in the latest round. A done ticket is
// finished, a review waits for its outcome, and a confirmation for its reply
// until it is no longer waiting.
func workedOutStep(t report.TicketRow) (worklog.Phase, int) {
	round := 1
	for _, r := range t.Rounds {
		round = max(round, r.Round)
	}
	switch {
	case t.Status == worklog.StatusDone:
		return "", round
	case t.Phase == worklog.PhaseReview:
		return worklog.PhaseReview, round
	case t.Phase == worklog.PhaseConfirmation && t.Status == worklog.StatusWaiting:
		return worklog.PhaseConfirmation, round
	}
	return t.Phase.Next(), round
}

// workedOutFlowStep is where a typed ticket stands in its type's steps when
// none was saved (e.g. time typed into the table): the step after the last
// one with time in the latest round, or the first; "" once done.
func workedOutFlowStep(t report.TicketRow, tt worklog.TicketType) (string, int) {
	round := max(1, t.ReviewBacks+1)
	if t.Status == worklog.StatusDone {
		return "", round
	}
	var hours map[worklog.Phase]float64
	for _, r := range t.Rounds {
		if r.Round == round {
			hours = r.PhaseHours
		}
	}
	steps := worklog.WorkflowOf(tt).Steps(round)
	next := 0
	for i, s := range steps {
		if s.Kind != worklog.StepAction && hours[s.Phase] > 0 {
			next = i + 1
		}
	}
	if next >= len(steps) {
		next = len(steps) - 1
	}
	if next < 0 {
		return "", round
	}
	return steps[next].Key, round
}

// withStartedRound adds the round the ticket's steps are in when no time has
// been logged in it yet: a review that asked for changes starts a review-back
// straight away, so it counts and shows (empty) before its first time.
func withStartedRound(t report.TicketRow, round int) report.TicketRow {
	if round-1 <= t.ReviewBacks {
		return t
	}
	t.ReviewBacks = round - 1
	rounds := make([]report.RoundRow, len(t.Rounds), len(t.Rounds)+1)
	copy(rounds, t.Rounds)
	t.Rounds = append(rounds, report.RoundRow{Round: round, PhaseHours: map[worklog.Phase]float64{}})
	return t
}

func optionalDay(s string) (*calendar.Day, error) {
	if s == "" {
		return nil, nil
	}
	d, err := calendar.ParseDay(s)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// MyWeek is everything a user's dashboard shows for one week.
type MyWeek struct {
	User     *user.User
	Period   calendar.Period
	CheckIns []*worklog.CheckIn
	Tickets  []report.TicketRow
	Person   *report.PersonRow
}

// CheckInService holds the use cases for a user logging their own work.
type CheckInService struct {
	checkIns worklog.Repository
	dates    worklog.TicketDatesRepository
	progress worklog.ProgressRepository
	planned  worklog.PlannedTicketRepository
	parents  worklog.ParentRepository
	types    worklog.TypeRepository
}

func NewCheckInService(
	checkIns worklog.Repository,
	dates worklog.TicketDatesRepository,
	progress worklog.ProgressRepository,
	planned worklog.PlannedTicketRepository,
	parents worklog.ParentRepository,
	types worklog.TypeRepository,
) *CheckInService {
	return &CheckInService{checkIns: checkIns, dates: dates, progress: progress, planned: planned, parents: parents, types: types}
}

// Week returns the user's check-ins and ticket summary for the week that day
// falls in.
func (s *CheckInService) Week(ctx context.Context, u *user.User, day string) (*MyWeek, error) {
	return weekFor(ctx, s.checkIns, s.dates, s.progress, s.planned, s.parents, s.types, u, day)
}

// weekFor builds u's dashboard. It is shared with MemberService so admins
// see exactly what the member sees.
func weekFor(
	ctx context.Context,
	checkIns worklog.Repository,
	ticketDates worklog.TicketDatesRepository,
	ticketProgress worklog.ProgressRepository,
	plannedTickets worklog.PlannedTicketRepository,
	ticketParents worklog.ParentRepository,
	ticketTypes worklog.TypeRepository,
	u *user.User,
	day string,
) (*MyWeek, error) {
	d, err := calendar.ParseDay(day)
	if err != nil {
		return nil, err
	}
	return timelineFor(ctx, checkIns, ticketDates, ticketProgress, plannedTickets, ticketParents, ticketTypes, u, calendar.WeekOf(d))
}

// timelineFor builds u's timeline for any period: a week for the dashboard,
// or all time for the team sheet.
func timelineFor(
	ctx context.Context,
	checkIns worklog.Repository,
	ticketDates worklog.TicketDatesRepository,
	ticketProgress worklog.ProgressRepository,
	plannedTickets worklog.PlannedTicketRepository,
	ticketParents worklog.ParentRepository,
	ticketTypes worklog.TypeRepository,
	u *user.User,
	period calendar.Period,
) (*MyWeek, error) {
	mine, err := checkIns.FindByUser(ctx, u.ID(), period)
	if err != nil {
		return nil, err
	}

	// Ticket totals cover the whole history, so load everything up to the
	// end of the week and keep only this user's part.
	all, err := checkIns.FindUntil(ctx, period.Last())
	if err != nil {
		return nil, err
	}
	var history []*worklog.CheckIn
	for _, c := range all {
		if c.OwnedBy(u.ID()) {
			history = append(history, c)
		}
	}
	data := report.Build(period, history, []*user.User{u}, calendar.DayOf(time.Now()))

	// Dates the user set on a ticket replace the worked-out ones
	dates, err := ticketDates.FindByUser(ctx, u.ID())
	if err != nil {
		return nil, err
	}
	// Tickets added without time yet, as rows with no hours
	planned, err := plannedTickets.FindByUser(ctx, u.ID())
	if err != nil {
		return nil, err
	}
	data.Tickets = withPlanned(u, period, history, dates, planned, data.Tickets)
	setAddedAt(data.Tickets, history, planned)
	// The spec ticket each backend or web ticket sits under
	parents, err := ticketParents.FindByUser(ctx, u.ID())
	if err != nil {
		return nil, err
	}
	// A spec ticket's total takes in the time on the tickets under it, all
	// of it as with its own total, whether or not they were worked this week
	childHours := map[worklog.TicketKey]float64{}
	childPhaseHours := map[worklog.TicketKey]map[worklog.Phase]float64{}
	// ...and through every spec above that, a spec under a spec having no
	// time of its own either
	for _, c := range history {
		for _, above := range worklog.Ancestors(c.Ticket(), parents) {
			childHours[above] += c.Hours()
			if childPhaseHours[above] == nil {
				childPhaseHours[above] = map[worklog.Phase]float64{}
			}
			childPhaseHours[above][c.Phase()] += c.Hours()
		}
	}
	// Every ticket under the spec that exists: time logged on it, or added to
	// the table without time. A link to a ticket never created (its form was
	// continued to the timer, then closed with nothing logged) doesn't count;
	// it waits in case that ticket is created later.
	exists := make(map[worklog.TicketKey]bool, len(planned))
	for _, c := range history {
		exists[c.Ticket()] = true
	}
	for key := range planned {
		exists[key] = true
	}
	childCount := map[worklog.TicketKey]int{}
	for child, parent := range parents {
		if exists[child] {
			childCount[parent]++
		}
	}
	// A parent's status is its tickets': each one's latest, worked this week
	// or not (one added without time, as it was added)
	lastCheckIn := map[worklog.TicketKey]*worklog.CheckIn{}
	for _, c := range history {
		if l, ok := lastCheckIn[c.Ticket()]; !ok || l.Before(c) {
			lastCheckIn[c.Ticket()] = c
		}
	}
	childStatuses := map[worklog.TicketKey][]worklog.Status{}
	for child, parent := range parents {
		if c, ok := lastCheckIn[child]; ok {
			childStatuses[parent] = append(childStatuses[parent], c.Status())
		} else if p, ok := planned[child]; ok {
			childStatuses[parent] = append(childStatuses[parent], p.Status)
		}
	}
	setStartedAt(data.Tickets, history, parents)
	for i, t := range data.Tickets {
		if parent, ok := parents[t.Ticket]; ok {
			data.Tickets[i].Parent = parent.Number
		}
		data.Tickets[i].ChildHours = math.Round(childHours[t.Ticket]*100) / 100
		data.Tickets[i].ChildCount = childCount[t.Ticket]
		if status := worklog.FamilyStatus(childStatuses[t.Ticket]); status != "" {
			data.Tickets[i].Status = status
		}
		if hours := childPhaseHours[t.Ticket]; hours != nil {
			for phase, h := range hours {
				hours[phase] = math.Round(h*100) / 100
			}
			data.Tickets[i].ChildPhaseHours = hours
		}
	}
	for i, t := range data.Tickets {
		if d, ok := dates[t.Ticket]; ok {
			if d.Start != nil {
				data.Tickets[i].StartDate = *d.Start
			}
			// Only a done ticket has ended; an end date kept from before it
			// was reopened doesn't show
			if t.Status == worklog.StatusDone {
				data.Tickets[i].EndDate = d.End
			} else {
				data.Tickets[i].EndDate = nil
			}
		}
	}

	progress, err := ticketProgress.FindByUser(ctx, u.ID())
	if err != nil {
		return nil, err
	}
	for i, t := range data.Tickets {
		if p, ok := progress[t.Ticket]; ok {
			data.Tickets[i].Step, data.Tickets[i].StepRound = p.Step, p.Round
			data.Tickets[i] = withStartedRound(data.Tickets[i], p.Round)
		} else if len(t.Rounds) > 0 {
			// A ticket added without time has no rounds: it keeps the first
			// step withPlanned gave it
			data.Tickets[i].Step, data.Tickets[i].StepRound = workedOutStep(t)
		}
	}

	// A parent's review-backs take in its tickets': each ticket's latest
	// round (with time, or a review-back started with none yet) less one,
	// whether or not it was worked this week
	latest := map[worklog.TicketKey]int{}
	for _, c := range history {
		latest[c.Ticket()] = max(latest[c.Ticket()], c.Round())
	}
	for key, p := range progress {
		if exists[key] {
			latest[key] = max(latest[key], p.Round)
		}
	}
	childReviewBacks := map[worklog.TicketKey]int{}
	for key, round := range latest {
		if round <= 1 {
			continue
		}
		for _, above := range worklog.Ancestors(key, parents) {
			childReviewBacks[above] += round - 1
		}
	}
	for i, t := range data.Tickets {
		data.Tickets[i].ChildReviewBacks = childReviewBacks[t.Ticket]
	}

	// A SPinno ticket's type, and where it stands in that type's steps
	types, err := ticketTypes.FindByUser(ctx, u.ID())
	if err != nil {
		return nil, err
	}
	madeParents, err := ticketTypes.FindMadeParents(ctx, u.ID())
	if err != nil {
		return nil, err
	}
	for i, t := range data.Tickets {
		data.Tickets[i].MadeParent = madeParents[t.Ticket]
		tt := worklog.TypeOf(types, parents, t.Ticket)
		if tt == "" {
			continue
		}
		data.Tickets[i].Type = tt
		if p, ok := progress[t.Ticket]; ok && p.Finished {
			data.Tickets[i].FlowStep, data.Tickets[i].StepRound = "", p.Round
			continue
		}
		if p, ok := progress[t.Ticket]; ok && p.FlowStep != "" {
			// A step since removed from the type moves on to the one after it
			if now, ok := worklog.StepNow(u.ID(), t.Ticket, tt, p.Round, p.FlowStep); ok {
				data.Tickets[i].FlowStep, data.Tickets[i].StepRound = now.FlowStep, now.Round
				continue
			}
		}
		data.Tickets[i].FlowStep, data.Tickets[i].StepRound = workedOutFlowStep(data.Tickets[i], tt)
	}

	week := &MyWeek{User: u, Period: period, CheckIns: mine, Tickets: data.Tickets}
	for i := range data.People {
		if data.People[i].UserID == u.ID() {
			week.Person = &data.People[i]
		}
	}
	return week, nil
}

// withPlanned adds the tickets u added without time to the week's rows: each
// shows, with no hours and on its first step, in every week from the one it
// was added in, until time is logged on it (then its check-ins are the row)
// or the week after its end date. Rows stay in the sheet's order.
func withPlanned(
	u *user.User,
	period calendar.Period,
	history []*worklog.CheckIn,
	dates map[worklog.TicketKey]worklog.TicketDates,
	planned map[worklog.TicketKey]worklog.PlannedTicket,
	rows []report.TicketRow,
) []report.TicketRow {
	if len(planned) == 0 {
		return rows
	}
	worked := make(map[worklog.TicketKey]bool, len(history))
	for _, c := range history {
		worked[c.Ticket()] = true
	}
	added := false
	for key, p := range planned {
		if worked[key] || period.Last().Before(p.AddedOn) {
			continue
		}
		if d, ok := dates[key]; ok && d.End != nil && d.End.Before(period.First()) {
			continue
		}
		rows = append(rows, report.TicketRow{
			Ticket:     key,
			Title:      p.Title,
			Assignees:  []string{u.Name()},
			Status:     p.Status,
			PhaseHours: map[worklog.Phase]float64{},
			Rounds:     []report.RoundRow{},
			Step:       worklog.PhaseInvestigation,
			StepRound:  1,
			Note:       p.Note,
		})
		added = true
	}
	if added {
		sort.SliceStable(rows, func(i, j int) bool {
			a, b := rows[i].Ticket, rows[j].Ticket
			if a.Project != b.Project {
				return a.Project < b.Project
			}
			return a.Number < b.Number
		})
	}
	return rows
}

// setStartedAt sets when work on each ticket, and on each of its rounds,
// began: the first time logged on it. A spec ticket's start is the first
// time logged on it or on any ticket under it.
func setStartedAt(rows []report.TicketRow, history []*worklog.CheckIn, parents map[worklog.TicketKey]worklog.TicketKey) {
	type roundKey struct {
		ticket worklog.TicketKey
		round  int
	}
	first := map[worklog.TicketKey]time.Time{}
	firstInRound := map[roundKey]time.Time{}
	earliest := func(m map[worklog.TicketKey]time.Time, k worklog.TicketKey, at time.Time) {
		if was, ok := m[k]; !ok || at.Before(was) {
			m[k] = at
		}
	}
	for _, c := range history {
		earliest(first, c.Ticket(), c.WorkedAt())
		for _, above := range worklog.Ancestors(c.Ticket(), parents) {
			earliest(first, above, c.WorkedAt())
		}
		k := roundKey{c.Ticket(), c.Round()}
		if was, ok := firstInRound[k]; !ok || c.WorkedAt().Before(was) {
			firstInRound[k] = c.WorkedAt()
		}
	}
	for i := range rows {
		rows[i].StartedAt = first[rows[i].Ticket]
		for j := range rows[i].Rounds {
			rows[i].Rounds[j].StartedAt = firstInRound[roundKey{rows[i].Ticket, rows[i].Rounds[j].Round}]
		}
	}
}

// setAddedAt sets when each ticket went into the user's timeline: when it
// was added without time, or else when its first time was logged.
func setAddedAt(rows []report.TicketRow, history []*worklog.CheckIn, planned map[worklog.TicketKey]worklog.PlannedTicket) {
	first := make(map[worklog.TicketKey]time.Time, len(rows))
	for _, c := range history {
		if at, ok := first[c.Ticket()]; !ok || c.WorkedAt().Before(at) {
			first[c.Ticket()] = c.WorkedAt()
		}
	}
	for key, p := range planned {
		if at, ok := first[key]; !ok || p.AddedAt.Before(at) {
			first[key] = p.AddedAt
		}
	}
	for i := range rows {
		rows[i].AddedAt = first[rows[i].Ticket]
	}
}

// AddTicketCommand is a ticket to add to the user's timeline before any time
// is logged on it.
type AddTicketCommand struct {
	Project string
	Repo    string
	Ticket  string
	Title   string
	Parent  string // a backend or web ticket's spec ticket number; "" for none
	Type    string // a SPinno ticket's type; "" for its repo's default
}

// AddTicket puts a ticket in the user's timeline with no time yet, so it shows
// in the table (not started) without running a timer. A ticket the user
// already has, with time or without, is refused.
func (s *CheckInService) AddTicket(ctx context.Context, u *user.User, cmd AddTicketCommand) (worklog.PlannedTicket, error) {
	key, err := worklog.ParseTicketKey(cmd.Project, cmd.Repo, cmd.Ticket)
	if err != nil {
		return worklog.PlannedTicket{}, err
	}
	if err := ownProject(ctx, s.checkIns, s.planned, u, key); err != nil {
		return worklog.PlannedTicket{}, err
	}
	if in, err := inTimeline(ctx, s.checkIns, s.planned, u.ID(), key); err != nil {
		return worklog.PlannedTicket{}, err
	} else if in {
		return worklog.PlannedTicket{}, worklog.ErrTicketExists
	}
	planned, err := worklog.NewPlannedTicket(u.ID(), key, cmd.Title, worklog.StatusNotStarted, "", time.Now())
	if err != nil {
		return worklog.PlannedTicket{}, err
	}
	var ticketType worklog.TicketType
	if len(worklog.TypesFor(key.Repo)) > 0 {
		if ticketType, err = worklog.ParseType(key.Repo, cmd.Type); err != nil {
			return worklog.PlannedTicket{}, err
		}
	}
	// Checked before saving anything, so a bad parent adds nothing
	var parent *worklog.TicketKey
	if strings.TrimSpace(cmd.Parent) != "" {
		p, err := s.existingSpec(ctx, u, key, cmd.Parent)
		if err != nil {
			return worklog.PlannedTicket{}, err
		}
		// A new ticket is a child; its spec can't be one
		if err := s.twoLevels(ctx, u, nil, p); err != nil {
			return worklog.PlannedTicket{}, err
		}
		parent = &p
	}
	if err := s.planned.Save(ctx, planned); err != nil {
		return worklog.PlannedTicket{}, err
	}
	if parent != nil {
		if err := s.parents.Set(ctx, u.ID(), key, parent); err != nil {
			return worklog.PlannedTicket{}, err
		}
	}
	// A parent ticket has no type: a spec added with none and under no spec.
	// A child spec is added with its type, even before its parent spec is set.
	if ticketType != "" && !(key.Repo == project.RepoSpec && parent == nil && strings.TrimSpace(cmd.Type) == "") {
		if err := s.types.Set(ctx, u.ID(), key, ticketType); err != nil {
			return worklog.PlannedTicket{}, err
		}
	}
	return planned, nil
}

// SetType changes a SPinno ticket's type. Its steps change with it, so it
// starts again from where its time puts it in the new type's steps.
func (s *CheckInService) SetType(ctx context.Context, u *user.User, projectName, repo, ticket, ticketType string) error {
	key, err := worklog.ParseTicketKey(projectName, repo, ticket)
	if err != nil {
		return err
	}
	t, err := worklog.ParseType(key.Repo, ticketType)
	if err != nil {
		return err
	}
	parents, types, err := hierarchy(ctx, s.parents, s.types, u.ID())
	if err != nil {
		return err
	}
	if worklog.IsParentLevel(key, parents, types) {
		return fmt.Errorf("%w: a parent ticket has no type; put it under a spec ticket to give it one", worklog.ErrInvalid)
	}
	if worklog.TypeOf(types, parents, key) == t {
		_, chosen := types[key]
		if chosen {
			return nil
		}
		return s.types.Set(ctx, u.ID(), key, t)
	}
	if err := s.types.Set(ctx, u.ID(), key, t); err != nil {
		return err
	}
	return s.progress.Delete(ctx, u.ID(), key)
}

// typeOf is a ticket's type ("" outside SPinno).
func (s *CheckInService) typeOf(ctx context.Context, userID user.ID, key worklog.TicketKey) (worklog.TicketType, error) {
	return typeOfTicket(ctx, s.parents, s.types, userID, key)
}

// SetParent puts a SPinno backend or web ticket under a spec ticket (its
// number), or under none when parent is "". It can be moved at any time,
// even before time is logged on it.
func (s *CheckInService) SetParent(ctx context.Context, u *user.User, projectName, repo, ticket, parent string) error {
	key, err := worklog.ParseTicketKey(projectName, repo, ticket)
	if err != nil {
		return err
	}
	// The parent it leaves, which may be left with no tickets
	before, err := s.parents.FindByUser(ctx, u.ID())
	if err != nil {
		return err
	}
	old, hadParent := before[key]
	if strings.TrimSpace(parent) == "" {
		// A child spec taken out of its parent stays a child: its type is
		// saved, so it isn't taken for a parent (one with no type)
		if hadParent && key.Repo == project.RepoSpec {
			types, err := s.types.FindByUser(ctx, u.ID())
			if err != nil {
				return err
			}
			if _, typed := types[key]; !typed {
				if err := s.types.Set(ctx, u.ID(), key, worklog.TypeOf(types, before, key)); err != nil {
					return err
				}
			}
		}
		if err := s.parents.Set(ctx, u.ID(), key, nil); err != nil {
			return err
		}
		if hadParent {
			return s.leftParent(ctx, u, old)
		}
		return nil
	}
	p, err := s.existingSpec(ctx, u, key, parent)
	if err != nil {
		return err
	}
	if err := s.notUnderItself(ctx, u, key, p); err != nil {
		return err
	}
	if err := s.twoLevels(ctx, u, nil, p); err != nil {
		return err
	}
	// A parent made by dragging becomes a child again; one added as a
	// parent can't
	if err := s.becomeChild(ctx, u, key); err != nil {
		return err
	}
	if err := s.parents.Set(ctx, u.ID(), key, &p); err != nil {
		return err
	}
	if hadParent && old != p {
		return s.leftParent(ctx, u, old)
	}
	return nil
}

// leftParent is called once a ticket has left a parent: a parent made by
// dragging that has no tickets under it any more is a child again (its
// type and steps back). One added as a parent stays a parent.
func (s *CheckInService) leftParent(ctx context.Context, u *user.User, parent worklog.TicketKey) error {
	made, err := s.types.FindMadeParents(ctx, u.ID())
	if err != nil || !made[parent] {
		return err
	}
	parents, err := s.parents.FindByUser(ctx, u.ID())
	if err != nil {
		return err
	}
	for _, above := range parents {
		if above == parent {
			return nil
		}
	}
	return s.types.SetMadeParent(ctx, u.ID(), parent, false)
}

// MakeParent makes a child spec ticket a parent, as dragging it out of any
// parent does: it leaves its parent, and its type (kept for if it's dragged
// back) and steps are put aside, so tickets can go under it.
func (s *CheckInService) MakeParent(ctx context.Context, u *user.User, projectName, repo, ticket string) error {
	key, err := worklog.ParseTicketKey(projectName, repo, ticket)
	if err != nil {
		return err
	}
	if key.Repo != project.RepoSpec {
		return fmt.Errorf("%w: only a spec ticket can be a parent", worklog.ErrInvalid)
	}
	parents, types, err := hierarchy(ctx, s.parents, s.types, u.ID())
	if err != nil {
		return err
	}
	if worklog.IsParentLevel(key, parents, types) {
		return fmt.Errorf("%w: #%s is a parent ticket already", worklog.ErrInvalid, key.Number)
	}
	// Saved, so it comes back if it's made a child again
	if err := s.types.Set(ctx, u.ID(), key, worklog.TypeOf(types, parents, key)); err != nil {
		return err
	}
	if err := s.parents.Set(ctx, u.ID(), key, nil); err != nil {
		return err
	}
	if err := s.types.SetMadeParent(ctx, u.ID(), key, true); err != nil {
		return err
	}
	if old, ok := parents[key]; ok {
		return s.leftParent(ctx, u, old)
	}
	return nil
}

// MakeChild makes a parent made by dragging a child again, under no parent
// (dragged out of the parents), its type and steps back. Not one added as a
// parent, nor one with tickets under it.
func (s *CheckInService) MakeChild(ctx context.Context, u *user.User, projectName, repo, ticket string) error {
	key, err := worklog.ParseTicketKey(projectName, repo, ticket)
	if err != nil {
		return err
	}
	return s.becomeChild(ctx, u, key)
}

// becomeChild readies a ticket to go under a parent: a child already is
// one; a parent made by dragging becomes a child again, with its type back,
// as long as nothing is under it; a parent added as one can't.
func (s *CheckInService) becomeChild(ctx context.Context, u *user.User, key worklog.TicketKey) error {
	parents, types, err := hierarchy(ctx, s.parents, s.types, u.ID())
	if err != nil {
		return err
	}
	if !worklog.IsParentLevel(key, parents, types) {
		return nil
	}
	made, err := s.types.FindMadeParents(ctx, u.ID())
	if err != nil {
		return err
	}
	if !made[key] {
		return fmt.Errorf("%w: #%s was added as a parent ticket, so it can't become a child", worklog.ErrInvalid, key.Number)
	}
	for _, above := range parents {
		if above == key {
			return fmt.Errorf("%w: tickets are under #%s: move them out first to make it a child", worklog.ErrInvalid, key.Number)
		}
	}
	return s.types.SetMadeParent(ctx, u.ID(), key, false)
}

// ownProject refuses a new ticket outside the user's project, the team they
// chose at signup: a BlackBox member's tickets are BlackBox tickets, a
// SPinno member's SPinno ones. A ticket they already have in the other
// project (from before) can still be worked on.
func ownProject(ctx context.Context, checkIns worklog.Repository, planned worklog.PlannedTicketRepository, u *user.User, key worklog.TicketKey) error {
	if key.Project == u.Team() {
		return nil
	}
	if in, err := inTimeline(ctx, checkIns, planned, u.ID(), key); err != nil || in {
		return err
	}
	return fmt.Errorf("%w: your project is %s, so your tickets are %s tickets", worklog.ErrInvalid, u.Team(), u.Team())
}

// noParentTime refuses time on a parent ticket: it takes none of its own,
// its time is the tickets' under it.
func noParentTime(ctx context.Context, parents worklog.ParentRepository, types worklog.TypeRepository, userID user.ID, key worklog.TicketKey) error {
	if key.Repo != project.RepoSpec {
		return nil
	}
	under, chosen, err := hierarchy(ctx, parents, types, userID)
	if err != nil {
		return err
	}
	if worklog.IsParentLevel(key, under, chosen) {
		return fmt.Errorf("%w: #%s is a parent ticket, so it takes no time of its own: log it on the tickets under it", worklog.ErrInvalid, key.Number)
	}
	return nil
}

// noParentHours refuses a ticket's hours (any above zero) when it's a parent
func (s *CheckInService) noParentHours(ctx context.Context, u *user.User, cmd TicketWorkCommand) error {
	hasHours := false
	for _, h := range cmd.PhaseHours {
		hasHours = hasHours || h > 0
	}
	if !hasHours {
		return nil
	}
	key, err := worklog.ParseTicketKey(cmd.Project, cmd.Repo, cmd.Ticket)
	if err != nil {
		return err
	}
	return noParentTime(ctx, s.parents, s.types, u.ID(), key)
}

// twoLevels keeps tickets two levels deep, parent then child: tickets go
// only under a parent ticket, and a parent ticket (child, when given) can't
// go under another parent.
func (s *CheckInService) twoLevels(ctx context.Context, u *user.User, child *worklog.TicketKey, parent worklog.TicketKey) error {
	parents, types, err := hierarchy(ctx, s.parents, s.types, u.ID())
	if err != nil {
		return err
	}
	if _, under := parents[parent]; under || !worklog.IsParentLevel(parent, parents, types) {
		return fmt.Errorf("%w: #%s is a child ticket, not a parent, so tickets can't go under it", worklog.ErrInvalid, parent.Number)
	}
	if child != nil && worklog.IsParentLevel(*child, parents, types) {
		return fmt.Errorf("%w: #%s is a parent ticket, so it can't go under another parent", worklog.ErrInvalid, child.Number)
	}
	return nil
}

// DeleteRound removes one round of the user's ticket (the original work or a
// review-back): its time, and the round itself when the ticket's steps are in
// it, as a review-back just started (no time yet) is nothing but that. The
// ticket keeps some time (deleting the ticket removes all of it), unless it
// was added without time.
func (s *CheckInService) DeleteRound(ctx context.Context, u *user.User, projectName, repo, ticket string, round int) error {
	key, err := worklog.ParseTicketKey(projectName, repo, ticket)
	if err != nil {
		return err
	}
	if round < 1 {
		return fmt.Errorf("%w: round must be 1 or more", worklog.ErrInvalid)
	}
	existing, err := s.checkIns.FindByTicket(ctx, u.ID(), key)
	if err != nil {
		return err
	}
	var remove []worklog.ID
	others := 0
	for _, c := range existing {
		if c.Round() == round {
			remove = append(remove, c.ID())
		} else {
			others++
		}
	}
	progress, err := s.progress.FindByUser(ctx, u.ID())
	if err != nil {
		return err
	}
	step, hasStep := progress[key]
	stepInRound := hasStep && step.Round == round
	if len(remove) == 0 && !stepInRound {
		return worklog.ErrNotFound
	}
	if len(remove) > 0 && others == 0 {
		if _, err := s.planned.Find(ctx, u.ID(), key); errors.Is(err, worklog.ErrNotFound) {
			return fmt.Errorf("%w: this is the ticket's only round with time: delete the ticket instead", worklog.ErrInvalid)
		} else if err != nil {
			return err
		}
	}
	if len(remove) > 0 {
		if err := s.checkIns.SaveChanges(ctx, nil, remove); err != nil {
			return err
		}
	}
	// The steps are worked out again from the time left, unless they're in a
	// later round (a review-back started after this one), which stays
	if stepInRound || (len(remove) > 0 && !(hasStep && step.Round > round)) {
		return s.progress.Delete(ctx, u.ID(), key)
	}
	return nil
}

// existingSpec is the spec ticket (by number) a backend or web ticket is to
// go under, which must already be in the user's timeline: time logged on it,
// or added without time. A ticket can't sit under a spec that isn't there.
func (s *CheckInService) existingSpec(ctx context.Context, u *user.User, child worklog.TicketKey, number string) (worklog.TicketKey, error) {
	spec, err := worklog.SpecParent(child, number)
	if err != nil {
		return worklog.TicketKey{}, err
	}
	if in, err := inTimeline(ctx, s.checkIns, s.planned, u.ID(), spec); err != nil {
		return worklog.TicketKey{}, err
	} else if in {
		return spec, nil
	}
	return worklog.TicketKey{}, fmt.Errorf("%w: parent #%s isn't in your timeline yet: add that parent ticket first", worklog.ErrInvalid, spec.Number)
}

// notUnderItself refuses to put a spec ticket under one of the specs below
// it, which would make a loop.
func (s *CheckInService) notUnderItself(ctx context.Context, u *user.User, child, parent worklog.TicketKey) error {
	parents, err := s.parents.FindByUser(ctx, u.ID())
	if err != nil {
		return err
	}
	for _, above := range worklog.Ancestors(parent, parents) {
		if above == child {
			return fmt.Errorf("%w: Spec #%s is under Spec #%s already, so #%s can't go under it", worklog.ErrInvalid, parent.Number, child.Number, child.Number)
		}
	}
	return nil
}

func (s *CheckInService) Create(ctx context.Context, u *user.User, cmd CheckInCommand) (*worklog.CheckIn, error) {
	now := time.Now()
	entry, err := cmd.entry(now)
	if err != nil {
		return nil, err
	}
	if err := ownProject(ctx, s.checkIns, s.planned, u, entry.Ticket); err != nil {
		return nil, err
	}
	if err := noParentTime(ctx, s.parents, s.types, u.ID(), entry.Ticket); err != nil {
		return nil, err
	}
	c, err := newCheckIn(ctx, s.checkIns, u.ID(), worklog.SourceApp, entry, now)
	if err != nil {
		return nil, err
	}
	if err := s.checkIns.Save(ctx, c); err != nil {
		return nil, err
	}
	// Now not done (e.g. back to ongoing): the ticket hasn't ended
	if err := s.clearEndUnlessDone(ctx, u.ID(), entry.Ticket); err != nil {
		return nil, err
	}
	return c, nil
}

// CreateMany saves one check-in per phase that has hours, all or none.
func (s *CheckInService) CreateMany(ctx context.Context, u *user.User, cmd TicketWorkCommand) ([]*worklog.CheckIn, error) {
	key, err := worklog.ParseTicketKey(cmd.Project, cmd.Repo, cmd.Ticket)
	if err != nil {
		return nil, err
	}
	if err := ownProject(ctx, s.checkIns, s.planned, u, key); err != nil {
		return nil, err
	}
	dates, hasDates, err := cmd.ticketDates(u.ID())
	if err != nil {
		return nil, err
	}
	if err := s.noParentHours(ctx, u, cmd); err != nil {
		return nil, err
	}
	hours := make(map[worklog.Phase]float64, len(cmd.PhaseHours))
	for key, h := range cmd.PhaseHours {
		p, err := worklog.ParsePhase(key)
		if err != nil {
			return nil, err
		}
		hours[p] = h
	}

	now := time.Now()
	var checkIns []*worklog.CheckIn
	for _, p := range worklog.Phases() {
		h, ok := hours[p]
		if !ok || h == 0 {
			continue
		}
		entry, err := cmd.checkIn(p, h).entry(now)
		if err != nil {
			return nil, err
		}
		c, err := newCheckIn(ctx, s.checkIns, u.ID(), worklog.SourceApp, entry, now)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.Label(), err)
		}
		checkIns = append(checkIns, c)
	}
	if len(checkIns) == 0 {
		return nil, fmt.Errorf("%w: enter the time spent on at least one phase", worklog.ErrInvalid)
	}
	if err := s.checkIns.SaveChanges(ctx, checkIns, nil); err != nil {
		return nil, err
	}
	// Time typed in by hand moves the ticket on; the step follows it
	if err := s.progress.Delete(ctx, u.ID(), checkIns[0].Ticket()); err != nil {
		return nil, err
	}
	if hasDates {
		if err := s.dates.Save(ctx, dates); err != nil {
			return nil, err
		}
	}
	if err := s.clearEndUnlessDone(ctx, u.ID(), checkIns[0].Ticket()); err != nil {
		return nil, err
	}
	return checkIns, nil
}

// SetTotals corrects the user's time on one round of a ticket (cmd.Round) so
// each phase adds up to the hours given, like editing a row of the progress
// sheet; phases left out go to zero, and other rounds are left alone. Added time is logged at cmd.WorkedAt, and removed time comes
// off the newest entries first, so older history stays put. The title
// applies to all the user's entries on the ticket; the status and note to
// the latest one, which is what the row shows. All or nothing.
func (s *CheckInService) SetTotals(ctx context.Context, u *user.User, cmd TicketWorkCommand) ([]*worklog.CheckIn, error) {
	key, err := worklog.ParseTicketKey(cmd.Project, cmd.Repo, cmd.Ticket)
	if err != nil {
		return nil, err
	}
	if err := ownProject(ctx, s.checkIns, s.planned, u, key); err != nil {
		return nil, err
	}
	// A parent's details can be saved, but not time
	if err := s.noParentHours(ctx, u, cmd); err != nil {
		return nil, err
	}
	status, err := worklog.ParseStatus(cmd.Status)
	if err != nil {
		return nil, err
	}
	dates, hasDates, err := cmd.ticketDates(u.ID())
	if err != nil {
		return nil, err
	}

	targets := make(map[worklog.Phase]int, len(cmd.PhaseHours))
	for key, h := range cmd.PhaseHours {
		phase, err := worklog.ParsePhase(key)
		if err != nil {
			return nil, err
		}
		if math.IsNaN(h) || h < 0 {
			return nil, fmt.Errorf("%w: %s: time can't be negative", worklog.ErrInvalid, phase.Label())
		}
		targets[phase] = minutesOf(h)
	}
	if cmd.Round == 0 {
		cmd.Round = 1
	}

	existing, err := s.checkIns.FindByTicket(ctx, u.ID(), key)
	if err != nil {
		return nil, err
	}
	byPhase := map[worklog.Phase][]*worklog.CheckIn{}
	for _, c := range existing {
		if c.Round() == cmd.Round {
			byPhase[c.Phase()] = append(byPhase[c.Phase()], c)
		}
	}

	now := time.Now()
	workedAt := cmd.WorkedAt.Truncate(time.Minute)
	var added []*worklog.CheckIn
	changed := map[worklog.ID]*worklog.CheckIn{}
	removed := map[worklog.ID]bool{}
	// Whether any time moved, as opposed to only the details
	timeChanged := false
	wasDone := len(existing) > 0 && latestOf(existing).Status() == worklog.StatusDone

	for _, phase := range worklog.Phases() {
		entries := byPhase[phase]
		current := 0
		for _, c := range entries {
			current += minutesOf(c.Hours())
		}
		delta := targets[phase] - current
		if delta != 0 {
			timeChanged = true
		}

		switch {
		case delta > 0:
			// Top up an entry already at that time, so the unique key holds
			var slot *worklog.CheckIn
			for _, c := range entries {
				if c.WorkedAt().Equal(workedAt) {
					slot = c
				}
			}
			if slot != nil {
				if err := setMinutes(slot, minutesOf(slot.Hours())+delta); err != nil {
					return nil, fmt.Errorf("%s: %w", phase.Label(), err)
				}
				changed[slot.ID()] = slot
				continue
			}
			entry, err := cmd.checkIn(phase, float64(delta)/60).entry(now)
			if err != nil {
				return nil, err
			}
			c, err := newCheckIn(ctx, s.checkIns, u.ID(), worklog.SourceApp, entry, now)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", phase.Label(), err)
			}
			added = append(added, c)

		case delta < 0:
			remaining := -delta
			for i := len(entries) - 1; i >= 0 && remaining > 0; i-- {
				c := entries[i]
				m := minutesOf(c.Hours())
				if m <= remaining {
					removed[c.ID()] = true
					remaining -= m
					continue
				}
				if err := setMinutes(c, m-remaining); err != nil {
					return nil, fmt.Errorf("%s: %w", phase.Label(), err)
				}
				changed[c.ID()] = c
				remaining = 0
			}
		}
	}

	var result []*worklog.CheckIn
	for _, c := range existing {
		if !removed[c.ID()] {
			result = append(result, c)
		}
	}
	result = append(result, added...)
	// Clearing a round is fine (e.g. a review-back logged by mistake), but
	// the ticket must keep some time, unless it was added without time: then
	// its details are saved on it, and any time cleared goes
	if len(result) == 0 {
		planned, err := s.planned.Find(ctx, u.ID(), key)
		if errors.Is(err, worklog.ErrNotFound) {
			return nil, fmt.Errorf("%w: enter the time spent on at least one phase", worklog.ErrInvalid)
		}
		if err != nil {
			return nil, err
		}
		if planned, err = worklog.NewPlannedTicket(u.ID(), key, cmd.Title, status, cmd.Note, planned.AddedAt); err != nil {
			return nil, err
		}
		remove := make([]worklog.ID, 0, len(removed))
		for id := range removed {
			remove = append(remove, id)
		}
		if len(remove) > 0 {
			if err := s.checkIns.SaveChanges(ctx, nil, remove); err != nil {
				return nil, err
			}
			if err := s.progress.Delete(ctx, u.ID(), key); err != nil {
				return nil, err
			}
		}
		if err := s.planned.Save(ctx, planned); err != nil {
			return nil, err
		}
		if hasDates {
			if err := s.dates.Save(ctx, dates); err != nil {
				return nil, err
			}
		}
		if err := s.clearEndUnlessDone(ctx, u.ID(), key); err != nil {
			return nil, err
		}
		return []*worklog.CheckIn{}, nil
	}

	title := strings.TrimSpace(cmd.Title)
	latest := result[0]
	for _, c := range result {
		e := c.Entry()
		if c == latest || latest.Before(c) {
			latest = c
		}
		if e.Title != title {
			e.Title = title
			if err := c.Revise(e); err != nil {
				return nil, err
			}
			changed[c.ID()] = c
		}
	}
	e := latest.Entry()
	e.Status = status
	e.Note = cmd.Note
	if err := latest.Revise(e); err != nil {
		return nil, err
	}
	changed[latest.ID()] = latest

	save := added
	isNew := make(map[worklog.ID]bool, len(added))
	for _, c := range added {
		isNew[c.ID()] = true
	}
	for id, c := range changed {
		if !isNew[id] {
			save = append(save, c)
		}
	}
	remove := make([]worklog.ID, 0, len(removed))
	for id := range removed {
		remove = append(remove, id)
	}
	if err := s.checkIns.SaveChanges(ctx, save, remove); err != nil {
		return nil, err
	}
	// New time, or being marked done (or no longer done), decides the step
	// again; a change to only the title, note or another status keeps it, so
	// a review-back just started (no time yet) stays
	if timeChanged || wasDone != (status == worklog.StatusDone) {
		if err := s.progress.Delete(ctx, u.ID(), key); err != nil {
			return nil, err
		}
	}
	if hasDates {
		if err := s.dates.Save(ctx, dates); err != nil {
			return nil, err
		}
	}
	// Set back from Done (e.g. to Ongoing): the end date goes
	if err := s.clearEndUnlessDone(ctx, u.ID(), key); err != nil {
		return nil, err
	}
	return result, nil
}

// setMinutes changes how long a check-in says was spent.
func setMinutes(c *worklog.CheckIn, minutes int) error {
	e := c.Entry()
	e.Hours = float64(minutes) / 60
	return c.Revise(e)
}

func minutesOf(hours float64) int { return int(math.Round(hours * 60)) }

func (s *CheckInService) Update(ctx context.Context, u *user.User, id string, cmd CheckInCommand) (*worklog.CheckIn, error) {
	c, err := s.owned(ctx, u, id)
	if err != nil {
		return nil, err
	}
	entry, err := cmd.entry(time.Now())
	if err != nil {
		return nil, err
	}
	if err := ownProject(ctx, s.checkIns, s.planned, u, entry.Ticket); err != nil {
		return nil, err
	}
	if err := noParentTime(ctx, s.parents, s.types, u.ID(), entry.Ticket); err != nil {
		return nil, err
	}
	if err := c.Revise(entry); err != nil {
		return nil, err
	}
	if err := s.checkIns.Save(ctx, c); err != nil {
		return nil, err
	}
	// Now not done (e.g. back to ongoing): the ticket hasn't ended
	if err := s.clearEndUnlessDone(ctx, u.ID(), entry.Ticket); err != nil {
		return nil, err
	}
	return c, nil
}

// DeleteTicket removes the user's whole timeline row for a ticket: their
// time on it in every round, and its dates. Other people's time stays.
func (s *CheckInService) DeleteTicket(ctx context.Context, u *user.User, projectName, repo, ticket string) error {
	key, err := worklog.ParseTicketKey(projectName, repo, ticket)
	if err != nil {
		return err
	}
	before, err := s.parents.FindByUser(ctx, u.ID())
	if err != nil {
		return err
	}
	n, err := s.checkIns.DeleteTicket(ctx, u.ID(), key)
	if err != nil {
		return err
	}
	if n == 0 {
		return worklog.ErrNotFound
	}
	// Its parent may be left with no tickets
	if old, ok := before[key]; ok {
		return s.leftParent(ctx, u, old)
	}
	return nil
}

func (s *CheckInService) Delete(ctx context.Context, u *user.User, id string) error {
	if _, err := s.owned(ctx, u, id); err != nil {
		return err
	}
	return s.checkIns.Delete(ctx, worklog.ID(id))
}

// owned loads a check-in, treating other people's as not found so their
// ids can't be probed.
func (s *CheckInService) owned(ctx context.Context, u *user.User, id string) (*worklog.CheckIn, error) {
	c, err := s.checkIns.FindByID(ctx, worklog.ID(id))
	if err != nil {
		return nil, err
	}
	if !c.OwnedBy(u.ID()) {
		return nil, worklog.ErrNotFound
	}
	return c, nil
}
