# 05 — Stats panel shell + Chart.js integration

**Goal:** Créer le composant racine `StatsPanel.svelte` avec ses 3 onglets vides, intégrer Chart.js + wrappers Svelte 5 natifs (`<canvas>` + `$effect`), poser le toggle global PR/MWC et la commande `:stats`. Les onglets ne contiennent encore aucun contenu — fiches 06/07/08 les peuplent.

**Depends on:** 04

**Impact:** Cadre commun UI ; structure de rendu, thème et patterns de chart réutilisables.

## Context

- Plan §Principes UX — **critères d'acceptation** de cette fiche : densité faible, palette restreinte, cohérence avec `MatchPanel`/`TournamentPanel`.
- Chart.js v4 : `npm install chart.js`. Pas de wrapper (`svelte-chartjs` est Svelte 3/4).
- Pattern Svelte 5 runes : `$effect` pour init/teardown du chart, `$props()` pour les données, `$derived` pour les options.
- Pattern composant panneau : suivre `frontend/src/components/MatchPanel.svelte` (header avec titre et actions, corps scrollable).

## Tasks

### 1. Installer Chart.js

- [x] `cd frontend && npm install chart.js@^4`
- [x] Vérifier que `package.json` contient la dépendance en `dependencies` (pas `devDependencies`).

### 2. Écrire les wrappers charts

- [x] Dossier `frontend/src/components/stats/charts/`.
- [x] `LineChart.svelte` — reçoit `{ labels, datasets, options, onPointClick }`. Utilise un `<canvas bind:this={canvas}>` et un `$effect` qui :
  1. détruit l'instance précédente si existe,
  2. crée `new Chart(canvas, { type: 'line', data: { labels, datasets }, options })`,
  3. attache un handler `onClick` qui appelle `onPointClick(dataIndex, datasetIndex)` si fourni,
  4. retourne un cleanup `chart.destroy()`.
- [x] `BarChart.svelte` — idem, `type: 'bar'`.
- [x] `ScatterChart.svelte` — idem, `type: 'scatter'`. Support du point size via `options.datasets.scatter.pointRadius`.
- [x] `Histogram.svelte` — `BarChart` avec `options.scales.x.type = 'category'` et labels = limites de bucket.
- [x] Commun : `options.responsive = true`, `options.maintainAspectRatio = false`, `options.plugins.tooltip` configuré (hover = info secondaire, cf. §Principes UX).
- [x] Palette partagée : exporter des constantes depuis `frontend/src/components/stats/charts/palette.js` (couleur primaire, couleurs des bandes de grade backgammon, couleur neutre des gridlines).

### 3. `StatsPanel.svelte` squelette

- [x] `frontend/src/components/stats/StatsPanel.svelte`.
- [ ] Structure :
  ```svelte
  <script>
    import { onMount } from 'svelte';
    import { statsFilterStore, statsResultStore, statsLoadingStore, statsMetricStore, refreshStats } from '../../stores/statsStore';
    import StatsFilterBar from './StatsFilterBar.svelte';
    import StatsDashboardTab from './StatsDashboardTab.svelte';
    import StatsProgressionTab from './StatsProgressionTab.svelte';
    import StatsErrorsTab from './StatsErrorsTab.svelte';

    let activeTab = $state('dashboard');
    let unsubscribeFilter;
    onMount(() => {
      refreshStats();
      unsubscribeFilter = statsFilterStore.subscribe(() => refreshStats());
      return () => unsubscribeFilter?.();
    });
  </script>

  <div class="stats-panel">
    <header>
      <h2>Stats</h2>
      <div class="metric-toggle">
        <button class:active={$statsMetricStore === 'pr'} onclick={() => statsMetricStore.set('pr')}>PR</button>
        <button class:active={$statsMetricStore === 'mwc'} onclick={() => statsMetricStore.set('mwc')}>MWC</button>
      </div>
    </header>
    <StatsFilterBar />
    <nav class="tabs">
      <button class:active={activeTab === 'dashboard'} onclick={() => activeTab = 'dashboard'}>Dashboard</button>
      <button class:active={activeTab === 'progression'} onclick={() => activeTab = 'progression'}>Progression</button>
      <button class:active={activeTab === 'errors'} onclick={() => activeTab = 'errors'}>Errors</button>
    </nav>
    <div class="tab-content">
      {#if $statsLoadingStore}
        <p>Loading…</p>
      {:else if activeTab === 'dashboard'}
        <StatsDashboardTab />
      {:else if activeTab === 'progression'}
        <StatsProgressionTab />
      {:else if activeTab === 'errors'}
        <StatsErrorsTab />
      {/if}
    </div>
  </div>
  ```
- [x] Styles : reprendre les tokens de couleur/typographie utilisés dans `MatchPanel.svelte`. Largeur min 480 px (`min-width: 480px`). Chaque onglet a `overflow-y: auto`.
- [x] **Règle UX** : layout sobre, un h2, une barre de filtre, trois onglets, zone de contenu. Pas de card colorée, pas de gradient.

### 4. Stubs des onglets

- [x] Créer `StatsDashboardTab.svelte`, `StatsProgressionTab.svelte`, `StatsErrorsTab.svelte` avec juste `<p>TODO</p>`. Fiches 06–08 les remplissent.
- [x] Créer `StatsFilterBar.svelte` stub avec juste un placeholder. Fiche 09 l'implémente.

### 5. Mount dans `App.svelte`

- [x] Ajouter l'import de `StatsPanel`.
- [x] Dans la zone de rendu des panneaux, ajouter :
  ```svelte
  {#if $openPanels.has(PANEL.STATS)}
    <StatsPanel />
  {/if}
  ```
- [x] Vérifier que la commande `:stats` ouvre/ferme bien le panneau avec un squelette visible.

### 6. Tests Vitest

- [x] `StatsPanel.test.js` : monter le composant avec un store mocké, vérifier que les 3 tabs sont rendus, que cliquer change `activeTab`, que toggle PR/MWC met à jour `statsMetricStore`.
- [x] Test d'accessibilité minimal : les boutons tabs ont `role="tab"` ou équivalent, focus management au clavier.

## Acceptance criteria

- [x] `:stats` dans la commande ouvre un panneau vide avec onglets et toggle PR/MWC.
- [x] Pas de warning Svelte 5 au démarrage.
- [x] `npm test` vert.
- [x] `npm run lint` clean.
- [x] `npm run build` produit une bundle sans erreur ; taille de `chart.js` visible dans le report (~70 KB gzip).
- [x] UX : respect des §Principes UX 1, 4, 5, 6, 10 du plan.

## Rollback

Revert = `git revert` + `npm install` (retire `chart.js`). Aucune donnée persistée.
