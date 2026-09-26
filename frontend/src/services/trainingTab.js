/**
 * Les règles d'une session de l'onglet Entraînement (ADR-0040), sans Svelte,
 * plateau ni Wails : un état de session entre, le suivant sort.
 *
 * Mode DÉCLARÉ : « Révéler » arrête le chrono et montre la vérité ; chaque
 * nombre est juste par défaut et l'on clique celui qu'on a raté (règle 2).
 */

/**
 * Les cinq exercices de l'ADR-0040 règle 3.
 *
 * `sources` : les sources de graine acceptées, dans l'ordre du lanceur ; vide
 * pour Scores, dont le vivier est les 36 scores non ordonnés.
 *
 * `mode` (règle 2) appartient à l'exercice, jamais à la session : le COMPTÉ ou
 * RÉCITÉ se déclare, l'ESTIMÉ se saisit (la taille de l'erreur est la leçon).
 *
 * `boardIsTheQuestion` : pour Pions le plateau EST la question (« Suivante »
 * reposerait la même) ; pour Bearoff c'est une graine jouée sur un à quatre
 * plis (ADR-0041 règle 2).
 *
 * `surface` : ce qui montre la question hors de l'onglet. Une question
 * `board` à l'écran possède le plateau (`questionOnBoard`).
 *
 * Décision se répond en mode CHOISI (coup joué au plateau via `quizPlay.js`,
 * ou action de videau cliquée), jugé contre l'analyse enregistrée : sa seule
 * source est donc la bibliothèque.
 *
 * Évaluation mêle les deux dans une question : chances de gain saisies, action
 * de videau choisie. Son `mode` est `entered` (« Valider » juge en une fois) et
 * le nombre choisi le déclare lui-même (`TrainingNumber.mode`).
 */
/**
 * @typedef {object} TrainingExercise
 * @property {string} id
 * @property {'declared'|'entered'|'chosen'} mode
 * @property {string[]} sources
 * @property {string} defaultSource
 * @property {'board'|'none'} surface
 * @property {boolean} [boardIsTheQuestion]
 */

/** @type {ReadonlyArray<Readonly<TrainingExercise>>} */
export const TRAINING_EXERCISES = Object.freeze([
    Object.freeze({ id: 'scores', mode: 'declared', sources: [], defaultSource: 'pool', surface: 'none' }),
    Object.freeze({ id: 'pips', mode: 'declared', sources: ['board', 'library'], defaultSource: 'board', surface: 'board', boardIsTheQuestion: true }),
    Object.freeze({ id: 'bearoff', mode: 'entered', sources: ['pool', 'board', 'library'], defaultSource: 'pool', surface: 'board' }),
    Object.freeze({ id: 'evaluation', mode: 'entered', sources: ['pool', 'board', 'library'], defaultSource: 'pool', surface: 'board' }),
    Object.freeze({ id: 'decision', mode: 'chosen', sources: ['library'], defaultSource: 'library', surface: 'board' })
]);

/** @param {string} exercise */
function modeOf(exercise) {
    return TRAINING_EXERCISES.find((e) => e.id === exercise)?.mode ?? 'declared';
}

/** @param {string} exercise */
export function isEnteredExercise(exercise) {
    return modeOf(exercise) === 'entered';
}

/** @param {string} exercise */
export function isChosenExercise(exercise) {
    return modeOf(exercise) === 'chosen';
}

/**
 * Le plateau appartient-il à la question ? Oui tant qu'une question `board` est
 * à l'écran, même révélée : sa vérité affichée décrit ce plateau.
 * @param {TrainingSessionState|null|undefined} session
 */
export function questionOnBoard(session) {
    if (!session?.question) return false;
    return TRAINING_EXERCISES.find((e) => e.id === session.exercise)?.surface === 'board';
}

/**
 * Les mots que `train <exercice>` accepte, alias hérités compris (`tp`,
 * `takepoint`, `epc`, `quiz`), pour qu'une commande d'hier marche encore.
 */
const EXERCISE_WORDS = Object.freeze({
    scores: 'scores',
    tp: 'scores',
    takepoint: 'scores',
    pips: 'pips',
    pip: 'pips',
    bearoff: 'bearoff',
    epc: 'bearoff',
    evaluation: 'evaluation',
    decision: 'decision',
    quiz: 'decision'
});

/**
 * @param {string} word
 * @returns {string|null}
 */
export function exerciseForCommand(word) {
    const key = String(word ?? '')
        .trim()
        .toLowerCase();
    return Object.prototype.hasOwnProperty.call(EXERCISE_WORDS, key) ? /** @type {Record<string, string>} */ (EXERCISE_WORDS)[key] : null;
}

