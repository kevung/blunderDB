import { get } from 'svelte/store';
import {
    LoadPosition,
    LoadPositionIDsByFilters,
    SaveTrainingSession,
    LoadTrainingSessions,
    LoadTrainingNumberStats,
    LoadAnalysis,
    GradeQuizChecker,
    GradeQuizCheckerMove,
    GradeQuizCube
} from '../../wailsjs/go/database/Database.js';
import { GenerateBearoffQuestion, GenerateEvaluationQuestion, LegalMoves } from '../../wailsjs/go/gui/App.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { emptySearchBoardPosition } from '../stores/searchExcludePositionStore.js';
import { trainingSessionStore, trainingElapsedStore, trainingJournalStore, trainingRefusalStore } from '../stores/trainingTabStore.js';
import {
    TRAINING_EXERCISES,
    newSession,
    askQuestion,
    reveal,
    toggleFault,
    setAnswer,
    recordQuestion,
    failNextQuestion,
    finishedSession,
    canAskAnother,
    answerChosen,
    attachCorrection
} from './trainingTab.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { newPlay, completedPlay, undoLast, resetPlay } from './quizPlay.js';
import { UNORDERED_SCORES, buildScoreCard, scoreCardNumbers } from './scoreCard.js';
import { computePipCount } from '../utils/boardGeometry.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// L'onglet Entraînement, côté application (ADR-0040, ADR-0041) : fabrique les
// questions, tient le chronomètre, écrit au journal. Les RÈGLES sont dans
// trainingTab.js, testable sans Svelte ni Wails.
//
// Le plateau montre la question mais ne porte aucun bouton : en Décision, le
// coup se joue au plateau, mais annuler, reprendre et valider sont au panneau.

/** Le battement du chronomètre : assez fin pour qu'une limite de 15 s se voie
 *  arriver, assez lâche pour ne rien coûter. */
const TICK_MS = 200;

/** @type {ReturnType<typeof setInterval>|null} */
let ticker = null;

/** @param {string} id */
export function exerciseById(id) {
    return TRAINING_EXERCISES.find((e) => e.id === id) || null;
}

// ── Le chronomètre ───────────────────────────────────────────────────────────
// Dans le service et non le composant : le panneau est démonté au changement
// d'onglet, et la limite doit continuer de courir.

function stopTicker() {
    if (ticker !== null) {
        clearInterval(ticker);
        ticker = null;
    }
}

function startTicker() {
    stopTicker();
    ticker = setInterval(() => {
        const session = get(trainingSessionStore);
        if (!session || !session.question || session.revealed) return;
        const elapsed = Math.max(0, Date.now() - session.startedAt);
        trainingElapsedStore.set(elapsed);
        if (session.limitSeconds > 0 && elapsed >= session.limitSeconds * 1000) {
            revealQuestion({ outOfTime: true });
        }
    }, TICK_MS);
}

// ── Les questions ────────────────────────────────────────────────────────────

/** Une question de l'exercice Scores : un des 36 scores non ordonnés, tiré au
 *  sort, et les cases que les tables définissent pour lui. */
function buildScoresQuestion() {
    const [a, b] = UNORDERED_SCORES[Math.floor(Math.random() * UNORDERED_SCORES.length)];
    const card = buildScoreCard(a, b);
    return { kind: 'scores', key: `${a}:${b}`, card, numbers: scoreCardNumbers(card) };
}

/** Les deux comptes de pions d'une position. Le compte des DEUX camps : c'est
 *  la différence qui décide, et compter un seul côté ne la donne pas.
 *  @param {any} position */
function pipNumbers(position) {
    const { pipCount1, pipCount2 } = computePipCount(position);
    return [
        { type: 'pips.bottom', value: pipCount1 },
        { type: 'pips.top', value: pipCount2 }
    ];
}

/**
 * Une question de Pions. Ne montre rien (« Fabriquer n'est pas montrer »).
 * @param {string} seedSource @param {any} seed la graine « plateau », capturée au démarrage
 */
async function buildPipsQuestion(seedSource, seed) {
    if (seedSource === 'library') {
        const { length } = get(positionsStore);
        if (length === 0) return { question: null, refusal: 'noQuestion' };
        const id = positionsStore.idAt(Math.floor(Math.random() * length));
        if (id == null) return { question: null, refusal: 'noQuestion' };
        const position = await LoadPosition(id);
        if (!position) return { question: null, refusal: 'noQuestion' };
        return { question: { kind: 'pips', key: String(id), positionId: id, position, numbers: pipNumbers(position) }, refusal: '' };
    }
    if (!seed?.board?.points) return { question: null, refusal: 'noQuestion' };
    return { question: { kind: 'pips', key: 'board', positionId: null, numbers: pipNumbers(seed) }, refusal: '' };
}

