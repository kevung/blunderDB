import { get } from 'svelte/store';
import { uiScaleStore, setUIScale, resetUIScale, MIN_UI_SCALE, MAX_UI_SCALE, UI_SCALE_STEP } from '../stores/uiScaleStore.js';
import { setStatusBarMessage } from './databaseService.js';

// The browser zoom chords, on the interface scale the configuration already holds. Registered in
// the capture tier so they work from a text field and under a modal, as in a browser.

/**
 * @param {KeyboardEvent} event
 * @returns {1 | -1 | 0 | null} the step direction, 0 for the reset, null when the key is not a zoom chord.
 */
function zoomAction(event) {
    if (!(event.ctrlKey || event.metaKey) || event.altKey) return null;
    if (event.key === '+' || event.key === '=' || event.code === 'NumpadAdd') return 1;
    if (event.key === '-' || event.key === '_' || event.code === 'NumpadSubtract' || event.code === 'Minus') return -1;
    if (event.key === '0' || event.code === 'Digit0' || event.code === 'Numpad0') return 0;
    return null;
}

/** @param {KeyboardEvent} event */
export function handleZoomKey(event) {
    const action = zoomAction(event);
    if (action === null) return;
    event.preventDefault();
    if (action === 0) {
        resetUIScale();
    } else {
        const current = get(uiScaleStore);
        const next = Math.min(MAX_UI_SCALE, Math.max(MIN_UI_SCALE, current + action * UI_SCALE_STEP));
        if (next !== current) setUIScale(next);
    }
    setStatusBarMessage(`${get(uiScaleStore)} %`);
}
