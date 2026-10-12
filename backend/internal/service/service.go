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
	Exercises repository.ExerciseRepository
	Workouts  repository.WorkoutRepository
	Sessions  repository.SessionRepository
}

func New(exercises repository.ExerciseRepository, workouts repository.WorkoutRepository, sessions repository.SessionRepository) *Service {
	return &Service{Exercises: exercises, Workouts: workouts, Sessions: sessions}
}

func (s *Service) SaveExercise(ctx context.Context, exercise *model.Exercise) error {
	exercise.Name = strings.TrimSpace(exercise.Name)
	exercise.Notes = strings.TrimSpace(exercise.Notes)
	if exercise.Name == "" || len(exercise.Name) > 150 || len(exercise.Notes) > 1000 {
		return ErrInvalid
	}
	return s.Exercises.Save(ctx, exercise)
}

// CopyWorkout makes an independent template. It deliberately does not copy any
// logged session: sessions are immutable historical records.
func (s *Service) CopyWorkout(ctx context.Context, sourceID uint) (model.Workout, error) {
	source, err := s.Workouts.Find(ctx, sourceID)
	if err != nil {
		return model.Workout{}, err
	}
	copy := model.Workout{Name: source.Name + " (copy)", Exercises: make([]model.WorkoutExercise, len(source.Exercises))}
	for i, exercise := range source.Exercises {
		copy.Exercises[i] = model.WorkoutExercise{ExerciseID: exercise.ExerciseID}
	}
	if err := s.SaveWorkout(ctx, &copy); err != nil {
		return model.Workout{}, err
	}
	return copy, nil
}

func (s *Service) SaveWorkout(ctx context.Context, workout *model.Workout) error {
	workout.Name = strings.TrimSpace(workout.Name)
	if len(workout.Exercises) == 0 || len(workout.Exercises) > 100 {
		return ErrInvalid
	}
	for i := range workout.Exercises {
		if workout.Exercises[i].ExerciseID == 0 {
			return ErrInvalid
		}
		exercise, err := s.Exercises.Find(ctx, workout.Exercises[i].ExerciseID)
		if err != nil {
			return ErrInvalid
		}
		// The entity is canonical: clients cannot invent or rename template exercises.
		workout.Exercises[i].Name = exercise.Name
	}
	if !validWorkout(workout) {
		return ErrInvalid
	}
	return s.Workouts.Save(ctx, workout)
}

func validWorkout(workout *model.Workout) bool {
	if strings.TrimSpace(workout.Name) == "" || len(workout.Name) > 100 || len(workout.Exercises) == 0 || len(workout.Exercises) > 100 {
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
		if strings.TrimSpace(exercise.Name) == "" || len(exercise.Name) > 150 || len(exercise.Sets) > 100 {
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
	defaults := defaultWorkouts()
	existingWorkouts, err := s.Workouts.List(ctx)
	if err != nil {
		return err
	}
	allWorkouts := append(append([]model.Workout{}, defaults...), existingWorkouts...)
	if err := s.seedDefaultExercises(ctx, allWorkouts); err != nil {
		return err
	}

	library, err := s.Exercises.List(ctx)
	if err != nil {
		return err
	}
	byName := make(map[string]uint, len(library))
	for _, exercise := range library {
		byName[exercise.Name] = exercise.ID
	}
	for index := range defaults {
		linkWorkoutExercises(&defaults[index], byName)
	}
	// Link templates created before ExerciseID existed without changing their
	// names, order, or historical sessions.
	for index := range existingWorkouts {
		if linkWorkoutExercises(&existingWorkouts[index], byName) {
			if err := s.Workouts.Save(ctx, &existingWorkouts[index]); err != nil {
				return err
			}
		}
	}

	if len(existingWorkouts) != 0 {
		return nil
	}
	for index := range defaults {
		if err := s.SaveWorkout(ctx, &defaults[index]); err != nil {
			return err
		}
	}
	return nil
}

// seedDefaultExercises keeps the reusable library useful on both a fresh
// install and an existing database created before the library was introduced.
func (s *Service) seedDefaultExercises(ctx context.Context, workouts []model.Workout) error {
	existing, err := s.Exercises.List(ctx)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(existing))
	for _, exercise := range existing {
		seen[exercise.Name] = struct{}{}
	}
	for _, workout := range workouts {
		for _, exercise := range workout.Exercises {
			if _, ok := seen[exercise.Name]; ok {
				continue
			}
			if err := s.SaveExercise(ctx, &model.Exercise{Name: exercise.Name}); err != nil {
				return err
			}
			seen[exercise.Name] = struct{}{}
		}
	}
	return nil
}

func linkWorkoutExercises(workout *model.Workout, exerciseIDs map[string]uint) bool {
	changed := false
	for index := range workout.Exercises {
		if id := exerciseIDs[workout.Exercises[index].Name]; id != 0 && workout.Exercises[index].ExerciseID != id {
			workout.Exercises[index].ExerciseID = id
			changed = true
		}
	}
	return changed
}

func defaultWorkouts() []model.Workout {
	return []model.Workout{
		{Name: "Workout A", Exercises: exercises("Dumbbell bench press", "One-arm dumbbell row", "Bulgarian split squat", "Seated dumbbell shoulder press", "Side plank (seconds as reps)")},
		{Name: "Workout B", Exercises: exercises("Dumbbell Romanian deadlift", "Incline dumbbell bench press", "Bent-over dumbbell row", "Goblet squat", "Dead bug")},
		{Name: "Workout C", Exercises: exercises("Dumbbell reverse lunge", "Dumbbell bench press", "One-arm dumbbell row", "Dumbbell Romanian deadlift", "Dumbbell lateral raise", "Side plank (seconds as reps)")},
	}
}

func exercises(names ...string) []model.WorkoutExercise {
	items := make([]model.WorkoutExercise, len(names))
	for index, name := range names {
		items[index] = model.WorkoutExercise{Name: name}
	}
	return items
}
