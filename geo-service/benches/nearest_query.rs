use std::sync::Arc;

use criterion::{black_box, criterion_group, criterion_main, Criterion};
use geo_service::rtree::{QueryConstraints, ShelterIndex, ShelterPoint, STATUS_OPEN};

fn build_index(count: usize) -> ShelterIndex {
    let mut points = Vec::with_capacity(count);

    for id in 0..count {
        let lat = 34.0000 + ((id % 1_000) as f64) * 0.0001;
        let lon = -118.5000 + ((id / 1_000) as f64) * 0.0001;
        let shelter_type = (id % 3) as i32;
        let occupancy = (id % 150) as i32;
        let capacity = 200_i32;

        points.push(ShelterPoint {
            id: id as i32 + 1,
            name: Arc::from(format!("Shelter-{id}").into_boxed_str()),
            lat,
            lon,
            shelter_type,
            capacity,
            occupancy,
            status: STATUS_OPEN,
            address: Arc::from("Benchmark Ave"),
        });
    }

    ShelterIndex::from_bulk(points)
}

fn bench_nearest_query(c: &mut Criterion) {
    let index = build_index(100_000);
    let constraints = QueryConstraints {
        radius_m: 5_000.0,
        limit: 10,
        allowed_types: &[],
        open_only: true,
    };

    c.bench_function("nearest_query_100k_limit10", |b| {
        b.iter(|| {
            let results = index.find_nearest(
                black_box(34.0522),
                black_box(-118.2437),
                black_box(constraints),
            );
            black_box(results);
        });
    });
}

criterion_group!(benches, bench_nearest_query);
criterion_main!(benches);