/**
 * Le PR d'une session de Décision, sur l'échelle des statistiques : 500 ×
 * erreur moyenne en équité normalisée (formule de `storage.pr`). Double
 * assumé d'`engine.QuizPR`, pour éviter un aller-retour.
 *
 * @param {number} sumErrorMp @param {number} decisions
 */
export function quizPR(sumErrorMp, decisions) {
    if (!decisions) return 0;
    return (500 * sumErrorMp) / 1000 / decisions;
}

/**
 * Y a-t-il une question suivante ? Non si le plateau est la question.
 * @param {string} exercise @param {string} seedSource
 */
export function canAskAnother(exercise, seedSource) {
    const declared = /** @type {{boardIsTheQuestion?: boolean}|undefined} */ (TRAINING_EXERCISES.find((e) => e.id === exercise));
    return !(declared?.boardIsTheQuestion && seedSource === 'board');
}

/** Les limites par question : aucune par défaut, sinon 15, 30 ou 60 secondes. */
export const TIME_LIMITS = Object.freeze([0, 15, 30, 60]);

/** Le nombre de sessions que la tendance regarde. */
export const TREND_WINDOW = 10;

/**
 * @typedef {object} TrainingNumber
 * @property {string} type le type de nombre — c'est LUI que le journal compte
 * @property {number} value la vérité
 * @property {number} [tolerance] en mode SAISI, l'écart toléré ; 0 sinon
 * @property {number} [precision] les décimales à l'affichage
 * @property {'chosen'} [mode] dans un exercice SAISI, un nombre qui se CHOISIT
 *   parmi quelques options et se juge exactement (l'action de videau
 *   d'Évaluation) ; absent, il se saisit
 * @property {string} [answer] pour un nombre choisi, l'option juste — rendue par
 *   le moteur, jamais recalculée ici
 *
 * @typedef {object} TrainingQuestion
 * @property {string} key de quoi la question est faite (un score, un id)
 * @property {TrainingNumber[]} numbers
 * @property {string} [kind] l'exercice qui l'a fabriquée — `scores`, `pips`, `bearoff`
 * @property {any} [card] la fiche de score, pour Scores
 * @property {number|null} [positionId] la position de la base dont elle est tirée, s'il y en a une
 * @property {any} [position] la position à montrer sur le plateau, s'il y en a une
 * @property {'checker'|'cube'} [prompt] pour Décision, la forme de la décision que la position porte
 * @property {any[]} [plays] pour une décision de pions, les coups légaux que le moteur a rendus
 * @property {string} [cubeVerdict] pour Évaluation, le verdict du moteur à quatre issues (`too_good` compris)
 * @property {string} [regime] pour Évaluation, d'où vient la vérité : `exact` ou `evaluated`
 * @property {string} [depth] pour Évaluation évaluée, la profondeur qui l'a produite
 * @property {any} [epc] pour Évaluation, l'EPC des deux camps quand la position en a un exact — montré, jamais demandé
 *
 * @typedef {object} QuizVerdict le jugement du moteur (`engine.QuizVerdict`)
 * @property {boolean} legal
 * @property {boolean} matched
 * @property {string} notation
 * @property {string} best
 * @property {number} errorMp
 *
 * @typedef {object} TrainingSessionState
 * @property {string} exercise
 * @property {string} seedSource
 * @property {number} limitSeconds
 * @property {TrainingQuestion|null} question
 * @property {boolean} revealed
 * @property {boolean} outOfTime
 * @property {boolean[]} faults
 * @property {string[]} answers en mode SAISI, ce qui a été tapé — ou l'option choisie — nombre par nombre
 * @property {(number|null)[]} deviations l'écart SIGNÉ de chaque nombre saisi, `null` quand il n'y en a pas
 * @property {string} questionError la raison pour laquelle aucune question n'est posée, ou ''
 * @property {number} startedAt
 * @property {number} elapsedMs
 * @property {number} askedQuestions
 * @property {number[]} times les temps des questions ENREGISTRÉES, hors délai exclu
 * @property {QuizVerdict|null} verdict en mode CHOISI, le jugement de la question à l'écran — ou, hors délai, sa seule correction
 * @property {number} decisions en mode CHOISI, les décisions jugées et enregistrées — le dénominateur du PR
 * @property {number} sumErrorMp leur coût cumulé, en millipoints d'équité normalisée
 * @property {{numberType: string, wrong: boolean, hasDeviation: boolean, deviation: number}[]} items
 */

