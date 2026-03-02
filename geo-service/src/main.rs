use std::sync::Arc;
use tokio::sync::RwLock;
use tonic::transport::Server;
use tracing::{error, info};

use geo_service::grpc::pb::shelter_service_server::ShelterServiceServer;
use geo_service::grpc::ShelterServiceImpl;
use geo_service::haversine;
use geo_service::rtree::ShelterIndex;
use geo_service::sync;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // ---- Observability ----
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info")),
        )
        .init();

    // ---- Configuration (env vars, no secrets in code) ----
    let database_url = std::env::var("DATABASE_URL").map_err(|_| {
        std::io::Error::new(
            std::io::ErrorKind::InvalidInput,
            "DATABASE_URL environment variable is required",
        )
    })?;

    let grpc_port = match std::env::var("GRPC_PORT") {
        Ok(raw_port) => raw_port.parse::<u16>().map_err(|parse_error| {
            std::io::Error::new(
                std::io::ErrorKind::InvalidInput,
                format!("invalid GRPC_PORT `{raw_port}`: {parse_error}"),
            )
        })?,
        Err(_) => 9001,
    };
    let listen_addr = std::net::SocketAddr::from(([0, 0, 0, 0], grpc_port));

    // ---- Database pool ----
    let pool = sqlx::postgres::PgPoolOptions::new()
        .max_connections(8)
        .connect(&database_url)
        .await
        .map_err(|e| {
            error!("failed to connect to PostgreSQL: {e}");
            e
        })?;

    info!("connected to PostgreSQL");

    let kernel = haversine::detect_best_kernel();
    info!("distance kernel selected: {}", kernel.as_str());

    // ---- Spatial index ----
    let index = Arc::new(RwLock::new(ShelterIndex::new()));

    // Initial load from PostGIS into the in-memory R-tree.
    sync::start_sync(pool, Arc::clone(&index)).await.map_err(|e| {
        error!("initial sync failed: {e}");
        e
    })?;

    // ---- gRPC server ----
    let service = ShelterServiceImpl {
        index: Arc::clone(&index),
    };

    info!("geo-service listening on {listen_addr}");

    Server::builder()
        .add_service(ShelterServiceServer::new(service))
        .serve(listen_addr)
        .await?;

    Ok(())
}
