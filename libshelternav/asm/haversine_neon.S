// haversine_neon.S -- NEON batch Haversine (ARM64, GNU as)
//
// Calling convention: AAPCS64 (ARM Architecture Procedure Call Standard)
//   d0  = origin_lat   (double)
//   d1  = origin_lon   (double)
//   x0  = lats[]       (pointer to double array)
//   x1  = lons[]       (pointer to double array)
//   x2  = distances_out[] (pointer to double array)
//   w3  = n            (int32, number of points)
//
// Register usage plan:
//   v0-v3   : input lat/lon pairs (2-wide double via NEON 128-bit)
//   v4-v7   : intermediate sin/cos approximation results
//   v8      : earth radius constant (6371000.0)
//   v9      : pi/180 constant
//   v10-v15 : scratch
//   x4-x7   : loop counters / temporaries
//
// TODO: Implement vectorised Haversine using 2-wide float64x2_t.
//       For now this is a stub that returns immediately.
//       The scalar C fallback (sn_haversine_batch) handles all calls.

    .text
    .global _sn_haversine_batch_neon
    .align 4

_sn_haversine_batch_neon:
    ret
