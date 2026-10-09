/**
 * reportService.test.js — le rapport HTML.
 *
 * Le document lui-même (autonome, échappé, périmètre) est construit par le
 * moteur et testé en Go (pkg/blunderdb/report). Ici : ce que l'écran lui
 * fournit — le filtre courant, la langue, un diagramme par décision.
 */

import { must } from './helpers/must.js';
import { describe, test, expect, vi, beforeEach } from 'vitest';

let stats = {};
let positions = [];
const StatsReportHTML = vi.fn(() => Promise.resolve('<!doctype html><html></html>'));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ComputeStats: () => Promise.resolve(stats),
    LoadPositionsByIDs: () => Promise.resolve(positions),
    StatsReportHTML: (/** @type {any[]} */ ...args) => StatsReportHTML(...args)
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    SaveBoardImageDialog: vi.fn(() => Promise.resolve('')),
    SaveBoardSVG: vi.fn(() => Promise.resolve(undefined))
}));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));

import { buildReportHTML } from '../services/reportService.js';

function samplePosition(id) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    points[6] = { checkers: 5, color: 0 };
    points[19] = { checkers: 5, color: 1 };
    return {
        id,
        board: { points, bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [7, 7],
        player_on_roll: 0,
        decision_type: 0
    };
}

beforeEach(() => {
    stats = { TopBlunders: [] };
    positions = [];
    StatsReportHTML.mockClear();
});

describe('le rapport HTML', () => {
    test('rend le document du moteur', async () => {
        const html = await buildReportHTML();
        expect(html.startsWith('<!doctype html>')).toBe(true);
        expect(StatsReportHTML).toHaveBeenCalledTimes(1);
    });

    test('donne au moteur le filtre courant, la langue et un diagramme par décision', async () => {
        stats.TopBlunders = [{ PositionID: 7, ErrorMP: 310, DecisionType: 0 }];
        positions = [samplePosition(7)];

        await buildReportHTML();
        const [filter, lang, diagrams] = StatsReportHTML.mock.calls[0];
        expect(filter).toBeDefined();
        expect(typeof lang).toBe('string');
        expect(Object.keys(diagrams)).toEqual(['7']);
        expect(must(diagrams)[7]).toContain('<svg');
    });
});
