import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';

const load = vi.fn();
vi.mock('../services/positionLoader.js', () => ({ loadPositionsFromStatsSelection: (/** @type {any[]} */ ...args) => load(...args) }));

import StatsStudyEffect from '../components/stats/StatsStudyEffect.svelte';
import StatsBiases from '../components/stats/StatsBiases.svelte';
import { studyEffectStore, biasesStore, studyLoopFilterStore } from '../stores/statsStore.js';

afterEach(() => {
    cleanup();
    studyEffectStore.set(null);
    biasesStore.set(null);
    studyLoopFilterStore.set(null);
    load.mockClear();
});

/**
 * @param {number} decisions
 * @param {number} plus
 * @param {number} minus
 * @param {string} verdict
 */
const bias = (decisions, plus, minus, verdict) => ({
    Decisions: decisions,
    Plus: plus,
    PlusMP: 100 * plus,
    Minus: minus,
    MinusMP: 50 * minus,
    Bias: (plus - minus) / decisions,
    Low: (plus - minus) / decisions - 0.05,
    High: (plus - minus) / decisions + 0.05,
    Verdict: verdict,
    Positions: decisions - 1,
    PlusPositions: plus,
    MinusPositions: minus
});

describe('StatsStudyEffect', () => {
    test('shows each studied family with both windows, the gain and its interval', () => {
        studyEffectStore.set({
            MinDecisions: 30,
            Unstudied: 4,
            Families: [
                {
                    GameType: 'holding',
                    Kind: 'checker',
                    Theme: 'blots',
                    StudiedOn: '2026-09-01',
                    Studied: 3,
                    Before: { Decisions: 200, Errors: 12, Loss: 0.24, Rate: 0.0012, Positions: 190 },
                    After: { Decisions: 150, Errors: 3, Loss: 0.03, Rate: 0.0002, Positions: 0 },
                    Gain: 0.001,
                    Low: 0.0003,
                    High: 0.0017,
                    Verdict: 'improved'
                }
            ]
        });
        render(StatsStudyEffect);
        const rows = screen.getAllByTestId('effect-family');
        expect(rows).toHaveLength(1);
        expect(rows[0].textContent).toContain('2026-09-01');
        expect(rows[0].textContent).toContain('12.00');
        expect(rows[0].textContent).toContain('+10.00');
        expect(rows[0].textContent).toContain('[+3.00, +17.00]');
        expect(screen.getByTestId('study-effect').textContent).toContain('4');
    });

    test('a window count shows its positions and opens them under the measure filter', async () => {
        const filter = { playerName: 'Alice' };
        studyLoopFilterStore.set(filter);
        studyEffectStore.set({
            MinDecisions: 30,
            Unstudied: 0,
            Families: [
                {
                    GameType: 'holding',
                    Kind: 'checker',
                    Theme: 'blots',
                    StudiedOn: '2026-09-01',
                    Studied: 1,
                    Before: { Decisions: 200, Errors: 12, Loss: 0.24, Rate: 0.0012, Positions: 190 },
                    After: { Decisions: 0, Errors: 0, Loss: 0, Rate: 0, Positions: 0 },
                    Gain: 0,
                    Low: 0,
                    High: 0,
                    Verdict: 'insufficient'
                }
            ]
        });
        render(StatsStudyEffect);
        expect(screen.queryByTestId('effect-after')).toBeNull();
        const link = screen.getByTestId('effect-before');
        expect(link.textContent).toBe('190');
        expect(screen.getByTestId('effect-family').textContent).toContain('(200)');
        await fireEvent.click(link);
        expect(load).toHaveBeenCalledWith(filter, { Kind: 'family_window', GameType: 'holding', FamilyKind: 'checker', StudiedOn: '2026-09-01', Window: 'before' });
    });

    test('says so when nothing has been studied', () => {
        studyEffectStore.set({ MinDecisions: 30, Unstudied: 2, Families: [] });
        render(StatsStudyEffect);
        expect(screen.queryAllByTestId('effect-family')).toHaveLength(0);
    });
});

describe('StatsBiases', () => {
    test('lists the three biases and only the score cells with enough decisions', () => {
        biasesStore.set({
            MinDecisions: 20,
            TakePass: bias(40, 10, 2, 'too_much'),
            Doubles: bias(30, 2, 3, 'balanced'),
            Blots: bias(10, 1, 0, 'insufficient'),
            BlotsUnread: 5,
            DoublesByScore: [
                { Money: true, MoverAway: 0, OpponentAway: 0, ...bias(25, 1, 6, 'too_little') },
                { MoverAway: 3, OpponentAway: 5, ...bias(4, 1, 0, 'insufficient') }
            ]
        });
        render(StatsBiases);
        const rows = screen.getAllByTestId('bias-row');
        expect(rows).toHaveLength(3);
        expect(rows[0].textContent).toContain('+20.0 %');
        expect(rows[0].textContent).toContain('1000 mp');
        const cells = screen.getAllByTestId('bias-score');
        expect(cells).toHaveLength(1);
        expect(cells[0].textContent).toContain('-20.0 %');
        expect(cells[0].textContent).toContain('money');
        expect(screen.getByTestId('bias-unread').textContent).toContain('5 of 15');
        expect(screen.getByTestId('bias-unread').textContent).toContain('33.3 %');
    });

    test('each count shows its positions and opens them as a "bias" selection', async () => {
        const filter = { playerName: 'Alice' };
        studyLoopFilterStore.set(filter);
        biasesStore.set({
            MinDecisions: 20,
            TakePass: bias(40, 10, 0, 'too_much'),
            Doubles: bias(30, 2, 3, 'balanced'),
            Blots: bias(10, 1, 0, 'insufficient'),
            BlotsUnread: 0,
            DoublesByScore: [{ MoverAway: 3, OpponentAway: 5, ...bias(25, 1, 6, 'too_little') }]
        });
        render(StatsBiases);
        expect(screen.getByTestId('bias-takePass-all').textContent).toBe('39');
        const cells = [...screen.getAllByTestId('bias-row')[0].querySelectorAll('td')].map((c) => c.textContent?.trim());
        expect(cells.slice(1, 7)).toEqual(['39', '40', '10 (1000 mp)', '10', '0 (0 mp)', '0']);
        expect(screen.queryByTestId('bias-takePass-minus')).toBeNull();
        await fireEvent.click(screen.getByTestId('bias-takePass-plus'));
        expect(load).toHaveBeenLastCalledWith(filter, { Kind: 'bias', Bias: 'take_pass', BiasScore: '', BiasSide: 'plus' });
        await fireEvent.click(screen.getByTestId('bias-score-3-5-minus'));
        expect(load).toHaveBeenLastCalledWith(filter, { Kind: 'bias', Bias: 'doubles', BiasScore: '3-5', BiasSide: 'minus' });
    });
});
