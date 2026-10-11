# My Gym App

Android (Kotlin + Jetpack Compose) and Go backend for tracking workouts A, B, C and custom routines.

## Features
- Login using a single account configured in the Go server's `.env`
- Preloaded workout templates A/B/C
- Create, edit and delete workouts; add, edit and remove exercises
- Log a workout and browse historical sessions on a month calendar
- **Historical sessions are immutable snapshots.** Editing or deleting a workout template cannot change completed session logs.
- SQLite database on the backend, not on the phone

## Backend setup
1. Install Go 1.23+.
2. `cd backend && cp .env.example .env`
3. Edit `.env`: set `APP_USERNAME`, a strong `APP_PASSWORD` (12+ characters), and `JWT_SECRET` (32+ random characters).
4. `go mod tidy && go run .`
5. The server starts on port 8080 by default. See [backend/README.md](backend/README.md) for endpoints.

## Android setup
1. Open the repository in Android Studio with JDK 17 and Android SDK 35 installed.
2. Sync Gradle and run the `app` configuration on an Android 8+ emulator/device.
3. Enter server URL on the sign-in screen: `http://10.0.2.2:8080` for an Android emulator connecting to your development machine. On a physical phone, use your computer's reachable LAN IP (for example `http://192.168.1.10:8080`).
4. Sign in with the credentials from the backend `.env`.

**Security:** Development builds permit HTTP for local testing. For production, deploy the Go API behind HTTPS, disable cleartext traffic in the Android manifest, and use an HTTPS URL. Never commit `.env`, use a public demo password, or expose the API publicly over plain HTTP.

## Weight convention
Paired-dumbbell weights are combined kg; single-arm exercises use one dumbbell's kg.

## MVP limitations
This is an initial implementation, not yet verified with a CI Android/Go build. The Android editor supports template editing and quick logging of its current targets; it does not yet provide a separate per-set logging screen. Sessions are read-only by design, and login tokens are kept in memory (sign in again after app restart).
