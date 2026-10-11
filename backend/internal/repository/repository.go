package repository

import (
 "context"
 "errors"
 "fmt"
 "gorm.io/gorm"
 "github.com/syahnur197/my-gym-app/backend/internal/model"
)

var ErrNotFound=errors.New("record not found")
type WorkoutRepository interface {
 List(context.Context)([]model.Workout,error)
 Find(context.Context,uint)(model.Workout,error)
 Save(context.Context,*model.Workout)error
 Delete(context.Context,uint)error
 Count(context.Context)(int64,error)
}
type SessionRepository interface {
 List(context.Context,string)([]model.Session,error)
 Find(context.Context,uint)(model.Session,error)
 Create(context.Context,*model.Session)error
}
type GormWorkoutRepository struct{db *gorm.DB}
type GormSessionRepository struct{db *gorm.DB}
func NewWorkout(db *gorm.DB)WorkoutRepository{return &GormWorkoutRepository{db}}
func NewSession(db *gorm.DB)SessionRepository{return &GormSessionRepository{db}}
func (r *GormWorkoutRepository)Count(ctx context.Context)(int64,error){var n int64;err:=r.db.WithContext(ctx).Model(&model.Workout{}).Count(&n).Error;return n,err}
func (r *GormWorkoutRepository)List(ctx context.Context)([]model.Workout,error){
 var out []model.Workout
 err:=r.db.WithContext(ctx).Preload("Exercises",func(db *gorm.DB)*gorm.DB{return db.Order("position ASC")}).Order("id ASC").Find(&out).Error
 if out==nil{out=[]model.Workout{}}
 return out,err
}
func (r *GormWorkoutRepository)Find(ctx context.Context,id uint)(model.Workout,error){
 var w model.Workout
 err:=r.db.WithContext(ctx).Preload("Exercises",func(db *gorm.DB)*gorm.DB{return db.Order("position ASC")}).First(&w,id).Error
 if errors.Is(err,gorm.ErrRecordNotFound){return w,ErrNotFound}
 return w,err
}
func (r *GormWorkoutRepository)Save(ctx context.Context,w *model.Workout)error{
 return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB)error{
  if w.ID==0 {
   if err:=tx.Omit("Exercises").Create(w).Error;err!=nil{return err}
  }else{
   res:=tx.Model(&model.Workout{}).Where("id = ?",w.ID).Update("name",w.Name)
   if res.Error!=nil{return res.Error}
   if res.RowsAffected==0{return ErrNotFound}
   if err:=tx.Where("workout_id = ?",w.ID).Delete(&model.WorkoutExercise{}).Error;err!=nil{return err}
  }
  for i:=range w.Exercises {w.Exercises[i].ID=0;w.Exercises[i].WorkoutID=w.ID;w.Exercises[i].Position=i}
  if len(w.Exercises)>0{return tx.Create(&w.Exercises).Error}
  return nil
 })
}
func (r *GormWorkoutRepository)Delete(ctx context.Context,id uint)error{
 return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB)error{
  res:=tx.Delete(&model.Workout{},id)
  if res.Error!=nil{return res.Error}
  if res.RowsAffected==0{return ErrNotFound}
  return tx.Where("workout_id = ?",id).Delete(&model.WorkoutExercise{}).Error
 })
}
func (r *GormSessionRepository)List(ctx context.Context,month string)([]model.Session,error){
 var out []model.Session
 q:=r.db.WithContext(ctx).Preload("Exercises",func(db *gorm.DB)*gorm.DB{return db.Order("position ASC")}).Order("date DESC, id DESC")
 if month!=""{q=q.Where("date >= ? AND date < ?",month+"-01",nextMonth(month))}
 err:=q.Find(&out).Error
 if out==nil{out=[]model.Session{}}
 return out,err
}
func nextMonth(month string)string{var y,m int;fmtSscanfMonth(month,&y,&m);if m==12{return fmtMonth(y+1,1)+"-01"};return fmtMonth(y,m+1)+"-01"}
func fmtSscanfMonth(s string,y,m *int){_,_=fmt.Sscanf(s,"%d-%d",y,m)}
func fmtMonth(y,m int)string{return fmt.Sprintf("%04d-%02d",y,m)}
func (r *GormSessionRepository)Find(ctx context.Context,id uint)(model.Session,error){
 var s model.Session
 err:=r.db.WithContext(ctx).Preload("Exercises",func(db *gorm.DB)*gorm.DB{return db.Order("position ASC")}).First(&s,id).Error
 if errors.Is(err,gorm.ErrRecordNotFound){return s,ErrNotFound}
 return s,err
}
func (r *GormSessionRepository)Create(ctx context.Context,s *model.Session)error{
 return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB)error{
  // Persist snapshots only; never write template exercises here.
  for i:=range s.Exercises{s.Exercises[i].Position=i}
  return tx.Create(s).Error
 })
}
