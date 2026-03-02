// heap_arm64.S -- Min-heap sift-down (ARM64, GNU as)
//
// Calling convention: AAPCS64
//   x0 = entries[]     (pointer to HeapEntry array)
//   w1 = size          (int32, number of entries)
//   w2 = index         (int32, index to sift down from)
//
// HeapEntry layout (16 bytes):
//   offset 0:  int32_t node_idx  (4 bytes, padded to 8)
//   offset 8:  double  f_score   (8 bytes)
//
// Register usage:
//   x3      : current index
//   x4      : left child index
//   x5      : right child index
//   x6      : smallest child index
//   d0      : current f_score
//   d1      : child f_score (comparison)
//   x7-x9   : scratch / temporaries
//
// TODO: Implement branchless sift-down with CSEL for the A* hot loop.
//       For now this is a stub; the C min-heap in astar.c is used.

    .text
    .global _sn_heap_siftdown_arm64
    .align 4

_sn_heap_siftdown_arm64:
    ret
