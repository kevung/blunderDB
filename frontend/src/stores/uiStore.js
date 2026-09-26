import { writable, derived } from 'svelte/store';

import { trainingPipOverrideStore } from './trainingTabStore.js';

/**
 * Le texte de la barre d'état : une chaîne traduite ou un descripteur `tMsg()`, retraduit à
 * chaque changement de langue.
 * @type {import('svelte/store').Writable<string | import('../i18n').StatusMessage>}
 */
export const statusBarTextStore = writable('');
export const statusBarModeStore = writable('NORMAL');

export const commandTextStore = writable('');

export const commentTextStore = writable('');

// Active tab in the bottom panel ('analysis', 'comments', 'filter-library', 'search', 'search-history', 'collections', 'matches', 'tournaments')
export const activeTabStore = writable('matches');

// Whether the command input is active in the status bar
export const showCommandInputStore = writable(false);

// Whether the command palette (Ctrl+Maj+P) is open — CommandPalette.svelte.
export const commandPaletteOpenStore = writable(false);

// A match someone asked to open from outside the match panel (the command
// palette): its id, until MatchPanel has loaded its list and opened it.
/** @type {import('svelte/store').Writable<number | null>} */
export const matchOpenRequestStore = writable(null);

export const currentPositionIndexStore = writable(0);

// ── Modal identifiers (exclusive — only one modal at a time) ──
export const MODAL = {
    MET: 'met',
    CUBE_MATRIX: 'cubeMatrix',
    TAGS: 'tags',
    TAKE_POINT_2_LAST: 'takePoint2Last',
    TAKE_POINT_2_LIVE: 'takePoint2Live',
    TAKE_POINT_4_LAST: 'takePoint4Last',
    TAKE_POINT_4_LIVE: 'takePoint4Live',
    GAMMON_VALUE_1: 'gammonValue1',
    GAMMON_VALUE_2: 'gammonValue2',
    GAMMON_VALUE_4: 'gammonValue4',
    WARNING: 'warning',
    PROTECTED_COPY: 'protectedCopy',
    GO_TO_POSITION: 'goToPosition',
    EXPORT_DATABASE: 'exportDatabase',
    TAKE_POINT_2: 'takePoint2',
    TAKE_POINT_4: 'takePoint4',
    HELP: 'help',
    COMMAND: 'command',
    CONFIG: 'config',
    TOUR: 'tour',
    TRASH: 'trash',
    LOG: 'log',
    CONTACT_SHEET: 'contactSheet'
};

// ── Panel identifiers (can be open simultaneously) ──
// No ANALYSIS or COMMENT entry: those tabs read `$activeTabStore` directly, TabbedPanel's
// mount/destroy on every switch being their open/close signal.
export const PANEL = {
    MATCH: 'match',
    COLLECTION: 'collection',
    TOURNAMENT: 'tournament',
    STATS: 'stats'
};

// ── Single modal store (only one modal at a time) ──
/** @type {import('svelte/store').Writable<string | null>} */
export const activeModal = writable(null);

// Initial tab requested for the Config modal (e.g. the Eval panel's download
// hint opens straight onto the Bearoff tab). Consumed once by ConfigModal.
export const configInitialTabStore = writable(null);

// ── Panel set (multiple panels can be open) ──
/** @type {import('svelte/store').Writable<Set<string>>} */
export const openPanels = writable(new Set());

// ── Modal helpers ──
/** @param {string} name */
export function openModal(name) {
    activeModal.set(name);
}
export function closeModal() {
    activeModal.set(null);
}
/** @param {string} name */
export function toggleModal(name) {
    activeModal.update((current) => (current === name ? null : name));
}

// ── Panel helpers ──
/** @param {string} name */
export function openPanel(name) {
    openPanels.update((s) => {
        const next = new Set(s);
        next.add(name);
        return next;
    });
}
/** @param {string} name */
export function closePanel(name) {
    openPanels.update((s) => {
        const next = new Set(s);
        next.delete(name);
        return next;
    });
}
/** @param {string} name */
export function togglePanel(name) {
    openPanels.update((s) => {
        const next = new Set(s);
        if (next.has(name)) next.delete(name);
        else next.add(name);
        return next;
    });
}

// ── Derived stores (automatic — no manual enumeration) ──
export const isAnyModalOpen = derived(activeModal, ($m) => $m !== null);

export const matchPanelRefreshTriggerStore = writable(0);

// Incremented after any DB mutation that can affect stats (import, delete match,
// delete position, save analysis). Reset automatically when a new database is opened
// via databasePathStore (see statsStore.js statsInvalidationKeyStore).
export const dbMutationCounterStore = writable(0);

export const positionReloadTriggerStore = writable(0);

export const showPipcountStore = writable(true);

/**
 * Le pipcount est-il visible ? La préférence (`showPipcountStore`, touche `p`), sauf pendant une
 * question de Pions qui la surcharge. Le plateau s'y abonne pour repeindre : `drawBoard()` lit
 * sa valeur impérativement, et une visibilité changée sans repaint ne changerait rien à l'écran.
 */
export const pipcountVisibleStore = derived([showPipcountStore, trainingPipOverrideStore], ([$preference, $override]) => ($override === null ? $preference : $override));
