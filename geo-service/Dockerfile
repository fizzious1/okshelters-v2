# ---------- build stage ----------
FROM rust:1.77-slim AS builder

RUN apt-get update && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy proto definitions (needed by build.rs / tonic-build).
COPY proto/ /app/proto/

# Cache dependency builds: copy manifests first, build a dummy main, then
# overlay the real source. This keeps the expensive cargo build layer
# cached when only application code changes.
COPY geo-service/Cargo.toml geo-service/build.rs /app/geo-service/

WORKDIR /app/geo-service

# Create a dummy main so cargo can resolve and compile dependencies.
RUN mkdir src && echo "fn main() {}" > src/main.rs && cargo build --release 2>/dev/null || true

# Now copy real source and rebuild.
COPY geo-service/src /app/geo-service/src
RUN touch src/main.rs && cargo build --release

# ---------- runtime stage ----------
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/geo-service/target/release/geo-service /usr/local/bin/geo-service

EXPOSE 9001

ENTRYPOINT ["geo-service"]