/** La tolérance de l'EPC : un demi-pion. C'est la granularité à laquelle il
 *  change une décision de course — plus fin ne mesurerait que la patience. */
const EPC_TOLERANCE = 0.5;

/** Le nombre de tirages avant de renoncer à trouver un bearoff dans la liste
 *  parcourue. Borné, parce qu'une base sans course en ferait autrement une
 *  boucle : au bout, on refuse EN LE NOMMANT plutôt que de tourner. */
const MAX_LIBRARY_DRAWS = 30;

/**
 * Tire `count` éléments distincts : sans remise, pour que chaque tirage du
 * budget regarde une candidate de plus.
 * @param {number[]} candidates @param {number} count
 */
function drawDistinct(candidates, count) {
    const pool = candidates.slice();
    for (let i = pool.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [pool[i], pool[j]] = [pool[j], pool[i]];
    }
    return pool.slice(0, Math.min(count, pool.length));
}

/** Les deux EPC d'une position engendrée, dans l'ordre du plateau.
 *  @param {any} epc */
function epcNumbers(epc) {
    return [
        { type: 'epc.bottom', value: epc?.bottom?.epc?.epc ?? 0, tolerance: EPC_TOLERANCE, precision: 1 },
        { type: 'epc.top', value: epc?.top?.epc?.epc ?? 0, tolerance: EPC_TOLERANCE, precision: 1 }
    ];
}

/**
 * Une question de Bearoff (ADR-0041), en un seul appel : le moteur joue les
 * plis, compose la position et rend les deux EPC, hors du chrono.
 *
 * La source `base` laisse le moteur juger chaque candidate : le domaine n'est
 * écrit qu'en Go. Ne montre rien (« Fabriquer n'est pas montrer »).
 *
 * @param {string} seedSource @param {any} seed la graine « plateau », capturée au démarrage
 */
async function buildBearoffQuestion(seedSource, seed) {
    if (seedSource === 'library') {
        const { length } = get(positionsStore);
        const phased = await bearoffIndices();
        const candidates = phased.length > 0 ? phased : Array.from({ length }, (_, i) => i);
        let last = 'notBearoff';
        for (const index of drawDistinct(candidates, MAX_LIBRARY_DRAWS)) {
            const id = positionsStore.idAt(index);
            if (id == null) continue;
            const loaded = await LoadPosition(id);
            if (!loaded) continue;
            const generated = await GenerateBearoffQuestion(/** @type {any} */ ({ source: 'library', seed: loaded }));
            if (!generated?.generated) {
                last = generated?.refusal || last;
                continue;
            }
            return { question: bearoffQuestion(generated, String(id), id, loaded), refusal: '' };
        }
        return { question: null, refusal: length === 0 ? 'noQuestion' : last };
    }

    const request = seedSource === 'board' ? { source: 'board', seed } : { source: 'pool' };
    const generated = await GenerateBearoffQuestion(/** @type {any} */ (request));
    if (!generated?.generated) return { question: null, refusal: generated?.refusal || 'noQuestion' };
    return { question: bearoffQuestion(generated, ''), refusal: generated.refusal || '' };
}

/**
 * @param {any} generated @param {string} key @param {number|null} positionId
 * @param {any} [loaded] la position de la base, telle que chargée, quand la question en vient
 */
function bearoffQuestion(generated, key, positionId = null, loaded = null) {
    return {
        kind: 'bearoff',
        key: key || `pool:${generated.plies}:${JSON.stringify(generated.position.board.bearoff)}`,
        positionId,
        // Engendrée : portée par la question. Tirée de la base : montrée par
        // son id (`showQuestion`), la copie chargée ne servant que si la liste
        // ne la contient plus.
        position: positionId == null ? generated.position : loaded,
        numbers: epcNumbers(generated.epc)
    };
}

