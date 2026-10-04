import { describe, test, expect, beforeEach } from 'vitest';
import { rememberedTabHeight, rememberTabHeight } from '../utils/tabHeights.js';

beforeEach(() => localStorage.clear());

describe('tabHeights', () => {
    test('chaque onglet garde sa hauteur', () => {
        rememberTabHeight('stats', 460);
        rememberTabHeight('search', 220.4);
        expect(rememberedTabHeight('stats')).toBe(460);
        expect(rememberedTabHeight('search')).toBe(220);
        expect(rememberedTabHeight('eval')).toBeNull();
    });
    test('une valeur absurde ou un stockage corrompu ne casse rien', () => {
        rememberTabHeight('stats', NaN);
        rememberTabHeight('stats', -5);
        expect(rememberedTabHeight('stats')).toBeNull();
        localStorage.setItem('blunderdb.tabPanelHeights', '{oops');
        expect(rememberedTabHeight('stats')).toBeNull();
    });
});
