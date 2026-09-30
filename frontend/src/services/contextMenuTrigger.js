/**
 * contextMenuTrigger.js — l'ouverture d'un menu contextuel sur un objet, à la souris comme au
 * clavier : clic droit, touche Menu ou Maj+F10 sur l'objet focalisé. Aucun autre service ne
 * capte ces touches ; le menu s'ouvre sur l'objet, jamais au centre de la fenêtre.
 */

/**
 * @typedef {{ label: string, onClick: () => void, shortcut?: string, disabled?: boolean }} MenuItem
 * @typedef {{ x: number, y: number, items: MenuItem[] }} MenuRequest
 */

/**
 * La touche Menu (`ContextMenu`) ou Maj+F10.
 *
 * @param {KeyboardEvent} e
 * @returns {boolean}
 */
export function isMenuKey(e) {
    if (e.ctrlKey || e.metaKey || e.altKey) return false;
    return e.key === 'ContextMenu' || (e.key === 'F10' && e.shiftKey);
}

/**
 * Où ancrer le menu : sous le pointeur pour un vrai clic droit, sinon au coin bas-gauche de
 * l'objet (le `contextmenu` que produit le clavier n'a pas de coordonnées fiables).
 *
 * @param {MouseEvent | KeyboardEvent} e
 * @param {Element} el
 * @returns {{ x: number, y: number }}
 */
export function menuPoint(e, el) {
    if (e instanceof MouseEvent && e.button === 2 && (e.clientX || e.clientY)) return { x: e.clientX, y: e.clientY };
    const r = el.getBoundingClientRect();
    return { x: r.left, y: r.bottom };
}

/**
 * Transforme un appui en demande de menu, ou rend `null` si l'événement n'en est pas une.
 * `items` est évalué à la demande : un objet sans action n'ouvre rien.
 *
 * @param {MouseEvent | KeyboardEvent} e
 * @param {() => MenuItem[]} items
 * @returns {MenuRequest | null}
 */
export function menuRequest(e, items) {
    if (e.type === 'keydown' && !isMenuKey(/** @type {KeyboardEvent} */ (e))) return null;
    const list = items();
    if (list.length === 0) return null;
    e.preventDefault();
    e.stopPropagation();
    const el = /** @type {Element} */ (e.currentTarget);
    return { ...menuPoint(e, el), items: list };
}