/**
 * Les index, dans la liste parcourue, des positions en phase `bearoff`
 * (ADR-0035), calculés une fois par session. Sans cette restriction, trente
 * tirages à l'aveugle manquent souvent les rares bearoffs d'une base. Elle ne
 * juge pas : le domaine (4 à 15 pions) reste en Go.
 *
 * Liste vide (phases jamais calculées, ou aucun bearoff) : le tirage retombe
 * sur la liste entière.
 *
 * @returns {Promise<number[]>}
 */
function bearoffIndices() {
    if (!bearoffPhaseIndices) {
        bearoffPhaseIndices = (async () => {
            /** @type {number[]} */
            let ids = [];
            try {
                const filters = /** @type {any} */ ({ filter: emptySearchBoardPosition(), excludeFilter: emptySearchBoardPosition(), gamePhaseFilter: 'bearoff' });
                ids = (await LoadPositionIDsByFilters(filters)) || [];
            } catch (error) {
                logger.error('could not narrow the training draw to bear-offs:', error);
            }
            return ids.map((id) => positionsStore.indexOf(id)).filter((index) => index >= 0);
        })();
    }
    return bearoffPhaseIndices;
}

/** Tolérance des chances de gain d'Évaluation, en points de pourcentage : sépare
 *  70 % de 80 % (prise facile contre passe) sans exiger une décimale que le
 *  moteur ne tient pas d'une profondeur à l'autre. */
const WIN_TOLERANCE = 5;

/**
 * Une question d'Évaluation (ADR-0040 règle 3), en un seul appel : le moteur
 * joue les plis, cadre la position en argent et rend la vérité (chances,
 * verdict de videau, bouton juste, EPC si exact). L'interface ne compare
 * aucune équité.
 *
 * La source `base` laisse le moteur juger chaque candidate (domaine écrit en
 * Go). Ne montre rien (« Fabriquer n'est pas montrer »).
 *
 * @param {string} seedSource @param {any} seed la graine « plateau », capturée au démarrage
 */
async function buildEvaluationQuestion(seedSource, seed) {
    if (seedSource === 'library') {
        const { length } = get(positionsStore);
        let last = 'notMoneyCubeDecision';
        for (const index of drawDistinct(
            Array.from({ length }, (_, i) => i),
            MAX_LIBRARY_DRAWS
        )) {
            const id = positionsStore.idAt(index);
            if (id == null) continue;
            const loaded = await LoadPosition(id);
            if (!loaded) continue;
            const generated = await GenerateEvaluationQuestion(/** @type {any} */ ({ source: 'library', seed: loaded }));
            if (!generated?.generated) {
                last = generated?.refusal || last;
                continue;
            }
            return { question: evaluationQuestion(generated, String(id), id, loaded), refusal: '' };
        }
        return { question: null, refusal: length === 0 ? 'noQuestion' : last };
    }

    const request = seedSource === 'board' ? { source: 'board', seed } : { source: 'pool' };
    const generated = await GenerateEvaluationQuestion(/** @type {any} */ (request));
    if (!generated?.generated) return { question: null, refusal: generated?.refusal || 'noQuestion' };
    return { question: evaluationQuestion(generated, ''), refusal: generated.refusal || '' };
}

/**
 * Chances de gain saisies et action de videau choisie (`mode: 'chosen'`),
 * jugée contre le bouton que le moteur rend juste.
 *
 * @param {any} generated @param {string} key @param {number|null} positionId
 * @param {any} [loaded] la position de la base, telle que chargée, quand la question en vient
 */
function evaluationQuestion(generated, key, positionId = null, loaded = null) {
    return {
        kind: 'evaluation',
        key: key || `pool:${generated.plies}:${JSON.stringify(generated.position?.board?.points?.map((/** @type {any} */ p) => p.checkers * (p.color === 1 ? -1 : 1)))}`,
        positionId,
        position: positionId == null ? generated.position : loaded,
        cubeVerdict: generated.cubeVerdict,
        regime: generated.regime,
        depth: generated.depth || '',
        epc: generated.epc ?? null,
        numbers: [
            { type: 'eval.win', value: generated.winChance, tolerance: WIN_TOLERANCE, precision: 1 },
            { type: 'eval.cube', value: 0, mode: 'chosen', answer: generated.cubeAnswer }
        ]
    };
}

/** Analyses lues au plus pour une question de Décision, sans quoi une liste
 *  sans analyse ferait lire toute la base. Au-delà, refus nommé ; « Réessayer »
 *  reprend sur les positions non lues. */
const MAX_DECISION_DRAWS = 60;

