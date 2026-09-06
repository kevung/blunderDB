import { get } from 'svelte/store';
import { LoadPosition, LoadAnalysis, ComputeEPCFromPosition, GradeQuizCheckerMove, GradeQuizCube } from '../../wailsjs/go/database/Database.js';
import { positionsStore } from '../stores/positionStore.js';
import { trainingDrillStore, trainingQuestionsStore, trainingIndexStore, trainingAnswersStore, trainingActiveStore, trainingVerdictStore, trainingCurrentStore } from '../stores/trainingStore.js';
import { grade, summarize, saveSession, pipTruth, takePointTruth, quizPR } from './trainingService.js';
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
 * Compose les questions d'un quiz (#294, fiche J.4).
 *
 * Une question de quiz demande une position ANALYSÉE : l'erreur se mesure
 * contre l'analyse enregistrée, et une position sans analyse ne pose pas de
 * question, elle en pose une à laquelle personne ne sait répondre. La
 * décision est celle que la position porte — coup de pions ou action de
 * videau — jamais une décision inventée pour l'exercice.
 */
async function buildQuizQuestions() {
    const { length } = get(positionsStore);
    if (length === 0) return [];
    const questions = [];
    const tried = new Set();
    for (const index of drawIndices(length, Math.min(MAX_DRAWS, length))) {
        if (questions.length >= SESSION_LENGTH) break;
        const id = positionsStore.idAt(index);
        if (id == null || tried.has(id)) continue;
        tried.add(id);
        let analysis;
        try {
            analysis = await LoadAnalysis(id);
        } catch {
            continue;
        }
        const isCube = !!analysis?.doublingCubeAnalysis;
        const hasMoves = (analysis?.checkerAnalysis?.moves || []).length > 0;
        if (!isCube && !hasMoves) continue;
        questions.push({ drill: 'quiz', positionId: id, truth: 0, prompt: isCube ? 'cube' : 'checker' });
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
    let questions;
    if (drill === 'takepoint') questions = buildTakePointQuestions();
    else if (drill === 'quiz') questions = await buildQuizQuestions();
    else questions = await buildBoardQuestions(drill);
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
 * Juge une réponse de quiz. Le jugement est fait par le BACKEND — engine, le
 * même code que la ligne de commande et le démon appellent — pas ici : la note
 * d'un quiz doit valoir la même chose d'un client à l'autre, et une seconde
 * implémentation en JavaScript aurait été une seconde note.
 *
 * Un coup de pions se répond au clavier, en notation (« 13/7 8/7 ») ; une
 * décision de videau se clique. Le juge accepte aussi un PLATEAU — c'est ce
 * que le geste « jouer le coup sur le damier » appellera le jour où il
 * existera — et les deux chemins passent par la même reconnaissance : la
 * réponse est résolue contre les coups légaux du générateur AVANT d'être
 * comparée à l'analyse, donc un coup impossible est refusé comme illégal et
 * non classé « non évalué ».
 * @param {string} answer la notation du coup, ou 'nd'/'dt'/'dp'
 */
export async function answerQuiz(answer) {
    const question = get(trainingCurrentStore);
    if (!question) return null;
    let verdict;
    try {
        verdict = question.prompt === 'cube' ? await GradeQuizCube(question.positionId, answer) : await GradeQuizCheckerMove(question.positionId, answer);
    } catch (error) {
        logger.error('could not grade the quiz answer:', error);
        setStatusBarMessage(tMsg('training.gradeFailed'));
        return null;
    }
    // Un coup illégal ou non classé ne coûte RIEN : le quiz mesure la qualité
    // d'une décision, et charger une erreur qu'on n'a pas mesurée gonflerait
    // le PR de session avec du bruit. Le verdict le dit, et c'est tout.
    const entry = { correct: verdict.matched && verdict.errorMp === 0, error: verdict.errorMp, ms: Math.max(0, Date.now() - questionStartedAt) };
    trainingAnswersStore.update((list) => [...list, entry]);
    trainingVerdictStore.set({ ...entry, quiz: verdict, truth: 0, answer: verdict.notation });
    return entry;
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
    // Le PR de session n'existe que pour le quiz : il se calcule sur une
    // erreur d'équité, et un compte de pions n'en produit pas.
    const pr =
        drill === 'quiz'
            ? quizPR(
                  answers.reduce((sum, a) => sum + (a.error || 0), 0),
                  answers.length
              )
            : null;
    stopTraining();
    if (summary.count === 0) return summary;
    try {
        await saveSession({ drill, ...summary, ...(pr === null ? {} : { pr }) });
    } catch (error) {
        logger.error('could not record the training session:', error);
    }
    setStatusBarMessage(
        pr === null
            ? tMsg('training.finished', {
                  correct: summary.correct,
                  n: summary.count,
                  seconds: (summary.medianMs / 1000).toFixed(1)
              })
            : tMsg('training.finishedQuiz', {
                  correct: summary.correct,
                  n: summary.count,
                  pr: pr.toFixed(2)
              })
    );
    return { ...summary, pr };
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
