// Panel height remembered per tab: Stats wants a tall dock, Search a short one, and a single
// stored height made each switch undo the last drag. Kept in the viewer's browser: it is a
// layout convenience, not data.
const KEY = 'blunderdb.tabPanelHeights';

function read() {
    try {
        const parsed = JSON.parse(localStorage.getItem(KEY) || '{}');
        return parsed && typeof parsed === 'object' ? parsed : {};
    } catch {
        return {};
    }
}

/** @param {string} tab @returns {number|null} */
export function rememberedTabHeight(tab) {
    const h = read()[tab];
    return Number.isFinite(h) && h > 0 ? h : null;
}

/** @param {string} tab @param {number} height */
export function rememberTabHeight(tab, height) {
    if (!Number.isFinite(height) || height <= 0) return;
    try {
        localStorage.setItem(KEY, JSON.stringify({ ...read(), [tab]: Math.round(height) }));
    } catch {
        // storage blocked: the height simply is not remembered
    }
}
