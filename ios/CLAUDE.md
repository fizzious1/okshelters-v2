# CLAUDE.md - ios/

## Scope

This file applies only to code under `ios/`.
Do not define or modify behavior for other modules here.

## Purpose

iOS app shell for ShelterNav:

- SwiftUI screens and interaction
- iOS platform integrations (location, haptics, app lifecycle)
- Swift bridge to `libshelternav`

## Local Rules

1. Keep frame rendering smooth and avoid main-thread blocking work.
2. Keep FFI boundary explicit in `Bridge/`.
3. Keep view-state logic in view models, not in view bodies.
4. Keep this module focused on iOS app concerns only.

## Current Local Structure

- `ShelterNav/App.swift`
- `ShelterNav/Bridge/`
- `ShelterNav/ViewModels/`
- `ShelterNav/Views/`
- `ShelterNav/Theme/`
- `ShelterNav/Models.swift`

## Out of Scope

- Backend routing/auth/caching
- Rust service implementation
- C algorithm internals

## Git Workflow (Mandatory)

- Work only in branches of the GitHub repository.
- Do not do feature work directly on `main`.
- Merge the working branch back into the repository when the task is done.
