package web

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
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
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
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
	mux.HandleFunc("GET /workouts/{id}/log", h.protect(h.logEditor))
	mux.HandleFunc("POST /workouts/{id}/log", h.protect(h.logWorkout))
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
		fmt.Fprintf(&b, `<article><a class="card block hover:border-cyan-400 hover:shadow-md transition" href="/workouts/%d/log"><div class="flex justify-between items-start gap-3"><h2 class="text-xl font-semibold">%s</h2><span class="text-cyan-600 text-sm font-medium">Open ›</span></div><ul class="mt-3 space-y-1 text-slate-600">`, p.ID, esc(p.Name))
		for _, e := range p.Exercises {
			fmt.Fprintf(&b, "<li>%s</li>", esc(e.Name))
		}
		fmt.Fprintf(&b, `</ul></a><div class="flex gap-2 mt-2 px-2"><a class="link text-sm" href="/workouts/%d/edit">Edit template</a><form method="post" action="/workouts/%d/delete" onsubmit="return confirm('Delete this workout template? History will be preserved.')"><input type="hidden" name="_csrf" value="%s"><button class="text-sm text-slate-500">Delete</button></form></div></article>`, p.ID, p.ID, csrf)
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
	fmt.Fprintf(&b, `<h1 class="text-2xl font-bold mb-5">%s</h1><p class="text-slate-600 mb-4">Templates contain exercises only. Record sets, reps and weights when you train.</p><form method="post" action="%s" class="card space-y-4" x-data="editor(JSON.parse(atob('%s')))"><input type="hidden" name="_csrf" value="%s"><label class="block">Workout name<input name="name" required maxlength="100" value="%s" class="field"></label><template x-for="(e,i) in exercises" :key="e.key"><div class="rounded-xl border p-3 space-y-2"><div class="flex justify-between"><strong x-text="'Exercise '+(i+1)"></strong><button type="button" class="link" @click="exercises.splice(i,1)">Remove</button></div><label>Exercise name<input required maxlength="150" class="field" :name="'exercise_name_'+i" x-model="e.name"></label></div></template><input type="hidden" name="count" :value="exercises.length"><button type="button" class="btn-secondary" @click="add()">+ Add exercise</button><div class="flex gap-3"><button class="btn">Save workout</button><a href="/workouts" class="btn-secondary">Cancel</a></div></form>`, map[bool]string{true: "Edit workout", false: "New workout"}[p.ID != 0], action, base64.StdEncoding.EncodeToString(raw), h.csrf(w, r), esc(p.Name))
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
		p.Exercises = append(p.Exercises, model.WorkoutExercise{Name: r.FormValue(fmt.Sprintf("exercise_name_%d", i))})
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
func (h *Handler) logEditor(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	workout, err := h.svc.Workouts.Find(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<form method="post" action="/workouts/%d/log" x-data="workoutLogger(JSON.parse(atob('%s')))"><input type="hidden" name="_csrf" value="%s"><header class="-mx-4 -mt-4 mb-4 bg-zinc-900 text-white px-4 pt-4"><div class="flex items-center justify-between pb-3"><a href="/workouts" class="text-cyan-400 text-2xl leading-none" aria-label="Back">‹</a><h1 class="font-medium text-lg" x-text="exercises[active].name"></h1><a class="text-cyan-400 text-sm" :href="'/exercises/history?name='+encodeURIComponent(exercises[active].name)">History</a></div><div class="grid grid-cols-3 text-center text-sm font-bold"><button type="button" class="border-b-4 border-cyan-400 py-3">TRACK</button><a class="py-3 text-zinc-400" :href="'/exercises/history?name='+encodeURIComponent(exercises[active].name)">HISTORY</a><span class="py-3 text-zinc-400">GRAPH</span></div></header><input type="hidden" name="date" value="%s"><template x-for="(exercise, exerciseIndex) in exercises" :key="exercise.key"><div x-show="active===exerciseIndex"><input type="hidden" :name="'exercise_name_'+exerciseIndex" :value="exercise.name"><section class="px-1"><h2 class="border-b-2 border-cyan-400 text-sm font-bold text-slate-600 pb-1">WEIGHT (kg)</h2><div class="grid grid-cols-[3.5rem_1fr_3.5rem] items-center gap-3 py-3"><button type="button" class="stepper" @click="weight=Math.max(0,weight-2.5)">−</button><input aria-label="Weight in kilograms" type="number" min="0" max="10000" step="0.5" class="track-value" x-model.number="weight"><button type="button" class="stepper" @click="weight+=2.5">+</button></div><h2 class="border-b-2 border-cyan-400 text-sm font-bold text-slate-600 pb-1">REPS</h2><div class="grid grid-cols-[3.5rem_1fr_3.5rem] items-center gap-3 py-3"><button type="button" class="stepper" @click="reps=Math.max(0,reps-1)">−</button><input aria-label="Repetitions" type="number" min="0" max="10000" step="1" class="track-value" x-model.number="reps"><button type="button" class="stepper" @click="reps+=1">+</button></div><div class="grid grid-cols-2 gap-2"><button type="button" class="save-set" @click="save(exercise)">SAVE</button><button type="button" class="clear-set" @click="clear()">CLEAR</button></div><ol class="mt-4 divide-y"><template x-for="(set, setIndex) in exercise.sets" :key="set.key"><li><button type="button" class="w-full grid grid-cols-[2rem_2rem_1fr_1fr] text-left py-3" :class="selectedSet===setIndex?'bg-cyan-100':''" @click="select(exercise,setIndex)"><span class="text-slate-400">▤</span><b x-text="setIndex+1"></b><span><b x-text="set.weightKg"></b> <small>kgs</small></span><span><b x-text="set.reps"></b> <small>reps</small></span></button></li></template></ol><input type="hidden" :name="'set_count_'+exerciseIndex" :value="exercise.sets.length"><template x-for="(set, setIndex) in exercise.sets" :key="'fields-'+set.key"><span><input type="hidden" :name="'weight_'+exerciseIndex+'_'+setIndex" :value="set.weightKg"><input type="hidden" :name="'reps_'+exerciseIndex+'_'+setIndex" :value="set.reps"></span></template></section></div></template><input type="hidden" name="exercise_count" :value="exercises.length"><nav class="fixed bottom-0 inset-x-0 bg-white border-t shadow-[0_-2px_8px_rgba(0,0,0,.12)] px-3 py-2 flex items-center justify-between gap-2"><button type="button" class="nav-arrow" @click="previous()" :disabled="active===0">‹</button><div class="text-center text-sm"><b x-text="active+1"></b> / <span x-text="exercises.length"></span><br><span class="text-slate-500">exercise</span></div><button type="button" class="nav-arrow" @click="next()" :disabled="active===exercises.length-1">›</button><button class="finish-workout">Finish</button></nav></form>`, workout.ID, base64.StdEncoding.EncodeToString(mustJSON(workout.Exercises)), h.csrf(w, r), time.Now().Format("2006-01-02"))
	h.render(w, r, "Log workout", template.HTML(b.String()))
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func (h *Handler) logWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	count, err := strconv.Atoi(r.FormValue("exercise_count"))
	if err != nil || count < 1 || count > 100 {
		http.Error(w, "invalid exercise count", http.StatusBadRequest)
		return
	}
	exercises := make([]model.SessionExercise, 0, count)
	for exerciseIndex := 0; exerciseIndex < count; exerciseIndex++ {
		setCount, err := strconv.Atoi(r.FormValue(fmt.Sprintf("set_count_%d", exerciseIndex)))
		if err != nil || setCount < 0 || setCount > 100 {
			http.Error(w, "invalid set count", http.StatusBadRequest)
			return
		}
		name := r.FormValue(fmt.Sprintf("exercise_name_%d", exerciseIndex))
		sets := make([]model.SessionSet, 0, setCount)
		for setIndex := 0; setIndex < setCount; setIndex++ {
			weight, weightErr := strconv.ParseFloat(r.FormValue(fmt.Sprintf("weight_%d_%d", exerciseIndex, setIndex)), 64)
			reps, repsErr := strconv.Atoi(r.FormValue(fmt.Sprintf("reps_%d_%d", exerciseIndex, setIndex)))
			if weightErr != nil || repsErr != nil {
				http.Error(w, "invalid set", http.StatusBadRequest)
				return
			}
			sets = append(sets, model.SessionSet{WeightKg: weight, Reps: reps})
		}
		exercises = append(exercises, model.SessionExercise{Name: name, Sets: sets})
	}
	if _, err := h.svc.Log(r.Context(), id, r.FormValue("date"), exercises); err != nil {
		http.Error(w, "could not save workout", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/calendar?date="+r.FormValue("date"), http.StatusSeeOther)
}
func (h *Handler) calendar(w http.ResponseWriter, r *http.Request) {
	selectedDate := r.URL.Query().Get("date")
	if selectedDate != "" {
		parsed, err := time.Parse("2006-01-02", selectedDate)
		if err != nil || parsed.Format("2006-01-02") != selectedDate {
			http.Error(w, "invalid date", http.StatusBadRequest)
			return
		}
	}

	currentMonth := time.Now().Truncate(24 * time.Hour)
	if selectedDate != "" {
		currentMonth, _ = time.Parse("2006-01-02", selectedDate)
	}
	firstMonth := time.Date(currentMonth.Year(), currentMonth.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, -2, 0)
	byDate := map[string][]model.Session{}
	for month := firstMonth; month.Before(firstMonth.AddDate(0, 6, 0)); month = month.AddDate(0, 1, 0) {
		sessions, err := h.svc.Sessions.List(r.Context(), month.Format("2006-01"))
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		for _, session := range sessions {
			byDate[session.Date] = append(byDate[session.Date], session)
		}
	}

	var b strings.Builder
	b.WriteString(`<div class="mb-5"><h1 class="text-2xl font-bold">Calendar</h1><p class="text-slate-500 mt-1">Tap a marked day to review that workout.</p></div><div class="space-y-9">`)
	for month := firstMonth; month.Before(firstMonth.AddDate(0, 6, 0)); month = month.AddDate(0, 1, 0) {
		writeCalendarMonth(&b, month, byDate, selectedDate)
	}
	b.WriteString(`</div>`)
	if selectedDate != "" {
		selectedSessions := byDate[selectedDate]
		fmt.Fprintf(&b, `<div class="fixed inset-0 z-50 bg-slate-950/50 flex items-end sm:items-center justify-center p-3"><section role="dialog" aria-modal="true" aria-labelledby="day-summary-title" class="bg-white w-full max-w-md rounded-2xl shadow-xl max-h-[85vh] overflow-y-auto"><div class="sticky top-0 bg-white p-5 border-b flex justify-between items-center"><h2 id="day-summary-title" class="text-lg font-bold">%s</h2><a class="link" href="/calendar" aria-label="Close">Close</a></div><div class="p-5 space-y-5">`, esc(formatLongDate(selectedDate)))
		for _, session := range selectedSessions {
			fmt.Fprintf(&b, `<a href="/sessions/%d" class="block hover:bg-cyan-50"><section><div class="flex justify-between gap-3"><h3 class="font-semibold">%s</h3><span class="link text-sm">Open ›</span></div>`, session.ID, esc(session.WorkoutName))
			for _, exercise := range session.Exercises {
				fmt.Fprintf(&b, `<div class="mt-3 border-t-2 border-cyan-400 pt-2"><strong class="text-sm uppercase">%s</strong><ol class="mt-1 space-y-1">`, esc(exercise.Name))
				for _, set := range exercise.Sets {
					fmt.Fprintf(&b, `<li class="grid grid-cols-2 text-slate-700"><span><b>%g</b> kg</span><span><b>%d</b> reps</span></li>`, set.WeightKg, set.Reps)
				}
				b.WriteString(`</ol></div>`)
			}
			b.WriteString(`</section></a>`)
		}
		if len(selectedSessions) == 0 {
			b.WriteString(`<p class="text-slate-500">No session was logged on this date.</p>`)
		}
		b.WriteString(`</div></section></div>`)
	}
	h.render(w, r, "Calendar", template.HTML(b.String()))
}

func writeCalendarMonth(b *strings.Builder, month time.Time, byDate map[string][]model.Session, selectedDate string) {
	fmt.Fprintf(b, `<section id="month-%s"><h2 class="text-center font-bold uppercase tracking-wide mb-3">%s</h2><div class="grid grid-cols-7 text-center text-[11px] font-semibold text-slate-500 mb-2">`, month.Format("2006-01"), month.Format("January 2006"))
	for _, weekday := range []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"} {
		fmt.Fprintf(b, `<span>%s</span>`, weekday)
	}
	b.WriteString(`</div><div class="grid grid-cols-7 gap-y-2 text-center">`)
	for blank := 0; blank < int(month.Weekday()); blank++ {
		b.WriteString(`<span></span>`)
	}
	for day := month; day.Month() == month.Month(); day = day.AddDate(0, 0, 1) {
		date := day.Format("2006-01-02")
		sessions := byDate[date]
		active := ""
		if date == selectedDate {
			active = "bg-cyan-500 text-white rounded-full"
		}
		fmt.Fprintf(b, `<a href="/calendar?date=%s" class="min-h-11 py-1 flex flex-col items-center justify-center %s" aria-label="%s%s"><span>%d</span>`, date, active, esc(formatLongDate(date)), map[bool]string{true: ", logged workout"}[len(sessions) > 0], day.Day())
		if len(sessions) > 0 {
			b.WriteString(`<span class="flex gap-0.5 mt-1" aria-hidden="true">`)
			for index := 0; index < len(sessions) && index < 4; index++ {
				colors := []string{"bg-cyan-500", "bg-violet-500", "bg-amber-500", "bg-rose-500"}
				fmt.Fprintf(b, `<i class="w-1.5 h-1.5 rounded-full %s"></i>`, colors[index])
			}
			b.WriteString(`</span>`)
		}
		b.WriteString(`</a>`)
	}
	b.WriteString(`</div></section>`)
}

func formatLongDate(date string) string {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return parsed.Format("Monday, January 2, 2006")
}
func (h *Handler) exerciseHistory(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	sessions, err := h.svc.ExerciseHistory(r.Context(), name)
	if err != nil {
		http.Error(w, "invalid exercise", http.StatusBadRequest)
		return
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<a href="/calendar" class="link">← Calendar</a><h1 class="text-2xl font-bold mt-4">%s</h1><p class="text-slate-500 mt-1">History</p><div class="mt-5 space-y-6">`, esc(name))
	if len(sessions) == 0 {
		b.WriteString(`<p class="text-slate-500">No logged sets yet.</p>`)
	}
	for _, session := range sessions {
		if len(session.Exercises) == 0 {
			continue
		}
		fmt.Fprintf(&b, `<section><div class="flex justify-between items-center border-b-2 border-cyan-400 pb-1"><h2 class="font-bold uppercase">%s</h2><a class="link text-sm" href="/sessions/%d">Session</a></div><ol class="pt-2 space-y-2">`, esc(formatLongDate(session.Date)), session.ID)
		for _, set := range session.Exercises[0].Sets {
			fmt.Fprintf(&b, `<li class="grid grid-cols-2 text-lg"><span><b>%g</b> <span class="text-sm text-slate-500">kg</span></span><span><b>%d</b> <span class="text-sm text-slate-500">reps</span></span></li>`, set.WeightKg, set.Reps)
		}
		b.WriteString(`</ol></section>`)
	}
	b.WriteString(`</div>`)
	h.render(w, r, "Exercise history", template.HTML(b.String()))
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
	for _, exercise := range s.Exercises {
		fmt.Fprintf(&b, `<li class="border-t"><a class="block pt-3 hover:bg-cyan-50" href="/exercises/history?name=%s"><div class="flex justify-between gap-3"><strong>%s</strong><span class="text-cyan-600 text-sm">History ›</span></div><ol class="mt-2 space-y-1 text-slate-600">`, url.QueryEscape(exercise.Name), esc(exercise.Name))
		for _, set := range exercise.Sets {
			fmt.Fprintf(&b, "<li>Set %d — %g kg × %d reps</li>", set.Position+1, set.WeightKg, set.Reps)
		}
		b.WriteString("</ol></a></li>")
	}
	b.WriteString("</ul></article>")
	h.render(w, r, "Session", template.HTML(b.String()))
}

