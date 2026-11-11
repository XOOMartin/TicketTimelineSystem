package rest

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"crud_app/internal/application"
	"crud_app/internal/domain/worklog"
)

// TeamHandler serves who on the team is busy right now, for every logged-in
// user's dashboard.
type TeamHandler struct {
	team *application.TeamService
}

func NewTeamHandler(team *application.TeamService) *TeamHandler {
	return &TeamHandler{team: team}
}

// Register expects r to already require a logged-in user.
func (h *TeamHandler) Register(r *mux.Router) {
	r.HandleFunc("/team/activity", h.activity).Methods(http.MethodGet)
}

// workingOnResponse is a ticket and phase: what someone is timing now, or
// last logged time on
type workingOnResponse struct {
	TicketKey  string `json:"ticketKey"`
	Title      string `json:"title"`
	Round      int    `json:"round"`
	Phase      string `json:"phase"`
	PhaseLabel string `json:"phaseLabel"`
}

type memberActivityResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Team string `json:"team"`
	// busy (a ticket ongoing) or available (none)
	State string `json:"state"`
	// What they're on: the ticket they're timing, else their latest ongoing
	// ticket (logged at CurrentAt); null when available
	Current   *workingOnResponse `json:"current"`
	CurrentAt *time.Time         `json:"currentAt"`
	// Their timer: "running", "paused", or "" for none; with how long so
	// far, pauses left out
	Timer          string `json:"timer"`
	ElapsedSeconds int    `json:"elapsedSeconds"`
	// The last time they logged, and on what; null before their first
	Last       *workingOnResponse `json:"last"`
	LastAt     *time.Time         `json:"lastAt"`
	TodayHours float64            `json:"todayHours"`
	// Tickets being worked on, and on hold
	OngoingTickets int `json:"ongoingTickets"`
	WaitingTickets int `json:"waitingTickets"`
	BlockedTickets int `json:"blockedTickets"`
}

type teamActivityResponse struct {
	AsOf    time.Time                `json:"asOf"`
	Members []memberActivityResponse `json:"members"`
}

func workingOn(key worklog.TicketKey, title string, round int, phase worklog.Phase) *workingOnResponse {
	return &workingOnResponse{
		TicketKey: key.String(), Title: title, Round: round, Phase: string(phase), PhaseLabel: phase.ShortLabel(),
	}
}

func (h *TeamHandler) activity(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	team, err := h.team.Activity(r.Context(), now)
	if err != nil {
		writeError(w, err)
		return
	}
	res := teamActivityResponse{AsOf: now, Members: make([]memberActivityResponse, len(team))}
	for i, m := range team {
		row := memberActivityResponse{
			ID: string(m.User.ID()), Name: m.User.Name(), Team: m.User.Team().String(), State: string(m.State),
			ElapsedSeconds: int(m.Elapsed / time.Second), TodayHours: m.TodayHours,
			OngoingTickets: m.OngoingTickets, WaitingTickets: m.WaitingTickets, BlockedTickets: m.BlockedTickets,
		}
		switch {
		case m.Timer != nil:
			t := m.Timer
			row.Current = workingOn(t.Ticket, t.Title, t.Round, t.Phase)
			row.Timer = "running"
			if t.IsPaused() {
				row.Timer = "paused"
			}
		case m.LatestOngoing != nil:
			c := m.LatestOngoing
			row.Current = workingOn(c.Ticket(), c.Title(), c.Round(), c.Phase())
			at := c.WorkedAt()
			row.CurrentAt = &at
		}
		if c := m.LastCheckIn; c != nil {
			row.Last = workingOn(c.Ticket(), c.Title(), c.Round(), c.Phase())
			at := c.WorkedAt()
			row.LastAt = &at
		}
		res.Members[i] = row
	}
	writeJSON(w, http.StatusOK, res)
}
