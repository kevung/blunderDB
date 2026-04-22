# 03 — Instrumentation perf (`logger.perf` + tracker de runes)

**Goal :** Obtenir des mesures chiffrées (`performance.measure`) pour identifier où le temps est passé dans les transitions d'onglet et repérer les `$effect` qui se déclenchent en boucle.

**Depends on :** 01.

**Impact :** Alimente le diagnostic de la Fiche 04 (audit) et le benchmark de la Fiche 06 (avant/après).

## Context

- `frontend/src/utils/logger.js` existe et gère déjà le conditionnel dev/prod via `import.meta.env.DEV`.
- Aucun `performance.mark` ou `performance.measure` dans le code actuel (vérifié par grep).
- L'objectif n'est **pas** une télémétrie permanente mais un outillage **de diagnostic** — activable par flag, désactivé en prod par défaut, pas d'impact perf.

## Files touched

- **Edit:** `frontend/src/utils/logger.js` — ajouter `perf()`.
- **New:** `frontend/src/utils/trackRuneDeps.js` — wrappers optionnels.
- **New:** `frontend/src/__tests__/logger.perf.test.js`.
- **Edit (léger, 5 points max) :** `App.svelte`, `MatchPanel.svelte`, `StatsPanel.svelte`, `StatusBar.svelte`, `TabbedPanel.svelte` — wraps ciblés.

## Tasks

### 1. Étendre `logger.js`

- [ ] Ajouter à `frontend/src/utils/logger.js` :
  ```js
  const PERF_THRESHOLD_MS = Number(import.meta.env.VITE_PERF_THRESHOLD_MS ?? 16);
  const PERF_ENABLED = import.meta.env.DEV && PERF_THRESHOLD_MS >= 0;

  function perf(label, fn) {
      if (!PERF_ENABLED) return fn();
      const mark = `perf-${label}-${Math.random().toString(36).slice(2, 7)}`;
      performance.mark(`${mark}-start`);
      const result = fn();
      const finish = () => {
          performance.mark(`${mark}-end`);
          const measure = performance.measure(label, `${mark}-start`, `${mark}-end`);
          if (measure.duration >= PERF_THRESHOLD_MS) {
              console.log(`[perf] ${label} ${measure.duration.toFixed(2)}ms`);
          }
          performance.clearMarks(`${mark}-start`);
          performance.clearMarks(`${mark}-end`);
          performance.clearMeasures(label);
      };
      if (result && typeof result.then === 'function') {
          return result.finally(finish);
      }
      finish();
      return result;
  }
  ```
- [ ] Exporter `perf` dans l'objet `logger` (ou comme named export).
- [ ] **Ne pas logger** en prod ni si `VITE_PERF_THRESHOLD_MS` est négatif.

### 2. Utilitaire `trackRuneDeps.js` (optionnel)

- [ ] `frontend/src/utils/trackRuneDeps.js` — wrappers légers :
  - `trackedState(label, initial)` — retourne un getter/setter qui logge chaque mutation si `VITE_TRACK_RUNES=1`.
  - `trackedEffect(label, fn)` — wrap de `$effect` qui incrémente un compteur et logge si le compteur > seuil (détecte les boucles).
- [ ] **Implémentation minimale** ; l'objectif est diagnostique, à retirer quand les fixes sont validés. Note dans le fichier qu'il est **temporaire**.

### 3. Points d'instrumentation stratégiques (≤ 8)

- [ ] `App.svelte` : wrapper l'effet de `activeTabStore` (converti en `$effect` en Fiche 05.a) dans `perf('App:activeTabHandler', ...)`.
- [ ] `App.svelte` : wrapper le futur effet EPC dans `perf('App:epcSync', ...)`.
- [ ] `TabbedPanel.svelte` : wrapper l'effet `$effect.pre` de montage dans `perf('TabbedPanel:mountTab', ...)`.
- [ ] `MatchPanel.svelte` : wrapper `loadMatches()` (déjà async) — juste un `perf('MatchPanel:loadMatches', () => loadMatches())` au point d'appel.
- [ ] `StatsPanel.svelte` : wrapper `refreshStats(filter)` dans le futur `$effect` (Fiche 05.c).

### 4. Onglet de debug caché (optionnel)

- [ ] Ajouter dans `commandProcessor.js` une commande `:perf` qui active/désactive `window.__PERF__` et logge un résumé en console. Pas de UI dédiée — le DevTools console suffit. Marquer cette section comme « debug only » dans le code.

### 5. Test

- [ ] `frontend/src/__tests__/logger.perf.test.js` :
  - Test 1 : `perf(label, () => 42)` retourne 42.
  - Test 2 : `perf(label, async () => 42)` retourne une promesse résolue à 42.
  - Test 3 : en mode prod (mocker `import.meta.env.DEV = false`), `console.log` n'est **pas** appelé.
  - Test 4 : en mode dev avec seuil 0, une fonction synchrone de durée > 0 déclenche un log.

### 6. Documentation courte

- [ ] Dans l'entête de `logger.js`, documenter en 5-10 lignes comment activer le seuil à 0 pour voir tous les effets.
- [ ] Commentaire d'entête dans `trackRuneDeps.js` précisant que c'est **temporaire** et à retirer après validation des fixes.

## Acceptance

- [ ] `logger.perf` fonctionne, tests verts.
- [ ] 4-5 points stratégiques instrumentés ; lancer l'app en dev avec `VITE_PERF_THRESHOLD_MS=0` produit un journal lisible.
- [ ] Aucun impact perf en prod (vérifié par test + inspection).

## Status

- [ ] `logger.perf` + tests
- [ ] `trackRuneDeps` (optionnel)
- [ ] Points stratégiques wrappés
- [ ] Commande `:perf` (optionnelle)
- [ ] Doc
