// @ts-nocheck — Wails runtime mocké.
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { get } from 'svelte/store';

const rt = vi.hoisted(() => ({ WindowFullscreen: vi.fn(), WindowUnfullscreen: vi.fn(), WindowIsFullscreen: vi.fn(() => false) }));
vi.mock('../../wailsjs/runtime/runtime.js', () => rt);

import { directionFullscreenStore, enterDirectionFullscreen, exitDirectionFullscreen, toggleDirectionFullscreen, directionFullscreenKey } from '../services/directionFullscreen.js';
import { handleEscapeCapture, closeOnEscape } from '../services/escapeService.js';
import { openDirectionIdStore } from '../stores/directionStore.js';
import { activeTabStore } from '../stores/uiStore.js';

const key = (k, init = {}) => new KeyboardEvent('keydown', { key: k, cancelable: true, ...init });

beforeEach(() => {
    vi.clearAllMocks();
    rt.WindowIsFullscreen.mockReturnValue(false);
    activeTabStore.set('tournaments');
    openDirectionIdStore.set(1);
});
afterEach(async () => {
    await exitDirectionFullscreen();
    openDirectionIdStore.set(null);
});

describe('plein écran de la Direction', () => {
    test('entre, passe la fenêtre en plein écran, puis la rend à son état', async () => {
        await enterDirectionFullscreen();
        expect(get(directionFullscreenStore)).toBe(true);
        expect(rt.WindowFullscreen).toHaveBeenCalledTimes(1);
        await exitDirectionFullscreen();
        expect(get(directionFullscreenStore)).toBe(false);
        expect(rt.WindowUnfullscreen).toHaveBeenCalledTimes(1);
    });

    test('une fenêtre déjà en plein écran y reste à la sortie', async () => {
        rt.WindowIsFullscreen.mockReturnValue(true);
        await enterDirectionFullscreen();
        expect(rt.WindowFullscreen).not.toHaveBeenCalled();
        await exitDirectionFullscreen();
        expect(rt.WindowUnfullscreen).not.toHaveBeenCalled();
    });

    test('sans page Direction affichée, rien ne se passe', async () => {
        activeTabStore.set('search');
        await enterDirectionFullscreen();
        expect(get(directionFullscreenStore)).toBe(false);
        expect(directionFullscreenKey(key('F11'))).toBe(false);
    });

    test('quitter la page met fin au mode ; changer d’onglet de direction non', async () => {
        await enterDirectionFullscreen();
        openDirectionIdStore.set(2);
        expect(get(directionFullscreenStore)).toBe(true);
        activeTabStore.set('search');
        expect(get(directionFullscreenStore)).toBe(false);
    });

    test('Échap sort, mais un menu ouvert au-dessus se ferme d’abord', async () => {
        await enterDirectionFullscreen();
        const closeMenu = vi.fn();
        const off = closeOnEscape(closeMenu);
        handleEscapeCapture(key('Escape'));
        expect(closeMenu).toHaveBeenCalledTimes(1);
        expect(get(directionFullscreenStore)).toBe(true);
        off();
        handleEscapeCapture(key('Escape'));
        expect(get(directionFullscreenStore)).toBe(false);
    });

    test('F11 nu bascule ; avec un modificateur ou sous un menu, non', async () => {
        expect(directionFullscreenKey(key('F11'))).toBe(true);
        expect(directionFullscreenKey(key('F11', { shiftKey: true }))).toBe(false);
        const off = closeOnEscape(() => {});
        expect(directionFullscreenKey(key('F11'))).toBe(false);
        off();
        await toggleDirectionFullscreen();
        expect(get(directionFullscreenStore)).toBe(true);
        // Sa propre entrée dans la pile d'Échap n'empêche pas d'en sortir.
        expect(directionFullscreenKey(key('F11'))).toBe(true);
    });
});
