<!-- Lot du plan tasks/plan-amelioration-2026-09b/README.md. -->

# Lot C — Moteur d'évaluation (gammonNet, race, EPC, lot d'analyse)

État vérifié le 2026-09-02 : couverture `engine` 67,0 %, `gammonnet` 87,9 %,
`race` 87,3 %. Profil d'une décision 2-ply **au score** : `Value` (videau)
44,8 % cumulé dont `levelSolve` 38,9 %, `EvaluateBatch` 48,4 % ;
`BenchmarkDecision2Ply` 702 ms contre `…Match` 1 036 ms — l'écart est
entièrement du videau. Allocations : 85 en money, 1 472 au score.

Règle du lot : une modification qui change ce que le moteur calcule passe
d'abord par gammonNet amont (son ADR-0003, critère « le gain survit au
changement de langage ») et par le gold ; une modification d'interface ou de
parallélisme reste ici et doit rester bit-identique.

C.1 à C.6 = **étape 1** (correction, tests) ; C.7 à C.12 = **étape 2** (perf,
architecture) ; C.13 = amont.

---

## C.1 — Trous de test qui laissent passer une valuation jamais exécutée [S] — correction (#188)

- `position.go:246` `terminalValue` : **0 %** de couverture ; `valueSweep`
  (`search.go:561`) ne prend jamais la branche `isOver()`. Aucune position où
  un coup termine la partie n'est recherchée dans toute la suite — c'est la
  valuation terminale au score (backgammon à 2-away) qui n'est jamais exécutée.
- MET : `met_test.go` ne teste que le convertisseur ; ni une valeur de
  `kazarossXG2PreCrawford` contre la publication, ni `MET[i][j] + MET[j][i] == 1`,
  ni le recouvrement Zadeh/Kazaross à 24/25 ; `GnuBGGetME` écrête au-delà de
  64-away (`met.go:361-366`) sans test.
- Crawford `[1,1]` non testé (`domaineval.go:335`) ; `RegimeEvaluated`
  (`race/eval.go:22`) sans test unitaire (pas de `gammonnet_eval_test.go`).
- [x] Trois positions terminales (single, gammon, backgammon) au corpus gold ;
      test unitaire `terminalValue` au score contre `terminalEquity` en money.
- [x] Trois tests de propriété sur la MET ; test de bord 64-away.
- [x] Cas `[1,1]` ; `internal/gui/gammonnet_eval_test.go` pour les trois régimes.
- [x] `FuzzLoadBGNN` (format binaire à longueurs, `network.go:88-197`) et
      `FuzzDecide` (probs × états de match arbitraires : NaN, `count < 2`,
      divisions dégénérées de `segment`) ajoutés à `fuzz.yml`.

## C.2 — Le parallélisme est un invariant que rien ne garde entre deux tags [S] — fiabilité (#189)

`build.yml:58-63` : le shard `gammonnet` tourne sans `-race`, le shard `rest`
l'exclut ; `TestParallelSearchIsBitIdentical` est sauté en `-short`
(`search_test.go:283-285`) donc sur chaque push ; le gold de recherche
(`gold_test.go:80`, `BLUNDERDB_GOLD`) ne tourne dans **aucun** job. `test-os`
macOS (arm64, repli pur Go) est en `continue-on-error` et `-short`.
- [x] Job `go test -race -run 'Parallel|Concurrent|Batch' ./pkg/blunderdb/engine/gammonnet/`
      (court par construction) sur chaque push. Fait : job `race-gammonnet`
      (`build.yml`), ~45-65 s.
- [x] Variante 1-ply non-`short` de la bit-identité parallèle (~20 ms). Fait :
      `TestParallelSearchIsBitIdenticalOnePly` (`search_test.go`).
- [x] Job hebdomadaire (dans `fuzz.yml` ou un `nightly.yml`) : `BLUNDERDB_GOLD=1`
      sans `-race`, plus `kernel_identity_test` sur macos (arm64). Fait dans
      `fuzz.yml` : jobs `gammonnet-gold` et `gammonnet-kernel-identity-arm64`
      (pas de `nightly.yml` créé — E.12/#228 décidera où vont les autres
      jobs hebdomadaires).
- [x] Retirer `continue-on-error` de `test-os` (E.1) — préalable de #151. Fait
      dans E.1/#217.

## C.3 — Incohérences money/match visibles à l'utilisateur [S] — correction (#190)

- `internal/gui/gammonnet_eval.go:298` : `evaluateRaceRegime` construit
  `DefaultConfig(ply)` (`UseMatch=false`, `UseCube=false`) puis `:338` `Decide`
  avec l'état de match : distribution issue d'un arbre money cubeless,
  verdict tarifé au score cubeful. ADR-0023 le nomme « Open ». Une ligne :
  `ConfigForPosition`.
