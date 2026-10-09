# ADR-0085 — Un panneau suit une grammaire unique

Statut : acceptée.
Voir aussi : ADR-0008 (échelle typographique), ADR-0021 (blocs d'analyse à écart constant),
ADR-0048 (ordre des outils par fréquence).

## Contexte

Les treize panneaux du dock bas ont grandi un par un. `style.css` centrait le texte sur `html`
et `#app` : tout élément qui ne disait rien héritait du centre, d'où des titres centrés ou
étirés à côté d'autres à gauche, des colonnes de coups centrées, un « Démarrer » orphelin au
milieu d'une ligne. Chaque panneau plaçait son action principale ailleurs, et un panneau sans
contenu (Analyse sans analyse) restait blanc. Rien ne disait pour qui un panneau est conçu.

## Décision

1. **Trois personas** (`CONTEXT.md`, « Who the interface is for ») : le compétiteur qui
   révise, le joueur qui s'entraîne, le coach. Un panneau sert d'abord l'un d'eux.
2. **Pas de centrage hérité.** La racine ne centre rien ; un bloc centré le dit lui-même.
3. **Une bande d'en-tête par vue** (`panels/PanelHeader.svelte`) :
   `[←] titre · compteur · filtres ⟶ secondaires [primaire]`. Le titre ne s'étire jamais ;
   l'action primaire, une seule, est en bout de bande à droite (`NewButton` si elle crée).
4. **Contenu au début de ligne** ; dans un tableau, texte à gauche, nombres à droite.
5. **Seul l'état vide est centré** (`panels/EmptyState.svelte`), cinq mots au plus ; son
   bouton est l'action propre au panneau (`action`), l'import seulement là où le contenu
   vient d'un import.
6. **Un lanceur est une grille libellé │ contrôle** (`panels/FormGrid.svelte`,
   `panels/FormRow.svelte`) ; le bouton qui lance part dans la bande.
7. **Un compte de positions est un lien** (`panels/CountLink.svelte`) qui ouvre ces positions.
8. **Libellés courts, pas de prose d'aide** dans un panneau : un `placeholder`, un `title` ou
   le mécanisme lui-même.
9. **Jetons d'espacement** (`--space-*`, `--radius`, `--font-size-*`), plus d'`em`.

## Conséquences

- Collections, Commentaires et Analyse suivent la grammaire ; les autres panneaux la
  rejoignent par tranches, sans nouveau composant.
- Retirer le centre de la racine touche aussi ce qui ne se déclarait pas centré hors des
  panneaux : chaque régression se corrige au composant, jamais en le remettant à la racine.
