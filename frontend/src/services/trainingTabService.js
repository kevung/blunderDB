import { get } from 'svelte/store';
import { LoadPosition, SaveTrainingSession, LoadTrainingSessions, LoadTrainingNumberStats } from '../../wailsjs/go/database/Database.js';
import { GenerateBearoffQuestion } from '../../wailsjs/go/gui/App.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { trainingSessionStore, trainingElapsedStore, trainingJournalStore, trainingRefusalStore } from '../stores/trainingTabStore.js';
import { TRAINING_EXERCISES, newSession, askQuestion, reveal, toggleFault, setAnswer, recordQuestion, failNextQuestion, finishedSession, canAskAnother } from './trainingTab.js';
import { UNORDERED_SCORES, buildScoreCard, scoreCardNumbers } from './scoreCard.js';
import { computePipCount } from '../utils/boardGeometry.js';
import { showImportedPosition } from './importService.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// L'onglet Entraînement, côté application (#320 puis #321, ADR-0040/0041).
//
// Le service fabrique les questions, tient le chronomètre et écrit au journal.
// Les RÈGLES — ce qu'une révélation produit, ce qu'une échéance produit à sa
// place, ce qu'une session finie laisse — sont dans trainingTab.js, qui ne
// connaît ni Svelte ni Wails et se vérifie donc sans eux.
//
// Tout le geste d'une session est dans le panneau : le plateau montre la
// question de Pions et, une fois révélée, sa réponse, mais il ne porte aucun
// bouton. C'est l'objection qui a supprimé la barre d'entraînement.

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
//
// Il vit dans le service et non dans le composant : le panneau est démonté dès
// qu'on change d'onglet, et une limite qui cesse de courir parce qu'on a
// regardé ailleurs ne serait plus une limite.

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
 * Une question de l'exercice Pions. La source `plateau` prend la position
 * telle quelle ; la source `base` en tire une de la liste parcourue et
 * l'amène sur le plateau.
 * @param {string} seedSource
 */
async function buildPipsQuestion(seedSource) {
    if (seedSource === 'library') {
        const { length } = get(positionsStore);
        if (length === 0) return { question: null, refusal: 'noQuestion' };
        const id = positionsStore.idAt(Math.floor(Math.random() * length));
        if (id == null) return { question: null, refusal: 'noQuestion' };
        const position = await LoadPosition(id);
        if (!position) return { question: null, refusal: 'noQuestion' };
        await showImportedPosition(id);
        return { question: { kind: 'pips', key: String(id), positionId: id, numbers: pipNumbers(position) }, refusal: '' };
    }
    const position = get(positionStore);
    if (!position?.board?.points) return { question: null, refusal: 'noQuestion' };
    return { question: { kind: 'pips', key: 'board', positionId: null, numbers: pipNumbers(position) }, refusal: '' };
}

/** La tolérance de l'EPC : un demi-pion. C'est la granularité à laquelle il
 *  change une décision de course — plus fin ne mesurerait que la patience. */
const EPC_TOLERANCE = 0.5;

/** Le nombre de tirages avant de renoncer à trouver un bearoff dans la liste
 *  parcourue. Borné, parce qu'une base sans course en ferait autrement une
 *  boucle : au bout, on refuse EN LE NOMMANT plutôt que de tourner. */
const MAX_LIBRARY_DRAWS = 30;

/**
 * Tire `count` index DISTINCTS dans [0, length[. Sans remise : retomber deux
 * fois sur la même position ferait passer le budget de tirages sans avoir
 * regardé de candidate de plus.
 * @param {number} length @param {number} count
 */
function drawDistinctIndices(length, count) {
    const pool = Array.from({ length }, (_, i) => i);
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
 * Une question de l'exercice Bearoff (#321, ADR-0041).
 *
 * TOUT se fait en un seul appel : le moteur joue les plis, compose la position
 * ET rend les deux EPC. Le chrono démarre à l'affichage et ne contient donc ni
 * la génération ni un second aller-retour pour la vérité.
 *
 * La source `base` tire dans la liste parcourue et laisse le moteur juger
 * chaque candidate : le domaine n'est écrit qu'à un endroit, en Go, et une
 * seconde définition ici finirait par diverger de celle qui refuse.
 *
 * @param {string} seedSource
 */
async function buildBearoffQuestion(seedSource) {
    if (seedSource === 'library') {
        const { length } = get(positionsStore);
        let last = 'notBearoff';
        for (const index of drawDistinctIndices(length, MAX_LIBRARY_DRAWS)) {
            const id = positionsStore.idAt(index);
            if (id == null) continue;
            const seed = await LoadPosition(id);
            if (!seed) continue;
            const generated = await GenerateBearoffQuestion({ source: 'library', seed });
            if (!generated?.generated) {
                last = generated?.refusal || last;
                continue;
            }
            await showImportedPosition(id);
            return { question: bearoffQuestion(generated, String(id), id), refusal: '' };
        }
        return { question: null, refusal: length === 0 ? 'noQuestion' : last };
    }

    const request = seedSource === 'board' ? { source: 'board', seed: get(positionStore) } : { source: 'pool' };
    const generated = await GenerateBearoffQuestion(request);
    if (!generated?.generated) return { question: null, refusal: generated?.refusal || 'noQuestion' };
    return { question: bearoffQuestion(generated, ''), refusal: generated.refusal || '' };
}

/**
 * @param {any} generated @param {string} key @param {number|null} positionId
 */
function bearoffQuestion(generated, key, positionId = null) {
    return {
        kind: 'bearoff',
        key: key || `pool:${generated.plies}:${JSON.stringify(generated.position.board.bearoff)}`,
        positionId,
        // La position engendrée n'est dans aucune base : elle est portée par la
        // question, et c'est le service qui l'amène sur le plateau.
        position: generated.position,
        numbers: epcNumbers(generated.epc)
    };
}

/**
 * @param {string} exercise @param {string} seedSource
 * @returns {Promise<{question: any, refusal: string}>}
 */
async function buildQuestion(exercise, seedSource) {
    if (exercise === 'scores') return { question: buildScoresQuestion(), refusal: '' };
    if (exercise === 'bearoff') return buildBearoffQuestion(seedSource);
    return buildPipsQuestion(seedSource);
}

/**
 * Met la question sur le plateau quand elle en porte une. Les questions tirées
 * de la base y sont déjà arrivées par `showImportedPosition` ; une position
 * ENGENDRÉE n'a pas d'identifiant, donc rien d'autre ne peut l'y mettre.
 * @param {any} question
 */
function showQuestionOnBoard(question) {
    if (!question?.position) return;
    positionStore.set({ ...question.position, id: 0 });
}

// ── La session ───────────────────────────────────────────────────────────────

/**
 * La question SUIVANTE, en cours de fabrication (ADR-0041 règle 5).
 *
 * Elle se fabrique pendant qu'on répond à la précédente : la génération se
 * cache derrière le temps de réflexion, et le chrono — qui part à l'affichage
 * — ne la voit jamais. Sans ce préchargement, chaque « Suivante » aurait
 * facturé sa propre fabrication à la question qu'elle ouvre.
 *
 * @type {Promise<{question: any, refusal: string}>|null}
 */
let prefetched = null;

/** Lance la fabrication de la question suivante, sans l'attendre.
 *  @param {string} exercise @param {string} seedSource */
function prefetchNextQuestion(exercise, seedSource) {
    if (!canAskAnother(exercise, seedSource)) {
        prefetched = null;
        return;
    }
    prefetched = buildQuestion(exercise, seedSource).catch((error) => {
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
        return await buildQuestion(exercise, seedSource);
    } catch (error) {
        logger.error('could not build the next training question:', error);
        return { question: null, refusal: 'noQuestion' };
    }
}

/**
 * Démarre une session. Refuse en le NOMMANT plutôt que d'ouvrir une session
 * vide : une graine hors du domaine de l'exercice ne démarre rien, et la
 * phrase dit lequel (ADR-0041 règle 3). Un plateau vide, lui, retombe sur le
 * vivier — et garde la phrase, sans quoi la géométrie aurait changé sans que
 * personne l'ait décidé.
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
    prefetched = null;
    let built;
    try {
        built = await buildQuestion(exercise, source);
    } catch (error) {
        logger.error('could not build the first training question:', error);
        built = { question: null, refusal: 'noQuestion' };
    }
    trainingRefusalStore.set(built.refusal || '');
    if (!built.question) {
        setStatusBarMessage(tMsg(refusalMessageKey(built.refusal)));
        return false;
    }
    showQuestionOnBoard(built.question);
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
 * Arrête le chrono et montre la vérité. En mode SAISI, c'est « Valider » : le
 * même geste, le même arrêt du chrono, et l'application juge.
 *
 * `outOfTime` n'est pas un argument que l'interface passe : c'est le
 * chronomètre lui-même qui l'apporte à l'échéance.
 * @param {{outOfTime?: boolean}} [opts]
 */
export function revealQuestion({ outOfTime = false } = {}) {
    const session = get(trainingSessionStore);
    if (!session || !session.question || session.revealed) return;
    const next = reveal(session, Date.now(), { outOfTime });
    trainingSessionStore.set(next);
    trainingElapsedStore.set(next.elapsedMs);
}

/** Coche — ou décoche — le nombre qu'on a raté. @param {number} index */
export function markFault(index) {
    const session = get(trainingSessionStore);
    if (!session) return;
    trainingSessionStore.set(toggleFault(session, index));
}

/**
 * Ce qu'on tape dans un champ, en mode SAISI. Rien n'est jugé ici : le
 * jugement est à « Valider », une fois.
 * @param {number} index @param {string} text
 */
export function setTrainingAnswer(index, text) {
    const session = get(trainingSessionStore);
    if (!session) return;
    trainingSessionStore.set(setAnswer(session, index, text));
}

/**
 * Enregistre la question révélée et en pose une nouvelle.
 *
 * Quand la suivante ne peut PAS être bâtie — la position tirée a été supprimée
 * entre-temps, la liste parcourue s'est vidée — la session reste ouverte et le
 * dit. Elle rebasculait sur le lanceur : « Terminer » disparaissait, les
 * nombres déjà répondus devenaient inatteignables, et « Démarrer » les
 * écrasait. Tout un journal de session partait sans un mot.
 */
export async function nextTrainingQuestion() {
    const session = get(trainingSessionStore);
    if (!session || !session.revealed) return;
    const recorded = recordQuestion(session);
    await askNextQuestion(recorded);
}

/**
 * Repose une question après un échec, sans rien enregistrer de plus : la
 * question précédente l'a déjà été.
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
        trainingSessionStore.set(failNextQuestion(session, built.refusal || 'noQuestion'));
        trainingElapsedStore.set(0);
        setStatusBarMessage(tMsg(refusalMessageKey(built.refusal)));
        return;
    }
    showQuestionOnBoard(built.question);
    trainingSessionStore.set(askQuestion(session, built.question, Date.now()));
    trainingElapsedStore.set(0);
    prefetchNextQuestion(session.exercise, session.seedSource);
}

/**
 * Termine la session et l'écrit au journal. Une question révélée mais pas
 * enregistrée l'est d'abord : on vient de la lire, elle compte.
 */
export async function finishTrainingSession() {
    const session = get(trainingSessionStore);
    if (!session) return null;
    const closed = session.revealed ? recordQuestion(session) : session;
    stopTicker();
    prefetched = null;
    trainingSessionStore.set(null);
    trainingElapsedStore.set(0);
    const row = finishedSession(closed);
    if (row.numbersAsked === 0) return row;
    try {
        await SaveTrainingSession(row);
    } catch (error) {
        logger.error('could not record the training session:', error);
        setStatusBarMessage(tMsg('training.journalFailed'));
        return row;
    }
    await refreshTrainingJournal();
    return row;
}

/** Quitte la session sans rien enregistrer. */
export function quitTrainingSession() {
    stopTicker();
    prefetched = null;
    trainingSessionStore.set(null);
    trainingElapsedStore.set(0);
}

// ── Le journal ───────────────────────────────────────────────────────────────

/**
 * Relit le journal : les sessions de chaque exercice et leur détail par type
 * de nombre. Appelé à l'ouverture de l'onglet et après chaque « Terminer » —
 * les deux seuls moments où il peut avoir changé.
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
