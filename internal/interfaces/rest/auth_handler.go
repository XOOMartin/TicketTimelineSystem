package rest

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"crud_app/internal/application"
	"crud_app/internal/domain/user"
)

const sessionCookie = "session"

type userKey struct{}

// currentUser is set by RequireUser, so handlers behind it can rely on it.
func currentUser(r *http.Request) *user.User {
	u, _ := r.Context().Value(userKey{}).(*user.User)
	return u
}

type AuthHandler struct {
	auth *application.AuthService
	// secureCookie should be true whenever the site is served over HTTPS.
	secureCookie bool
}

func NewAuthHandler(auth *application.AuthService, secureCookie bool) *AuthHandler {
	return &AuthHandler{auth: auth, secureCookie: secureCookie}
}

func (h *AuthHandler) Register(r *mux.Router) {
	r.HandleFunc("/auth/signup", h.signup).Methods(http.MethodPost)
	r.HandleFunc("/auth/login", h.login).Methods(http.MethodPost)
	r.HandleFunc("/auth/logout", h.logout).Methods(http.MethodPost)
	r.Handle("/auth/me", h.RequireUser(http.HandlerFunc(h.me))).Methods(http.MethodGet)
}

// RequireUser rejects requests without a valid session cookie.
func (h *AuthHandler) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		if c, err := r.Cookie(sessionCookie); err == nil {
			token = c.Value
		}
		u, err := h.auth.Authenticate(r.Context(), token)
		if err != nil {
			writeError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	})
}

// RequireAdmin must run after RequireUser.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u := currentUser(r); u == nil || !u.IsAdmin() {
			writeError(w, user.ErrForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *AuthHandler) signup(w http.ResponseWriter, r *http.Request) {
	var req signupRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	u, token, err := h.auth.Signup(r.Context(), application.SignupCommand{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Password:  req.Password,
		Team:      req.Team,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	h.setSession(w, token, int(application.SessionDuration.Seconds()))
	writeJSON(w, http.StatusCreated, toUserResponse(u))
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	u, token, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	h.setSession(w, token, int(application.SessionDuration.Seconds()))
	writeJSON(w, http.StatusOK, toUserResponse(u))
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := h.auth.Logout(r.Context(), c.Value); err != nil {
			writeError(w, err)
			return
		}
	}
	h.setSession(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, toUserResponse(currentUser(r)))
}

// setSession sets the cookie for maxAge seconds, or clears it when maxAge is
// negative.
func (h *AuthHandler) setSession(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.secureCookie,
		// Lax keeps the cookie off cross-site POSTs, which blocks CSRF.
		SameSite: http.SameSiteLaxMode,
	})
}
