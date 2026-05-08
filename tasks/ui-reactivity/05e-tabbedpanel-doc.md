# 05.e — TabbedPanel : règle {#if} + `data-testid`

**Goal :** Documenter explicitement la contrainte que le pattern de rendu du TabbedPanel impose aux composants enfants, et ajouter les `data-testid` stratégiques requis par les specs Playwright.

**Depends on :** 04.

**Impact :** Prévient la ré-introduction du bug (un composant enfant qui stockerait son état dans `$state` local perdrait cet état au changement d'onglet). Fournit aux tests E2E des sélecteurs stables.

## Context

**Note 2026-04-27 :** le code actuel de `TabbedPanel.svelte` utilise un pattern `{#if $activeTabStore === 'x'}` simple — les panneaux sont démontés à chaque changement d'onglet, contrairement au pattern keep-alive (`mountedTabs`) décrit initialement dans cette fiche. Le commentaire ajouté reflète le comportement réel.

Le pattern `{#if}` signifie que les `$effect` des enfants ne tournent PAS en arrière-plan, mais que tout état local (`$state`) est réinitialisé à chaque visite. Les stores Svelte doivent porter l'état persistant.

## Files touched

- **Edit:** `frontend/src/components/TabbedPanel.svelte` — commentaire d'entête.
- **Edit:** `frontend/src/components/StatusBar.svelte` — `data-testid="status-bar-message"` sur le span info-message.

> `data-testid="tab-{tab.id}"` et `data-testid="status-bar"` étaient déjà présents avant cette fiche.

## Tasks

### 1. Commentaire d'entête TabbedPanel

- [x] Commentaire ajouté avant `<script>` dans `TabbedPanel.svelte` (7 lignes, contrainte invariante du pattern `{#if}`) :
  ```svelte
  <!--
    TabbedPanel utilise un pattern {#if} : les panneaux enfants sont démontés quand
    on quitte leur onglet et remontés au retour. Contrainte pour les composants
    enfants : tout état local ($state) est réinitialisé à chaque visite — stocker
    dans un store Svelte tout état devant survivre aux changements d'onglet.
    Éviter de faire dépendre un $effect d'une valeur de store « active en
    arrière-plan » : l'effet sera de toute façon inactif hors onglet.
    Voir tasks/ui-reactivity/ pour la règle générale.
  -->
  ```

### 2. Data-testid

- [x] `data-testid="tab-{tab.id}"` sur chaque bouton d'onglet — **déjà présent** avant cette fiche.
- [ ] ~~`data-testid="tab-panel-{tabName}"`~~ — non ajouté (optionnel, non requis par les specs actuelles).
- [x] `data-testid="status-bar"` sur `div.status-bar` — **déjà présent** avant cette fiche.
- [x] `data-testid="status-bar-message"` ajouté sur le `<span class="info-message">` de `StatusBar.svelte`.

### 3. Montage paresseux (optionnel)

- [x] **Statu quo** — code actuel utilise `{#if}` simple ; aucun problème de perf identifié en fiche 04.

### 4. Vérif

- [x] Tests Vitest : 321 tests verts, aucune régression.
- [x] Specs Playwright utilisent `[data-testid="tab-*"]` et `[data-testid="status-bar"]` — sélecteurs stables confirmés.

### 5. Commit

- [x] `docs(ui): TabbedPanel {#if} constraint comment + status-bar-message testid`.

## Acceptance

- [x] Commentaire d'entête en place.
- [x] `data-testid` complets sur onglets + status bar (message span inclus).
- [x] Specs Playwright utilisent ces sélecteurs (déjà écrites, aucune mise à jour nécessaire).
- [x] Pas de régression UI (321 tests verts).

## Status

- [x] Commentaire TabbedPanel
- [x] data-testid sur onglets (déjà présent)
- [x] data-testid sur StatusBar + message span
- [x] Specs Playwright vérifiées (aucune MàJ nécessaire)
- [x] Commit
