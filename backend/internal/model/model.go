package model

import "time"

// Workout is an editable template: it contains exercise names, not prescriptions.
type Workout struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	Name      string            `gorm:"not null" json:"name"`
	Exercises []WorkoutExercise `gorm:"foreignKey:WorkoutID;constraint:OnDelete:CASCADE" json:"exercises"`
}

type WorkoutExercise struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	WorkoutID uint   `gorm:"index;not null" json:"-"`
	Position  int    `gorm:"not null" json:"-"`
	Name      string `gorm:"not null" json:"name"`
}

// Session is an immutable record of a completed workout on a specific date.
type Session struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// No foreign key: deleting a template must never delete historical sessions.
	WorkoutID   uint              `gorm:"index;not null" json:"workoutId"`
	WorkoutName string            `gorm:"not null" json:"workoutName"`
	Date        string            `gorm:"index;not null" json:"date"`
	CreatedAt   time.Time         `json:"-"`
	Exercises   []SessionExercise `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE" json:"exercises"`
}

type SessionExercise struct {
	ID        uint         `gorm:"primaryKey" json:"-"`
	SessionID uint         `gorm:"index;not null" json:"-"`
	Position  int          `gorm:"not null" json:"-"`
	Name      string       `gorm:"not null" json:"name"`
	Sets      []SessionSet `gorm:"foreignKey:SessionExerciseID;constraint:OnDelete:CASCADE" json:"sets"`
}

// SessionSet holds the actual weight and reps for one completed set.
type SessionSet struct {
	ID                uint    `gorm:"primaryKey" json:"-"`
	SessionExerciseID uint    `gorm:"index;not null" json:"-"`
	Position          int     `gorm:"not null" json:"position"`
	WeightKg          float64 `gorm:"not null" json:"weightKg"`
	Reps              int     `gorm:"not null" json:"reps"`
}