/**
 * Les positions déjà lues par la session de Décision (posées ou sans
 * analyse) : ni reposées, ni relues (ADR-0040 règle 5).
 * @type {Set<number>}
 */
let decisionSeen = new Set();

/** Les questions de Décision fabriquées par cette session, préchargée comprise. */
let decisionsBuilt = 0;

/**
 * Une question de Décision (ADR-0040 règle 3). Elle exige une position
 * analysée et pose la décision que la position porte, pion ou videau.
 *
 * Les coups légaux sont demandés ici, pendant le préchargement, où l'attente
 * ne se voit pas ; sans coup offert, la position est passée. Ne montre rien
 * (« Fabriquer n'est pas montrer »).
 */
async function buildDecisionQuestion() {
    if (!get(databasePathStore)) return { question: null, refusal: 'noLibrary' };
    const { length } = get(positionsStore);
    /** @type {number[]} */
    const candidates = [];
    for (let index = 0; index < length; index++) {
        const id = positionsStore.idAt(index);
        if (id != null && !decisionSeen.has(id)) candidates.push(id);
    }
    if (candidates.length === 0) return { question: null, refusal: decisionsBuilt > 0 ? 'decisionsExhausted' : 'noAnalysis' };
    for (const id of drawDistinct(candidates, MAX_DECISION_DRAWS)) {
        // Marquée AVANT la lecture : le préchargement ne doit pas tirer la
        // position que la question courante est en train de lire.
        decisionSeen.add(id);
        let analysis;
        try {
            analysis = await LoadAnalysis(id);
        } catch {
            continue;
        }
        const isCube = !!analysis?.doublingCubeAnalysis;
        const hasMoves = (analysis?.checkerAnalysis?.moves || []).length > 0;
        if (!isCube && !hasMoves) continue;
        const position = await LoadPosition(id);
        if (!position) continue;
        const key = String(id);
        if (isCube) {
            decisionsBuilt++;
            return { question: { kind: 'decision', key, positionId: id, position, prompt: 'cube', numbers: [{ type: 'decision.cube', value: 0 }] }, refusal: '' };
        }
        let plays;
        try {
            plays = await LegalMoves(position);
        } catch (error) {
            logger.error('could not load the legal moves for a decision question:', error);
            continue;
        }
        if (!plays?.length) continue;
        decisionsBuilt++;
        return { question: { kind: 'decision', key, positionId: id, position, plays, prompt: 'checker', numbers: [{ type: 'decision.checker', value: 0 }] }, refusal: '' };
    }
    return { question: null, refusal: 'noAnalysis' };
}

// ── Fabriquer n'est pas montrer ──────────────────────────────────────────────
// `buildQuestion` calcule la position et sa vérité sans toucher au plateau, à
// l'onglet ni à l'index ; `showQuestion` le fait une fois, quand la question
// est posée. Sinon le préchargement de n+1 fait sauter le plateau pendant
// qu'on répond à n.

/**
 * @param {string} exercise @param {string} seedSource @param {any} seed
 * @returns {Promise<{question: any, refusal: string}>}
 */
async function buildQuestion(exercise, seedSource, seed) {
    if (exercise === 'scores') return { question: buildScoresQuestion(), refusal: '' };
    if (exercise === 'bearoff') return buildBearoffQuestion(seedSource, seed);
    if (exercise === 'evaluation') return buildEvaluationQuestion(seedSource, seed);
    if (exercise === 'decision') return buildDecisionQuestion();
    return buildPipsQuestion(seedSource, seed);
}

/**
 * Amène la question au plateau : par son id si elle vient de la base (sinon la
 * position engendrée elle-même) ; Scores n'y touche pas.
 *
 * Par l'index de la liste, jamais `showImportedPosition`, qui bascule sur
 * l'onglet Analyse et cacherait l'onglet où l'on répond (ADR-0040 règle 1).
 * Si la liste a changé, la copie chargée prend le relais.
 * @param {any} question
 */
async function showQuestion(question) {
    if (question?.positionId != null) {
        const index = positionsStore.indexOf(question.positionId);
        if (index >= 0) {
            // -1 d'abord : pointer l'index déjà courant ne rechargerait rien.
            currentPositionIndexStore.set(-1);
            currentPositionIndexStore.set(index);
            return;
        }
        if (question.position) positionStore.set({ ...question.position });
        return;
    }
    if (question?.position) positionStore.set({ ...question.position, id: 0 });
}

