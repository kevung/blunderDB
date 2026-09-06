import { LoadMetadata, SaveMetadata } from '../../wailsjs/go/database/Database.js';
import { GRADE_BANDS, bandForPR } from '../components/stats/gradeBands.js';
import { logger } from '../utils/logger.js';

// Les objectifs de progression (#274, fiche I.18).
//
// « PR < 5 d'ici douze semaines » : une cible, une échéance, et une tendance
// qui dit où l'on va. Rien de plus — un objectif qui se met à noter, à
// féliciter ou à rappeler serait une autre fonctionnalité, et pas celle-ci.
//
// L'objectif vit dans les MÉTADONNÉES de la base, pas dans la configuration :
// il porte sur cette bibliothèque-là, et suit donc le fichier plutôt que la
// machine. Aucun schéma n'est touché — `metadata` est déjà une table de
// clés/valeurs, et le stockage est déjà exposé aux trois modes, donc la ligne
// de commande et le démon lisent l'objectif sans qu'on leur ajoute quoi que ce
// soit.

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
 * Écrit l'objectif. Les autres métadonnées sont relues et réécrites : la
 * table est un dictionnaire unique, et l'enregistrer partiellement effacerait
 * ce qu'on n'a pas relu — le nom de l'utilisateur, par exemple.
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
 * Une cible proposée depuis le niveau actuel : la borne basse de la bande
 * courante, c'est-à-dire l'entrée dans la bande suivante.
 *
 * Proposer « un peu mieux » n'aurait aucun sens ancré ; proposer une bande
 * en dit une : passer d'intermédiaire à avancé se voit, se raconte, et
 * correspond à ce qu'un joueur se fixe réellement. Depuis la meilleure bande,
 * il n'y a plus de palier à viser et la proposition se contente d'un cran.
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
 * La tendance sur une série de PR, par les moindres carrés : la pente par
 * point et la valeur projetée à l'échéance.
 *
 * `series` est une liste de nombres dans l'ordre chronologique. Moins de trois
 * points ne fait pas une tendance, et le dire vaut mieux que tracer une droite
 * entre deux tournois.
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
