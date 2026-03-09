# CLAUDE.md - geo-service/

## Scope

This file applies only to code under `geo-service/`.
Do not define or modify behavior for other modules here.

## Purpose

Rust geospatial service:

- in-memory spatial index
- nearest query gRPC endpoint
- sync load from Postgres

## Local Rules

1. Keep hot paths allocation-aware and low latency.
2. Keep gRPC contracts aligned with `proto/`.
3. Keep sync/index concerns within this module.
4. Keep observability in place (tracing and structured logs).

## Current Local Structure

- `src/main.rs`
- `src/grpc.rs`
- `src/rtree.rs`
- `src/haversine.rs`
- `src/sync.rs`
- `Cargo.toml`, `build.rs`, `Dockerfile`

## Out of Scope

- HTTP gateway middleware behavior
- iOS/Android UI behavior
- C client-library internals

## Git Workflow (Mandatory)

- Work only in branches of the GitHub repository.
- Do not do feature work directly on `main`.
- Merge the working branch back into the repository when the task is done.
