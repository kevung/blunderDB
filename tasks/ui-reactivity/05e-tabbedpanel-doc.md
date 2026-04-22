# 05.e — TabbedPanel : règle keep-alive + `data-testid`

**Goal :** Documenter explicitement la contrainte que le pattern keep-alive du TabbedPanel impose aux composants enfants, et ajouter les `data-testid` stratégiques requis par les specs Playwright.

**Depends on :** 04.

**Impact :** Prévient la ré-introduction du bug (un nouveau composant enfant qui dépendrait de données non visibles via `$effect` créerait un effet fantôme même en tab caché). Fournit aux tests E2E des sélecteurs stables.

## Context

Extrait actuel (TabbedPanel.svelte:34-45) :

```js
let mountedTabs = $state(new Set([get(activeTabStore)]));
$effect.pre(() => {
    const tab = $activeTabStore;
    if (tab && !mountedTabs.has(tab)) {
        mountedTabs = new Set([...mountedTabs, tab]);
    }
});
```

Le pattern en lui-même est correct (keep-alive : une fois monté, un onglet reste dans le DOM). Mais les composants enfants gardent leurs `$effect` actifs même en onglet caché, ce qui peut créer des mises à jour inutiles si leurs effets dépendent de données qui bougent en arrière-plan.

## Files touched

- **Edit:** `frontend/src/components/TabbedPanel.svelte` — commentaire d'entête + optionnel `data-testid`.
- **Edit (ponctuel):** les boutons d'onglets dans `TabbedPanel.svelte` ou le composant parent qui les rend — ajouter `data-testid="tab-<name>"` sur chaque onglet.
- **Edit:** `frontend/src/components/StatusBar.svelte` — `data-testid="status-bar"` sur le div racine.

## Tasks

### 1. Commentaire d'entête TabbedPanel

- [ ] Ajouter en tête du `<script>` de `TabbedPanel.svelte` un commentaire court (5-8 lignes) :
  ```svelte
  <!--
    TabbedPanel utilise un pattern keep-alive : un onglet reste monté après
    sa première visite. Conséquence : les $effect des composants enfants
    tournent même quand l'onglet est caché. Les enfants ne doivent donc pas
    dépendre dans leurs effets de données qui changent en arrière-plan sans
    pertinence pour leur onglet (sinon : travail gaspillé, cascades, logs perf
    qui se déclenchent inutilement). Voir tasks/ui-reactivity/ pour la règle.
  -->
  ```
- [ ] Seul commentaire autorisé par la fiche : c'est une contrainte invariante du pattern, pas une doc « ce que fait le code ».

### 2. Data-testid

- [ ] Pour chaque onglet rendu dans `TabbedPanel.svelte` (la barre de tabs), ajouter `data-testid="tab-{tabName}"` sur le bouton/lien.
  - Ex : `<button data-testid="tab-stats" onclick={...}>Stats</button>`.
- [ ] Ajouter `data-testid="tab-panel-{tabName}"` sur le conteneur de chaque panneau enfant (optionnel — utilisable par Playwright pour vérifier qu'un panneau est rendu/caché).
- [ ] Ajouter `data-testid="status-bar"` sur le `div.status-bar` de `StatusBar.svelte`. Sur l'info-message, `data-testid="status-bar-message"`.
- [ ] Garder ces `data-testid` au strict minimum — un par surface testable. Ne pas ajouter à tous les éléments.

### 3. Montage paresseux (optionnel, à évaluer)

- [ ] Si la Fiche 04 (audit) révèle que certains panels sont coûteux à monter et qu'un montage systématique au premier affichage pose un vrai problème de perf : passer en **montage 100% paresseux** via `{#if activeTab === '<x>' || mountedTabs.has('<x>')}`. Déjà le cas dans le code actuel, vérifier.
- [ ] Sinon : statu quo.

### 4. Vérif

- [ ] `wails dev` : aucun changement visible pour l'utilisateur.
- [ ] Spec Playwright (Fiche 02) peut utiliser les `data-testid` ajoutés → specs plus lisibles et stables.

### 5. Commit

- [ ] `docs(ui): document TabbedPanel keep-alive rule + add stable data-testid`.

## Acceptance

- [ ] Commentaire d'entête en place.
- [ ] `data-testid` ajoutés sur onglets + status bar.
- [ ] Specs Playwright utilisent ces sélecteurs (à mettre à jour si déjà écrites).
- [ ] Pas de régression UI.

## Status

- [ ] Commentaire TabbedPanel
- [ ] data-testid sur onglets
- [ ] data-testid sur StatusBar
- [ ] Specs Playwright mises à jour si besoin
- [ ] Commit
