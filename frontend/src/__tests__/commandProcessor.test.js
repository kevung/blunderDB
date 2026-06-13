import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// Mock Wails bindings before importing commandProcessor
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveComment: vi.fn().mockResolvedValue(undefined),
    Migrate_1_0_0_to_1_1_0: vi.fn().mockResolvedValue(undefined),
    Migrate_1_1_0_to_1_2_0: vi.fn().mockResolvedValue(undefined),
    Migrate_1_2_0_to_1_3_0: vi.fn().mockResolvedValue(undefined),
    ClearCommandHistory: vi.fn().mockResolvedValue(undefined),
    SaveSearchHistory: vi.fn().mockResolvedValue(undefined)
}));

import { parseFilters, processCommand, initCommandProcessor } from '../commandProcessor.js';
import { SaveComment } from '../../wailsjs/go/database/Database.js';
import { translate, resolveStatusMessage } from '../i18n';

// The status store now holds a tMsg() descriptor ({ i18nKey, i18nParams }) so
// the status bar can re-translate live. Resolve it to the English string the
// way StatusBar does (via the translate function) for these assertions.
const statusText = () => resolveStatusMessage(get(statusBarTextStore), translate);
import { currentPositionIndexStore, statusBarTextStore, logEntriesStore, activeModal, MODAL } from '../stores/uiStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { statusBarModeStore } from '../stores/uiStore.js';
import { commandHistoryStore } from '../stores/commandHistoryStore.js';

