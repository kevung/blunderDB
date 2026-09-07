/**
 * Les règles d'une session de l'onglet Entraînement (ADR-0040).
 *
 * Ce module ne connaît ni Svelte, ni le plateau, ni Wails : il prend un état
 * de session et rend le suivant. Le service qui l'entoure
 * (trainingTabService.js) fabrique les questions, tient le chronomètre et
 * écrit au journal ; ici vivent les seules décisions qui définissent
 * l'exercice — ce qu'une révélation produit, ce qu'une échéance produit à sa
 * place, et ce qu'une session finie laisse derrière elle.
 *
 * Le mode DÉCLARÉ, en une phrase : « Révéler » arrête le chrono et montre la
 * vérité ; chaque nombre est juste par défaut, et l'on clique celui qu'on a
 * raté. Ce qui est compté ou récité est juste ou faux, et le taper n'apprend
 * rien de plus (règle 2).
 */

/**
 * Les exercices de cette tranche. Bearoff, Évaluation et Décision viennent
 * ensuite : ils demandent le générateur de positions de l'ADR-0041 et le mode
 * SAISI, dont l'écart signé est tout l'intérêt.
 *
 * `sources` est la liste des sources de graine que l'exercice accepte, dans
 * l'ordre où le lanceur les propose ; une liste vide veut dire que la question
 * ne vient d'aucune position — c'est le cas de Scores, dont le vivier est les
 * 36 scores non ordonnés.
 */
export const TRAINING_EXERCISES = Object.freeze([
    Object.freeze({ id: 'scores', mode: 'declared', sources: [], defaultSource: 'pool' }),
    Object.freeze({ id: 'pips', mode: 'declared', sources: ['board', 'library'], defaultSource: 'board' })
]);

/** Les limites par question : aucune par défaut, sinon 15, 30 ou 60 secondes. */
export const TIME_LIMITS = Object.freeze([0, 15, 30, 60]);

/** Le nombre de sessions que la tendance regarde. */
export const TREND_WINDOW = 10;

/**
 * @typedef {object} TrainingNumber
 * @property {string} type le type de nombre — c'est LUI que le journal compte
 * @property {number} value la vérité
 *
 * @typedef {object} TrainingQuestion
 * @property {string} key de quoi la question est faite (un score, un id)
 * @property {TrainingNumber[]} numbers
 *
 * @typedef {object} TrainingSessionState
 * @property {string} exercise
 * @property {string} seedSource
 * @property {number} limitSeconds
 * @property {TrainingQuestion|null} question
 * @property {boolean} revealed
 * @property {boolean} outOfTime
 * @property {boolean[]} faults
 * @property {string} questionError la raison pour laquelle aucune question n'est posée, ou ''
 * @property {number} startedAt
 * @property {number} elapsedMs
 * @property {number} askedQuestions
 * @property {number[]} times les temps des questions ENREGISTRÉES, hors délai exclu
 * @property {{numberType: string, wrong: boolean, hasDeviation: boolean, deviation: number}[]} items
 */

/**
 * Une session vide. Elle n'a pas de longueur : un vivier de scores est infini,
 * et c'est « Terminer » qui décide de la fin (règle 5).
 * @param {{exercise: string, seedSource?: string, limitSeconds?: number}} opts
 * @returns {TrainingSessionState}
 */
export function newSession({ exercise, seedSource = '', limitSeconds = 0 }) {
    return {
        exercise,
        seedSource,
        limitSeconds,
        question: null,
        revealed: false,
        outOfTime: false,
        faults: [],
        questionError: '',
        startedAt: 0,
        elapsedMs: 0,
        askedQuestions: 0,
        times: [],
        items: []
    };
}

/**
 * Pose une question : le chrono part à l'affichage, jamais avant (la
 * fabrication de la question se cache derrière le temps de réflexion de la
 * précédente, elle ne se facture pas à celle-ci).
 * @param {TrainingSessionState} session @param {TrainingQuestion} question @param {number} now
 */
export function askQuestion(session, question, now) {
    return {
        ...session,
        question,
        revealed: false,
        outOfTime: false,
        faults: question.numbers.map(() => false),
        questionError: '',
        startedAt: now,
        elapsedMs: 0
    };
}

/**
 * Aucune question n'a pu être posée — la position tirée a disparu, la source
 * s'est vidée. La session RESTE OUVERTE et le dit : ce qui a déjà été répondu
 * est encore là, « Terminer » l'enregistre, et l'on peut réessayer. Une
 * session ne se perd jamais sans que l'utilisateur l'ait décidé, et un écran
 * qui rebascule tout seul sur le lanceur est une perte, pas une information.
 * @param {TrainingSessionState} session @param {string} reason
 */
export function failNextQuestion(session, reason) {
    return { ...session, question: null, revealed: false, outOfTime: false, faults: [], questionError: reason };
}

/**
 * Arrête le chrono et montre la vérité. Hors délai, la question compte tous
 * ses nombres faux : on ne mesure pas une réponse qui n'a pas été donnée, et
 * la cocher ensuite n'aurait pas de sens — c'est un verdict, pas une
 * déclaration.
 * @param {TrainingSessionState} session @param {number} now
 * @param {{outOfTime?: boolean}} [opts]
 */
