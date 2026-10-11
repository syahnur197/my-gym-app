package main

import (
 "context"
 "crypto/hmac"
 "crypto/rand"
 "crypto/sha256"
 "crypto/subtle"
 "database/sql"
 "encoding/base64"
 "encoding/json"
 "errors"
 "fmt"
 "log"
 "net/http"
 "os"
 "strconv"
 "strings"
 "time"

 _ "modernc.org/sqlite"
)

type Exercise struct {
 Name string `json:"name"`
 Sets int `json:"sets"`
 Reps int `json:"reps"`
 WeightKg float64 `json:"weightKg"`
}
type Workout struct {
 ID int64 `json:"id"`
 Name string `json:"name"`
 Exercises []Exercise `json:"exercises"`
}
type Session struct {
 ID int64 `json:"id"`
 WorkoutID int64 `json:"workoutId"`
 WorkoutName string `json:"workoutName"`
 Date string `json:"date"`
 Exercises []Exercise `json:"exercises"`
}
type Server struct { db *sql.DB; username, password string; secret []byte }
func main() {
 loadEnv(".env")
 username, password, secret := os.Getenv("APP_USERNAME"), os.Getenv("APP_PASSWORD"), os.Getenv("JWT_SECRET")
 if username=="" || len(password)<12 || len(secret)<32 { log.Fatal("set APP_USERNAME, APP_PASSWORD (12+ chars), JWT_SECRET (32+ chars) in environment or .env") }
 path:=os.Getenv("DATABASE_PATH"); if path=="" {path="./gym.db"}
 db,err:=sql.Open("sqlite",path); if err!=nil {log.Fatal(err)}; defer db.Close()
 db.SetMaxOpenConns(1)
 if err=initDB(db);err!=nil {log.Fatal(err)}
 s:=&Server{db:db,username:username,password:password,secret:[]byte(secret)}
 mux:=http.NewServeMux()
 mux.HandleFunc("GET /api/health",func(w http.ResponseWriter,r *http.Request){writeJSON(w,200,map[string]string{"status":"ok"})})
 mux.HandleFunc("POST /api/login",s.login)
 mux.Handle("GET /api/workouts",s.auth(http.HandlerFunc(s.listWorkouts)))
 mux.Handle("POST /api/workouts",s.auth(http.HandlerFunc(s.createWorkout)))
 mux.Handle("GET /api/workouts/{id}",s.auth(http.HandlerFunc(s.getWorkout)))
 mux.Handle("PUT /api/workouts/{id}",s.auth(http.HandlerFunc(s.updateWorkout)))
 mux.Handle("DELETE /api/workouts/{id}",s.auth(http.HandlerFunc(s.deleteWorkout)))
 mux.Handle("GET /api/sessions",s.auth(http.HandlerFunc(s.listSessions)))
 mux.Handle("POST /api/sessions",s.auth(http.HandlerFunc(s.createSession)))
 mux.Handle("GET /api/sessions/{id}",s.auth(http.HandlerFunc(s.getSession)))
 port:=os.Getenv("PORT");if port==""{port="8080"}
 log.Printf("listening on :%s",port)
 log.Fatal(http.ListenAndServe(":"+port,limitBody(mux)))
}
func loadEnv(path string) {
 data,err:=os.ReadFile(path);if err!=nil{return}
 for _,line:=range strings.Split(string(data),"\n") {
  line=strings.TrimSpace(line);if line==""||strings.HasPrefix(line,"#"){continue}
  p:=strings.SplitN(line,"=",2);if len(p)!=2{continue}
  k:=strings.TrimSpace(p[0]);v:=strings.Trim(strings.TrimSpace(p[1]),"\"'")
  if os.Getenv(k)=="" {os.Setenv(k,v)}
 }
}
func initDB(db *sql.DB)error{
 _,err:=db.Exec(`PRAGMA foreign_keys=ON;
 CREATE TABLE IF NOT EXISTS workouts(id INTEGER PRIMARY KEY, name TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS workout_exercises(id INTEGER PRIMARY KEY, workout_id INTEGER NOT NULL REFERENCES workouts(id) ON DELETE CASCADE, position INTEGER NOT NULL, name TEXT NOT NULL, sets INTEGER NOT NULL, reps INTEGER NOT NULL, weight_kg REAL NOT NULL);
 CREATE TABLE IF NOT EXISTS sessions(id INTEGER PRIMARY KEY, workout_id INTEGER NOT NULL, workout_name TEXT NOT NULL, session_date TEXT NOT NULL, created_at TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS session_exercises(id INTEGER PRIMARY KEY, session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE, position INTEGER NOT NULL, name TEXT NOT NULL, sets INTEGER NOT NULL, reps INTEGER NOT NULL, weight_kg REAL NOT NULL);
 CREATE INDEX IF NOT EXISTS idx_session_date ON sessions(session_date);`)
 if err!=nil{return err}
 var count int; if err=db.QueryRow("SELECT count(*) FROM workouts").Scan(&count);err!=nil{return err}
 if count==0 {
  defaults:=[]Workout{
   {Name:"Workout A",Exercises:[]Exercise{{"Dumbbell bench press",3,10,50},{"One-arm dumbbell row",3,10,25},{"Bulgarian split squat",3,10,30},{"Seated dumbbell shoulder press",2,10,40},{"Side plank (seconds as reps)",3,40,0}}},
   {Name:"Workout B",Exercises:[]Exercise{{"Dumbbell Romanian deadlift",3,10,50},{"Incline dumbbell bench press",3,10,50},{"Bent-over dumbbell row",3,10,50},{"Goblet squat",3,12,25},{"Dead bug",3,10,0}}},
   {Name:"Workout C",Exercises:[]Exercise{{"Dumbbell reverse lunge",3,10,30},{"Dumbbell bench press",3,10,50},{"One-arm dumbbell row",3,10,25},{"Dumbbell Romanian deadlift",2,10,50},{"Dumbbell lateral raise",2,15,12},{"Side plank (seconds as reps)",2,40,0}}},
  }
  for _,w:=range defaults {if _,err=saveWorkout(db,0,w);err!=nil{return err}}
 }
 return nil
}
func limitBody(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){r.Body=http.MaxBytesReader(w,r.Body,1<<20);next.ServeHTTP(w,r)})}
func writeJSON(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_ = json.NewEncoder(w).Encode(v)}
func problem(w http.ResponseWriter,status int,msg string){writeJSON(w,status,map[string]string{"error":msg})}
func decode(r *http.Request,v any)error{d:=json.NewDecoder(r.Body);d.DisallowUnknownFields();if err:=d.Decode(v);err!=nil{return err};var extra any;if d.Decode(&extra)==nil{return errors.New("extra JSON")};return nil}
func (s *Server) login(w http.ResponseWriter,r *http.Request){
 var v struct{Username string `json:"username"`;Password string `json:"password"`}
 if decode(r,&v)!=nil{problem(w,400,"invalid JSON");return}
 a:=subtle.ConstantTimeCompare([]byte(v.Username),[]byte(s.username))
 b:=subtle.ConstantTimeCompare([]byte(v.Password),[]byte(s.password))
 if a&b!=1{problem(w,401,"invalid credentials");return}
 payload:=fmt.Sprintf("%s:%d:%s",s.username,time.Now().Add(24*time.Hour).Unix(),randomID())
 p:=base64.RawURLEncoding.EncodeToString([]byte(payload))
 mac:=hmac.New(sha256.New,s.secret);mac.Write([]byte(p))
 token:=p+"."+base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
 writeJSON(w,200,map[string]string{"token":token,"tokenType":"Bearer"})
}
func randomID()string{b:=make([]byte,16);_,_=rand.Read(b);return base64.RawURLEncoding.EncodeToString(b)}
func (s *Server) auth(next http.Handler)http.Handler{return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
 header:=strings.TrimPrefix(r.Header.Get("Authorization"),"Bearer ")
 if header==""||header==r.Header.Get("Authorization"){problem(w,401,"bearer token required");return}
 parts:=strings.Split(header,".");if len(parts)!=2{problem(w,401,"invalid token");return}
 mac:=hmac.New(sha256.New,s.secret);mac.Write([]byte(parts[0]))
 sig,err:=base64.RawURLEncoding.DecodeString(parts[1]);if err!=nil||!hmac.Equal(sig,mac.Sum(nil)){problem(w,401,"invalid token");return}
 raw,err:=base64.RawURLEncoding.DecodeString(parts[0]);if err!=nil{problem(w,401,"invalid token");return}
 fields:=strings.Split(string(raw),":");if len(fields)!=3||fields[0]!=s.username{problem(w,401,"invalid token");return}
 exp,err:=strconv.ParseInt(fields[1],10,64);if err!=nil||time.Now().Unix()>=exp{problem(w,401,"expired token");return}
 next.ServeHTTP(w,r)
})}
func validWorkout(v Workout)bool{
 if strings.TrimSpace(v.Name)==""||len(v.Name)>100||len(v.Exercises)>100{return false}
 for _,e:=range v.Exercises{if strings.TrimSpace(e.Name)==""||len(e.Name)>150||e.Sets<1||e.Sets>100||e.Reps<0||e.Reps>10000||e.WeightKg<0||e.WeightKg>10000{return false}}
 return true
}
func saveWorkout(db *sql.DB,id int64,v Workout)(int64,error){
 tx,err:=db.Begin();if err!=nil{return 0,err};defer tx.Rollback()
 if id==0{res,e:=tx.Exec("INSERT INTO workouts(name) VALUES(?)",v.Name);if e!=nil{return 0,e};id,e=res.LastInsertId();if e!=nil{return 0,e}
 }else{res,e:=tx.Exec("UPDATE workouts SET name=? WHERE id=?",v.Name,id);if e!=nil{return 0,e};n,_:=res.RowsAffected();if n==0{return 0,sql.ErrNoRows};if _,e=tx.Exec("DELETE FROM workout_exercises WHERE workout_id=?",id);e!=nil{return 0,e}}
 for i,e:=range v.Exercises{if _,err=tx.Exec("INSERT INTO workout_exercises(workout_id,position,name,sets,reps,weight_kg) VALUES(?,?,?,?,?,?)",id,i,e.Name,e.Sets,e.Reps,e.WeightKg);err!=nil{return 0,err}}
 return id,tx.Commit()
}
func getExercises(db *sql.DB,table,foreign string,id int64)([]Exercise,error){
 q:="SELECT name,sets,reps,weight_kg FROM "+table+" WHERE "+foreign+"=? ORDER BY position"
 rows,err:=db.Query(q,id);if err!=nil{return nil,err};defer rows.Close()
 out:=[]Exercise{};for rows.Next(){var e Exercise;if err=rows.Scan(&e.Name,&e.Sets,&e.Reps,&e.WeightKg);err!=nil{return nil,err};out=append(out,e)}
 return out,rows.Err()
}
func (s *Server) workout(id int64)(Workout,error){
 var v Workout;v.ID=id
 if err:=s.db.QueryRow("SELECT name FROM workouts WHERE id=?",id).Scan(&v.Name);err!=nil{return v,err}
 var err error;v.Exercises,err=getExercises(s.db,"workout_exercises","workout_id",id);return v,err
}
func pathID(r *http.Request)(int64,error){id,e:=strconv.ParseInt(r.PathValue("id"),10,64);if id<1{return 0,errors.New("invalid id")};return id,e}
func (s *Server) listWorkouts(w http.ResponseWriter,r *http.Request){
 rows,err:=s.db.Query("SELECT id FROM workouts ORDER BY id");if err!=nil{problem(w,500,"database error");return}
 ids:=[]int64{};for rows.Next(){var id int64;if rows.Scan(&id)==nil{ids=append(ids,id)}};rows.Close()
 out:=[]Workout{};for _,id:=range ids{v,e:=s.workout(id);if e==nil{out=append(out,v)}}
 writeJSON(w,200,out)
}
func (s *Server) getWorkout(w http.ResponseWriter,r *http.Request){id,e:=pathID(r);if e!=nil{problem(w,400,"invalid id");return};v,e:=s.workout(id);if e!=nil{problem(w,404,"workout not found");return};writeJSON(w,200,v)}
func (s *Server) createWorkout(w http.ResponseWriter,r *http.Request){var v Workout;if decode(r,&v)!=nil||!validWorkout(v){problem(w,400,"invalid workout");return};id,e:=saveWorkout(s.db,0,v);if e!=nil{problem(w,500,"database error");return};v.ID=id;writeJSON(w,201,v)}
func (s *Server) updateWorkout(w http.ResponseWriter,r *http.Request){id,e:=pathID(r);if e!=nil{problem(w,400,"invalid id");return};var v Workout;if decode(r,&v)!=nil||!validWorkout(v){problem(w,400,"invalid workout");return};_,e=saveWorkout(s.db,id,v);if e==sql.ErrNoRows{problem(w,404,"workout not found");return};if e!=nil{problem(w,500,"database error");return};v.ID=id;writeJSON(w,200,v)}
func (s *Server) deleteWorkout(w http.ResponseWriter,r *http.Request){id,e:=pathID(r);if e!=nil{problem(w,400,"invalid id");return};res,e:=s.db.Exec("DELETE FROM workouts WHERE id=?",id);if e!=nil{problem(w,500,"database error");return};n,_:=res.RowsAffected();if n==0{problem(w,404,"workout not found");return};w.WriteHeader(204)}
func (s *Server) session(id int64)(Session,error){v:=Session{ID:id};err:=s.db.QueryRow("SELECT workout_id,workout_name,session_date FROM sessions WHERE id=?",id).Scan(&v.WorkoutID,&v.WorkoutName,&v.Date);if err!=nil{return v,err};v.Exercises,err=getExercises(s.db,"session_exercises","session_id",id);return v,err}
func (s *Server) createSession(w http.ResponseWriter,r *http.Request){
 var v Session;if decode(r,&v)!=nil||v.WorkoutID<1{problem(w,400,"invalid session");return}
 original,e:=s.workout(v.WorkoutID);if e!=nil{problem(w,404,"workout not found");return}
 if v.Date==""{v.Date=time.Now().Format("2006-01-02")}
 if _,e=time.Parse("2006-01-02",v.Date);e!=nil{problem(w,400,"invalid date");return}
 if v.Exercises==nil{v.Exercises=original.Exercises}
 if !validWorkout(Workout{Name:original.Name,Exercises:v.Exercises}){problem(w,400,"invalid exercises");return}
 tx,e:=s.db.Begin();if e!=nil{problem(w,500,"database error");return};defer tx.Rollback()
 res,e:=tx.Exec("INSERT INTO sessions(workout_id,workout_name,session_date,created_at) VALUES(?,?,?,?)",v.WorkoutID,original.Name,v.Date,time.Now().UTC().Format(time.RFC3339));if e!=nil{problem(w,500,"database error");return}
 id,e:=res.LastInsertId();if e!=nil{problem(w,500,"database error");return}
 for i,x:=range v.Exercises{if _,e=tx.Exec("INSERT INTO session_exercises(session_id,position,name,sets,reps,weight_kg) VALUES(?,?,?,?,?,?)",id,i,x.Name,x.Sets,x.Reps,x.WeightKg);e!=nil{problem(w,500,"database error");return}}
 if tx.Commit()!=nil{problem(w,500,"database error");return}
 v.ID=id;v.WorkoutName=original.Name;writeJSON(w,201,v)
}
func (s *Server) getSession(w http.ResponseWriter,r *http.Request){id,e:=pathID(r);if e!=nil{problem(w,400,"invalid id");return};v,e:=s.session(id);if e!=nil{problem(w,404,"session not found");return};writeJSON(w,200,v)}
func (s *Server) listSessions(w http.ResponseWriter,r *http.Request){
 month:=r.URL.Query().Get("month");query:="SELECT id FROM sessions";args:=[]any{}
 if month!=""{if len(month)!=7||month[4]!='-'{problem(w,400,"month must be YYYY-MM");return};if _,e:=time.Parse("2006-01",month);e!=nil{problem(w,400,"invalid month");return};query+=" WHERE substr(session_date,1,7)=?";args=append(args,month)}
 query+=" ORDER BY session_date DESC,id DESC"
 rows,e:=s.db.QueryContext(context.Background(),query,args...);if e!=nil{problem(w,500,"database error");return}
 ids:=[]int64{};for rows.Next(){var id int64;if rows.Scan(&id)==nil{ids=append(ids,id)}};rows.Close()
 out:=[]Session{};for _,id:=range ids{v,e:=s.session(id);if e==nil{out=append(out,v)}}
 writeJSON(w,200,out)
}
