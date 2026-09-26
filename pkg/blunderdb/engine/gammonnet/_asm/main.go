// SPDX-License-Identifier: MIT

// Générateur avo du noyau dense AVX2 (ADR-0024).
//
// Il vit dans un module à part (`_asm/go.mod`) et sous un répertoire que l'outil
// Go ignore : avo n'entre donc pas dans les dépendances de blunderDB, et
// `govulncheck` comme `golangci-lint` ne voient que le `.s` produit.
//
// Régénérer :
//
//	cd pkg/blunderdb/engine/gammonnet/_asm && go run . \
//	    -out ../kernel_avx2_amd64.s -stubs ../kernel_avx2_amd64.go -pkg gammonnet
//
// (le `//go:generate` de kernel_amd64.go fait exactement cela)
//
// Ce que le noyau calcule, pour une couche dense de `outDim` sorties sur `in`
// entrées, sur un lot de huit positions rangées en feature-major :
//
//	acc[n]      = bias[i]                         n ∈ [0, 8)
//	acc[n]     += w[i*in+j] * act[j*8+n]          j croissant
//	out[i*8+n]  = relu(acc[n])  ou  acc[n]
//
// Une voie = une position : la vectorisation porte sur le LOT, jamais sur la
// réduction. Chaque voie somme sur j croissant, depuis le biais, en float32,
// avec `VMULPS` puis `VADDPS` **séparés**. Aucune FMA n'est émise : `VFMADD*`
// garderait plus de précision que network.go et casserait l'identité
// (ADR-0024).
package main

import (
	. "github.com/mmcloughlin/avo/build"
	. "github.com/mmcloughlin/avo/operand"
	"github.com/mmcloughlin/avo/reg"
)

// lanes est la largeur du lot, EvalBatchWidth côté Go : huit float32, soit un
// registre YMM plein.
const lanes = 8

// tile est le nombre de sorties traitées ensemble : la colonne d'activations
// est chargée une fois et sert aux `tile` lignes de poids, et les `tile`
// accumulateurs donnent autant de chaînes de dépendance indépendantes.
//
// Six : sur Zen 3/4 `VADDPS` a 3 cycles de latence sur deux pipes (FP2/FP3),
// il faut 3 × 2 chaînes pour les saturer. Huit manque de registres généraux
// et fait prendre BP, le pointeur de trame du runtime ;
// kernel_avx2_amd64_test.go refuse un `.s` qui mentionne BP.
const tile = 6

// Assertions de compilation (reprise de `gn_tile.h`) : une constante uint
// négative arrête la compilation.
//
//  1. lanes est une puissance de deux <= 8 : elle sert d'ÉCHELLE d'adressage
//     x86 dans `Mem{Base: actp, Index: jb, Scale: lanes}`.
//  2. tile > 0, sinon la boucle tuilée ne finit pas.
//
// outDim n'a pas à être un multiple de tile : le compteur décroissant et la
// queue une-sortie-à-la-fois traitent tout reste.
const (
	_ uint = 0 - (lanes & (lanes - 1))
	_ uint = 8 - lanes
	_ uint = tile - 1
)

func main() {
	ConstraintExpr("amd64,!purego")

	dense("denseAVX2ReLU", true)
	dense("denseAVX2Linear", false)

	Generate()
}

