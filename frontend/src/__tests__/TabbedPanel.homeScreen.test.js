/**
 * Behind the home screen the bottom panel cannot be seen: its active tab (Matches by default,
 * the largest chunk) is left to the idle preload instead of being fetched before the first
 * paint, and is fetched as soon as a database opens or the home screen is set aside.
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

import TabbedPanel from '../components/TabbedPanel.svelte';
import { panels, activePanelToLoad } from '../components/lazyComponents.js';
import { activeTabStore, homeDismissedStore } from '../stores/uiStore.js';
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
    homeDismissedStore.set(false);
    databasePathStore.set('');
});

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    homeDismissedStore.set(false);
    databasePathStore.set('');
});

describe('activePanelToLoad', () => {
    test('nothing behind the home screen, the active tab otherwise', () => {
        expect(activePanelToLoad('matches', { homeScreenShown: true })).toBeNull();
        expect(activePanelToLoad('matches', { homeScreenShown: false })).toBe('matches');
    });
});

describe('TabbedPanel behind the home screen', () => {
    test('the default tab is Matches', () => {
        expect(get(activeTabStore)).toBe('matches');
    });

    test('the active panel is not fetched while the home screen covers it', async () => {
        mount();
        await tick();
        expect(load).not.toHaveBeenCalled();
    });

    test('opening a database fetches it', async () => {
        mount();
        await tick();
        databasePathStore.set('/tmp/x.db');
        await tick();
        expect(load).toHaveBeenCalledWith('matches');
    });

    test('setting the home screen aside fetches it', async () => {
        mount();
        await tick();
        homeDismissedStore.set(true);
        await tick();
        expect(load).toHaveBeenCalledWith('matches');
    });
});
