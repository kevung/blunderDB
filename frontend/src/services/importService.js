import { tMsg } from '../i18n';
import { get } from 'svelte/store';
import {
    OpenImportDatabaseDialog,
    OpenPositionFilesDialog,
    OpenPositionFolderDialog,
    CollectImportableFiles,
    ReadFileContent,
    ShowAlert,
    ShowQuestionDialog,
    IsDirectory,
    LooksLikeOGID
} from '../../wailsjs/go/gui/App.js';
import {
    SaveIndividualPosition,
    SaveAnalysis,
    LoadComment,
    SaveComment,
    AnalyzeImportDatabase,
    CommitImportDatabase,
    CancelImport,
    ImportXGMatch,
    ImportGnuBGMatch,
    ImportGnuBGMatchFromText,
    ImportBGFMatch,
    ImportBGFPosition,
    ImportBGFPositionFromText,
    ImportXGPPosition,
    ParsePositionText,
    RefreshSearchStatistics,
    CountPositionsWithoutAnalysis,
    BeginImportBatch,
    FinishImportBatch,
    ImportReport
} from '../../wailsjs/go/database/Database.js';
import { GetGammonNetAutoAnalyze, GetGammonNetAnalysisPly, GetGammonNetPruneK } from '../../wailsjs/go/main/Config.js';
import { StartGammonNetBatch } from '../../wailsjs/go/gui/App.js';
import { ClipboardGetText } from '../../wailsjs/runtime/runtime.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, positionsStore, pastePositionTextStore, matchContextStore, clipboardPositionStore } from '../stores/positionStore.js';
import { analysisStore } from '../stores/analysisStore.js';
import { currentPositionIndexStore, statusBarModeStore, commentTextStore, openPanel, PANEL, activeTabStore, matchPanelRefreshTriggerStore, dbMutationCounterStore } from '../stores/uiStore.js';
import {
    showImportProgressModalStore,
    importModalModeStore,
    importAnalysisStore,
    importResultStore,
    showFileImportModalStore,
    fileImportModeStore,
    fileImportTotalFilesStore,
    fileImportCurrentIndexStore,
    fileImportCurrentFileStore,
    fileImportResultsStore,
    fileImportReportStore
} from '../stores/importModalStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';

// Pending import path (module-level)
let pendingImportPath = null;
let fileImportCancelled = false;

// Reload the whole position table from the backend. This is the expensive
// step of any import — every row crosses the Wails IPC — so an import path
// performs it once, after its last write, never per file: a batch of N files
// reloads at the end of the loop, not N times inside it.
async function reloadPositions() {
    const { loadAllPositions } = await import('./positionService.js');
    await loadAllPositions();
}

// Leave match mode the way loadAllPositions() does. Import paths that already
// reloaded have left it on their own; only a path that never reached a reload
// (dialog cancelled, import failed) still has to.
async function leaveMatchModeIfStillIn() {
    if (!get(matchContextStore).isMatchMode && get(statusBarModeStore) !== 'MATCH') return;
    matchContextStore.set({
        isMatchMode: false,
        matchID: null,
        movePositions: [],
        currentIndex: 0,
        player1Name: '',
        player2Name: ''
    });
    await reloadPositions();
}

// gammonNet auto-analyze after import (#129, ADR-0013): a config toggle, not
// a per-import-type special case — any import completion checks whether
// there is a backlog of positions with no analysis at all, and if the
// setting is on, hands the batch off to the goroutine+cancel+events shell in
// gui.App (gammonnet_batch.go). Uniform across import shapes (.mat/.txt with
// no analysis, .xg/.sgf/.bgf that already carry one, a whole-database merge)
// on purpose: the batch itself is a no-op when the count is zero, so there is
// nothing to special-case by import type.
async function maybeAutoAnalyzeAfterImport() {
    try {
        const auto = await GetGammonNetAutoAnalyze();
        if (!auto) return;
        const remaining = await CountPositionsWithoutAnalysis();
        if (!remaining) return;
        const [ply, pruneK] = await Promise.all([GetGammonNetAnalysisPly(), GetGammonNetPruneK()]);
        await StartGammonNetBatch(ply, pruneK, 0);
    } catch (error) {
        logger.error('gammonNet auto-analyze check failed:', error);
    }
}

export async function importDatabase() {
    logger.log('importDatabase');

    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDbOpenedFirst'));
        return;
    }

    try {
        const importFilePath = await OpenImportDatabaseDialog();
        if (!importFilePath) {
            logger.log('No import database selected');
            return;
        }

        logger.log('Analyzing import from:', importFilePath);

        showImportProgressModalStore.set(true);
        importModalModeStore.set('analyzing');
        pendingImportPath = importFilePath;

        try {
            const analysis = await AnalyzeImportDatabase(importFilePath);
            logger.log('Import analysis:', analysis);

            importAnalysisStore.set({
                toAdd: analysis.toAdd,
                toMerge: analysis.toMerge,
                toSkip: analysis.toSkip,
                total: analysis.total,
                importPath: importFilePath
            });
            importModalModeStore.set('preview');
        } catch (analyzeError) {
            showImportProgressModalStore.set(false);
            throw analyzeError;
        }
    } catch (error) {
        logger.error('Error analyzing import:', error);
        setStatusBarMessage(tMsg('status.errorAnalyzingImport', { error }));
        await ShowAlert(`Error analyzing import: ${error}`);
        statusBarModeStore.set('NORMAL');
    }
}

export async function handleImportCommit() {
    try {
        await handleImportCommitCore();
    } finally {
        await maybeAutoAnalyzeAfterImport();
    }
}

