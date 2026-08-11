# Fiche 02 — Filet de sécurité sur la rétention et le contrat storage

Branche : `test/filet-retention`

## Objectif

Rendre impossible une régression silencieuse du prédicat de rétention
(`positionIsHeldSQL`, 3 copies volontaires) et combler les asymétries du
contrat storage. Aucun changement de comportement produit : que des tests.

## Constats

- `testMatchDeleteCascadeRetention` (`storagetest/contract.go:1361-1443`)
  couvre 5 cas de rétention : purged, commented, individual, inCollection,
  inDeck. **Manquent** : `flagged` (ADR-0006), `anki_card`… et surtout le cas
  le plus courant — une position partagée par **deux matchs** (première clause
  du prédicat).
- `testMatchSwapCopyOnWrite` (`contract.go:348`) vérifie la copie mais jamais
  la purge de l'ancienne position ni la survie d'une position retenue.
- Méthodes du contrat testées d'un seul côté (donc jamais en CI avant la
  fiche 01) : `Collections.CopyPosition` (PG seulement), `Stats.MatchDetail`,
  `Stats.PositionIDsByMatch`, `Stats.PositionIDsByTournament` (PG seulement,
  0 test sqlite).
- Rien ne vérifie mécaniquement que les 3 copies du prédicat citent les mêmes
  tables/colonnes (l'invariant n'existe que dans CLAUDE.md).
- `delete_match_test.go` (côté `database/`, la copie que le GUI/CLI exécutent)
  ne teste ni `flagged` ni `anki_card`.

## Tâches

- [x] Contrat : ajouter au cas `Match/DeleteCascadeRetention` les rétentions
      `flagged`, `anki_card`, et « position partagée par un second match ».
- [x] Contrat : dans `testMatchSwapCopyOnWrite`, asserter que l'ancienne
      position orpheline est purgée (ErrNotFound) et qu'une position retenue
      (commentée/flaggée) survit.
- [x] Contrat : ajouter des cas pour `Collections.CopyPosition`,
      `Stats.MatchDetail`, `Stats.PositionIDsByMatch`,
      `Stats.PositionIDsByTournament` ; supprimer les doublons devenus inutiles
      dans `sqlite/*_test.go` et `postgres/*_test.go`.
- [x] `database/delete_match_test.go` : cas `flagged` et `anki_card` sur la
      copie legacy du prédicat.
- [x] Test anti-dérive des 3 copies : normaliser les 3 constantes SQL
      (placeholders, `= 1` vs booléen, espaces) et comparer l'ensemble
      {tables, colonnes} référencé. Le test vit dans un package qui peut
      importer les trois (p. ex. `tests/`).

## Critères de fin

- [x] Retirer artificiellement la clause `flagged` d'une copie fait échouer au
  moins un test (vérifier les 3 copies, une par une, puis restaurer).
- [x] `go test ./...` et le job Postgres (fiche 01) verts.

## Notes d'exécution

- **Tâche 1 (rétention).** Les clauses `flagged`/`anki_card` existaient déjà
  dans `positionIsHeldSQL` (ADR-0006 était déjà implémenté côté production) ;
  seuls les cas de test manquaient. `testMatchDeleteCascadeRetention` a
  8 positions maintenant : les 5 existantes + `flagged`, `ankiCard` (rejoint le
  même deck que `inDeck` — même mécanisme SQL, la duplication de style est
  volontaire pour nommer explicitement la clause du prédicat) et
  `sharedWithSecondMatch` (un second match vivant référence la même position
  par une `move`, seule assertion qui couvre vraiment la première clause du
  prédicat — le cas le plus fréquent en usage réel).
