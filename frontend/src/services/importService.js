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
} from '../../wailsjs/go/main/App.js';
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
    ImportXGPPosition
} from '../../wailsjs/go/main/Database.js';
import { ClipboardGetText } from '../../wailsjs/runtime/runtime.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, positionsStore, pastePositionTextStore, matchContextStore, clipboardPositionStore } from '../stores/positionStore.js';
import { analysisStore } from '../stores/analysisStore.js';
import { currentPositionIndexStore, statusBarModeStore, commentTextStore, openPanel, PANEL, matchPanelRefreshTriggerStore } from '../stores/uiStore.js';
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

// Pending import path (module-level)
let pendingImportPath = null;
let fileImportCancelled = false;

export async function importDatabase() {
    console.log('importDatabase');

    if (!get(databasePathStore)) {
        setStatusBarMessage('No database opened. Please open a database first.');
        return;
    }

    try {
        const importFilePath = await OpenImportDatabaseDialog();
        if (!importFilePath) {
            console.log('No import database selected');
            return;
        }

        console.log('Analyzing import from:', importFilePath);

        showImportProgressModalStore.set(true);
        importModalModeStore.set('analyzing');
        pendingImportPath = importFilePath;

        try {
            const analysis = await AnalyzeImportDatabase(importFilePath);
            console.log('Import analysis:', analysis);

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
        console.error('Error analyzing import:', error);
        setStatusBarMessage(`Error analyzing import: ${error}`);
        await ShowAlert(`Error analyzing import: ${error}`);
        statusBarModeStore.set('NORMAL');
    }
}

export async function handleImportCommit() {
    if (!pendingImportPath) {
        console.error('No pending import path');
        return;
    }

    console.log('Committing import from:', pendingImportPath);
    importModalModeStore.set('committing');

    try {
        const result = await CommitImportDatabase(pendingImportPath);
        console.log('Import result:', result);

        importResultStore.set({
            added: result.added,
            merged: result.merged,
            skipped: result.skipped,
            total: result.total
        });
        importModalModeStore.set('completed');

        setStatusBarMessage(`Import completed: ${result.added} added, ${result.merged} merged, ${result.skipped} skipped`);

        const { loadAllPositions } = await import('./positionService.js');
        await loadAllPositions();
    } catch (error) {
        console.error('Error committing import:', error);
        showImportProgressModalStore.set(false);
        setStatusBarMessage(`Error committing import: ${error}`);
        await ShowAlert(`Error committing import: ${error}`);
        statusBarModeStore.set('NORMAL');
    } finally {
        pendingImportPath = null;
    }
}

export function handleImportCancel() {
    console.log('Import cancelled by user');

    if (get(importModalModeStore) === 'committing') {
        console.log('Aborting ongoing commit transaction');
        CancelImport().catch((err) => {
            console.error('Error calling CancelImport:', err);
        });
    }

    showImportProgressModalStore.set(false);
    pendingImportPath = null;
    importModalModeStore.set('analyzing');
    setStatusBarMessage('Import cancelled');
    statusBarModeStore.set('NORMAL');
}

export function handleImportClose() {
    console.log('Import completed and closed');
    showImportProgressModalStore.set(false);
    pendingImportPath = null;
    importModalModeStore.set('analyzing');
    statusBarModeStore.set('NORMAL');
}

export async function importDatabaseByPath(importFilePath) {
    if (!get(databasePathStore)) {
        setStatusBarMessage('No database opened. Please open a database first.');
        return;
    }

    try {
        console.log('Analyzing import from:', importFilePath);

        showImportProgressModalStore.set(true);
        importModalModeStore.set('analyzing');
        pendingImportPath = importFilePath;

        try {
            const analysis = await AnalyzeImportDatabase(importFilePath);
            console.log('Import analysis:', analysis);

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
        console.error('Error analyzing import:', error);
        setStatusBarMessage(`Error analyzing import: ${error}`);
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
        console.log('Position already exists with ID:', positionExistsResult.id);
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
            console.log('Analysis and comment updated for position ID:', positionExistsResult.id);
            setStatusBarMessage('Position already exists, analysis and comment merged');

            const positions = get(positionsStore);
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(positions.findIndex((pos) => pos.id === positionExistsResult.id));
            commentTextStore.set(mergedComment);
        } catch (error) {
            console.error('Error updating analysis and comment:', error);
            setStatusBarMessage('Error updating analysis and comment');
        }
        return;
    }

    try {
        const positionID = await SavePosition(positionData);
        console.log('Position saved with ID:', positionID);

        positionData.id = positionID;
        parsedAnalysis.positionId = positionID;
        await SaveAnalysis(positionID, parsedAnalysis);
        await SaveComment(positionID, parsedAnalysis.comment);
        console.log('Analysis and comment saved for position ID:', positionID);

        const { loadAllPositions } = await import('./positionService.js');
        await loadAllPositions();
        setStatusBarMessage(successMessage);
    } catch (error) {
        console.error('Error saving position, analysis, and comment:', error);
        setStatusBarMessage('Error saving position, analysis, and comment');
    }
}

export async function importPosition() {
    const wasMatchMode = get(statusBarModeStore) === 'MATCH';
    if (!get(databasePathStore)) {
        setStatusBarMessage('No database opened');
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
        console.error('Error importing position:', error);
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
        setStatusBarMessage('No database opened');
        return;
    }
    try {
        const dirPath = await OpenPositionFolderDialog();
        if (!dirPath) return;

        const files = await CollectImportableFiles(dirPath);
        if (!files || files.length === 0) {
            setStatusBarMessage('No importable files found in folder');
            return;
        }

        await importMultipleFiles(files);
    } catch (error) {
        console.error('Error importing folder:', error);
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
        console.log('Importing XGP position file:', filePath);
        try {
            const posID = await ImportXGPPosition(filePath);
            setStatusBarMessage(`XGP position imported successfully (ID: ${posID})`);
            const { loadAllPositions } = await import('./positionService.js');
            await loadAllPositions();
        } catch (error) {
            console.error('Error importing XGP position:', error);
            setStatusBarMessage('Error importing XGP position: ' + error);
            await ShowAlert('Error importing XGP position: ' + error);
        }
    } else if (isXGFile) {
        console.log('Importing XG match file:', filePath);
        try {
            const matchID = await ImportXGMatch(filePath);
            setStatusBarMessage(`XG match imported successfully (ID: ${matchID})`);
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            openPanel(PANEL.MATCH);
        } catch (error) {
            console.error('Error importing XG match:', error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                setStatusBarMessage('This match has already been imported');
            } else {
                setStatusBarMessage('Error importing XG match: ' + error);
                await ShowAlert('Error importing XG match: ' + error);
            }
        }
    } else if (isBGFFile) {
        console.log('Importing BGF match file:', filePath);
        try {
            const matchID = await ImportBGFMatch(filePath);
            setStatusBarMessage(`BGBlitz match imported successfully (ID: ${matchID})`);
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            openPanel(PANEL.MATCH);
        } catch (error) {
            console.error('Error importing BGF match:', error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                setStatusBarMessage('This match has already been imported');
            } else {
                setStatusBarMessage('Error importing BGBlitz match: ' + error);
                await ShowAlert('Error importing BGBlitz match: ' + error);
            }
        }
    } else if (isSGFFile || isMATFile) {
        const formatName = isSGFFile ? 'GnuBG SGF' : 'Jellyfish MAT';
        console.log(`Importing ${formatName} match file:`, filePath);
        try {
            const matchID = await ImportGnuBGMatch(filePath);
            setStatusBarMessage(`${formatName} match imported successfully (ID: ${matchID})`);
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            openPanel(PANEL.MATCH);
        } catch (error) {
            console.error(`Error importing ${formatName} match:`, error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                setStatusBarMessage('This match has already been imported');
            } else {
                setStatusBarMessage(`Error importing ${formatName} match: ` + error);
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
        console.error('Error reading file:', response.error);
        setStatusBarMessage('Error reading file: ' + response.error);
        return;
    }
    const content = response.content;

    const isJellyfishTXT = content && /^\s*\d+\s+point\s+match\s*$/m.test(content);
    const isBGBlitzTXT = content && content.includes('Position-ID:');

    if (isJellyfishTXT) {
        console.log('Importing Jellyfish TXT match file:', filePath);
        try {
            const matchID = await ImportGnuBGMatch(filePath);
            setStatusBarMessage(`Jellyfish TXT match imported successfully (ID: ${matchID})`);
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            openPanel(PANEL.MATCH);
        } catch (error) {
            console.error('Error importing Jellyfish TXT match:', error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                setStatusBarMessage('This match has already been imported');
            } else {
                setStatusBarMessage('Error importing Jellyfish TXT match: ' + error);
                await ShowAlert('Error importing Jellyfish TXT match: ' + error);
            }
        }
    } else if (isBGBlitzTXT) {
        console.log('Importing BGBlitz TXT position:', filePath);
        try {
            const posID = await ImportBGFPosition(filePath);
            setStatusBarMessage(`BGBlitz position imported successfully (ID: ${posID})`);
            const { loadAllPositions } = await import('./positionService.js');
            await loadAllPositions();
        } catch (error) {
            console.error('Error importing BGBlitz position:', error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate') || errorStr.includes('already exists')) {
                setStatusBarMessage('This position already exists');
            } else {
                setStatusBarMessage('Error importing BGBlitz position: ' + error);
                await ShowAlert('Error importing BGBlitz position: ' + error);
            }
        }
    } else {
        console.log('File content:', content);
        const { positionData, parsedAnalysis } = parsePosition(content);
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
        await savePositionAndAnalysis(positionData, parsedAnalysis, 'Imported position and analysis saved successfully');
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
        const { positionData, parsedAnalysis } = parsePosition(content);
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
    }
    const { loadAllPositions } = await import('./positionService.js');
    await loadAllPositions();

    const results = get(fileImportResultsStore);
    const msg = `Import done: ${results.succeeded} imported, ${results.skipped} skipped, ${results.failed} failed`;
    setStatusBarMessage(msg);
}

export function handleFileImportCancel() {
    fileImportCancelled = true;
    showFileImportModalStore.set(false);
    fileImportModeStore.set('idle');
    setStatusBarMessage('Import cancelled');
}

export function handleFileImportClose() {
    showFileImportModalStore.set(false);
    fileImportModeStore.set('idle');
}

export async function pastePosition() {
    if (!get(databasePathStore)) {
        setStatusBarMessage('No database opened');
        return;
    }
    console.log('pastePosition');

    if (get(statusBarModeStore) === 'EDIT') {
        await pastePositionToBoard();
        return;
    }

    let result;
    try {
        result = await ClipboardGetText();
    } catch (error) {
        console.error('Error pasting from clipboard:', error);
        return;
    }

    pastePositionTextStore.set(result);

    const isGnuBGMatch = result && /^\s*\d+\s+point\s+match\s*$/m.test(result) && /^\s*Game\s+\d+\s*$/m.test(result);
    const isBGBlitzTXT = result && result.includes('Position-ID:');

    if (isGnuBGMatch) {
        try {
            const matchID = await ImportGnuBGMatchFromText(result);
            setStatusBarMessage(`Match imported from clipboard successfully (ID: ${matchID})`);
            matchPanelRefreshTriggerStore.update((n) => n + 1);
            openPanel(PANEL.MATCH);
        } catch (error) {
            console.error('Error pasting GnuBG match:', error);
            const errorStr = String(error);
            if (errorStr.includes('duplicate match') || errorStr.includes('already been imported')) {
                setStatusBarMessage('This match has already been imported');
            } else {
                setStatusBarMessage('Error importing match from clipboard: ' + error);
            }
        }
    } else if (isBGBlitzTXT) {
        try {
            const posID = await ImportBGFPositionFromText(result);
            setStatusBarMessage(`BGBlitz position pasted successfully (ID: ${posID})`);
            const { loadAllPositions } = await import('./positionService.js');
            await loadAllPositions();
        } catch (error) {
            console.error('Error pasting BGBlitz position:', error);
            setStatusBarMessage('Error pasting BGBlitz position: ' + error);
        }
    } else {
        const { positionData, parsedAnalysis } = parsePosition(result);
        await savePositionAndAnalysis(positionData, parsedAnalysis, 'Pasted position and analysis saved successfully');
    }
    statusBarModeStore.set('NORMAL');
}

async function pastePositionToBoard() {
    try {
        const clipboardText = await ClipboardGetText();
        if (clipboardText && clipboardText.includes('XGID=')) {
            try {
                const { positionData } = parsePosition(clipboardText);
                applyPositionToBoard(positionData);
                setStatusBarMessage('Position pasted to board from clipboard');
                return;
            } catch (e) {
                console.log('Clipboard text has XGID but parse failed, trying internal clipboard:', e);
            }
        }

        const clipboardPosition = get(clipboardPositionStore);
        if (clipboardPosition) {
            applyPositionToBoard(clipboardPosition);
            setStatusBarMessage('Position pasted to board');
            return;
        }

        setStatusBarMessage('No position to paste (use Ctrl-C to copy a position first)');
    } catch (error) {
        const clipboardPosition = get(clipboardPositionStore);
        if (clipboardPosition) {
            applyPositionToBoard(clipboardPosition);
            setStatusBarMessage('Position pasted to board');
            return;
        }
        console.error('Error pasting position to board:', error);
        setStatusBarMessage('No position to paste');
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
            console.error('Error in DB drop dialog:', error);
            setStatusBarMessage('Error handling dropped database');
        }
    }
}

export async function handleFileDrop(x, y, paths) {
    console.log('Files dropped:', paths);

    if (!paths || paths.length === 0) return;

    const { dbFiles, importFiles, folders, unsupported } = await classifyDroppedFiles(paths);

    if (unsupported.length > 0) {
        const exts = [...new Set(unsupported.map((p) => '.' + p.split('.').pop()))].join(', ');
        console.warn('Unsupported file extensions dropped:', exts);
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
            console.error('Error collecting files from folder:', folder, error);
        }
    }

    if (allImportFiles.length > 0) {
        if (!get(databasePathStore)) {
            setStatusBarMessage('No database opened. Please open a database before importing files.');
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
        setStatusBarMessage(`Unsupported file type(s): ${exts}`);
    } else if (folders.length > 0 && allImportFiles.length === 0 && dbFiles.length === 0) {
        setStatusBarMessage('No importable files found in dropped folder(s)');
    }
}

// ── Position text parser ────────────────────────────────────────

export function parsePosition(fileContent) {
    if (!fileContent || fileContent.trim().length === 0) {
        throw new Error('File is empty or invalid.');
    }

    let normalizedContent = fileContent.replace(/\r\n|\r/g, '\n').trim();
    const lines = normalizedContent.split('\n').map((line) => line.trim());

    const isFrench = normalizedContent.includes('Joueur') || normalizedContent.includes('Adversaire') || normalizedContent.includes('Videau');
    const isJapanese = normalizedContent.includes('プレーヤー') || normalizedContent.includes('対戦相手') || normalizedContent.includes('キューブ');
    const isInternalCheckerAnalysisFormat = normalizedContent.includes('Analysis:\nChecker Move Analysis:');
    const isInternalDoublingAnalysisFormat = normalizedContent.includes('Analysis:\nDoubling Cube Analysis:');
    const isGerman = normalizedContent.includes('Spieler') || normalizedContent.includes('Gegner') || normalizedContent.includes('Dopplerwürfel');

    normalizedContent = normalizedContent.replace(/,/g, '.');

    const xgidLine = lines.find((line) => line.startsWith('XGID='));
    const xgid = xgidLine ? xgidLine.split('=')[1] : null;

    if (!xgid) {
        throw new Error('XGID not found in the file content.');
    }

    const [positionPart, cubeValue, cubeOwner, playerDownOnDiagram, dicePart, score1, score2, isCrawford, matchLength, _dummy] = xgid.split(':');

    const board = { points: Array(26).fill({ checkers: 0, color: -1 }) };

    if (positionPart) {
        const pointChars = positionPart.split('');
        let pointIndex = 0;
        pointChars.forEach((char) => {
            if (char >= 'A' && char <= 'Z') {
                const numCheckers = char.charCodeAt(0) - 'A'.charCodeAt(0) + 1;
                board.points[pointIndex] = { checkers: numCheckers, color: 0 };
            } else if (char >= 'a' && char <= 'z') {
                const numCheckers = char.charCodeAt(0) - 'a'.charCodeAt(0) + 1;
                board.points[pointIndex] = { checkers: numCheckers, color: 1 };
            }
            pointIndex++;
        });
    }

    const diceValues = dicePart ? dicePart.split('').map((num) => parseInt(num)) : [0, 0];
    const dice = [diceValues[0], diceValues[1]];

    const player1Score = parseInt(score1);
    const player2Score = parseInt(score2);
    const matchLengthValue = parseInt(matchLength);
    const playerOnRoll = parseInt(playerDownOnDiagram) === 1 ? 0 : 1;

    let hasJacoby = 0,
        hasBeaver = 0,
        awayScores = [matchLengthValue - player1Score, matchLengthValue - player2Score];
    if (parseInt(isCrawford) === 0) {
        awayScores = awayScores.map((score) => (score === 1 ? 0 : score));
    }
    if (matchLengthValue === 0) {
        awayScores = [-1, -1];
        switch (parseInt(isCrawford)) {
            case 1:
                hasJacoby = 1;
                break;
            case 2:
                hasBeaver = 1;
                break;
            case 3:
                hasJacoby = 1;
                hasBeaver = 1;
                break;
        }
    }

    const player1Bearoff = 15 - board.points.reduce((sum, point) => sum + (point.color === 0 ? point.checkers : 0), 0);
    const player2Bearoff = 15 - board.points.reduce((sum, point) => sum + (point.color === 1 ? point.checkers : 0), 0);
    board.bearoff = [player1Bearoff, player2Bearoff];

    const decisionLine = lines.find((line) => line.includes(isFrench ? 'jouer' : isJapanese ? 'to play' : isGerman ? 'spielen' : 'to play'));
    const decisionType = decisionLine || isInternalCheckerAnalysisFormat ? 0 : 1;

    const positionData = {
        board: board,
        cube: {
            owner: parseInt(cubeOwner) === 1 ? 0 : parseInt(cubeOwner) === -1 ? 1 : -1,
            value: parseInt(cubeValue)
        },
        dice: dice,
        score: awayScores,
        player_on_roll: playerOnRoll,
        decision_type: decisionType,
        has_jacoby: hasJacoby,
        has_beaver: hasBeaver
    };

    const parsedAnalysis = { xgid, analysisType: '', checkerAnalysis: [], doublingCubeAnalysis: {}, analysisEngineVersion: '' };

    const engineVersionMatch = normalizedContent.match(/eXtreme Gammon Version: (.+?)(?:\. MET: (.+))?$/m);
    if (engineVersionMatch) {
        parsedAnalysis.analysisEngineVersion = `eXtreme Gammon Version: ${engineVersionMatch[1]}`;
        if (engineVersionMatch[2]) {
            parsedAnalysis.analysisEngineVersion += `, MET: ${engineVersionMatch[2]}`;
        }
    }

    const engineName = engineVersionMatch ? 'XG' : '';

    if (isInternalDoublingAnalysisFormat) {
        parsedAnalysis.analysisType = 'DoublingCube';
        const doublingCubeAnalysisRegex =
            /Doubling Cube Analysis:\nAnalysis Depth: "(.+)"\nPlayer Win Chances: ([-.\d]+)%\nPlayer Gammon Chances: ([-.\d]+)%\nPlayer Backgammon Chances: ([-.\d]+)%\nOpponent Win Chances: ([-.\d]+)%\nOpponent Gammon Chances: ([-.\d]+)%\nOpponent Backgammon Chances: ([-.\d]+)%\nCubeless No Double Equity: ([-.\d]+)\nCubeless Double Equity: ([-.\d]+)\nCubeful No Double Equity: ([-.\d]+)\nCubeful No Double Error: ([-.\d]+)\nCubeful Double Take Equity: ([-.\d]+)\nCubeful Double Take Error: ([-.\d]+)\nCubeful Double Pass Equity: ([-.\d]+)\nCubeful Double Pass Error: ([-.\d]+)\nBest Cube Action: (.+)\nWrong Pass Percentage: ([-.\d]+)%\nWrong Take Percentage: ([-.\d]+)%/;
        const doublingCubeMatch = doublingCubeAnalysisRegex.exec(normalizedContent);
        if (doublingCubeMatch) {
            parsedAnalysis.doublingCubeAnalysis = {
                analysisDepth: doublingCubeMatch[1].trim(),
                analysisEngine: engineName,
                playerWinChances: parseFloat(doublingCubeMatch[2]),
                playerGammonChances: parseFloat(doublingCubeMatch[3]),
                playerBackgammonChances: parseFloat(doublingCubeMatch[4]),
                opponentWinChances: parseFloat(doublingCubeMatch[5]),
                opponentGammonChances: parseFloat(doublingCubeMatch[6]),
                opponentBackgammonChances: parseFloat(doublingCubeMatch[7]),
                cubelessNoDoubleEquity: parseFloat(doublingCubeMatch[8]),
                cubelessDoubleEquity: parseFloat(doublingCubeMatch[9]),
                cubefulNoDoubleEquity: parseFloat(doublingCubeMatch[10]),
                cubefulNoDoubleError: parseFloat(doublingCubeMatch[11]),
                cubefulDoubleTakeEquity: parseFloat(doublingCubeMatch[12]),
                cubefulDoubleTakeError: parseFloat(doublingCubeMatch[13]),
                cubefulDoublePassEquity: parseFloat(doublingCubeMatch[14]),
                cubefulDoublePassError: parseFloat(doublingCubeMatch[15]),
                bestCubeAction: doublingCubeMatch[16].trim(),
                wrongPassPercentage: parseFloat(doublingCubeMatch[17]),
                wrongTakePercentage: parseFloat(doublingCubeMatch[18])
            };
        }
    } else if (isInternalCheckerAnalysisFormat) {
        parsedAnalysis.analysisType = 'CheckerMove';
        const moveRegex =
            /^Move (\d+): (.+)\nAnalysis Depth: "(.+)"\nEquity: ([-.\d]+)\nEquity Error: ([-.\d]+)\nPlayer Win Chance: ([-.\d]+)%\nPlayer Gammon Chance: ([-.\d]+)%\nPlayer Backgammon Chance: ([-.\d]+)%\nOpponent Win Chance: ([-.\d]+)%\nOpponent Gammon Chance: ([-.\d]+)%\nOpponent Backgammon Chance: ([-.\d]+)%/gm;
        let moveMatch;
        while ((moveMatch = moveRegex.exec(normalizedContent)) !== null) {
            parsedAnalysis.checkerAnalysis.push({
                index: parseInt(moveMatch[1], 10),
                move: moveMatch[2].trim(),
                analysisDepth: moveMatch[3].trim(),
                analysisEngine: engineName,
                equity: parseFloat(moveMatch[4]),
                equityError: parseFloat(moveMatch[5]),
                playerWinChance: parseFloat(moveMatch[6]),
                playerGammonChance: parseFloat(moveMatch[7]),
                playerBackgammonChance: parseFloat(moveMatch[8]),
                opponentWinChance: parseFloat(moveMatch[9]),
                opponentGammonChance: parseFloat(moveMatch[10]),
                opponentBackgammonChance: parseFloat(moveMatch[11])
            });
        }
    } else if (/^ {4}(\d+)\./gm.test(normalizedContent)) {
        parsedAnalysis.analysisType = 'CheckerMove';
        const moveRegex = new RegExp(
            isFrench
                ? /^ {4}(\d+)\.\s(.{11})\s(.{28})\séq:(.{5,7})\s(?:\((-?[-.\d]{5,7})\))?/
                : isJapanese
                  ? /^ {4}(\d+)\.\s(.{11})\s(.{28})\seq:(.{5,7})\s(?:\((-?[-.\d]{5,7})\))?/
                  : isGerman
                    ? /^ {4}(\d+)\.\s(.{11})\s(.{28})\seq:(.{5,7})\s(?:\((-?[-.\d]{5,7})\))?/
                    : /^ {4}(\d+)\.\s(.{11})\s(.{28})\seq:(.{5,7})\s(?:\((-?[-.\d]{5,7})\))?/,
            'gm'
        );
        let moveMatch;
        while ((moveMatch = moveRegex.exec(normalizedContent)) !== null) {
            const moveDetails = {
                index: parseInt(moveMatch[1], 10),
                analysisDepth: moveMatch[2].trim(),
                analysisEngine: engineName,
                move: moveMatch[3].trim(),
                equity: parseFloat(moveMatch[4]),
                equityError: moveMatch[5] ? parseFloat(moveMatch[5]) : 0
            };
            const lineStart = moveMatch.index + moveMatch[0].length;
            const remainingContent = normalizedContent.slice(lineStart);
            const playerRegex = isFrench
                ? /Joueur:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                : isJapanese
                  ? /プレーヤー:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                  : isGerman
                    ? /Spieler:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : /Player:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/;
            const opponentRegex = isFrench
                ? /Adversaire:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                : isJapanese
                  ? /対戦相手:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                  : isGerman
                    ? /Gegner:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : /Opponent:\s*(\d+\.\d+)%.*\(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/;
            const playerMatch = playerRegex.exec(remainingContent);
            const opponentMatch = opponentRegex.exec(remainingContent);
            if (playerMatch) {
                moveDetails.playerWinChance = parseFloat(playerMatch[1]);
                moveDetails.playerGammonChance = parseFloat(playerMatch[2]);
                moveDetails.playerBackgammonChance = parseFloat(playerMatch[3]);
            }
            if (opponentMatch) {
                moveDetails.opponentWinChance = parseFloat(opponentMatch[1]);
                moveDetails.opponentGammonChance = parseFloat(opponentMatch[2]);
                moveDetails.opponentBackgammonChance = parseFloat(opponentMatch[3]);
            }
            parsedAnalysis.checkerAnalysis.push(moveDetails);
        }
        if (playerOnRoll === 1) {
            parsedAnalysis.checkerAnalysis.forEach((move) => {
                const tempWinChance = move.playerWinChance;
                const tempGammonChance = move.playerGammonChance;
                const tempBackgammonChance = move.playerBackgammonChance;
                move.playerWinChance = move.opponentWinChance;
                move.playerGammonChance = move.opponentGammonChance;
                move.playerBackgammonChance = move.opponentBackgammonChance;
                move.opponentWinChance = tempWinChance;
                move.opponentGammonChance = tempGammonChance;
                move.opponentBackgammonChance = tempBackgammonChance;
            });
        }
    } else if (
        (isFrench && (normalizedContent.includes('Equités sans videau') || normalizedContent.includes('Equités avec videau'))) ||
        (isJapanese && (normalizedContent.includes('Cubeless Equities') || normalizedContent.includes('Cubeful Equities'))) ||
        (isGerman && (normalizedContent.includes('Equities ohne Dopplerwürfel') || normalizedContent.includes('Equities mit Dopplerwürfel'))) ||
        (!isFrench && !isJapanese && !isGerman && (normalizedContent.includes('Cubeless Equities') || normalizedContent.includes('Cubeful Equities')))
    ) {
        parsedAnalysis.analysisType = 'DoublingCube';

        const analysisDepthMatch = normalizedContent.match(
            new RegExp(isFrench ? /Analysé avec\s+([^\n]*)/ : isJapanese ? /Analyzed in\s+([^\n]*)/ : isGerman ? /Analysiert in\s+([^\n]*)/ : /Analyzed in\s+([^\n]*)/)
        );
        const playerWinMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Chance de gain du joueur:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : isJapanese
                      ? /Player Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                      : isGerman
                        ? /Spieler Gewinnchancen:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                        : /Player Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
            )
        );
        const opponentWinMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Chance de gain de l'adversaire:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                    : isJapanese
                      ? /Opponent Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                      : isGerman
                        ? /Gewinnchancen des Gegners:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
                        : /Opponent Winning Chances:\s+(\d+\.\d+)% \(G:(\d+\.\d+)% B:(\d+\.\d+)%\)/
            )
        );
        const cubelessMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Equités sans videau\s*:\s*Pas de double=([+\-\d.]+).\s*Double=([+\-\d.]+)/
                    : isJapanese
                      ? /Cubeless Equities:\s*No Double=([+\-\d.]+).\s*Double=([+\-\d.]+)./
                      : isGerman
                        ? /Equities ohne Dopplerwürfel\s*:\s*Nicht Doppeln=([+\-\d.]+).\s*Doppeln=([+\-\d.]+)/
                        : /Cubeless Equities:\s*No Double=([+\-\d.]+).\s*Double=([+\-\d.]+)/
            )
        );
        const cubefulNoDoubleMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Pas de double\s*:\s*([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                    : isJapanese
                      ? /ノーダブル\s*:\s*([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                      : isGerman
                        ? /Nicht Doppeln\s*:\s*([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                        : /No double\s*:\s*([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
            )
        );
        const cubefulDoubleTakeMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Double\/Prend:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                    : isJapanese
                      ? /ダブル\/テイク:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                      : isGerman
                        ? /Doppeln\/Annehmen:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                        : /Double\/Take:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
            )
        );
        const cubefulDoublePassMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Double\/Passe:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                    : isJapanese
                      ? /ダブル\/パス:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                      : isGerman
                        ? /Doppeln\/Ablehnen:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                        : /Double\/Pass:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
            )
        );
        const redoubleNoMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Pas de redouble\s*:\s*([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                    : isJapanese
                      ? /ノーリダブル\s*:\s*([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                      : isGerman
                        ? /Nicht Redoppeln\s*:\s*([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                        : /No redouble\s*:\s*([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
            )
        );
        const redoubleTakeMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Redouble\/Prend:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                    : isJapanese
                      ? /リダブル\/テイク:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                      : isGerman
                        ? /Redoppeln\/Annehmen:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                        : /Redouble\/Take:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
            )
        );
        const redoublePassMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Redouble\/Passe:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                    : isJapanese
                      ? /リダブル\/パス:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                      : isGerman
                        ? /Redoppeln\/Ablehnen:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                        : /Redouble\/Pass:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
            )
        );
        const cubefulDoubleBeaverMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Double\/Beaver:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                    : isJapanese
                      ? /ダブル\/ビーバー:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                      : isGerman
                        ? /Doppeln\/Beaver:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
                        : /Double\/Beaver:\s+([+\-\d.]+)(?: \(([+\-\d.]+)\))?/
            )
        );
        const bestCubeActionMatch = normalizedContent.match(
            new RegExp(isFrench ? /Meilleur action du videau:\s*(.*)/ : isJapanese ? /ベストキューブアクション：\s*(.*)/ : isGerman ? /Beste Dopplerwürfel Aktion\s*(.*)/ : /Best Cube action:\s*(.*)/)
        );
        const wrongPassPercentageMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Pourcentage de passes incorrectes pour rendre la décision de double correcte:\s*(\d+\.\d+)%/
                    : isJapanese
                      ? /ダブルを正当化するのに必要な相手がパスする確率:\s*(\d+\.\d+)%/
                      : isGerman
                        ? /Prozent von falschen Ablehnen gebraucht damit Doppelentscheidung richtig wäre.:\s*(\d+\.\d+)%/
                        : /Percentage of wrong pass needed to make the double decision right:\s*(\d+\.\d+)%/
            )
        );
        const wrongTakePercentageMatch = normalizedContent.match(
            new RegExp(
                isFrench
                    ? /Pourcentage de prises incorrectes pour rendre la décision de double correcte:\s*(\d+\.\d+)%/
                    : isJapanese
                      ? /ダブルを正当化するのに必要な相手がテイクする確率:\s*(\d+\.\d+)%/
                      : isGerman
                        ? /Prozent von falschen Annehmen gebraucht damit Doppelentscheidung richtig wäre.:\s*(\d+\.\d+)%/
                        : /Percentage of wrong take needed to make the double decision right:\s*(\d+\.\d+)%/
            )
        );

        parsedAnalysis.doublingCubeAnalysis.analysisEngine = engineName;
        if (analysisDepthMatch) parsedAnalysis.doublingCubeAnalysis.analysisDepth = analysisDepthMatch[1].trim();
        if (playerWinMatch) {
            parsedAnalysis.doublingCubeAnalysis.playerWinChances = parseFloat(playerWinMatch[1]);
            parsedAnalysis.doublingCubeAnalysis.playerGammonChances = parseFloat(playerWinMatch[2]);
            parsedAnalysis.doublingCubeAnalysis.playerBackgammonChances = parseFloat(playerWinMatch[3]);
        }
        if (opponentWinMatch) {
            parsedAnalysis.doublingCubeAnalysis.opponentWinChances = parseFloat(opponentWinMatch[1]);
            parsedAnalysis.doublingCubeAnalysis.opponentGammonChances = parseFloat(opponentWinMatch[2]);
            parsedAnalysis.doublingCubeAnalysis.opponentBackgammonChances = parseFloat(opponentWinMatch[3]);
        }
        if (cubelessMatch) {
            parsedAnalysis.doublingCubeAnalysis.cubelessNoDoubleEquity = parseFloat(cubelessMatch[1]);
            parsedAnalysis.doublingCubeAnalysis.cubelessDoubleEquity = parseFloat(cubelessMatch[2]);
        }
        if (cubefulNoDoubleMatch) {
            parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleEquity = parseFloat(cubefulNoDoubleMatch[1]);
            parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleError = cubefulNoDoubleMatch[2] ? parseFloat(cubefulNoDoubleMatch[2]) : 0;
        }
        if (cubefulDoubleTakeMatch) {
            parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeEquity = parseFloat(cubefulDoubleTakeMatch[1]);
            parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeError = cubefulDoubleTakeMatch[2] ? parseFloat(cubefulDoubleTakeMatch[2]) : 0;
        }
        if (cubefulDoublePassMatch) {
            parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassEquity = parseFloat(cubefulDoublePassMatch[1]);
            parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassError = cubefulDoublePassMatch[2] ? parseFloat(cubefulDoublePassMatch[2]) : 0;
        }
        if (redoubleNoMatch) {
            parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleEquity = parseFloat(redoubleNoMatch[1]);
            parsedAnalysis.doublingCubeAnalysis.cubefulNoDoubleError = redoubleNoMatch[2] ? parseFloat(redoubleNoMatch[2]) : 0;
        }
        if (redoubleTakeMatch) {
            parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeEquity = parseFloat(redoubleTakeMatch[1]);
            parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeError = redoubleTakeMatch[2] ? parseFloat(redoubleTakeMatch[2]) : 0;
        }
        if (redoublePassMatch) {
            parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassEquity = parseFloat(redoublePassMatch[1]);
            parsedAnalysis.doublingCubeAnalysis.cubefulDoublePassError = redoublePassMatch[2] ? parseFloat(redoublePassMatch[2]) : 0;
        }
        if (cubefulDoubleBeaverMatch) {
            parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeEquity = parseFloat(cubefulDoubleBeaverMatch[1]);
            parsedAnalysis.doublingCubeAnalysis.cubefulDoubleTakeError = cubefulDoubleBeaverMatch[2] ? parseFloat(cubefulDoubleBeaverMatch[2]) : 0;
        }
        if (bestCubeActionMatch) parsedAnalysis.doublingCubeAnalysis.bestCubeAction = bestCubeActionMatch[1].trim();
        if (wrongPassPercentageMatch) parsedAnalysis.doublingCubeAnalysis.wrongPassPercentage = parseFloat(wrongPassPercentageMatch[1]);
        if (wrongTakePercentageMatch) parsedAnalysis.doublingCubeAnalysis.wrongTakePercentage = parseFloat(wrongTakePercentageMatch[1]);

        if (playerOnRoll === 1) {
            const tempWinChances = parsedAnalysis.doublingCubeAnalysis.playerWinChances;
            const tempGammonChances = parsedAnalysis.doublingCubeAnalysis.playerGammonChances;
            const tempBackgammonChances = parsedAnalysis.doublingCubeAnalysis.playerBackgammonChances;
            parsedAnalysis.doublingCubeAnalysis.playerWinChances = parsedAnalysis.doublingCubeAnalysis.opponentWinChances;
            parsedAnalysis.doublingCubeAnalysis.playerGammonChances = parsedAnalysis.doublingCubeAnalysis.opponentGammonChances;
            parsedAnalysis.doublingCubeAnalysis.playerBackgammonChances = parsedAnalysis.doublingCubeAnalysis.opponentBackgammonChances;
            parsedAnalysis.doublingCubeAnalysis.opponentWinChances = tempWinChances;
            parsedAnalysis.doublingCubeAnalysis.opponentGammonChances = tempGammonChances;
            parsedAnalysis.doublingCubeAnalysis.opponentBackgammonChances = tempBackgammonChances;
        }
    }

    const commentSection = extractCommentSection(normalizedContent, parsedAnalysis.analysisType === 'DoublingCube');
    parsedAnalysis.comment = commentSection;

    return { positionData, parsedAnalysis };
}