/**
 * Arme le plateau pour une Décision : un coup de pions s'y joue, contraint aux
 * coups légaux ; une action de videau rend le plateau à l'application. Les
 * autres exercices n'y touchent pas (la transcription s'en sert aussi).
 * @param {any} question
 */
function armBoard(question) {
    if (question?.kind !== 'decision') return;
    quizPlayStore.set(question.prompt === 'checker' && question.plays?.length ? newPlay(question.position, question.plays) : null);
}

/** Désarme le plateau à la fin d'une session de Décision, et seulement d'elle.
 *  @param {import('./trainingTab.js').TrainingSessionState|null} session */
function disarmBoard(session) {
    if (session?.exercise === 'decision') quizPlayStore.set(null);
}

// ── La session ───────────────────────────────────────────────────────────────

/**
 * La question suivante, fabriquée pendant qu'on répond (ADR-0041 règle 5), hors
 * du chrono qui part à l'affichage.
 *
 * @type {Promise<{question: any, refusal: string}>|null}
 */
let prefetched = null;

/**
 * La graine « plateau », capturée une fois au démarrage (ADR-0041 règle 2).
 * Relue à chaque question, elle dériverait de proche en proche jusqu'à
 * quatre pions, où `playOut` resert indéfiniment la même position.
 *
 * @type {any}
 */
let boardSeed = null;

/**
 * Les index bearoff de la liste parcourue, calculés une fois par session (la
 * liste est celle du démarrage).
 *
 * @type {Promise<number[]>|null}
 */
let bearoffPhaseIndices = null;

/** Lance la fabrication de la question suivante, sans l'attendre.
 *  @param {string} exercise @param {string} seedSource */
function prefetchNextQuestion(exercise, seedSource) {
    if (!canAskAnother(exercise, seedSource)) {
        prefetched = null;
        return;
    }
    prefetched = buildQuestion(exercise, seedSource, boardSeed).catch((error) => {
        logger.error('could not prepare the next training question:', error);
        return { question: null, refusal: 'noQuestion' };
    });
}

/** Prend la question préchargée, ou en fabrique une si aucune ne l'était.
 *  @param {string} exercise @param {string} seedSource */
async function takeNextQuestion(exercise, seedSource) {
    const pending = prefetched;
    prefetched = null;
    if (pending) return pending;
    try {
        return await buildQuestion(exercise, seedSource, boardSeed);
    } catch (error) {
        logger.error('could not build the next training question:', error);
        return { question: null, refusal: 'noQuestion' };
    }
}

/**
 * Démarre une session, ou refuse en nommant la raison (ADR-0041 règle 3) :
 * une graine hors domaine ne démarre rien. Un plateau vide retombe sur le
 * vivier en gardant la phrase.
 *
 * @param {{exercise: string, seedSource?: string, limitSeconds?: number}} opts
 */
export async function startTrainingSession({ exercise, seedSource = '', limitSeconds = 0 }) {
    const declared = exerciseById(exercise);
    if (!declared) {
        setStatusBarMessage(tMsg('training.unknownExercise', { exercise }));
        return false;
    }
    const source = declared.sources.includes(seedSource) ? seedSource : declared.defaultSource;
    disarmBoard(get(trainingSessionStore));
    prefetched = null;
    bearoffPhaseIndices = null;
    decisionSeen = new Set();
    decisionsBuilt = 0;
    // La graine « plateau » se prend ici, et ne se relit jamais.
    boardSeed = get(positionStore);
    let built;
    try {
        built = await buildQuestion(exercise, source, boardSeed);
    } catch (error) {
        logger.error('could not build the first training question:', error);
        built = { question: null, refusal: 'noQuestion' };
    }
    trainingRefusalStore.set(built.refusal || '');
    if (!built.question) {
        setStatusBarMessage(tMsg(refusalMessageKey(built.refusal)));
        return false;
    }
    await showQuestion(built.question);
    armBoard(built.question);
    trainingSessionStore.set(askQuestion(newSession({ exercise, seedSource: source, limitSeconds }), built.question, Date.now()));
    trainingElapsedStore.set(0);
    startTicker();
    prefetchNextQuestion(exercise, source);
    if (built.refusal) setStatusBarMessage(tMsg(refusalMessageKey(built.refusal)));
    return true;
}

/** La clé de traduction d'un refus, générique quand le code est inconnu.
 *  @param {string} code */
