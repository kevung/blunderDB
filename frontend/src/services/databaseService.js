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

            // The Matches panel may already be visible (it is the default tab),
            // so its own open-transition load won't fire against the new DB.
            // Bump the refresh trigger so it reloads the (now empty) match list
            // — same mechanism used after open/import. See openDatabaseByPath.
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
        setStatusBarMessage(tMsg('commands.dbVersion', { version: dbVersion }));

        if (getMajorVersion(dbVersion) !== getMajorVersion(modelVersion)) {
            warningMessageStore.set(translate('commands.dbVersionMismatch', { dbVersion, modelVersion }));
            openModal(MODAL.WARNING);
        }

        // Read-only fallback: another blunderDB instance holds the write lock, so
        // the backend opened this database read-only (per the single-writer guard).
        // Surface it non-blockingly — a title suffix (persistent) plus a status
        // message — rather than a modal, so browsing/searching still works.
        const readOnly = await IsReadOnly().catch(() => false);
        const filename = getFilenameFromPath(filePath);
        if (readOnly) {
            WindowSetTitle(`blunderDB - ${filename} ${tMsg('commands.readOnlySuffix')}`);
            setStatusBarMessage(tMsg('commands.dbReadOnly'));
        } else {
            setStatusBarMessage(tMsg('commands.dbOpened'));
            WindowSetTitle(`blunderDB - ${filename}`);
        }

        // The Matches panel may already be visible (it is the default tab,
        // opened at mount before the DB finished loading), so its own
        // open-transition load won't have fired against the now-open DB. Bump
        // the refresh trigger so it loads the match list — same mechanism used
        // after an import. Do this BEFORE restoreSessionState (which loads all
        // positions and can be slow) so the match list appears promptly, in
        // parallel with the restore rather than serialised after it.
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

// unlockProtectedCopy turns the protected copy into an ordinary database and opens it. A
// wrong password comes back as an error on the prompt rather than closing it: the recipient
// gets to try again.
// The backend's errors cross the bridge as plain strings, so they are matched by text. Both
// come from constants this project owns (issuance.ErrWrongPassphrase and
// ErrPassphraseRequired), which is what makes the match safe; anything else is shown as-is
// rather than swallowed.
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
