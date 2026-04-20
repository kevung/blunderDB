const focusableSelector = 'a[href], button:not([disabled]), textarea, input:not([disabled]), select, [tabindex]:not([tabindex="-1"])';

export function trapFocus(node) {
    const previouslyFocused = document.activeElement;

    function handleKeydown(e) {
        if (e.key !== 'Tab') return;
        const focusable = [...node.querySelectorAll(focusableSelector)];
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

    const first = node.querySelector(focusableSelector);
    if (first) first.focus();

    return {
        destroy() {
            node.removeEventListener('keydown', handleKeydown);
            if (previouslyFocused && previouslyFocused.focus) {
                previouslyFocused.focus();
            }
        }
    };
}
