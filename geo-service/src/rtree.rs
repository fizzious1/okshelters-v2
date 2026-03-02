use std::cmp::Ordering;
use std::collections::HashMap;
use std::sync::Arc;

use rstar::{AABB, PointDistance, RTree, RTreeObject};

/// Open shelter status value from the database enum.
pub const STATUS_OPEN: i32 = 1;

/// A shelter stored inside the in-memory R-tree.
#[derive(Debug, Clone)]
pub struct ShelterPoint {
    /// Database primary key.
    pub id: i32,
    /// Display name, interned once at load time.
    pub name: Arc<str>,
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
    pub address: Arc<str>,
}

type Envelope = AABB<[f64; 2]>;

impl RTreeObject for ShelterPoint {
    type Envelope = Envelope;

    #[inline(always)]
    fn envelope(&self) -> Self::Envelope {
        AABB::from_point([self.lon, self.lat])
    }
}

impl PointDistance for ShelterPoint {
    #[inline(always)]
    fn distance_2(&self, point: &[f64; 2]) -> f64 {
        let dx = self.lon - point[0];
        let dy = self.lat - point[1];
        dx * dx + dy * dy
    }
}

/// Query controls for nearest-neighbor lookups.
#[derive(Clone, Copy)]
pub struct QueryConstraints<'a> {
    /// Maximum Haversine distance in meters.
    pub radius_m: f64,
    /// Maximum number of results to return.
    pub limit: usize,
    /// Optional shelter type filter; empty means "all types".
    pub allowed_types: &'a [i32],
    /// When true, only shelters with `STATUS_OPEN` are returned.
    pub open_only: bool,
}

/// Thread-safe in-memory shelter index.
pub struct ShelterIndex {
    tree: RTree<ShelterPoint>,
    by_id: HashMap<i32, ShelterPoint>,
}

impl ShelterIndex {
    /// Create an empty index.
    pub fn new() -> Self {
        Self {
            tree: RTree::new(),
            by_id: HashMap::new(),
        }
    }

    /// Bulk-load shelters into the R-tree using STR packing.
    pub fn from_bulk(points: Vec<ShelterPoint>) -> Self {
        let mut by_id = HashMap::with_capacity(points.len());
        for point in points {
            by_id.insert(point.id, point);
        }

        let mut tree_points = Vec::with_capacity(by_id.len());
        for point in by_id.values() {
            tree_points.push(point.clone());
        }

        Self {
            tree: RTree::bulk_load(tree_points),
            by_id,
        }
    }

    /// Replace the entire index using a fresh bulk-load set.
    pub fn replace_bulk(&mut self, points: Vec<ShelterPoint>) {
        *self = Self::from_bulk(points);
    }

    /// Insert or replace one shelter by `id`.
    pub fn upsert(&mut self, point: ShelterPoint) {
        if let Some(previous) = self.by_id.insert(point.id, point.clone()) {
            let _ = self.tree.remove(&previous);
        }
        self.tree.insert(point);
    }

    /// Alias for upsert semantics.
    pub fn insert(&mut self, point: ShelterPoint) {
        self.upsert(point);
    }

    /// Remove a shelter by `id`. Returns `true` when present.
    pub fn remove(&mut self, id: i32) -> bool {
        match self.by_id.remove(&id) {
            Some(previous) => {
                let _ = self.tree.remove(&previous);
                true
            }
            None => false,
        }
    }

    /// Find nearest shelters to (`lat`, `lon`) constrained by radius/type/status.
    #[inline(always)]
    pub fn find_nearest(
        &self,
        lat: f64,
        lon: f64,
        constraints: QueryConstraints<'_>,
    ) -> Vec<(ShelterPoint, f64)> {
        use crate::haversine::haversine;

        if constraints.limit == 0
            || constraints.radius_m <= 0.0
            || self.by_id.is_empty()
        {
            return Vec::new();
        }

        let query_point = [lon, lat];
        let tree_size = self.tree.size();
        let mut fetch = constraints
            .limit
            .saturating_mul(8)
            .max(128)
            .min(tree_size);

        let mut candidates =
            Vec::with_capacity(constraints.limit.min(fetch));

        loop {
            candidates.clear();

            for point in self.tree.nearest_neighbor_iter(&query_point).take(fetch) {
                if constraints.open_only && point.status != STATUS_OPEN {
                    continue;
                }

                if !matches_type_filter(point.shelter_type, constraints.allowed_types) {
                    continue;
                }

                let distance_m = haversine(lat, lon, point.lat, point.lon);
                if distance_m <= constraints.radius_m {
                    candidates.push((point.clone(), distance_m));
                }
            }

            if candidates.len() >= constraints.limit || fetch >= tree_size {
                break;
            }

            fetch = fetch.saturating_mul(2).min(tree_size);
        }

        candidates.sort_by(|left, right| {
            match left
                .1
                .partial_cmp(&right.1)
                .unwrap_or(Ordering::Equal)
            {
                Ordering::Equal => left.0.id.cmp(&right.0.id),
                non_equal => non_equal,
            }
        });
        candidates.truncate(constraints.limit);
        candidates
    }

