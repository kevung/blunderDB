import { writable, get } from 'svelte/store';
import { SaveDatabaseDialog, OpenDatabaseDialog, DeleteFile, PrepareDemoDatabase } from '../../wailsjs/go/gui/App.js';
import {
    SetupDatabase,
    CheckDatabaseVersion,
    OpenDatabase,
    GetDatabaseVersion,
    IsReadOnly,
    IsProtectedCopyPath,
    OpenProtectedCopyPath,
    DeleteProtectedCopyPath
} from '../../wailsjs/go/database/Database.js';
import { WindowSetTitle, Quit } from '../../wailsjs/runtime/runtime.js';
import { SaveLastDatabasePath } from '../../wailsjs/go/main/Config.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { analysisStore, emptyAnalysis, selectedMoveStore } from '../stores/analysisStore.js';
import { statusBarTextStore, statusBarModeStore, commentTextStore, openModal, closeModal, MODAL, matchPanelRefreshTriggerStore } from '../stores/uiStore.js';
import { ankiDecksStore, selectedAnkiDeckStore, ankiReviewCardStore, ankiDeckStatsStore, ankiViewModeStore, hideAnkiAnswer } from '../stores/ankiStore.js';
import { logger } from '../utils/logger.js';
// NOTE: these UI messages are translated at emission time via the non-reactive
// `translate` helper; already-displayed messages do not retranslate on language change.
import { translate, tMsg } from '../i18n';

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
    // Opening another database ends whatever review was in progress, so the
    // next card starts from a hidden answer (ADR-0025 rule 5).
    hideAnkiAnswer();
}

function resetAnalysisAndCommentStores() {
    analysisStore.set(emptyAnalysis());
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
            setStatusBarMessage(tMsg('commands.dbCreated'));
            const filename = getFilenameFromPath(filePath);
            WindowSetTitle(`blunderDB - ${filename}`);
            logger.log(`New database created at ${filePath}`);

            // The Matches panel may already be visible (default tab): bump its
            // refresh trigger so it reloads against the new DB.
            matchPanelRefreshTriggerStore.update((n) => n + 1);

            const { loadAllPositions } = await import('./positionService.js');
            await loadAllPositions();
        } else {
            logger.log('No file selected');
        }
    } catch (error) {
        logger.error('Error opening file dialog:', error);
        setStatusBarMessage(tMsg('commands.errorCreatingDb'));
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
        setStatusBarMessage(tMsg('commands.errorOpeningDb'));
    }
}

// Load the embedded sample database (a couple of matches + a tournament with
// analysis) so users — and the guided tours — have real content to explore.
// Decompresses to a fresh temp file and reuses the normal open flow.
export async function loadDemoDatabase() {
    try {
        const filePath = await PrepareDemoDatabase();
        if (!filePath) return;
        await openDatabaseByPath(filePath);
    } catch (error) {
        logger.error('Error loading demo database:', error);
        setStatusBarMessage(tMsg('commands.errorOpeningDb'));
    }
}

// A protected copy is not a database yet. The recipient is asked for its password once,
// here; the result is an ordinary file they work with from then on — nothing about the
// opening is recorded anywhere. These stores drive the prompt in App.svelte.
export const protectedCopyPathStore = writable('');
export const protectedCopyErrorStore = writable('');

export async function openDatabaseByPath(filePath) {
    if (await IsProtectedCopyPath(filePath).catch(() => false)) {
        protectedCopyErrorStore.set('');
        protectedCopyPathStore.set(filePath);
        openModal(MODAL.PROTECTED_COPY);
        return;
    }

    // Reset the mode synchronously before any await: a finally block would run
    // after restoreSessionState's microtasks and overwrite the EVAL/EDIT mode
    // it re-enters.
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
        setStatusBarMessage(tMsg('commands.dbVersion', { version: dbVersion }));

        if (getMajorVersion(dbVersion) !== getMajorVersion(modelVersion)) {
            warningMessageStore.set(translate('commands.dbVersionMismatch', { dbVersion, modelVersion }));
            openModal(MODAL.WARNING);
        }

        // Another instance holds the write lock, so the database opened
        // read-only: say so non-blockingly (title suffix + status message).
        const readOnly = await IsReadOnly().catch(() => false);
        const filename = getFilenameFromPath(filePath);
        if (readOnly) {
            WindowSetTitle(`blunderDB - ${filename} ${tMsg('commands.readOnlySuffix')}`);
            setStatusBarMessage(tMsg('commands.dbReadOnly'));
        } else {
            setStatusBarMessage(tMsg('commands.dbOpened'));
            WindowSetTitle(`blunderDB - ${filename}`);
        }

        // Bump the Matches panel's refresh trigger (it may have mounted before
        // the DB opened), before the slower restoreSessionState, so the match
        // list loads in parallel.
        matchPanelRefreshTriggerStore.update((n) => n + 1);

        const { restoreSessionState } = await import('./sessionService.js');
        await restoreSessionState();
    } catch (error) {
        logger.error('Error opening database:', error);
        setStatusBarMessage(tMsg('commands.errorOpeningDb'));
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

export { setStatusBarMessage };

// unlockProtectedCopy turns the protected copy into an ordinary database and
// opens it; a wrong password errors on the prompt so the recipient can retry.
// Errors cross the bridge as strings and are matched on this project's own
// constants (issuance.ErrWrongPassphrase, ErrPassphraseRequired); anything
// else is shown as-is.
function protectedCopyMessage(error) {
    const text = String(error);
    if (text.includes('wrong passphrase')) return translate('issuance.wrongPassword');
    if (text.includes('protected by a passphrase')) return translate('issuance.passwordRequiredToOpen');
    return text;
}

export async function unlockProtectedCopy(password, removeContainer = false) {
    const source = get(protectedCopyPathStore);
    if (!source) return;
    try {
        const opened = await OpenProtectedCopyPath(source, password);
        // Only once the database is out: a failure here must never cost the container, which
        // may be the only copy the recipient has.
        if (removeContainer) {
            await DeleteProtectedCopyPath(source).catch((error) => logger.error('Error removing the protected file:', error));
        }
        closeModal();
        protectedCopyPathStore.set('');
        protectedCopyErrorStore.set('');
        await openDatabaseByPath(opened);
        // openDatabaseByPath reports the ordinary "database opened" message; say explicitly
        // that the password was accepted, since that is what the user was just asked for.
        setStatusBarMessage(tMsg('issuance.protectedOpened', { name: getFilenameFromPath(opened) }));
    } catch (error) {
        logger.error('Error opening protected copy:', error);
        protectedCopyErrorStore.set(protectedCopyMessage(error));
    }
}

export function cancelProtectedCopy() {
    closeModal();
    protectedCopyPathStore.set('');
    protectedCopyErrorStore.set('');
    setStatusBarMessage(tMsg('commands.errorOpeningDb'));
}