async function handleImportCommitCore() {
    if (!pendingImportPath) {
        logger.error('No pending import path');
        return;
    }

    logger.log('Committing import from:', pendingImportPath);
    importModalModeStore.set('committing');

    try {
        const result = await CommitImportDatabase(pendingImportPath);
        logger.log('Import result:', result);

        importResultStore.set({
            added: result.added,
            merged: result.merged,
            skipped: result.skipped,
            total: result.total
        });
        importModalModeStore.set('completed');

        setStatusBarMessage(tMsg('status.importCompleted', { added: result.added, merged: result.merged, skipped: result.skipped }));

        await reloadPositions();
    } catch (error) {
        // The user hitting Cancel mid-commit surfaces here as a rejected
        // promise too (CommitImportDatabase's context is what CancelImport
        // actually cancels) — Go wraps that case in ErrImportCancelled
        // (pkg/blunderdb/database/db_import_db.go), whose message always
        // starts with "import cancelled by user". Treat it as the
        // successful cancellation it is, not a failure: no error alert (#241).
        if (String(error).includes('import cancelled by user')) {
            logger.log('Import commit cancelled by user');
            showImportProgressModalStore.set(false);
            statusBarModeStore.set('NORMAL');
            return;
        }
        logger.error('Error committing import:', error);
        showImportProgressModalStore.set(false);
        setStatusBarMessage(tMsg('status.errorCommittingImport', { error }));
        await ShowAlert(`Error committing import: ${error}`);
        statusBarModeStore.set('NORMAL');
    } finally {
        pendingImportPath = null;
    }
}

export function handleImportCancel() {
    logger.log('Import cancelled by user');

    if (get(importModalModeStore) === 'committing') {
        logger.log('Aborting ongoing commit transaction');
        CancelImport().catch((err) => {
            logger.error('Error calling CancelImport:', err);
        });
    }

    showImportProgressModalStore.set(false);
    pendingImportPath = null;
    importModalModeStore.set('analyzing');
    setStatusBarMessage(tMsg('status.importCancelled'));
    statusBarModeStore.set('NORMAL');
}

export function handleImportClose() {
    logger.log('Import completed and closed');
    showImportProgressModalStore.set(false);
    pendingImportPath = null;
    importModalModeStore.set('analyzing');
    statusBarModeStore.set('NORMAL');
}

export async function importDatabaseByPath(importFilePath) {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDbOpenedFirst'));
        return;
    }

    try {
        logger.log('Analyzing import from:', importFilePath);

        showImportProgressModalStore.set(true);
        importModalModeStore.set('analyzing');
        pendingImportPath = importFilePath;

        try {
            const analysis = await AnalyzeImportDatabase(importFilePath);
            logger.log('Import analysis:', analysis);

            importAnalysisStore.set({
                toAdd: analysis.toAdd,
                toMerge: analysis.toMerge,
                toSkip: analysis.toSkip,
                total: analysis.total,
                importPath: importFilePath
            });
            importModalModeStore.set('preview');
        } catch (analyzeError) {
            showImportProgressModalStore.set(false);
            throw analyzeError;
        }
    } catch (error) {
        logger.error('Error analyzing import:', error);
        setStatusBarMessage(tMsg('status.errorAnalyzingImport', { error }));
        await ShowAlert(`Error analyzing import: ${error}`);
        statusBarModeStore.set('NORMAL');
    }
}

// Show a freshly imported position on the board with its analysis tab open.
// (Match imports keep opening the match list instead — that is where a match is
// read.) loadAllPositions() always selects the matches tab, so this has to run
// after every reload an import performs, not before.
export async function showImportedPosition(positionID) {
    if (positionID) {
        let index = positionsStore.indexOf(positionID);
        if (index < 0) {
            // The position is not in the current view (search subset, match mode,
            // or a brand-new row): reload the full list so we can point at it.
            await reloadPositions();
            index = positionsStore.indexOf(positionID);
        }
        if (index >= 0) {
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(index);
        }
    }
    activeTabStore.set('analysis');
}

// Returns the ID of the saved (or merged) position, or null on failure.
//
// `reload` (default true) refreshes the position table after a brand-new row
// is written. A batch import passes false: it reloads once itself, after its
// last file, instead of paying one full reload per position (see
// importMultipleFilesCore).
export async function savePositionAndAnalysis(positionData, parsedAnalysis, successMessage, { reload = true } = {}) {
    if (Array.isArray(parsedAnalysis.checkerAnalysis)) {
        parsedAnalysis.checkerAnalysis = { moves: parsedAnalysis.checkerAnalysis };
    }

    delete parsedAnalysis.creationDate;
    delete parsedAnalysis.lastModifiedDate;

    // One backend call decides existence AND writes: SaveIndividualPosition
    // deduplicates on the Zobrist hash (the same notion of "same position" the
    // rest of the app uses) and records that the user brought this position in
    // on its own rather than inside a match — see docs/adr/0002.
    //
    // The previous code called PositionExists first and, when the position was
    // already stored, skipped the write entirely. That skipped the provenance
    // flag in exactly the case it exists for: importing a match, then saving one
    // of its positions from the board. It also compared marshalled JSON in an
    // O(n) scan of the whole table, a second, divergent notion of identity.
    let saveResult;
    try {
        saveResult = await SaveIndividualPosition(positionData);
    } catch (error) {
        logger.error('Error saving position:', error);
        setStatusBarMessage(tMsg('status.errorSavingPosition'));
        return null;
    }
    const positionID = saveResult.id;

    if (saveResult.existed) {
        logger.log('Position already exists with ID:', positionID);
        try {
            parsedAnalysis.positionId = positionID;
            await SaveAnalysis(positionID, parsedAnalysis);

            let existingComment = await LoadComment(positionID);
            const newComment = parsedAnalysis.comment || '';
            const trimmedExisting = (existingComment || '').trim();
            const trimmedNew = newComment.trim();
            let mergedComment = trimmedExisting;

            if (trimmedNew && !trimmedExisting.includes(trimmedNew)) {
                if (trimmedExisting) {
                    mergedComment = `${trimmedExisting}\n\n${trimmedNew}`;
                } else {
                    mergedComment = trimmedNew;
                }
            }

            await SaveComment(positionID, mergedComment);
            logger.log('Analysis and comment updated for position ID:', positionID);
            setStatusBarMessage(tMsg('status.positionMerged'));

            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(positionsStore.indexOf(positionID));
            commentTextStore.set(mergedComment);
        } catch (error) {
            logger.error('Error updating analysis and comment:', error);
            setStatusBarMessage(tMsg('status.errorUpdatingAnalysisComment'));
            return null;
        }
        return positionID;
    }

    try {
        logger.log('Position saved with ID:', positionID);

        positionData.id = positionID;
        parsedAnalysis.positionId = positionID;
        await SaveAnalysis(positionID, parsedAnalysis);
        await SaveComment(positionID, parsedAnalysis.comment);
        logger.log('Analysis and comment saved for position ID:', positionID);

        if (reload) await reloadPositions();
        setStatusBarMessage(successMessage);
        return positionID;
    } catch (error) {
        logger.error('Error saving position, analysis, and comment:', error);
        setStatusBarMessage(tMsg('status.errorSavingPosition'));
        return null;
    }
}

