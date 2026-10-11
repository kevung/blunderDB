import { writable } from 'svelte/store';

// The maintenance pass in flight: { kind, done, total } while it runs, null when idle.
// Fed by StatusBar's listeners on maintenance:*, the only ones always mounted.
export const maintenanceStore = writable(/** @type {{ kind: string, done: number, total: number } | null} */ (null));

// Bumped when a pass ends, whatever its outcome: what shows a count reads it again.
export const maintenanceEndedStore = writable(0);
