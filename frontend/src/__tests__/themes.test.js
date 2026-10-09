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
    root.style.removeProperty('color-scheme');
    delete root.dataset.theme;
    delete root.dataset.scheme;
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

    // #402 : le schéma déclaré dit au moteur comment peindre ses contrôles
    // natifs. Un thème à surface sombre déclaré `light` garderait des boutons
    // et des cases blancs sur sa surface sombre ; l'inverse, des contrôles
    // sombres sur une surface claire. La clarté de la surface tranche.
    test('le schéma de chaque thème suit la clarté de sa surface', () => {
        const luminance = (hex) => {
            const [r, g, b] = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255).map((c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
            return 0.2126 * r + 0.7152 * g + 0.0722 * b;
        };
        for (const [name, theme] of Object.entries(THEMES)) {
            const expected = luminance(theme.ui['--color-surface']) < 0.5 ? 'dark' : 'light';
            expect(theme.scheme, `${name}`).toBe(expected);
        }
    });

    test('appliquer un thème déclare son schéma de couleur sur la racine', () => {
        applyThemeTokens('dark');
        expect(document.documentElement.style.colorScheme).toBe('dark');
        expect(document.documentElement.dataset.scheme).toBe('dark');
        applyThemeTokens('print');
        expect(document.documentElement.style.colorScheme).toBe('light');
        expect(document.documentElement.dataset.scheme).toBe('light');
    });

    test('`system` résout vers un thème réel', () => {
        expect(['light', 'dark']).toContain(resolveTheme(THEME_SYSTEM));
    });
});

// WCAG 2.x contrast of two #rrggbb colours.
function luminance(hex) {
    const channel = (/** @type {number} */ i) => {
        const c = parseInt(hex.slice(1 + 2 * i, 3 + 2 * i), 16) / 255;
        return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
    };
    return 0.2126 * channel(0) + 0.7152 * channel(1) + 0.0722 * channel(2);
}
function contrast(a, b) {
    const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
    return (hi + 0.05) / (lo + 0.05);
}

describe('contraste AA des jetons', () => {
    const PAIRS = [
        ['--color-text', '--color-surface'],
        ['--color-text', '--color-surface-alt'],
        ['--color-text-muted', '--color-surface'],
        ['--color-text-muted', '--color-surface-alt'],
        ['--color-primary', '--color-surface'],
        ['--color-danger', '--color-surface']
    ];
    for (const [name, theme] of Object.entries(THEMES)) {
        test(`${name} : le texte atteint 4,5:1 sur ses fonds`, () => {
            for (const [fg, bg] of PAIRS) {
                expect(contrast(theme.ui[fg], theme.ui[bg]), `${name} ${fg} sur ${bg}`).toBeGreaterThanOrEqual(4.5);
            }
        });
    }
});
