/**
 * databaseService.test.js
 *
 * databaseService.js orchestrates opening/creating/closing a database: file
 * dialogs, the version check, the read-only fallback, the protected-copy
 * prompt, and the various stores it resets along the way. It had 0 test file
 * and 7.4% line coverage (D.13, #214) — every Wails binding it calls is
 * mocked here; the stores it reads and writes are the real Svelte stores.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    SaveDatabaseDialog: vi.fn(),
    OpenDatabaseDialog: vi.fn(),
    DeleteFile: vi.fn(),
    PrepareDemoDatabase: vi.fn()
}));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SetupDatabase: vi.fn(),
    CheckDatabaseVersion: vi.fn(),
    OpenDatabase: vi.fn(),
    GetDatabaseVersion: vi.fn(),
    IsReadOnly: vi.fn(),
    IsProtectedCopyPath: vi.fn(),
    OpenProtectedCopyPath: vi.fn(),
    DeleteProtectedCopyPath: vi.fn()
}));

vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    WindowSetTitle: vi.fn(),
    Quit: vi.fn()
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    SaveLastDatabasePath: vi.fn()
}));

// Dynamically imported from within databaseService.js — mocked so the
// orchestration under test never pulls in the real (much heavier) modules.
vi.mock('../services/positionService.js', () => ({
    loadAllPositions: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../services/sessionService.js', () => ({
    restoreSessionState: vi.fn().mockResolvedValue(undefined),
    saveSessionState: vi.fn().mockResolvedValue(undefined)
}));

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
import { loadAllPositions } from '../services/positionService.js';
import { restoreSessionState, saveSessionState } from '../services/sessionService.js';

import {
    newDatabase,
    openDatabase,
    loadDemoDatabase,
    openDatabaseByPath,
    exitApp,
    closeWarningModal,
    unlockProtectedCopy,
    cancelProtectedCopy,
    warningMessageStore,
    protectedCopyPathStore,
    protectedCopyErrorStore,
    setStatusBarMessage
} from '../services/databaseService.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { statusBarTextStore, statusBarModeStore, commentTextStore, activeModal, MODAL } from '../stores/uiStore.js';
import { ankiDecksStore, selectedAnkiDeckStore, ankiViewModeStore, ankiAnswerShownStore } from '../stores/ankiStore.js';

// ── Helpers ───────────────────────────────────────────────────────────────────

function resetStores() {
    databasePathStore.set('');
    statusBarTextStore.set('');
    statusBarModeStore.set('NORMAL');
    commentTextStore.set('');
    selectedMoveStore.set(null);
    warningMessageStore.set('');
    protectedCopyPathStore.set('');
    protectedCopyErrorStore.set('');
    ankiDecksStore.set(['stale']);
    selectedAnkiDeckStore.set({ id: 1 });
    ankiViewModeStore.set('review');
    ankiAnswerShownStore.set(true);
    activeModal.set(null);
}

beforeEach(() => {
    vi.clearAllMocks();
    resetStores();
    // Default happy-path resolutions; individual tests override with
    // `.mockResolvedValueOnce` / `.mockRejectedValueOnce` as needed.
    CheckDatabaseVersion.mockResolvedValue('2.18.0');
    GetDatabaseVersion.mockResolvedValue('2.18.0');
    IsReadOnly.mockResolvedValue(false);
    IsProtectedCopyPath.mockResolvedValue(false);
});

// ── newDatabase ───────────────────────────────────────────────────────────────

describe('newDatabase', () => {
    test('creates the file, opens it and reloads the library', async () => {
        SaveDatabaseDialog.mockResolvedValue('/tmp/new.db');

        await newDatabase();

        expect(DeleteFile).toHaveBeenCalledWith('/tmp/new.db');
        expect(get(databasePathStore)).toBe('/tmp/new.db');
        expect(SetupDatabase).toHaveBeenCalledWith('/tmp/new.db');
        expect(WindowSetTitle).toHaveBeenCalledWith('blunderDB - new.db');
        expect(loadAllPositions).toHaveBeenCalled();
        expect(get(statusBarTextStore)).not.toBe('');
        expect(get(statusBarModeStore)).toBe('NORMAL');
    });

    test('resets the analysis/comment/Anki stores before creating the file', async () => {
        SaveDatabaseDialog.mockResolvedValue('/tmp/new.db');

        await newDatabase();

        expect(get(commentTextStore)).toBe('');
        expect(get(selectedMoveStore)).toBeNull();
        expect(get(ankiDecksStore)).toEqual([]);
        expect(get(selectedAnkiDeckStore)).toBeNull();
        expect(get(ankiViewModeStore)).toBe('list');
        expect(get(ankiAnswerShownStore)).toBe(false);
    });

    test('a cancelled dialog is a no-op', async () => {
        SaveDatabaseDialog.mockResolvedValue('');

        await newDatabase();

        expect(SetupDatabase).not.toHaveBeenCalled();
        expect(get(databasePathStore)).toBe('');
    });

    test('DeleteFile failing (no prior file) does not abort database creation', async () => {
        SaveDatabaseDialog.mockResolvedValue('/tmp/new.db');
        DeleteFile.mockRejectedValue(new Error('ENOENT'));

        await newDatabase();

        expect(SetupDatabase).toHaveBeenCalledWith('/tmp/new.db');
        expect(get(statusBarModeStore)).toBe('NORMAL');
    });

    test('SetupDatabase failing reports the error and still resets the mode', async () => {
        SaveDatabaseDialog.mockResolvedValue('/tmp/new.db');
        SetupDatabase.mockRejectedValue(new Error('disk full'));

        await newDatabase();

        expect(get(statusBarTextStore)).not.toBe('');
        expect(get(statusBarModeStore)).toBe('NORMAL');
        expect(loadAllPositions).not.toHaveBeenCalled();
    });
});

// ── openDatabase / loadDemoDatabase (dialog wrappers) ────────────────────────

describe('openDatabase', () => {
    test('delegates to openDatabaseByPath with the chosen file', async () => {
        OpenDatabaseDialog.mockResolvedValue('/tmp/existing.db');

        await openDatabase();

        expect(get(databasePathStore)).toBe('/tmp/existing.db');
        expect(OpenDatabase).toHaveBeenCalledWith('/tmp/existing.db');
    });

    test('a cancelled dialog never touches the database', async () => {
        OpenDatabaseDialog.mockResolvedValue('');

        await openDatabase();

        expect(OpenDatabase).not.toHaveBeenCalled();
        expect(get(databasePathStore)).toBe('');
    });

    test('a dialog error reports it without throwing', async () => {
        OpenDatabaseDialog.mockRejectedValue(new Error('dialog crashed'));

        await expect(openDatabase()).resolves.toBeUndefined();
        expect(get(statusBarTextStore)).not.toBe('');
    });
});

describe('loadDemoDatabase', () => {
    test('decompresses the demo database and opens it', async () => {
        PrepareDemoDatabase.mockResolvedValue('/tmp/demo.db');

        await loadDemoDatabase();

        expect(get(databasePathStore)).toBe('/tmp/demo.db');
        expect(OpenDatabase).toHaveBeenCalledWith('/tmp/demo.db');
    });

    test('no path returned is a no-op', async () => {
        PrepareDemoDatabase.mockResolvedValue('');

        await loadDemoDatabase();

        expect(OpenDatabase).not.toHaveBeenCalled();
    });

    test('a failure to prepare the demo database reports an error', async () => {
        PrepareDemoDatabase.mockRejectedValue(new Error('embed missing'));

        await loadDemoDatabase();

        expect(get(statusBarTextStore)).not.toBe('');
        expect(OpenDatabase).not.toHaveBeenCalled();
    });
});

// ── openDatabaseByPath ────────────────────────────────────────────────────────

describe('openDatabaseByPath', () => {
    test('protected copy: prompts for a password instead of opening', async () => {
        IsProtectedCopyPath.mockResolvedValue(true);

        await openDatabaseByPath('/tmp/protected.blunderdb');

        expect(get(protectedCopyPathStore)).toBe('/tmp/protected.blunderdb');
        expect(get(protectedCopyErrorStore)).toBe('');
        expect(get(activeModal)).toBe(MODAL.PROTECTED_COPY);
        expect(OpenDatabase).not.toHaveBeenCalled();
        // Nothing is persisted as "the" database path until it is unlocked.
        expect(get(databasePathStore)).toBe('');
    });

    test('a mismatched major version opens a warning modal', async () => {
        CheckDatabaseVersion.mockResolvedValue('1.0.0');
        GetDatabaseVersion.mockResolvedValue('2.18.0');

        await openDatabaseByPath('/tmp/old.db');

        expect(get(warningMessageStore)).not.toBe('');
        expect(get(activeModal)).toBe(MODAL.WARNING);
    });

    test('matching major versions do not open the warning modal', async () => {
        CheckDatabaseVersion.mockResolvedValue('2.0.0');
        GetDatabaseVersion.mockResolvedValue('2.18.0');

        await openDatabaseByPath('/tmp/ok.db');

        expect(get(activeModal)).toBeNull();
    });

    test('read-only fallback: title and status reflect it, session is still restored', async () => {
        IsReadOnly.mockResolvedValue(true);

        await openDatabaseByPath('/tmp/locked.db');

        expect(WindowSetTitle).toHaveBeenCalledWith(expect.stringContaining('locked.db'));
        expect(restoreSessionState).toHaveBeenCalled();
    });

    test('IsReadOnly failing is treated as read-write (fails open)', async () => {
        IsReadOnly.mockRejectedValue(new Error('bridge error'));

        await openDatabaseByPath('/tmp/x.db');

        expect(WindowSetTitle).toHaveBeenCalledWith('blunderDB - x.db');
    });

    test('happy path resets stores, persists the path and restores the session', async () => {
        await openDatabaseByPath('/tmp/normal.db');

        expect(get(databasePathStore)).toBe('/tmp/normal.db');
        expect(SaveLastDatabasePath).toHaveBeenCalledWith('/tmp/normal.db');
        expect(OpenDatabase).toHaveBeenCalledWith('/tmp/normal.db');
        expect(restoreSessionState).toHaveBeenCalled();
        expect(get(statusBarModeStore)).toBe('NORMAL');
    });

    test('OpenDatabase failing reports the error and resets the mode without restoring a session', async () => {
        OpenDatabase.mockRejectedValue(new Error('corrupt file'));

        await openDatabaseByPath('/tmp/broken.db');

        expect(get(statusBarTextStore)).not.toBe('');
        expect(get(statusBarModeStore)).toBe('NORMAL');
        expect(restoreSessionState).not.toHaveBeenCalled();
    });
});

// ── exitApp / closeWarningModal ──────────────────────────────────────────────

describe('exitApp', () => {
    test('saves the session then quits', async () => {
        await exitApp();
        expect(saveSessionState).toHaveBeenCalled();
        expect(Quit).toHaveBeenCalled();
    });
});

describe('closeWarningModal', () => {
    test('clears the active modal', () => {
        activeModal.set(MODAL.WARNING);
        closeWarningModal();
        expect(get(activeModal)).toBeNull();
    });
});

// ── Protected-copy unlock flow ────────────────────────────────────────────────

describe('unlockProtectedCopy', () => {
    beforeEach(() => {
        protectedCopyPathStore.set('/tmp/container.blunderdb');
    });

    test('a correct password opens the recovered database and clears the prompt', async () => {
        OpenProtectedCopyPath.mockResolvedValue('/tmp/recovered.db');

        await unlockProtectedCopy('right-pass');

        expect(OpenProtectedCopyPath).toHaveBeenCalledWith('/tmp/container.blunderdb', 'right-pass');
        expect(get(protectedCopyPathStore)).toBe('');
        expect(get(activeModal)).toBeNull();
        expect(get(databasePathStore)).toBe('/tmp/recovered.db');
        expect(DeleteProtectedCopyPath).not.toHaveBeenCalled();
    });

    test('removeContainer=true deletes the container after a successful unlock', async () => {
        OpenProtectedCopyPath.mockResolvedValue('/tmp/recovered.db');

        await unlockProtectedCopy('right-pass', true);

        expect(DeleteProtectedCopyPath).toHaveBeenCalledWith('/tmp/container.blunderdb');
    });

    test('the container survives even if deleting it fails', async () => {
        OpenProtectedCopyPath.mockResolvedValue('/tmp/recovered.db');
        DeleteProtectedCopyPath.mockRejectedValue(new Error('permission denied'));

        await expect(unlockProtectedCopy('right-pass', true)).resolves.toBeUndefined();
        expect(get(databasePathStore)).toBe('/tmp/recovered.db');
    });

    test('a wrong password keeps the prompt open with an error, never touching the container', async () => {
        OpenProtectedCopyPath.mockRejectedValue(new Error('wrong passphrase'));

        await unlockProtectedCopy('bad-pass');

        expect(get(protectedCopyPathStore)).toBe('/tmp/container.blunderdb');
        expect(get(protectedCopyErrorStore)).not.toBe('');
        expect(DeleteProtectedCopyPath).not.toHaveBeenCalled();
        expect(get(databasePathStore)).toBe('');
    });

    test('no path pending is a no-op', async () => {
        protectedCopyPathStore.set('');
        await unlockProtectedCopy('whatever');
        expect(OpenProtectedCopyPath).not.toHaveBeenCalled();
    });
});

describe('cancelProtectedCopy', () => {
    test('closes the prompt and clears the pending path/error', () => {
        protectedCopyPathStore.set('/tmp/container.blunderdb');
        protectedCopyErrorStore.set('wrong password');
        activeModal.set(MODAL.PROTECTED_COPY);

        cancelProtectedCopy();

        expect(get(protectedCopyPathStore)).toBe('');
        expect(get(protectedCopyErrorStore)).toBe('');
        expect(get(activeModal)).toBeNull();
        expect(get(statusBarTextStore)).not.toBe('');
    });
});

describe('setStatusBarMessage', () => {
    test('writes straight through to statusBarTextStore', () => {
        setStatusBarMessage('hello');
        expect(get(statusBarTextStore)).toBe('hello');
    });
});
