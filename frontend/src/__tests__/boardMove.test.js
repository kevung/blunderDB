/**
 * La grammaire d'un coup saisi au plateau (ADR-0086) : quel dé un clic dépense, l'interversion et
 * la validation au clic sur les dés, le grisé des dés (demi-voile d'un double, dés gris d'un coup
 * achevé), la sortie, le coup libre, la reprise au clic droit. Les coups légaux sont écrits à la
 * main, comme `App.LegalMoves` les rendrait : la légalité reste celle de quizPlay.js.
 */
import { describe, test, expect } from 'vitest';
import { newPlay, OFF } from '../services/quizPlay.js';
import { boardRightClick, diceClick, diceLeft, diceShade, orderedDice, playClickedChecker, playIsDone } from '../services/boardMove.js';

/** @param {[number, number][]} pairs */
const steps = (pairs) => pairs.map(([from, to]) => ({ from, to }));

/** @param {any[]} list @returns {any[]} */
const plays = (list) => list.map((pairs) => ({ steps: steps(pairs) }));

/**
 * Une position de Noir (joueur 0, du 24 vers le 1) au trait.
 * @param {Record<number, [number, number]>} stacks point → [pions, couleur]
 * @param {number[]} dice
 */
function position(stacks, dice) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [pt, [n, color]] of Object.entries(stacks)) points[Number(pt)] = { checkers: n, color };
    return { board: { points, bearoff: [0, 0] }, dice, player_on_roll: 0, decision_type: 0 };
}

/** @param {any} play */
const pairs = (play) => play.steps.map((/** @type {any} */ s) => [s.from, s.to]);

// 3-1 : Noir a des pions sur 13, 8 et 6.
const pos31 = position({ 13: [5, 0], 8: [3, 0], 6: [5, 0], 1: [2, 1] }, [3, 1]);
const plays31 = plays([
    [
        [8, 5],
        [6, 5]
    ],
    [
        [13, 10],
        [10, 9]
    ],
    [
        [13, 10],
        [6, 5]
    ],
    [
        [8, 7],
        [8, 5]
    ],
    [
        [13, 12],
        [8, 5]
    ]
]);

describe('un clic sur un pion', () => {
    test('part du dé de gauche, puis du dé de droite', () => {
        let play = newPlay(pos31, plays31);
        play = playClickedChecker(play, 8, [3, 1]);
        expect(play.steps).toEqual([{ from: 8, to: 5, die: 3 }]);
        play = playClickedChecker(play, 6, [3, 1]);
        expect(pairs(play)).toEqual([
            [8, 5],
            [6, 5]
        ]);
        expect(playIsDone(play, [3, 1])).toBe(true);
    });

    test('dés intervertis : le pion part de l’autre dé', () => {
        const play = playClickedChecker(newPlay(pos31, plays31), 8, orderedDice([3, 1], true));
        expect(play.steps).toEqual([{ from: 8, to: 7, die: 1 }]);
    });

    test('un dé injouable pour ce pion : le suivant est essayé', () => {
        // Aucun coup légal n'avance le 6 de trois : le 1 est joué.
        const play = playClickedChecker(newPlay(pos31, plays31), 6, [3, 1]);
        expect(play.steps).toEqual([{ from: 6, to: 5, die: 1 }]);
    });

    test('aucun dé ne le déplace : rien, sans message', () => {
        const start = newPlay(pos31, plays31);
        // Le pion adverse, un point vide, le plateau de sortie.
        expect(playClickedChecker(start, 1, [3, 1])).toBe(start);
        expect(playClickedChecker(start, 20, [3, 1])).toBe(start);
        expect(playClickedChecker(start, OFF, [3, 1])).toBe(start);
    });

    test('deux dés sur le même pion : le second clic part du point d’arrivée', () => {
        let play = playClickedChecker(newPlay(pos31, plays31), 13, [3, 1]);
        play = playClickedChecker(play, 10, [3, 1]);
        expect(pairs(play)).toEqual([
            [13, 10],
            [10, 9]
        ]);
    });

    test('sortie : le dé qui mène au-delà du plateau sort le pion', () => {
        const pos = position({ 3: [1, 0], 2: [1, 0] }, [6, 1]);
        const legal = plays([
            [
                [3, OFF],
                [2, 1]
            ],
            [
                [3, 2],
                [2, OFF]
            ]
        ]);
        const play = playClickedChecker(newPlay(pos, legal), 3, [6, 1]);
        expect(play.steps).toEqual([{ from: 3, to: OFF, die: 6 }]);
        expect(play.board.points[3].checkers).toBe(0);
    });
});

