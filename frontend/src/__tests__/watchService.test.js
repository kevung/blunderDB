/**
 * watchService.test.js — le dossier surveillé (#258, fiche I.2).
 *
 * Ce qui est vérifié ici est ce qu'un test Go ne peut pas voir : le Go REGARDE
 * et l'interface IMPORTE, donc c'est ici que se décide ce qui arrive quand deux
 * salves de fichiers se suivent, et quand aucune base n'est ouverte.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

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
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    StartFolderWatch: (...a) => StartFolderWatch(...a),
    StopFolderWatch: (...a) => StopFolderWatch(...a),
    FolderWatchStatus: (...a) => FolderWatchStatus(...a)
}));

let watchSetting = { on: false, path: '', intervalSeconds: 0 };
const SaveWatchFolder = vi.fn(() => Promise.resolve(undefined));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetWatchFolder: () => Promise.resolve(watchSetting),
    SaveWatchFolder: (...a) => SaveWatchFolder(...a)
}));

let importedBatches = [];
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
vi.mock('../services/importService.js', () => ({
    importWatchedFiles: (...a) => importWatchedFiles(...a)
}));

import { databasePathStore } from '../stores/databaseStore.js';
import { watchImportNoticeStore, watchStatusStore } from '../stores/watchStore.js';
import { initFolderWatch, saveWatchSetting } from '../services/watchService.js';

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

        await emitFiles(['/matches/a.xg', '/matches/b.mat']);
        await vi.waitFor(() => expect(importedBatches.length).toBe(1));
        expect(importedBatches[0]).toEqual(['/matches/a.xg', '/matches/b.mat']);
        await vi.waitFor(() => expect(get(watchImportNoticeStore)).toEqual({ succeeded: 2, skipped: 0, failed: 0 }));
    });

    // Deux salves rapprochées ne doivent pas lancer deux imports concurrents
    // sur la même base : la seconde attend, et arrive en un seul lot.
    test('met les salves en file plutôt que de les importer en parallèle', async () => {
        await initFolderWatch();
        holdNext = true;

        const first = emitFiles(['/matches/a.xg']);
        await vi.waitFor(() => expect(importedBatches.length).toBe(1));
        emitFiles(['/matches/b.xg']);
        emitFiles(['/matches/c.xg']);
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
        await emitFiles(['/matches/a.xg']);
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
});
