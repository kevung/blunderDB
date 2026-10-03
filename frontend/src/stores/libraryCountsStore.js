import { writable } from 'svelte/store';

// Le compteur de bibliothèque (« 412 positions · 38 blunders · 5 matchs »), chaque nombre ouvrant
// ce qu'il compte. Rafraîchi quand il peut changer (ouverture, import, suppression), jamais en
// boucle. Le décompte ne balaye jamais une grande table : au-delà d'un seuil le nombre est une
// estimation (« ≈ ») et celui des blunders, qui n'en a pas d'honnête, n'est pas donné (null).

/**
 * @typedef {{positions: number, blunders: number | null, matches: number,
 *   approximate: {positions: boolean, matches: boolean}, blunderThresholdMP: number} | null} LibraryCounts
 */

/** @type {import('svelte/store').Writable<LibraryCounts>} */
export const libraryCountsStore = writable(null);

/** Rafraîchit le compteur ; sans base ouverte il disparaît plutôt que de garder d'anciens chiffres. */
export async function refreshLibraryCounts() {
    const { get } = await import('svelte/store');
    const { databasePathStore } = await import('./databaseStore.js');
    if (!get(databasePathStore)) {
        libraryCountsStore.set(null);
        return;
    }
    try {
        const { GetDatabaseStatsEstimate } = await import('../../wailsjs/go/database/Database.js');
        const { loadLibrarySettings } = await import('../services/librarySettingsService.js');
        const [stats, settings] = await Promise.all([GetDatabaseStatsEstimate().then((s) => s || {}), loadLibrarySettings()]);
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
    } catch {
        // Un compteur est un confort : s'il échoue, il s'efface au lieu de
        // s'interposer.
        libraryCountsStore.set(null);
    }
}

/** Le nombre tel qu'on l'affiche : « ≈ » devant une estimation, « ? » quand il n'est pas connu. */
export function formatCount(n, approximate = false) {
    if (n == null) return '?';
    return approximate ? `≈ ${n}` : String(n);
}
