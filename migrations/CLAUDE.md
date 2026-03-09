# CLAUDE.md - migrations/

## Scope

This file applies only to code under `migrations/`.
Do not define or modify behavior for other modules here.

## Purpose

Database schema migration module:

- versioned SQL migrations
- Postgres/PostGIS schema evolution for shelter data

## Local Rules

1. Migrations must be ordered and append-only.
2. Avoid destructive schema rewrites unless explicitly requested.
3. Keep indexes aligned with query patterns.
4. Keep enum/column semantics consistent with service contracts.

## Current Local Structure

- `001_init.sql`

## Out of Scope

- gateway handler logic
- geo-service runtime logic
- mobile feature implementation

## Git Workflow (Mandatory)

- Work only in branches of the GitHub repository.
- Do not do feature work directly on `main`.
- Merge the working branch back into the repository when the task is done.
