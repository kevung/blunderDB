import { get } from 'svelte/store';
import {
    LoadAllPositions as LoadAllPositionsDB,
    DeletePosition,
    DeleteAnalysis,
    UpdatePosition,
    SaveAnalysis,
    LoadAnalysis,
    LoadPositionsByFilters,
    ComputeEPCFromPosition,
    SaveLastVisitedPosition,
    SaveEditPosition,
    SaveExcludePosition,
    SaveFilter,
    LoadComment
} from '../../wailsjs/go/database/Database.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, positionsStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore.js';
import { searchExcludePositionStore, emptySearchBoardPosition, boardHasCheckers } from '../stores/searchExcludePositionStore.js';
import { analysisStore } from '../stores/analysisStore.js';
import { epcDataStore, resetEpcReveal } from '../stores/epcStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';
import { viewStore } from '../stores/viewStore.js';
import { currentPositionIndexStore, statusBarTextStore, statusBarModeStore, commentTextStore, openModal, MODAL, activeTabStore, showPipcountStore } from '../stores/uiStore.js';
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

export function generateXGID(position) {
    const { board, cube, dice, score, player_on_roll, decision_type } = position;

    let positionPart = '';
    for (let i = 0; i < 26; i++) {
        const point = board.points[i];
        if (point.checkers > 0) {
            const charCode = point.color === 0 ? 'A'.charCodeAt(0) : 'a'.charCodeAt(0);
            positionPart += String.fromCharCode(charCode + point.checkers - 1);
        } else {
            positionPart += '-';
        }
    }

    const cubeValue = cube.value;
    const cubeOwner = cube.owner === 0 ? 1 : cube.owner === 1 ? -1 : 0;
    const dicePart = decision_type === 1 ? '00' : dice.join('');
    const matchLength = score[0] === -1 || score[1] === -1 ? 0 : Math.max(score[0], score[1]);
    const actualScore1 = score[0] === -1 ? 0 : matchLength - score[0];
    const actualScore2 = score[1] === -1 ? 0 : matchLength - score[1];
    const isCrawford = score[0] === 1 || score[1] === 1 ? 1 : 0;
    const dummy = 0;
    const playerOnRoll = player_on_roll === 0 ? 1 : -1;

    return `${positionPart}:${cubeValue}:${cubeOwner}:${playerOnRoll}:${dicePart}:${actualScore1}:${actualScore2}:${isCrawford}:${matchLength}:${dummy}`;
}

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

    const positionCopy = JSON.parse(JSON.stringify(position));
    positionStore.set(positionCopy);

    let analysis = null;
    try {
        analysis = await LoadAnalysis(position.id);
    } catch (_error) {
        /* No analysis for this position */
    }

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

    let comment = '';
    try {
        comment = await LoadComment(position.id);
    } catch (_error) {
        /* No comment for this position */
    }
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

