# Audit des tests — état de reprise

Demande : une passe complète sur les tests pour vérifier leur utilité, leur pertinence, leur
non-redondance et leur orthogonalité. À lancer dans une session neuve, en orchestrateur
(ADR-0055) : l'orchestrateur ne lit pas de test, il délègue par zone à Opus.

## Méthode

1. **Mesurer, sans rien changer** (un ouvrier Sonnet) : par zone, nombre de tests, durée
   (`go test -json`, `vitest --reporter=json`), couverture par paquet (`-coverprofile`,
   `vitest --coverage`). Rendre un tableau, pas les sorties.
2. **Auditer par zone** (Opus, lecture seule) : pour chaque test, une des quatre cases —
   utile ; redondant (ses assertions sont portées ailleurs **et** le retirer ne perd aucune
   ligne de couverture) ; obsolète (teste un comportement disparu, ou tautologique) ;
   mal placé (teste deux choses, ou la même chose que la suite de contrat). Rendre la liste
   des candidats avec la preuve (le test qui couvre déjà, le delta de couverture).
3. **Appliquer** dans un worktree par zone (Opus) : retirer ou fusionner les candidats
   prouvés ; suite verte et couverture égale après chaque zone. Ce qui demande un arbitrage
   remonte à l'utilisateur.

## Ne se retirent jamais, même redondants en lignes

Ils tiennent un invariant qu'aucune couverture ne mesure : la suite de contrat
`storage/storagetest/`, les tests de parité (`parity_test.go`, `*_parity_test.go`,
`position_is_held_predicate_test.go`), les `*.sync.test.js` et `cli_doc_sync_test.go`,
`TestMigrationSteps_ContinuousChain` et `migration_test.go`, les gold de gammonNet
(`cube_gold.bin`, `gold_test.go`), `kernel_identity_test.go`, `TestIntegrationGate`,
`TestDemoDatabaseIsCurrent`, `go test ./cmd/help-gen`, `trust_boundary_test.go`.

## Zones

Go : `engine/gammonnet`, reste de `engine`, `database`, `storage`, `ingest`+`domain`+`parser`,
paquets fonctionnels, `internal/server`, `internal/cli`, `internal/gui`+`cmd`.
Frontend : `src/__tests__` (213 fichiers, 2 743 tests), `tests/e2e` (Playwright).
