// The « Ailleurs » view and the in-place swap (ADR-0101, ADR-0103). Both read what the other open
// library groups know of the board on screen through one bound call, ElsewhereInGroups, which
// runs the storage.Elsewhere service of the CLI and the daemon over the waiting handles. Nothing
// here writes: the swap draws the other board over this one and touches no store of the board.
import { get } from 'svelte/store';
import { ElsewhereInGroups } from '../../wailsjs/go/gui/App.js';
import { elsewhereStore, elsewherePinStore, elsewherePeekStore } from '../stores/elsewhereStore.js';
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore, statusBarTextStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { groupIdsStore, activeGroupId, activateGroup } from '../stores/libraryGroupStore.js';
import { tMsg } from '../i18n';
import { logger } from '../utils/logger.js';

/** How many nearest boards each other group lists. */
export const ELSEWHERE_NEIGHBOURS = 5;

/** A held key longer than this returns on release; a tap leaves the swap until the next press. */
export const SWAP_HOLD_MS = 300;

/**
 * The board as the hash sees it: the id names a row of this library only.
 * @param {any} position
 */
export function boardKey(position) {
    if (!position) return '';
    return JSON.stringify({ ...position, id: 0 });
}

/** The groups other than the active one. */
export function otherGroups() {
    const active = get(activeGroupId);
    return get(groupIdsStore).filter((id) => id !== active);
}

/** @type {Promise<import('../stores/elsewhereStore.js').ElsewhereView | null> | null} */
let inflight = null;
let inflightKey = '';

/**
 * Read what the other groups know of position, once per board: a second call for the same board
 * returns the answer already read, or the read under way.
 * @param {any} [position]
 * @param {{ force?: boolean }} [options]
 * @returns {Promise<import('../stores/elsewhereStore.js').ElsewhereView | null>}
 */
export async function loadElsewhere(position = get(positionStore), { force = false } = {}) {
    if (!get(databasePathStore) || !position || otherGroups().length === 0) {
        elsewhereStore.set({ key: '', view: null, loading: false, error: '' });
        return null;
    }
    const key = boardKey(position);
    const now = get(elsewhereStore);
    if (!force && now.key === key && now.view && !now.loading) return now.view;
    if (!force && inflight && inflightKey === key) return inflight;
    elsewhereStore.set({ key, view: now.key === key ? now.view : null, loading: true, error: '' });
    inflightKey = key;
    const read = (async () => {
        try {
            const view = await ElsewhereInGroups(JSON.parse(JSON.stringify(position)), ELSEWHERE_NEIGHBOURS);
            if (get(elsewhereStore).key === key) elsewhereStore.set({ key, view, loading: false, error: '' });
            return view;
        } catch (error) {
            logger.error('Error reading the other libraries:', error);
            if (get(elsewhereStore).key === key) elsewhereStore.set({ key, view: null, loading: false, error: String(error) });
            return null;
        } finally {
            if (inflightKey === key) inflight = null;
        }
    })();
    inflight = read;
    return read;
}

/**
 * Bring group forward on its position positionId: the group's own list when it holds it, else
 * its whole library focused on it.
 * @param {number} group
 * @param {number} positionId
 */
export async function openElsewhere(group, positionId) {
    endSwap();
    if (!(await activateGroup(group))) return false;
    if (!positionId) return true;
    const index = await positionsStore.findIndex(positionId);
    if (index >= 0) {
        currentPositionIndexStore.set(index);
        return true;
    }
    const { loadAllPositions } = await import('./positionService.js');
    await loadAllPositions({ focusId: positionId });
    return true;
}

/**
 * Pin a board of another group for the swap; pinning it again unpins it.
 * @param {import('../stores/elsewhereStore.js').ElsewherePin} pin
 */
export function togglePin(pin) {
    const now = get(elsewherePinStore);
    elsewherePinStore.set(now && now.group === pin.group && now.positionId === pin.positionId ? null : pin);
}

/** The pin, while its group is still open and waiting. */
function livePin() {
    const pin = get(elsewherePinStore);
    return pin && otherGroups().includes(pin.group) ? pin : null;
}

