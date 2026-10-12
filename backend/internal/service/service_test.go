package service_test

import (
	"context"
	"testing"

	"github.com/syahnur197/my-gym-app/backend/internal/database"
	"github.com/syahnur197/my-gym-app/backend/internal/model"
	"github.com/syahnur197/my-gym-app/backend/internal/repository"
	"github.com/syahnur197/my-gym-app/backend/internal/service"
)

func newService(t *testing.T, name string) (*service.Service, context.Context) {
	t.Helper()
	db, err := database.Open("file:" + name + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	return service.New(repository.NewExercise(db), repository.NewWorkout(db), repository.NewSession(db)), context.Background()
}

func addExercise(t *testing.T, svc *service.Service, ctx context.Context, name string) model.Exercise {
	t.Helper()
	exercise := model.Exercise{Name: name}
	if err := svc.SaveExercise(ctx, &exercise); err != nil {
		t.Fatal(err)
	}
	return exercise
}

func TestHistoricalSessionUnaffectedByTemplateEditOrDelete(t *testing.T) {
	svc, ctx := newService(t, "history_test")
	bench := addExercise(t, svc, ctx, "Bench press")
	squat := addExercise(t, svc, ctx, "Squat")
	workout := model.Workout{Name: "Workout A", Exercises: []model.WorkoutExercise{{ExerciseID: bench.ID}}}
	if err := svc.SaveWorkout(ctx, &workout); err != nil {
		t.Fatal(err)
	}
	session, err := svc.Log(ctx, workout.ID, "2026-10-11", []model.SessionExercise{{Name: "Bench press", Sets: []model.SessionSet{{WeightKg: 50, Reps: 10}}}})
	if err != nil {
		t.Fatal(err)
	}
	workout.Name, workout.Exercises = "Renamed A", []model.WorkoutExercise{{ExerciseID: squat.ID}}
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
	if saved.WorkoutName != "Workout A" || saved.Exercises[0].Name != "Bench press" {
		t.Fatalf("history changed: %+v", saved)
	}
}

func TestCopyWorkoutCreatesIndependentTemplate(t *testing.T) {
	svc, ctx := newService(t, "copy_test")
	bench := addExercise(t, svc, ctx, "Bench press")
	row := addExercise(t, svc, ctx, "Row")
	workout := model.Workout{Name: "Workout A", Exercises: []model.WorkoutExercise{{ExerciseID: bench.ID}, {ExerciseID: row.ID}}}
	if err := svc.SaveWorkout(ctx, &workout); err != nil {
		t.Fatal(err)
	}
	copy, err := svc.CopyWorkout(ctx, workout.ID)
	if err != nil {
		t.Fatal(err)
	}
	if copy.ID == workout.ID || copy.Name != "Workout A (copy)" || len(copy.Exercises) != 2 {
		t.Fatalf("unexpected copy: %+v", copy)
	}
}

func TestSeedLinksDefaultLibraryAndExistingTemplates(t *testing.T) {
	svc, ctx := newService(t, "seed_test")
	if err := svc.Seed(ctx); err != nil {
		t.Fatal(err)
	}
	workouts, err := svc.Workouts.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(workouts) != 3 || workouts[0].Exercises[0].ExerciseID == 0 {
		t.Fatalf("default plans were not linked: %+v", workouts)
	}
	exercises, err := svc.Exercises.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(exercises) != 12 {
		t.Fatalf("got %d exercises, want 12", len(exercises))
	}
}