export async function loadAllPositions() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    try {
        const positions = await LoadAllPositionsDB();

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

        positionsStore.set(Array.isArray(positions) ? positions : []);
        if (positions && positions.length > 0) {
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(positions.length - 1);
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

        const loadedPositions = await LoadPositionsByFilters({
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

        if (loadedPositions && loadedPositions.length > 0) {
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

            positionsStore.set(Array.isArray(loadedPositions) ? loadedPositions : []);

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

function saveCurrentMatchPosition() {
    if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode) {
        const matchCtx = get(matchContextStore);
        const currentMovePos = matchCtx.movePositions[matchCtx.currentIndex];
        if (currentMovePos) {
            lastVisitedMatchStore.set({
                matchID: matchCtx.matchID,
                currentIndex: matchCtx.currentIndex,
                gameNumber: currentMovePos.game_number
            });
            SaveLastVisitedPosition(matchCtx.matchID, matchCtx.currentIndex).catch((e) => {
                logger.error('Error persisting last visited position:', e);
            });
        }
    }
}

export async function firstPosition() {
    if (get(statusBarModeStore) === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotBrowseEdit'));
        return;
    }
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode) {
        const matchCtx = get(matchContextStore);
        const currentGameNumber = matchCtx.movePositions[matchCtx.currentIndex].game_number;

        let targetIndex = -1;
        for (let i = matchCtx.currentIndex - 1; i >= 0; i--) {
            if (matchCtx.movePositions[i].game_number < currentGameNumber) {
                targetIndex = i;
                break;
            }
        }

        if (targetIndex === -1) {
            targetIndex = 0;
        } else {
            const targetGameNumber = matchCtx.movePositions[targetIndex].game_number;
            for (let i = 0; i < matchCtx.movePositions.length; i++) {
                if (matchCtx.movePositions[i].game_number === targetGameNumber) {
                    targetIndex = i;
                    break;
                }
            }
        }

        matchContextStore.update((ctx) => ({ ...ctx, currentIndex: targetIndex }));
        const movePos = matchCtx.movePositions[targetIndex];
        await showPosition(movePos.position);
        statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
        saveCurrentMatchPosition();
    } else {
        const positions = get(positionsStore);
        if (positions && positions.length > 0) {
            currentPositionIndexStore.set(0);
        }
    }
}

export async function previousPosition() {
    if (get(statusBarModeStore) === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotBrowseEdit'));
        return;
    }
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode) {
        const matchCtx = get(matchContextStore);
        if (matchCtx.currentIndex > 0) {
            let newIndex = matchCtx.currentIndex - 1;
            while (newIndex >= 0) {
                const movePos = matchCtx.movePositions[newIndex];
                if (movePos.move_type === 'checker' || movePos.move_type === 'cube') break;
                newIndex--;
            }

            if (newIndex >= 0) {
                matchContextStore.update((ctx) => ({ ...ctx, currentIndex: newIndex }));
                const movePos = matchCtx.movePositions[newIndex];
                await showPosition(movePos.position);
                statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
                saveCurrentMatchPosition();
            }
        }
    } else {
        const positions = get(positionsStore);
        if (positions && get(currentPositionIndexStore) > 0) {
            currentPositionIndexStore.set(get(currentPositionIndexStore) - 1);
        }
    }
}

export async function nextPosition() {
    if (get(statusBarModeStore) === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotBrowseEdit'));
        return;
    }
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode) {
        const matchCtx = get(matchContextStore);
        if (matchCtx.currentIndex < matchCtx.movePositions.length - 1) {
            let newIndex = matchCtx.currentIndex + 1;
            while (newIndex < matchCtx.movePositions.length) {
                const movePos = matchCtx.movePositions[newIndex];
                if (movePos.move_type === 'checker' || movePos.move_type === 'cube') break;
                newIndex++;
            }

            if (newIndex < matchCtx.movePositions.length) {
                matchContextStore.update((ctx) => ({ ...ctx, currentIndex: newIndex }));
                const movePos = matchCtx.movePositions[newIndex];
                await showPosition(movePos.position);
                statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
                saveCurrentMatchPosition();
            }
        }
    } else {
        const positions = get(positionsStore);
        if (positions && get(currentPositionIndexStore) < positions.length - 1) {
            currentPositionIndexStore.set(get(currentPositionIndexStore) + 1);
        }
    }
}

export async function lastPosition() {
    if (get(statusBarModeStore) === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotBrowseEdit'));
        return;
    }
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    if (get(statusBarModeStore) === 'MATCH' && get(matchContextStore).isMatchMode) {
        const matchCtx = get(matchContextStore);
        const currentGameNumber = matchCtx.movePositions[matchCtx.currentIndex].game_number;

        let targetIndex = -1;
        for (let i = matchCtx.currentIndex + 1; i < matchCtx.movePositions.length; i++) {
            if (matchCtx.movePositions[i].game_number > currentGameNumber) {
                targetIndex = i;
                break;
            }
        }

        if (targetIndex === -1) {
            const maxGameNumber = Math.max(...matchCtx.movePositions.map((p) => p.game_number));
            for (let i = 0; i < matchCtx.movePositions.length; i++) {
                if (matchCtx.movePositions[i].game_number === maxGameNumber) {
                    targetIndex = i;
                    break;
                }
            }
        }

        if (targetIndex !== -1) {
            matchContextStore.update((ctx) => ({ ...ctx, currentIndex: targetIndex }));
            const movePos = matchCtx.movePositions[targetIndex];
            await showPosition(movePos.position);
            statusBarTextStore.set(`${matchCtx.player1Name} vs ${matchCtx.player2Name}`);
            saveCurrentMatchPosition();
        }
    } else {
        const positions = get(positionsStore);
        if (positions && positions.length > 0) {
            currentPositionIndexStore.set(positions.length - 1);
        }
    }
}

export function gotoPosition() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    if (get(statusBarModeStore) === 'EDIT') {
        setStatusBarMessage(tMsg('status.cannotGoToEdit'));
        return;
    }
    openModal(MODAL.GO_TO_POSITION);
}

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
        const positionID = positions[get(currentPositionIndexStore)].id;
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
        const originalPosition = positions[currentIndex];

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

        if (positionJSON !== originalPositionJSON) {
            await DeleteAnalysis(positionID);
            logger.log('Analysis deleted for position ID:', positionID);
        }

        analysis.xgid = generateXGID(position);
        await UpdatePosition(position);
        logger.log('Position updated with ID:', positionID);
        await SaveAnalysis(positionID, analysis);
        logger.log('Analysis updated for position ID:', positionID);

        await loadAllPositions();
        currentPositionIndexStore.set(currentIndex);
        setStatusBarMessage(tMsg('status.positionUpdated'));
        statusBarModeStore.set('NORMAL');
    } catch (error) {
        logger.error('Error updating position and analysis:', error);
        setStatusBarMessage(tMsg('status.errorUpdatingPosition'));
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
            epcDataStore.set({ bottomEPC: null, topEPC: null, race: null, error: null });
            statusBarTextStore.set(tMsg('commands.epcNotAvailable'));
        }
    } catch (error) {
        logger.error('Error computing EPC:', error);
        epcDataStore.set({ bottomEPC: null, topEPC: null, race: null, error: 'Error computing EPC' });
        statusBarTextStore.set(tMsg('commands.epcErrorComputing'));
    }
}

