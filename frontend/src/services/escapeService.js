/**
 * escapeService.js — ce qui est ouvert se ferme sur Échap AVANT tout geste global (#414).
 *
 * ## La règle
 *
 * Un appui d'Échap ferme d'abord la dernière chose ouverte qui a quelque chose à fermer — un
 * menu contextuel, la fiche de résultat, la reprise de la direction —, et rien d'autre ne le
 * voit. Seulement quand rien de tel n'est ouvert, l'Échap continue son chemin : les paliers
 * des panneaux, puis le répartiteur global (sortie de champ, retour d'une sous-recherche #410).
 * Les modales gardent leur propre Échap (Modal.svelte) : une surcouche ouverte SOUS une modale
 * n'est pas fermée à sa place.
 *
 * ## Pourquoi un registre écouté en capture, et pas un `<svelte:window onkeydown>`
 *
 * Ces surcouches écoutaient Échap par `<svelte:window onkeydown>`, et aucune ne le recevait. Le
 * répartiteur (`keyboardService.handleKeyDown`), enregistré sur `window` dans `onMount`
 * d'App.svelte, appelait `event.stopPropagation()`. Sur `window`, en bulle, cela n'arrête aucun
 * écouteur natif — mais Svelte 5 enveloppe chaque gestionnaire déclaratif (`create_event`) et
 * ne l'appelle pas si `event.cancelBubble` est vrai, même sur la même cible. Tout
 * `<svelte:window onkeydown>` monté APRÈS App — un menu qui s'ouvre, la page Direction — était
 * donc muet, pour toutes les touches ; ceux montés avant (ViewTabs) marchaient, par l'ordre de
 * montage seul.
 *
 * Retirer ce `stopPropagation` ne suffisait pas : l'écouteur de la surcouche serait passé APRÈS
 * le répartiteur, qui aurait déjà quitté les résultats d'une sous-recherche, et après les
 * écouteurs `document` des panneaux, qui consomment Échap pour leurs propres paliers. Aucun
 * ordre d'enregistrement ne garantit la priorité ; la phase de capture sur `window`, si : elle
 * passe avant la cible, la délégation de Svelte, les écouteurs `document` et le répartiteur.
 * Un seul écouteur de capture, posé ici au premier enregistrement, consulte une pile : la
 * dernière surcouche ouverte est la première fermée, et un nouveau composant n'a qu'à
 * s'enregistrer — il n'a aucun ordre d'écouteurs à connaître.
 *
 * ## Usage, dans un composant
 *
 *     $effect(() => {
 *         if (open) return closeOnEscape(() => (open = false));
 *     });
 *
 * Le retour de `closeOnEscape` retire l'entrée ; `$effect` l'appelle à la fermeture et au
 * démontage.
 */

/** @typedef {{ close: () => void, modalsBelow: number }} Closable */

/** @type {Closable[]} La dernière ouverte en haut de la pile. */
const stack = [];

let installed = false;

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
 * Une surcouche est-elle ouverte ? Ce qui est ouvert garde aussi ses autres touches : la file des
 * propositions ne les lui prend pas (directionKeys.js, #415).
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
    if (!installed) {
        window.addEventListener('keydown', handleEscapeCapture, true);
        installed = true;
    }
    /** @type {Closable} */
    const entry = { close, modalsBelow: openModalCount() };
    stack.push(entry);
    return () => {
        const i = stack.indexOf(entry);
        if (i >= 0) stack.splice(i, 1);
    };
}
