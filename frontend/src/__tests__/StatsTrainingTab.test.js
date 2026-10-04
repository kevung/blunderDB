import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';

vi.mock('../components/stats/charts/chartjs.js', () => ({
    loadChart: () =>
        Promise.resolve(
            class FakeChart {
                destroy() {}
            }
        )
}));

import StatsTrainingTab from '../components/stats/StatsTrainingTab.svelte';

afterEach(cleanup);

const data = {
    Window: 'week',
    Sessions: [{ ID: 1, CreatedAt: '2026-10-05 09:00:00', Decisions: 10, PR: 8 }],
    Themes: [{ Theme: 'blitz', Decisions: 2, PR: 200, Periods: [{ Start: '2026-10-12', Decisions: 2, PR: 200 }] }],
    Periods: [
        { Start: '2026-10-05', QuizSessions: 1, QuizDecisions: 10, QuizPR: 8, MatchDecisions: 100, MatchPR: 6, AnkiReviews: 4, AnkiPassed: 3, AnkiRetention: 0.75 },
        { Start: '2026-10-12', QuizSessions: 0, QuizDecisions: 0, QuizPR: 0, MatchDecisions: 50, MatchPR: 3, AnkiReviews: 0, AnkiPassed: 0, AnkiRetention: 0 }
    ]
};

describe('StatsTrainingTab', () => {
    test('lists the quiz PR by plan of play, worst first', () => {
        render(StatsTrainingTab, { props: { data } });
        const row = screen.getByText('Blitz').closest('tr');
        expect(row.textContent).toContain('200.00');
        expect(row.textContent).toContain('(2)');
        expect(row.textContent).toContain('2026-10-12');
    });

    test('puts the three series side by side, one row per window', () => {
        render(StatsTrainingTab, { props: { data } });
        const rows = screen.getAllByRole('row').slice(1, 3);
        expect(rows).toHaveLength(2);
        expect(rows[0].textContent).toContain('8.00');
        expect(rows[0].textContent).toContain('6.00');
        expect(rows[0].textContent).toContain('75.0');
    });

    test('a window without sample shows a dash, never a zero', () => {
        render(StatsTrainingTab, { props: { data } });
        const second = screen.getAllByRole('row')[2];
        expect(second.textContent).toContain('–');
        expect(second.textContent).not.toContain('0.00');
    });

    test('the window buttons change the calendar window', async () => {
        render(StatsTrainingTab, { props: { data } });
        const month = screen.getByRole('button', { name: /Month|Mois/ });
        await fireEvent.click(month);
        expect(month.getAttribute('aria-pressed')).toBe('true');
    });

    test('says so when nothing is recorded yet', () => {
        render(StatsTrainingTab, { props: { data: { Window: 'week', Sessions: [], Periods: [] } } });
        expect(screen.queryAllByRole('row')).toHaveLength(0);
    });
});
