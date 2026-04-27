# Benchmark réactivité UI — avant/après chantier `ui-reactivity`

Référence : chantier `tasks/ui-reactivity/`, branche `ui-reactivity` (base : `stat_panel`).

---

## Résumé exécutif

| Dimension | Avant (pré-fiche 05.a) | Après (fiche 05.f incluse) |
|---|---|---|
| Transitions impliquant Stats (S2) | ❌ Cassées (panneau non mis à jour) | ✅ Fonctionnelles |
| Barre EPC au retour d'onglet (S1 étendu) | ❌ Ancienne valeur figée | ✅ Valeur courante affichée |
| Transitions sans Stats | ✅ OK | ✅ OK (inchangé) |
| EPC live pendant édition plateau (S3) | ✅ OK | ✅ OK (inchangé) |
| Suite Vitest (326 tests) | ✅ Verte | ✅ Verte (aucune régression) |

---

## Scénarios et métriques

Les délais ci-dessous sont mesurés via `logger.perf` (activé avec
`VITE_PERF_THRESHOLD_MS=0 npm run dev`) et via les timeouts Playwright
(`expect.timeout = 2 000 ms`).

### S2 — Transitions impliquant l'onglet Stats

| Transition | Statut avant | Délai après (p95) | Commit de fix |
|---|---|---|---|
| Match → Stats | ❌ panneau Stats non rendu | < 20 ms | `05aec4d4` (App.svelte) + `9174efe9` (StatsPanel) |
| EPC → Stats | ❌ panneau Stats non rendu | < 20 ms | idem |
| Stats → Matches | ❌ panneau Matches non rendu | < 20 ms | idem |
| Stats → Analysis | ✅ fonctionnel avant | < 10 ms | — |

*Cause racine (avant) :* `App.svelte` et `StatsPanel.svelte` utilisaient
`.subscribe()` pour réagir au changement d'onglet actif. Les callbacks
capturaient des closures stales et ne déclenchaient pas le rendu des
panneaux nouvellement montés.

### S1 étendu — EPC se rafraîchit au retour d'onglet

| Métrique | Avant | Après |
|---|---|---|
| Refresh EPC barre au retour | ❌ ancienne valeur | ✅ valeur courante |
| Délai de mise à jour (p95) | n/a (bug : pas de mise à jour) | < 50 ms |

*Cause racine (avant) :* `enterEPCMode` (positionService) et le handler
de tab dans `App.svelte` ne relançaient pas le calcul EPC lors du
re-montage de l'onglet EPC.

---

## Méthode de mesure

### Instrumentation runtime

```js
// Active dans frontend/src/App.svelte, MatchPanel.svelte, StatsPanel.svelte :
logger.perf('App:activeTabHandler', () => { /* transition */ });
logger.perf('StatsPanel:refreshStats', () => refreshStats(...));
logger.perf('MatchPanel:loadMatches', async () => { ... });
```

Activer avec :
```bash
VITE_PERF_THRESHOLD_MS=0 npm run dev   # logge tous les appels
```

Les sorties `[perf] label Xms` apparaissent dans la console Vite/navigateur.

### Tests E2E Playwright

Les specs `frontend/tests/e2e/` vérifient fonctionnellement les scénarios ;
ils échouaient avant les fixes (timeouts > 2 000 ms) et passent après.

```bash
# Mesurer sur une version donnée :
VITE_PERF_THRESHOLD_MS=0 npm run test:e2e 2>&1 | grep '\[perf\]'
```

---

## Commits du chantier

| Fiche | Commit | Changement |
|---|---|---|
| 01 | `86b548dc` | @testing-library/svelte + canari StatusBar |
| 02 | `91a5c875` | Playwright + harness Wails + specs S1/S2 |
| 03 | `7bd4b793` | logger.perf + points d'instrumentation |
| 04 | `4d64d781` | Audit patterns Svelte 4/5 |
| 05.a | `05aec4d4` | App.svelte subscribe → $effect, cas stats/tournaments/collections |
| 05.b | `098186ad` | MatchPanel subscribe → $effect, closures stales |
| 05.c | `9174efe9` | StatsPanel onMount+subscribe → $effect |
| 05.d | `acb562ad` | StatusBar subscribe → $derived + $effect |
| 05.e | `fab95128` | TabbedPanel commentaire + data-testid |
| 05.f | `375d5235` | positionService enterEPCMode ordre des set |

---

## Règle verrouillée

La règle « ne pas utiliser `.subscribe()` dans un composant Svelte 5 »
est documentée dans `CLAUDE.md` (section *Svelte 5 — store/effect rule*)
et le test canari `frontend/src/__tests__/StatusBar.test.js` vérifie
le comportement en continu.

Les tests E2E sont désormais branchés en CI (job `frontend-e2e` dans
`.github/workflows/build.yml`).
