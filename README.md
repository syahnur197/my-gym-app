# My Gym — Go PWA

A lightweight, single-user workout tracker. **Go serves the server-rendered application directly**: no React, Vue, Next.js, mobile build, or separate API.

## Stack
- **Go 1.23+**, standard `net/http` routing and `html/template`
- **GORM + SQLite** with startup AutoMigrate
- Repository and service layers for persistence and validation
- Server-rendered HTML with **Alpine.js** and **Tailwind CSS**
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
- A reusable exercise library feeds ordered workout templates; templates contain no prescribed sets, reps or weights.
- Copy a template into an independent editable workout plan.
- Tap any calendar date, choose a template, and log that workout on the selected date.
- Record each completed set independently, including its own weight and reps.
- A calendar exposes completed sessions by date.
- Exercise history retains completed sets grouped by session date.
- **Immutable history**: workout name and exercises are copied into session tables. Editing or deleting a template cannot change prior logs.

## Security
Use HTTPS for remote access. Set `COOKIE_SECURE=true` in production. Do not commit `.env`. Cookie-based forms use a CSRF token. For public deployment, add login rate limiting and a hardened reverse proxy.

## PWA details
The manifest and service worker make the app installable on supported browsers. Logging requires network access. The service worker caches only the manifest and icon; it does not offer offline data entry or synchronization. Alpine and Tailwind are loaded from CDNs, so an internet connection is required for first load.

## Deploy with Docker Compose
```sh
cd backend
cp .env.example .env
# Set APP_USERNAME, APP_PASSWORD, JWT_SECRET and COOKIE_SECURE=true behind HTTPS.
docker compose up -d --build
```

The named `gym-data` volume persists the SQLite database. The supplied compose file includes Cloudflare Tunnel; set `TUNNEL_TOKEN` or remove the `cloudflared` service when using another proxy.

## Notes
- Paired dumbbell weights are **combined kg**; one-arm exercise weights refer to one dumbbell.
- The app intentionally has no standalone JSON API. Keep browser behaviour in the server-rendered handlers unless a real external client is introduced.
