import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';

vi.mock('../services/positionLoader.js', () => ({
    loadPositionsFromSelection: vi.fn()
}));
vi.mock('../services/recurringStudy.js', () => ({
    quizOnIds: vi.fn(),
    quizOnWorstGroups: vi.fn(),
    deckFromIds: vi.fn(),
    collectionFromIds: vi.fn()
}));

import StatsRecurringErrors from '../components/stats/StatsRecurringErrors.svelte';
import { loadPositionsFromSelection } from '../services/positionLoader.js';
import { quizOnIds, quizOnWorstGroups, deckFromIds, collectionFromIds } from '../services/recurringStudy.js';
import { must } from './helpers/must.js';

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

const data = {
    NumDecisions: 10,
    ThresholdMP: 50,
    Groups: [
        { GameType: 'holding', Kind: 'checker', Theme: 'blots', Count: 3, SumErrorMP: 400, PRCost: 20, PositionIDs: [7, 3] },
        { GameType: 'race', Kind: 'cube', Theme: 'offer_missed', Count: 1, SumErrorMP: 120, PRCost: 6, PositionIDs: [9] }
    ]
};

describe('StatsRecurringErrors', () => {
    test('lists the groups in the order the backend ranked them, with the PR cost', () => {
        render(StatsRecurringErrors, { props: { data } });
        const rows = screen.getAllByRole('row').slice(1);
        expect(rows).toHaveLength(2);
        expect(rows[0].textContent).toContain('20.00');
        expect(rows[1].textContent).toContain('6.00');
    });

    test('a click on a group opens its positions', async () => {
        render(StatsRecurringErrors, { props: { data } });
        const links = document.querySelectorAll('.group-link');
        await fireEvent.click(links[1]);
        expect(loadPositionsFromSelection).toHaveBeenCalledWith([9]);
    });

    test('each row offers a quiz, a deck and a collection on its own positions', async () => {
        render(StatsRecurringErrors, { props: { data } });
        const row = screen.getAllByRole('row')[1];
        const [quiz, deck, collection] = row.querySelectorAll('.study-btn');
        await fireEvent.click(quiz);
        expect(quizOnIds).toHaveBeenCalledWith([7, 3]);
        await fireEvent.click(deck);
        expect(deckFromIds).toHaveBeenCalledWith(expect.stringContaining('blots'), [7, 3]);
        await fireEvent.click(collection);
        expect(collectionFromIds).toHaveBeenCalledWith(expect.stringContaining('blots'), [7, 3]);
    });

    test('the quiz of the worst groups heads the section', async () => {
        render(StatsRecurringErrors, { props: { data } });
        await fireEvent.click(screen.getByTestId('recurring-quiz-worst'));
        expect(quizOnWorstGroups).toHaveBeenCalledWith();
    });

    test('the unthemed errors stay out of the ranking, listed apart and clickable', async () => {
        const withRest = { ...data, Unthemed: [{ GameType: 'contact', Kind: '', Theme: 'none', Count: 9, SumErrorMP: 900, PRCost: 45, PositionIDs: [4, 5] }] };
        render(StatsRecurringErrors, { props: { data: withRest } });
        expect(screen.getAllByRole('row').slice(1)).toHaveLength(2);
        const item = screen.getByRole('listitem');
        expect(item.textContent).toContain('45.00');
        await fireEvent.click(must(item.querySelector('button')));
        expect(loadPositionsFromSelection).toHaveBeenCalledWith([4, 5]);
    });

    test('says so when the filter has no error', () => {
        render(StatsRecurringErrors, { props: { data: { NumDecisions: 4, Groups: [] } } });
        expect(screen.queryAllByRole('row')).toHaveLength(0);
    });
});
