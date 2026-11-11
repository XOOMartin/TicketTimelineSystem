package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"crud_app/internal/application"
)

type MovieHandler struct {
	service *application.MovieService
}

func NewMovieHandler(service *application.MovieService) *MovieHandler {
	return &MovieHandler{service: service}
}

func (h *MovieHandler) Register(r *mux.Router) {
	r.HandleFunc("/movies", h.list).Methods(http.MethodGet)
	r.HandleFunc("/movies/{id}", h.get).Methods(http.MethodGet)
	r.HandleFunc("/movies", h.create).Methods(http.MethodPost)
	r.HandleFunc("/movies/{id}", h.update).Methods(http.MethodPut)
	r.HandleFunc("/movies/{id}", h.delete).Methods(http.MethodDelete)
}

func (h *MovieHandler) list(w http.ResponseWriter, r *http.Request) {
	movies, err := h.service.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMovieResponses(movies))
}

func (h *MovieHandler) get(w http.ResponseWriter, r *http.Request) {
	m, err := h.service.Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMovieResponse(m))
}

func (h *MovieHandler) create(w http.ResponseWriter, r *http.Request) {
	var req movieRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	m, err := h.service.Create(r.Context(), req.toCommand())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMovieResponse(m))
}

func (h *MovieHandler) update(w http.ResponseWriter, r *http.Request) {
	var req movieRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	m, err := h.service.Update(r.Context(), mux.Vars(r)["id"], req.toCommand())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toMovieResponse(m))
}

// delete responds with the remaining movies, as the original API did.
func (h *MovieHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), mux.Vars(r)["id"]); err != nil {
		writeError(w, err)
		return
	}
	h.list(w, r)
}