// ---------------------------------------------------------------------------
// parseFilters — pure function, no mocks needed
// ---------------------------------------------------------------------------
describe('parseFilters', () => {
    test('returns all fields with empty filters', () => {
        const result = parseFilters([], 's');
        expect(result).toHaveProperty('includeCube', false);
        expect(result).toHaveProperty('includeScore', false);
        expect(result).toHaveProperty('noContactFilter', false);
        expect(result).toHaveProperty('decisionTypeFilter', false);
        expect(result).toHaveProperty('diceRollFilter', false);
        expect(result).toHaveProperty('mirrorPositionFilter', false);
        expect(result.pipCountFilter).toBeUndefined();
        expect(result.winRateFilter).toBeUndefined();
        expect(result.matchIDsFilter).toBe('');
        expect(result.tournamentIDsFilter).toBe('');
    });

    // -- boolean flags -------------------------------------------------------
    test('includeCube with "cube"', () => {
        expect(parseFilters(['cube'], 's cube').includeCube).toBe(true);
    });

    test('includeCube with "cu" shorthand', () => {
        expect(parseFilters(['cu'], 's cu').includeCube).toBe(true);
    });

    test('includeCube with "c" shorthand', () => {
        expect(parseFilters(['c'], 's c').includeCube).toBe(true);
    });

    test('includeScore with "score"', () => {
        expect(parseFilters(['score'], 's score').includeScore).toBe(true);
    });

    test('includeScore with "sc" shorthand', () => {
        expect(parseFilters(['sc'], 's sc').includeScore).toBe(true);
    });

    test('noContactFilter with "nc"', () => {
        expect(parseFilters(['nc'], 's nc').noContactFilter).toBe(true);
    });

    test('decisionTypeFilter with "d"', () => {
        expect(parseFilters(['d'], 's d').decisionTypeFilter).toBe(true);
    });

    test('diceRollFilter with "D"', () => {
        const r = parseFilters(['D'], 's D');
        expect(r.diceRollFilter).toBe(true);
        expect(r.diceRollMode).toBe('both');
    });

    test('diceRollFilter with "D1" (first die only)', () => {
        const r = parseFilters(['D1'], 's D1');
        expect(r.diceRollFilter).toBe(true);
        expect(r.diceRollMode).toBe('first');
    });

    test('mirrorPositionFilter with "M"', () => {
        expect(parseFilters(['M'], 's M').mirrorPositionFilter).toBe(true);
    });

    // -- range / comparison filters ------------------------------------------
    test('pip count filter greater-than', () => {
        expect(parseFilters(['p>30'], 's p>30').pipCountFilter).toBe('p>30');
    });

    test('pip count filter less-than', () => {
        expect(parseFilters(['p<10'], 's p<10').pipCountFilter).toBe('p<10');
    });

    test('pip count filter range', () => {
        expect(parseFilters(['p5,20'], 's p5,20').pipCountFilter).toBe('p5,20');
    });

    test('win rate filter greater-than', () => {
        expect(parseFilters(['w>50'], 's w>50').winRateFilter).toBe('w>50');
    });

    test('win rate filter less-than', () => {
        expect(parseFilters(['w<30'], 's w<30').winRateFilter).toBe('w<30');
    });

    test('gammon rate filter', () => {
        expect(parseFilters(['g>10'], 's g>10').gammonRateFilter).toBe('g>10');
    });

    test('backgammon rate filter', () => {
        expect(parseFilters(['b>5'], 's b>5').backgammonRateFilter).toBe('b>5');
    });

    test('backgammon filter excludes bo prefix', () => {
        const result = parseFilters(['bo3,5'], 's bo3,5');
        expect(result.backgammonRateFilter).toBeUndefined();
        expect(result.player1OutfieldBlotFilter).toBe('bo3,5');
    });

    test('backgammon filter excludes bj prefix', () => {
        const result = parseFilters(['bj2,4'], 's bj2,4');
        expect(result.backgammonRateFilter).toBeUndefined();
        expect(result.player1JanBlotFilter).toBe('bj2,4');
    });

    test('player2 win rate filter (uppercase W)', () => {
        expect(parseFilters(['W>60'], 's W>60').player2WinRateFilter).toBe('W>60');
    });

    test('player2 gammon rate filter (uppercase G)', () => {
        expect(parseFilters(['G>15'], 's G>15').player2GammonRateFilter).toBe('G>15');
    });

    test('player2 backgammon rate filter (uppercase B)', () => {
        expect(parseFilters(['B>2'], 's B>2').player2BackgammonRateFilter).toBe('B>2');
    });

    test('player2 backgammon filter excludes BO prefix', () => {
        const result = parseFilters(['BO3,5'], 's BO3,5');
        expect(result.player2BackgammonRateFilter).toBeUndefined();
        expect(result.player2OutfieldBlotFilter).toBe('BO3,5');
    });

    // -- checker off filters -------------------------------------------------
    test('player1 checker off with range', () => {
        expect(parseFilters(['o3,8'], 's o3,8').player1CheckerOffFilter).toBe('o3,8');
    });

    test('player1 checker off single value gets expanded', () => {
        expect(parseFilters(['o5'], 's o5').player1CheckerOffFilter).toBe('o5,5');
    });

    test('player1 checker off greater-than not expanded', () => {
        expect(parseFilters(['o>3'], 's o>3').player1CheckerOffFilter).toBe('o>3');
    });

    test('player2 checker off with range', () => {
        expect(parseFilters(['O2,7'], 's O2,7').player2CheckerOffFilter).toBe('O2,7');
    });

    test('player2 checker off single value gets expanded', () => {
        expect(parseFilters(['O4'], 's O4').player2CheckerOffFilter).toBe('O4,4');
    });

    // -- back checker filters ------------------------------------------------
    test('player1 back checker with range', () => {
        expect(parseFilters(['k1,6'], 's k1,6').player1BackCheckerFilter).toBe('k1,6');
    });

    test('player1 back checker single value gets expanded', () => {
        expect(parseFilters(['k3'], 's k3').player1BackCheckerFilter).toBe('k3,3');
    });

    test('player2 back checker with range', () => {
        expect(parseFilters(['K1,6'], 's K1,6').player2BackCheckerFilter).toBe('K1,6');
    });

    test('player2 back checker single value gets expanded', () => {
        expect(parseFilters(['K3'], 's K3').player2BackCheckerFilter).toBe('K3,3');
    });

    // -- zone filters --------------------------------------------------------
    test('player1 checker in zone with range', () => {
        expect(parseFilters(['z2,5'], 's z2,5').player1CheckerInZoneFilter).toBe('z2,5');
    });

    test('player1 checker in zone single value gets expanded', () => {
        expect(parseFilters(['z4'], 's z4').player1CheckerInZoneFilter).toBe('z4,4');
    });

    test('player2 checker in zone with range', () => {
        expect(parseFilters(['Z1,3'], 's Z1,3').player2CheckerInZoneFilter).toBe('Z1,3');
    });

    test('player2 checker in zone single value gets expanded', () => {
        expect(parseFilters(['Z2'], 's Z2').player2CheckerInZoneFilter).toBe('Z2,2');
    });

    // -- equity / pip / date -------------------------------------------------
    test('equity filter', () => {
        expect(parseFilters(['e>0.5'], 's e>0.5').equityFilter).toBe('e>0.5');
    });

    test('equity filter range', () => {
        expect(parseFilters(['e-0.3,0.5'], 's e-0.3,0.5').equityFilter).toBe('e-0.3,0.5');
    });

    test('absolute pip count filter', () => {
        expect(parseFilters(['P>100'], 's P>100').player1AbsolutePipCountFilter).toBe('P>100');
    });

    test('date filter', () => {
        expect(parseFilters(['T>2024-01-01'], 's T>2024-01-01').dateFilter).toBe('T>2024-01-01');
    });

    // -- move error filter ---------------------------------------------------
    test('move error filter greater-than', () => {
        expect(parseFilters(['E>0.05'], 's E>0.05').moveErrorFilter).toBe('E>0.05');
    });

    test('move error filter less-than', () => {
        expect(parseFilters(['E<0.1'], 's E<0.1').moveErrorFilter).toBe('E<0.1');
    });

    test('move error filter range', () => {
        expect(parseFilters(['E0.01,0.1'], 's E0.01,0.1').moveErrorFilter).toBe('E0.01,0.1');
    });

    // -- text / move pattern -------------------------------------------------
    test('search text with double quotes', () => {
        const result = parseFilters(['t"hello"'], 's t"hello"');
        expect(result.searchText).toBe('t"hello"');
    });

    test('search text with single quotes', () => {
        const result = parseFilters(["t'hello'"], "s t'hello'");
        expect(result.searchText).toBe("t'hello'");
    });

    test('move pattern with double quotes', () => {
        const result = parseFilters(['m"24/23"'], 's m"24/23"');
        expect(result.movePatternFilter).toBe('m"24/23"');
    });

    test('move pattern with single quotes', () => {
        const result = parseFilters(["m'24/23'"], "s m'24/23'");
        expect(result.movePatternFilter).toBe("m'24/23'");
    });

    test('no search text returns empty string', () => {
        expect(parseFilters(['p>30'], 's p>30').searchText).toBe('');
    });

    test('no move pattern returns empty string', () => {
        expect(parseFilters(['p>30'], 's p>30').movePatternFilter).toBe('');
    });

    // -- blot filters --------------------------------------------------------
    test('player1 outfield blot filter', () => {
        expect(parseFilters(['bo>2'], 's bo>2').player1OutfieldBlotFilter).toBe('bo>2');
    });

    test('player2 outfield blot filter', () => {
        expect(parseFilters(['BO>3'], 's BO>3').player2OutfieldBlotFilter).toBe('BO>3');
    });

    test('player1 jan blot filter', () => {
        expect(parseFilters(['bj>1'], 's bj>1').player1JanBlotFilter).toBe('bj>1');
    });

    test('player2 jan blot filter', () => {
        expect(parseFilters(['BJ>2'], 's BJ>2').player2JanBlotFilter).toBe('BJ>2');
    });

    // -- match / tournament ID filters ---------------------------------------
    test('single match ID', () => {
        expect(parseFilters(['ma42'], 's ma42').matchIDsFilter).toBe('42');
    });

    test('multiple match IDs', () => {
        const result = parseFilters(['ma1', 'ma2', 'ma3'], 's ma1 ma2 ma3');
        expect(result.matchIDsFilter).toBe('1;2;3');
    });

    test('no match IDs returns empty string', () => {
        expect(parseFilters(['p>30'], 's p>30').matchIDsFilter).toBe('');
    });

    test('single tournament ID', () => {
        expect(parseFilters(['tn7'], 's tn7').tournamentIDsFilter).toBe('7');
    });

    test('multiple tournament IDs', () => {
        const result = parseFilters(['tn10', 'tn20'], 's tn10 tn20');
        expect(result.tournamentIDsFilter).toBe('10;20');
    });

    test('no tournament IDs returns empty string', () => {
        expect(parseFilters([], 's').tournamentIDsFilter).toBe('');
    });

    // -- position ID filter --------------------------------------------------
    test('single position ID', () => {
        expect(parseFilters(['id42'], 's id42').positionIDsFilter).toBe('42');
    });

    test('position ID range stays a single comma token', () => {
        expect(parseFilters(['id5,10'], 's id5,10').positionIDsFilter).toBe('5,10');
    });

    test('multiple position IDs join with semicolons', () => {
        const result = parseFilters(['id5', 'id10', 'id15'], 's id5 id10 id15');
        expect(result.positionIDsFilter).toBe('5;10;15');
    });

    test('no position IDs returns empty string', () => {
        expect(parseFilters([], 's').positionIDsFilter).toBe('');
    });

    // -- combined filters ----------------------------------------------------
    test('multiple filters combined', () => {
        const filters = ['cube', 'p>30', 'w>50', 'E>0.05', 'nc'];
        const result = parseFilters(filters, 's cube p>30 w>50 E>0.05 nc');
        expect(result.includeCube).toBe(true);
        expect(result.pipCountFilter).toBe('p>30');
        expect(result.winRateFilter).toBe('w>50');
        expect(result.moveErrorFilter).toBe('E>0.05');
        expect(result.noContactFilter).toBe(true);
    });
});

