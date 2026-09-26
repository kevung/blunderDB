import { LoadMetadata, SaveMetadata } from '../../wailsjs/go/database/Database.js';
import { GRADE_BANDS, bandForPR } from '../components/stats/gradeBands.js';
import { logger } from '../utils/logger.js';

// Les objectifs de progression : « PR < 5 d'ici douze semaines », une cible,
// une échéance, une tendance — rien qui note ou rappelle.
//
// Stocké dans les MÉTADONNÉES de la base, pas la configuration : l'objectif
// suit la bibliothèque, sans schéma nouveau, et la ligne de commande comme le
// démon le lisent déjà.

const KEY_TARGET = 'goal_pr';
const KEY_WEEKS = 'goal_weeks';
const KEY_SET_AT = 'goal_set_at';

/** @typedef {{target: number, weeks: number, setAt: string} | null} Goal */

/** Lit l'objectif de la base, ou null s'il n'y en a pas. */
export async function loadGoal() {
    try {
        const meta = (await LoadMetadata()) || {};
        const target = parseFloat(meta[KEY_TARGET]);
        const weeks = parseInt(meta[KEY_WEEKS], 10);
        if (!Number.isFinite(target) || !Number.isFinite(weeks) || weeks <= 0) return null;
        return { target, weeks, setAt: meta[KEY_SET_AT] || '' };
    } catch (err) {
        logger.error('could not read the progression goal:', err);
        return null;
    }
}

/**
 * Écrit l'objectif en relisant les autres métadonnées : la table est un
 * dictionnaire unique, qu'une écriture partielle effacerait.
 * @param {number} target @param {number} weeks
 */
export async function saveGoal(target, weeks) {
    const meta = (await LoadMetadata()) || {};
    meta[KEY_TARGET] = String(target);
    meta[KEY_WEEKS] = String(weeks);
    meta[KEY_SET_AT] = new Date().toISOString().slice(0, 10);
    await SaveMetadata(meta);
    return { target, weeks, setAt: meta[KEY_SET_AT] };
}

/** Efface l'objectif. */
export async function clearGoal() {
    const meta = (await LoadMetadata()) || {};
    delete meta[KEY_TARGET];
    delete meta[KEY_WEEKS];
    delete meta[KEY_SET_AT];
    await SaveMetadata(meta);
}

/**
 * Une cible proposée : la borne basse de la bande courante, soit l'entrée dans
 * la suivante — un palier que les joueurs se fixent. Depuis la meilleure
 * bande, un simple cran.
 * @param {number} currentPR
 */
export function suggestTarget(currentPR) {
    if (!Number.isFinite(currentPR) || currentPR <= 0) return null;
    const band = bandForPR(currentPR);
    if (band.min > 0) return band.min;
    // Déjà dans la meilleure bande : la seule cible honnête est un cran de
    // mieux que maintenant, arrondi au dixième.
    return Math.max(0.1, Math.round((currentPR - 0.5) * 10) / 10);
}

/**
 * La tendance d'une série chronologique de PR, par moindres carrés : pente par
 * point et valeur projetée à l'échéance. Moins de trois points : null.
 *
 * @param {number[]} series
 * @param {number} pointsAhead combien de points séparent le dernier de l'échéance
 * @returns {{slope: number, projected: number} | null}
 */
export function trend(series, pointsAhead) {
    const points = (series || []).filter((v) => Number.isFinite(v) && v > 0);
    if (points.length < 3) return null;

    const n = points.length;
    const meanX = (n - 1) / 2;
    const meanY = points.reduce((a, b) => a + b, 0) / n;
    let num = 0;
    let den = 0;
    for (let i = 0; i < n; i++) {
        num += (i - meanX) * (points[i] - meanY);
        den += (i - meanX) * (i - meanX);
    }
    if (den === 0) return null;
    const slope = num / den;
    const intercept = meanY - slope * meanX;
    const projected = slope * (n - 1 + pointsAhead) + intercept;
    return { slope, projected: Math.max(0, projected) };
}

/** Le nom de bande d'un PR, pour dire une cible en mots. */
export function bandKeyForPR(pr) {
    return bandForPR(pr).key;
}

export { GRADE_BANDS };
