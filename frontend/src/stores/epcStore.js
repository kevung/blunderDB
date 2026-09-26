import { writable } from 'svelte/store';

// What the Eval panel last computed: bottomEPC / topEPC (exact EPC blocks), race (pure bearoff
// only: { regime, on_roll, source_checkers, win_prob, sigma, p99, money? }), error. Typed loosely:
// the panel adds its own fields beside the generated engine.EPCResult.
/** @type {import('svelte/store').Writable<{bottomEPC: any, topEPC: any, race: any, error: any, [field: string]: any}>} */
export const epcDataStore = writable({
    bottomEPC: null,
    topEPC: null,
    race: null,
    error: null
});

// Challenge ("défi") training mode: when on, every edit re-masks the three
// panel zones and the user reveals them one by one by clicking. Persisted via
// Config.SaveEpcChallenge; initialised from Config at startup.
export const epcChallengeStore = writable(false);

// Zones revealed since the last edit: bottom, top, and the one decision block (ADR-0017). Reset by
// updateEPC, so keyboard edits re-mask too.
export const epcRevealedStore = writable({ bottom: false, top: false, decision: false });

export function resetEpcReveal() {
    epcRevealedStore.set({ bottom: false, top: false, decision: false });
}
