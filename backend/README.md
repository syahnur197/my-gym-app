# My Gym backend

My Gym is a single-user, server-rendered Go application. It has no public JSON API: the browser UI and its form handlers are the application.

## Architecture
- `internal/model`: GORM models for the exercise library, templates, and immutable sessions
- `internal/database`: SQLite connection and startup migration
- `internal/repository`: persistence interfaces and GORM implementations
- `internal/service`: validation, template and immutable-session logic
- `internal/web`: server-rendered routes, authentication forms and PWA assets
- `internal/auth`: signed session-cookie issuance and verification

## Start
```sh
cp .env.example .env
# Set a strong password and 32+ character JWT_SECRET.
go run .
```

## Deploy with Docker Compose
```sh
cp .env.example .env
# Set APP_USERNAME, APP_PASSWORD, JWT_SECRET and COOKIE_SECURE=true behind HTTPS.
docker compose up -d --build
```

The SQLite database lives in the named `gym-data` volume. Back it up before upgrades with `docker compose exec web sh -c 'cp /data/gym.db /data/gym.db.backup'`. The optional `cloudflared` service requires `TUNNEL_TOKEN` in `.env`; remove that service if you use another reverse proxy.

Requires Go 1.23+ and a C compiler. GORM automatically migrates additive model changes on startup.

## Routes
- `GET /health`
- `GET /login`, `POST /login`, `POST /logout`
- `GET /workouts`, `GET /workouts/new`, `GET /workouts/{id}/edit`
- `POST /workouts`, `POST /workouts/{id}`, `POST /workouts/{id}/copy`, `POST /workouts/{id}/delete`
- `GET /workouts/{id}/log?date=YYYY-MM-DD`, `POST /workouts/{id}/log`
- `GET /calendar`, `GET /sessions/{id}`
- `GET /exercises`, `POST /exercises`, `POST /exercises/{id}/delete`
- `GET /exercises/history?name=...`
- `GET /manifest.webmanifest`, `GET /sw.js`, `GET /icon.svg`

All browser routes except `/health` and login require the signed HttpOnly session cookie. Mutating forms are protected by a CSRF token.

## Caveats
- The PWA requires network connectivity to log workouts; it has no offline sync.
- Frontend JS/CSS dependencies use CDNs.
- Use HTTPS and `COOKIE_SECURE=true` for production. Add login rate limiting before exposing publicly.
