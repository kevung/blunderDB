import { get } from 'svelte/store';
import {
    ListPositionIDs,
    DeletePosition,
    DeleteAnalysis,
    UpdatePosition,
    SaveAnalysis,
    LoadAnalysis,
    LoadPositionIDsByFilters,
    ComputeEPCFromPosition,
    SaveLastVisitedPosition,
    SaveEditPosition,
    SaveExcludePosition,
    SaveFilter,
    LoadComment
} from '../../wailsjs/go/database/Database.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, positionsStore, matchContextStore } from '../stores/positionStore.js';
import { searchExcludePositionStore, emptySearchBoardPosition, boardHasCheckers } from '../stores/searchExcludePositionStore.js';
import { analysisStore } from '../stores/analysisStore.js';
import { epcDataStore, resetEpcReveal } from '../stores/epcStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';
import { viewStore } from '../stores/viewStore.js';
import { currentPositionIndexStore, statusBarTextStore, statusBarModeStore, commentTextStore, activeTabStore } from '../stores/uiStore.js';
import { activeCollectionStore } from '../stores/collectionStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { confirmAction } from './confirmService.js';
import { logger } from '../utils/logger.js';
import { forgetContextBeforeEPC } from './modeMachine.js';
// Ctrl-G status line (keyboardService imports it from here).
export { showDatesAndMetadata } from './metadataStatus.js';

// The mode automaton (NORMAL / MATCH / COLLECTION / EDIT / EPC) lives in
// modeMachine.js; its transitions stay reachable from here so that callers
// keep one import for everything position-related.
export { enterEditMode, exitEditMode, toggleEPCMode, sendPositionToEval, enterEPCMode, exitEPCMode, toggleMatchMode, handleOpenCollection, exitCollectionMode } from './modeMachine.js';
// NOTE: these UI messages are translated at emission time via the non-reactive
// `translate` helper; already-displayed messages do not retranslate on language change.
import { tMsg, t } from '../i18n';

// The cube block an analysis carries when nothing was evaluated: every
// figure at zero, so the panels render their blank cells.
function emptyDoublingCubeAnalysis() {
    return {
        analysisDepth: '',
        playerWinChances: 0,
        playerGammonChances: 0,
        playerBackgammonChances: 0,
        opponentWinChances: 0,
        opponentGammonChances: 0,
        opponentBackgammonChances: 0,
        cubelessNoDoubleEquity: 0,
        cubelessDoubleEquity: 0,
        cubefulNoDoubleEquity: 0,
        cubefulNoDoubleError: 0,
        cubefulDoubleTakeEquity: 0,
        cubefulDoubleTakeError: 0,
        cubefulDoublePassEquity: 0,
        cubefulDoublePassError: 0,
        bestCubeAction: '',
        wrongPassPercentage: 0,
        wrongTakePercentage: 0
    };
}

// Session/search tracking state
let lastSearchCommand = '';
let lastSearchPosition = null;
let hasActiveSearch = false;

export function getSearchState() {
    return { lastSearchCommand, lastSearchPosition, hasActiveSearch };
}

export function setSearchState(cmdOrObj, pos, active) {
    if (cmdOrObj !== null && typeof cmdOrObj === 'object' && 'lastSearchCommand' in cmdOrObj) {
        lastSearchCommand = cmdOrObj.lastSearchCommand;
        lastSearchPosition = cmdOrObj.lastSearchPosition;
        hasActiveSearch = cmdOrObj.hasActiveSearch;
    } else {
        lastSearchCommand = cmdOrObj;
        lastSearchPosition = pos;
        hasActiveSearch = active;
    }
}

// generateXGID lives in xgid.js — re-exported so existing callers keep one import,
// and imported here too since this module still calls it directly.
export { generateXGID } from './xgid.js';
import { generateXGID } from './xgid.js';

