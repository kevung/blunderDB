// A Svelte action reporting an element's width as it changes, so a chart can
// draw at its pixel size. Where ResizeObserver is missing (jsdom), the width is
// read once and the chart keeps its default.

/**
 * @param {HTMLElement} node
 * @param {(width: number) => void} onwidth
 */
export function trackWidth(node, onwidth) {
    let report = onwidth;
    report(node.clientWidth);
    if (typeof ResizeObserver === 'undefined') return { update: (/** @type {(width: number) => void} */ fn) => (report = fn) };
    const observer = new ResizeObserver(() => report(node.clientWidth));
    observer.observe(node);
    return {
        update: (/** @type {(width: number) => void} */ fn) => (report = fn),
        destroy: () => observer.disconnect()
    };
}
