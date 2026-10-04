/**
 * keepFocus.js — empêcher le focus de retomber sur <body> quand l'élément qui le portait disparaît.
 *
 * Un bouton « Lancer » cliqué, une fiche refermée après un résultat : l'élément focalisé quitte le
 * DOM, le navigateur ramène le focus sur <body>, et le Tab suivant repart du haut de la page (ou
 * ouvre la Recherche). L'action observe le nœud : si le focus était dedans et qu'il a atterri sur
 * <body> après une mutation, il va à l'élément que `fallback` désigne.
 *
 * @param {HTMLElement} node
 * @param {() => (HTMLElement | null | undefined)} fallback
 */
export function keepFocus(node, fallback) {
    let inside = false;
    /** @param {FocusEvent} e */
    const onFocusIn = (e) => {
        inside = node.contains(/** @type {Node | null} */ (e.target));
    };
    // A click on something unfocusable drops the focus to <body> on purpose: only a later focusin
    // (the clicked button taking it) marks the node as the focus holder again.
    const onPointerDown = () => (inside = false);
    document.addEventListener('pointerdown', onPointerDown, true);
    document.addEventListener('focusin', onFocusIn);
    const observer = new MutationObserver(() => {
        const active = document.activeElement;
        if (!inside || (active && active !== document.body)) return;
        const target = fallback();
        if (!target || !target.isConnected) return;
        target.focus({ preventScroll: true });
    });
    observer.observe(node, { childList: true, subtree: true });
    return {
        destroy() {
            document.removeEventListener('pointerdown', onPointerDown, true);
            document.removeEventListener('focusin', onFocusIn);
            observer.disconnect();
        }
    };
}
