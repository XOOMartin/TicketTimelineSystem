package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"crud_app/internal/application"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

// TimerHandler serves the timer a member runs while working on a phase.
type TimerHandler struct {
	timers *application.TimerService
}

func NewTimerHandler(timers *application.TimerService) *TimerHandler {
	return &TimerHandler{timers: timers}
}

// Register expects r to already require a logged-in user.
func (h *TimerHandler) Register(r *mux.Router) {
	r.HandleFunc("/timer", h.current).Methods(http.MethodGet)
	r.HandleFunc("/timer", h.start).Methods(http.MethodPost)
	r.HandleFunc("/timer/stop", h.stop).Methods(http.MethodPost)
	r.HandleFunc("/timer/pause", h.pause).Methods(http.MethodPost)
	r.HandleFunc("/timer/resume", h.resume).Methods(http.MethodPost)
	// The site's pages say when they're closed (a beacon on the way out) and
	// open again; a timer ends when its site stays closed
	r.HandleFunc("/timer/closing", h.closing).Methods(http.MethodPost)
	r.HandleFunc("/timer/open", h.open).Methods(http.MethodPost)
}

type timerRequest struct {
	Project string `json:"project"`
	Repo    string `json:"repo"`
	Ticket  string `json:"ticket"`
	Title   string `json:"title"`
	Round   int    `json:"round"`
	Phase   string `json:"phase"`
	Step    string `json:"step"` // a typed ticket's workflow step; then phase isn't needed
}

type timerResponse struct {
	Project   string    `json:"project"`
	Repo      string    `json:"repo"`
	Ticket    string    `json:"ticket"`
	TicketKey string    `json:"ticketKey"`
	Title     string    `json:"title"`
	Round     int       `json:"round"`
	Phase     string    `json:"phase"`
	StartedAt time.Time `json:"startedAt"`
	// The clock shows (pausedAt, or now) − startedAt − pausedSeconds
	PausedAt      *time.Time `json:"pausedAt"`
	PausedSeconds int        `json:"pausedSeconds"`
	// When the site was closed; the time recorded stops there
	ClosingAt *time.Time `json:"closingAt"`
	// The step of a typed ticket's workflow being timed; "" for none
	Step string `json:"step"`
}

type stoppedTimerResponse struct {
	Timer    timerResponse     `json:"timer"`
	Minutes  int               `json:"minutes"` // logged; 0 when under a minute
	CheckIns []checkInResponse `json:"checkIns"`
}

// currentTimerResponse is the running timer, if any, and the one a start
// ended on the way.
type currentTimerResponse struct {
	Timer   *timerResponse        `json:"timer"`
	Stopped *stoppedTimerResponse `json:"stopped,omitempty"`
}

func toTimerResponse(t *worklog.Timer) *timerResponse {
	if t == nil {
		return nil
	}
	return &timerResponse{
		Project:       t.Ticket.Project.String(),
		Repo:          string(t.Ticket.Repo),
		Ticket:        t.Ticket.Number,
		TicketKey:     t.Ticket.String(),
		Title:         t.Title,
		Round:         t.Round,
		Phase:         string(t.Phase),
		StartedAt:     t.StartedAt,
		PausedAt:      t.PausedAt,
		PausedSeconds: int(t.Paused / time.Second),
		ClosingAt:     t.ClosingAt,
		Step:          t.FlowStep,
	}
}

func toStoppedTimerResponse(s *application.StoppedTimer) *stoppedTimerResponse {
	if s == nil {
		return nil
	}
	return &stoppedTimerResponse{Timer: *toTimerResponse(&s.Timer), Minutes: s.Minutes, CheckIns: toCheckInResponses(s.CheckIns)}
}

func (h *TimerHandler) current(w http.ResponseWriter, r *http.Request) {
	t, err := h.timers.Current(r.Context(), currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, currentTimerResponse{Timer: toTimerResponse(t)})
}

func (h *TimerHandler) start(w http.ResponseWriter, r *http.Request) {
	var req timerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	t, stopped, err := h.timers.Start(r.Context(), currentUser(r), application.TimerCommand{
		Project: req.Project,
		Repo:    req.Repo,
		Ticket:  req.Ticket,
		Title:   req.Title,
		Round:   req.Round,
		Phase:   req.Phase,
		Step:    req.Step,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, currentTimerResponse{Timer: toTimerResponse(t), Stopped: toStoppedTimerResponse(stopped)})
}

func (h *TimerHandler) pause(w http.ResponseWriter, r *http.Request) {
	h.change(w, r, h.timers.Pause)
}

func (h *TimerHandler) resume(w http.ResponseWriter, r *http.Request) {
	h.change(w, r, h.timers.Resume)
}

func (h *TimerHandler) change(w http.ResponseWriter, r *http.Request, action func(context.Context, *user.User) (*worklog.Timer, error)) {
	t, err := action(r.Context(), currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, currentTimerResponse{Timer: toTimerResponse(t)})
}

func (h *TimerHandler) closing(w http.ResponseWriter, r *http.Request) {
	if err := h.timers.Closing(r.Context(), currentUser(r)); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// openRequest says whether the page was opened fresh (a new tab or window),
// not reloaded nor already open; no body means it wasn't
type openRequest struct {
	Fresh bool `json:"fresh"`
}

// open answers with the running timer, and the one that ended while the site
// was closed, if any
func (h *TimerHandler) open(w http.ResponseWriter, r *http.Request) {
	var req openRequest
	if r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	t, stopped, err := h.timers.Open(r.Context(), currentUser(r), req.Fresh)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, currentTimerResponse{Timer: toTimerResponse(t), Stopped: toStoppedTimerResponse(stopped)})
}

func (h *TimerHandler) stop(w http.ResponseWriter, r *http.Request) {
	stopped, err := h.timers.Stop(r.Context(), currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toStoppedTimerResponse(stopped))
}
