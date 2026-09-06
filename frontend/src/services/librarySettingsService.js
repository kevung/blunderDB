import { GetLibrarySettings, SaveLibrarySettings } from '../../wailsjs/go/database/Database.js';
import { logger } from '../utils/logger.js';

// Les réglages de la bibliothèque (ADR-0043) : les deux seuils qui décident de
// ce qu'est une erreur et de ce qu'est un blunder.
//
// Ils vivent dans la bibliothèque, pas dans la configuration de la machine :
// le même fichier compte les mêmes blunders où qu'on l'ouvre, la ligne de
// commande les lit (`blunderdb info`), et un tenant du démon porte les siens.
// Le stockage garde des millipoints — l'unité que parlent `E>x` et
// `--move-error-min` — et l'interface montre l'équité, l'unité de toutes les
// tables. La conversion tient en deux fonctions, ici, une seule fois.

/** @typedef {{errorThresholdMP: number, blunderThresholdMP: number}} LibrarySettings */

/** Les défauts du contrat, répétés ici pour l'affichage sans base ouverte. */
export const DEFAULT_SETTINGS = { errorThresholdMP: 50, blunderThresholdMP: 100 };

// Les seuils natifs des deux moteurs que les utilisateurs connaissent, offerts
// comme des préréglages nommés et non comme la vérité : chacun a tracé sa
// ligne, blunderDB montre laquelle est en vigueur.
export const THRESHOLD_PRESETS = [
    { key: 'blunderdb', errorThresholdMP: 50, blunderThresholdMP: 100 },
    { key: 'xg', errorThresholdMP: 20, blunderThresholdMP: 80 },
    { key: 'gnubg', errorThresholdMP: 40, blunderThresholdMP: 80 }
];

/** Millipoints → équité, l'unité affichée. */
export function mpToEquity(mp) {
    return (Number(mp) || 0) / 1000;
}

/**
 * Équité → millipoints. Arrondi à l'entier : le stockage ne connaît pas le
 * dixième de millipoint, et une saisie à quatre décimales ne doit pas être
 * refusée, seulement arrondie.
 */
export function equityToMP(equity) {
    return Math.round((Number(equity) || 0) * 1000);
}

/** Lit les seuils de la base ouverte, ou les défauts si la lecture échoue. */
export async function loadLibrarySettings() {
    try {
        const s = await GetLibrarySettings();
        return {
            errorThresholdMP: Number(s?.errorThresholdMP ?? DEFAULT_SETTINGS.errorThresholdMP),
            blunderThresholdMP: Number(s?.blunderThresholdMP ?? DEFAULT_SETTINGS.blunderThresholdMP)
        };
    } catch (err) {
        logger.error('could not read the library settings:', err);
        return { ...DEFAULT_SETTINGS };
    }
}

/**
 * Écrit les seuils. Le refus d'une paire inversée vient du stockage, qui la
 * valide comme paire ; l'appelant le laisse remonter pour l'afficher.
 * @param {LibrarySettings} settings
 */
export async function saveLibrarySettings(settings) {
    await SaveLibrarySettings({
        errorThresholdMP: settings.errorThresholdMP,
        blunderThresholdMP: settings.blunderThresholdMP
    });
    return settings;
}
