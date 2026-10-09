import { describe, test, expect, vi, afterEach, beforeEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

const db = vi.hoisted(() => ({ LoadAnalysis: vi.fn(() => Promise.resolve(null)), SaveAnalysis: vi.fn(), SavePosition: vi.fn() }));
vi.mock('../../wailsjs/go/database/Database.js', () => db);

import { analysisStore } from '../stores/analysisStore.js';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { trainingSessionStore } from '../stores/trainingTabStore.js';
import { analysisChallengeStore, analysisMaskStore, toggleAnalysisChallenge, revealAnalysis } from '../stores/analysisChallengeStore.js';
import AnalysisPanel from '../components/AnalysisPanel.svelte';

/** @param {number} d1 @param {number} d2 */
const withDice = (d1, d2) => ({ ...emptyPosition(), id: 7, dice: [d1, d2] });

describe('AnalysisPanel — the challenge mask', () => {
    beforeEach(() => {
        positionStore.set(withDice(3, 1));
        analysisStore.set(
            /** @type {any} */ ({
                checkerAnalysis: {
                    moves: [
                        { move: '8/5 6/5', equity: 0.1 },
                        { move: '24/21 13/11', equity: -0.05 }
                    ]
                },
                analysisType: 'CheckerMove'
            })
        );
    });

    afterEach(() => {
        cleanup();
        analysisChallengeStore.set(false);
        trainingSessionStore.set(null);
    });

    test('off, the analysis shows', async () => {
        const { container } = render(AnalysisPanel);
        await tick();
        expect(container.querySelector('.answer-mask')).toBeNull();
        expect(container.textContent).toContain('8/5 6/5');
    });

    test('on, a grey zone covers the analysis until clicked', async () => {
        toggleAnalysisChallenge();
        const { container } = render(AnalysisPanel);
        await tick();
        const mask = /** @type {HTMLElement} */ (container.querySelector('button.answer-mask'));
        expect(mask).not.toBeNull();
        expect(container.textContent).not.toContain('8/5 6/5');
        await fireEvent.click(mask);
        expect(container.querySelector('.answer-mask')).toBeNull();
        expect(container.textContent).toContain('8/5 6/5');
    });

    test('the mask comes back on the next position, not on a repaint of the same one', async () => {
        toggleAnalysisChallenge();
        const { container } = render(AnalysisPanel);
        await tick();
        await fireEvent.click(/** @type {HTMLElement} */ (container.querySelector('button.answer-mask')));
        positionStore.set({ ...get(positionStore) });
        await tick();
        expect(container.querySelector('.answer-mask')).toBeNull();
        positionStore.set(withDice(6, 5));
        await tick();
        expect(container.querySelector('button.answer-mask')).not.toBeNull();
    });

    test('a position left then come back to is masked again', async () => {
        toggleAnalysisChallenge();
        const { container } = render(AnalysisPanel);
        await tick();
        await fireEvent.click(/** @type {HTMLElement} */ (container.querySelector('button.answer-mask')));
        const a = get(positionStore);
        positionStore.set(withDice(6, 5));
        await tick();
        positionStore.set({ ...a });
        await tick();
        expect(container.querySelector('button.answer-mask')).not.toBeNull();
    });

    test('revealing writes nothing to the database', async () => {
        toggleAnalysisChallenge();
        revealAnalysis();
        expect(get(analysisMaskStore)).toBeNull();
        expect(db.SaveAnalysis).not.toHaveBeenCalled();
        expect(db.SavePosition).not.toHaveBeenCalled();
    });

    test('an open training question keeps its inert mask, challenge or not', async () => {
        trainingSessionStore.set(/** @type {any} */ ({ exercise: 'decision', question: {}, revealed: false }));
        toggleAnalysisChallenge();
        revealAnalysis();
        const { container } = render(AnalysisPanel);
        await tick();
        expect(get(analysisMaskStore)).toBe('training');
        expect(container.querySelector('.answer-mask.inert')).not.toBeNull();
        expect(container.querySelector('button.answer-mask')).toBeNull();
    });
});
