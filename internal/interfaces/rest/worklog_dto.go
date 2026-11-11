package rest

import (
	"time"

	"crud_app/internal/application"
	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/report"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

type signupRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Team      string `json:"team"`
}

// userRequest is an account as an admin adds or edits it; an empty password
// keeps the current one when editing
type userRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Team      string `json:"team"`
	Role      string `json:"role"`
}

// adminUserResponse is an account in the admin's user management
type adminUserResponse struct {
	userResponse
	CreatedAt time.Time  `json:"createdAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

type usersResponse struct {
	Active  []adminUserResponse `json:"active"`
	Deleted []adminUserResponse `json:"deleted"`
}

func toAdminUserResponse(u *user.User) adminUserResponse {
	return adminUserResponse{userResponse: toUserResponse(u), CreatedAt: u.CreatedAt(), DeletedAt: u.DeletedAt()}
}

func toAdminUserResponses(users []*user.User) []adminUserResponse {
	res := make([]adminUserResponse, len(users))
	for i, u := range users {
		res[i] = toAdminUserResponse(u)
	}
	return res
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Team      string `json:"team"`
	Role      string `json:"role"`
}

type option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// optionsResponse lists the choices for dropdowns, so the frontend never
// hard-codes them.
type optionsResponse struct {
	Projects []string `json:"projects"`
	// Each project's repos, e.g. SPinno's spec, backend and web; none for
	// BlackBox. hasParent marks those whose tickets sit under a spec ticket.
	Repos    map[string][]repoOption `json:"repos"`
	Phases   []option                `json:"phases"`
	Statuses []option                `json:"statuses"`
	MaxHours int                     `json:"maxHours"`
	// The types each SPinno repo's tickets can have (the first is the
	// default), and each type's steps; BlackBox's phases as steps under
	// blackBoxWorkflowKey
	TicketTypes map[string][]option         `json:"ticketTypes"`
	Workflows   map[string]workflowResponse `json:"workflows"`
}

// blackBoxWorkflowKey is where BlackBox's steps sit among the workflows; no
// ticket type has that name.
const blackBoxWorkflowKey = "blackbox"

type workflowResponse struct {
	First []flowStepResponse `json:"first"` // the original work
	Again []flowStepResponse `json:"again"` // a review-back round after Cho-san's changes; empty when there are none
	Redo  []flowStepResponse `json:"redo"`  // a review-back round after the leads' changes: the original steps again
}

type flowStepResponse struct {
	Key             string `json:"key"`
	Label           string `json:"label"`
	Who             string `json:"who"`
	Kind            string `json:"kind"`  // work (timed), review (typed time + result) or action (one click)
	Phase           string `json:"phase"` // the column its time goes in
	Hint            string `json:"hint"`
	ChangesTo       string `json:"changesTo"`
	ChangesNewRound bool   `json:"changesNewRound"`
	ChangesRestart  bool   `json:"changesRestart"`
	ApproveLabel    string `json:"approveLabel"`
	WaitLabel       string `json:"waitLabel,omitempty"` // a result that keeps the step open; none for most
}

func toFlowStepResponses(steps []worklog.FlowStep) []flowStepResponse {
	res := []flowStepResponse{}
	for _, s := range steps {
		res = append(res, flowStepResponse{
			Key: s.Key, Label: s.Label, Who: s.Who, Kind: string(s.Kind), Phase: string(s.Phase), Hint: s.Hint,
			ChangesTo: s.ChangesTo, ChangesNewRound: s.ChangesNewRound,
			ChangesRestart: s.ChangesRestart, ApproveLabel: s.ApproveLabel, WaitLabel: s.WaitLabel,
		})
	}
	return res
}

type repoOption struct {
	Value     string `json:"value"`
	Label     string `json:"label"`
	HasParent bool   `json:"hasParent"`
}

type checkInRequest struct {
	Project  string    `json:"project"`
	Repo     string    `json:"repo"`
	Ticket   string    `json:"ticket"`
	Title    string    `json:"title"`
	Round    int       `json:"round"`
	Phase    string    `json:"phase"`
	Status   string    `json:"status"`
	Hours    float64   `json:"hours"`
	Note     string    `json:"note"`
	WorkedAt time.Time `json:"workedAt"`
}

// ticketWorkRequest is a check-in covering several phases; phases without
// hours are left out or sent as 0.
type ticketWorkRequest struct {
	Project    string             `json:"project"`
	Repo       string             `json:"repo"`
	Ticket     string             `json:"ticket"`
	Title      string             `json:"title"`
	Round      int                `json:"round"`
	Status     string             `json:"status"`
	StartDate  string             `json:"startDate"` // YYYY-MM-DD; "" for none
	EndDate    string             `json:"endDate"`
	Note       string             `json:"note"`
	WorkedAt   time.Time          `json:"workedAt"`
	PhaseHours map[string]float64 `json:"phaseHours"`
}

// addTicketRequest is a ticket added to the timeline with no time yet; the
// project and number are in the path.
type addTicketRequest struct {
	Title  string `json:"title"`
	Parent string `json:"parent"` // a backend or web ticket's spec ticket number
	Type   string `json:"type"`   // a SPinno ticket's type; "" for its repo's default
}

// typeRequest changes a SPinno ticket's type; the ticket is in the path.
type typeRequest struct {
	Type string `json:"type"`
}

// parentRequest moves a backend or web ticket under a spec ticket ("" for
// none); the ticket is in the path.
type parentRequest struct {
	Parent string `json:"parent"`
}

type stepRequest struct {
	Round int    `json:"round"` // 0 for the latest
	Step  string `json:"step"`  // a phase
}

type outcomeRequest struct {
	Step    string  `json:"step"`    // a typed ticket's workflow step; then phase isn't needed
	Phase   string  `json:"phase"`   // confirmation or review, or a timed phase with its time typed in
	Outcome string  `json:"outcome"` // no_changes, changes_requested, waiting, approved, or done (typed time)
	Round   int     `json:"round"`
	Hours   float64 `json:"hours"`
	Comment string  `json:"comment"`
	Title   string  `json:"title"` // for a ticket with no time yet
}

type checkInResponse struct {
	ID        string    `json:"id"`
	Project   string    `json:"project"`
	Repo      string    `json:"repo"`
	Ticket    string    `json:"ticket"`
	TicketKey string    `json:"ticketKey"`
	Title     string    `json:"title"`
	Round     int       `json:"round"`
	Phase     string    `json:"phase"`
	Status    string    `json:"status"`
	Hours     float64   `json:"hours"`
	Note      string    `json:"note"`
	WorkedAt  time.Time `json:"workedAt"`
	Source    string    `json:"source"`
}

type periodResponse struct {
	First string `json:"first"`
	Last  string `json:"last"`
}

type ticketRowResponse struct {
	Project     string             `json:"project"`
	Repo        string             `json:"repo"` // SPinno: spec, backend or web; "" for none
	Ticket      string             `json:"ticket"`
	TicketKey   string             `json:"ticketKey"`
	Title       string             `json:"title"`
	Assignees   []string           `json:"assignees"`
	Phase       string             `json:"phase"`
	Status      string             `json:"status"`
	PeriodHours float64            `json:"periodHours"`
	TotalHours  float64            `json:"totalHours"`
	PhaseHours  map[string]float64 `json:"phaseHours"`
	ReviewBacks int                `json:"reviewBacks"`
	Rounds      []roundRowResponse `json:"rounds"`
	Step        string             `json:"step"` // phase worked on next; "" when finished
	StepRound   int                `json:"stepRound"`
	// A SPinno ticket's type, and the step of its workflow it's on ("" once
	// done); both "" for BlackBox
	Type       string    `json:"type"`
	FlowStep   string    `json:"flowStep"`
	MadeParent bool      `json:"madeParent"` // made a parent by dragging: can be dragged back to a child
	LastUpdate time.Time `json:"lastUpdate"`
	Note       string    `json:"note"`
	// YYYY-MM-DD; EndDate is empty while the ticket is open, and StartDate
	// in reports saved before it was tracked.
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	// When the ticket went into the user's timeline, for sorting the newest
	// first; left out in reports
	AddedAt *time.Time `json:"addedAt,omitempty"`
	// The spec ticket (its number) a backend or web ticket sits under
	Parent string `json:"parent"`
	// A spec ticket's time on the tickets under it, on top of totalHours,
	// that time by phase, and how many tickets sit under it
	ChildHours       float64            `json:"childHours"`
	ChildPhaseHours  map[string]float64 `json:"childPhaseHours"`
	ChildCount       int                `json:"childCount"`
	ChildReviewBacks int                `json:"childReviewBacks"` // the review-backs of the tickets under a parent
	// When work on it began (its first time logged); left out when none, and
	// in reports
	StartedAt *time.Time `json:"startedAt,omitempty"`
}

// roundRowResponse is one pass through the phases of a ticket.
type roundRowResponse struct {
	Round      int                `json:"round"` // 1 = original work
	PhaseHours map[string]float64 `json:"phaseHours"`
	Hours      float64            `json:"hours"`
	FirstDay   string             `json:"firstDay"` // YYYY-MM-DD
	LastDay    string             `json:"lastDay"`
	StartedAt  *time.Time         `json:"startedAt,omitempty"` // its first time logged
}

type projectRowResponse struct {
	Project    string             `json:"project"`
	Tickets    int                `json:"tickets"`
	ByStatus   map[string]int     `json:"byStatus"`
	PhaseHours map[string]float64 `json:"phaseHours"`
	Hours      float64            `json:"hours"`
}

type personRowResponse struct {
	UserID        string   `json:"userId"`
	Name          string   `json:"name"`
	Team          string   `json:"team"`
	CheckIns      int      `json:"checkIns"`
	Hours         float64  `json:"hours"`
	ExpectedHours float64  `json:"expectedHours"`
	MissedDays    []string `json:"missedDays"`
}

type myWeekResponse struct {
	Period   periodResponse      `json:"period"`
	CheckIns []checkInResponse   `json:"checkIns"`
	Tickets  []ticketRowResponse `json:"tickets"`
	Person   *personRowResponse  `json:"person"`
}

// memberDashboardResponse is a member's dashboard as an admin sees it.
type memberDashboardResponse struct {
	User userResponse `json:"user"`
	myWeekResponse
}

type memberWeekResponse struct {
	User       userResponse          `json:"user"`
	Activity   personRowResponse     `json:"activity"`
	LastImport *importRecordResponse `json:"lastImport"`
	Submission string                `json:"submission"`
}

type membersResponse struct {
	Period  periodResponse       `json:"period"`
	Members []memberWeekResponse `json:"members"`
}

type assignRequest struct {
	Team string `json:"team"`
	Role string `json:"role"`
}

type importRecordResponse struct {
	Filename   string    `json:"filename"`
	Added      int       `json:"added"`
	Skipped    int       `json:"skipped"`
	ImportedAt time.Time `json:"importedAt"`
}

type importRowResponse struct {
	Line      int              `json:"line"`
	CheckIn   *checkInResponse `json:"checkIn"`
	Error     string           `json:"error"`
	Duplicate bool             `json:"duplicate"`
}

type sheetImportResponse struct {
	Period  periodResponse      `json:"period"`
	Rows    []importRowResponse `json:"rows"`
	Valid   int                 `json:"valid"`
	Skipped int                 `json:"skipped"`
	Added   int                 `json:"added"`
}

type generateReportRequest struct {
	Kind string `json:"kind"`
	Day  string `json:"day"`
}

type reportSummaryResponse struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	Period      periodResponse `json:"period"`
	GeneratedBy string         `json:"generatedBy"`
	GeneratedAt time.Time      `json:"generatedAt"`
	TicketCount int            `json:"ticketCount"`
}

type reportResponse struct {
	reportSummaryResponse
	Tickets  []ticketRowResponse  `json:"tickets"`
	Projects []projectRowResponse `json:"projects"`
	People   []personRowResponse  `json:"people"`
}

func toUserResponse(u *user.User) userResponse {
	return userResponse{ID: string(u.ID()), Name: u.Name(), Username: u.Username(), FirstName: u.FirstName(), LastName: u.LastName(), Team: u.Team().String(), Role: string(u.Role())}
}

func toOptionsResponse() optionsResponse {
	res := optionsResponse{
		MaxHours: worklog.MaxHours, Repos: map[string][]repoOption{},
		TicketTypes: map[string][]option{}, Workflows: map[string]workflowResponse{},
	}
	for _, p := range project.All() {
		res.Projects = append(res.Projects, p.String())
		repos := []repoOption{}
		for _, r := range p.Repos() {
			repos = append(repos, repoOption{Value: string(r), Label: r.Label(), HasParent: r.HasParent()})
			types := []option{}
			for _, t := range worklog.TypesFor(r) {
				types = append(types, option{Value: string(t), Label: t.Label()})
			}
			if len(types) > 0 {
				res.TicketTypes[string(r)] = types
			}
		}
		res.Repos[p.String()] = repos
	}
	for _, p := range worklog.Phases() {
		res.Phases = append(res.Phases, option{Value: string(p), Label: p.Label()})
	}
	for _, s := range worklog.Statuses() {
		res.Statuses = append(res.Statuses, option{Value: string(s), Label: s.Label()})
	}
	for _, t := range worklog.Types() {
		res.Workflows[string(t)] = toWorkflowResponse(worklog.WorkflowOf(t))
	}
	res.Workflows[blackBoxWorkflowKey] = toWorkflowResponse(worklog.BlackBoxWorkflow())
	return res
}

func toWorkflowResponse(w worklog.Workflow) workflowResponse {
	return workflowResponse{
		First: toFlowStepResponses(w.First), Again: toFlowStepResponses(w.Again), Redo: toFlowStepResponses(w.Redo),
	}
}

func (req checkInRequest) toCommand() application.CheckInCommand {
	return application.CheckInCommand{
		Project:  req.Project,
		Repo:     req.Repo,
		Ticket:   req.Ticket,
		Title:    req.Title,
		Round:    req.Round,
		Phase:    req.Phase,
		Status:   req.Status,
		Hours:    req.Hours,
		Note:     req.Note,
		WorkedAt: req.WorkedAt,
	}
}

func (req ticketWorkRequest) toCommand() application.TicketWorkCommand {
	return application.TicketWorkCommand{
		Project:    req.Project,
		Repo:       req.Repo,
		Ticket:     req.Ticket,
		Title:      req.Title,
		Round:      req.Round,
		Status:     req.Status,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		Note:       req.Note,
		WorkedAt:   req.WorkedAt,
		PhaseHours: req.PhaseHours,
	}
}

func toCheckInResponse(c *worklog.CheckIn) checkInResponse {
	return checkInResponse{
		ID:        string(c.ID()),
		Project:   c.Ticket().Project.String(),
		Repo:      string(c.Ticket().Repo),
		Ticket:    c.Ticket().Number,
		TicketKey: c.Ticket().String(),
		Title:     c.Title(),
		Round:     c.Round(),
		Phase:     string(c.Phase()),
		Status:    string(c.Status()),
		Hours:     c.Hours(),
		Note:      c.Note(),
		WorkedAt:  c.WorkedAt(),
		Source:    string(c.Source()),
	}
}

func toCheckInResponses(checkIns []*worklog.CheckIn) []checkInResponse {
	res := make([]checkInResponse, len(checkIns))
	for i, c := range checkIns {
		res[i] = toCheckInResponse(c)
	}
	return res
}

func toPeriodResponse(p calendar.Period) periodResponse {
	return periodResponse{First: p.First().String(), Last: p.Last().String()}
}

// timeOrNil leaves a zero time out of a response.
func timeOrNil(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func phaseHours(m map[worklog.Phase]float64) map[string]float64 {
	res := make(map[string]float64, len(m))
	for k, v := range m {
		res[string(k)] = v
	}
	return res
}

func toTicketRowResponses(rows []report.TicketRow) []ticketRowResponse {
	res := make([]ticketRowResponse, len(rows))
	for i, t := range rows {
		var end string
		if t.EndDate != nil {
			end = t.EndDate.String()
		}
		var addedAt *time.Time
		if !t.AddedAt.IsZero() {
			at := t.AddedAt
			addedAt = &at
		}
		res[i] = ticketRowResponse{
			Project:     t.Ticket.Project.String(),
			Repo:        string(t.Ticket.Repo),
			Ticket:      t.Ticket.Number,
			TicketKey:   t.Ticket.String(),
			Title:       t.Title,
			Assignees:   t.Assignees,
			Phase:       string(t.Phase),
			Status:      string(t.Status),
			PeriodHours: t.PeriodHours,
			TotalHours:  t.TotalHours,
			PhaseHours:  phaseHours(t.PhaseHours),
			ReviewBacks: t.ReviewBacks,
			Rounds:      toRoundRowResponses(t.Rounds),
			Step:        string(t.Step),
			StepRound:   t.StepRound,
			Type:        string(t.Type),
			FlowStep:    t.FlowStep,
			MadeParent:  t.MadeParent,
			LastUpdate:  t.LastUpdate,
			Note:        t.Note,
			StartDate:   t.StartDate.String(),
			EndDate:     end,
			AddedAt:     addedAt,
			Parent:      t.Parent,
			ChildHours:  t.ChildHours,
			// Only the phases the tickets under it have time on
			ChildPhaseHours:  phaseHours(t.ChildPhaseHours),
			ChildCount:       t.ChildCount,
			ChildReviewBacks: t.ChildReviewBacks,
			StartedAt:        timeOrNil(t.StartedAt),
		}
	}
	return res
}

func toRoundRowResponses(rounds []report.RoundRow) []roundRowResponse {
	res := make([]roundRowResponse, len(rounds))
	for i, r := range rounds {
		res[i] = roundRowResponse{
			Round:      r.Round,
			PhaseHours: phaseHours(r.PhaseHours),
			Hours:      r.Hours,
			FirstDay:   r.FirstDay.String(),
			LastDay:    r.LastDay.String(),
			StartedAt:  timeOrNil(r.StartedAt),
		}
	}
	return res
}

func toProjectRowResponses(rows []report.ProjectRow) []projectRowResponse {
	res := make([]projectRowResponse, len(rows))
	for i, p := range rows {
		byStatus := make(map[string]int, len(p.ByStatus))
		for k, v := range p.ByStatus {
			byStatus[string(k)] = v
		}
		res[i] = projectRowResponse{
			Project:    p.Project.String(),
			Tickets:    p.Tickets,
			ByStatus:   byStatus,
			PhaseHours: phaseHours(p.PhaseHours),
			Hours:      p.Hours,
		}
	}
	return res
}

func toPersonRowResponse(p report.PersonRow) personRowResponse {
	missed := make([]string, len(p.MissedDays))
	for i, d := range p.MissedDays {
		missed[i] = d.String()
	}
	return personRowResponse{
		UserID:        string(p.UserID),
		Name:          p.Name,
		Team:          p.Team.String(),
		CheckIns:      p.CheckIns,
		Hours:         p.Hours,
		ExpectedHours: p.ExpectedHours,
		MissedDays:    missed,
	}
}

func toMyWeekResponse(w *application.MyWeek) myWeekResponse {
	res := myWeekResponse{
		Period:   toPeriodResponse(w.Period),
		CheckIns: toCheckInResponses(w.CheckIns),
		Tickets:  toTicketRowResponses(w.Tickets),
	}
	if w.Person != nil {
		p := toPersonRowResponse(*w.Person)
		res.Person = &p
	}
	return res
}

func toMemberDashboardResponse(w *application.MyWeek) memberDashboardResponse {
	return memberDashboardResponse{User: toUserResponse(w.User), myWeekResponse: toMyWeekResponse(w)}
}

func toMembersResponse(period calendar.Period, members []application.MemberWeek) membersResponse {
	res := membersResponse{Period: toPeriodResponse(period), Members: make([]memberWeekResponse, len(members))}
	for i, m := range members {
		mr := memberWeekResponse{
			User:       toUserResponse(m.User),
			Activity:   toPersonRowResponse(m.Activity),
			Submission: string(m.Submission),
		}
		if imp := m.LastImport; imp != nil {
			mr.LastImport = &importRecordResponse{Filename: imp.Filename, Added: imp.Added, Skipped: imp.Skipped, ImportedAt: imp.ImportedAt}
		}
		res.Members[i] = mr
	}
	return res
}

func toSheetImportResponse(r *application.SheetImport) sheetImportResponse {
	res := sheetImportResponse{Period: toPeriodResponse(r.Period), Added: r.Added, Rows: make([]importRowResponse, len(r.Rows))}
	for i, row := range r.Rows {
		rr := importRowResponse{Line: row.Line, Error: row.Error, Duplicate: row.Duplicate}
		if row.CheckIn != nil {
			c := toCheckInResponse(row.CheckIn)
			rr.CheckIn = &c
		}
		if row.Error == "" && !row.Duplicate {
			res.Valid++
		} else {
			res.Skipped++
		}
		res.Rows[i] = rr
	}
	return res
}

func toReportSummaryResponse(r *report.Report) reportSummaryResponse {
	return reportSummaryResponse{
		ID:          string(r.ID()),
		Kind:        string(r.Kind()),
		Period:      toPeriodResponse(r.Period()),
		GeneratedBy: r.GeneratedBy(),
		GeneratedAt: r.GeneratedAt(),
		TicketCount: len(r.Data().Tickets),
	}
}

func toReportResponse(r *report.Report) reportResponse {
	d := r.Data()
	people := make([]personRowResponse, len(d.People))
	for i, p := range d.People {
		people[i] = toPersonRowResponse(p)
	}
	return reportResponse{
		reportSummaryResponse: toReportSummaryResponse(r),
		Tickets:               toTicketRowResponses(d.Tickets),
		Projects:              toProjectRowResponses(d.Projects),
		People:                people,
	}
}
