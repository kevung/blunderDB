# Diriger un tournoi — intégration au reste du logiciel

Ce que le chantier touche, fichier par fichier, et ce qu'il ne touche pas. Les règles sont
dans [fonctionnel.md](fonctionnel.md), les gestes dans [ux.md](ux.md), le moteur dans
[nicomaque.md](nicomaque.md).

## 1. Base de données

`DatabaseVersion` **2.23.0** (2.22.0 est prise par le journal d'Entraînement), dans les trois
copies (`database/db_schema.go` + `db_migration.go`, `storage/sqlite/schema_sqlite.go`,
`storage/postgres/migrations/024_direction.sql`), avec un pas `migrate_2_22_0_to_2_23_0`
enregistré dans `migrationSteps` et un test dans `migration_test.go`.

```sql
CREATE TABLE direction (
  tournament_id INTEGER PRIMARY KEY REFERENCES tournament(id) ON DELETE CASCADE,
  created_at DATETIME, updated_at DATETIME,
  format_version TEXT NOT NULL,      -- version du journal Nicomaque
  engine_version TEXT NOT NULL,      -- tag du module, pour l'info et le crédit
  state TEXT NOT NULL,               -- draft | running | finished
  config TEXT NOT NULL,              -- la configuration en préparation (réécrite tant que draft)
  output_dir TEXT DEFAULT ''         -- dossier de la page HTML
);
CREATE TABLE direction_event (
  tournament_id INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  kind TEXT NOT NULL, time DATETIME NOT NULL,
  payload TEXT NOT NULL,             -- l'événement Nicomaque, JSON
  PRIMARY KEY (tournament_id, seq)
);
```

`match` gagne `direction_match_id TEXT DEFAULT ''` (l'identifiant Nicomaque du Slot, `M12`),
avec un index unique partiel `(tournament_id, direction_match_id)` quand la valeur est non
vide. `direction_event` est **append-only** : aucun `UPDATE`, aucun `DELETE` hors la cascade
de suppression du Tournament ou de la Direction.

L'export (`ExportDatabase`, `ExportTournaments`) emporte les deux tables pour les tournois
exportés et étend `issuance.CarriedMetadataKeys` si un document s'y ajoute — jamais par
exclusion (invariant de `CLAUDE.md`).

## 2. Moteur et paquet `direction`

Nouveau paquet `pkg/blunderdb/direction/` : la seule chose qui connaît Nicomaque.

- `Open(store, tournamentID) (*Direction, error)` : lit les événements, `Replay`, garde
  l'état en mémoire ; `Propose()`, `Apply(ev)` (écrit puis applique), `State()`, `Warnings()`.
- Traduction des codes (D9, N1) : le paquet expose les codes ; les libellés sont rendus par
  le frontend et par `help-gen`, jamais par Go.
- Rattachement : `AttachMatch(slot, matchID)`, `DetachMatch(slot)`, `SuggestAttachments()`.
- Sorties : `WritePage(dir)`, `PairingSheet(round)`, `StandingsCSV()`, `DirectoryCSV()`,
  `JournalJSON()`.
- Annuaire : `Directory(store)` — vue dérivée sur toutes les Directions de la base.
- `go.mod` : `github.com/PileOfCells/backgammon-tournoi v0.x.y` épinglé au tag.

Le paquet ne touche ni `engine/` ni `transcript/`. Il ne fait pas de SQL : il passe par le
contrat `Storage` (nouveau `DirectionStore`) et par `Database` pour le chemin bureau/CLI.

## 3. Interface Wails (`internal/gui/`)

Un fichier `direction.go` : lister/créer/ouvrir/supprimer une Direction, ajouter un
événement, obtenir l'état et les propositions, rattacher/détacher, produire les sorties,
choisir le dossier de la page HTML (dialogue système). Les méthodes sont liées et
`frontend/wailsjs/` régénéré (jamais édité à la main). Aucune méthode ne prend de texte
traduit : que des codes et des identifiants.

## 4. Frontend

- `stores/directionStore.js` : la Direction ouverte, son état, ses propositions, ses
  avertissements, l'horloge ; un seul store, rechargé après chaque événement.
- `components/direction/` : `DirectionView.svelte` (la zone principale), `ProposalList`,
  `TableGrid`, `ResultCard`, `PlayersView`, `BracketsView`, `StandingsView`,
  `HistoryView`, `SettingsView`, `CreditModal`.
- `App.svelte` : la zone principale affiche `DirectionView` au lieu de `Board` quand
  l'onglet Tournoi est actif et qu'une Direction est ouverte (D15) — le seul endroit du
  fichier qui change.
- `TournamentPanel.svelte` : une ligne « dirigé » par tournoi, [Ouvrir la direction] /
  [Diriger ce tournoi], les Slots à côté des matchs rattachés.
- `commandProcessor.js` + `commandVocabulary.js` (verrouillés par
  `commandVocabulary.sync.test.js`) : `direct`, `result`, `start`, `withdraw`, `table`,
  `propose`.
- Règle Svelte 5 : `$store` / `$effect`, jamais `.subscribe()`. Tokens de taille de
  `style.css` ; l'exception typographique de la fiche de résultat est nommée dans ADR-0008.

## 5. CLI (D10)

`internal/cli/cli_tournament.go`, enregistré dans `handlers()` (liste unique) :
`blunderdb tournament verify|standings|page|export|list`. Non interactif : la console de TD
est `tournoi-td`, côté Nicomaque. `CLI_USAGE.md`, `cli.rst` et `blunderdb help` reçoivent la
commande et ses sous-commandes (`scripts/doc-inventory.sh` bloque sinon).

## 6. Documentation

- `manuel.rst` : une section « Diriger un tournoi » (les six vues, la fiche de résultat, les
  sorties, le crédit) ; `guide_utilisateur.rst` : une tâche « diriger son premier tournoi »
  qui renvoie au manuel par `:ref:` (règle « le guide montre une tâche, le manuel décrit un
  écran »).
- `raccourcis.rst` : `Ctrl+Maj+D`, `j`/`k`/`Entrée`, chiffres, `Échap`.
- `cmd_mode.rst` : les six commandes. `cli.rst` : la sous-commande.
- `a_propos.rst` : le crédit Nicomaque / Nicolas Harmand et le lien.
- Les **huit `.po`** dans le même commit (`scripts/doc-po-update.sh`, puis
  `scripts/doc-i18n-check.sh` doit finir sur « all translations complete »).
- `make help` après chaque changement de ces trois sources (ADR-0034).
- Prix estimé : ~120 lignes de `.rst` → ~2 400 lignes de `.po` sur huit langues (règle « une
  page est proposée avec son prix »).

## 7. Ce qui n'est pas touché

`engine/` et ses sous-paquets, `transcript/` (sauf l'en-tête hérité d'un Slot, qui passe par
les champs existants), la recherche, les statistiques, l'Anki, l'Entraînement, l'import de
fichiers, `internal/server/` et le front web (D4, ADR-0039), le Zobrist, la Corbeille, la
purge des orphelins, `issuance` (hors la liste blanche d'export).

## 8. Risques

| Risque | Parade |
|---|---|
| Nicomaque n'a jamais dirigé un vrai tournoi | le lot 1 se termine par un tournoi de club réellement dirigé, en doublant sur papier ; ce qui manque devient des issues |
| Le format du journal change après un vrai tournoi | N1 (version + codes) est fait **avant** ; un journal `Version: 0` reste rejouable |
| La zone principale sans plateau | prototype jetable avant le lot 1 ; le plateau revient dès qu'on change d'onglet |
| Deux dépôts à faire avancer ensemble | chaque issue blunderDB nomme l'issue Nicomaque qu'elle consomme et le tag attendu |
| Les neuf langues du site Nicomaque | même chaîne que blunderDB, scripts réutilisés (N13) |
