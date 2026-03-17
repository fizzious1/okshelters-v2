/**
 * kdtree.c -- KD-tree spatial index for shelters.
 *
 * Alternates split axis between latitude (axis 0) and longitude (axis 1)
 * at each level of the tree.
 *
 * TODO: Replace malloc with a pre-allocated arena allocator to eliminate
 *       dynamic allocation in the build/insert path.
 */

#include "shelternav.h"

#include <math.h>
#include <stdlib.h>

/* -------------------------------------------------------------------
 * Internal node representation
 * ------------------------------------------------------------------- */

struct SN_KDNode {
    SN_Shelter  shelter;
    SN_KDNode  *left;
    SN_KDNode  *right;
    int         split_axis;   /* 0 = lat, 1 = lon */
};

/* -------------------------------------------------------------------
 * Helpers
 * ------------------------------------------------------------------- */

/** Return the split-axis value for a query point. */
static inline double point_axis_val(double lat, double lon, int axis)
{
    return (axis == 0) ? lat : lon;
}

/** Allocate and initialise one KD-tree node. */
static SN_KDNode *node_create(const SN_Shelter *shelter, int split_axis)
{
    SN_KDNode *node = (SN_KDNode *)malloc(sizeof(*node));
    if (node == NULL) {
        return NULL;
    }

    node->shelter = *shelter;
    node->left = NULL;
    node->right = NULL;
    node->split_axis = split_axis;
    return node;
}

/* -------------------------------------------------------------------
 * Stack-allocated result collector (used during search)
 * ------------------------------------------------------------------- */

typedef struct {
    SN_Shelter *buf;
    int         capacity;
    int         count;
} ResultBuf;

static void result_buf_init(ResultBuf *rb, SN_Shelter *buf, int capacity)
{
    rb->buf      = buf;
    rb->capacity = capacity;
    rb->count    = 0;
}

static int result_buf_push(ResultBuf *rb, const SN_Shelter *s)
{
    if (rb->count >= rb->capacity) {
        return -1; /* full */
    }
    rb->buf[rb->count] = *s;
    rb->count++;
    return 0;
}

/* -------------------------------------------------------------------
 * Public API: create / destroy
 * ------------------------------------------------------------------- */

SN_KDNode *sn_kdtree_create(void)
{
    /* Empty tree is represented as a NULL root pointer. */
    return NULL;
}

void sn_kdtree_destroy(SN_KDNode *tree)
{
    if (tree == NULL) {
        return;
    }
    sn_kdtree_destroy(tree->left);
    sn_kdtree_destroy(tree->right);
    /* TODO: arena-free instead of individual free */
    free(tree);
}

/* -------------------------------------------------------------------
 * Insert (iterative, alternating axis)
 * ------------------------------------------------------------------- */

int sn_kdtree_insert(SN_KDNode **tree, const SN_Shelter *shelter)
{
    if (tree == NULL || shelter == NULL) {
        return SN_ERR_INVALID_ARG;
    }

    if (*tree == NULL) {
        *tree = node_create(shelter, 0);
        return (*tree == NULL) ? SN_ERR_OUT_OF_MEMORY : SN_OK;
    }

    SN_KDNode *parent = NULL;
    SN_KDNode *cursor = *tree;
    SN_KDNode **insert_slot = NULL;
    int depth = 0;

    while (cursor != NULL) {
        parent = cursor;
        insert_slot = NULL;

        const int axis = cursor->split_axis;
        const double node_val = (axis == 0)
            ? cursor->shelter.lat
            : cursor->shelter.lon;
        const double ins_val = point_axis_val(shelter->lat, shelter->lon, axis);

        if (ins_val < node_val) {
            insert_slot = &cursor->left;
            cursor = cursor->left;
        } else {
            insert_slot = &cursor->right;
            cursor = cursor->right;
        }

        depth++;
    }

    SN_KDNode *new_node = node_create(shelter, depth & 1);
    if (new_node == NULL) {
        return SN_ERR_OUT_OF_MEMORY;
    }

    if (parent == NULL || insert_slot == NULL) {
        free(new_node);
        return SN_ERR_INVALID_ARG;
    }

    *insert_slot = new_node;
    return SN_OK;
}

