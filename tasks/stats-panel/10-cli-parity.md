# 10 — CLI parity (`list --type stats`)

**Goal:** Étendre la commande existante `./blunderdb list --type stats` pour qu'elle reproduise en texte le contenu du Dashboard : PR global / checker / cube, MWC si applicable, rolling N, top blunders. Flag `--metric pr|mwc` pour basculer la métrique affichée. Pas de dépendance ajoutée.

**Depends on:** 02 (MWC intégré à `ComputeStats`)

**Impact:** Cohérence CLI/GUI. Utile pour scripts, tests automatisés, rapports.

## Context

- `cli.go` — fonction `showStats()` actuelle retourne juste des compteurs bruts (cf. plan §Context).
- Pattern CLI : `tabwriter` pour l'alignement, couleurs optionnelles via `tty` detection.
- Les flags sont parsés via `flag.FlagSet` dans `cli.go` (cf. autres sous-commandes).

## Tasks

### 1. Étendre `showStats`

- [ ] Ajouter des flags à la sous-commande existante :
  - `--metric pr|mwc` (défaut `pr`).
  - `--player <name>` (défaut vide = toutes perspectives).
  - `--tournament <id>` (répétable ou CSV).
  - `--from <YYYY-MM-DD>`, `--to <YYYY-MM-DD>`.
  - `--decision-type all|checker|cube` (défaut `all`).
  - `--top-blunders N` (défaut 10).
  - `--format text|json` (défaut `text`). JSON utile pour intégrations.
- [ ] Construire `StatsFilter` à partir des flags, appeler `db.ComputeStats(filter)`.

### 2. Format texte

- [ ] Sections ordonnées :
  1. **Header** : base, filtre appliqué (récapitulatif), métrique choisie.
  2. **Totals** : positions, matchs, tournois, décisions.
  3. **PR/MWC** : global, checker, cube — aligné en colonne.
  4. **Rolling** : 5, 10, 50, 100, 250, 500, 1000 — tableau avec N, PR/MWC, nb décisions effectivement utilisées.
  5. **Top N blunders** : position ID, type, magnitude EMG, nom du match, date.
  6. **Cube action breakdown** : NoDouble/DoubleTake/DoublePass/TooGood — lignes avec nb décisions, blunder rate, PR.
- [ ] Utiliser `tabwriter.NewWriter` pour l'alignement propre.
- [ ] Afficher `—` (em-dash) pour les valeurs MWC indisponibles (money-game).

### 3. Format JSON

- [ ] Marshaller `StatsResult` directement via `encoding/json.MarshalIndent`.
- [ ] Prérequis : tags JSON sur `StatsResult` et ses sous-structs (fiche 01 peut les ajouter si pas déjà fait).

### 4. Tests

- [ ] `cli_stats_test.go` (nouveau ou étendu) :
  - Exécute `showStats` avec différents flags sur une fixture.
  - Vérifie que la sortie texte contient les sections attendues.
  - Vérifie que `--format json` produit un JSON parseable avec les bons champs.
  - Vérifie que `--metric mwc` change la présentation (labels « MWC » au lieu de « PR »).
- [ ] Test de régression : l'ancienne invocation sans flags (`list --type stats`) produit toujours une sortie utile (défauts raisonnables).

### 5. Documentation CLI

- [ ] Mettre à jour `CLI_USAGE.md` avec la nouvelle section `stats` détaillée :
  - Description des flags.
  - Exemples d'invocation typiques.
  - Format de sortie text et JSON (extrait).

## Acceptance criteria

- [ ] `./blunderdb list --db testdata/Quiz.db --type stats` produit un rapport texte lisible.
- [ ] `./blunderdb list --db testdata/Quiz.db --type stats --metric mwc` bascule en MWC.
- [ ] `./blunderdb list --db testdata/Quiz.db --type stats --format json` produit un JSON valide.
- [ ] Les valeurs CLI correspondent bit-à-bit à celles du Dashboard GUI pour le même filtre (test manuel ou diff scripté).
- [ ] `go test -run TestCLIStats ./...` vert.
- [ ] `go test ./...` reste vert.

## Rollback

Revert = `git revert`. Les flags ajoutés sont rétrocompatibles (tous avec défauts).
