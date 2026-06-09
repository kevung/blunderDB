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
    IsDirectory
} from '../../wailsjs/go/gui/App.js';
import {
    SavePosition,
    SaveAnalysis,
    PositionExists,
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
    ParsePositionText
} from '../../wailsjs/go/database/Database.js';
import { ClipboardGetText } from '../../wailsjs/runtime/runtime.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, positionsStore, pastePositionTextStore, matchContextStore, clipboardPositionStore } from '../stores/positionStore.js';
import { analysisStore } from '../stores/analysisStore.js';
import { currentPositionIndexStore, statusBarModeStore, commentTextStore, openPanel, PANEL, matchPanelRefreshTriggerStore, dbMutationCounterStore } from '../stores/uiStore.js';
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
    fileImportResultsStore
} from '../stores/importModalStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';

// Pending import path (module-level)
let pendingImportPath = null;
let fileImportCancelled = false;

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

        const { loadAllPositions } = await import('./positionService.js');
        await loadAllPositions();
    } catch (error) {
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

export async function savePositionAndAnalysis(positionData, parsedAnalysis, successMessage) {
    if (Array.isArray(parsedAnalysis.checkerAnalysis)) {
        parsedAnalysis.checkerAnalysis = { moves: parsedAnalysis.checkerAnalysis };
    }

    delete parsedAnalysis.creationDate;
    delete parsedAnalysis.lastModifiedDate;

    const positionExistsResult = await PositionExists(positionData);
    if (positionExistsResult.exists) {
        logger.log('Position already exists with ID:', positionExistsResult.id);
        try {
            parsedAnalysis.positionId = positionExistsResult.id;
            await SaveAnalysis(positionExistsResult.id, parsedAnalysis);

            let existingComment = await LoadComment(positionExistsResult.id);
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

            await SaveComment(positionExistsResult.id, mergedComment);
            logger.log('Analysis and comment updated for position ID:', positionExistsResult.id);
            setStatusBarMessage(tMsg('status.positionMerged'));

            const positions = get(positionsStore);
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(positions.findIndex((pos) => pos.id === positionExistsResult.id));
            commentTextStore.set(mergedComment);
        } catch (error) {
            logger.error('Error updating analysis and comment:', error);
            setStatusBarMessage(tMsg('status.errorUpdatingAnalysisComment'));
        }
        return;
    }

    try {
        const positionID = await SavePosition(positionData);
        logger.log('Position saved with ID:', positionID);

        positionData.id = positionID;
        parsedAnalysis.positionId = positionID;
        await SaveAnalysis(positionID, parsedAnalysis);
        await SaveComment(positionID, parsedAnalysis.comment);
        logger.log('Analysis and comment saved for position ID:', positionID);

        const { loadAllPositions } = await import('./positionService.js');
        await loadAllPositions();
        setStatusBarMessage(successMessage);
    } catch (error) {
        logger.error('Error saving position, analysis, and comment:', error);
        setStatusBarMessage(tMsg('status.errorSavingPosition'));
    }
}

export async function importPosition() {
    const wasMatchMode = get(statusBarModeStore) === 'MATCH';
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return;
    }
    try {
        const files = await OpenPositionFilesDialog();
        if (!files || files.length === 0) return;

        if (files.length === 1) {
            await importSingleFile(files[0]);
        } else {
            await importMultipleFiles(files);
        }
    } catch (error) {
        logger.error('Error importing position:', error);
    } finally {
        if (wasMatchMode) {
            matchContextStore.set({
                isMatchMode: false,
                matchID: null,
                movePositions: [],
                currentIndex: 0,
                player1Name: '',
                player2Name: ''
            });
            const { loadAllPositions } = await import('./positionService.js');
            loadAllPositions();
        }
        statusBarModeStore.set('NORMAL');
    }
}

