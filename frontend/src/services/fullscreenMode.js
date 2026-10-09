/**
 * fullscreenMode.js — un mode qui prend toute la fenêtre, fenêtre Wails comprise.
 *
 * La page Direction (directionFullscreen.js) et le théâtre de la Transcription
 * (transcriptionTheatre.js) partagent la même mécanique : le mode ne vit que tant que sa page
 * est disponible (la quitter y met fin), Échap en sort après ce qui est ouvert au-dessus
 * (l'entrée de la pile d'Échap est posée sous eux), et la sortie rend la fenêtre à l'état
 * d'avant : déjà en plein écran système, elle y reste.
 */

import { get, writable } from 'svelte/store';
import { WindowFullscreen, WindowUnfullscreen, WindowIsFullscreen } from '../../wailsjs/runtime/runtime.js';
import { closeOnEscape } from './escapeService.js';

/** Un appel du runtime Wails ; absent en navigateur (tests, e2e) où il n'y a rien à faire. */
async function runtimeCall(/** @type {() => unknown} */ fn) {
    try {
        return await fn();
    } catch {
        return undefined;
    }
}

/**
 * @param {import('svelte/store').Readable<boolean>} available le mode a-t-il où s'afficher ?
 * @param {{ onEnter?: () => void }} [hooks] appelé une fois le mode actif (le focus, par exemple)
 */
export function createFullscreenMode(available, hooks = {}) {
    const active = writable(false);
    /** @type {Array<() => void>} Ce qu'il faut défaire à la sortie. */
    let teardown = [];
    let windowWasFullscreen = false;

    /** Entre dans le mode ; sans effet quand il n'a pas où s'afficher. */
    async function enter() {
        if (get(active) || !get(available)) return;
        active.set(true);
        teardown = [closeOnEscape(exit), available.subscribe((shown) => !shown && exit())];
        hooks.onEnter?.();
        windowWasFullscreen = Boolean(await runtimeCall(WindowIsFullscreen));
        if (!windowWasFullscreen && get(active)) await runtimeCall(WindowFullscreen);
    }

    /** Sort du mode et rend la fenêtre à son état précédent. */
    async function exit() {
        if (!get(active)) return;
        active.set(false);
        const undo = teardown;
        teardown = [];
        for (const fn of undo) fn();
        if (!windowWasFullscreen) await runtimeCall(WindowUnfullscreen);
    }

    function toggle() {
        return get(active) ? exit() : enter();
    }

    return { active, enter, exit, toggle };
}

/**
 * F11 nu, sans modificateur.
 *
 * @param {KeyboardEvent} event
 */
export function isBareF11(event) {
    return event.key === 'F11' && !event.ctrlKey && !event.metaKey && !event.altKey && !event.shiftKey;
}
