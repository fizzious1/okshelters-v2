use rstar::{primitives::GeomWithData, RTree, RTreeObject, AABB};
use serde::{Deserialize, Serialize};

/// A shelter stored inside the R-tree.
///
/// We use `GeomWithData` from rstar with a 2-D point (lon, lat) as the
/// geometry and the full shelter metadata as the associated payload. This
/// avoids a second lookup after the spatial query.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ShelterPoint {
    /// Database primary key.
    pub id: i32,
    pub name: String,
    /// Latitude in degrees (WGS-84).
    pub lat: f64,
    /// Longitude in degrees (WGS-84).
    pub lon: f64,
    /// 0 = emergency, 1 = overnight, 2 = long-term.
    pub shelter_type: i32,
    pub capacity: i32,
    pub occupancy: i32,
    /// 0 = closed, 1 = open, 2 = full.
    pub status: i32,
    pub address: String,
}

// ---------------------------------------------------------------------------
// rstar integration
// ---------------------------------------------------------------------------

/// The envelope type rstar uses for 2-D indexing.
type Envelope = AABB<[f64; 2]>;

impl RTreeObject for ShelterPoint {
    type Envelope = Envelope;

    fn envelope(&self) -> Self::Envelope {
        // rstar works in Cartesian space.  For small search radii the
        // distortion from treating (lon, lat) as Cartesian is acceptable;
        // final ordering is done by true Haversine distance.
        AABB::from_point([self.lon, self.lat])
    }
}

impl rstar::PointDistance for ShelterPoint {
    fn distance_2(&self, point: &[f64; 2]) -> f64 {
        let dx = self.lon - point[0];
        let dy = self.lat - point[1];
        dx * dx + dy * dy
    }
}

// ---------------------------------------------------------------------------
// Index wrapper
// ---------------------------------------------------------------------------

/// Thread-safe wrapper around an `RTree<ShelterPoint>`.
///
/// The index lives behind `Arc<RwLock<ShelterIndex>>` so readers never block
/// each other while the CDC sync task holds a write lock only during bulk
/// mutations.
pub struct ShelterIndex {
    tree: RTree<ShelterPoint>,
}

impl ShelterIndex {
    /// Create an empty index.
    pub fn new() -> Self {
        Self {
            tree: RTree::new(),
        }
    }

    /// Bulk-load from a pre-existing vector. Much faster than repeated inserts
    /// because rstar can use STR packing.
    pub fn from_bulk(points: Vec<ShelterPoint>) -> Self {
        Self {
            tree: RTree::bulk_load(points),
        }
    }

    /// Insert a single shelter into the index.
    pub fn insert(&mut self, point: ShelterPoint) {
        self.tree.insert(point);
    }

    /// Remove a shelter by matching on `id`. Returns `true` if found.
    pub fn remove(&mut self, id: i32) -> bool {
        // rstar requires the exact object for removal. We locate it first via
        // a linear scan on the (small-ish, < 1M) dataset. In the future this
        // can be backed by a HashMap<id, ShelterPoint> side index.
        let maybe = self
            .tree
            .iter()
            .find(|p| p.id == id)
            .cloned();

        if let Some(point) = maybe {
            self.tree.remove(&point);
            true
        } else {
            false
        }
    }

    /// Find the nearest shelters within `radius_m` meters of (`lat`, `lon`),
    /// returning at most `limit` results sorted by ascending Haversine
    /// distance.
    ///
    /// Hot path -- this is the primary read query.
    #[inline(always)]
    pub fn find_nearest(
        &self,
        lat: f64,
        lon: f64,
        radius_m: f64,
        limit: usize,
    ) -> Vec<(ShelterPoint, f64)> {
        use crate::haversine::haversine;

        // rstar::nearest_neighbor_iter produces candidates in approximate
        // Cartesian-distance order. We pull more than `limit` to account for
        // projection distortion, compute true Haversine, filter by radius,
        // sort, and truncate.
        let query_point = [lon, lat];

        // Over-fetch factor: at mid-latitudes the Cartesian approximation is
        // decent, but we still pull 4x to be safe.
        let fetch = limit.saturating_mul(4).max(64);

        let mut results: Vec<(ShelterPoint, f64)> = self
            .tree
            .nearest_neighbor_iter(&query_point)
            .take(fetch)
            .filter_map(|p| {
                let dist = haversine(lat, lon, p.lat, p.lon);
                if dist <= radius_m {
                    Some((p.clone(), dist))
                } else {
                    None
                }
            })
            .collect();

        // Sort by true distance ascending.
        results.sort_by(|a, b| a.1.partial_cmp(&b.1).unwrap_or(std::cmp::Ordering::Equal));
        results.truncate(limit);
        results
    }

    /// Total number of shelters in the index.
    pub fn len(&self) -> usize {
        self.tree.size()
    }

    /// Whether the index is empty.
    pub fn is_empty(&self) -> bool {
        self.tree.size() == 0
    }
}

// Needed so that `remove` can match objects inside the tree.
impl PartialEq for ShelterPoint {
    fn eq(&self, other: &Self) -> bool {
        self.id == other.id
    }
}

impl Eq for ShelterPoint {}

#[cfg(test)]
mod tests {
    use super::*;

    fn sample_shelters() -> Vec<ShelterPoint> {
        vec![
            ShelterPoint {
                id: 1,
                name: "Downtown Shelter".into(),
                lat: 35.4676,
                lon: -97.5164,
                shelter_type: 0,
                capacity: 100,
                occupancy: 42,
                status: 1,
                address: "100 Main St".into(),
            },
            ShelterPoint {
                id: 2,
                name: "Midtown Haven".into(),
                lat: 35.4900,
                lon: -97.5200,
                shelter_type: 1,
                capacity: 50,
                occupancy: 10,
                status: 1,
                address: "200 Oak Ave".into(),
            },
            ShelterPoint {
                id: 3,
                name: "Far Away Shelter".into(),
                lat: 36.1540,
                lon: -95.9928,
                shelter_type: 2,
                capacity: 200,
                occupancy: 150,
                status: 1,
                address: "300 Tulsa Blvd".into(),
            },
        ]
    }

    #[test]
    fn bulk_load_and_len() {
        let idx = ShelterIndex::from_bulk(sample_shelters());
        assert_eq!(idx.len(), 3);
    }

    #[test]
    fn find_nearest_within_radius() {
        let idx = ShelterIndex::from_bulk(sample_shelters());
        // Search near downtown OKC, 10 km radius -- should find the two OKC
        // shelters but NOT Tulsa.
        let results = idx.find_nearest(35.47, -97.52, 10_000.0, 10);
        assert_eq!(results.len(), 2);
        // Closest should be Downtown Shelter.
        assert_eq!(results[0].0.id, 1);
    }

    #[test]
    fn remove_by_id() {
        let mut idx = ShelterIndex::from_bulk(sample_shelters());
        assert!(idx.remove(2));
        assert_eq!(idx.len(), 2);
        assert!(!idx.remove(999));
    }
}
