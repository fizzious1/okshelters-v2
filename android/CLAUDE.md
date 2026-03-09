# CLAUDE.md - android/

## Scope

This file applies only to code under `android/`.
Do not define or modify behavior for other modules here.

## Purpose

Android app shell for ShelterNav:

- Jetpack Compose UI
- Android lifecycle, permissions, and packaging
- JNI boundary to `libshelternav`

## Local Rules

1. Keep UI smooth (60fps target).
2. Keep heavy work off the main thread.
3. Keep Android-specific logic in this module; do not move backend or core-native logic here.
4. Use only trusted Android dependencies already used by this module unless explicitly required.

## Current Local Structure

- `app/build.gradle.kts`
- `app/src/main/AndroidManifest.xml`
- module Gradle files under `android/`

## Out of Scope

- Backend API implementation
- Rust geo-service implementation
- C core algorithm implementation

## Git Workflow (Mandatory)

- Work only in branches of the GitHub repository.
- Do not do feature work directly on `main`.
- Merge the working branch back into the repository when the task is done.