// ── Tab toggles ──────────────────────────────────────────────────────────────
// The `toggleXPanel` names date from when each panel was a floating window;
// today every one of them selects a tab of the tabbed panel (App.svelte's tab
// effect opens the matching PANEL). One table, one function; the eight names
// stay as re-exports for keyboardService, commandProcessor and App.svelte.
//
//   tab      the activeTabStore value to select
//   guard    extra precondition, returning a status-bar message key to refuse
//   silent   no message when no database is open (metadata: the tab just
//            stays where it is)
const TAB_TOGGLES = Object.freeze({
    analysis: { tab: 'analysis' },
    comments: {
        tab: 'comments',
        guard: () => (get(positionsStore)[get(currentPositionIndexStore)] ? null : 'status.noCurrentPositionComment')
    },
    metadata: { tab: 'metadata', silent: true, guard: () => (get(statusBarModeStore) === 'EDIT' ? 'status.cannotShowMetadataEdit' : null) },
    anki: { tab: 'anki' },
    matches: { tab: 'matches' },
    collections: { tab: 'collections' },
    tournaments: { tab: 'tournaments' },
    stats: { tab: 'stats' }
});

/** Select the tab of `id` (a TAB_TOGGLES key) if a database is open. */
export function toggleTab(id) {
    const entry = TAB_TOGGLES[id];
    if (!entry) throw new Error(`toggleTab: unknown tab '${id}'`);
    logger.log(`toggleTab ${id}`);
    if (!get(databasePathStore)) {
        if (!entry.silent) setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    const refusal = entry.guard?.();
    if (refusal) {
        setStatusBarMessage(tMsg(refusal));
        return;
    }
    activeTabStore.set(entry.tab);
}

export const toggleAnalysisPanel = () => toggleTab('analysis');
export const toggleCommentPanel = () => toggleTab('comments');
// Bound to the `meta` command and Ctrl+M (a tab, not a modal).
export const toggleMetadataPanel = () => toggleTab('metadata');
export const toggleAnkiPanel = () => toggleTab('anki');
export const toggleMatchPanel = () => toggleTab('matches');
export const toggleCollectionPanelAction = () => toggleTab('collections');
export const toggleTournamentPanel = () => toggleTab('tournaments');
export const toggleStatsPanel = () => toggleTab('stats');

export function togglePipcount() {
    logger.log('togglePipcount');
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    showPipcountStore.set(!get(showPipcountStore));
    if (get(statusBarModeStore) === 'MATCH') {
        const currentPosition = get(positionStore);
        positionStore.set({ ...currentPosition });
    } else {
        const currentIndex = get(currentPositionIndexStore);
        currentPositionIndexStore.set(-1);
        currentPositionIndexStore.set(currentIndex);
    }
}

export function loadRandomPosition() {
    logger.log('loadRandomPosition');
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    const positions = get(positionsStore);
    if (positions && positions.length > 0) {
        let randomIndex = Math.floor(Math.random() * positions.length);
        while (randomIndex === get(currentPositionIndexStore)) {
            randomIndex = Math.floor(Math.random() * positions.length);
        }
        logger.log('Random position index:', randomIndex);
        currentPositionIndexStore.set(randomIndex);
    }
}

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
