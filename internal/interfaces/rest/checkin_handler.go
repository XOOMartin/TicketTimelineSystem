package rest

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"crud_app/internal/application"
	"crud_app/internal/domain/calendar"
	"crud_app/internal/interfaces/excel"
)

// CheckInHandler serves a member's own check-ins.
type CheckInHandler struct {
	checkIns *application.CheckInService
}

func NewCheckInHandler(checkIns *application.CheckInService) *CheckInHandler {
	return &CheckInHandler{checkIns: checkIns}
}

// Register expects r to already require a logged-in user.
func (h *CheckInHandler) Register(r *mux.Router) {
	r.HandleFunc("/checkins", h.week).Methods(http.MethodGet)
	r.HandleFunc("/checkins", h.create).Methods(http.MethodPost)
	r.HandleFunc("/checkins/batch", h.createMany).Methods(http.MethodPost)
	r.HandleFunc("/timeline/{project}/{ticket}", h.addTicket).Methods(http.MethodPost)
	r.HandleFunc("/timeline/{project}/{ticket}", h.setTotals).Methods(http.MethodPut)
	r.HandleFunc("/timeline/{project}/{ticket}", h.deleteTicket).Methods(http.MethodDelete)
	r.HandleFunc("/timeline/{project}/{ticket}/step", h.setStep).Methods(http.MethodPut)
	r.HandleFunc("/timeline/{project}/{ticket}/outcome", h.recordOutcome).Methods(http.MethodPost)
	r.HandleFunc("/timeline/{project}/{ticket}/parent", h.setParent).Methods(http.MethodPut)
	r.HandleFunc("/timeline/{project}/{ticket}/make-parent", h.makeParent).Methods(http.MethodPost)
	r.HandleFunc("/timeline/{project}/{ticket}/make-child", h.makeChild).Methods(http.MethodPost)
	r.HandleFunc("/timeline/{project}/{ticket}/type", h.setType).Methods(http.MethodPut)
	r.HandleFunc("/timeline/{project}/{ticket}/rounds/{round}", h.deleteRound).Methods(http.MethodDelete)
	r.HandleFunc("/checkins/export", h.export).Methods(http.MethodGet)
	r.HandleFunc("/checkins/{id}", h.update).Methods(http.MethodPut)
	r.HandleFunc("/checkins/{id}", h.delete).Methods(http.MethodDelete)
}

// RegisterPublic adds routes that work without logging in.
func RegisterPublic(r *mux.Router) {
	r.HandleFunc("/options", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, toOptionsResponse())
	}).Methods(http.MethodGet)
}

func (h *CheckInHandler) week(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	week, err := h.checkIns.Week(r.Context(), u, dayParam(r, "week"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMyWeekResponse(week))
}

func (h *CheckInHandler) create(w http.ResponseWriter, r *http.Request) {
	var req checkInRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := h.checkIns.Create(r.Context(), currentUser(r), req.toCommand())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCheckInResponse(c))
}

// createMany logs several phases of one ticket at once.
func (h *CheckInHandler) createMany(w http.ResponseWriter, r *http.Request) {
	var req ticketWorkRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	checkIns, err := h.checkIns.CreateMany(r.Context(), currentUser(r), req.toCommand())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCheckInResponses(checkIns))
}

// setTotals corrects a row of the timeline: the user's total time on each
// phase of the ticket.
func (h *CheckInHandler) setTotals(w http.ResponseWriter, r *http.Request) {
	var req ticketWorkRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	cmd := req.toCommand()
	cmd.Project = mux.Vars(r)["project"]
	cmd.Repo = repoParam(r)
	cmd.Ticket = mux.Vars(r)["ticket"]
	checkIns, err := h.checkIns.SetTotals(r.Context(), currentUser(r), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCheckInResponses(checkIns))
}

