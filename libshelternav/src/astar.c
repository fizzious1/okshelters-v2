/**
 * astar.c -- A* shortest-path routing on a road graph.
 *
 * This is a skeleton.  The core algorithm is stubbed out and returns
 * SN_ERR_NOT_IMPLEMENTED until the road-graph data model is finalised.
 *
 * Design notes:
 *   - Priority queue is a fixed-size min-heap on the stack (no malloc).
 *   - Heuristic is Haversine distance (admissible for geographic graphs).
 *   - Graph uses an adjacency-list representation backed by flat arrays
 *     so the entire structure can live in a single arena allocation.
 */

#include "shelternav.h"

#include <string.h>

/* -------------------------------------------------------------------
 * Internal: road-graph representation (stub)
 * ------------------------------------------------------------------- */

/**
 * Edge in the road graph.
 */
typedef struct {
    int32_t   target_node;   /* index into SN_RoadGraph.nodes[] */
    double    cost_m;        /* edge weight in metres             */
} SN_Edge;

/**
 * Node in the road graph.
 */
typedef struct {
    SN_LatLon pos;
    int32_t   edge_start;    /* index into SN_RoadGraph.edges[]  */
    int32_t   edge_count;    /* number of outgoing edges          */
} SN_GraphNode;

/**
 * Full road-graph structure.
 * Flat arrays — no per-node allocation.
 */
struct SN_RoadGraph {
    SN_GraphNode *nodes;
    int32_t       node_count;
    SN_Edge      *edges;
    int32_t       edge_count;
};

/* -------------------------------------------------------------------
 * Internal: fixed-size min-heap (priority queue)
 *
 * Stack-allocated.  No dynamic memory.
 * ------------------------------------------------------------------- */

#define SN_HEAP_CAPACITY 8192

typedef struct {
    int32_t node_idx;
    double  f_score;   /* g + h */
} HeapEntry;

typedef struct {
    HeapEntry entries[SN_HEAP_CAPACITY];
    int       size;
} MinHeap;

static void heap_init(MinHeap *h)
{
    h->size = 0;
}

static void heap_swap(HeapEntry *a, HeapEntry *b)
{
    HeapEntry tmp = *a;
    *a = *b;
    *b = tmp;
}

static int heap_push(MinHeap *h, int32_t node_idx, double f_score)
{
    if (h->size >= SN_HEAP_CAPACITY) {
        return -1; /* heap full */
    }

    int i = h->size;
    h->entries[i].node_idx = node_idx;
    h->entries[i].f_score  = f_score;
    h->size++;

    /* Sift up */
    while (i > 0) {
        int parent = (i - 1) / 2;
        if (h->entries[parent].f_score <= h->entries[i].f_score) {
            break;
        }
        heap_swap(&h->entries[parent], &h->entries[i]);
        i = parent;
    }

    return 0;
}

static int heap_pop(MinHeap *h, HeapEntry *out)
{
    if (h->size <= 0) {
        return -1; /* empty */
    }

    *out = h->entries[0];
    h->size--;

    if (h->size > 0) {
        h->entries[0] = h->entries[h->size];

        /* Sift down */
        int i = 0;
        for (;;) {
            int left  = 2 * i + 1;
            int right = 2 * i + 2;
            int smallest = i;

            if (left < h->size &&
                h->entries[left].f_score < h->entries[smallest].f_score) {
                smallest = left;
            }
            if (right < h->size &&
                h->entries[right].f_score < h->entries[smallest].f_score) {
                smallest = right;
            }
            if (smallest == i) {
                break;
            }
            heap_swap(&h->entries[i], &h->entries[smallest]);
            i = smallest;
        }
    }

    return 0;
}

/* -------------------------------------------------------------------
 * A* (stub)
 * ------------------------------------------------------------------- */

int sn_route_astar(const SN_RoadGraph *graph,
                   SN_LatLon start,
                   SN_LatLon end,
                   SN_Maneuver path_out[],
                   int *path_len)
{
    (void)graph;
    (void)start;
    (void)end;
    (void)path_out;
    (void)path_len;

    /*
     * TODO: Implement full A* once the road-graph ingestion pipeline
     * is in place.  Steps:
     *   1. Map start/end LatLon to nearest graph nodes.
     *   2. Run A* with Haversine heuristic (sn_haversine).
     *   3. Back-trace came_from[] to build the path.
     *   4. Convert node sequence to SN_Maneuver array.
     *
     * The MinHeap above is ready to use.  The heap sift-down will
     * eventually be replaced by ASM (heap_x86.asm / heap_arm64.S)
     * for the hot inner loop.
     */

    return SN_ERR_NOT_IMPLEMENTED;
}
