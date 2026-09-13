/**
 * modeMachine.js — the board's mode automaton.
 *
 * The board is always in exactly one mode. The mode itself is published by
 * `statusBarModeStore` — every component reads it there, so that store *is*
 * the `mode` half of the machine's state and is never duplicated here:
 *
 *   NORMAL      browsing the library (positionsStore's ids + currentPositionIndexStore)
 *   MATCH       replaying a match (matchContextStore drives the navigation)
 *   COLLECTION  browsing a collection (positionsStore holds its ids)
 *   EDIT        the search tab: the board is a query being drawn
 *   EVAL        the Eval tab: the board is a scratch pad for the engine
 *   TRANSCRIBE  the Transcription tab: the board is the draft's Cursor
 *
 * EDIT, EVAL and TRANSCRIBE are *scratch* modes: the board they show is not a
 * library record. Entering one snapshots what was being studied and leaving it
 * restores that snapshot — this is the `savedContext` half of the state. The
 * snapshot starts with the mode it was taken in, and every exit resumes that
 * mode through returnToStudiedMode(): a match or a collection is returned to,
 * never dropped to NORMAL (#406).
 * A snapshot holds positions only, never an analysis: on the way back the
 * analysis is fetched again through showPosition(), and while a scratch
 * mode is on, analysisStore still describes the position studied *before*
 * (see project memory « plateaux brouillons : analysisStore périmé »).
 *
 * Transitions (each is one exported function):
 *
 *   enterEditMode       NORMAL | MATCH | COLLECTION | EVAL → EDIT
 *   exitEditMode        EDIT → the mode it was entered from (MATCH | COLLECTION | NORMAL)
 *   enterEvalMode       NORMAL | MATCH | COLLECTION | EDIT → EVAL
 *   exitEvalMode        EVAL → the mode it was entered from (MATCH | COLLECTION | NORMAL)
 *   toggleEvalMode      EVAL → (exit + analysis tab) | * → Eval tab
 *   enterTranscribeMode NORMAL | MATCH | COLLECTION | EDIT | EVAL → TRANSCRIBE
 *   exitTranscribeMode  TRANSCRIBE → the mode it was entered from (MATCH | COLLECTION | NORMAL)
 *   sendPositionToEval  * → EVAL on a given position (id cleared)
 *   toggleMatchMode     MATCH → NORMAL | * → MATCH
 *   handleOpenCollection * → COLLECTION
 *   exitCollectionMode  COLLECTION → NORMAL
 *
 * The scratch modes are reached from the tab bar: App.svelte's tab effect
 * calls enterEditMode/exitEditMode and enterEvalMode/exitEvalMode when the
 * active tab changes, and it runs the *exit* of the previous tab's mode
 * before the *entry* of the new one. The machine still copes with a direct
 * EDIT ↔ EVAL call by leaving the current scratch mode first, so that the
 * saved context of one scratch mode is never buried under the other's.
 *
 * positionService.js re-exports every transition, so callers keep importing
 * them from there; the two modules are a deliberate import cycle
 * (positionService owns the loaders — showPosition, loadAllPositions — and
 * the machine owns the transitions that call them). Nothing here runs at
 * module evaluation, which is what makes the cycle safe.
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
 * transition and consumed (nulled) by the matching exit:
 *
 *   beforeEval  { mode, matchContext, position, positionIndex, ids }
 *               written by enterEvalMode, consumed by exitEvalMode. `mode` and
 *               `matchContext` let the exit return to the studied match instead
 *               of dropping to NORMAL while matchContext still says a match is
 *               on (bug 2) — that left match navigation broken.
 *   beforeEdit  { mode, matchContext }: the mode EDIT was entered from. Written
 *               on *every* enterEditMode so a stale snapshot from an earlier
 *               match-entered EDIT can never be restored into a later
 *               NORMAL-entered one; consumed by exitEditMode (bug 2 again:
 *               leaving the search tab used to reload the whole library and
 *               bounce the user to the Matches tab). It holds no list: the one
 *               behind the query board — library, match, collection — stays in
 *               positionsStore.
 *   beforeTranscribe  the same photograph, taken by enterTranscribeMode and
 *               consumed by exitTranscribeMode. It is a slot of its own and not
 *               a second use of beforeEval: the two panels can be visited one
 *               after the other, and one snapshot buried under the other is the
 *               bug the EDIT/EVAL pair already had.
 *   evalSeed    the position the Eval panel must open on instead of its
 *               default bearoff. A hand-off between sendPositionToEval() and
 *               enterEvalMode(), which run one tick apart (the tab switch reaches
 *               enterEvalMode through App.svelte's tab effect, not a direct
 *               call) — hence a slot rather than a parameter.
 *   lastEvalBoard the board the Eval panel was last left on, photographed by
 *               exitEvalMode and reused by the next enterEvalMode. Unlike the
 *               slots above it is NOT consumed on use: leaving and returning
 *               to the panel used to hand back the default bearoff, throwing
 *               away whatever the user had built. It is a board, never a
 *               library record — id is forced to 0 and no analysis travels
 *               with it (the panel evaluates live, and a stale analysis
 *               describing another position is exactly the bug the scratch
 *               boards had). It outlives a library reload on purpose —
 *               forgetContextBeforeEval drops beforeEval, not this: a scratch
 *               board belongs to the session, not to the open database.
 */
