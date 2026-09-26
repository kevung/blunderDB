import { GetLibrarySettings, SaveLibrarySettings } from '../../wailsjs/go/database/Database.js';
import { logger } from '../utils/logger.js';

// Les réglages de la bibliothèque (ADR-0046) : les seuils d'erreur et de
// blunder. Dans la bibliothèque, pas la machine : même compte partout, lu par
// `blunderdb info` et par tenant du démon. Stockés en millipoints (l'unité de
// `E>x`), montrés en équité ; la conversion tient ici.

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
 * Équité → millipoints, arrondi à l'entier (une saisie plus fine est arrondie,
 * pas refusée).
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
 * Écrit les seuils. Le stockage refuse une paire inversée ; l'erreur remonte
 * à l'appelant pour affichage.
 * @param {LibrarySettings} settings
 */
export async function saveLibrarySettings(settings) {
    await SaveLibrarySettings({
        errorThresholdMP: settings.errorThresholdMP,
        blunderThresholdMP: settings.blunderThresholdMP
    });
    return settings;
}