    /// Number of shelters currently indexed.
    pub fn len(&self) -> usize {
        self.by_id.len()
    }

    /// Whether the index contains no shelters.
    pub fn is_empty(&self) -> bool {
        self.by_id.is_empty()
    }
}

#[inline(always)]
fn matches_type_filter(shelter_type: i32, allowed_types: &[i32]) -> bool {
    if allowed_types.is_empty() {
        return true;
    }

    if allowed_types.len() == 1 {
        return shelter_type == allowed_types[0];
    }

    allowed_types.contains(&shelter_type)
}

impl PartialEq for ShelterPoint {
    fn eq(&self, other: &Self) -> bool {
        self.id == other.id
    }
}

impl Eq for ShelterPoint {}

#[cfg(test)]
mod tests {
    use super::*;

    fn arc(value: &str) -> Arc<str> {
        Arc::from(value)
    }

    fn sample_shelters() -> Vec<ShelterPoint> {
        vec![
            ShelterPoint {
                id: 1,
                name: arc("Downtown Shelter"),
                lat: 35.4676,
                lon: -97.5164,
                shelter_type: 0,
                capacity: 100,
                occupancy: 42,
                status: STATUS_OPEN,
                address: arc("100 Main St"),
            },
            ShelterPoint {
                id: 2,
                name: arc("Midtown Haven"),
                lat: 35.4900,
                lon: -97.5200,
                shelter_type: 1,
                capacity: 50,
                occupancy: 10,
                status: STATUS_OPEN,
                address: arc("200 Oak Ave"),
            },
            ShelterPoint {
                id: 3,
                name: arc("Closed Shelter"),
                lat: 35.4800,
                lon: -97.5210,
                shelter_type: 1,
                capacity: 80,
                occupancy: 80,
                status: 0,
                address: arc("300 Pine Rd"),
            },
        ]
    }

    #[test]
    fn bulk_load_and_len() {
        let idx = ShelterIndex::from_bulk(sample_shelters());
        assert_eq!(idx.len(), 3);
    }

    #[test]
    fn find_nearest_filters_open_and_type() {
        let idx = ShelterIndex::from_bulk(sample_shelters());
        let allowed_types = [1];

        let results = idx.find_nearest(
            35.47,
            -97.52,
            QueryConstraints {
                radius_m: 10_000.0,
                limit: 10,
                allowed_types: &allowed_types,
                open_only: true,
            },
        );

        assert_eq!(results.len(), 1);
        assert_eq!(results[0].0.id, 2);
    }

    #[test]
    fn upsert_replaces_existing_by_id() {
        let mut idx = ShelterIndex::from_bulk(sample_shelters());
        idx.upsert(ShelterPoint {
            id: 2,
            name: arc("Midtown Haven Updated"),
            lat: 35.4902,
            lon: -97.5195,
            shelter_type: 1,
            capacity: 55,
            occupancy: 9,
            status: STATUS_OPEN,
            address: arc("200 Oak Ave"),
        });

        let results = idx.find_nearest(
            35.4902,
            -97.5195,
            QueryConstraints {
                radius_m: 500.0,
                limit: 10,
                allowed_types: &[],
                open_only: false,
            },
        );

        assert_eq!(results[0].0.id, 2);
        assert_eq!(&*results[0].0.name, "Midtown Haven Updated");
    }

    #[test]
    fn remove_by_id() {
        let mut idx = ShelterIndex::from_bulk(sample_shelters());
        assert!(idx.remove(2));
        assert_eq!(idx.len(), 2);
        assert!(!idx.remove(999));
    }
}