export function reveal(session, now, { outOfTime = false } = {}) {
    if (!session.question || session.revealed) return session;
    return {
        ...session,
        revealed: true,
        outOfTime,
        elapsedMs: Math.max(0, now - session.startedAt),
        faults: session.question.numbers.map(() => outOfTime)
    };
}

/**
 * Coche — ou décoche — le nombre qu'on a raté. Sans effet tant que la question
 * n'est pas révélée (elle est encore ouverte) et sur une question hors délai
 * (son verdict est déjà rendu).
 * @param {TrainingSessionState} session @param {number} index
 */
export function toggleFault(session, index) {
    if (!session.revealed || session.outOfTime) return session;
    if (index < 0 || index >= session.faults.length) return session;
    const faults = session.faults.slice();
    faults[index] = !faults[index];
    return { ...session, faults };
}

/**
 * Enregistre la question révélée dans la session et la retire de l'écran.
 * Rien n'est écrit en base ici : la session entière l'est à « Terminer », et
 * « Quitter » la jette.
 * @param {TrainingSessionState} session
 */
export function recordQuestion(session) {
    if (!session.question || !session.revealed) return session;
    const items = session.question.numbers.map((number, i) => ({
        numberType: number.type,
        wrong: session.faults[i] === true,
        // Le mode déclaré ne produit aucun écart : un compte de pions ou une
        // case de table est juste ou faux. Les colonnes existent pour le mode
        // SAISI des tranches suivantes, et rester à `false` ici est ce qui
        // garde la moyenne des écarts vide plutôt que nulle.
        hasDeviation: false,
        deviation: 0
    }));
    return {
        ...session,
        question: null,
        revealed: false,
        outOfTime: false,
        faults: [],
        questionError: '',
        askedQuestions: session.askedQuestions + 1,
        times: session.outOfTime ? session.times : [...session.times, session.elapsedMs],
        items: [...session.items, ...items]
    };
}

/**
 * La ligne de journal d'une session terminée, avec ses nombres. Une question
 * révélée mais pas enregistrée n'y figure pas : elle n'a pas été validée.
 * @param {TrainingSessionState} session
 */
export function finishedSession(session) {
    return {
        exercise: session.exercise,
        seedSource: session.seedSource,
        numbersAsked: session.items.length,
        faults: session.items.filter((item) => item.wrong).length,
        deviations: session.items.filter((item) => item.hasDeviation).length,
        meanDeviation: meanOf(session.items.filter((item) => item.hasDeviation).map((item) => Math.abs(item.deviation))),
        medianMs: Math.round(medianOf(session.times) ?? 0),
        pr: 0,
        items: session.items
    };
}

/**
 * Le bilan d'un exercice, lu sur ses lignes de journal les plus récentes
 * d'abord. Des nombres, pas des phrases : le taux de fautes, l'écart moyen
 * quand il en existe un, le temps médian, et la tendance.
 *
 * Le temps médian est la MÉDIANE DES MÉDIANES de session — une question où
 * l'on est allé chercher un café ne dit rien du rythme, et une session entière
 * non plus. La tendance est l'écart entre le taux de fautes des dix dernières
 * sessions et celui de toutes : négatif, on s'améliore. Elle ne s'annonce pas
 * AU-DELÀ de dix sessions : à dix pile, la fenêtre récente EST le tout, l'écart
 * vaut structurellement zéro, et afficher ce zéro dirait « vous stagnez » là où
 * il n'y a rien à comparer.
 *
 * @param {{numbersAsked: number, faults: number, deviations: number, meanDeviation: number, medianMs: number}[]} sessions
 */
export function summarizeExercise(sessions) {
    const rows = Array.isArray(sessions) ? sessions : [];
    const numbersAsked = sum(rows.map((r) => r.numbersAsked || 0));
    const faults = sum(rows.map((r) => r.faults || 0));
    const deviations = sum(rows.map((r) => r.deviations || 0));
    const deviationTotal = sum(rows.map((r) => (r.deviations || 0) * (r.meanDeviation || 0)));
    const recent = rows.slice(0, TREND_WINDOW);
    const recentAsked = sum(recent.map((r) => r.numbersAsked || 0));
    const recentFaults = sum(recent.map((r) => r.faults || 0));
    const faultRate = numbersAsked > 0 ? faults / numbersAsked : null;
    const recentFaultRate = recentAsked > 0 ? recentFaults / recentAsked : null;
    return {
        sessions: rows.length,
        numbersAsked,
        faults,
        faultRate,
        deviations,
        meanDeviation: deviations > 0 ? deviationTotal / deviations : null,
        medianMs: medianOf(rows.map((r) => r.medianMs || 0)),
        recentFaultRate,
        trend: rows.length > TREND_WINDOW && faultRate !== null && recentFaultRate !== null ? recentFaultRate - faultRate : null
    };
}

/** @param {number[]} values */
function sum(values) {
    return values.reduce((total, v) => total + v, 0);
}

/** @param {number[]} values @returns {number|null} */
function meanOf(values) {
    return values.length === 0 ? 0 : sum(values) / values.length;
}

/** @param {number[]} values @returns {number|null} */
function medianOf(values) {
    if (values.length === 0) return null;
    const sorted = values.slice().sort((a, b) => a - b);
    const middle = Math.floor(sorted.length / 2);
    return sorted.length % 2 ? sorted[middle] : (sorted[middle - 1] + sorted[middle]) / 2;
}
