/**
 * showPosition : les réponses de LoadAnalysis/LoadComment reviennent dans le
 * désordre quand une touche de navigation est maintenue ; seule la dernière
 * position demandée écrit les stores d'analyse et de commentaire.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const bindings = vi.hoisted(() => ({
    LoadAnalysis: vi.fn(),
    LoadComment: vi.fn(),
    ListPositionIDs: vi.fn(() => Promise.resolve([])),
    LoadPositionsByIDs: vi.fn(() => Promise.resolve([])),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', () => bindings);
vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));
vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));
vi.mock('../services/confirmService.js', async (importOriginal) => ({
    ...(await importOriginal()),
    confirmAction: vi.fn(() => Promise.resolve(true))
}));

import { showPosition, loadAnalysisForPosition } from '../services/positionService.js';
import { analysisStore } from '../stores/analysisStore.js';
import { positionStore } from '../stores/positionStore.js';
import { commentTextStore } from '../stores/uiStore.js';

const pos = (id) => ({ id, board: { points: [], bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice: [1, 2], score: [3, 3], player_on_roll: 0, decision_type: 0 });

// A reply the test resolves by hand, at the moment of its choosing.
function deferred() {
    let resolve;
    const promise = new Promise((r) => (resolve = r));
    return { promise, resolve };
}

beforeEach(() => {
    vi.clearAllMocks();
});

describe('showPosition', () => {
    test('une analyse périmée, arrivée après la suivante, est ignorée', async () => {
        const replies = { 1: { analysis: deferred(), comment: deferred() }, 2: { analysis: deferred(), comment: deferred() } };
        bindings.LoadAnalysis.mockImplementation((id) => replies[id].analysis.promise);
        bindings.LoadComment.mockImplementation((id) => replies[id].comment.promise);

        const first = showPosition(pos(1));
        const second = showPosition(pos(2));

        replies[2].analysis.resolve({ positionId: 2, xgid: 'N2' });
        replies[2].comment.resolve('note 2');
        await second;
        replies[1].analysis.resolve({ positionId: 1, xgid: 'N1' });
        replies[1].comment.resolve('note 1');
        await first;

        expect(get(positionStore).id).toBe(2);
        expect(get(analysisStore).positionId).toBe(2);
        expect(get(commentTextStore)).toBe('note 2');
    });

    test('une analyse périmée, arrivée avant la suivante, est ignorée aussi', async () => {
        const a1 = deferred();
        const a2 = deferred();
        bindings.LoadAnalysis.mockImplementation((id) => (id === 1 ? a1.promise : a2.promise));
        bindings.LoadComment.mockResolvedValue('');

        const first = showPosition(pos(1));
        const second = showPosition(pos(2));
        a1.resolve({ positionId: 1 });
        await first;
        expect(get(analysisStore).positionId).not.toBe(1);

        a2.resolve({ positionId: 2 });
        await second;
        expect(get(analysisStore).positionId).toBe(2);
    });

    test('loadAnalysisForPosition respecte le même jeton', async () => {
        const a1 = deferred();
        bindings.LoadAnalysis.mockImplementation((id) => (id === 1 ? a1.promise : Promise.resolve({ positionId: 2 })));
        bindings.LoadComment.mockResolvedValue('');

        const stale = loadAnalysisForPosition(pos(1));
        await showPosition(pos(2));
        a1.resolve({ positionId: 1 });
        await stale;

        expect(get(analysisStore).positionId).toBe(2);
    });
});
