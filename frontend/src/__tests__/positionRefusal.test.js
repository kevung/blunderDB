/**
 * positionRefusal.test.js — why a board cannot be written to the library.
 *
 * A pure function (#400): the status bar says its answer through
 * isValidPosition, and a panel can read it to disable its save button without
 * saying anything. Each rule is checked on a board that breaks only it.
 */

import { describe, test, expect } from 'vitest';
import { positionRefusal } from '../services/positionRefusal.js';

/** A legal money position: two checkers each, player 1 on roll, cube centred. */
function legal() {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    points[6] = { checkers: 2, color: 0 };
    points[19] = { checkers: 2, color: 1 };
    return {
        board: { points, bearoff: [13, 13] },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [5, 5],
        player_on_roll: 0,
        decision_type: 0
    };
}

describe('positionRefusal', () => {
    test('a legal board is not refused', () => {
        expect(positionRefusal(legal())).toBeNull();
    });

    test('more than 15 checkers for player 1', () => {
        const p = legal();
        p.board.points[8] = { checkers: 14, color: 0 };
        expect(positionRefusal(p)).toBe('status.invalidP1Over15');
    });

    test('more than 15 checkers for player 2', () => {
        const p = legal();
        p.board.points[17] = { checkers: 14, color: 1 };
        expect(positionRefusal(p)).toBe('status.invalidP2Over15');
    });

    test('player 1 has no checker left', () => {
        const p = legal();
        p.board.points[6] = { checkers: 0, color: -1 };
        expect(positionRefusal(p)).toBe('status.invalidP1BorneOff');
    });

    test('player 2 has no checker left', () => {
        const p = legal();
        p.board.points[19] = { checkers: 0, color: -1 };
        expect(positionRefusal(p)).toBe('status.invalidP2BorneOff');
    });

    test('a double when the opponent owns the cube', () => {
        const p = legal();
        p.decision_type = 1;
        p.cube = { owner: 1, value: 1 };
        expect(positionRefusal(p)).toBe('status.invalidCubeUnavailable');
        // The player on roll owning it, or a centred cube, is fine.
        p.cube = { owner: 0, value: 1 };
        expect(positionRefusal(p)).toBeNull();
    });

    test('a double at Crawford (one away)', () => {
        const p = legal();
        p.decision_type = 1;
        p.score = [1, 3];
        expect(positionRefusal(p)).toBe('status.invalidCrawford');
        // The same score on a checker decision is legal.
        p.decision_type = 0;
        expect(positionRefusal(p)).toBeNull();
    });

    test('an unlimited score on one side only', () => {
        const p = legal();
        p.score = [-1, 5];
        expect(positionRefusal(p)).toBe('status.invalidUnlimitedScore');
        p.score = [5, -1];
        expect(positionRefusal(p)).toBe('status.invalidUnlimitedScore');
        // Unlimited on both sides is a money game.
        p.score = [-1, -1];
        expect(positionRefusal(p)).toBeNull();
    });

    test('the first broken rule is the one named', () => {
        const p = legal();
        p.board.points[8] = { checkers: 14, color: 0 };
        p.score = [-1, 5];
        expect(positionRefusal(p)).toBe('status.invalidP1Over15');
    });
});