var page = template.Must(template.New("page").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><meta name="theme-color" content="#4338ca"><link rel="manifest" href="/manifest.webmanifest"><link rel="icon" href="/icon.svg"><title>{{.Title}} · My Gym</title><script src="https://cdn.tailwindcss.com"></script><script defer src="https://unpkg.com/alpinejs@3.14.9/dist/cdn.min.js"></script><script src="https://unpkg.com/htmx.org@2.0.4"></script><style type="text/tailwindcss">@layer components{.btn{@apply inline-flex items-center justify-center rounded-xl bg-indigo-700 text-white px-4 py-2 font-medium hover:bg-indigo-800}.btn-secondary{@apply inline-flex items-center justify-center rounded-xl bg-slate-100 text-slate-800 px-4 py-2 font-medium hover:bg-slate-200}.card{@apply bg-white rounded-2xl border border-slate-200 p-5 shadow-sm}.field{@apply mt-1 block w-full border border-slate-300 rounded-lg px-3 py-2}.link{@apply text-indigo-700 font-medium}.stepper{@apply h-10 bg-slate-200 text-3xl leading-none text-white}.track-value{@apply w-full border-b border-slate-300 text-center text-3xl font-bold py-1 bg-transparent}.save-set{@apply bg-emerald-400 text-white font-bold py-3}.clear-set{@apply bg-sky-500 text-white font-bold py-3}.nav-arrow{@apply text-cyan-500 text-4xl px-4 disabled:text-slate-300}.finish-workout{@apply bg-emerald-500 text-white font-bold rounded px-4 py-2}}</style><script>function editor(initial){return {exercises:initial.map((e,i)=>({name:e.name,key:i})),nextKey:initial.length,add(){this.exercises.push({key:this.nextKey++,name:''})}}}function workoutLogger(initial){let key=0;return {active:0,weight:0,reps:0,selectedSet:-1,exercises:initial.map(e=>({name:e.name,key:key++,sets:[]})),save(exercise){if(this.selectedSet>=0){exercise.sets[this.selectedSet].weightKg=this.weight;exercise.sets[this.selectedSet].reps=this.reps;this.selectedSet=-1}else{exercise.sets.push({key:key++,weightKg:this.weight,reps:this.reps})}},clear(){this.weight=0;this.reps=0;this.selectedSet=-1},select(exercise,index){this.selectedSet=index;this.weight=exercise.sets[index].weightKg;this.reps=exercise.sets[index].reps},previous(){if(this.active>0){this.active--;this.clear()}},next(){if(this.active<this.exercises.length-1){this.active++;this.clear()}}}}if('serviceWorker'in navigator){window.addEventListener('load',()=>navigator.serviceWorker.register('/sw.js'))}</script></head><body class="bg-slate-50 min-h-screen text-slate-900">{{if ne .Title "Log workout"}}<header class="bg-white border-b"><nav class="max-w-4xl mx-auto p-4 flex items-center justify-between gap-3"><a href="/workouts" class="font-bold text-xl text-indigo-700">My Gym</a>{{if .LoggedIn}}<div class="flex items-center gap-3"><a href="/workouts" class="link" hx-boost="true">Workouts</a><a href="/calendar" class="link" hx-boost="true">Calendar</a><form method="post" action="/logout"><input type="hidden" name="_csrf" value="{{.CSRF}}"><button class="text-sm text-slate-500">Sign out</button></form></div>{{end}}</nav></header>{{end}}<main class="max-w-4xl mx-auto p-4 md:p-8">{{.Body}}</main></body></html>`))

const manifest = `{"name":"My Gym","short_name":"My Gym","start_url":"/workouts","scope":"/","display":"standalone","background_color":"#f8fafc","theme_color":"#4338ca","icons":[{"src":"/icon.svg","sizes":"any","type":"image/svg+xml","purpose":"any maskable"}]}`
const serviceWorker = `const CACHE='gym-shell-v1';const ASSETS=['/icon.svg','/manifest.webmanifest'];self.addEventListener('install',e=>e.waitUntil(caches.open(CACHE).then(c=>c.addAll(ASSETS))));self.addEventListener('activate',e=>e.waitUntil(caches.keys().then(keys=>Promise.all(keys.filter(k=>k!==CACHE).map(k=>caches.delete(k))))));self.addEventListener('fetch',e=>{const u=new URL(e.request.url);if(e.request.method!=='GET')return;if(u.origin!==self.location.origin)return;if(ASSETS.includes(u.pathname))e.respondWith(caches.match(e.request).then(r=>r||fetch(e.request)));});`
const icon = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 192 192"><rect width="192" height="192" rx="38" fill="#4338ca"/><path d="M28 78h24v36H28zm112 0h24v36h-24zM56 67h18v58H56zm62 0h18v58h-18zM72 89h48v14H72z" fill="white"/></svg>`
