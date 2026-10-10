/**
 * openMatchCallers.test.js
 *
 * openMatchInPanel opens a match as a double-click on its row does (match mode, analysis view).
 * Its three callers reach it through the real loader: the import journal, the Stats dashboard
 * and the Stats progression chart each leave the same request for the Matches panel.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup } from '@testing-library/svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/gui/App.js', () => ({ ShowAlert: vi.fn() }));
vi.mock('../../wailsjs/go/database/Database.js', () => ({ LoadComment: vi.fn() }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ ClipboardGetText: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ loadAllPositions: vi.fn() }));

import { openJournalMatch } from '../services/importService.js';
import { activeTabStore, matchOpenRequestStore } from '../stores/uiStore.js';

beforeEach(() => {
    matchOpenRequestStore.set(null);
    activeTabStore.set('analysis');
});
afterEach(cleanup);

describe('openMatchInPanel callers', () => {
    test('the import journal asks for the match and switches to the Matches tab', () => {
        openJournalMatch(42);
        expect(get(matchOpenRequestStore)).toBe(42);
        expect(get(activeTabStore)).toBe('matches');
    });
});
