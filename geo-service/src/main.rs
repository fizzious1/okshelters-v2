mod grpc;
mod haversine;
mod rtree;
mod sync;

use std::sync::Arc;
use tokio::sync::RwLock;
use tonic::transport::Server;
use tracing::{error, info};

use grpc::pb::shelter_service_server::ShelterServiceServer;
use grpc::ShelterServiceImpl;
use rtree::ShelterIndex;

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
        "DATABASE_URL environment variable is required"
    })?;

    let listen_addr = std::env::var("LISTEN_ADDR")
        .unwrap_or_else(|_| "0.0.0.0:9001".to_string())
        .parse()?;

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
