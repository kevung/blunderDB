/**
 * Opening a file another library group holds brings that group forward instead. When the switch
 * is refused (a Duel, a training session, a job), the status bar keeps the reason, not a claim
 * that the group is shown.
 */

import { test, expect, vi, beforeEach } from 'vitest';
import { get, writable } from 'svelte/store';

vi.mock('../../wailsjs/go/gui/App.js', () => ({ LibraryGroupHolding: vi.fn(async () => 2) }));
vi.mock('../../wailsjs/go/database/Database.js', () => ({ IsProtectedCopyPath: vi.fn(async () => false), OpenDatabase: vi.fn() }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ WindowSetTitle: vi.fn(), Quit: vi.fn() }));
vi.mock('../stores/libraryGroupStore.js', async () => {
    const { statusBarTextStore } = await import('../stores/uiStore.js');
    return {
        activeGroupId: writable(1),
        activateGroup: vi.fn(async () => {
            if (globalThis.__refuse) {
                statusBarTextStore.set({ i18nKey: 'libraryGroups.duel', i18nParams: null });
                return false;
            }
            return true;
        })
    };
});

import { openDatabaseByPath } from '../services/databaseService.js';
import { OpenDatabase } from '../../wailsjs/go/database/Database.js';
import { statusBarTextStore } from '../stores/uiStore.js';

beforeEach(() => {
    statusBarTextStore.set('');
    vi.mocked(OpenDatabase).mockClear();
});

test('a refused switch leaves its reason in the status bar', async () => {
    globalThis.__refuse = true;
    await openDatabaseByPath('/libs/b.db');
    expect(get(statusBarTextStore)).toEqual({ i18nKey: 'libraryGroups.duel', i18nParams: null });
    expect(OpenDatabase).not.toHaveBeenCalled();
});

test('a switch done says the file is already open in the group shown', async () => {
    globalThis.__refuse = false;
    await openDatabaseByPath('/libs/b.db');
    expect(get(statusBarTextStore)).toEqual({ i18nKey: 'libraryGroups.alreadyOpen', i18nParams: { name: 'b.db' } });
    expect(OpenDatabase).not.toHaveBeenCalled();
});
