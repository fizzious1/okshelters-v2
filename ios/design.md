# iOS Design

## Scope

This design covers only `ios/`.

## Goal

Provide the iOS application shell for ShelterNav.

## Responsibilities

- SwiftUI app flow and screen composition
- iOS-specific UX patterns (sheet/overlay/haptics)
- Swift bridge boundary to `libshelternav`
- iOS-side state management via view models

## Current State

- SwiftUI structure exists (`App`, `Views`, `ViewModels`, `Theme`, `Bridge`).
- Visual tokens/components are present.
- Map and bridge integration still contain TODO stubs.

## Local Constraints

- Keep heavy work off the main thread.
- Keep bridge interactions contained to `Bridge/`.
- Keep iOS feature logic within this module only.
