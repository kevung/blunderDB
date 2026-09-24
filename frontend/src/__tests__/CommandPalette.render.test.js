/**
 * CommandPalette.render.test.js — the palette mounted, driven by the keyboard
 * the way the app drives it (#287).
 *
 * The global dispatcher is registered on window BEFORE the component is
 * rendered, as App.svelte does (its onMount runs after the children mount, but
 * the escape registry listens in capture): a keyboard test that registers it
 * afterwards hides the ordering bugs #414 was about.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => ({
    ...(await importOriginal()),
    GetAllMatches: vi.fn(() => Promise.resolve([{ id: 5, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', tournament_name: 'Open de Lyon' }])),
    LoadFilters: vi.fn(() => Promise.resolve([{ id: 1, name: 'Blitz ratés', command: 's gt:blitz', pinned: true }]))
}));

vi.mock('../services/importService.js', () => ({
    importDatabase: vi.fn(),
    importPosition: vi.fn(),
    importFolder: vi.fn(),
    pastePosition: vi.fn()
}));

import { handleKeyDown } from '../services/keyboardService.js';
import { activeTabStore, commandPaletteOpenStore, matchOpenRequestStore, commandTextStore, showCommandInputStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import CommandPalette from '../components/CommandPalette.svelte';

async function settle() {
    for (let i = 0; i < 4; i++) {
        await new Promise((r) => setTimeout(r, 0));
        await tick();
    }
}

/** @param {string} key @param {KeyboardEventInit} [init] */
function pressOnWindow(key, init = {}) {
    const target = document.activeElement || document.body;
    target.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...init }));
}

function input() {
    return /** @type {HTMLInputElement} */ (document.querySelector('.palette-input'));
}

function options() {
    return [...document.querySelectorAll('[role="option"]')];
}

beforeEach(() => {
    window.addEventListener('keydown', handleKeyDown);
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('analysis');
    commandPaletteOpenStore.set(false);
    matchOpenRequestStore.set(null);
});

afterEach(() => {
    window.removeEventListener('keydown', handleKeyDown);
    cleanup();
    commandPaletteOpenStore.set(false);
    showCommandInputStore.set(false);
    commandTextStore.set('');
    databasePathStore.set('');
});

describe('the command palette', () => {
    test('Ctrl+Maj+P opens it with the field focused, and does not open the comments', async () => {
        render(CommandPalette);
        pressOnWindow('P', { ctrlKey: true, shiftKey: true });
        await settle();
        expect(document.querySelector('[data-testid="command-palette"]')).toBeTruthy();
        expect(document.activeElement).toBe(input());
        expect(get(activeTabStore)).toBe('analysis');
        // At rest it lists the pinned filter first, then the tabs and commands.
        expect(options()[0].textContent).toContain('Blitz ratés');
    });

    test('Ctrl+K is still the Anki tab, not the palette', async () => {
        render(CommandPalette);
        pressOnWindow('k', { ctrlKey: true });
        await settle();
        expect(get(commandPaletteOpenStore)).toBe(false);
        expect(get(activeTabStore)).toBe('anki');
    });

    test('typing ranks approximately, arrows move, Enter runs the choice', async () => {
        render(CommandPalette);
        commandPaletteOpenStore.set(true);
        await settle();
        await fireEvent.input(input(), { target: { value: 'anki' } });
        await tick();
        expect(options()[0].textContent).toContain('Anki');
        expect(options()[0].getAttribute('aria-selected')).toBe('true');
        // "a" alone matches more than one entry: the arrows have somewhere to go.
        await fireEvent.input(input(), { target: { value: 'a' } });
        await tick();
        expect(options().length).toBeGreaterThan(1);

        await fireEvent.keyDown(input(), { key: 'ArrowDown' });
        await tick();
        expect(options()[1].getAttribute('aria-selected')).toBe('true');
        await fireEvent.keyDown(input(), { key: 'ArrowUp' });
        await tick();
        expect(options()[0].getAttribute('aria-selected')).toBe('true');

        await fireEvent.input(input(), { target: { value: 'anki' } });
        await tick();
        await fireEvent.keyDown(input(), { key: 'Enter' });
        await settle();
        expect(get(commandPaletteOpenStore)).toBe(false);
        expect(get(activeTabStore)).toBe('anki');
    });

    test('a match is found by its player and opened through the match panel', async () => {
        render(CommandPalette);
        commandPaletteOpenStore.set(true);
        await settle();
        await fireEvent.input(input(), { target: { value: 'alce bob' } });
        await tick();
        expect(options()[0].textContent).toContain('Alice – Bob');
        await fireEvent.keyDown(input(), { key: 'Enter' });
        await settle();
        expect(get(matchOpenRequestStore)).toBe(5);
        expect(get(activeTabStore)).toBe('matches');
    });

    test('a search command opens the command line with the command written', async () => {
        render(CommandPalette);
        commandPaletteOpenStore.set(true);
        await settle();
        await fireEvent.input(input(), { target: { value: 's' } });
        await tick();
        await fireEvent.keyDown(input(), { key: 'Enter' });
        await settle();
        expect(get(showCommandInputStore)).toBe(true);
        expect(get(commandTextStore)).toBe('s ');
    });

    test('Escape closes it before anything else sees the key', async () => {
        render(CommandPalette);
        commandPaletteOpenStore.set(true);
        await settle();
        pressOnWindow('Escape');
        await settle();
        expect(get(commandPaletteOpenStore)).toBe(false);
        expect(document.querySelector('[data-testid="command-palette"]')).toBeNull();
    });

    test('Ctrl+Maj+P again closes it, from inside its own field', async () => {
        render(CommandPalette);
        commandPaletteOpenStore.set(true);
        await settle();
        pressOnWindow('P', { ctrlKey: true, shiftKey: true });
        await settle();
        expect(get(commandPaletteOpenStore)).toBe(false);
    });

    test('a query that matches nothing says so', async () => {
        render(CommandPalette);
        commandPaletteOpenStore.set(true);
        await settle();
        await fireEvent.input(input(), { target: { value: 'zzzzqqq' } });
        await tick();
        expect(options()).toHaveLength(0);
        expect(document.querySelector('.palette-list .empty')).toBeTruthy();
    });
});
