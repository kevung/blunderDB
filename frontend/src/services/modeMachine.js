/**
 * modeMachine.js — the board's mode automaton.
 *
 * The mode lives in `statusBarModeStore` (never duplicated here):
 *
 *   NORMAL      browsing the library
 *   MATCH       replaying a match (matchContextStore drives navigation)
 *   COLLECTION  browsing a collection
 *   EDIT        search tab: the board is a query being drawn
 *   EVAL        Eval tab: the board is a scratch pad for the engine
 *   TRANSCRIBE  Transcription tab: the board is the draft's Cursor
 *
 * EDIT, EVAL and TRANSCRIBE are scratch modes: entering snapshots what was
 * studied (`savedContext`), and every exit resumes the snapshot's mode through
 * returnToStudiedMode() — a match or collection is never dropped to NORMAL.
 * A snapshot holds positions, never an analysis: showPosition() refetches it on
 * the way back, and during a scratch mode analysisStore still describes the
 * position studied before.
 *
 * Transitions (one exported function each):
 *
 *   enterEditMode       NORMAL | MATCH | COLLECTION | EVAL → EDIT
 *   exitEditMode        EDIT → the mode it was entered from
 *   enterEvalMode       NORMAL | MATCH | COLLECTION | EDIT → EVAL
 *   exitEvalMode        EVAL → the mode it was entered from
 *   toggleEvalMode      EVAL → (exit + analysis tab) | * → Eval tab
 *   enterTranscribeMode NORMAL | MATCH | COLLECTION | EDIT | EVAL → TRANSCRIBE
 *   exitTranscribeMode  TRANSCRIBE → the mode it was entered from
 *   sendPositionToEval  * → EVAL on a given position (id cleared)
 *   toggleMatchMode     MATCH → NORMAL | * → MATCH
 *   handleOpenCollection * → COLLECTION
 *   exitCollectionMode  COLLECTION → NORMAL
 *   leaveSubSearchResults NORMAL (results of an `ss` run from a collection or
 *                       match) → that COLLECTION | MATCH
 *
 * displayedPositionIDs() is the only answer to "which list is on screen".
 *
 * App.svelte's tab effect runs the previous tab's exit before the new tab's
 * entry; a direct EDIT ↔ EVAL call still leaves the current scratch mode first,
 * so one snapshot is never buried under the other.
 *
 * positionService.js re-exports every transition: a deliberate import cycle
 * (it owns the loaders, this module the transitions), safe because nothing
 * here runs at module evaluation.
 */

import { get } from 'svelte/store';
import { ListPositionIDs, SaveLastVisitedPosition, GetLastVisitedMatch, GetMatchMovePositions } from '../../wailsjs/go/database/Database.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, positionsStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { epcDataStore } from '../stores/epcStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';
import { currentPositionIndexStore, statusBarTextStore, statusBarModeStore, PANEL, closePanel, activeTabStore } from '../stores/uiStore.js';
import { activeCollectionStore, collectionPositionsStore, selectedCollectionStore } from '../stores/collectionStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { showPosition, loadAllPositions, loadAnalysisForPosition, setSearchState, getSearchState } from './positionService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

export const MODE = Object.freeze({
    NORMAL: 'NORMAL',
    MATCH: 'MATCH',
    COLLECTION: 'COLLECTION',
    EDIT: 'EDIT',
    EVAL: 'EVAL',
    // The Transcription tab (ADR-0045). A scratch mode like EDIT and EVAL: the
    // board shows the Action the Cursor is on, which belongs to a draft and
    // not to the library.
    TRANSCRIBE: 'TRANSCRIBE'
});

const NO_MATCH_CONTEXT = Object.freeze({
    isMatchMode: false,
    matchID: null,
    movePositions: [],
    currentIndex: 0,
    player1Name: '',
    player2Name: ''
});

