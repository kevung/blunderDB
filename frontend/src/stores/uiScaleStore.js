import { writable } from 'svelte/store';
import { GetUIScale, SaveUIScale } from '../../wailsjs/go/main/Config.js';
import { logger } from '../utils/logger.js';

// Interface scale as a percentage. The whole UI (toolbar icons, fonts, panels,
// modals and the SVG board) is rendered at UI_SCALE% of its native size via the
// `--ui-scale` CSS custom property and `zoom` on the main container. Bounds and
// default mirror config.go (MinUIScale/MaxUIScale/DefaultUIScale).
export const MIN_UI_SCALE = 50;
export const MAX_UI_SCALE = 200;
export const DEFAULT_UI_SCALE = 100;
export const UI_SCALE_STEP = 10;

export const uiScaleStore = writable(DEFAULT_UI_SCALE);

// Coerce any value into a valid, integral percentage within bounds.
function sanitize(scale) {
    const n = Math.round(Number(scale));
    if (!Number.isFinite(n) || n === 0) return DEFAULT_UI_SCALE;
    return Math.min(MAX_UI_SCALE, Math.max(MIN_UI_SCALE, n));
}

// Push the scale into the DOM via the CSS variable consumed by .main-container.
// This is cheap (a single reflow) and is safe to call on every slider tick.
function applyVar(scale) {
    if (typeof document === 'undefined') return;
    document.documentElement.style.setProperty('--ui-scale', String(scale / 100));
}

// Ask the board to re-fit: two.js sizes itself from its container's client box,
// which changes with zoom. This is expensive, so only call it when the scale is
// committed (e.g. on slider release), not on every intermediate drag value.
function requestBoardRefit() {
    if (typeof window !== 'undefined') {
        window.dispatchEvent(new Event('resize'));
    }
}

// Load the persisted scale at startup. Falls back to the default on any error.
export async function initUIScale() {
    let scale = DEFAULT_UI_SCALE;
    try {
        scale = sanitize(await GetUIScale());
    } catch (err) {
        logger.error('Failed to load UI scale, using default:', err);
    }
    uiScaleStore.set(scale);
    applyVar(scale);
    requestBoardRefit();
}

// Live preview while dragging: update the store + CSS zoom only. The whole UI
// (board SVG included) scales via `zoom` for instant feedback, but the board is
// not re-fitted and nothing is persisted until the value is committed.
export function previewUIScale(scale) {
    const next = sanitize(scale);
    uiScaleStore.set(next);
    applyVar(next);
}

// Commit the interface scale: apply it, re-fit the board once and persist it.
export function setUIScale(scale) {
    const next = sanitize(scale);
    uiScaleStore.set(next);
    applyVar(next);
    requestBoardRefit();
    SaveUIScale(next).catch((err) => logger.error('Failed to save UI scale:', err));
}

// Restore the default scale and persist it.
export function resetUIScale() {
    setUIScale(DEFAULT_UI_SCALE);
}
