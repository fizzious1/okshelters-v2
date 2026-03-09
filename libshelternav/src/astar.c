/**
 * astar.c -- A* shortest-path routing on a road graph.
 *
 * Design notes:
 *   - Priority queue is a fixed-size min-heap on the stack (no malloc).
 *   - Heuristic is Haversine distance (admissible for geographic graphs).
 *   - Graph uses an adjacency-list representation backed by flat arrays
 *     so the entire structure can live in a single arena allocation.
 */

#include "shelternav.h"

#include <float.h>
#include <stddef.h>

/* -------------------------------------------------------------------
 * Internal: road-graph representation
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
 * A*
 * ------------------------------------------------------------------- */

static int nearest_node_index(const SN_RoadGraph *graph, SN_LatLon point)
{
    if (graph == NULL || graph->nodes == NULL || graph->node_count <= 0) {
        return -1;
    }

    int best_idx = -1;
    double best_dist = DBL_MAX;

    for (int i = 0; i < graph->node_count; i++) {
        const SN_GraphNode *node = &graph->nodes[i];
        const double dist = sn_haversine(point.lat, point.lon,
                                         node->pos.lat, node->pos.lon);
        if (dist < best_dist) {
            best_dist = dist;
            best_idx = i;
        }
    }

    return best_idx;
}

int sn_route_astar(const SN_RoadGraph *graph,
                   SN_LatLon start,
                   SN_LatLon end,
                   SN_Maneuver path_out[],
                   int *path_len)
{
    if (graph == NULL || path_out == NULL || path_len == NULL || *path_len <= 0) {
        return SN_ERR_INVALID_ARG;
    }
    if (graph->nodes == NULL || graph->edges == NULL ||
        graph->node_count <= 0 || graph->edge_count < 0) {
        return SN_ERR_INVALID_ARG;
    }
    if (graph->node_count > SN_HEAP_CAPACITY) {
        return SN_ERR_INVALID_ARG;
    }

    const int start_idx = nearest_node_index(graph, start);
    const int end_idx = nearest_node_index(graph, end);
    if (start_idx < 0 || end_idx < 0) {
        return SN_ERR_NO_PATH;
    }

    const int output_capacity = *path_len;
    *path_len = 0;

    if (start_idx == end_idx) {
        path_out[0].point = graph->nodes[start_idx].pos;
        path_out[0].distance_m = 0.0;
        path_out[0].instruction[0] = '\0';
        *path_len = 1;
        return SN_OK;
    }

    double g_score[SN_HEAP_CAPACITY];
    int32_t came_from[SN_HEAP_CAPACITY];
    unsigned char closed[SN_HEAP_CAPACITY];
    int32_t reversed_path[SN_HEAP_CAPACITY];

    for (int i = 0; i < graph->node_count; i++) {
        g_score[i] = DBL_MAX;
        came_from[i] = -1;
        closed[i] = 0;
    }

    const SN_LatLon end_pos = graph->nodes[end_idx].pos;
    MinHeap open_set;
    heap_init(&open_set);

    g_score[start_idx] = 0.0;
    const SN_LatLon start_pos = graph->nodes[start_idx].pos;
    const double start_heuristic = sn_haversine(start_pos.lat, start_pos.lon,
                                                end_pos.lat, end_pos.lon);
    if (heap_push(&open_set, start_idx, start_heuristic) != 0) {
        return SN_ERR_INVALID_ARG;
    }

    int found = 0;
    HeapEntry current_entry;

    while (heap_pop(&open_set, &current_entry) == 0) {
        const int current = current_entry.node_idx;
        if (current < 0 || current >= graph->node_count) {
            continue;
        }
        if (closed[current]) {
            continue;
        }
        if (current == end_idx) {
            found = 1;
            break;
        }

        closed[current] = 1;

        const SN_GraphNode *node = &graph->nodes[current];
        if (node->edge_start < 0 || node->edge_count < 0) {
            continue;
        }

        const int64_t edge_start = node->edge_start;
        const int64_t edge_count = node->edge_count;
        const int64_t edge_end = edge_start + edge_count;
        if (edge_start < 0 || edge_count < 0 ||
            edge_end < edge_start || edge_end > graph->edge_count) {
            continue;
        }

        for (int64_t edge_idx = edge_start; edge_idx < edge_end; edge_idx++) {
            const SN_Edge *edge = &graph->edges[edge_idx];
            const int next = edge->target_node;

            if (next < 0 || next >= graph->node_count || closed[next]) {
                continue;
            }
            if (edge->cost_m < 0.0) {
                continue;
            }

            const double tentative_g = g_score[current] + edge->cost_m;
            if (tentative_g >= g_score[next]) {
                continue;
            }

            came_from[next] = current;
            g_score[next] = tentative_g;

            const SN_LatLon next_pos = graph->nodes[next].pos;
            const double heuristic = sn_haversine(next_pos.lat, next_pos.lon,
                                                  end_pos.lat, end_pos.lon);
            const double f_score = tentative_g + heuristic;

            if (heap_push(&open_set, next, f_score) != 0) {
                return SN_ERR_INVALID_ARG;
            }
        }
    }

    if (!found) {
        return SN_ERR_NO_PATH;
    }

    int rev_count = 0;
    for (int node_idx = end_idx; node_idx >= 0; node_idx = came_from[node_idx]) {
        if (rev_count >= SN_HEAP_CAPACITY) {
            return SN_ERR_INVALID_ARG;
        }
        reversed_path[rev_count++] = node_idx;
        if (node_idx == start_idx) {
            break;
        }
    }

    if (rev_count == 0 || reversed_path[rev_count - 1] != start_idx) {
        return SN_ERR_NO_PATH;
    }
    if (rev_count > output_capacity) {
        return SN_ERR_INVALID_ARG;
    }

    for (int i = 0; i < rev_count; i++) {
        const int node_idx = reversed_path[rev_count - 1 - i];
        const SN_LatLon point = graph->nodes[node_idx].pos;

        path_out[i].point = point;
        path_out[i].instruction[0] = '\0';
        if (i == 0) {
            path_out[i].distance_m = 0.0;
        } else {
            const SN_LatLon prev = path_out[i - 1].point;
            path_out[i].distance_m = sn_haversine(prev.lat, prev.lon,
                                                  point.lat, point.lon);
        }
    }

    *path_len = rev_count;
    return SN_OK;
}