export async function importPosition() {
    const wasMatchMode = get(statusBarModeStore) === 'MATCH';
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return;
    }
    let outcome = null;
    try {
        const files = await OpenPositionFilesDialog();
        if (!files || files.length === 0) return;

        if (files.length === 1) {
            outcome = await importSingleFile(files[0]);
        } else {
            outcome = await importMultipleFiles(files);
        }
    } catch (error) {
        logger.error('Error importing position:', error);
    } finally {
        if (wasMatchMode) await leaveMatchModeIfStillIn();
        statusBarModeStore.set('NORMAL');
        // Re-apply the analysis view: leaving match mode reloads every position,
        // which resets the tab back to matches.
        if (outcome && outcome.type === 'position') {
            await showImportedPosition(outcome.id);
        }
    }
}

export async function importFolder() {
    const wasMatchMode = get(statusBarModeStore) === 'MATCH';
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return;
    }
    let outcome = null;
    try {
        const dirPath = await OpenPositionFolderDialog();
        if (!dirPath) return;

        const files = await CollectImportableFiles(dirPath);
        if (!files || files.length === 0) {
            setStatusBarMessage(tMsg('status.noImportableFolder'));
            return;
        }

        outcome = await importMultipleFiles(files);
    } catch (error) {
        logger.error('Error importing folder:', error);
    } finally {
        if (wasMatchMode) await leaveMatchModeIfStillIn();
        statusBarModeStore.set('NORMAL');
        if (outcome && outcome.type === 'position') {
            await showImportedPosition(outcome.id);
        }
    }
}

// Returns { type: 'position' | 'match', id } on success, null on failure.
export async function importSingleFile(filePath) {
    // One file the user dropped or picked is one import, and it gets the same
    // end-of-import report a folder does (#257) — but only when it produced a
    // MATCH. Importing a single position is a two-second gesture that lands on
    // the board; interrupting it with a report of one line would be noise.
    const batchID = await beginImportBatch(filePath, extensionOf(filePath));
    let outcome = null;
    try {
        outcome = await importSingleFileCore(filePath);
        return outcome;
    } finally {
        await finishImportBatch(batchID);
        if (outcome && outcome.type === 'match' && get(fileImportReportStore)) {
            fileImportTotalFilesStore.set(1);
            fileImportCurrentIndexStore.set(1);
            fileImportCurrentFileStore.set(filePath);
            fileImportResultsStore.set({ succeeded: 1, failed: 0, skipped: 0, errors: [] });
            fileImportModeStore.set('completed');
            showFileImportModalStore.set(true);
        }
        await maybeAutoAnalyzeAfterImport();
    }
}

// extensionOf is the import format a batch records for a single file: its
// extension without the dot, "" when it has none.
function extensionOf(filePath) {
    const base = String(filePath).split('/').pop().split('\\').pop();
    const dot = base.lastIndexOf('.');
    return dot > 0 ? base.slice(dot + 1).toLowerCase() : '';
}