describe('un double', () => {
    const pos22 = position({ 13: [5, 0], 1: [2, 1] }, [2, 2]);
    const legal = plays([
        [
            [13, 11],
            [13, 11],
            [11, 9],
            [11, 9]
        ]
    ]);

    test('quatre pas ; chaque dé dessiné se voile à moitié, puis se grise', () => {
        let play = newPlay(pos22, legal);
        expect(diceShade(play, [2, 2])).toEqual([0, 0]);
        const shades = [];
        for (const point of [13, 13, 11, 11]) {
            play = playClickedChecker(play, point, [2, 2]);
            shades.push(diceShade(play, [2, 2]));
        }
        expect(play.steps).toHaveLength(4);
        expect(shades).toEqual([
            [0.5, 0],
            [1, 0],
            [1, 0.5],
            [1, 1]
        ]);
    });

    test('le clic sur les dés n’intervertit rien, et valide le coup achevé', () => {
        let play = newPlay(pos22, legal);
        expect(diceClick(play, [2, 2])).toBeNull();
        for (const point of [13, 13, 11, 11]) play = playClickedChecker(play, point, [2, 2]);
        expect(diceClick(play, [2, 2])).toBe('validate');
    });
});

describe('le clic sur les dés', () => {
    test('rien de joué : intervertit', () => {
        expect(diceClick(newPlay(pos31, plays31), [3, 1])).toBe('swap');
    });

    test('un pas joué, un seul dé reste : sans effet', () => {
        const play = playClickedChecker(newPlay(pos31, plays31), 8, [3, 1]);
        expect(diceLeft(play, [3, 1])).toEqual([1]);
        expect(diceClick(play, [3, 1])).toBeNull();
    });

    test('un pas joué garde son dé quand l’ordre change ensuite', () => {
        const play = playClickedChecker(newPlay(pos31, plays31), 8, [3, 1]);
        expect(diceShade(play, orderedDice([3, 1], true))).toEqual([0, 1]);
        expect(diceLeft(play, orderedDice([3, 1], true))).toEqual([1]);
    });

    test('coup achevé : valide', () => {
        let play = playClickedChecker(newPlay(pos31, plays31), 8, [3, 1]);
        play = playClickedChecker(play, 6, [3, 1]);
        expect(diceClick(play, [3, 1])).toBe('validate');
    });
});

describe('les dés grisés', () => {
    test('jet simple : chaque dé se grise quand son pas est joué', () => {
        let play = newPlay(pos31, plays31);
        expect(diceShade(play, [3, 1])).toEqual([0, 0]);
        play = playClickedChecker(play, 6, [3, 1]);
        expect(diceShade(play, [3, 1])).toEqual([0, 1]);
        play = playClickedChecker(play, 8, [3, 1]);
        expect(diceShade(play, [3, 1])).toEqual([1, 1]);
    });

    test('coup partiel forcé : le dé non joué se grise aussi, et un clic valide', () => {
        // 6-5 : seul le 6 se joue.
        const pos = position({ 13: [1, 0], 8: [2, 1], 2: [2, 1] }, [6, 5]);
        let play = newPlay(pos, plays([[[13, 7]]]));
        expect(diceShade(play, [6, 5])).toEqual([0, 0]);
        expect(diceClick(play, [6, 5])).toBe('swap');
        play = playClickedChecker(play, 13, [6, 5]);
        expect(play.steps).toEqual([{ from: 13, to: 7, die: 6 }]);
        expect(diceShade(play, [6, 5])).toEqual([1, 1]);
        expect(diceClick(play, [6, 5])).toBe('validate');
    });

    test('aucun coup légal : les dés sont gris d’emblée', () => {
        for (const legal of /** @type {any[][]} */ ([[], [{ steps: [] }]])) {
            const play = newPlay(pos31, legal);
            expect(diceShade(play, [3, 1])).toEqual([1, 1]);
            expect(diceClick(play, [3, 1])).toBe('validate');
        }
    });
});

describe('le coup libre', () => {
    test('le clic avance du premier dé non joué, sans règle', () => {
        const pos = position({ 13: [2, 0], 10: [2, 1] }, [3, 1]);
        let play = { ...newPlay(pos, []), free: true, rolled: [3, 1] };
        // 13-3 tombe sur un point adverse tenu : un coup libre le pose quand même.
        play = playClickedChecker(play, 13, [3, 1]);
        expect(play.steps).toEqual([{ from: 13, to: 10, die: 3 }]);
        expect(diceClick(play, [3, 1])).toBeNull();
        play = playClickedChecker(play, 13, [3, 1]);
        expect(play.steps[1]).toEqual({ from: 13, to: 12, die: 1 });
        expect(diceShade(play, [3, 1])).toEqual([1, 1]);
        expect(diceClick(play, [3, 1])).toBe('validate');
    });

    test('un pas lâché qu’aucun dé ne couvre achève le coup', () => {
        const pos = position({ 13: [2, 0] }, [3, 1]);
        const play = { ...newPlay(pos, []), free: true, rolled: [3, 1], steps: [{ from: 13, to: 5 }] };
        expect(diceClick(play, [3, 1])).toBe('validate');
    });
});

describe('le clic droit sur le damier', () => {
    test('rien de joué : le menu (null) ; un pas joué : reprise', () => {
        const start = newPlay(pos31, plays31);
        expect(boardRightClick(start)).toBeNull();
        expect(boardRightClick(playClickedChecker(start, 8, [3, 1]))).toBe('reset');
        expect(boardRightClick(null)).toBeNull();
    });
});
