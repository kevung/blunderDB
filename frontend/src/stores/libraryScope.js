// What each store holds with respect to the open library, for the library groups (ADR-0101).
// databaseLoadedStore stays true from one library to the next, so a store left out of this
// file would carry one library's data into another group. Every exported *Store of src/stores
// is classified here once (libraryScope.test.js holds it):
//
// - photographed: the group's own state, saved when the group is left and put back when it
//   returns, reset when a library is opened in a new group. Editing in progress lives here, so
//   a draft survives a round trip through another group.
// - reloaded: read from the library; emptied when the library changes, then read again by its
//   panel on libraryChangedStore (or on databasePathStore, which differs from one group to
//   the next).
// - refused: a session the switch cannot carry; the switch is refused while it runs.
// - outside: preferences, the board's transient state, jobs and modals that already refuse a
//   switch, or the group machinery itself.

import { get } from 'svelte/store';
import {
    transcriptionStore,
    transcriptionListStore,
    transcriptionHistoryStore,
    transcriptionHistoryActionStore,
    transcriptionInfoStore,
    transcriptionPromptStore,
    transcriptionWheelStore,
    transcriptionKeyStore,
    transcriptionCubeRequestStore,
    transcriptionBoardSwapStore,
    transcriptionArrowsStore,
    transcriptionModeStore
} from './transcriptionStore.js';
import { initialKeyState } from '../services/transcriptionKeys.js';
import { lessonStore, lessonStepIndexStore, lessonDoneStore, lessonEditorTargetStore } from './lessonStore.js';
import { ankiDecksStore, selectedAnkiDeckStore, ankiReviewCardStore, ankiDeckStatsStore, ankiViewModeStore, ankiReviewActionStore, ankiPausedSessionStore, ankiAnswerShownStore } from './ankiStore.js';
import { studyQueueStore, studyQueueIndexStore, studyQueueBacklogStore, studyQueueLastMarkedStore, studyQueueActiveStore } from './studyQueueStore.js';
import { quizPlayStore, quizPlayValidateStore } from './quizPlayStore.js';
import { duelStore, duelListStore, duelFoldedStore, duelBoardStore } from './duelStore.js';
import { collectionsStore, selectedCollectionStore, collectionPositionsStore, activeCollectionStore } from './collectionStore.js';
import { tournamentsStore, selectedTournamentStore, tournamentMatchesStore, tournamentIDsFilterStore } from './tournamentStore.js';
import { searchHistoryStore, lastSearchStore } from './searchHistoryStore.js';
import { filterLibraryStore } from './filterLibraryStore.js';
import { directionSummariesStore } from './directionStore.js';
import { searchExcludePositionStore, emptySearchBoardPosition } from './searchExcludePositionStore.js';
import { searchCriteriaStore, searchEmptyStore } from './searchParamsStore.js';
import { positionBeforeFilterLibraryStore, positionIndexBeforeFilterLibraryStore, lastVisitedMatchStore } from './positionStore.js';
import { rankedDistancesStore, rankedTargetStore } from './rankedStore.js';
import { rolloutStore, idleRollout } from './rolloutStore.js';
import { databaseReadOnlyStore } from './databaseStore.js';
import { commandHistoryStore } from './commandHistoryStore.js';
import { matchListStore, emptyMatchList } from './matchListStore.js';

/**
 * @typedef {import('svelte/store').Writable<any>} AnyWritable
 * @typedef {[AnyWritable, () => any]} Scoped a store and the value a library starts from
 */

