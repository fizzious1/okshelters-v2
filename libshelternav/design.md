# libshelternav Design

## Scope

This design covers only `libshelternav/`.

## Goal

Provide a portable native C core library for mobile clients.

## Responsibilities

- KD-tree nearest lookup
- Haversine scalar operations (with SIMD extension points)
- route API surface and supporting structures
- local SQLite sync ingestion into native structures

## Current State

- Public C API exists in `include/shelternav.h`.
- KD-tree and sync implementations exist.
- A* routing and SIMD/ASM paths are currently stubbed.

## Local Constraints

- Preserve ABI stability.
- Keep memory handling explicit and safe.
- Keep architecture-specific code isolated under `asm/`.
