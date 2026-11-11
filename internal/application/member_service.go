package application

import (
	"context"
	"fmt"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/report"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// Submission says how a member's week got into the system.
type Submission string

const (
	SubmissionApp      Submission = "app"
	SubmissionImported Submission = "imported"
	SubmissionMissing  Submission = "missing"
)

type MemberWeek struct {
	User       *user.User
	Activity   report.PersonRow
	LastImport *worklog.Import
	Submission Submission
}

// ImportRow is one spreadsheet row. Err is set when the row couldn't even be
// read, e.g. a date cell that isn't a date.
type ImportRow struct {
	Line    int
	Command CheckInCommand
	Err     error
}

type ImportRowResult struct {
	Line      int
	CheckIn   *worklog.CheckIn // nil when Error is set
	Error     string
	Duplicate bool // already in the system, will be skipped
}

type SheetImport struct {
	Period calendar.Period
	Rows   []ImportRowResult
	// Added is only set once the import has been saved.
	Added int
}

func (r *SheetImport) count() (valid, skipped int) {
	for _, row := range r.Rows {
		if row.Error == "" && !row.Duplicate {
			valid++
		} else {
			skipped++
		}
	}
	return valid, skipped
}

// MemberService holds the admin use cases for managing people and loading
// their spreadsheets.
type MemberService struct {
	users    user.Repository
	checkIns worklog.Repository
	imports  worklog.ImportRepository
	dates    worklog.TicketDatesRepository
	progress worklog.ProgressRepository
	planned  worklog.PlannedTicketRepository
	parents  worklog.ParentRepository
	types    worklog.TypeRepository
}

func NewMemberService(
	users user.Repository,
	checkIns worklog.Repository,
	imports worklog.ImportRepository,
	dates worklog.TicketDatesRepository,
	progress worklog.ProgressRepository,
	planned worklog.PlannedTicketRepository,
	parents worklog.ParentRepository,
	types worklog.TypeRepository,
) *MemberService {
	return &MemberService{
		users: users, checkIns: checkIns, imports: imports, dates: dates, progress: progress,
		planned: planned, parents: parents, types: types,
	}
}

// Week lists every user with what they logged in the week that day falls in.
func (s *MemberService) Week(ctx context.Context, day string) (calendar.Period, []MemberWeek, error) {
	d, err := calendar.ParseDay(day)
	if err != nil {
		return calendar.Period{}, nil, err
	}
	period := calendar.WeekOf(d)
	users, err := s.users.FindAll(ctx)
	if err != nil {
		return period, nil, err
	}
	checkIns, err := s.checkIns.FindUntil(ctx, period.Last())
	if err != nil {
		return period, nil, err
	}
	imports, err := s.imports.LatestForWeek(ctx, period.First())
	if err != nil {
		return period, nil, err
	}

	data := report.Build(period, checkIns, users, calendar.DayOf(time.Now()))
	activity := make(map[user.ID]report.PersonRow, len(data.People))
	for _, p := range data.People {
		activity[p.UserID] = p
	}

	members := make([]MemberWeek, len(users))
	for i, u := range users {
		m := MemberWeek{User: u, Activity: activity[u.ID()], Submission: SubmissionMissing}
		if imp, ok := imports[u.ID()]; ok {
			m.LastImport = &imp
			m.Submission = SubmissionImported
		} else if m.Activity.CheckIns > 0 {
			m.Submission = SubmissionApp
		}
		members[i] = m
	}
	return period, members, nil
}

// MemberTimeline is one member's whole timeline: every ticket in it, as the
// member sees it, with all of its time.
type MemberTimeline struct {
	User    *user.User
	Tickets []report.TicketRow
}

// TeamTimelines is every user's whole timeline up to now, for the team sheet
// export, ordered by team then name. Everyone is in it, admins too, even with
// nothing in their timeline: each gets a tab.
func (s *MemberService) TeamTimelines(ctx context.Context, now time.Time) ([]MemberTimeline, error) {
	today := calendar.DayOf(now)
	all, err := s.checkIns.FindUntil(ctx, today)
	if err != nil {
		return nil, err
	}
	first := today
	for _, c := range all {
		if c.Day().Before(first) {
			first = c.Day()
		}
	}
	period, err := calendar.NewPeriod(first, today)
	if err != nil {
		return nil, err
	}
	return s.TeamTimelinesFor(ctx, period)
}

// TeamTimelinesFor is every user's timeline for the period, as each sees it
// for that week or day: the tickets in it then, with their totals as of the
// period's end. For a report's download in the team sheet's layout.
func (s *MemberService) TeamTimelinesFor(ctx context.Context, period calendar.Period) ([]MemberTimeline, error) {
	users, err := s.users.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	var team []MemberTimeline
	for _, u := range users {
		t, err := timelineFor(ctx, s.checkIns, s.dates, s.progress, s.planned, s.parents, s.types, u, period)
		if err != nil {
			return nil, err
		}
		team = append(team, MemberTimeline{User: u, Tickets: t.Tickets})
	}
	return team, nil
}

// Dashboard is the member's own dashboard for the week that day falls in.
func (s *MemberService) Dashboard(ctx context.Context, userID, day string) (*MyWeek, error) {
	u, err := s.users.FindByID(ctx, user.ID(userID))
	if err != nil {
		return nil, err
	}
	return weekFor(ctx, s.checkIns, s.dates, s.progress, s.planned, s.parents, s.types, u, day)
}

// PreviewImport checks a member's spreadsheet rows without saving anything.
func (s *MemberService) PreviewImport(ctx context.Context, userID, day string, rows []ImportRow) (*SheetImport, error) {
	u, err := s.users.FindByID(ctx, user.ID(userID))
	if err != nil {
		return nil, err
	}
	d, err := calendar.ParseDay(day)
	if err != nil {
		return nil, err
	}
	period := calendar.WeekOf(d)
	existing, err := s.checkIns.FindByUser(ctx, u.ID(), period)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(existing))
	for _, c := range existing {
		seen[dedupeKey(c)] = true
	}

	now := time.Now()
	result := &SheetImport{Period: period}
	for _, row := range rows {
		res := ImportRowResult{Line: row.Line}
		c, err := s.checkInFromRow(ctx, u, row, now)
		switch {
		case err != nil:
			res.Error = err.Error()
		case !period.Contains(c.Day()):
			res.Error = fmt.Sprintf("%s is outside the week of %s", c.Day(), period.First())
		default:
			res.CheckIn = c
			key := dedupeKey(c)
			res.Duplicate = seen[key]
			seen[key] = true
		}
		result.Rows = append(result.Rows, res)
	}
	return result, nil
}

