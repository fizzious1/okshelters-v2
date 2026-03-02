use std::sync::Arc;
use std::time::Duration;

use sqlx::{PgPool, Row};
use tokio::sync::RwLock;
use tracing::{info, warn};

use crate::rtree::{ShelterIndex, ShelterPoint};

const POLL_INTERVAL: Duration = Duration::from_secs(5);
const DELTA_BATCH_LIMIT: i64 = 5_000;

/// Bootstrap the in-memory index from PostgreSQL and start a poll-based
/// delta updater.
///
/// This poll loop is an interim CDC strategy. It tracks `updated_at`
/// mutations for inserts/updates; hard deletes require a tombstone stream.
pub async fn start_sync(
    pool: PgPool,
    index: Arc<RwLock<ShelterIndex>>,
) -> Result<(), sqlx::Error> {
    info!("sync: starting initial load from PostgreSQL");

    let (snapshot_points, mut watermark_epoch) = load_snapshot(&pool).await?;
    let snapshot_count = snapshot_points.len();

    {
        let mut write_guard = index.write().await;
        write_guard.replace_bulk(snapshot_points);
    }

    info!("sync: loaded {snapshot_count} shelters into R-tree");
    warn!("sync: poll mode does not observe hard deletes without tombstones");

    let poll_pool = pool.clone();
    let poll_index = Arc::clone(&index);
    let _ = tokio::spawn(async move {
        info!("sync: delta poll loop started (interval={}s)", POLL_INTERVAL.as_secs());
        loop {
            match load_deltas(&poll_pool, watermark_epoch).await {
                Ok((updates, next_watermark)) => {
                    if !updates.is_empty() {
                        let mut write_guard = poll_index.write().await;
                        for point in updates {
                            write_guard.upsert(point);
                        }
                        info!(
                            "sync: applied delta batch, index_size={}",
                            write_guard.len(),
                        );
                    }
                    watermark_epoch = watermark_epoch.max(next_watermark);
                }
                Err(error) => {
                    warn!("sync: delta poll failed: {error}");
                }
            }

            tokio::time::sleep(POLL_INTERVAL).await;
        }
    });

    Ok(())
}

async fn load_snapshot(pool: &PgPool) -> Result<(Vec<ShelterPoint>, f64), sqlx::Error> {
    let rows = sqlx::query(
        r#"
        SELECT
            id,
            name,
            ST_Y(location::geometry) AS lat,
            ST_X(location::geometry) AS lon,
            type AS shelter_type,
            capacity,
            COALESCE(occupancy, 0) AS occupancy,
            COALESCE(status, 1) AS status,
            COALESCE(address, '') AS address,
            COALESCE(EXTRACT(EPOCH FROM updated_at), 0)::double precision AS updated_epoch
        FROM shelters
        "#,
    )
    .fetch_all(pool)
    .await?;

    let mut points = Vec::with_capacity(rows.len());
    let mut watermark_epoch = 0.0_f64;
    for row in rows {
        let (point, row_epoch) = decode_row(row)?;
        watermark_epoch = watermark_epoch.max(row_epoch);
        points.push(point);
    }

    Ok((points, watermark_epoch))
}

async fn load_deltas(
    pool: &PgPool,
    since_epoch: f64,
) -> Result<(Vec<ShelterPoint>, f64), sqlx::Error> {
    let rows = sqlx::query(
        r#"
        SELECT
            id,
            name,
            ST_Y(location::geometry) AS lat,
            ST_X(location::geometry) AS lon,
            type AS shelter_type,
            capacity,
            COALESCE(occupancy, 0) AS occupancy,
            COALESCE(status, 1) AS status,
            COALESCE(address, '') AS address,
            COALESCE(EXTRACT(EPOCH FROM updated_at), 0)::double precision AS updated_epoch
        FROM shelters
        WHERE COALESCE(EXTRACT(EPOCH FROM updated_at), 0)::double precision > $1
        ORDER BY updated_at ASC
        LIMIT $2
        "#,
    )
    .bind(since_epoch)
    .bind(DELTA_BATCH_LIMIT)
    .fetch_all(pool)
    .await?;

    let mut points = Vec::with_capacity(rows.len());
    let mut watermark_epoch = since_epoch;
    for row in rows {
        let (point, row_epoch) = decode_row(row)?;
        watermark_epoch = watermark_epoch.max(row_epoch);
        points.push(point);
    }

    Ok((points, watermark_epoch))
}

fn decode_row(row: sqlx::postgres::PgRow) -> Result<(ShelterPoint, f64), sqlx::Error> {
    let id: i32 = row.try_get("id")?;
    let name: String = row.try_get("name")?;
    let lat: f64 = row.try_get("lat")?;
    let lon: f64 = row.try_get("lon")?;
    let shelter_type: i16 = row.try_get("shelter_type")?;
    let capacity: i16 = row.try_get("capacity")?;
    let occupancy: i16 = row.try_get("occupancy")?;
    let status: i16 = row.try_get("status")?;
    let address: String = row.try_get("address")?;
    let updated_epoch: f64 = row.try_get("updated_epoch")?;

    Ok((
        ShelterPoint {
            id,
            name: Arc::from(name.into_boxed_str()),
            lat,
            lon,
            shelter_type: i32::from(shelter_type),
            capacity: i32::from(capacity),
            occupancy: i32::from(occupancy),
            status: i32::from(status),
            address: Arc::from(address.into_boxed_str()),
        },
        updated_epoch,
    ))
}
