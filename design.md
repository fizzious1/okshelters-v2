# ShelterNav — Design Doc & Technical Spec

## 1. Overview

Real-time shelter locator with interactive map and instant navigation. Sub-100ms query responses. Offline-capable. Minimal memory footprint.

---

## 2. Architecture

```
┌─────────────┐     gRPC/Proto     ┌──────────────────┐
│  Mobile App  │ ◄───────────────► │   API Gateway     │ (Go)
│  (Rust/C)    │                   │   :8080           │
└─────┬───────┘                   └────────┬──────────┘
      │                                     │
      │ Local SQLite                        │ gRPC internal
      │ (offline cache)                     ▼
      │                           ┌──────────────────┐
      │                           │  Geo Query Svc    │ (Rust)
      │                           │  R-tree in-memory │
      │                           └────────┬──────────┘
      │                                     │
      │                                     ▼
      │                           ┌──────────────────┐
      │                           │  PostgreSQL +     │
      │                           │  PostGIS          │
      │                           └──────────────────┘
```

---

## 3. Component Breakdown

### 3.1 Geo Query Service — Rust

Core hot path. All spatial queries run here.

- **In-memory R-tree** via `rstar` crate for nearest-neighbor queries. O(log n).
- **Data**: Shelter ID, lat/lon, capacity, type, status. Loaded from Postgres on boot, synced via CDC (Debezium).
- **Endpoint**: `FindNearest(lat, lon, radius_m, limit) → []Shelter`
- **SIMD-accelerated Haversine** distance calc via inline assembly (x86 AVX2 / ARM NEON):

```rust
// Hot path: batch distance calc
#[cfg(target_arch = "x86_64")]
unsafe fn haversine_batch_avx2(
    user_lat: f64, user_lon: f64,
    lats: &[f64], lons: &[f64], out: &mut [f64]
) {
    // AVX2 intrinsics: _mm256_* for 4x f64 parallel haversine
    // Processes 4 shelter coords per cycle
}
```

- **Fallback**: Pure Rust scalar Haversine when SIMD unavailable.
- **Binds**: `0.0.0.0:9001`, gRPC via `tonic`.

### 3.2 API Gateway — Go

Thin, concurrent request router.

- Go `net/http` + `grpc-go` client to Geo Query Service.
- Rate limiting (token bucket), auth (JWT), request validation.
- Response caching: 1s TTL on identical geo queries (LRU, `groupcache`).
- Protobuf serialization on the wire, JSON for client REST fallback.
- Health checks, Prometheus metrics.

### 3.3 Mobile Client — C core + platform shell

Performance-critical logic in C, compiled as shared lib (`.so`/`.dylib`).

**C Core Library (`libshelternav`):**

```c
// Spatial index for offline cached shelters
typedef struct {
    int32_t id;
    double lat, lon;
    uint8_t status; // 0=closed, 1=open, 2=full
    uint16_t capacity;
} Shelter;

// KD-tree for local nearest-neighbor
typedef struct KDNode {
    Shelter shelter;
    struct KDNode *left, *right;
} KDNode;

// Returns sorted nearest shelters. Stack-allocated result buffer.
int find_nearest(KDNode *root, double lat, double lon,
                 double radius_m, Shelter *out, int max_results);

// A* pathfinding on cached road graph (for offline nav)
int route_astar(const RoadGraph *g, LatLon start, LatLon end,
                LatLon *path_out, int *path_len);
```

- **Offline SQLite** cache: last-known shelter data + simplified road graph tiles.
- **Platform shell**: Swift (iOS) / Kotlin (Android) for UI, map rendering (MapLibre Native), calls into `libshelternav` via FFI.
- **Map tiles**: Vector tiles, pre-cached for user's region.

### 3.4 Navigation Engine — C with ASM hotspots

Embedded in `libshelternav`. For turn-by-turn after shelter selection.

- Contraction Hierarchies preprocessed on server, transferred as compact binary blob.
- Client-side A* on CH graph. ~5ms query for 50km radius.
- Assembly optimized priority queue (heap operations) for x86/ARM:

