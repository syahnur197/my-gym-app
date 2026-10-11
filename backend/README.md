# Go backend and PWA

The Go server hosts the web UI at `/` and the existing JSON API at `/api/*`.

## Architecture
- `internal/model`: GORM models
- `internal/database`: SQLite connection and `AutoMigrate`
- `internal/repository`: interfaces and GORM persistence implementations
- `internal/service`: validation, defaults, immutable session snapshots
- `internal/controller`: JSON API controllers
- `internal/web`: server-rendered HTML controllers and PWA assets
- `internal/auth`: signed token issuance and verification

## Start
```sh
cp .env.example .env
# Edit credentials in .env
go mod tidy
go run .
```
Requires Go 1.23+ and a C compiler. GORM automatically migrates models on startup. For production, versioned migrations are safer for destructive schema changes.

## Browser routes
- `GET /login`, `POST /login`, `POST /logout`
- `GET /workouts`, `GET /workouts/new`, `GET /workouts/{id}/edit`
- `POST /workouts`, `POST /workouts/{id}`, `POST /workouts/{id}/delete`
- `POST /workouts/{id}/log`
- `GET /calendar?month=YYYY-MM`, `GET /sessions/{id}`
- `GET /manifest.webmanifest`, `GET /sw.js`, `GET /icon.svg`

## JSON API
- `GET /api/health`
- `POST /api/login`
- `GET /api/workouts`, `POST /api/workouts`
- `GET /api/workouts/{id}`, `PUT /api/workouts/{id}`, `DELETE /api/workouts/{id}`
- `GET /api/sessions?month=YYYY-MM`, `POST /api/sessions`, `GET /api/sessions/{id}`

API endpoints except health and login require a bearer token. Web routes use a signed HttpOnly cookie and CSRF-protected forms.

## Caveats
- The PWA requires network connectivity to log workouts; it has no offline sync.
- Frontend JS/CSS dependencies use CDNs.
- Use HTTPS and `COOKIE_SECURE=true` for production. Add login rate limiting before exposing publicly.