function extractCommentSection(content, isDoublingCube) {
    if (isDoublingCube) {
        const commentRegex = /(?:Best Cube action: .+|Meilleur action du videau: .+|Percentage of wrong .+|Pourcentage de passes incorrectes .+%)\n\n([\s\S]+?)\n\neXtreme Gammon Version:/;
        let match = commentRegex.exec(content);
        return match ? match[1].trim() : '';
    } else {
        const lines = content.split('\n');
        let lastOpponentIndex = -1;

        for (let i = lines.length - 1; i >= 0; i--) {
            if (lines[i].includes('Opponent') || lines[i].includes('Adversaire')) {
                lastOpponentIndex = i;
                break;
            }
        }

        if (lastOpponentIndex === -1) return '';

        let blankLineCount = 0;
        let commentStartIndex = -1;
        for (let i = lastOpponentIndex + 1; i < lines.length; i++) {
            if (lines[i].trim() === '') blankLineCount++;
            else blankLineCount = 0;
            if (blankLineCount === 2) {
                commentStartIndex = i + 1;
                break;
            }
        }

        if (commentStartIndex === -1) return '';

        let commentEndIndex = -1;
        for (let i = commentStartIndex; i < lines.length; i++) {
            if (lines[i].trim() === '' && lines[i + 1] && lines[i + 1].startsWith('eXtreme Gammon Version:')) {
                commentEndIndex = i;
                break;
            }
        }

        if (commentEndIndex === -1) return '';
        return lines.slice(commentStartIndex, commentEndIndex).join('\n').trim();
    }
}
