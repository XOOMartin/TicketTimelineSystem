package excel

import (
	"strings"

	"crud_app/internal/domain/report"
	"crud_app/internal/domain/worklog"
)

// writeTicketTable writes a member's own export of their tickets.
func writeTicketTable(s *sheet, title, periodHours string, rows []report.TicketRow, withAssignees bool) error {
	if err := s.writeTitle(title); err != nil {
		return err
	}
	names := []string{"Project", "Ticket", "Title / short description"}
	widths := []float64{11, 16, 40}
	if withAssignees {
		names = append(names, "Assignees")
		widths = append(widths, 22)
	}
	names = append(names, "Current phase", "Status", periodHours, "Total hours")
	widths = append(widths, 24, 11, 10, 10)
	for _, ph := range worklog.Phases() {
		names = append(names, ph.Label()+" (h)")
		widths = append(widths, 13)
	}
	names = append(names, "Review-backs", "Last update", "Notes / blocker")
	widths = append(widths, 10, 17, 50)

	headerRow := s.row
	if err := s.writeHeader(names, widths); err != nil {
		return err
	}
	for _, t := range rows {
		values := []any{t.Ticket.Project.String(), t.Ticket.String(), t.Title}
		if withAssignees {
			values = append(values, strings.Join(t.Assignees, ", "))
		}
		values = append(values, t.Phase.Label(), t.Status.Label(), t.PeriodHours, t.TotalHours)
		for _, ph := range worklog.Phases() {
			values = append(values, blankIfZero(t.PhaseHours[ph]))
		}
		values = append(values, t.ReviewBacks, t.LastUpdate.Local().Format("2006-01-02 15:04"), t.Note)
		if err := s.writeRow(values); err != nil {
			return err
		}
	}

	first := 4 // period hours column, after Project, Ticket, Title
	if withAssignees {
		first++
	}
	cols := []int{first + 2, first + 3}
	for i := range worklog.Phases() {
		cols = append(cols, first+4+i)
	}
	if err := s.formatHours(headerRow+1, cols...); err != nil {
		return err
	}
	return s.addFilter(headerRow)
}

// blankIfZero leaves phase cells empty for phases nobody worked on, so the
// timeline stands out.
func blankIfZero(h float64) any {
	if h == 0 {
		return nil
	}
	return h
}