async function importSingleFileCore(filePath) {
    const lowerPath = filePath.toLowerCase();
    const isXGFile = lowerPath.endsWith('.xg');
    const isXGPFile = lowerPath.endsWith('.xgp');
    const isBGFFile = lowerPath.endsWith('.bgf');
    const isSGFFile = lowerPath.endsWith('.sgf');
    const isMATFile = lowerPath.endsWith('.mat');
    const isTXTFile = lowerPath.endsWith('.txt');

    if (isXGPFile) {
        logger.log('Importing XGP position file:', filePath);
        try {
            const posID = await ImportXGPPosition(filePath);
            setStatusBarMessage(tMsg('status.xgpPosImported', { posID }));
            await reloadPositions();
            await showImportedPosition(posID);
            return { type: 'position', id: posID };
        } catch (error) {
            logger.error('Error importing XGP position:', error);
            setStatusBarMessage(tMsg('status.errorImportingXgpPos', { error }));
            await ShowAlert('Error importing XGP position: ' + error);
        }
    } else if (isXGFile) {
        logger.log('Importing XG match file:', filePath);
        try {
            const matchID = await ImportXGMatch(filePath);
            setStatusBarMessage(tMsg('status.xgMatchImported', { matchID }));
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            dbMutationCounterStore.update((n) => n + 1);
            await reloadPositions();
            openPanel(PANEL.MATCH);
            activeTabStore.set('matches');
            return { type: 'match', id: matchID };
        } catch (error) {
            logger.error('Error importing XG match:', error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                setStatusBarMessage(tMsg('status.matchAlreadyImported'));
            } else {
                setStatusBarMessage(tMsg('status.errorImportingXgMatch', { error }));
                await ShowAlert('Error importing XG match: ' + error);
            }
        }
    } else if (isBGFFile) {
        logger.log('Importing BGF match file:', filePath);
        try {
            const matchID = await ImportBGFMatch(filePath);
            setStatusBarMessage(tMsg('status.bgblitzMatchImported', { matchID }));
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            dbMutationCounterStore.update((n) => n + 1);
            await reloadPositions();
            openPanel(PANEL.MATCH);
            activeTabStore.set('matches');
            return { type: 'match', id: matchID };
        } catch (error) {
            logger.error('Error importing BGF match:', error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                setStatusBarMessage(tMsg('status.matchAlreadyImported'));
            } else {
                setStatusBarMessage(tMsg('status.errorImportingBgblitzMatch', { error }));
                await ShowAlert('Error importing BGBlitz match: ' + error);
            }
        }
    } else if (isSGFFile || isMATFile) {
        const formatName = isSGFFile ? 'GnuBG SGF' : 'Jellyfish MAT';
        logger.log(`Importing ${formatName} match file:`, filePath);
        try {
            const matchID = await ImportGnuBGMatch(filePath);
            setStatusBarMessage(tMsg('status.matchImported', { formatName, matchID }));
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            dbMutationCounterStore.update((n) => n + 1);
            await reloadPositions();
            openPanel(PANEL.MATCH);
            activeTabStore.set('matches');
            return { type: 'match', id: matchID };
        } catch (error) {
            logger.error(`Error importing ${formatName} match:`, error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                setStatusBarMessage(tMsg('status.matchAlreadyImported'));
            } else {
                setStatusBarMessage(tMsg('status.errorImportingFormatMatch', { formatName, error }));
                await ShowAlert(`Error importing ${formatName} match: ` + error);
            }
        }
    } else if (isTXTFile) {
        return await importTxtFile(filePath);
    }
    return null;
}

async function importTxtFile(filePath) {
    const response = await ReadFileContent(filePath);
    if (response.error) {
        logger.error('Error reading file:', response.error);
        setStatusBarMessage(tMsg('status.errorReadingFile', { error: response.error }));
        return null;
    }
    const content = response.content;

    const isJellyfishTXT = content && /^\s*\d+\s+point\s+match\s*$/m.test(content);
    const isBGBlitzTXT = content && content.includes('Position-ID:');

    if (isJellyfishTXT) {
        logger.log('Importing Jellyfish TXT match file:', filePath);
        try {
            const matchID = await ImportGnuBGMatch(filePath);
            setStatusBarMessage(tMsg('status.jellyfishMatchImported', { matchID }));
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            dbMutationCounterStore.update((n) => n + 1);
            await reloadPositions();
            openPanel(PANEL.MATCH);
            activeTabStore.set('matches');
            return { type: 'match', id: matchID };
        } catch (error) {
            logger.error('Error importing Jellyfish TXT match:', error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                setStatusBarMessage(tMsg('status.matchAlreadyImported'));
            } else {
                setStatusBarMessage(tMsg('status.errorImportingJellyfish', { error }));
                await ShowAlert('Error importing Jellyfish TXT match: ' + error);
            }
        }
    } else if (isBGBlitzTXT) {
        logger.log('Importing BGBlitz TXT position:', filePath);
        try {
            const posID = await ImportBGFPosition(filePath);
            setStatusBarMessage(tMsg('status.bgblitzPosImported', { posID }));
            await reloadPositions();
            await showImportedPosition(posID);
            return { type: 'position', id: posID };
        } catch (error) {
            logger.error('Error importing BGBlitz position:', error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate') || errorStr.includes('already exists')) {
                setStatusBarMessage(tMsg('status.positionAlreadyExists'));
            } else {
                setStatusBarMessage(tMsg('status.errorImportingBgblitzPos', { error }));
                await ShowAlert('Error importing BGBlitz position: ' + error);
            }
        }
    } else {
        logger.log('File content:', content);
        const { positionData, parsedAnalysis } = await parsePositionText(content);
        positionStore.set({ ...positionData, id: 0, board: { ...positionData.board, bearoff: [15, 15] } });
        analysisStore.set({
            positionId: null,
            xgid: parsedAnalysis.xgid,
            player1: '',
            player2: '',
            analysisType: parsedAnalysis.analysisType,
            analysisEngineVersion: parsedAnalysis.analysisEngineVersion,
            checkerAnalysis: { moves: parsedAnalysis.checkerAnalysis },
            doublingCubeAnalysis: {
                analysisDepth: parsedAnalysis.doublingCubeAnalysis.analysisDepth || '',
                playerWinChances: parsedAnalysis.doublingCubeAnalysis.playerWinChances || 0,
                playerGammonChances: parsedAnalysis.doublingCubeAnalysis.playerGammonChances || 0,
                playerBackgammonChances: parsedAnalysis.doublingCubeAnalysis.playerBackgammonChances || 0,
                opponentWinChances: parsedAnalysis.doublingCubeAnalysis.opponentWinChances || 0,
                opponentGammonChances: parsedAnalysis.doublingCubeAnalysis.opponentGammonChances || 0,
                opponentBackgammonChances: parsedAnalysis.doublingCubeAnalysis.opponentBackgammonChances || 0,
                cubelessNoDoubleEquity: parsedAnalysis.doublingCubeAnalysis.cubelessNoDoubleEquity || 0,
                cubelessDoubleEquity: parsedAnalysis.doublingCubeAnalysis.cubelessDoubleEquity || 0,
                cubefulNoDoubleEquity: parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleEquity || 0,
                cubefulNoDoubleError: parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleError || 0,
                cubefulDoubleTakeEquity: parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeEquity || 0,
                cubefulDoubleTakeError: parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeError || 0,
                cubefulDoublePassEquity: parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassEquity || 0,
                cubefulDoublePassError: parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassError || 0,
                bestCubeAction: parsedAnalysis.doublingCubeAnalysis.bestCubeAction || '',
                wrongPassPercentage: parsedAnalysis.doublingCubeAnalysis.wrongPassPercentage || 0,
                wrongTakePercentage: parsedAnalysis.doublingCubeAnalysis.wrongTakePercentage || 0
            },
            allCubeAnalyses: [],
            playedMove: '',
            playedCubeAction: '',
            playedMoves: [],
            playedCubeActions: [],
            creationDate: '',
            lastModifiedDate: ''
        });
        const posID = await savePositionAndAnalysis(positionData, parsedAnalysis, tMsg('status.importedPositionSaved'));
        if (posID) {
            await showImportedPosition(posID);
            return { type: 'position', id: posID };
        }
    }
    return null;
}

