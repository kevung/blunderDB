/**
 * directionKeys.js — à qui vont J / K / ↓ / ↑ / ENTRÉE quand la page Direction est affichée (#415).
 *
 * ## La règle
 *
 * Tant que la page Direction remplace le plateau (onglet Tournois actif ET une Direction
 * ouverte, App.svelte), ces touches nues appartiennent à la page : la file des propositions
 * (`components/direction/ProposalList.svelte`) les prend — J / ↓ proposition suivante, K / ↑
 * précédente, ENTRÉE confirme la proposition choisie. Le panneau Tournois, visible en même
 * temps, les laisse passer ; il ne les reprend (J / K = tournoi suivant / précédent) que
 * lorsque le focus est DANS le panneau — un clic dans sa liste. Jamais dans un champ de saisie.
 * Échap n'en fait pas partie : il suit la règle d'`escapeService` (#414).
 *
 * ## Le mécanisme : une règle, trois lecteurs
 *
 * - La file écoute en phase de CAPTURE sur `window`, comme `escapeService` : elle passe avant
 *   l'écouteur `document` du panneau Tournois et avant le répartiteur global, quel que soit
 *   l'ordre de montage — la page s'ouvre après eux. Un `<svelte:window onkeydown>` ne suffisait
 *   pas : le panneau arrêtait toute touche nue (`stopPropagation`), et Svelte 5 n'appelle pas
 *   un gestionnaire déclaratif quand `event.cancelBubble` est vrai. Quand la file agit, elle
 *   consomme l'appui (`preventDefault` + `stopImmediatePropagation`).
 * - Quand elle n'agit pas — un autre onglet de la Direction, une modale ou une surcouche
 *   ouverte par-dessus, la confirmation de « Tout lancer » —, l'appui continue son chemin.
 *   C'est pourquoi le panneau Tournois et le répartiteur lisent eux aussi `directionOwnsKey` :
 *   le panneau pour ne pas changer de tournoi sous la page, le répartiteur pour ne pas
 *   parcourir le plateau qu'elle cache.
 *
 * La page prend le focus en s'ouvrant (DirectionView), et la prise de focus différée du panneau
 * Tournois s'abstient tant qu'elle est affichée : sans quoi le clic sur « Ouvrir la direction »,
 * qui est dans le panneau, y laisserait le clavier.
 */

import { get } from 'svelte/store';
import { directionPageShownStore } from '../stores/directionStore.js';
import { hasOpenOverlay } from './escapeService.js';
import { isTypingTarget } from '../utils/panelFocus.js';
import { isBareLetter } from '../utils/keys.js';

/** La page Direction remplace-t-elle le plateau en ce moment ? */
export function directionPageShown() {
    return get(directionPageShownStore);
}

/**
 * La règle : l'appui appartient-il à la page Direction ?
 *
 * @param {KeyboardEvent} event
 * @returns {boolean}
 */
export function directionOwnsKey(event) {
    const bare = !event.ctrlKey && !event.metaKey && !event.altKey && !event.shiftKey;
    const pageKey = isBareLetter(event, 'j') || isBareLetter(event, 'k') || (bare && ['ArrowDown', 'ArrowUp', 'Enter'].includes(event.key));
    if (!pageKey || !directionPageShown()) return false;
    const active = document.activeElement;
    if (active?.closest('.tournament-panel')) return false;
    return !isTypingTarget(active) && !isTypingTarget(/** @type {Element | null} */ (event.target));
}

/** Ce qui est ouvert par-dessus la page garde ses touches : une modale, une surcouche (#414). */
export function somethingOpenAbove() {
    return hasOpenOverlay() || document.querySelector('[aria-modal="true"]') !== null;
}
