package controller

import (
 "encoding/json"
 "errors"
 "io"
 "net/http"
 "strconv"
 "strings"
 "time"
 "github.com/syahnur197/my-gym-app/backend/internal/auth"
 "github.com/syahnur197/my-gym-app/backend/internal/model"
 "github.com/syahnur197/my-gym-app/backend/internal/repository"
 "github.com/syahnur197/my-gym-app/backend/internal/service"
)

type Controller struct{svc *service.Service;auth *auth.Manager}
func New(s *service.Service,a *auth.Manager)*Controller{return &Controller{s,a}}
func jsonResponse(w http.ResponseWriter,status int,data any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(data)}
func fail(w http.ResponseWriter,status int,msg string){jsonResponse(w,status,map[string]string{"error":msg})}
func parse(r *http.Request,v any)error{
 dec:=json.NewDecoder(http.MaxBytesReader(nil,r.Body,1<<20));dec.DisallowUnknownFields()
 if err:=dec.Decode(v);err!=nil{return err}
 var tail any;if err:=dec.Decode(&tail);err!=io.EOF{return errors.New("extra JSON content")}
 return nil
}
func id(r *http.Request)(uint,error){n,err:=strconv.ParseUint(r.PathValue("id"),10,32);if err!=nil||n==0{return 0,service.ErrInvalid};return uint(n),nil}
func handleError(w http.ResponseWriter,err error){
 switch{
 case errors.Is(err,repository.ErrNotFound):fail(w,404,"not found")
 case errors.Is(err,service.ErrInvalid):fail(w,400,"invalid request")
 default:fail(w,500,"internal server error")
 }
}
func (c *Controller) Routes()http.Handler{
 mux:=http.NewServeMux()
 mux.HandleFunc("GET /api/health",func(w http.ResponseWriter,r *http.Request){jsonResponse(w,200,map[string]string{"status":"ok"})})
 mux.HandleFunc("POST /api/login",c.login)
 mux.Handle("GET /api/workouts",c.protected(http.HandlerFunc(c.listWorkouts)))
 mux.Handle("POST /api/workouts",c.protected(http.HandlerFunc(c.createWorkout)))
 mux.Handle("GET /api/workouts/{id}",c.protected(http.HandlerFunc(c.getWorkout)))
 mux.Handle("PUT /api/workouts/{id}",c.protected(http.HandlerFunc(c.updateWorkout)))
 mux.Handle("DELETE /api/workouts/{id}",c.protected(http.HandlerFunc(c.deleteWorkout)))
 mux.Handle("GET /api/sessions",c.protected(http.HandlerFunc(c.listSessions)))
 mux.Handle("POST /api/sessions",c.protected(http.HandlerFunc(c.createSession)))
 mux.Handle("GET /api/sessions/{id}",c.protected(http.HandlerFunc(c.getSession)))
 return mux
}
func (c *Controller) protected(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
 h:=r.Header.Get("Authorization");if !strings.HasPrefix(h,"Bearer ")||!c.auth.Verify(strings.TrimPrefix(h,"Bearer ")){fail(w,401,"unauthorized");return}
 next.ServeHTTP(w,r)
})}
func (c *Controller) login(w http.ResponseWriter,r *http.Request){
 var input struct{Username string `json:"username"`;Password string `json:"password"`}
 if parse(r,&input)!=nil{fail(w,400,"invalid JSON");return}
 token,err:=c.auth.Login(input.Username,input.Password);if err!=nil{fail(w,401,"invalid credentials");return}
 jsonResponse(w,200,map[string]string{"token":token,"tokenType":"Bearer"})
}
func (c *Controller) listWorkouts(w http.ResponseWriter,r *http.Request){out,err:=c.svc.Workouts.List(r.Context());if err!=nil{handleError(w,err);return};jsonResponse(w,200,out)}
func (c *Controller) getWorkout(w http.ResponseWriter,r *http.Request){n,err:=id(r);if err!=nil{handleError(w,err);return};out,err:=c.svc.Workouts.Find(r.Context(),n);if err!=nil{handleError(w,err);return};jsonResponse(w,200,out)}
func (c *Controller) createWorkout(w http.ResponseWriter,r *http.Request){
 var input model.Workout;if parse(r,&input)!=nil{fail(w,400,"invalid JSON");return}
 input.ID=0;if err:=c.svc.SaveWorkout(r.Context(),&input);err!=nil{handleError(w,err);return}
 jsonResponse(w,201,input)
}
func (c *Controller) updateWorkout(w http.ResponseWriter,r *http.Request){
 n,err:=id(r);if err!=nil{handleError(w,err);return}
 var input model.Workout;if parse(r,&input)!=nil{fail(w,400,"invalid JSON");return}
 input.ID=n;if err:=c.svc.SaveWorkout(r.Context(),&input);err!=nil{handleError(w,err);return}
 jsonResponse(w,200,input)
}
func (c *Controller) deleteWorkout(w http.ResponseWriter,r *http.Request){
 n,err:=id(r);if err!=nil{handleError(w,err);return}
 if err:=c.svc.Workouts.Delete(r.Context(),n);err!=nil{handleError(w,err);return}
 w.WriteHeader(204)
}
func (c *Controller) listSessions(w http.ResponseWriter,r *http.Request){
 month:=r.URL.Query().Get("month")
 if month!=""{parsed,err:=time.Parse("2006-01",month);if err!=nil||parsed.Format("2006-01")!=month{fail(w,400,"month must be YYYY-MM");return}}
 out,err:=c.svc.Sessions.List(r.Context(),month);if err!=nil{handleError(w,err);return};jsonResponse(w,200,out)
}
func (c *Controller) getSession(w http.ResponseWriter,r *http.Request){n,err:=id(r);if err!=nil{handleError(w,err);return};out,err:=c.svc.Sessions.Find(r.Context(),n);if err!=nil{handleError(w,err);return};jsonResponse(w,200,out)}
func (c *Controller) createSession(w http.ResponseWriter,r *http.Request){
 var input struct{WorkoutID uint `json:"workoutId"`;Date string `json:"date"`;Exercises []model.SessionExercise `json:"exercises"`}
 if parse(r,&input)!=nil||input.WorkoutID==0{fail(w,400,"invalid session");return}
 out,err:=c.svc.Log(r.Context(),input.WorkoutID,input.Date,input.Exercises);if err!=nil{handleError(w,err);return}
 jsonResponse(w,201,out)
}