async function importSingleFileBatch(filePath) {
    const lowerPath = filePath.toLowerCase();
    const isXGFile = lowerPath.endsWith('.xg');
    const isXGPFile = lowerPath.endsWith('.xgp');
    const isBGFFile = lowerPath.endsWith('.bgf');
    const isSGFFile = lowerPath.endsWith('.sgf');
    const isMATFile = lowerPath.endsWith('.mat');
    const isTXTFile = lowerPath.endsWith('.txt');

    if (isXGPFile) {
        const posID = await ImportXGPPosition(filePath);
        return { type: 'position', id: posID };
    } else if (isXGFile) {
        const matchID = await ImportXGMatch(filePath);
        return { type: 'match', id: matchID };
    } else if (isBGFFile) {
        const matchID = await ImportBGFMatch(filePath);
        return { type: 'match', id: matchID };
    } else if (isSGFFile || isMATFile) {
        const matchID = await ImportGnuBGMatch(filePath);
        return { type: 'match', id: matchID };
    } else if (isTXTFile) {
        return await importTxtFileBatch(filePath);
    }
    throw new Error('Unsupported file type');
}

async function importTxtFileBatch(filePath) {
    const response = await ReadFileContent(filePath);
    if (response.error) throw new Error(response.error);
    const content = response.content;

    const isJellyfishTXT = content && /^\s*\d+\s+point\s+match\s*$/m.test(content);
    const isBGBlitzTXT = content && content.includes('Position-ID:');

    if (isJellyfishTXT) {
        const matchID = await ImportGnuBGMatch(filePath);
        return { type: 'match', id: matchID };
    } else if (isBGBlitzTXT) {
        const posID = await ImportBGFPosition(filePath);
        return { type: 'position', id: posID };
    } else {
        const { positionData, parsedAnalysis } = await parsePositionText(content);
        positionStore.set({ ...positionData, id: 0, board: { ...positionData.board, bearoff: [15, 15] } });
        const posID = await savePositionAndAnalysis(positionData, parsedAnalysis, '', { reload: false });
        return { type: 'position', id: posID };
    }
}

// Returns { type: 'position', id } when only positions were imported (so the
// caller can show the last one), null when the batch contained any match.
export async function importMultipleFiles(files) {
    try {
        return await importMultipleFilesCore(files);
    } finally {
        await maybeAutoAnalyzeAfterImport();
    }
}

/**
 * Importe les fichiers qu'un dossier surveillé a vus arriver (#258, fiche
 * I.2) — le même import, sans fenêtre modale.
 *
 * L'utilisateur était en train d'étudier une position quand ses matchs sont
 * arrivés : lui reprendre l'écran serait le pire moment. Rien d'autre ne
 * change — mêmes doublons détectés, même lot d'import, même compte rendu,
 * même analyse automatique — et la barre de statut porte la notification.
 *
 * @param {string[]} files
 * @returns {Promise<{succeeded: number, skipped: number, failed: number} | null>}
 */
export async function importWatchedFiles(files) {
    try {
        await importMultipleFilesCore(files, { quiet: true });
    } finally {
        await maybeAutoAnalyzeAfterImport();
    }
    const r = get(fileImportResultsStore);
    return { succeeded: r.succeeded, skipped: r.skipped, failed: r.failed };
}

// beginImportBatch opens the batch the end-of-import report will be about
// (#257), returning 0 when one cannot be opened. Never fatal: the report is a
// convenience, and losing it must not cost the user the import.
async function beginImportBatch(source, format) {
    try {
        return await BeginImportBatch(source, format);
    } catch (error) {
        logger.error('could not open an import batch:', error);
        return 0;
    }
}

// finishImportBatch closes the batch and publishes its report, or clears the
// report when there is none to publish. Failures are swallowed for the same
// reason as above.
async function finishImportBatch(batchID) {
    if (!batchID) {
        fileImportReportStore.set(null);
        return;
    }
    try {
        await FinishImportBatch(batchID, {});
        fileImportReportStore.set(await ImportReport(batchID));
    } catch (error) {
        logger.error('could not read the import report:', error);
        fileImportReportStore.set(null);
    }
}