/** Saved and put back with the group; a library opened in a new group starts from the value. */
/** @type {Record<string, Scoped>} */
export const PHOTOGRAPHED = {
    // The transcription draft being edited; the backend keeps its session with the library.
    transcriptionStore: [transcriptionStore, () => null],
    transcriptionHistoryStore: [transcriptionHistoryStore, () => ({ canUndo: false, canRedo: false })],
    transcriptionHistoryActionStore: [transcriptionHistoryActionStore, () => null],
    transcriptionInfoStore: [transcriptionInfoStore, () => null],
    transcriptionPromptStore: [transcriptionPromptStore, () => null],
    transcriptionWheelStore: [transcriptionWheelStore, () => null],
    transcriptionKeyStore: [transcriptionKeyStore, () => initialKeyState()],
    transcriptionCubeRequestStore: [transcriptionCubeRequestStore, () => null],
    transcriptionBoardSwapStore: [transcriptionBoardSwapStore, () => false],
    transcriptionArrowsStore: [transcriptionArrowsStore, () => null],
    transcriptionModeStore: [transcriptionModeStore, () => null],
    // The Lesson being read.
    lessonStore: [lessonStore, () => null],
    lessonStepIndexStore: [lessonStepIndexStore, () => 0],
    lessonDoneStore: [lessonDoneStore, () => ({})],
    lessonEditorTargetStore: [lessonEditorTargetStore, () => 0],
    // The Anki deck open and the card under review.
    selectedAnkiDeckStore: [selectedAnkiDeckStore, () => null],
    ankiViewModeStore: [ankiViewModeStore, () => 'list'],
    ankiReviewCardStore: [ankiReviewCardStore, () => null],
    ankiDeckStatsStore: [ankiDeckStatsStore, () => null],
    ankiReviewActionStore: [ankiReviewActionStore, () => null],
    ankiPausedSessionStore: [ankiPausedSessionStore, () => null],
    ankiAnswerShownStore: [ankiAnswerShownStore, () => false],
    // The study queue walked through.
    studyQueueStore: [studyQueueStore, () => []],
    studyQueueIndexStore: [studyQueueIndexStore, () => 0],
    studyQueueBacklogStore: [studyQueueBacklogStore, () => false],
    studyQueueLastMarkedStore: [studyQueueLastMarkedStore, () => 0],
    studyQueueActiveStore: [studyQueueActiveStore, () => false],
    // The move being played on the board by a review or a Duel, and an ended Duel still shown
    // (a running one refuses the switch: viewStore.viewsLocked).
    quizPlayStore: [quizPlayStore, () => null],
    quizPlayValidateStore: [quizPlayValidateStore, () => null],
    duelStore: [duelStore, () => null],
    duelFoldedStore: [duelFoldedStore, () => false],
    duelBoardStore: [duelBoardStore, () => ({ prompt: null })],
    // Where the user was in this library.
    lastSearchStore: [lastSearchStore, () => null],
    lastVisitedMatchStore: [lastVisitedMatchStore, () => ({ matchID: null, currentIndex: 0, gameNumber: 1 })],
    positionBeforeFilterLibraryStore: [positionBeforeFilterLibraryStore, () => null],
    positionIndexBeforeFilterLibraryStore: [positionIndexBeforeFilterLibraryStore, () => -1],
    searchExcludePositionStore: [searchExcludePositionStore, () => emptySearchBoardPosition()],
    // The Search criteria, typed or run: match and tournament ids name rows of this library only.
    searchCriteriaStore: [searchCriteriaStore, () => null],
    searchEmptyStore: [searchEmptyStore, () => false],
    rankedDistancesStore: [rankedDistancesStore, () => new Map()],
    rankedTargetStore: [rankedTargetStore, () => 0],
    // The Matches list as it was painted: a return repaints nothing, and the list is read again
    // only when the library's match count moved meanwhile (refreshAfterShow).
    matchListStore: [matchListStore, () => emptyMatchList()],
    rolloutStore: [rolloutStore, () => idleRollout()],
    databaseReadOnlyStore: [databaseReadOnlyStore, () => false]
};

/** Photographed by another mechanism: the views (viewStore), the list, the group photo. */
export const PHOTOGRAPHED_ELSEWHERE = [
    'viewStore',
    'positionStore',
    'positionsStore',
    'matchContextStore',
    'listOriginStore',
    'currentPositionIndexStore',
    'analysisStore',
    'selectedMoveStore',
    'evalAnalysisStore',
    'commentTextStore',
    'activeTabStore',
    'statusBarModeStore',
    'databasePathStore',
    'databaseLoadedStore',
    'libraryCountsStore',
    // Through searchCriteriaStore, which flushes the Search form on screen first.
    'searchParamsStore',
    // Derived from photographed stores.
    'transcriptionCursorStore',
    'lessonStepStore',
    'studyQueueCurrentStore',
    'quizPlayCompleteStore',
    'duelHoldsBoardStore'
];

/** Emptied when the library changes; their panels read them again on libraryChangedStore. */
/** @type {Record<string, Scoped>} */
export const RELOADED = {
    collectionsStore: [collectionsStore, () => []],
    selectedCollectionStore: [selectedCollectionStore, () => null],
    activeCollectionStore: [activeCollectionStore, () => null],
    collectionPositionsStore: [collectionPositionsStore, () => []],
    tournamentsStore: [tournamentsStore, () => []],
    selectedTournamentStore: [selectedTournamentStore, () => null],
    tournamentMatchesStore: [tournamentMatchesStore, () => []],
    tournamentIDsFilterStore: [tournamentIDsFilterStore, () => null],
    ankiDecksStore: [ankiDecksStore, () => []],
    searchHistoryStore: [searchHistoryStore, () => []],
    filterLibraryStore: [filterLibraryStore, () => []],
    transcriptionListStore: [transcriptionListStore, () => []],
    duelListStore: [duelListStore, () => []],
    directionSummariesStore: [directionSummariesStore, () => []],
    // The status bar reads it again each time the command line opens.
    commandHistoryStore: [commandHistoryStore, () => []]
};

