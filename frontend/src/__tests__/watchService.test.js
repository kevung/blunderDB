// @ts-nocheck — doublures de test : les signatures typées (Wails, stores) exigent des objets complets.
/**
 * watchService.test.js — le dossier surveillé (#258, fiche I.2).
 *
 * Ce qui est vérifié ici est ce qu'un test Go ne peut pas voir : le Go REGARDE
 * et l'interface IMPORTE, donc c'est ici que se décide ce qui arrive quand deux
 * salves de fichiers se suivent, et quand aucune base n'est ouverte.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

/** @type {((files: unknown) => void) | null} */
let emitFiles = null;

vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn((name, cb) => {
        if (name === 'folder-watch:files') emitFiles = cb;
        return () => {};
    })
}));

const StartFolderWatch = vi.fn(() => Promise.resolve({ running: true, folder: '/matches', intervalSeconds: 10 }));
const StopFolderWatch = vi.fn(() => Promise.resolve(undefined));
const FolderWatchStatus = vi.fn(() => Promise.resolve({ running: false, folder: '', intervalSeconds: 0 }));
const ImportFilesIn = vi.fn((/** @type {number} */ group, /** @type {string[]} */ files) => Promise.resolve({ succeeded: files.length, skipped: 0, failed: 0, group }));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    StartFolderWatch: (/** @type {any[]} */ ...a) => StartFolderWatch(...a),
    StopFolderWatch: (/** @type {any[]} */ ...a) => StopFolderWatch(...a),
    FolderWatchStatus: (/** @type {any[]} */ ...a) => FolderWatchStatus(...a),
    ImportFilesIn: (/** @type {any[]} */ ...a) => ImportFilesIn(...a),
    OpenLibraryGroup: vi.fn(),
    ActivateLibraryGroup: vi.fn(),
    CloseLibraryGroup: vi.fn()
}));

let watchSetting = { on: false, path: '', intervalSeconds: 0 };
const SaveWatchFolder = vi.fn(() => Promise.resolve(undefined));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetWatchFolder: () => Promise.resolve(watchSetting),
    SaveWatchFolder: (/** @type {any[]} */ ...a) => SaveWatchFolder(...a)
}));

/** @type {unknown[]} */
let importedBatches = [];
/** @type {(() => void) | null} */
let importResolve = null;
// Un seul import peut être retenu à la fois, et seulement le premier : un
// import laissé en suspens à la fin d'un test bloquerait la file du module
// pour tous les suivants.
let holdNext = false;
const importWatchedFiles = vi.fn((files) => {
    importedBatches.push(files);
    if (holdNext) {
        holdNext = false;
        return new Promise((r) => (importResolve = () => r({ succeeded: files.length, skipped: 0, failed: 0 })));
    }
    return Promise.resolve({ succeeded: files.length, skipped: 0, failed: 0 });
});
const maybeAutoAnalyzeAfterImport = vi.fn(() => Promise.resolve());
vi.mock('../services/importService.js', () => ({
    importWatchedFiles: (/** @type {any[]} */ ...a) => importWatchedFiles(...a),
    maybeAutoAnalyzeAfterImport: (/** @type {any[]} */ ...a) => maybeAutoAnalyzeAfterImport(...a)
}));

import { databasePathStore } from '../stores/databaseStore.js';
import { watchImportNoticeStore, watchStatusStore } from '../stores/watchStore.js';
import { initFolderWatch, saveWatchSetting } from '../services/watchService.js';
import { activeGroupId, groupIdsStore, groupActivityStore } from '../stores/libraryGroupStore.js';

beforeEach(() => {
    // emitFiles n'est PAS remis à null : watchService ne s'abonne qu'une fois
    // (le module garde son drapeau), donc le rappel capté au premier
    // initFolderWatch est celui de tout le fichier.
    importedBatches = [];
    importResolve = null;
    holdNext = false;
    watchSetting = { on: false, path: '', intervalSeconds: 0 };
    databasePathStore.set('/some/base.db');
    watchImportNoticeStore.set(null);
    vi.clearAllMocks();
});

