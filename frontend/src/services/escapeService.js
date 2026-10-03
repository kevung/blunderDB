/**
 * escapeService.js — ce qui est ouvert se ferme sur Échap AVANT tout geste global.
 *
 * Échap ferme d'abord la dernière surcouche ouverte (menu contextuel, fiche de résultat,
 * reprise de la direction), et rien d'autre ne le voit. Sinon il continue : paliers des
 * panneaux, puis répartiteur global. Les modales gardent leur Échap (Modal.svelte) ; une
 * surcouche SOUS une modale n'est pas fermée à sa place.
 *
 * Pourquoi la capture : le palier OVERLAY de keyDispatch.js passe avant tout, avant même
 * l'élément qui a le focus, alors qu'un `<svelte:window onkeydown>` passerait après les
 * panneaux qui consomment Échap, et que Svelte 5 le saute dès que `cancelBubble` est vrai.
 * Un seul gestionnaire consulte une pile (dernière ouverte, première fermée) : un composant
 * n'a qu'à s'enregistrer.
 *
 * Usage :
 *
 *     $effect(() => {
 *         if (open) return closeOnEscape(() => (open = false));
 *     });
 *
 * Le retour de `closeOnEscape` retire l'entrée ; `$effect` l'appelle à la fermeture et au
 * démontage.
 */

/** @typedef {{ close: () => void, modalsBelow: number }} Closable */

import { registerKeys } from './keyDispatch.js';

/** @type {Closable[]} La dernière ouverte en haut de la pile. */
const stack = [];

/** @type {(() => void) | null} */
let unregister = null;

function openModalCount() {
    return document.querySelectorAll('[aria-modal="true"]').length;
}

/**
 * L'écouteur de capture. Exporté pour les tests qui l'appellent sans `window`.
 *
 * @param {KeyboardEvent} event
 */
export function handleEscapeCapture(event) {
    if (event.key !== 'Escape' || stack.length === 0) return;
    const top = stack[stack.length - 1];
    // Une modale ouverte par-dessus la surcouche garde son Échap (Modal.svelte).
    if (openModalCount() > top.modalsBelow) return;
    event.preventDefault();
    event.stopImmediatePropagation();
    top.close();
}

/**
 * Une surcouche est-elle ouverte ? Elle garde alors aussi ses autres touches (directionKeys.js).
 *
 * @returns {boolean}
 */
export function hasOpenOverlay() {
    return stack.length > 0;
}

/**
 * Enregistre une chose ouverte que Échap ferme, par-dessus les précédentes.
 *
 * @param {() => void} close
 * @returns {() => void} retire l'entrée (à la fermeture, au démontage)
 */
export function closeOnEscape(close) {
    if (stack.length === 0) unregister = registerKeys('overlay', handleEscapeCapture);
    /** @type {Closable} */
    const entry = { close, modalsBelow: openModalCount() };
    stack.push(entry);
    return () => {
        const i = stack.indexOf(entry);
        if (i >= 0) stack.splice(i, 1);
        if (stack.length === 0 && unregister) {
            unregister();
            unregister = null;
        }
    };
}
