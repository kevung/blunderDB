/**
 * goalService.test.js — les objectifs de progression (#274, fiche I.18).
 *
 * Deux calculs valent d'être tenus : la cible PROPOSÉE, qui doit tomber sur un
 * palier plutôt que sur un « un peu mieux » sans ancrage, et la tendance, qui
 * doit refuser de se prononcer sur deux points.
 */

import { describe, test, expect, vi } from 'vitest';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadMetadata: vi.fn(() => Promise.resolve({})),
    SaveMetadata: vi.fn(() => Promise.resolve(undefined))
}));

import { suggestTarget, trend } from '../services/goalService.js';

describe('la cible proposée', () => {
    // Proposer une BANDE en dit une : passer d'intermédiaire à avancé se voit
    // et se raconte, là où « 0,5 de mieux » ne s'ancre à rien.
    test("propose l'entrée dans la bande suivante", () => {
        expect(suggestTarget(7.4)).toBe(6); // intermédiaire → avancé
        expect(suggestTarget(5.0)).toBe(4); // avancé → expert
        expect(suggestTarget(3.1)).toBe(2); // expert → classe mondiale
        expect(suggestTarget(13)).toBe(12); // débutant → occasionnel
    });

    // Depuis la meilleure bande il n'y a plus de palier : un cran, et c'est
    // dit comme tel.
    test('depuis la meilleure bande, propose un cran', () => {
        expect(suggestTarget(1.5)).toBe(1);
    });

    test('ne propose rien sans niveau mesuré', () => {
        expect(suggestTarget(0)).toBeNull();
        expect(suggestTarget(NaN)).toBeNull();
    });
});

describe('la tendance', () => {
    // Deux points ne font pas une tendance, et le dire vaut mieux que tracer
    // une droite entre deux tournois.
    test('refuse de se prononcer sous trois points', () => {
        expect(trend([6, 5], 4)).toBeNull();
        expect(trend([], 4)).toBeNull();
    });

    test('voit une amélioration régulière', () => {
        const t = trend([9, 8, 7, 6], 2);
        expect(t.slope).toBeCloseTo(-1, 5);
        expect(t.projected).toBeCloseTo(4, 5);
    });

    test('ne projette jamais un PR négatif', () => {
        const t = trend([3, 2, 1], 20);
        expect(t.projected).toBe(0);
    });

    test('ignore les points sans mesure', () => {
        expect(trend([6, 0, 5, NaN, 4], 1).slope).toBeCloseTo(-1, 5);
    });
});
