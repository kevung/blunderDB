/**
 * themes.test.js — les thèmes nommés (#286, fiche I.30, ADR-0038).
 *
 * Ce qui est vérifié : aucun thème n'oublie un jeton. Un jeton absent laisse la
 * valeur du thème précédent, ce qui est la façon la plus sûre de produire du
 * texte illisible — et c'est le genre de défaut qu'un test attrape et qu'un
 * coup d'œil ne rattrape pas.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { THEMES, THEME_NAMES, THEME_SYSTEM, UI_COLOR_TOKENS, resolveTheme, applyThemeTokens } from '../utils/themes.js';
import { BOARD_COLOR_KEYS } from '../stores/boardColorsStore.js';

afterEach(() => {
    const root = document.documentElement;
    for (const token of UI_COLOR_TOKENS) root.style.removeProperty(token);
    delete root.dataset.theme;
});

describe('les thèmes nommés', () => {
    test('chaque thème définit TOUS les jetons de couleur', () => {
        for (const [name, theme] of Object.entries(THEMES)) {
            for (const token of UI_COLOR_TOKENS) {
                expect(theme.ui[token], `${name} n'a pas ${token}`).toBeTruthy();
            }
        }
    });

    test('chaque thème définit TOUTES les couleurs du plateau', () => {
        for (const [name, theme] of Object.entries(THEMES)) {
            for (const key of BOARD_COLOR_KEYS) {
                expect(theme.board[key], `${name} n'a pas ${key}`).toBeTruthy();
            }
        }
    });

    test('les quatre thèmes attendus sont là, `system` en tête', () => {
        expect(THEME_NAMES[0]).toBe(THEME_SYSTEM);
        expect(THEME_NAMES).toContain('light');
        expect(THEME_NAMES).toContain('dark');
        expect(THEME_NAMES).toContain('contrast');
        expect(THEME_NAMES).toContain('print');
    });

    // Un nom inconnu — une configuration écrite par une version future, un
    // fichier édité à la main — ne doit pas laisser l'interface sans couleurs.
    test('un nom inconnu retombe sur le thème clair', () => {
        expect(resolveTheme('mauve')).toBe('light');
    });

    test('appliquer un thème écrit ses jetons sur la racine', () => {
        const theme = applyThemeTokens('dark');
        expect(document.documentElement.dataset.theme).toBe('dark');
        for (const token of UI_COLOR_TOKENS) {
            expect(document.documentElement.style.getPropertyValue(token)).toBe(theme.ui[token]);
        }
    });

    test('`system` résout vers un thème réel', () => {
        expect(['light', 'dark']).toContain(resolveTheme(THEME_SYSTEM));
    });
});