/**
 * The `savedContext` half of the state. Each slot is written by one entry
 * transition and nulled by the matching exit:
 *
 *   beforeEval  { mode, matchContext, position, positionIndex, ids }, so the
 *               exit returns to a studied match instead of NORMAL.
 *   beforeEdit  { mode, matchContext }, rewritten on *every* enterEditMode so a
 *               stale snapshot is never restored. No list: the one behind the
 *               query board stays in positionsStore.
 *   beforeTranscribe  the same photograph, in its own slot so that visiting
 *               both panels never buries one snapshot under the other.
 *   evalSeed    the position Eval must open on. A slot, not a parameter:
 *               sendPositionToEval() and enterEvalMode() run a tick apart
 *               (through App.svelte's tab effect).
 *   lastEvalBoard the board Eval was last left on. NOT consumed on use, and
 *               survives a library reload: a scratch board belongs to the
 *               session. id forced to 0, no analysis (it would be stale).
 *   beforeSubSearch the collection or match an `ss` was run from:
 *               { mode: COLLECTION, collection, ids, positionIndex, position, resultIds }
 *               or { mode: MATCH, matchContext, resultIds }. The way back is
 *               offered only while `resultIds` is still the list shown. A
 *               sub-search inside those results keeps it; a library search,
 *               reload, or opened match/collection drops it.
 */
/**
 * Annotated because every slot starts `null`, which the checker would infer.
 *
 * @type {{beforeTranscribe: any, beforeEval: any, beforeEdit: any, beforeSubSearch: any, evalSeed: any, lastEvalBoard: any}}
 */
const savedContext = {
    beforeTranscribe: null,
    beforeSubSearch: null,
    beforeEval: null,
    beforeEdit: null,
    evalSeed: null,
    lastEvalBoard: null
};

/** Read-only snapshot of the machine's state, for tests and debugging. */
export function modeState() {
    return { mode: get(statusBarModeStore), savedContext: { ...savedContext } };
}

/**
 * Forget what enterEvalMode saved: after a library reload, exitEvalMode must
 * reload too rather than restore a stale list.
 */
export function forgetContextBeforeEval() {
    savedContext.beforeEval = null;
}

function currentMode() {
    return get(statusBarModeStore);
}

/**
 * The mode a scratch mode was entered from, plus the match context needed to
 * resume it. Every scratch entry takes one; every exit hands it to
 * returnToStudiedMode() — one rule for the three panels.
 */
function photographStudiedMode() {
    return { mode: currentMode(), matchContext: { ...get(matchContextStore) } };
}

/**
 * Put back the mode a scratch mode was entered from. A collection is resumed
 * only while still active (a search or opened match has replaced the list).
 * The caller restores the position afterwards, so no effect sees it under a
 * scratch mode.
 *
 * @param {{ mode: string, matchContext: any } | null | undefined} saved
 * @returns {string} the mode now on
 */
function returnToStudiedMode(saved) {
    if (saved?.mode === MODE.MATCH && saved.matchContext?.isMatchMode) {
        matchContextStore.set(saved.matchContext);
        statusBarModeStore.set(MODE.MATCH);
        return MODE.MATCH;
    }
    if (saved?.mode === MODE.COLLECTION && get(activeCollectionStore)) {
        statusBarModeStore.set(MODE.COLLECTION);
        return MODE.COLLECTION;
    }
    statusBarModeStore.set(MODE.NORMAL);
    return MODE.NORMAL;
}

// ── The list on screen, and the way back from a sub-search ───────────────────

/**
 * The positions of the list on screen, searched by `ss` and the Search panel's
 * "search in current results".
 *
 *   MATCH       matchContextStore.movePositions
 *   EDIT        the match photographed on entry (enterEditMode empties
 *               matchContextStore), otherwise positionsStore
 *   otherwise   positionsStore: library, search results, collection
 *
 * The backend receives a set: a position met twice is searched once.
 *
 * @returns {number[]}
 */
export function displayedPositionIDs() {
    const mode = currentMode();
    const behindQueryBoard = mode === MODE.EDIT ? savedContext.beforeEdit : null;
    const matchContext = mode === MODE.MATCH ? get(matchContextStore) : behindQueryBoard?.mode === MODE.MATCH ? behindQueryBoard.matchContext : null;
    const ids = matchContext?.isMatchMode ? (matchContext.movePositions ?? []).map((/** @type {any} */ mp) => mp?.position?.id) : (get(positionsStore)?.ids ?? []);
    return [...new Set(ids.filter((/** @type {any} */ id) => id != null))];
}

/**
 * The collection or match being studied, seen through the query board when
 * the search ran from the Search tab (there the collection's position comes
 * from the list's cache, as in exitEditMode).
 */
