package excel

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"crud_app/internal/application"
	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/report"
	"crud_app/internal/domain/worklog"
)

// The team sheet: everyone's tickets laid out, coloured and formatted like
// the team's Ticket_Progress_Timeline spreadsheet. The Summary (formulas
// over the member tabs), the BlackBox task allocation, the SPinno
// parent/child timeline and its task allocation, the Guide, then one tab per
// user.

const (
	summaryTab     = "Summary"
	blackBoxTasks  = "Blackbox Ticket Allocation"
	spinnoTab      = "SPinno Ticket Timeline"
	spinnoTasksTab = "SPinno Ticket Allocation"
	guideTab       = "Guide"

	// A member tab's first ticket row, how far its rows are made ready (with
	// the Total Hours formula, dropdowns and colours) for tickets added by
	// hand, and how far the Summary's formulas look
	memberFirstRow = 4
	memberReadyRow = 500
	memberLastRow  = 1000

	// Hours as Excel durations: "2 h 30 m", a zero as "-" where shown
	durationFmt     = `[h]" h "mm" m"`
	durationDashFmt = `[h]" h "mm" m";;\-`
	// The row of a project's total in the Summary: cells shown as "0.00 hrs"
	nameFmt = `0.00" hrs";;\-`

	// Borders: the light grey of the tables, and black
	grey  = "BFBFBF"
	black = "000000"
)

// The sheet's statuses, in its dropdown's order
var sheetStatuses = []string{"Not Started", "In Progress", "For Review", "Review Back", "Done", "On Hold"}

// The SPinno allocation's own status and PR status choices, as in the team's
// sheet, after the app's
var (
	spinnoTaskStatuses = append(append([]string{}, sheetStatuses...), "Review (Cho-san)", "Review (Sir Jerwine)")
	spinnoPRStatuses   = []string{"Not Started", "In Progress", "Review (Sir Jerwine)", "Merged", "Deployed", "Review (Cho-san)"}
)

// Each SPinno repo's GitHub issues, which the ticket numbers link to
var spinnoIssues = map[project.Repo]string{
	project.RepoSpec: "https://github.com/SPinnoDeveloper/spinno-next-spec/issues/",
	project.RepoWeb:  "https://github.com/SPinnoDeveloper/spinno-next-web/issues/",
}

// ticketURL is the ticket's GitHub issue, "" when not known
func ticketURL(k worklog.TicketKey) string {
	if k.Project != project.SPinno || spinnoIssues[k.Repo] == "" {
		return ""
	}
	return spinnoIssues[k.Repo] + k.Number
}

// Colours of each project's Summary table: its header, the header over the
// phases, its heading, its members' rows, and its total row (fill, text)
type palette struct {
	header, phaseHeader, heading, rowFill, totalFill, totalText string
}

var (
	blackBoxPalette = palette{"1E1E1E", "3F4248", "1E1E1E", "FFFFFF", "000000", "FFFFFF"}
	spinnoPalette   = palette{"3DB9C1", "2B8A90", "3DB9C1", "", "D8F1F3", "000000"}
	allPalette      = palette{"343A40", "343A40", "343A40", "FFFFFF", "343A40", "FFFFFF"}
)

func paletteOf(p project.Project) palette {
	if p == project.SPinno {
		return spinnoPalette
	}
	return blackBoxPalette
}

// WriteTeamSheet writes the users' timelines as the team sheet. subtitle,
// shown under the Summary's title, says what it covers ("" for all time).
func WriteTeamSheet(w io.Writer, members []application.MemberTimeline, subtitle string) error {
	f := excelize.NewFile()
	st := &styler{f: f, ids: map[cellStyle]int{}}
	tabs := memberTabNames(members)
	spinno := spinnoFamilies(members)

	if err := writeSummary(f, st, members, tabs, subtitle); err != nil {
		return err
	}
	if err := writeBlackBoxTasks(f, st, members); err != nil {
		return err
	}
	if err := writeSPinnoTimeline(f, st, members, spinno); err != nil {
		return err
	}
	if err := writeSPinnoTasks(f, st, members, spinno); err != nil {
		return err
	}
	if err := writeGuide(f, st); err != nil {
		return err
	}
	for i, m := range members {
		if err := writeMemberTab(f, st, tabs[i], m); err != nil {
			return err
		}
	}
	// No values are stored with the formulas: Excel works them out on open
	if err := f.SetCalcProps(&excelize.CalcPropsOptions{FullCalcOnLoad: boolPtr(true)}); err != nil {
		return err
	}
	return save(f, w)
}

func boolPtr(b bool) *bool { return &b }

// ---- Styles -------------------------------------------------------------------

// cellStyle is one look of a cell; styler makes each look once.
type cellStyle struct {
	family    string // "" for Arial
	size      float64
	bold      bool
	italic    bool
	underline bool
	color     string // text colour, "" for the default
	fill      string // "" for none
	numFmt    string
	align     string // horizontal: "", "center", "left"
	vertical  string // "" for center
	wrap      bool
	border    string // colour of a thin border all round, "" for none
	total     bool   // a total row's border: thin sides, medium top, double bottom
}

type styler struct {
	f   *excelize.File
	ids map[cellStyle]int
}

func (s *styler) id(cs cellStyle) (int, error) {
	if id, ok := s.ids[cs]; ok {
		return id, nil
	}
	family, size := cs.family, cs.size
	if family == "" {
		family = "Arial"
	}
	if size == 0 {
		size = 10
		if family == "Calibri" {
			size = 11
		}
	}
	vertical := cs.vertical
	if vertical == "" {
		vertical = "center"
	}
	font := &excelize.Font{Family: family, Size: size, Bold: cs.bold, Italic: cs.italic, Color: cs.color}
	if cs.underline {
		font.Underline = "single"
	}
	style := &excelize.Style{
		Font:      font,
		Alignment: &excelize.Alignment{Horizontal: cs.align, Vertical: vertical, WrapText: cs.wrap},
	}
	if cs.fill != "" {
		style.Fill = excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{cs.fill}}
	}
	if cs.numFmt != "" {
		numFmt := cs.numFmt
		style.CustomNumFmt = &numFmt
	}
	switch {
	case cs.total:
		style.Border = []excelize.Border{
			{Type: "left", Color: black, Style: 1}, {Type: "right", Color: black, Style: 1},
			{Type: "top", Color: black, Style: 2}, {Type: "bottom", Color: black, Style: 6},
		}
	case cs.border != "":
		for _, side := range []string{"left", "top", "right", "bottom"} {
			style.Border = append(style.Border, excelize.Border{Type: side, Color: cs.border, Style: 1})
		}
	}
	id, err := s.f.NewStyle(style)
	if err != nil {
		return 0, err
	}
	s.ids[cs] = id
	return id, nil
}

