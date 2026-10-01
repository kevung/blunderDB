/**
 * directionGridKeys.js — le clavier de la grille des tables : à quelle case vont les flèches,
 * et quand M / X demandent un changement de table. Pur : les cases arrivent en paramètre,
 * la grille (TableGrid.svelte) agit sur ce qu'il rend.
 *
 * - ← / → : la case voisine dans l'ordre de lecture ; ↑ / ↓ : la case la plus proche, à
 *   l'horizontale, de la rangée voisine (le nombre de colonnes dépend de la largeur).
 * - Début / Fin : première et dernière case.
 * - M et X : sur un match en cours, ouvrent le champ de table de la fiche. Les deux gestes
 *   sont un seul appel : viser une table occupée échange les deux matchs.
 */

/** Le temps laissé pour le second chiffre d'un numéro de table. */
export const TABLE_DIGIT_DELAY_MS = 400;

/**
 * @param {KeyboardEvent} e
 * @param {HTMLElement[]} cells les cases de la grille, dans l'ordre du DOM
 * @param {HTMLElement} current la case qui a le focus
 * @param {boolean} running un match s'y joue
 * @returns {{ focus?: HTMLElement, move?: boolean } | null}
 */
export function gridKeyAction(e, cells, current, running) {
    if (e.ctrlKey || e.metaKey || e.altKey) return null;
    const i = cells.indexOf(current);
    if (i < 0) return null;
    if (!e.shiftKey && (e.key === 'm' || e.key === 'M' || e.key === 'x' || e.key === 'X')) return running ? { move: true } : null;
    if (e.shiftKey) return null;
    switch (e.key) {
        case 'ArrowRight':
            return i + 1 < cells.length ? { focus: cells[i + 1] } : null;
        case 'ArrowLeft':
            return i > 0 ? { focus: cells[i - 1] } : null;
        case 'Home':
            return { focus: cells[0] };
        case 'End':
            return { focus: cells[cells.length - 1] };
        case 'ArrowDown':
        case 'ArrowUp': {
            const r = current.getBoundingClientRect();
            const cx = r.left + r.width / 2;
            const cy = r.top + r.height / 2;
            const down = e.key === 'ArrowDown';
            // La rangée voisine : les cases strictement au-dessus / au-dessous de la case
            // courante, dont le centre est le plus proche verticalement.
            const beyond = cells.filter((c) => {
                const b = c.getBoundingClientRect();
                const y = b.top + b.height / 2;
                return down ? y > r.bottom - 1 : y < r.top + 1;
            });
            if (beyond.length === 0) return null;
            const rowY = (/** @type {HTMLElement} */ c) => {
                const b = c.getBoundingClientRect();
                return b.top + b.height / 2;
            };
            const near = beyond.reduce((m, c) => (Math.abs(rowY(c) - cy) < Math.abs(m - cy) ? rowY(c) : m), rowY(beyond[0]));
            const row = beyond.filter((c) => Math.abs(rowY(c) - near) < 2);
            const cxOf = (/** @type {HTMLElement} */ c) => {
                const b = c.getBoundingClientRect();
                return b.left + b.width / 2;
            };
            return { focus: row.reduce((m, c) => (Math.abs(cxOf(c) - cx) < Math.abs(cxOf(m) - cx) ? c : m), row[0]) };
        }
        default:
            return null;
    }
}
