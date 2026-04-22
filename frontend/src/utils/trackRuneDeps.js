/**
 * @file trackRuneDeps.js
 * @temporary Utilitaire de diagnostic Svelte 5 — à retirer après validation des fixes
 * (fiches 05.a–05.f). Ne pas importer en prod.
 *
 * Activation : VITE_TRACK_RUNES=1 npm run dev
 *
 * Usage typique dans un composant .svelte :
 *   import { trackedState, trackedEffect } from '../utils/trackRuneDeps.js';
 *
 *   const count = trackedState('count', 0);
 *   count.set(1);      // logge la mutation
 *   count.get();       // lit la valeur
 *
 *   $effect(() => trackedEffect('myEffect', () => { ... }));
 */

/* eslint-disable no-console */

const TRACK_RUNES = import.meta.env.VITE_TRACK_RUNES === '1';

/** Seuil au-delà duquel on avertit d'une possible boucle d'effet. */
const EFFECT_LOOP_THRESHOLD = 10;

/** @type {Record<string, number>} */
const _effectCounts = {};

/**
 * Crée un accesseur/mutateur qui logge chaque mutation si VITE_TRACK_RUNES=1.
 *
 * @template T
 * @param {string} label
 * @param {T} initial
 * @returns {{ get: () => T, set: (val: T) => void }}
 */
export function trackedState(label, initial) {
    let value = initial;
    return {
        get: () => value,
        set: (newVal) => {
            if (TRACK_RUNES) {
                console.debug(`[rune:${label}] set`, newVal);
            }
            value = newVal;
        },
    };
}

/**
 * Wrapper léger à appeler À L'INTÉRIEUR d'un `$effect` Svelte 5.
 * Incrémente un compteur et avertit si le nombre de déclenchements
 * dépasse le seuil (détection de boucles réactives).
 *
 * Exemple dans un composant .svelte :
 *   $effect(() => trackedEffect('statsPanelRefresh', () => {
 *       refreshStats($statsFilterStore);
 *   }));
 *
 * @param {string} label
 * @param {() => void} fn
 * @returns {void}
 */
export function trackedEffect(label, fn) {
    _effectCounts[label] = (_effectCounts[label] ?? 0) + 1;
    if (TRACK_RUNES && _effectCounts[label] > EFFECT_LOOP_THRESHOLD) {
        console.warn(`[rune:${label}] possible loop — run #${_effectCounts[label]}`);
    }
    return fn();
}