export function isValidPosition(position) {
    const player1Checkers = position.board.points.reduce((acc, point) => acc + (point.color === 0 ? point.checkers : 0), 0);
    const player2Checkers = position.board.points.reduce((acc, point) => acc + (point.color === 1 ? point.checkers : 0), 0);

    if (player1Checkers > 15) {
        setStatusBarMessage(tMsg('status.invalidP1Over15'));
        return false;
    }
    if (player2Checkers > 15) {
        setStatusBarMessage(tMsg('status.invalidP2Over15'));
        return false;
    }
    if (player1Checkers === 0) {
        setStatusBarMessage(tMsg('status.invalidP1BorneOff'));
        return false;
    }
    if (player2Checkers === 0) {
        setStatusBarMessage(tMsg('status.invalidP2BorneOff'));
        return false;
    }

    if (position.decision_type === 1) {
        if (position.cube.owner !== position.player_on_roll && position.cube.owner !== -1) {
            setStatusBarMessage(tMsg('status.invalidCubeUnavailable'));
            return false;
        }
        if (position.score[position.player_on_roll] === 1) {
            setStatusBarMessage(tMsg('status.invalidCrawford'));
            return false;
        }
    }

    if ((position.score[0] === -1 && position.score[1] !== -1) || (position.score[1] === -1 && position.score[0] !== -1)) {
        setStatusBarMessage(tMsg('status.invalidUnlimitedScore'));
        return false;
    }

    return true;
}

export function mirrorPositionForSearch(pos) {
    const mirrored = JSON.parse(JSON.stringify(pos));

    const tempPoints = [...mirrored.board.points];
    for (let i = 0; i < 26; i++) {
        // color 2 = "must be empty" exclusion marker: keep it through the mirror.
        const c = tempPoints[i].color;
        mirrored.board.points[25 - i] = {
            color: c === -1 || c === 2 ? c : 1 - c,
            checkers: tempPoints[i].checkers
        };
    }

    [mirrored.board.bearoff[0], mirrored.board.bearoff[1]] = [mirrored.board.bearoff[1], mirrored.board.bearoff[0]];
    mirrored.player_on_roll = 1 - mirrored.player_on_roll;
    [mirrored.score[0], mirrored.score[1]] = [mirrored.score[1], mirrored.score[0]];
    if (mirrored.cube.owner !== -1) {
        mirrored.cube.owner = 1 - mirrored.cube.owner;
    }

    return mirrored;
}

export async function showPosition(position) {
    if (!position) {
        logger.error('Invalid position:', position);
        return;
    }

    // JSON round-trip, not structuredClone: a position reaching here in MATCH
    // mode comes out of a Svelte 5 reactive proxy, and structuredClone throws
    // DataCloneError on a Proxy ("#<Object> could not be cloned"). The throw
    // landed BEFORE the LoadAnalysis below, so browsing a match advanced the
    // status bar — its move counter reads the match context, updated by the
    // caller — while the board and the analysis stayed on the previous move
    // (D.8, #208, fixed 2026-09-04; e2e match-navigation caught it). A
    // position is plain JSON off the Wails bridge, so the round trip is exact.
    const positionCopy = JSON.parse(JSON.stringify(position));
    positionStore.set(positionCopy);

    // The analysis and the comment are two independent round trips over the
    // Wails bridge; running them concurrently instead of one after the other
    // halves the latency a browsing step pays for them (D.8, #208). Each call
    // is deferred into a .then() so a binding that throws synchronously (as
    // an unmocked one does in a few tests) becomes a rejection allSettled
    // catches, rather than an exception that skips the settle entirely.
    const [analysisResult, commentResult] = await Promise.allSettled([Promise.resolve().then(() => LoadAnalysis(position.id)), Promise.resolve().then(() => LoadComment(position.id))]);
    const analysis = analysisResult.status === 'fulfilled' ? analysisResult.value : null;
    const comment = commentResult.status === 'fulfilled' ? commentResult.value : '';

    const matchCtx = get(matchContextStore);
    const inMatchMode = get(statusBarModeStore) === 'MATCH' && matchCtx.isMatchMode;
    const isFirstPositionOfGame =
        inMatchMode && matchCtx.movePositions.length > 0 && (matchCtx.movePositions[matchCtx.currentIndex]?.move_number === 0 || matchCtx.movePositions[matchCtx.currentIndex]?.move_number === 1);

    let currentPlayedMove = '';
    let currentPlayedCubeAction = '';
    let allPlayedMoves = analysis?.playedMoves || [];
    let allPlayedCubeActions = analysis?.playedCubeActions || [];

    if (inMatchMode && matchCtx.movePositions.length > 0) {
        const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
        if (currentMovePos) {
            currentPlayedMove = currentMovePos.checker_move || '';
            currentPlayedCubeAction = currentMovePos.cube_action || '';
        }
    } else {
        currentPlayedMove = analysis?.playedMove || '';
        currentPlayedCubeAction = analysis?.playedCubeAction || '';
    }

    analysisStore.set({
        positionId: analysis?.positionId || null,
        xgid: analysis?.xgid || '',
        player1: analysis?.player1 || '',
        player2: analysis?.player2 || '',
        analysisType: analysis?.analysisType || '',
        analysisEngineVersion: analysis?.analysisEngineVersion || '',
        checkerAnalysis: analysis?.checkerAnalysis || { moves: [] },
        doublingCubeAnalysis: isFirstPositionOfGame ? null : analysis?.doublingCubeAnalysis || emptyDoublingCubeAnalysis(),
        allCubeAnalyses: isFirstPositionOfGame ? [] : analysis?.allCubeAnalyses || [],
        playedMove: currentPlayedMove,
        playedCubeAction: isFirstPositionOfGame ? '' : currentPlayedCubeAction,
        playedMoves: allPlayedMoves,
        playedCubeActions: isFirstPositionOfGame ? [] : allPlayedCubeActions,
        creationDate: analysis?.creationDate || '',
        lastModifiedDate: analysis?.lastModifiedDate || ''
    });

    commentTextStore.set(comment || '');
}

