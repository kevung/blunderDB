/**
 * directionKeys.js — à qui vont J / K / ↓ / ↑ / ENTRÉE quand la page Direction est affichée.
 *
 * Tant qu'elle remplace le plateau (onglet Tournois actif ET Direction ouverte), ces touches
 * nues vont à la file des propositions (`ProposalList.svelte`) : J / ↓ suivante, K / ↑
 * précédente, ENTRÉE confirme. Le panneau Tournois ne les reprend que si le focus est dans le
 * panneau ; jamais dans un champ de saisie. Échap suit `escapeService`.
 *
 * ENTRÉE active d'abord le bouton, lien ou onglet focalisé (`INTERACTIVE`) et ne confirme que
 * si le focus est sur la page ou la file : arrivé au clavier sur « Tout lancer », on ne doit
 * pas lancer un match non demandé (seule la reprise le défait). J / K / ↓ / ↑ placent le
 * focus sur la file pour que « clic sur l'onglet, J, ENTRÉE » confirme.
 *
 * Mécanisme — une règle, trois lecteurs :
 * - la file écoute en CAPTURE sur `window` (comme `escapeService`), avant le panneau Tournois
 *   et le répartiteur quel que soit l'ordre de montage ; quand elle agit, elle consomme
 *   l'appui (`preventDefault` + `stopImmediatePropagation`) ;
 * - quand elle n'agit pas (autre onglet, modale ou surcouche, confirmation de « Tout
 *   lancer »), le panneau Tournois et le répartiteur lisent `directionOwnsKey` pour ne pas
 *   changer de tournoi ni parcourir le plateau caché.
 *
 * La page prend le focus à l'ouverture et la prise de focus différée du panneau Tournois
 * s'abstient tant qu'elle est affichée.
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

/** Ce qui fait son propre geste sur ENTRÉE. La file elle-même (un conteneur) n'en est pas. */
const INTERACTIVE = 'button, a[href], [role="button"], [role="tab"], summary, input, select, textarea, [contenteditable]';

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
    if (event.key === 'Enter' && active?.matches(INTERACTIVE)) return false;
    return !isTypingTarget(active) && !isTypingTarget(/** @type {Element | null} */ (event.target));
}

/** Ce qui est ouvert par-dessus la page garde ses touches : une modale, une surcouche. */
export function somethingOpenAbove() {
    return hasOpenOverlay() || document.querySelector('[aria-modal="true"]') !== null;
}
