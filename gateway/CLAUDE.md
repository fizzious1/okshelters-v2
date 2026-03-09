# CLAUDE.md - gateway/

## Scope

This file applies only to code under `gateway/`.
Do not define or modify behavior for other modules here.

## Purpose

HTTP gateway module:

- Request validation and transport mapping
- Middleware (auth, rate limit, cache, logging, recovery)
- gRPC client calls to geo-service

## Local Rules

1. Keep gateway thin; no business logic or DB access here.
2. Prefer Go stdlib for HTTP, JSON, context, and logging.
3. Keep handlers deterministic and testable.
4. Keep middleware order explicit in `main.go`.

## Current Local Structure

- `main.go`
- `handler/`
- `middleware/`
- `client/`
- generated `pb/`

## Out of Scope

- Postgres schema changes
- Rust R-tree implementation
- Mobile UI concerns

## Git Workflow (Mandatory)

- Work only in branches of the GitHub repository.
- Do not do feature work directly on `main`.
- Merge the working branch back into the repository when the task is done.
