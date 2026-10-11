package web

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/syahnur197/my-gym-app/backend/internal/auth"
	"github.com/syahnur197/my-gym-app/backend/internal/model"
	"github.com/syahnur197/my-gym-app/backend/internal/service"
)

type Handler struct {
	svc    *service.Service
	auth   *auth.Manager
	secure bool
}

func New(s *service.Service, a *auth.Manager, secure bool) *Handler { return &Handler{s, a, secure} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.home)
	mux.HandleFunc("GET /login", h.loginPage)
	mux.HandleFunc("POST /login", h.login)
	mux.HandleFunc("POST /logout", h.protect(h.logout))
	mux.HandleFunc("GET /workouts", h.protect(h.workouts))
	mux.HandleFunc("GET /workouts/new", h.protect(h.editor))
	mux.HandleFunc("GET /workouts/{id}/edit", h.protect(h.editor))
	mux.HandleFunc("POST /workouts", h.protect(h.saveWorkout))
	mux.HandleFunc("POST /workouts/{id}", h.protect(h.saveWorkout))
	mux.HandleFunc("POST /workouts/{id}/delete", h.protect(h.deleteWorkout))
	mux.HandleFunc("POST /workouts/{id}/log", h.protect(h.logWorkout))
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
		fmt.Fprint(w, icon)
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
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	v := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{Name: "gym_csrf", Value: v, Path: "/", Secure: h.secure, SameSite: http.SameSiteStrictMode})
	return v
}
func (h *Handler) loginPage(w http.ResponseWriter, r *http.Request) {
	if h.authorized(r) {
		http.Redirect(w, r, "/workouts", 303)
		return
	}
	h.render(w, r, "Sign in", template.HTML(`<section class="max-w-md mx-auto bg-white p-6 rounded-2xl shadow"><h1 class="text-2xl font-bold mb-4">Sign in</h1><form method="post" action="/login" class="space-y-4"><label class="block">Username<input name="username" required autocomplete="username" class="field"></label><label class="block">Password<input name="password" type="password" required autocomplete="current-password" class="field"></label><button class="btn w-full">Sign in</button></form></section>`))
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", 400)
		return
	}
	token, err := h.auth.Login(r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		http.Error(w, "invalid credentials", 401)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "gym_session", Value: token, Path: "/", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
	http.Redirect(w, r, "/workouts", 303)
}
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "gym_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/login", 303)
}
func (h *Handler) render(w http.ResponseWriter, r *http.Request, title string, body template.HTML) {
	csrf := h.csrf(w, r)
	data := struct {
		Title    string
		Body     template.HTML
		CSRF     string
		LoggedIn bool
	}{title, body, csrf, h.authorized(r)}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.Execute(w, data); err != nil {
		fmt.Println(err)
	}
}
func esc(s string) string { return template.HTMLEscapeString(s) }
func (h *Handler) workouts(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.Workouts.List(r.Context())
	if err != nil {
		http.Error(w, "database error", 500)
		return
	}
	var b strings.Builder
	b.WriteString(`<div class="flex items-center justify-between mb-5"><h1 class="text-2xl font-bold">Workouts</h1><a class="btn" href="/workouts/new">+ New workout</a></div><div class="grid gap-4 md:grid-cols-2">`)
	csrf := h.csrf(w, r)
	for _, p := range plans {
		fmt.Fprintf(&b, `<article class="card"><div class="flex justify-between items-start"><h2 class="text-xl font-semibold">%s</h2><a class="link" href="/workouts/%d/edit">Edit</a></div><ul class="mt-3 space-y-1 text-slate-600">`, esc(p.Name), p.ID)
		for _, e := range p.Exercises {
			fmt.Fprintf(&b, "<li>%s — %d × %d · %g kg</li>", esc(e.Name), e.Sets, e.Reps, e.WeightKg)
		}
		fmt.Fprintf(&b, `</ul><div class="flex gap-2 mt-5 flex-wrap"><form method="post" action="/workouts/%d/log"><input type="hidden" name="_csrf" value="%s"><button class="btn">Log today</button></form><form method="post" action="/workouts/%d/delete" onsubmit="return confirm('Delete this workout template? History will be preserved.')"><input type="hidden" name="_csrf" value="%s"><button class="btn-secondary">Delete</button></form></div></article>`, p.ID, csrf, p.ID, csrf)
	}
	b.WriteString("</div>")
	h.render(w, r, "Workouts", template.HTML(b.String()))
}
func parseID(r *http.Request) (uint, error) {
	n, e := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if e != nil || n == 0 {
		return 0, fmt.Errorf("invalid ID")
	}
	return uint(n), nil
}
func (h *Handler) editor(w http.ResponseWriter, r *http.Request) {
	p := model.Workout{Name: "New workout", Exercises: []model.WorkoutExercise{}}
	if r.PathValue("id") != "" {
		id, e := parseID(r)
		if e != nil {
			http.Error(w, "invalid ID", 400)
			return
		}
		p, e = h.svc.Workouts.Find(r.Context(), id)
		if e != nil {
			http.NotFound(w, r)
			return
		}
	}
	raw, _ := json.Marshal(p.Exercises)
	var b strings.Builder
	action := "/workouts"
	if p.ID != 0 {
		action = fmt.Sprintf("/workouts/%d", p.ID)
	}
	fmt.Fprintf(&b, `<h1 class="text-2xl font-bold mb-5">%s</h1><form method="post" action="%s" class="card space-y-4" x-data="editor(JSON.parse(atob('%s')))"><input type="hidden" name="_csrf" value="%s"><label class="block">Workout name<input name="name" required maxlength="100" value="%s" class="field"></label><template x-for="(e,i) in exercises" :key="e.key"><div class="rounded-xl border p-3 space-y-2"><div class="flex justify-between"><strong x-text="'Exercise '+(i+1)"></strong><button type="button" class="link" @click="exercises.splice(i,1)">Remove</button></div><label>Exercise name<input required maxlength="150" class="field" :name="'exercise_name_'+i" x-model="e.name"></label><div class="grid grid-cols-3 gap-2"><label>Sets<input required type="number" min="1" max="100" class="field" :name="'sets_'+i" x-model.number="e.sets"></label><label>Reps<input required type="number" min="0" max="10000" class="field" :name="'reps_'+i" x-model.number="e.reps"></label><label>Weight (kg)<input required type="number" min="0" max="10000" step="0.01" class="field" :name="'weight_'+i" x-model.number="e.weightKg"></label></div></div></template><input type="hidden" name="count" :value="exercises.length"><button type="button" class="btn-secondary" @click="add()">+ Add exercise</button><div class="flex gap-3"><button class="btn">Save workout</button><a href="/workouts" class="btn-secondary">Cancel</a></div></form>`, map[bool]string{true: "Edit workout", false: "New workout"}[p.ID != 0], action, base64.StdEncoding.EncodeToString(raw), h.csrf(w, r), esc(p.Name))
	h.render(w, r, "Edit workout", template.HTML(b.String()))
}
func (h *Handler) saveWorkout(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", 400)
		return
	}
	n, e := strconv.Atoi(r.FormValue("count"))
	if e != nil || n < 0 || n > 100 {
		http.Error(w, "invalid exercise count", 400)
		return
	}
	p := model.Workout{Name: r.FormValue("name"), Exercises: make([]model.WorkoutExercise, 0, n)}
	if r.PathValue("id") != "" {
		p.ID, e = parseID(r)
		if e != nil {
			http.Error(w, "invalid ID", 400)
			return
		}
	}
	for i := 0; i < n; i++ {
		sets, e1 := strconv.Atoi(r.FormValue(fmt.Sprintf("sets_%d", i)))
		reps, e2 := strconv.Atoi(r.FormValue(fmt.Sprintf("reps_%d", i)))
		weight, e3 := strconv.ParseFloat(r.FormValue(fmt.Sprintf("weight_%d", i)), 64)
		if e1 != nil || e2 != nil || e3 != nil {
			http.Error(w, "invalid exercise values", 400)
			return
		}
		p.Exercises = append(p.Exercises, model.WorkoutExercise{Name: r.FormValue(fmt.Sprintf("exercise_name_%d", i)), Sets: sets, Reps: reps, WeightKg: weight})
	}
	if err := h.svc.SaveWorkout(r.Context(), &p); err != nil {
		http.Error(w, "invalid workout", 400)
		return
	}
	http.Redirect(w, r, "/workouts", 303)
}
func (h *Handler) deleteWorkout(w http.ResponseWriter, r *http.Request) {
	id, e := parseID(r)
	if e != nil {
		http.Error(w, "invalid ID", 400)
		return
	}
	if e = h.svc.Workouts.Delete(r.Context(), id); e != nil {
		http.Error(w, "delete failed", 500)
		return
	}
	http.Redirect(w, r, "/workouts", 303)
}
func (h *Handler) logWorkout(w http.ResponseWriter, r *http.Request) {
	id, e := parseID(r)
	if e != nil {
		http.Error(w, "invalid ID", 400)
		return
	}
	_, e = h.svc.Log(r.Context(), id, time.Now().Format("2006-01-02"), nil)
	if e != nil {
		http.Error(w, "log failed", 500)
		return
	}
	http.Redirect(w, r, "/calendar", 303)
}
func (h *Handler) calendar(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	start, e := time.Parse("2006-01", month)
	if e != nil || start.Format("2006-01") != month {
		http.Error(w, "invalid month", 400)
		return
	}
	logs, e := h.svc.Sessions.List(r.Context(), month)
	if e != nil {
		http.Error(w, "database error", 500)
		return
	}
	byDate := map[string][]model.Session{}
	for _, s := range logs {
		byDate[s.Date] = append(byDate[s.Date], s)
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="flex items-center justify-between mb-5"><a class="btn-secondary" href="/calendar?month=%s">←</a><h1 class="text-xl font-bold">%s</h1><a class="btn-secondary" href="/calendar?month=%s">→</a></div><div class="grid grid-cols-7 gap-1 text-center">`, start.AddDate(0, -1, 0).Format("2006-01"), start.Format("January 2006"), start.AddDate(0, 1, 0).Format("2006-01"))
	for _, day := range []string{"M", "T", "W", "T", "F", "S", "S"} {
		fmt.Fprintf(&b, `<div class="text-slate-500 font-semibold py-2">%s</div>`, day)
	}
	offset := (int(start.Weekday()) + 6) % 7
	for i := 0; i < offset; i++ {
		b.WriteString("<div></div>")
	}
	for d := start; d.Month() == start.Month(); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		n := len(byDate[key])
		style := "bg-white"
		if n > 0 {
			style = "bg-indigo-100 border-indigo-300"
		}
		fmt.Fprintf(&b, `<a href="#day-%s" class="border rounded-lg p-2 min-h-14 %s"><span>%d</span>`, key, style, d.Day())
		if n > 0 {
			fmt.Fprintf(&b, `<span class="block text-xs text-indigo-700">%d session(s)</span>`, n)
		}
		b.WriteString("</a>")
	}
	b.WriteString("</div><section class='mt-6 space-y-4'><h2 class='text-xl font-bold'>Logged sessions</h2>")
	for _, s := range logs {
		fmt.Fprintf(&b, `<a id="day-%s" href="/sessions/%d" class="card block hover:border-indigo-400"><strong>%s</strong> · %s <span class="link">View →</span></a>`, s.Date, s.ID, esc(s.WorkoutName), s.Date)
	}
	if len(logs) == 0 {
		b.WriteString("<p class='text-slate-500'>No sessions this month.</p>")
	}
	b.WriteString("</section>")
	h.render(w, r, "Calendar", template.HTML(b.String()))
}
func (h *Handler) session(w http.ResponseWriter, r *http.Request) {
	id, e := parseID(r)
	if e != nil {
		http.Error(w, "invalid ID", 400)
		return
	}
	s, e := h.svc.Sessions.Find(r.Context(), id)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<a href="/calendar" class="link">← Calendar</a><article class="card mt-4"><h1 class="text-2xl font-bold">%s</h1><p class="text-slate-500">%s · Saved snapshot</p><ul class="mt-4 space-y-3">`, esc(s.WorkoutName), esc(s.Date))
	for _, x := range s.Exercises {
		fmt.Fprintf(&b, "<li>%s — %d × %d @ %g kg</li>", esc(x.Name), x.Sets, x.Reps, x.WeightKg)
	}
	b.WriteString("</ul></article>")
	h.render(w, r, "Session", template.HTML(b.String()))
}

