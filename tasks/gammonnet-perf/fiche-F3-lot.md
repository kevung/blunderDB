<!-- Fiche du plan tasks/gammonnet-perf/README.md. Le contexte, les décisions
et le tableau d'ensemble vivent dans le README ; cette fiche ne porte que son
propre travail et ses propres chiffres. -->

# F3 — Le lot analyse en parallèle [M] — #147 — **FAIT**

- [x] `db_gammonnet_batch.go` : `AnalyzeMissingWithGammonNet` et `AnalyzeStaleGammonNet`
      délèguent à `analyzeIDsWithGammonNet`, qui distribue les ids à `jobs` goroutines par
      compteur atomique ; chaque goroutine possède **un `Searcher` série réutilisé**
      (`gammonnet.NewBatchSearcher` + `Searcher.Reconfigure`), cache conservé d'une position
      à l'autre. Le commentaire faux des lignes 63-66 (« une seule recherche utilise déjà
      plusieurs cœurs ») est supprimé.
- [x] `yield` : chaque goroutine l'appelle avant de prendre une position ; « le lot cède en
      une position » est devenu « en au plus `jobs` positions », écrit dans le commentaire
      de la méthode et dans celui de `waitForInteractiveEvaluation`.
- [x] Écriture en base : une seule goroutine écrit (canal de résultats) ; `Database.mu` ne
      voit pas de contention et l'ordre d'écriture ne change pas le résultat.
- [x] Annulation par `ctx` conservée, vérifiée par chaque goroutine avant chaque position ;
      les résultats déjà calculés sont drainés et écrits (rien de perdu, rien de neuf).
- [x] Progression : compteur détenu par la goroutine d'écriture, donc monotone — l'indice de
      boucle ne veut plus rien dire à plusieurs.
- [x] `gammonnet.EvaluatePositionWith(searcher, …)` : la variante qui prend un `*Searcher`
      fourni ; `EvaluatePosition` reste la forme des appelants unitaires (panneau Eval,
      endpoint par position).
- [x] CLI `analyze --jobs N` (défaut `runtime.NumCPU()`), aide et `CLI_USAGE.md` ;
      serveur (`handlers_gammonnet.go`) et GUI : `NumCPU`, rien d'exposé.
- [x] Tests : `TestAnalyzeGammonNetParallelMatchesSerial` (mêmes analyses au bit à
      `jobs` ∈ {1, 2, NumCPU}), progression monotone, annulation avant/pendant, `yield`
      bloquant tout le lot, et côté moteur
      `TestEvaluatePositionWithReusedSearcherIsBitIdentical` (searcher réutilisé, cache
      chaud, référentiels alternés → résultat identique à un searcher neuf).

#### Mesure (2026-09-02)

Ryzen 7 PRO 6850U (**8 cœurs physiques / 16 threads**, mobile 15-28 W), Go 1.25.13.
200 positions réelles issues d'un match XG (`testdata/HsbtMarseille_…7p.xg`, analyses XG
effacées ; 184 sont évaluables, 16 refusées par le moteur et donc jamais écrites), base
restaurée depuis une copie identique avant chaque run, machine au repos (charge 1 min
observée entre 1,2 et 2,5 au départ des runs). Deux passes par valeur, la meilleure est
retenue ; la seconde passe reste à 3-11 % de la première.

**Le lot n'écrit pas un bit de différence.** Les cinq bases produites à 2-ply ont des
lignes `analysis` **rigoureusement identiques** (`sha256` sur `position_id`,
`best_cube_action`, `cube_error`, `best_move_equity_error` et les six taux, en écartant
les seuls horodatages d'écriture qui vivent dans le blob JSON) : `19ec886c791e716b…` pour
`jobs` = 1, 2, 4, 8 et 16. Idem à 1-ply.

**2-ply canonique (k=12), 200 positions** — la mesure de référence :

| `--jobs` | durée | facteur |
|---|---|---|
| 1 | 302,5 s | — |
| 2 | 168,2 s | ×1,80 |
| 4 | 120,6 s | ×2,51 |
| 8 | 74,1 s | **×4,08** |
| 16 | 59,0 s | ×5,13 |

**1-ply (k=12), les mêmes 200 positions** — le régime où le lot est court :

| `--jobs` | durée | facteur |
|---|---|---|
| 1 | 9,82 s | — |
| 2 | 5,50 s | ×1,79 |
| 4 | 3,43 s | ×2,86 |
| 8 | 2,31 s | ×4,25 |
| 16 | 2,07 s | ×4,74 |

**Ce que ces chiffres disent, et ce qu'ils ne disent pas.**

1. Le chiffre honnête est **×4,1 sur 8 cœurs physiques** (×5,1 sur 16 threads logiques),
   pas le ×8 que l'estimation de la section 4 espérait. Il faut donc corriger cette
   estimation : « ~2 h » pour 10 000 positions devient ~3 h 40.
2. **Ce n'est pas le lot qui plafonne, c'est la machine.** Deux indices concordants :
   le facteur est le même à 1-ply et à 2-ply (×4,25 et ×4,08) alors que le coût par
   position varie d'un facteur 40 — donc ni la goroutine d'écriture ni SQLite ne sont en
   cause, sinon le lot court serait le plus pénalisé ; et le passage de 8 à 16 rapporte
   encore ×1,26, ce qu'un code purement limité par la fréquence ne donnerait pas. Sur un
   6850U mobile, la fréquence tous cœurs soutenue est très inférieure au boost mono-cœur,
   et huit `Searcher` concurrents relisent chacun les 2,1 Mo de poids : le L3 partagé ne
   tient plus l'ensemble de travail, le SMT recouvre une partie de ces attentes mémoire.
3. Corollaire pour F1 : **un noyau qui divise le calcul par huit ne divisera pas le lot
   par huit**. Une fois la pression mémoire du réseau réduite (activations feature-major,
   poids compactés), c'est aussi ce facteur d'échelle qu'il faudra remesurer — il devrait
   remonter, pas rester à 4.

Recette, pour refaire la mesure : importer le match, effacer les analyses de 200 positions
(`DELETE FROM analysis WHERE position_id IN (SELECT id FROM position ORDER BY id LIMIT
200)`), puis `blunderdb analyze --db … --ply 2 --prune-k 12 --jobs N` sur une copie neuve
de la base à chaque fois (recopier par-dessus une base laissée avec son WAL la corrompt).