export async function loadAnalysisForPosition(position) {
    if (!position || !position.id) return;

    try {
        const analysis = await LoadAnalysis(position.id);
        if (analysis) {
            analysisStore.set(analysis);
        } else {
            analysisStore.set({
                positionId: position.id,
                xgid: '',
                player1: '',
                player2: '',
                analysisType: '',
                analysisEngineVersion: '',
                checkerAnalysis: { moves: [] },
                doublingCubeAnalysis: null,
                allCubeAnalyses: [],
                playedMove: '',
                playedCubeAction: '',
                playedMoves: [],
                playedCubeActions: [],
                creationDate: '',
                lastModifiedDate: ''
            });
        }
    } catch (error) {
        logger.error('Error loading analysis:', error);
    }
}

/**
 * Reload the whole library and leave every other mode. Lands on `focusId`
 * when the library holds it — the position the user is leaving, so that
 * exiting a match keeps the studied position on the board (#201) — and on
 * the last position otherwise, the historical default of a fresh list.
 *
 * @param {{ focusId?: number | null }} [options]
 */
export async function loadAllPositions({ focusId = null } = {}) {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    try {
        // Ids only: the positions are fetched by window as the user browses
        // (positionList.js). A library reload may follow an edit, so the
        // window cache is dropped with the list.
        const ids = (await ListPositionIDs()) || [];

        if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode && get(matchContextStore).matchID) {
            SaveLastVisitedPosition(get(matchContextStore).matchID, get(matchContextStore).currentIndex).catch((e) => {
                logger.error('Error persisting last visited position:', e);
            });
        }
        statusBarModeStore.set('NORMAL');
        matchContextStore.set({
            isMatchMode: false,
            matchID: null,
            movePositions: [],
            currentIndex: 0,
            player1Name: '',
            player2Name: ''
        });
        forgetContextBeforeEPC();
        activeCollectionStore.set(null);

        positionsStore.setIds(ids, { reset: true });
        if (ids.length > 0) {
            const focusIdx = focusId == null ? -1 : ids.indexOf(focusId);
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(focusIdx >= 0 ? focusIdx : ids.length - 1);
            activeTabStore.set('matches');

            hasActiveSearch = false;
            lastSearchCommand = '';
            lastSearchPosition = null;
            lastSearchStore.set(null);
            const { saveSessionState } = await import('./sessionService.js');
            saveSessionState();
        } else {
            currentPositionIndexStore.set(-1);
            setStatusBarMessage(tMsg('commands.noPositionsFound'));
            logger.log('No positions found.');
        }
    } catch (error) {
        logger.error('Error loading all positions:', error);
        setStatusBarMessage(tMsg('status.errorLoadingAllPositions'));
    }
}