function subSearchOriginNow() {
    const mode = currentMode();
    const behindQueryBoard = mode === MODE.EDIT ? savedContext.beforeEdit : null;
    const studied = behindQueryBoard ? behindQueryBoard.mode : mode;
    if (studied === MODE.MATCH) {
        const matchContext = behindQueryBoard ? behindQueryBoard.matchContext : get(matchContextStore);
        if (matchContext?.isMatchMode) return { mode: MODE.MATCH, matchContext: { ...matchContext } };
        return null;
    }
    if (studied === MODE.COLLECTION && get(activeCollectionStore)) {
        const positionIndex = get(currentPositionIndexStore);
        const onScreen = behindQueryBoard ? positionsStore.peek(positionIndex) : get(positionStore);
        return {
            mode: MODE.COLLECTION,
            collection: get(activeCollectionStore),
            ids: [...(get(positionsStore)?.ids ?? [])],
            positionIndex,
            position: onScreen ? JSON.parse(JSON.stringify(onScreen)) : null
        };
    }
    return null;
}

/** Whether the list on screen is still the one the last sub-search put there. */
function subSearchResultsOnScreen() {
    const saved = savedContext.beforeSubSearch;
    if (!saved) return false;
    const mode = currentMode();
    const listMode = mode === MODE.EDIT ? savedContext.beforeEdit?.mode : mode;
    if (listMode !== MODE.NORMAL) return false;
    const ids = get(positionsStore)?.ids ?? [];
    return ids.length === saved.resultIds.length && ids.every((id, i) => id === saved.resultIds[i]);
}

/**
 * Called before a search's results replace the list. A sub-search from a
 * collection or match records it; one inside such results keeps the origin;
 * any other search forgets it.
 *
 * @param {boolean} isSubSearch the search was restricted to the list on screen
 * @param {number[]} resultIds the list about to be shown
 * @returns {string | null} the mode the results can return to (MATCH | COLLECTION), or null
 */
export function noteSubSearchOrigin(isSubSearch, resultIds) {
    if (!isSubSearch) {
        savedContext.beforeSubSearch = null;
        return null;
    }
    const origin = subSearchOriginNow();
    if (origin) {
        savedContext.beforeSubSearch = { ...origin, resultIds: [...resultIds] };
    } else if (subSearchResultsOnScreen()) {
        savedContext.beforeSubSearch = { ...savedContext.beforeSubSearch, resultIds: [...resultIds] };
    } else {
        savedContext.beforeSubSearch = null;
    }
    return savedContext.beforeSubSearch?.mode ?? null;
}

/** Forget where a sub-search came from: the list it could return to is gone. */
export function forgetSubSearchOrigin() {
    savedContext.beforeSubSearch = null;
}

/**
 * Whether sub-search results are on screen, so Escape leaves them before a
 * panel with nothing to close closes itself.
 *
 * @returns {boolean}
 */
export function canLeaveSubSearchResults() {
    return currentMode() === MODE.NORMAL && subSearchResultsOnScreen();
}

/**
 * Leave `ss` results for the collection or match they came from, on the
 * position left. Mode first, then showPosition, as the scratch exits do.
 * Returns false when the list on screen is no longer those results.
 *
 * @returns {Promise<boolean>} whether a list was returned to
 */
export async function leaveSubSearchResults() {
    if (!canLeaveSubSearchResults()) return false;
    const saved = savedContext.beforeSubSearch;
    savedContext.beforeSubSearch = null;

    setSearchState('', null, false);
    lastSearchStore.set(null);
    statusBarTextStore.set('');

    if (saved.mode === MODE.MATCH) {
        matchContextStore.set(saved.matchContext);
        statusBarModeStore.set(MODE.MATCH);
        const movePos = saved.matchContext.movePositions?.[saved.matchContext.currentIndex];
        if (movePos) await showPosition(movePos.position);
        return true;
    }

    activeCollectionStore.set(saved.collection);
    statusBarModeStore.set(MODE.COLLECTION);
    positionsStore.setIds(saved.ids);
    currentPositionIndexStore.set(saved.positionIndex);
    if (saved.position) await showPosition(saved.position);
    return true;
}

async function persistLastVisitedMatchPosition() {
    const ctx = get(matchContextStore);
    if (!ctx.isMatchMode || !ctx.matchID) return;
    try {
        await SaveLastVisitedPosition(ctx.matchID, ctx.currentIndex);
    } catch (e) {
        logger.error('Error saving last visited position:', e);
    }
}

