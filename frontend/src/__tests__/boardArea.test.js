import { describe, test, expect, afterEach } from 'vitest';
import { isOnBoard } from '../services/boardArea.js';

// La zone principale (`.scrollable-content`) montre le plateau, ou la page Direction qui le
// remplace (ADR-0047). La molette et Tab n'ont leur sens de plateau (changer de position,
// ouvrir la Recherche) qu'au-dessus du plateau : sur la page Direction, la molette défile et
// Tab passe au champ suivant (simulation 2026-09, #434, #435).
describe('isOnBoard', () => {
    afterEach(() => {
        document.body.innerHTML = '';
    });

    test('un élément du plateau est sur le plateau', () => {
        document.body.innerHTML = '<div class="scrollable-content"><canvas id="c"></canvas></div>';
        expect(isOnBoard(document.getElementById('c'))).toBe(true);
    });

    test('un élément de la page Direction n’est pas sur le plateau', () => {
        document.body.innerHTML = '<div class="scrollable-content"><div class="direction-view"><div class="body"><input id="score" /></div></div></div>';
        expect(isOnBoard(document.getElementById('score'))).toBe(false);
    });

    test('un élément hors de la zone principale n’est pas sur le plateau', () => {
        document.body.innerHTML = '<div class="scrollable-content"></div><button id="b"></button>';
        expect(isOnBoard(document.getElementById('b'))).toBe(false);
    });

    test('rien (null) n’est pas sur le plateau', () => {
        expect(isOnBoard(null)).toBe(false);
    });
});
