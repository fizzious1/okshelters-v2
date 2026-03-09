# Proto Design

## Scope

This design covers only `proto/`.

## Goal

Maintain shared protobuf contracts and generation config.

## Responsibilities

- service and message schemas in `shelter.proto`
- lint and breaking-change policy in buf config
- generation configuration for downstream modules

## Current State

- Nearest and route contracts are defined.
- buf config is present and targets Go output for `gateway/pb`.

## Local Constraints

- Keep field numbers stable.
- Prefer additive schema evolution.
- Keep package and versioning explicit for compatibility.
