// The split screen (ADR-0101 §8). Only the board area is cut in two: each pane holds a library
// group on a fixed side and never changes side. The active group's pane is live; the other is
// frozen (its board where the group was left, and its analysis) and dimmed. Clicking it, or
// Ctrl+Alt+←/→, activates its group through activateGroup and nothing else moves. Toolbar, tabs,
// panels and the one status bar always speak for the active group. Below SPLIT_MIN_WIDTH the
// split is refused; under SPLIT_PANEL_BESIDE_WIDTH the panel goes under the boards.
import { get } from 'svelte/store';
import { splitStore, splitShakeStore, splitFrozenStore } from '../stores/splitStore.js';
import { statusBarTextStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { groupIdsStore, activeGroupId, parkedGroupsStore, activateGroup } from '../stores/libraryGroupStore.js';
import { positionStore, matchContextStore, positionsStore } from '../stores/positionStore.js';
import { tMsg } from '../i18n';

/** Narrower than this (CSS pixels), two boards and the panel do not fit. */
export const SPLIT_MIN_WIDTH = 1280;
/** From this width (CSS pixels) the panel stays beside the boards; below, it goes under them. */
export const SPLIT_PANEL_BESIDE_WIDTH = 1600;

/** @param {number} width */
export const splitPanelBeside = (width) => width >= SPLIT_PANEL_BESIDE_WIDTH;

const REMEMBERED = 'blunderdb.split';

// Until the last session's split is put back (or found absent), the split off at startup is not
// a choice of the user and must not overwrite the one remembered.
let remembering = false;

/** Whether changes of the split are kept for the next session yet. */
export const splitRemembering = () => remembering;

/** Startup is past the point where the last split could be put back. */
export function markSplitRestored() {
    remembering = true;
}

/**
 * The board area changed shape without the window doing so: what sizes itself on the window's
 * resize (the board, the panels) measures again once the new layout is in place.
 */
export function relayout() {
    requestAnimationFrame(() => window.dispatchEvent(new Event('resize')));
}

/** The window's width in CSS pixels, the interface scale taken out. */
export function windowWidth() {
    const scale = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ui-scale')) || 1;
    return window.innerWidth / scale;
}

/**
 * The groups the two panes hold. A side whose group closed takes the first other group, else
 * the one left; with no group named yet, the active one goes left.
 * @param {import('../stores/splitStore.js').SplitState} state
 * @param {number[]} ids
 * @param {number} active
 * @returns {{ left: number, right: number }}
 */
export function paneGroups(state, ids, active) {
    let left = state.left != null && ids.includes(state.left) ? state.left : null;
    let right = state.right != null && ids.includes(state.right) ? state.right : null;
    if (left == null && right == null) {
        left = active;
        right = ids.find((id) => id !== active) ?? active;
    } else if (left == null) {
        left = ids.find((id) => id !== right) ?? /** @type {number} */ (right);
    } else if (right == null) {
        right = ids.find((id) => id !== left) ?? left;
    }
    // The active group is always on screen: a group neither side holds takes the right side.
    if (active !== left && active !== right) right = active;
    return { left, right: /** @type {number} */ (right) };
}

/**
 * The side whose pane is live.
 * @param {import('../stores/splitStore.js').SplitState} state
 * @param {{ left: number, right: number }} groups
 * @param {number} active
 * @returns {'left' | 'right'}
 */
export function liveSide(state, groups, active) {
    if (groups.left === groups.right) return state.sameSide;
    return groups.left === active ? 'left' : 'right';
}

/**
 * The active group moved from `from` to `to`. A group a pane holds leaves both sides as they
 * are; a group neither holds (activated from its tab, or just opened) takes the live side.
 * @param {import('../stores/splitStore.js').SplitState} state
 * @param {number} from
 * @param {number} to
 * @param {number[]} ids
 * @returns {import('../stores/splitStore.js').SplitState}
 */
export function followActivation(state, from, to, ids) {
    if (!state.on || from === to) return state;
    const groups = paneGroups(state, ids.includes(from) ? ids : [...ids, from], from);
    if (to === groups.left || to === groups.right) {
        if (state.left === groups.left && state.right === groups.right) return state;
        return { ...state, ...groups };
    }
    const side = liveSide(state, groups, from);
    return { ...state, ...groups, [side]: to };
}

/**
 * Turn the split on or off. Too narrow a window refuses it and says why. The active group takes
 * the left side, the first other group the right one (the same library when it is alone).
 * @param {number} [width]
 * @returns {boolean} whether the split is on afterwards
 */
export function toggleSplit(width = windowWidth()) {
    const now = get(splitStore);
    if (now.on) {
        splitStore.set({ ...now, on: false });
        relayout();
        return false;
    }
    if (width < SPLIT_MIN_WIDTH) {
        statusBarTextStore.set(tMsg('split.tooNarrow', { min: SPLIT_MIN_WIDTH }));
        return false;
    }
    const active = get(activeGroupId);
    const groups = paneGroups({ on: true, left: null, right: null, sameSide: 'left' }, get(groupIdsStore), active);
    splitStore.set({ on: true, ...groups, sameSide: 'left' });
    relayout();
    return true;
}

/** The window shrank under the minimum: the split closes and says why. */
export function closeSplitIfNarrow(width = windowWidth()) {
    const now = get(splitStore);
    if (!now.on || width >= SPLIT_MIN_WIDTH) return;
    splitStore.set({ ...now, on: false });
    relayout();
    statusBarTextStore.set(tMsg('split.closedNarrow', { min: SPLIT_MIN_WIDTH }));
}

/** @param {'left' | 'right'} side */
function shake(side) {
    splitShakeStore.update((s) => ({ side, n: s.n + 1 }));
}

/**
 * Bring the pane of `side` live. Another library: its group is activated, and a refusal (a Duel,
 * a training session, an import: switchRefusal) leaves the focus where it was, shakes the pane
 * and lets the status bar say why. The same library on both sides: the two views trade places.
 * @param {'left' | 'right'} side
 * @returns {Promise<boolean>} whether that pane is live afterwards
 */
export async function activatePane(side) {
    const state = get(splitStore);
    if (!state.on) return false;
    const active = get(activeGroupId);
    const groups = paneGroups(state, get(groupIdsStore), active);
    if (groups.left === groups.right) return swapSameLibrary(state, side);
    const target = groups[side];
    if (target === active) return true;
    const ok = await activateGroup(target);
    if (!ok) shake(side);
    return ok;
}

/**
 * Two views of one library: the frozen one comes live on its position, and the position left
 * freezes in the other pane. A match replayed has no list to find the position in: refused.
 * @param {import('../stores/splitStore.js').SplitState} state
 * @param {'left' | 'right'} side
 */
async function swapSameLibrary(state, side) {
    if (state.sameSide === side) return true;
    const frozen = get(splitFrozenStore).positionId;
    const index = !get(matchContextStore)?.isMatchMode && frozen != null ? await positionsStore.findIndex(frozen) : -1;
    if (index == null || index < 0) {
        statusBarTextStore.set(tMsg('split.sameLibraryNotInList'));
        shake(side);
        return false;
    }
    const leaving = positionsStore.idAt(get(currentPositionIndexStore)) ?? get(positionStore)?.id ?? null;
    splitStore.update((s) => ({ ...s, sameSide: side }));
    splitFrozenStore.update((f) => ({ ...f, refreeze: { positionId: leaving, n: (f.refreeze?.n ?? 0) + 1 } }));
    currentPositionIndexStore.set(index);
    return true;
}

/**
 * @typedef {{ kind: 'library' | 'match' | 'collection', id: number }} ViewerSource
 * @typedef {{ source: ViewerSource, positionId: number | null, index: number }} ViewerStart
 */

/**
 * Where the frozen pane of a group starts: where that group was left. A match replayed stays on
 * its move; otherwise the library, on the position shown.
 * @param {any} photo a parked group's photograph (libraryGroupStore), or null for the active group
 * @returns {ViewerStart}
 */
export function viewerStart(photo) {
    /** @type {any} */
    let view;
    if (photo) {
        view = photo.views?.views?.find((/** @type {any} */ v) => v.id === photo.views.activeViewId) ?? null;
    } else {
        const index = get(currentPositionIndexStore);
        view = { matchContext: get(matchContextStore), positionId: positionsStore.idAt(index) ?? get(positionStore)?.id ?? null };
    }
    const match = view?.matchContext;
    if (match?.isMatchMode && match.matchID) return { source: { kind: 'match', id: match.matchID }, positionId: null, index: match.currentIndex || 0 };
    return { source: { kind: 'library', id: 0 }, positionId: view?.positionId ?? view?.position?.id ?? null, index: 0 };
}

/** @param {number} group */
export function viewerStartOf(group) {
    return group === get(activeGroupId) ? viewerStart(null) : viewerStart(get(parkedGroupsStore).get(group) ?? null);
}

/**
 * The split as it is kept from one session to the next: on or off, and the file on each side.
 * @param {import('../stores/splitStore.js').SplitState} state
 * @param {(group: number | null) => string} pathOf
 */
export function rememberSplit(state, pathOf) {
    try {
        localStorage.setItem(REMEMBERED, JSON.stringify({ on: state.on, left: pathOf(state.left), right: pathOf(state.right) }));
    } catch {
        // Storage refused (private profile): the split is simply not remembered.
    }
}

/** @returns {{ on: boolean, left: string, right: string } | null} */
export function rememberedSplit() {
    try {
        const saved = JSON.parse(localStorage.getItem(REMEMBERED) ?? 'null');
        if (saved && typeof saved.left === 'string' && typeof saved.right === 'string') return { on: !!saved.on, left: saved.left, right: saved.right };
    } catch {
        // Unreadable: nothing remembered.
    }
    return null;
}

/**
 * Put back the split of the last session, once the last library is open: the other side's file
 * is opened in a group of its own and the library that was active comes back on screen.
 * @param {string} activePath
 * @param {(path: string) => Promise<number | null>} openOther opens path in a new group and
 *   brings the active group back; resolves to the new group, or null when it could not be opened
 * @param {number} [width]
 * @returns {Promise<boolean>}
 */
export async function restoreSplit(activePath, openOther, width = windowWidth()) {
    try {
        return await putBackSplit(activePath, openOther, width);
    } finally {
        markSplitRestored();
    }
}

/** @param {string} activePath @param {(path: string) => Promise<number | null>} openOther @param {number} width */
async function putBackSplit(activePath, openOther, width) {
    const saved = rememberedSplit();
    if (!saved?.on || !activePath || width < SPLIT_MIN_WIDTH) return false;
    if (activePath !== saved.left && activePath !== saved.right) return false;
    const home = get(activeGroupId);
    const otherPath = activePath === saved.left ? saved.right : saved.left;
    let other = home;
    if (otherPath && otherPath !== activePath) {
        const opened = await openOther(otherPath);
        if (opened == null || get(activeGroupId) !== home) return false;
        other = opened;
    }
    const left = activePath === saved.left ? home : other;
    const right = left === home ? other : home;
    splitStore.set({ on: true, left, right, sameSide: 'left' });
    relayout();
    return true;
}