/**
 * The slots are all `null` at rest, so without this annotation the checker
 * infers the type `null` for each and rejects every assignment to them.
 *
 * @type {{beforeTranscribe: any, beforeEval: any, beforeEdit: any, evalSeed: any, lastEvalBoard: any}}
 */
const savedContext = {
    beforeTranscribe: null,
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
 * Forget what enterEvalMode saved. loadAllPositions() calls this: reloading
 * the whole library redefines what "the position before EVAL" is, so a
 * later exitEvalMode reloads too instead of restoring a stale list.
 */
export function forgetContextBeforeEval() {
    savedContext.beforeEval = null;
}

function currentMode() {
    return get(statusBarModeStore);
}

/**
 * The mode half of a scratch mode's snapshot: the mode it was entered from,
 * and the match context that mode needs to be resumed. Every entry into a
 * scratch mode takes one (beforeEdit, beforeEval, beforeTranscribe) and every
 * exit hands it to returnToStudiedMode() — one rule for the three panels, so
 * that a list the user was studying cannot be resumed by one exit and dropped
 * by another (#406: Search left the collection on the way in, and Eval came
 * back to its ids in NORMAL mode).
 */
function photographStudiedMode() {
    return { mode: currentMode(), matchContext: { ...get(matchContextStore) } };
}

/**
 * Put back the mode a scratch mode was entered from: the match (its context
 * restored), the collection, or the library. A collection is resumed only
 * while it is still the active one — a gesture that dropped it (a search, a
 * match opened) has already replaced the list behind the board.
 *
 * The caller restores the position; the mode comes first, so that no effect
 * sees the studied position under a scratch mode.
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
 * A position just saved from a scratch board (scratchBoard.js) joins the list
 * the board's exit will put back — and only when that list is the whole
 * library. Any other list is what the user was studying (a match, a search
 * result, a collection, a deck, a statistics selection): leaving the panel
 * must find it exactly as it was, so it is left alone.
 *
 * The list behind the board is the machine's to know. In EDIT it is still in
 * positionsStore (enterEditMode keeps it there, exitEditMode redraws from it);
 * in EVAL it is the id snapshot beforeEval holds. Both snapshots carry the
 * mode they were taken in, and only a board entered from NORMAL can have the
 * library behind it: a collection holding every position, in library order,
 * is still a collection (#406).
 *
 * "The whole library" is checked on the ids rather than inferred from flags:
 * a deck or a statistics selection is shown in NORMAL mode with no search
 * active. The list is the library when it plus the new id is exactly what
 * ListPositionIDs answers. The index is never touched: the id goes last.
 *
 * @param {number} id the position just written (a new one: a known position
 *   is already in the library)
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
        // Leave the scratch board of the Eval tab first — through the exit
        // transition, not toggleEvalMode(): the toggle also flips the active tab
        // to 'analysis', which bounced a user who had just clicked the search
        // tab. The exit's synchronous prefix restores the mode (NORMAL or MATCH)
        // and the studied position before we snapshot them below.
        exitEvalMode();
    }

    // Snapshot what is studied — the library, a match or a collection — so that
    // leaving the search tab returns to it.
    savedContext.beforeEdit = photographStudiedMode();

    if (currentMode() === MODE.MATCH) {
        logger.log('Exiting MATCH mode to enter EDIT');
        await persistLastVisitedMatchPosition();
        matchContextStore.set({ ...NO_MATCH_CONTEXT });
        // Deliberately NOT loadAllPositions() here (bug 2): it runs async without
        // await and, on resolving, sets mode NORMAL and flips activeTab to
        // 'matches' — racing this function and bouncing the user off the search
        // tab (the studied match position was lost). EDIT clears the board below
        // to build a query and positionsStore isn't consulted in EDIT, so there is
        // nothing to load; exitEditMode restores the snapshot taken above.
    }

    // A collection is NOT left (#406). exitCollectionMode() here closed its
    // panel, emptied its stores and reloaded the whole library, so the way out
    // of the search tab could only return to the library. As for a match, what
    // was studied stays behind the query board — the collection's ids in
    // positionsStore — and exitEditMode resumes the mode from the snapshot.

    if (currentMode() !== MODE.EDIT) {
        statusBarModeStore.set(MODE.EDIT);
        // Clear the selected analysis move so its move arrows are erased when
        // leaving a match/analysis position for the search tab. The board only
        // auto-clears the selection on a position-ID change, and here the id is
        // unchanged (the blank board keeps the studied position's id), so the
        // arrows would otherwise persist over the empty EDIT board.
        selectedMoveStore.set(null);
        // A copy is blanked, never the object on the board: handleOpenCollection
        // puts the collection's own record there, which is also the list's
        // cached entry, and blanking it in place emptied the very record
        // exitEditMode puts back (#406) — what #201 was for the library.
        positionStore.update((pos) => blankEditBoard(JSON.parse(JSON.stringify(pos))));
    }
}

/** EDIT → the mode it was entered from: MATCH | COLLECTION | NORMAL. */
export async function exitEditMode() {
    if (currentMode() !== MODE.EDIT) return;

    const saved = savedContext.beforeEdit;
    savedContext.beforeEdit = null;
    // Entered from a match: return to the studied move rather than dropping
    // into the flat "all positions" list (bug 2).
    if (returnToStudiedMode(saved) === MODE.MATCH) {
        const snap = saved.matchContext;
        const movePos = snap.movePositions?.[snap.currentIndex];
        if (movePos) {
            await showPosition(movePos.position);
            statusBarTextStore.set(`${snap.player1Name} vs ${snap.player2Name}`);
        }
        return;
    }
    // NORMAL or COLLECTION: the list is the one positionsStore still holds.
    // Put the studied position back on the board synchronously, from the
    // window cache, before bumping the index. The redraw the bump triggers
    // (App.svelte's nav effect) fetches asynchronously, and whoever runs right
    // after this exit — App.svelte calls it without await and then
    // enterEvalMode, which photographs the board — would otherwise see the
    // blank query board under the library's id, and put it back on screen on
    // the way out of Eval (#201). enterEditMode blanked a clone, so the cache
    // still holds the record intact; on a miss the nav effect fetches it.
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
 * Open the Eval panel on `position` instead of its default bearoff — the
 * "study THIS position" gesture, reached from the board's context menu or
 * from a Ctrl-C/Ctrl-V round trip.
 *
 * The copy is detached and its id cleared: the Eval board is a scratch pad,
 * and a position carrying a database id there would let a later Ctrl-U write
 * an edited board back over the record it came from. Everything else travels
 * as-is, including player_on_roll: the Eval panel reads the on-roll side for
 * its own facts table and gammonNet evaluates from it, so a mirrored match
 * position keeps both its orientation on screen and its meaning to the engine.
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
    // Normally the tab switch reaches enterEvalMode() through App.svelte's tab
    // effect. If the Eval tab is somehow already selected without EVAL mode
    // being on, that set() is a no-op and the effect never re-runs, so enter
    // directly rather than leave the seed stranded.
    if (get(activeTabStore) === 'eval') enterEvalMode();
    else activeTabStore.set('eval');
}

/**
 * NORMAL | MATCH | COLLECTION | EDIT → EVAL. The mode is set to EVAL *before*
 * the scratch board lands in positionStore, in one synchronous run, or the
 * board's Eval effect (updateEPC in App.svelte) fires on the wrong position. The function is async only
 * for the way in from EDIT, and that await sits before the run, never inside.
 */
export async function enterEvalMode() {
    if (currentMode() === MODE.EVAL) return;

    if (currentMode() === MODE.TRANSCRIBE) {
        await exitTranscribeMode();
        if (currentMode() === MODE.EVAL) return;
    }

    if (currentMode() === MODE.EDIT) {
        // Leave the search tab's scratch board first, so the snapshot below is
        // the studied position (or match), not the blank query board. Awaited:
        // enterEditMode blanks the board under the library's id, and only a
        // completed exit guarantees the record is back (#201) — see
        // exitEditMode for why its own restore is synchronous.
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
 * EVAL → the mode it was entered from: MATCH | COLLECTION | NORMAL. The mode is restored
 * synchronously, before the studied position is put back, so the board's
 * Eval effect never sees the restored position under EVAL mode.
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
        // Reload through showPosition (not a bare positionStore.set) so the
        // analysis is fetched again and the analysis panel is repopulated on
        // return. In MATCH mode the nav effect no longer redraws (bug 1
        // guard), so without this the panel stayed empty after EVAL; in
        // NORMAL mode this simply mirrors the index-driven redraw.
        await showPosition(saved.position);
    }
}

// ── TRANSCRIBE (Transcription tab) ───────────────────────────────────────────

/**
 * NORMAL | MATCH | COLLECTION | EDIT | EVAL → TRANSCRIBE.
 *
 * The transcription panel takes the board over: it shows the Action the Cursor
 * is on, which is a draft's board and never a library record. So entering
 * photographs what was being studied, exactly as the Eval tab does, and
 * leaving puts it back — a user who steps into the panel to look at a draft
 * must come back to the position, or the match, they were on.
 *
 * The board itself is not replaced here. Until a draft is opened there is no
 * Cursor to show, and blanking the board would lose the studied position for
 * nothing; the panel puts the Cursor's position on it when one exists.
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
        // than land on the last position of the library (#201).
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

        // Mode first, then the position through showPosition: with the match
        // context and mode already set it reads the played move from the match
        // and hides the cube analysis on a game's opening position, exactly as
        // firstPosition/nextPosition do when navigating the match.
        statusBarModeStore.set(MODE.MATCH);
        const startMovePos = movePositions[startIndex];
        await showPosition(startMovePos.position);
        selectedMoveStore.set(null);
        // Player names are shown in the match-info header bar above the board
        // (MatchInfoBar.svelte); no longer echoed in the status bar.

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
 * Any mode → COLLECTION, browsing `collectionPositions`.
 *
 * @param {any} collection
 * @param {any[]} collectionPositions
 */
export function handleOpenCollection(collection, collectionPositions) {
    if (!collectionPositions || collectionPositions.length === 0) {
        statusBarTextStore.set(tMsg('commands.collectionEmpty'));
        return;
    }

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
