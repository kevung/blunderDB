/**
 * panelKeyboardGuard.test.js
 *
 * fiche-09: MatchPanel/TournamentPanel/CollectionPanel each install their own
 * document-level keydown handler for in-panel navigation (j/k, Escape by
 * levels, Delete…), and each had reimplemented "let this key through to the
 * rest of the app" ad hoc — and the copies had drifted: TournamentPanel and
 * CollectionPanel swallowed '?' and Space (help/command-line couldn't open
 * from those panels), and MatchPanel had no editable-field check at all
 * (typing 'j' in a text field triggered row navigation instead of being
 * typed).
 *
 * All three now delegate that decision to keyboardService.panelKeyGuard.
 * These tests exercise the *real* mounted panel components — not the guard
 * function in isolation — dispatching a keydown and checking whether it
 * reaches a `window` listener. A panel that calls stopPropagation() for a key
 * that should traverse is the exact bug this locks down.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    GetAllMatches: vi.fn(() => Promise.resolve([])),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    DeleteMatch: vi.fn(() => Promise.resolve()),
    UpdateMatch: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    GetGamesByMatch: vi.fn(() => Promise.resolve([])),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SetMatchTournamentByName: vi.fn(() => Promise.resolve()),
    SwapMatchPlayers: vi.fn(() => Promise.resolve()),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve()),
    GetMatchDetailStats: vi.fn(() => Promise.resolve(null)),
    CreateTournament: vi.fn(),
    DeleteTournament: vi.fn(),
    UpdateTournament: vi.fn(),
    GetTournamentMatches: vi.fn(() => Promise.resolve([])),
    RemoveMatchFromTournament: vi.fn(),
    AddMatchToTournament: vi.fn(),
    UpdateTournamentComment: vi.fn(),
    ReorderTournamentMatches: vi.fn(),
    CreateCollection: vi.fn(),
    GetAllCollections: vi.fn(() => Promise.resolve([])),
    DeleteCollection: vi.fn(),
    AddPositionToCollection: vi.fn(),
    RemovePositionFromCollection: vi.fn(),
    GetCollectionPositions: vi.fn(() => Promise.resolve([])),
    ReorderCollectionPositions: vi.fn(),
    ReorderCollections: vi.fn(),
    UpdateCollection: vi.fn(),
    GetPositionCollections: vi.fn(() => Promise.resolve([])),
    GetPositionIndexMap: vi.fn(() => Promise.resolve({}))
}));

import MatchPanel from '../components/MatchPanel.svelte';
import TournamentPanel from '../components/TournamentPanel.svelte';
import CollectionPanel from '../components/CollectionPanel.svelte';
import { openPanels, PANEL } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { lastVisitedMatchStore } from '../stores/positionStore.js';
import { tournamentsStore } from '../stores/tournamentStore.js';

let windowSpy;

beforeEach(() => {
    windowSpy = vi.fn();
    window.addEventListener('keydown', windowSpy);
});

afterEach(() => {
    cleanup();
    window.removeEventListener('keydown', windowSpy);
    openPanels.set(new Set());
    databasePathStore.set('');
    tournamentsStore.set([]);
    lastVisitedMatchStore.set({ matchID: null, currentIndex: 0, gameNumber: 1 });
});

/** Dispatch a keydown targeted at `document` itself — never matches the editable-field check. */
function keyOnDocument(opts) {
    document.dispatchEvent(new KeyboardEvent('keydown', { bubbles: true, cancelable: true, ...opts }));
}

/** Dispatch a keydown on a focused, throwaway <input> so event.target is editable. */
function keyInEditableField(opts) {
    const input = document.createElement('input');
    document.body.appendChild(input);
    input.focus();
    input.dispatchEvent(new KeyboardEvent('keydown', { bubbles: true, cancelable: true, ...opts }));
    input.remove();
}

describe('MatchPanel keyboard guard', () => {
    beforeEach(async () => {
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
        render(MatchPanel);
        await tick();
        await tick();
        windowSpy.mockClear(); // ignore any keydown noise from mounting
    });

    test('Ctrl combos traverse to the rest of the app', () => {
        keyOnDocument({ key: 'x', ctrlKey: true });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('Space traverses (opens the command line)', () => {
        keyOnDocument({ key: ' ', code: 'Space' });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('"?" traverses (opens help)', () => {
        keyOnDocument({ key: '?' });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('SHIFT-J and SHIFT-K traverse (they switch views, like Ctrl-PageUp/PageDown)', () => {
        keyOnDocument({ key: 'J', shiftKey: true });
        expect(windowSpy).toHaveBeenCalled();
        windowSpy.mockClear();
        keyOnDocument({ key: 'K', shiftKey: true });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('typing "j" in an editable field traverses instead of navigating rows', () => {
        keyInEditableField({ key: 'j' });
        expect(windowSpy).toHaveBeenCalled();
    });

    test("'j' outside an editable field is the panel's own row-navigation key, not forwarded", () => {
        keyOnDocument({ key: 'j' });
        expect(windowSpy).not.toHaveBeenCalled();
    });
});

describe('TournamentPanel keyboard guard', () => {
    beforeEach(async () => {
        openPanels.set(new Set([PANEL.TOURNAMENT]));
        render(TournamentPanel);
        await tick();
        windowSpy.mockClear();
    });

    test('Ctrl combos traverse to the rest of the app', () => {
        keyOnDocument({ key: 'y', ctrlKey: true });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('Space traverses (opens the command line)', () => {
        keyOnDocument({ key: ' ', code: 'Space' });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('"?" traverses (opens help)', () => {
        keyOnDocument({ key: '?' });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('SHIFT-J and SHIFT-K traverse (they switch views, like Ctrl-PageUp/PageDown)', () => {
        keyOnDocument({ key: 'J', shiftKey: true });
        expect(windowSpy).toHaveBeenCalled();
        windowSpy.mockClear();
        keyOnDocument({ key: 'K', shiftKey: true });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('typing in an editable field traverses', () => {
        keyInEditableField({ key: 'j' });
        expect(windowSpy).toHaveBeenCalled();
    });

    test("'j' outside an editable field is the panel's own list-navigation key, not forwarded", () => {
        keyOnDocument({ key: 'j' });
        expect(windowSpy).not.toHaveBeenCalled();
    });
});

describe('CollectionPanel keyboard guard', () => {
    beforeEach(async () => {
        openPanels.set(new Set([PANEL.COLLECTION]));
        render(CollectionPanel);
        await tick();
        windowSpy.mockClear();
    });

    test('Ctrl combos traverse to the rest of the app', () => {
        keyOnDocument({ key: 'b', ctrlKey: true });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('Space traverses (opens the command line)', () => {
        keyOnDocument({ key: ' ', code: 'Space' });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('"?" traverses (opens help)', () => {
        keyOnDocument({ key: '?' });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('SHIFT-J and SHIFT-K traverse (they switch views, like Ctrl-PageUp/PageDown)', () => {
        keyOnDocument({ key: 'J', shiftKey: true });
        expect(windowSpy).toHaveBeenCalled();
        windowSpy.mockClear();
        keyOnDocument({ key: 'K', shiftKey: true });
        expect(windowSpy).toHaveBeenCalled();
    });

    test('typing in an editable field traverses', () => {
        keyInEditableField({ key: 'j' });
        expect(windowSpy).toHaveBeenCalled();
    });

    test("'Delete' is the panel's own key, not forwarded (j/k/arrows are, for position browsing)", () => {
        keyOnDocument({ key: 'Delete' });
        expect(windowSpy).not.toHaveBeenCalled();
    });
});
