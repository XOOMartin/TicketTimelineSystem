package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"crud_app/internal/application"
)

type ActivityHandler struct {
	activities *application.ActivityService
	reviews    *application.ReviewService
}

func NewActivityHandler(activities *application.ActivityService, reviews *application.ReviewService) *ActivityHandler {
	return &ActivityHandler{activities: activities, reviews: reviews}
}

func (h *ActivityHandler) Register(r *mux.Router) {
	r.HandleFunc("/activity/import", h.importPrompts).Methods(http.MethodPost)
	r.HandleFunc("/activity/days", h.listDays).Methods(http.MethodGet)
	r.HandleFunc("/activity/days/{day}/prompts", h.listPrompts).Methods(http.MethodGet)
	r.HandleFunc("/activity/days/{day}/review", h.getReview).Methods(http.MethodGet)
	r.HandleFunc("/activity/days/{day}/review", h.generateReview).Methods(http.MethodPost)
}

func (h *ActivityHandler) importPrompts(w http.ResponseWriter, r *http.Request) {
	res, err := h.activities.Import(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toImportResponse(res))
}

func (h *ActivityHandler) listDays(w http.ResponseWriter, r *http.Request) {
	days, err := h.activities.Days(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDaySummaryResponses(days))
}

func (h *ActivityHandler) listPrompts(w http.ResponseWriter, r *http.Request) {
	prompts, err := h.activities.Prompts(r.Context(), mux.Vars(r)["day"])
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toPromptResponses(prompts))
}

func (h *ActivityHandler) getReview(w http.ResponseWriter, r *http.Request) {
	rv, err := h.reviews.Get(r.Context(), mux.Vars(r)["day"])
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toReviewResponse(rv))
}

// generateReview runs the AI analysis, which can take a minute or more.
func (h *ActivityHandler) generateReview(w http.ResponseWriter, r *http.Request) {
	rv, err := h.reviews.Generate(r.Context(), mux.Vars(r)["day"])
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toReviewResponse(rv))
}
