import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup, screen } from '@testing-library/svelte';

import StatsStudyEffect from '../components/stats/StatsStudyEffect.svelte';
import StatsBiases from '../components/stats/StatsBiases.svelte';
import { studyEffectStore, biasesStore } from '../stores/statsStore.js';

afterEach(() => {
    cleanup();
    studyEffectStore.set(null);
    biasesStore.set(null);
});

const bias = (decisions, plus, minus, verdict) => ({
    Decisions: decisions,
    Plus: plus,
    PlusMP: 100 * plus,
    Minus: minus,
    MinusMP: 50 * minus,
    Bias: (plus - minus) / decisions,
    Low: (plus - minus) / decisions - 0.05,
    High: (plus - minus) / decisions + 0.05,
    Verdict: verdict
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
                    Before: { Decisions: 200, Errors: 12, Loss: 0.24, Rate: 0.0012 },
                    After: { Decisions: 150, Errors: 3, Loss: 0.03, Rate: 0.0002 },
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
                { MoverAway: 0, OpponentAway: 0, ...bias(25, 1, 6, 'too_little') },
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
    });
});
