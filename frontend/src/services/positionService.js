import { get } from 'svelte/store';
import {
    ListPositionIDs,
    TrashPosition,
    DeleteAnalysis,
    UpdatePosition,
    SaveAnalysis,
    LoadAnalysis,
    LoadPositionIDsByFilters,
    RankPositionIDsByFilters,
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
import { rankedDistancesStore, rankedTargetStore } from '../stores/rankedStore.js';
import { GetLikeLimit, GetLikeMaxDistance } from '../../wailsjs/go/main/Config.js';
import { activeCollectionStore } from '../stores/collectionStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { confirmAction } from './confirmService.js';
import { logger } from '../utils/logger.js';
import { forgetContextBeforeEval, forgetSubSearchOrigin, noteSubSearchOrigin } from './modeMachine.js';
// Ctrl-G status line (keyboardService imports it from here).
export { showDatesAndMetadata } from './metadataStatus.js';

// The mode automaton lives in modeMachine.js; its transitions are re-exported
// here so callers keep one import.
export {
    enterEditMode,
    exitEditMode,
    toggleEvalMode,
    sendPositionToEval,
    enterEvalMode,
    exitEvalMode,
    enterTranscribeMode,
    exitTranscribeMode,
    toggleMatchMode,
    handleOpenCollection,
    exitCollectionMode,
    leaveSubSearchResults,
    canLeaveSubSearchResults,
    displayedPositionIDs
} from './modeMachine.js';
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

// Re-exported for existing callers; also used directly below.
export { generateXGID } from './xgid.js';
import { generateXGID } from './xgid.js';

// The rules live in positionRefusal.js, a pure function a panel can read to
// disable its save button; this wrapper says the refusal in the status bar.
export { positionRefusal } from './positionRefusal.js';
import { positionRefusal } from './positionRefusal.js';

export function isValidPosition(position) {
    const refusal = positionRefusal(position);
    if (refusal) {
        setStatusBarMessage(tMsg(refusal));
        return false;
    }
    return true;
}

/**
 * The board a search sends as its "at least" checker structure.
 *
 * The backend reads any board with a checker as a structure results must
 * contain (HasBoardFilter). In EDIT that is the drawn query; elsewhere it is
 * the full position on screen, which would narrow `s E>80` to that one
 * position (TestSearch_DisplayedBoardOutsideEdit). So outside EDIT only the
 * checkers are dropped: dice, cube, score, side on roll and decision type
 * still travel for the `D`, `cube`, `score` and `d` tokens. History records
 * this board, so a replay asks the same question.
 *
 * @param {any} [position] defaults to the board on screen
 * @returns {any} a copy outside EDIT; the position itself in EDIT
 */
export function searchQueryBoard(position = get(positionStore)) {
    if (!position || get(statusBarModeStore) === 'EDIT') return position;
    const board = JSON.parse(JSON.stringify(position));
    board.board = { ...board.board, points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })) };
    return board;
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

    // JSON round-trip, not structuredClone: in MATCH mode the position is a
    // Svelte 5 proxy, on which structuredClone throws DataCloneError.
    const positionCopy = JSON.parse(JSON.stringify(position));
    positionStore.set(positionCopy);

    // Analysis and comment fetched concurrently. Each call is deferred into
    // .then() so a synchronous throw becomes a rejection allSettled catches.
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
 * when the library holds it (so exiting a match keeps the studied position),
 * else on the last position.
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
        forgetContextBeforeEval();
        forgetSubSearchOrigin();
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

// Explicit user reload (Ctrl+R, toolbar, `e`): a study action, so show the
// analysis panel rather than the Matches tab loadAllPositions lands on — only
// with an open, non-empty database.
export async function reloadAllPositions() {
    await loadAllPositions();
    if (get(databasePathStore) && get(positionsStore).length > 0) {
        activeTabStore.set('analysis');
    }
}

