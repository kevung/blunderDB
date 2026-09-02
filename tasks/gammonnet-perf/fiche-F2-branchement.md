<!-- Fiche du plan tasks/gammonnet-perf/README.md. Le contexte, les décisions
et le tableau d'ensemble vivent dans le README ; cette fiche ne porte que son
propre travail et ses propres chiffres. -->

# F2 — La recherche alimente le noyau [M] — #146 — **FAIT** (branche `feat/gammonnet-search-batches`)

- [x] `shallowFill` trie puis groupe : positions terminales, hits de cache et encodages
      refusés sortent du lot (traitement inchangé), les survivants sont encodés dans le
      brouillon du `Searcher` et partent par tranches de `EvalBatchWidth`. La convention du
      lot partiel reste celle du noyau — au-delà de `n`, les voies dupliquent la position
      `n-1` — donc l'appelant ne raisonne jamais sur la queue. `flushBatch` redistribue et
      tient les mêmes compteurs : `evals` + `cache.store` pour le grand réseau, `pruneEvals`
      pour l'élagage. Le petit réseau d'élagage passe par le même chemin et **son contrat ne
      bouge pas** : ni lecture ni écriture du cache (`cache.go`).
- [x] Le tampon de lot (`batchFeat`, `batchProbs`, `batchOf` : 6,3 Ko) vit dans le
      `Searcher`. **Zéro allocation ajoutée par nœud**, vérifié : `-benchmem` passe de
      546 056 o / 7 430 all. à 1 035 400 o / 7 440 all. à `-benchtime 1x`, mais à
      668 392 o / 7 432 all. à `-benchtime 4x` — le surcoût s'amortit exactement par
      quatre, c'est donc le brouillon *paresseux* des deux `Evaluator` (F1, ~465 Ko),
      une fois par `Searcher`, et rien sur le chemin chaud.
- [x] ~~Le probe donne le remplissage~~ — **fait, et D4 est tranchée** : 84,3 % à 2-ply
      canonique (section 1), au-dessus du seuil de 70 %. Le regroupement des 21 lancers
      n'aura pas lieu.
- [x] `TestParallelSearchIsBitIdentical`, `gold_test.go` (`BLUNDERDB_GOLD=1` : 85 + 123
      décisions, max|Δ| 7,153e-07 et 3,902e-07), `cube_gold_test.go`, `parity_test.go`,
      `kernel_identity_test.go` : **verts sans retouche** ; `go vet ./...`, `golangci-lint`
      v2.11.4 (0 issue), `go test ./...`, `-race` sur le paquet : verts. L'`integration_gate`
      échoue **par les deux mêmes décisions qu'avant** (0,0552 et 0,0738, à la quatrième
      décimale, question de profondeur/MET documentée dans le test) et tourne en **105 s au
      lieu de 765 s** — ×7,3 sur une charge réelle.
- [x] **Cible mesurée** : décision 2-ply canonique série **0,345 s** contre 5,85 s avant —
      D1 (< 1 s) **atteinte avec un facteur 3 de marge**.

  `BenchmarkDecision2Ply -benchtime 1x -count 5`, cinq décisions à froid : **5,33 / 5,58 /
  5,85 / 5,89 / 5,98 s** avant, **0,341 / 0,342 / 0,347 / 0,347 / 0,349 s** après. Soit **×17**, contre ×25 pour le noyau en isolation : l'écart est ce que la décision
  passe hors réseau, qui devient visible exactement comme ADR-0024 l'annonçait (2,4 %
  avant, ~15 % maintenant — c'est le chantier #150). **Machine NON oisive** (charge 4 à 6,
  d'autres agents sur le dépôt) ; `BenchmarkEvaluateBatchKernelSiblings` sort entre 13,8
  et 23,2 µs/position dans les deux séries, donc les deux colonnes ont vu la même machine.

  Probe correspondant (`BLUNDERDB_PROBE=1`, noyau avx2, mêmes conditions) :

  | configuration | durée | évals | élagage | cache | remplissage |
  |---|---|---|---|---|---|
  | 0-ply k=12 | 1 ms | 12 | 16 | 0 | 87,5 % |
  | 1-ply k=12 sans filtre | 70 ms | 2 676 | 4 934 | 155 | 81,7 % |
  | 1-ply k=12 filtre 3 | 16 ms | 660 | 1 140 | 40 | 81,2 % |
  | 2-ply k=12 (0,1,3) | **360 ms** | 13 438 | 36 395 | 1 704 | **84,3 %** |
  | 2-ply k=12 (0,2,8) | 1,659 s | 61 836 | 152 005 | 9 472 | 83,0 % |
  | 2-ply k=12 (0,1,3), `WithWorkers(16)` | 117 ms | 14 038 | 36 395 | 1 104 | 84,3 % |

- [x] **Le remplissage réel vaut ce que #145 avait simulé** : 84,3 % à 2-ply canonique, à
      la décimale. `batchFilled`/`batchSlotted` mesurent désormais le lot au lieu de le
      simuler et disent la même chose — ce qui valide rétroactivement D4.
- [x] **Une seule dérive de compteur, sans effet** : 13 438 évaluations au lieu de 13 439,
      1 704 hits au lieu de 1 703. La version position par position rangeait le candidat
      *i* avant de chercher le *i+1* ; le lot cherche les huit avant de ranger les huit, et
      évince donc une entrée de moins dans cette table à adressage direct. Un hit rend les
      bits qu'un calcul aurait rendus (`cache.go`) : seuls les compteurs bougent. Et deux
      candidats d'un même appel ne sont jamais la même position (`moves_gen` déduplique par
      plateau résultant), donc rien ne s'évalue deux fois dans un lot.
