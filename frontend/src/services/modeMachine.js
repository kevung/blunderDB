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
 *   EPC         the Eval tab: the board is a scratch pad for the engine
 *
 * EDIT and EPC are *scratch* modes: the board they show is not a library
 * record. Entering one snapshots what was being studied and leaving it
 * restores that snapshot — this is the `savedContext` half of the state.
 * A snapshot holds positions only, never an analysis: on the way back the
 * analysis is fetched again through showPosition(), and while a scratch
 * mode is on, analysisStore still describes the position studied *before*
 * (see project memory « plateaux brouillons : analysisStore périmé »).
 *
 * Transitions (each is one exported function):
 *
 *   enterEditMode       NORMAL | MATCH | COLLECTION | EPC → EDIT
 *   exitEditMode        EDIT → MATCH (entered from a match) | NORMAL
 *   enterEPCMode        NORMAL | MATCH | COLLECTION | EDIT → EPC
 *   exitEPCMode         EPC → MATCH (entered from a match) | NORMAL
 *   toggleEPCMode       EPC → (exit + analysis tab) | * → Eval tab
 *   sendPositionToEval  * → EPC on a given position (id cleared)
 *   toggleMatchMode     MATCH → NORMAL | * → MATCH
 *   handleOpenCollection * → COLLECTION
 *   exitCollectionMode  COLLECTION → NORMAL
 *
 * The scratch modes are reached from the tab bar: App.svelte's tab effect
 * calls enterEditMode/exitEditMode and enterEPCMode/exitEPCMode when the
 * active tab changes, and it runs the *exit* of the previous tab's mode
 * before the *entry* of the new one. The machine still copes with a direct
 * EDIT ↔ EPC call by leaving the current scratch mode first, so that the
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
import { showPosition, loadAllPositions, loadAnalysisForPosition, setSearchState } from './positionService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

