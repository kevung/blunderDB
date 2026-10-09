/**
 * The match sheet at a glance: a summary per player above the charts, the
 * review's details folded below the transcript, sections that remember being
 * open, and a final score in the header.
 */
import { describe, test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { createRawSnippet, tick } from 'svelte';
import MatchReview from '../components/MatchReview.svelte';
import MatchReviewDetails from '../components/MatchReviewDetails.svelte';
import MatchSection from '../components/MatchSection.svelte';
import { finalScore, isSectionOpen } from '../utils/matchSheet.js';

const moves = [100, 101, 102, 103].map((id, i) => ({ move_id: id, player_on_roll: i % 2, game_number: 1, move_number: i + 1 }));
const losses = moves.map((m, i) => ({ move_id: m.move_id, mwc_loss: [0.012, 0.034, 0, 0.021][i], difficulty: 0.002, avoidable: i === 1 }));
const player = (seat) => ({
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

    test('a decision to review shows it in the transcript', async () => {
        const onselect = vi.fn();
        const { getAllByTestId } = render(MatchReviewDetails, { review, losses, movePositions: moves, player1: 'Alice', player2: 'Bob', onselect });
        await fireEvent.click(getAllByTestId('review-decision')[1]);
        expect(onselect).toHaveBeenCalledWith(1);
    });

    test('the loss and difficulty table is there, with each player total', () => {
        const { getByTestId } = render(MatchReviewDetails, { review, losses, movePositions: moves, player1: 'Alice', player2: 'Bob' });
        expect(getByTestId('loss-total-0').textContent).toBe('1.20 %');
        expect(getByTestId('ratio-1').textContent).toBe('12.50');
    });
});

describe('a folded section', () => {
    const body = createRawSnippet(() => ({ render: () => '<p>inside</p>' }));

    test('is folded by default, its content still in the page', () => {
        const { getByTestId } = render(MatchSection, { id: 'review', title: 'Review details', children: body });
        const section = /** @type {HTMLDetailsElement} */ (getByTestId('match-section-review'));
        expect(section.open).toBe(false);
        expect(section.textContent).toContain('inside');
    });

    test('remembers being opened, and being folded again', async () => {
        const first = render(MatchSection, { id: 'review', title: 'Review details', children: body });
        const section = /** @type {HTMLDetailsElement} */ (first.getByTestId('match-section-review'));
        section.open = true;
        await fireEvent(section, new Event('toggle'));
        expect(isSectionOpen('review')).toBe(true);
        cleanup();

        const again = render(MatchSection, { id: 'review', title: 'Review details', children: body });
        const reopened = /** @type {HTMLDetailsElement} */ (again.getByTestId('match-section-review'));
        expect(reopened.open).toBe(true);
        reopened.open = false;
        await fireEvent(reopened, new Event('toggle'));
        expect(isSectionOpen('review')).toBe(false);
    });

    test('opening calls onopen, so a section loads what it shows on demand', async () => {
        const onopen = vi.fn();
        const { getByTestId } = render(MatchSection, { id: 'stats', title: 'Stats', onopen, children: body });
        expect(onopen).not.toHaveBeenCalled();
        const section = /** @type {HTMLDetailsElement} */ (getByTestId('match-section-stats'));
        section.open = true;
        await fireEvent(section, new Event('toggle'));
        await tick();
        expect(onopen).toHaveBeenCalledTimes(1);
    });

    test('a storage that throws only leaves the section folded', () => {
        const denied = () => {
            throw new Error('denied');
        };
        vi.stubGlobal('localStorage', { getItem: denied, setItem: denied, removeItem: denied });
        try {
            const { getByTestId } = render(MatchSection, { id: 'info', title: 'Info', children: body });
            expect(/** @type {HTMLDetailsElement} */ (getByTestId('match-section-info')).open).toBe(false);
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
