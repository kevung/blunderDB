/**
 * matchSheetService.test.js — la feuille de match imprimable.
 *
 * La feuille elle-même (numérotation, renvois, ArkID, code) est construite et
 * testée en Go (pkg/blunderdb/printsheet, database). Ici : ce que l'écran lui
 * passe — les matchs, les choix de la boîte, la langue — et le chemin du
 * fichier, enregistré puis ouvert.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';

const MatchSheetHTML = vi.fn(() => Promise.resolve('<!doctype html><html></html>'));
const SaveBoardImageDialog = vi.fn(() => Promise.resolve('/tmp/feuille.html'));
const SaveBoardSVG = vi.fn(() => Promise.resolve(undefined));
const OpenLocalPage = vi.fn(() => Promise.resolve(undefined));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    MatchSheetHTML: (/** @type {any[]} */ ...args) => MatchSheetHTML(...args)
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    SaveBoardImageDialog: (/** @type {any[]} */ ...args) => SaveBoardImageDialog(...args),
    SaveBoardSVG: (/** @type {any[]} */ ...args) => SaveBoardSVG(...args),
    OpenLocalPage: (/** @type {any[]} */ ...args) => OpenLocalPage(...args)
}));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));

import { DEFAULT_SHEET_OPTIONS, MAX_SHEET_MATCHES, printMatchSheet, sheetFilename } from '../services/matchSheetService.js';

beforeEach(() => {
    vi.clearAllMocks();
});

describe('printMatchSheet', () => {
    test('les blunders de l’adversaire sont décochés par défaut', () => {
        expect(DEFAULT_SHEET_OPTIONS.opponentBlunders).toBe(false);
        expect(DEFAULT_SHEET_OPTIONS.analysis).toBe(true);
    });

    test('passe les matchs, les choix et la langue, enregistre puis ouvre la feuille', async () => {
        const path = await printMatchSheet([4, 9], { ...DEFAULT_SHEET_OPTIONS, opponentBlunders: true });
        expect(path).toBe('/tmp/feuille.html');
        const [ids, opts] = /** @type {any[]} */ (MatchSheetHTML.mock.calls[0]);
        expect(ids).toEqual([4, 9]);
        expect(opts.opponentBlunders).toBe(true);
        expect(typeof opts.language).toBe('string');
        expect(SaveBoardImageDialog).toHaveBeenCalledWith('html', 'blunderdb-matchs-2.html');
        expect(SaveBoardSVG).toHaveBeenCalledWith('/tmp/feuille.html', '<!doctype html><html></html>');
        expect(OpenLocalPage).toHaveBeenCalledWith('/tmp/feuille.html');
    });

    test('une boîte d’enregistrement annulée n’écrit ni n’ouvre rien', async () => {
        SaveBoardImageDialog.mockImplementationOnce(() => Promise.resolve(''));
        expect(await printMatchSheet([4], DEFAULT_SHEET_OPTIONS)).toBeNull();
        expect(SaveBoardSVG).not.toHaveBeenCalled();
        expect(OpenLocalPage).not.toHaveBeenCalled();
    });

    test('sans match, rien n’est construit', async () => {
        expect(await printMatchSheet([], DEFAULT_SHEET_OPTIONS)).toBeNull();
        expect(MatchSheetHTML).not.toHaveBeenCalled();
        expect(sheetFilename([7])).toBe('blunderdb-match-7.html');
    });

    test('au-delà du plafond, rien n’est construit ni imprimé', async () => {
        const ids = Array.from({ length: MAX_SHEET_MATCHES + 1 }, (_, i) => i + 1);
        expect(await printMatchSheet(ids, DEFAULT_SHEET_OPTIONS)).toBeNull();
        expect(MatchSheetHTML).not.toHaveBeenCalled();
        expect(SaveBoardImageDialog).not.toHaveBeenCalled();
    });
});