// put writes a value (a string starting with "=" is a formula) and styles it.
func (s *styler) put(sheet, cell string, value any, cs cellStyle) error {
	if err := s.set(sheet, cell, value); err != nil {
		return err
	}
	return s.style(sheet, cell, cell, cs)
}

// set writes a value, or a formula, leaving the cell's look as it is.
func (s *styler) set(sheet, cell string, value any) error {
	if text, ok := value.(string); ok && strings.HasPrefix(text, "=") {
		return s.f.SetCellFormula(sheet, cell, text[1:])
	}
	if value == nil {
		return nil
	}
	return s.f.SetCellValue(sheet, cell, value)
}

func (s *styler) style(sheet, from, to string, cs cellStyle) error {
	id, err := s.id(cs)
	if err != nil {
		return err
	}
	return s.f.SetCellStyle(sheet, from, to, id)
}

// merge styles a run of cells alike (so the border goes all round), puts the
// value in the first and merges them.
func (s *styler) merge(sheet string, row, from, to int, value any, cs cellStyle) error {
	if err := s.style(sheet, ref(from, row), ref(to, row), cs); err != nil {
		return err
	}
	if err := s.set(sheet, ref(from, row), value); err != nil {
		return err
	}
	if to == from {
		return nil
	}
	return s.f.MergeCell(sheet, ref(from, row), ref(to, row))
}

// link makes the cell open url, "" for none.
func (s *styler) link(sheet, cell, url string) error {
	if url == "" {
		return nil
	}
	return s.f.SetCellHyperLink(sheet, cell, url, "External")
}

// setWidths sets column widths in characters, 0 leaving one as it is.
func setWidths(f *excelize.File, sheet string, widths []float64) error {
	for i, w := range widths {
		if w == 0 {
			continue
		}
		if err := f.SetColWidth(sheet, colName(i+1), colName(i+1), w); err != nil {
			return err
		}
	}
	return nil
}

func setHeights(f *excelize.File, sheet string, height float64, rows ...int) error {
	for _, r := range rows {
		if err := f.SetRowHeight(sheet, r, height); err != nil {
			return err
		}
	}
	return nil
}

func freeze(f *excelize.File, sheet string, cols, rows int) error {
	pane := "bottomRight"
	if cols == 0 {
		pane = "bottomLeft"
	}
	return f.SetPanes(sheet, &excelize.Panes{
		Freeze: true, XSplit: cols, YSplit: rows,
		TopLeftCell: ref(cols+1, rows+1), ActivePane: pane,
	})
}

func dropdown(f *excelize.File, sheet, sqref string, choices []string) error {
	if sqref == "" || len(choices) == 0 {
		return nil
	}
	dv := excelize.NewDataValidation(true)
	dv.SetSqref(sqref)
	if err := dv.SetDropList(choices); err != nil {
		return err
	}
	return f.AddDataValidation(sheet, dv)
}

// quoted is a sheet name as a formula names it: 'Name'!A1
func quoted(sheet string) string {
	return "'" + strings.ReplaceAll(sheet, "'", "''") + "'"
}

