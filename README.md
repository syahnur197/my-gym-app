# My Gym — Go PWA

A lightweight workout tracker for A/B/C and custom routines. **Go serves both the backend and the server-rendered frontend.** No React, Vue, Next.js, or Android build.

## Stack
- **Go 1.23+**, standard `net/http` routing and `html/template`
- **GORM + SQLite** with startup AutoMigrate
- Repository, service, controller layers for JSON API
- Server-rendered HTML with **HTMX**, **Alpine.js**, **Tailwind CSS**
- Web app manifest and service worker for PWA installation
- Single-user authentication configured via local `.env`; signed, HttpOnly session cookie

## Run
```sh
cd backend
cp .env.example .env
# Set APP_USERNAME, APP_PASSWORD (12+ chars), JWT_SECRET (32+ chars)
go mod tidy
go run .
```
Open **http://localhost:8080** and sign in. A C compiler is needed for the GORM SQLite driver.

## Features
- Seeded Workout A, B, C templates
- Create, rename, edit and delete workouts and their exercises
- Log a session with a single click using the template's current values
- Monthly calendar of completed sessions; inspect historical exercise details
- **Immutable history**: session exercises and workout names are copied into separate tables. Editing/deleting a template cannot change previous logs.
- Go JSON API retained under `/api/*`

## PWA details
The manifest and service worker make the app installable on supported browsers. **Logging requires network access to the Go server.** The current service worker caches only the app manifest and icon; it does not offer offline data entry or synchronization. HTMX, Alpine and Tailwind are loaded from CDNs, so an internet connection is required for first load and the UI isn't fully offline capable.

## Security
Use HTTPS for remote access. Set `COOKIE_SECURE=true` in production (behind a TLS-terminating reverse proxy). Do not commit `.env`. Cookie-based web forms use a CSRF token; API endpoints continue to use bearer tokens. For public deployment, add login rate limiting and a hardened reverse proxy.

## Notes
- Paired dumbbell weights are **combined kg**; one-arm exercise weights refer to one dumbbell.
- This is an MVP. Logging currently snapshots the template; a per-set performance editor and offline sync are not yet implemented.
- Code has been pushed but has **not been locally compiled or browser-tested** in this environment.