// Explicit user reload of the whole library (Ctrl+R, the toolbar reload button,
// the `e` command). loadAllPositions on its own lands on the Matches tab; an
// explicit reload is a "study" action, so surface the analysis panel for the
// shown position instead. Guard on an open DB with positions so an empty reload
// stays on the neutral state.
export async function reloadAllPositions() {
    await loadAllPositions();
    if (get(databasePathStore) && get(positionsStore).length > 0) {
        activeTabStore.set('analysis');
    }
}

// loadPositionsByFilters takes one options object rather than the ~38 positional
// arguments it used to. The list had grown long enough that adding a filter meant
// inserting an argument at the same index in six call sites, and getting an index
// wrong shifts every later filter silently: the search still runs, it just answers
// a different question.
export async function loadPositionsByFilters({
    filters = [],
    includeCube = false,
    includeScore = false,
    pipCountFilter = '',
    winRateFilter = '',
    gammonRateFilter = '',
    backgammonRateFilter = '',
    player2WinRateFilter = '',
    player2GammonRateFilter = '',
    player2BackgammonRateFilter = '',
    player1CheckerOffFilter = '',
    player2CheckerOffFilter = '',
    player1BackCheckerFilter = '',
    player2BackCheckerFilter = '',
    player1CheckerInZoneFilter = '',
    player2CheckerInZoneFilter = '',
    searchText = '',
    player1AbsolutePipCountFilter = '',
    equityFilter = '',
    decisionTypeFilter = false,
    diceRollFilter = false,
    movePatternFilter = '',
    dateFilter = '',
    player1OutfieldBlotFilter = '',
    player2OutfieldBlotFilter = '',
    player1JanBlotFilter = '',
    player2JanBlotFilter = '',
    noContactFilter = false,
    mirrorPositionFilter = false,
    individuallyImportedFilter = false,
    flaggedFilter = false,
    moveErrorFilter = '',
    searchCommand = '',
    matchIDsFilter = '',
    tournamentIDsFilter = '',
    restrictToPositionIDs = '',
    openInNewTab = false,
    diceRollMode = 'both',
    exceptDiceFilter = '',
    positionIDsFilter = '',
    playerFilter = ''
} = {}) {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    // Feedback for the query itself, which can take a moment on a large database. Set before
    // the backend call so the user sees it immediately, not after the fact.
    const previousStatusMessage = get(statusBarTextStore);
    setStatusBarMessage(tMsg('status.searching'));
    document.body.style.cursor = 'wait';

    try {
        let currentPosition = get(positionStore);

        // The exclude ("Sauf") structure must use the same mirror orientation as
        // the include structure so its points/colors stay aligned with stored
        // positions. The mirror decision is driven by the include board.
        const applyMirror = currentPosition.player_on_roll === 1;

        if (applyMirror) {
            currentPosition = mirrorPositionForSearch(currentPosition);
        }

        currentPosition = {
            ...currentPosition,
            has_jacoby: currentPosition.has_jacoby ? 1 : 0,
            has_beaver: currentPosition.has_beaver ? 1 : 0,
            decision_type: typeof currentPosition.decision_type === 'string' ? (currentPosition.decision_type ? 1 : 0) : currentPosition.decision_type || 0
        };

        let excludePosition = get(searchExcludePositionStore);
        if (boardHasCheckers(excludePosition)) {
            if (applyMirror) {
                excludePosition = mirrorPositionForSearch(excludePosition);
            }
            excludePosition = {
                ...excludePosition,
                has_jacoby: excludePosition.has_jacoby ? 1 : 0,
                has_beaver: excludePosition.has_beaver ? 1 : 0,
                decision_type: typeof excludePosition.decision_type === 'string' ? (excludePosition.decision_type ? 1 : 0) : excludePosition.decision_type || 0
            };
        } else {
            // Empty board → ignored by the backend (hasBoardFilter); send a clean
            // empty position rather than undefined.
            excludePosition = emptySearchBoardPosition();
        }

        const searchFilterPositionJSON = JSON.stringify(currentPosition);

        // Cube sub-type (only meaningful when the decision-type filter is a cube
        // decision): `dr` = take/pass responses, `dd` = double/no-double. Derived
        // from the filter tokens so both the panel and the command line share it.
        const cubeResponseFilter = Array.isArray(filters) ? (filters.includes('dr') ? 'takepass' : filters.includes('dd') ? 'double' : '') : '';

        // Comment presence: `co` = has a comment, `xco` = has none. Derived from
        // the tokens like cubeResponseFilter above, so the panel, the command
        // line and replayed history entries all go through one code path.
        const commentFilter = Array.isArray(filters) ? (filters.includes('xco') ? 'none' : filters.includes('co') ? 'has' : '') : '';

        // Only ids travel the Wails bridge here (D.8, #208): a search
        // returning thousands of rows used to ship every one of them whole
        // (megabytes of JSON) before the board ever showed more than one at a
        // time. positionsStore (positionList.js) keeps the id list and
        // fetches the window it is about to show through LoadPositionsByIDs —
        // the same lazy path the library already uses.
        const ids = await LoadPositionIDsByFilters({
            filter: currentPosition,
            excludeFilter: excludePosition,
            includeCube,
            includeScore,
            pipCountFilter,
            winRateFilter,
            gammonRateFilter,
            backgammonRateFilter,
            player2WinRateFilter,
            player2GammonRateFilter,
            player2BackgammonRateFilter,
            player1CheckerOffFilter,
            player2CheckerOffFilter,
            player1BackCheckerFilter,
            player2BackCheckerFilter,
            player1CheckerInZoneFilter,
            player2CheckerInZoneFilter,
            searchText,
            commentFilter,
            player1AbsolutePipCountFilter,
            equityFilter,
            decisionTypeFilter,
            cubeResponseFilter,
            diceRollFilter,
            diceRollMode,
            exceptDiceFilter,
            movePatternFilter,
            dateFilter,
            player1OutfieldBlotFilter,
            player2OutfieldBlotFilter,
            player1JanBlotFilter,
            player2JanBlotFilter,
            noContactFilter,
            mirrorFilter: mirrorPositionFilter,
            individuallyImportedFilter,
            flaggedFilter,
            moveErrorFilter,
            matchIDsFilter,
            tournamentIDsFilter,
            playerFilter,
            positionIDsFilter,
            restrictToPositionIDs
        });

        if (ids && ids.length > 0) {
            if (openInNewTab) {
                viewStore.addView();
            }

            statusBarModeStore.set('NORMAL');
            matchContextStore.set({
                isMatchMode: false,
                matchID: null,
                movePositions: [],
                currentIndex: 0,
                player1Name: '',
                player2Name: ''
            });
            activeCollectionStore.set(null);

            positionsStore.setIds(Array.isArray(ids) ? ids : []);

            if (get(currentPositionIndexStore) === 0) {
                currentPositionIndexStore.set(1);
            }
            currentPositionIndexStore.set(0);

            activeTabStore.set('analysis');

            hasActiveSearch = true;
            lastSearchCommand = searchCommand || '';
            lastSearchPosition = JSON.parse(searchFilterPositionJSON);
            lastSearchStore.set({ command: lastSearchCommand, position: searchFilterPositionJSON });

            const { saveSessionState } = await import('./sessionService.js');
            saveSessionState();
        } else {
            setStatusBarMessage(tMsg('status.noMatchingPositions'));
            if (get(activeTabStore) === 'search') {
                statusBarModeStore.set('EDIT');
            }
        }
    } catch (error) {
        logger.error('Error loading positions by filters:', error);
        setStatusBarMessage(tMsg('status.errorLoadingByFilters'));
        if (get(activeTabStore) === 'search') {
            statusBarModeStore.set('EDIT');
        }
    } finally {
        document.body.style.cursor = '';
        // The success path above sets no message of its own (the position count updates
        // separately); restore whatever was shown before the search only if nothing else
        // — the no-match or error branch — has already replaced the "searching" placeholder.
        const current = get(statusBarTextStore);
        if (current && typeof current === 'object' && current.i18nKey === 'status.searching') {
            statusBarTextStore.set(previousStatusMessage);
        }
    }
}