/**
 * Une session vide, sans longueur : « Terminer » décide de la fin (règle 5).
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
        answers: [],
        deviations: [],
        questionError: '',
        startedAt: 0,
        elapsedMs: 0,
        askedQuestions: 0,
        times: [],
        items: [],
        verdict: null,
        decisions: 0,
        sumErrorMp: 0
    };
}

/**
 * Pose une question : le chrono part à l'affichage, jamais avant.
 * @param {TrainingSessionState} session @param {TrainingQuestion} question @param {number} now
 * @returns {TrainingSessionState}
 */
export function askQuestion(session, question, now) {
    return {
        ...session,
        question,
        revealed: false,
        outOfTime: false,
        faults: question.numbers.map(() => false),
        answers: question.numbers.map(() => ''),
        deviations: question.numbers.map(() => null),
        questionError: '',
        verdict: null,
        startedAt: now,
        elapsedMs: 0
    };
}

/**
 * Enregistre la saisie d'un champ sans la juger : juger en cours de frappe
 * annoncerait la réponse.
 * @param {TrainingSessionState} session @param {number} index @param {string} text
 * @returns {TrainingSessionState}
 */
export function setAnswer(session, index, text) {
    if (!session.question || session.revealed) return session;
    if (index < 0 || index >= session.answers.length) return session;
    const answers = session.answers.slice();
    answers[index] = text;
    return { ...session, answers };
}

/**
 * Aucune question n'a pu être posée. La session reste ouverte et le dit, pour
 * que « Terminer » garde ce qui a été répondu et qu'on puisse réessayer.
 * @param {TrainingSessionState} session @param {string} reason
 * @returns {TrainingSessionState}
 */
export function failNextQuestion(session, reason) {
    return { ...session, question: null, revealed: false, outOfTime: false, faults: [], answers: [], deviations: [], verdict: null, questionError: reason };
}

/**
 * Arrête le chrono et montre la vérité. Hors délai, tous les nombres sont
 * faux : c'est un verdict, rien à cocher.
 * @param {TrainingSessionState} session @param {number} now
 * @param {{outOfTime?: boolean}} [opts]
 * @returns {TrainingSessionState}
 */
export function reveal(session, now, { outOfTime = false } = {}) {
    if (!session.question || session.revealed) return session;
    // En mode CHOISI, rien ne se révèle sans réponse : c'est le verdict du juge
    // qui arrête le chrono (`answerChosen`). Seule l'échéance révèle à sa place.
    if (isChosenExercise(session.exercise) && !outOfTime) return session;
    const judged = isEnteredExercise(session.exercise) && !outOfTime;
    const verdicts = session.question.numbers.map((number, i) => (judged ? judgeNumber(number, session.answers[i]) : { wrong: outOfTime, deviation: null }));
    return {
        ...session,
        revealed: true,
        outOfTime,
        elapsedMs: Math.max(0, now - session.startedAt),
        faults: verdicts.map((v) => v.wrong),
        deviations: verdicts.map((v) => v.deviation)
    };
}

/**
 * Le verdict du juge en mode CHOISI : arrête le chrono et décide seul de la
 * faute. Juste = classé et sans coût ; illégal ou non évalué = faute à coût
 * nul. Un second verdict ne remplace pas le premier.
 *
 * @param {TrainingSessionState} session @param {QuizVerdict} verdict @param {number} now
 * @returns {TrainingSessionState}
 */
export function answerChosen(session, verdict, now) {
    if (!session.question || session.revealed || !verdict) return session;
    const correct = !!verdict.matched && verdict.errorMp === 0;
    return {
        ...session,
        revealed: true,
        outOfTime: false,
        elapsedMs: Math.max(0, now - session.startedAt),
        faults: session.question.numbers.map(() => !correct),
        deviations: session.question.numbers.map(() => null),
        verdict
    };
}

/**
 * Le meilleur coup d'une question CHOISIE restée sans réponse. Arrivé après un
 * aller-retour, il ne s'attache qu'à sa question (`key`), jamais sur un verdict.
 * @param {TrainingSessionState} session @param {string} key @param {QuizVerdict} correction
 * @returns {TrainingSessionState}
 */
export function attachCorrection(session, key, correction) {
    if (!session.question || session.question.key !== key || !session.outOfTime || session.verdict || !correction) return session;
    return { ...session, verdict: correction };
}

