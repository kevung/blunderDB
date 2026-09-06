/**
 * La session d'entraînement, côté règles (ADR-0040 règles 2, 5 et 6).
 *
 * Quatre choses se décident ici et nulle part ailleurs : ce qu'une question
 * révélée produit comme fautes, ce qu'une question hors délai produit à la
 * place, ce qu'une session finie écrit au journal, et ce que le journal relu
 * donne comme bilan. Aucune ne demande de plateau, de Wails ni de navigateur —
 * c'est la couture, et c'est là qu'on la tient.
 */
import { describe, test, expect } from 'vitest';
import { TRAINING_EXERCISES, TIME_LIMITS, newSession, askQuestion, reveal, toggleFault, recordQuestion, finishedSession, summarizeExercise } from '../services/trainingTab.js';

/** @param {string[]} types */
function question(types) {
    return { key: '3:5', numbers: types.map((type, i) => ({ type, value: i })) };
}

describe('le catalogue de cette tranche', () => {
    test('deux exercices, tous deux déclarés', () => {
        expect(TRAINING_EXERCISES.map((e) => e.id)).toEqual(['scores', 'pips']);
        for (const e of TRAINING_EXERCISES) expect(e.mode).toBe('declared');
    });

    test('la limite par question : aucune par défaut, sinon 15 / 30 / 60 s', () => {
        expect(TIME_LIMITS).toEqual([0, 15, 30, 60]);
    });
});

describe('le geste déclaré', () => {
    test('chaque nombre est juste par défaut', () => {
        let s = askQuestion(newSession({ exercise: 'scores', seedSource: 'pool' }), question(['gv1', 'gv2']), 1000);
        s = reveal(s, 4000);
        expect(s.faults).toEqual([false, false]);
        s = recordQuestion(s);
        expect(s.items).toEqual([
            { numberType: 'gv1', wrong: false, hasDeviation: false, deviation: 0 },
            { numberType: 'gv2', wrong: false, hasDeviation: false, deviation: 0 }
        ]);
    });

    test('on clique le nombre qu’on a raté, et on peut se dédire', () => {
        let s = askQuestion(newSession({ exercise: 'scores' }), question(['gv1', 'gv2']), 0);
        s = reveal(s, 1000);
        s = toggleFault(s, 1);
        expect(s.faults).toEqual([false, true]);
        s = toggleFault(s, 1);
        expect(s.faults).toEqual([false, false]);
    });

    test('rien ne se coche avant « Révéler » : la question est encore ouverte', () => {
        let s = askQuestion(newSession({ exercise: 'scores' }), question(['gv1']), 0);
        s = toggleFault(s, 0);
        expect(s.faults).toEqual([false]);
    });

    test('le chrono s’arrête à « Révéler » et ne repart pas', () => {
        let s = askQuestion(newSession({ exercise: 'scores' }), question(['gv1']), 1000);
        s = reveal(s, 6500);
        expect(s.elapsedMs).toBe(5500);
        s = toggleFault(s, 0);
        expect(s.elapsedMs).toBe(5500);
    });
});

describe('hors délai', () => {
    test('tous les nombres sont faux, et le cochage ne s’applique plus', () => {
        let s = askQuestion(newSession({ exercise: 'scores', limitSeconds: 15 }), question(['gv1', 'gv2']), 0);
        s = reveal(s, 15000, { outOfTime: true });
        expect(s.outOfTime).toBe(true);
        expect(s.faults).toEqual([true, true]);
        s = toggleFault(s, 0);
        expect(s.faults, 'un verdict ne se déclare pas').toEqual([true, true]);
    });

    test('le temps d’une question hors délai n’entre pas dans le temps médian', () => {
        let s = newSession({ exercise: 'scores', limitSeconds: 15 });
        s = recordQuestion(reveal(askQuestion(s, question(['gv1']), 0), 4000));
        s = recordQuestion(reveal(askQuestion(s, question(['gv2']), 0), 15000, { outOfTime: true }));
        expect(finishedSession(s).medianMs).toBe(4000);
    });
});

