# 05 — Collection / tournament / Anki CRUD tests

**Goal:** Test coverage for the three major untested CRUD subsystems in `db.go`: collections (14 methods, ~L14030–14882), tournaments (14 methods, ~L14883–15765), and Anki/FSRS (13 methods, ~L15766–16223).

**Depends on:** Nothing.

**Impact:** Medium — these are user-facing features with zero test coverage.

## Context

All three subsystems follow the same pattern: create entity, add child items, reorder, remove, delete with cascade. Tests can use in-memory DB with pre-imported match data.

## Files touched

- **New:** `collection_test.go`
- **New:** `tournament_test.go`
- **New:** `anki_test.go`

## Tasks

### 1. Collection CRUD tests (`collection_test.go`)

- [ ] **`TestCreateCollection`**: Create a collection, verify ID > 0 and name matches
- [ ] **`TestGetAllCollections`**: Create 3 collections, verify `GetAllCollections()` returns all 3 with correct position counts (initially 0)
- [ ] **`TestUpdateCollection`**: Create, update name+description, verify changes persisted via `GetCollectionByID()`
- [ ] **`TestDeleteCollection`**: Create collection with positions, delete it, verify positions still exist in DB (FK cascade only removes the association)
- [ ] **`TestAddPositionToCollection`**: Import a match, add a position to a collection, verify `GetCollectionPositions()` returns it
- [ ] **`TestAddPositionsToCollection`**: Batch add 5 positions, verify all present
- [ ] **`TestAddDuplicatePosition`**: Add same position twice, verify UNIQUE constraint handled gracefully (no error or idempotent)
- [ ] **`TestRemovePositionFromCollection`**: Add then remove, verify collection is empty
- [ ] **`TestRemovePositionsFromCollection`**: Batch remove, verify correct positions removed
- [ ] **`TestReorderCollections`**: Create 3 collections with sort_order 0,1,2 → reorder to 2,0,1 → verify new order in `GetAllCollections()`
- [ ] **`TestReorderCollectionPositions`**: Add 3 positions, reorder, verify new sort_order
- [ ] **`TestMovePositionBetweenCollections`**: Move position from collection A to B, verify removed from A and present in B
- [ ] **`TestCopyPositionToCollection`**: Copy position to second collection, verify present in both
- [ ] **`TestGetPositionCollections`**: Add position to 2 collections, verify `GetPositionCollections()` returns both
- [ ] **`TestExportCollections`**: Create collection with positions+analysis, export to temp file, verify exported DB has the collection and its positions

### 2. Tournament CRUD tests (`tournament_test.go`)

- [ ] **`TestCreateTournament`**: Create tournament, verify ID > 0 and name
- [ ] **`TestGetAllTournaments`**: Create 3 tournaments, verify all returned with match counts (initially 0)
- [ ] **`TestUpdateTournament`**: Update name/date/location/comment, verify via `GetAllTournaments()`
- [ ] **`TestDeleteTournament`**: Create tournament with matches, delete tournament, verify matches still exist (unlinked, not deleted)
- [ ] **`TestAddMatchToTournament`**: Import a match, assign to tournament via `AddMatchToTournament()`, verify `GetTournamentMatches()` returns it
- [ ] **`TestRemoveMatchFromTournament`**: Add then remove match, verify match still exists but `tournament_id` is NULL
- [ ] **`TestSetMatchTournamentByName`**: Use the name-based assignment (creates tournament if needed), verify tournament created and match linked
- [ ] **`TestSetMatchTournamentByName_Existing`**: Assign to existing tournament by name, verify no duplicate tournament created
- [ ] **`TestReorderTournamentMatches`**: Add 3 matches, reorder, verify `tournament_sort_order` updated
- [ ] **`TestGetMatchTournament`**: Assign match to tournament, verify `GetMatchTournament()` returns correct tournament
- [ ] **`TestUpdateTournamentComment`**: Set comment, verify persisted
- [ ] **`TestUpdateMatchComment`**: Set match comment, verify persisted
- [ ] **`TestExportTournaments`**: Create tournament with matches, export to temp file, verify exported DB has tournament, matches, and positions

### 3. Anki/FSRS tests (`anki_test.go`)

- [ ] **`TestCreateAnkiDeck`**: Create deck with source_type="collection", verify ID and name
- [ ] **`TestGetAllAnkiDecks`**: Create 2 decks, verify both returned with card counts
- [ ] **`TestUpdateAnkiDeck`**: Update name+description, verify persisted
- [ ] **`TestUpdateAnkiDeckParams`**: Update FSRS params (retention, max_interval, fuzz), verify
- [ ] **`TestDeleteAnkiDeck`**: Create deck with cards, delete, verify cards cascade-deleted
- [ ] **`TestSyncAnkiDeck_Collection`**: Create collection with 5 positions, create deck sourced from it, sync → verify 5 cards created
- [ ] **`TestSyncAnkiDeck_Idempotent`**: Sync twice, verify no duplicate cards
- [ ] **`TestSyncAnkiDeck_AddNew`**: Sync, add position to collection, re-sync → verify new card added, existing cards unchanged
- [ ] **`TestGetNextAnkiCard`**: Sync deck, get next card, verify it returns a valid card with position data
- [ ] **`TestGetNextAnkiCard_Empty`**: Get next card from empty deck, verify nil/empty response
- [ ] **`TestReviewAnkiCard_Again`**: Review card with rating "Again", verify: reps incremented, lapses incremented, state reset to learning, due date soon
- [ ] **`TestReviewAnkiCard_Good`**: Review card with "Good", verify: reps incremented, stability increased, due date in future
- [ ] **`TestReviewAnkiCard_Easy`**: Review card with "Easy", verify: reps incremented, largest stability boost, longest interval
- [ ] **`TestReviewAnkiCard_Progression`**: Review sequence: Again → Good → Good → Easy, verify state transitions New→Learning→Review→Review with increasing intervals
- [ ] **`TestResetAnkiDeck`**: Review several cards, reset deck, verify all cards back to new state (reps=0, stability=0, state=0)
- [ ] **`TestGetAnkiDeckStats`**: Review some cards, verify stats (total, new, learning, review, due counts)
- [ ] **`TestGetAnkiDeckPositions`**: Sync deck, verify `GetAnkiDeckPositions()` returns the correct position data

### 4. Import cancellation test (bonus)

- [ ] **`TestImportCancellation`**: Start importing a large match in a goroutine, set `importCancelled = 1` mid-import, verify the import function returns early and the transaction is rolled back (no partial data)

## Acceptance criteria

- [ ] ≥15 collection tests, ≥13 tournament tests, ≥17 Anki tests
- [ ] All tests pass with `go test -run 'TestCreate|TestGet|TestUpdate|TestDelete|TestAdd|TestRemove|TestSync|TestReview|TestReset|TestExport' -count=1 -timeout 120s`
- [ ] Tests use in-memory DB (`:memory:`)
- [ ] FK cascade behavior verified (collection delete doesn't delete positions; tournament delete doesn't delete matches; deck delete cascades to cards)
- [ ] `go test -race` passes

## Rollback

Delete the three test files: `git revert`. Tests are additive only.
