import { writable, get } from 'svelte/store';
import { FillMatchStatsInBackground } from '../../wailsjs/go/database/Database.js';

// The maintenance pass in flight: { kind, done, total } while it runs, null when idle.
// Fed by StatusBar's listeners on maintenance:*, the only ones always mounted.
export const maintenanceStore = writable(/** @type {{ kind: string, done: number, total: number } | null} */ (null));

// Bumped when a pass ends, whatever its outcome: what shows a count reads it again.
export const maintenanceEndedStore = writable(0);

/** The line whose rows the background fill computes. */
export const STATS_MATCHES = 'stats.matches';

/**
 * Asks the backend to fill the library's missing match statistics in the background, once a
 * library is brought forward. The fill reports as the stats.matches line running, so what waits
 * for those rows waits on maintenanceStore. A line left from another library's fill is dropped:
 * the backend stopped that fill when the file changed.
 */
export async function startMatchStatsFill() {
    if (get(maintenanceStore)?.kind === STATS_MATCHES) maintenanceStore.set(null);
    const started = await FillMatchStatsInBackground().then(Boolean, () => false);
    if (started && !get(maintenanceStore)) maintenanceStore.set({ kind: STATS_MATCHES, done: 0, total: 0 });
}