```asm
; ARM64 - optimized heap sift-down for A* open set
; Avoids branch misprediction via conditional select
heap_sift_down:
    ldr x2, [x0, x1, lsl #3]    ; load current
    lsl x3, x1, #1
    add x3, x3, #1              ; left child
    ldr x4, [x0, x3, lsl #3]
    csel x5, x3, x1, lt         ; branchless min-select
    ...
```

---

## 4. Data Model

### PostgreSQL + PostGIS

```sql
CREATE TABLE shelters (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    location    GEOGRAPHY(POINT, 4326) NOT NULL,
    address     TEXT,
    type        SMALLINT NOT NULL,  -- 0=emergency, 1=overnight, 2=long-term
    capacity    SMALLINT NOT NULL,
    occupancy   SMALLINT DEFAULT 0,
    status      SMALLINT DEFAULT 1, -- 0=closed, 1=open, 2=full
    phone       TEXT,
    updated_at  TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_shelters_geo ON shelters USING GIST(location);
CREATE INDEX idx_shelters_status ON shelters (status) WHERE status = 1;
```

### Protobuf (wire format)

```protobuf
syntax = "proto3";

service ShelterService {
  rpc FindNearest(NearestRequest) returns (NearestResponse);
  rpc GetRoute(RouteRequest) returns (RouteResponse);
}

message NearestRequest {
  double lat = 1;
  double lon = 2;
  uint32 radius_m = 3;   // default 5000
  uint32 limit = 4;      // default 10
  repeated int32 types = 5;
}

message ShelterInfo {
  int32 id = 1;
  string name = 2;
  double lat = 3;
  double lon = 4;
  int32 type = 5;
  int32 capacity = 6;
  int32 occupancy = 7;
  int32 status = 8;
  string address = 9;
  double distance_m = 10;
}

message NearestResponse {
  repeated ShelterInfo shelters = 1;
}
```

---

## 5. Performance Targets

| Metric | Target |
|---|---|
| Nearest-shelter query (server) | < 5ms p99 |
| End-to-end API response | < 100ms p99 |
| Route computation (client, 50km) | < 10ms |
| Offline nearest query (client) | < 2ms |
| App cold start to map render | < 1.5s |
| Memory (client C core) | < 8MB |
| R-tree capacity (server) | 1M shelters, < 200MB RAM |

---

## 6. Key Design Decisions

1. **Rust for geo service** — memory safety without GC pauses. Predictable latency.
2. **C for client core** — minimal overhead, universal FFI, deterministic memory.
3. **Go for gateway** — goroutine concurrency is ideal for I/O-bound routing.
4. **Assembly for hot loops** — Haversine batches and heap ops are tight loops called millions of times. 2-4x speedup over compiled code measured in benchmarks.
5. **In-memory R-tree over PostGIS for reads** — PostGIS is source of truth; R-tree serves live traffic. Eliminates DB round-trip.
6. **Protobuf over JSON** — ~10x smaller payloads, zero-copy deserialization possible.
7. **Offline-first** — Client always has a usable local dataset. Network enhances, never gates.

---

## 7. Build & Deploy

- **Rust service**: `cargo build --release`, Docker scratch image. ~12MB binary.
- **Go gateway**: `go build -ldflags="-s -w"`, Docker scratch. ~8MB binary.
- **C lib**: CMake cross-compile via NDK (Android) / Xcode toolchain (iOS). `-O3 -march=native -flto`.
- **ASM**: NASM (x86), integrated in CMake. ARM asm via `.S` files with GNU as.
- **CI**: GitHub Actions. Bench regression tests gate merges (criterion.rs for Rust, Google Benchmark for C).
- **Deploy**: Kubernetes. Geo service scales horizontally (stateless after R-tree load). Gateway auto-scales on RPS.

---

## 8. Sync & Updates

- Server pushes shelter status changes via **SSE** (Server-Sent Events) to connected clients.
- Client syncs full dataset on first launch, then delta updates (last `updated_at` timestamp).
- Stale client data (>24h offline) triggers full re-sync on reconnect.