- Deux prédicats « money » divergents : `gammonnet_eval.go:293`
  (`!= -1`) vs `domaineval.go:145` (`< 0`).
- `domaineval.go:326-329` : un score mi-money mi-match remonte « non
  évaluable à ce score » — message à préciser.
- Libellé de la colonne Équité jamais fait (ADR-0016 pt 6) : `fr.json:311,561`
  disent « Équité (millièmes) » / « Équité » ; il faut « Équité (money) » /
  « Équité (match) » en 9 langues — la seule chose qui explique le changement
  d'échelle à l'utilisateur.
- `race.Money` → `race.CubeVerdict` (`money.go:36`) : le nom ment sur trois
  champs sur quatre depuis ADR-0016 ; plus rien ne bloque.
- Cube max, Jacoby, beaver : trois drapeaux en base, aucun affiché à côté du
  verdict.
- [ ] Les six points ; test `gammonnet_eval_test.go` du régime évalué au score.

## C.4 — Le lot d'analyse retente à l'infini ce qu'il ne peut pas évaluer [S] — correction (#191)

`db_gammonnet_batch.go:279-295` : le commentaire promet « nil, nil = rien à
écrire (danse, état de videau hors MET) » mais `EvaluatePositionWith` renvoie
`ErrNotEvaluable` → `failed: true` → la position est retentée à chaque passe et
jamais comptée. `:259-274` : `done++` inconditionnel, `onProgress` non ; fin de
lot « Done. » sans résumé (`cli_analyze.go:108`). `isStaleGammonNetOnly`
(`:106-127`) ignore `AnalysisDepth` : passer de 0-ply à 2-ply ne rend rien
périmé. `AnalyzeStaleGammonNet` (`:164`) n'a **aucun déclencheur** (0 appel
front, 0 CLI) alors qu'`EngineVersion` a bougé trois fois en trois jours.
- [ ] `errors.Is(err, ErrNotEvaluable)` = « rien à écrire », compté à part
      (`refused`) ; résumé de fin `evaluated / refused / failed` en CLI, GUI
      (toast) et serveur (événement NDJSON final).
- [ ] Progression monotone sur les positions **traitées**, échecs inclus dans
      le résumé.
- [ ] `AnalysisDepth` entre dans le prédicat de péremption, profondeur cible
      en paramètre.
- [ ] `blunderdb analyze --stale` et bouton « Ré-analyser les positions
      périmées (N) » dans l'onglet gammonNet de la configuration ; route
      `/v1/gammonnet.sweepStale`. Documenté dans `manuel.rst`.
- [ ] `NOT IN (sous-requête)` → `NOT EXISTS` (`:34,48`).

## C.5 — Efficacité du videau : deux questions de modèle à trancher en amont [S vérif / M fix] — correction (#192) — FAIT (ADR-0029)

- `search.go:588` valorise chaque feuille avec `s.cfg.CubeX` fixé à la racine
  (`domaineval.go:159`, `DefaultEfficiency(owner)`) alors que `owner` est
  miroité à chaque ply (`rankPlays:400`, `oneRoll:654`) : 0,566 contre 0,687
  un ply sur deux, sur chaque feuille.
