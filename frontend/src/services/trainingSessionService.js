import { get } from 'svelte/store';
import { LoadPosition, ComputeEPCFromPosition } from '../../wailsjs/go/database/Database.js';
import { positionsStore } from '../stores/positionStore.js';
import { trainingDrillStore, trainingQuestionsStore, trainingIndexStore, trainingAnswersStore, trainingActiveStore, trainingVerdictStore, trainingCurrentStore } from '../stores/trainingStore.js';
import { grade, summarize, saveSession, pipTruth, takePointTruth } from './trainingService.js';
import { showImportedPosition } from './importService.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// Les micro-entraînements (#273, fiche I.17), côté application.
//
// Le service compose une session : il tire des positions de la base, calcule
// la réponse attendue AVANT de poser la question, et amène chaque position sur
// le plateau. Le jugement lui-même est dans trainingService.js, qui ne connaît
// ni le plateau ni Wails et se teste donc sans eux.
//
// Les positions sont tirées de la liste PARCOURUE, pas d'une requête à part :
// s'entraîner sur ce qu'on vient de chercher est le geste, et une seconde
// notion de « le corpus d'entraînement » aurait été un réglage de plus à
// expliquer.

/** Le nombre de questions d'une session. Cinq : assez pour qu'une moyenne
 *  veuille dire quelque chose, assez peu pour tenir dans une pause. */
export const SESSION_LENGTH = 5;

/** Le nombre de tirages avant d'abandonner la recherche d'une position qui
 *  convient à l'exercice (une course, pour l'EPC). */
const MAX_DRAWS = 60;

let questionStartedAt = 0;

/**
 * Tire au hasard `count` index distincts dans [0, length[. Sans remise : la
 * même position deux fois dans une session de cinq se remarquerait.
 * @param {number} length @param {number} count
 */
function drawIndices(length, count) {
    const pool = [];
    for (let i = 0; i < length; i++) pool.push(i);
    for (let i = pool.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [pool[i], pool[j]] = [pool[j], pool[i]];
    }
    return pool.slice(0, Math.min(count, pool.length));
}

/**
 * Compose les questions d'une session de comptage de pions ou d'EPC.
 * @param {string} drill
 */
async function buildBoardQuestions(drill) {
    const { length } = get(positionsStore);
    if (length === 0) return [];
    const questions = [];
    const tried = new Set();
    const candidates = drawIndices(length, Math.min(MAX_DRAWS, length));
    for (const index of candidates) {
        if (questions.length >= SESSION_LENGTH) break;
        const id = positionsStore.idAt(index);
        if (id == null || tried.has(id)) continue;
        tried.add(id);
        let position;
        try {
            position = await LoadPosition(id);
        } catch {
            continue;
        }
        if (!position) continue;
        if (drill === 'pips') {
            questions.push({ drill, positionId: id, truth: pipTruth(position), prompt: '' });
            continue;
        }
        // EPC : seule une position que le moteur accepte fait une question.
        // Une position de contact renverrait un refus, et poser une question
        // dont la réponse est « on ne sait pas » n'entraîne à rien.
        try {
            const result = await ComputeEPCFromPosition(position);
            // Bottom = Noir (index 0), Top = Blanc — la convention de race.EPC.
            const side = position.player_on_roll === 0 ? result?.bottom : result?.top;
            const truth = side?.epc?.epc;
            if (!Number.isFinite(truth)) continue;
            questions.push({ drill, positionId: id, truth, prompt: '' });
        } catch {
            continue;
        }
    }
    return questions;
}

/**
 * Compose les questions d'une session de point de prise. Celles-là ne
 * demandent pas de position : le score EST la question, et la table tp2 en
 * donne la réponse.
 */
function buildTakePointQuestions() {
    const questions = [];
    const seen = new Set();
    while (questions.length < SESSION_LENGTH && seen.size < 49) {
        const away1 = 2 + Math.floor(Math.random() * 7);
        const away2 = 2 + Math.floor(Math.random() * 7);
        const key = `${away1}:${away2}`;
        if (seen.has(key)) continue;
        seen.add(key);
        const truth = takePointTruth(away1, away2);
        if (truth == null) continue;
        questions.push({ drill: 'takepoint', positionId: null, truth, prompt: key });
    }
    return questions;
}

/**
 * Démarre une session. Refuse en le disant plutôt que d'ouvrir une session
 * vide : une base sans course ne peut pas faire travailler l'EPC, et le dire
 * vaut mieux que cinq questions sans réponse.
 * @param {string} drill
 */
export async function startTraining(drill) {
    const questions = drill === 'takepoint' ? buildTakePointQuestions() : await buildBoardQuestions(drill);
    if (questions.length === 0) {
        setStatusBarMessage(tMsg('training.noQuestions'));
        return false;
    }
    trainingDrillStore.set(drill);
    trainingQuestionsStore.set(questions);
    trainingIndexStore.set(0);
    trainingAnswersStore.set([]);
    trainingVerdictStore.set(null);
    trainingActiveStore.set(true);
    await showCurrentQuestion();
    return true;
}

async function showCurrentQuestion() {
    questionStartedAt = Date.now();
    const question = get(trainingCurrentStore);
    if (question?.positionId != null) {
        await showImportedPosition(question.positionId);
    }
}

/**
 * Juge la réponse et l'enregistre. Le verdict reste affiché jusqu'au passage
 * à la question suivante : une correction qu'on n'a pas le temps de lire
 * n'apprend rien.
 * @param {number} answer
 */
export function answerCurrent(answer) {
    const question = get(trainingCurrentStore);
    if (!question) return null;
    const verdict = grade(question.drill, answer, question.truth);
    const entry = { ...verdict, ms: Math.max(0, Date.now() - questionStartedAt) };
    trainingAnswersStore.update((list) => [...list, entry]);
    trainingVerdictStore.set({ ...entry, truth: question.truth, answer });
    return entry;
}

/** Passe à la question suivante, ou termine la session à la dernière. */
export async function nextQuestion() {
    const questions = get(trainingQuestionsStore);
    const index = get(trainingIndexStore);
    if (index + 1 >= questions.length) {
        await finishTraining();
        return;
    }
    trainingIndexStore.set(index + 1);
    trainingVerdictStore.set(null);
    await showCurrentQuestion();
}

/** Termine la session et range son résumé. */
export async function finishTraining() {
    const drill = get(trainingDrillStore);
    const answers = get(trainingAnswersStore);
    const summary = summarize(answers);
    stopTraining();
    if (summary.count === 0) return summary;
    try {
        await saveSession({ drill, ...summary });
    } catch (error) {
        logger.error('could not record the training session:', error);
    }
    setStatusBarMessage(
        tMsg('training.finished', {
            correct: summary.correct,
            n: summary.count,
            seconds: (summary.medianMs / 1000).toFixed(1)
        })
    );
    return summary;
}

/** Quitte la session sans rien enregistrer. */
export function stopTraining() {
    trainingActiveStore.set(false);
    trainingDrillStore.set(null);
    trainingQuestionsStore.set([]);
    trainingIndexStore.set(0);
    trainingAnswersStore.set([]);
    trainingVerdictStore.set(null);
}
