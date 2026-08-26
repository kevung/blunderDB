# Fiche — Onglet « Joueurs » du panneau Stats (issue #116)

Spécification co-construite le 2026-08-26 (session de grilling sur l'issue
[#116](https://github.com/kevung/blunderDB/issues/116), demande d'iddqd43 :
« a table with overall stats, including PR, checkers, cube, wins, losses,
and luck » pour suivre tous les joueurs d'une compétition).

**Statut : LIVRÉ** — lots 0 à 5 mergés sur `main` le 2026-08-26. Chaque lot a
embarqué sa documentation, comme le veut la règle CLAUDE.md.

| Lot | Contenu | Ce que l'exécution a appris |
|---|---|---|
| 0 | Corrections de stats préalables | Le dénominateur Snowie filtré par joueur était bien deux fois trop petit ; corrigé sur les deux backends, seuil blunder harmonisé sur `>= 100`. Le test de contrat épingle l'additivité des deux Snowie ER d'un match. |
| 1 | `move.luck_mp`, schéma **2.15.0** | Convention et unité vérifiées empiriquement : le même match en `.xg` et en `.sgf` gnuBG, 189 lancers comparés un à un. Règle ajoutée en cours de route — XG écrit 0 aussi bien pour un lancer neutre que pour une chance non calculée, donc un match entièrement à zéro est lu comme dépourvu de données. **Les imports gnuBG ne fournissent encore aucune chance** : bug amont dans gnubgparser, voir FOLLOWUPS 9. |
| 2 | `StatsStore.PlayerTable` | Arithmétique partagée dans `storage/stats_playertable.go`. La convention de siège de `game.winner` a été validée sur un vrai match importé — une fixture écrite sur la même hypothèse n'aurait rien prouvé. |
| 3 | Onglet « Joueurs » | Libellés dans les 9 langues, `manuel.rst` à jour. |
| 4 | Parité CLI + serveur | `list --type players` (text/json/csv) et `POST /v1/stats.playerTable`. |
| 5 | Clôture documentaire | `stats_parity.rst` couvre désormais la chance et la dérivation V/D. |

Écarts assumés avec la spécification initiale :

- La colonne **Erreurs** est calculée et exposée (API, JSON, CSV) mais n'a pas
  de colonne propre dans le tableau : à l'écran elle faisait doublon avec
  Décisions et Blunders.
- Le DDL `move` de `db_migration.go` n'a **pas** reçu la colonne : c'est du
  schéma d'époque recréé pendant la chaîne de migration, que le palier 2.15.0
  altère ensuite. C'est ce qu'avait fait le commit `flagged` avant lui.

Références : ADR-0010 (le luck est un fait du coup), `CONTEXT.md` (entrée
**Player** ajoutée lors de cette session), `doc/source/stats_parity.rst`
(modèle de calcul existant), tableau par joueur de gnuBG
(`gnubg/gtkrelational.c`, requête `GROUP BY name` sur `matchstat`) comme
référence de forme.

---

## Décisions actées (et leurs raisons)

1. **Objet** : un tableau comparatif, une ligne par joueur — pas un dashboard
   d'agrégats globaux. Une ligne « Total » pourra s'ajouter plus tard.
2. **Une ligne = un nom tel qu'enregistré** (`player1_name`/`player2_name`),
   pas une identité de personne. Le regroupement d'orthographes reste l'affaire
   de MergePlayers (destructif, existant) ; une « Player identity » persistante
   est une suite possible, hors périmètre (voir FOLLOWUPS).
3. **UI** : 4ᵉ onglet « Joueurs » du panneau Stats, à côté de Dashboard /
   Progression / Erreurs.
4. **Barre de filtres dans cet onglet** : période, tournoi et longueur de match
   restent actifs ; **sélecteur de joueur grisé** (le tableau les montre tous) ;
   **type de décision grisé** (les colonnes ventilent déjà checker/cube).
5. **Tri** : par défaut PR croissant ; toutes les colonnes triables au clic sur
   l'en-tête. Joueurs sans décision comptée : PR affiché « — », triés en fin.
6. **Pas de masquage silencieux** : tous les joueurs sont affichés ; les
   colonnes Matchs et Décisions donnent au lecteur de quoi juger la
   significativité. Pas de seuil ni d'option de seuil en v1.
7. **Clic sur une ligne** : sélectionne ce joueur dans le filtre et bascule sur
   l'onglet Dashboard (mécanismes existants : store de filtre + onglet actif).
8. **Parité totale dès la v1** : contrat Storage (les deux backends + tests
   contractuels), façade Database + binding Wails, route serveur, CLI.
9. **Luck : dans le périmètre**, avec bump de schéma (décision utilisateur).
   Voir lot 2 et ADR-0010.
10. **Préalables** : deux corrections de stats détectées pendant la session en
    confrontant le code blunderDB au code source de gnuBG (lot 0).

## Colonnes du tableau (v1)

| Colonne | Définition |
|---|---|
| Joueur | nom tel qu'enregistré |
| Matchs | nombre de matchs où le nom apparaît |
| V / D | victoires / défaites (dérivation ci-dessous) |
| Décisions | décisions comptées (`statsCountedExpr`, inchangé) |
| PR | `pr()` existant (500 × Σerr/1000/N), clé = joueur |
| PR checker | idem, `decision_type = 0` |
| PR cube | idem, `decision_type = 1` |
| Snowie ER | après fix du lot 0 ; dénominateur = coups checker **des deux joueurs** des matchs du joueur |
| Blunders | count `erreur >= 100` MP (seuil harmonisé au lot 0) |
| Luck | luck rate = Σ`luck_mp` / nombre de lancers à luck connu, en mEMG signé ; « — » si aucune donnée |

Exclus de la v1 (volontairement) : MWC loss (une somme de MWC agrégée sur des
matchs hétérogènes ne se classe pas), détail des 4 catégories cube (c'est le
rôle de l'onglet Erreurs après le clic), colonne « inachevés » (V+D ≤ Matchs,
assumé).

### Dérivation V / D

Aucun vainqueur n'est stocké au niveau match ; on agrège les games
(`points_won` cumulés par siège — attention au retournement de sièges déjà géré
au swap dans `matches_sqlite.go` / `matches_postgres.go`) :

- **Match à N points** : V = le joueur dont le score cumulé atteint
  `match_length`. Personne ne l'atteint → match inachevé, ni V ni D.
- **Money session** (`match_length = 0`) : V = strictement plus de points
  cumulés que l'adversaire ; égalité → ni V ni D (esprit du `MatchResult()`
  ±1/0 de gnubg).
- **Match sans games** : inachevé.

---

## Lot 0 — Préalables : corrections de stats

Trouvailles de la confrontation blunderDB ↔ gnuBG (le reste du modèle est
conforme : dénominateurs checker/cube, définition « close » 0.16 avec clip 1.0,
conversion EMG→MWC identiques à gnubg ; PR ×500, exclusion Marseille et
Snowie ×500 sont des divergences XG assumées, épinglées par
`testdata/stats_reference/`).

1. **Fix SnowieGlobal filtré par joueur** — `stats_sqlite.go:294-308` et
   l'équivalent postgres : le dénominateur compte les coups checker du jeu de
   lignes *filtré*, donc seulement ceux du joueur sélectionné → ~2× trop grand.
   La définition (et le propre calcul par-match du même fichier,
   `snowieP1Checker + snowieP2Checker`, et `stats_reference/SCHEMA.md`) exige
   les coups des **deux** joueurs. Corriger, couvrir par un test qui compare
   SnowieGlobal(joueur) au Snowie par-match agrégé.
2. **Harmoniser le seuil blunder sur `>= 100`** — aujourd'hui `> 100` en SQL
   (breakdown cube, `stats_sqlite.go:366` + postgres) vs `>= 100` en Go
   (MatchDetail). Un blunder à exactement 0.100 en est un.
3. **FOLLOWUPS** : noter qu'un `move.cube_action` dégénéré `Unknown(…)` passe le
   `NOT IN ('', 'No Double', 'NoDouble')` de `statsCountedExpr` et serait compté
   comme action active (aucune occurrence connue ; pour mémoire).

Doc du lot : ajuster `stats_parity.rst` (dénominateur Snowie explicite, seuil
blunder `>= 0.100`).

## Lot 1 — Schéma & ingestion du luck

Voir ADR-0010 pour la décision de fond.

- **Colonne `luck_mp INTEGER NULL` sur la table `move`** (pas `analysis`, pas
  `position`). Millipoints EMG **signés** (positif = chanceux), arrondi comme
  les erreurs. `NULL` = inconnu, jamais 0 (0 est un lancer neutre réel).
- **Bump `DatabaseVersion` 2.14.0 → 2.15.0**, triple-sync obligatoire :
  migration desktop (`database/db_migration.go` + DDL `db_schema.go`), schéma
  `storage/sqlite`, migrations `storage/postgres` — sinon `migration_test`
  échoue une version trop tôt. Test de migration.
- **Ingestion** : XG → `MoveEntry.ErrLuck` (xgparser ; vérifier à
  l'implémentation la convention de signe/unité contre un match affiché dans
  XG) ; GnuBG SGF → propriété `LU` (gnubgparser `LuckRating.Value`). BGF et
  `.mat` : pas de luck dans le format → NULL. Fixture d'ingestion qui épingle
  quelques valeurs par lancer et le total par joueur contre la sortie de
  l'outil source.
- **Conséquence assumée** (décision utilisateur) : les bases existantes
  affichent « — » tant que les matchs ne sont pas réimportés. Le repair façon
  #115 est impossible : le luck n'est pas dans le JSON d'analyse stocké.
- `annexe_db_scheme.rst` : mettre à jour la version du schéma uniquement (la
  page est une note conceptuelle, ne pas la re-détailler).

Doc du lot : section luck dans `stats_parity.rst` — définition gnubg
(luck = équité du meilleur coup avec le dé réel − espérance sur les 36 issues,
éval 0-ply cubeful), ce que blunderDB en reprend (la valeur calculée par
l'outil source, jamais recalculée), unité, sémantique NULL, luck rate.

## Lot 2 — Contrat Storage + backends

- Nouvelle méthode du contrat `StatsStore` : proposition
  `PlayerTable(ctx, scope string, filter StatsFilter) ([]PlayerRow, error)`
  (`PlayerNames` est déjà pris par la liste nom+fréquence). Le filtre n'honore
  que période / tournoi / longueur de match (décision 4) ; `PlayerName` et
  `DecisionType` y sont ignorés.
- `PlayerRow` : Name, Matches, Wins, Losses, Decisions, sommes d'erreurs (pour
  PR/PRChecker/PRCube en règle somme/somme — jamais moyenne de PR), SnowieER,
  Blunders, LuckMPSum, LuckRolls. Précédent de forme : `TournamentPlayerAcc` +
  `PickReferencePlayer` (`storage/stats.go:207-252`), accumulation par nom déjà
  écrite deux fois pour les badges de tournoi.
- Le prédicat joueur réutilise la forme existante
  `((m.player1_name = nom AND mv.player = 1) OR (m.player2_name = nom AND mv.player = -1))`
  en `GROUP BY` nom du joueur au trait.
- Snowie par joueur : pour chaque joueur, numérateur = ses erreurs
  (checker + cube), dénominateur = Σ sur *ses* matchs des coups checker des
  deux joueurs (forcés inclus) — même convention que le par-match existant.
- **Tests contractuels partagés** dans `storage/storagetest/` (les deux
  backends doivent passer) : cas multi-joueurs, multi-orthographes (deux noms
  = deux lignes), match inachevé, money session, luck NULL partout, luck
  partiel.

## Lot 3 — GUI : l'onglet

- `StatsPanel.svelte` : 4ᵉ onglet « Joueurs » (i18n frontend : libellés dans
  les json de `frontend/src/i18n`, comme les autres libellés d'UI).
- Nouveau composant `StatsPlayersTab.svelte` + accès via `statsStore`
  (même cache clé filtre + `statsInvalidationKeyStore`).
- Grisage : sélecteur joueur et type de décision `disabled` quand l'onglet
  Joueurs est actif (`StatsFilterBar.svelte`).
- Tri client (les données tiennent en mémoire), défaut PR croissant, PR « — »
  en fin de tri.
- Clic ligne → écrit le nom dans le store de filtre + onglet actif = Dashboard.
- **Svelte 5 : `$store`/`$effect` uniquement, jamais `.subscribe()`.**
- Typographie : tokens `--font-size-*`, pas de `font-size` absolu (ADR-0008) ;
  les chiffres du tableau peuvent relever de l'exception « statistics figures »
  déjà nommée par l'ADR.
- Bindings Wails régénérés (nouvelle méthode exposée sur `Database`) —
  redémarrer `wails dev`.
- Tests : unitaires composant (imiter `StatsDashboardTab.test.js`), e2e
  Playwright bascule d'onglet (imiter `tab-switch-stats.spec.js`).

Doc du lot : `manuel.rst` section Stats — l'onglet Joueurs (colonnes, tri,
grisage des filtres, clic, « — » = luck inconnu / réimport nécessaire).

## Lot 4 — Parité CLI + serveur

- Serveur : route `POST /v1/stats.playerTable` dans `handlers_stats.go`
  (10ᵉ route du même moule), atteignable aussi via `call`.
- CLI : `blunderdb list --type players` avec `--from/--to/--tournament` et
  `--format table|json|csv` — le CSV est le cas d'usage automation d'iddqd43.
  Tests dans `internal/cli/` (fixtures via `TestMain` chdir racine).
- `CLI_USAGE.md` + `doc/source/cli.rst` + `doc/source/mode_headless.rst`
  (nouvelle méthode listée).

## Lot 5 — Clôture documentation

Passe finale de cohérence (l'essentiel a été livré lot par lot) :

- `stats_parity.rst` : relire le modèle complet — décisions comptées, PR,
  Snowie corrigé, seuil blunder, luck, dérivation V/D — c'est la demande
  explicite de l'utilisateur : documenter la feature **et** la manière de
  calculer.
- `manuel.rst`, `cli.rst`, `mode_headless.rst` : renvois croisés.
- Changelog `index.rst` : à la release uniquement (règle major-only), pas dans
  ces lots.
- Réponse à iddqd43 dans l'issue #116 à la livraison.

---

## Hors périmètre (suites possibles, à reporter dans FOLLOWUPS)

- Identité de joueur persistante (regroupement d'alias non destructif) ; le
  champ `PlayerAliases` du filtre Go n'est d'ailleurs toujours pas branché au
  frontend.
- Ligne « Total / Moyenne » du tableau ; luck dans les onglets Dashboard /
  Progression ; colonne MWC ; détail des catégories cube par joueur ; seuil
  minimal de décisions configurable ; surlignage de la ligne du joueur
  sélectionné.
