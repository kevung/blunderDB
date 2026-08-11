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

- [ ] Contrat : ajouter au cas `Match/DeleteCascadeRetention` les rétentions
      `flagged`, `anki_card`, et « position partagée par un second match ».
- [ ] Contrat : dans `testMatchSwapCopyOnWrite`, asserter que l'ancienne
      position orpheline est purgée (ErrNotFound) et qu'une position retenue
      (commentée/flaggée) survit.
- [ ] Contrat : ajouter des cas pour `Collections.CopyPosition`,
      `Stats.MatchDetail`, `Stats.PositionIDsByMatch`,
      `Stats.PositionIDsByTournament` ; supprimer les doublons devenus inutiles
      dans `sqlite/*_test.go` et `postgres/*_test.go`.
- [ ] `database/delete_match_test.go` : cas `flagged` et `anki_card` sur la
      copie legacy du prédicat.
- [ ] Test anti-dérive des 3 copies : normaliser les 3 constantes SQL
      (placeholders, `= 1` vs booléen, espaces) et comparer l'ensemble
      {tables, colonnes} référencé. Le test vit dans un package qui peut
      importer les trois (p. ex. `tests/`).

## Critères de fin

- Retirer artificiellement la clause `flagged` d'une copie fait échouer au
  moins un test (vérifier les 3 copies, une par une, puis restaurer).
- `go test ./...` et le job Postgres (fiche 01) verts.

## Risques & garde-fous

- Le test anti-dérive doit rester grossier (tables + colonnes) pour ne pas
  devenir bruyant sur des différences de dialecte légitimes.
- Ne PAS « corriger » la triplication elle-même : elle est documentée et
  voulue (CLAUDE.md).
