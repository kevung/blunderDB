# Fiche 06 — Espace disque : compactage accessible

Branche : `feat/vacuum`

## Objectif

Donner à l'utilisateur un moyen de récupérer l'espace libéré par les
suppressions (matchs, tournois, purges) : aujourd'hui aucune commande
VACUUM n'existe (la seule occurrence du mot dans le code est un commentaire
de préparation de démo, `internal/gui/app.go:24`), donc une base de travail
ne rétrécit jamais.

## Tâches

- [x] Méthode `Vacuum()` sur le wrapper `Database` (et le contrat Storage si
      pertinent — côté Postgres, no-op documenté ou `VACUUM (ANALYZE)` ciblé).
      Attention : `VACUUM` SQLite ne s'exécute pas dans une transaction et
      exige ~2× l'espace du fichier ; vérifier l'espace libre avant, message
      d'erreur clair sinon.
- [x] Sous-commande CLI `blunderdb vacuum <db>` (fichier `internal/cli/cli_vacuum.go`,
      enregistrement dans `cli.go`, aide `printUsage`), affichant la taille
      avant/après.
- [x] GUI : entrée « Compacter la base » (menu/ConfigModal), avec taille
      gagnée dans la barre d'état ; passer par la même méthode bound.
- [x] Enchaîner `ANALYZE` après le VACUUM (synergie fiche 05).
- [x] Doc française dans la même branche : `doc/source/manuel.rst` (section
      maintenance) + `doc/source/cli.rst` + `CLI_USAGE.md` + aide intégrée
      (`frontend/src/i18n/help/*.js` — les 9 langues pour la nouvelle entrée
      de commande si une commande `:vacuum` est exposée ; sinon doc GUI/CLI
      seulement).
- [x] Tests : CLI (créer, remplir, supprimer, vacuum, taille réduite) ;
      GUI-level via la méthode `Database`.

## Critères de fin

- [x] Scénario mesurable : base gonflée par des suppressions → vacuum → fichier
  réduit ; testé automatiquement.
- [x] Doc FR livrée dans la branche.

## Risques & garde-fous

- Ne jamais vacuum automatiquement à l'ouverture (coût imprévisible sur de
  grosses bases) : action explicite de l'utilisateur uniquement.
- WAL : checkpoint (`wal_checkpoint(TRUNCATE)`) avant VACUUM pour que la
  taille affichée soit honnête.

## Notes d'exécution

Exécutée dans `/home/unger/src/blunderDB-fiche06` (branche `feat/vacuum`).
`go vet`, `go build`, `go test ./...`, `golangci-lint run` et la suite
frontend (`npm run lint`, `npm run format:check`, 727 tests vitest, dont les
gardes i18n `i18nKeys.sync.test.js`/`i18nOrphanKeys.sync.test.js` — KNOWN_GAPS
resté vide) verts avant chaque commit. Aucun fichier de la zone de
coordination (`storage/sqlite/sqlite.go`, `storage/sqlite/search*.go`,
`schema_sqlite.go`, `db_schema.go`, `cli_import.go`, tests frontend
`utils`/`viewStore`) n'a été touché.

Choix faits sur les points laissés ouverts par la fiche :

