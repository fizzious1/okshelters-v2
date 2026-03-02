/**
 * kdtree.c -- KD-tree spatial index for shelters.
 *
 * Alternates split axis between latitude (axis 0) and longitude (axis 1)
 * at each level of the tree.
 *
 * TODO: Replace malloc with a pre-allocated arena allocator to eliminate
 *       dynamic allocation in the query path.
 */

#include "shelternav.h"

#include <stdlib.h>
#include <string.h>
#include <math.h>

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

/** Return the split-axis value for a node's shelter. */
static double node_axis_val(const SN_KDNode *node)
{
    return (node->split_axis == 0) ? node->shelter.lat : node->shelter.lon;
}

/** Return the split-axis value for a query point. */
static double point_axis_val(double lat, double lon, int axis)
{
    return (axis == 0) ? lat : lon;
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
 * Insert (recursive, alternating axis)
 * ------------------------------------------------------------------- */

static SN_KDNode *insert_recursive(SN_KDNode *node,
                                   const SN_Shelter *shelter,
                                   int depth)
{
    if (node == NULL) {
        /* TODO: allocate from arena instead of malloc */
        SN_KDNode *new_node = malloc(sizeof(SN_KDNode));
        if (new_node == NULL) {
            return NULL;
        }
        new_node->shelter    = *shelter;
        new_node->left       = NULL;
        new_node->right      = NULL;
        new_node->split_axis = depth % 2;
        return new_node;
    }

    int    axis     = node->split_axis;
    double node_val = node_axis_val(node);
    double ins_val  = point_axis_val(shelter->lat, shelter->lon, axis);

    if (ins_val < node_val) {
        node->left  = insert_recursive(node->left, shelter, depth + 1);
    } else {
        node->right = insert_recursive(node->right, shelter, depth + 1);
    }

    return node;
}

int sn_kdtree_insert(SN_KDNode **tree, const SN_Shelter *shelter)
{
    if (tree == NULL || shelter == NULL) {
        return SN_ERR_INVALID_ARG;
    }

    SN_KDNode *result = insert_recursive(*tree, shelter, 0);
    if (result == NULL && *tree != NULL) {
        /* Allocation failure on a non-empty tree */
        return SN_ERR_OUT_OF_MEMORY;
    }
    *tree = result;
    return SN_OK;
}

/* -------------------------------------------------------------------
 * Nearest-neighbor search with distance pruning
 * ------------------------------------------------------------------- */

/**
 * Rough degree-to-metre factor used for axis-aligned pruning.
 * Not exact — only used to skip subtrees that are obviously too far.
 * Actual distance is always confirmed with sn_haversine.
 */
#define SN_DEG_TO_M_LAT 111320.0
#define SN_DEG_TO_M_LON_EQ 111320.0  /* at equator; shrinks with cos(lat) */

static void search_recursive(const SN_KDNode *node,
                             double lat, double lon,
                             double radius_m,
                             ResultBuf *rb)
{
    if (node == NULL) {
        return;
    }

    /* Check this node against the radius using true Haversine distance. */
    double dist = sn_haversine(lat, lon,
                               node->shelter.lat, node->shelter.lon);
    if (dist <= radius_m) {
        result_buf_push(rb, &node->shelter);
    }

    /* Axis-aligned distance for pruning. */
    double query_val = point_axis_val(lat, lon, node->split_axis);
    double node_val  = node_axis_val(node);
    double diff_deg  = query_val - node_val;

    double deg_to_m = (node->split_axis == 0)
        ? SN_DEG_TO_M_LAT
        : SN_DEG_TO_M_LON_EQ * cos(lat * M_PI / 180.0);
    double diff_m = fabs(diff_deg) * deg_to_m;

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
    search_recursive(near_child, lat, lon, radius_m, rb);

    /* Only search the far side if the splitting plane is within radius. */
    if (diff_m <= radius_m) {
        search_recursive(far_child, lat, lon, radius_m, rb);
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

    ResultBuf rb;
    result_buf_init(&rb, out, max_results);

    search_recursive(tree, lat, lon, radius_m, &rb);

    return rb.count;
}
