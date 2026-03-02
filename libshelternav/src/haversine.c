/**
 * haversine.c -- Great-circle distance computation.
 *
 * Scalar implementation.  sn_haversine_batch is the hot-path entry point
 * that will be replaced by AVX2 (x86_64) or NEON (arm64) SIMD variants
 * on supported platforms.
 */

#include "shelternav.h"

#include <math.h>

/* Earth mean radius in metres (WGS-84 derived). */
#define EARTH_RADIUS_M 6371000.0

/* Degrees to radians. */
#define DEG_TO_RAD (M_PI / 180.0)

/* -------------------------------------------------------------------
 * Scalar Haversine
 * ------------------------------------------------------------------- */

#if defined(__GNUC__) || defined(__clang__)
__attribute__((always_inline))
#endif
static inline double haversine_impl(double lat1, double lon1,
                                    double lat2, double lon2)
{
    double dlat = (lat2 - lat1) * DEG_TO_RAD;
    double dlon = (lon2 - lon1) * DEG_TO_RAD;

    double rlat1 = lat1 * DEG_TO_RAD;
    double rlat2 = lat2 * DEG_TO_RAD;

    double a = sin(dlat * 0.5) * sin(dlat * 0.5)
             + cos(rlat1) * cos(rlat2)
             * sin(dlon * 0.5) * sin(dlon * 0.5);

    double c = 2.0 * atan2(sqrt(a), sqrt(1.0 - a));

    return EARTH_RADIUS_M * c;
}

double sn_haversine(double lat1, double lon1, double lat2, double lon2)
{
    return haversine_impl(lat1, lon1, lat2, lon2);
}

/* -------------------------------------------------------------------
 * Batch Haversine (scalar fallback)
 *
 * Computes distances from a single origin to N destination points.
 * distances_out must be pre-allocated by the caller (no allocation here).
 * ------------------------------------------------------------------- */

void sn_haversine_batch(double origin_lat, double origin_lon,
                        const double lats[], const double lons[],
                        double distances_out[], int n)
{
    if (lats == NULL || lons == NULL || distances_out == NULL || n <= 0) {
        return;
    }

    /*
     * Scalar loop.  To be replaced at link time by:
     *   - _sn_haversine_batch_avx2  (x86_64)
     *   - _sn_haversine_batch_neon  (arm64)
     * when SIMD implementations are ready.
     */
    for (int i = 0; i < n; i++) {
        distances_out[i] = haversine_impl(origin_lat, origin_lon,
                                          lats[i], lons[i]);
    }
}
