package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes returns the router to mount at /auth.
func (a *Authenticator) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/login", a.login)
	r.Post("/logout", a.logout)
	r.Get("/me", a.me)
	return r
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (a *Authenticator) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body.")
		return
	}

	if !a.checkCredentials(req.Username, req.Password) {
		writeError(w, http.StatusUnauthorized, "Invalid username or password.")
		return
	}

	token, expires, err := a.createSession(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not start a session.")
		return
	}

	http.SetCookie(w, a.sessionCookie(token, expires))
	writeJSON(w, http.StatusOK, userPayload(a.adminUser))
}

func (a *Authenticator) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		_ = a.destroySession(r.Context(), c.Value)
	}
	http.SetCookie(w, a.clearCookie())
	w.WriteHeader(http.StatusNoContent)
}

func (a *Authenticator) me(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(cookieName)
	if err != nil || !a.validToken(r.Context(), c.Value) {
		if err == nil {
			http.SetCookie(w, a.clearCookie()) // drop a stale/expired cookie
		}
		writeError(w, http.StatusUnauthorized, "Not authenticated.")
		return
	}
	writeJSON(w, http.StatusOK, userPayload(a.adminUser))
}

// Middleware rejects requests without a live session with 401.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil || !a.validToken(r.Context(), c.Value) {
			writeError(w, http.StatusUnauthorized, "Not authenticated.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func userPayload(username string) map[string]any {
	return map[string]any{"user": map[string]string{"username": username}}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
