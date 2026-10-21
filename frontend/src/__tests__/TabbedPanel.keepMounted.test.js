/**
 * The Matches panel stays mounted, hidden, once visited: a tab switch no longer tears its
 * table down. Each of its global effects stops while it is hidden, and its own state (header
 * sort, filter) survives the switch together with the shared list it drives.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

const MATCH = { id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 2 };

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(() => Promise.resolve([MATCH])),
    CountMatches: vi.fn(() => Promise.resolve(1)),
    GetMatchByID: vi.fn(() => Promise.resolve(MATCH)),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    ListTranscriptions: vi.fn(() => Promise.resolve([])),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    GetGamesByMatch: vi.fn(() => Promise.resolve([])),
    GetMatchDetailStats: vi.fn(() => Promise.resolve(null)),
    GetMatchMoveGrades: vi.fn(() => Promise.resolve([])),
    GetMatchTimeSummary: vi.fn(() => Promise.resolve(null)),
    GetMatchOrigin: vi.fn(() => Promise.resolve(null)),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetTabOrder: vi.fn(() => Promise.resolve([])),
    SaveTabOrder: vi.fn(() => Promise.resolve()),
    GetHiddenTabs: vi.fn(() => Promise.resolve([])),
    SaveHiddenTabs: vi.fn(() => Promise.resolve())
}));

import TabbedPanel from '../components/TabbedPanel.svelte';
import MatchPanel from '../components/MatchPanel.svelte';
import PanelProbe, { probeLog } from './fixtures/PanelProbe.svelte';
import { panels } from '../components/lazyComponents.js';
import { activeTabStore, homeDismissedStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { matchListStore, emptyMatchList } from '../stores/matchListStore.js';
import { applyTabPanels } from '../services/tabHandler.js';

const noop = () => {};
const settle = async () => {
    for (let i = 0; i < 5; i++) {
        await tick();
        await new Promise((r) => setTimeout(r, 0));
    }
};

/** As App.svelte does on every tab change. */
async function switchTo(/** @type {string} */ tab) {
    activeTabStore.set(tab);
    applyTabPanels(tab);
    await settle();
}

const matchPanel = () => document.getElementById('matchPanel');
const isHidden = (/** @type {Element | null} */ el) => !!el?.closest('[hidden]');

beforeEach(async () => {
    vi.spyOn(panels, 'load').mockResolvedValue(undefined);
    panels.components.set({ matches: MatchPanel, analysis: PanelProbe, comments: PanelProbe });
    probeLog.mounts = 0;
    probeLog.destroys = 0;
    databasePathStore.set('/tmp/test.db');
    homeDismissedStore.set(true);
    await switchTo('analysis');
    render(TabbedPanel, {
        props: { onLoadPositionsByFilters: noop, onCloseAnalysis: noop, onCloseComment: noop, onOpenCollection: noop, onAddToFilterLibrary: noop }
    });
    await settle();
});

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    panels.components.set({});
    applyTabPanels('analysis');
    databasePathStore.set('');
    homeDismissedStore.set(false);
});

describe('TabbedPanel keeps the Matches panel mounted', () => {
    test('a switch back unhides the same panel; the other tabs still remount', async () => {
        await switchTo('matches');
        const first = matchPanel();
        expect(first).not.toBeNull();
        expect(isHidden(first)).toBe(false);

        await switchTo('analysis');
        expect(matchPanel()).toBe(first);
        expect(isHidden(first)).toBe(true);
        expect(probeLog.mounts).toBe(2);

        await switchTo('matches');
        expect(matchPanel()).toBe(first);
        expect(isHidden(first)).toBe(false);
        expect(probeLog.destroys).toBe(2);
    });

    test('hidden, it claims no key; shown again, it does', async () => {
        await switchTo('matches');
        await switchTo('analysis');
        const hiddenKey = new KeyboardEvent('keydown', { key: '/', bubbles: true, cancelable: true });
        window.dispatchEvent(hiddenKey);
        expect(hiddenKey.defaultPrevented).toBe(false);

        await switchTo('matches');
        const shownKey = new KeyboardEvent('keydown', { key: '/', bubbles: true, cancelable: true });
        window.dispatchEvent(shownKey);
        expect(shownKey.defaultPrevented).toBe(true);
    });

    test('hidden, it does not blur a field another panel just focused', async () => {
        await switchTo('matches');
        await switchTo('analysis');
        const input = /** @type {HTMLInputElement} */ (document.querySelector('[data-testid="probe-input"]'));
        input.focus();
        await fireEvent.click(input);
        expect(document.activeElement).toBe(input);
    });

    test('the header sort survives a switch, and the shared list keeps following it', async () => {
        await switchTo('matches');
        const sortButton = /** @type {HTMLButtonElement} */ (matchPanel()?.querySelector('th.sortable .sort-btn'));
        await fireEvent.click(sortButton);
        await settle();
        const { column, direction } = get(matchListStore);
        expect(column).not.toBeNull();

        await switchTo('analysis');
        await switchTo('matches');
        expect(get(matchListStore).column).toBe(column);
        expect(get(matchListStore).direction).toBe(direction);
        expect(matchPanel()?.querySelector('.sort-arrow')).not.toBeNull();
    });

    test('an embedded sheet beside the hidden panel leaves the shared list and the panel id alone', async () => {
        await switchTo('matches');
        const sortButton = /** @type {HTMLButtonElement} */ (matchPanel()?.querySelector('th.sortable .sort-btn'));
        await fireEvent.click(sortButton);
        await settle();
        const before = get(matchListStore);
        await switchTo('analysis');

        render(MatchPanel, { props: { embeddedMatchId: MATCH.id } });
        await settle();
        expect(document.querySelectorAll('#matchPanel')).toHaveLength(1);
        expect(get(matchListStore).column).toBe(before.column);
        expect(get(matchListStore).direction).toBe(before.direction);
    });

    test('back after another library group took the list, the header and the field follow that list', async () => {
        await switchTo('matches');
        await fireEvent.click(/** @type {HTMLButtonElement} */ (matchPanel()?.querySelector('th.sortable .sort-btn')));
        const field = /** @type {HTMLInputElement} */ (matchPanel()?.querySelector('input.match-filter'));
        await fireEvent.input(field, { target: { value: 'smith' } });
        await new Promise((r) => setTimeout(r, 300));
        await settle();
        expect(get(matchListStore).text).toBe('smith');

        await switchTo('analysis');
        // What a group switch does (libraryScope.putScopedState): the list of a library not read yet.
        matchListStore.set(emptyMatchList());
        await switchTo('matches');

        expect(field.value).toBe('');
        expect(matchPanel()?.querySelector('.sort-arrow')).toBeNull();
        expect(get(matchListStore).column).toBeNull();
        expect(get(matchListStore).text).toBe('');
    });

    test('back with the database closed meanwhile, the previous rows are gone', async () => {
        await switchTo('matches');
        expect(get(matchListStore).rows).toHaveLength(1);
        await switchTo('analysis');
        databasePathStore.set('');
        await switchTo('matches');

        expect(get(matchListStore).rows).toHaveLength(0);
        expect(matchPanel()?.textContent).not.toContain('Alice');
    });
});
