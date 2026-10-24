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
    RecreateDatabase: vi.fn(),
    IsOpenDatabase: vi.fn(() => Promise.resolve(false)),
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
vi.mock('../stores/libraryCountsStore.js', () => ({ refreshLibraryCounts: vi.fn() }));
vi.mock('../services/positionService.js', () => ({
    loadAllPositions: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../services/sessionService.js', () => ({
    restoreSessionState: vi.fn().mockResolvedValue(undefined),
    saveSessionState: vi.fn().mockResolvedValue(undefined)
}));

import { SaveDatabaseDialog, OpenDatabaseDialog, PrepareDemoDatabase } from '../../wailsjs/go/gui/App.js';
import { libraryMark, writingOnLibrary } from '../stores/groupActivityStore.js';
import { tMsg } from '../i18n';
import {
    RecreateDatabase,
    IsOpenDatabase,
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
import { refreshLibraryCounts } from '../stores/libraryCountsStore.js';
import { loadAllPositions } from '../services/positionService.js';
import { restoreSessionState, saveSessionState } from '../services/sessionService.js';
import { openDirectionIdStore } from '../stores/directionStore.js';

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
import { searchEmptyStore } from '../stores/searchParamsStore.js';
import { statusBarTextStore, statusBarModeStore, commentTextStore, activeModal, MODAL } from '../stores/uiStore.js';
import { ankiDecksStore, selectedAnkiDeckStore, ankiViewModeStore, ankiAnswerShownStore } from '../stores/ankiStore.js';
import { transcriptionStore } from '../stores/transcriptionStore.js';
import { lessonStore, lessonStepIndexStore } from '../stores/lessonStore.js';
import { transferHistoryStore } from '../stores/transferBasketStore.js';
import { activeGroupId } from '../stores/groupActivityStore.js';

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
    vi.mocked(CheckDatabaseVersion).mockResolvedValue('2.18.0');
    vi.mocked(GetDatabaseVersion).mockResolvedValue('2.18.0');
    vi.mocked(IsReadOnly).mockResolvedValue(false);
    vi.mocked(IsProtectedCopyPath).mockResolvedValue(false);
    vi.mocked(OpenDatabase).mockResolvedValue(undefined);
});

// ── newDatabase ───────────────────────────────────────────────────────────────

describe('newDatabase', () => {
    test('creates the file, opens it and reloads the library', async () => {
        vi.mocked(SaveDatabaseDialog).mockResolvedValue('/tmp/new.db');

        await newDatabase();

        expect(get(databasePathStore)).toBe('/tmp/new.db');
        expect(RecreateDatabase).toHaveBeenCalledWith('/tmp/new.db');
        expect(WindowSetTitle).toHaveBeenCalledWith('blunderDB - new.db');
        expect(loadAllPositions).toHaveBeenCalled();
        expect(get(statusBarTextStore)).not.toBe('');
        expect(get(statusBarModeStore)).toBe('NORMAL');
    });

    test('resets the analysis/comment/Anki stores before creating the file', async () => {
        vi.mocked(SaveDatabaseDialog).mockResolvedValue('/tmp/new.db');

        await newDatabase();

        expect(get(commentTextStore)).toBe('');
        expect(get(selectedMoveStore)).toBeNull();
        expect(get(ankiDecksStore)).toEqual([]);
        expect(get(selectedAnkiDeckStore)).toBeNull();
        expect(get(ankiViewModeStore)).toBe('list');
        expect(get(ankiAnswerShownStore)).toBe(false);
    });

    test('a cancelled dialog is a no-op', async () => {
        vi.mocked(SaveDatabaseDialog).mockResolvedValue('');

        await newDatabase();

        expect(RecreateDatabase).not.toHaveBeenCalled();
        expect(get(databasePathStore)).toBe('');
    });

    test('a write holding the library refuses the creation: nothing changes on screen, and it is said', async () => {
        vi.mocked(SaveDatabaseDialog).mockResolvedValue('/tmp/old.db');
        databasePathStore.set('/tmp/old.db');
        commentTextStore.set('kept');
        vi.mocked(RecreateDatabase).mockRejectedValueOnce(new Error('database is busy: write'));

        await newDatabase();

        expect(get(databasePathStore)).toBe('/tmp/old.db');
        expect(get(commentTextStore)).toBe('kept');
        expect(get(statusBarTextStore)).toEqual(tMsg('libraryGroups.busyWrite'));
        expect(loadAllPositions).not.toHaveBeenCalled();
    });

    test('the creation waits for a write still on its way: the write lands in the library it read', async () => {
        vi.mocked(SaveDatabaseDialog).mockResolvedValue('/tmp/old.db');
        databasePathStore.set('/tmp/old.db');
        /** @type {string[]} */
        const order = [];
        /** @type {() => void} */
        let answer = () => {};
        const answered = new Promise((r) => (answer = () => r(undefined)));
        const write = writingOnLibrary(libraryMark(), async () => (await answered, order.push('write'), true));
        vi.mocked(RecreateDatabase).mockImplementationOnce(async () => void order.push('recreate'));
        const created = newDatabase();
        await new Promise((r) => setTimeout(r, 0));
        answer();
        await Promise.all([write, created]);
        expect(order).toEqual(['write', 'recreate']);
    });

    test('a failed creation reports the error and still resets the mode', async () => {
        vi.mocked(SaveDatabaseDialog).mockResolvedValue('/tmp/new.db');
        databasePathStore.set('/tmp/old.db');
        vi.mocked(RecreateDatabase).mockRejectedValueOnce(new Error('disk full'));

        await newDatabase();

        expect(get(statusBarTextStore)).not.toBe('');
        expect(get(statusBarModeStore)).toBe('NORMAL');
        expect(loadAllPositions).not.toHaveBeenCalled();
        expect(get(databasePathStore)).toBe('');
    });

    test('a creation refused before any change leaves the library the backend kept on screen', async () => {
        vi.mocked(SaveDatabaseDialog).mockResolvedValue('/tmp/new.db');
        databasePathStore.set('/tmp/old.db');
        vi.mocked(RecreateDatabase).mockRejectedValueOnce(new Error('already open in another group'));
        vi.mocked(IsOpenDatabase).mockResolvedValueOnce(true);

        await newDatabase();

        expect(IsOpenDatabase).toHaveBeenCalledWith('/tmp/old.db');
        expect(get(databasePathStore)).toBe('/tmp/old.db');
    });
});

