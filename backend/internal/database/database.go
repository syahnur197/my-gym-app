package database

import (
 "fmt"
 "gorm.io/driver/sqlite"
 "gorm.io/gorm"
 "gorm.io/gorm/logger"
 "github.com/syahnur197/my-gym-app/backend/internal/model"
)

func Open(path string)(*gorm.DB,error){
 db,err:=gorm.Open(sqlite.Open(path),&gorm.Config{Logger:logger.Default.LogMode(logger.Warn)})
 if err!=nil{return nil,fmt.Errorf("open sqlite: %w",err)}
 sqlDB,err:=db.DB();if err!=nil{return nil,err}
 sqlDB.SetMaxOpenConns(1)
 if err:=db.Exec("PRAGMA foreign_keys = ON").Error;err!=nil{return nil,err}
 if err:=db.AutoMigrate(&model.Workout{},&model.WorkoutExercise{},&model.Session{},&model.SessionExercise{});err!=nil{return nil,fmt.Errorf("auto migrate: %w",err)}
 return db,nil
}
