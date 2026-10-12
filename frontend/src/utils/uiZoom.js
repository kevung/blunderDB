// The interface scale is a CSS `zoom` on the main container: lengths laid out inside it are
// multiplied by the factor, while event coordinates, getBoundingClientRect and innerWidth stay
// in window pixels. A window-pixel length written back as a CSS length inside the container
// is divided by the factor first, or it lands scale × too far.

/** @returns {number} the current interface zoom factor (1 at 100 %). */
export function uiZoom() {
    if (typeof document === 'undefined') return 1;
    const factor = Number.parseFloat(document.documentElement.style.getPropertyValue('--ui-scale'));
    return Number.isFinite(factor) && factor > 0 ? factor : 1;
}

/**
 * @param {number} px a length in window pixels
 * @returns {number} the same length in the CSS pixels of the zoomed container
 */
export function toCssPx(px) {
    return px / uiZoom();
}
