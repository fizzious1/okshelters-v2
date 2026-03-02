/// Earth's mean radius in meters (WGS-84).
const EARTH_RADIUS_M: f64 = 6_371_008.8;

/// Compute the great-circle distance in meters between two (lat, lon) pairs
/// using the Haversine formula. Inputs are in **degrees**.
///
/// This is the scalar reference implementation. It is used as the fallback for
/// all architectures and as the ground-truth for testing the SIMD variants.
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
///
/// On x86-64 with AVX2, this will eventually use SIMD intrinsics. On AArch64
/// with NEON, likewise. For now every target falls through to the scalar loop.
/// The function signature is stable; only the body changes per architecture.
#[inline(always)]
pub fn haversine_batch(
    origin_lat: f64,
    origin_lon: f64,
    targets: &[(f64, f64)],
    out: &mut [f64],
) {
    debug_assert_eq!(
        targets.len(),
        out.len(),
        "targets and out slices must have equal length"
    );

    // ------- architecture-specific fast paths (stubs) --------

    #[cfg(target_arch = "x86_64")]
    {
        // TODO: AVX2 implementation using _mm256_* intrinsics.
        // Falls through to scalar below.
        let _ = (); // silence unused-cfg lint
    }

    #[cfg(target_arch = "aarch64")]
    {
        // TODO: NEON implementation using vfmaq_f64 / vsinq etc.
        // Falls through to scalar below.
        let _ = (); // silence unused-cfg lint
    }

    // ------- scalar fallback (always compiled) --------
    for (i, &(lat, lon)) in targets.iter().enumerate() {
        out[i] = haversine(origin_lat, origin_lon, lat, lon);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Sanity check: distance between identical points should be zero.
    #[test]
    fn same_point_is_zero() {
        let d = haversine(35.0, -97.0, 35.0, -97.0);
        assert!(d.abs() < 1e-6, "expected ~0, got {d}");
    }

    /// Known distance: Oklahoma City (35.4676, -97.5164) to Tulsa
    /// (36.1540, -95.9928) is roughly 168 km.
    #[test]
    fn okc_to_tulsa() {
        let d = haversine(35.4676, -97.5164, 36.1540, -95.9928);
        let km = d / 1000.0;
        assert!(
            (km - 168.0).abs() < 5.0,
            "expected ~168 km, got {km:.1} km"
        );
    }

    /// Batch variant must agree with scalar for every element.
    #[test]
    fn batch_matches_scalar() {
        let origin = (35.4676, -97.5164);
        let targets = vec![
            (36.1540, -95.9928),
            (35.0, -97.0),
            (34.0, -96.0),
        ];
        let mut out = vec![0.0; targets.len()];
        haversine_batch(origin.0, origin.1, &targets, &mut out);

        for (i, &(lat, lon)) in targets.iter().enumerate() {
            let expected = haversine(origin.0, origin.1, lat, lon);
            assert!(
                (out[i] - expected).abs() < 1e-6,
                "mismatch at index {i}: batch={} scalar={expected}",
                out[i]
            );
        }
    }
}
