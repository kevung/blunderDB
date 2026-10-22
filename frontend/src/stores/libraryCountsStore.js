import { writable } from 'svelte/store';

// Le compteur de bibliothèque (« 412 positions · 38 blunders · 5 matchs »), chaque nombre ouvrant
// ce qu'il compte. Rafraîchi quand il peut changer (ouverture, import, suppression), jamais en
// boucle. Le décompte ne balaye jamais une grande table : au-delà d'un seuil le nombre est une
// estimation (« ≈ ») et celui des blunders, qui n'en a pas d'honnête, vient du décompte gardé en
// cache ; une écriture l'efface, et il n'est pas donné (null) le temps qu'un recompte en tâche de
// fond le refasse — la barre d'état relit alors le compteur.

/**
 * @typedef {{positions: number, blunders: number | null, matches: number,
 *   approximate: {positions: boolean, matches: boolean}, blunderThresholdMP: number} | null} LibraryCounts
 */

/** @type {import('svelte/store').Writable<LibraryCounts>} */
export const libraryCountsStore = writable(null);

let refreshSeq = 0;
/** @type {Promise<void> | null} */
let scheduled = null;

/**
 * Rafraîchit le compteur ; sans base ouverte il disparaît plutôt que de garder d'anciens chiffres.
 * Les demandes d'un même tour (l'ouverture d'une base en émet plusieurs : le service, l'effet
 * de la barre d'état) partagent une seule lecture ; une demande venue pendant une lecture en
 * lance une nouvelle, car la base a pu changer entre-temps.
 */
export function refreshLibraryCounts() {
    if (scheduled) return scheduled;
    scheduled = new Promise((resolve) => setTimeout(resolve, 0)).then(() => {
        scheduled = null;
        return readLibraryCounts();
    });
    return scheduled;
}

async function readLibraryCounts() {
    // Une réponse n'est appliquée que si aucun rafraîchissement plus récent n'a démarré.
    const seq = ++refreshSeq;
    const { get } = await import('svelte/store');
    const { databasePathStore } = await import('./databaseStore.js');
    if (!get(databasePathStore)) {
        if (seq === refreshSeq) libraryCountsStore.set(null);
        return;
    }
    try {
        const { GetDatabaseStatsEstimate } = await import('../../wailsjs/go/database/Database.js');
        const { loadLibrarySettings } = await import('../services/librarySettingsService.js');
        const { readOfLibrary } = await import('./groupActivityStore.js');
        const [stats, settings] = await Promise.all([readOfLibrary(() => GetDatabaseStatsEstimate()).then((s) => s || {}), loadLibrarySettings()]);
        if (seq !== refreshSeq) return;
        libraryCountsStore.set({
            positions: Number(stats.position_count || 0),
            blunders: stats.blunder_count == null ? null : Number(stats.blunder_count),
            matches: Number(stats.match_count || 0),
            blunderThresholdMP: settings.blunderThresholdMP,
            approximate: {
                positions: (stats.approximate || []).includes('positions'),
                matches: (stats.approximate || []).includes('matches')
            }
        });
    } catch (error) {
        const { isLibraryLeft } = await import('./groupActivityStore.js');
        // Les compteurs d'une base quittée ne s'affichent pas dans la suivante, qui relit les siens.
        if (isLibraryLeft(error)) return;
        // Un compteur est un confort : s'il échoue, il s'efface au lieu de
        // s'interposer.
        if (seq === refreshSeq) libraryCountsStore.set(null);
    }
}

/** Le nombre tel qu'on l'affiche : « ≈ » devant une estimation, « ? » quand il n'est pas connu. */
export function formatCount(/** @type {number} */ n, approximate = false) {
    if (n == null) return '?';
    return approximate ? `≈ ${n}` : String(n);
}
