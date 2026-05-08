# 00 — Scénarios de reproduction & branche de travail

**Goal :** Figer des scénarios déterministes qui reproduisent les symptômes de non-réactivité, puis créer la branche de travail où atterriront tous les fixes.

**Depends on :** —

**Impact :** Document de référence pour les tests automatisés (01, 02) et pour valider les fixes (05.*). Branche isolée pour tracer les changements.

## Context

Les symptômes rapportés par l'utilisateur :
- Transitions entre panneaux parfois instantanées, parfois lentes, parfois bloquées.
- Informations EPC dans la barre d'état potentiellement figées.

Hypothèse principale issue de l'exploration : le handler `activeTabStore.subscribe()` dans `App.svelte` ne traite pas les onglets `stats`, `tournaments`, `collections`, `anki`, et mélange `.subscribe()` (Svelte 4) avec des runes (Svelte 5).

## Files touched

- **New:** `doc/archive/ui-reactivity-scenarios.md`
- **Git:** créer branche `ui-reactivity` depuis `stat_panel` (et non `main`, car l'onglet Stats source des bugs n'existe pas sur `main`).

## Tasks

### 1. Tests manuels

- [x] S1 — EPC 66.47 conservé au retour d'onglet avec position inchangée (stat_panel).
- [ ] S1 étendu — charger une position A, noter EPC_A, bascule Stats, charger position B (via commande), retour EPC, vérifier que EPC affiché = EPC_B.
- [x] S2 — transitions impliquant Stats cassées : Match↔Stats, EPC↔Stats, Stats↔Anki. Match↔EPC OK.
- [x] S3 — édition plateau en mode EPC met à jour la barre d'état en temps réel.

### 2. Rédaction du document de scénarios

- [x] `doc/archive/ui-reactivity-scenarios.md` créé avec :
  - Contexte (pourquoi `main` n'est pas le baseline).
  - 3 scénarios avec étapes atomiques et observables.
  - Résultats observés le 2026-04-22.
  - Traduction en tests automatisés (pointeurs vers Fiches 01 et 02).

### 3. Branche

- [x] `git checkout -b ui-reactivity` depuis `stat_panel`.
- [ ] Push optionnel : `git push -u origin ui-reactivity` quand le premier commit significatif sera prêt (laisser à la discrétion utilisateur).

## Acceptance

- [x] Au moins 3 scénarios déterministes rédigés avec étapes atomiques.
- [x] Branche `ui-reactivity` créée.
- [x] Le document pointe les tests automatisés à écrire en Fiche 01/02.

## Status

- [x] Tests manuels primaires (S1 base, S2, S3)
- [ ] Test manuel S1 étendu (position différente entre visites EPC)
- [x] Document scénarios rédigé
- [x] Branche créée
