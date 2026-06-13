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
    GetLastVisitedMatch,
    GetMatchMovePositions,
    SaveEditPosition,
    SaveExcludePosition,
    SaveFilter,
    LoadComment
} from '../../wailsjs/go/database/Database.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, positionsStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore.js';
import { searchExcludePositionStore, emptySearchBoardPosition, boardHasCheckers } from '../stores/searchExcludePositionStore.js';
import { analysisStore, selectedMoveStore } from '../stores/analysisStore.js';
import { epcDataStore } from '../stores/epcStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';
import { viewStore } from '../stores/viewStore.js';
import { currentPositionIndexStore, statusBarTextStore, statusBarModeStore, commentTextStore, PANEL, closePanel, openModal, MODAL, activeTabStore, showPipcountStore } from '../stores/uiStore.js';
import { activeCollectionStore, collectionPositionsStore, selectedCollectionStore } from '../stores/collectionStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
// NOTE: these UI messages are translated at emission time via the non-reactive
// `translate` helper; already-displayed messages do not retranslate on language change.
import { tMsg } from '../i18n';

// Module-level state for EPC mode save/restore
let savedPositionBeforeEPC = null;
let savedPositionIndexBeforeEPC = -1;
let savedPositionsBeforeEPC = null;

// Module-level state for COLLECTION mode save/restore
let _savedPositionBeforeCollection = null;
let _savedPositionIndexBeforeCollection = -1;
let _savedPositionsBeforeCollection = null;

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
        doublingCubeAnalysis: isFirstPositionOfGame
            ? null
            : analysis?.doublingCubeAnalysis || {
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
              },
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
    logger.error('[BUG-TRACE] loadAllPositions called from:\n' + new Error().stack);
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
        savedPositionsBeforeEPC = null;
        savedPositionBeforeEPC = null;
        savedPositionIndexBeforeEPC = -1;
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

