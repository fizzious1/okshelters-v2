# Migrations Design

## Scope

This design covers only `migrations/`.

## Goal

Maintain ordered SQL schema migrations for the ShelterNav database.

## Responsibilities

- define table/index creation and evolution SQL
- preserve rollout-safe migration ordering
- keep schema aligned with service query requirements

## Current State

- `001_init.sql` creates PostGIS extension, `shelters` table, and core indexes.

## Local Constraints

- keep migrations append-only and versioned
- avoid destructive changes unless explicitly required
- keep column semantics consistent with service contracts
