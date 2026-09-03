// A Svelte action that focuses its node once, on mount — a drop-in
// replacement for the HTML `autofocus` attribute (#205, svelte's
// a11y_autofocus warning). Every one of this project's `autofocus` uses was
// on an element that mounts fresh through an `{#if editing}` block (an
// inline-edit input appearing over a row) rather than on page load, which is
// exactly the risk the HTML attribute carries and the compiler is warning
// about: a browser applies `autofocus` unconditionally as soon as the
// element is parsed, which can silently steal focus a screen reader user (or
// a different in-page flow) had already moved elsewhere, and two autofocus
// elements present at once make an arbitrary browser-picked winner. A Svelte
// action instead runs exactly once, exactly when this particular element is
// created — the behaviour every call site actually wanted — with no
// ambiguity about which element wins.
//
// Usage: `<input use:autofocus />` in place of `<input autofocus />`.
export function autofocus(node) {
    node.focus();
}
