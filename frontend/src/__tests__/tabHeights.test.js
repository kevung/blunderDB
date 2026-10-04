import { describe, test, expect, beforeEach, vi } from 'vitest';

const saved = vi.fn(() => Promise.resolve());
let stored = {};
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetTabPanelHeights: () => Promise.resolve(stored),
    SaveTabPanelHeight: (...a) => saved(...a)
}));

import { initTabHeights, tabPanelHeight, rememberTabHeight } from '../utils/tabHeights.js';
import { DEFAULT_PANEL_HEIGHT } from '../stores/panelLayoutStore.js';

beforeEach(() => {
    saved.mockClear();
    stored = {};
});

describe('tabHeights', () => {
    test('un onglet sans hauteur mémorisée prend la hauteur par défaut, pas celle d un autre', async () => {
        await initTabHeights();
        rememberTabHeight('stats', 460);
        expect(tabPanelHeight('stats')).toBe(460);
        expect(tabPanelHeight('search')).toBe(DEFAULT_PANEL_HEIGHT);
        await vi.waitFor(() => expect(saved).toHaveBeenCalledWith('stats', 460));
    });
    test("la hauteur d'une ancienne config vaut pour l'onglet sans hauteur propre", async () => {
        stored = { '*': 520, eval: 300 };
        await initTabHeights();
        expect(tabPanelHeight('stats')).toBe(520);
        expect(tabPanelHeight('eval')).toBe(300);
    });
    test('les hauteurs persistées sont relues au démarrage', async () => {
        stored = { eval: 300 };
        await initTabHeights();
        expect(tabPanelHeight('eval')).toBe(300);
    });
    test('une valeur absurde n est ni retenue ni envoyée', async () => {
        await initTabHeights();
        rememberTabHeight('stats', NaN);
        rememberTabHeight('stats', -5);
        expect(tabPanelHeight('stats')).toBe(DEFAULT_PANEL_HEIGHT);
        expect(saved).not.toHaveBeenCalled();
    });
});
