// L'onglet affiché du panneau Données de référence. Un store, pas un $state :
// TabbedPanel démonte le panneau quitté, et les commandes `met`, `tp…`, `gv…`
// choisissent l'onglet avant que le panneau ne soit monté.
import { writable } from 'svelte/store';

/** @typedef {'met' | 'takePoints' | 'gammonValues' | 'aggregated' | 'neil'} ReferenceSection */

/** @type {ReadonlyArray<ReferenceSection>} */
export const REFERENCE_SECTIONS = Object.freeze(['met', 'takePoints', 'gammonValues', 'aggregated', 'neil']);

export const referenceSectionStore = writable(/** @type {ReferenceSection} */ ('met'));
