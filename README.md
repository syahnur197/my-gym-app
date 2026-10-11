# My Gym App

Kotlin/Jetpack Compose Android workout tracker with a Go API using **GORM**, **repository layer**, **service layer**, **controller layer**, and **AutoMigrate**.

## Features
- Workout templates A/B/C and custom workouts
- Add, edit, delete workouts and exercises
- Monthly calendar of completed sessions
- Immutable historical snapshots: editing or deleting a workout template never modifies past session records
- Username/password configured in backend `.env`, signed bearer-token authentication
- SQLite persistence on the Go backend

## Backend
```sh
cd backend
cp .env.example .env
# Edit APP_USERNAME, APP_PASSWORD and JWT_SECRET
go mod tidy
go run .
```
Requires Go 1.23+ and a C compiler for GORM SQLite. Database tables are automatically migrated at startup. See [backend/README.md](backend/README.md) for architecture and API documentation.

## Android
Open in Android Studio (JDK 17, Android SDK 35), sync Gradle and run. The Android emulator can reach the local Go backend at `http://10.0.2.2:8080`. A physical phone needs the host computer's LAN IP. Sign in using the backend credentials.

**Development status:** Not yet verified by Android or Go compilation in this environment. The app currently logs a template's planned exercise values with one tap; detailed per-set entry is future work. Production deployment requires HTTPS and disabling Android cleartext traffic.

## Weight convention
For two dumbbells, weights are combined kilograms; for one-arm exercises, the weight is for a single dumbbell.
