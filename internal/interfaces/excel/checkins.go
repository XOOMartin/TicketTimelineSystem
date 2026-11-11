package excel

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"crud_app/internal/application"
	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// CheckInSheet is the sheet an admin imports. Its columns are matched by
// header name, so people may reorder them or add their own.
const CheckInSheet = "Check-ins"

const (
	colDate    = "Date"
	colTime    = "Time"
	colProject = "Project"
	colTicket  = "Ticket"
	colTitle   = "Title"
	colPhase   = "Phase"
	colStatus  = "Status"
	colHours   = "Hours"
	colNote    = "Note"
	// Optional: blank or missing means the original work
	colRound = "Round"
)

var requiredColumns = []string{colDate, colTime, colProject, colTicket, colTitle, colPhase, colStatus, colHours, colNote}

// Round goes last so files made before it existed still line up.
var checkInColumns = []string{colDate, colTime, colProject, colTicket, colTitle, colPhase, colStatus, colHours, colNote, colRound}

// Rows with a dropdown in the template, enough for a busy week.
const dropdownRows = 500

// WriteWeek writes a member's week: the importable Check-ins sheet and a
// Tickets summary. With no check-ins it doubles as a blank template.
func WriteWeek(w io.Writer, u *user.User, week *application.MyWeek) error {
	f := excelize.NewFile()

	s, err := addSheet(f, CheckInSheet)
	if err != nil {
		return err
	}
	if err := s.writeHeader(checkInColumns, []float64{12, 8, 11, 10, 40, 26, 12, 8, 50, 8}); err != nil {
		return err
	}
	// The dashboard lists newest first; a timesheet reads oldest first.
	checkIns := append([]*worklog.CheckIn(nil), week.CheckIns...)
	sort.SliceStable(checkIns, func(i, j int) bool { return checkIns[i].WorkedAt().Before(checkIns[j].WorkedAt()) })
	for _, c := range checkIns {
		at := c.WorkedAt().Local()
		err := s.writeRow([]any{
			at.Format(calendar.DayLayout), at.Format("15:04"),
			c.Ticket().Project.String(), c.Ticket().Number, c.Title(),
			c.Phase().Label(), c.Status().Label(), c.Hours(), c.Note(), c.Round(),
		})
		if err != nil {
			return err
		}
	}
	if err := s.formatHours(2, 8); err != nil {
		return err
	}
	if err := addDropdowns(f); err != nil {
		return err
	}

	t, err := addSheet(f, "Tickets")
	if err != nil {
		return err
	}
	title := fmt.Sprintf("%s — week of %s to %s", u.Name(), week.Period.First(), week.Period.Last())
	if err := writeTicketTable(t, title, "Hours this week", week.Tickets, false); err != nil {
		return err
	}
	return save(f, w)
}

func addDropdowns(f *excelize.File) error {
	lists := map[string][]string{"C": {}, "F": {}, "G": {}}
	for _, p := range project.All() {
		lists["C"] = append(lists["C"], p.String())
	}
	for _, p := range worklog.Phases() {
		lists["F"] = append(lists["F"], p.Label())
	}
	for _, st := range worklog.Statuses() {
		lists["G"] = append(lists["G"], st.Label())
	}
	for col, values := range lists {
		dv := excelize.NewDataValidation(true)
		dv.Sqref = fmt.Sprintf("%s2:%s%d", col, col, dropdownRows)
		if err := dv.SetDropList(values); err != nil {
			return err
		}
		if err := f.AddDataValidation(CheckInSheet, dv); err != nil {
			return err
		}
	}
	return nil
}

