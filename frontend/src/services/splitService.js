// The split screen (ADR-0101). Two panes of the same width: the live group, and a read-only viewer
// of a library group (its board, analysis and comment, walked through its library, a match move
// by move or a collection). Clicking the viewer activates its group: the two libraries trade
// roles and each keeps its pane. Below SPLIT_MIN_WIDTH the split is refused and the in-place swap
// (x) offered instead.
import { get } from 'svelte/store';
import { splitStore } from '../stores/splitStore.js';
import { statusBarTextStore } from '../stores/uiStore.js';
import { groupIdsStore, activeGroupId, parkedGroupsStore, activateGroup } from '../stores/libraryGroupStore.js';
import { positionStore, matchContextStore, positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { tMsg } from '../i18n';

/** Narrower than this (CSS pixels), a pane could not show a board and its panel side by side. */
export const SPLIT_MIN_WIDTH = 1280;

/**
 * The live pane changed width without the window doing so: what sizes itself on the window's
 * resize (the board, the panels) measures again once the new layout is in place.
 */
function relayout() {
    requestAnimationFrame(() => window.dispatchEvent(new Event('resize')));
}

/** The window's width in CSS pixels, the interface scale taken out. */
export function windowWidth() {
    const scale = parseFloat(getComputedStyle(document.documentElement).getPropertyValue('--ui-scale')) || 1;
    return window.innerWidth / scale;
}

/**
 * Turn the split on or off. Too narrow a window refuses it and offers the swap.
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
    splitStore.set({ ...now, on: true });
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

/**
 * The group the viewer shows: the one it names while it is open, else the first other group,
 * else the active one (a second look at the same library).
 * @param {number | null} wanted
 * @param {number[]} ids
 * @param {number} active
 */
export function viewerGroup(wanted, ids, active) {
    if (wanted != null && ids.includes(wanted)) return wanted;
    return ids.find((id) => id !== active) ?? active;
}

/**
 * The active group moved from `from` to `to`: when the viewer's group came on screen, the viewer
 * takes the group left and the two panes trade sides, so each library stays where it was seen.
 * @param {import('../stores/splitStore.js').SplitState} state
 * @param {number} from
 * @param {number} to
 * @param {number[]} ids
 * @returns {import('../stores/splitStore.js').SplitState}
 */
export function followActivation(state, from, to, ids) {
    if (from === to || viewerGroup(state.group, ids, from) !== to) return state;
    return { ...state, group: from, side: state.side === 'left' ? 'right' : 'left' };
}

/** Bring the viewer's group live, in its pane. */
export async function activateViewer() {
    const state = get(splitStore);
    const target = viewerGroup(state.group, get(groupIdsStore), get(activeGroupId));
    if (target === get(activeGroupId)) return false;
    return activateGroup(target);
}

/**
 * @typedef {{ kind: 'library' | 'match' | 'collection', id: number }} ViewerSource
 * @typedef {{ source: ViewerSource, positionId: number | null, index: number }} ViewerStart
 */

/**
 * Where the viewer of a group starts: where that group was left. A match replayed stays on its
 * move; otherwise the library, on the position shown.
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
