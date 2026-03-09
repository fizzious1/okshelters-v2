# CLAUDE.md - libshelternav/

## Scope

This file applies only to code under `libshelternav/`.
Do not define or modify behavior for other modules here.

## Purpose

Native C library for mobile clients:

- KD-tree nearest lookup
- Haversine distance operations
- route function surface (`sn_route_astar`)
- local SQLite sync helpers

## Local Rules

1. Keep C ABI stable (`include/shelternav.h`).
2. Keep memory behavior explicit and safe.
3. Keep scalar reference logic correct before SIMD/ASM optimizations.
4. Keep platform-specific assembly isolated under `asm/`.

## Current Local Structure

- `include/`
- `src/`
- `asm/`
- `CMakeLists.txt`

## Out of Scope

- iOS/Android UI state management
- gateway HTTP behavior
- Rust service process lifecycle

## Git Workflow (Mandatory)

- Work only in branches of the GitHub repository.
- Do not do feature work directly on `main`.
- Merge the working branch back into the repository when the task is done.
