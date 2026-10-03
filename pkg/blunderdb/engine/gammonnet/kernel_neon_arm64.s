// SPDX-License-Identifier: MIT

//go:build arm64 && !purego

#include "textflag.h"

// Hand-written NEON twin of kernel_avx2_amd64.s; avo has no arm64 back end.
//
// Same contract as denseGo (kernel_go.go), which is the reference this file is
// checked against bit for bit (kernel_identity_test.go):
//
//   - act and out are feature-major, act[j*8+n]: lane n is position n. A batch
//     of 8 positions is two 4-lane registers, lo = lanes 0-3, hi = lanes 4-7.
//   - every accumulator starts at bias[i] and takes the terms in ascending j.
//   - the multiply and the add are two instructions, FMUL then FADD, never
//     FMLA: a fused multiply-add rounds once instead of twice and returns
//     other bits (ADR-0024). TestNEONKernelHasNoFMA reads this file back.
//
// A tile is 8 output rows x 8 lanes = 16 independent accumulators, so the
// FADD latency chain of each one is hidden behind the fifteen others; the
// 4-pipe Apple cores need at least 12 in flight.
//
// Go's assembler has no mnemonic for the vector FMUL / FADD / FCMGT, so they
// are spelled as their A64 encodings (single precision, Q=1, 4S arrangement).
// Each macro reads d = n OP m with register numbers.

#define FMUL4S(d, n, m) WORD $(0x6E20DC00 | ((m)<<16) | ((n)<<5) | (d))
#define FADD4S(d, n, m) WORD $(0x4E20D400 | ((m)<<16) | ((n)<<5) | (d))
#define FCMGTZ4S(d, n) WORD $(0x4EA0C800 | ((n)<<5) | (d))

// One row of one j step: V2 = broadcast w[row][j] (the row pointer walks the
// row), then lo += w*act_lo and hi += w*act_hi, in that order of operands so
// the expression is the same as denseGo's acc + float32(w*col).
#define ROWSTEP(ptr, lo, hi) \
	VLD1R.P 4(ptr), [V2.S4]; \
	FMUL4S(3, 2, 0); \
	FMUL4S(4, 2, 1); \
	FADD4S(lo, lo, 3); \
	FADD4S(hi, hi, 4)

// Start a row's two accumulators at its bias and advance the bias pointer.
#define BIAS(lo, hi) \
	VLD1R (R1), [hi.S4]; \
	VLD1R.P 4(R1), [lo.S4]

// ReLU as denseGo writes it, !(v > 0) -> +0: the FCMGT mask is all ones only
// where v > 0, so a NaN and both zeros come out as +0. FMAX would let a NaN
// through, and this is the one place where NEON and VMAXPS disagree.
#define RELU(n, v) \
	FCMGTZ4S(5, n); \
	VAND V5.B16, v.B16, v.B16

// func denseNEONReLU(w *float32, bias *float32, act *float32, out *float32, in int, outDim int)
TEXT ·denseNEONReLU(SB), NOSPLIT|NOFRAME, $0-48
	MOVD w+0(FP), R0
	MOVD bias+8(FP), R1
	MOVD act+16(FP), R2
	MOVD out+24(FP), R3
	MOVD in+32(FP), R4
	MOVD outDim+40(FP), R5
	LSL  $2, R4, R16       // row stride in bytes

relutile:
	CMP  $8, R5
	BLT  relutail
	BIAS(V8, V9)
	BIAS(V10, V11)
	BIAS(V12, V13)
	BIAS(V14, V15)
	BIAS(V16, V17)
	BIAS(V18, V19)
	BIAS(V20, V21)
	BIAS(V22, V23)
	MOVD R0, R8
	ADD  R16, R8, R9
	ADD  R16, R9, R10
	ADD  R16, R10, R11
	ADD  R16, R11, R12
	ADD  R16, R12, R13
	ADD  R16, R13, R14
	ADD  R16, R14, R15
	MOVD R2, R6
	MOVD R4, R7

