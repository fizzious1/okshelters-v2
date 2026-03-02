; heap_x86.asm -- Min-heap sift-down (x86_64, NASM)
;
; Calling convention: System V AMD64 ABI
;   rdi = entries[]     (pointer to HeapEntry array)
;   esi = size          (int32, number of entries)
;   edx = index         (int32, index to sift down from)
;
; HeapEntry layout (16 bytes):
;   offset 0:  int32_t node_idx  (4 bytes, padded to 8)
;   offset 8:  double  f_score   (8 bytes)
;
; Register usage:
;   rax     : current index
;   rbx     : child index (smallest)
;   rcx     : scratch
;   xmm0    : current f_score
;   xmm1    : child f_score (comparison)
;
; TODO: Implement branchless sift-down with CMOV for the A* hot loop.
;       For now this is a stub; the C min-heap in astar.c is used.

section .text
global _sn_heap_siftdown_x86

_sn_heap_siftdown_x86:
    ret
