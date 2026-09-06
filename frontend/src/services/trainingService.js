import { LoadMetadata, SaveMetadata } from '../../wailsjs/go/database/Database.js';
import { logger } from '../utils/logger.js';

// Micro-entraînements (#273, fiche I.17).
//
// Anki fait réviser un jugement ; ceci fait travailler des calculs qui se font
// en partie, sous la pendule, et qu'aucune révision espacée ne muscle.
//
// Depuis #320 il n'en reste que DEUX ici : l'EPC et le quiz. Le compte de
// pions et le point de prise sont passés à l'onglet Entraînement, où ils se
// répondent en mode déclaré — on révèle, on coche ce qu'on a raté — parce que
// les taper n'apprenait rien de plus que de les lire. La bande garde ce qui
// se SAISIT, le temps que les tranches suivantes l'y déplacent aussi.
//
// Ce fichier ne connaît ni Svelte ni le plateau : il choisit une question,
// juge une réponse, et range une session. C'est ce qui le rend testable, et
// c'est aussi ce qui permettra à J.4 (#294) de s'y brancher plutôt que de
// réécrire une seconde notion de « note d'entraînement ».

/** Les exercices que la bande sert encore : `epc` est un calcul (#273),
 *  `quiz` le module complet (#294), où le coup se joue sur le plateau et
 *  l'erreur se mesure contre l'analyse enregistrée. Les exercices déclarés
 *  vivent dans l'onglet (services/trainingTab.js). */
export const DRILLS = Object.freeze(['epc', 'quiz']);

// La tolérance de chaque exercice, et pourquoi elle vaut ce qu'elle vaut.
//
//   epc — 0.5. L'EPC est une estimation ; le demi-pion est la granularité à
//         laquelle il change une décision de course.
//
// Un exercice absent de cette table ne tolère rien (`grade` lit 0 par défaut),
// ce qui est la règle et non l'exception : seul un nombre ESTIMÉ a une
// tolérance.
export const TOLERANCE = Object.freeze({ epc: 0.5 });

const KEY_SESSIONS = 'training_sessions';
const MAX_SESSIONS = 50;

/**
 * Juge une réponse. `error` est signé (positif = surestimation) parce que le
 * sens de l'erreur est ce qu'on apprend : compter deux pions de trop n'est
 * pas la même faute que deux de moins.
 * @param {string} drill @param {number} answer @param {number} truth
 */
export function grade(drill, answer, truth) {
    if (!Number.isFinite(answer) || !Number.isFinite(truth)) {
        return { correct: false, error: null };
    }
    const error = answer - truth;
    return { correct: Math.abs(error) <= (TOLERANCE[drill] ?? 0), error };
}

/**
 * Le résumé d'une session : combien de bonnes réponses, l'erreur absolue
 * moyenne, le temps médian. La MÉDIANE et non la moyenne pour le temps :
 * une question où l'on est allé chercher un café ne dit rien du rythme.
 * @param {{correct: boolean, error: number|null, ms: number}[]} answers
 */
export function summarize(answers) {
    const n = answers.length;
    if (n === 0) return { count: 0, correct: 0, rate: 0, meanError: 0, medianMs: 0 };
    const correct = answers.filter((a) => a.correct).length;
    const errors = answers.map((a) => Math.abs(a.error ?? 0));
    const times = answers.map((a) => a.ms).sort((x, y) => x - y);
    const mid = Math.floor(times.length / 2);
    return {
        count: n,
        correct,
        rate: correct / n,
        meanError: errors.reduce((s, e) => s + e, 0) / n,
        medianMs: times.length % 2 ? times[mid] : (times[mid - 1] + times[mid]) / 2
    };
}

/**
 * Les sessions passées, les plus récentes d'abord. Comme l'objectif de
 * progression (#274), elles vivent dans les métadonnées de la BASE et non
 * dans la configuration : elles portent sur cette bibliothèque-là et suivent
 * donc le fichier plutôt que la machine.
 */
export async function loadSessions() {
    try {
        const meta = (await LoadMetadata()) || {};
        const raw = meta[KEY_SESSIONS];
        if (!raw) return [];
        const parsed = JSON.parse(raw);
        return Array.isArray(parsed) ? parsed : [];
    } catch (err) {
        logger.error('could not read the training history:', err);
        return [];
    }
}

/**
 * Range une session. L'historique est borné à cinquante entrées : il sert à
 * voir une progression, pas à tenir un registre, et une métadonnée qui
 * grossit sans fin finirait par voyager dans chaque export.
 * @param {{drill: string, date?: string}} session
 */
export async function saveSession(session) {
    const meta = (await LoadMetadata()) || {};
    let history;
    try {
        history = JSON.parse(meta[KEY_SESSIONS] || '[]');
        if (!Array.isArray(history)) history = [];
    } catch {
        history = [];
    }
    const entry = { ...session, date: session.date || new Date().toISOString().slice(0, 10) };
    history.unshift(entry);
    meta[KEY_SESSIONS] = JSON.stringify(history.slice(0, MAX_SESSIONS));
    await SaveMetadata(meta);
    return entry;
}

/**
 * Le PR d'une session de quiz, sur la MÊME échelle que celui que les
 * statistiques calculent pour le jeu réel : 500 × erreur moyenne en équité
 * normalisée. C'est ce qui rend les deux nombres comparables, et c'était le
 * point de la fiche — sans quoi le module aurait inventé une échelle de plus.
 *
 * Cette fonction double engine.QuizPR côté Go, et le double est assumé : le
 * NOMBRE affiché après une session est calculé ici sur des verdicts déjà
 * rendus, sans aller-retour, tandis que le Go sert les clients du démon. La
 * formule, elle, est celle de storage.pr et n'a pas d'autre variante.
 *
 * @param {number} sumErrorMp @param {number} decisions
 */
export function quizPR(sumErrorMp, decisions) {
    if (!decisions) return 0;
    return (500 * sumErrorMp) / 1000 / decisions;
}
