package web

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/syahnur197/my-gym-app/backend/internal/auth"
	"github.com/syahnur197/my-gym-app/backend/internal/service"
)

type Handler struct {
	svc    *service.Service
	auth   *auth.Manager
	secure bool
}

func New(s *service.Service, a *auth.Manager, secure bool) *Handler { return &Handler{s, a, secure} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("GET /", h.home)
	mux.HandleFunc("GET /login", h.loginPage)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /logout", h.protect(h.logout))
	mux.HandleFunc("GET /workouts", h.protect(h.workouts))
	mux.HandleFunc("POST /workouts/{id}/copy", h.protect(h.copyWorkout))
	mux.HandleFunc("GET /workouts/new", h.protect(h.editor))
	mux.HandleFunc("GET /workouts/{id}/edit", h.protect(h.editor))
	mux.HandleFunc("POST /workouts", h.protect(h.saveWorkout))
	mux.HandleFunc("POST /workouts/{id}", h.protect(h.saveWorkout))
	mux.HandleFunc("POST /workouts/{id}/delete", h.protect(h.deleteWorkout))
	mux.HandleFunc("GET /workouts/{id}/log", h.protect(h.logEditor))
	mux.HandleFunc("POST /workouts/{id}/log", h.protect(h.logWorkout))
	mux.HandleFunc("GET /exercises", h.protect(h.exercises))
	mux.HandleFunc("POST /exercises", h.protect(h.saveExercise))
	mux.HandleFunc("POST /exercises/{id}/delete", h.protect(h.deleteExercise))
	mux.HandleFunc("GET /exercises/history", h.protect(h.exerciseHistory))
	mux.HandleFunc("GET /calendar", h.protect(h.calendar))
	mux.HandleFunc("GET /sessions/{id}", h.protect(h.session))
	mux.HandleFunc("GET /manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		fmt.Fprint(w, manifest)
	})
	mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Service-Worker-Allowed", "/")
		fmt.Fprint(w, serviceWorker)
	})
	mux.HandleFunc("GET /icon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write(icon)
	})
}
func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/workouts", http.StatusSeeOther)
}
func (h *Handler) authorized(r *http.Request) bool {
	c, e := r.Cookie("gym_session")
	return e == nil && h.auth.Verify(c.Value)
}
func (h *Handler) protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.authorized(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodPost {
			cookie, e := r.Cookie("gym_csrf")
			if e != nil || cookie.Value == "" || r.FormValue("_csrf") != cookie.Value {
				http.Error(w, "invalid CSRF token", http.StatusForbidden)
				return
			}
		}
		next(w, r)
	}
}
func (h *Handler) csrf(w http.ResponseWriter, r *http.Request) string {
	if c, e := r.Cookie("gym_csrf"); e == nil && len(c.Value) > 20 {
		return c.Value
	}
	// One response can render several forms before the browser has received the
	// cookie. Reuse a token already queued on this response so every form and
	// the final Set-Cookie value agree.
	for _, header := range w.Header().Values("Set-Cookie") {
		first, _, _ := strings.Cut(header, ";")
		if value, ok := strings.CutPrefix(first, "gym_csrf="); ok && len(value) > 20 {
			return value
		}
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	v := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{Name: "gym_csrf", Value: v, Path: "/", Secure: h.secure, SameSite: http.SameSiteStrictMode})
	return v
}
