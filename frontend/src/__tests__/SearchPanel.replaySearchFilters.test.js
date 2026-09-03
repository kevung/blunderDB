/**
 * SearchPanel.replaySearchFilters.test.js — #203
 *
 * `s D xD65` excludes the 6-5 roll; double-clicking the same line in the
 * search-history tab used to bring the 6-5 back, silently: `xD` (exclude
 * dice), `id` (position id) and the derived comment-presence mode were parsed
 * on the typed-command path (commandProcessor.js's parseFilters) but dropped
 * on the history/library replay path (searchFilterService.js's
 * parseSearchCommand, which had its own, less complete copy of the grammar).
 * Both paths now share one parser (parseSearchTokens); this locks the replay
 * path so the fields it used to drop reach onLoadPositionsByFilters again.
 */

import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

import SearchPanel from '../components/SearchPanel.svelte';
import { searchHistoryStore } from '../stores/searchHistoryStore.js';
import { filterLibraryStore } from '../stores/filterLibraryStore.js';

afterEach(() => {
    cleanup();
    searchHistoryStore.set([]);
    filterLibraryStore.set([]);
});

async function replayHistoryEntry(command) {
    searchHistoryStore.set([{ timestamp: 1700000000000, command, position: null, excludePosition: null }]);
    const onLoadPositionsByFilters = vi.fn();
    const utils = render(SearchPanel, { props: { onLoadPositionsByFilters, onAddToFilterLibrary: vi.fn() } });
    await tick();

    // Switch to the "history" sub-tab (index 1, as in SearchPanel.saveFilterDialog.test.js).
    const subTabButtons = utils.container.querySelectorAll('.sub-tab-btn');
    await fireEvent.click(subTabButtons[1]);
    await tick();

    const row = utils.container.querySelector('.history-table tbody tr');
    expect(row).not.toBeNull();
    await fireEvent.dblClick(row);
    await tick();

    expect(onLoadPositionsByFilters).toHaveBeenCalledTimes(1);
    return onLoadPositionsByFilters.mock.calls[0][0];
}

describe('SearchPanel — replaying a history entry keeps every filter (#203)', () => {
    test('xD (exclude-dice) survives the double-click replay', async () => {
        const opts = await replayHistoryEntry('s D xD65');
        expect(opts.diceRollFilter).toBe(true);
        expect(opts.exceptDiceFilter).toBe('65');
    });

    test('several xD tokens survive, joined with ";"', async () => {
        const opts = await replayHistoryEntry('s xD65 xD54');
        expect(opts.exceptDiceFilter).toBe('65;54');
    });

    test('id (position id) survives the double-click replay', async () => {
        const opts = await replayHistoryEntry('s id5,10');
        expect(opts.positionIDsFilter).toBe('5,10');
    });

    test('co/xco keep resolving to the right comment presence on replay', async () => {
        expect((await replayHistoryEntry('s co')).filters).toContain('co');
        expect((await replayHistoryEntry('s xco')).filters).toContain('xco');
    });
});
