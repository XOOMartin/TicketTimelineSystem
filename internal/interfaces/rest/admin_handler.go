package rest

import (
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"

	"crud_app/internal/application"
	"crud_app/internal/domain/calendar"
	"crud_app/internal/domain/report"
	"crud_app/internal/interfaces/excel"
)

const maxUploadBytes = 5 << 20

// AdminHandler serves member management, spreadsheet imports and reports.
type AdminHandler struct {
	members *application.MemberService
	reports *application.ReportService
	users   *application.UserAdminService
}

func NewAdminHandler(members *application.MemberService, reports *application.ReportService, users *application.UserAdminService) *AdminHandler {
	return &AdminHandler{members: members, reports: reports, users: users}
}

// Register expects r to already require an admin.
func (h *AdminHandler) Register(r *mux.Router) {
	r.HandleFunc("/admin/members", h.listMembers).Methods(http.MethodGet)
	r.HandleFunc("/admin/members/{id}", h.assignMember).Methods(http.MethodPut)
	r.HandleFunc("/admin/members/{id}/week", h.memberWeek).Methods(http.MethodGet)
	r.HandleFunc("/admin/members/{id}/import/preview", h.previewImport).Methods(http.MethodPost)
	r.HandleFunc("/admin/members/{id}/import", h.importSheet).Methods(http.MethodPost)
	r.HandleFunc("/admin/overview", h.overview).Methods(http.MethodGet)
	r.HandleFunc("/admin/reports", h.listReports).Methods(http.MethodGet)
	r.HandleFunc("/admin/reports", h.generateReport).Methods(http.MethodPost)
	r.HandleFunc("/admin/reports/{id}", h.getReport).Methods(http.MethodGet)
	r.HandleFunc("/admin/reports/{id}/xlsx", h.downloadReport).Methods(http.MethodGet)
	r.HandleFunc("/admin/team-sheet.xlsx", h.downloadTeamSheet).Methods(http.MethodGet)
	// User management: deleting is soft (restorable); /permanent is for good
	r.HandleFunc("/admin/users", h.listUsers).Methods(http.MethodGet)
	r.HandleFunc("/admin/users", h.createUser).Methods(http.MethodPost)
	r.HandleFunc("/admin/users/{id}", h.updateUser).Methods(http.MethodPut)
	r.HandleFunc("/admin/users/{id}", h.deleteUser).Methods(http.MethodDelete)
	r.HandleFunc("/admin/users/{id}/restore", h.restoreUser).Methods(http.MethodPost)
	r.HandleFunc("/admin/users/{id}/permanent", h.hardDeleteUser).Methods(http.MethodDelete)
}

func (h *AdminHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	period, members, err := h.members.Week(r.Context(), dayParam(r, "week"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMembersResponse(period, members))
}

// memberWeek is one member's dashboard, as they would see it.
func (h *AdminHandler) memberWeek(w http.ResponseWriter, r *http.Request) {
	week, err := h.members.Dashboard(r.Context(), mux.Vars(r)["id"], dayParam(r, "week"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMemberDashboardResponse(week))
}

func (h *AdminHandler) assignMember(w http.ResponseWriter, r *http.Request) {
	var req assignRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	u, err := h.users.Assign(r.Context(), currentUser(r), mux.Vars(r)["id"], req.Team, req.Role)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}

func (h *AdminHandler) previewImport(w http.ResponseWriter, r *http.Request) {
	rows, _, ok := readUpload(w, r)
	if !ok {
		return
	}
	result, err := h.members.PreviewImport(r.Context(), mux.Vars(r)["id"], dayParam(r, "week"), rows)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toSheetImportResponse(result))
}

func (h *AdminHandler) importSheet(w http.ResponseWriter, r *http.Request) {
	rows, filename, ok := readUpload(w, r)
	if !ok {
		return
	}
	result, err := h.members.Import(r.Context(), currentUser(r), mux.Vars(r)["id"], dayParam(r, "week"), filename, rows)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toSheetImportResponse(result))
}

