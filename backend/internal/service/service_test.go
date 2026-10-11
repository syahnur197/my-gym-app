package service_test

import (
 "context"
 "testing"

 "github.com/syahnur197/my-gym-app/backend/internal/database"
 "github.com/syahnur197/my-gym-app/backend/internal/model"
 "github.com/syahnur197/my-gym-app/backend/internal/repository"
 "github.com/syahnur197/my-gym-app/backend/internal/service"
)

func TestHistoricalSessionUnaffectedByTemplateEditOrDelete(t *testing.T){
 db,err:=database.Open("file:history_test?mode=memory&cache=shared")
 if err!=nil{t.Fatal(err)}
 s:=service.New(repository.NewWorkout(db),repository.NewSession(db))
 ctx:=context.Background()
 workout:=model.Workout{Name:"Workout A",Exercises:[]model.WorkoutExercise{{Name:"Bench press",Sets:3,Reps:10,WeightKg:50}}}
 if err:=s.SaveWorkout(ctx,&workout);err!=nil{t.Fatal(err)}
 session,err:=s.Log(ctx,workout.ID,"2026-10-11",nil);if err!=nil{t.Fatal(err)}
 workout.Name="Renamed A"
 workout.Exercises=[]model.WorkoutExercise{{Name:"Squat",Sets:5,Reps:5,WeightKg:60}}
 if err:=s.SaveWorkout(ctx,&workout);err!=nil{t.Fatal(err)}
 if err:=s.Workouts.Delete(ctx,workout.ID);err!=nil{t.Fatal(err)}
 saved,err:=s.Sessions.Find(ctx,session.ID);if err!=nil{t.Fatal(err)}
 if saved.WorkoutName!="Workout A"||len(saved.Exercises)!=1||saved.Exercises[0].Name!="Bench press"||saved.Exercises[0].WeightKg!=50{
  t.Fatalf("historical session mutated: %+v",saved)
 }
}
