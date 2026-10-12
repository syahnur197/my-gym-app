package web

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/syahnur197/my-gym-app/backend/internal/model"
)

type workoutsData struct{ Workouts []model.Workout }
type editorData struct {
	Workout   model.Workout
	Action    string
	IsEdit    bool
	Exercises string
	Library   string
}
type loggerData struct {
	Workout   model.Workout
	Date      string
	Exercises string
}
type exercisesData struct{ Exercises []model.Exercise }
type historyData struct {
	Name     string
	Sessions []model.Session
}
type sessionData struct{ Session model.Session }
type calendarDay struct {
	Blank, Selected, Sessions bool
	Date                      string
	Number                    int
	Dots                      []string
}
type calendarMonth struct {
	Title string
	Days  []calendarDay
}
type calendarSelection struct {
	Date, LongDate string
	Sessions       []model.Session
	Templates      []model.Workout
}
type calendarData struct {
	Months   []calendarMonth
	Selected *calendarSelection
}

func (h *Handler) loginPage(w http.ResponseWriter, r *http.Request) {
	if h.authorized(r) {
		http.Redirect(w, r, "/workouts", http.StatusSeeOther)
		return
	}
	h.renderView(w, r, "Sign in", "login", nil)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	token, err := h.auth.Login(r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "gym_session", Value: token, Path: "/", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
	http.Redirect(w, r, "/workouts", http.StatusSeeOther)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "gym_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) workouts(w http.ResponseWriter, r *http.Request) {
	workouts, err := h.svc.Workouts.List(r.Context())
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	h.renderView(w, r, "Workouts", "workouts", workoutsData{Workouts: workouts})
}

func parseID(r *http.Request) (uint, error) {
	value, err := strconv.ParseUint(r.PathValue("id"), 10, 32)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("invalid ID")
	}
	return uint(value), nil
}

