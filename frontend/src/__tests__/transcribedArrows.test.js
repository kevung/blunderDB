/**
 * transcribedArrows.test.js — l'oracle des flèches en transcription : le coup écrit, rejoué sur la
 * position que le Replay donne, un pas qui ne se joue plus marqué illégal depuis la case écrite ;
 * le coup du moteur seulement pour une ligne désignée, et dans un autre ton.
 */
import { describe, test, expect } from 'vitest';
import { recordedSteps, transcriptionArrows, screenMoves } from '../services/transcribedArrows.js';

/**
 * @param {number} onRoll
 * @param {Record<number, [number, number]>} stacks point → [camp, pions]
 */
function position(onRoll, stacks) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [point, [color, checkers]] of Object.entries(stacks)) points[Number(point)] = { color, checkers };
    return { board: { points, bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice: [3, 1], score: [7, 7], player_on_roll: onRoll };
}

describe('recordedSteps', () => {
    test('a legal play: every step from the point written, none flagged', () => {
        expect(recordedSteps('8/5 6/5', position(0, { 8: [0, 2], 6: [0, 5] }))).toEqual([
            { from: 8, to: 5, index: 0 },
            { from: 6, to: 5, index: 0 }
        ]);
    });

    test('a correction upstream emptied the source: the step still leaves the point written, flagged', () => {
        expect(recordedSteps('8/5 6/5', position(0, { 6: [0, 5] }))).toEqual([
            { from: 8, to: 5, index: 0, illegal: true },
            { from: 6, to: 5, index: 0 }
        ]);
    });

    test('a point held by the opponent is no landing; a lone opponent is hit', () => {
        expect(recordedSteps('8/5 6/2', position(0, { 8: [0, 1], 6: [0, 1], 5: [1, 2], 2: [1, 1] })).map((m) => !!m.illegal)).toEqual([true, false]);
    });

    test('a checker on the bar enters first, whatever the order written', () => {
        expect(recordedSteps('24/21 bar/24', position(0, { 25: [0, 1], 24: [0, 1] })).map((m) => !!m.illegal)).toEqual([false, false]);
        expect(recordedSteps('13/10 6/5', position(0, { 25: [0, 1], 13: [0, 2], 6: [0, 2] })).map((m) => !!m.illegal)).toEqual([true, true]);
    });

    test('a doubled step with one checker on its point: the second one is flagged', () => {
        expect(recordedSteps('13/10(2)', position(0, { 13: [0, 1] })).map((m) => !!m.illegal)).toEqual([false, true]);
    });

    test('the side on top counts in its own numbering', () => {
        // Camp 1 : son 8 est le point 17 du modèle, sa barre le point 0.
        expect(recordedSteps('8/5 6/5', position(1, { 19: [1, 2] })).map((m) => !!m.illegal)).toEqual([true, false]);
        expect(recordedSteps('bar/22', position(1, { 0: [1, 1] })).map((m) => !!m.illegal)).toEqual([false]);
    });
});

describe('transcriptionArrows', () => {
    const info = { kind: 'checker', notation: '8/5 6/5', before: position(0, { 6: [0, 5] }) };

    test('by default, the written move, its faulty step flagged', () => {
        const arrows = transcriptionArrows({ info });
        expect(arrows?.tone).toBe('record');
        expect(arrows?.notation).toBe('8/5 6/5');
        expect(arrows?.moves[0]).toEqual({ from: 8, to: 5, index: 0, illegal: true });
    });

    test('a line the user points at takes over, in the engine tone', () => {
        const arrows = transcriptionArrows({ info, engineMove: '13/10 6/5' });
        expect(arrows?.tone).toBe('engine');
        expect(arrows?.moves).toEqual([
            { from: 13, to: 10, index: 0 },
            { from: 6, to: 5, index: 0 }
        ]);
    });

    test('pointing at the written move is the written move', () => {
        expect(transcriptionArrows({ info, engineMove: '8/5 6/5' })?.tone).toBe('record');
    });

    test('a dance, an unrecorded move or no Action draws nothing of its own', () => {
        expect(transcriptionArrows({ info: { ...info, kind: 'dance', notation: 'Cannot Move' } })).toBeNull();
        expect(transcriptionArrows({ info: { ...info, kind: 'unrecorded', notation: '???' } })).toBeNull();
        expect(transcriptionArrows({})).toBeNull();
    });

    test('flipped, the steps are turned for the drawing and keep their flag', () => {
        expect(screenMoves(transcriptionArrows({ info })?.moves ?? [], true)[0]).toEqual({ from: 17, to: 20, index: 0, illegal: true });
    });
});

// La notation est numérotée depuis le camp qui joue : sa barre y est 25, et le plateau la dessine
// du côté de ce camp, en bas comme en haut.
describe('une entrée depuis la barre part de la barre du camp qui joue', () => {
    test.each([
        [0, false, 25],
        [1, true, 0]
    ])('camp %i (dessiné en haut : %s) : la flèche du moteur part du point %i', (camp, flipped, bar) => {
        const info = { kind: 'checker', notation: '13/7', before: position(camp, { 13: [camp, 2] }) };
        const moves = screenMoves(transcriptionArrows({ info, engineMove: 'bar/22' })?.moves ?? [], flipped);
        expect(moves[0]).toEqual({ from: bar, to: flipped ? 3 : 22, index: 0 });
    });

    test.each([0, 1])('camp %i : le coup écrit depuis la barre se joue depuis sa barre', (camp) => {
        const barPoint = camp === 0 ? 25 : 0;
        const info = { kind: 'checker', notation: 'bar/22', before: position(camp, { [barPoint]: [camp, 1] }) };
        const moves = transcriptionArrows({ info })?.moves ?? [];
        expect(moves).toEqual([{ from: 25, to: 22, index: 0 }]);
    });
});
