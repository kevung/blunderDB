import { writable } from 'svelte/store';

// Le compteur de bibliothèque (#287, fiche I.31).
//
// « 412 positions · 38 blunders · 5 matchs » : ce que la base contient, lisible
// d'un coup d'œil, et chaque nombre ouvre ce qu'il compte. Un chiffre qu'on ne
// peut pas suivre est une décoration.
//
// Le compte est rafraîchi aux moments où il peut avoir changé — ouverture d'une
// base, import, suppression — jamais en boucle : trois COUNT sur une
// bibliothèque de cent mille positions ne coûtent rien une fois, et coûteraient
// tout à chaque frappe.

/** @typedef {{positions: number, blunders: number, matches: number} | null} LibraryCounts */

/** @type {import('svelte/store').Writable<LibraryCounts>} */
export const libraryCountsStore = writable(null);

/**
 * Rafraîchit le compteur. Sans base ouverte, il disparaît plutôt que de rester
 * sur les chiffres de la précédente.
 */
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
