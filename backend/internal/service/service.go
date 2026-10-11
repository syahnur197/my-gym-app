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

var ErrInvalid=errors.New("invalid request")
type Service struct{Workouts repository.WorkoutRepository;Sessions repository.SessionRepository}
func New(w repository.WorkoutRepository,s repository.SessionRepository)*Service{return &Service{w,s}}
func valid(name string,ex []model.WorkoutExercise)bool{
 if strings.TrimSpace(name)==""||len(name)>100||len(ex)>100{return false}
 for _,e:=range ex{if strings.TrimSpace(e.Name)==""||len(e.Name)>150||e.Sets<1||e.Sets>100||e.Reps<0||e.Reps>10000||e.WeightKg<0||e.WeightKg>10000||math.IsNaN(e.WeightKg)||math.IsInf(e.WeightKg,0){return false}}
 return true
}
func (s *Service)SaveWorkout(ctx context.Context,w *model.Workout)error{
 if !valid(w.Name,w.Exercises){return ErrInvalid}
 return s.Workouts.Save(ctx,w)
}
func (s *Service)Log(ctx context.Context,workoutID uint,date string,performed []model.SessionExercise)(model.Session,error){
 w,err:=s.Workouts.Find(ctx,workoutID);if err!=nil{return model.Session{},err}
 if date==""{date=time.Now().Format("2006-01-02")}
 d,err:=time.Parse("2006-01-02",date);if err!=nil||d.Format("2006-01-02")!=date{return model.Session{},ErrInvalid}
 if performed==nil{
  performed=make([]model.SessionExercise,len(w.Exercises))
  for i,e:=range w.Exercises{performed[i]=model.SessionExercise{Name:e.Name,Sets:e.Sets,Reps:e.Reps,WeightKg:e.WeightKg,Position:i}}
 }
 if len(performed)>100{return model.Session{},ErrInvalid}
 for _,e:=range performed{if strings.TrimSpace(e.Name)==""||len(e.Name)>150||e.Sets<1||e.Sets>100||e.Reps<0||e.Reps>10000||e.WeightKg<0||e.WeightKg>10000||math.IsNaN(e.WeightKg)||math.IsInf(e.WeightKg,0){return model.Session{},ErrInvalid}}
 entry:=model.Session{WorkoutID:workoutID,WorkoutName:w.Name,Date:date,Exercises:performed}
 if err:=s.Sessions.Create(ctx,&entry);err!=nil{return model.Session{},err}
 return entry,nil
}
func (s *Service)Seed(ctx context.Context)error{
 n,err:=s.Workouts.Count(ctx);if err!=nil{return err};if n!=0{return nil}
 defaults:=[]model.Workout{
  {Name:"Workout A",Exercises:[]model.WorkoutExercise{{Name:"Dumbbell bench press",Sets:3,Reps:10,WeightKg:50},{Name:"One-arm dumbbell row",Sets:3,Reps:10,WeightKg:25},{Name:"Bulgarian split squat",Sets:3,Reps:10,WeightKg:30},{Name:"Seated dumbbell shoulder press",Sets:2,Reps:10,WeightKg:40},{Name:"Side plank (seconds as reps)",Sets:3,Reps:40}}},
  {Name:"Workout B",Exercises:[]model.WorkoutExercise{{Name:"Dumbbell Romanian deadlift",Sets:3,Reps:10,WeightKg:50},{Name:"Incline dumbbell bench press",Sets:3,Reps:10,WeightKg:50},{Name:"Bent-over dumbbell row",Sets:3,Reps:10,WeightKg:50},{Name:"Goblet squat",Sets:3,Reps:12,WeightKg:25},{Name:"Dead bug",Sets:3,Reps:10}}},
  {Name:"Workout C",Exercises:[]model.WorkoutExercise{{Name:"Dumbbell reverse lunge",Sets:3,Reps:10,WeightKg:30},{Name:"Dumbbell bench press",Sets:3,Reps:10,WeightKg:50},{Name:"One-arm dumbbell row",Sets:3,Reps:10,WeightKg:25},{Name:"Dumbbell Romanian deadlift",Sets:2,Reps:10,WeightKg:50},{Name:"Dumbbell lateral raise",Sets:2,Reps:15,WeightKg:12},{Name:"Side plank (seconds as reps)",Sets:2,Reps:40}}},
 }
 for i:=range defaults{if err:=s.SaveWorkout(ctx,&defaults[i]);err!=nil{return err}}
 return nil
}
