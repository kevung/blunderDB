import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, fireEvent, waitFor } from '@testing-library/svelte';

const contrast = vi.fn();
const load = vi.fn();
const score = vi.fn();
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    PlayerContrast: (...args) => contrast(...args),
    ScoreMoves: (...args) => score(...args)
}));
vi.mock('../services/positionLoader.js', () => ({ loadPositionsFromSelection: (...args) => load(...args) }));

const { default: PlayerComparison } = await import('../components/stats/PlayerComparison.svelte');

const row = (name) => ({ name, decisions: 100, pr: 5, pr_checker: 5, pr_cube: 5, snowie_er: 5, blunders: 2, matches: 3, wins: 1, losses: 2 });

afterEach(() => {
    cleanup();
    contrast.mockReset();
    load.mockReset();
    score.mockReset();
});

describe('PlayerComparison contrast', () => {
    test('asks the backend only on demand, then opens the positions', async () => {
        contrast.mockResolvedValue({
            player_a: 'Alice',
            player_b: 'Bob',
            threshold_mp: 50,
            common_positions: 4,
            positions: [
                { position_id: 7, well_played: 'b' },
                { position_id: 3, well_played: 'a' }
            ]
        });
        const { container } = render(PlayerComparison, { a: row('Alice'), b: row('Bob') });
        expect(contrast).not.toHaveBeenCalled();
        await fireEvent.click(container.querySelector('[data-testid="player-contrast"] button'));
        await waitFor(() => expect(contrast).toHaveBeenCalledTimes(1));
        expect(contrast.mock.calls[0].slice(0, 2)).toEqual(['Alice', 'Bob']);
        const open = await waitFor(() => container.querySelectorAll('[data-testid="player-contrast"] button')[0]);
        await fireEvent.click(open);
        expect(load).toHaveBeenCalledWith([7, 3]);
    });

    test('says so when no common position contrasts', async () => {
        contrast.mockResolvedValue({ common_positions: 5, positions: [] });
        const { container } = render(PlayerComparison, { a: row('Alice'), b: row('Bob') });
        await fireEvent.click(container.querySelector('[data-testid="player-contrast"] button'));
        await waitFor(() => expect(container.querySelector('[data-testid="player-contrast"] p')).not.toBeNull());
        expect(container.querySelector('[data-testid="player-contrast"] button')).toBeNull();
    });

    test('offers to score the plays an import left unscored, then reloads', async () => {
        contrast
            .mockResolvedValueOnce({ common_positions: 0, positions: [], unscored_moves: 9 })
            .mockResolvedValueOnce({ common_positions: 4, positions: [{ position_id: 7, well_played: 'b' }], unscored_moves: 0 });
        score.mockResolvedValue(9);
        const { container } = render(PlayerComparison, { a: row('Alice'), b: row('Bob') });
        await fireEvent.click(container.querySelector('[data-testid="player-contrast"] button'));
        await waitFor(() => expect(container.querySelector('[data-testid="contrast-unscored"]')).not.toBeNull());
        await fireEvent.click(container.querySelector('[data-testid="player-contrast"] button'));
        await waitFor(() => expect(contrast).toHaveBeenCalledTimes(2));
        expect(score).toHaveBeenCalledTimes(1);
        await waitFor(() => expect(container.querySelector('[data-testid="contrast-unscored"]')).toBeNull());
    });
});
