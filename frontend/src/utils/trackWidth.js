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

/**
 * Like trackWidth, for a chart that also takes the height its container
 * gives it.
 *
 * @param {HTMLElement} node
 * @param {(width: number, height: number) => void} onsize
 */
export function trackSize(node, onsize) {
    let report = onsize;
    report(node.clientWidth, node.clientHeight);
    if (typeof ResizeObserver === 'undefined') return { update: (/** @type {(width: number, height: number) => void} */ fn) => (report = fn) };
    const observer = new ResizeObserver(() => report(node.clientWidth, node.clientHeight));
    observer.observe(node);
    return {
        update: (/** @type {(width: number, height: number) => void} */ fn) => (report = fn),
        destroy: () => observer.disconnect()
    };
}
