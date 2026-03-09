# ShelterNav

Real-time shelter locator with interactive maps, instant navigation, and full offline support. Designed for emergency scenarios where sub-second access to shelter information is critical.

## Architecture

```
Mobile (iOS / Android)
  ├── Local SQLite + KD-tree (offline)
  └── REST ──→ API Gateway (Go, :8080)
                  └── gRPC ──→ Geo Service (Rust, :9001)
                                  └── R-tree (in-memory) ← CDC ← PostgreSQL + PostGIS
```

| Component | Language | Role |
|---|---|---|
| **Geo Service** | Rust | Spatial queries via in-memory R-tree, gRPC server |
| **API Gateway** | Go | REST API, auth, rate limiting, response caching |
| **Client Core** | C + ASM | Offline KD-tree queries, A* routing, SIMD Haversine |
| **iOS App** | Swift / SwiftUI | MapLibre map, shelter UI, FFI bridge to C lib |
| **Android App** | Kotlin / Compose | MapLibre map, shelter UI, JNI bridge to C lib |
| **Database** | PostgreSQL 16 | PostGIS 3.4, source of truth for shelter data |

## Quick Start

The fastest way to run the full backend stack:

```bash
docker-compose up --build
```

This starts PostgreSQL (with PostGIS + migrations), the Rust geo-service, and the Go gateway. Once running:

- **Gateway API**: http://localhost:8080
- **Health check**: http://localhost:8080/healthz
- **Geo Service gRPC**: localhost:9001

## Web UI

ShelterNav includes a browser-based map interface served directly by the Go gateway. It uses MapLibre GL JS for the interactive map, matches the native app dark theme, and falls back to built-in demo data when the geo-service is unavailable.

**With Docker (full stack):**

```bash
docker-compose up --build
```

Then open http://localhost:8080 in your browser.

**Local dev (gateway only, demo mode):**

```bash
cd gateway && go run .
```

Then open http://localhost:8080. Without the geo-service running, the UI automatically enters demo mode and displays sample Oklahoma City shelters with mock navigation routes.

The web UI features a dark-themed interactive map centered on Oklahoma City, a frosted-glass search bar, a draggable bottom sheet with shelter cards showing real-time capacity bars, floating action buttons for geolocation and nearest-shelter lookup, and a full navigation overlay with route visualization and turn-by-turn maneuvers.

## Prerequisites

For local development outside Docker:

- **Rust** 1.77+
- **Go** 1.22+
- **CMake** 3.20+ and a C17 compiler
- **PostgreSQL** 16 with PostGIS 3.4
- **Buf** v2 (protobuf codegen)
- **Xcode** (iOS) / **Android NDK** (Android)

## Building Components Individually

### Geo Service (Rust)

```bash
cd geo-service
cargo build --release
cargo test
cargo bench
```

Requires `DATABASE_URL` env var pointing to a PostGIS-enabled database.

### API Gateway (Go)

```bash
cd gateway
go build ./...
go test ./...
go vet ./...
```

Environment variables:
- `LISTEN_ADDR` — HTTP listen address (default `:8080`)
- `GEO_SERVICE_ADDR` — gRPC address of geo-service (default `localhost:9001`)

### Client Core Library (C)

```bash
cd libshelternav
mkdir -p build && cd build
cmake -DCMAKE_BUILD_TYPE=Release ..
make -j$(nproc)
ctest
```

Cross-compile for Android:

```bash
cmake -DCMAKE_TOOLCHAIN_FILE=$NDK/build/cmake/android.toolchain.cmake \
      -DANDROID_ABI=arm64-v8a ..
```

### Protobuf Codegen

```bash
cd proto
buf generate
```

## API Endpoints

### `GET /v1/shelters/nearest`

Find shelters near a location.

| Param | Type | Default | Description |
|---|---|---|---|
| `lat` | float | required | Latitude |
| `lon` | float | required | Longitude |
| `radius` | int | 5000 | Search radius in meters |
| `limit` | int | 10 | Max results (up to 100) |

### `GET /v1/route`

Compute a route between two points.

| Param | Type | Description |
|---|---|---|
| `start_lat` | float | Origin latitude |
| `start_lon` | float | Origin longitude |
| `end_lat` | float | Destination latitude |
| `end_lon` | float | Destination longitude |

### `GET /healthz`

Returns `{"status":"ok"}` when the gateway is running.

## Performance Targets

| Metric | Budget |
|---|---|
| Nearest query (server) | < 5ms p99 |
| API response E2E | < 100ms p99 |
| Client offline query | < 2ms |
| Route calc (50km) | < 10ms |
| Map frame time | < 12ms (60fps) |
| App cold start to map | < 1.5s |

## Project Structure

```
proto/            Protobuf service definitions (Buf v2)
geo-service/      Rust gRPC spatial query service
gateway/          Go HTTP gateway + middleware
gateway/web/      Browser-based map UI (HTML/CSS/JS, no build step)
libshelternav/    C core library + SIMD assembly
ios/              SwiftUI iOS app
android/          Jetpack Compose Android app
migrations/       PostgreSQL schema migrations
```

## License

All rights reserved.