async function importMultipleFilesCore(files, { quiet = false } = {}) {
    fileImportCancelled = false;
    fileImportTotalFilesStore.set(files.length);
    fileImportCurrentIndexStore.set(0);
    fileImportCurrentFileStore.set('');
    fileImportResultsStore.set({ succeeded: 0, failed: 0, skipped: 0, errors: [] });
    fileImportReportStore.set(null);
    fileImportModeStore.set('importing');
    if (!quiet) showFileImportModalStore.set(true);

    // One batch for the whole selection: what the user asked for in one
    // gesture is one import, whether it was a folder or five dropped files.
    const batchID = await beginImportBatch(files.length === 1 ? files[0] : `${files.length} files`, 'mixed');

    let hadMatches = false;
    let lastPositionID = null;

    for (let i = 0; i < files.length; i++) {
        if (fileImportCancelled) break;
        const filePath = files[i];
        fileImportCurrentIndexStore.set(i + 1);
        fileImportCurrentFileStore.set(filePath);

        try {
            const result = await importSingleFileBatch(filePath);
            fileImportResultsStore.update((r) => ({ ...r, succeeded: r.succeeded + 1 }));
            if (result && result.type === 'match') hadMatches = true;
            if (result && result.type === 'position' && result.id) lastPositionID = result.id;
        } catch (error) {
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported') || errorStr.includes('duplicate') || errorStr.includes('already exists')) {
                fileImportResultsStore.update((r) => ({ ...r, skipped: r.skipped + 1 }));
            } else {
                fileImportResultsStore.update((r) => ({
                    ...r,
                    failed: r.failed + 1,
                    errors: [...r.errors, { file: filePath, message: errorStr.replace(/^Error:\s*/, '') }]
                }));
            }
        }
    }

    await finishImportBatch(batchID);
    fileImportModeStore.set('completed');

    // Mirrors the CLI batch importer (cli_import.go), which runs a plain
    // ANALYZE once after its own file loop: a GUI drag-drop/folder import can
    // add as many rows as a CLI batch, but until now only the CLI path
    // refreshed query-planner statistics afterwards — the GUI relied entirely
    // on ensureSearchStats' one-time backfill at file open, which never fires
    // again once a database has any stats at all, however stale (fiche-05
    // T7). Best-effort and skipped when nothing actually got imported.
    if (get(fileImportResultsStore).succeeded > 0) {
        RefreshSearchStatistics();
    }

    if (hadMatches) {
        matchPanelRefreshTriggerStore.update((n) => n + 1);
        dbMutationCounterStore.update((n) => n + 1);
    }
    await reloadPositions();

    const results = get(fileImportResultsStore);
    setStatusBarMessage(
        tMsg(quiet ? 'status.watchImportDone' : 'status.importDone', {
            succeeded: results.succeeded,
            skipped: results.skipped,
            failed: results.failed
        })
    );

    // A batch of positions only (no match): land on the last one with its analysis.
    // Never on a watched import: moving the board out from under someone who
    // did not ask for anything is exactly what "non blocking" rules out.
    if (!quiet && !hadMatches && lastPositionID) {
        await showImportedPosition(lastPositionID);
        return { type: 'position', id: lastPositionID };
    }
    return null;
}

export function handleFileImportCancel() {
    fileImportCancelled = true;
    showFileImportModalStore.set(false);
    fileImportModeStore.set('idle');
    setStatusBarMessage(tMsg('status.importCancelled'));
}

export function handleFileImportClose() {
    showFileImportModalStore.set(false);
    fileImportModeStore.set('idle');
}

export async function pastePosition() {
    try {
        await pastePositionCore();
    } finally {
        await maybeAutoAnalyzeAfterImport();
    }
}

async function pastePositionCore() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return;
    }
    logger.log('pastePosition');

    // EDIT (the search board) and EPC (the Eval panel) are the two modes whose
    // board is an editable scratch pad rather than a database record: there,
    // Ctrl-V drops the position ONTO the board instead of importing it into
    // the database — the paste side of the Ctrl-C that copied it.
    const mode = get(statusBarModeStore);
    if (mode === 'EDIT' || mode === 'EPC') {
        await pastePositionToBoard();
        return;
    }

    let result;
    try {
        result = await ClipboardGetText();
    } catch (error) {
        logger.error('Error pasting from clipboard:', error);
        return;
    }

    pastePositionTextStore.set(result);

    const isGnuBGMatch = result && /^\s*\d+\s+point\s+match\s*$/m.test(result) && /^\s*Game\s+\d+\s*$/m.test(result);
    const isBGBlitzTXT = result && result.includes('Position-ID:');

    if (isGnuBGMatch) {
        try {
            const matchID = await ImportGnuBGMatchFromText(result);
            setStatusBarMessage(tMsg('status.clipboardMatchImported', { matchID }));
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            dbMutationCounterStore.update((n) => n + 1);
            await reloadPositions();
            openPanel(PANEL.MATCH);
            activeTabStore.set('matches');
        } catch (error) {
            logger.error('Error pasting GnuBG match:', error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                setStatusBarMessage(tMsg('status.matchAlreadyImported'));
            } else {
                setStatusBarMessage(tMsg('status.errorImportingClipboardMatch', { error }));
            }
        }
    } else if (isBGBlitzTXT) {
        try {
            const posID = await ImportBGFPositionFromText(result);
            setStatusBarMessage(tMsg('status.bgblitzPosPasted', { posID }));
            await reloadPositions();
            await showImportedPosition(posID);
        } catch (error) {
            logger.error('Error pasting BGBlitz position:', error);
            setStatusBarMessage(tMsg('status.errorPastingBgblitzPos', { error }));
        }
    } else {
        // An OGID needs no branch here: parser.ParsePosition reads it like it
        // reads an XGID, so a pasted OpenGammon identifier lands in the
        // database through exactly this call (#260).
        const { positionData, parsedAnalysis } = await parsePositionText(result);
        const posID = await savePositionAndAnalysis(positionData, parsedAnalysis, tMsg('status.pastedPositionSaved'));
        if (posID) await showImportedPosition(posID);
    }
    statusBarModeStore.set('NORMAL');
}

