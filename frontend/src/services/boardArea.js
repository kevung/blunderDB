/**
 * boardArea.js — où finit le plateau.
 *
 * La zone principale (`.scrollable-content`) montre le plateau, ou la page Direction qui le
 * remplace quand un tournoi est dirigé (ADR-0047). Deux gestes y ont un sens de plateau : la
 * molette change de position, Tab ouvre la Recherche (#204). Sur la page Direction, ce sont une
 * page et ses champs : la molette y défile et Tab y passe au champ suivant. Les confondre
 * faisait défiler la bibliothèque cachée derrière la grille des tables et quitter la Direction
 * au milieu d'une saisie de score (simulation 2026-09, #434, #435).
 */

/** Ce qui, dans la zone principale, n'est pas le plateau. */
const NOT_THE_BOARD = '.direction-view';

/**
 * @param {Element | null | undefined} el
 * @returns {boolean} vrai si `el` est dans la zone du plateau et pas sur une page qui le remplace
 */
export function isOnBoard(el) {
    if (!el || typeof el.closest !== 'function') return false;
    return !!el.closest('.scrollable-content') && !el.closest(NOT_THE_BOARD);
}
