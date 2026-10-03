// pageStepSetting.js — the persisted PageUp / PageDown step (Settings > Interface).
import { GetPageStep, SavePageStep } from '../../wailsjs/go/main/Config.js';
import { pageStepStore, PAGE_STEPS, PAGE_STEP_DEFAULT } from '../stores/uiStore.js';
import { logger } from '../utils/logger.js';

/** @param {unknown} step @returns {string} */
function sanitize(step) {
    return PAGE_STEPS.includes(/** @type {string} */ (step)) ? /** @type {string} */ (step) : PAGE_STEP_DEFAULT;
}

/** Loads the persisted step into pageStepStore. */
export async function initPageStep() {
    try {
        pageStepStore.set(sanitize(await GetPageStep()));
    } catch (err) {
        logger.error('could not read the page step:', err);
    }
}

/**
 * Chooses the step: applies it at once and persists it.
 *
 * @param {string} step - one of PAGE_STEPS.
 */
export function setPageStep(step) {
    const next = sanitize(step);
    pageStepStore.set(next);
    SavePageStep(next).catch((err) => logger.error('could not save the page step:', err));
}
