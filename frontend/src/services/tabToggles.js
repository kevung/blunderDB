// tabToggles.js — the "Afficher/cacher" shortcuts (raccourcis.rst, #202).
//
// Extracted from positionService.js (fiche D.10, #210): one of that module's six
// responsibilities, self-contained around one table and one function. The
// `toggleXPanel` names date from when each panel was a floating window; today every
// one of them selects a tab of the tabbed panel (App.svelte's tab effect opens the
// matching PANEL). Re-exported from positionService.js so existing callers
// (keyboardService, commandProcessor, App.svelte, Toolbar.svelte) keep one import.
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
    matches: { tab: 'matches' },
    collections: { tab: 'collections' },
    tournaments: { tab: 'tournaments' },
    stats: { tab: 'stats' },
    search: { tab: 'search', noDbMessage: 'status.searchHistoryRequiresDb' }
});

// The tab to fall back to on a "toggle back" when there is nothing more
// specific to return to yet (app just started, database just opened) — the
// same tab activeTabStore itself starts on.
const DEFAULT_TAB = 'matches';

// The tab that was active right before the last toggleTab() actually
// switched INTO a different one — module-level, on purpose: the eight
// shortcuts below (raccourcis.rst's "Afficher/cacher", #202) share one
// "previous tab" memory rather than one per shortcut, so Ctrl-L then Ctrl-P
// then Ctrl-L again returns to "comments", not to whatever was active before
// Ctrl-L's first press. Read fresh from activeTabStore on every call, so it
// stays correct even when the active tab changed by some other means
// (clicking a tab directly, a match/search/import flow selecting one, …).
let previousTab = null;

/**
 * Select the tab of `id` (a TAB_TOGGLES key) if a database is open — and,
 * unlike a plain "show" action, toggle BACK to whichever tab was active
 * before if `id`'s tab is already the one showing. Every one of these
 * shortcuts is documented as "Afficher/cacher" (show/hide); before this,
 * pressing one a second time was a no-op (#202).
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

export const toggleAnalysisPanel = () => toggleTab('analysis');
export const toggleCommentPanel = () => toggleTab('comments');
// Bound to the `meta` command and Ctrl+M (a tab, not a modal).
export const toggleMetadataPanel = () => toggleTab('metadata');
export const toggleAnkiPanel = () => toggleTab('anki');
export const toggleMatchPanel = () => toggleTab('matches');
export const toggleCollectionPanelAction = () => toggleTab('collections');
export const toggleTournamentPanel = () => toggleTab('tournaments');
export const toggleStatsPanel = () => toggleTab('stats');
export const toggleSearchPanel = () => toggleTab('search');

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
