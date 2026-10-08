/**
 * Where a match video plays: beside the board (the main area splits, video on the left) or
 * inside the panel that opened it. The panel keeps its player component — the board side only
 * lends it a place in the DOM — so its keys, its Repères and the jump on a cell drive the same
 * player wherever it shows, and switching places neither reloads a file nor loses its instant.
 *
 *   videoPlacementStore  'board' | 'panel', the viewer's choice, remembered across sessions;
 *   videoStageOwnerStore the panel whose video sits beside the board, null when none;
 *   videoStageTargetStore the element the board side offers, null while it is not shown.
 */
import { writable } from 'svelte/store';

const PLACEMENT_KEY = 'blunderdb.video.placement';

function readPlacement() {
    try {
        return localStorage.getItem(PLACEMENT_KEY) === 'panel' ? 'panel' : 'board';
    } catch (_e) {
        return 'board';
    }
}

/** @type {import('svelte/store').Writable<'board' | 'panel'>} */
export const videoPlacementStore = writable(readPlacement());

/** @param {'board' | 'panel'} placement */
export function setVideoPlacement(placement) {
    videoPlacementStore.set(placement);
    try {
        localStorage.setItem(PLACEMENT_KEY, placement);
    } catch (_e) {
        /* storage unavailable: the choice lasts the session */
    }
}

/** @type {import('svelte/store').Writable<string | null>} */
export const videoStageOwnerStore = writable(null);

/** @type {import('svelte/store').Writable<HTMLElement | null>} */
export const videoStageTargetStore = writable(null);

/** @param {string} owner */
export function claimVideoStage(owner) {
    videoStageOwnerStore.set(owner);
}

/**
 * Only the holder lets go: a panel unmounting late must not take the board side from the
 * panel that claimed it since.
 *
 * @param {string} owner
 */
export function releaseVideoStage(owner) {
    videoStageOwnerStore.update((current) => (current === owner ? null : current));
}
