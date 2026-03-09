# Geo-Service Design

## Scope

This design covers only `geo-service/`.

## Goal

Serve low-latency geospatial queries over gRPC using in-memory indexing.

## Responsibilities

- Initial load from Postgres
- maintain in-memory shelter index
- serve `FindNearest` requests over gRPC
- keep query path fast and predictable

## Current State

- Service startup, DB load, and nearest query path are implemented.
- `GetRoute` is currently unimplemented.
- Sync is initial-load only; live CDC updates are not yet implemented.

## Local Constraints

- Keep hot-path allocations minimal.
- Keep service API aligned with `proto/` definitions.
- Keep index logic and sync logic encapsulated in this module.