func dense(name string, relu bool) {
	TEXT(name, NOSPLIT, "func(w, bias, act, out *float32, in, outDim int)")
	if relu {
		Doc(
			name+" évalue une couche dense sur un lot de 8 positions et applique ReLU.",
			"",
			"Les activations sont feature-major (act[j*8+n]), les poids row-major",
			"[outDim][in]. Chaque voie n accumule sur j croissant en float32,",
			"multiplication et addition séparées — jamais de FMA (ADR-0024).",
		)
	} else {
		Doc(
			name+" évalue une couche dense sur un lot de 8 positions, sans activation.",
			"",
			"Même contrat arithmétique que denseAVX2ReLU : voie par position,",
			"somme sur j croissant, VMULPS puis VADDPS séparés, jamais de FMA.",
		)
	}
	Pragma("noescape")

	wp := Load(Param("w"), GP64())
	biasp := Load(Param("bias"), GP64())
	actp := Load(Param("act"), GP64())
	outp := Load(Param("out"), GP64())
	in := Load(Param("in"), GP64())
	outDim := Load(Param("outDim"), GP64())

	// rowBytes : la longueur d'une ligne de poids en octets. Elle sert à la
	// fois de pas entre deux lignes et de borne de la boucle interne, qui
	// compte en octets pour n'avoir qu'un seul registre d'indice.
	rowBytes := GP64()
	MOVQ(in, rowBytes)
	SHLQ(U8(2), rowBytes)

	// remaining : sorties restant à produire. Un compteur décroissant économise
	// deux registres généraux (le noyau tient sans déborder à tile = 6), et
	// évite `outDim & ^(tile-1)`, qui n'arrondit que pour une puissance de deux.
	remaining := outDim

	// Le zéro de ReLU : +0.0 dans les huit voies. VMAXPS(zero, acc, acc) rend
	// acc quand acc > 0 et +0.0 sinon — y compris pour acc = −0.0 et pour NaN,
	// exactement comme le `if sum > 0 { out = sum } else { out = 0 }` scalaire.
	zero := YMM()
	VXORPS(zero, zero, zero)

	acc := make([]reg.VecVirtual, tile)
	tmp := make([]reg.VecVirtual, tile)
	ptr := make([]reg.GPVirtual, tile)
	for t := 0; t < tile; t++ {
		acc[t] = YMM()
		tmp[t] = YMM()
		ptr[t] = GP64()
	}
	col := YMM()
	jb := GP64()

	// ---- boucle tuilée : `tile` sorties à la fois -------------------------
	Label("tileloop")
	CMPQ(remaining, U32(tile))
	JL(LabelRef("tailhead"))

	for t := 0; t < tile; t++ {
		VBROADCASTSS(Mem{Base: biasp}.Offset(4*t), acc[t])
	}
	MOVQ(wp, ptr[0])
	for t := 1; t < tile; t++ {
		MOVQ(ptr[t-1], ptr[t])
		ADDQ(rowBytes, ptr[t])
	}

	XORQ(jb, jb)
	Label("tilej")
	VMOVUPS(Mem{Base: actp, Index: jb, Scale: lanes}, col)
	for t := 0; t < tile; t++ {
		VBROADCASTSS(Mem{Base: ptr[t], Index: jb, Scale: 1}, tmp[t])
		VMULPS(col, tmp[t], tmp[t])
		VADDPS(tmp[t], acc[t], acc[t])
	}
	ADDQ(U8(4), jb)
	CMPQ(jb, rowBytes)
	JL(LabelRef("tilej"))

	for t := 0; t < tile; t++ {
		if relu {
			VMAXPS(zero, acc[t], acc[t])
		}
		VMOVUPS(acc[t], Mem{Base: outp}.Offset(4*lanes*t))
	}

	for t := 0; t < tile; t++ {
		ADDQ(rowBytes, wp)
	}
	ADDQ(U8(4*tile), biasp)
	ADDQ(U32(4*lanes*tile), outp)
	SUBQ(U8(tile), remaining)
	JMP(LabelRef("tileloop"))

	// ---- reste : une sortie à la fois -------------------------------------
	Label("tailhead")
	Label("tailloop")
	CMPQ(remaining, U32(0))
	JLE(LabelRef("done"))

	VBROADCASTSS(Mem{Base: biasp}, acc[0])
	MOVQ(wp, ptr[0])
	XORQ(jb, jb)
	Label("tailj")
	VMOVUPS(Mem{Base: actp, Index: jb, Scale: lanes}, col)
	VBROADCASTSS(Mem{Base: ptr[0], Index: jb, Scale: 1}, tmp[0])
	VMULPS(col, tmp[0], tmp[0])
	VADDPS(tmp[0], acc[0], acc[0])
	ADDQ(U8(4), jb)
	CMPQ(jb, rowBytes)
	JL(LabelRef("tailj"))

	if relu {
		VMAXPS(zero, acc[0], acc[0])
	}
	VMOVUPS(acc[0], Mem{Base: outp})

	ADDQ(rowBytes, wp)
	ADDQ(U8(4), biasp)
	ADDQ(U8(4*lanes), outp)
	DECQ(remaining)
	JMP(LabelRef("tailloop"))

	Label("done")
	VZEROUPPER()
	RET()
}
