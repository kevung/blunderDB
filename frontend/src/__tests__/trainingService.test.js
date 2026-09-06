/**
 * trainingService.test.js — les micro-entraînements (#273, fiche I.17).
 *
 * Ce qui vaut d'être tenu ici n'est pas l'interface mais le jugement : une
 * tolérance qui dérive change la note sans que personne ne s'en aperçoive, et
 * un résumé de session qui prend la moyenne des temps ment dès qu'une question
 * a duré dix minutes.
 */

import { describe, test, expect, vi } from 'vitest';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadMetadata: vi.fn(() => Promise.resolve({})),
    SaveMetadata: vi.fn(() => Promise.resolve(undefined))
}));

import { grade, summarize, takePointTruth, pipTruth, TOLERANCE } from '../services/trainingService.js';

describe('la note', () => {
    // Un compte de pions est une addition : « à un pion près » n'existe pas à
    // la table, et une tolérance apprendrait à peu près à compter.
    test('le comptage de pions ne tolère rien', () => {
        expect(TOLERANCE.pips).toBe(0);
        expect(grade('pips', 167, 167).correct).toBe(true);
        expect(grade('pips', 168, 167).correct).toBe(false);
    });

    test("l'EPC tolère le demi-pion, le point de prise deux points", () => {
        expect(grade('epc', 87.4, 87.0).correct).toBe(true);
        expect(grade('epc', 87.6, 87.0).correct).toBe(false);
        expect(grade('takepoint', 24, 22).correct).toBe(true);
        expect(grade('takepoint', 25, 22).correct).toBe(false);
    });

    // Le SENS de l'erreur est ce qu'on apprend : deux pions de trop n'est pas
    // la même faute que deux de moins.
    test("l'erreur est signée", () => {
        expect(grade('pips', 170, 167).error).toBe(3);
        expect(grade('pips', 164, 167).error).toBe(-3);
    });

    test('une réponse vide est fausse, pas une exception', () => {
        expect(grade('pips', NaN, 167)).toEqual({ correct: false, error: null });
    });
});

describe('le résumé de session', () => {
    const answers = [
        { correct: true, error: 0, ms: 4000 },
        { correct: false, error: -3, ms: 9000 },
        { correct: true, error: 0, ms: 5000 },
        { correct: true, error: 0, ms: 600000 }
    ];

    // La médiane et non la moyenne : une question où l'on est allé chercher un
    // café ne dit rien du rythme, et c'est le rythme qu'on mesure.
    test('le temps est médian, donc insensible à la question abandonnée', () => {
        expect(summarize(answers).medianMs).toBe(7000);
    });

    test("compte les bonnes réponses et l'erreur absolue moyenne", () => {
        const s = summarize(answers);
        expect(s.count).toBe(4);
        expect(s.correct).toBe(3);
        expect(s.rate).toBe(0.75);
        expect(s.meanError).toBeCloseTo(0.75, 6);
    });

    test('une session vide ne divise pas par zéro', () => {
        expect(summarize([])).toEqual({ count: 0, correct: 0, rate: 0, meanError: 0, medianMs: 0 });
    });
});

describe('les vérités attendues', () => {
    // La table commence à 2-away : demander 1-away doit rendre « pas de
    // réponse » plutôt qu'une case voisine prise au hasard.
    test('le point de prise refuse un score hors table', () => {
        expect(takePointTruth(1, 4)).toBeNull();
        expect(takePointTruth(4, 1)).toBeNull();
        expect(takePointTruth(2, 2)).toBe(32.5);
    });

    test('le compte de pions est celui du joueur au trait', () => {
        const position = {
            player_on_roll: 0,
            board: {
                points: Array.from({ length: 26 }, (_, i) => (i === 6 ? { checkers: 2, color: 0 } : { checkers: 0, color: -1 })),
                bearoff: [0, 0]
            }
        };
        const white = pipTruth(position);
        position.player_on_roll = 1;
        expect(pipTruth(position)).not.toBe(white);
    });
});

describe('le PR de session (#294)', () => {
    // Le point du module : le nombre affiché après une session doit être sur la
    // MÊME échelle que le PR que les statistiques calculent pour le jeu réel,
    // sans quoi les comparer — ce que l'utilisateur fera — ne compare rien.
    // 500 × erreur moyenne en équité normalisée, la formule de storage.pr.
    test('vaut 500 × erreur moyenne, comme celui du jeu réel', async () => {
        const { quizPR } = await import('../services/trainingService.js');
        expect(quizPR(80, 1)).toBeCloseTo(40, 9);
        expect(quizPR(160, 4)).toBeCloseTo(20, 9);
    });

    test('sans décision, vaut 0 — à lire avec le compte, pas comme un sans-faute', async () => {
        const { quizPR } = await import('../services/trainingService.js');
        expect(quizPR(0, 0)).toBe(0);
    });
});
