/**
 * transcriptionPlay.test.js — T2.3 : le coup joué au plateau et ses dés déduits.
 *
 * Ce qui se mesure ici est PUR, donc mesurable sans plateau ni composant : les
 * jets encore compatibles après chaque pas, la déduction quand il n'en reste
 * qu'un, le refus de deviner quand il en reste deux, et le budget de gestes
 * d'ux.md §4.1 — quatre pas à la souris sous six secondes.
 *
 * Les coups légaux ne sont pas calculés ici : ils sont DONNÉS, comme le moteur
 * les rend. C'est tout l'intérêt du réducteur, qui ne connaît aucune règle du
 * backgammon et n'a donc rien à simuler pour être éprouvé.
 */

import { describe, test, expect } from 'vitest';
import { selectSource, playHop } from '../services/quizPlay.js';
import { ROLLS, rollKey, newBoardPlay, undoBoardStep, compatibleRolls, choosableRolls, deducedDice } from '../services/transcriptionPlay.js';

const BLACK = 0;

/** Une position : les piles données, le reste vide. */
function positionWith(stacks, mover = BLACK) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [point, [checkers, color]] of Object.entries(stacks)) {
        points[Number(point)] = { checkers, color };
    }
    return {
        id: 0,
        board: { points, bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice: [0, 0],
        score: [7, 7],
        player_on_roll: mover,
        decision_type: 0
    };
}

const step = (from, to) => ({ from, to, hit: false });
const play = (...steps) => ({ steps, notation: steps.map((s) => `${s.from}/${s.to}`).join(' '), result: {} });

// Un milieu de partie ordinaire : cinq pions en 13, trois en 8, cinq en 6.
const POSITION = positionWith({ 13: [5, BLACK], 8: [3, BLACK], 6: [5, BLACK], 24: [2, BLACK] });

// Les coups légaux tels que le moteur les rendrait, jet par jet. Seuls les jets
// qui comptent pour ces tests sont posés : un jet absent est un jet sans coup,
// et il s'écarte de lui-même.
const BY_ROLL = [
    { dice: [6, 1], plays: [play(step(13, 7), step(8, 7)), play(step(13, 7), step(13, 12))] },
    { dice: [6, 2], plays: [play(step(13, 7), step(13, 11))] },
    { dice: [6, 6], plays: [play(step(13, 7), step(13, 7), step(8, 2), step(8, 2))] },
    { dice: [2, 1], plays: [play(step(13, 11), step(13, 12))] }
];

/** Un glissé : la source est choisie, le pion est lâché sur la destination. */
function drag(state, from, to) {
    return playHop(selectSource(state, from), from, to);
}

describe('les jets se réduisent avec les pas', () => {
    test('avant tout pas, tous les jets qui ont un coup sont possibles', () => {
        const state = newBoardPlay(POSITION, BY_ROLL);
        expect(compatibleRolls(state)).toEqual(['21', '61', '62', '66']);
        expect(deducedDice(state)).toBeNull();
    });

    test('un pas ne garde que les jets dont un coup légal le contient', () => {
        const state = drag(newBoardPlay(POSITION, BY_ROLL), 13, 7);
        // 2-1 n'a aucun coup qui déplace un pion de 13 à 7 : il tombe.
        expect(compatibleRolls(state)).toEqual(['61', '62', '66']);
        // Aucun de ces trois coups n'est achevé en un pas : rien à enregistrer.
        expect(choosableRolls(state)).toEqual([]);
        expect(deducedDice(state)).toBeNull();
    });

    test('le second pas laisse un seul jet, et les dés se déduisent', () => {
        let state = drag(newBoardPlay(POSITION, BY_ROLL), 13, 7);
        state = drag(state, 8, 7);
        expect(compatibleRolls(state)).toEqual(['61']);
        expect(deducedDice(state)).toEqual([6, 1]);
    });

    test('les quatre pas d’un double se déduisent en 6-6', () => {
        let state = newBoardPlay(POSITION, BY_ROLL);
        for (const [from, to] of [
            [13, 7],
            [13, 7],
            [8, 2],
            [8, 2]
        ]) {
            state = drag(state, from, to);
        }
        expect(deducedDice(state)).toEqual([6, 6]);
        expect(state.steps).toHaveLength(4);
    });

    // L'ordre est libre : `LegalMoves` déduplique par plateau résultant et ne
    // rend qu'un ordre des deux pas, mais les deux se jouent (quizPlay.js).
    test('l’ordre des pas ne change pas le jet déduit', () => {
        let state = newBoardPlay(POSITION, BY_ROLL);
        state = drag(state, 8, 7);
        state = drag(state, 13, 7);
        expect(deducedDice(state)).toEqual([6, 1]);
    });
});