export function refusalMessageKey(code) {
    return code && code !== 'noQuestion' ? `training.refusal.${code}` : 'training.noQuestion';
}

/**
 * Arrête le chrono et montre la vérité (« Valider » en mode SAISI).
 * `outOfTime` n'est passé que par le chronomètre, à l'échéance.
 * @param {{outOfTime?: boolean}} [opts]
 */
export function revealQuestion({ outOfTime = false } = {}) {
    const session = get(trainingSessionStore);
    if (!session || !session.question || session.revealed) return;
    const next = reveal(session, Date.now(), { outOfTime });
    trainingSessionStore.set(next);
    trainingElapsedStore.set(next.elapsedMs);
    if (next.revealed && next.question?.kind === 'decision') fetchCorrection(next.question);
}

// ── Décision : répondre ──────────────────────────────────────────────────────
// Le jugement est fait par le backend (le code que le démon appelle aussi),
// pour qu'une note vaille la même chose d'un client à l'autre. Les deux façons
// de répondre ne diffèrent que par l'appel au juge.

/**
 * Juge le coup construit au plateau, par la position résultante que le moteur
 * a rendue — jamais le damier d'affichage, pour qu'une erreur d'affichage ne
 * devienne pas une note. Sans coup complet, rien n'est jugé.
 */
export async function answerDecisionBoard() {
    const session = get(trainingSessionStore);
    const question = session?.question;
    if (!session || session.revealed || question?.kind !== 'decision' || question.prompt !== 'checker') return;
    const state = get(quizPlayStore);
    const play = state ? completedPlay(state) : null;
    if (!play) return;
    await gradeDecision(question, () => GradeQuizChecker(/** @type {number} */ (question.positionId), play.result.board));
}

/**
 * Juge une action de videau : `nd`, `dt` ou `dp`.
 * @param {string} action
 */
export async function answerDecisionCube(action) {
    const session = get(trainingSessionStore);
    const question = session?.question;
    if (!session || session.revealed || question?.kind !== 'decision' || question.prompt !== 'cube') return;
    await gradeDecision(question, () => GradeQuizCube(/** @type {number} */ (question.positionId), action));
}

/**
 * Tronc commun des deux réponses. Le chrono s'arrête au geste, pas au retour du
 * juge. Un juge en échec laisse la question ouverte : une réponse non notée
 * n'est pas comptée.
 * @param {any} question @param {() => Promise<any>} judge
 */
async function gradeDecision(question, judge) {
    const answeredAt = Date.now();
    let verdict;
    try {
        verdict = await judge();
    } catch (error) {
        logger.error('could not grade the decision answer:', error);
        setStatusBarMessage(tMsg('training.gradeFailed'));
        return;
    }
    const session = get(trainingSessionStore);
    // La session a pu passer à autre chose pendant l'aller-retour.
    if (!session || session.question !== question) return;
    const next = answerChosen(session, verdict, answeredAt);
    trainingSessionStore.set(next);
    trainingElapsedStore.set(next.elapsedMs);
}

/**
 * La correction d'une décision sans réponse à l'échéance : le meilleur coup,
 * que le juge rend pour une réponse vide sans rien compter.
 * @param {any} question
 */
function fetchCorrection(question) {
    const id = /** @type {number} */ (question.positionId);
    const ask = question.prompt === 'cube' ? GradeQuizCube(id, '') : GradeQuizCheckerMove(id, '');
    Promise.resolve(ask)
        .then((correction) => {
            const session = get(trainingSessionStore);
            if (session) trainingSessionStore.set(attachCorrection(session, question.key, correction));
        })
        .catch((error) => logger.error('could not read the correction of the decision:', error));
}

/** Annule le dernier pas du coup joué au plateau. */
export function undoDecisionStep() {
    const question = openCheckerQuestion();
    if (question) quizPlayStore.update((state) => (state ? undoLast(state, question.position) : state));
}

/** Remet la position telle que la question la pose : le coup reprend de zéro. */
export function resetDecisionPlay() {
    const question = openCheckerQuestion();
    if (question) quizPlayStore.update((state) => (state ? resetPlay(state, question.position) : state));
}

/** La question de Décision de pions ouverte, s'il y en a une. */
function openCheckerQuestion() {
    const session = get(trainingSessionStore);
    const question = session?.question;
    if (!session || session.revealed || question?.kind !== 'decision' || question.prompt !== 'checker' || !question.position) return null;
    return question;
}

