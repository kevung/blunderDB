/**
 * keyboardService.transcriptionUndo.test.js — CTRL-Z / CTRL-MAJ-Z sur le
 * brouillon de transcription.
 *
 * C'est la seule ligne d'ux.md §3 qui porte un CTRL, et elle est donc liée ICI
 * et pas dans le panneau : une combinaison CTRL est toujours globale
 * (`isAlwaysGlobal`), et ce fichier tient les deux moitiés de cette décision —
 * que le répartiteur pose bien le geste, et qu'un champ de saisie garde CTRL-Z
 * pour lui, où c'est l'annulation du WebView et pas celle du brouillon.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../services/clipboardService.js', () => ({ copyPosition: vi.fn(), copyBoardImage: vi.fn(), copyBoardWithAnalysisImage: vi.fn() }));
vi.mock('../services/importService.js', () => ({ pastePosition: vi.fn(), importDatabase: vi.fn(), importPosition: vi.fn(), importFolder: vi.fn() }));
vi.mock('../services/exportService.js', () => ({ exportDatabase: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ newDatabase: vi.fn(), openDatabase: vi.fn(), exitApp: vi.fn(), setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({
    deletePosition: vi.fn(),
    saveCurrentPosition: vi.fn(),
    firstPosition: vi.fn(),
    previousPosition: vi.fn(),
    nextPosition: vi.fn(),
    lastPosition: vi.fn(),
    updatePosition: vi.fn(),
    toggleAnalysisPanel: vi.fn(),
    toggleCommentPanel: vi.fn(),
    toggleMetadataPanel: vi.fn(),
    toggleAnkiPanel: vi.fn(),
    toggleCollectionPanelAction: vi.fn(),
    toggleTournamentPanel: vi.fn(),
    toggleTranscriptionPanel: vi.fn(),
    toggleStatsPanel: vi.fn(),
    toggleSearchPanel: vi.fn(),
    toggleEPCMode: vi.fn(),
    togglePipcount: vi.fn(),
    reloadAllPositions: vi.fn(),
    loadRandomPosition: vi.fn(),
    showDatesAndMetadata: vi.fn()
}));

const { handleKeyDown } = await import('../services/keyboardService.js');
const { activeModal } = await import('../stores/uiStore.js');
const { transcriptionHistoryActionStore } = await import('../stores/transcriptionStore.js');
const { get } = await import('svelte/store');

function press(init, target) {
    const event = new KeyboardEvent('keydown', { cancelable: true, bubbles: true, ...init });
    if (target) target.dispatchEvent(event);
    else handleKeyDown(event);
    return event;
}

beforeEach(() => {
    activeModal.set(null);
    transcriptionHistoryActionStore.set(null);
});

describe('CTRL-Z et CTRL-MAJ-Z', () => {
    test('CTRL-Z demande une annulation', () => {
        const event = press({ key: 'z', code: 'KeyZ', ctrlKey: true });
        expect(get(transcriptionHistoryActionStore)).toBe('undo');
        expect(event.defaultPrevented).toBe(true);
    });

    test('CTRL-MAJ-Z demande un rétablissement — et passe AVANT la branche CTRL-Z', () => {
        press({ key: 'z', code: 'KeyZ', ctrlKey: true, shiftKey: true });
        expect(get(transcriptionHistoryActionStore)).toBe('redo');
    });

    test('sans brouillon ouvert le geste est simplement posé : c’est le panneau qui décide', () => {
        // Rien ici ne connaît le brouillon, et c'est voulu — le répartiteur ne
        // touche pas à la base, il pose une demande que le panneau sert ou non.
        press({ key: 'z', code: 'KeyZ', ctrlKey: true });
        expect(get(transcriptionHistoryActionStore)).toBe('undo');
    });
});

describe('un champ de saisie garde CTRL-Z', () => {
    let field;

    beforeEach(() => {
        field = document.createElement('input');
        document.body.appendChild(field);
        field.focus();
    });

    afterEach(() => {
        field.remove();
    });

    test('l’annulation du champ n’est pas celle du brouillon', () => {
        const event = new KeyboardEvent('keydown', { key: 'z', code: 'KeyZ', ctrlKey: true, cancelable: true });
        Object.defineProperty(event, 'target', { value: field });
        handleKeyDown(event);
        expect(get(transcriptionHistoryActionStore)).toBeNull();
        // Rien n'est empêché : le WebView fait son propre défaire.
        expect(event.defaultPrevented).toBe(false);
    });
});
