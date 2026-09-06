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

import { grade, summarize, DRILLS, TOLERANCE } from '../services/trainingService.js';

describe('la note', () => {
    // Depuis #320, la bande ne sert plus que ce qui se SAISIT : le compte de
    // pions et le point de prise sont passés à l'onglet, en mode déclaré.
    test('la bande ne sert plus que l’EPC et le quiz', () => {
        expect(DRILLS).toEqual(['epc', 'quiz']);
    });

    test("l'EPC tolère le demi-pion", () => {
        expect(TOLERANCE.epc).toBe(0.5);
        expect(grade('epc', 87.4, 87.0).correct).toBe(true);
        expect(grade('epc', 87.6, 87.0).correct).toBe(false);
    });

    // Seul un nombre ESTIMÉ a une tolérance : un exercice qui n'en déclare pas
    // ne tolère rien, et c'est la règle, pas l'oubli.
    test('un exercice sans tolérance déclarée ne tolère rien', () => {
        expect(TOLERANCE.quiz).toBeUndefined();
        expect(grade('quiz', 167, 167).correct).toBe(true);
        expect(grade('quiz', 167.5, 167).correct).toBe(false);
    });

    // Le SENS de l'erreur est ce qu'on apprend : deux pions de trop n'est pas
    // la même faute que deux de moins.
    test("l'erreur est signée", () => {
        expect(grade('epc', 170, 167).error).toBe(3);
        expect(grade('epc', 164, 167).error).toBe(-3);
    });

    test('une réponse vide est fausse, pas une exception', () => {
        expect(grade('epc', NaN, 167)).toEqual({ correct: false, error: null });
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
