# My Gym App

Offline Android workout tracker built with Kotlin and Jetpack Compose.

## Features
- Preloaded full-body templates **A**, **B**, **C**
- Add, rename and delete workout templates
- Add, edit and remove exercises and targets within each template
- Log a session with actual weight, reps and sets
- **Immutable history:** session exercises are copied into snapshot rows when saved; changing a template never changes previously logged sessions
- Monthly calendar with workout-day indicators; tap a date to inspect its sessions
- All data stored locally using Android SQLite

## Run
Open this repository in Android Studio, use JDK 17, sync Gradle and run the `app` configuration on an Android 8+ device/emulator. An internet connection is only required for initial dependency download.

## Weight convention
For exercises using two dumbbells, weights are **combined kilograms**. For single-arm exercises, weight is for **one dumbbell**. Exercise notes can clarify this.

## Data model
`templates` -> `template_exercises` defines editable plans.
`sessions` -> `session_exercises` stores a permanent copy of each completed session, including exercise names, target sets/reps, performed sets/reps and weight. No foreign key from session exercises to mutable template exercises.

## Limitations
Local-only MVP; no cloud sync or data export yet. Calendar dates use the device's local date. A saved session is read-only to protect its original log.
