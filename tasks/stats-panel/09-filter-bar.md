# 09 — Barre de filtre commune

**Goal:** Peupler `StatsFilterBar.svelte` : barre partagée par les 3 onglets qui expose joueur (auto-détecté), tournois (multi-select), plage de dates, type de décision (radio), longueur de match. Persister le filtre dans `config.yaml` via la struct `Config` existante.

**Depends on:** 05

**Impact:** Sans ça, le panneau affiche toujours la perspective toutes-confondues — l'utilisateur ne peut pas se restreindre à ses propres matchs.

## Context

- Plan §Barre de filtre, §Décisions tranchées (auto-détection + override).
- Store : `statsFilterStore` dans `statsStore.js` (fiche 04).
- Backend : `StatsFilter` Go (fiche 01) — les types doivent matcher.
- Liste des joueurs : dériver de `match.player1_name ∪ match.player2_name` en SQL (une nouvelle méthode `GetAllPlayerNames`). Fréquence utilisée pour auto-détection.
- Config existante : `config.go` (struct `Config`, méthode `Load`, `Save`). Ajouter un sous-objet `StatsFilter` avec les champs persistés.

## Tasks

### 1. Backend : `GetAllPlayerNames`

- [ ] Dans `db_stats.go` :
  ```go
  type PlayerFrequency struct {
      Name  string
      Count int  // nb de matchs où ce nom apparaît
  }
  func (db *Database) GetAllPlayerNames() ([]PlayerFrequency, error)
  ```
- [ ] SQL : `SELECT name, COUNT(*) FROM (SELECT player1_name AS name FROM match UNION ALL SELECT player2_name FROM match) WHERE name != '' GROUP BY name ORDER BY COUNT(*) DESC`.
- [ ] Test : fixture avec 5 matchs, Alice en p1 dans 3, en p2 dans 2, Bob en p2 dans 5. `GetAllPlayerNames()` retourne `[{Alice, 5}, {Bob, 5}]` ou ordre alphabétique au rang égal.

### 2. Persistance dans `Config`

- [ ] Étendre `config.go` struct `Config` :
  ```go
  type Config struct {
      // … existing …
      StatsFilter StatsFilterPersisted `yaml:"stats_filter,omitempty"`
  }
  type StatsFilterPersisted struct {
      PlayerName    string   `yaml:"player_name"`
      TournamentIDs []int64  `yaml:"tournament_ids"`
      DateFrom      string   `yaml:"date_from"`
      DateTo        string   `yaml:"date_to"`
      DecisionType  int      `yaml:"decision_type"`
      MatchLength   []int    `yaml:"match_length"`
      Metric        string   `yaml:"metric"` // "pr" | "mwc"
  }
  ```
- [ ] Méthodes bound à Wails : `GetStatsFilter()` et `SaveStatsFilter(filter)` sur `*Config`.
- [ ] Tests : écriture puis relecture via `config_test.go`.

### 3. Auto-détection du joueur

- [ ] Côté frontend, au premier affichage (filtre vide + `PlayerName === ''` dans la config) :
  1. Appeler `GetAllPlayerNames()`.
  2. Prendre `result[0].Name` s'il existe.
  3. Mettre à jour `statsFilterStore` + appeler `SaveStatsFilter`.
- [ ] Si la config contient déjà `PlayerName`, ne pas auto-détecter.

### 4. Composant `StatsFilterBar.svelte`

- [ ] Layout horizontal compact (§Principes UX 4 : densité faible) : une ligne avec tous les contrôles, possiblement enroulée sur 2 lignes en mobile.
- [ ] Contrôles :
  - **Joueur** : `<select>` alimenté par `GetAllPlayerNames()`. Option « Toutes perspectives » pour désactiver le filtre. Option affichant le nom + fréquence entre parenthèses.
  - **Tournois** : multi-select compact (composant custom ou `<select multiple>` sobre). Option « Tous » par défaut.
  - **Plage de dates** : 2 `<input type="date">` côte à côte (From / To).
  - **Type de décision** : 3 boutons radio (All / Checker / Cube).
  - **Longueur de match** : dropdown multi-select ou série de checkboxes rapides (1, 3, 5, 7, 9, 11, 13, 15, 21, Other).
- [ ] Bouton « Reset filters » à droite qui remet le filtre par défaut (sauf le joueur auto-détecté).
- [ ] Styles discrets : fond panneau, inputs bordés minimalement, pas de labels en gras.

### 5. Synchronisation filtre ↔ store ↔ config

- [ ] Chaque modification de contrôle → update `statsFilterStore` → trigger `refreshStats()` (déjà abonné dans `StatsPanel`) + appel `SaveStatsFilter` avec debounce 500 ms.
- [ ] Toggle PR/MWC (header du panneau) → update `statsMetricStore` → `SaveStatsFilter` (le toggle est persisté aussi).

### 6. Gestion des cas limites

- [ ] Aucun match dans la base → liste joueurs vide, contrôles visibles mais désactivés avec un hint « Importez des matchs pour activer les filtres ».
- [ ] Joueur sélectionné n'existe plus (base changée) → revenir à « Toutes perspectives », alerter discrètement.
- [ ] `DateFrom > DateTo` → afficher une bordure rouge discrète, ne pas refresh tant que la plage est invalide.

### 7. Tests Vitest

- [ ] Mount avec store mocké, vérifier que toutes les options apparaissent.
- [ ] Changer le select joueur → `statsFilterStore.playerName` mis à jour.
- [ ] Changer la date → store mis à jour.
- [ ] Bouton reset → filtre par défaut restauré (sauf `PlayerName`).
- [ ] Auto-détection : monter avec config vide et 3 joueurs (freq 10/5/2) → `statsFilterStore.playerName === 'TopPlayer'` (celui à freq 10).

## Acceptance criteria

- [ ] Barre de filtre visible et fonctionnelle dans les 3 onglets.
- [ ] Filtre persiste entre redémarrages (via `config.yaml`).
- [ ] Auto-détection du joueur au premier lancement.
- [ ] Respect §Principes UX 4 (layout compact, pas de verbiage), 8 (états vides gérés).
- [ ] `npm test` + `go test ./...` verts.

## Rollback

Revert = `git revert`. La struct `Config` supporte l'omission de `stats_filter` (`omitempty`) — les utilisateurs existants ne voient aucune diff.
