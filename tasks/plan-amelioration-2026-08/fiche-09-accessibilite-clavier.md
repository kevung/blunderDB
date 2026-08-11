# Fiche 09 — Accessibilité et cohérence clavier

Branche : `fix/a11y-clavier`

## Objectif

Rendre la barre d'onglets utilisable au clavier, corriger les rôles ARIA
mensongers, unifier les 4 gardes clavier de panneaux qui ont divergé.

## Constats

- `TabbedPanel.svelte:113-137` : onglets pilotés uniquement par
  `onmousedown` — ni `onclick`, ni `onkeydown`, ni `role="tablist"`.
  Tab + Entrée ne fait rien. (`ConfigModal:273-275` et `stats/StatsPanel:40-47`
  font correctement les rôles.)
- `MatchPanel:712`, `TournamentPanel:572`, `CollectionPanel:609` :
  `role="dialog" aria-modal="true"` sur des panneaux **dockés** (le reste de
  l'app reste actif) → masque l'application aux lecteurs d'écran.
- Divergence des handlers clavier de panneaux (tableau de l'audit) :
  dans `TournamentPanel`/`CollectionPanel`, `?` (aide) et Espace (ligne de
  commande) sont avalés ; dans `MatchPanel`, taper `j` dans un champ
  d'édition déclenche la navigation.
- Mini-dialogue « enregistrer le filtre » (`SearchPanel:1724-1755`) : pas de
  `role="dialog"`, pas de focus trap, pas d'Escape, pas d'autofocus.
- `FileImportProgressModal`/`ImportProgressModal` : Escape inopérant même à
  l'état terminal (bouton Fermer visible).
- Raccourci non documenté : `p` (bascule compte de course en révision Anki,
  `keyboardService.js:102-104`) ; sections raccourcis manquantes pour
  Collections et Tournois dans `doc/source/raccourcis.rst` ; inversion
  `MAJ-J`/`MAJ-K` vs `j`/`k` non documentée.

## Tâches

- [x] `TabbedPanel` : conserver `onmousedown` (drag), ajouter activation
      clavier (Entrée/Espace) + navigation par flèches, `role="tablist"`,
      `role="tab"`, `aria-selected`, `tabindex` roving.
- [x] Remplacer `aria-modal` par `role="region"` + `aria-label` sur les 3
      panneaux dockés.
- [x] `panelKeyGuard(event, {allowNavKeys})` dans `keyboardService.js` :
      centralise « ce qui traverse toujours » (Ctrl, Espace, `?`, saisie dans
      champ éditable) ; migrer les 4 handlers dessus. Tests vitest : pour
      chaque panneau, `?`, Espace, Ctrl et frappe dans un champ éditable.
- [x] Mini-dialogue de SearchPanel : `role="dialog"`, `use:trapFocus`
      (util existant), Escape, autofocus du champ.
- [x] Escape ferme les modales d'import à l'état terminal uniquement.
- [x] Doc `raccourcis.rst` (FR) : sections Collections et Tournois, touche
      `p` Anki, note sur l'inversion MAJ-J/K.

## Notes d'exécution

Ré-audit par grep avant modification (les fiches 07/08 avaient déplacé les
lignes citées dans les constats) : les 3 emplacements `role="dialog"
aria-modal="true"` sur panneaux dockés étaient bien à `MatchPanel.svelte:705`,
`TournamentPanel.svelte:573`, `CollectionPanel.svelte:596` (numéros décalés
mais bugs identiques). Les 2 bugs de divergence clavier confirmés tels que
décrits : `TournamentPanel`/`CollectionPanel` avalaient `?`/Espace (pas de
passthrough du tout), `MatchPanel` n'avait aucun garde-fou de champ éditable.
Le mini-dialogue de `SearchPanel` était bien à la ligne ~1725 (`showSaveDialog`),
inchangé dans sa forme depuis l'audit.

**`panelKeyGuard` migré vers 3 gardes, pas 4.** `SearchPanel.svelte` a
également son propre `document.addEventListener('keydown', …)` (4ᵉ fichier),
mais son `handleKeyDown` (ligne ~979) est structurellement différent : il ne
fait rien (laisse tout traverser) quand la cible n'est pas un champ éditable,
et ne réagit qu'à Entrée quand elle l'est — donc `?`/Espace/Ctrl y
fonctionnent déjà depuis n'importe quel focus non-éditable, et il n'a pas de
raccourci « propre » à un caractère (type `j`) qui pourrait être détourné par
une frappe. Le migrer vers `panelKeyGuard(event)` aurait *changé* son
comportement : `panelKeyGuard` fait passer Ctrl avant même de vérifier le
champ éditable (comme les 3 autres panneaux, déjà ainsi avant cette fiche),
alors que `SearchPanel` bloque aujourd'hui tout, y compris Ctrl, dès que la
cible est éditable — un CTRL-S pendant la frappe d'un nom de filtre
déclencherait `saveCurrentPosition()` au lieu d'être avalé. Comme la consigne
est de ne remapper aucun raccourci existant, ce 4ᵉ handler n'a pas été touché ;
seul son mini-dialogue « enregistrer le filtre » a été rendu accessible
(tâche séparée, voir ci-dessus).

Tests ajoutés (`frontend/src/__tests__/`) :
`TabbedPanel.a11y.test.js` (8 tests, activation Entrée/Espace + flèches +
tabindex roving), `panelKeyboardGuard.test.js` (15 tests — 3 panneaux × 5 cas,
en montant les vrais composants et en vérifiant la propagation
document→window plutôt que la seule fonction `panelKeyGuard`),
`SearchPanel.saveFilterDialog.test.js` (rôle/aria-modal/autofocus/Escape),
`ImportProgressModals.escape.test.js` (Escape inopérant hors état terminal,
y compris le cas `preview` sans rien à importer qui a son propre bouton
Fermer).

Build Sphinx (`doc && python build.py`, 9 langues) lancé pour valider le RST
de `raccourcis.rst` — aucun avertissement propre au fichier.

Non fait, hors périmètre de la fiche : `AnalysisPanel.svelte` a aussi
`role="dialog" aria-modal="true"` sur un panneau docké (même défaut), mais
n'est pas dans la liste des 3 panneaux visés par cette fiche — laissé tel
quel.

## Critères de fin

- Parcours 100 % clavier : changer d'onglet, ouvrir l'aide avec `?` et la
  ligne de commande avec Espace depuis chaque panneau.
- Tests des gardes clavier verts ; vitest global vert.

## Risques & garde-fous

- Ne PAS remapper `j`/`k`/`MAJ-J`/`MAJ-K` (habitudes utilisateurs) : on
  documente, on ne change pas.
- Respecter la convention layout : lettres via `event.key`, chiffres via
  `event.code` (mémoire projet, helper `letter()`).
- `HelpModal` doit garder son `stopImmediatePropagation` (race connue).
