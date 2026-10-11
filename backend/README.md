# Go API

## Start
Copy `.env.example` to `.env`, set a strong `APP_PASSWORD` and a random `JWT_SECRET` (32+ characters), then:

```bash
cd backend
go mod tidy
go run .
```

The server reads `.env` from the working directory without external dotenv dependencies. Environment variables override file values. `.env` is ignored by Git.

## Authentication
`POST /api/login` with `{"username":"admin","password":"..."}`. The response contains a signed bearer token valid for 24 hours. All other `/api/*` endpoints except `/api/health` require `Authorization: Bearer TOKEN`. Credentials are compared in constant time and never returned by the API. Use HTTPS in production; never expose this server directly over public HTTP.

## API
- `GET /api/health`
- `POST /api/login`
- `GET /api/workouts`
- `POST /api/workouts` body `{"name":"Workout D","exercises":[{"name":"Squat","sets":3,"reps":10,"weightKg":30}]}`
- `GET /api/workouts/{id}`
- `PUT /api/workouts/{id}` same body as POST
- `DELETE /api/workouts/{id}`
- `GET /api/sessions?month=2026-10` or without month for all
- `POST /api/sessions` body `{"workoutId":1,"date":"2026-10-11","exercises":[{"name":"Bench press","sets":3,"reps":10,"weightKg":50}]}`
- `GET /api/sessions/{id}`

Editing/deleting a workout never changes session history: sessions persist their own immutable exercise snapshots. Session POST accepts recorded exercise details (if omitted, it snapshots the current template). Session history is read-only. Local SQLite persists data at `DATABASE_PATH`.

## Notes
- Single-user MVP: credentials are provisioned through environment variables, not stored in SQLite.
- A reverse proxy with TLS is required for remote access. Configure firewall and rate limiting before deploying publicly.
- CORS is not enabled; native Android HTTP clients do not require it.
