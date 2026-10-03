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
 * plein écran système, elle y reste.
 */

import { get, writable } from 'svelte/store';
import { WindowFullscreen, WindowUnfullscreen, WindowIsFullscreen } from '../../wailsjs/runtime/runtime.js';
import { directionPageShownStore } from '../stores/directionStore.js';
import { closeOnEscape } from './escapeService.js';
import { directionPageShown, somethingOpenAbove } from './directionKeys.js';

/** Le plein écran de la Direction est-il actif ? */
export const directionFullscreenStore = writable(false);

/** @type {Array<() => void>} Ce qu'il faut défaire à la sortie. */
let teardown = [];
let windowWasFullscreen = false;

/** Un appel du runtime Wails ; absent en navigateur (tests, e2e) où il n'y a rien à faire. */
async function runtimeCall(/** @type {() => unknown} */ fn) {
    try {
        return await fn();
    } catch {
        return undefined;
    }
}

/** Entre en plein écran ; sans effet hors de la page Direction. */
export async function enterDirectionFullscreen() {
    if (get(directionFullscreenStore) || !get(directionPageShownStore)) return;
    directionFullscreenStore.set(true);
    teardown = [closeOnEscape(exitDirectionFullscreen), directionPageShownStore.subscribe((shown) => !shown && exitDirectionFullscreen())];
    windowWasFullscreen = Boolean(await runtimeCall(WindowIsFullscreen));
    if (!windowWasFullscreen && get(directionFullscreenStore)) await runtimeCall(WindowFullscreen);
}

/** Sort du plein écran et rend la fenêtre à son état précédent. */
export async function exitDirectionFullscreen() {
    if (!get(directionFullscreenStore)) return;
    directionFullscreenStore.set(false);
    const undo = teardown;
    teardown = [];
    for (const fn of undo) fn();
    if (!windowWasFullscreen) await runtimeCall(WindowUnfullscreen);
}

export function toggleDirectionFullscreen() {
    return get(directionFullscreenStore) ? exitDirectionFullscreen() : enterDirectionFullscreen();
}

/**
 * F11 nu bascule le plein écran, sur la page Direction seulement (et pour en sortir, où que
 * l'on soit puisque la sortie suit la page). Ce qui est ouvert au-dessus garde ses touches.
 *
 * @param {KeyboardEvent} event
 * @returns {boolean}
 */
export function directionFullscreenKey(event) {
    if (event.key !== 'F11' || event.ctrlKey || event.metaKey || event.altKey || event.shiftKey) return false;
    // Le plein écran a lui-même une entrée dans la pile d'Échap : elle ne compte pas comme une surcouche.
    const active = get(directionFullscreenStore);
    if (active ? document.querySelector('[aria-modal="true"], .context-menu') : somethingOpenAbove()) return false;
    return active || directionPageShown();
}
