package report

import (
	"math"
	"sort"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// HoursPerDay is the working time expected from each person on a weekday.
const HoursPerDay = 8

// Build computes a report for the period. checkIns must be every check-in up
// to the end of the period, oldest first, so ticket totals and review-backs
// count the whole history. people are the users to list on the People sheet;
// anyone else with check-ins in the period is listed too.
func Build(period calendar.Period, checkIns []*worklog.CheckIn, people []*user.User, today calendar.Day) Data {
	// Phases logged together at one time are taken in workflow order, so the
	// latest phase and status are the furthest along.
	checkIns = append([]*worklog.CheckIn(nil), checkIns...)
	sort.SliceStable(checkIns, func(i, j int) bool { return checkIns[i].Before(checkIns[j]) })

	names := make(map[user.ID]*user.User, len(people))
	for _, u := range people {
		names[u.ID()] = u
	}

	tickets := map[worklog.TicketKey]*ticketState{}
	var order []worklog.TicketKey
	perPerson := map[user.ID]*PersonRow{}
	worked := map[user.ID]map[calendar.Day]bool{}

	for _, c := range checkIns {
		if period.Last().Before(c.Day()) {
			continue
		}
		t, ok := tickets[c.Ticket()]
		if !ok {
			t = &ticketState{
				row:       TicketRow{Ticket: c.Ticket(), PhaseHours: map[worklog.Phase]float64{}},
				assignees: map[user.ID]bool{},
				rounds:    map[int]*RoundRow{},
			}
			tickets[c.Ticket()] = t
			order = append(order, c.Ticket())
		}
		t.add(c, period.Contains(c.Day()))

		if !period.Contains(c.Day()) {
			continue
		}
		p, ok := perPerson[c.UserID()]
		if !ok {
			p = &PersonRow{UserID: c.UserID()}
			perPerson[c.UserID()] = p
			worked[c.UserID()] = map[calendar.Day]bool{}
		}
		p.CheckIns++
		p.Hours += c.Hours()
		worked[c.UserID()][c.Day()] = true
	}

	var data Data
	byProject := map[project.Project]*ProjectRow{}
	for _, key := range order {
		t := tickets[key]
		if !t.inPeriod {
			continue
		}
		row := t.row
		for id := range t.assignees {
			row.Assignees = append(row.Assignees, nameOf(names, id))
		}
		sort.Strings(row.Assignees)
		for _, r := range t.rounds {
			row.Rounds = append(row.Rounds, *r)
		}
		sort.Slice(row.Rounds, func(i, j int) bool { return row.Rounds[i].Round < row.Rounds[j].Round })
		row.PeriodHours = round(row.PeriodHours)
		row.TotalHours = round(row.TotalHours)
		data.Tickets = append(data.Tickets, row)

		pr, ok := byProject[key.Project]
		if !ok {
			pr = &ProjectRow{Project: key.Project, ByStatus: map[worklog.Status]int{}, PhaseHours: map[worklog.Phase]float64{}}
			byProject[key.Project] = pr
		}
		pr.Tickets++
		pr.ByStatus[row.Status]++
		for phase, h := range t.periodPhaseHours {
			pr.PhaseHours[phase] = round(pr.PhaseHours[phase] + h)
		}
		pr.Hours = round(pr.Hours + row.PeriodHours)
	}
	sort.SliceStable(data.Tickets, func(i, j int) bool {
		a, b := data.Tickets[i].Ticket, data.Tickets[j].Ticket
		if a.Project != b.Project {
			return a.Project < b.Project
		}
		return a.Number < b.Number
	})
	for _, p := range project.All() {
		if pr, ok := byProject[p]; ok {
			data.Projects = append(data.Projects, *pr)
		}
	}

	// Weekdays up to today are the days someone was expected to check in.
	var workdays []calendar.Day
	for _, d := range period.Days() {
		if !d.IsWeekend() && !today.Before(d) {
			workdays = append(workdays, d)
		}
	}
	for _, u := range people {
		if _, ok := perPerson[u.ID()]; !ok && !u.IsAdmin() {
			perPerson[u.ID()] = &PersonRow{UserID: u.ID()}
		}
	}
	for id, p := range perPerson {
		p.Name = nameOf(names, id)
		if u, ok := names[id]; ok {
			p.Team = u.Team()
		}
		p.Hours = round(p.Hours)
		p.ExpectedHours = float64(len(workdays) * HoursPerDay)
		for _, d := range workdays {
			if !worked[id][d] {
				p.MissedDays = append(p.MissedDays, d)
			}
		}
		data.People = append(data.People, *p)
	}
	sort.Slice(data.People, func(i, j int) bool {
		a, b := data.People[i], data.People[j]
		if a.Team != b.Team {
			return a.Team < b.Team
		}
		return a.Name < b.Name
	})
	return data
}

// ticketState accumulates one ticket's check-ins in time order.
type ticketState struct {
	row              TicketRow
	assignees        map[user.ID]bool
	periodPhaseHours map[worklog.Phase]float64
	inPeriod         bool
	rounds           map[int]*RoundRow
}

func (t *ticketState) add(c *worklog.CheckIn, inPeriod bool) {
	r, ok := t.rounds[c.Round()]
	if !ok {
		r = &RoundRow{Round: c.Round(), PhaseHours: map[worklog.Phase]float64{}, FirstDay: c.Day()}
		t.rounds[c.Round()] = r
	}
	r.PhaseHours[c.Phase()] = round(r.PhaseHours[c.Phase()] + c.Hours())
	r.Hours = round(r.Hours + c.Hours())
	r.LastDay = c.Day()
	if c.Round()-1 > t.row.ReviewBacks {
		t.row.ReviewBacks = c.Round() - 1
	}

	// Check-ins arrive oldest first, so the first one starts the ticket and
	// the latest one wins.
	if t.row.StartDate.String() == "" {
		t.row.StartDate = c.Day()
	}
	if c.Status() == worklog.StatusDone {
		end := c.Day()
		t.row.EndDate = &end
	} else {
		t.row.EndDate = nil
	}
	t.row.Title = c.Title()
	t.row.Phase = c.Phase()
	t.row.Status = c.Status()
	t.row.Note = c.Note()
	t.row.LastUpdate = c.WorkedAt()
	t.row.TotalHours += c.Hours()
	t.row.PhaseHours[c.Phase()] = round(t.row.PhaseHours[c.Phase()] + c.Hours())

	if inPeriod {
		t.inPeriod = true
		t.assignees[c.UserID()] = true
		t.row.PeriodHours += c.Hours()
		if t.periodPhaseHours == nil {
			t.periodPhaseHours = map[worklog.Phase]float64{}
		}
		t.periodPhaseHours[c.Phase()] += c.Hours()
	}
}

func nameOf(names map[user.ID]*user.User, id user.ID) string {
	if u, ok := names[id]; ok {
		return u.Name()
	}
	return "(deleted user)"
}

// round keeps sums of quarter hours free of float noise like 7.999999.
func round(h float64) float64 { return math.Round(h*100) / 100 }
