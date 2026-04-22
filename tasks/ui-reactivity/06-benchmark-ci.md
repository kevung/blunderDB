# 06 — Benchmark avant/après, CI, règle dans CLAUDE.md

**Goal :** Chiffrer l'amélioration (mesures avant/après), brancher les tests en CI, inscrire la règle « subscribe → $effect » dans `CLAUDE.md` pour empêcher la régression.

**Depends on :** 05.a à 05.f terminées.

**Impact :** Clôture du chantier. Rend les gains visibles et verrouillés.

## Context

Les fixes des Fiches 05.* corrigent les bugs mais sans traces persistantes. Il faut :
1. Démontrer le gain (mesures chiffrées).
2. Empêcher la régression future (règle + CI).

## Files touched

- **New:** `doc/archive/ui-reactivity-benchmark.md`.
- **Edit:** `.github/workflows/build.yml` — ajouter exécution de `npm test` (s'il n'y est pas) et `npm run test:e2e`.
- **Edit:** `CLAUDE.md` — ajouter section courte sur la règle subscribe/effect.
- **Edit:** `tasks/ui-reactivity/README.md` — cocher toutes les fiches en status.

## Tasks

### 1. Benchmark

- [ ] Sur commit « avant » (premier commit de la branche `ui-reactivity` avant le fix 05.a), lancer `VITE_PERF_THRESHOLD_MS=0 npm run test:e2e` et collecter les mesures de `logger.perf` (les specs Playwright peuvent les lire via `page.on('console')` et les agréger).
- [ ] Sur commit « après » (dernier commit après 05.f), idem.
- [ ] Tableau dans `doc/archive/ui-reactivity-benchmark.md` :
  ```markdown
  | Scénario | Métrique | Avant | Après | Delta |
  |---|---|---|---|---|
  | S2 — Match → Stats | transition (ms) | X | Y | -Z% |
  | S1 étendu — EPC retour | refresh delay (ms) | X | Y | -Z% |
  ```
- [ ] Cible indicative : transitions < 100 ms p95, refresh EPC < 50 ms.

### 2. Mise à jour CI

- [ ] Dans `.github/workflows/build.yml`, dans le job Linux (ubuntu-latest) :
  ```yaml
  - name: Frontend unit tests
    working-directory: frontend
    run: npm test
  - name: Install Playwright browsers
    working-directory: frontend
    run: npx playwright install --with-deps chromium
  - name: Frontend E2E tests
    working-directory: frontend
    run: npm run test:e2e
    env:
      CI: true
  ```
- [ ] Vérifier en poussant une PR de test que la CI passe.

### 3. Règle dans `CLAUDE.md`

- [ ] Ajouter dans `CLAUDE.md` une section courte (≤ 15 lignes) :
  ```markdown
  ### Svelte 5 — règle store/effect

  Dans ce projet, tout accès à un store depuis un composant Svelte 5 doit se faire via l'auto-subscribe `$storeName` ou via `$effect(() => { const v = $storeName; ... })`. **Ne pas utiliser** `.subscribe()` dans un composant (exceptions rares à justifier en commit). Raison : les callbacks de `.subscribe()` capturent des closures stales et leurs dépendances internes (`$otherStore`, `get(x)`) ne sont pas trackées par le compilateur, ce qui a produit le chantier `tasks/ui-reactivity/` suite à la migration Svelte 5.
  ```

### 4. Nettoyage

- [ ] Retirer `frontend/src/utils/trackRuneDeps.js` (Fiche 03) si c'était temporaire — sauf si utile pour la suite.
- [ ] Vérifier qu'aucune annotation `logger.perf` n'est laissée dans des chemins chauds en prod (elles sont no-op en prod grâce au flag, mais retirer les plus verbeuses).
- [ ] Cocher toutes les cases dans les fiches du chantier et dans le README.

### 5. Commit

- [ ] `chore(ui): benchmark, CI, and CLAUDE.md rule for Svelte 5 reactivity`.

## Acceptance

- [ ] Benchmark produit, deltas significatifs.
- [ ] CI verte sur ubuntu-latest avec `npm test` et `npm run test:e2e`.
- [ ] `CLAUDE.md` mis à jour.
- [ ] Chantier clôturé.

## Status

- [ ] Mesures avant/après
- [ ] `ui-reactivity-benchmark.md`
- [ ] CI mise à jour
- [ ] `CLAUDE.md` mis à jour
- [ ] Nettoyage
- [ ] Commit final