func (h *Handler) copyWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	copy, err := h.svc.CopyWorkout(r.Context(), id)
	if err != nil {
		http.Error(w, "could not copy template", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/workouts/%d/edit", copy.ID), http.StatusSeeOther)
}

func (h *Handler) editor(w http.ResponseWriter, r *http.Request) {
	workout := model.Workout{Name: "New workout", Exercises: []model.WorkoutExercise{}}
	if r.PathValue("id") != "" {
		id, err := parseID(r)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}
		workout, err = h.svc.Workouts.Find(r.Context(), id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	library, err := h.svc.Exercises.List(r.Context())
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	exercises, _ := json.Marshal(workout.Exercises)
	options, _ := json.Marshal(library)
	action := "/workouts"
	if workout.ID != 0 {
		action = fmt.Sprintf("/workouts/%d", workout.ID)
	}
	h.renderView(w, r, "Edit workout", "editor", editorData{Workout: workout, Action: action, IsEdit: workout.ID != 0, Exercises: base64.StdEncoding.EncodeToString(exercises), Library: base64.StdEncoding.EncodeToString(options)})
}

func (h *Handler) saveWorkout(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	count, err := strconv.Atoi(r.FormValue("count"))
	if err != nil || count < 1 || count > 100 {
		http.Error(w, "a workout needs at least one exercise", http.StatusBadRequest)
		return
	}
	workout := model.Workout{Name: r.FormValue("name"), Exercises: make([]model.WorkoutExercise, 0, count)}
	if r.PathValue("id") != "" {
		workout.ID, err = parseID(r)
		if err != nil {
			http.Error(w, "invalid ID", http.StatusBadRequest)
			return
		}
	}
	for index := 0; index < count; index++ {
		id, err := strconv.ParseUint(r.FormValue(fmt.Sprintf("exercise_id_%d", index)), 10, 32)
		if err != nil || id == 0 {
			http.Error(w, "select an exercise from the library for every row", http.StatusBadRequest)
			return
		}
		workout.Exercises = append(workout.Exercises, model.WorkoutExercise{ExerciseID: uint(id)})
	}
	if err := h.svc.SaveWorkout(r.Context(), &workout); err != nil {
		http.Error(w, "invalid workout", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/workouts", http.StatusSeeOther)
}

func (h *Handler) deleteWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	if err := h.svc.Workouts.Delete(r.Context(), id); err != nil {
		http.Error(w, "delete failed", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/workouts", http.StatusSeeOther)
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
	date, ok := validDate(r.URL.Query().Get("date"))
	if !ok {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}
	encoded, _ := json.Marshal(workout.Exercises)
	h.renderView(w, r, "Log workout", "logger", loggerData{Workout: workout, Date: date, Exercises: base64.StdEncoding.EncodeToString(encoded)})
}

func validDate(value string) (string, bool) {
	if value == "" {
		return time.Now().Format("2006-01-02"), true
	}
	parsed, err := time.Parse("2006-01-02", value)
	return value, err == nil && parsed.Format("2006-01-02") == value
}

func (h *Handler) logWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
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
		exercises = append(exercises, model.SessionExercise{Name: r.FormValue(fmt.Sprintf("exercise_name_%d", exerciseIndex)), Sets: sets})
	}
	if _, err := h.svc.Log(r.Context(), id, r.FormValue("date"), exercises); err != nil {
		http.Error(w, "could not save workout", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/calendar?date="+url.QueryEscape(r.FormValue("date")), http.StatusSeeOther)
}

func (h *Handler) calendar(w http.ResponseWriter, r *http.Request) {
	selectedDate, ok := validDate(r.URL.Query().Get("date"))
	if !ok {
		http.Error(w, "invalid date", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("date") == "" {
		selectedDate = ""
	}
	current := time.Now()
	if selectedDate != "" {
		current, _ = time.Parse("2006-01-02", selectedDate)
	}
	first := time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, -2, 0)
	byDate := map[string][]model.Session{}
	for month := first; month.Before(first.AddDate(0, 6, 0)); month = month.AddDate(0, 1, 0) {
		sessions, err := h.svc.Sessions.List(r.Context(), month.Format("2006-01"))
		if err != nil {
			http.Error(w, "database error", 500)
			return
		}
		for _, session := range sessions {
			byDate[session.Date] = append(byDate[session.Date], session)
		}
	}
	data := calendarData{Months: makeCalendarMonths(first, byDate, selectedDate)}
	if selectedDate != "" {
		templates, err := h.svc.Workouts.List(r.Context())
		if err != nil {
			http.Error(w, "database error", 500)
			return
		}
		data.Selected = &calendarSelection{Date: selectedDate, LongDate: formatLongDate(selectedDate), Sessions: byDate[selectedDate], Templates: templates}
	}
	h.renderView(w, r, "Calendar", "calendar", data)
}

func makeCalendarMonths(first time.Time, sessions map[string][]model.Session, selected string) []calendarMonth {
	months := make([]calendarMonth, 0, 6)
	for month := first; month.Before(first.AddDate(0, 6, 0)); month = month.AddDate(0, 1, 0) {
		view := calendarMonth{Title: month.Format("January 2006")}
		for blank := 0; blank < int(month.Weekday()); blank++ {
			view.Days = append(view.Days, calendarDay{Blank: true})
		}
		for day := month; day.Month() == month.Month(); day = day.AddDate(0, 0, 1) {
			date := day.Format("2006-01-02")
			count := len(sessions[date])
			dots := []string{}
			colors := []string{"bg-cyan-500", "bg-violet-500", "bg-amber-500", "bg-rose-500"}
			for i := 0; i < count && i < len(colors); i++ {
				dots = append(dots, colors[i])
			}
			view.Days = append(view.Days, calendarDay{Date: date, Number: day.Day(), Selected: date == selected, Sessions: count > 0, Dots: dots})
		}
		months = append(months, view)
	}
	return months
}

func formatLongDate(date string) string {
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return parsed.Format("Monday, January 2, 2006")
}

func (h *Handler) exercises(w http.ResponseWriter, r *http.Request) {
	exercises, err := h.svc.Exercises.List(r.Context())
	if err != nil {
		http.Error(w, "database error", 500)
		return
	}
	h.renderView(w, r, "Exercises", "exercises", exercisesData{Exercises: exercises})
}
func (h *Handler) saveExercise(w http.ResponseWriter, r *http.Request) {
	exercise := model.Exercise{Name: r.FormValue("name"), Notes: r.FormValue("notes")}
	if err := h.svc.SaveExercise(r.Context(), &exercise); err != nil {
		http.Error(w, "Exercise names must be unique and no longer than 150 characters.", 400)
		return
	}
	http.Redirect(w, r, "/exercises", 303)
}
func (h *Handler) deleteExercise(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid ID", 400)
		return
	}
	if err := h.svc.Exercises.Delete(r.Context(), id); err != nil {
		http.Error(w, "exercise is used by a workout template or could not be removed", 400)
		return
	}
	http.Redirect(w, r, "/exercises", 303)
}
func (h *Handler) exerciseHistory(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	sessions, err := h.svc.ExerciseHistory(r.Context(), name)
	if err != nil {
		http.Error(w, "invalid exercise", 400)
		return
	}
	h.renderView(w, r, "Exercise history", "history", historyData{Name: name, Sessions: sessions})
}
func (h *Handler) session(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.Error(w, "invalid ID", 400)
		return
	}
	session, err := h.svc.Sessions.Find(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.renderView(w, r, "Session", "session", sessionData{Session: session})
}
