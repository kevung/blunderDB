import { writable } from 'svelte/store';

// gammonNet batch progress: { done, total } while running, null when idle. Fed by StatusBar's
// listeners on gammonnet-batch:progress/done/cancelled/error.
export const gammonNetBatchStore = writable(null);
