/**
 * HomeScreen.test.js — l'écran d'accueil (#284, fiche I.28).
 *
 * Ce qui est vérifié : les quatre chemins sont là, « importer mes matchs »
 * enchaîne bien la base neuve ET le sélecteur de fichiers (c'est l'enchaînement
 * qui est la fonctionnalité, pas les deux gestes pris séparément), et la base
 * récente n'est proposée que si son fichier existe encore — un bouton qui
 * échoue au clic est pire que pas de bouton.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent } from '@testing-library/svelte';

const newDatabase = vi.fn(() => Promise.resolve());
const openDatabase = vi.fn(() => Promise.resolve());
const openDatabaseByPath = vi.fn(() => Promise.resolve());
const loadDemoDatabase = vi.fn(() => Promise.resolve());
vi.mock('../services/databaseService.js', () => ({
    newDatabase: (...a) => newDatabase(...a),
    openDatabase: (...a) => openDatabase(...a),
    openDatabaseByPath: (...a) => openDatabaseByPath(...a),
    loadDemoDatabase: (...a) => loadDemoDatabase(...a),
    setStatusBarMessage: vi.fn()
}));

const importPosition = vi.fn(() => Promise.resolve());
vi.mock('../services/importService.js', () => ({
    importPosition: (...a) => importPosition(...a)
}));

let lastPath = '';
let pathExists = true;
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetLastDatabasePath: () => Promise.resolve(lastPath)
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    PathExists: () => Promise.resolve(pathExists)
}));

import HomeScreen from '../components/HomeScreen.svelte';

beforeEach(() => {
    lastPath = '';
    pathExists = true;
    vi.clearAllMocks();
});
afterEach(cleanup);

describe("l'écran d'accueil", () => {
    test('offre les quatre chemins', async () => {
        render(HomeScreen, { props: {} });
        expect(await screen.findByText('Import my matches')).toBeTruthy();
        expect(screen.getByText('Open the sample database')).toBeTruthy();
        expect(screen.getByText('Guided tour')).toBeTruthy();
        expect(screen.getByText('Open a database')).toBeTruthy();
    });

    // L'enchaînement EST la fonctionnalité : une base neuve puis les fichiers,
    // sans que l'utilisateur ait à comprendre qu'il lui faut d'abord l'une.
    test('"import my matches" creates the database then opens the picker', async () => {
        render(HomeScreen, { props: {} });
        await fireEvent.click(await screen.findByText('Import my matches'));
        await vi.waitFor(() => expect(newDatabase).toHaveBeenCalled());
        await vi.waitFor(() => expect(importPosition).toHaveBeenCalled());
    });

    test('propose la base récente quand son fichier existe encore', async () => {
        lastPath = '/home/quelquun/parties.db';
        render(HomeScreen, { props: {} });
        const resume = await screen.findByText('Reopen parties.db');
        await fireEvent.click(resume);
        expect(openDatabaseByPath).toHaveBeenCalledWith('/home/quelquun/parties.db');
    });

    test('ne propose pas une base récente qui a disparu', async () => {
        lastPath = '/home/quelquun/effacee.db';
        pathExists = false;
        render(HomeScreen, { props: {} });
        await screen.findByText('Import my matches');
        expect(screen.queryByText('Reopen effacee.db')).toBeNull();
    });

    test("s'écarte à la demande", async () => {
        const onDismiss = vi.fn();
        render(HomeScreen, { props: { onDismiss } });
        await fireEvent.click(await screen.findByText('Carry on without opening a database'));
        expect(onDismiss).toHaveBeenCalled();
    });
});
