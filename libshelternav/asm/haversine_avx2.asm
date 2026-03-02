; haversine_avx2.asm -- AVX2 batch Haversine (x86_64, NASM)
;
; Calling convention: System V AMD64 ABI
;   - rdi : origin_lat  (double, passed in xmm0)
;   - rsi : origin_lon  (double, passed in xmm1)
;   - rdx : lats[]      (pointer)
;   - rcx : lons[]      (pointer)
;   - r8  : distances_out[] (pointer)
;   - r9  : n           (int32)
;
; Correction per System V AMD64:
;   xmm0 = origin_lat
;   xmm1 = origin_lon
;   rdi   = lats[]
;   rsi   = lons[]
;   rdx   = distances_out[]
;   ecx   = n
;
; Register usage plan:
;   ymm0-ymm3  : input lat/lon vectors
;   ymm4-ymm7  : intermediate sin/cos results
;   ymm8       : earth radius constant
;   ymm9       : pi/180 constant
;   ymm10-ymm15: scratch
;
; TODO: Implement vectorised Haversine using 4-wide double (ymm).
;       For now this is a stub that returns immediately.
;       The scalar C fallback (sn_haversine_batch) handles all calls.

section .text
global _sn_haversine_batch_avx2

_sn_haversine_batch_avx2:
    ret