export async function importFolder() {
    const wasMatchMode = get(statusBarModeStore) === 'MATCH';
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return;
    }
    try {
        const dirPath = await OpenPositionFolderDialog();
        if (!dirPath) return;

        const files = await CollectImportableFiles(dirPath);
        if (!files || files.length === 0) {
            setStatusBarMessage(tMsg('status.noImportableFolder'));
            return;
        }

        await importMultipleFiles(files);
    } catch (error) {
        logger.error('Error importing folder:', error);
    } finally {
        if (wasMatchMode) {
            matchContextStore.set({
                isMatchMode: false,
                matchID: null,
                movePositions: [],
                currentIndex: 0,
                player1Name: '',
                player2Name: ''
            });
            const { loadAllPositions } = await import('./positionService.js');
            loadAllPositions();
        }
        statusBarModeStore.set('NORMAL');
    }
}

export async function importSingleFile(filePath) {
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
            const { loadAllPositions } = await import('./positionService.js');
            await loadAllPositions();
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
            openPanel(PANEL.MATCH);
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
            openPanel(PANEL.MATCH);
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
            openPanel(PANEL.MATCH);
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
        await importTxtFile(filePath);
    }
}

async function importTxtFile(filePath) {
    const response = await ReadFileContent(filePath);
    if (response.error) {
        logger.error('Error reading file:', response.error);
        setStatusBarMessage(tMsg('status.errorReadingFile', { error: response.error }));
        return;
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
            openPanel(PANEL.MATCH);
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
            const { loadAllPositions } = await import('./positionService.js');
            await loadAllPositions();
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
        await savePositionAndAnalysis(positionData, parsedAnalysis, tMsg('status.importedPositionSaved'));
    }
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
        await savePositionAndAnalysis(positionData, parsedAnalysis, '');
        return { type: 'position', id: 0 };
    }
}

export async function importMultipleFiles(files) {
    fileImportCancelled = false;
    fileImportTotalFilesStore.set(files.length);
    fileImportCurrentIndexStore.set(0);
    fileImportCurrentFileStore.set('');
    fileImportResultsStore.set({ succeeded: 0, failed: 0, skipped: 0, errors: [] });
    fileImportModeStore.set('importing');
    showFileImportModalStore.set(true);

    let hadMatches = false;

    for (let i = 0; i < files.length; i++) {
        if (fileImportCancelled) break;
        const filePath = files[i];
        fileImportCurrentIndexStore.set(i + 1);
        fileImportCurrentFileStore.set(filePath);

        try {
            const result = await importSingleFileBatch(filePath);
            fileImportResultsStore.update((r) => ({ ...r, succeeded: r.succeeded + 1 }));
            if (result && result.type === 'match') hadMatches = true;
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

    fileImportModeStore.set('completed');

    if (hadMatches) {
        matchPanelRefreshTriggerStore.update((n) => n + 1);
        dbMutationCounterStore.update((n) => n + 1);
    }
    const { loadAllPositions } = await import('./positionService.js');
    await loadAllPositions();

    const results = get(fileImportResultsStore);
    setStatusBarMessage(tMsg('status.importDone', { succeeded: results.succeeded, skipped: results.skipped, failed: results.failed }));
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
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return;
    }
    logger.log('pastePosition');

    if (get(statusBarModeStore) === 'EDIT') {
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
            openPanel(PANEL.MATCH);
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
            const { loadAllPositions } = await import('./positionService.js');
            await loadAllPositions();
        } catch (error) {
            logger.error('Error pasting BGBlitz position:', error);
            setStatusBarMessage(tMsg('status.errorPastingBgblitzPos', { error }));
        }
    } else {
        const { positionData, parsedAnalysis } = await parsePositionText(result);
        await savePositionAndAnalysis(positionData, parsedAnalysis, tMsg('status.pastedPositionSaved'));
    }
    statusBarModeStore.set('NORMAL');
}

async function pastePositionToBoard() {
    try {
        const clipboardText = await ClipboardGetText();
        if (clipboardText && clipboardText.includes('XGID=')) {
            try {
                const { positionData } = await parsePositionText(clipboardText);
                applyPositionToBoard(positionData);
                setStatusBarMessage(tMsg('status.positionPastedClipboard'));
                return;
            } catch (e) {
                logger.log('Clipboard text has XGID but parse failed, trying internal clipboard:', e);
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