// memberTabNames names each user's tab after them: characters a sheet name
// can't have are dropped, and a second "Martin" gets "Martin (2)".
func memberTabNames(members []application.MemberTimeline) []string {
	taken := map[string]bool{}
	for _, name := range []string{summaryTab, blackBoxTasks, spinnoTab, spinnoTasksTab, guideTab} {
		taken[strings.ToLower(name)] = true
	}
	names := make([]string, len(members))
	for i, m := range members {
		base := strings.Map(func(r rune) rune {
			if strings.ContainsRune(`[]:*?/\`, r) {
				return -1
			}
			return r
		}, strings.TrimSpace(m.User.ShortName()))
		base = strings.Trim(base, "'")
		if base == "" {
			base = "Member"
		}
		name := truncate(base, 31)
		for n := 2; taken[strings.ToLower(name)]; n++ {
			suffix := fmt.Sprintf(" (%d)", n)
			name = truncate(base, 31-len([]rune(suffix))) + suffix
		}
		taken[strings.ToLower(name)] = true
		names[i] = name
	}
	return names
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// excelDate is a day ("2026-11-03") as an Excel date number, nil for none.
// A number, not a time, so the cell keeps its yyyy-mm-dd format.
func excelDate(day string) any {
	d, err := time.Parse(calendar.DayLayout, day)
	if err != nil {
		return nil
	}
	return d.Sub(time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)).Hours() / 24
}

// duration is hours as an Excel duration (days), to the minute.
func duration(hours float64) float64 {
	return math.Round(hours*60) / (24 * 60)
}

// sheetStatus is a ticket's status as the sheet's dropdown names it.
func sheetStatus(t report.TicketRow) string {
	switch t.Status {
	case worklog.StatusNotStarted:
		return "Not Started"
	case worklog.StatusWaiting:
		return "For Review"
	case worklog.StatusBlocked:
		return "On Hold"
	case worklog.StatusDone:
		return "Done"
	}
	if t.ReviewBacks > 0 {
		return "Review Back"
	}
	return "In Progress"
}

// ticketID is the Ticket ID the sheet shows: "#3287", or "Spec#181",
// "Web#3759" and "Backend#120" for SPinno.
func ticketID(k worklog.TicketKey) string {
	if k.Repo == "" {
		return "#" + k.Number
	}
	repo := string(k.Repo)
	return strings.ToUpper(repo[:1]) + repo[1:] + "#" + k.Number
}

// ticketNumber is the Ticket ID a member's tab shows: the number alone,
// as a number (3287), or as written if it isn't one.
func ticketNumber(k worklog.TicketKey) any {
	if n, err := strconv.Atoi(k.Number); err == nil {
		return n
	}
	return k.Number
}

// isParentTicket: a spec with no type, which tickets go under
func isParentTicket(t report.TicketRow) bool {
	return t.Ticket.Repo == project.RepoSpec && t.Type == ""
}

// byNumber orders ticket numbers as numbers: 99 before 120.
func byNumber(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// ---- Summary -----------------------------------------------------------------

func writeSummary(f *excelize.File, st *styler, members []application.MemberTimeline, tabs []string, subtitle string) error {
	sh := summaryTab
	if err := newTab(f, sh); err != nil {
		return err
	}
	if err := setWidths(f, sh, []float64{32.4, 9, 8, 13, 14, 15, 16.7, 10, 15.3, 11, 0, 14.6, 14.9}); err != nil {
		return err
	}
	if err := st.put(sh, "A1", "Team Ticket Timeline — Summary", cellStyle{size: 22, bold: true, color: "212529", vertical: "bottom"}); err != nil {
		return err
	}
	if subtitle != "" {
		if err := st.put(sh, "A2", subtitle, cellStyle{italic: true, color: "7F7F7F", vertical: "bottom"}); err != nil {
			return err
		}
	}
	if err := freeze(f, sh, 0, 3); err != nil {
		return err
	}

	row := 4
	totals := map[project.Project]int{} // each project's total row
	for _, p := range project.All() {
		var rows []int
		pal := paletteOf(p)
		if err := st.put(sh, ref(1, row), p.String()+" Project", cellStyle{size: 12, bold: true, color: pal.heading, vertical: "bottom"}); err != nil {
			return err
		}
		row++
		if err := summaryHeader(f, st, row, "Member", pal); err != nil {
			return err
		}
		row++
		// The project's team, and anyone else with tickets in it
		for i, m := range members {
			if m.User.Team() != p && !hasProject(m, p) {
				continue
			}
			if err := summaryMemberRow(st, row, m.User.ShortName(), quoted(tabs[i]), p, pal.rowFill); err != nil {
				return err
			}
			rows = append(rows, row)
			row++
		}
		label := p.String() + " TOTAL"
		if p == project.BlackBox {
			label = "BLACKBOX TOTAL"
		}
		sum := func(col int) string {
			if len(rows) == 0 {
				return "=0"
			}
			return fmt.Sprintf("=SUM(%s:%s)", ref(col, rows[0]), ref(col, rows[len(rows)-1]))
		}
		if err := summaryTotalRow(st, row, label, sum, pal); err != nil {
			return err
		}
		totals[p] = row
		row += 3
	}

	// All Projects: each project's total, then the grand total
	if err := st.put(sh, ref(1, row), "All Projects", cellStyle{size: 12, bold: true, color: allPalette.heading, vertical: "bottom"}); err != nil {
		return err
	}
	row++
	if err := summaryHeader(f, st, row, "Project", allPalette); err != nil {
		return err
	}
	row++
	first := row
	for _, p := range project.All() {
		total := totals[p]
		look := cellStyle{bold: true, color: "1E1E1E", fill: "FFFFFF", border: grey, vertical: "bottom", numFmt: nameFmt}
		if err := st.put(sh, ref(1, row), p.String(), look); err != nil {
			return err
		}
		for col := 2; col <= 13; col++ {
			value := fmt.Sprintf("=%s", ref(col, total))
			if col == 11 || col == 13 {
				value = ratio(row, col)
			}
			if err := st.put(sh, ref(col, row), value, summaryCell(col, cellStyle{color: "000000", fill: "FFFFFF"})); err != nil {
				return err
			}
		}
		row++
	}
	last := row - 1
	sum := func(col int) string { return fmt.Sprintf("=SUM(%s:%s)", ref(col, first), ref(col, last)) }
	return summaryTotalRow(st, row, "GRAND TOTAL", sum, allPalette)
}

// summaryHeader: the first column's name, Tickets and Done, the phases,
// then the totals and averages
func summaryHeader(f *excelize.File, st *styler, row int, first string, pal palette) error {
	names := []string{first, "Tickets", "Done"}
	for _, ph := range worklog.Phases() {
		names = append(names, ph.Label())
	}
	names = append(names, "Total Hours", "Avg Hours / Ticket", "Total Review-Backs", "Avg Review-Backs / Ticket")
	for i, name := range names {
		fill := pal.header
		if i >= 3 && i < 3+len(worklog.Phases()) {
			fill = pal.phaseHeader
		}
		cs := cellStyle{bold: true, color: "FFFFFF", fill: fill, align: "center", wrap: true, border: grey}
		if err := st.put(summaryTab, ref(i+1, row), name, cs); err != nil {
			return err
		}
	}
	return f.SetRowHeight(summaryTab, row, 45)
}

// summaryCell styles a Summary number: counts as whole numbers, hours as
// durations, average review-backs to two places.
func summaryCell(col int, cs cellStyle) cellStyle {
	if !cs.total {
		cs.border = grey
	}
	cs.vertical = "bottom"
	switch {
	case col == 2 || col == 3 || col == 12:
		cs.numFmt = "0"
	case col == 13:
		cs.numFmt = "0.00"
	default:
		cs.numFmt = durationDashFmt
	}
	return cs
}

// ratio: average hours (column K) or review-backs (column M) per ticket
func ratio(row, col int) string {
	return fmt.Sprintf("=IF(%s=0,0,%s/%s)", ref(2, row), ref(col-1, row), ref(2, row))
}

// hasProject: the member has a ticket in the project
func hasProject(m application.MemberTimeline, p project.Project) bool {
	for _, t := range m.Tickets {
		if t.Ticket.Project == p {
			return true
		}
	}
	return false
}

// summaryMemberRow adds up the member's tickets in project p only, by their
// tab's Project column, so a ticket counts in its own project's table.
func summaryMemberRow(st *styler, row int, name, tab string, p project.Project, fill string) error {
	// The member's first name, underlined as in the team's sheet. Not a
	// link to their tab: Google Sheets shows such a link as "#gid=…" in
	// place of the name.
	look := cellStyle{bold: true, underline: true, color: "000000", fill: fill, border: grey, vertical: "bottom", numFmt: nameFmt}
	if err := st.put(summaryTab, ref(1, row), name, look); err != nil {
		return err
	}
	rng := func(col string) string {
		return fmt.Sprintf("%s!%s%d:%s%d", tab, col, memberFirstRow, col, memberLastRow)
	}
	sumIf := func(col string) string {
		return fmt.Sprintf(`=SUMIF(%s,"%s",%s)`, rng("A"), p, rng(col))
	}
	values := []string{
		fmt.Sprintf(`=COUNTIF(%s,"%s")`, rng("A"), p),
		fmt.Sprintf(`=COUNTIFS(%s,"%s",%s,"Done")`, rng("A"), p, rng("D")),
	}
	for i := range worklog.Phases() {
		values = append(values, sumIf(colName(7+i)))
	}
	values = append(values,
		fmt.Sprintf("=SUM(%s:%s)", ref(4, row), ref(9, row)),
		ratio(row, 11),
		sumIf("M"),
		ratio(row, 13),
	)
	for i, v := range values {
		col := i + 2
		// What comes from the member's tab is green; what's worked out here black
		color := "000000"
		if col <= 9 || col == 12 {
			color = "008000"
		}
		if err := st.put(summaryTab, ref(col, row), v, summaryCell(col, cellStyle{color: color, fill: fill})); err != nil {
			return err
		}
	}
	return nil
}

func summaryTotalRow(st *styler, row int, label string, sum func(col int) string, pal palette) error {
	look := cellStyle{bold: true, color: pal.totalText, fill: pal.totalFill, total: true, vertical: "bottom"}
	name := look
	if pal == allPalette {
		name.numFmt = nameFmt
	}
	if err := st.put(summaryTab, ref(1, row), label, name); err != nil {
		return err
	}
	for col := 2; col <= 13; col++ {
		value := sum(col)
		if col == 11 || col == 13 {
			value = ratio(row, col)
		}
		if err := st.put(summaryTab, ref(col, row), value, summaryCell(col, look)); err != nil {
			return err
		}
	}
	return nil
}

// ---- SPinno families ---------------------------------------------------------

// spinnoTicket is a SPinno ticket in the team's timeline, its time added up
// over everyone who has it.
type spinnoTicket struct {
	row       report.TicketRow
	assignees []string
	hours     map[worklog.Phase]float64
	reviews   int
	parent    bool
}

// spinnoFamily is a parent and the tickets under it, or (parent nil) one
// ticket under no parent.
type spinnoFamily struct {
	parent *spinnoTicket
	kids   []*spinnoTicket
}

// spinnoFamilies: tickets under no parent first, then each parent with its
// tickets, in ticket number order.
func spinnoFamilies(members []application.MemberTimeline) []spinnoFamily {
	tickets := map[worklog.TicketKey]*spinnoTicket{}
	for _, m := range members {
		for _, t := range m.Tickets {
			if t.Ticket.Project != project.SPinno {
				continue
			}
			s, ok := tickets[t.Ticket]
			if !ok {
				s = &spinnoTicket{row: t, hours: map[worklog.Phase]float64{}}
				tickets[t.Ticket] = s
			}
			s.assignees = append(s.assignees, m.User.ShortName())
			for ph, h := range t.PhaseHours {
				s.hours[ph] += h
			}
			s.reviews = max(s.reviews, t.ReviewBacks)
			s.parent = s.parent || isParentTicket(t)
		}
	}
	children := map[string][]*spinnoTicket{}
	var parents, alone []*spinnoTicket
	for _, s := range tickets {
		switch {
		case s.parent:
			parents = append(parents, s)
		case s.row.Parent != "":
			children[s.row.Parent] = append(children[s.row.Parent], s)
		default:
			alone = append(alone, s)
		}
	}
	// A parent only someone else has: its tickets still go under it
	for number := range children {
		key := worklog.TicketKey{Project: project.SPinno, Repo: project.RepoSpec, Number: number}
		if t, ok := tickets[key]; !ok || !t.parent {
			parents = append(parents, &spinnoTicket{row: report.TicketRow{Ticket: key}, hours: map[worklog.Phase]float64{}, parent: true})
		}
	}
	order := func(list []*spinnoTicket) {
		sort.Slice(list, func(i, j int) bool {
			a, b := list[i].row.Ticket, list[j].row.Ticket
			if a.Number != b.Number {
				return byNumber(a.Number, b.Number)
			}
			return a.Repo < b.Repo
		})
	}
	order(parents)
	order(alone)
	var families []spinnoFamily
	for _, s := range alone {
		families = append(families, spinnoFamily{kids: []*spinnoTicket{s}})
	}
	for _, p := range parents {
		kids := children[p.row.Ticket.Number]
		order(kids)
		families = append(families, spinnoFamily{parent: p, kids: kids})
	}
	return families
}

// rowsOf lists the families' rows in order: each parent, then its tickets.
func rowsOf(families []spinnoFamily) []*spinnoTicket {
	var rows []*spinnoTicket
	for _, fam := range families {
		if fam.parent != nil {
			rows = append(rows, fam.parent)
		}
		rows = append(rows, fam.kids...)
	}
	return rows
}

func spinnoNames(members []application.MemberTimeline) []string {
	var names []string
	for _, m := range members {
		if m.User.Team() == project.SPinno {
			names = append(names, m.User.ShortName())
		}
	}
	return names
}

// ---- SPinno Ticket Timeline ----------------------------------------------------

func writeSPinnoTimeline(f *excelize.File, st *styler, members []application.MemberTimeline, families []spinnoFamily) error {
	sh := spinnoTab
	if err := newTab(f, sh); err != nil {
		return err
	}
	if err := setWidths(f, sh, []float64{32.4, 14, 0, 9, 3.4, 18, 16.7, 15.4, 19.1, 11, 18.9, 14.6}); err != nil {
		return err
	}
	if err := st.put(sh, "A1", "Team Ticket Timeline — Summary", cellStyle{size: 22, bold: true, color: "212529", vertical: "bottom"}); err != nil {
		return err
	}
	if err := st.put(sh, "A3", "SPinno Project", cellStyle{size: 17, bold: true, color: "3DB9C1", fill: "ECECEE", vertical: "bottom"}); err != nil {
		return err
	}
	if err := setHeights(f, sh, 23.25, 1, 2, 3); err != nil {
		return err
	}
	if err := freeze(f, sh, 0, 1); err != nil {
		return err
	}

	// Header
	head := func(fill string) cellStyle {
		return cellStyle{bold: true, color: "FFFFFF", fill: fill, align: "center", wrap: true, border: black}
	}
	if err := st.put(sh, "A4", "Ticket #", head("3DB9C1")); err != nil {
		return err
	}
	if err := st.put(sh, "B4", "Label", head("3DB9C1")); err != nil {
		return err
	}
	if err := st.merge(sh, 4, 3, 5, "Assignee", head("3DB9C1")); err != nil {
		return err
	}
	for i, ph := range worklog.Phases() {
		if err := st.put(sh, ref(6+i, 4), ph.Label(), head("2B8A90")); err != nil {
			return err
		}
	}
	if err := st.put(sh, "L4", "Total Hours", head("3DB9C1")); err != nil {
		return err
	}
	if err := st.put(sh, "M4", "Total Review-Backs", head("3DB9C1")); err != nil {
		return err
	}
	if err := f.SetRowHeight(sh, 4, 45); err != nil {
		return err
	}

	row := 5
	first := row
	var parentRows, childRows []string
	for _, fam := range families {
		if fam.parent != nil {
			if err := spinnoParentRow(st, row, fam.parent, len(fam.kids)); err != nil {
				return err
			}
			parentRows = append(parentRows, fmt.Sprintf("B%d", row))
			row++
		}
		for _, k := range fam.kids {
			if err := spinnoChildRow(st, row, k); err != nil {
				return err
			}
			childRows = append(childRows, fmt.Sprintf("B%d", row))
			row++
		}
	}
	last := row - 1

	// The total, right under the last ticket however many there are: every
	// child (a parent's row is its children's); 0 with no tickets
	look := cellStyle{bold: true, fill: "D8F1F3", total: true, vertical: "bottom"}
	if err := st.put(sh, ref(1, row), "SPinno TOTAL", look); err != nil {
		return err
	}
	if err := st.style(sh, ref(2, row), ref(5, row), look); err != nil {
		return err
	}
	for col := 6; col <= 12; col++ {
		cs := look
		cs.numFmt = durationFmt
		value := "=0"
		if last >= first {
			value = fmt.Sprintf(`=SUMIF($B$%d:$B$%d,"Child",%s:%s)`, first, last, ref(col, first), ref(col, last))
		}
		if err := st.put(sh, ref(col, row), value, cs); err != nil {
			return err
		}
	}
	if err := f.SetRowHeight(sh, row, 23.25); err != nil {
		return err
	}

	// Dropdowns, as in the team's sheet
	if err := dropdown(f, sh, strings.Join(parentRows, " "), []string{"Epic", "Child", "Parent"}); err != nil {
		return err
	}
	if err := dropdown(f, sh, strings.Join(childRows, " "), []string{"Epic", "Child"}); err != nil {
		return err
	}
	if last >= first {
		return dropdown(f, sh, fmt.Sprintf("C%d:C%d", first, last), spinnoNames(members))
	}
	return nil
}

// spinnoTicketCells writes a row's ticket (a link), label and assignees (C
// to E). look is the row's colours.
func spinnoTicketCells(st *styler, row int, s *spinnoTicket, label string, look cellStyle) error {
	sh := spinnoTab
	id := look
	id.family, id.size, id.bold, id.underline, id.align, id.wrap, id.border = "Calibri", 12, true, true, "center", true, black
	if err := st.put(sh, ref(1, row), ticketID(s.row.Ticket), id); err != nil {
		return err
	}
	if err := st.link(sh, ref(1, row), ticketURL(s.row.Ticket)); err != nil {
		return err
	}
	// The label and assignees: plain on a child's row; white on a parent's
	// black row, where the name is black too, as in the team's sheet
	plain := cellStyle{family: "Calibri", align: "center", border: black}
	if label == "Parent" {
		plain.color, plain.fill = "FFFFFF", "000000"
	}
	if err := st.put(sh, ref(2, row), label, plain); err != nil {
		return err
	}
	who := plain
	who.bold = true
	if label == "Parent" {
		who.color = "000000"
	}
	if err := st.merge(sh, row, 3, 5, strings.Join(s.assignees, ", "), who); err != nil {
		return err
	}
	return st.f.SetRowHeight(sh, row, 23.25)
}

func spinnoChildRow(st *styler, row int, s *spinnoTicket) error {
	if err := spinnoTicketCells(st, row, s, "Child", cellStyle{color: "0000FF", fill: "FFFFFF"}); err != nil {
		return err
	}
	hours := cellStyle{size: 11, color: "008000", align: "center", numFmt: durationFmt, border: black}
	for i, ph := range worklog.Phases() {
		var value any = "-"
		if h := s.hours[ph]; h > 0 {
			value = duration(h)
		}
		if err := st.put(spinnoTab, ref(6+i, row), value, hours); err != nil {
			return err
		}
	}
	total := fmt.Sprintf(`=IF(COUNT(F%d:K%d)=0,"-",SUM(F%d:K%d))`, row, row, row, row)
	if err := st.put(spinnoTab, ref(12, row), total, hours); err != nil {
		return err
	}
	reviews := hours
	reviews.numFmt = ""
	return st.put(spinnoTab, ref(13, row), s.reviews, reviews)
}

// spinnoParentRow: its columns add up the kids' rows just below it, plus
// any time of its own (logged while it was a child)
func spinnoParentRow(st *styler, row int, s *spinnoTicket, kids int) error {
	if err := spinnoTicketCells(st, row, s, "Parent", cellStyle{color: "FFFFFF", fill: "000000"}); err != nil {
		return err
	}
	hours := cellStyle{size: 11, color: "FFFFFF", fill: "000000", align: "center", numFmt: durationFmt, border: black}
	sumKids := func(col int) string {
		if kids == 0 {
			return "0"
		}
		return fmt.Sprintf("SUM(%s:%s)", ref(col, row+1), ref(col, row+kids))
	}
	for i, ph := range worklog.Phases() {
		value := "=" + sumKids(6+i)
		if own := duration(s.hours[ph]); own > 0 {
			value = fmt.Sprintf("=%s+%s", sumKids(6+i), trimFloat(own))
		}
		if err := st.put(spinnoTab, ref(6+i, row), value, hours); err != nil {
			return err
		}
	}
	total := fmt.Sprintf(`=IF(COUNT(F%d:K%d)=0,"",SUM(F%d:K%d))`, row, row, row, row)
	if err := st.put(spinnoTab, ref(12, row), total, hours); err != nil {
		return err
	}
	reviews := hours
	reviews.numFmt = ""
	return st.put(spinnoTab, ref(13, row), fmt.Sprintf("=%s+%d", sumKids(13), s.reviews), reviews)
}

func trimFloat(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.10f", v), "0"), ".")
}

// ---- Allocation of task --------------------------------------------------------

// allocationTitle: "ALLOCATION OF TASK" across the table, boxed
func allocationTitle(st *styler, sheet string, to int, height float64) error {
	look := cellStyle{size: 16, bold: true, color: "343A40", align: "center", border: black}
	if err := st.merge(sheet, 1, 1, to, "ALLOCATION OF TASK", look); err != nil {
		return err
	}
	return st.f.SetRowHeight(sheet, 1, height)
}

// writeBlackBoxTasks: each BlackBox ticket and who it's assigned to. Who
// assigned it isn't kept by the app, so Assignee is left to fill in.
func writeBlackBoxTasks(f *excelize.File, st *styler, members []application.MemberTimeline) error {
	sh := blackBoxTasks
	if err := newTab(f, sh); err != nil {
		return err
	}
	if err := setWidths(f, sh, []float64{32.4, 9, 8, 0, 23.4, 24, 13, 14, 15, 16.7}); err != nil {
		return err
	}
	if err := allocationTitle(st, sh, 10, 35.25); err != nil {
		return err
	}
	if err := st.merge(sh, 2, 1, 10, "BlackBox", cellStyle{size: 12, bold: true, color: "FFFFFF", fill: "000000", border: black}); err != nil {
		return err
	}
	if err := f.SetRowHeight(sh, 2, 35.25); err != nil {
		return err
	}
	head := cellStyle{bold: true, color: "000000", fill: "EFEFEF", align: "center", wrap: true, border: black}
	for _, h := range []struct {
		from, to int
		name     string
	}{{1, 4, "Task"}, {5, 5, "Assignee"}, {6, 6, "Status"}, {7, 10, "Assigned To"}} {
		if err := st.merge(sh, 3, h.from, h.to, h.name, head); err != nil {
			return err
		}
	}

	row := 4
	look := cellStyle{bold: true, color: "1E1E1E", fill: "FFFFFF", border: black, wrap: true}
	for _, m := range members {
		for _, t := range m.Tickets {
			if t.Ticket.Project != project.BlackBox {
				continue
			}
			task := ticketID(t.Ticket)
			if t.Title != "" {
				task += " " + t.Title
			}
			for _, c := range []struct {
				from, to int
				value    any
			}{{1, 4, task}, {5, 5, nil}, {6, 6, sheetStatus(t)}, {7, 10, m.User.ShortName()}} {
				if err := st.merge(sh, row, c.from, c.to, c.value, look); err != nil {
					return err
				}
			}
			if err := f.SetRowHeight(sh, row, 26.25); err != nil {
				return err
			}
			row++
		}
	}
	if row == 4 {
		return nil
	}
	return dropdown(f, sh, fmt.Sprintf("F4:F%d", row-1), sheetStatuses)
}

// writeSPinnoTasks: the SPinno families again, with each ticket's task and
// who it's assigned to. PRs aren't kept by the app: PR and PR Status are
// left to fill in.
func writeSPinnoTasks(f *excelize.File, st *styler, members []application.MemberTimeline, families []spinnoFamily) error {
	sh := spinnoTasksTab
	if err := newTab(f, sh); err != nil {
		return err
	}
	if err := setWidths(f, sh, []float64{19.1, 13.1, 9, 8, 0, 37.6, 26.3, 14, 23, 16.7, 10, 3.9}); err != nil {
		return err
	}
	if err := allocationTitle(st, sh, 12, 27.75); err != nil {
		return err
	}
	if err := st.style(sh, "A2", "J2", cellStyle{family: "Calibri"}); err != nil {
		return err
	}
	if err := f.MergeCell(sh, "B2", "J2"); err != nil {
		return err
	}
	if err := st.merge(sh, 3, 1, 12, "SPinno", cellStyle{size: 12, bold: true, color: "FFFFFF", fill: "3DB9C1", border: black}); err != nil {
		return err
	}
	head := cellStyle{bold: true, color: "000000", fill: "D8F1F3", align: "center", wrap: true, border: black}
	calibriHead := cellStyle{family: "Calibri", bold: true, fill: "D8F1F3", align: "center", border: black}
	for _, h := range []struct {
		from, to int
		name     string
		look     cellStyle
	}{
		{1, 1, "Ticket #", head}, {2, 2, "Label", head}, {3, 6, "Task", head}, {7, 7, "Status", head},
		{8, 8, "PR", calibriHead}, {9, 9, "PR Status", calibriHead}, {10, 12, "Assigned To", calibriHead},
	} {
		if err := st.merge(sh, 4, h.from, h.to, h.name, h.look); err != nil {
			return err
		}
	}
	if err := setHeights(f, sh, 27.75, 2, 3, 4); err != nil {
		return err
	}

	row := 5
	var parentRows, childRows []string
	for _, s := range rowsOf(families) {
		// Parent rows are shaded grey, as in the team's sheet
		shade := ""
		label := "Child"
		if s.parent {
			shade, label = "F3F3F3", "Parent"
		}
		idFill := shade
		if idFill == "" {
			idFill = "FFFFFF"
		}
		id := cellStyle{family: "Calibri", size: 12, bold: true, underline: true, color: "0000FF", fill: idFill, align: "center", wrap: true, border: black}
		if err := st.put(sh, ref(1, row), ticketID(s.row.Ticket), id); err != nil {
			return err
		}
		if err := st.link(sh, ref(1, row), ticketURL(s.row.Ticket)); err != nil {
			return err
		}
		calibri := cellStyle{family: "Calibri", fill: shade, border: black}
		if err := st.put(sh, ref(2, row), label, calibri); err != nil {
			return err
		}
		task := cellStyle{family: "Calibri", size: 12, color: "000000", fill: shade, wrap: true, border: black}
		if err := st.merge(sh, row, 3, 6, s.row.Title, task); err != nil {
			return err
		}
		status := cellStyle{bold: true, color: "1E1E1E", fill: idFill, border: black}
		value := any(nil)
		if s.row.Status != "" {
			value = sheetStatus(s.row)
		}
		if err := st.put(sh, ref(7, row), value, status); err != nil {
			return err
		}
		pr := calibri
		pr.align = "center"
		if err := st.put(sh, ref(8, row), nil, pr); err != nil {
			return err
		}
		if err := st.put(sh, ref(9, row), nil, calibri); err != nil {
			return err
		}
		who := cellStyle{family: "Calibri", bold: true, align: "center", border: black}
		if err := st.merge(sh, row, 10, 12, strings.Join(s.assignees, ", "), who); err != nil {
			return err
		}
		if err := f.SetRowHeight(sh, row, 27.75); err != nil {
			return err
		}
		if s.parent {
			parentRows = append(parentRows, fmt.Sprintf("B%d", row))
		} else {
			childRows = append(childRows, fmt.Sprintf("B%d", row))
		}
		row++
	}
	if row == 5 {
		return nil
	}
	last := row - 1
	for _, dv := range []struct {
		sqref   string
		choices []string
	}{
		{strings.Join(parentRows, " "), []string{"Epic", "Child", "Parent"}},
		{strings.Join(childRows, " "), []string{"Epic", "Child"}},
		{fmt.Sprintf("G5:G%d", last), spinnoTaskStatuses},
		{fmt.Sprintf("I5:I%d", last), spinnoPRStatuses},
		{fmt.Sprintf("J5:J%d", last), spinnoNames(members)},
	} {
		if dv.sqref == "" {
			continue
		}
		if err := dropdown(f, sh, dv.sqref, dv.choices); err != nil {
			return err
		}
	}
	return nil
}

// ---- Guide -------------------------------------------------------------------

// Each line of the guide: what (bold when a step), then its explanation
var guideSteps = []struct {
	label, text string
	bold        bool
}{
	{"1. Go to your own tab (your name).", "One row = one ticket. Fill the white cells only (Total Hours is shaded and fills itself in).", true},
	{"2. Ticket details", "Project, Ticket ID, Title, Status (dropdown), Start Date, End Date (yyyy-mm-dd).", true},
	{"3. Timeline — hours per phase", "Log hours (decimals OK, e.g. 1.5) under each phase. Leave blank / 0 if a phase didn't apply.", true},
	{"   Check Spec / Investigation", "Reading the spec, reproducing, investigating the issue.", false},
	{"   Confirmation / Review to SE/MGRs", "Time spent confirming the approach / spec with SE or managers.", false},
	{"   Development / Spec Changes (Planning)", "Planning the solution, spec updates before coding.", false},
	{"   Coding (Implementation)", "Actual coding work.", false},
	{"   Testing", "Self-testing / QA of your implementation.", false},
	{"   Review After Implementation", "Total hours spent on fixes after review comments (all review-back rounds combined).", false},
	{"4. Review-Back Count (#)", "HOW MANY TIMES the ticket was returned after review. 0 = approved first time. Turns red at 2+.", true},
	{"5. Total Hours", "Auto-calculated = sum of all 6 phases. Don't type here.", true},
	{"6. Summary tab", "Separate tables for BlackBox and SPinno, then an All Projects roll-up. Auto-totals per member: tickets, hours per phase, average hours per ticket, review-backs.", true},
	{"Adding a new member", "Right-click any member tab → Move or Copy → Create a copy, rename it, clear the rows, then add a row in the matching project table on the Summary tab (copy an existing member row and change the tab name in its formulas).", false},
}

// The example row under the steps: each field and its example value
var guideExample = [][2]any{
	{"Project", "BlackBox"}, {"Ticket ID", "BB-1234"}, {"Title", "Fix login token refresh"},
	{"Status", "Done"}, {"Start", "2026-11-02"}, {"End", "2026-11-04"},
	{"Spec/Inv", 2}, {"Confirm", 0.5}, {"Planning", 1.5}, {"Coding", 6}, {"Testing", 2}, {"Review After", 1.5},
	{"Review-Back #", "2   ← returned twice after review; the 1.5 hrs above covers both rework rounds"},
	{"Total Hrs", 13.5},
}

func writeGuide(f *excelize.File, st *styler) error {
	sh := guideTab
	if err := newTab(f, sh); err != nil {
		return err
	}
	if err := setWidths(f, sh, []float64{34, 90, 8.7}); err != nil {
		return err
	}
	if err := st.put(sh, "A1", "How to use this tracker", cellStyle{size: 14, bold: true, color: "212529", vertical: "top"}); err != nil {
		return err
	}
	if err := st.style(sh, "B1", "B1", cellStyle{wrap: true, vertical: "top"}); err != nil {
		return err
	}
	row := 2
	for _, s := range guideSteps {
		if err := st.put(sh, ref(1, row), s.label, cellStyle{bold: s.bold, color: "000000", vertical: "top"}); err != nil {
			return err
		}
		if err := st.put(sh, ref(2, row), s.text, cellStyle{wrap: true, vertical: "top"}); err != nil {
			return err
		}
		row++
	}
	row++ // a blank line
	if err := st.put(sh, ref(1, row), "Example row (format reference)", cellStyle{bold: true, color: "212529", vertical: "top"}); err != nil {
		return err
	}
	if err := st.style(sh, ref(2, row), ref(2, row), cellStyle{wrap: true, vertical: "top"}); err != nil {
		return err
	}
	row++
	for _, e := range guideExample {
		if err := st.put(sh, ref(1, row), e[0], cellStyle{color: "595959", vertical: "bottom"}); err != nil {
			return err
		}
		if err := st.put(sh, ref(2, row), e[1], cellStyle{italic: true, align: "left", vertical: "bottom"}); err != nil {
			return err
		}
		row++
	}
	return setHeights(f, sh, 15.75, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31)
}

// ---- Member tabs -------------------------------------------------------------

func writeMemberTab(f *excelize.File, st *styler, sh string, m application.MemberTimeline) error {
	if err := newTab(f, sh); err != nil {
		return err
	}
	widths := []float64{16, 22, 36, 13, 17, 16, 22, 20, 20, 22, 11, 26, 18.9, 11, 30}
	if err := setWidths(f, sh, widths); err != nil {
		return err
	}
	if err := st.put(sh, "A1", m.User.ShortName()+" — Ticket Progress Timeline", cellStyle{size: 14, bold: true, color: "1E1E1E", vertical: "bottom"}); err != nil {
		return err
	}

	// Two header rows: the groups, then the columns
	head := func(fill string) cellStyle {
		return cellStyle{bold: true, color: "FFFFFF", fill: fill, align: "center", wrap: true, border: grey}
	}
	for _, g := range []struct {
		from, to int
		name     string
		fill     string
	}{
		{1, 6, "TICKET DETAILS", "1E1E1E"},
		{7, 12, "TIMELINE — HOURS SPENT PER PHASE", "3F4248"},
		{13, 14, "RESULT", "1E1E1E"},
		{15, 15, "", "1E1E1E"},
	} {
		if err := st.merge(sh, 2, g.from, g.to, g.name, head(g.fill)); err != nil {
			return err
		}
	}
	names := []string{"Project", "Ticket ID", "Title / Short Description", "Status", "Start Date", "End Date"}
	for _, ph := range worklog.Phases() {
		names = append(names, ph.Label())
	}
	names = append(names, "Review-Back Count (#)", "Total Hours", "Remarks")
	for i, name := range names {
		fill := "1E1E1E"
		if i >= 6 && i < 12 {
			fill = "3F4248"
		}
		if err := st.put(sh, ref(i+1, 3), name, head(fill)); err != nil {
			return err
		}
	}
	if err := f.SetRowHeight(sh, 2, 15); err != nil {
		return err
	}
	if err := f.SetRowHeight(sh, 3, 45); err != nil {
		return err
	}
	if err := freeze(f, sh, 3, 3); err != nil {
		return err
	}

	// Every row down to memberReadyRow looks the same, ready to fill in:
	// white boxes, the Total Hours column grey with its formula
	first, last := memberFirstRow, memberReadyRow
	cell := cellStyle{fill: "FFFFFF", border: grey}
	center := cell
	center.align = "center"
	id := center
	id.color = "0000FF"
	date := center
	date.numFmt = "yyyy-mm-dd"
	hours := cell
	hours.numFmt = durationFmt
	count := center
	count.numFmt = "0"
	total := center
	total.bold, total.fill, total.numFmt = true, "ECECEE", durationFmt
	text := cell
	text.wrap = true
	looks := []cellStyle{cell, id, text, center, date, date, hours, hours, hours, hours, hours, hours, count, total, text}
	for c, look := range looks {
		if err := st.style(sh, ref(c+1, first), ref(c+1, last), look); err != nil {
			return err
		}
	}
	for r := first; r <= last; r++ {
		formula := fmt.Sprintf(`=IF(COUNT(G%d:L%d)=0,"",SUM(G%d:L%d))`, r, r, r, r)
		if err := st.set(sh, ref(14, r), formula); err != nil {
			return err
		}
	}

	// The member's tickets
	for i, t := range m.Tickets {
		r := first + i
		values := make([]any, 15)
		values[0] = t.Ticket.Project.String()
		values[1] = ticketNumber(t.Ticket)
		values[2] = t.Title
		values[3] = sheetStatus(t)
		values[4] = excelDate(t.StartDate.String())
		if t.EndDate != nil {
			values[5] = excelDate(t.EndDate.String())
		}
		for i, ph := range worklog.Phases() {
			if h := t.PhaseHours[ph]; h > 0 {
				values[6+i] = duration(h)
			}
		}
		if t.ReviewBacks > 0 {
			values[12] = t.ReviewBacks
		}
		if t.Note != "" {
			values[14] = t.Note
		}
		for c, v := range values {
			if err := st.set(sh, ref(c+1, r), v); err != nil {
				return err
			}
		}
		if url := ticketURL(t.Ticket); url != "" {
			linked := id
			linked.underline = true
			if err := st.style(sh, ref(2, r), ref(2, r), linked); err != nil {
				return err
			}
			if err := st.link(sh, ref(2, r), url); err != nil {
				return err
			}
		}
	}

	// Filter buttons, dropdowns and colours, as in the team's sheet
	if err := f.AutoFilter(sh, fmt.Sprintf("A3:O%d", last), nil); err != nil {
		return err
	}
	// After the filter, which resets it: the team's tab colour
	tab := "FF" + paletteOf(m.User.Team()).header
	if err := f.SetSheetProps(sh, &excelize.SheetPropsOptions{TabColorRGB: &tab}); err != nil {
		return err
	}
	if err := dropdown(f, sh, fmt.Sprintf("D%d:D%d", first, last), sheetStatuses); err != nil {
		return err
	}
	for _, rng := range []string{"G%d:L%d", "M%d:M%d"} {
		dv := excelize.NewDataValidation(true)
		dv.SetSqref(fmt.Sprintf(rng, first, last))
		if err := dv.SetRange(0, 1e9, excelize.DataValidationTypeDecimal, excelize.DataValidationOperatorGreaterThanOrEqual); err != nil {
			return err
		}
		if err := f.AddDataValidation(sh, dv); err != nil {
			return err
		}
	}
	statusFills := []struct{ status, fill string }{
		{"Done", "C6EFCE"}, {"In Progress", "DDEBF7"}, {"Review Back", "FCE4D6"}, {"For Review", "FFF2CC"}, {"On Hold", "EDEDED"},
	}
	var rules []excelize.ConditionalFormatOptions
	for _, s := range statusFills {
		format, err := f.NewConditionalStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{s.fill}}})
		if err != nil {
			return err
		}
		rules = append(rules, excelize.ConditionalFormatOptions{Type: "cell", Criteria: "==", Value: `"` + s.status + `"`, Format: &format})
	}
	if err := f.SetConditionalFormat(sh, fmt.Sprintf("D%d:D%d", first, last), rules); err != nil {
		return err
	}
	red, err := f.NewConditionalStyle(&excelize.Style{
		Font: &excelize.Font{Color: "9C0006"},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F8CBAD"}},
	})
	if err != nil {
		return err
	}
	return f.SetConditionalFormat(sh, fmt.Sprintf("M%d:M%d", first, last), []excelize.ConditionalFormatOptions{
		{Type: "cell", Criteria: ">=", Value: "2", Format: &red},
	})
}
