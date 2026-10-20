// What the other open libraries know of the board on screen (the « Ailleurs » view, ADR-0101):
// the last answer of ElsewhereInGroups, the result pinned for the in-place swap, and the swap
// itself while it shows. All three name rows of other groups by their ids, so a group switch
// empties them (libraryScope.js: reloaded).
import { writable } from 'svelte/store';

/**
 * @typedef {{ group: number, path: string, positionId: number, analysis?: any, metName?: string, otherMet: boolean,
 *   comments: any[], plays: any[], matchesTotal: number, playsTruncated: boolean,
 *   neighbours: { zobrist: number, distance: number, position: any }[] }} ElsewhereGroupReport
 * @typedef {{ zobrist: number, judge: { settings: any, metDigest: string }, groups: ElsewhereGroupReport[] }} ElsewhereView
 * @typedef {{ key: string, view: ElsewhereView | null, loading: boolean, error: string }} ElsewhereState
 * @typedef {{ group: number, path: string, positionId: number, position: any, distance: number }} ElsewherePin
 *   a board of another group the swap shows instead of the same board there
 * @typedef {{ group: number, path: string, position: any, report: ElsewhereGroupReport | null, pinned: boolean }} ElsewherePeek
 */

/** @returns {ElsewhereState} */
export const idleElsewhere = () => ({ key: '', view: null, loading: false, error: '' });

export const elsewhereStore = writable(idleElsewhere());
export const elsewherePinStore = writable(/** @type {ElsewherePin | null} */ (null));
export const elsewherePeekStore = writable(/** @type {ElsewherePeek | null} */ (null));
