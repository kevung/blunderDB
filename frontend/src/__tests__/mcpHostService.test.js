/**
 * mcpHostService : une demande d'outil d'affichage ouvre une vue nommée, lance la recherche
 * par la barre de commande et bascule ; hors des modes où une recherche tourne, elle est refusée.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const handlers = vi.hoisted(() => /** @type {Record<string, (e?: any) => void>} */ ({}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: (/** @type {string} */ name, /** @type {(e?: any) => void} */ fn) => {
        handlers[name] = fn;
        return () => {};
    },
    EventsOff: () => {}
}));
const commands = vi.hoisted(() => /** @type {string[]} */ ([]));
vi.mock('../commandProcessor.js', () => ({ processCommand: (c) => commands.push(c) }));

import { openSearchView, showPositionById, initMCPHost } from '../services/mcpHostService.js';
import { viewStore } from '../stores/viewStore.js';
import { statusBarModeStore, statusBarTextStore, matchPanelRefreshTriggerStore, dbMutationCounterStore } from '../stores/uiStore';

beforeEach(() => {
    commands.length = 0;
    statusBarModeStore.set('NORMAL');
});

describe('openSearchView', () => {
    test('opens a named view, runs the search in it and shows it', () => {
        const before = get(viewStore.views).length;
        expect(openSearchView('mes erreurs de videau', 'cube E>80')).toBe(true);
        const views = get(viewStore.views);
        expect(views).toHaveLength(before + 1);
        const active = views.find((v) => v.id === get(viewStore.activeViewId));
        expect(active?.name).toBe('mes erreurs de videau');
        expect(commands).toEqual(['s cube E>80']);
    });

    test('a position is shown through a search on its id', () => {
        showPositionById(42);
        expect(commands).toEqual(['s id42']);
    });

    test('refused in a match: the view would inherit a mode that refuses the search', () => {
        statusBarModeStore.set('MATCH');
        const before = get(viewStore.views).length;
        expect(openSearchView('x', 'cube')).toBe(false);
        expect(get(viewStore.views)).toHaveLength(before);
        expect(commands).toEqual([]);
        expect(get(statusBarTextStore)).toBeTruthy();
    });
});

describe('a write tool', () => {
    test('the library on screen is read again by every panel', async () => {
        await initMCPHost();
        const matches = get(matchPanelRefreshTriggerStore);
        const mutations = get(dbMutationCounterStore);
        handlers['mcp:wrote']?.();
        expect(get(matchPanelRefreshTriggerStore)).toBe(matches + 1);
        expect(get(dbMutationCounterStore)).toBe(mutations + 1);
    });
});
