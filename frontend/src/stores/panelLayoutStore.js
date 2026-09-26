import { writable, derived } from 'svelte/store';
import { GetPanelPosition, SavePanelPosition, GetPanelHeight, SavePanelHeight, GetPanelWidth, SavePanelWidth } from '../../wailsjs/go/main/Config.js';
import { logger } from '../utils/logger.js';

// Where the tabbed panel lives: `bottom` (full-width band), `side` (column right of the board,
// letting it grow on wide screens), `auto` (from the window aspect). Mirrors config.go.
export const PANEL_BOTTOM = 'bottom';
export const PANEL_SIDE = 'side';
export const PANEL_AUTO = 'auto';
export const DEFAULT_PANEL_POSITION = PANEL_BOTTOM;

// Auto-mode hysteresis on width / height. The board is ≈ 0.72 × as tall as wide, so horizontal
// space is wasted beyond ≈ 1.39:1; the dead-band around it prevents flapping (a hair below,
// since toolbar and status bar are subtracted from the window).
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

// Window aspect ratio, refreshed on resize. writable dedupes (safe_not_equal), so a synthetic
// 'resize' from a board re-fit does not notify — that keeps the layout effect from looping.
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

// Panel size in px: height in bottom mode, width in side mode. Must equal config.go's
// DefaultPanelHeight/Width, or the panel jumps on first launch (panelDefaults.sync.test.js).
export const DEFAULT_PANEL_HEIGHT = 250;
export const DEFAULT_PANEL_WIDTH = 420;

export const panelHeightStore = writable(DEFAULT_PANEL_HEIGHT);
export const panelWidthStore = writable(DEFAULT_PANEL_WIDTH);

// Load the persisted panel size at startup. App.svelte owns the size as local
// $state during an active drag (continuous mousemove updates would be wasted
// churn on a store); this only seeds that local state once, at launch.
export async function initPanelSize() {
    try {
        panelHeightStore.set(await GetPanelHeight());
    } catch (err) {
        logger.error('Failed to load panel height, using default:', err);
    }
    try {
        panelWidthStore.set(await GetPanelWidth());
    } catch (err) {
        logger.error('Failed to load panel width, using default:', err);
    }
}

// Persist the panel size reached at the end of a resize-handle drag (called
// on mouseup, not on every intermediate pixel — see App.svelte).
export function savePanelHeight(height) {
    panelHeightStore.set(height);
    SavePanelHeight(height).catch((err) => logger.error('Failed to save panel height:', err));
}

export function savePanelWidth(width) {
    panelWidthStore.set(width);
    SavePanelWidth(width).catch((err) => logger.error('Failed to save panel width:', err));
}
