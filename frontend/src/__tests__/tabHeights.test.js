import { describe, test, expect, beforeEach, vi } from 'vitest';

const saved = vi.fn(() => Promise.resolve());
let stored = {};
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetTabPanelHeights: () => Promise.resolve(stored),
    SaveTabPanelHeight: (/** @type {any[]} */ ...a) => saved(...a)
}));

import { initTabHeights, panelHeightValue, rememberPanelHeight } from '../utils/tabHeights.js';
import { DEFAULT_PANEL_HEIGHT } from '../stores/panelLayoutStore.js';

beforeEach(() => {
    saved.mockClear();
    stored = {};
});

describe('panel height', () => {
    test('sans hauteur mémorisée, la hauteur par défaut', async () => {
        await initTabHeights();
        expect(panelHeightValue()).toBe(DEFAULT_PANEL_HEIGHT);
    });
    test('une hauteur choisie est la même pour tous les onglets et persistée sous la clé commune', async () => {
        await initTabHeights();
        rememberPanelHeight(460);
        expect(panelHeightValue()).toBe(460);
        await vi.waitFor(() => expect(saved).toHaveBeenCalledWith('*', 460));
    });
    test('la hauteur persistée est relue au démarrage; les entrées par onglet d une ancienne config sont ignorées', async () => {
        stored = { '*': 520, eval: 300 };
        await initTabHeights();
        expect(panelHeightValue()).toBe(520);
    });
    test('une valeur absurde n est ni retenue ni envoyée', async () => {
        await initTabHeights();
        rememberPanelHeight(NaN);
        rememberPanelHeight(-5);
        expect(panelHeightValue()).toBe(DEFAULT_PANEL_HEIGHT);
        expect(saved).not.toHaveBeenCalled();
    });
});
