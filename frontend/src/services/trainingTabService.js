import { get } from 'svelte/store';
import { LoadPosition, SaveTrainingSession, LoadTrainingSessions, LoadTrainingNumberStats } from '../../wailsjs/go/database/Database.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { trainingSessionStore, trainingElapsedStore, trainingJournalStore } from '../stores/trainingTabStore.js';
import { TRAINING_EXERCISES, newSession, askQuestion, reveal, toggleFault, recordQuestion, finishedSession } from './trainingTab.js';
import { UNORDERED_SCORES, buildScoreCard, scoreCardNumbers } from './scoreCard.js';
import { computePipCount } from '../utils/boardGeometry.js';
import { showImportedPosition } from './importService.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// L'onglet Entraînement, côté application (#320, ADR-0040).
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
 *  la différence qui décide, et compter un seul côté ne la donne pas. */
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
        if (length === 0) return null;
        const id = positionsStore.idAt(Math.floor(Math.random() * length));
        if (id == null) return null;
        const position = await LoadPosition(id);
        if (!position) return null;
        await showImportedPosition(id);
        return { kind: 'pips', key: String(id), positionId: id, numbers: pipNumbers(position) };
    }
    const position = get(positionStore);
    if (!position?.board?.points) return null;
    return { kind: 'pips', key: 'board', positionId: null, numbers: pipNumbers(position) };
}

/** @param {string} exercise @param {string} seedSource */
async function buildQuestion(exercise, seedSource) {
    return exercise === 'scores' ? buildScoresQuestion() : buildPipsQuestion(seedSource);
}

// ── La session ───────────────────────────────────────────────────────────────

/**
 * Démarre une session. Refuse en le disant plutôt que d'ouvrir une session
 * vide : une source « base » sans position parcourue ne peut poser aucune
 * question, et le dire vaut mieux qu'un panneau qui n'affiche rien.
 * @param {{exercise: string, seedSource?: string, limitSeconds?: number}} opts
 */
export async function startTrainingSession({ exercise, seedSource = '', limitSeconds = 0 }) {
    const declared = exerciseById(exercise);
    if (!declared) {
        setStatusBarMessage(tMsg('training.unknownExercise', { exercise }));
        return false;
    }
    const source = declared.sources.includes(seedSource) ? seedSource : declared.defaultSource;
    let question;
    try {
        question = await buildQuestion(exercise, source);
    } catch (error) {
        logger.error('could not build the first training question:', error);
        question = null;
    }
    if (!question) {
        setStatusBarMessage(tMsg('training.noQuestion'));
        return false;
    }
    trainingSessionStore.set(askQuestion(newSession({ exercise, seedSource: source, limitSeconds }), question, Date.now()));
    trainingElapsedStore.set(0);
    startTicker();
    return true;
}

/**
 * Arrête le chrono et montre la vérité. `outOfTime` n'est pas un argument que
 * l'interface passe : c'est le chronomètre lui-même qui l'apporte à
 * l'échéance.
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

/** Enregistre la question révélée et en pose une nouvelle. */
export async function nextTrainingQuestion() {
    const session = get(trainingSessionStore);
    if (!session || !session.revealed) return;
    const recorded = recordQuestion(session);
    let question;
    try {
        question = await buildQuestion(recorded.exercise, recorded.seedSource);
    } catch (error) {
        logger.error('could not build the next training question:', error);
        question = null;
    }
    if (!question) {
        trainingSessionStore.set(recorded);
        setStatusBarMessage(tMsg('training.noQuestion'));
        return;
    }
    trainingSessionStore.set(askQuestion(recorded, question, Date.now()));
    trainingElapsedStore.set(0);
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