/** @param {any} pos */
function blankEditBoard(pos) {
    pos.board.points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    pos.board.bearoff = [15, 15];
    pos.cube = { owner: -1, value: 0 };
    pos.score = [7, 7];
    pos.dice = [3, 1];
    pos.decision_type = 0;
    pos.player_on_roll = 0;
    return pos;
}

// ── Saving a scratch board ───────────────────────────────────────────────────

/**
 * A position just saved from a scratch board joins the list the exit will put
 * back — only when that list is the whole library; any other studied list is
 * left untouched.
 *
 * In EDIT that list is positionsStore; in EVAL, beforeEval's ids. Only a board
 * entered from NORMAL can have the library behind it, and "whole library" is
 * checked on ids (list + id === ListPositionIDs), not inferred from flags: a
 * deck or statistics selection is also NORMAL with no search. The id goes
 * last; the index is untouched.
 *
 * @param {number} id the position just written (a new one)
 * @returns {Promise<boolean>} whether the id was added
 */
export async function joinLibraryBehindScratchBoard(id) {
    /** @type {(number | null)[]} */
    let ids;
    /** @type {(next: number[]) => void} */
    let replace;
    if (currentMode() === MODE.EDIT) {
        const saved = savedContext.beforeEdit;
        if (saved && saved.mode !== MODE.NORMAL) return false;
        ids = get(positionsStore)?.ids ?? [];
        replace = (next) => positionsStore.setIds(next);
    } else if (currentMode() === MODE.EVAL) {
        const saved = savedContext.beforeEval;
        if (!saved || saved.mode !== MODE.NORMAL || !saved.ids) return false;
        ids = saved.ids;
        replace = (next) => {
            saved.ids = next;
        };
    } else {
        return false;
    }
    if (getSearchState().hasActiveSearch || ids.includes(id)) return false;

    let library;
    try {
        library = (await ListPositionIDs()) || [];
    } catch (error) {
        logger.error('Error listing the library after a scratch-board save:', error);
        return false;
    }
    const next = [...ids, id];
    if (library.length !== next.length || library.some((libraryId, i) => libraryId !== next[i])) return false;
    replace(next);
    return true;
}

// ── EDIT ─────────────────────────────────────────────────────────────────────

/** NORMAL | MATCH | COLLECTION | EVAL → EDIT. */
export async function enterEditMode() {
    logger.log('enterEditMode');
    if (!get(databasePathStore)) return;

    if (currentMode() === MODE.TRANSCRIBE) {
        // Same reason as the Eval branch below: leave the transcription panel
        // through its exit, which restores the studied position before the
        // snapshot is taken.
        await exitTranscribeMode();
    }

    if (currentMode() === MODE.EVAL) {
        // Through the exit, not toggleEvalMode(), which would also flip the tab
        // to 'analysis'. Its synchronous prefix restores mode and position
        // before the snapshot below.
        exitEvalMode();
    }

    // Snapshot what is studied — the library, a match or a collection — so that
    // leaving the search tab returns to it.
    savedContext.beforeEdit = photographStudiedMode();

    if (currentMode() === MODE.MATCH) {
        logger.log('Exiting MATCH mode to enter EDIT');
        await persistLastVisitedMatchPosition();
        matchContextStore.set({ ...NO_MATCH_CONTEXT });
        // Not loadAllPositions(): unawaited, it would set NORMAL and flip the
        // tab to 'matches' on resolving. EDIT does not read positionsStore, and
        // exitEditMode restores the snapshot.
    }

    // A collection is not left: its ids stay in positionsStore behind the
    // query board and exitEditMode resumes it from the snapshot.

    if (currentMode() !== MODE.EDIT) {
        statusBarModeStore.set(MODE.EDIT);
        // Clear the selected move: the board only clears it on an id change,
        // and the blank board keeps the studied id, so arrows would persist.
        selectedMoveStore.set(null);
        // Blank a copy: the object on the board may be the list's cached
        // record, which exitEditMode puts back.
        positionStore.update((pos) => blankEditBoard(JSON.parse(JSON.stringify(pos))));
    }
}

