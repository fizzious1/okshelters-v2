# Gateway Design

## Scope

This design covers only `gateway/`.

## Goal

Provide a thin HTTP edge that validates requests, applies middleware controls, and forwards geospatial calls to gRPC upstream.

## Responsibilities

- HTTP route handling
- request validation and JSON response shaping
- auth, rate limiting, response caching, logging, recovery
- gRPC client calls via generated protobuf bindings

## Current State

- Core routes are implemented (`nearest`, `route`, `healthz`, `readyz`).
- Middleware chain is implemented and test coverage exists for key middleware.
- Route behavior depends on upstream `GetRoute` implementation.

## Local Constraints

- Keep gateway thin (no database logic here).
- Keep middleware deterministic and low overhead.
- Keep route parsing and response envelopes consistent.
