# SIMD sans CGO en Go pour une inférence MLP float32 bit-identique (état septembre 2026)

## TL;DR
- **Écrivez le noyau AVX2 avec `avo` pour amd64 et de l'assembleur Plan 9 à la main pour arm64/NEON (avo ne génère PAS d'arm64), gardez un repli Go pur activé par build tag, et NE migrez PAS encore vers `simd/archsimd`** (expérimental, `GOEXPERIMENT=simd`, amd64 seulement en Go 1.26 ; le NEON n'arrive qu'en Go 1.27 RC1). C'est la seule combinaison qui satisfait simultanément « pas de CGO », multi-arch et déterminisme bit-à-bit sur la toolchain Go 1.25 que vous imposez.
- **Le déterminisme bit-à-bit est atteignable et fondé sur la spécification Go** : la conversion explicite `float32(a*b)+c` est une barrière de fusion garantie par la spec (elle empêche la contraction FMA que le compilateur Go applique automatiquement sur arm64). Go n'active NI FTZ NI DAZ (amd64, config MXCSR 0x1F80) NI FZ (arm64, FPCR à 0), donc les dénormalisés sont traités de façon identique et conforme IEEE-754 sur les deux architectures — à condition qu'aucune bibliothèque C `-ffast-math` ne pollue ces registres, risque écarté ici puisque `CGO_ENABLED=0`.
- **Gain attendu** : un noyau AVX2 batché 8 voies sans FMA vise typiquement 4–8× sur la boucle scalaire Go actuelle (404 μs) ; l'absence de FMA coûte ~2× sur le débit arithmétique de pointe théorique, mais pour une couche 512×512 le facteur limitant est souvent la bande passante mémoire des poids, pas les FLOPs. [EXTRAPOLÉ]

## Key Findings

### 1. Écrire du SIMD en Go sans CGO — état 2026

**avo (github.com/mmcloughlin/avo)** reste l'outil de référence, mais c'est un générateur **x86-64 uniquement**. Le paquet public est `github.com/mmcloughlin/avo/x86` ; il n'existe aucun paquet `arm64`/`neon`. Le dépôt est mûr et actif — **3k étoiles et 96 forks** d'après la page GitHub mmcloughlin/avo (« Fork 96 · Star 3k »). Il supporte l'AVX-512 complet (jeux d'instructions étendus, registres de masque K), génère les fichiers `_amd64.s` + les stubs `.go`, gère l'allocation de registres virtuels et le calcul des offsets d'arguments. La dernière version étiquetée est **v0.6.0, publiée le 7 janvier** d'après les GitHub Releases (« v0.6.0 · 07 Jan … internal/data: bump versions by @mmcloughlin in #346 »). Les APIs sont déclarées « experimental / subject to change » — épinglez une version dans `go.mod`. [ÉDITEUR/DÉCLARÉ]

Pour **arm64/NEON**, vous écrirez l'assembleur Plan 9 arm64 à la main (fichiers `_arm64.s`). Documentation officielle : `https://go.dev/doc/asm`. Notables sur l'assembleur arm64 de Go : alignement automatique à 16 octets, instruction `PCALIGN` pour contrôler l'alignement, syntaxe et renommage de registres spécifiques. [ÉDITEUR/DÉCLARÉ]

