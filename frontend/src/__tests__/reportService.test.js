/**
 * reportService.test.js — le rapport HTML (#279, fiche I.23).
 *
 * Ce qui compte dans un document destiné à circuler : qu'il soit AUTONOME (une
 * seule page, aucune ressource externe), qu'il dise son périmètre, et qu'il
 * échappe ce qui vient d'un fichier importé — les noms de joueurs et les coups
 * arrivent de l'extérieur.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';

let stats = {};
let positions = [];
let analysis = null;

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ComputeStats: () => Promise.resolve(stats),
    LoadPositionsByIDs: () => Promise.resolve(positions),
    LoadAnalysis: () => Promise.resolve(analysis)
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
    stats = {
        Totals: { NumPositions: 412, NumMatches: 5, NumDecisions: 380 },
        PRGlobal: 4.71,
        PRChecker: 4.2,
        PRCube: 8.3,
        TopBlunders: []
    };
    positions = [];
    analysis = null;
});

describe('le rapport HTML', () => {
    test('est un document complet et autonome', async () => {
        const html = await buildReportHTML();
        expect(html.startsWith('<!doctype html>')).toBe(true);
        expect(html).toContain('</html>');
        // Aucune ressource externe : ni script, ni feuille de style distante,
        // ni image liée. C'est ce qui permet de l'envoyer par courriel.
        expect(html).not.toContain('<script');
        expect(html).not.toContain('<link');
        expect(html).not.toMatch(/<img[^>]+src="http/);
    });

    test('porte les indicateurs du périmètre courant', async () => {
        const html = await buildReportHTML();
        expect(html).toContain('412');
        expect(html).toContain('4.71');
    });

    test('dit quand il n’a aucune décision fautive à montrer', async () => {
        const html = await buildReportHTML();
        expect(html).toContain('No faulty decision in this scope.');
    });

    test('intègre un diagramme par décision, en ligne', async () => {
        stats.TopBlunders = [{ PositionID: 7, ErrorMP: 310, DecisionType: 0, PlayerNames: 'Alice vs Bob', MatchDate: '2026-03-01' }];
        positions = [samplePosition(7)];
        analysis = { checkerAnalysis: { moves: [{ move: '13/7 8/7' }] } };

        const html = await buildReportHTML();
        expect(html).toContain('<svg');
        expect(html).toContain('0.310');
        expect(html).toContain('13/7 8/7');
    });

    // Les noms viennent d'un fichier importé, donc de l'extérieur.
    test('échappe ce qui vient des données', async () => {
        stats.TopBlunders = [{ PositionID: 7, ErrorMP: 100, DecisionType: 1, PlayerNames: '<script>alert(1)</script>', MatchDate: '' }];
        positions = [samplePosition(7)];

        const html = await buildReportHTML();
        expect(html).not.toContain('<script>alert(1)</script>');
        expect(html).toContain('&lt;script&gt;');
    });
});
