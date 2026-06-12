import { driver } from 'driver.js';
import 'driver.js/dist/driver.css';
import { translate } from '../i18n';
import { openModal, closeModal, MODAL, activeTabStore } from '../stores/uiStore';
import { GetTourSeen, SaveTourSeen } from '../../wailsjs/go/main/Config.js';
import { TOURS, getTourById } from '../tours.js';
import { logger } from '../utils/logger.js';

export { TOURS };

// Build driver.js steps from a tour definition, resolving i18n at start time
// (a tour is short-lived, the language won't change mid-tour). Steps whose
// target element is not in the DOM are skipped so the tour never points at
// nothing.
function buildSteps(tour) {
    return tour.steps
        .filter((s) => !s.element || s.activateTab || document.querySelector(s.element))
        .map((s) => ({
            element: s.element || undefined,
            // Steps may activate a tab before highlighting, so the relevant panel
            // (e.g. Search) is actually visible under the spotlight.
            onHighlightStarted: s.activateTab ? () => activeTabStore.set(s.activateTab) : undefined,
            popover: {
                title: translate(s.titleKey),
                description: translate(s.bodyKey),
                side: s.side || 'bottom',
                align: s.align || 'center'
            }
        }));
}

/** Start a guided tour by id. Closes any open modal first. */
export function startTour(tourId) {
    const tour = getTourById(tourId);
    if (!tour) {
        logger.error('Unknown tour:', tourId);
        return;
    }
    closeModal();
    const steps = buildSteps(tour);
    if (steps.length === 0) return;
    const d = driver({
        showProgress: true,
        // driver.js replaces {{current}}/{{total}} itself — keep it i18n-neutral.
        progressText: '{{current}} / {{total}}',
        nextBtnText: translate('tour.next'),
        prevBtnText: translate('tour.prev'),
        doneBtnText: translate('tour.done'),
        // Tighter, crisper highlight cut-out (the defaults leave a loose gap that
        // looks untidy around full-width bars).
        stagePadding: 4,
        stageRadius: 6,
        overlayColor: '#000000',
        overlayOpacity: 0.6,
        disableActiveInteraction: true,
        smoothScroll: true,
        steps
    });
    d.drive();
}

/**
 * On first launch, show the tour catalog once and remember it so it never
 * auto-opens again. Failures (e.g. config read) are swallowed — the tour is
 * non-essential and must never block startup.
 */
export async function maybeRunFirstRunTour() {
    try {
        const seen = await GetTourSeen();
        if (!seen) {
            openModal(MODAL.TOUR);
            await SaveTourSeen(true);
        }
    } catch (err) {
        logger.error('First-run tour check failed:', err);
    }
}
