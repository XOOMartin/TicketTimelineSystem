package rest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/project"
	"crud_app/internal/domain/report"
	"crud_app/internal/domain/user"
	"crud_app/internal/domain/worklog"
)

type errorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func writeBadRequest(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusBadRequest, errorBody{Error: message})
}

// writeError maps domain errors to HTTP status codes. The domain never knows
// about HTTP; this is the only place that translates.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, user.ErrNotFound), errors.Is(err, worklog.ErrNotFound), errors.Is(err, report.ErrNotFound),
		errors.Is(err, worklog.ErrNoTimer):
		status = http.StatusNotFound
	case errors.Is(err, user.ErrInvalid), errors.Is(err, worklog.ErrInvalid), errors.Is(err, report.ErrInvalid),
		errors.Is(err, calendar.ErrInvalid), errors.Is(err, project.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, user.ErrUsernameTaken), errors.Is(err, worklog.ErrDuplicate), errors.Is(err, worklog.ErrTicketExists):
		status = http.StatusConflict
	case errors.Is(err, user.ErrBadCredentials), errors.Is(err, user.ErrUnauthenticated):
		status = http.StatusUnauthorized
	case errors.Is(err, user.ErrForbidden), errors.Is(err, user.ErrDeactivated):
		status = http.StatusForbidden
	}
	if status == http.StatusInternalServerError {
		log.Printf("internal error: %v", err)
		writeJSON(w, status, errorBody{Error: "internal server error"})
		return
	}
	writeJSON(w, status, errorBody{Error: err.Error()})
}

// writeXLSX sends a workbook as a download. The workbook is built into a
// buffer first, so a failure can still be reported as JSON.
func writeXLSX(w http.ResponseWriter, filename string, build func(*bytes.Buffer) error) {
	var buf bytes.Buffer
	if err := build(&buf); err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	if _, err := buf.WriteTo(w); err != nil {
		log.Printf("write xlsx: %v", err)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeBadRequest(w, "invalid JSON body")
		return false
	}
	return true
}

// dayParam reads ?week= or ?day=, defaulting to today.
func dayParam(r *http.Request, name string) string {
	if v := r.URL.Query().Get(name); v != "" {
		return v
	}
	return calendar.DayOf(time.Now()).String()
}
