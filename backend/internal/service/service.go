package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/syahnur197/my-gym-app/backend/internal/model"
	"github.com/syahnur197/my-gym-app/backend/internal/repository"
)

var ErrInvalid = errors.New("invalid request")

type Service struct {
	Workouts repository.WorkoutRepository
	Sessions repository.SessionRepository
}

func New(workouts repository.WorkoutRepository, sessions repository.SessionRepository) *Service {
	return &Service{Workouts: workouts, Sessions: sessions}
}

func (s *Service) SaveWorkout(ctx context.Context, workout *model.Workout) error {
	if !validWorkout(workout) {
		return ErrInvalid
	}
	return s.Workouts.Save(ctx, workout)
}

func validWorkout(workout *model.Workout) bool {
	if strings.TrimSpace(workout.Name) == "" || len(workout.Name) > 100 || len(workout.Exercises) > 100 {
		return false
	}
	for _, exercise := range workout.Exercises {
		if strings.TrimSpace(exercise.Name) == "" || len(exercise.Name) > 150 {
			return false
		}
	}
	return true
}

// Log records an immutable session. Templates supply exercise names only; every set is supplied at log time.
func (s *Service) Log(ctx context.Context, workoutID uint, date string, exercises []model.SessionExercise) (model.Session, error) {
	workout, err := s.Workouts.Find(ctx, workoutID)
	if err != nil {
		return model.Session{}, err
	}

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil || parsedDate.Format("2006-01-02") != date || !validSessionExercises(exercises) {
		return model.Session{}, ErrInvalid
	}

	session := model.Session{
		WorkoutID:   workoutID,
		WorkoutName: workout.Name,
		Date:        date,
		Exercises:   exercises,
	}
	if err := s.Sessions.Create(ctx, &session); err != nil {
		return model.Session{}, err
	}
	return session, nil
}

func validSessionExercises(exercises []model.SessionExercise) bool {
	if len(exercises) == 0 || len(exercises) > 100 {
		return false
	}
	for _, exercise := range exercises {
		if strings.TrimSpace(exercise.Name) == "" || len(exercise.Name) > 150 || len(exercise.Sets) == 0 || len(exercise.Sets) > 100 {
			return false
		}
		for _, set := range exercise.Sets {
			if set.Reps < 0 || set.Reps > 10000 || set.WeightKg < 0 || set.WeightKg > 10000 || math.IsNaN(set.WeightKg) || math.IsInf(set.WeightKg, 0) {
				return false
			}
		}
	}
	return true
}

func (s *Service) ExerciseHistory(ctx context.Context, name string) ([]model.Session, error) {
	if strings.TrimSpace(name) == "" || len(name) > 150 {
		return nil, ErrInvalid
	}
	return s.Sessions.HistoryForExercise(ctx, name)
}

func (s *Service) Seed(ctx context.Context) error {
	count, err := s.Workouts.Count(ctx)
	if err != nil || count != 0 {
		return err
	}

	defaults := []model.Workout{
		{Name: "Workout A", Exercises: exercises("Dumbbell bench press", "One-arm dumbbell row", "Bulgarian split squat", "Seated dumbbell shoulder press", "Side plank (seconds as reps)")},
		{Name: "Workout B", Exercises: exercises("Dumbbell Romanian deadlift", "Incline dumbbell bench press", "Bent-over dumbbell row", "Goblet squat", "Dead bug")},
		{Name: "Workout C", Exercises: exercises("Dumbbell reverse lunge", "Dumbbell bench press", "One-arm dumbbell row", "Dumbbell Romanian deadlift", "Dumbbell lateral raise", "Side plank (seconds as reps)")},
	}
	for index := range defaults {
		if err := s.SaveWorkout(ctx, &defaults[index]); err != nil {
			return err
		}
	}
	return nil
}

func exercises(names ...string) []model.WorkoutExercise {
	items := make([]model.WorkoutExercise, len(names))
	for index, name := range names {
		items[index] = model.WorkoutExercise{Name: name}
	}
	return items
}