---

## 9. File Tree

```
shelternav/
├── proto/                  # Shared protobuf definitions
│   └── shelter.proto
├── geo-service/            # Rust
│   ├── src/
│   │   ├── main.rs
│   │   ├── rtree.rs        # R-tree index
│   │   ├── haversine.rs    # Scalar + SIMD distance
│   │   ├── grpc.rs         # tonic server
│   │   └── sync.rs         # Postgres CDC consumer
│   └── Cargo.toml
├── gateway/                # Go
│   ├── main.go
│   ├── handler/
│   ├── middleware/          # auth, rate-limit, cache
│   └── go.mod
├── libshelternav/          # C + ASM
│   ├── include/
│   │   └── shelternav.h
│   ├── src/
│   │   ├── kdtree.c
│   │   ├── astar.c
│   │   ├── haversine.c
│   │   └── sync.c
│   ├── asm/
│   │   ├── heap_arm64.S
│   │   ├── heap_x86.asm
│   │   ├── haversine_avx2.asm
│   │   └── haversine_neon.S
│   └── CMakeLists.txt
├── ios/                    # Swift UI shell
│   ├── ShelterNav/
│   │   ├── App.swift
│   │   ├── Views/
│   │   │   ├── MapView.swift         # MapLibre wrapper
│   │   │   ├── ShelterSheet.swift    # Bottom sheet detail
│   │   │   ├── NavigationView.swift  # Turn-by-turn overlay
│   │   │   ├── SearchBar.swift
│   │   │   └── Components/
│   │   │       ├── ShelterPin.swift
│   │   │       ├── PulseRing.swift   # User location pulse
│   │   │       ├── RoutePolyline.swift
│   │   │       └── StatusBadge.swift
│   │   ├── ViewModels/
│   │   │   ├── MapViewModel.swift
│   │   │   └── NavigationViewModel.swift
│   │   ├── Bridge/
│   │   │   └── ShelterNavBridge.swift  # C FFI wrapper
│   │   ├── Theme/
│   │   │   ├── Colors.swift
│   │   │   ├── Typography.swift
│   │   │   └── Haptics.swift
│   │   └── Assets.xcassets
│   └── ShelterNav.xcodeproj
├── android/                # Kotlin UI shell
│   ├── app/src/main/
│   │   ├── java/.../shelternav/
│   │   │   ├── MainActivity.kt
│   │   │   ├── ui/
│   │   │   │   ├── map/MapScreen.kt
│   │   │   │   ├── sheet/ShelterDetailSheet.kt
│   │   │   │   ├── nav/NavigationOverlay.kt
│   │   │   │   ├── search/SearchBar.kt
│   │   │   │   └── components/
│   │   │   │       ├── ShelterMarker.kt
│   │   │   │       ├── PulseRing.kt
│   │   │   │       ├── RoutePolyline.kt
│   │   │   │       └── StatusBadge.kt
│   │   │   ├── viewmodel/
│   │   │   │   ├── MapViewModel.kt
│   │   │   │   └── NavigationViewModel.kt
│   │   │   ├── bridge/
│   │   │   │   └── ShelterNavJNI.kt   # C FFI via JNI
│   │   │   └── theme/
│   │   │       ├── Color.kt
│   │   │       ├── Type.kt
│   │   │       └── Theme.kt
│   │   └── res/
│   └── build.gradle.kts
├── migrations/             # SQL
│   └── 001_init.sql
└── docker-compose.yml
```

---

## 10. UI & Frontend Design

### 10.1 Design Language

**Dark-first, high contrast, emergency-grade readability.**

| Token | Value | Usage |
|---|---|---|
| `bg-primary` | `#0F1117` | Map background tint, app chrome |
| `bg-surface` | `#1A1D27` | Bottom sheets, cards |
| `bg-surface-raised` | `#242836` | Elevated cards, search bar |
| `accent-green` | `#34D399` | Open shelters, available capacity |
| `accent-amber` | `#FBBF24` | Near-full shelters, warnings |
| `accent-red` | `#F87171` | Full/closed shelters |
| `accent-blue` | `#60A5FA` | Route line, user location pulse |
| `text-primary` | `#F1F5F9` | Headings, shelter names |
| `text-secondary` | `#94A3B8` | Addresses, metadata |
| `font` | SF Pro (iOS) / Google Sans (Android) | System-native, no custom font load |