describe('la session finie', () => {
    test('rend la ligne du journal et ses nombres', () => {
        let s = newSession({ exercise: 'scores', seedSource: 'pool' });
        s = askQuestion(s, question(['tp2.live', 'tp2.last']), 0);
        s = toggleFault(reveal(s, 3000), 0);
        s = recordQuestion(s);
        s = askQuestion(s, question(['tp2.live']), 0);
        s = recordQuestion(reveal(s, 5000));

        const row = finishedSession(s);
        expect(row.exercise).toBe('scores');
        expect(row.seedSource).toBe('pool');
        expect(row.numbersAsked).toBe(3);
        expect(row.faults).toBe(1);
        expect(row.medianMs).toBe(4000);
        expect(row.deviations).toBe(0);
        expect(row.items).toHaveLength(3);
        expect(row.items[0]).toEqual({ numberType: 'tp2.live', wrong: true, hasDeviation: false, deviation: 0 });
    });

    test('une question révélée mais pas enregistrée ne compte pas', () => {
        let s = newSession({ exercise: 'pips', seedSource: 'board' });
        s = reveal(askQuestion(s, question(['pips.bottom', 'pips.top']), 0), 2000);
        expect(finishedSession(s).numbersAsked).toBe(0);
    });
});

describe('le bilan', () => {
    const sessions = [
        { exercise: 'scores', numbersAsked: 10, faults: 5, deviations: 0, meanDeviation: 0, medianMs: 6000 },
        { exercise: 'scores', numbersAsked: 10, faults: 1, deviations: 0, meanDeviation: 0, medianMs: 4000 },
        { exercise: 'scores', numbersAsked: 10, faults: 0, deviations: 0, meanDeviation: 0, medianMs: 2000 }
    ];

    test('compte les sessions, le taux de fautes et le temps médian', () => {
        const b = summarizeExercise(sessions);
        expect(b.sessions).toBe(3);
        expect(b.numbersAsked).toBe(30);
        expect(b.faults).toBe(6);
        expect(b.faultRate).toBeCloseTo(0.2);
        expect(b.medianMs).toBe(4000);
    });

    test('sans écart, il n’y en a pas — et non pas zéro', () => {
        expect(summarizeExercise(sessions).meanDeviation).toBeNull();
    });

    test('l’écart moyen ne porte que sur les nombres qui en ont un', () => {
        const b = summarizeExercise([
            { exercise: 'bearoff', numbersAsked: 4, faults: 2, deviations: 2, meanDeviation: 3, medianMs: 1000 },
            { exercise: 'bearoff', numbersAsked: 2, faults: 1, deviations: 0, meanDeviation: 0, medianMs: 1000 }
        ]);
        expect(b.meanDeviation).toBe(3);
    });

    test('la tendance compare les dix dernières au tout, en points de taux', () => {
        // Les sessions arrivent les plus récentes d'abord (ORDER BY id DESC).
        const many = [];
        for (let i = 0; i < 10; i++) many.push({ exercise: 'scores', numbersAsked: 10, faults: 1, deviations: 0, meanDeviation: 0, medianMs: 1000 });
        for (let i = 0; i < 10; i++) many.push({ exercise: 'scores', numbersAsked: 10, faults: 5, deviations: 0, meanDeviation: 0, medianMs: 1000 });
        const b = summarizeExercise(many);
        expect(b.recentFaultRate).toBeCloseTo(0.1);
        expect(b.faultRate).toBeCloseTo(0.3);
        expect(b.trend).toBeCloseTo(-0.2);
    });

    test('moins de dix sessions : pas de tendance à annoncer', () => {
        expect(summarizeExercise(sessions).trend).toBeNull();
    });

    test('aucune session : un bilan vide, pas une division par zéro', () => {
        const b = summarizeExercise([]);
        expect(b.sessions).toBe(0);
        expect(b.faultRate).toBeNull();
        expect(b.medianMs).toBeNull();
        expect(b.trend).toBeNull();
    });
});
