const focusableSelector = 'a[href], button:not([disabled]), textarea, input:not([disabled]), select, [tabindex]:not([tabindex="-1"])';

// A hidden match (an input in a collapsed section, a `hidden` panel kept mounted) still satisfies
// focusableSelector. `display` does not inherit, so every ancestor up to `node` is checked.
// Not `offsetParent === null`: it also flips for `position: fixed` elements, and jsdom always
// reports null, which would drop every element in the tests.
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
