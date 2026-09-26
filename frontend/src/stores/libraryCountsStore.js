import { writable } from 'svelte/store';

// Le compteur de bibliothèque (« 412 positions · 38 blunders · 5 matchs »), chaque nombre ouvrant
// ce qu'il compte. Rafraîchi quand il peut changer (ouverture, import, suppression), jamais en
// boucle : trois COUNT sur cent mille positions coûteraient tout à chaque frappe.

/** @typedef {{positions: number, blunders: number, matches: number} | null} LibraryCounts */

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
        const { GetDatabaseStats } = await import('../../wailsjs/go/database/Database.js');
        const stats = (await GetDatabaseStats()) || {};
        libraryCountsStore.set({
            positions: Number(stats.position_count || 0),
            blunders: Number(stats.blunder_count || 0),
            matches: Number(stats.match_count || 0)
        });
    } catch {
        // Un compteur est un confort : s'il échoue, il s'efface au lieu de
        // s'interposer.
        libraryCountsStore.set(null);
    }
}
