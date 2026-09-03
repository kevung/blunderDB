/**
 * positionService.toggleTab.test.js
 *
 * Les `toggleXPanel` sont des entrées d'une table : chaque nom sélectionne son
 * onglet, refuse sans base ouverte, et deux d'entre elles portent une garde
 * supplémentaire (commentaire sans position courante, métadonnées en EDIT).
 * Depuis #202, un second appel sur l'onglet déjà actif est une vraie bascule :
 * il revient à l'onglet précédent plutôt que de ne rien faire.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadAllPositions: vi.fn(() => Promise.resolve([])),
    DeletePosition: vi.fn(),
    DeleteAnalysis: vi.fn(),
    UpdatePosition: vi.fn(),
    SaveAnalysis: vi.fn(),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    ComputeEPCFromPosition: vi.fn(() => Promise.resolve({})),
    SaveLastVisitedPosition: vi.fn(),
    GetLastVisitedMatch: vi.fn(() => Promise.resolve(null)),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    SaveEditPosition: vi.fn(),
    SaveExcludePosition: vi.fn(),
    SaveFilter: vi.fn(),
    LoadComment: vi.fn(() => Promise.resolve(''))
}));

vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));

import { setStatusBarMessage } from '../services/databaseService.js';
import { activeTabStore, statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { tMsg } from '../i18n';
import {
    toggleTab,
    toggleAnalysisPanel,
    toggleCommentPanel,
    toggleMetadataPanel,
    toggleAnkiPanel,
    toggleMatchPanel,
    toggleCollectionPanelAction,
    toggleTournamentPanel,
    toggleStatsPanel,
    toggleSearchPanel
} from '../services/positionService.js';

const NAMED = [
    [toggleAnalysisPanel, 'analysis'],
    [toggleCommentPanel, 'comments'],
    [toggleMetadataPanel, 'metadata'],
    [toggleAnkiPanel, 'anki'],
    [toggleMatchPanel, 'matches'],
    [toggleCollectionPanelAction, 'collections'],
    [toggleTournamentPanel, 'tournaments'],
    [toggleStatsPanel, 'stats'],
    [toggleSearchPanel, 'search']
];

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/fake/db.sqlite');
    statusBarModeStore.set('NORMAL');
    activeTabStore.set('epc');
    positionsStore.set([{ id: 1 }]);
    currentPositionIndexStore.set(0);
});

describe('toggleTab', () => {
    test.each(NAMED)('%o sélectionne l’onglet %s', (fn, tab) => {
        fn();
        expect(get(activeTabStore)).toBe(tab);
        expect(setStatusBarMessage).not.toHaveBeenCalled();
    });

    test('sans base ouverte : message et onglet inchangé', () => {
        databasePathStore.set('');
        for (const [fn, tab] of NAMED) {
            if (tab === 'metadata') continue;
            setStatusBarMessage.mockClear();
            fn();
            expect(get(activeTabStore), tab).toBe('epc');
            // search has its own no-database message (status.searchHistoryRequiresDb);
            // every other entry falls back to the generic one.
            expect(setStatusBarMessage, tab).toHaveBeenCalledWith(tMsg(tab === 'search' ? 'status.searchHistoryRequiresDb' : 'commands.noDatabaseOpened'));
        }
    });

    test('métadonnées sans base ouverte : silencieux', () => {
        databasePathStore.set('');
        toggleMetadataPanel();
        expect(get(activeTabStore)).toBe('epc');
        expect(setStatusBarMessage).not.toHaveBeenCalled();
    });

    test('commentaire sans position courante : refusé', () => {
        currentPositionIndexStore.set(-1);
        toggleCommentPanel();
        expect(get(activeTabStore)).toBe('epc');
        expect(setStatusBarMessage).toHaveBeenCalledWith(tMsg('status.noCurrentPositionComment'));
    });

    test('métadonnées en mode EDIT : refusé', () => {
        statusBarModeStore.set('EDIT');
        toggleMetadataPanel();
        expect(get(activeTabStore)).toBe('epc');
        expect(setStatusBarMessage).toHaveBeenCalledWith(tMsg('status.cannotShowMetadataEdit'));
    });

    test('onglet inconnu : erreur franche plutôt qu’un no-op silencieux', () => {
        expect(() => toggleTab('nope')).toThrow(/unknown tab/);
    });

    describe('bascule (#202)', () => {
        test('un second appel sur le même onglet revient à l’onglet précédent', () => {
            activeTabStore.set('matches');
            toggleAnalysisPanel(); // matches -> analysis
            expect(get(activeTabStore)).toBe('analysis');
            toggleAnalysisPanel(); // analysis -> back to matches
            expect(get(activeTabStore)).toBe('matches');
        });

        test('la mémoire est partagée entre les raccourcis, pas empilée par onglet', () => {
            activeTabStore.set('matches');
            toggleAnalysisPanel(); // matches -> analysis, previous = matches
            toggleCommentPanel(); // analysis -> comments, previous = analysis
            expect(get(activeTabStore)).toBe('comments');
            toggleAnalysisPanel(); // comments -> analysis, previous = comments
            expect(get(activeTabStore)).toBe('analysis');
            toggleAnalysisPanel(); // analysis -> back to comments
            expect(get(activeTabStore)).toBe('comments');
        });

        test('si l’onglet précédent mémorisé est l’onglet courant, bascule vers l’onglet par défaut (matches)', () => {
            // Construct previousTab === 'stats' deliberately (a toggle FROM
            // stats always remembers 'stats', never a value equal to the tab
            // being toggled), then bypass toggleTab to land back on 'stats'
            // by another means, so the next toggle finds a previous tab that
            // is no help at all.
            activeTabStore.set('stats');
            toggleAnalysisPanel(); // stats -> analysis, previous = stats
            activeTabStore.set('stats'); // bypasses toggleTab entirely
            toggleStatsPanel(); // already on stats; previous ('stats') === stats
            expect(get(activeTabStore)).toBe('matches');
        });

        test('un changement d’onglet par un autre moyen est pris en compte au prochain appel', () => {
            activeTabStore.set('matches');
            toggleAnalysisPanel(); // matches -> analysis, previous = matches
            activeTabStore.set('collections'); // bypasses toggleTab entirely
            toggleAnalysisPanel(); // collections -> analysis, previous = collections
            expect(get(activeTabStore)).toBe('analysis');
            toggleAnalysisPanel(); // analysis -> back to collections
            expect(get(activeTabStore)).toBe('collections');
        });
    });
});
