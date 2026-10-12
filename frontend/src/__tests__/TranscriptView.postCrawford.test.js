/**
 * TranscriptView.postCrawford.test.js — la première partie d'un brouillon
 * repris en cours de match se déclare post-Crawford.
 *
 * La case n'existe que là où la déclaration vaut (rules.go, ensureGame) : la
 * première partie, un joueur à un point du match. Cochée, elle reflète l'en-tête
 * du document, que le moteur seul écrit.
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';

import TranscriptView from '../components/TranscriptView.svelte';

const POSITION = () => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [6, 3],
    score: [1, 3],
    player_on_roll: 0,
    decision_type: 0
});

/**
 * Un 5 points dont la première partie s'ouvre à `score`.
 *
 * @param {[number, number]} score
 * @param {{ postCrawford?: boolean, number?: number }} [opts]
 */
function opened(score, { postCrawford = false, number = 1 } = {}) {
    const rows = [{ side: 0, kind: 'checker', dice: [6, 3], score }];
    return {
        document: { header: { match_length: 5, post_crawford: postCrawford }, actions: rows, cursor: 1 },
        actions: rows.map((a, index) => ({
            index,
            side: a.side,
            kind: a.kind,
            before: POSITION(),
            has_position: true,
            notation: '8/2 6/3',
            game_index: 0,
            game_number: number,
            score,
            move_number: 0,
            opens_game: true,
            inconsistencies: []
        })),
        games: [{ number, initial_score: score, declared: true, derived_score: score, winner: 0, points_won: 0, crawford: !postCrawford, finished: false, first: 0, last: 0 }],
        next: { expects: 'checker', side: 1, position: POSITION(), crawford: !postCrawford },
        score,
        cursor: 1
    };
}

const box = (/** @type {HTMLElement} */ c) => /** @type {HTMLInputElement | null} */ (c.querySelector('.post-crawford input'));

afterEach(() => cleanup());

describe('la case post-Crawford', () => {
    test('un joueur à un point du match : la case est offerte et reflète l’en-tête', () => {
        const { container } = render(TranscriptView, { props: { annotated: opened([4, 2], { postCrawford: true }), onPostCrawford: () => {} } });
        expect(box(container)?.checked).toBe(true);
    });

    test('aucun joueur à un point : pas de case', () => {
        const { container } = render(TranscriptView, { props: { annotated: opened([3, 2]), onPostCrawford: () => {} } });
        expect(box(container)).toBeNull();
    });

    test('sans rappel (un match enregistré), pas de case', () => {
        const { container } = render(TranscriptView, { props: { annotated: opened([4, 2]) } });
        expect(box(container)).toBeNull();
    });

    test('cocher la case demande la déclaration', async () => {
        /** @type {boolean[]} */
        const got = [];
        const { container } = render(TranscriptView, { props: { annotated: opened([2, 4]), onPostCrawford: (/** @type {boolean} */ on) => got.push(on) } });
        const input = /** @type {HTMLInputElement} */ (box(container));
        expect(input.checked).toBe(false);
        await fireEvent.click(input);
        expect(got).toEqual([true]);
    });
});