// One options object, not positional arguments: a wrong index would silently
// shift every later filter and answer a different question.
export async function loadPositionsByFilters({
    filters = [],
    // Le classement par similarité (ADR-0043). `likeFilter` dit que la requête
    // CLASSE au lieu de seulement restreindre ; les autres jetons continuent de
    // dire lesquelles sont candidates.
    likeFilter = false,
    likeTargetId = 0,
    likeMaxDistance = 0,
    likeWidened = false,
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
    playerFilter = '',
    gamePhaseFilter = '',
    gameTypeFilter = '',
    encounterFilter = '',
    commentOriginFilter = '',
    tagFilter = '',
    // The board a saved filter was stored with: its structure or `like`
    // target, used whatever the mode. Otherwise the board on screen, a
    // structure only in EDIT.
    queryBoard = null
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
        // The structure comes from the board only in EDIT, the query board.
        let currentPosition = queryBoard ?? searchQueryBoard(get(positionStore));

        // The exclude ("Sauf") structure follows the include board's mirror
        // decision, so both stay aligned with stored positions.
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

        // Cube sub-type (cube decisions only): `dr` = take/pass, `dd` =
        // double/no-double, derived from tokens for panel and command line alike.
        const cubeResponseFilter = Array.isArray(filters) ? (filters.includes('dr') ? 'takepass' : filters.includes('dd') ? 'double' : '') : '';

        // `co` = has a comment, `xco` = has none; same token path as above.
        const commentFilter = Array.isArray(filters) ? (filters.includes('xco') ? 'none' : filters.includes('co') ? 'has' : '') : '';

        // Only ids cross the Wails bridge; positionsStore fetches the window
        // it shows through LoadPositionsByIDs.
        // La cible d'un `like` nu se résout ici : « cette position » est celle
        // qu'on feuillette. En mode ÉDITION, le plateau DESSINÉ est la cible —
        // un dessin approximatif que `like` pardonne.
        let likeTarget = likeTargetId;
        if (likeFilter && !likeTarget && !queryBoard && get(statusBarModeStore) !== 'EDIT') {
            likeTarget = positionsStore.idAt(get(currentPositionIndexStore)) || 0;
            if (!likeTarget) {
                setStatusBarMessage(tMsg('similar.noPosition'));
                return;
            }
        }

        // Dans un classement, le plateau est la CIBLE, jamais un motif : comme
        // motif il exigerait de chaque candidate la structure exacte de la
        // cible, qui est exclue, et viderait le résultat. `filter` part vide.
        const rankAgainstDrawnBoard = likeFilter && !likeTarget;
        const payload = {
            filter: likeFilter ? emptySearchBoardPosition() : currentPosition,
            likeTargetBoard: rankAgainstDrawnBoard ? currentPosition : emptySearchBoardPosition(),
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
            gamePhaseFilter,
            gameTypeFilter,
            encounterFilter,
            commentOriginFilter,
            tagFilter,
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
            restrictToPositionIDs,
            likeFilter,
            likeTargetId: likeTarget,
            likeMaxDistance,
            likeWidened
        };

        // Appel propre au classement : la distance fait partie de la réponse,
        // sans elle une voisine ne se distingue pas d'une coïncidence.
        let ids;
        let rankedSummary = null;
        if (likeFilter) {
            let ranked;
            try {
                // Les deux réglages du panneau de configuration, que le jeton
                // surcharge pour une requête : `like<12` l'emporte sur le
                // plafond enregistré, et l'emporte seulement là (ADR-0043).
                const [limit, ceiling] = await Promise.all([GetLikeLimit(), GetLikeMaxDistance()]);
                if (!payload.likeMaxDistance) payload.likeMaxDistance = ceiling || 0;
                ranked = (await RankPositionIDsByFilters(payload, limit || 0)) || [];
            } catch (error) {
                logger.error('could not rank the neighbours:', error);
                setStatusBarMessage(tMsg('similar.failed'));
                return;
            }
            ids = ranked.map((n) => n.id);
            rankedDistancesStore.set(new Map(ranked.map((n) => [n.id, n.distance])));
            rankedTargetStore.set(likeTarget);
            if (ranked.length > 0) {
                rankedSummary = tMsg('similar.found', {
                    n: ranked.length,
                    nearest: ranked[0].distance,
                    farthest: ranked[ranked.length - 1].distance
                });
            }
        } else {
            ids = await LoadPositionIDsByFilters(payload);
            rankedDistancesStore.set(new Map());
            rankedTargetStore.set(0);
        }

        if (ids && ids.length > 0) {
            if (openInNewTab) {
                viewStore.addView();
            }

            // Before any store moves: a sub-search run from a collection or a
            // match remembers it, so that leaving the results returns there.
            const subSearchOrigin = noteSubSearchOrigin(Boolean(restrictToPositionIDs), Array.isArray(ids) ? ids : []);

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

            // La fourchette des distances, dite une fois la liste posée, montre
            // si le classement a trouvé des voisines ou des inconnues.
            if (rankedSummary) {
                setStatusBarMessage(rankedSummary);
            } else if (subSearchOrigin) {
                // Say where the results come from and how to get back: the list
                // the user was studying is no longer on screen.
                setStatusBarMessage(tMsg(subSearchOrigin === 'MATCH' ? 'status.subSearchInMatch' : 'status.subSearchInCollection'));
            }
        } else {
            // Un classement vide dit « aucune n'est proche », pas « aucune ne
            // correspond » (ADR-0043).
            setStatusBarMessage(tMsg(likeFilter ? 'similar.none' : 'status.noMatchingPositions'));
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
        // Restore the pre-search message unless a no-match or error branch
        // already replaced the "searching" placeholder.
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
        // Through the trash: the delete really happens, but a snapshot
        // is written first, so `trash` can put it back for thirty days.
        await TrashPosition(positionID);
        logger.log('Position and associated analysis deleted with ID:', positionID);

        await loadAllPositions();
        setStatusBarMessage(tMsg('status.positionDeletedUndo'));
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

        // The position row goes first: if the edit is refused (now a
        // duplicate), the analysis must not have been deleted yet.
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

// Ctrl-S, the toolbar button and `w`. The board that can be saved is a
// scratch board, and scratchBoard.js is its one write path.
export async function saveCurrentPosition() {
    const { saveScratchBoard } = await import('./scratchBoard.js');
    await saveScratchBoard();
}

export async function updateEPC(position) {
    try {
        // Typed contract from engine/race (ADR-0009):
        // { bottom: {all_in_home, checker_count, farthest, points, epc?},
        //   top: {…}, race?: {…} }.
        const result = await ComputeEPCFromPosition(position);
        const bottomEPC = result?.bottom?.epc || null;
        const topEPC = result?.top?.epc || null;
        const race = result?.race || null;
        // Width of the one-sided table each side was answered from (ADR-0027
        // §9); shown only when not six, i.e. a checker outside home board.
        const bottomPoints = result?.bottom?.points ?? 0;
        const topPoints = result?.top?.points ?? 0;
        // Any recomputation re-masks the challenge overlays: this runs on the
        // same signal as the data itself, so keyboard edits re-mask too.
        resetEpcReveal();
        if (bottomEPC || topEPC || race) {
            epcDataStore.set({
                bottomEPC,
                topEPC,
                bottomPoints,
                topPoints,
                race,
                error: null
            });
            // Deliberately NO values in the status bar: the panel displays
            // everything, and the challenge (défi) mode masks the panel — a
            // status-bar copy would leak the answers.
            statusBarTextStore.set('');
        } else {
            // No race data is the ordinary case in the Eval panel: the race
            // block stays hidden, no status message.
            epcDataStore.set({ bottomEPC: null, topEPC: null, bottomPoints: 0, topPoints: 0, race: null, error: null });
            statusBarTextStore.set('');
        }
    } catch (error) {
        logger.error('Error computing EPC:', error);
        epcDataStore.set({ bottomEPC: null, topEPC: null, bottomPoints: 0, topPoints: 0, race: null, error: 'Error computing EPC' });
        statusBarTextStore.set(tMsg('commands.epcErrorComputing'));
    }
}

// Tab toggles live in tabToggles.js, re-exported for one import.
export {
    toggleTab,
    showTab,
    toggleAnalysisPanel,
    toggleCommentPanel,
    toggleMetadataPanel,
    toggleAnkiPanel,
    toggleTrainingPanel,
    showTrainingPanel,
    toggleMatchPanel,
    toggleCollectionPanelAction,
    toggleTournamentPanel,
    toggleStatsPanel,
    toggleSearchPanel,
    toggleTranscriptionPanel,
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