Light mode supported but secondary. System toggle respected.

### 10.2 Map Configuration — MapLibre Native

Custom style for visual appeal and performance:

```json
{
  "id": "shelternav-dark",
  "sources": {
    "openmaptiles": {
      "type": "vector",
      "url": "https://tiles.shelternav.app/v1/tiles.json"
    }
  },
  "layers": [
    { "id": "background", "type": "background",
      "paint": { "background-color": "#0F1117" } },
    { "id": "water", "type": "fill",
      "paint": { "fill-color": "#151926", "fill-opacity": 0.8 } },
    { "id": "roads-minor", "type": "line",
      "paint": { "line-color": "#1E2230", "line-width": 1 } },
    { "id": "roads-major", "type": "line",
      "paint": { "line-color": "#2A3040", "line-width": 2 } },
    { "id": "buildings", "type": "fill-extrusion",
      "paint": {
        "fill-extrusion-color": "#1A1F2E",
        "fill-extrusion-height": ["get", "height"],
        "fill-extrusion-opacity": 0.6
      }
    }
  ]
}
```

**Key map features:**
- 3D building extrusions at zoom > 14 (depth, realism)
- Desaturated base map so shelter pins pop visually
- Smooth camera animations: `flyTo` with `bearing` and `pitch` adjustments
- Terrain/hillshade layer for outdoor context

### 10.3 Screen Flow

```
┌─────────────────────────────────────────────┐
│  ┌─────────────────────────────────────┐    │
│  │  🔍 Search shelters...        ☰     │    │  ← Floating search bar
│  └─────────────────────────────────────┘    │    Frosted glass bg
│                                             │
│              ┌───┐                          │
│              │ 🟢│  Shelter pin             │  ← Map fills entire
│              └───┘     (color = status)     │    screen edge-to-edge
│                                             │
│         ┌───┐          ┌───┐                │
│         │ 🟡│          │ 🔴│                │
│         └───┘          └───┘                │
│                                             │
│                  ◉ ← User location          │  ← Blue dot + animated
│               (  ·  )  pulse ring           │    ripple ring
│                                             │
│  ┌──┐                                       │
│  │📍│ Re-center          ┌──┐               │  ← Floating action
│  └──┘                    │⚡│ Nearest        │    buttons, bottom-right
│                          └──┘               │
│─────────────────────────────────────────────│
│ ┌─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┐  │
│   ━━━  (drag handle)                       │  ← Bottom sheet (peek)
│ │                                         │  │
│   ▸ 3 shelters nearby · Closest 0.4 km      │
│ │                                         │  │
│ └─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┘  │
└─────────────────────────────────────────────┘
```

**Bottom sheet — expanded (swipe up):**

```
┌─────────────────────────────────────────────┐
│  ━━━                                        │
│                                             │
│  ┌─────────────────────────────────────┐    │
│  │ 🟢 Hope Community Shelter     0.4km │    │  ← Card: status dot,
│  │    123 Main St · 34/50 beds         │    │    name, distance,
│  │    ┌──────────┐  ┌──────────┐       │    │    capacity bar
│  │    │ Navigate  │  │  Call 📞 │       │    │
│  │    └──────────┘  └──────────┘       │    │
│  │    ▓▓▓▓▓▓▓▓▓▓░░░░  68% full        │    │  ← Capacity bar
│  └─────────────────────────────────────┘    │    (green→amber→red)
│                                             │
│  ┌─────────────────────────────────────┐    │
│  │ 🟡 Riverside Safe Haven      1.2km  │    │
│  │    456 River Rd · 47/50 beds        │    │
│  │    ▓▓▓▓▓▓▓▓▓▓▓▓▓▓░  94% full       │    │
│  └─────────────────────────────────────┘    │
│                                             │
└─────────────────────────────────────────────┘
```

