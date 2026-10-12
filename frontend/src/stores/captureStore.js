/**
 * captureStore.js — what the last screen capture left in doubt.
 *
 * A capture read onto the scratch board is a proposal: the points and the cube the reading could
 * not settle are ringed there until the user leaves EDIT, so a guess never passes unseen.
 */
import { writable } from 'svelte/store';

/**
 * @typedef {{ field: string, point?: number, reason: string }} CaptureDoubt
 */

/** @type {import('svelte/store').Writable<CaptureDoubt[]>} */
export const captureDoubtsStore = writable([]);