export async function loadPositionsByFilters(
    filters,
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
    player1AbsolutePipCountFilter,
    equityFilter,
    decisionTypeFilter,
    diceRollFilter,
    movePatternFilter,
    dateFilter,
    player1OutfieldBlotFilter,
    player2OutfieldBlotFilter,
    player1JanBlotFilter,
    player2JanBlotFilter,
    noContactFilter,
    mirrorPositionFilter,
    moveErrorFilter,
    searchCommand = '',
    matchIDsFilter = '',
    tournamentIDsFilter = '',
    restrictToPositionIDs = '',
    openInNewTab = false,
    diceRollMode = 'both',
    positionIDsFilter = '',
    playerFilter = ''
) {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
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
            player1AbsolutePipCountFilter,
            equityFilter,
            decisionTypeFilter,
            cubeResponseFilter,
            diceRollFilter,
            diceRollMode,
            movePatternFilter,
            dateFilter,
            player1OutfieldBlotFilter,
            player2OutfieldBlotFilter,
            player1JanBlotFilter,
            player2JanBlotFilter,
            noContactFilter,
            mirrorFilter: mirrorPositionFilter,
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
        analysis.doublingCubeAnalysis = {
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
    analysis.doublingCubeAnalysis = {
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
    analysis.analysisEngineVersion = '';

    const { savePositionAndAnalysis } = await import('./importService.js');
    await savePositionAndAnalysis(position, analysis, tMsg('status.positionSaved'));
    statusBarModeStore.set('NORMAL');
}

export async function enterEditMode() {
    logger.log('enterEditMode');
    if (!get(databasePathStore)) return;

    if (get(statusBarModeStore) === 'MATCH') {
        logger.log('Exiting MATCH mode to enter EDIT');
        if (get(matchContextStore).isMatchMode && get(matchContextStore).matchID) {
            try {
                await SaveLastVisitedPosition(get(matchContextStore).matchID, get(matchContextStore).currentIndex);
            } catch (e) {
                logger.error('Error saving last visited position:', e);
            }
        }
        matchContextStore.set({
            isMatchMode: false,
            matchID: null,
            movePositions: [],
            currentIndex: 0,
            player1Name: '',
            player2Name: ''
        });
        loadAllPositions();
    }

    if (get(statusBarModeStore) === 'COLLECTION') {
        await exitCollectionMode();
    }

    if (get(statusBarModeStore) === 'EPC') {
        toggleEPCMode();
    }

    if (get(statusBarModeStore) !== 'EDIT') {
        statusBarModeStore.set('EDIT');
        closePanel(PANEL.COMMENT);
        closePanel(PANEL.ANALYSIS);
        positionStore.update((pos) => {
            pos.board.points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
            pos.board.bearoff = [15, 15];
            pos.cube = { owner: -1, value: 0 };
            pos.score = [7, 7];
            pos.dice = [3, 1];
            pos.decision_type = 0;
            pos.player_on_roll = 0;
            return pos;
        });
    }
}

export function exitEditMode() {
    if (get(statusBarModeStore) === 'EDIT') {
        statusBarModeStore.set('NORMAL');
        const currentIndex = get(currentPositionIndexStore);
        currentPositionIndexStore.set(-1);
        currentPositionIndexStore.set(currentIndex);
    }
}

export function toggleEPCMode() {
    if (get(statusBarModeStore) === 'EPC') {
        exitEPCMode();
        activeTabStore.set('analysis');
    } else {
        activeTabStore.set('epc');
    }
}

export function enterEPCMode() {
    if (get(statusBarModeStore) === 'EPC') return;

    savedPositionBeforeEPC = get(positionStore) ? { ...get(positionStore) } : null;
    savedPositionIndexBeforeEPC = get(currentPositionIndexStore);
    savedPositionsBeforeEPC = get(positionsStore) ? [...get(positionsStore)] : null;

    const epcPoints = Array(26).fill({ checkers: 0, color: -1 });
    epcPoints[1] = { checkers: 2, color: 0 };
    epcPoints[2] = { checkers: 2, color: 0 };
    epcPoints[3] = { checkers: 2, color: 0 };
    epcPoints[4] = { checkers: 3, color: 0 };
    epcPoints[5] = { checkers: 3, color: 0 };
    epcPoints[6] = { checkers: 3, color: 0 };

    const epcPosition = {
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

    statusBarModeStore.set('EPC');
    closePanel(PANEL.COMMENT);
    closePanel(PANEL.ANALYSIS);

    positionsStore.set([epcPosition]);
    positionStore.set(epcPosition);
    currentPositionIndexStore.set(0);
}

export function exitEPCMode() {
    if (get(statusBarModeStore) !== 'EPC') return;

    statusBarModeStore.set('NORMAL');
    statusBarTextStore.set('');
    epcDataStore.set({ bottomEPC: null, topEPC: null, error: null });
    if (savedPositionsBeforeEPC) {
        positionsStore.set(savedPositionsBeforeEPC);
        if (savedPositionBeforeEPC) {
            positionStore.set(savedPositionBeforeEPC);
            currentPositionIndexStore.set(savedPositionIndexBeforeEPC);
        }
        savedPositionsBeforeEPC = null;
        savedPositionBeforeEPC = null;
        savedPositionIndexBeforeEPC = -1;
    } else {
        loadAllPositions();
    }
}

export async function updateEPC(position) {
    try {
        const result = await ComputeEPCFromPosition(position);
        if (result && result.bottomEPC) {
            epcDataStore.set({
                bottomEPC: result.bottomEPC,
                topEPC: result.topEPC || null,
                error: null
            });
            const epc = result.bottomEPC;
            statusBarTextStore.set(tMsg('commands.epcStatus', { epc: epc.epc.toFixed(2), pips: epc.pipCount, wastage: epc.wastage.toFixed(2), rolls: epc.meanRolls.toFixed(3) }));
        } else {
            epcDataStore.set({ bottomEPC: null, topEPC: null, error: null });
            statusBarTextStore.set(tMsg('commands.epcNotAvailable'));
        }
    } catch (error) {
        logger.error('Error computing EPC:', error);
        epcDataStore.set({ bottomEPC: null, topEPC: null, error: 'Error computing EPC' });
        statusBarTextStore.set(tMsg('commands.epcErrorComputing'));
    }
}

export async function toggleMatchMode() {
    logger.log('toggleMatchMode');
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }

    if (get(statusBarModeStore) === 'MATCH') {
        logger.error('[BUG-TRACE] toggleMatchMode EXIT from:\n' + new Error().stack);
        logger.log('Exiting MATCH mode to NORMAL mode via toggleMatchMode');
        if (get(matchContextStore).isMatchMode && get(matchContextStore).matchID) {
            try {
                await SaveLastVisitedPosition(get(matchContextStore).matchID, get(matchContextStore).currentIndex);
            } catch (e) {
                logger.error('Error saving last visited position:', e);
            }
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
        loadAllPositions();
        return;
    }

    if (get(statusBarModeStore) === 'EDIT' || get(statusBarModeStore) === 'EPC' || get(statusBarModeStore) === 'COLLECTION') {
        statusBarModeStore.set('NORMAL');
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

        const startMovePos = movePositions[startIndex];
        positionStore.set(startMovePos.position);

        let analysis = null;
        try {
            analysis = await LoadAnalysis(startMovePos.position.id);
        } catch (_error) {
            /* ignored */
        }

        const currentPlayedMove = startMovePos.checker_move || '';
        const currentPlayedCubeAction = startMovePos.cube_action || '';

        analysisStore.set({
            positionId: analysis?.positionId || null,
            xgid: analysis?.xgid || '',
            player1: analysis?.player1 || '',
            player2: analysis?.player2 || '',
            analysisType: analysis?.analysisType || '',
            analysisEngineVersion: analysis?.analysisEngineVersion || '',
            checkerAnalysis: analysis?.checkerAnalysis || { moves: [] },
            doublingCubeAnalysis: analysis?.doublingCubeAnalysis || {
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
            },
            allCubeAnalyses: analysis?.allCubeAnalyses || [],
            playedMove: currentPlayedMove,
            playedCubeAction: currentPlayedCubeAction,
            playedMoves: analysis?.playedMoves || [],
            playedCubeActions: analysis?.playedCubeActions || [],
            creationDate: analysis?.creationDate || '',
            lastModifiedDate: analysis?.lastModifiedDate || ''
        });

        commentTextStore.set('');
        selectedMoveStore.set(null);
        statusBarModeStore.set('MATCH');
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

export function toggleAnalysisPanel() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    logger.log('toggleAnalysisPanel');
    activeTabStore.set('analysis');
}

export function toggleCommentPanel() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    const positions = get(positionsStore);
    if (!positions[get(currentPositionIndexStore)]) {
        setStatusBarMessage(tMsg('status.noCurrentPositionComment'));
        return;
    }
    logger.log('toggleCommentPanel called');
    activeTabStore.set('comments');
}

// Opens the Metadata tab (not a modal — the standalone MetadataModal was removed).
// Bound to the `meta` command and Ctrl+M.
export function toggleMetadataPanel() {
    if (get(databasePathStore)) {
        if (get(statusBarModeStore) === 'EDIT') {
            setStatusBarMessage(tMsg('status.cannotShowMetadataEdit'));
        } else {
            activeTabStore.set('metadata');
        }
    }
}

export function toggleAnkiPanel() {
    logger.log('toggleAnkiPanel');
    if (!get(databasePathStore)) {
        statusBarTextStore.set(tMsg('commands.noDatabaseLoaded'));
        return;
    }
    activeTabStore.set('anki');
}

export function toggleMatchPanel() {
    logger.log('toggleMatchPanel');
    if (!get(databasePathStore)) {
        statusBarTextStore.set(tMsg('commands.noDatabaseLoaded'));
        return;
    }
    activeTabStore.set('matches');
}

export function toggleCollectionPanelAction() {
    logger.log('toggleCollectionPanelAction');
    if (!get(databasePathStore)) {
        statusBarTextStore.set(tMsg('commands.noDatabaseLoaded'));
        return;
    }
    activeTabStore.set('collections');
}

export function toggleTournamentPanel() {
    logger.log('toggleTournamentPanel');
    if (!get(databasePathStore)) {
        statusBarTextStore.set(tMsg('commands.noDatabaseLoaded'));
        return;
    }
    activeTabStore.set('tournaments');
}

export function toggleStatsPanel() {
    logger.log('toggleStatsPanel');
    if (!get(databasePathStore)) {
        statusBarTextStore.set(tMsg('commands.noDatabaseLoaded'));
        return;
    }
    activeTabStore.set('stats');
}

export async function exitCollectionMode() {
    logger.log('Exiting COLLECTION mode to NORMAL mode');
    const lastViewedPosition = get(positionStore);
    statusBarModeStore.set('NORMAL');
    activeCollectionStore.set(null);
    selectedCollectionStore.set(null);
    collectionPositionsStore.set([]);
    closePanel(PANEL.COLLECTION);
    _savedPositionBeforeCollection = null;
    _savedPositionIndexBeforeCollection = -1;
    _savedPositionsBeforeCollection = null;
    try {
        const allPositions = await LoadAllPositionsDB();
        positionsStore.set(Array.isArray(allPositions) ? allPositions : []);
        if (allPositions && allPositions.length > 0) {
            let targetIdx = allPositions.length - 1;
            if (lastViewedPosition && lastViewedPosition.id) {
                const foundIdx = allPositions.findIndex((p) => p.id === lastViewedPosition.id);
                if (foundIdx >= 0) targetIdx = foundIdx;
            }
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(targetIdx);
            loadAnalysisForPosition(allPositions[targetIdx]);
            hasActiveSearch = false;
            lastSearchCommand = '';
            lastSearchPosition = null;
            lastSearchStore.set(null);
        }
    } catch (error) {
        logger.error('Error reloading positions after collection exit:', error);
        loadAllPositions();
    }
}

export function handleOpenCollection(collection, collectionPositions) {
    if (!collectionPositions || collectionPositions.length === 0) {
        statusBarTextStore.set(tMsg('commands.collectionEmpty'));
        return;
    }

    _savedPositionBeforeCollection = get(positionStore);
    _savedPositionIndexBeforeCollection = get(currentPositionIndexStore);
    _savedPositionsBeforeCollection = get(positionsStore);

    if (get(matchContextStore).isMatchMode) {
        matchContextStore.update((ctx) => ({
            ...ctx,
            isMatchMode: false,
            matchID: null,
            movePositions: [],
            currentIndex: 0
        }));
    }

    statusBarModeStore.set('COLLECTION');
    positionsStore.set(collectionPositions);
    positionStore.set(collectionPositions[0]);
    currentPositionIndexStore.set(0);
    loadAnalysisForPosition(collectionPositions[0]);
    statusBarTextStore.set(tMsg('commands.collectionLoaded', { name: collection.name, count: collectionPositions.length }));
}

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