var page = template.Must(template.New("page").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="theme-color" content="#4338ca"><link rel="manifest" href="/manifest.webmanifest"><link rel="icon" href="/icon.svg"><title>{{.Title}} · My Gym</title><script src="https://cdn.tailwindcss.com"></script><script defer src="https://unpkg.com/alpinejs@3.14.9/dist/cdn.min.js"></script><script src="https://unpkg.com/htmx.org@2.0.4"></script><style type="text/tailwindcss">@layer components{.btn{@apply inline-flex items-center justify-center rounded-xl bg-indigo-700 text-white px-4 py-2 font-medium hover:bg-indigo-800}.btn-secondary{@apply inline-flex items-center justify-center rounded-xl bg-slate-100 text-slate-800 px-4 py-2 font-medium hover:bg-slate-200}.card{@apply bg-white rounded-2xl border border-slate-200 p-5 shadow-sm}.field{@apply mt-1 block w-full border border-slate-300 rounded-lg px-3 py-2}.link{@apply text-indigo-700 font-medium}}</style><script>function editor(initial){return {exercises:initial.map((e,i)=>({...e,key:i})),nextKey:initial.length,add(){this.exercises.push({key:this.nextKey++,name:'',sets:3,reps:10,weightKg:0})}}}if('serviceWorker'in navigator){window.addEventListener('load',()=>navigator.serviceWorker.register('/sw.js'))}</script></head><body class="bg-slate-50 min-h-screen text-slate-900"><header class="bg-white border-b"><nav class="max-w-4xl mx-auto p-4 flex items-center justify-between gap-3"><a href="/workouts" class="font-bold text-xl text-indigo-700">My Gym</a>{{if .LoggedIn}}<div class="flex items-center gap-3"><a href="/workouts" class="link" hx-boost="true">Workouts</a><a href="/calendar" class="link" hx-boost="true">Calendar</a><form method="post" action="/logout"><input type="hidden" name="_csrf" value="{{.CSRF}}"><button class="text-sm text-slate-500">Sign out</button></form></div>{{end}}</nav></header><main class="max-w-4xl mx-auto p-4 md:p-8">{{.Body}}</main></body></html>`))

const manifest = `{"name":"My Gym","short_name":"My Gym","start_url":"/workouts","scope":"/","display":"standalone","background_color":"#f8fafc","theme_color":"#4338ca","icons":[{"src":"/icon.svg","sizes":"any","type":"image/svg+xml","purpose":"any maskable"}]}`
const serviceWorker = `const CACHE='gym-shell-v1';const ASSETS=['/icon.svg','/manifest.webmanifest'];self.addEventListener('install',e=>e.waitUntil(caches.open(CACHE).then(c=>c.addAll(ASSETS))));self.addEventListener('activate',e=>e.waitUntil(caches.keys().then(keys=>Promise.all(keys.filter(k=>k!==CACHE).map(k=>caches.delete(k))))));self.addEventListener('fetch',e=>{const u=new URL(e.request.url);if(e.request.method!=='GET')return;if(u.origin!==self.location.origin)return;if(ASSETS.includes(u.pathname))e.respondWith(caches.match(e.request).then(r=>r||fetch(e.request)));});`
const icon = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 192 192"><rect width="192" height="192" rx="38" fill="#4338ca"/><path d="M28 78h24v36H28zm112 0h24v36h-24zM56 67h18v58H56zm62 0h18v58h-18zM72 89h48v14H72z" fill="white"/></svg>`