// ---------------------------------------------------------------------------
// processCommand — needs store + callback mocking
// ---------------------------------------------------------------------------
describe('processCommand', () => {
    let callbacks;

    beforeEach(() => {
        callbacks = {
            onNewDatabase: vi.fn(),
            onOpenDatabase: vi.fn(),
            onImportDatabase: vi.fn(),
            onExportDatabase: vi.fn(),
            importPosition: vi.fn(),
            onSavePosition: vi.fn(),
            onUpdatePosition: vi.fn(),
            onDeletePosition: vi.fn(),
            onToggleAnalysis: vi.fn(),
            onToggleComment: vi.fn(),
            exitApp: vi.fn(),
            onToggleHelp: vi.fn(),
            onLoadAllPositions: vi.fn(),
            onLoadPositionsByFilters: vi.fn(),
            toggleMetadataPanel: vi.fn(),
            toggleSearchHistoryPanel: vi.fn(),
            toggleMatchPanel: vi.fn(),
            toggleCollectionPanel: vi.fn(),
            toggleEPCMode: vi.fn(),
            toggleMatchMode: vi.fn(),
            onToggleStats: vi.fn(),
            onLoadBlunders: vi.fn()
        };
        initCommandProcessor(callbacks);

        // Reset stores to defaults
        currentPositionIndexStore.set(0);
        statusBarTextStore.set('');
        logEntriesStore.set([]);
        activeModal.set(null);
        positionsStore.set([]);
        databasePathStore.set('');
        statusBarModeStore.set('NORMAL');
        commandHistoryStore.set([]);
    });

    // -- numeric navigation --------------------------------------------------
    test('numeric input navigates to position', () => {
        positionsStore.set([{ id: 1 }, { id: 2 }, { id: 3 }]);
        processCommand('2');
        expect(get(currentPositionIndexStore)).toBe(1);
    });

    test('numeric input clamps to first position', () => {
        positionsStore.set([{ id: 1 }, { id: 2 }]);
        processCommand('0');
        expect(get(currentPositionIndexStore)).toBe(0);
    });

    test('numeric input clamps to last position', () => {
        positionsStore.set([{ id: 1 }, { id: 2 }]);
        processCommand('99');
        expect(get(currentPositionIndexStore)).toBe(1);
    });

    // -- tag insertion -------------------------------------------------------
    test('# command adds tag log entry', () => {
        processCommand('#blunder');
        const entries = get(logEntriesStore);
        expect(entries.length).toBe(1);
        expect(entries[0].message).toContain('Tags added');
    });

    test('# command saves the comment to the current position id (not the array index)', () => {
        // Regression: SaveComment expects a position *id*, not the array index.
        SaveComment.mockClear();
        positionsStore.set([{ id: 10 }, { id: 20 }, { id: 30 }]);
        currentPositionIndexStore.set(1); // index 1 -> position id 20
        processCommand('#blunder');
        expect(SaveComment).toHaveBeenCalledTimes(1);
        expect(SaveComment.mock.calls[0][0]).toBe(20);
    });

    // -- simple callback commands --------------------------------------------
    const callbackCommands = [
        ['new', 'onNewDatabase'],
        ['ne', 'onNewDatabase'],
        ['n', 'onNewDatabase'],
        ['open', 'onOpenDatabase'],
        ['op', 'onOpenDatabase'],
        ['o', 'onOpenDatabase'],
        ['import_db', 'onImportDatabase'],
        ['idb', 'onImportDatabase'],
        ['export_db', 'onExportDatabase'],
        ['edb', 'onExportDatabase'],
        ['import', 'importPosition'],
        ['i', 'importPosition'],
        ['write', 'onSavePosition'],
        ['wr', 'onSavePosition'],
        ['w', 'onSavePosition'],
        ['write!', 'onUpdatePosition'],
        ['wr!', 'onUpdatePosition'],
        ['w!', 'onUpdatePosition'],
        ['delete', 'onDeletePosition'],
        ['del', 'onDeletePosition'],
        ['d', 'onDeletePosition'],
        ['list', 'onToggleAnalysis'],
        ['l', 'onToggleAnalysis'],
        ['comment', 'onToggleComment'],
        ['co', 'onToggleComment'],
        ['quit', 'exitApp'],
        ['q', 'exitApp'],
        ['help', 'onToggleHelp'],
        ['he', 'onToggleHelp'],
        ['h', 'onToggleHelp'],
        ['e', 'onLoadAllPositions'],
        ['history', 'toggleSearchHistoryPanel'],
        ['hi', 'toggleSearchHistoryPanel'],
        ['match', 'toggleMatchPanel'],
        ['ma', 'toggleMatchPanel'],
        ['collection', 'toggleCollectionPanel'],
        ['coll', 'toggleCollectionPanel'],
        ['stats', 'onToggleStats'],
        ['st', 'onToggleStats'],
        ['blunders', 'onLoadBlunders'],
        ['bl', 'onLoadBlunders'],
        ['epc', 'toggleEPCMode'],
        ['m', 'toggleMatchMode']
    ];

    test.each(callbackCommands)('"%s" calls %s', (cmd, cb) => {
        processCommand(cmd);
        expect(callbacks[cb]).toHaveBeenCalledOnce();
    });

    // -- modal commands ------------------------------------------------------
    test('met opens MET modal', () => {
        processCommand('met');
        expect(get(activeModal)).toBe(MODAL.MET);
    });

    test('tp2_last opens TAKE_POINT_2_LAST modal', () => {
        processCommand('tp2_last');
        expect(get(activeModal)).toBe(MODAL.TAKE_POINT_2_LAST);
    });

    test('tp2_live opens TAKE_POINT_2_LIVE modal', () => {
        processCommand('tp2_live');
        expect(get(activeModal)).toBe(MODAL.TAKE_POINT_2_LIVE);
    });

    test('tp4_last opens TAKE_POINT_4_LAST modal', () => {
        processCommand('tp4_last');
        expect(get(activeModal)).toBe(MODAL.TAKE_POINT_4_LAST);
    });

    test('tp4_live opens TAKE_POINT_4_LIVE modal', () => {
        processCommand('tp4_live');
        expect(get(activeModal)).toBe(MODAL.TAKE_POINT_4_LIVE);
    });

    test('gv1 opens GAMMON_VALUE_1 modal', () => {
        processCommand('gv1');
        expect(get(activeModal)).toBe(MODAL.GAMMON_VALUE_1);
    });

    test('gv2 opens GAMMON_VALUE_2 modal', () => {
        processCommand('gv2');
        expect(get(activeModal)).toBe(MODAL.GAMMON_VALUE_2);
    });

    test('gv4 opens GAMMON_VALUE_4 modal', () => {
        processCommand('gv4');
        expect(get(activeModal)).toBe(MODAL.GAMMON_VALUE_4);
    });

    test('tp2 opens TAKE_POINT_2 modal', () => {
        processCommand('tp2');
        expect(get(activeModal)).toBe(MODAL.TAKE_POINT_2);
    });

    test('tp4 opens TAKE_POINT_4 modal', () => {
        processCommand('tp4');
        expect(get(activeModal)).toBe(MODAL.TAKE_POINT_4);
    });

    // -- meta command --------------------------------------------------------
    test('meta opens the Metadata tab when database loaded', () => {
        databasePathStore.set('/some/path.db');
        processCommand('meta');
        expect(callbacks.toggleMetadataPanel).toHaveBeenCalled();
    });

    test('meta shows error when no database loaded', () => {
        databasePathStore.set('');
        processCommand('meta');
        expect(get(activeModal)).toBeNull();
        expect(statusText()).toBe('No database loaded.');
    });

    // -- search commands -----------------------------------------------------
    test('s command calls onLoadPositionsByFilters in NORMAL mode', () => {
        statusBarModeStore.set('NORMAL');
        processCommand('s');
        expect(callbacks.onLoadPositionsByFilters).toHaveBeenCalled();
    });

    test('s with filters calls onLoadPositionsByFilters', () => {
        statusBarModeStore.set('NORMAL');
        processCommand('s p>30 w>50');
        expect(callbacks.onLoadPositionsByFilters).toHaveBeenCalled();
        const args = callbacks.onLoadPositionsByFilters.mock.calls[0];
        // First arg is the filters array
        expect(args[0]).toEqual(['p>30', 'w>50']);
    });

    test('s command blocked outside NORMAL/EDIT mode', () => {
        statusBarModeStore.set('MATCH');
        processCommand('s p>30');
        expect(callbacks.onLoadPositionsByFilters).not.toHaveBeenCalled();
        expect(statusText()).toBe('Search requires NORMAL or EDIT mode.');
    });

    test('s in EDIT mode is allowed', () => {
        statusBarModeStore.set('EDIT');
        processCommand('s');
        expect(callbacks.onLoadPositionsByFilters).toHaveBeenCalled();
    });

    // -- sub-search commands -------------------------------------------------
    test('ss command calls onLoadPositionsByFilters with current IDs', () => {
        statusBarModeStore.set('NORMAL');
        positionsStore.set([{ id: 10 }, { id: 20 }, { id: 30 }]);
        processCommand('ss');
        expect(callbacks.onLoadPositionsByFilters).toHaveBeenCalled();
    });

    test('ss with filters passes parsed filters', () => {
        statusBarModeStore.set('NORMAL');
        positionsStore.set([{ id: 1 }, { id: 2 }]);
        processCommand('ss p>30');
        expect(callbacks.onLoadPositionsByFilters).toHaveBeenCalled();
    });

    test('ss with no positions shows error', () => {
        statusBarModeStore.set('NORMAL');
        positionsStore.set([]);
        processCommand('ss');
        expect(statusText()).toBe('No current results to search in.');
    });

    test('ss blocked outside NORMAL/EDIT mode', () => {
        statusBarModeStore.set('MATCH');
        positionsStore.set([{ id: 1 }]);
        processCommand('ss p>30');
        expect(callbacks.onLoadPositionsByFilters).not.toHaveBeenCalled();
        expect(statusText()).toBe('Search in results is not available in current mode.');
    });
});