/** EDIT → the mode it was entered from: MATCH | COLLECTION | NORMAL. */
export async function exitEditMode() {
    if (currentMode() !== MODE.EDIT) return;

    const saved = savedContext.beforeEdit;
    savedContext.beforeEdit = null;
    // Entered from a match: return to the studied move rather than dropping
    // into the flat "all positions" list.
    if (returnToStudiedMode(saved) === MODE.MATCH) {
        const snap = saved.matchContext;
        const movePos = snap.movePositions?.[snap.currentIndex];
        if (movePos) {
            await showPosition(movePos.position);
            statusBarTextStore.set(`${snap.player1Name} vs ${snap.player2Name}`);
        }
        return;
    }
    // Restore the studied position synchronously from the window cache before
    // bumping the index: App.svelte calls this unawaited then enterEvalMode,
    // which would otherwise photograph the blank query board. On a cache miss
    // the nav effect fetches it.
    const currentIndex = get(currentPositionIndexStore);
    const cached = positionsStore.peek(currentIndex);
    if (cached) positionStore.set(JSON.parse(JSON.stringify(cached)));
    // Bump the index through -1 so the navigation effect redraws the library
    // position (and reloads its analysis) the blank EDIT board replaced.
    currentPositionIndexStore.set(-1);
    currentPositionIndexStore.set(currentIndex);
}

// ── EVAL (Eval tab) ──────────────────────────────────────────────────────────

/** EVAL → exit and show the analysis tab; otherwise open the Eval tab. */
export function toggleEvalMode() {
    if (currentMode() === MODE.EVAL) {
        exitEvalMode();
        activeTabStore.set('analysis');
    } else {
        activeTabStore.set('eval');
    }
}

// The board the Eval panel opens on when nothing was handed to it: the
// canonical 6-point bearoff.
function defaultEvalPosition() {
    const points = Array(26).fill({ checkers: 0, color: -1 });
    points[1] = { checkers: 2, color: 0 };
    points[2] = { checkers: 2, color: 0 };
    points[3] = { checkers: 2, color: 0 };
    points[4] = { checkers: 3, color: 0 };
    points[5] = { checkers: 3, color: 0 };
    points[6] = { checkers: 3, color: 0 };

    return {
        id: 0,
        board: { points, bearoff: [0, 15] },
        cube: { owner: -1, value: 0 },
        dice: [0, 0],
        score: [-1, -1],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        max_cube: 0,
        has_beaver: 0
    };
}

/**
 * Open the Eval panel on `position` instead of its default bearoff.
 *
 * The copy is detached and its id cleared, so a later Ctrl-U cannot write the
 * edited board over the source record. player_on_roll travels as-is: the Eval
 * facts and gammonNet both read it.
 *
 * @param {any} position
 */
export function sendPositionToEval(position) {
    if (!position) return;
    const seed = JSON.parse(JSON.stringify(position));
    seed.id = 0;

    if (currentMode() === MODE.EVAL) {
        // Already in the Eval panel: replace the board in place. Going through
        // the tab store would be a no-op and enterEvalMode() returns early.
        positionsStore.set([seed]);
        positionStore.set(seed);
        currentPositionIndexStore.set(0);
        return;
    }

    savedContext.evalSeed = seed;
    // If the Eval tab is already active without EVAL mode, set() is a no-op
    // and the tab effect never runs: enter directly.
    if (get(activeTabStore) === 'eval') enterEvalMode();
    else activeTabStore.set('eval');
}

/**
 * NORMAL | MATCH | COLLECTION | EDIT → EVAL. EVAL is set before the scratch
 * board lands in positionStore, in one synchronous run, or updateEPC fires on
 * the wrong position. The only await (leaving EDIT) sits before that run.
 */
export async function enterEvalMode() {
    if (currentMode() === MODE.EVAL) return;

    if (currentMode() === MODE.TRANSCRIBE) {
        await exitTranscribeMode();
        if (currentMode() === MODE.EVAL) return;
    }

    if (currentMode() === MODE.EDIT) {
        // Awaited: only a completed exit guarantees the studied record is back
        // in place of the blank query board before the snapshot below.
        await exitEditMode();
        if (currentMode() === MODE.EVAL) return;
    }

    savedContext.beforeEval = {
        ...photographStudiedMode(),
        position: get(positionStore) ? { ...get(positionStore) } : null,
        positionIndex: get(currentPositionIndexStore),
        ids: get(positionsStore)?.ids ?? null
    };

    // A position sent from the library wins; otherwise pick up the board the
    // panel was last left on, and only fall back to the default bearoff the
    // first time it is opened.
    const evalPosition = savedContext.evalSeed ?? savedContext.lastEvalBoard ?? defaultEvalPosition();
    savedContext.evalSeed = null;

    statusBarModeStore.set(MODE.EVAL);

    positionsStore.set([evalPosition]);
    positionStore.set(evalPosition);
    currentPositionIndexStore.set(0);
}