// ── openDatabase / loadDemoDatabase (dialog wrappers) ────────────────────────

describe('openDatabase', () => {
    test('delegates to openDatabaseByPath with the chosen file', async () => {
        vi.mocked(OpenDatabaseDialog).mockResolvedValue('/tmp/existing.db');

        await openDatabase();

        expect(get(databasePathStore)).toBe('/tmp/existing.db');
        expect(OpenDatabase).toHaveBeenCalledWith('/tmp/existing.db');
    });

    test('a cancelled dialog never touches the database', async () => {
        vi.mocked(OpenDatabaseDialog).mockResolvedValue('');

        await openDatabase();

        expect(OpenDatabase).not.toHaveBeenCalled();
        expect(get(databasePathStore)).toBe('');
    });

    test('a dialog error reports it without throwing', async () => {
        vi.mocked(OpenDatabaseDialog).mockRejectedValue(new Error('dialog crashed'));

        await expect(openDatabase()).resolves.toBeUndefined();
        expect(get(statusBarTextStore)).not.toBe('');
    });
});

describe('loadDemoDatabase', () => {
    test('decompresses the demo database and opens it', async () => {
        vi.mocked(PrepareDemoDatabase).mockResolvedValue('/tmp/demo.db');

        await loadDemoDatabase();

        expect(get(databasePathStore)).toBe('/tmp/demo.db');
        expect(OpenDatabase).toHaveBeenCalledWith('/tmp/demo.db');
    });

    test('no path returned is a no-op', async () => {
        vi.mocked(PrepareDemoDatabase).mockResolvedValue('');

        await loadDemoDatabase();

        expect(OpenDatabase).not.toHaveBeenCalled();
    });

    test('a failure to prepare the demo database reports an error', async () => {
        vi.mocked(PrepareDemoDatabase).mockRejectedValue(new Error('embed missing'));

        await loadDemoDatabase();

        expect(get(statusBarTextStore)).not.toBe('');
        expect(OpenDatabase).not.toHaveBeenCalled();
    });
});

// ── openDatabaseByPath ────────────────────────────────────────────────────────

describe('a Lesson being read', () => {
    const lesson = { id: 1, name: 'L', steps: [{ id: 1 }, { id: 2 }] };

    test('is dropped when another database is opened', async () => {
        lessonStore.set(lesson);
        lessonStepIndexStore.set(1);

        await openDatabaseByPath('/tmp/other.db');

        expect(get(lessonStore)).toBeNull();
        expect(get(lessonStepIndexStore)).toBe(0);
    });

    test('is dropped when a database is created', async () => {
        vi.mocked(SaveDatabaseDialog).mockResolvedValue('/tmp/new.db');
        lessonStore.set(lesson);
        lessonStepIndexStore.set(1);

        await newDatabase();

        expect(get(lessonStore)).toBeNull();
        expect(get(lessonStepIndexStore)).toBe(0);
    });
});

