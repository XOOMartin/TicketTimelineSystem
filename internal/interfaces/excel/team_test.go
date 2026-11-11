package excel

import (
	"bytes"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"crud_app/internal/application"
	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/report"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

func teamFixture(t *testing.T) []application.MemberTimeline {
	t.Helper()
	member := func(id, name string, team project.Project) *user.User {
		u, err := user.New(user.ID(id), name, id+"@example.com", "hash", team, user.RoleMember, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	day := func(s string) calendar.Day {
		d, err := calendar.ParseDay(s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	done := day("2026-11-04")
	key := func(p project.Project, repo project.Repo, n string) worklog.TicketKey {
		return worklog.TicketKey{Project: p, Repo: repo, Number: n}
	}
	return []application.MemberTimeline{
		{User: member("bb1", "Martin", project.BlackBox), Tickets: []report.TicketRow{
			{
				Ticket: key(project.BlackBox, "", "3287"), Title: "Status-only interceptor", Status: worklog.StatusDone,
				StartDate: day("2026-11-03"), EndDate: &done, ReviewBacks: 2, Note: "Merged",
				PhaseHours: map[worklog.Phase]float64{worklog.PhaseInvestigation: 0.5, worklog.PhaseCoding: 2},
			},
			{
				Ticket: key(project.BlackBox, "", "353"), Title: "Go upgrade", Status: worklog.StatusOngoing,
				StartDate: day("2026-11-04"), ReviewBacks: 1,
				PhaseHours: map[worklog.Phase]float64{worklog.PhaseInvestigation: 4.25},
			},
		}},
		// Same name as the first: gets "Martin (2)"
		{User: member("bb2", "Martin", project.BlackBox)},
		{User: member("sp1", "Markrey", project.SPinno), Tickets: []report.TicketRow{
			{Ticket: key(project.SPinno, project.RepoSpec, "181"), Title: "CSV menus", Status: worklog.StatusOngoing,
				PhaseHours: map[worklog.Phase]float64{}},
			{Ticket: key(project.SPinno, project.RepoWeb, "3759"), Title: "CSV menus - implementation", Status: worklog.StatusDone,
				Type: "implementation", Parent: "181", ReviewBacks: 3,
				PhaseHours: map[worklog.Phase]float64{worklog.PhaseInvestigation: 2, worklog.PhaseCoding: 4}},
			{Ticket: key(project.SPinno, project.RepoSpec, "230"), Title: "CSV menus - integration test", Status: worklog.StatusWaiting,
				Type: "investigation", Parent: "181", ReviewBacks: 1,
				PhaseHours: map[worklog.Phase]float64{worklog.PhaseTesting: 8.5}},
			{Ticket: key(project.SPinno, project.RepoSpec, "325"), Title: "Login module", Status: worklog.StatusNotStarted,
				Type: "investigation", PhaseHours: map[worklog.Phase]float64{}},
		}},
	}
}

func TestWriteTeamSheet(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTeamSheet(&buf, teamFixture(t), ""); err != nil {
		t.Fatal(err)
	}
	// TEAM_SHEET_OUT=path keeps a copy to open and look at
	if out := os.Getenv("TEAM_SHEET_OUT"); out != "" {
		if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Summary", "Blackbox Ticket Allocation", "SPinno Ticket Timeline", "SPinno Ticket Allocation", "Guide", "Martin", "Martin (2)", "Markrey"}
	if got := f.GetSheetList(); len(got) != len(want) {
		t.Fatalf("sheets = %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("sheets = %v, want %v", got, want)
			}
		}
	}

	cell := func(sheet, ref string) string {
		t.Helper()
		v, err := f.GetCellValue(sheet, ref)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	formula := func(sheet, ref string) string {
		t.Helper()
		v, err := f.GetCellFormula(sheet, ref)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	// Summary: BlackBox's members, then SPinno's, each with a total
	checks := map[string]string{
		"A4": "BlackBox Project", "A5": "Member", "A6": "Martin", "A7": "Martin", "A8": "BLACKBOX TOTAL",
		"A11": "SPinno Project", "A13": "Markrey", "A14": "SPinno TOTAL",
		"A17": "All Projects", "A19": "BlackBox", "A20": "SPinno", "A21": "GRAND TOTAL",
	}
	for ref, v := range checks {
		if got := cell("Summary", ref); got != v {
			t.Errorf("Summary!%s = %q, want %q", ref, got, v)
		}
	}
	formulas := map[string]string{
		"B6": `COUNTIF('Martin'!A4:A1000,"BlackBox")`, "C7": `COUNTIFS('Martin (2)'!A4:A1000,"BlackBox",'Martin (2)'!D4:D1000,"Done")`,
		"D6": `SUMIF('Martin'!A4:A1000,"BlackBox",'Martin'!G4:G1000)`, "B13": `COUNTIF('Markrey'!A4:A1000,"SPinno")`, "J6": "SUM(D6:I6)", "K6": "IF(B6=0,0,J6/B6)",
		"B8": "SUM(B6:B7)", "B19": "B8", "B20": "B14", "J21": "SUM(J19:J20)",
	}
	for ref, v := range formulas {
		if got := formula("Summary", ref); got != v {
			t.Errorf("Summary!%s formula = %q, want %q", ref, got, v)
		}
	}

	// Summary names are plain text: a link to a tab would show in Google
	// Sheets as "#gid=…" in place of the name
	for _, ref := range []string{"A6", "A7", "A13"} {
		if ok, target, _ := f.GetCellHyperLink("Summary", ref); ok {
			t.Errorf("Summary!%s links to %q, want no link", ref, target)
		}
	}

	// A member tab: the sheet's columns, statuses and durations
	if got := cell("Martin", "B4"); got != "3287" {
		t.Errorf("Martin!B4 = %q", got)
	}
	if got := cell("Martin", "D5"); got != "Review Back" {
		t.Errorf("Martin!D5 = %q, want Review Back", got)
	}
	if got := cell("Martin", "G5"); got != "4 h 15 m" {
		t.Errorf("Martin!G5 = %q, want 4 h 15 m", got)
	}
	if got := cell("Martin", "E4"); got != "2026-11-03" {
		t.Errorf("Martin!E4 = %q, want 2026-11-03", got)
	}
	if got := formula("Martin", "N4"); got != `IF(COUNT(G4:L4)=0,"",SUM(G4:L4))` {
		t.Errorf("Martin!N4 formula = %q", got)
	}

	// SPinno timeline: the ticket under no parent first, then the parent
	// with its children summed
	rows := map[string]string{"A5": "Spec#325", "B5": "Child", "A6": "Spec#181", "B6": "Parent", "C6": "Markrey", "A7": "Spec#230", "A8": "Web#3759", "A9": "SPinno TOTAL"}
	for ref, v := range rows {
		if got := cell("SPinno Ticket Timeline", ref); got != v {
			t.Errorf("SPinno!%s = %q, want %q", ref, got, v)
		}
	}
	if got := formula("SPinno Ticket Timeline", "F6"); got != "SUM(F7:F8)" {
		t.Errorf("SPinno!F6 formula = %q, want SUM(F7:F8)", got)
	}
	// The total sits right under the last ticket
	if got := formula("SPinno Ticket Timeline", "F9"); got != `SUMIF($B$5:$B$8,"Child",F5:F8)` {
		t.Errorf("SPinno!F9 formula = %q", got)
	}

	// The allocations: each BlackBox ticket with who has it; the SPinno
	// families with their tasks
	alloc := map[string]map[string]string{
		"Blackbox Ticket Allocation": {"A1": "ALLOCATION OF TASK", "A2": "BlackBox", "A4": "#3287 Status-only interceptor", "F4": "Done", "G4": "Martin", "A5": "#353 Go upgrade"},
		"SPinno Ticket Allocation":   {"A3": "SPinno", "A5": "Spec#325", "A6": "Spec#181", "B6": "Parent", "C6": "CSV menus", "G7": "For Review", "J8": "Markrey"},
	}
	for sheet, cells := range alloc {
		for ref, v := range cells {
			if got := cell(sheet, ref); got != v {
				t.Errorf("%s!%s = %q, want %q", sheet, ref, got, v)
			}
		}
	}
	// SPinno tickets link to their GitHub issues
	if ok, target, _ := f.GetCellHyperLink("SPinno Ticket Timeline", "A8"); !ok || target != "https://github.com/SPinnoDeveloper/spinno-next-web/issues/3759" {
		t.Errorf("SPinno!A8 links to %q", target)
	}

	// The formulas work out to the right numbers (durations are days)
	calc := func(sheet, ref string) string {
		t.Helper()
		v, err := f.CalcCellValue(sheet, ref, excelize.Options{RawCellValue: true})
		if err != nil {
			t.Fatalf("%s!%s: %v", sheet, ref, err)
		}
		return v
	}
	values := []struct{ sheet, ref, want string }{
		{"Summary", "B6", "2"},                                 // Martin's tickets
		{"Summary", "C6", "1"},                                 // ...one done
		{"Summary", "L6", "3"},                                 // review-backs
		{"Summary", "J6", "0.28125"},                           // 6 h 45 m
		{"Summary", "B8", "2"},                                 // BlackBox total, the second Martin has none
		{"Summary", "B14", "4"},                                // Markrey's tickets
		{"Summary", "J21", "0.885416666666667"},                // 6 h 45 m + 14 h 30 m
		{"SPinno Ticket Timeline", "F6", "0.0833333333333333"}, // the parent: 2 h from #3759
		{"SPinno Ticket Timeline", "L6", "0.604166666666667"},  // 6 h + 8 h 30 m
		{"SPinno Ticket Timeline", "M6", "4"},                  // 3 + 1 review-backs
		{"SPinno Ticket Timeline", "L9", "0.604166666666667"},  // total: the children only
		{"SPinno Ticket Timeline", "M9", "4"},                  // review-backs, the children only
	}
	for _, v := range values {
		if got := calc(v.sheet, v.ref); !sameNumber(got, v.want) {
			t.Errorf("%s!%s = %s, want %s", v.sheet, v.ref, got, v.want)
		}
	}
}

// sameNumber compares two numbers written out, to 9 places
func sameNumber(a, b string) bool {
	x, err1 := strconv.ParseFloat(a, 64)
	y, err2 := strconv.ParseFloat(b, 64)
	if err1 != nil || err2 != nil {
		return a == b
	}
	return math.Abs(x-y) < 1e-9
}

// A ticket counts in its own project's table: a BlackBox member's SPinno
// ticket goes in the SPinno table, with the member listed there too.
func TestTeamSheetSplitsProjects(t *testing.T) {
	u, err := user.New("bb1", "Martin", "bb1@example.com", "hash", project.BlackBox, user.RoleMember, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	members := []application.MemberTimeline{{User: u, Tickets: []report.TicketRow{
		{Ticket: worklog.TicketKey{Project: project.BlackBox, Number: "353"}, Title: "Go upgrade", Status: worklog.StatusOngoing,
			PhaseHours: map[worklog.Phase]float64{worklog.PhaseInvestigation: 2}},
		{Ticket: worklog.TicketKey{Project: project.SPinno, Repo: project.RepoWeb, Number: "4234"}, Title: "child", Status: worklog.StatusDone,
			Type: "implementation", ReviewBacks: 3, PhaseHours: map[worklog.Phase]float64{worklog.PhaseCoding: 3}},
	}}}
	var buf bytes.Buffer
	if err := WriteTeamSheet(&buf, members, ""); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	// BlackBox: Martin row 6, total 7. SPinno: heading 10, Martin row 12.
	for _, c := range []struct{ ref, want string }{
		{"A6", "Martin"}, {"A12", "Martin"}, {"A13", "SPinno TOTAL"},
		{"B6", "1"}, {"C6", "0"}, {"D6", "0.0833333333333333"}, {"G6", "0"}, {"L6", "0"},
		{"B12", "1"}, {"C12", "1"}, {"D12", "0"}, {"G12", "0.125"}, {"L12", "3"},
	} {
		got, err := f.CalcCellValue("Summary", c.ref, excelize.Options{RawCellValue: true})
		if err != nil {
			t.Fatalf("Summary!%s: %v", c.ref, err)
		}
		if !sameNumber(got, c.want) {
			t.Errorf("Summary!%s = %s, want %s", c.ref, got, c.want)
		}
	}
	if got, _ := f.GetCellValue("Martin", "B5"); got != "4234" {
		t.Errorf("Martin!B5 = %q, want 4234", got)
	}
}

// The export shows each user by their first name only: the tab, the Summary
// row, the tab's title and as assignee. An account without names keeps its
// name; the username is never shown.
func TestTeamSheetNames(t *testing.T) {
	u, err := user.New("sp1", "Markrey Manabat", "markrey", "hash", project.SPinno, user.RoleMember, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	u.SetNames("Markrey", "Manabat")
	old, err := user.New("sp2", "Martin", "martin@example.com", "hash", project.SPinno, user.RoleMember, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ticket := func(n string) []report.TicketRow {
		return []report.TicketRow{{Ticket: worklog.TicketKey{Project: project.SPinno, Repo: project.RepoWeb, Number: n},
			Title: "Task " + n, Status: worklog.StatusOngoing, Type: "implementation", PhaseHours: map[worklog.Phase]float64{}}}
	}
	members := []application.MemberTimeline{{User: u, Tickets: ticket("3759")}, {User: old, Tickets: ticket("3760")}}
	var buf bytes.Buffer
	if err := WriteTeamSheet(&buf, members, ""); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got := f.GetSheetList(); got[5] != "Markrey" || got[6] != "Martin" {
		t.Errorf("member tabs = %v, want Markrey then Martin", got[5:])
	}
	for _, c := range []struct{ sheet, ref, want string }{
		// No BlackBox members: its table is just heading, header and total
		{"Summary", "A11", "Markrey"},
		{"Summary", "A12", "Martin"},
		{"Markrey", "A1", "Markrey — Ticket Progress Timeline"},
		{"SPinno Ticket Timeline", "C5", "Markrey"},
		{"SPinno Ticket Timeline", "C6", "Martin"},
		{"SPinno Ticket Allocation", "J5", "Markrey"},
	} {
		if got, _ := f.GetCellValue(c.sheet, c.ref); got != c.want {
			t.Errorf("%s!%s = %q, want %q", c.sheet, c.ref, got, c.want)
		}
	}
}

// With no SPinno tickets the total still has its place, under the header,
// at zero.
func TestTeamSheetNoSPinnoTickets(t *testing.T) {
	u, err := user.New("bb1", "Martin", "martin", "hash", project.BlackBox, user.RoleMember, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := WriteTeamSheet(&buf, []application.MemberTimeline{{User: u}}, ""); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := f.GetCellValue("SPinno Ticket Timeline", "A5"); got != "SPinno TOTAL" {
		t.Errorf("A5 = %q, want SPinno TOTAL", got)
	}
	if got, _ := f.CalcCellValue("SPinno Ticket Timeline", "L5"); got != "0" && got != "0 h 00 m" {
		t.Errorf("L5 = %q, want 0", got)
	}
}