async function pastePositionToBoard() {
    try {
        const clipboardText = await ClipboardGetText();
        if (clipboardText && clipboardText.includes('XGID=')) {
            try {
                const { positionData } = await parsePositionOnly(clipboardText);
                applyPositionToBoard(positionData);
                setStatusBarMessage(tMsg('status.positionPastedClipboard'));
                return;
            } catch (e) {
                logger.log('Clipboard text has XGID but parse failed, trying internal clipboard:', e);
            }
        }

        if (clipboardText && !clipboardText.includes('XGID=') && (await LooksLikeOGID(clipboardText))) {
            try {
                const { positionData } = await parsePositionOnly(clipboardText);
                applyPositionToBoard(positionData);
                setStatusBarMessage(tMsg('status.positionPastedClipboard'));
                return;
            } catch (e) {
                logger.log('Clipboard text looks like an OGID but decoding failed:', e);
            }
        }

        const clipboardPosition = get(clipboardPositionStore);
        if (clipboardPosition) {
            applyPositionToBoard(clipboardPosition);
            setStatusBarMessage(tMsg('status.positionPasted'));
            return;
        }

        setStatusBarMessage(tMsg('status.noPositionToPasteHint'));
    } catch (error) {
        const clipboardPosition = get(clipboardPositionStore);
        if (clipboardPosition) {
            applyPositionToBoard(clipboardPosition);
            setStatusBarMessage(tMsg('status.positionPasted'));
            return;
        }
        logger.error('Error pasting position to board:', error);
        setStatusBarMessage(tMsg('status.noPositionToPaste'));
    }
}

// parsePositionOnly parses text when only the board is wanted. When the
// backend refuses the analysis block (an XG language it does not know), the
// bare XGID line is parsed instead: the board is in the XGID, and pasting it
// onto the board must not depend on the language the analysis was written in.
async function parsePositionOnly(text) {
    try {
        return await parsePositionText(text);
    } catch (e) {
        if (!String(e).includes('analysis block not recognised')) throw e;
        const xgidLine = text.split(/\r?\n/).find((line) => line.trim().startsWith('XGID='));
        if (!xgidLine) throw e;
        return parsePositionText(xgidLine.trim());
    }
}

function applyPositionToBoard(posData) {
    positionStore.update((pos) => {
        pos.board.points = posData.board.points.map((p) => ({ checkers: p.checkers, color: p.color }));
        pos.board.bearoff = [...posData.board.bearoff];
        pos.cube = { owner: posData.cube.owner, value: posData.cube.value };
        pos.dice = [...posData.dice];
        pos.score = [...posData.score];
        pos.player_on_roll = posData.player_on_roll;
        pos.decision_type = posData.decision_type;
        if (posData.has_jacoby !== undefined) pos.has_jacoby = posData.has_jacoby;
        if (posData.has_beaver !== undefined) pos.has_beaver = posData.has_beaver;
        return pos;
    });
}

// ── Drag & Drop ────────────────────────────────────────────────

export async function classifyDroppedFiles(paths) {
    const dbFiles = [];
    const importFiles = [];
    const folders = [];
    const unsupported = [];
    for (const p of paths) {
        const isDir = await IsDirectory(p);
        if (isDir) {
            folders.push(p);
        } else {
            const ext = p.toLowerCase().split('.').pop();
            if (ext === 'db') {
                dbFiles.push(p);
            } else if (['txt', 'xg', 'xgp', 'sgf', 'mat', 'bgf'].includes(ext)) {
                importFiles.push(p);
            } else {
                unsupported.push(p);
            }
        }
    }
    return { dbFiles, importFiles, folders, unsupported };
}

export async function handleDbFileDrop(dbPath) {
    const { openDatabaseByPath } = await import('./databaseService.js');
    if (!get(databasePathStore)) {
        await openDatabaseByPath(dbPath);
    } else {
        const filename = dbPath.split('/').pop().split('\\').pop();
        try {
            const answer = await ShowQuestionDialog('Database already open', `A database is already open.\n\nWhat would you like to do with "${filename}"?`, ['Open', 'Merge', 'Cancel'], 'Merge');
            if (answer === 'Open') {
                await openDatabaseByPath(dbPath);
            } else if (answer === 'Merge') {
                await importDatabaseByPath(dbPath);
            }
        } catch (error) {
            logger.error('Error in DB drop dialog:', error);
            setStatusBarMessage(tMsg('status.errorHandlingDroppedDb'));
        }
    }
}