// Position navigation (first/previous/next/last/goto/random) lives in
// positionNavigation.js — re-exported so existing callers keep one import.
export { firstPosition, previousPosition, nextPosition, lastPosition, gotoPosition } from './positionNavigation.js';

export async function deletePosition() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    logger.log('deletePosition');

    const positions = get(positionsStore);
    if (!positions || positions.length === 0) {
        setStatusBarMessage(tMsg('status.noPositionsToDelete'));
        return;
    }

    if (!(await confirmAction(get(t)('status.confirmDeletePosition'), { confirmLabel: get(t)('common.delete') }))) return;

    try {
        const positionID = positionsStore.idAt(get(currentPositionIndexStore));
        await DeletePosition(positionID);
        logger.log('Position and associated analysis deleted with ID:', positionID);

        await loadAllPositions();
        setStatusBarMessage(tMsg('status.positionDeleted'));
    } catch (error) {
        logger.error('Error deleting position and associated analysis:', error);
        setStatusBarMessage(tMsg('status.errorDeletingPosition'));
    } finally {
        statusBarModeStore.set('NORMAL');
    }
}

// duplicatePositionId reads the id out of the backend's refusal when an edit
// turns a position into one that already exists (storage.DuplicatePositionError:
// "this position already exists (id N)"); null for any other error.
export function duplicatePositionId(error) {
    const m = /already exists \(id (\d+)\)/.exec(String(error));
    return m ? Number(m[1]) : null;
}