/** Read again by another mechanism when the library changes. */
export const RELOADED_ELSEWHERE = [
    // forgetDirection empties the open Direction; restoreDirection reopens it from the group's
    // memory, photographed with the group.
    'directionStore',
    'openDirectionIdStore',
    'pendingProposalsStore',
    'epreuveTabsStore',
    'rencontreParticipantsStore',
    'directionWarningsStore',
    'directionJumpStore',
    'directionOpenStore',
    'directionViewLoadedStore',
    'directionPageShownStore',
    'hallOpenStore',
    // Keyed on databasePathStore, which differs from one group to the next.
    'ankiDueStore',
    'listBlunderCountStore',
    'statsInvalidationKeyStore',
    'statsFilterStore',
    'statsResultStore',
    'statsLoadingStore',
    'statsErrorStore',
    'statsResultFilterStore',
    'playerTableStore',
    'playerTableFilterStore',
    'playerTableLoadingStore',
    'playerTableErrorStore',
    'recurringErrorsStore',
    'recurringErrorsLoadingStore',
    'recurringErrorsErrorStore',
    'studyPlanStore',
    'studyPlanLoadingStore',
    'studyPlanErrorStore',
    'studyEffectStore',
    'biasesStore',
    'studyLoopFilterStore',
    'studyLoopLoadingStore',
    'studyLoopErrorStore',
    'trainingStatsStore',
    'trainingStatsLoadingStore',
    'trainingStatsErrorStore'
];

/** A training session in progress: its clock and journal do not follow a group. */
export const REFUSED = [
    'trainingSessionStore',
    'trainingElapsedStore',
    'trainingJournalStore',
    'trainingPipOverrideStore',
    'trainingRefusalStore',
    'trainingAnalysisHiddenStore',
    'trainingHoldsBoardStore'
];

export const OUTSIDE = [
    // The board's transient state, recomputed from the position on screen.
    'analysisChallengeStore',
    'analysisChallengeHiddenStore',
    'analysisMaskStore',
    'captureDoubtsStore',
    'epcDataStore',
    'epcChallengeStore',
    'epcRevealedStore',
    'pastePositionTextStore',
    'clipboardPositionStore',
    'transcriptionNoticeStore',
    'duelNowStore',
    'duelAnimatingStore',
    'duelPipcountHiddenStore',
    'showPipcountStore',
    'pipcountVisibleStore',
    // Preferences and layout.
    'boardColorsStore',
    'boardOrientationStore',
    'themeStore',
    'uiScaleStore',
    'panelPositionStore',
    'effectivePositionStore',
    'panelWidthStore',
    'skipDuplicatesStore',
    'rolloutChoiceStore',
    'statsMetricStore',
    'statsTabStore',
    'trainingWindowStore',
    'searchStructureModeStore',
    'searchOfferedCubeStore',
    'metaStore',
    'homeDismissedStore',
    'pageStepStore',
    'configInitialTabStore',
    'videoPlacementStore',
    'videoSideStore',
    'videoStageOwnerStore',
    'videoStageTargetStore',
    // The command line, the palette, the status bar and one-shot requests.
    'statusBarTextStore',
    'commandTextStore',
    'showCommandInputStore',
    'commandPaletteOpenStore',
    'commandPaletteScopeStore',
    'matchOpenRequestStore',
    'tournamentOpenRequestStore',
    'matchPanelRefreshTriggerStore',
    'dbMutationCounterStore',
    'positionReloadTriggerStore',
    // Jobs and modals. A job either refuses a switch while it runs (libraryBusy, modals) or
    // runs attached to its library and names its group (an import, the watch, a batch).
    'bearoffProgressStore',
    'bearoffErrorStore',
    'gammonNetBatchStore',
    'maintenanceStore',
    'maintenanceEndedStore',
    'watchStatusStore',
    'watchImportNoticeStore',
    'exportModalModeStore',
    'exportPositionCountStore',
    'exportMetadataStore',
    'exportOptionsStore',
    'exportMatchesStore',
    'exportDecksStore',
    'exportCollectionCoverageStore',
    'exportScopeStore',
    'showImportProgressModalStore',
    'importModalModeStore',
    'importAnalysisStore',
    'importResultStore',
    'showFileImportModalStore',
    'fileImportModeStore',
    'fileImportTotalFilesStore',
    'fileImportCurrentIndexStore',
    'fileImportCurrentFileStore',
    'fileImportResultsStore',
    'fileImportReportStore',
    'fileImportJournalStore',
    'fileImportInterruptedStore',
    'fileImportProgressStore',
    'fileImportMinimizedStore',
    'fileImportGroupStore',
    'fileImportBusyStore',
    // The group machinery.
    'groupIdsStore',
    'parkedGroupsStore',
    'lastSwitchMsStore',
    'tabGroupsStore',
    'libraryChangedStore',
    'groupActivityStore',
    'importForegroundStore'
];

/** @param {Record<string, Scoped>} scope */
function take(scope) {
    /** @type {Record<string, any>} */
    const out = {};
    for (const [name, [store]] of Object.entries(scope)) out[name] = get(store);
    return out;
}

/** @param {Record<string, Scoped>} scope @param {Record<string, any> | null} values */
function put(scope, values) {
    for (const [name, [store, initial]] of Object.entries(scope)) {
        store.set(values && name in values ? values[name] : initial());
    }
}

/** The group's own state, to put back with putScopedState. */
export function takeScopedState() {
    return take(PHOTOGRAPHED);
}

/**
 * Install a group's own state (null: a library just opened starts from scratch) and empty what
 * the next library must read again. Runs before the group is shown, never on a photograph.
 * @param {Record<string, any> | null} photo
 */
export function putScopedState(photo) {
    put(PHOTOGRAPHED, photo);
    put(RELOADED, null);
}
