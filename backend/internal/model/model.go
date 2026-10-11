package model

import "time"

// Workout is an editable template. Sessions never reference its exercises.
type Workout struct {
	ID        uint              `gorm:"primaryKey" json:"id"`
	Name      string            `gorm:"not null" json:"name"`
	Exercises []WorkoutExercise `gorm:"foreignKey:WorkoutID;constraint:OnDelete:CASCADE" json:"exercises"`
}
type WorkoutExercise struct {
	ID        uint    `gorm:"primaryKey" json:"-"`
	WorkoutID uint    `gorm:"index;not null" json:"-"`
	Position  int     `gorm:"not null" json:"-"`
	Name      string  `gorm:"not null" json:"name"`
	Sets      int     `gorm:"not null" json:"sets"`
	Reps      int     `gorm:"not null" json:"reps"`
	WeightKg  float64 `gorm:"not null" json:"weightKg"`
}
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
	ID        uint    `gorm:"primaryKey" json:"-"`
	SessionID uint    `gorm:"index;not null" json:"-"`
	Position  int     `gorm:"not null" json:"-"`
	Name      string  `gorm:"not null" json:"name"`
	Sets      int     `gorm:"not null" json:"sets"`
	Reps      int     `gorm:"not null" json:"reps"`
	WeightKg  float64 `gorm:"not null" json:"weightKg"`
}