describe('le dossier surveillé', () => {
    test("ne démarre aucune surveillance tant que la configuration n'en demande pas", async () => {
        await initFolderWatch();
        expect(emitFiles).toBeTypeOf('function');
        expect(StartFolderWatch).not.toHaveBeenCalled();
    });

    test('démarre la surveillance que la configuration décrit', async () => {
        watchSetting = { on: true, path: '/matches', intervalSeconds: 30 };
        await initFolderWatch();
        expect(StartFolderWatch).toHaveBeenCalledWith('/matches', 30);
        expect(get(watchStatusStore).running).toBe(true);
    });

    test('importe les fichiers annoncés et publie la notification', async () => {
        await initFolderWatch();
        expect(emitFiles).toBeTypeOf('function');

        await emitFiles({ group: 0, files: ['/matches/a.xg', '/matches/b.mat'] });
        await vi.waitFor(() => expect(importedBatches.length).toBe(1));
        expect(importedBatches[0]).toEqual(['/matches/a.xg', '/matches/b.mat']);
        await vi.waitFor(() => expect(get(watchImportNoticeStore)).toEqual({ succeeded: 2, skipped: 0, failed: 0 }));
    });

    // Deux salves rapprochées ne doivent pas lancer deux imports concurrents
    // sur la même base : la seconde attend, et arrive en un seul lot.
    test('met les salves en file plutôt que de les importer en parallèle', async () => {
        await initFolderWatch();
        holdNext = true;

        const first = emitFiles({ group: 0, files: ['/matches/a.xg'] });
        await vi.waitFor(() => expect(importedBatches.length).toBe(1));
        emitFiles({ group: 0, files: ['/matches/b.xg'] });
        emitFiles({ group: 0, files: ['/matches/c.xg'] });
        expect(importedBatches.length).toBe(1); // toujours un seul import en vol

        importResolve();
        await first;
        await vi.waitFor(() => expect(importResolve).toBeTypeOf('function'));
        await vi.waitFor(() => expect(importedBatches.length).toBe(2));
        expect(importedBatches[1]).toEqual(['/matches/b.xg', '/matches/c.xg']);
    });

    // Sans base ouverte on n'importe pas — mais on ne jette pas non plus : le
    // dossier a déjà annoncé ces fichiers et ne les annoncera plus, donc les
    // oublier ici les perdrait définitivement. Ils partent à l'ouverture.
    test("garde les fichiers en file quand aucune base n'est ouverte, et les importe à l'ouverture", async () => {
        await initFolderWatch();
        databasePathStore.set('');
        await emitFiles({ group: 0, files: ['/matches/a.xg'] });
        expect(importedBatches.length).toBe(0);

        databasePathStore.set('/some/base.db');
        await vi.waitFor(() => expect(importedBatches.length).toBe(1));
        expect(importedBatches[0]).toEqual(['/matches/a.xg']);
    });

    test('un réglage enregistré prend effet tout de suite', async () => {
        await saveWatchSetting(true, '/matches', 20);
        expect(SaveWatchFolder).toHaveBeenCalledWith(true, '/matches', 20);
        expect(StartFolderWatch).toHaveBeenCalledWith('/matches', 20);

        await saveWatchSetting(false, '/matches', 20);
        expect(StopFolderWatch).toHaveBeenCalled();
    });

    test("les fichiers d'un groupe non affiché vont à sa bibliothèque, en fond ; une base attend son retour", async () => {
        await initFolderWatch();
        groupIdsStore.set([1, 2]);
        activeGroupId.set(2);
        try {
            await emitFiles({ group: 1, files: ['/matches/a.xg', '/matches/club.db'] });
            expect(ImportFilesIn).toHaveBeenCalledWith(1, ['/matches/a.xg'], '/matches/a.xg');
            expect(importedBatches).toEqual([]);
            expect(get(groupActivityStore).get(1)?.state).toBe('done');
            expect(maybeAutoAnalyzeAfterImport).toHaveBeenCalledWith(1);
            // Le groupe revient : la base à fusionner part par le chemin habituel.
            activeGroupId.set(1);
            await vi.waitFor(() => expect(importedBatches).toEqual([['/matches/club.db']]));
        } finally {
            groupIdsStore.set([1]);
            activeGroupId.set(1);
            groupActivityStore.set(new Map());
        }
    });
});
