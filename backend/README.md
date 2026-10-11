# Go backend (GORM + SQLite)

## Structure
- `main.go` — configuration and dependency wiring
- `internal/model/` — GORM entities and relationships
- `internal/database/` — connection and **GORM AutoMigrate** on startup
- `internal/repository/` — persistence interfaces and GORM implementations
- `internal/service/` — business rules, validation, initial workout templates, immutable session snapshots
- `internal/controller/` — HTTP handlers and routing
- `internal/auth/` — HMAC-signed bearer tokens

## Start
Install Go 1.23+ and a C compiler (the standard GORM SQLite driver uses CGO). Then:

```sh
cd backend
cp .env.example .env
# Edit .env with your own APP_USERNAME, APP_PASSWORD and JWT_SECRET
go mod tidy
go run .
```

GORM calls `AutoMigrate` for the four tables on startup; no manual schema setup is needed. Note: AutoMigrate is additive and is not a replacement for versioned destructive migrations in production.

## Authentication
`POST /api/login` with `{"username":"admin","password":"..."}` returns a signed token valid for 24 hours. All other `/api/*` endpoints except `/api/health` require `Authorization: Bearer TOKEN`. Single-user credentials are loaded from environment or `.env`, not stored in the database. Never commit `.env`.

## Endpoints
- `GET /api/health`
- `POST /api/login`
- `GET /api/workouts`
- `POST /api/workouts` body `{"name":"Workout D","exercises":[{"name":"Squat","sets":3,"reps":10,"weightKg":30}]}`
- `GET /api/workouts/{id}`
- `PUT /api/workouts/{id}` same body as POST
- `DELETE /api/workouts/{id}`
- `GET /api/sessions?month=2026-10` or omit month for all sessions
- `POST /api/sessions` body `{"workoutId":1,"date":"2026-10-11","exercises":[{"name":"Bench press","sets":3,"reps":10,"weightKg":50}]}`
- `GET /api/sessions/{id}`

Session exercises and workout name are copied into historical records. Editing or deleting a workout does **not** change old sessions. Sessions are intentionally read-only.

## Security and deployment
Use HTTPS behind a reverse proxy for remote access. Set strong credentials, restrict public access and add proxy-side rate limiting. The Android app's HTTP allowance is for local development only.