describe('rien n’est deviné', () => {
    // Deux jets dont le second dé n'est pas jouable : le même pas unique les
    // achève tous les deux. Le plateau ne peut pas trancher, et il ne tranche pas.
    const AMBIGUOUS = [
        { dice: [6, 1], plays: [play(step(13, 7))] },
        { dice: [6, 2], plays: [play(step(13, 7))] }
    ];

    test('deux jets achèvent le même coup : aucun dé n’est déduit', () => {
        const state = drag(newBoardPlay(POSITION, AMBIGUOUS), 13, 7);
        expect(deducedDice(state)).toBeNull();
        // Ce sont eux, et eux seuls, que le triangle laisse cliquables.
        expect(choosableRolls(state)).toEqual(['61', '62']);
    });

    test('un pas qu’aucun coup n’offre ne fait rien, et ne plante pas', () => {
        const state = drag(newBoardPlay(POSITION, BY_ROLL), 13, 7);
        const after = drag(state, 6, 5);
        expect(after.steps).toEqual([{ from: 13, to: 7 }]);
        expect(deducedDice(after)).toBeNull();
    });

    test('un coup laissé à moitié n’enregistre rien', () => {
        const state = drag(newBoardPlay(POSITION, BY_ROLL), 13, 7);
        expect(choosableRolls(state)).toEqual([]);
        expect(deducedDice(state)).toBeNull();
    });

    test('le dernier pas se défait sans reprendre le coup au début', () => {
        let state = drag(newBoardPlay(POSITION, BY_ROLL), 13, 7);
        state = drag(state, 8, 7);
        const back = undoBoardStep(state);
        expect(back.steps).toEqual([{ from: 13, to: 7 }]);
        expect(back.board.points[7].checkers).toBe(1);
    });
});

describe('le budget d’ux.md §4.1 : quatre pas à la souris ≤ 6 s', () => {
    // Les valeurs de Card, Moran & Newell (ux.md §1). Un glissé est un P — viser
    // le pion — et un clic, soit deux B ; les deux H sont le passage clavier ↔
    // souris, au début et à la fin du coup. M est hors comparaison.
    const P = 1.1;
    const B = 0.1;
    const H = 0.4;
    const cost = (gestures) => Math.round((2 * H + gestures * (P + 2 * B)) * 100) / 100;

    /** Le coup joué en comptant les gestes, un glissé par pas. */
    function playByDragging(state, hops) {
        let gestures = 0;
        for (const [from, to] of hops) {
            state = drag(state, from, to);
            gestures += 1;
        }
        return { state, gestures };
    }

    test('un double se joue en quatre gestes, et aucun dé n’est tapé', () => {
        const { state, gestures } = playByDragging(newBoardPlay(POSITION, BY_ROLL), [
            [13, 7],
            [13, 7],
            [8, 2],
            [8, 2]
        ]);
        expect(gestures).toBe(4);
        expect(cost(gestures)).toBeLessThanOrEqual(6);
        // Les dés sont déduits : le budget ne paie que les pas.
        expect(deducedDice(state)).toEqual([6, 6]);
    });

    test('un jet ordinaire se joue en deux gestes', () => {
        const { state, gestures } = playByDragging(newBoardPlay(POSITION, BY_ROLL), [
            [13, 7],
            [8, 7]
        ]);
        expect(gestures).toBe(2);
        expect(cost(gestures)).toBeLessThanOrEqual(3.4);
        expect(deducedDice(state)).toEqual([6, 1]);
    });
});

describe('les jets du triangle', () => {
    test('les vingt et un jets distincts, dé fort d’abord', () => {
        expect(ROLLS).toHaveLength(21);
        expect(ROLLS.every(([high, low]) => high >= low)).toBe(true);
    });

    test('3-1 et 1-3 sont le même jet', () => {
        expect(rollKey([1, 3])).toBe('31');
        expect(rollKey([3, 1])).toBe('31');
    });
});
