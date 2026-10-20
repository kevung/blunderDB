import { writable } from 'svelte/store';

// gammonNet batch progress: { done, total } while running, { job } while queued behind another
// job, null when idle. Fed by StatusBar's listeners on gammonnet-batch:*.
/** @type {import('svelte/store').Writable<{ done?: number, total?: number, job?: string, group?: number | string } | null>} */
export const gammonNetBatchStore = writable(null);
