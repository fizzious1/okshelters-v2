/**
 * haversine.c -- Great-circle distance computation.
 *
 * Scalar implementation.  sn_haversine_batch is the hot-path entry point
 * that will be replaced by AVX2 (x86_64) or NEON (arm64) SIMD variants
 * on supported platforms.
 */

#include "shelternav.h"

#include <math.h>
#include <stddef.h>

/* Earth mean radius in metres (WGS-84 derived). */
#define EARTH_RADIUS_M 6371000.0

/* Pi as a compile-time constant (portable across strict C17 environments). */
#define SN_PI 3.14159265358979323846

/* Degrees to radians. */
#define DEG_TO_RAD (SN_PI / 180.0)

/* -------------------------------------------------------------------
 * Scalar Haversine
 * ------------------------------------------------------------------- */

#if defined(__GNUC__) || defined(__clang__)
__attribute__((always_inline))
#endif
static inline double haversine_impl(double lat1, double lon1,
                                    double lat2, double lon2)
{
    const double lat1_rad = lat1 * DEG_TO_RAD;
    const double lon1_rad = lon1 * DEG_TO_RAD;
    const double lat2_rad = lat2 * DEG_TO_RAD;
    const double lon2_rad = lon2 * DEG_TO_RAD;

    const double half_dlat = (lat2_rad - lat1_rad) * 0.5;
    const double half_dlon = (lon2_rad - lon1_rad) * 0.5;

    const double sin_dlat = sin(half_dlat);
    const double sin_dlon = sin(half_dlon);

    double a = (sin_dlat * sin_dlat)
             + (cos(lat1_rad) * cos(lat2_rad) * sin_dlon * sin_dlon);
    if (a < 0.0) {
        a = 0.0;
    } else if (a > 1.0) {
        a = 1.0;
    }

    const double c = 2.0 * atan2(sqrt(a), sqrt(1.0 - a));

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

    const double origin_lat_rad = origin_lat * DEG_TO_RAD;
    const double origin_lon_rad = origin_lon * DEG_TO_RAD;
    const double origin_cos = cos(origin_lat_rad);

    for (int i = 0; i < n; i++) {
        const double lat2_rad = lats[i] * DEG_TO_RAD;
        const double lon2_rad = lons[i] * DEG_TO_RAD;

        const double half_dlat = (lat2_rad - origin_lat_rad) * 0.5;
        const double half_dlon = (lon2_rad - origin_lon_rad) * 0.5;

        const double sin_dlat = sin(half_dlat);
        const double sin_dlon = sin(half_dlon);

        double a = (sin_dlat * sin_dlat)
                 + (origin_cos * cos(lat2_rad) * sin_dlon * sin_dlon);
        if (a < 0.0) {
            a = 0.0;
        } else if (a > 1.0) {
            a = 1.0;
        }

        const double c = 2.0 * atan2(sqrt(a), sqrt(1.0 - a));
        distances_out[i] = EARTH_RADIUS_M * c;
    }
}
