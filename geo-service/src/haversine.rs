use std::sync::OnceLock;

/// Earth's mean radius in meters (WGS-84).
const EARTH_RADIUS_M: f64 = 6_371_008.8;

/// Batch distance kernel selected at runtime.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DistanceKernel {
    /// Portable scalar fallback.
    Scalar,
    /// x86_64 AVX2 implementation.
    Avx2,
    /// AArch64 NEON implementation.
    Neon,
}

impl DistanceKernel {
    /// Human-readable kernel name for logs and metrics.
    pub const fn as_str(self) -> &'static str {
        match self {
            DistanceKernel::Scalar => "scalar",
            DistanceKernel::Avx2 => "avx2",
            DistanceKernel::Neon => "neon",
        }
    }
}

/// Detect the fastest available batch-distance kernel once at startup.
pub fn detect_best_kernel() -> DistanceKernel {
    static KERNEL: OnceLock<DistanceKernel> = OnceLock::new();

    *KERNEL.get_or_init(|| {
        #[cfg(target_arch = "x86_64")]
        {
            if is_x86_feature_detected!("avx2") {
                return DistanceKernel::Avx2;
            }
        }

        #[cfg(target_arch = "aarch64")]
        {
            return DistanceKernel::Neon;
        }

        DistanceKernel::Scalar
    })
}

/// Compute the great-circle distance in meters between two (lat, lon) pairs
/// using the Haversine formula. Inputs are in degrees.
#[inline(always)]
pub fn haversine(lat1: f64, lon1: f64, lat2: f64, lon2: f64) -> f64 {
    let lat1_r = lat1.to_radians();
    let lat2_r = lat2.to_radians();
    let dlat = (lat2 - lat1).to_radians();
    let dlon = (lon2 - lon1).to_radians();

    let a = (dlat * 0.5).sin().powi(2)
        + lat1_r.cos() * lat2_r.cos() * (dlon * 0.5).sin().powi(2);

    EARTH_RADIUS_M * 2.0 * a.sqrt().asin()
}

/// Batch Haversine: compute distances from a single origin to many targets.
#[inline(always)]
pub fn haversine_batch(
    origin_lat: f64,
    origin_lon: f64,
    targets: &[(f64, f64)],
    out: &mut [f64],
) {
    haversine_batch_with_kernel(
        detect_best_kernel(),
        origin_lat,
        origin_lon,
        targets,
        out,
    );
}

/// Batch Haversine using a caller-selected kernel.
#[inline(always)]
pub fn haversine_batch_with_kernel(
    kernel: DistanceKernel,
    origin_lat: f64,
    origin_lon: f64,
    targets: &[(f64, f64)],
    out: &mut [f64],
) {
    debug_assert_eq!(
        targets.len(),
        out.len(),
        "targets and out slices must have equal length",
    );

    match kernel {
        DistanceKernel::Scalar => {
            scalar_batch(origin_lat, origin_lon, targets, out);
        }
        DistanceKernel::Avx2 => {
            // SAFETY: current AVX2 implementation falls back to scalar and
            // does not use target-specific intrinsics yet.
            unsafe { haversine_batch_avx2(origin_lat, origin_lon, targets, out) };
        }
        DistanceKernel::Neon => {
            // SAFETY: current NEON implementation falls back to scalar and
            // does not use target-specific intrinsics yet.
            unsafe { haversine_batch_neon(origin_lat, origin_lon, targets, out) };
        }
    }
}

#[inline(always)]
fn scalar_batch(
    origin_lat: f64,
    origin_lon: f64,
    targets: &[(f64, f64)],
    out: &mut [f64],
) {
    for (index, &(lat, lon)) in targets.iter().enumerate() {
        out[index] = haversine(origin_lat, origin_lon, lat, lon);
    }
}

#[inline(always)]
unsafe fn haversine_batch_avx2(
    origin_lat: f64,
    origin_lon: f64,
    targets: &[(f64, f64)],
    out: &mut [f64],
) {
    scalar_batch(origin_lat, origin_lon, targets, out);
}

#[inline(always)]
unsafe fn haversine_batch_neon(
    origin_lat: f64,
    origin_lon: f64,
    targets: &[(f64, f64)],
    out: &mut [f64],
) {
    scalar_batch(origin_lat, origin_lon, targets, out);
}

#[cfg(test)]
mod tests {
    use super::*;
    use proptest::collection::vec;
    use proptest::prelude::*;

    #[test]
    fn same_point_is_zero() {
        let distance_m = haversine(35.0, -97.0, 35.0, -97.0);
        assert!(distance_m.abs() < 1e-6, "expected ~0, got {distance_m}");
    }

    #[test]
    fn okc_to_tulsa() {
        let distance_m = haversine(35.4676, -97.5164, 36.1540, -95.9928);
        let distance_km = distance_m / 1000.0;
        assert!(
            (distance_km - 168.0).abs() < 5.0,
            "expected ~168 km, got {distance_km:.1} km",
        );
    }

    #[test]
    fn batch_matches_scalar_for_known_points() {
        let origin = (35.4676, -97.5164);
        let targets = vec![
            (36.1540, -95.9928),
            (35.0, -97.0),
            (34.0, -96.0),
        ];
        let mut out = vec![0.0; targets.len()];

        haversine_batch(origin.0, origin.1, &targets, &mut out);

        for (index, &(lat, lon)) in targets.iter().enumerate() {
            let expected = haversine(origin.0, origin.1, lat, lon);
            assert!(
                (out[index] - expected).abs() < 1e-9,
                "mismatch at index {index}: batch={} scalar={expected}",
                out[index],
            );
        }
    }

    proptest! {
        #[test]
        fn batch_matches_scalar_property(
            origin_lat in -90.0f64..90.0f64,
            origin_lon in -180.0f64..180.0f64,
            targets in vec((-90.0f64..90.0f64, -180.0f64..180.0f64), 1..64),
        ) {
            let mut out = vec![0.0; targets.len()];

            haversine_batch(origin_lat, origin_lon, &targets, &mut out);

            for (index, &(lat, lon)) in targets.iter().enumerate() {
                let expected = haversine(origin_lat, origin_lon, lat, lon);
                let delta = (out[index] - expected).abs();
                prop_assert!(
                    delta < 1e-9,
                    "index={index} delta={delta} batch={} scalar={expected}",
                    out[index],
                );
            }
        }
    }
}
