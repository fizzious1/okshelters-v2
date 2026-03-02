use std::sync::Arc;
use tokio::sync::RwLock;
use tracing::{info, warn};

use crate::rtree::{ShelterIndex, ShelterPoint};

/// Row shape returned by the initial-load query.
///
/// `sqlx::FromRow` is not used here because the `location` column is a
/// PostGIS geography that we decompose via `ST_Y` / `ST_X` in the SQL.
#[allow(dead_code)]
struct ShelterRow {
    id: i32,
    name: String,
    lat: f64,
    lon: f64,
    shelter_type: i16,
    capacity: i16,
    occupancy: i16,
    status: i16,
    address: Option<String>,
}

/// Perform the initial full load of all shelters from PostgreSQL into the
/// in-memory R-tree, then (in a future iteration) subscribe to a CDC stream
/// so the index stays up-to-date.
///
/// # Errors
///
/// Returns a `sqlx::Error` if the database query fails.
pub async fn start_sync(
    pool: sqlx::PgPool,
    index: Arc<RwLock<ShelterIndex>>,
) -> Result<(), sqlx::Error> {
    info!("sync: starting initial load from PostgreSQL");

    let rows = sqlx::query_as!(
        ShelterRow,
        r#"
        SELECT
            id,
            name,
            ST_Y(location::geometry) AS "lat!",
            ST_X(location::geometry) AS "lon!",
            type           AS "shelter_type!",
            capacity       AS "capacity!",
            COALESCE(occupancy, 0) AS "occupancy!",
            COALESCE(status, 1)    AS "status!",
            address
        FROM shelters
        "#
    )
    .fetch_all(&pool)
    .await?;

    let count = rows.len();
    let points: Vec<ShelterPoint> = rows
        .into_iter()
        .map(|r| ShelterPoint {
            id: r.id,
            name: r.name,
            lat: r.lat,
            lon: r.lon,
            shelter_type: i32::from(r.shelter_type),
            capacity: i32::from(r.capacity),
            occupancy: i32::from(r.occupancy),
            status: i32::from(r.status),
            address: r.address.unwrap_or_default(),
        })
        .collect();

    // Bulk-load is significantly faster than repeated inserts because rstar
    // uses STR packing to build a near-optimal tree in O(n log n).
    {
        let mut idx = index.write().await;
        *idx = ShelterIndex::from_bulk(points);
    }

    info!("sync: loaded {count} shelters into R-tree");

    // TODO: subscribe to PostgreSQL logical replication / CDC stream here.
    // For now, the initial snapshot is the only data source.
    warn!("sync: CDC not yet implemented; index will not reflect live updates");

    Ok(())
}
