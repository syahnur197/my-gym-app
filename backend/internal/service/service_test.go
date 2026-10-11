package service_test

import (
	"context"
	"testing"

	"github.com/syahnur197/my-gym-app/backend/internal/database"
	"github.com/syahnur197/my-gym-app/backend/internal/model"
	"github.com/syahnur197/my-gym-app/backend/internal/repository"
	"github.com/syahnur197/my-gym-app/backend/internal/service"
)

func TestHistoricalSessionUnaffectedByTemplateEditOrDelete(t *testing.T) {
	db, err := database.Open("file:history_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	svc := service.New(repository.NewWorkout(db), repository.NewSession(db))
	ctx := context.Background()
	workout := model.Workout{Name: "Workout A", Exercises: []model.WorkoutExercise{{Name: "Bench press"}}}
	if err := svc.SaveWorkout(ctx, &workout); err != nil {
		t.Fatal(err)
	}

	session, err := svc.Log(ctx, workout.ID, "2026-10-11", []model.SessionExercise{{
		Name: "Bench press",
		Sets: []model.SessionSet{{WeightKg: 50, Reps: 10}, {WeightKg: 52.5, Reps: 8}},
	}})
	if err != nil {
		t.Fatal(err)
	}

	workout.Name = "Renamed A"
	workout.Exercises = []model.WorkoutExercise{{Name: "Squat"}}
	if err := svc.SaveWorkout(ctx, &workout); err != nil {
		t.Fatal(err)
	}
	if err := svc.Workouts.Delete(ctx, workout.ID); err != nil {
		t.Fatal(err)
	}

	saved, err := svc.Sessions.Find(ctx, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.WorkoutName != "Workout A" || len(saved.Exercises) != 1 || saved.Exercises[0].Name != "Bench press" || len(saved.Exercises[0].Sets) != 2 || saved.Exercises[0].Sets[1].WeightKg != 52.5 || saved.Exercises[0].Sets[1].Reps != 8 {
		t.Fatalf("historical session mutated: %+v", saved)
	}
}

func TestExerciseHistoryIncludesOnlyMatchingExercise(t *testing.T) {
	db, err := database.Open("file:exercise_history_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	svc := service.New(repository.NewWorkout(db), repository.NewSession(db))
	ctx := context.Background()
	workout := model.Workout{Name: "Workout A", Exercises: []model.WorkoutExercise{{Name: "Bench press"}, {Name: "Squat"}}}
	if err := svc.SaveWorkout(ctx, &workout); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct {
		date   string
		weight float64
	}{{"2026-10-01", 45}, {"2026-10-11", 50}} {
		_, err = svc.Log(ctx, workout.ID, entry.date, []model.SessionExercise{{Name: "Bench press", Sets: []model.SessionSet{{WeightKg: entry.weight, Reps: 10}}}, {Name: "Squat", Sets: []model.SessionSet{{WeightKg: 60, Reps: 5}}}})
		if err != nil {
			t.Fatal(err)
		}
	}
	history, err := svc.ExerciseHistory(ctx, "Bench press")
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Date != "2026-10-11" || len(history[0].Exercises) != 1 || history[0].Exercises[0].Name != "Bench press" || history[0].Exercises[0].Sets[0].WeightKg != 50 {
		t.Fatalf("unexpected history: %+v", history)
	}
}
