package report

import (
	"testing"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

func mustUser(t *testing.T, id, name string, role user.Role) *user.User {
	t.Helper()
	u, err := user.New(user.ID(id), name, id+"@example.com", "hash", project.BlackBox, role, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func at(day string, hour int) time.Time {
	d, _ := calendar.ParseDay(day)
	return d.Start().Add(time.Duration(hour) * time.Hour)
}

func checkIn(t *testing.T, n int, userID string, ticket string, phase worklog.Phase, status worklog.Status, hours float64, when time.Time) *worklog.CheckIn {
	t.Helper()
	c, err := worklog.New(worklog.ID(string(rune('a'+n))), user.ID(userID), worklog.SourceApp, worklog.Entry{
		Ticket:   worklog.TicketKey{Project: project.BlackBox, Number: ticket},
		Title:    "Ticket " + ticket,
		Phase:    phase,
		Status:   status,
		Hours:    hours,
		Note:     "note",
		WorkedAt: when,
	}, when)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// inRound moves a check-in to a later round of work on its ticket.
func inRound(t *testing.T, c *worklog.CheckIn, round int) *worklog.CheckIn {
	t.Helper()
	e := c.Entry()
	e.Round = round
	if err := c.Revise(e); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestBuildWeekly(t *testing.T) {
	ana := mustUser(t, "ana", "Ana", user.RoleMember)
	ben := mustUser(t, "ben", "Ben", user.RoleMember)
	boss := mustUser(t, "boss", "Boss", user.RoleAdmin)

	// Week of Mon 2026-10-26.
	checkIns := []*worklog.CheckIn{
		// Before the week: counts towards totals, not this week's hours.
		checkIn(t, 0, "ana", "3221", worklog.PhaseInvestigation, worklog.StatusOngoing, 3, at("2026-10-23", 10)),
		checkIn(t, 1, "ana", "3221", worklog.PhaseCoding, worklog.StatusOngoing, 4, at("2026-10-26", 10)),
		checkIn(t, 2, "ben", "3221", worklog.PhaseReview, worklog.StatusWaiting, 1, at("2026-10-27", 10)),
		// Review sent it back: the rework is the next round.
		inRound(t, checkIn(t, 3, "ana", "3221", worklog.PhaseCoding, worklog.StatusOngoing, 2.25, at("2026-10-27", 14)), 2),
		inRound(t, checkIn(t, 4, "ana", "3221", worklog.PhaseTesting, worklog.StatusDone, 1.5, at("2026-10-28", 9)), 2),
		// After the week: ignored entirely.
		checkIn(t, 5, "ana", "3221", worklog.PhaseReview, worklog.StatusDone, 8, at("2026-11-02", 9)),
		// A ticket only touched before the week is left out.
		checkIn(t, 6, "ben", "9999", worklog.PhaseCoding, worklog.StatusOngoing, 2, at("2026-10-20", 9)),
	}

	week := calendar.WeekOf(mustDay(t, "2026-10-28"))
	data := Build(week, checkIns, []*user.User{ana, ben, boss}, mustDay(t, "2026-10-28"))

	if len(data.Tickets) != 1 {
		t.Fatalf("got %d tickets, want 1", len(data.Tickets))
	}
	row := data.Tickets[0]
	if row.Ticket.String() != "BlackBox#3221" {
		t.Errorf("ticket = %s", row.Ticket)
	}
	if row.PeriodHours != 8.75 || row.TotalHours != 11.75 {
		t.Errorf("hours = %v this week, %v total; want 8.75, 11.75", row.PeriodHours, row.TotalHours)
	}
	if row.PhaseHours[worklog.PhaseCoding] != 6.25 || row.PhaseHours[worklog.PhaseInvestigation] != 3 {
		t.Errorf("phase hours = %v", row.PhaseHours)
	}
	if row.ReviewBacks != 1 {
		t.Errorf("review-backs = %d, want 1", row.ReviewBacks)
	}
	if len(row.Rounds) != 2 {
		t.Fatalf("got %d rounds, want 2: %+v", len(row.Rounds), row.Rounds)
	}
	if r := row.Rounds[0]; r.Round != 1 || r.Hours != 8 || r.PhaseHours[worklog.PhaseCoding] != 4 {
		t.Errorf("original round = %+v, want 8 h with 4 h coding", r)
	}
	if r := row.Rounds[1]; r.Round != 2 || r.Hours != 3.75 || r.PhaseHours[worklog.PhaseCoding] != 2.25 ||
		r.FirstDay.String() != "2026-10-27" || r.LastDay.String() != "2026-10-28" {
		t.Errorf("review-back round = %+v, want 3.75 h from 2026-10-27 to 2026-10-28", r)
	}
	if row.StartDate.String() != "2026-10-23" || row.EndDate == nil || row.EndDate.String() != "2026-10-28" {
		t.Errorf("dates = %s to %v, want 2026-10-23 to 2026-10-28", row.StartDate, row.EndDate)
	}
	if row.Phase != worklog.PhaseTesting || row.Status != worklog.StatusDone {
		t.Errorf("latest = %s/%s, want testing/done", row.Phase, row.Status)
	}
	if len(row.Assignees) != 2 || row.Assignees[0] != "Ana" || row.Assignees[1] != "Ben" {
		t.Errorf("assignees = %v", row.Assignees)
	}

	if len(data.Projects) != 1 || data.Projects[0].Hours != 8.75 || data.Projects[0].ByStatus[worklog.StatusDone] != 1 {
		t.Errorf("projects = %+v", data.Projects)
	}

	// The admin logged nothing, so is left off; Ana and Ben are listed.
	if len(data.People) != 2 {
		t.Fatalf("got %d people, want 2: %+v", len(data.People), data.People)
	}
	for _, p := range data.People {
		// Mon–Wed are past weekdays, so 24 expected hours.
		if p.ExpectedHours != 24 {
			t.Errorf("%s expected hours = %v, want 24", p.Name, p.ExpectedHours)
		}
		switch p.Name {
		case "Ana":
			if p.Hours != 7.75 || p.CheckIns != 3 || len(p.MissedDays) != 0 {
				t.Errorf("Ana = %+v", p)
			}
		case "Ben":
			if len(p.MissedDays) != 2 {
				t.Errorf("Ben missed %v, want Mon and Wed", p.MissedDays)
			}
		}
	}
}

func mustDay(t *testing.T, s string) calendar.Day {
	t.Helper()
	d, err := calendar.ParseDay(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestBuildPhasesLoggedTogether(t *testing.T) {
	ana := mustUser(t, "ana", "Ana", user.RoleMember)
	when := at("2026-11-03", 10)

	// Saved in one go, listed review first: still workflow order, so review
	// is the latest phase. Both are the original round.
	checkIns := []*worklog.CheckIn{
		checkIn(t, 0, "ana", "3287", worklog.PhaseReview, worklog.StatusDone, 0.5, when),
		checkIn(t, 1, "ana", "3287", worklog.PhaseCoding, worklog.StatusDone, 2, when),
	}
	data := Build(calendar.WeekOf(mustDay(t, "2026-11-03")), checkIns, []*user.User{ana}, mustDay(t, "2026-11-03"))

	row := data.Tickets[0]
	if row.ReviewBacks != 0 {
		t.Errorf("review-backs = %d, want 0", row.ReviewBacks)
	}
	if row.Phase != worklog.PhaseReview || row.TotalHours != 2.5 {
		t.Errorf("latest = %s with %v h, want review with 2.5 h", row.Phase, row.TotalHours)
	}
}

func TestBuildLaterRoundIsLatest(t *testing.T) {
	ana := mustUser(t, "ana", "Ana", user.RoleMember)
	when := at("2026-11-03", 10)

	// Same time: the review-back's coding comes after the original review,
	// so the ticket is back in coding and ongoing.
	checkIns := []*worklog.CheckIn{
		checkIn(t, 0, "ana", "7001", worklog.PhaseReview, worklog.StatusWaiting, 0.5, when),
		inRound(t, checkIn(t, 1, "ana", "7001", worklog.PhaseCoding, worklog.StatusOngoing, 1, when), 2),
	}
	data := Build(calendar.WeekOf(mustDay(t, "2026-11-03")), checkIns, []*user.User{ana}, mustDay(t, "2026-11-03"))

	row := data.Tickets[0]
	if row.Phase != worklog.PhaseCoding || row.Status != worklog.StatusOngoing || row.ReviewBacks != 1 {
		t.Errorf("latest = %s/%s with %d review-backs, want coding/ongoing with 1", row.Phase, row.Status, row.ReviewBacks)
	}
}
