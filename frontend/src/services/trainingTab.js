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
 * Les exercices servis à ce jour. Évaluation vient ensuite (#322).
 *
 * `sources` est la liste des sources de graine que l'exercice accepte, dans
 * l'ordre où le lanceur les propose ; une liste vide veut dire que la question
 * ne vient d'aucune position — c'est le cas de Scores, dont le vivier est les
 * 36 scores non ordonnés.
 *
 * `mode` est le mode de réponse (ADR-0040 règle 2), propriété de l'exercice et
 * jamais réglage de session : ce qui est COMPTÉ ou RÉCITÉ se déclare (on
 * révèle, on coche ce qu'on a raté), ce qui est ESTIMÉ se saisit — parce que
 * là, la taille de l'erreur est la leçon.
 *
 * `boardIsTheQuestion` distingue les deux façons dont la source « plateau » se
 * comporte. Pour Pions, le plateau EST la question : il n'y en a qu'une à
 * poser, et « Suivante » reposerait la même. Pour Bearoff, le plateau est une
 * GRAINE que le moteur joue sur un à quatre plis (ADR-0041 règle 2), donc
 * chaque question diffère et l'enchaînement a un sens.
 *
 * `surface` est ce qui, hors de l'onglet, montre la question (ADR-0040 règle
 * 2) : le plateau, ou rien. Tant qu'une question à surface `board` est à
 * l'écran, le plateau lui APPARTIENT — le répartiteur ne le fait pas défiler
 * sous elle (`questionOnBoard`).
 *
 * Décision (#323) se répond en mode CHOISI : le coup se joue sur le plateau
 * (`quizPlay.js`) ou l'action de videau se clique, et le juge du moteur rend
 * le verdict contre l'analyse enregistrée. Sa seule source est la
 * bibliothèque : une question demande une analyse, et seule une position de la
 * base en porte une.
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
 * Le plateau appartient-il à la question en ce moment ? Oui tant qu'une
 * question d'un exercice à surface `board` est à l'écran, révélée ou non :
 * révélée, sa vérité est encore affichée dans le panneau, et elle ne décrirait
 * plus rien si la liste avait défilé dessous (#323, défaut hérité de #321).
 * @param {TrainingSessionState|null|undefined} session
 */
export function questionOnBoard(session) {
    if (!session?.question) return false;
    return TRAINING_EXERCISES.find((e) => e.id === session.exercise)?.surface === 'board';
}

/**
 * Les mots que `train <exercice>` accepte. Les alias sont ceux que les doigts
 * ont appris : `tp` et `takepoint` pour la fiche de score, comme les tables du
 * même nom ; `epc` pour Bearoff, qui s'appelait ainsi dans la bande ; `quiz`
 * pour Décision, qui en est la suite dans l'onglet (#323). Une commande qu'on
 * tapait hier ne doit pas répondre « exercice inconnu » aujourd'hui.
 */
const EXERCISE_WORDS = Object.freeze({
    scores: 'scores',
    tp: 'scores',
    takepoint: 'scores',
    pips: 'pips',
    pip: 'pips',
    bearoff: 'bearoff',
    epc: 'bearoff',
    decision: 'decision',
    quiz: 'decision'
});

/**
 * L'exercice qu'un mot de la commande `train` désigne, ou `null`.
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
 * Le PR d'une session de Décision, sur la MÊME échelle que celui que les
 * statistiques calculent pour le jeu réel : 500 × erreur moyenne en équité
 * normalisée. C'est ce qui rend les deux nombres comparables — sans quoi
 * l'exercice aurait inventé une échelle de plus.
 *
 * Cette fonction double `engine.QuizPR` côté Go, et le double est assumé : le
 * nombre est calculé ici sur des verdicts déjà rendus, sans aller-retour. La
 * formule, elle, est celle de `storage.pr` et n'a pas d'autre variante.
 *
 * @param {number} sumErrorMp @param {number} decisions
 */
export function quizPR(sumErrorMp, decisions) {
    if (!decisions) return 0;
    return (500 * sumErrorMp) / 1000 / decisions;
}

/**
 * Y a-t-il une question SUIVANTE à poser ? Non quand la source est le plateau
 * d'un exercice dont le plateau est la question : on reposerait la même.
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
 * @property {string[]} answers en mode SAISI, ce qui a été tapé, nombre par nombre
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
 * Pose une question : le chrono part à l'affichage, jamais avant (la
 * fabrication de la question se cache derrière le temps de réflexion de la
 * précédente, elle ne se facture pas à celle-ci).
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
 * Enregistre ce qui est tapé dans un champ, sans le juger : le jugement est à
 * « Valider », une seule fois, et un champ qui se juge en cours de frappe
 * annoncerait la réponse avant qu'on ait fini de la donner.
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
 * Aucune question n'a pu être posée — la position tirée a disparu, la source
 * s'est vidée. La session RESTE OUVERTE et le dit : ce qui a déjà été répondu
 * est encore là, « Terminer » l'enregistre, et l'on peut réessayer. Une
 * session ne se perd jamais sans que l'utilisateur l'ait décidé, et un écran
 * qui rebascule tout seul sur le lanceur est une perte, pas une information.
 * @param {TrainingSessionState} session @param {string} reason
 * @returns {TrainingSessionState}
 */
export function failNextQuestion(session, reason) {
    return { ...session, question: null, revealed: false, outOfTime: false, faults: [], answers: [], deviations: [], verdict: null, questionError: reason };
}

/**
 * Arrête le chrono et montre la vérité. Hors délai, la question compte tous
 * ses nombres faux : on ne mesure pas une réponse qui n'a pas été donnée, et
 * la cocher ensuite n'aurait pas de sens — c'est un verdict, pas une
 * déclaration.
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
 * Le verdict du juge sur une question en mode CHOISI (Décision, #323). Il
 * arrête le chrono comme « Révéler », et il est seul à décider de la faute :
 * n'est juste qu'un coup — ou une action — classé et sans coût. Un coup
 * illégal ou légal mais non évalué est une faute qui ne coûte rien ; le
 * verdict garde les trois issues distinctes pour le panneau.
 *
 * Un second verdict ne remplace pas le premier : la réponse est donnée.
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
 * La correction d'une question CHOISIE restée sans réponse : le meilleur coup,
 * demandé au juge après l'échéance. Elle n'arrive qu'après un aller-retour,
 * donc elle ne s'attache qu'à la question qui l'a demandée (`key`), et jamais
 * par-dessus un verdict.
 * @param {TrainingSessionState} session @param {string} key @param {QuizVerdict} correction
 * @returns {TrainingSessionState}
 */
export function attachCorrection(session, key, correction) {
    if (!session.question || session.question.key !== key || !session.outOfTime || session.verdict || !correction) return session;
    return { ...session, verdict: correction };
}

/**
 * Le jugement d'un nombre SAISI (ADR-0040 règle 2). L'écart est SIGNÉ et il est
 * gardé même quand la réponse est bonne : surestimer n'est pas sous-estimer, et
 * c'est le sens de l'erreur qu'on vient apprendre — « je surestime les
 * positions à trous » ne se lit que sur des écarts signés.
 *
 * Un champ vide ou illisible est une faute SANS écart : on ne mesure pas une
 * réponse qui n'a pas été donnée, et la compter zéro tirerait la moyenne vers
 * une justesse qui n'a pas eu lieu.
 *
 * @param {TrainingNumber} number @param {string} text
 */
function judgeNumber(number, text) {
    const value = parseFloat(String(text ?? '').replace(',', '.'));
    if (!Number.isFinite(value)) return { wrong: true, deviation: null };
    const deviation = value - number.value;
    return { wrong: Math.abs(deviation) > (number.tolerance ?? 0), deviation };
}

/**
 * Coche — ou décoche — le nombre qu'on a raté. Sans effet tant que la question
 * n'est pas révélée (elle est encore ouverte) et sur une question hors délai
 * (son verdict est déjà rendu).
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
 * Enregistre la question révélée dans la session et la retire de l'écran.
 * Rien n'est écrit en base ici : la session entière l'est à « Terminer », et
 * « Quitter » la jette.
 * @param {TrainingSessionState} session
 * @returns {TrainingSessionState}
 */
export function recordQuestion(session) {
    if (!session.question || !session.revealed) return session;
    const items = session.question.numbers.map((number, i) => {
        // Le mode déclaré ne produit aucun écart : un compte de pions ou une
        // case de table est juste ou faux. Rester à `false` ici est ce qui
        // garde la moyenne des écarts vide plutôt que nulle — une colonne à
        // zéro dirait « sans erreur », pas « sans mesure ».
        const deviation = session.deviations[i];
        return {
            numberType: number.type,
            wrong: session.faults[i] === true,
            hasDeviation: deviation !== null && deviation !== undefined,
            deviation: deviation ?? 0
        };
    });
    // Le PR ne compte que les décisions JUGÉES : une question hors délai n'a
    // pas de réponse, donc ni coût ni décision — on ne mesure pas une réponse
    // qui n'a pas été donnée. Un coup illégal, lui, a été joué : il compte, et
    // ne coûte rien.
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
        // Le PR n'existe que pour Décision : ailleurs aucune erreur d'équité
        // n'est mesurée, et un zéro dirait « sans faute » (storage.TrainingSession).
        pr: session.exercise === 'decision' ? quizPR(session.sumErrorMp, session.decisions) : 0,
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
 * Le PR, pour Décision, est celui de la DERNIÈRE session, tel qu'enregistré :
 * une moyenne de PR de sessions de longueurs différentes serait une échelle de
 * plus, et le journal n'a pas de quoi la pondérer juste.
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