**Navigation mode (after tapping "Navigate"):**

```
┌─────────────────────────────────────────────┐
│                                             │
│  ┌─────────────────────────────────────┐    │
│  │  ← Back    NAVIGATING    ETA 6 min  │    │  ← Top bar overlay
│  └─────────────────────────────────────┘    │
│                                             │
│           Map auto-follows user             │
│           Route: glowing blue polyline      │  ← Animated dashed
│           with animated direction flow      │    gradient line
│                                             │
│                  ◉ ─ ─ ─ ─ ─ 🏠             │
│                (you)    (shelter)            │
│                                             │
│─────────────────────────────────────────────│
│  ┌─────────────────────────────────────┐    │
│  │  ↱ Turn right on Oak Ave            │    │  ← Next maneuver card
│  │    in 200m                          │    │    Large, glanceable
│  └─────────────────────────────────────┘    │
│  ┌─────────────────────────────────────┐    │
│  │  Hope Community Shelter             │    │
│  │  🟢 Open · 0.4 km · 6 min          │    │
│  └─────────────────────────────────────┘    │
└─────────────────────────────────────────────┘
```

### 10.4 Animations & Micro-interactions

All animations native (Core Animation / Compose), no JS bridge.

| Interaction | Animation | Duration |
|---|---|---|
| Shelter pin appear | Scale from 0 + fade in, staggered | 200ms |
| Pin tap → sheet expand | Spring animation, map camera adjusts | 350ms |
| User location pulse | Repeating scale + opacity ring | 2s loop |
| Route draw | Polyline traces path progressively | 800ms |
| Navigate tap | Camera `flyTo` with pitch 45° + bearing align | 600ms |
| Sheet swipe | Interruptible spring with velocity tracking | gesture-driven |
| Status change (live) | Pin color crossfade + subtle bounce | 300ms |
| Capacity bar fill | Animated width with easing | 400ms |

### 10.5 Shelter Pin Design

Custom rendered, not bitmap — crisp at all zoom levels.

```
     ╭───╮
     │ ● │   ← Circle: fill color = status (green/amber/red)
     ╰─┬─╯      Border: 2px white for contrast
       ▽        Shadow: soft 4px blur, 30% black

  Clustered (zoom < 13):
     ╭─────╮
     │  12  │   ← Count badge, pill shape
     ╰─────╯      Size scales with count
```

### 10.6 Platform Implementation

**iOS** — SwiftUI + MapLibre Native iOS SDK
- `UIViewRepresentable` wrapper for MapLibre `MGLMapView`
- Bottom sheet: custom `UISheetPresentationController` with detents (peek: 120pt, half, full)
- Haptic feedback on pin tap (`UIImpactFeedbackGenerator`, `.medium`)
- Calls `libshelternav` via Swift C interop (bridging header)

**Android** — Jetpack Compose + MapLibre Native Android SDK
- `AndroidView` composable wrapping `MapView`
- Bottom sheet: `ModalBottomSheet` with Material 3
- Haptic: `HapticFeedbackType.LongPress` on pin tap
- Calls `libshelternav` via JNI (`external fun findNearest(...)`)

### 10.7 Frontend Performance Rules

1. **Map pins**: Use MapLibre symbol layers, NOT individual views/markers. Handles 10k+ pins at 60fps.
2. **Clustering**: Server-side via `supercluster` algorithm in Rust geo service. Client just renders.
3. **Tile caching**: Aggressive — 200MB offline tile budget. Pre-cache user's metro area.
4. **Image assets**: Zero bitmaps. All icons are vector (SF Symbols / Material Icons) or programmatic.
5. **Sheet rendering**: Lazy list for shelter cards. Only visible + 2 buffer cards in memory.
6. **Frame budget**: All UI work < 12ms per frame (targeting 16.6ms for 60fps, 4ms headroom).
7. **Main thread**: Zero network or DB calls. All via `async/await` dispatched to background.