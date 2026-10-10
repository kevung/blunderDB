/**
 * transcriptionTheatre.js — le mode théâtre de la Transcription.
 *
 * Pendant la transcription d'une vidéo, la vidéo prend toute la fenêtre (fenêtre Wails en
 * plein écran comprise, fullscreenMode.js) et un mini-plateau flottant montre la position et
 * les flèches du candidat sélectionné : on suit le match sans quitter le clavier de saisie,
 * qui reste celui du panneau.
 *
 * Le panneau dit quand le théâtre a de quoi s'afficher (`theatreAvailableStore` : onglet
 * Transcription, brouillon ouvert, vidéo attachée) ; la perte de l'une de ces conditions y met
 * fin. La couche du théâtre (TranscriptionTheatre.svelte) offre au lecteur une place
 * (`theatreTargetStore`) où le dock du panneau se déplace, comme il le fait à côté du plateau.
 */

import { get, writable } from 'svelte/store';
import { createFullscreenMode, isBareF11 } from './fullscreenMode.js';

/** Le théâtre peut-il s'ouvrir ? Tenu par le panneau de transcription. */
export const theatreAvailableStore = writable(false);

/** @type {import('svelte/store').Writable<HTMLElement | null>} La place offerte au lecteur. */
export const theatreTargetStore = writable(null);

/** @type {import('svelte/store').Writable<(() => void) | null>} Le geste de rotation que tient le panneau, offert à la barre du théâtre. */
export const theatreRotateStore = writable(null);

const mode = createFullscreenMode(theatreAvailableStore);

/** Le théâtre est-il ouvert ? */
export const theatreStore = mode.active;
export const enterTheatre = mode.enter;
export const exitTheatre = mode.exit;
export const toggleTheatre = mode.toggle;

/**
 * L'élément est-il dans la couche du théâtre ? Le panneau y garde ses touches comme chez lui.
 *
 * @param {Element | null} element
 */
export function theatreHolds(element) {
    return !!element && !!document.querySelector('[data-testid="transcription-theatre"]')?.contains(element);
}

/**
 * F11 nu bascule le théâtre quand il peut s'ouvrir, et en sort toujours. Une modale ou un menu
 * ouverts gardent leurs touches.
 *
 * @param {KeyboardEvent} event
 * @returns {boolean}
 */
export function theatreKey(event) {
    if (!isBareF11(event)) return false;
    if (document.querySelector('[aria-modal="true"], .context-menu')) return false;
    return get(theatreStore) || get(theatreAvailableStore);
}