/** Coche — ou décoche — le nombre qu'on a raté. @param {number} index */
export function markFault(index) {
    const session = get(trainingSessionStore);
    if (!session) return;
    trainingSessionStore.set(toggleFault(session, index));
}

/**
 * Saisie d'un champ en mode SAISI ; le jugement est à « Valider ».
 * @param {number} index @param {string} text
 */
export function setTrainingAnswer(index, text) {
    const session = get(trainingSessionStore);
    if (!session) return;
    trainingSessionStore.set(setAnswer(session, index, text));
}

/**
 * Enregistre la question révélée et en pose une nouvelle. Si la suivante ne
 * peut être bâtie, la session reste ouverte et le dit, pour que « Terminer »
 * garde les réponses déjà données.
 */
export async function nextTrainingQuestion() {
    const session = get(trainingSessionStore);
    if (!session || !session.revealed) return;
    const recorded = recordQuestion(session);
    await askNextQuestion(recorded);
}

/**
 * Repose une question après un échec, sans rien enregistrer de plus.
 */
export async function retryTrainingQuestion() {
    const session = get(trainingSessionStore);
    if (!session || session.question) return;
    await askNextQuestion(session);
}

/** @param {import('./trainingTab.js').TrainingSessionState} session */
async function askNextQuestion(session) {
    const built = await takeNextQuestion(session.exercise, session.seedSource);
    if (!built.question) {
        disarmBoard(session);
        trainingSessionStore.set(failNextQuestion(session, built.refusal || 'noQuestion'));
        trainingElapsedStore.set(0);
        setStatusBarMessage(tMsg(refusalMessageKey(built.refusal)));
        return;
    }
    await showQuestion(built.question);
    armBoard(built.question);
    trainingSessionStore.set(askQuestion(session, built.question, Date.now()));
    trainingElapsedStore.set(0);
    prefetchNextQuestion(session.exercise, session.seedSource);
}

/**
 * Termine la session et l'écrit au journal, après la question révélée non
 * encore enregistrée.
 */
export async function finishTrainingSession() {
    const session = get(trainingSessionStore);
    if (!session) return null;
    const closed = session.revealed ? recordQuestion(session) : session;
    stopTicker();
    disarmBoard(session);
    prefetched = null;
    boardSeed = null;
    bearoffPhaseIndices = null;
    decisionSeen = new Set();
    decisionsBuilt = 0;
    trainingSessionStore.set(null);
    trainingElapsedStore.set(0);
    const row = finishedSession(closed);
    if (row.numbersAsked === 0) return row;
    try {
        await SaveTrainingSession(/** @type {any} */ (row));
    } catch (error) {
        logger.error('could not record the training session:', error);
        setStatusBarMessage(tMsg('training.journalFailed'));
        return row;
    }
    await refreshTrainingJournal();
    if (row.exercise === 'decision') {
        const correct = row.numbersAsked - row.faults;
        setStatusBarMessage(tMsg('training.finishedDecision', { correct, n: row.numbersAsked, pr: row.pr.toFixed(2) }));
    }
    return row;
}

/** Quitte la session sans rien enregistrer. */
export function quitTrainingSession() {
    stopTicker();
    disarmBoard(get(trainingSessionStore));
    prefetched = null;
    boardSeed = null;
    bearoffPhaseIndices = null;
    decisionSeen = new Set();
    decisionsBuilt = 0;
    trainingSessionStore.set(null);
    trainingElapsedStore.set(0);
}

// ── Le journal ───────────────────────────────────────────────────────────────

/**
 * Relit le journal, à l'ouverture de l'onglet et après chaque « Terminer » (les
 * seuls moments où il change).
 */
export async function refreshTrainingJournal() {
    if (!get(databasePathStore)) {
        trainingJournalStore.set({});
        return;
    }
    /** @type {Record<string, {sessions: any[], numbers: any[]}>} */
    const journal = {};
    for (const exercise of TRAINING_EXERCISES) {
        try {
            const [sessions, numbers] = await Promise.all([LoadTrainingSessions(exercise.id, 0), LoadTrainingNumberStats(exercise.id)]);
            journal[exercise.id] = { sessions: sessions || [], numbers: numbers || [] };
        } catch (error) {
            logger.error(`could not read the training journal for ${exercise.id}:`, error);
            journal[exercise.id] = { sessions: [], numbers: [] };
        }
    }
    trainingJournalStore.set(journal);
}