/* -------------------------------------------------------------------
 * Nearest-neighbor search with distance pruning
 * ------------------------------------------------------------------- */

/**
 * Conservative degree-to-metre lower bounds used for axis-aligned pruning.
 * Distances are always confirmed with sn_haversine before accepting a result.
 */
#define SN_DEG_TO_M_LAT_MIN 110574.0
#define SN_DEG_TO_M_LON_EQ 111320.0  /* at equator; shrinks with cos(lat) */

typedef struct {
    double query_lat;
    double query_lon;
    double radius_m;
    double radius_lat_deg;
    double lon_deg_to_m;
    ResultBuf *result_buf;
} SearchCtx;

static void search_recursive(const SN_KDNode *node,
                             const SearchCtx *ctx)
{
    if (node == NULL) {
        return;
    }

    if (ctx->result_buf->count >= ctx->result_buf->capacity) {
        return;
    }

    /*
     * Fast reject: meridional distance is a strict lower bound for great-circle
     * distance, so if this alone exceeds the radius we can skip Haversine.
     */
    const double lat_delta_deg = fabs(node->shelter.lat - ctx->query_lat);
    if (lat_delta_deg <= ctx->radius_lat_deg) {
        const double dist = sn_haversine(ctx->query_lat, ctx->query_lon,
                                         node->shelter.lat, node->shelter.lon);
        if (dist <= ctx->radius_m) {
            (void)result_buf_push(ctx->result_buf, &node->shelter);
        }
    }

    /* Axis-aligned distance for pruning. */
    const int axis = node->split_axis;
    const double query_val = point_axis_val(ctx->query_lat, ctx->query_lon, axis);
    const double node_val = (axis == 0) ? node->shelter.lat : node->shelter.lon;
    const double diff_deg = query_val - node_val;

    const double deg_to_m = (axis == 0)
        ? SN_DEG_TO_M_LAT_MIN
        : ctx->lon_deg_to_m;
    const double diff_m = fabs(diff_deg) * deg_to_m;

    /* Decide which subtree to search first (nearer side). */
    const SN_KDNode *near_child;
    const SN_KDNode *far_child;
    if (diff_deg < 0.0) {
        near_child = node->left;
        far_child  = node->right;
    } else {
        near_child = node->right;
        far_child  = node->left;
    }

    /* Always search the near side. */
    search_recursive(near_child, ctx);

    /* Only search the far side if the splitting plane is within radius. */
    if (diff_m <= ctx->radius_m) {
        search_recursive(far_child, ctx);
    }
}

int sn_find_nearest(const SN_KDNode *tree,
                    double lat, double lon,
                    double radius_m,
                    SN_Shelter out[],
                    int max_results)
{
    if (out == NULL || max_results <= 0) {
        return SN_ERR_INVALID_ARG;
    }
    if (radius_m < 0.0) {
        return SN_ERR_INVALID_ARG;
    }
    if (tree == NULL) {
        return 0;
    }

    ResultBuf rb;
    result_buf_init(&rb, out, max_results);

    const double query_lat_rad = lat * (3.14159265358979323846 / 180.0);
    double lon_deg_to_m = SN_DEG_TO_M_LON_EQ * cos(query_lat_rad);
    if (lon_deg_to_m < 0.0) {
        lon_deg_to_m = -lon_deg_to_m;
    }

    SearchCtx ctx;
    ctx.query_lat = lat;
    ctx.query_lon = lon;
    ctx.radius_m = radius_m;
    ctx.radius_lat_deg = radius_m / SN_DEG_TO_M_LAT_MIN;
    ctx.lon_deg_to_m = lon_deg_to_m;
    ctx.result_buf = &rb;

    search_recursive(tree, &ctx);

    return rb.count;
}
