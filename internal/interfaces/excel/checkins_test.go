package excel

import (
	"bytes"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"crud_app/internal/application"
	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// A member's export must import back unchanged, review-back round included.
func TestWriteWeekRoundTrip(t *testing.T) {
	u, err := user.New("u1", "Ana", "ana@example.com", "hash", project.SPinno, user.RoleMember, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 10, 28, 14, 30, 0, 0, time.Local)
	c, err := worklog.New("c1", u.ID(), worklog.SourceApp, worklog.Entry{
		Ticket:   worklog.TicketKey{Project: project.SPinno, Number: "1231"},
		Title:    "Export CSV",
		Round:    2,
		Phase:    worklog.PhaseConfirmation,
		Status:   worklog.StatusBlocked,
		Hours:    1.75,
		Note:     "Waiting for spec",
		WorkedAt: when,
	}, when)
	if err != nil {
		t.Fatal(err)
	}
	week := &application.MyWeek{Period: calendar.WeekOf(c.Day()), CheckIns: []*worklog.CheckIn{c}}

	var buf bytes.Buffer
	if err := WriteWeek(&buf, u, week); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadCheckIns(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	got := rows[0]
	if got.Err != nil {
		t.Fatal(got.Err)
	}
	want := application.CheckInCommand{
		Project: "SPinno", Ticket: "1231", Title: "Export CSV", Round: 2,
		Phase: worklog.PhaseConfirmation.Label(), Status: worklog.StatusBlocked.Label(),
		Hours: 1.75, Note: "Waiting for spec", WorkedAt: when,
	}
	if got.Command != want {
		t.Errorf("got %+v\nwant %+v", got.Command, want)
	}
	if got.Line != 2 {
		t.Errorf("line = %d, want 2", got.Line)
	}
}

// People editing in Excel turn text into real date and time cells, reorder
// columns and leave bad values; all of that should be read or reported.
func TestReadCheckInsHandEdited(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	set := func(cell string, v any) {
		if err := f.SetCellValue(sheet, cell, v); err != nil {
			t.Fatal(err)
		}
	}
	for i, h := range []string{"Ticket", "Date", "Time", "Project", "Title", "Phase", "Status", "Hours", "Note"} {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		set(cell, h)
	}
	// Row 2: a real Excel date cell and a time typed as 2:15 PM.
	set("A2", "BlackBox#3221")
	set("B2", time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC))
	set("C2", "2:15 PM")
	set("D2", "blackbox")
	set("E2", "Fix profile save")
	set("F2", "coding")
	set("G2", "Ongoing")
	set("H2", 2)
	// Row 4 (row 3 left blank): hours isn't a number.
	set("A4", "3222")
	set("B4", "2026-10-27")
	set("H4", "two")

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	rows, err := ReadCheckIns(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (blank row skipped)", len(rows))
	}
	if rows[0].Err != nil {
		t.Fatal(rows[0].Err)
	}
	wantAt := time.Date(2026, 10, 27, 14, 15, 0, 0, time.Local)
	if !rows[0].Command.WorkedAt.Equal(wantAt) || rows[0].Command.Ticket != "BlackBox#3221" {
		t.Errorf("row 2 = %+v", rows[0].Command)
	}
	if rows[1].Err == nil || rows[1].Line != 4 {
		t.Errorf("row 4 should fail on hours, got line %d err %v", rows[1].Line, rows[1].Err)
	}
}