relutilej:
	VLD1.P 32(R6), [V0.S4, V1.S4]
	ROWSTEP(R8, 8, 9)
	ROWSTEP(R9, 10, 11)
	ROWSTEP(R10, 12, 13)
	ROWSTEP(R11, 14, 15)
	ROWSTEP(R12, 16, 17)
	ROWSTEP(R13, 18, 19)
	ROWSTEP(R14, 20, 21)
	ROWSTEP(R15, 22, 23)
	SUBS $1, R7, R7
	BNE  relutilej

	RELU(8, V8)
	RELU(9, V9)
	RELU(10, V10)
	RELU(11, V11)
	RELU(12, V12)
	RELU(13, V13)
	RELU(14, V14)
	RELU(15, V15)
	RELU(16, V16)
	RELU(17, V17)
	RELU(18, V18)
	RELU(19, V19)
	RELU(20, V20)
	RELU(21, V21)
	RELU(22, V22)
	RELU(23, V23)
	VST1.P [V8.S4, V9.S4], 32(R3)
	VST1.P [V10.S4, V11.S4], 32(R3)
	VST1.P [V12.S4, V13.S4], 32(R3)
	VST1.P [V14.S4, V15.S4], 32(R3)
	VST1.P [V16.S4, V17.S4], 32(R3)
	VST1.P [V18.S4, V19.S4], 32(R3)
	VST1.P [V20.S4, V21.S4], 32(R3)
	VST1.P [V22.S4, V23.S4], 32(R3)
	MOVD R15, R0           // the last row pointer stopped at row 8 of the tile
	SUB  $8, R5, R5
	B    relutile

relutail:
	CBZ  R5, reludone
	BIAS(V8, V9)
	MOVD R2, R6
	MOVD R4, R7

relutailj:
	VLD1.P 32(R6), [V0.S4, V1.S4]
	ROWSTEP(R0, 8, 9)
	SUBS $1, R7, R7
	BNE  relutailj

	RELU(8, V8)
	RELU(9, V9)
	VST1.P [V8.S4, V9.S4], 32(R3)
	SUB  $1, R5, R5
	B    relutail

reludone:
	RET

// func denseNEONLinear(w *float32, bias *float32, act *float32, out *float32, in int, outDim int)
TEXT ·denseNEONLinear(SB), NOSPLIT|NOFRAME, $0-48
	MOVD w+0(FP), R0
	MOVD bias+8(FP), R1
	MOVD act+16(FP), R2
	MOVD out+24(FP), R3
	MOVD in+32(FP), R4
	MOVD outDim+40(FP), R5
	LSL  $2, R4, R16

lintile:
	CMP  $8, R5
	BLT  lintail
	BIAS(V8, V9)
	BIAS(V10, V11)
	BIAS(V12, V13)
	BIAS(V14, V15)
	BIAS(V16, V17)
	BIAS(V18, V19)
	BIAS(V20, V21)
	BIAS(V22, V23)
	MOVD R0, R8
	ADD  R16, R8, R9
	ADD  R16, R9, R10
	ADD  R16, R10, R11
	ADD  R16, R11, R12
	ADD  R16, R12, R13
	ADD  R16, R13, R14
	ADD  R16, R14, R15
	MOVD R2, R6
	MOVD R4, R7

lintilej:
	VLD1.P 32(R6), [V0.S4, V1.S4]
	ROWSTEP(R8, 8, 9)
	ROWSTEP(R9, 10, 11)
	ROWSTEP(R10, 12, 13)
	ROWSTEP(R11, 14, 15)
	ROWSTEP(R12, 16, 17)
	ROWSTEP(R13, 18, 19)
	ROWSTEP(R14, 20, 21)
	ROWSTEP(R15, 22, 23)
	SUBS $1, R7, R7
	BNE  lintilej

	VST1.P [V8.S4, V9.S4], 32(R3)
	VST1.P [V10.S4, V11.S4], 32(R3)
	VST1.P [V12.S4, V13.S4], 32(R3)
	VST1.P [V14.S4, V15.S4], 32(R3)
	VST1.P [V16.S4, V17.S4], 32(R3)
	VST1.P [V18.S4, V19.S4], 32(R3)
	VST1.P [V20.S4, V21.S4], 32(R3)
	VST1.P [V22.S4, V23.S4], 32(R3)
	MOVD R15, R0
	SUB  $8, R5, R5
	B    lintile

lintail:
	CBZ  R5, lindone
	BIAS(V8, V9)
	MOVD R2, R6
	MOVD R4, R7

lintailj:
	VLD1.P 32(R6), [V0.S4, V1.S4]
	ROWSTEP(R0, 8, 9)
	SUBS $1, R7, R7
	BNE  lintailj

	VST1.P [V8.S4, V9.S4], 32(R3)
	SUB  $1, R5, R5
	B    lintail

lindone:
	RET
