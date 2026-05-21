import { writable } from 'svelte/store';
import { SaveDatabaseDialog, OpenDatabaseDialog, DeleteFile } from '../../wailsjs/go/main/App.js';
import { SetupDatabase, CheckDatabaseVersion, OpenDatabase, GetDatabaseVersion } from '../../wailsjs/go/database/Database.js';
import { WindowSetTitle, Quit } from '../../wailsjs/runtime/runtime.js';
import { SaveLastDatabasePath } from '../../wailsjs/go/main/Config.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { analysisStore, selectedMoveStore } from '../stores/analysisStore.js';
import { statusBarTextStore, statusBarModeStore, commentTextStore, openModal, closeModal, MODAL } from '../stores/uiStore.js';
import { ankiDecksStore, selectedAnkiDeckStore, ankiReviewCardStore, ankiDeckStatsStore, ankiViewModeStore } from '../stores/ankiStore.js';
import { logger } from '../utils/logger.js';

export const warningMessageStore = writable('');

function setStatusBarMessage(message) {
    statusBarTextStore.set(message);
}

function getFilenameFromPath(filePath) {
    return filePath.split('/').pop();
}

function resetAnkiStores() {
    ankiDecksStore.set([]);
    selectedAnkiDeckStore.set(null);
    ankiReviewCardStore.set(null);
    ankiDeckStatsStore.set(null);
    ankiViewModeStore.set('list');
}

function resetAnalysisAndCommentStores() {
    analysisStore.set({
        positionId: null,
        xgid: '',
        player1: '',
        player2: '',
        analysisType: '',
        analysisEngineVersion: '',
        checkerAnalysis: { moves: [] },
        doublingCubeAnalysis: {
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
        allCubeAnalyses: [],
        playedMove: '',
        playedCubeAction: '',
        playedMoves: [],
        playedCubeActions: [],
        creationDate: '',
        lastModifiedDate: ''
    });
    commentTextStore.set('');
    selectedMoveStore.set(null);
}

function getMajorVersion(version) {
    return version.split('.')[0];
}

export async function newDatabase() {
    logger.log('newDatabase');
    try {
        const filePath = await SaveDatabaseDialog();
        if (filePath) {
            resetAnalysisAndCommentStores();
            resetAnkiStores();

            try {
                await DeleteFile(filePath);
                logger.log('Existing file deleted:', filePath);
            } catch (error) {
                logger.log('No existing file to delete or error deleting file:', error);
            }

            databasePathStore.set(filePath);
            logger.log('databasePathStore:', filePath);
            await SetupDatabase(filePath);
            setStatusBarMessage('New database created successfully');
            const filename = getFilenameFromPath(filePath);
            WindowSetTitle(`blunderDB - ${filename}`);
            logger.log(`New database created at ${filePath}`);
            const { loadAllPositions } = await import('./positionService.js');
            await loadAllPositions();
        } else {
            logger.log('No file selected');
        }
    } catch (error) {
        logger.error('Error opening file dialog:', error);
        setStatusBarMessage('Error creating new database');
    } finally {
        statusBarModeStore.set('NORMAL');
    }
}

export async function openDatabase() {
    logger.log('openDatabase');
    try {
        const filePath = await OpenDatabaseDialog();
        if (!filePath) {
            logger.log('No Database selected');
            return;
        }

        await openDatabaseByPath(filePath);
    } catch (error) {
        logger.error('Error opening file dialog:', error);
        setStatusBarMessage('Error opening database');
    }
}

export async function openDatabaseByPath(filePath) {
    // Reset mode synchronously before any await so it can't race with the
    // Svelte effect microtask that restoreSessionState schedules later.
    // A finally block would run AFTER those microtasks and overwrite the
    // EPC/EDIT mode that the tab handler correctly re-enters on session restore.
    statusBarModeStore.set('NORMAL');
    try {
        resetAnalysisAndCommentStores();
        resetAnkiStores();

        databasePathStore.set(filePath);
        logger.log('databasePathStore:', filePath);

        await SaveLastDatabasePath(filePath);
        await OpenDatabase(filePath);

        const dbVersion = await CheckDatabaseVersion();
        const modelVersion = await GetDatabaseVersion();
        logger.log(`Database version: ${dbVersion}`);
        logger.log(`Model version: ${modelVersion}`);
        setStatusBarMessage(`Database version: ${dbVersion}`);

        if (getMajorVersion(dbVersion) !== getMajorVersion(modelVersion)) {
            warningMessageStore.set(
                `Major database version mismatch. The database schema might be incompatible with the current version of blunderDB. Continuing to edit the database is done at your own risk. Backup your file before proceeding any further.\nDatabase version: ${dbVersion}\nExpected version: ${modelVersion}`
            );
            openModal(MODAL.WARNING);
        }

        setStatusBarMessage('Database opened successfully');
        const filename = getFilenameFromPath(filePath);
        WindowSetTitle(`blunderDB - ${filename}`);

        const { restoreSessionState } = await import('./sessionService.js');
        await restoreSessionState();
    } catch (error) {
        logger.error('Error opening database:', error);
        setStatusBarMessage('Error opening database');
        statusBarModeStore.set('NORMAL');
    }
}

export async function exitApp() {
    const { saveSessionState } = await import('./sessionService.js');
    await saveSessionState();
    Quit();
}

export function closeWarningModal() {
    closeModal();
}

export { getFilenameFromPath, setStatusBarMessage, resetAnkiStores, resetAnalysisAndCommentStores, getMajorVersion };
