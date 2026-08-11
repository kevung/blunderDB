/**
 * TabbedPanel.a11y.test.js
 *
 * fiche-09: the tab bar was driven only by onmousedown (drag-to-reorder) —
 * no onclick, no onkeydown, no role="tablist"/"tab". Tab + Enter did
 * nothing. This locks the keyboard behaviour added on top of the existing
 * drag: Enter/Space activate the focused tab, arrow keys move focus with a
 * roving tabindex (manual-activation ARIA tabs pattern — arrows move focus
 * only, they don't themselves switch tabs, since switching remounts the
 * panel below).
 *
 * Scope note: tests only navigate to tabs whose panel does nothing on mount
 * while its own openPanels flag is unset (matches/tournaments/collections) —
 * see MatchPanel/TournamentPanel/CollectionPanel `visible` guards — so no
 * Wails backend mocking is needed here.
 */

import { describe, test, expect, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { tick } from 'svelte';

import TabbedPanel from '../components/TabbedPanel.svelte';
import { activeTabStore } from '../stores/uiStore.js';
import { openPanels } from '../stores/uiStore.js';

function noop() {}

function mount() {
    return render(TabbedPanel, {
        props: {
            onLoadPositionsByFilters: noop,
            onCloseAnalysis: noop,
            onCloseComment: noop,
            onOpenCollection: noop,
            onAddToFilterLibrary: noop
        }
    });
}

beforeEach(() => {
    activeTabStore.set('matches');
    openPanels.set(new Set());
});

afterEach(() => {
    cleanup();
    activeTabStore.set('matches');
    openPanels.set(new Set());
});

describe('TabbedPanel — roles', () => {
    test('the tab bar exposes role="tablist" and each tab role="tab"', () => {
        const { container } = mount();
        expect(container.querySelector('[role="tablist"]')).not.toBeNull();
        const tabs = container.querySelectorAll('[role="tab"]');
        expect(tabs.length).toBeGreaterThan(0);
    });

    test('aria-selected reflects activeTabStore', () => {
        const { container } = mount();
        const matchesTab = container.querySelector('[data-testid="tab-matches"]');
        const tournamentsTab = container.querySelector('[data-testid="tab-tournaments"]');
        expect(matchesTab.getAttribute('aria-selected')).toBe('true');
        expect(tournamentsTab.getAttribute('aria-selected')).toBe('false');
    });
});

describe('TabbedPanel — keyboard activation (Enter/Space)', () => {
    test('Enter on a focused tab activates it', async () => {
        const { container } = mount();
        const tournamentsTab = container.querySelector('[data-testid="tab-tournaments"]');

        await fireEvent.keyDown(tournamentsTab, { key: 'Enter' });
        await tick();

        expect(get(activeTabStore)).toBe('tournaments');
        expect(tournamentsTab.getAttribute('aria-selected')).toBe('true');
    });

    test('Space on a focused tab activates it', async () => {
        const { container } = mount();
        const collectionsTab = container.querySelector('[data-testid="tab-collections"]');

        await fireEvent.keyDown(collectionsTab, { key: ' ' });
        await tick();

        expect(get(activeTabStore)).toBe('collections');
    });
});

describe('TabbedPanel — roving tabindex + arrow navigation', () => {
    test('only the selected tab is tabbable initially', () => {
        const { container } = mount();
        const matchesTab = container.querySelector('[data-testid="tab-matches"]');
        const tournamentsTab = container.querySelector('[data-testid="tab-tournaments"]');
        expect(matchesTab.getAttribute('tabindex')).toBe('0');
        expect(tournamentsTab.getAttribute('tabindex')).toBe('-1');
    });

    test('ArrowRight moves focus to the next tab without switching tabs', async () => {
        const { container } = mount();
        const matchesTab = container.querySelector('[data-testid="tab-matches"]');
        const tournamentsTab = container.querySelector('[data-testid="tab-tournaments"]');

        matchesTab.focus();
        await fireEvent.keyDown(matchesTab, { key: 'ArrowRight' });
        await tick();

        expect(document.activeElement).toBe(tournamentsTab);
        expect(tournamentsTab.getAttribute('tabindex')).toBe('0');
        expect(matchesTab.getAttribute('tabindex')).toBe('-1');
        // Manual activation: moving focus does not select the tab.
        expect(get(activeTabStore)).toBe('matches');
    });

    test('ArrowLeft from the first tab wraps to the last tab', async () => {
        const { container } = mount();
        const tabs = container.querySelectorAll('[role="tab"]');
        const firstTab = tabs[0];
        const lastTab = tabs[tabs.length - 1];

        firstTab.focus();
        await fireEvent.keyDown(firstTab, { key: 'ArrowLeft' });
        await tick();

        expect(document.activeElement).toBe(lastTab);
        expect(get(activeTabStore)).toBe('matches');
    });

    test('a tab activated via Enter becomes the new roving tabindex target', async () => {
        const { container } = mount();
        const tournamentsTab = container.querySelector('[data-testid="tab-tournaments"]');
        const matchesTab = container.querySelector('[data-testid="tab-matches"]');

        tournamentsTab.focus();
        await fireEvent.keyDown(tournamentsTab, { key: 'Enter' });
        await tick();

        expect(tournamentsTab.getAttribute('tabindex')).toBe('0');
        expect(matchesTab.getAttribute('tabindex')).toBe('-1');
    });
});