/**
 * What the swap shows, from a view read for the very board it will draw: with a pin, that
 * pinned board and its own report in its group; otherwise the board on screen in the first other
 * group that holds it. The banner describes the drawn board only, so a report of another board
 * is never borrowed.
 * @param {import('../stores/elsewhereStore.js').ElsewhereView | null} view
 * @param {any} position the board on screen
 * @param {import('../stores/elsewhereStore.js').ElsewherePin | null} [pin]
 * @returns {import('../stores/elsewhereStore.js').ElsewherePeek | null}
 */
export function swapTarget(view, position, pin = null) {
    const groups = view?.groups ?? [];
    if (pin) {
        const report = groups.find((g) => g.group === pin.group && g.positionId === pin.positionId);
        return report ? { group: pin.group, path: pin.path, position: pin.position, report, pinned: true } : null;
    }
    const held = groups.find((g) => g.positionId);
    if (!held) return null;
    return { group: held.group, path: held.path, position, report: held, pinned: false };
}

let pressedAt = 0;
// Moves at every start and end, so a read still under way when the key is released shows nothing.
let swapGeneration = 0;

/**
 * Show the other library's board in place of this one. The board drawn is read afresh: a group
 * may have been written since the last read (a job of its own), and a pin is read as its own board.
 */
export async function startSwap() {
    const generation = ++swapGeneration;
    if (otherGroups().length === 0) {
        statusBarTextStore.set(tMsg('elsewhere.noOtherLibrary'));
        return false;
    }
    const position = get(positionStore);
    const pin = livePin();
    let view = null;
    if (pin) {
        try {
            view = await ElsewhereInGroups(JSON.parse(JSON.stringify(pin.position)), -1);
        } catch (error) {
            logger.error('Error reading the pinned board:', error);
        }
    } else {
        view = await loadElsewhere(position, { force: true });
    }
    if (generation !== swapGeneration) return false;
    const target = swapTarget(view, position, pin);
    if (!target) {
        statusBarTextStore.set(tMsg('elsewhere.notHeld'));
        return false;
    }
    elsewherePeekStore.set(target);
    return true;
}

/** Put this library's board back. */
export function endSwap() {
    swapGeneration++;
    if (get(elsewherePeekStore)) elsewherePeekStore.set(null);
}

/**
 * The swap key went down: a press while the swap shows returns, otherwise it shows.
 * @param {KeyboardEvent} event
 */
export function swapKeyDown(event) {
    if (event.repeat) return;
    if (get(elsewherePeekStore)) {
        pressedAt = 0;
        endSwap();
        return;
    }
    pressedAt = performance.now();
    startSwap().catch((error) => logger.error('Error swapping in another library:', error));
}

/** The swap key came up: a hold returns, a tap leaves the swap until the next press. */
export function swapKeyUp() {
    if (pressedAt && performance.now() - pressedAt >= SWAP_HOLD_MS) endSwap();
    pressedAt = 0;
}

/**
 * The line an analysis read elsewhere is shown by: its best play and the engine that found it.
 * @param {any} analysis
 * @returns {{ kind: 'checker' | 'cube' | 'none', best: string, engine: string }}
 */
export function analysisSummary(analysis) {
    if (!analysis) return { kind: 'none', best: '', engine: '' };
    const engine = analysis.analysisEngineVersion || analysis.analysisType || '';
    const moves = analysis.checkerAnalysis?.moves ?? [];
    if (moves.length > 0) return { kind: 'checker', best: moves[0].move || '', engine };
    const cube = analysis.doublingCubeAnalysis;
    if (cube?.bestCubeAction) return { kind: 'cube', best: cube.bestCubeAction, engine };
    return { kind: 'none', best: '', engine };
}

/**
 * A cost in millipoints as the panels show it, in points; '' when unscored.
 * @param {number | null | undefined} errorMP
 */
export function formatCost(errorMP) {
    return errorMP == null ? '' : (errorMP / 1000).toFixed(3);
}
