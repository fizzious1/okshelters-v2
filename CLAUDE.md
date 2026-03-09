# CLAUDE.md — ShelterNav Build Instructions

## ⚠️ PRIME DIRECTIVES

1. **PERFORMANCE IS THE #1 PRIORITY.** Every decision — language, data structure, serialization, rendering — must optimize for latency and throughput. If there's a faster way, use it.

2. **TRIPLE-CHECK EVERY IMPORT.** Before adding ANY dependency:
   - Verify it exists on the official package registry (crates.io, pkg.go.dev, Maven Central, CocoaPods/SPM)
   - Confirm the exact package name, author/org, and latest stable version
   - Check: Is this maintained? Last commit < 6 months? > 1000 GitHub stars?
   - **DO NOT hallucinate package names.** If unsure, search first.

3. **TRUSTED LIBRARIES ONLY.** Use dependencies from established orgs:
   - **Google**: protobuf, grpc, material-components, android-jetpack, benchmark
   - **Microsoft**: (none currently needed, but approved if applicable)
   - **Mozilla**: rust core ecosystem (serde, tokio, tonic)
   - **Apple**: SwiftUI, Core Location, MapKit (only as fallback)
   - **MapLibre**: maplibre-native, maplibre-gl (Linux Foundation backed)
   - **PostgreSQL**: PostGIS, libpq
   - **SQLite**: official amalgamation only
   - **Rust blessed**: rstar, criterion, rayon, crossbeam
   - **Go stdlib preferred.** Minimize external Go deps.
   - **REJECT**: Unknown GitHub repos, < 500 stars, single-maintainer hobby projects, anything with known CVEs

## Project Overview

Real-time shelter locator. Interactive map, instant navigation, offline-capable.

## Tech Stack

| Component | Language | Key Libs |
|---|---|---|
| Geo Query Service | Rust | tonic, rstar, serde, tokio, sqlx |
| API Gateway | Go | stdlib net/http, google.golang.org/grpc, groupcache |
| Client Core Lib | C + ASM | SQLite amalgamation, custom KD-tree, A* |
| iOS Shell | Swift (SwiftUI) | MapLibre Native iOS, CoreLocation |
| Android Shell | Kotlin (Compose) | MapLibre Native Android, Jetpack, Material 3 |
| Database | PostgreSQL 16 | PostGIS 3.4 |
| Wire Format | Protobuf 3 | google.protobuf |

## Architecture Rules

- All spatial queries go through in-memory R-tree (Rust), NOT direct PostGIS for reads
- PostGIS is source of truth; R-tree syncs via CDC
- Client always works offline with local SQLite + KD-tree
- Protobuf on all internal gRPC. JSON only at public REST boundary.
- Zero allocations in hot paths. Pre-allocate buffers.
- SIMD (AVX2/NEON) for batch Haversine. Scalar fallback required.

## Code Style

### Rust
- `#[inline(always)]` on hot-path functions
- No `unwrap()` in production paths — use `?` or explicit error handling
- `cargo clippy -- -D warnings` must pass
- Benchmarks via `criterion` for any function in the query path

### Go
- `go vet` and `staticcheck` must pass
- Context propagation on all handlers
- No global mutable state
- Structured logging via `slog` (stdlib)

### C
- C17 standard. `-Wall -Wextra -Werror -O3 -flto`
- No dynamic allocation in query path — stack or arena allocator
- All public functions prefixed `sn_` (namespace)
- Valgrind/ASan clean

### Assembly
- Restricted to: Haversine batch, heap operations, SIMD utilities
- Every ASM function MUST have an equivalent C/Rust fallback
- Documented with calling convention and register usage comments
- Tested against scalar reference implementation with fuzzing

### Swift / Kotlin
- Platform conventions (SwiftUI property wrappers, Compose state)
- Zero business logic — pure UI presentation + FFI bridge
- All data flows through ViewModels
- Animations: native only (Core Animation / Compose), no third-party animation libs

## Performance Contracts

| Metric | Budget | Enforced By |
|---|---|---|
| Nearest query (server) | < 5ms p99 | criterion bench in CI |
| API response E2E | < 100ms p99 | integration test |
| Client offline query | < 2ms | C benchmark in CI |
| Route calc (client) | < 10ms for 50km | C benchmark |
| Map frame time | < 12ms (60fps) | Instruments / GPU profiler |
| App cold start → map | < 1.5s | manual + automated UI test |

CI **blocks merge** if any perf contract regresses > 10%.

## Build Commands

```bash
# Rust geo service
cd geo-service && cargo build --release && cargo test && cargo bench

# Go gateway
cd gateway && go build ./... && go test ./... && go vet ./...

# C core lib (Linux/macOS)
cd libshelternav && mkdir build && cd build && cmake -DCMAKE_BUILD_TYPE=Release .. && make -j$(nproc) && ctest

# C cross-compile for Android
cmake -DCMAKE_TOOLCHAIN_FILE=$NDK/build/cmake/android.toolchain.cmake -DANDROID_ABI=arm64-v8a ..

# Protobuf codegen
cd proto && buf generate

# Full integration
docker-compose up --build
```

## File Ownership

```
geo-service/     → Rust engineer
gateway/         → Go engineer
libshelternav/   → C/Systems engineer
ios/             → iOS engineer
android/         → Android engineer
proto/           → Shared (breaking changes need all-team review)
migrations/      → Backend (reviewed by all)
```

## Security

- No secrets in code. Env vars or Vault.
- All network: TLS 1.3 minimum
- SQLite: WAL mode, no user-supplied SQL (parameterized only)
- JNI/FFI boundary: validate all inputs from managed code before passing to C
- Fuzz all C parsing functions (AFL++ / libFuzzer)

## When In Doubt

1. Is it faster? → Do it.
2. Is the dependency trustworthy? → Check the rules above. If still unsure, don't add it. Write it yourself.
3. Can it work offline? → It must.
4. Does it fit in 12ms? → If not, profile and optimize before shipping.
## Git Workflow (Mandatory)

- Work only in branches of the GitHub repository.
- Do not do feature work directly on `main`.
- Merge the working branch back into the repository when the task is done.