// readUpload parses the "file" field of a multipart upload as a check-in
// workbook.
func readUpload(w http.ResponseWriter, r *http.Request) ([]application.ImportRow, string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeBadRequest(w, fmt.Sprintf("attach an .xlsx file up to %d MB", maxUploadBytes>>20))
		return nil, "", false
	}
	defer file.Close()

	rows, err := excel.ReadCheckIns(file)
	if err != nil {
		writeBadRequest(w, err.Error())
		return nil, "", false
	}
	return rows, filepath.Base(header.Filename), true
}

// overview is this week's report computed on the fly, for the dashboard.
func (h *AdminHandler) overview(w http.ResponseWriter, r *http.Request) {
	rp, err := h.reports.Preview(r.Context(), dayParam(r, "week"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toReportResponse(rp))
}

func (h *AdminHandler) listReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.reports.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	res := make([]reportSummaryResponse, len(reports))
	for i, rp := range reports {
		res[i] = toReportSummaryResponse(rp)
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AdminHandler) generateReport(w http.ResponseWriter, r *http.Request) {
	var req generateReportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	rp, err := h.reports.Generate(r.Context(), req.Kind, req.Day, currentUser(r).Name())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toReportResponse(rp))
}

func (h *AdminHandler) getReport(w http.ResponseWriter, r *http.Request) {
	rp, err := h.reports.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toReportResponse(rp))
}

// downloadReport exports the report's day or week in the layout of the
// team's Ticket_Progress_Timeline spreadsheet: each user's timeline for it
func (h *AdminHandler) downloadReport(w http.ResponseWriter, r *http.Request) {
	rp, err := h.reports.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, err)
		return
	}
	members, err := h.members.TeamTimelinesFor(r.Context(), rp.Period())
	if err != nil {
		writeError(w, err)
		return
	}
	p := rp.Period()
	subtitle := fmt.Sprintf("Weekly report — %s to %s", p.First(), p.Last())
	if p.First() == p.Last() {
		subtitle = fmt.Sprintf("Daily report — %s", p.First())
	}
	kind := "Weekly Report"
	if rp.Kind() == report.KindDaily {
		kind = "Daily Report"
	}
	name := exportName(p.First(), kind)
	writeXLSX(w, name, func(buf *bytes.Buffer) error { return excel.WriteTeamSheet(buf, members, subtitle) })
}

// downloadTeamSheet exports everyone's whole timeline in the layout of the
// team's Ticket_Progress_Timeline spreadsheet
func (h *AdminHandler) downloadTeamSheet(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	members, err := h.members.TeamTimelines(r.Context(), now)
	if err != nil {
		writeError(w, err)
		return
	}
	name := exportName(calendar.DayOf(now), "Team Sheet")
	writeXLSX(w, name, func(buf *bytes.Buffer) error { return excel.WriteTeamSheet(buf, members, "") })
}

func (h *AdminHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	active, deleted, err := h.users.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, usersResponse{Active: toAdminUserResponses(active), Deleted: toAdminUserResponses(deleted)})
}

func (h *AdminHandler) createUser(w http.ResponseWriter, r *http.Request) {
	var req userRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	u, err := h.users.Create(r.Context(), userCommand(req))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAdminUserResponse(u))
}

func (h *AdminHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	var req userRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	u, err := h.users.Update(r.Context(), currentUser(r), mux.Vars(r)["id"], userCommand(req))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAdminUserResponse(u))
}

// deleteUser soft-deletes the account: restorable, its data kept
func (h *AdminHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	if err := h.users.Delete(r.Context(), currentUser(r), mux.Vars(r)["id"]); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) restoreUser(w http.ResponseWriter, r *http.Request) {
	u, err := h.users.Restore(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAdminUserResponse(u))
}

// hardDeleteUser removes a deleted account and all its data for good
func (h *AdminHandler) hardDeleteUser(w http.ResponseWriter, r *http.Request) {
	if err := h.users.HardDelete(r.Context(), currentUser(r), mux.Vars(r)["id"]); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func userCommand(req userRequest) application.UserCommand {
	return application.UserCommand{
		FirstName: req.FirstName, LastName: req.LastName, Username: req.Username,
		Password: req.Password, Team: req.Team, Role: req.Role,
	}
}
