/**
 * The panels and modals that are not on the first screen, each fetched as its own chunk.
 *
 * Imported statically, they were all parsed before the window could paint. Here a
 * component is fetched the first time it is shown, and `preloadDeferredComponents`
 * fetches the rest once the window has painted, one chunk per idle slice so that the
 * main thread stays free for the first clicks. A click that comes before a chunk is
 * there shows a loading indicator (`failed` tells the panel when the fetch gave up).
 */
import { writable } from 'svelte/store';
import { logger } from '../utils/logger.js';

/**
 * @param {Record<string, () => Promise<{ default: import('svelte').Component<any> }>>} loaders
 */
export function lazyRegistry(loaders) {
    /** @type {import('svelte/store').Writable<Record<string, import('svelte').Component<any>>>} */
    const components = writable({});
    /** @type {Map<string, Promise<void>>} */
    const pending = new Map();
    /** Names whose last fetch failed; cleared when a new attempt starts. */
    const failed = writable(/** @type {Set<string>} */ (new Set()));
    /** @param {string} name @param {boolean} on */
    const setFailed = (name, on) =>
        failed.update((set) => {
            const next = new Set(set);
            if (on) next.add(name);
            else next.delete(name);
            return next;
        });

    /** @param {string} name */
    function load(name) {
        const loader = loaders[name];
        if (!loader) return Promise.resolve();
        let promise = pending.get(name);
        if (!promise) {
            setFailed(name, false);
            promise = loader().then(
                (module) => components.update((loaded) => ({ ...loaded, [name]: module.default })),
                (error) => {
                    // Forget the failure so that the next opening asks again. The browser
                    // keeps a failed module import failed, so that attempt only succeeds if
                    // the browser did not cache the failure; the panel offers a reload.
                    pending.delete(name);
                    setFailed(name, true);
                    logger.error(`could not load the ${name} component:`, error);
                }
            );
            pending.set(name, promise);
        }
        return promise;
    }

    return {
        components,
        failed,
        load,
        names: () => Object.keys(loaders),
        loadAll: () => Promise.all(Object.keys(loaders).map(load)).then(() => {})
    };
}

/** Keyed by the tab id of services/tabCatalog.js. */
export const panels = lazyRegistry({
    analysis: () => import('./AnalysisPanel.svelte'),
    comments: () => import('./CommentPanel.svelte'),
    search: () => import('./SearchPanel.svelte'),
    collections: () => import('./CollectionPanel.svelte'),
    anki: () => import('./AnkiPanel.svelte'),
    training: () => import('./TrainingPanel.svelte'),
    duel: () => import('./DuelPanel.svelte'),
    matches: () => import('./MatchPanel.svelte'),
    tournaments: () => import('./TournamentPanel.svelte'),
    stats: () => import('./stats/StatsPanel.svelte'),
    eval: () => import('./EvalPanel.svelte'),
    transcription: () => import('./TranscriptionPanel.svelte'),
    metadata: () => import('./MetadataPanel.svelte')
});

/** Keyed by the MODAL id that opens each one, plus the three opened by their own store. */
export const modals = lazyRegistry({
    met: () => import('./MetModal.svelte'),
    cubeMatrix: () => import('./CubeMatrixModal.svelte'),
    tags: () => import('./TagsModal.svelte'),
    lessonEditor: () => import('./LessonEditorModal.svelte'),
    goToPosition: () => import('./GoToPositionModal.svelte'),
    exportDatabase: () => import('./ExportDatabaseModal.svelte'),
    help: () => import('./HelpModal.svelte'),
    config: () => import('./ConfigModal.svelte'),
    tour: () => import('./TourCatalogModal.svelte'),
    trash: () => import('./TrashModal.svelte'),
    resumeBatch: () => import('./ResumeBatchModal.svelte'),
    log: () => import('./LogModal.svelte'),
    contactSheet: () => import('./ContactSheetModal.svelte'),
    dataTable: () => import('./DataTableModal.svelte'),
    importProgress: () => import('./ImportProgressModal.svelte'),
    fileImportProgress: () => import('./FileImportProgressModal.svelte')
});

/**
 * Runs `task` when the main thread is idle, or after a short pause where there is no idle callback (WebKit).
 * @param {() => void} task
 */
function whenIdle(task) {
    if (typeof requestIdleCallback === 'function') requestIdleCallback(() => task(), { timeout: 500 });
    else setTimeout(task, 50);
}

/**
 * Fetches every deferred panel and modal, one chunk per idle slice, `first` (the active
 * tab) ahead of the rest. With `{ immediate: true }` all are requested at once.
 *
 * @param {{ first?: string, immediate?: boolean }} [options]
 */
export function preloadDeferredComponents({ first, immediate = false } = {}) {
    if (immediate) return Promise.all([panels.loadAll(), modals.loadAll()]).then(() => {});
    const queue = [...panels.names().map((name) => () => panels.load(name)), ...modals.names().map((name) => () => modals.load(name))];
    if (first && panels.names().includes(first)) queue.unshift(() => panels.load(first));
    return new Promise((/** @type {(value?: any) => void} */ resolve) => {
        const next = () => {
            const step = queue.shift();
            if (!step) return resolve();
            // Wait for the fetch, then for the next idle slice: evaluating a chunk is what
            // occupies the thread, and a click in between loads its own panel first.
            step().then(() => whenIdle(next));
        };
        whenIdle(next);
    });
}
