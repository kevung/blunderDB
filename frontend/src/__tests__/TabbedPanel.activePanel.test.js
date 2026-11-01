/**
 * The bottom panel fetches its active tab at once, with or without a database: nothing covers it.
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

import TabbedPanel from '../components/TabbedPanel.svelte';
import { panels } from '../components/lazyComponents.js';
import { activeTabStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

function noop() {}

const mount = () =>
    render(TabbedPanel, {
        props: { onLoadPositionsByFilters: noop, onCloseAnalysis: noop, onCloseComment: noop, onOpenCollection: noop, onAddToFilterLibrary: noop }
    });

/** @type {import('vitest').MockInstance} */
let load;

beforeEach(() => {
    load = vi.spyOn(panels, 'load').mockResolvedValue(undefined);
    activeTabStore.set('matches');
    databasePathStore.set('');
});

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    databasePathStore.set('');
});

describe('TabbedPanel', () => {
    test('the default tab is Matches', () => {
        expect(get(activeTabStore)).toBe('matches');
    });

    test('without a database, the active panel is fetched', async () => {
        mount();
        await tick();
        expect(load).toHaveBeenCalledWith('matches');
    });
});
