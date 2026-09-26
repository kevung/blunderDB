/**
 * boardArea.js — où finit le plateau. Dans `.scrollable-content`, la molette
 * change de position et Tab ouvre la Recherche, sauf sur la page Direction
 * (ADR-0047), où ils défilent et passent au champ suivant.
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
