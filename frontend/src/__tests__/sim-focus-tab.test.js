import { describe, test, expect, afterEach } from 'vitest';
import { get } from 'svelte/store';
import { handleKeyDown } from '../services/keyboardService.js';
import { activeTabStore } from '../stores/uiStore.js';
import { openDirectionIdStore, directionViewLoadedStore } from '../stores/directionStore.js';

afterEach(() => {
    document.body.innerHTML = '';
    openDirectionIdStore.set(null);
    directionViewLoadedStore.set(false);
});

function tab(init = {}) {
    const e = new KeyboardEvent('keydown', { key: 'Tab', code: 'Tab', bubbles: true, cancelable: true, ...init });
    handleKeyDown(e);
    return e;
}

describe('Tab avec le focus sur body, page Direction affichée', () => {
    test('garde l’onglet et place le focus sur le premier élément de la Direction', () => {
        document.body.innerHTML =
            '<div class="scrollable-content"><div class="direction-view" tabindex="-1"><div class="pane" hidden><button id="h">h</button></div><button id="first">a</button></div></div>';
        openDirectionIdStore.set(1);
        directionViewLoadedStore.set(true);
        activeTabStore.set('tournaments');
        const e = tab();
        expect(get(activeTabStore)).toBe('tournaments');
        expect(e.defaultPrevented).toBe(true);
        expect(document.activeElement?.id).toBe('first');
    });
    test('Maj+Tab aussi', () => {
        document.body.innerHTML = '<div class="direction-view" tabindex="-1"><button id="first">a</button></div>';
        openDirectionIdStore.set(1);
        directionViewLoadedStore.set(true);
        activeTabStore.set('tournaments');
        tab({ shiftKey: true });
        expect(get(activeTabStore)).toBe('tournaments');
        expect(document.activeElement?.id).toBe('first');
    });
    test('hors Direction, Tab ouvre toujours la Recherche', () => {
        activeTabStore.set('tournaments');
        tab();
        expect(get(activeTabStore)).toBe('search');
    });
});