export const MODE = Object.freeze({
    NORMAL: 'NORMAL',
    MATCH: 'MATCH',
    COLLECTION: 'COLLECTION',
    EDIT: 'EDIT',
    EPC: 'EPC'
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
 *   beforeEPC   { mode, matchContext, position, positionIndex, positions }
 *               written by enterEPCMode, consumed by exitEPCMode. `mode` and
 *               `matchContext` let the exit return to the studied match instead
 *               of dropping to NORMAL while matchContext still says a match is
 *               on (bug 2) — that left match navigation broken.
 *   beforeEdit  the match context EDIT was entered from, or null. Written on
 *               *every* enterEditMode so a stale snapshot from an earlier
 *               match-entered EDIT can never be restored into a later
 *               NORMAL-entered one; consumed by exitEditMode (bug 2 again:
 *               leaving the search tab used to reload the whole library and
 *               bounce the user to the Matches tab).
 *   epcSeed     the position the Eval panel must open on instead of its
 *               default bearoff. A hand-off between sendPositionToEval() and
 *               enterEPCMode(), which run one tick apart (the tab switch reaches
 *               enterEPCMode through App.svelte's tab effect, not a direct
 *               call) — hence a slot rather than a parameter.
 */
const savedContext = {
    beforeEPC: null,
    beforeEdit: null,
    epcSeed: null
};

/** Read-only snapshot of the machine's state, for tests and debugging. */
export function modeState() {
    return { mode: get(statusBarModeStore), savedContext: { ...savedContext } };
}

/**
 * Forget what enterEPCMode saved. loadAllPositions() calls this: reloading
 * the whole library redefines what "the position before EPC" is, so a
 * later exitEPCMode reloads too instead of restoring a stale list.
 */
export function forgetContextBeforeEPC() {
    savedContext.beforeEPC = null;
}

function currentMode() {
    return get(statusBarModeStore);
}

function inMatch() {
    return currentMode() === MODE.MATCH && get(matchContextStore).isMatchMode;
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

// ── EDIT ─────────────────────────────────────────────────────────────────────

/** NORMAL | MATCH | COLLECTION | EPC → EDIT. */
export async function enterEditMode() {
    logger.log('enterEditMode');
    if (!get(databasePathStore)) return;

    if (currentMode() === MODE.EPC) {
        // Leave the scratch board of the Eval tab first — through the exit
        // transition, not toggleEPCMode(): the toggle also flips the active tab
        // to 'analysis', which bounced a user who had just clicked the search
        // tab. The exit's synchronous prefix restores the mode (NORMAL or MATCH)
        // and the studied position before we snapshot them below.
        exitEPCMode();
    }

    // Snapshot the studied match (if any) so leaving the search tab restores it.
    savedContext.beforeEdit = inMatch() ? { ...get(matchContextStore) } : null;

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

    if (currentMode() === MODE.COLLECTION) {
        await exitCollectionMode();
    }

    if (currentMode() !== MODE.EDIT) {
        statusBarModeStore.set(MODE.EDIT);
        closePanel(PANEL.COMMENT);
        closePanel(PANEL.ANALYSIS);
        // Clear the selected analysis move so its move arrows are erased when
        // leaving a match/analysis position for the search tab. The board only
        // auto-clears the selection on a position-ID change, and here the id is
        // unchanged (we blank the same position object below), so the arrows
        // would otherwise persist over the empty EDIT board.
        selectedMoveStore.set(null);
        positionStore.update(blankEditBoard);
    }
}

/** EDIT → MATCH (if entered from a match) | NORMAL. */
export async function exitEditMode() {
    if (currentMode() !== MODE.EDIT) return;

    // If we entered search from a match, return to that studied position
    // rather than dropping into the flat "all positions" list (bug 2).
    const snap = savedContext.beforeEdit;
    savedContext.beforeEdit = null;
    if (snap && snap.isMatchMode) {
        matchContextStore.set(snap);
        statusBarModeStore.set(MODE.MATCH);
        const movePos = snap.movePositions?.[snap.currentIndex];
        if (movePos) {
            await showPosition(movePos.position);
            statusBarTextStore.set(`${snap.player1Name} vs ${snap.player2Name}`);
        }
        return;
    }
    statusBarModeStore.set(MODE.NORMAL);
    // Put the library position back on the board synchronously, from the
    // window cache, before bumping the index. The redraw the bump triggers
    // (App.svelte's nav effect) fetches asynchronously, and whoever runs right
    // after this exit — App.svelte calls it without await and then
    // enterEPCMode, which photographs the board — would otherwise see the
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

// ── EPC (Eval tab) ───────────────────────────────────────────────────────────

/** EPC → exit and show the analysis tab; otherwise open the Eval tab. */
export function toggleEPCMode() {
    if (currentMode() === MODE.EPC) {
        exitEPCMode();
        activeTabStore.set('analysis');
    } else {
        activeTabStore.set('epc');
    }
}

// The board the Eval panel opens on when nothing was handed to it: the
// canonical 6-point bearoff the EPC trainer starts from.
function defaultEPCPosition() {
    const epcPoints = Array(26).fill({ checkers: 0, color: -1 });
    epcPoints[1] = { checkers: 2, color: 0 };
    epcPoints[2] = { checkers: 2, color: 0 };
    epcPoints[3] = { checkers: 2, color: 0 };
    epcPoints[4] = { checkers: 3, color: 0 };
    epcPoints[5] = { checkers: 3, color: 0 };
    epcPoints[6] = { checkers: 3, color: 0 };

    return {
        id: 0,
        board: { points: epcPoints, bearoff: [0, 15] },
        cube: { owner: -1, value: 0 },
        dice: [0, 0],
        score: [-1, -1],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
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
 */
export function sendPositionToEval(position) {
    if (!position) return;
    const seed = JSON.parse(JSON.stringify(position));
    seed.id = 0;

    if (currentMode() === MODE.EPC) {
        // Already in the Eval panel: replace the board in place. Going through
        // the tab store would be a no-op and enterEPCMode() returns early.
        positionsStore.set([seed]);
        positionStore.set(seed);
        currentPositionIndexStore.set(0);
        return;
    }

    savedContext.epcSeed = seed;
    // Normally the tab switch reaches enterEPCMode() through App.svelte's tab
    // effect. If the Eval tab is somehow already selected without EPC mode
    // being on, that set() is a no-op and the effect never re-runs, so enter
    // directly rather than leave the seed stranded.
    if (get(activeTabStore) === 'epc') enterEPCMode();
    else activeTabStore.set('epc');
}

/**
 * NORMAL | MATCH | COLLECTION | EDIT → EPC. The mode is set to EPC *before*
 * the scratch board lands in positionStore, in one synchronous run, or the
 * board's EPC effect fires on the wrong position. The function is async only
 * for the way in from EDIT, and that await sits before the run, never inside.
 */
export async function enterEPCMode() {
    if (currentMode() === MODE.EPC) return;

    if (currentMode() === MODE.EDIT) {
        // Leave the search tab's scratch board first, so the snapshot below is
        // the studied position (or match), not the blank query board. Awaited:
        // enterEditMode blanks the board under the library's id, and only a
        // completed exit guarantees the record is back (#201) — see
        // exitEditMode for why its own restore is synchronous.
        await exitEditMode();
        if (currentMode() === MODE.EPC) return;
    }

    savedContext.beforeEPC = {
        mode: currentMode(),
        matchContext: { ...get(matchContextStore) },
        position: get(positionStore) ? { ...get(positionStore) } : null,
        positionIndex: get(currentPositionIndexStore),
        ids: get(positionsStore)?.ids ?? null
    };

    const epcPosition = savedContext.epcSeed ?? defaultEPCPosition();
    savedContext.epcSeed = null;

    statusBarModeStore.set(MODE.EPC);
    closePanel(PANEL.COMMENT);
    closePanel(PANEL.ANALYSIS);

    positionsStore.set([epcPosition]);
    positionStore.set(epcPosition);
    currentPositionIndexStore.set(0);
}

/**
 * EPC → MATCH (if entered from a match) | NORMAL. The mode is restored
 * synchronously, before the studied position is put back, so the board's
 * EPC effect never sees the restored position under an EPC mode.
 */
export async function exitEPCMode() {
    if (currentMode() !== MODE.EPC) return;

    const saved = savedContext.beforeEPC;
    savedContext.beforeEPC = null;

    statusBarTextStore.set('');
    epcDataStore.set({ bottomEPC: null, topEPC: null, race: null, error: null });

    const returnToMatch = saved?.mode === MODE.MATCH && saved.matchContext?.isMatchMode;
    if (returnToMatch) {
        matchContextStore.set(saved.matchContext);
        statusBarModeStore.set(MODE.MATCH);
        statusBarTextStore.set(`${saved.matchContext.player1Name} vs ${saved.matchContext.player2Name}`);
    } else {
        statusBarModeStore.set(MODE.NORMAL);
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
        // guard), so without this the panel stayed empty after EPC; in
        // NORMAL mode this simply mirrors the index-driven redraw.
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
        savedContext.beforeEPC = null;
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

/** Any mode → COLLECTION, browsing `collectionPositions`. */
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
