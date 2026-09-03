/**
 * onChange(getter, fn, initial) — for the common `$effect` shape "run `fn` only when
 * `getter()`'s value actually changed since the last run," with the new value (and
 * the one it replaced) as arguments.
 *
 * Three components (EPCPanel, TournamentPanel, MatchPanel) each kept their own
 * `_prevActive`/`_prevVisible` local and repeated the same three lines around it —
 * remember the previous value, compare, call the side effect only on an actual
 * change, then update the remembered value, in that order, so `fn` never sees a
 * value it has already been called with (fiche D.10, #210).
 *
 * Returns a plain function meant to be passed straight to `$effect`:
 * `$effect(onChange(() => visible, (opened) => { ... }, false))`. Store/rune reads
 * inside `getter` are still tracked as the effect's own dependencies — Svelte's
 * `$effect` tracks every reactive read made synchronously during its callback,
 * wherever in the call graph it happens, so calling through this helper does not
 * hide them.
 *
 * Not a fit for every "previous value" pattern: MetadataPanel's `wasActive` also
 * gates a SAVE on leaving the tab, but its LOAD on entering runs on every effect
 * re-run while the tab stays active (not only on the transition into it) and is
 * marked complete asynchronously, after `loadMetadata()` resolves — collapsing that
 * into "changed since last run" would start skipping reloads a real bug relied on
 * and would let a quick enter-then-leave (before the load promise settles) save
 * before ever having loaded. Left as its own local flag on purpose.
 */
export function onChange(getter, fn, initial) {
    let previous = initial;
    return () => {
        const value = getter();
        if (value !== previous) {
            const old = previous;
            previous = value;
            fn(value, old);
        }
    };
}