- `cube.go:707-708,746-747` : la branche « double pris » (`eDT`, videau
  détenu par l'adversaire) est tarifée à l'efficacité du propriétaire courant.
- [x] Lu : le C **ne miroite pas** `cube_x` (`gn_search.c:299,740` le passe
      figé à côté du propriétaire miroité ; `gn_cube.c:754,790,810` tarife
      `e_dt` à l'efficacité de l'appelant). Ce n'est **pas** un trou de port
      mais une divergence de modèle — assumée, chiffrée et tranchée par
      **ADR-0029** : les trois `x` sont des coefficients de BRANCHE ajustés
      contre trois colonnes d'un oracle exact, gnubg indexe par classe de
      position (P6), on garde le nôtre et on ne corrige pas ici. Commentaires
      explicites posés à `SearchConfig.CubeX`, `DefaultEfficiency` et `eDT` ;
      correctif proposé pour l'amont écrit dans l'ADR (point 4), non appliqué.
      Mesuré sur 669 décisions réelles (`cube_efficiency_measure_test.go`,
      `BLUNDERDB_MEASURE_CUBEX`) : 55,2 % à videau tourné, 0,005 d'équité
      normalisée par feuille, **0 verdict basculé sur 604**, **0 coup changé
      sur 60**.
- [x] Rejoués (`TestMeasureGateRedCasesAtDepth`,
      `BLUNDERDB_MEASURE_GATE_DEPTH`) : **inchangés** à 2-ply k=12, 2-ply sans
      élagage et 3-ply k=12 — 0,0552 et 0,0738, mêmes coups. Et il n'y a pas
      de « MET de XG » à essayer : `engine/met.go` **est** Kazaross-XG2. La
      troisième explication tombe donc comme les deux précédentes ; résultat
      écrit dans l'en-tête de `integration_gate_test.go`. Reste le jugement du
      réseau sur deux plateaux, qui n'est pas une question de réglage.

## C.6 — Deux règles de verdict pour la même question [M] — cohérence (#193)

`race/money.go:53-79` (`MoneyFromEntry`, 3 voies, jamais `TooGood`) vs
`gammonnet/cube.go:646-658` (`Verdict`, 4 voies, exporté « pour être partagé »,
0 usage). ADR-0020 dit « une décision de videau a une seule forme ».
- [ ] `MoneyFromEntry` passe par `Verdict` ; ou l'ADR-0020 est amendée pour
      dire pourquoi la table exacte a droit à sa règle. Choix recommandé : une
      seule règle.
- [ ] Beaver : `HasBeaver` est haché et jamais lu par `Decide` ; modéliser
      « take → beaver » (money, videau centré, `eDT > +1` pour le preneur) après
      décision amont (spec §2), ou dire dans la doc que le drapeau est
      décoratif. Lié à B.3.

---

## C.7 — Le videau est devenu le poste dominant au score : forme close de `levelSolve` [M amont + S port] — perf (#194)

`cube.go:509-526` bissecte 60 fois une fonction **linéaire par morceaux et
monotone** dont les 3-4 segments sont connus (`levelLive`, `:471-497`) :
le rapport [P6](../../docs/recherche/P6-videau-janowski.md) confirme que **la bissection
est superflue** — Janowski donne take point, cash point et too-good en forme close, et au
score le take point live est un produit récursif fini, donc encore une forme close.
38,9 % du profil d'une décision 2-ply au score. L'inversion en forme close est
O(1) et exacte. Gain conceptuel → **gammonNet d'abord** (`gn_cube.c`, mesure,
spec), régénération de `cube_gold.bin`, puis port. Le « valuer le videau par
lot » du BACKLOG est la seconde marche.
- [ ] Amont : PR gammonNet + mesure ; tag.
- [ ] Ici : port, `cube_gold.bin` régénéré, `EngineVersion` inchangée si le
      résultat est bit-identique (il doit l'être ; sinon c'est une
      Configuration et les bases sont périmées).
- [ ] Benchmarks manquants **avant** : `BenchmarkProbs2Ply`,
      `BenchmarkCubeDecisionAtScore`, `BenchmarkBuildLevels`, débit du lot.

## C.8 — La décision de videau du panneau n'utilise qu'une fraction des cœurs [M] — perf (#195)

`search_probs.go:127-155` : les 21 lancers de la racine sont sériels, chacun
déclenche `deepenLevel` avec un candidat → 21 barrières de 21 tâches là où
`Plays` en fait une de 63 ; `probsAt(level+1)` est intégralement sériel.
`rollsInParallel` (`search.go:816-822`) est du code mort — exactement la file
dont `probsAt` a besoin.
- [ ] Aplatir la boucle des lancers dans la même file (`rollTask`) ; distinguer
      « profondeur de récursion » et « suis-je à la racine ».
- [ ] Supprimer le paramètre `parallel` constant de `positionEquity`.
- [ ] Bit-identité vérifiée par le test de C.2 étendu à `Probs`.

## C.9 — Le panneau construit trois pools par frappe [M] — latence, mémoire (#196)

`gammonnet_eval.go:160,228,306` : `EvaluatePosition` (`WithWorkers`),
`preRollFacts` (second `Searcher`, refait un `Probs`), `evaluateRaceRegime`
(troisième). `WithWorkers` alloue `n` `Searcher` neufs avec un `evalCache` de
65 536 entrées (≈ 3,7 Mo) chacun : sur 16 cœurs, ~190 Mo alloués et jetés par
frappe, caches froids ×3. Les chiffres « +36 % » et « 376 µs » des commentaires
(`:19-20,29-30`) datent d'avant le noyau.
- [ ] Un pool réutilisé par l'`App` (motif `NewBatchSearcher` + `Reconfigure`,
      qui garde le cache), partagé par les trois recherches.
