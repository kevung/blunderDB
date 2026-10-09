/**
 * The match sheet at a glance: a summary per player above the tabs, the
 * decisions to review and the review's details each in a tab, the tab the user
 * left remembered, and a final score in the header.
 */
import { describe, test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import MatchReview from '../components/MatchReview.svelte';
import MatchReviewDecisions from '../components/MatchReviewDecisions.svelte';
import MatchReviewDetails from '../components/MatchReviewDetails.svelte';
import { finalScore, rememberTab, rememberedTab } from '../utils/matchSheet.js';

const moves = [100, 101, 102, 103].map((id, i) => ({ move_id: id, player_on_roll: i % 2, game_number: 1, move_number: i + 1 }));
const losses = moves.map((m, i) => ({ move_id: m.move_id, mwc_loss: [0.012, 0.034, 0, 0.021][i], difficulty: 0.002, avoidable: i === 1 }));
const player = (/** @type {number} */ seat) => ({
    pr: seat ? 7.35 : 4.82,
    pr_interval: { available: true, low: 2.9, high: 6.7, units: 3 },
    decisions: 2,
    mwc7: { available: true, loss: 0.058, has_interval: false, elo: 1800 },
    mwc_loss: seat ? 0.055 : 0.017,
    to_review: [{ move_id: seat ? 101 : 102, game_number: 1, move_number: seat ? 2 : 3, decision_type: 'checker', mwc_loss: 0.034, avoidable_loss: 0.03 }],
    pace: { hasty: 1, deliberate: 0, unknown: 0, hasty_loss: 0.034, deliberate_loss: 0 },
    luck: { available: true, result: seat ? -1 : 1, luck: seat ? -0.31 : 0.31, adjusted: seat ? -0.52 : 0.52, error_balance: 0.04, rolls_measured: 3, rolls: 3 },
    difficulty: { decisions: 2, loss: 0.05, difficulty: 0.004, excess: 0.046, ratio: 12.5, avoidable: 1 }
});
const review = { match_id: 7, players: [player(0), player(1)] };

afterEach(cleanup);
beforeEach(() => localStorage.clear());

describe('the summary atop the sheet', () => {
    test('one line per player: PR, the match MWC loss, the luck verdict with one figure', () => {
        const { getByTestId, container } = render(MatchReview, { review, player1: 'Alice', player2: 'Bob' });
        expect(getByTestId('review-pr-0').textContent).toBe('4.82');
        expect(getByTestId('review-mwc-1').textContent).toBe('5.50 %');
        expect(getByTestId('review-luck-0').textContent).toContain('won by the play');
        expect(getByTestId('review-luck-0').textContent).toContain('+52.0 %');
        // The details stay out of the summary: no interval, no 7-point loss, no list.
        expect(container.textContent).not.toContain('[');
        expect(container.textContent).not.toContain('7 pts');
        expect(container.querySelector('ol')).toBeNull();
    });

    test('a match without scored decisions has no summary', () => {
        const { container } = render(MatchReview, { review: { players: [{ decisions: 0 }, { decisions: 0 }] }, player1: 'A', player2: 'B' });
        expect(container.querySelector('[data-testid="match-review"]')).toBeNull();
    });
});

describe('the review details', () => {
    test('the 7-point loss says what it is and why', () => {
        const { getByTestId } = render(MatchReviewDetails, { review, losses, movePositions: moves, player1: 'Alice', player2: 'Bob' });
        const mwc7 = getByTestId('review-mwc7-0');
        expect(mwc7.textContent).toContain('MWC loss scaled to 7 pts');
        expect(mwc7.title).toContain('compare matches of different lengths');
    });

    test('the loss and difficulty table is there, with each player total', () => {
        const { getByTestId } = render(MatchReviewDetails, { review, losses, movePositions: moves, player1: 'Alice', player2: 'Bob' });
        expect(getByTestId('loss-total-0').textContent).toBe('1.20 %');
        expect(getByTestId('ratio-1').textContent).toBe('12.50');
    });

    test('the details leave the decisions to review to their own tab', () => {
        const { container } = render(MatchReviewDetails, { review, losses, movePositions: moves, player1: 'Alice', player2: 'Bob' });
        expect(container.querySelector('[data-testid="review-decision"]')).toBeNull();
    });
});

describe('the decisions to review', () => {
    test('a decision to review shows it on the board', async () => {
        const onselect = vi.fn();
        const { getAllByTestId } = render(MatchReviewDecisions, { review, movePositions: moves, player1: 'Alice', player2: 'Bob', onselect });
        await fireEvent.click(getAllByTestId('review-decision')[1]);
        expect(onselect).toHaveBeenCalledWith(1);
    });

    test('with the split of the errors by time', () => {
        const { getAllByTestId } = render(MatchReviewDecisions, { review, movePositions: moves, player1: 'Alice', player2: 'Bob' });
        expect(getAllByTestId('error-pace')).toHaveLength(2);
    });
});

describe('the tab the sheet opens on', () => {
    test('the transcript, until the user picks another tab', () => {
        expect(rememberedTab()).toBe('transcript');
        rememberTab('charts');
        expect(rememberedTab()).toBe('charts');
    });

    test('a stored name that is no tab, or a storage that throws, opens the transcript', () => {
        localStorage.setItem('blunderdb.matchTab', 'gone');
        expect(rememberedTab()).toBe('transcript');
        const denied = () => {
            throw new Error('denied');
        };
        vi.stubGlobal('localStorage', { getItem: denied, setItem: denied, removeItem: denied });
        try {
            expect(() => rememberTab('stats')).not.toThrow();
            expect(rememberedTab()).toBe('transcript');
        } finally {
            vi.unstubAllGlobals();
        }
    });
});

describe('the final score in the header', () => {
    test('adds the last game to its starting score and names who reached the length', () => {
        const games = [
            { game_number: 2, initial_score: [5, 4], winner: -1, points_won: 4 },
            { game_number: 1, initial_score: [0, 0], winner: 1, points_won: 5 }
        ];
        expect(finalScore(games, 7)).toEqual({ score: [5, 8], winner: 1 });
    });

    test('an unfinished match, or money play, has a score and no winner', () => {
        expect(finalScore([{ game_number: 1, initial_score: [3, 2], winner: 0, points_won: 0 }], 7)).toEqual({ score: [3, 2], winner: null });
        expect(finalScore([{ game_number: 1, initial_score: [0, 0], winner: 1, points_won: 12 }], 0)).toEqual({ score: [12, 0], winner: null });
        expect(finalScore([], 7)).toBeNull();
    });
});
