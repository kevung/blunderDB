import { writable, derived } from 'svelte/store';

// L'onglet Duel (ADR-0072). L'état vit ici car TabbedPanel démonte le panneau à chaque
// changement d'onglet, et la barre d'état montre score et horloges onglet replié.

/**
 * Le Duel ouvert, tel que l'Arbitre le rend (`database.DuelState` : `state`, `sheet`), ou `null`.
 * @type {import('svelte/store').Writable<any>}
 */
export const duelStore = writable(null);

/** Les Duels en suspens (`duel.Summary[]`). */
export const duelListStore = writable(/** @type {any[]} */ ([]));

/** L'heure des horloges, en ms : un battement tant qu'un Duel est ouvert. */
export const duelNowStore = writable(Date.now());

/** Le Bot rejoue ses coups au plateau : rien ne se joue pendant l'animation. */
export const duelAnimatingStore = writable(false);

/**
 * Le Duel tient-il le plateau ? Vrai d'un Duel ouvert et non terminé : la bibliothèque ne se
 * parcourt plus, ni l'édition ni Eval ne s'ouvrent, et le moteur se tait (ADR-0072 règle 9).
 * @type {import('svelte/store').Readable<boolean>}
 */
export const duelHoldsBoardStore = derived(duelStore, ($duel) => !!$duel?.state && !$duel.state.ended);

/** L'onglet Duel est replié (Ctrl+H) : horloges et score restent dans la barre d'état. */
export const duelFoldedStore = writable(false);

// Un Duel qui ne tient plus le plateau rend l'onglet déplié au suivant.
duelHoldsBoardStore.subscribe((holds) => {
    if (!holds) duelFoldedStore.set(false);
});