// addTicket puts a ticket in the timeline before any time is logged on it.
func (h *CheckInHandler) addTicket(w http.ResponseWriter, r *http.Request) {
	var req addTicketRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	vars := mux.Vars(r)
	_, err := h.checkIns.AddTicket(r.Context(), currentUser(r), application.AddTicketCommand{
		Project: vars["project"],
		Repo:    repoParam(r),
		Ticket:  vars["ticket"],
		Title:   req.Title,
		Parent:  req.Parent,
		Type:    req.Type,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// deleteTicket removes a row of the timeline.
func (h *CheckInHandler) deleteTicket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.checkIns.DeleteTicket(r.Context(), currentUser(r), vars["project"], repoParam(r), vars["ticket"]); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// setParent moves a SPinno backend or web ticket under a spec ticket.
func (h *CheckInHandler) setParent(w http.ResponseWriter, r *http.Request) {
	var req parentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	vars := mux.Vars(r)
	if err := h.checkIns.SetParent(r.Context(), currentUser(r), vars["project"], repoParam(r), vars["ticket"], req.Parent); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// makeParent makes a child spec ticket a parent (dragged out of any parent)
func (h *CheckInHandler) makeParent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.checkIns.MakeParent(r.Context(), currentUser(r), vars["project"], repoParam(r), vars["ticket"]); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// makeChild makes a parent made by dragging a child again (dragged out)
func (h *CheckInHandler) makeChild(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if err := h.checkIns.MakeChild(r.Context(), currentUser(r), vars["project"], repoParam(r), vars["ticket"]); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// setType changes a SPinno ticket's type, and so its steps.
func (h *CheckInHandler) setType(w http.ResponseWriter, r *http.Request) {
	var req typeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	vars := mux.Vars(r)
	if err := h.checkIns.SetType(r.Context(), currentUser(r), vars["project"], repoParam(r), vars["ticket"], req.Type); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteRound removes one round of a ticket: its time, or a review-back
// just started.
func (h *CheckInHandler) deleteRound(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	round, err := strconv.Atoi(vars["round"])
	if err != nil {
		writeBadRequest(w, "round must be a number")
		return
	}
	if err := h.checkIns.DeleteRound(r.Context(), currentUser(r), vars["project"], repoParam(r), vars["ticket"], round); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// repoParam is the ticket's repo for the /timeline/{project}/{ticket} routes:
// ?repo=spec, backend or web for a SPinno ticket, none otherwise.
func repoParam(r *http.Request) string { return r.URL.Query().Get("repo") }

// setStep moves a ticket to a step of its progress, e.g. skipping one.
func (h *CheckInHandler) setStep(w http.ResponseWriter, r *http.Request) {
	var req stepRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	vars := mux.Vars(r)
	if err := h.checkIns.SetStep(r.Context(), currentUser(r), vars["project"], repoParam(r), vars["ticket"], req.Round, req.Step); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// recordOutcome finishes a confirmation or review step.
func (h *CheckInHandler) recordOutcome(w http.ResponseWriter, r *http.Request) {
	var req outcomeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	vars := mux.Vars(r)
	err := h.checkIns.RecordOutcome(r.Context(), currentUser(r), vars["project"], repoParam(r), vars["ticket"], application.OutcomeCommand{
		Step:    req.Step,
		Phase:   req.Phase,
		Outcome: req.Outcome,
		Round:   req.Round,
		Hours:   req.Hours,
		Comment: req.Comment,
		Title:   req.Title,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CheckInHandler) update(w http.ResponseWriter, r *http.Request) {
	var req checkInRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := h.checkIns.Update(r.Context(), currentUser(r), mux.Vars(r)["id"], req.toCommand())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCheckInResponse(c))
}

func (h *CheckInHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.checkIns.Delete(r.Context(), currentUser(r), mux.Vars(r)["id"]); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CheckInHandler) export(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	week, err := h.checkIns.Week(r.Context(), u, dayParam(r, "week"))
	if err != nil {
		writeError(w, err)
		return
	}
	name := exportName(week.Period.First(), fileSafe(u.Name())+" Check-ins")
	writeXLSX(w, name, func(buf *bytes.Buffer) error { return excel.WriteWeek(buf, u, week) })
}

// fileSafe turns a name into something usable in a filename (spaces are
// fine; slashes and quotes aren't).
func fileSafe(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '"' {
			return '_'
		}
		return r
	}, s)
}

// exportName is a download's file name: its date first, the way people say
// it, so the newest is easy to find, e.g. "Nov 9 2026 Weekly Report.xlsx".
func exportName(day calendar.Day, what string) string {
	return day.Start().Format("Jan 2 2006") + " " + what + ".xlsx"
}
