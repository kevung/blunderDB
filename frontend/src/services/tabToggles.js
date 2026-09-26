// tabToggles.js — the "Afficher/cacher" shortcuts (raccourcis.rst). Each
// `toggleXPanel` selects a tab of the tabbed panel. Re-exported from
// positionService.js.
import { get } from 'svelte/store';
import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore, positionStore } from '../stores/positionStore.js';
import { currentPositionIndexStore, statusBarModeStore, activeTabStore, showPipcountStore } from '../stores/uiStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// ── Tab toggles ──────────────────────────────────────────────────────────────
//
//   tab         the activeTabStore value to select
//   guard       extra precondition (checked only when SWITCHING INTO the
//               tab, never when toggling back out of it), returning a
//               status-bar message key to refuse
//   silent      no "no database" message (metadata: the tab just stays where
//               it is)
//   noDbMessage status-bar message key to use instead of the generic
//               "no database" one when no database is open
const TAB_TOGGLES = Object.freeze({
    analysis: { tab: 'analysis' },
    comments: {
        tab: 'comments',
        guard: () => (positionsStore.idAt(get(currentPositionIndexStore)) != null ? null : 'status.noCurrentPositionComment')
    },
    metadata: { tab: 'metadata', silent: true, guard: () => (get(statusBarModeStore) === 'EDIT' ? 'status.cannotShowMetadataEdit' : null) },
    anki: { tab: 'anki' },
    training: { tab: 'training' },
    matches: { tab: 'matches' },
    collections: { tab: 'collections' },
    tournaments: { tab: 'tournaments' },
    stats: { tab: 'stats' },
    search: { tab: 'search', noDbMessage: 'status.searchHistoryRequiresDb' },
    transcription: { tab: 'transcription' }
});

// The tab to fall back to on a "toggle back" when there is nothing more
// specific to return to yet (app just started, database just opened) — the
// same tab activeTabStore itself starts on.
const DEFAULT_TAB = 'matches';

// The tab active before the last toggleTab() switched away: one memory shared
// by all shortcuts, so Ctrl-L, Ctrl-P, Ctrl-L returns to "comments". The
// current tab is read fresh from activeTabStore, whatever changed it.
let previousTab = null;

/**
 * Select the tab of `id` (a TAB_TOGGLES key) if a database is open, or toggle
 * BACK to the previous tab if it is already showing ("Afficher/cacher").
 */
export function toggleTab(id) {
    const entry = TAB_TOGGLES[id];
    if (!entry) throw new Error(`toggleTab: unknown tab '${id}'`);
    logger.log(`toggleTab ${id}`);
    if (!get(databasePathStore)) {
        if (!entry.silent) setStatusBarMessage(tMsg(entry.noDbMessage ?? 'commands.noDatabaseOpened'));
        return;
    }

    const current = get(activeTabStore);
    if (current === entry.tab) {
        activeTabStore.set(previousTab && previousTab !== entry.tab ? previousTab : DEFAULT_TAB);
        return;
    }

    // The guard only gates entering the tab (e.g. "no current position to
    // comment on") — it never applies to the toggle-back branch above, or
    // leaving a tab would be blocked by the precondition for being on it.
    const refusal = entry.guard?.();
    if (refusal) {
        setStatusBarMessage(tMsg(refusal));
        return;
    }
    previousTab = current;
    activeTabStore.set(entry.tab);
}

/**
 * Sélectionne l'onglet de `id` sans jamais le refermer, pour une commande qui
 * démarre quelque chose dans l'onglet (`train scores`). Rend `false` si un
 * refus (pas de base) l'a empêché.
 * @param {string} id
 */
export function showTab(id) {
    const entry = TAB_TOGGLES[id];
    if (!entry) throw new Error(`showTab: unknown tab '${id}'`);
    if (get(activeTabStore) === entry.tab) return true;
    toggleTab(id);
    return get(activeTabStore) === entry.tab;
}

export const toggleAnalysisPanel = () => toggleTab('analysis');
export const toggleCommentPanel = () => toggleTab('comments');
// Bound to the `meta` command and Ctrl+M (a tab, not a modal).
export const toggleMetadataPanel = () => toggleTab('metadata');
export const toggleAnkiPanel = () => toggleTab('anki');
export const toggleTrainingPanel = () => toggleTab('training');
// Ouvrir, non basculer (cmd_mode.rst) : refermer l'onglet sous une session
// laisserait le chronomètre courir hors écran. Ctrl+J reste la bascule.
export const showTrainingPanel = () => showTab('training');
export const toggleMatchPanel = () => toggleTab('matches');
export const toggleCollectionPanelAction = () => toggleTab('collections');
// Ctrl+Y and `direct`: the round trip between the room and the board. The
// tournament view takes the main area only with this tab active AND a
// Direction open (ADR-0047).
export const toggleTournamentPanel = () => toggleTab('tournaments');
export const toggleStatsPanel = () => toggleTab('stats');
export const toggleSearchPanel = () => toggleTab('search');
// Bound to the `transcribe`/`tr` command and Ctrl+Maj+T.
export const toggleTranscriptionPanel = () => toggleTab('transcription');

export function togglePipcount() {
    logger.log('togglePipcount');
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('commands.noDatabaseOpened'));
        return;
    }
    showPipcountStore.set(!get(showPipcountStore));
    if (get(statusBarModeStore) === 'MATCH') {
        const currentPosition = get(positionStore);
        positionStore.set({ ...currentPosition });
    } else {
        const currentIndex = get(currentPositionIndexStore);
        currentPositionIndexStore.set(-1);
        currentPositionIndexStore.set(currentIndex);
    }
}
