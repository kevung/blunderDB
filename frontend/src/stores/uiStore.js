import { writable, derived } from 'svelte/store';

export const statusBarTextStore = writable('');
export const statusBarModeStore = writable('NORMAL');

export const commandTextStore = writable('');

export const commentTextStore = writable('');

// Active tab in the bottom panel ('log', 'analysis', 'comments', 'filter-library', 'search', 'search-history', 'collections', 'matches', 'tournaments')
export const activeTabStore = writable('matches');

// Session log entries: array of { timestamp, type, message }
// type: 'info' (default), 'command', 'result', 'error'
export const logEntriesStore = writable([]);

export function addLogEntry(message, type = 'info') {
    logEntriesStore.update((entries) => {
        const entry = { timestamp: new Date(), type, message };
        return [...entries, entry];
    });
}

// Whether the command input is active in the status bar
export const showCommandInputStore = writable(false);

export const currentPositionIndexStore = writable(0);

// ── Modal identifiers (exclusive — only one modal at a time) ──
export const MODAL = {
    MET: 'met',
    TAKE_POINT_2_LAST: 'takePoint2Last',
    TAKE_POINT_2_LIVE: 'takePoint2Live',
    TAKE_POINT_4_LAST: 'takePoint4Last',
    TAKE_POINT_4_LIVE: 'takePoint4Live',
    GAMMON_VALUE_1: 'gammonValue1',
    GAMMON_VALUE_2: 'gammonValue2',
    GAMMON_VALUE_4: 'gammonValue4',
    WARNING: 'warning',
    GO_TO_POSITION: 'goToPosition',
    EXPORT_DATABASE: 'exportDatabase',
    TAKE_POINT_2: 'takePoint2',
    TAKE_POINT_4: 'takePoint4',
    HELP: 'help',
    COMMAND: 'command',
    CONFIG: 'config',
    TOUR: 'tour'
};

// ── Panel identifiers (can be open simultaneously) ──
export const PANEL = {
    ANALYSIS: 'analysis',
    COMMENT: 'comment',
    SEARCH_HISTORY: 'searchHistory',
    MATCH: 'match',
    COLLECTION: 'collection',
    TOURNAMENT: 'tournament',
    STATS: 'stats'
};

// ── Single modal store (only one modal at a time) ──
export const activeModal = writable(null);

// ── Panel set (multiple panels can be open) ──
export const openPanels = writable(new Set());

// ── Modal helpers ──
export function openModal(name) {
    activeModal.set(name);
}
export function closeModal() {
    activeModal.set(null);
}
export function toggleModal(name) {
    activeModal.update((current) => (current === name ? null : name));
}

// ── Panel helpers ──
export function openPanel(name) {
    openPanels.update((s) => {
        const next = new Set(s);
        next.add(name);
        return next;
    });
}
export function closePanel(name) {
    openPanels.update((s) => {
        const next = new Set(s);
        next.delete(name);
        return next;
    });
}
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