export async function updatePosition() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    if (get(statusBarModeStore) !== 'EDIT') {
        setStatusBarMessage(tMsg('status.updateOnlyEdit'));
        return;
    }
    logger.log('updatePosition');

    const positions = get(positionsStore);
    if (positions.length === 0) {
        setStatusBarMessage(tMsg('status.noPositionsToUpdate'));
        return;
    }

    const position = get(positionStore);
    const analysis = get(analysisStore);

    if (!isValidPosition(position)) return;

    try {
        const currentIndex = get(currentPositionIndexStore);
        const originalPosition = await positionsStore.getPosition(currentIndex);
        if (!originalPosition) {
            setStatusBarMessage(tMsg('status.noPositionsToUpdate'));
            return;
        }

        analysis.xgid = '';
        analysis.analysisType = '';
        analysis.checkerAnalysis = { moves: [] };
        analysis.doublingCubeAnalysis = emptyDoublingCubeAnalysis();
        analysis.analysisEngineVersion = '';

        if (Array.isArray(analysis.checkerAnalysis)) {
            analysis.checkerAnalysis = { moves: analysis.checkerAnalysis };
        }

        if (position.decision_type === 1) {
            position.dice = [0, 0];
        }

        const positionID = originalPosition.id;
        const positionJSON = JSON.stringify(position);
        const originalPositionJSON = JSON.stringify(originalPosition);

        // The position row goes first: if the edit is refused (it has become
        // a position that already exists), nothing else must have changed —
        // deleting the analysis before a refused update lost it for good.
        analysis.xgid = generateXGID(position);
        await UpdatePosition(position);
        logger.log('Position updated with ID:', positionID);

        if (positionJSON !== originalPositionJSON) {
            await DeleteAnalysis(positionID);
            logger.log('Analysis deleted for position ID:', positionID);
        }
        await SaveAnalysis(positionID, analysis);
        logger.log('Analysis updated for position ID:', positionID);

        await loadAllPositions();
        currentPositionIndexStore.set(currentIndex);
        setStatusBarMessage(tMsg('status.positionUpdated'));
        statusBarModeStore.set('NORMAL');
    } catch (error) {
        logger.error('Error updating position and analysis:', error);
        const existing = duplicatePositionId(error);
        if (existing !== null) {
            setStatusBarMessage(tMsg('status.positionAlreadyExistsWithId', { id: existing }));
        } else {
            setStatusBarMessage(tMsg('status.errorUpdatingPosition'));
        }
    } finally {
        statusBarModeStore.set('NORMAL');
    }
}