- [ ] `defaultCacheLog2` réduit pour les ouvriers (mesurer : +4,5 % d'évals en
      double aujourd'hui, 59 Mo par recherche à 16 cœurs).
- [ ] État du panneau en champs de `App`, plus en variables de paquet
      (`gammonnet_eval.go:103-106`, `gammonnet_batch.go:24-27`).
- [ ] Le lot cède la main en au plus `jobs` positions (~5 s à 16 jobs) pendant
      que le panneau lance son propre pool : réduire `jobs` quand une évaluation
      interactive est en vol.
- [ ] Commentaires de coût mis à jour (0-ply ~1 ms, 2-ply NumCPU 72 ms).

## C.10 — Allocations et conversions résiduelles [S] — propreté (#197)

- `swapMatchState` : 1 472 allocations par décision au score (`search.go:453-459`)
  → `[MaxPly+2]MatchState` pré-calculé (l'état ne dépend que de la parité).
- `notationIndex` reconvertit chaque coup légal par `FromDomain`
  (`domaineval.go:437`) : 62,8 µs et 6,6 Ko par position, 5,5 s et 580 Mo de
  churn sur 88 000 positions, pour de l'affichage → notation depuis le `Play`
  du moteur (`moves.go:51`).
- `raceCubeStateFor` duplique `race.cubeStateFor` (`gammonnet_eval.go:377-387`)
  → exporter `race.CubeStateFor`.
- [ ] Les trois, bench avant/après.

## C.11 — Surface exportée morte [S] — lisibilité (#198)

`Equity`, `TakePoint`, `Verdict`, `MoneyEquity`, `InvertProbs`, `Counters`,
`BatchFill`, `ResetCounters`, `KernelName`, `KernelError`,
`EmbeddedPruneNetwork`, `NewSearcherWith`, `Reconfigure` : 0 usage hors
paquet, chacun avec une justification pour un appelant qui n'existe pas.
- [ ] Dé-exporter ce qui n'a que des usages de test ; garder `Verdict`/
      `TakePoint` **et** les utiliser (C.6) ; les compteurs restent pour le probe
      mais couverts par un test.
- [ ] `analysiscodec.go` / `positioncodec.go` / `bearoff_export.go` /
      `epc.go:61 PipCounts` à 0 % : ce sont les fonctions qui écrivent les
      colonnes scalaires (le bug `CommitImportDatabase` d'hier) → tests.

## C.12 — Documentation du moteur [S] — navigation (#199)

- `CLAUDE.md:167-170` ne cite ni `engine/gammonnet` (5 000 l.) ni `met.go` ;
  `:189` `cmd/` sans `calibrace`.
- Paquet `engine` sans commentaire de paquet (`epc.go`, `bitboards.go`,
  `zobrist.go`, `met.go`).
- `cube.go:382-386` (« use_cube is a follow-up ») périmé ; `tasks/gammonnet-perf/README.md`
  § « ce qui n'explique pas ces chiffres » liste comme ouvert ce qui est fait.
- ADR à écrire : « GPU/WASM sont écartés » (ADR-0024 fait de la reproductibilité
  bit-à-bit la définition de « périmé » ; une ligne évite la question).
- [ ] Les quatre points.

---

## C.13 — Amont gammonNet (décisions, pas des fiches ici) (#200)

À porter dans le dépôt gammonNet avec leur jauge de force ; blunderDB suit.
1. Forme close de `levelSolve` (C.7).
2. Efficacité miroitée / branche DT (C.5) si le C fait pareil.
3. Réseau distillé 60-100 k MAC (P4 : ×5-9, priorité 1 amont) — nouvelle
   Configuration, `EngineVersion`, bases périmées.
4. Filtres de coups à seuil d'équité (movefilter) — approximatif, jauge.
5. Beaver/raccoon dans `Decide` (spec §2).
6. Réseau de course dédié pour la zone sans contact hors bearoff pur
   (`race/eval.go:80-84`).
7. NEON arm64 (#151) : après C.2 (filet macOS) et après avoir mesuré le coût
   réel sur M1 (P2 : NEON sans FMA ≈ 0,5× d'un FMLA).

Les fonctionnalités moteur orientées produit (rollouts, 3-ply mesuré, PR sur
matchs non analysés, MET configurable, cache persistant, rapport XG vs
gammonNet) sont au lot I et J.

---

## Résumé du lot

| Fiche | Effort | Étape |
|---|---|---|
| C.1, C.2, C.3, C.4 | S | 1 |
| C.5, C.6 | S vérif / M | 1 |
| C.7 | M amont + S | 2 |
| C.8, C.9 | M | 2 |
| C.10, C.11, C.12 | S | 2 |