export async function handleFileDrop(x, y, paths) {
    logger.log('Files dropped:', paths);

    if (!paths || paths.length === 0) return;

    const { dbFiles, importFiles, folders, unsupported } = await classifyDroppedFiles(paths);

    if (unsupported.length > 0) {
        const exts = [...new Set(unsupported.map((p) => '.' + p.split('.').pop()))].join(', ');
        logger.warn('Unsupported file extensions dropped:', exts);
    }

    if (dbFiles.length > 0) {
        await handleDbFileDrop(dbFiles[0]);
        if (importFiles.length === 0 && folders.length === 0) return;
    }

    let allImportFiles = [...importFiles];
    for (const folder of folders) {
        try {
            const folderFiles = await CollectImportableFiles(folder);
            if (folderFiles && folderFiles.length > 0) {
                allImportFiles = allImportFiles.concat(folderFiles);
            }
        } catch (error) {
            logger.error('Error collecting files from folder:', folder, error);
        }
    }

    if (allImportFiles.length > 0) {
        if (!get(databasePathStore)) {
            setStatusBarMessage(tMsg('status.noDbOpenedBeforeImport'));
            await ShowAlert('No database opened. Please open or drop a database file first.');
            return;
        }

        if (allImportFiles.length === 1) {
            await importSingleFile(allImportFiles[0]);
        } else {
            await importMultipleFiles(allImportFiles);
        }
    }

    if (dbFiles.length === 0 && allImportFiles.length === 0 && unsupported.length > 0) {
        const exts = [...new Set(unsupported.map((p) => '.' + p.split('.').pop()))].join(', ');
        setStatusBarMessage(tMsg('status.unsupportedFileType', { exts }));
    } else if (folders.length > 0 && allImportFiles.length === 0 && dbFiles.length === 0) {
        setStatusBarMessage(tMsg('status.noImportableDropped'));
    }
}
// ── Position text parser ────────────────────────────────────────
//
// Parsing lives in the Go backend now (pkg/blunderdb/parser, exposed as
// ParsePositionText over Wails) so the GUI, CLI and server share one
// implementation and can't drift — see testdata/parse_corpus.json and its
// dual contract tests. parsePositionText() calls the backend and reshapes the
// result into the legacy { positionData, parsedAnalysis } shape the callers
// already consume (parsedAnalysis.checkerAnalysis as a bare array,
// doublingCubeAnalysis as an object, comment inline).
export async function parsePositionText(content) {
    const result = await ParsePositionText(content);
    const a = result.analysis || {};
    const moves = a.checkerAnalysis && Array.isArray(a.checkerAnalysis.moves) ? a.checkerAnalysis.moves : [];
    return {
        positionData: result.position,
        parsedAnalysis: {
            xgid: a.xgid || '',
            analysisType: a.analysisType || '',
            analysisEngineVersion: a.analysisEngineVersion || '',
            checkerAnalysis: moves,
            doublingCubeAnalysis: a.doublingCubeAnalysis || {},
            comment: result.comment || ''
        }
    };
}

// openImportedPosition lands on one of the report's worst decisions and closes
// the modal — the whole point of listing them is being able to go and look.
export async function openImportedPosition(positionID) {
    showFileImportModalStore.set(false);
    fileImportModeStore.set('idle');
    await showImportedPosition(positionID);
}

// analyzeRemainingAfterImport starts the evaluator on what the import brought
// in without an analysis. It is the report's one action: the panel says "12
// positions with no analysis" and this is what the user does about it.
//
// The batch is the whole database's unanalysed set, not the import's alone —
// StartGammonNetBatch has no narrowing — which is honest rather than
// surprising: a user who asks to analyse after an import wants the gaps gone.
export async function analyzeRemainingAfterImport() {
    showFileImportModalStore.set(false);
    fileImportModeStore.set('idle');
    try {
        const [ply, pruneK] = await Promise.all([GetGammonNetAnalysisPly(), GetGammonNetPruneK()]);
        await StartGammonNetBatch(ply, pruneK, 0);
    } catch (error) {
        logger.error('could not start the analysis after import:', error);
        setStatusBarMessage(tMsg('status.errorStartingAnalysis', { error }));
    }
}

/**
 * Importe une position depuis un identifiant TAPÉ, plutôt que collé (#262,
 * fiche I.6).
 *
 * Le presse-papier marche déjà, et c'est le geste courant. Il ne marche pas
 * quand l'identifiant arrive d'ailleurs : d'un message, d'un forum lu dans un
 * terminal, d'un script. La commande `import XGID=…` couvre ce cas-là, avec
 * exactement le même chemin — même analyse du texte, même déduplication, même
 * ouverture de la position importée.
 *
 * OGID est reconnu depuis que sa grammaire a été relevée sur des positions
 * réelles (#260), et par le même lecteur : parser.ParsePosition lit un OGID
 * comme il lit un XGID. Il ne reste ici qu'un aiguillage — savoir si le texte
 * EST un identifiant, un OGID n'ayant pas de préfixe obligatoire. Un OGID ne
 * porte qu'une position, jamais d'évaluation : la fiche d'analyse arrive vide,
 * comme pour un XGID nu. Ce qui n'est ni l'un ni l'autre est refusé en le
 * disant, plutôt que deviné.
 *
 * @param {string} text
 */
export async function importIdentifier(text) {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return null;
    }
    const trimmed = (text || '').trim();
    if (!trimmed) {
        setStatusBarMessage(tMsg('status.importIdentifierEmpty'));
        return null;
    }
    if (!trimmed.includes('XGID=') && !(await LooksLikeOGID(trimmed))) {
        setStatusBarMessage(tMsg('status.importIdentifierUnknown'));
        return null;
    }
    try {
        const { positionData, parsedAnalysis } = await parsePositionText(trimmed);
        const posID = await savePositionAndAnalysis(positionData, parsedAnalysis, tMsg('status.importIdentifierSaved'));
        if (posID) await showImportedPosition(posID);
        return posID;
    } catch (error) {
        logger.error('could not import the pasted identifier:', error);
        setStatusBarMessage(tMsg('status.importIdentifierFailed', { error }));
        return null;
    } finally {
        await maybeAutoAnalyzeAfterImport();
    }
}

/**
 * Enrichit un match depuis un fichier (#262, fiche I.6).
 *
 * Il n'y a rien de nouveau sous ce bouton, et c'est le propos : réimporter le
 * même match dans un autre format l'enrichit déjà en place — la déduplication
 * par empreinte canonique reconnaît qu'il s'agit du même match et fusionne les
 * analyses et les commentaires du second fichier dans le premier. Ce que le
 * bouton apporte, c'est de le rendre trouvable : personne ne devine qu'un
 * import est aussi un enrichissement.
 *
 * Le compte rendu qui suit dit lequel des deux a eu lieu — « enrichis : 1 »
 * plutôt que « importés : 1 ».
 */
export async function enrichMatchFromFile() {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return;
    }
    let files;
    try {
        files = await OpenPositionFilesDialog();
    } catch (error) {
        logger.error('could not choose a file to enrich from:', error);
        return;
    }
    if (!files || files.length === 0) return;
    await importMultipleFiles(files);
}