export async function saveCurrentPosition() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    if (get(statusBarModeStore) !== 'EDIT') {
        setStatusBarMessage(tMsg('status.saveOnlyEdit'));
        return;
    }

    logger.log('saveCurrentPosition');

    const position = get(positionStore);
    const analysis = get(analysisStore);

    if (!isValidPosition(position)) return;

    analysis.xgid = generateXGID(position);
    analysis.analysisType = '';
    analysis.checkerAnalysis = { moves: [] };
    analysis.doublingCubeAnalysis = emptyDoublingCubeAnalysis();
    analysis.analysisEngineVersion = '';

    const { savePositionAndAnalysis } = await import('./importService.js');
    await savePositionAndAnalysis(position, analysis, tMsg('status.positionSaved'));
    statusBarModeStore.set('NORMAL');
}

export async function updateEPC(position) {
    try {
        // Typed contract from engine/race (ADR-0009):
        // { bottom: {all_in_home, checker_count, epc?}, top: {…}, race?: {…} }.
        const result = await ComputeEPCFromPosition(position);
        const bottomEPC = result?.bottom?.epc || null;
        const topEPC = result?.top?.epc || null;
        const race = result?.race || null;
        // Any recomputation re-masks the challenge overlays: this runs on the
        // same signal as the data itself, so keyboard edits re-mask too.
        resetEpcReveal();
        if (bottomEPC || topEPC || race) {
            epcDataStore.set({
                bottomEPC,
                topEPC,
                race,
                error: null
            });
            // Deliberately NO values in the status bar: the panel displays
            // everything, and the challenge (défi) mode masks the panel — a
            // status-bar copy would leak the answers.
            statusBarTextStore.set('');
        } else {
            // No race data (the checkers are not all in the home board) is the
            // ordinary case for the Eval panel, which evaluates any position:
            // the race block simply stays hidden. Announcing "EPC: N/A" in the
            // status bar was noise on the majority of positions.
            epcDataStore.set({ bottomEPC: null, topEPC: null, race: null, error: null });
            statusBarTextStore.set('');
        }
    } catch (error) {
        logger.error('Error computing EPC:', error);
        epcDataStore.set({ bottomEPC: null, topEPC: null, race: null, error: 'Error computing EPC' });
        statusBarTextStore.set(tMsg('commands.epcErrorComputing'));
    }
}

// Tab toggles ("Afficher/cacher", #202) live in tabToggles.js — re-exported so
// existing callers (keyboardService, commandProcessor, App.svelte, Toolbar.svelte)
// keep one import.
export {
    toggleTab,
    toggleAnalysisPanel,
    toggleCommentPanel,
    toggleMetadataPanel,
    toggleAnkiPanel,
    toggleMatchPanel,
    toggleCollectionPanelAction,
    toggleTournamentPanel,
    toggleStatsPanel,
    toggleSearchPanel,
    togglePipcount
} from './tabToggles.js';

export { loadRandomPosition } from './positionNavigation.js';

export async function addSearchToFilterLibrary(filterName, filterCommand, positionJson, excludePositionJson = '') {
    try {
        await SaveFilter(filterName, filterCommand);
        if (positionJson) {
            await SaveEditPosition(filterName, positionJson);
        }
        if (excludePositionJson) {
            await SaveExcludePosition(filterName, excludePositionJson);
        }
        statusBarTextStore.set(tMsg('commands.filterSaved'));
    } catch (error) {
        logger.error('Error saving filter:', error);
        statusBarTextStore.set(tMsg('commands.errorSavingFilter'));
    }
}
