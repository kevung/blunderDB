# 11 — Docs FR/EN, test E2E, changelog

**Goal:** Documenter le panneau Stats dans la doc Sphinx bilingue (`doc/source/fr/` et `doc/source/en/`), écrire un scénario E2E Vitest qui simule un parcours utilisateur complet (filtre → onglet → clic → drill-down), ajouter une entrée au changelog et préparer la release.

**Depends on:** 06, 07, 08, 09, 10

**Impact:** Finalisation. Sans docs, utilisateurs ne découvrent pas la feature ; sans test E2E, les régressions futures passeront sous le radar.

## Context

- `doc/source/` : Sphinx FR + EN, build via `cd doc && python build.py` (requis : `doc/requirements.txt` + LaTeX pour PDF).
- `doc/archive/` : notes de design archivées ; ne pas publier les fiches de tâche ici.
- `CHANGELOG.md` / entrée dans `doc/source/index.rst` : format suivi par `scripts/release.sh <version>`.
- Test E2E : Vitest n'est pas un runner E2E natif (c'est du unit test). Interpréter ici comme « test d'intégration qui monte `StatsPanel` avec tous ses onglets et simule les interactions ».

## Tasks

### 1. Documentation FR

- [x] Nouveau fichier `doc/source/stats.rst` (intégré dans la structure existante, ajouté au toctree).
- [x] Sections :
  1. **Introduction** — à quoi sert le panneau, quand l'ouvrir.
  2. **Ouverture** — commande `:stats` / `:st`.
  3. **Barre de filtre** — expliquer la perspective joueur, l'auto-détection, la plage de dates.
  4. **Toggle PR / MWC** — expliquer la différence :
     - PR = erreur normalisée money-game, scalaire single-value (seuils world-class / expert / …).
     - MWC cost = probabilité cumulée de victoire de match perdue par les erreurs (cumul sur le set).
     - Quand chacun est pertinent.
  5. **Onglet Dashboard** — description des cartes, interaction.
  6. **Onglet Progression** — courbe par tournoi, scatter par match, bandes de grade, menu contextuel de clic.
  7. **Onglet Erreurs** — breakdown cube, histogramme magnitudes.
  8. **Règle d'agrégation** — **expliquer explicitement** que le PR d'un tournoi est pondéré par décisions, pas moyenné sur les matchs. Exemple chiffré.
  9. **MWC : limitations** — money-game non applicable, valeurs qui dépendent de la MET Kazaross-XG2.
- [ ] Inclure 2-3 captures d'écran (onglet Dashboard, onglet Progression) sous `doc/source/_static/`.

### 2. Documentation EN

- [x] Traduction de tous les points ci-dessus dans `doc/source/locale/en/LC_MESSAGES/stats.po`.
- [x] Relire pour cohérence terminologique avec la doc existante (PR, MWC, blunder, etc.).

### 3. Changelog

- [x] Ajouter entrée à `doc/source/index.rst` (section changelog) :
  - « v0.19.0 — Added Stats panel with PR/MWC metrics, per-tournament progression chart, error-type breakdown, and interactive drill-down. »
- [x] Référence au nouveau fichier doc (`:ref:\`stats\``). 

### 4. Test d'intégration

- [x] `frontend/src/__tests__/StatsPanel.integration.test.js` :
  - Mock complet de Wails bindings avec une fixture de `StatsResult` réaliste.
  - Monte `StatsPanel`, vérifie que les 3 onglets s'affichent.
  - Navigue Dashboard → clique une carte → vérifie que `loadPositionsFromStatsSelection` a été appelé avec le bon `SelectionSpec`.
  - Change le filtre joueur → vérifie que `ComputeStats` est rappelé avec le nouveau filtre.
  - Bascule PR → MWC → vérifie que les valeurs affichées changent (sans refetch).
  - Passe en onglet Progression → clic sur un point de la courbe → menu contextuel → clic sur « Open tournament » → vérifie que `openTournamentInPanel` a été appelé.
  - Passe en onglet Erreurs → clic sur barre DoubleTake → vérifie que `loadPositionsFromStatsSelection` est appelé avec `{Kind: 'cube_action', CubeAction: 'DoubleTake', OnlyWithError: true}`.

### 5. Test manuel de bout en bout

- [ ] Rédiger un script de test manuel dans la fiche (à conserver avec la fiche) : liste d'étapes à exécuter dans `wails dev` sur une base réelle, couvrant tous les points de la §Vérification end-to-end du plan.
- [ ] Cocher les étapes lorsque effectuées.

### 6. Release

- [ ] Exécuter `scripts/release.sh <version>` — lit, met à jour la version dans `doc/source/conf.py`, `frontend/src/stores/metaStore.js` et crée le commit + tag.
- [ ] Push du tag déclenche CI matrix build + release GitHub.
- [ ] **Ne pas toucher** `DatabaseVersion` (schéma inchangé).

## Acceptance criteria

- [ ] `cd doc && python build.py` produit FR + EN sans warning.
- [x] Entrée changelog présente avec référence au doc.
- [x] `StatsPanel.integration.test.js` passe (29 tests, 0 failure).
- [ ] Une exécution manuelle du script de test E2E valide les 8 points de §Vérification end-to-end du plan.
- [ ] CI matrix verte sur les 4 plateformes (ubuntu-latest, ubuntu-22.04, windows-latest, macos-latest).

## Rollback

Revert = `git revert`. La release elle-même, si déjà publiée : nouvelle release patch qui désactive le panneau (au pire, commenter le mount dans `App.svelte`).
