/**
 * The panels and modals that are not on the first screen, each fetched as its own chunk.
 *
 * Imported statically, they were all parsed before the window could paint. Here a
 * component is fetched the first time it is shown, and `preloadDeferredComponents`
 * fetches the rest once the window is usable, so a later first click finds its
 * component already loaded and never opens an empty panel.
 */
import { writable } from 'svelte/store';
import { logger } from '../utils/logger.js';

/**
 * @param {Record<string, () => Promise<{ default: import('svelte').Component<any> }>>} loaders
 */
function lazyRegistry(loaders) {
    /** @type {import('svelte/store').Writable<Record<string, import('svelte').Component<any>>>} */
    const components = writable({});
    /** @type {Map<string, Promise<void>>} */
    const pending = new Map();

    /** @param {string} name */
    function load(name) {
        const loader = loaders[name];
        if (!loader) return Promise.resolve();
        let promise = pending.get(name);
        if (!promise) {
            promise = loader().then(
                (module) => components.update((loaded) => ({ ...loaded, [name]: module.default })),
                (error) => {
                    // Forget the failure: the next attempt to show it fetches again.
                    pending.delete(name);
                    logger.error(`could not load the ${name} component:`, error);
                }
            );
            pending.set(name, promise);
        }
        return promise;
    }

    return {
        components,
        load,
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

/** Every deferred panel and modal, fetched once the window is usable. */
export function preloadDeferredComponents() {
    return Promise.all([panels.loadAll(), modals.loadAll()]).then(() => {});
}
