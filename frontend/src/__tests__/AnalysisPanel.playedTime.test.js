import { must } from './helpers/must.js';
import { describe, test, expect, vi, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({ LoadAnalysis: vi.fn(() => Promise.resolve(null)) }));

import { matchContextStore } from '../stores/positionStore.js';
import AnalysisPanel from '../components/AnalysisPanel.svelte';

const ctx = (movePositions) => ({ isMatchMode: true, matchID: 1, movePositions, currentIndex: 0, player1Name: 'A', player2Name: 'B' });

describe('AnalysisPanel — the time of the played decision', () => {
    afterEach(() => {
        cleanup();
        matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
    });

    test('shown discreetly in match review when recorded', async () => {
        matchContextStore.set(ctx([{ move_type: 'checker', decision_ms: 5200, cube_decision_ms: 1000, position: {} }]));
        const { container } = render(AnalysisPanel);
        await tick();
        expect(must(container.querySelector('[data-testid="played-time"]')).textContent).toContain('◇ 1.0 s · 5.2 s');
    });

    test('absent, not zero, when unknown', async () => {
        matchContextStore.set(ctx([{ move_type: 'checker', position: {} }]));
        const { container } = render(AnalysisPanel);
        await tick();
        expect(container.querySelector('[data-testid="played-time"]')).toBeNull();
    });
});
