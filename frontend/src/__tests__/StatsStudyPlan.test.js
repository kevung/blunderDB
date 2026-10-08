import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';

vi.mock('../services/studyQueueService.js', () => ({ startStudyPlanQueue: vi.fn() }));
vi.mock('../services/recurringStudy.js', () => ({ quizOnPlan: vi.fn(), deckFromIds: vi.fn() }));

import StatsStudyPlan from '../components/stats/StatsStudyPlan.svelte';
import { studyPlanStore } from '../stores/statsStore.js';
import { startStudyPlanQueue } from '../services/studyQueueService.js';
import { quizOnPlan, deckFromIds } from '../services/recurringStudy.js';

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    studyPlanStore.set(null);
});

const family = (theme, recoverable, low, ids) => ({
    GameType: 'holding',
    Kind: 'checker',
    Theme: theme,
    Errors: 6,
    Avoidable: 5,
    Recoverable: recoverable,
    Low: low,
    High: 2 * recoverable - low,
    Positions: ids.map((id) => ({ PositionID: id }))
});

const plan = {
    NumDecisions: 400,
    ThresholdMP: 50,
    MinErrors: 5,
    Families: [family('blots', 0.1117, 0.0074, [412, 484]), family('gammon', 0.08, 0.001, [9])],
    Tentative: [{ ...family('point', 0.02, -0.01, [5]), Errors: 2 }],
    Unthemed: 41,
    Unpriced: 3
};

describe('StatsStudyPlan', () => {
    test('ranks the families as the backend did, with recoverable MWC and its interval', () => {
        studyPlanStore.set(plan);
        render(StatsStudyPlan);
        const rows = screen.getAllByTestId('plan-family');
        expect(rows).toHaveLength(2);
        expect(rows[0].textContent).toContain('11.17 %');
        expect(rows[0].textContent).toContain('[0.74 %, 21.60 %]');
    });

    test('a family short of evidence is named apart, never ranked', () => {
        studyPlanStore.set(plan);
        render(StatsStudyPlan);
        expect(screen.getAllByTestId('plan-family')).toHaveLength(2);
        expect(screen.getByTestId('study-plan').textContent).toMatch(/\(2\)/);
    });

    test('each family feeds the study queue, a quiz and a deck', async () => {
        studyPlanStore.set(plan);
        render(StatsStudyPlan);
        const [study, quiz, deck] = screen.getAllByTestId('plan-family')[1].querySelectorAll('.study-btn');
        await fireEvent.click(study);
        expect(startStudyPlanQueue).toHaveBeenCalledWith(expect.anything(), 2);
        await fireEvent.click(quiz);
        expect(quizOnPlan).toHaveBeenCalledWith(2);
        await fireEvent.click(deck);
        expect(deckFromIds).toHaveBeenCalledWith(expect.stringContaining('gammon'), [9]);
        await fireEvent.click(screen.getByTestId('plan-quiz-first'));
        expect(quizOnPlan).toHaveBeenCalledWith(0);
    });

    test('without enough evidence it says so instead of ranking noise', () => {
        studyPlanStore.set({ ...plan, Families: [] });
        render(StatsStudyPlan);
        expect(screen.queryAllByTestId('plan-family')).toHaveLength(0);
        expect(screen.queryByTestId('plan-quiz-first')).toBeNull();
    });
});
