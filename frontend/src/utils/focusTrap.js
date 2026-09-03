const focusableSelector = 'a[href], button:not([disabled]), textarea, input:not([disabled]), select, [tabindex]:not([tabindex="-1"])';

// A hidden match (an input inside a collapsed section, a `hidden` panel a
// dialog keeps mounted rather than tearing down) still satisfies
// focusableSelector — without this filter Tab could wrap onto, or away from,
// an element nobody can see or reach (#204). `display` does not inherit, so
// an ancestor's `display: none` does not by itself change a descendant's own
// *computed* display — the collapse only shows up by walking up and checking
// every ancestor's own value, which is what this does (stopping at `node`:
// the dialog itself may sit inside further DOM the trap has no business
// judging the visibility of). `offsetParent === null` is the usual one-line
// shortcut for this same question, but it also flips for reasons that do not
// mean "hidden" (an element that is itself `position: fixed`, as `Modal.svelte`'s
// overlay is — though none of its *descendants* are) and, in this project's
// test environment, jsdom does not implement layout and always reports it as
// null, which would make this filter untestable and silently drop every
// element in every trapFocus test.
function isVisible(el) {
    for (let node = el; node instanceof Element; node = node.parentElement) {
        const style = getComputedStyle(node);
        if (style.display === 'none' || style.visibility === 'hidden') return false;
    }
    return true;
}

function focusableIn(node) {
    return [...node.querySelectorAll(focusableSelector)].filter(isVisible);
}

export function trapFocus(node) {
    const previouslyFocused = document.activeElement;

    function handleKeydown(e) {
        if (e.key !== 'Tab') return;
        const focusable = focusableIn(node);
        if (focusable.length === 0) return;
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (e.shiftKey && document.activeElement === first) {
            e.preventDefault();
            last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
            e.preventDefault();
            first.focus();
        }
    }

    node.addEventListener('keydown', handleKeydown);

    // Nothing focusable inside (a table, a plain message): the node itself takes the
    // focus when it can, so the keys pressed on the dialog still bubble through it
    // rather than landing on <body>.
    const first = focusableIn(node)[0];
    if (first) first.focus();
    else if (node.hasAttribute('tabindex')) node.focus();

    return {
        destroy() {
            node.removeEventListener('keydown', handleKeydown);
            if (previouslyFocused && previouslyFocused.focus) {
                previouslyFocused.focus();
            }
        }
    };
}
