import { writable, derived } from 'svelte/store';
import { GetPanelPosition, SavePanelPosition } from '../../wailsjs/go/main/Config.js';
import { logger } from '../utils/logger.js';

// Where the tabbed panel lives relative to the board. `bottom` is the historical
// default (full-width horizontal band); `side` pins it as a vertical column to
// the right of the board so the board can grow to fill the otherwise-wasted
// horizontal space on wide screens; `auto` picks between the two from the window
// aspect ratio. Values mirror config.go (PanelPosition* constants).
export const PANEL_BOTTOM = 'bottom';
export const PANEL_SIDE = 'side';
export const PANEL_AUTO = 'auto';
export const DEFAULT_PANEL_POSITION = PANEL_BOTTOM;

// Auto-mode hysteresis on the window aspect ratio (width / height). The board
// keeps a fixed shape (height ≈ 0.72 × width), so horizontal space starts being
// wasted once the available area is wider than 1 / 0.72 ≈ 1.39:1 — that is the
// point where docking the panel to the side (and letting the board grow) pays
// off. The thresholds sit around that crossover; the dead-band between them
// prevents flapping when the window hovers near a single value. (The window is
// a touch wider than the board area once the toolbar/status bar are subtracted,
// so these are deliberately a hair below 1.39.)
const AUTO_TO_SIDE_ASPECT = 1.45;
const AUTO_TO_BOTTOM_ASPECT = 1.3;

function sanitize(pos) {
    return pos === PANEL_SIDE || pos === PANEL_AUTO ? pos : PANEL_BOTTOM;
}

// The raw, user-selected mode (bottom | side | auto).
export const panelPositionStore = writable(DEFAULT_PANEL_POSITION);

function currentAspect() {
    if (typeof window === 'undefined' || !window.innerHeight) return 1;
    return window.innerWidth / window.innerHeight;
}

// Window aspect ratio, refreshed on resize. Svelte's writable dedupes via
// safe_not_equal, so re-setting the same number (e.g. when a board re-fit
// dispatches a synthetic 'resize' that doesn't change the window size) does not
// notify subscribers — this is what keeps the layout effect from looping.
const windowAspectStore = writable(currentAspect());
if (typeof window !== 'undefined') {
    window.addEventListener('resize', () => windowAspectStore.set(currentAspect()));
}

// The *effective* position actually applied to the layout: `auto` collapses to
// `side`/`bottom` with hysteresis. `lastAutoResolved` is remembered across
// recomputations so we only flip once a threshold is crossed.
let lastAutoResolved = DEFAULT_PANEL_POSITION;
export const effectivePositionStore = derived([panelPositionStore, windowAspectStore], ([pos, aspect]) => {
    if (pos !== PANEL_AUTO) return pos;
    if (aspect >= AUTO_TO_SIDE_ASPECT) lastAutoResolved = PANEL_SIDE;
    else if (aspect <= AUTO_TO_BOTTOM_ASPECT) lastAutoResolved = PANEL_BOTTOM;
    return lastAutoResolved;
});

// Load the persisted mode at startup. Falls back to the default on any error.
// The board re-fit is driven by App.svelte's effect on effectivePositionStore.
export async function initPanelPosition() {
    let pos = DEFAULT_PANEL_POSITION;
    try {
        pos = sanitize(await GetPanelPosition());
    } catch (err) {
        logger.error('Failed to load panel position, using default:', err);
    }
    panelPositionStore.set(pos);
}

// Commit a new mode: update the store and persist it. The layout reflow and the
// board re-fit follow reactively from effectivePositionStore.
export function setPanelPosition(pos) {
    const next = sanitize(pos);
    panelPositionStore.set(next);
    SavePanelPosition(next).catch((err) => logger.error('Failed to save panel position:', err));
}
