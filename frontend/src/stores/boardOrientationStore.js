import { writable } from 'svelte/store';

/**
 * Side of the bearoff trays on the board ('left' | 'right'), changed by Ctrl+Left / Ctrl+Right.
 * One source for every board drawn on screen, so the theatre's mini board follows the main one.
 * @type {import('svelte/store').Writable<'left' | 'right'>}
 */
export const boardOrientationStore = writable('right');
