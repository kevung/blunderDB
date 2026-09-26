/**
 * onChange(getter, fn, initial) — for `$effect`: run `fn(value, previous)` only when `getter()`'s
 * value changed since the last run; `fn` never sees a value twice.
 * `$effect(onChange(() => visible, (opened) => { ... }, false))`. Reads inside `getter` stay
 * tracked (`$effect` tracks every synchronous reactive read).
 * Not for MetadataPanel's `wasActive`: its load runs on every re-run while active and completes
 * asynchronously, so "changed since last run" would skip needed reloads or save before loading.
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
