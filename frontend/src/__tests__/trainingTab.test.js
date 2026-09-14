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
    canAskAnother,
    answerChosen,
    attachCorrection,
    questionOnBoard,
    exerciseForCommand,
    quizPR
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

/** Une session de Bearoff avec les deux champs remplis.
 *  @param {string} bottom @param {string} top */
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
    test('cinq exercices, et le mode de réponse est une propriété de chacun', () => {
        expect(TRAINING_EXERCISES.map((e) => e.id)).toEqual(['scores', 'pips', 'bearoff', 'evaluation', 'decision']);
        expect(Object.fromEntries(TRAINING_EXERCISES.map((e) => [e.id, e.mode]))).toEqual({ scores: 'declared', pips: 'declared', bearoff: 'entered', evaluation: 'entered', decision: 'chosen' });
    });

    test('Évaluation accepte les trois sources, démarre sur le vivier, et occupe le plateau', () => {
        const evaluation = TRAINING_EXERCISES.find((e) => e.id === 'evaluation');
        expect(evaluation?.sources).toEqual(['pool', 'board', 'library']);
        expect(evaluation?.defaultSource).toBe('pool');
        expect(evaluation?.surface).toBe('board');
        expect(canAskAnother('evaluation', 'board'), 'le plateau est une graine, pas la question').toBe(true);
    });

    // ADR-0040 règle 3 : une question de Décision demande une analyse, que
    // seule la bibliothèque porte.
    test('Décision ne connaît qu’une source : la bibliothèque', () => {
        const decision = /** @type {import('../services/trainingTab.js').TrainingExercise} */ (TRAINING_EXERCISES.find((e) => e.id === 'decision'));
        expect(decision.sources).toEqual(['library']);
        expect(decision.defaultSource).toBe('library');
    });

    // La surface (ADR-0040 règle 2) : ce qui, hors de l'onglet, montre la
    // question. C'est elle qui dit si le plateau appartient à la question.
    test('seule la fiche de score n’a pas le plateau pour surface', () => {
        expect(Object.fromEntries(TRAINING_EXERCISES.map((e) => [e.id, e.surface]))).toEqual({ scores: 'none', pips: 'board', bearoff: 'board', evaluation: 'board', decision: 'board' });
    });

    test('Bearoff accepte les trois sources, et démarre sur le vivier', () => {
        const bearoff = /** @type {import('../services/trainingTab.js').TrainingExercise} */ (TRAINING_EXERCISES.find((e) => e.id === 'bearoff'));
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

describe('les mots de la commande `train`', () => {
    // `train quiz` démarrait l'ancien exercice de la bande ; il démarre
    // désormais Décision, qui en est la suite dans l'onglet (#323). Les doigts
    // n'ont rien à réapprendre.
    test('`decision` et `quiz` démarrent le même exercice', () => {
        expect(exerciseForCommand('decision')).toBe('decision');
        expect(exerciseForCommand('quiz')).toBe('decision');
    });

    test('les alias des autres exercices sont intacts, et la casse ne compte pas', () => {
        expect(exerciseForCommand('tp')).toBe('scores');
        expect(exerciseForCommand('takepoint')).toBe('scores');
        expect(exerciseForCommand('pip')).toBe('pips');
        expect(exerciseForCommand('epc')).toBe('bearoff');
        expect(exerciseForCommand('  Quiz ')).toBe('decision');
    });

    test('un mot inconnu ne démarre rien', () => {
        expect(exerciseForCommand('chess')).toBeNull();
        expect(exerciseForCommand('')).toBeNull();
    });
});

/** Une question de Décision, telle que le service la fabrique.
 *  @param {'checker'|'cube'} prompt */
function decisionQuestion(prompt = 'checker', id = 7) {
    return { kind: 'decision', key: String(id), positionId: id, prompt, numbers: [{ type: prompt === 'cube' ? 'decision.cube' : 'decision.checker', value: 0 }] };
}

/** Un verdict du juge (engine.QuizVerdict). */
function verdict(extra = {}) {
    return { legal: true, matched: true, notation: '13/7 8/7', best: '13/7 8/7', errorMp: 0, ...extra };
}

describe('le geste choisi (Décision)', () => {
    test('« Révéler » ne s’applique pas : c’est le verdict du juge qui révèle', () => {
        const s = askQuestion(newSession({ exercise: 'decision', seedSource: 'library' }), decisionQuestion(), 0);
        expect(reveal(s, 3000).revealed).toBe(false);
    });

    test('le verdict arrête le chrono, et un coup exact n’est pas une faute', () => {
        let s = askQuestion(newSession({ exercise: 'decision', seedSource: 'library' }), decisionQuestion(), 1000);
        s = answerChosen(s, verdict(), 4000);
        expect(s.revealed).toBe(true);
        expect(s.elapsedMs).toBe(3000);
        expect(s.faults).toEqual([false]);
        expect(s.verdict?.errorMp).toBe(0);
    });

    // Les trois issues restent distinctes (#294) : aucune n'est « juste ».
    test('un coup coûteux, illégal ou non évalué est une faute', () => {
        const asked = askQuestion(newSession({ exercise: 'decision' }), decisionQuestion(), 0);
        expect(answerChosen(asked, verdict({ errorMp: 42, best: '24/18 13/11' }), 1).faults).toEqual([true]);
        expect(answerChosen(asked, verdict({ legal: false, matched: false }), 1).faults).toEqual([true]);
        expect(answerChosen(asked, verdict({ matched: false }), 1).faults).toEqual([true]);
    });

    test('un second verdict ne remplace pas le premier', () => {
        let s = askQuestion(newSession({ exercise: 'decision' }), decisionQuestion(), 0);
        s = answerChosen(s, verdict({ errorMp: 42 }), 1000);
        s = answerChosen(s, verdict(), 2000);
        expect(s.verdict?.errorMp).toBe(42);
        expect(s.elapsedMs).toBe(1000);
    });

    test('rien ne se coche : c’est l’application qui juge', () => {
        let s = askQuestion(newSession({ exercise: 'decision' }), decisionQuestion(), 0);
        s = toggleFault(answerChosen(s, verdict({ errorMp: 42 }), 1000), 0);
        expect(s.faults).toEqual([true]);
    });

    test('la question suivante efface le verdict', () => {
        let s = askQuestion(newSession({ exercise: 'decision' }), decisionQuestion(), 0);
        s = recordQuestion(answerChosen(s, verdict({ errorMp: 42 }), 1000));
        s = askQuestion(s, decisionQuestion('cube', 8), 2000);
        expect(s.verdict).toBeNull();
    });

    test('hors délai, la question est fausse, et la correction s’y attache sans rien coûter', () => {
        let s = askQuestion(newSession({ exercise: 'decision', limitSeconds: 15 }), decisionQuestion(), 0);
        s = reveal(s, 15000, { outOfTime: true });
        expect(s.revealed).toBe(true);
        expect(s.faults).toEqual([true]);
        s = attachCorrection(s, '7', verdict({ legal: false, matched: false, notation: '', best: '24/18 13/11' }));
        expect(s.verdict?.best).toBe('24/18 13/11');
        // La correction d'une AUTRE question arrive trop tard : elle est ignorée.
        const other = attachCorrection(s, '8', verdict({ best: 'autre' }));
        expect(other.verdict?.best).toBe('24/18 13/11');
    });

    test('une correction ne s’attache pas à une question répondue', () => {
        let s = askQuestion(newSession({ exercise: 'decision' }), decisionQuestion(), 0);
        s = answerChosen(s, verdict({ errorMp: 42, best: 'vrai' }), 1000);
        expect(attachCorrection(s, '7', verdict({ best: 'autre' })).verdict?.best).toBe('vrai');
    });
});

describe('une question mêlée : saisi et choisi (Évaluation, #322)', () => {
    /** Les chances de gain, tolérance cinq points, et l'action de videau choisie. */
    function evaluationQuestion(win = 62.4, answer = 'dt') {
        return {
            key: 'pool:4',
            numbers: [
                { type: 'eval.win', value: win, tolerance: 5, precision: 1 },
                { type: 'eval.cube', value: 0, mode: /** @type {'chosen'} */ ('chosen'), answer }
            ]
        };
    }

    /** @param {string} win @param {string} cube */
    function answeredEvaluation(win, cube) {
        let s = askQuestion(newSession({ exercise: 'evaluation', seedSource: 'pool' }), evaluationQuestion(), 1000);
        s = setAnswer(s, 0, win);
        s = setAnswer(s, 1, cube);
        return s;
    }

    test('« Valider » juge les deux en une fois : l’un contre la tolérance, l’autre exactement', () => {
        const s = reveal(answeredEvaluation('66', 'dt'), 4000);
        expect(s.revealed).toBe(true);
        expect(s.elapsedMs).toBe(3000);
        expect(s.faults).toEqual([false, false]);
        expect(s.deviations[0]).toBeCloseTo(3.6, 9);
        expect(s.deviations[1], 'une action de videau n’a pas d’écart').toBeNull();
    });

    test('le mauvais bouton est une faute, même quand les chances sont justes', () => {
        expect(reveal(answeredEvaluation('62', 'dp'), 2000).faults).toEqual([false, true]);
    });

    test('rien de choisi est une faute, sans écart', () => {
        const s = reveal(answeredEvaluation('62', ''), 2000);
        expect(s.faults).toEqual([false, true]);
        expect(s.deviations[1]).toBeNull();
    });

    test('rien ne se coche : c’est l’application qui juge les deux', () => {
        const s = reveal(answeredEvaluation('90', 'nd'), 2000);
        expect(toggleFault(s, 0).faults).toEqual(s.faults);
        expect(toggleFault(s, 1).faults).toEqual(s.faults);
    });

    test('hors délai, les deux nombres sont faux et aucun écart n’entre dans la moyenne', () => {
        const s = reveal(answeredEvaluation('62', 'dt'), 60000, { outOfTime: true });
        expect(s.faults).toEqual([true, true]);
        expect(s.deviations).toEqual([null, null]);
    });

    test('le journal compte les deux types à part, et l’écart seulement sur les chances', () => {
        const row = finishedSession(recordQuestion(reveal(answeredEvaluation('58', 'dt'), 2000)));
        expect(row.items).toEqual([
            { numberType: 'eval.win', wrong: false, hasDeviation: true, deviation: expect.closeTo(-4.4, 9) },
            { numberType: 'eval.cube', wrong: false, hasDeviation: false, deviation: 0 }
        ]);
        expect(row.deviations).toBe(1);
        expect(row.pr, 'le PR n’existe que pour Décision').toBe(0);
    });

    test('la question occupe le plateau, et `train evaluation` la démarre', () => {
        const s = askQuestion(newSession({ exercise: 'evaluation', seedSource: 'pool' }), evaluationQuestion(), 0);
        expect(questionOnBoard(s)).toBe(true);
        expect(exerciseForCommand('evaluation')).toBe('evaluation');
        expect(exerciseForCommand('Evaluation')).toBe('evaluation');
    });

    test('Bearoff, lui, ne connaît pas de nombre choisi : un champ reste un champ', () => {
        expect(bearoffQuestion().numbers.some((n) => 'mode' in n)).toBe(false);
    });
});

describe('le PR de session (#294)', () => {
    test('vaut 500 × erreur moyenne en équité normalisée, comme celui du jeu réel', () => {
        // 30 + 10 millipoints sur deux décisions : 0,020 d'erreur moyenne → PR 10.
        expect(quizPR(40, 2)).toBeCloseTo(10);
    });

    test('sans décision, vaut 0 — à lire avec le compte, pas comme un sans-faute', () => {
        expect(quizPR(0, 0)).toBe(0);
    });

    test('la session finie l’écrit au journal, sur les décisions jugées', () => {
        let s = newSession({ exercise: 'decision', seedSource: 'library', limitSeconds: 15 });
        s = recordQuestion(answerChosen(askQuestion(s, decisionQuestion('checker', 1), 0), verdict({ errorMp: 30 }), 1000));
        s = recordQuestion(answerChosen(askQuestion(s, decisionQuestion('cube', 2), 0), verdict({ errorMp: 10 }), 1000));
        // Un coup illégal ne coûte rien, mais c'est une décision jugée.
        s = recordQuestion(answerChosen(askQuestion(s, decisionQuestion('checker', 3), 0), verdict({ legal: false, matched: false, errorMp: 0 }), 1000));
        // Hors délai : aucune réponse, donc rien à mesurer — ni coût, ni décision.
        s = recordQuestion(reveal(askQuestion(s, decisionQuestion('checker', 4), 0), 15000, { outOfTime: true }));

        const row = finishedSession(s);
        expect(row.pr).toBeCloseTo(quizPR(40, 3));
        expect(row.numbersAsked).toBe(4);
        expect(row.faults).toBe(4);
        expect(row.deviations, 'un coût n’est pas un écart d’estimation').toBe(0);
        expect(row.items.map((i) => i.numberType)).toEqual(['decision.checker', 'decision.cube', 'decision.checker', 'decision.checker']);
    });

    test('les autres exercices n’ont pas de PR', () => {
        let s = newSession({ exercise: 'scores' });
        s = recordQuestion(reveal(askQuestion(s, question(['gv1']), 0), 1000));
        expect(finishedSession(s).pr).toBe(0);
    });

    test('le bilan dit le PR de la dernière session', () => {
        expect(
            summarizeExercise([
                { exercise: 'decision', numbersAsked: 5, faults: 2, deviations: 0, meanDeviation: 0, medianMs: 1000, pr: 6.5 },
                { exercise: 'decision', numbersAsked: 5, faults: 2, deviations: 0, meanDeviation: 0, medianMs: 1000, pr: 9 }
            ]).lastPr
        ).toBe(6.5);
        expect(summarizeExercise([]).lastPr).toBeNull();
    });
});

describe('le plateau appartient-il à la question ?', () => {
    // Le défaut hérité de #321 : après « Valider », J / K faisaient défiler la
    // liste sous une question ouverte. La réponse à cette question est ce que
    // le répartiteur lit.
    test('oui tant qu’une question à surface plateau est à l’écran, révélée ou non', () => {
        const asked = askQuestion(newSession({ exercise: 'bearoff', seedSource: 'pool' }), bearoffQuestion(), 0);
        expect(questionOnBoard(asked)).toBe(true);
        expect(questionOnBoard(reveal(asked, 1000))).toBe(true);
        expect(questionOnBoard(recordQuestion(reveal(asked, 1000)))).toBe(false);
    });

    test('non pour une fiche de score, ni hors session', () => {
        expect(questionOnBoard(askQuestion(newSession({ exercise: 'scores' }), question(['gv1']), 0))).toBe(false);
        expect(questionOnBoard(null)).toBe(false);
    });
});
