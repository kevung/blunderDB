/**
 * directionDrag.js — le glisser-déposer d'une case de la grille des tables, par événements
 * pointeur (l'API HTML5 de glisser-déposer fonctionne mal sous WebKitGTK et interfère avec le
 * dépôt de fichiers de Wails). Pur : les cases arrivent en paramètre, TableGrid.svelte écoute
 * et agit sur ce qu'il rend.
 *
 * Le geste ne décide rien de ce qui est permis : un dépôt sur une table hors service part au
 * service Go, qui le refuse. Ici on ne dit que ce que le geste veut faire — déplacer, échanger,
 * ou rien — et à quelle case il tombe.
 */

/** Le déplacement, en pixels, au-delà duquel un appui devient un glissement et non un clic. */
export const DRAG_THRESHOLD_PX = 6;

/**
 * @param {{ x: number, y: number }} from
 * @param {{ x: number, y: number }} to
 */
export function pastThreshold(from, to) {
    return Math.hypot(to.x - from.x, to.y - from.y) >= DRAG_THRESHOLD_PX;
}

/**
 * Une case peut-elle être saisie : un match y joue (un match sans table aussi, pour qu'on lui
 * en donne une).
 *
 * @param {{ matchId?: string }} cell
 */
export function canGrab(cell) {
    return !!cell.matchId;
}

/**
 * Une case peut-elle recevoir : une table de la salle. Un match sans table n'est pas une
 * destination, et une table partagée héritée d'un ancien journal a deux cases pour un numéro.
 *
 * @param {{ noTable?: boolean, shared?: boolean, table: number }} cell
 */
export function canDrop(cell) {
    return !cell.noTable && !cell.shared && cell.table > 0;
}

/**
 * Ce que veut dire un dépôt de `source` sur `target`.
 *
 * @param {{ matchId?: string, table: number }} source
 * @param {{ matchId?: string, table: number, noTable?: boolean, shared?: boolean } | null} target
 * @returns {{ kind: 'none' } | { kind: 'move', matchId: string, table: number } | { kind: 'swap', matchId: string, table: number, from: number, to: number }}
 */
export function dropAction(source, target) {
    if (!target || !source.matchId || !canDrop(target)) return { kind: 'none' };
    if (target.table === source.table && !!target.matchId && target.matchId === source.matchId) return { kind: 'none' };
    // Un match sans table n'a pas de table à rendre : pas d'échange à confirmer (« Table 0 ↔ N »),
    // le geste part comme un déplacement et le service le refuse en disant pourquoi.
    if (target.matchId && source.table > 0) return { kind: 'swap', matchId: source.matchId, table: target.table, from: source.table, to: target.table };
    return { kind: 'move', matchId: source.matchId, table: target.table };
}
