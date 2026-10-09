/**
 * directionFullscreen.js — la page Direction en plein écran dédié.
 *
 * Sur un portable de salle (1366 px), le plateau, les barres et les panneaux ne servent à
 * rien pendant la conduite d'un tournoi : le mode plein écran les masque (App.svelte lit
 * `directionFullscreenStore`) et rend toute la fenêtre à la direction, fenêtre Wails
 * comprise. F11 entre et sort ; Échap sort, après ce qui est ouvert au-dessus (menu,
 * recherche) puisque l'entrée de la pile d'Échap est posée sous eux.
 *
 * Le mode ne vit que tant que la page Direction est affichée : la quitter (autre onglet de
 * l'application, direction fermée) y met fin. Les onglets internes de la direction ne la
 * touchent pas. À la sortie, la fenêtre revient à l'état d'avant : si elle était déjà en
 * plein écran système, elle y reste (fullscreenMode.js).
 */

import { get } from 'svelte/store';
import { directionPageShownStore } from '../stores/directionStore.js';
import { createFullscreenMode, isBareF11 } from './fullscreenMode.js';
import { directionPageShown, somethingOpenAbove } from './directionKeys.js';

const mode = createFullscreenMode(directionPageShownStore);

/** Le plein écran de la Direction est-il actif ? */
export const directionFullscreenStore = mode.active;

/** Entre en plein écran ; sans effet hors de la page Direction. */
export const enterDirectionFullscreen = mode.enter;

/** Sort du plein écran et rend la fenêtre à son état précédent. */
export const exitDirectionFullscreen = mode.exit;

export const toggleDirectionFullscreen = mode.toggle;

/**
 * F11 nu bascule le plein écran, sur la page Direction seulement (et pour en sortir, où que
 * l'on soit puisque la sortie suit la page). Ce qui est ouvert au-dessus garde ses touches.
 *
 * @param {KeyboardEvent} event
 * @returns {boolean}
 */
export function directionFullscreenKey(event) {
    if (!isBareF11(event)) return false;
    // Le plein écran a lui-même une entrée dans la pile d'Échap : elle ne compte pas comme une surcouche.
    const active = get(directionFullscreenStore);
    if (active ? document.querySelector('[aria-modal="true"], .context-menu') : somethingOpenAbove()) return false;
    return active || directionPageShown();
}