- **Tâche 2 (swap).** `makeMatch` prend maintenant l'id de position en
  paramètre. Deux scénarios ajoutés : une position seulement tenue par le
  match swappé (purgée sous son ancien id après le swap, `ErrNotFound`), et
  une position flaggée seulement tenue par le match swappé (survit sous son
  ancien id, alors qu'aucune `move` ne la référence plus). Le retenu utilisé
  est `flagged`, pas `commented` : un commentaire ne tient PAS une position
  par construction (documenté dans le prédicat lui-même), donc l'énoncé de la
  fiche (« commentée ou flaggée ») aurait fait échouer le test à tort — vérifié
  avant d'écrire l'assertion.
- **Tâche 3 (asymétries du contrat).** `Collections.CopyPosition` n'avait de
  test que côté PostgreSQL (`TestCollectionMoveCopy`) ; `Stats.MatchDetail` et
  `Stats.PositionIDsByMatch` n'étaient couverts que par le test de parité
  fixture-driven PostgreSQL (`stats_parity_postgres_test.go`, qui reste —
  différent objectif : comparer PostgreSQL à l'implémentation historique sur
  de vrais imports XG, pas juste exercer l'API) ; `Stats.PositionIDsByTournament`
  n'avait strictement aucun test sur aucun des deux backends. Les 4 nouveaux
  cas de contrat tournent maintenant sur SQLite et PostgreSQL. Seule
  suppression : le bloc `CopyPosition` de `TestCollectionMoveCopy`
  (postgres), strictement doublonné par le nouveau cas de contrat ; la partie
  `MovePosition` du même test, déjà redondante avec le contrat *avant* cette
  fiche, a été laissée intacte (hors périmètre).
- **Tâche 4 (copie legacy).** Deux nouveaux tests suivant le style de
  `TestDeleteMatchKeepsIndividuallyImportedPosition` déjà présent :
  `TestDeleteMatchKeepsFlaggedPosition` et `TestDeleteMatchKeepsAnkiCardPosition`.
- **Tâche 5 (anti-dérive).** Aucune des 3 constantes `positionIsHeldSQL`
  n'est exportée ; plutôt que de les exporter (changement de surface d'API
  pour une constante purement interne), le test
  (`tests/position_is_held_predicate_test.go`) lit les 3 fichiers sources tels
  quels avec `runtime.Caller(0)` pour localiser la racine du dépôt (même
  logique que le chdir de `database/main_test.go`, sans avoir besoin d'un
  `TestMain` puisque `tests/` est déjà à la racine et qu'aucun autre test du
  paquet n'est sensible au cwd), extrait le texte SQL par regex, puis compare
  l'ensemble des identifiants (tables/colonnes) après avoir exclu les
  mots-clés SQL — les placeholders (`?1`/`$1`) et les littéraux numériques
  sont ignorés automatiquement par la regex d'extraction (elle ne matche que
  les tokens commençant par une lettre), donc aucune normalisation
  supplémentaire n'était nécessaire pour `= 1` vs booléen nu.
- **Critère de fin obligatoire — vérification manuelle.** La clause `flagged`
  a été retirée tour à tour de CHACUNE des 3 copies (sqlite, postgres,
  database), une à la fois, avec restauration immédiate après chaque essai
  (diff vérifié vide après restauration) :
  - `sqlite/matches_sqlite.go` : `TestPositionIsHeldPredicateParity` **et**
    `TestContract_SQLite/Match/DeleteCascadeRetention` **et**
    `.../Match/SwapCopyOnWrite` échouent.
  - `postgres/matches_postgres.go` : `TestPositionIsHeldPredicateParity`
    échoue (le run complet du contrat Postgres n'a pas été refait pour cet
    essai destructif, mais le même code de contrat tourne sur les deux
    backends — voir `Match/DeleteCascadeRetention` ci-dessus — donc l'échec
    y est attendu aussi).
  - `database/db_match.go` : `TestPositionIsHeldPredicateParity` **et**
    `TestDeleteMatchKeepsFlaggedPosition` échouent.
  Dans les 3 cas, `TestPositionIsHeldPredicateParity` seul suffisait déjà à
  détecter la dérive — c'est le filet qui couvre les 3 copies simultanément,
  comme demandé.
- **Validation finale.** `go vet ./...`, `go test ./...` et `golangci-lint
  run` verts. `go test -tags postgres ./pkg/blunderdb/storage/postgres/...
  -timeout 900s` vert (Docker) : les 4 nouveaux cas de contrat
  (`DeleteCascadeRetention`, `SwapCopyOnWrite`, `Collection/CopyPosition`,
  `Stats/MatchDetail`, `Stats/PositionIDsByMatch`,
  `Stats/PositionIDsByTournament`) passent sur PostgreSQL sans aucune
  divergence — aucun commit de correction de parité n'a été nécessaire.
  (Note d'environnement, sans rapport avec le code : ce worktree n'avait pas
  de `frontend/dist` construit, ce qui bloquait `go vet ./...`/`go build ./...`
  sur le paquet racine à cause du `go:embed`. Copié depuis le checkout
  principal — dossier gitignored, aucun commit.)

## Risques & garde-fous

- Le test anti-dérive doit rester grossier (tables + colonnes) pour ne pas
  devenir bruyant sur des différences de dialecte légitimes.
- Ne PAS « corriger » la triplication elle-même : elle est documentée et
  voulue (CLAUDE.md).