// ReadCheckIns reads the Check-ins sheet of an uploaded workbook. A row that
// can't be read is returned with Err set, so the admin sees every problem at
// once instead of the first.
func ReadCheckIns(r io.Reader) ([]application.ImportRow, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, errors.New("the file is not a valid .xlsx workbook")
	}
	defer f.Close()

	name := CheckInSheet
	if idx, _ := f.GetSheetIndex(name); idx < 0 {
		name = f.GetSheetName(0)
	}
	// Raw values, so dates come back as Excel serial numbers whatever the
	// cell's display format is.
	rows, err := f.GetRows(name, excelize.Options{RawCellValue: true})
	if err != nil {
		return nil, err
	}

	headerAt, cols := findHeader(rows)
	if headerAt < 0 {
		return nil, fmt.Errorf("no header row found; the sheet needs the columns %s", strings.Join(requiredColumns, ", "))
	}

	var result []application.ImportRow
	for i := headerAt + 1; i < len(rows); i++ {
		get := func(col string) string {
			idx, ok := cols[col]
			if !ok || idx >= len(rows[i]) {
				return ""
			}
			return strings.TrimSpace(rows[i][idx])
		}
		if isBlank(rows[i]) {
			continue
		}
		row := application.ImportRow{Line: i + 1}
		row.Command, row.Err = readRow(get)
		result = append(result, row)
	}
	return result, nil
}

// findHeader returns the index of the row holding the column names, and the
// position of each known column in it.
func findHeader(rows [][]string) (int, map[string]int) {
	for i, row := range rows {
		cols := map[string]int{}
		for j, cell := range row {
			for _, name := range checkInColumns {
				if strings.EqualFold(strings.TrimSpace(cell), name) {
					cols[name] = j
				}
			}
		}
		if _, ok := cols[colDate]; ok {
			if _, ok := cols[colTicket]; ok {
				return i, cols
			}
		}
	}
	return -1, nil
}

func readRow(get func(string) string) (application.CheckInCommand, error) {
	cmd := application.CheckInCommand{
		Project: get(colProject),
		Ticket:  get(colTicket),
		Title:   get(colTitle),
		Phase:   get(colPhase),
		Status:  get(colStatus),
		Note:    get(colNote),
	}
	hours, err := strconv.ParseFloat(get(colHours), 64)
	if err != nil {
		return cmd, fmt.Errorf("hours %q is not a number", get(colHours))
	}
	cmd.Hours = hours
	if r := get(colRound); r != "" {
		if cmd.Round, err = strconv.Atoi(r); err != nil {
			return cmd, fmt.Errorf("round %q should be a whole number: 1 for the original work, 2 after the first review-back", r)
		}
	}
	cmd.WorkedAt, err = parseDateTime(get(colDate), get(colTime))
	return cmd, err
}

var dateLayouts = []string{calendar.DayLayout, "2006/01/02", "2006/1/2", "1/2/2006", "01/02/2006", "1/2/06"}
var timeLayouts = []string{"15:04", "15:04:05", "3:04 PM", "3:04PM", "3:04 pm"}

// parseDateTime reads a local date and time typed as text or stored as Excel
// serial numbers. A missing time means 09:00.
func parseDateTime(date, clock string) (time.Time, error) {
	var day time.Time
	if serial, err := strconv.ParseFloat(date, 64); err == nil {
		t, err := excelize.ExcelDateToTime(serial, false)
		if err != nil {
			return time.Time{}, fmt.Errorf("date %q is not a date", date)
		}
		day = t
		// A date cell that also holds a time, e.g. 2026-10-30 14:00.
		if clock == "" && serial != float64(int(serial)) {
			return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), nil
		}
	} else if day, err = parseLayouts(date, dateLayouts); err != nil {
		return time.Time{}, fmt.Errorf("date %q should look like 2026-10-30", date)
	}

	hour, minute := 9, 0
	if clock != "" {
		if frac, err := strconv.ParseFloat(clock, 64); err == nil && frac >= 0 && frac < 1 {
			minutes := int(frac*24*60 + 0.5)
			hour, minute = minutes/60, minutes%60
		} else if t, err := parseLayouts(clock, timeLayouts); err == nil {
			hour, minute = t.Hour(), t.Minute()
		} else {
			return time.Time{}, fmt.Errorf("time %q should look like 14:00", clock)
		}
	}
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, time.Local), nil
}

func parseLayouts(s string, layouts []string) (time.Time, error) {
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("no layout matched")
}

func isBlank(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