/**
 * Le jugement d'un nombre SAISI (ADR-0040 règle 2). L'écart est SIGNÉ et gardé
 * même si la réponse est bonne : le sens de l'erreur est la leçon. Un champ
 * vide ou illisible est une faute SANS écart (le compter zéro fausserait la
 * moyenne).
 *
 * Un nombre CHOISI se juge exactement contre l'option du moteur, sans
 * tolérance ; rien de choisi est une faute.
 *
 * @param {TrainingNumber} number @param {string} text
 */
function judgeNumber(number, text) {
    if (isChosenNumber(number)) return { wrong: !text || text !== number.answer, deviation: null };
    const value = parseFloat(String(text ?? '').replace(',', '.'));
    if (!Number.isFinite(value)) return { wrong: true, deviation: null };
    const deviation = value - number.value;
    return { wrong: Math.abs(deviation) > (number.tolerance ?? 0), deviation };
}

/**
 * @param {TrainingNumber|null|undefined} number
 */
export function isChosenNumber(number) {
    return number?.mode === 'chosen';
}

/**
 * Coche ou décoche le nombre raté ; sans effet avant la révélation ou hors
 * délai.
 * @param {TrainingSessionState} session @param {number} index
 * @returns {TrainingSessionState}
 */
export function toggleFault(session, index) {
    if (!session.revealed || session.outOfTime) return session;
    // En mode SAISI ou CHOISI, c'est l'application qui juge : cocher
    // reviendrait à se donner raison contre la tolérance ou contre l'analyse,
    // et ce qui est enregistré ne correspondrait plus à la faute comptée.
    if (isEnteredExercise(session.exercise) || isChosenExercise(session.exercise)) return session;
    if (index < 0 || index >= session.faults.length) return session;
    const faults = session.faults.slice();
    faults[index] = !faults[index];
    return { ...session, faults };
}

/**
 * Enregistre la question révélée dans la session. Rien n'est écrit en base
 * avant « Terminer » ; « Quitter » jette tout.
 * @param {TrainingSessionState} session
 * @returns {TrainingSessionState}
 */
export function recordQuestion(session) {
    if (!session.question || !session.revealed) return session;
    const items = session.question.numbers.map((number, i) => {
        // Mode déclaré : pas d'écart. `false` garde la moyenne vide plutôt que
        // nulle — zéro dirait « sans erreur », pas « sans mesure ».
        const deviation = session.deviations[i];
        return {
            numberType: number.type,
            wrong: session.faults[i] === true,
            hasDeviation: deviation !== null && deviation !== undefined,
            deviation: deviation ?? 0
        };
    });
    // Le PR ne compte que les décisions jugées : hors délai, ni coût ni
    // décision. Un coup illégal a été joué : il compte, sans coût.
    const judged = isChosenExercise(session.exercise) && !session.outOfTime && !!session.verdict;
    return {
        ...session,
        question: null,
        revealed: false,
        outOfTime: false,
        faults: [],
        answers: [],
        deviations: [],
        verdict: null,
        decisions: session.decisions + (judged ? 1 : 0),
        sumErrorMp: session.sumErrorMp + (judged ? session.verdict?.errorMp || 0 : 0),
        questionError: '',
        askedQuestions: session.askedQuestions + 1,
        times: session.outOfTime ? session.times : [...session.times, session.elapsedMs],
        items: [...session.items, ...items]
    };
}

/**
 * La ligne de journal d'une session terminée ; une question révélée non
 * enregistrée n'y figure pas.
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
        // Le PR n'existe que pour Décision : ailleurs aucune erreur d'équité
        // n'est mesurée, et un zéro dirait « sans faute » (storage.TrainingSession).
        pr: session.exercise === 'decision' ? quizPR(session.sumErrorMp, session.decisions) : 0,
        items: session.items
    };
}

/**
 * Le bilan d'un exercice, lignes de journal les plus récentes d'abord : taux
 * de fautes, écart moyen s'il existe, temps médian, tendance.
 *
 * Temps médian = médiane des médianes de session (robuste aux pauses).
 * Tendance = taux des dix dernières sessions moins taux global ; annoncée
 * seulement AU-DELÀ de dix, car à dix l'écart vaut zéro par construction.
 * PR de Décision = celui de la dernière session : une moyenne de sessions de
 * longueurs différentes ne se pondère pas juste.
 *
 * @param {{exercise?: string, numbersAsked: number, faults: number, deviations: number, meanDeviation: number, medianMs: number, pr?: number}[]} sessions
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
        trend: rows.length > TREND_WINDOW && faultRate !== null && recentFaultRate !== null ? recentFaultRate - faultRate : null,
        lastPr: rows.length > 0 ? (rows[0].pr ?? 0) : null
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
