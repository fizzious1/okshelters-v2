# Android Design

## Scope

This design covers only `android/`.

## Goal

Provide the Android application shell for ShelterNav.

## Responsibilities

- Android packaging and build setup (Gradle + manifest)
- Android UI layer (Compose)
- JNI integration path to `libshelternav`
- Android platform permissions and runtime behavior

## Current State

- Build files and manifest are present.
- NDK/CMake wiring points to `libshelternav`.
- App source implementation is still minimal and pending expansion.

## Local Constraints

- Keep UI work off the main thread where possible.
- Keep Android-only concerns in this module.
- Keep JNI interface narrow and stable.
