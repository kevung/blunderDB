// SPDX-License-Identifier: MIT

// Générateur avo du noyau dense AVX2 (fiche F1, ADR-0024).
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
// Une voie = une position. La vectorisation porte donc sur la dimension du LOT
// et jamais sur la réduction : chaque voie somme sur j dans l'ordre croissant,
// en partant du biais, en float32, avec `VMULPS` puis `VADDPS` **séparés**.
// Aucune instruction FMA n'est émise ici, et c'est une propriété du fichier, pas
// une préférence : `VFMADD*` garderait plus de précision que la boucle scalaire
// de network.go et mettrait l'accord inter-machines hors de portée (ADR-0024).
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
// Six, et pas quatre. Sur Zen 3/4 `VADDPS` a une latence de 3 cycles pour deux
// pipes d'addition disjointes (FP2/FP3, cf. la note de recherche P2) : il faut
// 3 × 2 = 6 chaînes indépendantes pour les saturer. Quatre laissait un tiers du
// débit — mesuré à −12 % sur le lot complet.
//
// Huit ne passe pas : l'allocateur d'avo n'a plus assez de registres généraux,
// déborde sur la pile et attrape BP, le pointeur de trame dont le runtime Go se
// sert pour dérouler la pile sur un signal. Pour y revenir il faudrait d'abord
// remplacer les `tile` pointeurs de ligne par un pointeur et des index
// d'échelle. kernel_avx2_amd64_test.go refuse un `.s` qui mentionne BP.
const tile = 6

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

	// remaining : combien de sorties restent à produire. Un compteur qui
	// décroît remplace le trio (indice courant, borne tuilée, outDim), ce qui
	// vaut deux registres généraux de moins — à `tile` = 6 c'est la différence
	// entre un noyau qui tient dans les quatorze registres utilisables et un
	// noyau qui déborde sur la pile.
	//
	// Il remplace surtout un arrondi qui n'en était pas un : `outDim & ^(tile-1)`
	// n'arrondit au multiple inférieur que pour une puissance de deux. À tile = 6
	// il laissait passer une tuile de trop et le noyau lisait six lignes de
	// poids au-delà de la matrice, une fois sur trois selon l'allocation.
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
