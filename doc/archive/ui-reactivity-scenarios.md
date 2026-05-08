# UI reactivity — scénarios de reproduction

Scénarios déterministes utilisés comme base pour les tests automatisés (Playwright / @testing-library/svelte) et le diagnostic post-migration Svelte 5. La portée est : TabbedPanel, MatchPanel, StatsPanel, EPCPanel, StatusBar, App.svelte.

## Contexte observé

- Le panneau **Stats** est nouveau (branche `stat_panel`, pas présent sur `main`) → les bugs ne se reproduisent **pas** sur `main` simplement parce que l'onglet qui les déclenche n'existe pas. Les fixes doivent donc partir de `stat_panel`, pas de `main`.
- Le handler `activeTabStore.subscribe()` dans `App.svelte` (~lignes 198-213) ne traite que `search`, `epc`, `matches`. Les onglets `stats`, `tournaments`, `collections`, `anki` n'ont aucune logique cascade → c'est l'hypothèse principale.

## Scénarios

### S1 — Persistance de l'EPC au retour d'onglet (stat_panel)

**Étapes**
1. Charger une DB, sélectionner une position.
2. Onglet **EPC** → noter la valeur affichée dans la barre d'état (observé : `EPC 66.47` sur position par défaut).
3. Bascule onglet **Stats**.
4. Retour onglet **EPC**.

**Observable**
- La valeur EPC affichée au retour doit correspondre à la position courante.

**Résultat observé** : ✅ 66.47 retrouvé au retour (position par défaut, pas de modification entre les deux).

**À confirmer** : refaire le test en modifiant la position (ex: déplacer un checker, ou charger une autre position) **entre** l'étape 3 et 4 pour vérifier que l'EPC reflète bien la nouvelle position et pas une valeur cachée.

---

### S2 — Transitions impliquant Stats (stat_panel) — **BUG CONFIRMÉ**

**Étapes** : effectuer une bascule d'onglets dans les deux sens entre chaque paire.

**Résultats observés**

| Transition | Comportement |
|---|---|
| Match ↔ Stats | ❌ L'onglet change mais le contenu ne s'actualise pas |
| Match ↔ EPC | ✅ OK |
| EPC ↔ Stats | ❌ Transition cassée |
| Stats ↔ Anki | ❌ Transition cassée |

**Pattern identifié** : **toute transition impliquant l'onglet `stats` échoue**. Les autres transitions sans `stats` fonctionnent. Correspond directement à l'absence de cas `stats` dans le handler `activeTabStore` d'App.svelte.

---

### S3 — Mise à jour EPC pendant édition du plateau (stat_panel)

**Étapes**
1. Onglet EPC.
2. Modifier le plateau (ajouter/enlever un checker).

**Observable** : la barre d'état EPC doit se mettre à jour à chaque modification.

**Résultat observé** : ✅ OK.

---

## Stratégie de branche

Puisque S2 prouve que le bug est lié à l'onglet Stats (nouveau), il faut travailler **depuis `stat_panel`** et non depuis `main`. Option retenue : voir feu vert utilisateur (continuer sur `stat_panel` directement **ou** créer `ui-reactivity` depuis `stat_panel`).

## Traduction en tests automatisés (Fiches 1-2)

- **Test S2 (Playwright, prioritaire)** : lancer 5 bascules `Match → Stats → Match → Stats → EPC`. Vérifier à chaque étape que le DOM interne à `TabbedPanel` contient le composant correspondant et pas un résidu de l'onglet précédent. Vérifier que les stores qui alimentent le contenu (statsStore pour Stats, etc.) reflètent l'onglet actif.
- **Test S1 étendu (Playwright)** : charger position A → EPC (noter valeur) → Stats → charger position B via commande → EPC → vérifier que la valeur EPC correspond à la position B.
- **Test S3 (testing-library/svelte)** : monter StatusBar avec `statusBarModeStore='EPC'`, muter `positionStore` et `epcDataStore`, vérifier le DOM.

## Notes

- S1 comme initialement formulé est un faux-positif : la position ne change pas entre les deux visites EPC, donc trouver la même valeur est normal. La version « étendue » (charger position différente entre les deux visites) est le vrai test.
- Les observations ont été faites manuellement le 2026-04-22 par l'utilisateur sur la branche `stat_panel`.
