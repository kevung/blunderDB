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

- [ ] `TabbedPanel` : conserver `onmousedown` (drag), ajouter activation
      clavier (Entrée/Espace) + navigation par flèches, `role="tablist"`,
      `role="tab"`, `aria-selected`, `tabindex` roving.
- [ ] Remplacer `aria-modal` par `role="region"` + `aria-label` sur les 3
      panneaux dockés.
- [ ] `panelKeyGuard(event, {allowNavKeys})` dans `keyboardService.js` :
      centralise « ce qui traverse toujours » (Ctrl, Espace, `?`, saisie dans
      champ éditable) ; migrer les 4 handlers dessus. Tests vitest : pour
      chaque panneau, `?`, Espace, Ctrl et frappe dans un champ éditable.
- [ ] Mini-dialogue de SearchPanel : `role="dialog"`, `use:trapFocus`
      (util existant), Escape, autofocus du champ.
- [ ] Escape ferme les modales d'import à l'état terminal uniquement.
- [ ] Doc `raccourcis.rst` (FR) : sections Collections et Tournois, touche
      `p` Anki, note sur l'inversion MAJ-J/K.

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
