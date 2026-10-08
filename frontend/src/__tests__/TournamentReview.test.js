import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent, within } from '@testing-library/svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({ GetTournamentReview: vi.fn() }));

import TournamentReview from '../components/TournamentReview.svelte';
import { GetTournamentReview } from '../../wailsjs/go/database/Database.js';

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

const none = { verdict: 'insufficient', delta: 0, low: 0, high: 0 };
/** @param {number} low @param {number} high */
const iv = (low, high) => ({ available: true, low, high, units: 3 });
/** @param {string} key @param {number} decisions @param {object} [versus] */
const cell = (key, decisions, versus = none) => ({
    key,
    decisions,
    pr: 6,
    pr_interval: iv(4, 8),
    usual_decisions: 300,
    usual_pr: 5,
    usual_interval: iv(4.5, 5.5),
    versus
});
const review = {
    tournament_id: 1,
    name: 'Open',
    player: 'Alice',
    matches: 3,
    decisions: 180,
    pr: 7.1,
    pr_interval: iv(5.2, 9.0),
    mwc7: { available: true, loss: 0.12, has_interval: true, low: 0.08, high: 0.16 },
    usual: {
        available: true,
        from: '2024-06-10',
        to: '2025-06-10',
        matches: 40,
        decisions: 2400,
        pr: 5.0,
        pr_interval: iv(4.6, 5.4),
        mwc7: { available: true, loss: 0.09, has_interval: true, low: 0.08, high: 0.1 }
    },
    pr_versus: { verdict: 'worse', delta: 2.1, low: 0.2, high: 4.0 },
    mwc7_versus: { verdict: 'usual', delta: 0.03, low: -0.01, high: 0.07 },
    rounds: [{ round: 1, match_id: 10, opponent: 'Bob', decisions: 60, pr: 6.5, pr_interval: iv(3, 10), mwc7: { available: true, loss: 0.1 }, versus: none }],
    by_rank: [cell('1-30', 90), cell('31-60', 60, { verdict: 'worse', delta: 3, low: 1, high: 5 }), cell('61-90', 30), cell('91+', 0)],
    by_pressure: [cell('dmp', 4), cell('crawford', 10), cell('post_crawford', 12), cell('other', 154)],
    by_clock: [cell('quick', 0), cell('considered', 0)],
    families: [{ GameType: 'holding', Kind: 'checker', Theme: 'blots', Errors: 6, Recoverable: 0.05, Low: 0.01, High: 0.09 }],
    tentative: 2
};

describe('TournamentReview', () => {
    test('shows the figures against the usual level, and a verdict only where it holds', async () => {
        vi.mocked(GetTournamentReview).mockResolvedValue(/** @type {any} */ (review));
        render(TournamentReview, { props: { tournamentId: 1, players: ['Alice', 'Bob'] } });

        expect(await screen.findByText(/2024-06-10/)).toBeTruthy();
        expect(GetTournamentReview).toHaveBeenCalledWith(1, '');
        const verdicts = screen.getAllByText(/^(worse|moins bien)/);
        expect(verdicts.length).toBe(2); // overall PR and the 31–60 slice
        expect(within(screen.getByTestId('tournament-review-rounds')).getByText('Bob')).toBeTruthy();
        expect(screen.getByText(/2 (more to confirm|autres à confirmer)/)).toBeTruthy();
    });

    test('choosing a player reloads the review for that player', async () => {
        vi.mocked(GetTournamentReview).mockResolvedValue(/** @type {any} */ (review));
        render(TournamentReview, { props: { tournamentId: 1, players: ['Alice', 'Bob'] } });
        await screen.findByText(/2024-06-10/);

        await fireEvent.change(screen.getByTestId('tournament-review-player'), { target: { value: 'Bob' } });

        await vi.waitFor(() => expect(GetTournamentReview).toHaveBeenCalledWith(1, 'Bob'));
    });
});