/**
 * EVAL → the mode it was entered from. Mode restored synchronously before the
 * position, so the Eval effect never sees it under EVAL.
 */
export async function exitEvalMode() {
    if (currentMode() !== MODE.EVAL) return;

    const saved = savedContext.beforeEval;
    savedContext.beforeEval = null;

    // Photograph the board on the way out so returning to the panel finds the
    // work rather than the default bearoff. A board, not a record: the id is
    // dropped so it can never be mistaken for a library position.
    const leaving = get(positionStore);
    savedContext.lastEvalBoard = leaving ? { ...leaving, id: 0 } : null;

    statusBarTextStore.set('');
    epcDataStore.set({ bottomEPC: null, topEPC: null, race: null, error: null });

    if (returnToStudiedMode(saved) === MODE.MATCH) {
        statusBarTextStore.set(`${saved.matchContext.player1Name} vs ${saved.matchContext.player2Name}`);
    }

    if (!saved?.ids) {
        loadAllPositions({ focusId: saved?.position?.id ?? null });
        return;
    }
    positionsStore.setIds(saved.ids);
    if (saved.position) {
        currentPositionIndexStore.set(saved.positionIndex);
        // showPosition, not positionStore.set: refetches the analysis (MATCH
        // mode's nav effect does not redraw on its own).
        await showPosition(saved.position);
    }
}

// ── TRANSCRIBE (Transcription tab) ───────────────────────────────────────────

/**
 * NORMAL | MATCH | COLLECTION | EDIT | EVAL → TRANSCRIBE.
 *
 * Photographs what was studied, as Eval does, so leaving returns to it. The
 * board is not replaced here: the panel puts the Cursor's position on it once
 * a draft is open.
 */
export async function enterTranscribeMode() {
    if (currentMode() === MODE.TRANSCRIBE) return;

    // Leave the other scratch modes first, so the snapshot below is the
    // studied position (or match) and not the query board or the Eval scratch
    // pad — the same ordering enterEvalMode applies to EDIT.
    if (currentMode() === MODE.EDIT) {
        await exitEditMode();
        if (currentMode() === MODE.TRANSCRIBE) return;
    }
    if (currentMode() === MODE.EVAL) {
        await exitEvalMode();
        if (currentMode() === MODE.TRANSCRIBE) return;
    }

    savedContext.beforeTranscribe = {
        ...photographStudiedMode(),
        position: get(positionStore) ? { ...get(positionStore) } : null,
        positionIndex: get(currentPositionIndexStore),
        ids: get(positionsStore)?.ids ?? null
    };

    statusBarModeStore.set(MODE.TRANSCRIBE);
}

/** TRANSCRIBE → the mode it was entered from: MATCH | COLLECTION | NORMAL. */
export async function exitTranscribeMode() {
    if (currentMode() !== MODE.TRANSCRIBE) return;

    const saved = savedContext.beforeTranscribe;
    savedContext.beforeTranscribe = null;

    statusBarTextStore.set('');

    if (returnToStudiedMode(saved) === MODE.MATCH) {
        statusBarTextStore.set(`${saved.matchContext.player1Name} vs ${saved.matchContext.player2Name}`);
    }

    if (!saved?.ids) {
        loadAllPositions({ focusId: saved?.position?.id ?? null });
        return;
    }
    positionsStore.setIds(saved.ids);
    if (saved.position) {
        currentPositionIndexStore.set(saved.positionIndex);
        // Through showPosition, not a bare set: the analysis panel is
        // repopulated on the way back, as it is on the way out of EVAL.
        await showPosition(saved.position);
    }
}

// ── MATCH ────────────────────────────────────────────────────────────────────

