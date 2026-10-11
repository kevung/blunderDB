import { writable } from 'svelte/store';

// gammonNet batch progress: { done, total } while running, { job } while queued behind another
// job, null when idle. Fed by StatusBar's listeners on gammonnet-batch:*.
export const gammonNetBatchStore = writable(null);
