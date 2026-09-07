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
import {
    TRAINING_EXERCISES,
    TIME_LIMITS,
    newSession,
    askQuestion,
    reveal,
    toggleFault,
    setAnswer,
    recordQuestion,
    failNextQuestion,
    finishedSession,
    summarizeExercise,
    canAskAnother
} from '../services/trainingTab.js';

/** Une question de Bearoff : deux EPC, tolérance un demi-pion. */
function bearoffQuestion(bottom = 87.4, top = 91.2) {
    return {
        key: 'pool:3',
        numbers: [
            { type: 'epc.bottom', value: bottom, tolerance: 0.5, precision: 1 },
            { type: 'epc.top', value: top, tolerance: 0.5, precision: 1 }
        ]
    };
}

/** Une session de Bearoff avec les deux champs remplis. */
function answered(bottom, top) {
    let s = askQuestion(newSession({ exercise: 'bearoff', seedSource: 'pool' }), bearoffQuestion(), 1000);
    s = setAnswer(s, 0, bottom);
    s = setAnswer(s, 1, top);
    return s;
}

/** @param {string[]} types */
function question(types) {
    return { key: '3:5', numbers: types.map((type, i) => ({ type, value: i })) };
}

describe('le catalogue servi à ce jour', () => {
    test('trois exercices, et le mode de réponse est une propriété de chacun', () => {
        expect(TRAINING_EXERCISES.map((e) => e.id)).toEqual(['scores', 'pips', 'bearoff']);
        expect(Object.fromEntries(TRAINING_EXERCISES.map((e) => [e.id, e.mode]))).toEqual({ scores: 'declared', pips: 'declared', bearoff: 'entered' });
    });

    test('Bearoff accepte les trois sources, et démarre sur le vivier', () => {
        const bearoff = TRAINING_EXERCISES.find((e) => e.id === 'bearoff');
        expect(bearoff.sources).toEqual(['pool', 'board', 'library']);
        expect(bearoff.defaultSource).toBe('pool');
    });

    // Pour Pions le plateau EST la question : « Suivante » reposerait la
    // même. Pour Bearoff le plateau est une GRAINE que le moteur joue sur un à
    // quatre plis, donc chaque question diffère.
    test('« Suivante » n’a de sens sur le plateau que là où le plateau est une graine', () => {
        expect(canAskAnother('pips', 'board')).toBe(false);
        expect(canAskAnother('pips', 'library')).toBe(true);
        expect(canAskAnother('bearoff', 'board')).toBe(true);
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

describe('quand aucune question ne peut être posée', () => {
    test('la session reste ouverte, nomme la raison, et garde ses nombres', () => {
        let s = newSession({ exercise: 'pips', seedSource: 'library' });
        s = recordQuestion(reveal(askQuestion(s, question(['pips.bottom', 'pips.top']), 0), 3000));
        s = failNextQuestion(s, 'noQuestion');
        expect(s.question).toBeNull();
        expect(s.questionError).toBe('noQuestion');
        expect(finishedSession(s).numbersAsked).toBe(2);
    });

    test('reposer une question efface la raison', () => {
        let s = failNextQuestion(newSession({ exercise: 'pips' }), 'noQuestion');
        s = askQuestion(s, question(['pips.bottom']), 0);
        expect(s.questionError).toBe('');
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

    // À dix pile, la fenêtre récente EST le tout : l'écart vaut structurellement
    // zéro, et l'afficher dirait « vous stagnez » là où il n'y a rien à comparer.
    test('à dix sessions exactement, il n’y a encore rien à comparer', () => {
        const ten = [];
        for (let i = 0; i < 10; i++) ten.push({ exercise: 'scores', numbersAsked: 10, faults: i, deviations: 0, meanDeviation: 0, medianMs: 1000 });
        expect(summarizeExercise(ten).sessions).toBe(10);
        expect(summarizeExercise(ten).trend).toBeNull();
    });

    test('aucune session : un bilan vide, pas une division par zéro', () => {
        const b = summarizeExercise([]);
        expect(b.sessions).toBe(0);
        expect(b.faultRate).toBeNull();
        expect(b.medianMs).toBeNull();
        expect(b.trend).toBeNull();
    });
});

describe('le geste saisi (ADR-0040 règle 2)', () => {
    test('dans la tolérance, c’est juste ; au-delà, c’est une faute', () => {
        // 87,4 contre 87,4 : exact. 91,8 contre 91,2 : six dixièmes, au-delà
        // du demi-pion.
        const s = reveal(answered('87.4', '91.8'), 4000);
        expect(s.faults).toEqual([false, true]);
    });

    test('la limite de la tolérance est dedans, pas dehors', () => {
        expect(reveal(answered('87.9', '91.2'), 4000).faults).toEqual([false, false]);
        expect(reveal(answered('87.91', '91.2'), 4000).faults).toEqual([true, false]);
    });

    // Le SENS de l'erreur est ce qu'on apprend : surestimer n'est pas
    // sous-estimer, et l'écart est gardé même quand la réponse est bonne.
    test('l’écart est signé, et gardé même sur une bonne réponse', () => {
        const s = recordQuestion(reveal(answered('90.4', '88.2'), 4000));
        expect(s.items[0]).toEqual({ numberType: 'epc.bottom', wrong: true, hasDeviation: true, deviation: 3 });
        expect(s.items[1]).toEqual({ numberType: 'epc.top', wrong: true, hasDeviation: true, deviation: -3 });

        const right = recordQuestion(reveal(answered('87.2', '91.2'), 4000));
        expect(right.items[0].wrong).toBe(false);
        expect(right.items[0].hasDeviation).toBe(true);
        expect(right.items[0].deviation).toBeCloseTo(-0.2, 6);
    });

    test('la virgule décimale se tape aussi à la française', () => {
        expect(reveal(answered('87,4', '91,2'), 4000).faults).toEqual([false, false]);
    });

    // On ne mesure pas une réponse qui n'a pas été donnée : la compter zéro
    // tirerait la moyenne des écarts vers une justesse qui n'a pas eu lieu.
    test('un champ vide est une faute SANS écart', () => {
        const s = recordQuestion(reveal(answered('', '91.2'), 4000));
        expect(s.items[0]).toEqual({ numberType: 'epc.bottom', wrong: true, hasDeviation: false, deviation: 0 });
        expect(s.items[1].hasDeviation).toBe(true);
    });

    test('hors délai : tout est faux et rien n’entre dans la moyenne', () => {
        const s = recordQuestion(reveal(answered('87.4', '91.2'), 4000, { outOfTime: true }));
        expect(s.items.map((i) => i.wrong)).toEqual([true, true]);
        expect(s.items.map((i) => i.hasDeviation)).toEqual([false, false]);
        expect(s.times, 'une question hors délai ne compte pas dans le temps médian').toEqual([]);
    });

    // C'est l'application qui juge, contre la tolérance : se cocher juste
    // ferait mentir la faute par rapport à l'écart enregistré.
    test('on ne se dédit pas d’un jugement : cocher est sans effet', () => {
        let s = reveal(answered('90.4', '91.2'), 4000);
        expect(s.faults).toEqual([true, false]);
        s = toggleFault(s, 0);
        expect(s.faults).toEqual([true, false]);
    });

    test('rien ne se tape après « Valider » : la question est close', () => {
        const s = setAnswer(reveal(answered('87.4', '91.2'), 4000), 0, '999');
        expect(s.answers[0]).toBe('87.4');
    });
});

describe('le journal d’un exercice saisi', () => {
    test('la ligne de session porte le nombre d’écarts et leur moyenne ABSOLUE', () => {
        let s = recordQuestion(reveal(answered('90.4', '88.2'), 4000));
        const row = finishedSession(s);
        expect(row.exercise).toBe('bearoff');
        expect(row.deviations).toBe(2);
        // Trois pions au-dessus et trois en dessous : la moyenne des écarts
        // SIGNÉS vaudrait zéro et dirait « sans erreur ». Le journal garde la
        // moyenne des valeurs absolues ; les signes vivent dans les items.
        expect(row.meanDeviation).toBe(3);
        expect(row.items.map((i) => i.deviation)).toEqual([3, -3]);
    });
});