/** MATCH → NORMAL, or any other mode → MATCH on the last visited match. */
export async function toggleMatchMode() {
    logger.log('toggleMatchMode');
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    if (currentMode() === MODE.MATCH) {
        logger.log('Exiting MATCH mode to NORMAL mode via toggleMatchMode');
        // The move being studied is a library position too: stay on it rather
        // than land on the last position of the library.
        const leavingId = get(positionStore)?.id ?? null;
        await persistLastVisitedMatchPosition();
        statusBarModeStore.set(MODE.NORMAL);
        matchContextStore.set({ ...NO_MATCH_CONTEXT });
        await loadAllPositions({ focusId: leavingId });
        return;
    }

    if (currentMode() !== MODE.NORMAL) {
        // A scratch or collection mode is abandoned, not exited: its board is
        // about to be replaced by the match position anyway.
        statusBarModeStore.set(MODE.NORMAL);
        savedContext.beforeEval = null;
        savedContext.beforeEdit = null;
    }
    activeCollectionStore.set(null);
    savedContext.beforeSubSearch = null;

    try {
        const match = await GetLastVisitedMatch();
        if (!match) {
            setStatusBarMessage(tMsg('status.noMatchesInDb'));
            return;
        }

        const movePositions = await GetMatchMovePositions(match.id);
        if (!movePositions || movePositions.length === 0) {
            setStatusBarMessage(tMsg('status.noMovesInMatch'));
            return;
        }

        let startIndex = 0;
        if (match.last_visited_position >= 0 && match.last_visited_position < movePositions.length) {
            startIndex = match.last_visited_position;
        }

        matchContextStore.set({
            isMatchMode: true,
            matchID: match.id,
            movePositions: movePositions,
            currentIndex: startIndex,
            player1Name: match.player1_name,
            player2Name: match.player2_name
        });

        // Mode first, then showPosition: it then reads the played move from the
        // match and hides the cube analysis on an opening position.
        statusBarModeStore.set(MODE.MATCH);
        const startMovePos = movePositions[startIndex];
        await showPosition(startMovePos.position);
        selectedMoveStore.set(null);
        // Player names are shown in the match-info header bar above the board
        // (MatchInfoBar.svelte), not the status bar.

        lastVisitedMatchStore.set({
            matchID: match.id,
            currentIndex: startIndex,
            gameNumber: startMovePos.game_number
        });
    } catch (error) {
        logger.error('Error entering match mode:', error);
        const errMsg = error?.toString() || '';
        if (errMsg.includes('no matches')) {
            setStatusBarMessage(tMsg('status.noMatchesInDb'));
        } else {
            setStatusBarMessage(tMsg('status.errorEnteringMatchMode'));
        }
    }
}

// ── COLLECTION ───────────────────────────────────────────────────────────────

/**
 * Any mode → COLLECTION.
 *
 * @param {any} collection
 * @param {any[]} collectionPositions
 */
export function handleOpenCollection(collection, collectionPositions) {
    if (!collectionPositions || collectionPositions.length === 0) {
        statusBarTextStore.set(tMsg('commands.collectionEmpty'));
        return;
    }

    savedContext.beforeSubSearch = null;
    if (get(matchContextStore).isMatchMode) {
        matchContextStore.update((ctx) => ({
            ...ctx,
            isMatchMode: false,
            matchID: null,
            movePositions: [],
            currentIndex: 0
        }));
    }

    statusBarModeStore.set(MODE.COLLECTION);
    positionsStore.set(collectionPositions);
    positionStore.set(collectionPositions[0]);
    currentPositionIndexStore.set(0);
    loadAnalysisForPosition(collectionPositions[0]);
    statusBarTextStore.set(tMsg('commands.collectionLoaded', { name: collection.name, count: collectionPositions.length }));
}

/** COLLECTION → NORMAL, back on the library at the last viewed position. */
export async function exitCollectionMode() {
    logger.log('Exiting COLLECTION mode to NORMAL mode');
    const lastViewedPosition = get(positionStore);
    statusBarModeStore.set(MODE.NORMAL);
    activeCollectionStore.set(null);
    selectedCollectionStore.set(null);
    collectionPositionsStore.set([]);
    closePanel(PANEL.COLLECTION);
    try {
        const ids = (await ListPositionIDs()) || [];
        positionsStore.setIds(ids, { reset: true });
        if (ids.length > 0) {
            let targetIdx = ids.length - 1;
            if (lastViewedPosition && lastViewedPosition.id) {
                const foundIdx = ids.indexOf(lastViewedPosition.id);
                if (foundIdx >= 0) targetIdx = foundIdx;
            }
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(targetIdx);
            loadAnalysisForPosition({ id: ids[targetIdx] });
            setSearchState('', null, false);
            lastSearchStore.set(null);
        }
    } catch (error) {
        logger.error('Error reloading positions after collection exit:', error);
        loadAllPositions();
    }
}
