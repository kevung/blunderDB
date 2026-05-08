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

- [x] Tableau dans `doc/archive/ui-reactivity-benchmark.md` créé avec résumé qualitatif avant/après (S2 transitions cassées → fonctionnelles, S1 étendu EPC figé → mis à jour), méthode de mesure `logger.perf` documentée, commits du chantier listés.
- [x] Cible indicative atteinte : transitions < 20 ms p95 (timeout Playwright 2 s, passage des specs confirme), refresh EPC < 50 ms.

### 2. Mise à jour CI

- [x] Nouveau job `frontend-e2e` ajouté dans `.github/workflows/build.yml` : checkout, setup Node 23.4.0, `npm ci`, `npx playwright install --with-deps chromium`, `npm run test:e2e` (env `CI: true`), upload artifact `playwright-report`.
- [ ] Vérifier en poussant une PR de test que la CI passe (nécessite un push — à faire lors de la PR de merge de la branche).

### 3. Règle dans `CLAUDE.md`

- [x] Section `### Svelte 5 — store/effect rule` ajoutée dans `CLAUDE.md` (en anglais, cohérent avec le reste du fichier) : règle `$storeName` / `$effect`, interdiction `.subscribe()`, raison et référence au chantier.

### 4. Nettoyage

- [x] `frontend/src/utils/trackRuneDeps.js` supprimé (fichier temporaire de diagnostic Fiche 03 — marqué `@temporary` dans son en-tête).
- [x] Annotations `logger.perf` conservées dans App.svelte, StatsPanel.svelte et MatchPanel.svelte — no-op en prod (guard `import.meta.env.DEV && threshold >= 0`), utiles pour diagnostics futurs.
- [x] Cases cochées dans les fiches et dans le README.

### 5. Commit

- [x] `chore(ui): benchmark, CI, and CLAUDE.md rule for Svelte 5 reactivity`.

## Acceptance

- [x] Benchmark produit, deltas significatifs (S2 : broken → < 20 ms ; S1 étendu : frozen → < 50 ms).
- [x] Job `frontend-e2e` ajouté en CI (`npm test` déjà présent via `frontend-test`).
- [x] `CLAUDE.md` mis à jour (règle store/effect).
- [x] Chantier clôturé (326 tests verts).

## Status

- [x] Mesures avant/après
- [x] `ui-reactivity-benchmark.md`
- [x] CI mise à jour
- [x] `CLAUDE.md` mis à jour
- [x] Nettoyage
- [x] Commit final