**Paquet simd / archsimd (Go 1.26, `GOEXPERIMENT=simd`)** — proposition **golang/go #73787** (« simd/archsimd: architecture-specific SIMD intrinsics under a GOEXPERIMENT »), acceptée. Chronologie confirmée :
- **Go 1.26** a été publié le **10 février 2026** (annonce officielle Go, Carlos Amedee au nom de l'équipe Go, go.dev/blog/go1.26 : « Today the Go team is pleased to release Go 1.26 »). Les notes de version (go.dev/doc/go1.26) précisent : « Go 1.26 introduces a new experimental `simd/archsimd` package, which can be enabled by setting the environment variable `GOEXPERIMENT=simd`… It is currently available on the amd64 architecture and supports 128-bit, [256-bit and 512-bit] » vector types. Le paquet bas-niveau a été renommé de `simd` vers `simd/archsimd` (proposition #76473) pour réserver `simd` à l'API portable haut-niveau. [ÉDITEUR/DÉCLARÉ]
- **Go 1.27 RC1** est sorti le **23 juin 2026** (issue #73787, mise à jour du 06/23/2026 : « Go 1.27 RC1 is released, which includes the experimental support of the portable size-agnostic simd package (#78902)… and the inclusion of archsimd support for Wasm and ARM64, all under `GOEXPERIMENT=simd` »). Les notes 1.27 (go.dev/doc/go1.27) confirment : « This release revises the amd64 API and adds support for arm64 "Neon" 128-bit SIMD and WebAssembly 128-bit SIMD ». La proposition #78979 vise à activer l'API archsimd amd64 **par défaut** (sans GOEXPERIMENT) et note un unique changement d'API depuis 1.26 (`ShiftAll{Left,Right}` prenant un `uint8`). [ÉDITEUR/DÉCLARÉ]

Conséquence : sur **Go 1.25**, `archsimd` n'est **pas** utilisable — il n'existe qu'à partir de 1.26, il est expérimental, et le NEON n'y arrive qu'en 1.27.

**Autres approches** :
- `github.com/kelindar/simd` : fonctions math auto-vectorisées, AVX2 (amd64) + NEON (arm64, y compris Apple Silicon), code auto-généré. API élémentaire (`AddFloat32s`, etc.), pas un GEMM batché.
- `github.com/viterin/vek` (+ `vek32`) : opérations vectorielles float32/float64, **compilées depuis C++ avec `-ffast-math`** — rédhibitoire pour le déterminisme (fusion FMA et réassociation autorisées, FTZ possible).
- `github.com/pehringer/simd` : arithmétique élémentaire float32/float64 en Go assembly, AMD64 + ARM64.
- `github.com/ajroetker/go-highway` : « écrire une fois, exécuter partout » ; sur AVX2/AVX-512 utilise directement `simd/archsimd` de Go 1.26, sur ARM64 s'appuie sur un paquet asm « because `simd/archsimd` does not yet support these architectures ».
- `gonum.org/v1/gonum/internal/asm/f32` & `f64` : noyaux BLAS (Axpy, Gemv, Dot…) en assembleur Plan 9 maintenus à la main ; bonne référence de structure mais orientés float64 et non centrés sur le batch.
- `github.com/klauspost/cpuid` et `golang.org/x/sys/cpu` pour la détection à l'exécution.

### 2. Déterminisme bit-à-bit

**La barrière de fusion FMA.** La spec Go (« Floating-point operators ») garantit : *« An explicit floating-point type conversion rounds to the precision of the target type, preventing fusion that would discard that rounding. »* Donc `float32(a*b) + c` **empêche** la contraction en FMA, alors que `a*b + c` **peut** être fusionné. C'est capital car **sur arm64 le compilateur Go détecte le motif `x*y + z` et émet automatiquement un FMADD** (issue #71204, confirmée : « On arm64, the compiler detects the `x*y + z` pattern and automatically uses FMA »). Sur amd64, le compilateur Go **ne** fusionne **pas** automatiquement `x*y+z` (même avec `GOAMD64=v3`, cf. #71204) — d'où la divergence historique amd64 vs arm64. La proposition #17895 a fixé la règle : les conversions explicites (et affectations comme `t := float64(x*y); t += z`) forcent l'arrondi ; les **parenthèses** ne forcent **pas** l'arrondi. [ÉDITEUR/DÉCLARÉ]

Précédent de référence : `golang.org/x/image/vector` utilise exactement cette technique, avec ce commentaire (cité dans mdempsky/unconvert #40) : *« explicit in order to disable the compiler's Fused Multiply Add (FMA) instruction selection… This package aims to have bit-exact identical results across all GOARCHes, and across pure Go code and assembly, so it disables FMA. »* [ÉDITEUR/DÉCLARÉ]

**Attention `unconvert`/linters** : les outils de « conversions redondantes » (mdempsky/unconvert #40, gopls) veulent supprimer ces `float32(...)` en apparence inutiles — protégez-les, sinon le déterminisme casse silencieusement.

**Désactiver la fusion globalement (débogage)** : il existe un flag compilateur `-gcflags=all=-d=nofma` (initialement proposé comme `-nofma`), ajouté pour « debugging differences in floating point behavior across architectures » (revue golang-codereviews). À utiliser pour *diagnostiquer* une divergence, **pas** comme garantie de production — préférez les conversions explicites, qui sont dans la spec, pas dans un flag interne. Notez aussi l'existence de #36971 : avec des `complex128`, il n'est **pas** possible d'empêcher la FMA — restez donc en float32 scalaire, pas en complexes.

**Registres de contrôle FP** (confirmé via l'ABI interne Go `src/cmd/compile/abi-internal` et `src/runtime/cgo/abi_amd64.h`) :
- **amd64** : la configuration de contrôle MXCSR de l'ABI Go correspond à **0x1F80** — toutes exceptions masquées (bits 7–12), arrondi au plus proche, **FZ (bit 15)=0** et **DAZ (bit 6)=0** (table verbatim de l'ABI : « FZ | 15 | 0 | Do not flush to zero » … « DAZ | 6 | 0 | Do not zero de-normals »). Donc **les dénormalisés ne sont PAS flushés** ; comportement IEEE-754 complet. **Correction importante** (vérifiée sur la source master) : le runtime Go actuel **ne contient pas** d'instruction `LDMXCSR $0x1F80` explicite dans `rt0_go` de `src/runtime/asm_amd64.s` — Go **hérite** de la valeur par défaut d'initialisation du processus ELF et **assume** qu'elle tient ; `abi_amd64.h` dit : *« MXCSR matches the Go ABI, so we don't have to set that, and Go doesn't modify it, so we don't have to save it. »* [ÉDITEUR/DÉCLARÉ]
- **arm64** : la configuration FPCR de l'ABI Go a **tous les champs de contrôle à zéro** — **FZ (bit 24)=0** (pas de flush-to-zero), **DN (bit 25)=0** (NaN d'opérandes propagés, mode default-NaN **désactivé**), **FIZ (bit 0)=0**, arrondi au plus proche, tous les pièges désactivés. Aucune écriture `MSR FPCR` dans `src/runtime/asm_arm64.s`. [ÉDITEUR/DÉCLARÉ]

Conséquence directe : **FTZ/DAZ/FZ étant OFF sur les deux architectures, le traitement des dénormalisés est identique et conforme IEEE-754**, ce qui est nécessaire au bit-à-bit. Le grand risque est qu'une bibliothèque C compilée `-ffast-math` installe un constructeur global qui met FTZ/DAZ (amd64) ou FZ (arm64) et pollue tout le processus (documenté hors-Go : moyix « Someone's Been Messing With My Subnormals! » ; bug analogue Mixxx #16126, « ARM64: Audio thread missing flush-to-zero (FPCR FZ bit) »). **`CGO_ENABLED=0` élimine ce risque** — un argument de plus pour l'assembleur Plan 9. Puisque Go ne « répare » pas activement MXCSR/FPCR, l'absence de cgo est votre meilleure garantie que ces registres restent à leur valeur par défaut conforme.

**Différences IEEE-754 résiduelles entre chemins** : tant que (a) aucun FMA n'est émis, (b) l'ordre de sommation est identique par voie, (c) FTZ/DAZ/FZ sont OFF, `VMULPS`+`VADDPS` (AVX2) et `FMUL`+`FADD` (NEON) produisent le même résultat bit-à-bit pour des opérandes normalisés ET dénormalisés en arrondi au plus proche. La divergence connue restante concerne la **charge utile (payload) et le signe des NaN générés** (0/0, ∞−∞), qui peuvent différer entre x86 et ARM. [HYPOTHÈSE] : pour un MLP ReLU à entrées et poids finis, aucun NaN ne devrait être *généré*, donc ce cas ne se présente pas en pratique — à confirmer par un test avec entrées extrêmes.

**Points NON TROUVÉS** (déclarés explicitement) : aucune issue golang/go spécifique documentant une corruption/non-restauration de MXCSR par un appel cgo/signal/changement de goroutine ; aucune issue golang/go spécifique sur FPCR arm64 modifié par du C via cgo. Les issues golang/go de divergence flottante arm64 vs amd64 identifiées (#44528, #36536, #69789, #40981) portent sur la FMA ou l'arrondi float→int, pas sur une pollution de FPCR/MXCSR.

### 3. Projets Go faisant exactement cela

- **Sourcegraph / Cody** (blog « From slow to SIMD », Camden Cheek, 17 janvier 2024) : dot-product float32 puis int8, écrit en assembleur Go à la main (AVX2), justement pour éviter CGO. Camden Cheek : *« I try hard to avoid Cgo whenever possible for many reasons… one of those reasons is that Cgo imposes a performance penalty, and performance of this snippet is paramount »* (benchmarks sur « Intel Xeon Platinum 8481C 2.70GHz CPU… c3-highcpu-44 GCE VM »). Référence pédagogique majeure : loop unrolling, bounds-check elimination (`a[i:i+4:i+4]`), passage à int8, détection à l'exécution + repli Go. [ÉDITEUR/DÉCLARÉ ; MESURE auto-déclarée]
- **kelindar/search** + article « Building a Faster, Leaner Vector Search Library in Go » (kelindar.dev, oct. 2024) : assembleur Go généré depuis C via `kelindar/gocc`, dot-product SIMD ~8 ns (AVX2). [MESURE auto-déclarée]
- **Écosystème échecs NNUE en Go** : `ChizhovVadim/CounterGo` (moteur UCI en Go pur) ; `adamtwiss/gochess` (NNUE v5, SIMD AVX2 x86-64 + NEON arm64, mais CGO pour Fathom/Syzygy — pas pour l'inférence). La note la plus pertinente pour vous : le moteur **Coda** (C++) obtient un noyau NNUE « **Bit-identical output — pure throughput** » en changeant de kernel NEON (USDOT/SDOT) sélectionné à l'exécution par feature CPU — preuve que l'objectif « bit-identique + dispatch par feature » est une pratique établie. La plupart des moteurs NNUE utilisent int8/int16 (quantifié), pas float32 — donc pas de projet Go public correspondant *exactement* à votre cas float32 batché. [ÉDITEUR/DÉCLARÉ]
- **gonum internal/asm** : structure canonique — `_amd64.s` + `_arm64.s` + fallback `.go`, sélection par build tags, tests d'identité asm vs Go pur.
- **go-highway (ajroetker)** : montre le patron de dispatch AVX2/AVX-512/NEON/fallback + wrappers `//go:noescape`.

### 4. Éléments pratiques

**Performance.** Boucle scalaire actuelle : 404 μs/évaluation. Un noyau AVX2 8-voies sans FMA vise 4–8× ; l'absence de FMA divise par ~2 le pic FLOP théorique (une FMA = 2 FLOP en une instruction ; VMULPS+VADDPS = mêmes 2 FLOP en 2 émissions). Pour cacher la latence de `VADDPS` (~3–4 cycles, chaînée par la dépendance de l'accumulateur), on utilise normalement 4–8 accumulateurs indépendants — **mais cela change l'ordre de sommation**, ce que votre contrainte interdit par voie : gardez **un seul accumulateur par voie SIMD**. Registres YMM = 256 bits = 8×float32. Couche 512×512 = 262 144 poids × 4 o = **1 Mio de poids**, souvent au-delà du L2 de plusieurs cœurs, donc **la lecture des poids domine** ; vectoriser sur le batch permet de réutiliser chaque poids (diffusé par `VBROADCASTSS`) sur toutes les positions du batch. [EXTRAPOLÉ]

**Disposition mémoire.** Format **`act[j][n]`** avec n = index de batch contigu (structure-of-arrays / batch entrelacé) : les 8 positions d'une voie SIMD sont adjacentes et chargées par un seul `VMOVUPS`. Alignement 32 octets souhaitable pour `VMOVAPS` (sinon `VMOVUPS` non aligné, coût quasi nul sur CPU récents). Pour garantir 32 o sans cgo : surallouer puis arrondir le pointeur de base au multiple de 32 via `unsafe`. Piège documenté (chewxy, « On the memory alignment of Go slice values ») : les petites slices ne sont PAS 32-aligned ; les grandes (≳ 64 éléments) le sont plus souvent car le runtime demande de la mémoire fraîche alignée. **Le GC Go actuel ne déplace pas les objets du tas** (non compactant) — un pointeur aligné vers une slice sur le tas reste valide ; en revanche **les objets sur la pile peuvent être déplacés lors de la croissance de pile** : n'appliquez pas l'astuce d'alignement à des tableaux stack-allocated passés à l'asm ; forcez l'échappement vers le tas (`make`, champ global). [ÉDITEUR/DÉCLARÉ + EXTRAPOLÉ]

**Pièges avo / Plan 9 connus.**
- **VZEROUPPER** : émettez-le avant chaque `RET` d'une fonction ayant touché des registres YMM, pour éviter la pénalité de transition AVX↔SSE. Sur Skylake+ chaque instruction non-VEX en état « upper dirty » est pénalisée ; note contre-intuitive (graphics32 #19) : sur Kaby Lake, un `VZEROUPPER` peut ralentir de 25 % les SSE non-VEX suivants — d'où l'alternative « tout en VEX ».
- **NOSPLIT & taille de frame** : les stubs avo sont `TEXT ·Fn(SB), NOSPLIT, $0-NN` ; NOSPLIT interdit le préambule de croissance de pile — sûr seulement si la frame est petite/nulle (sinon « nosplit stack overflow » au link).
- **go vet asmdecl** : vérifie la concordance déclaration ↔ offsets `FP` ; un bug historique d'avo ajoutait un padding erroné pour les signatures sans valeur de retour, déclenchant une erreur asmdecl (corrigé, issues #191/#195).
- **ABI** : depuis Go 1.17, Go utilise l'ABI de registres `ABIInternal` ; les fonctions asm en ABI0 sont appelées via des wrappers auto-générés (avo génère du ABI0 correct). Coût d'appel : quelques ns, et **l'assembleur Go n'est jamais inliné** — faites des noyaux « gros » (toute la couche, tout le batch) plutôt que par petit vecteur (argument central de la proposition #73787).
- avo ne peut pas émettre de `#ifdef GOAMD64_v3` (issue #235) — pas de compilation conditionnelle sur le micro-niveau amd64 dans le `.s` généré.

**Windows/amd64** : même ABI/assembleur que linux/amd64, rien de particulier pour un noyau pur-calcul (pas de syscalls). **darwin/arm64** : conventions arm64 standard ; le « darwin/universal » = compiler deux binaires (amd64 + arm64) puis `lipo -create` — étape d'empaquetage, les `_amd64.s`/`_arm64.s` étant simplement sélectionnés par GOARCH.

**CGO_ENABLED=0** : l'assembleur Plan 9 fonctionne parfaitement sans cgo (il fait partie de la toolchain, aucun compilateur C requis). C'est la **raison principale** de préférer avo/asm aux intrinsics C : conteneur statique multi-arch, cross-compilation triviale, et aucune pollution FP par `-ffast-math`.

## Details — squelette de noyau

### Générateur avo (amd64), `asm.go`
```go
//go:build ignore
package main

import (
    . "github.com/mmcloughlin/avo/build"
    . "github.com/mmcloughlin/avo/operand"
    . "github.com/mmcloughlin/avo/reg"
)

// out[i][n] = bias[i] + Σ_j w[i][j] * act[j][n]
// Vectorisation sur n (batch) : 8 positions par voie YMM, SANS FMA.
// Signature: func denseAVX2(out, bias, w, act []float32, I, J, N int)
// Hypothèses: N % 8 == 0 ; act en layout [J][N] (n contigu) ; w en [I][J].
func main() {
    TEXT("denseAVX2", NOSPLIT, "func(out, bias, w, act []float32, I, J, N int)")
    Doc("denseAVX2 calcule out[i][n]=bias[i]+sum_j w[i][j]*act[j][n], 8 voies batch, sans FMA.")

    outPtr := Load(Param("out").Base(), GP64())
    biasPtr := Load(Param("bias").Base(), GP64())
    wPtr := Load(Param("w").Base(), GP64())
    actPtr := Load(Param("act").Base(), GP64())
    I := Load(Param("I"), GP64())
    J := Load(Param("J"), GP64())
    N := Load(Param("N"), GP64())

    i := GP64(); XORQ(i, i)
    Label("loop_i")
    CMPQ(i, I); JGE(LabelRef("done_i"))

    n := GP64(); XORQ(n, n)          // n par blocs de 8
    Label("loop_n")
    CMPQ(n, N); JGE(LabelRef("done_n"))

    acc := YMM()                     // acc = broadcast(bias[i]) — UN seul accumulateur/voie
    VBROADCASTSS(Mem{Base: biasPtr, Index: i, Scale: 4}, acc)

    j := GP64(); XORQ(j, j)
    Label("loop_j")
    CMPQ(j, J); JGE(LabelRef("done_j"))

    wIdx := GP64(); MOVQ(i, wIdx); IMULQ(J, wIdx); ADDQ(j, wIdx)   // w[i*J+j]
    wb := YMM()
    VBROADCASTSS(Mem{Base: wPtr, Index: wIdx, Scale: 4}, wb)

    aIdx := GP64(); MOVQ(j, aIdx); IMULQ(N, aIdx); ADDQ(n, aIdx)   // act[j*N+n : +8]
    av := YMM()
    VMOVUPS(Mem{Base: actPtr, Index: aIdx, Scale: 4}, av)

    prod := YMM()                    // prod = wb*av ; acc = acc+prod  (SÉPARÉS, jamais VFMADD)
    VMULPS(wb, av, prod)
    VADDPS(prod, acc, acc)

    INCQ(j); JMP(LabelRef("loop_j"))
    Label("done_j")

    oIdx := GP64(); MOVQ(i, oIdx); IMULQ(N, oIdx); ADDQ(n, oIdx)   // out[i*N+n]
    VMOVUPS(acc, Mem{Base: outPtr, Index: oIdx, Scale: 4})

    ADDQ(Imm(8), n); JMP(LabelRef("loop_n"))
    Label("done_n")
    INCQ(i); JMP(LabelRef("loop_i"))
    Label("done_i")

    VZEROUPPER()                     // éviter la pénalité AVX->SSE
    RET()
    Generate()
}
```
> Ce squelette additionne dans l'ordre croissant de `j` (préserve l'ordre exigé) et n'utilise **jamais** de FMA. Un déroulage sur `j` avec plusieurs accumulateurs recombinés à la fin **changerait** l'ordre de sommation ; puisque votre contrainte impose un ordre strictement séquentiel par voie, gardez **un seul accumulateur** par voie (au prix de la latence de chaîne `VADDPS`).

### Sélection d'implémentation & stub Go
```go
// dense.go
package mlp

import "golang.org/x/sys/cpu"

//go:generate go run asm.go -out dense_amd64.s -stubs dense_stub_amd64.go

func Dense(out, bias, w, act []float32, I, J, N int) {
    if useSIMD { // amd64: cpu.X86.HasAVX2 ; arm64: cpu.ARM64.HasASIMD
        denseSIMD(out, bias, w, act, I, J, N)
        return
    }
    denseGo(out, bias, w, act, I, J, N) // repli portable
}

// denseGo — repli pur Go, bit-identique à la référence C.
func denseGo(out, bias, w, act []float32, I, J, N int) {
    for i := 0; i < I; i++ {
        for n := 0; n < N; n++ {
            acc := bias[i]
            for j := 0; j < J; j++ {
                // float32(...) explicite = barrière anti-FMA (spec Go)
                acc = acc + float32(w[i*J+j]*act[j*N+n])
            }
            out[i*N+n] = acc
        }
    }
}
```
Fichiers recommandés : `dense_amd64.s` (généré), `dense_stub_amd64.go` (généré, expose `denseSIMD` + `useSIMD=cpu.X86.HasAVX2`), `dense_arm64.s` (NEON écrit à la main), `dense_arm64.go` (stub `//go:noescape` + `useSIMD=cpu.ARM64.HasASIMD`), `dense_noasm.go` (`//go:build !amd64 && !arm64`, `useSIMD=false`), `dense.go` (dispatch), `denseGo` compilé pour **toutes** les arch et sélectionné quand la feature CPU manque.

### Test d'identité bit-à-bit
```go
func TestDenseBitIdentical(t *testing.T) {
    if !cpu.X86.HasAVX2 { t.Skip("pas d'AVX2") }
    I, J, N := 5, 512, 64
    out1 := make([]float32, I*N)
    out2 := make([]float32, I*N)
    bias := randSlice(I); w := randSlice(I*J); act := randSlice(J*N)

    denseSIMD(out1, bias, w, act, I, J, N)
    denseGo(out2, bias, w, act, I, J, N)

    for k := range out1 {
        if math.Float32bits(out1[k]) != math.Float32bits(out2[k]) { // BITS, pas de tolérance
            t.Fatalf("k=%d asm=%08x go=%08x", k,
                math.Float32bits(out1[k]), math.Float32bits(out2[k]))
        }
    }
}
```
> Comparez toujours via `math.Float32bits` (gère −0.0 vs +0.0, et évite qu'un `NaN==NaN` faux masque une divergence). Étendez avec des dénormalisés explicites et des valeurs proches de l'underflow pour vérifier que FTZ/DAZ sont bien OFF sur les deux chemins. CI multi-arch (GitHub Actions) : runner `ubuntu-24.04` (amd64) + runner ARM natif (`ubuntu-24.04-arm`) ou QEMU via `docker/setup-qemu-action` pour arm64 émulé, + `macos-14` (Apple Silicon) pour darwin/arm64.

## Tableau comparatif des options

| Option | Arch couvertes | CGO requis | Bit-à-bit maîtrisable | Maturité (sept. 2026) | Verdict |
|---|---|---|---|---|---|
| **avo (amd64) + asm Plan 9 main (arm64)** | amd64 + arm64 + tout (fallback) | Non | Oui (contrôle total : VMULPS+VADDPS, FMUL+FADD séparés) | avo mûr mais API « experimental », x86 only ; asm arm64 stable | **RECOMMANDÉ** |
| **Assembleur Plan 9 à la main partout** | toutes | Non | Oui | Stable | Alternative sans dépendance avo ; plus verbeux sur amd64 |
| **simd/archsimd (GOEXPERIMENT)** | 1.26 amd64 ; arm64 dès 1.27 RC1 | Non | En principe oui (intrinsics 1:1), garanties FMA à vérifier | Expérimental, absent en Go 1.25 | **À SURVEILLER**, pas maintenant |
| **kelindar/simd** | amd64 (AVX2) + arm64 (NEON) | Non | Non garanti (auto-généré, aucun contrôle d'ordre/FMA) | Actif | Non (ni bit-à-bit ni GEMM batché) |
| **viterin/vek** | amd64 (+ fallback) | Non | **Non** (`-ffast-math` → FMA + réassoc + FTZ) | Actif | **À éviter** |
| **CGO + intrinsics C** | toutes | **Oui** | Possible mais `-ffast-math`/FMA à désactiver, risque FTZ global | — | Exclu (CGO_ENABLED=0) |
| **gonum internal/asm** | amd64/arm64 | Non | Oui (asm main) | Mûr | Bonne référence de structure, pas clé en main pour le batch |

## Recommendations

**Étape 0 (immédiat — base de correction)** — Verrouillez d'abord le **repli Go pur** `denseGo` avec la barrière `float32(a*b)+c` sur CHAQUE produit et une addition séquentielle en `j` croissant. Écrivez le test d'identité bit-à-bit **contre la référence C** (pas seulement contre l'asm). Ajoutez un test/CI qui échoue si un `float32(...)` est retiré (protégez-le d'`unconvert`/`gopls`). Vérifiez sur amd64 ET arm64 que `denseGo` donne les mêmes bits — c'est le juge de paix. **Seuil de passage : 100 % des sorties bit-identiques sur les deux arch.**

**Étape 1 (noyau amd64)** — Générez `denseAVX2` avec avo (version épinglée). Un seul accumulateur YMM par voie batch pour respecter STRICTEMENT l'ordre de sommation. `VZEROUPPER` avant `RET`. Test d'identité asm vs `denseGo` (doit être exact, 0 différence de bits). Benchmark : **cible ≥ 4×** sur 404 μs (~100 μs). Si vous n'atteignez pas 4×, profilez la bande passante mémoire des poids AVANT d'ajouter des accumulateurs (qui changeraient l'ordre de sommation — à n'introduire QUE si vous relâchez la contrainte d'ordre, ce qui n'est pas votre cas).

**Étape 2 (noyau arm64/NEON)** — Écrivez `dense_arm64.s` à la main : `FMUL` puis `FADD` séparés (jamais `FMLA`, l'équivalent FMA NEON), 4 voies float32 par registre Q. Test d'identité vs `denseGo` sur un vrai runner ARM (Apple Silicon + Graviton/Ampere si possible). C'est le chemin le plus risqué en sourcing → validez-le tôt.

**Étape 3 (empaquetage)** — Matrice linux/amd64, windows/amd64, darwin/amd64, darwin/arm64 ; `lipo -create` pour l'universel macOS ; image conteneur `FROM scratch` avec binaire `CGO_ENABLED=0`. Vérifiez qu'aucune dépendance transitive n'active cgo.

**Seuils qui feraient évoluer la reco** :
- Passage à **Go 1.27 stable** avec `simd/archsimd` sorti de l'expérimental, NEON présent et garanties FMA documentées → réévaluez la migration (moins d'asm à maintenir, meilleure préemption/inlining).
- Si le profilage montre que vous êtes **borné mémoire** (probable pour 512×512), la vectorisation batch + réutilisation des poids compte plus que l'ISA ; AVX-512 (GOAMD64=v4) n'apporterait qu'un gain marginal.
- Si la contrainte d'ordre de sommation était un jour **relâchée**, plusieurs accumulateurs + FMA (via archsimd/intrinsics) débloqueraient ~2× supplémentaires.

## Caveats
- **Go 1.25 impose des limites** : `simd/archsimd` n'existe qu'à partir de 1.26 (amd64) et le NEON de 1.27 ; toute mention d'archsimd ci-dessus est donc prospective pour votre toolchain.
- Les **gains de performance (4–8×, ~2× coût FMA)** sont [EXTRAPOLÉ] d'après les principes SIMD et le blog Sourcegraph, **non mesurés** sur votre architecture 196→512→512→256→128→5. Mesurez.
- **NON TROUVÉ** : aucune issue golang/go dédiée à la corruption de MXCSR/FPCR via cgo/signaux/goroutines ; aucun projet Go public d'inférence MLP **float32** batché en avo avec test d'identité bit-à-bit correspondant exactement à votre cas (les moteurs NNUE Go sont majoritairement int8/int16 et/ou utilisent cgo pour d'autres composants).
- **NON TROUVÉ / non entièrement vérifié** : le comportement exact de génération de NaN par défaut (payload/signe) `VMULPS` vs `FMUL` en cas de NaN *produit* (non propagé) ; supposé non pertinent pour un MLP ReLU à entrées finies [HYPOTHÈSE].
- Le runtime Go **n'exécute pas** de `LDMXCSR`/`MSR FPCR` explicite au démarrage (vérifié sur master) : il hérite des valeurs par défaut du processus. Le résultat pratique (FTZ/DAZ/FZ OFF) est bien celui attendu, mais Go ne « répare » pas activement un MXCSR/FPCR pollué — d'où l'importance déterminante de `CGO_ENABLED=0`.
- avo a des **APIs déclarées instables** (« experimental phase ») ; épinglez la version dans `go.mod`.