describe('openDatabaseByPath', () => {
    test('a failed open leaves nothing open: empty path, neutral title, counts re-read', async () => {
        databasePathStore.set('/tmp/old.db');
        vi.mocked(OpenDatabase).mockRejectedValue(new Error('corrupt'));
        vi.mocked(refreshLibraryCounts).mockClear();
        vi.mocked(WindowSetTitle).mockClear();

        await openDatabaseByPath('/tmp/bad.db');

        expect(get(databasePathStore)).toBe('');
        expect(WindowSetTitle).toHaveBeenLastCalledWith('blunderDB');
        expect(refreshLibraryCounts).toHaveBeenCalledTimes(1);
        expect(SaveLastDatabasePath).not.toHaveBeenCalled();
    });

    test('an open refused by a write holding the library keeps the library on screen, and says so', async () => {
        databasePathStore.set('/tmp/old.db');
        vi.mocked(OpenDatabase).mockRejectedValueOnce(new Error('database is busy: write'));

        await openDatabaseByPath('/tmp/other.db');

        expect(get(databasePathStore)).toBe('/tmp/old.db');
        expect(get(statusBarTextStore)).toEqual(tMsg('libraryGroups.busyWrite'));
        expect(SaveLastDatabasePath).not.toHaveBeenCalled();
    });

    test('a failure after a successful open keeps the opened path', async () => {
        vi.mocked(CheckDatabaseVersion).mockRejectedValue(new Error('version unreadable'));
        vi.mocked(refreshLibraryCounts).mockClear();

        await openDatabaseByPath('/tmp/opened.db');

        expect(get(databasePathStore)).toBe('/tmp/opened.db');
        expect(SaveLastDatabasePath).toHaveBeenCalledWith('/tmp/opened.db');
        expect(refreshLibraryCounts).toHaveBeenCalledTimes(1);
    });

    test('the library counts are refreshed once the database is open, not before', async () => {
        /** @type {string[]} */
        const order = [];
        vi.mocked(OpenDatabase).mockImplementation(async () => void order.push('open'));
        vi.mocked(refreshLibraryCounts).mockImplementation(() => void order.push('counts'));

        await openDatabaseByPath('/tmp/counts.db');

        expect(order).toEqual(['open', 'counts']);
        vi.mocked(refreshLibraryCounts).mockReset();
    });

    test('the session (and the Direction in it) is restored only once the database is open', async () => {
        /** @type {string[]} */
        const order = [];
        vi.mocked(OpenDatabase).mockImplementation(async () => void order.push('open'));
        vi.mocked(restoreSessionState).mockImplementation(async () => void order.push('restore'));

        await openDatabaseByPath('/tmp/ordered.db');

        expect(order).toEqual(['open', 'restore']);
        vi.mocked(restoreSessionState).mockResolvedValue(undefined);
    });

    test('opening another database closes the Direction of the previous one', async () => {
        openDirectionIdStore.set(4);
        await openDatabaseByPath('/tmp/next.db');
        expect(get(openDirectionIdStore)).toBeNull();
    });

    test('protected copy: prompts for a password instead of opening', async () => {
        vi.mocked(IsProtectedCopyPath).mockResolvedValue(true);

        await openDatabaseByPath('/tmp/protected.blunderdb');

        expect(get(protectedCopyPathStore)).toBe('/tmp/protected.blunderdb');
        expect(get(protectedCopyErrorStore)).toBe('');
        expect(get(activeModal)).toBe(MODAL.PROTECTED_COPY);
        expect(OpenDatabase).not.toHaveBeenCalled();
        // Nothing is persisted as "the" database path until it is unlocked.
        expect(get(databasePathStore)).toBe('');
    });

    test('a mismatched major version opens a warning modal', async () => {
        vi.mocked(CheckDatabaseVersion).mockResolvedValue('1.0.0');
        vi.mocked(GetDatabaseVersion).mockResolvedValue('2.18.0');

        await openDatabaseByPath('/tmp/old.db');

        expect(get(warningMessageStore)).not.toBe('');
        expect(get(activeModal)).toBe(MODAL.WARNING);
    });

    test('matching major versions do not open the warning modal', async () => {
        vi.mocked(CheckDatabaseVersion).mockResolvedValue('2.0.0');
        vi.mocked(GetDatabaseVersion).mockResolvedValue('2.18.0');

        await openDatabaseByPath('/tmp/ok.db');

        expect(get(activeModal)).toBeNull();
    });

    test('opening a library drops the previous search\'s "no match" banner', async () => {
        searchEmptyStore.set(true);
        await openDatabaseByPath('/tmp/normal.db');
        expect(get(searchEmptyStore)).toBe(false);
    });

    test('the "opened" confirmation leaves the status bar, unless another message replaced it', async () => {
        vi.useFakeTimers();
        try {
            await openDatabaseByPath('/tmp/normal.db');
            expect(get(statusBarTextStore)).not.toBe('');
            vi.advanceTimersByTime(5000);
            expect(get(statusBarTextStore)).toBe('');

            await openDatabaseByPath('/tmp/normal.db');
            statusBarTextStore.set('autre message');
            vi.advanceTimersByTime(5000);
            expect(get(statusBarTextStore)).toBe('autre message');
        } finally {
            vi.useRealTimers();
        }
    });

    test('read-only fallback: title and status reflect it, session is still restored', async () => {
        vi.mocked(IsReadOnly).mockResolvedValue(true);

        await openDatabaseByPath('/tmp/locked.db');

        expect(WindowSetTitle).toHaveBeenCalledWith(expect.stringContaining('locked.db'));
        expect(restoreSessionState).toHaveBeenCalled();
    });

    test('IsReadOnly failing is treated as read-write (fails open)', async () => {
        vi.mocked(IsReadOnly).mockRejectedValue(new Error('bridge error'));

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
        vi.mocked(OpenDatabase).mockRejectedValue(new Error('corrupt file'));

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
        vi.mocked(OpenProtectedCopyPath).mockResolvedValue('/tmp/recovered.db');

        await unlockProtectedCopy('right-pass');

        expect(OpenProtectedCopyPath).toHaveBeenCalledWith('/tmp/container.blunderdb', 'right-pass');
        expect(get(protectedCopyPathStore)).toBe('');
        expect(get(activeModal)).toBeNull();
        expect(get(databasePathStore)).toBe('/tmp/recovered.db');
        expect(DeleteProtectedCopyPath).not.toHaveBeenCalled();
    });

    test('removeContainer=true deletes the container after a successful unlock', async () => {
        vi.mocked(OpenProtectedCopyPath).mockResolvedValue('/tmp/recovered.db');

        await unlockProtectedCopy('right-pass', true);

        expect(DeleteProtectedCopyPath).toHaveBeenCalledWith('/tmp/container.blunderdb');
    });

    test('the container survives even if deleting it fails', async () => {
        vi.mocked(OpenProtectedCopyPath).mockResolvedValue('/tmp/recovered.db');
        vi.mocked(DeleteProtectedCopyPath).mockRejectedValue(new Error('permission denied'));

        await expect(unlockProtectedCopy('right-pass', true)).resolves.toBeUndefined();
        expect(get(databasePathStore)).toBe('/tmp/recovered.db');
    });

    test('a wrong password keeps the prompt open with an error, never touching the container', async () => {
        vi.mocked(OpenProtectedCopyPath).mockRejectedValue(new Error('wrong passphrase'));

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

// A draft row id belongs to one library: left open across a change of file,
// its gestures would land on the draft of the same id in the next one.
describe('changing library lets go of the open transcription draft', () => {
    function openDraft() {
        transcriptionStore.set({ id: 1, annotated: { document: { header: {}, actions: [] } } });
    }

    test('opening another database', async () => {
        openDraft();
        await openDatabaseByPath('/tmp/other.db');
        expect(get(transcriptionStore)).toBeNull();
    });

    test('creating a new database', async () => {
        openDraft();
        vi.mocked(SaveDatabaseDialog).mockResolvedValue('/tmp/new.db');
        await newDatabase();
        expect(get(transcriptionStore)).toBeNull();
    });
});

// ── a copy between split panes to take back ──────────────────────────────────

describe('a copy into a group that opens or creates another library', () => {
    test('opening another library in place forgets the copy into that group, not into another', async () => {
        activeGroupId.set(2);
        transferHistoryStore.set([{ dst: 2, receipt: 5, name: 'club' }]);
        await openDatabaseByPath('/tmp/other.db');
        expect(get(transferHistoryStore)).toEqual([]);

        transferHistoryStore.set([{ dst: 1, receipt: 6, name: 'mine' }]);
        await openDatabaseByPath('/tmp/third.db');
        expect(get(transferHistoryStore)).toEqual([{ dst: 1, receipt: 6, name: 'mine' }]);
    });

    test('creating a library in place forgets the copy into that group', async () => {
        activeGroupId.set(2);
        transferHistoryStore.set([{ dst: 2, receipt: 7, name: 'club' }]);
        vi.mocked(SaveDatabaseDialog).mockResolvedValue('/tmp/fresh.db');
        vi.mocked(RecreateDatabase).mockResolvedValue(undefined);
        await newDatabase();
        expect(get(transferHistoryStore)).toEqual([]);
    });
});
