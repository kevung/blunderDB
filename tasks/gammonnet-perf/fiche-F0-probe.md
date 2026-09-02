<!-- Fiche du plan tasks/gammonnet-perf/README.md. Le contexte, les décisions
et le tableau d'ensemble vivent dans le README ; cette fiche ne porte que son
propre travail et ses propres chiffres. -->

# F0 — Le probe devient la mesure de référence [S] — **FAIT** (#145)

- [x] `cost_probe_test.go` : colonnes *remplissage* et *noyau* ; compteurs `batchFilled` /
      `batchSlotted` sur `Searcher`, exposés par `BatchFill()`.
- [x] Ligne « 2-ply canonique, `WithWorkers(NumCPU)` ».
- [x] Au passage : `Counters()` **agrège les workers**. Sans cela un relevé parallèle
      rapportait une fraction du coût, et rapetissait à mesure qu'on ajoutait des cœurs —
      l'inverse de ce qu'un probe sert à faire.
- [x] `kernel.go` : `EvalBatchWidth`, `KernelName()`, `batchSlots()` — la couture que F1
      remplira, posée maintenant pour que les mesures d'avant et d'après soient
      comparables.
- [x] `BenchmarkEvaluateBatch` : coût **par position** sur un lot de positions *distinctes*.
      `BenchmarkEvaluate` réévalue une seule position, ce qui flatte tout noyau et
      flatterait le plus un noyau groupé.
- [x] `BenchmarkDecision2Ply`, sauté sous `testing.Short()`.
- [x] Chiffres de départ consignés en section 1.
