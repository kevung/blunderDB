import { writable } from 'svelte/store';

// EPC data store: holds the current EPC computation results.
// {
//   bottomEPC: EPCResult | null,   // bottom player's EPC block (always exact)
//   topEPC:    EPCResult | null,   // top player's EPC block (always exact)
//   race:      RaceEval | null,    // race zone (pure bearoff only):
//                                  //   { regime: 'exact'|'estimated', on_roll,
//                                  //     source_checkers, win_prob, sigma, p99,
//                                  //     money?: { cube_state, cubeless, no_double,
//                                  //               double_take, double_pass, verdict } }
//   error:     string | null
// }
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

// Which zones the user has revealed since the last edit. Reset by updateEPC
// (the same code path that recomputes the data, so keyboard edits re-mask too).
export const epcRevealedStore = writable({ bottom: false, top: false, race: false });

export function resetEpcReveal() {
    epcRevealedStore.set({ bottom: false, top: false, race: false });
}