- **Signature de `Database.Vacuum()` — struct plutôt que trois valeurs de
  retour.** La fiche demandait `(tailleAvant, tailleAprès int64, err error)`.
  En le codant tel quel puis en régénérant les bindings (`wails generate
  module -tags webkit2_41`, l'outil est disponible localement), le
  `.d.ts` produit ne typait qu'un seul entier — indice que quelque chose
  clochait. Lecture de `internal/binding/boundMethod.go` (Wails v2.10.1,
  vendu dans le module cache) : `BoundMethod.Call` ne gère que
  `OutputCount() == 1` ou `== 2` (ce second cas suppose en plus, sans
  vérifier son type, que la 2ᵉ valeur est l'erreur) ; une méthode à 3
  sorties tombe dans aucun des deux cas du `switch` et renvoie
  silencieusement `nil, nil` au JS — pas d'erreur, pas de crash, juste un
  bouton qui ne rapporte jamais rien. Corrigé en suivant le précédent déjà
  posé dans ce paquet (`IndividualSaveResult`) : `type VacuumResult struct
  { SizeBefore, SizeAfter int64 }`, méthode `Vacuum() (VacuumResult,
  error)` — 2 sorties, binding correct, et toujours aussi direct à
  déstructurer côté CLI/tests Go. Reconfirmé après coup en régénérant :
  `database.VacuumResult` apparaît bien dans `models.ts` et
  `Vacuum():Promise<database.VacuumResult>` dans le `.d.ts`.
- **VACUUM sous WAL ne truffate pas le fichier tout seul.** Repéré par le
  test : après `VACUUM`, `PRAGMA page_count` tombait bien à 3, mais la
  taille du fichier sur disque ne bougeait pas — SQLite écrit le contenu
  reconstruit de VACUUM à travers le WAL sous ce mode de journalisation, le
  fichier principal ne se réduit qu'après un checkpoint. D'où le second
  `wal_checkpoint(TRUNCATE)` après `VACUUM`+`ANALYZE`, documenté comme
  étape 5 dans le commentaire de `Vacuum()` — sans lui, la commande aurait
  silencieusement rapporté « rien à libérer » alors que le compactage avait
  réellement eu lieu.
- **Bindings wailsjs régénérés à la main, pas par un `git checkout` du
  diff complet.** `wails generate module` (CLI locale v2.10.1, le dépôt vise
  v2.10.2 — léger écart de version) régénère tout `frontend/wailsjs/`, y
  compris des signatures déjà dérivées ailleurs sur `main` depuis la
  dernière régénération (`ExportCollections`/`ExportTournaments` ont gagné
  des paramètres, trois méthodes `Migrate_*` et `PositionExists` ont
  disparu) — dérive préexistante, hors périmètre de cette fiche. Le diff
  complet a été annulé et seules les trois entrées `Vacuum`/`VacuumResult`
  ont été ajoutées à la main dans `Database.js`, `Database.d.ts` et
  `models.ts`, au caractère près identique à ce que le générateur produit
  pour ces trois blocs (vérifié par une régénération complète jetable avant
  de revenir en arrière). Les changements de mode Unix sur
  `wailsjs/runtime/*` (générés en même temps, contenu inchangé) ont aussi
  été annulés.
- **GUI : onglet Interface, pas un onglet Maintenance dédié.** Aucun des
  quatre onglets existants (Interface / Couleurs / Bearoff / Identité) ne
  correspond exactement à « propriété de la base ouverte » — Bearoff gère
  une ressource externe à l'application, pas à une base donnée. Choisi
  Interface car c'est le plus simple : pas de nouvel onglet pour un unique
  bouton, confirmation légère via `confirmService` (déjà présent dans ce
  worktree, fiche 07 déjà mergée), résultat rapporté dans la barre d'état
  via `statusBarTextStore`/`tMsg`, à l'identique du reste de l'application.
- **Storage / Postgres : non touché**, conformément à la consigne de
  coordination — le contrat `storage.Storage` ne gagne pas de `Vacuum`, le
  daemon multi-tenant n'en a pas besoin (`ANALYZE` seul y aurait un profil
  de coût très différent d'un `VACUUM` fichier SQLite ; à revisiter
  séparément si un besoin réel apparaît côté Postgres).
- **Pas de commande in-app `:vacuum`** : seule une entrée GUI/CLI est
  exposée, donc aucune entrée dans `frontend/src/i18n/help/*.js` — la fiche
  autorise explicitement cette option quand aucune commande `:vacuum` n'est
  ajoutée à `commandProcessor.js`/`commandVocabulary.js`.

Mesure obtenue dans `TestVacuum_ReclaimsSpaceAfterDeletes`
(`pkg/blunderdb/database/db_vacuum_test.go`) : une base gonflée à ~6,78 Mio
par 3000 lignes de remplissage puis vidée à une seule ligne survivante
retombe à 249 856 octets après `Vacuum()` — 6 528 kio (~96 %) récupérés — et
la ligne survivante est vérifiée intacte. Le même scénario est rejoué côté
CLI dans `TestCLI_Vacuum_ReclaimsSpace`
(`internal/cli/cli_vacuum_test.go`), en passant par `blunderdb vacuum --db`.
