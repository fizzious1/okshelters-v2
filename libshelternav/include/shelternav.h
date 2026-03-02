/**
 * shelternav.h -- Public API for the ShelterNav client core library.
 *
 * All public symbols are prefixed with sn_.
 * Error convention: 0 = success, negative = error code.
 */

#ifndef SHELTERNAV_H
#define SHELTERNAV_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

/* -------------------------------------------------------------------
 * Error codes
 * ------------------------------------------------------------------- */
#define SN_OK                    0
#define SN_ERR_INVALID_ARG      -1
#define SN_ERR_OUT_OF_MEMORY    -2
#define SN_ERR_DB_OPEN          -3
#define SN_ERR_DB_QUERY         -4
#define SN_ERR_DB_CLOSE         -5
#define SN_ERR_NOT_IMPLEMENTED  -6
#define SN_ERR_NO_PATH          -7

/* -------------------------------------------------------------------
 * Data structures
 * ------------------------------------------------------------------- */

/** Geographic coordinate pair. */
typedef struct {
    double lat;
    double lon;
} SN_LatLon;

/** Shelter record — matches the on-disk SQLite schema. */
typedef struct {
    int32_t  id;
    double   lat;
    double   lon;
    uint8_t  status;
    uint16_t capacity;
    char     name[128];
    char     address[256];
} SN_Shelter;

/** Opaque KD-tree handle. */
typedef struct SN_KDNode SN_KDNode;

/** Opaque road-graph handle. */
typedef struct SN_RoadGraph SN_RoadGraph;

/** Single turn-by-turn maneuver in a route. */
typedef struct {
    SN_LatLon point;
    char      instruction[128];
    double    distance_m;
} SN_Maneuver;

/* -------------------------------------------------------------------
 * KD-tree API
 * ------------------------------------------------------------------- */

/**
 * Create an empty KD-tree.
 * Returns NULL-rooted tree (pointer that can be passed to insert).
 * Caller must eventually call sn_kdtree_destroy().
 */
SN_KDNode *sn_kdtree_create(void);

/**
 * Insert a shelter into the KD-tree.
 * Returns SN_OK on success, negative error code on failure.
 */
int sn_kdtree_insert(SN_KDNode **tree, const SN_Shelter *shelter);

/**
 * Destroy the KD-tree and free all nodes.
 */
void sn_kdtree_destroy(SN_KDNode *tree);

/**
 * Find shelters within radius_m metres of (lat, lon).
 * Results are written to out[] (up to max_results entries).
 * Returns the number of results found (>= 0), or negative error code.
 */
int sn_find_nearest(const SN_KDNode *tree,
                    double lat, double lon,
                    double radius_m,
                    SN_Shelter out[],
                    int max_results);

/* -------------------------------------------------------------------
 * Haversine distance
 * ------------------------------------------------------------------- */

/**
 * Compute the great-circle distance in metres between two points.
 */
double sn_haversine(double lat1, double lon1, double lat2, double lon2);

/**
 * Batch Haversine: compute distances from a single origin to N points.
 * Scalar fallback; will be replaced by SIMD on supported platforms.
 *
 * lats/lons: arrays of length n.
 * distances_out: pre-allocated array of length n (metres).
 */
void sn_haversine_batch(double origin_lat, double origin_lon,
                        const double lats[], const double lons[],
                        double distances_out[], int n);

/* -------------------------------------------------------------------
 * A* routing
 * ------------------------------------------------------------------- */

/**
 * Compute shortest path via A*.
 *
 * path_out:  caller-allocated array of SN_Maneuver.
 * path_len:  in: capacity of path_out; out: number of maneuvers written.
 *
 * Returns SN_OK on success, SN_ERR_NO_PATH if unreachable,
 * or SN_ERR_NOT_IMPLEMENTED while the stub is in place.
 */
int sn_route_astar(const SN_RoadGraph *graph,
                   SN_LatLon start,
                   SN_LatLon end,
                   SN_Maneuver path_out[],
                   int *path_len);

/* -------------------------------------------------------------------
 * SQLite sync
 * ------------------------------------------------------------------- */

/**
 * Open local SQLite database at path (WAL mode).
 * Only one database may be open at a time (module-level state).
 * Returns SN_OK on success.
 */
int sn_db_open(const char *path);

/**
 * Close the currently open database.
 */
int sn_db_close(void);

/**
 * Read all shelters from the local database and insert into tree.
 * Returns number of rows synced (>= 0), or negative error code.
 */
int sn_db_sync(SN_KDNode **tree);

#ifdef __cplusplus
}
#endif

#endif /* SHELTERNAV_H */