// Import saves the valid, new rows of a member's spreadsheet and records the
// import against the week.
func (s *MemberService) Import(ctx context.Context, admin *user.User, userID, day, filename string, rows []ImportRow) (*SheetImport, error) {
	result, err := s.PreviewImport(ctx, userID, day, rows)
	if err != nil {
		return nil, err
	}
	var fresh []*worklog.CheckIn
	for _, row := range result.Rows {
		if row.CheckIn != nil && !row.Duplicate {
			fresh = append(fresh, row.CheckIn)
		}
	}
	if len(fresh) == 0 {
		return nil, fmt.Errorf("%w: the file has no new rows for this week", worklog.ErrInvalid)
	}
	added, err := s.checkIns.SaveNew(ctx, fresh)
	if err != nil {
		return nil, err
	}
	result.Added = added
	_, skipped := result.count()

	err = s.imports.Save(ctx, worklog.Import{
		UserID:     user.ID(userID),
		Week:       result.Period.First(),
		Filename:   filename,
		Added:      added,
		Skipped:    skipped,
		ImportedBy: admin.ID(),
		ImportedAt: time.Now(),
	})
	return result, err
}

func (s *MemberService) checkInFromRow(ctx context.Context, u *user.User, row ImportRow, now time.Time) (*worklog.CheckIn, error) {
	if row.Err != nil {
		return nil, row.Err
	}
	entry, err := row.Command.entry(now)
	if err != nil {
		return nil, err
	}
	return newCheckIn(ctx, s.checkIns, u.ID(), worklog.SourceImport, entry, now)
}

// dedupeKey matches the UNIQUE key on the checkins table.
func dedupeKey(c *worklog.CheckIn) string {
	return fmt.Sprintf("%s|%d|%s|%s", c.Ticket(), c.Round(), c.Phase(), c.WorkedAt().UTC().Format(time.RFC3339))
}
