# CLAUDE.md - proto/

## Scope

This file applies only to code under `proto/`.
Do not define or modify behavior for other modules here.

## Purpose

Shared protobuf contract module:

- `.proto` schema definitions
- buf lint/breaking policy
- code generation configuration

## Local Rules

1. Treat field numbers as stable once published.
2. Prefer additive schema changes over breaking changes.
3. Keep package naming/versioning explicit.
4. Keep buf config synchronized with generation targets.

## Current Local Structure

- `shelter.proto`
- `buf.yaml`
- `buf.gen.yaml`

## Out of Scope

- runtime service logic
- SQL migrations
- mobile UI implementation

## Git Workflow (Mandatory)

- Work only in branches of the GitHub repository.
- Do not do feature work directly on `main`.
- Merge the working branch back into the repository when the task is done.
