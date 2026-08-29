import { writable } from 'svelte/store';

// gammonNet batch progress (#129): { done, total } while running, null when
// idle. Fed by StatusBar.svelte's EventsOn listeners on
// gammonnet-batch:progress/done/cancelled/error — the batch itself
// (internal/gui/gammonnet_batch.go) has no concept of a frontend store, it
// only emits Wails events.
export const gammonNetBatchStore = writable(null);
