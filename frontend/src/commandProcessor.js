import { get } from 'svelte/store';
import { commentTextStore, currentPositionIndexStore, statusBarModeStore, statusBarTextStore, logEntriesStore, addLogEntry } from './stores/uiStore';
import { showMetModalStore, showTakePoint2LastModalStore, showTakePoint2LiveModalStore, showTakePoint4LastModalStore, showTakePoint4LiveModalStore, showGammonValue1ModalStore, showGammonValue2ModalStore, showGammonValue4ModalStore, showMetadataModalStore, showTakePoint2ModalStore, showTakePoint4ModalStore } from './stores/uiStore';
import { positionsStore, positionStore } from './stores/positionStore';
import { databaseLoadedStore } from './stores/databaseStore';
import { commandHistoryStore } from './stores/commandHistoryStore';
import { searchHistoryStore } from './stores/searchHistoryStore';
import { SaveComment, Migrate_1_0_0_to_1_1_0, ClearCommandHistory } from '../wailsjs/go/main/Database.js';
import { SaveSearchHistory } from '../wailsjs/go/main/Database.js';
import { Migrate_1_1_0_to_1_2_0, Migrate_1_2_0_to_1_3_0 } from '../wailsjs/go/main/Database.js';

let callbacks = {};

export function initCommandProcessor(cbs) {
    callbacks = cbs;
}

export function processCommand(command) {
    const positions = get(positionsStore);
    const databaseLoaded = get(databaseLoadedStore);

    const match = command.match(/^(\d+)$/);
    if (match) {
        const positionNumber = parseInt(match[1], 10);
        let index;
        if (positionNumber < 1) {
            index = 0;
        } else if (positionNumber > positions.length) {
            index = positions.length - 1;
        } else {
            index = positionNumber - 1;
        }
        currentPositionIndexStore.set(index);
        addLogEntry(`Go to position ${index + 1}`, 'result');
    } else if (command.startsWith('#')) {
        const tags = command.slice(1).trim();
        insertTags(tags);
        addLogEntry(`Tags added: ${tags}`, 'result');
    } else if (command === 'new' || command === 'ne' || command === 'n') {
        callbacks.onNewDatabase?.();
    } else if (command === 'open' || command === 'op' || command === 'o') {
        callbacks.onOpenDatabase?.();
    } else if (command === 'import_db' || command === 'idb') {
        callbacks.onImportDatabase?.();
    } else if (command === 'export_db' || command === 'edb') {
        callbacks.onExportDatabase?.();
    } else if (command === 'import' || command === 'i') {
        callbacks.importPosition?.();
    } else if (command === 'write' || command === 'wr' || command === 'w') {
        callbacks.onSavePosition?.();
    } else if (command === 'write!' || command === 'wr!' || command === 'w!') {
        callbacks.onUpdatePosition?.();
    } else if (command === 'delete' || command === 'del' || command === 'd') {
        callbacks.onDeletePosition?.();
    } else if (command === 'list' || command === 'l') {
        callbacks.onToggleAnalysis?.();
    } else if (command === 'comment' || command === 'co') {
        callbacks.onToggleComment?.();
    } else if (command === 'quit' || command === 'q') {
        callbacks.exitApp?.();
    } else if (command === 'help' || command === 'he' || command === 'h') {
        callbacks.onToggleHelp?.();
    } else if (command === 'e') {
        callbacks.onLoadAllPositions?.();
    } else if (command.startsWith('ss')) {
        handleSubSearch(command, positions);
    } else if (command.startsWith('s')) {
        handleSearch(command);
    } else if (command === 'filter' || command === 'fl') {
        callbacks.toggleFilterLibraryPanel?.();
    } else if (command === 'history' || command === 'hi') {
        callbacks.toggleSearchHistoryPanel?.();
    } else if (command === 'match' || command === 'ma') {
        callbacks.toggleMatchPanel?.();
    } else if (command === 'collection' || command === 'coll') {
        callbacks.toggleCollectionPanel?.();
    } else if (command === 'epc') {
        callbacks.toggleEPCMode?.();
    } else if (command === 'm') {
        callbacks.toggleMatchMode?.();
    } else if (command === 'met') {
        showMetModalStore.set(true);
    } else if (command === 'tp2_last') {
        showTakePoint2LastModalStore.set(true);
    } else if (command === 'tp2_live') {
        showTakePoint2LiveModalStore.set(true);
    } else if (command === 'tp4_last') {
        showTakePoint4LastModalStore.set(true);
    } else if (command === 'tp4_live') {
        showTakePoint4LiveModalStore.set(true);
    } else if (command === 'gv1') {
        showGammonValue1ModalStore.set(true);
    } else if (command === 'gv2') {
        showGammonValue2ModalStore.set(true);
    } else if (command === 'gv4') {
        showGammonValue4ModalStore.set(true);
    } else if (command === 'meta') {
        if (databaseLoaded) {
            showMetadataModalStore.set(true);
        } else {
            statusBarTextStore.set('No database loaded.');
        }
    } else if (command === 'tp2') {
        showTakePoint2ModalStore.set(true);
    } else if (command === 'tp4') {
        showTakePoint4ModalStore.set(true);
    } else if (command === 'migrate_from_1_0_to_1_1') {
        Migrate_1_0_0_to_1_1_0().then(() => {
            statusBarTextStore.set('Database migrated to version 1.1.0 successfully.');
        }).catch(error => {
            console.error('Error migrating database:', error);
            statusBarTextStore.set('Error migrating database.');
        });
    } else if (command === 'migrate_from_1_1_to_1_2') {
        Migrate_1_1_0_to_1_2_0().then(() => {
            statusBarTextStore.set('Database migrated to version 1.2.0 successfully.');
        }).catch(error => {
            console.error('Error migrating database:', error);
            statusBarTextStore.set('Error migrating database.');
        });
    } else if (command === 'migrate_from_1_2_to_1_3') {
        Migrate_1_2_0_to_1_3_0().then(() => {
            statusBarTextStore.set('Database migrated to version 1.3.0 successfully.');
        }).catch(error => {
            console.error('Error migrating database:', error);
            statusBarTextStore.set('Error migrating database.');
        });
    } else if (command === 'cl' || command === 'clear') {
        ClearCommandHistory().then(() => {
            commandHistoryStore.set([]);
            logEntriesStore.set([]);
            statusBarTextStore.set('Command history cleared.');
        }).catch(error => {
            console.error('Error clearing command history:', error);
            statusBarTextStore.set('Error clearing command history.');
        });
    }
}

function handleSubSearch(command, positions) {
    const mode = get(statusBarModeStore);
    if (mode === 'NORMAL' || mode === 'EDIT') {
        if (positions.length === 0) {
            statusBarTextStore.set('No current results to search in.');
            return;
        }
        const currentIDs = positions.map(p => p.id).filter(id => id != null).join(',');
        const searchHistoryEntry = {
            command: command,
            position: JSON.stringify(get(positionStore)),
            timestamp: Date.now()
        };
        searchHistoryStore.update(history => {
            const newHistory = [searchHistoryEntry, ...history].slice(0, 100);
            return newHistory;
        });
        SaveSearchHistory(command, JSON.stringify(get(positionStore))).catch(err => {
            console.error('Error saving search history:', err);
        });

        if (command === 'ss') {
            callbacks.onLoadPositionsByFilters?.([], false, false, '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', false, false, '', '', '', '', '', '', false, false, '', command, '', '', currentIDs);
        } else {
            const filters = command.slice(2).trim().split(' ').map(filter => filter.trim());
            const parsedFilters = parseFilters(filters, command);
            callbacks.onLoadPositionsByFilters?.(
                filters, parsedFilters.includeCube, parsedFilters.includeScore,
                parsedFilters.pipCountFilter, parsedFilters.winRateFilter, parsedFilters.gammonRateFilter,
                parsedFilters.backgammonRateFilter, parsedFilters.player2WinRateFilter, parsedFilters.player2GammonRateFilter,
                parsedFilters.player2BackgammonRateFilter, parsedFilters.player1CheckerOffFilter, parsedFilters.player2CheckerOffFilter,
                parsedFilters.player1BackCheckerFilter, parsedFilters.player2BackCheckerFilter,
                parsedFilters.player1CheckerInZoneFilter, parsedFilters.player2CheckerInZoneFilter,
                parsedFilters.searchText, parsedFilters.player1AbsolutePipCountFilter, parsedFilters.equityFilter,
                parsedFilters.decisionTypeFilter, parsedFilters.diceRollFilter, parsedFilters.movePatternFilter,
                parsedFilters.dateFilter, parsedFilters.player1OutfieldBlotFilter, parsedFilters.player2OutfieldBlotFilter,
                parsedFilters.player1JanBlotFilter, parsedFilters.player2JanBlotFilter,
                parsedFilters.noContactFilter, parsedFilters.mirrorPositionFilter, parsedFilters.moveErrorFilter,
                command, parsedFilters.matchIDsFilter, parsedFilters.tournamentIDsFilter, currentIDs
            );
        }
    } else {
        statusBarTextStore.set('Search in results is not available in current mode.');
    }
}

function handleSearch(command) {
    const mode = get(statusBarModeStore);
    if (mode === 'NORMAL' || mode === 'EDIT') {
        const searchHistoryEntry = {
            command: command,
            position: JSON.stringify(get(positionStore)),
            timestamp: Date.now()
        };
        searchHistoryStore.update(history => {
            const newHistory = [searchHistoryEntry, ...history].slice(0, 100);
            return newHistory;
        });
        SaveSearchHistory(command, JSON.stringify(get(positionStore))).catch(err => {
            console.error('Error saving search history:', err);
        });

        if (command === 's') {
            callbacks.onLoadPositionsByFilters?.([], false, false, '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', false, false, '', '', '', '', '', '', false, false, '', command, '', '');
        } else {
            const filters = command.slice(1).trim().split(' ').map(filter => filter.trim());
            const parsedFilters = parseFilters(filters, command);
            callbacks.onLoadPositionsByFilters?.(
                filters, parsedFilters.includeCube, parsedFilters.includeScore,
                parsedFilters.pipCountFilter, parsedFilters.winRateFilter, parsedFilters.gammonRateFilter,
                parsedFilters.backgammonRateFilter, parsedFilters.player2WinRateFilter, parsedFilters.player2GammonRateFilter,
                parsedFilters.player2BackgammonRateFilter, parsedFilters.player1CheckerOffFilter, parsedFilters.player2CheckerOffFilter,
                parsedFilters.player1BackCheckerFilter, parsedFilters.player2BackCheckerFilter,
                parsedFilters.player1CheckerInZoneFilter, parsedFilters.player2CheckerInZoneFilter,
                parsedFilters.searchText, parsedFilters.player1AbsolutePipCountFilter, parsedFilters.equityFilter,
                parsedFilters.decisionTypeFilter, parsedFilters.diceRollFilter, parsedFilters.movePatternFilter,
                parsedFilters.dateFilter, parsedFilters.player1OutfieldBlotFilter, parsedFilters.player2OutfieldBlotFilter,
                parsedFilters.player1JanBlotFilter, parsedFilters.player2JanBlotFilter,
                parsedFilters.noContactFilter, parsedFilters.mirrorPositionFilter, parsedFilters.moveErrorFilter,
                command, parsedFilters.matchIDsFilter, parsedFilters.tournamentIDsFilter
            );
        }
    } else {
        statusBarTextStore.set('Search requires NORMAL or EDIT mode.');
    }
}

export function parseFilters(filters, command) {
    const includeCube = filters.includes('cube') || filters.includes('cu') || filters.includes('c') || filters.includes('cub');
    const includeScore = filters.includes('score') || filters.includes('sco') || filters.includes('sc') || filters.includes('s');
    const noContactFilter = filters.includes('nc');
    const decisionTypeFilter = filters.includes('d');
    const diceRollFilter = filters.includes('D');
    const mirrorPositionFilter = filters.includes('M');
    const pipCountFilter = filters.find(f => typeof f === 'string' && (f.startsWith('p>') || f.startsWith('p<') || f.startsWith('p')));
    const winRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('w>') || f.startsWith('w<') || f.startsWith('w')));
    const gammonRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('g>') || f.startsWith('g<') || f.startsWith('g')));
    const backgammonRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('b>') || f.startsWith('b<') || (f.startsWith('b') && !f.startsWith('bo'))) && !f.startsWith('bj'));
    const player2WinRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('W>') || f.startsWith('W<') || f.startsWith('W')));
    const player2GammonRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('G>') || f.startsWith('G<') || f.startsWith('G')));
    const player2BackgammonRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('B>') || f.startsWith('B<') || f.startsWith('B') && !f.startsWith('BO')) && !f.startsWith('BJ'));
    let player1CheckerOffFilter = filters.find(f => typeof f === 'string' && (f.startsWith('o>') || f.startsWith('o<') || f.startsWith('o')));
    if (player1CheckerOffFilter && !player1CheckerOffFilter.includes(',') && !player1CheckerOffFilter.includes('>') && !player1CheckerOffFilter.includes('<')) {
        player1CheckerOffFilter = `${player1CheckerOffFilter},${player1CheckerOffFilter.slice(1)}`;
    }
    let player2CheckerOffFilter = filters.find(f => typeof f === 'string' && (f.startsWith('O>') || f.startsWith('O<') || f.startsWith('O')));
    if (player2CheckerOffFilter && !player2CheckerOffFilter.includes(',') && !player2CheckerOffFilter.includes('>') && !player2CheckerOffFilter.includes('<')) {
        player2CheckerOffFilter = `${player2CheckerOffFilter},${player2CheckerOffFilter.slice(1)}`;
    }
    let player1BackCheckerFilter = filters.find(f => typeof f === 'string' && (f.startsWith('k>') || f.startsWith('k<') || f.startsWith('k')));
    if (player1BackCheckerFilter && !player1BackCheckerFilter.includes(',') && !player1BackCheckerFilter.includes('>') && !player1BackCheckerFilter.includes('<')) {
        player1BackCheckerFilter = `${player1BackCheckerFilter},${player1BackCheckerFilter.slice(1)}`;
    }
    let player2BackCheckerFilter = filters.find(f => typeof f === 'string' && (f.startsWith('K>') || f.startsWith('K<') || f.startsWith('K')));
    if (player2BackCheckerFilter && !player2BackCheckerFilter.includes(',') && !player2BackCheckerFilter.includes('>') && !player2BackCheckerFilter.includes('<')) {
        player2BackCheckerFilter = `${player2BackCheckerFilter},${player2BackCheckerFilter.slice(1)}`;
    }
    let player1CheckerInZoneFilter = filters.find(f => typeof f === 'string' && (f.startsWith('z>') || f.startsWith('z<') || f.startsWith('z')));
    if (player1CheckerInZoneFilter && !player1CheckerInZoneFilter.includes(',') && !player1CheckerInZoneFilter.includes('>') && !player1CheckerInZoneFilter.includes('<')) {
        player1CheckerInZoneFilter = `${player1CheckerInZoneFilter},${player1CheckerInZoneFilter.slice(1)}`;
    }
    let player2CheckerInZoneFilter = filters.find(f => typeof f === 'string' && (f.startsWith('Z>') || f.startsWith('Z<') || f.startsWith('Z')));
    if (player2CheckerInZoneFilter && !player2CheckerInZoneFilter.includes(',') && !player2CheckerInZoneFilter.includes('>') && !player2CheckerInZoneFilter.includes('<')) {
        player2CheckerInZoneFilter = `${player2CheckerInZoneFilter},${player2CheckerInZoneFilter.slice(1)}`;
    }
    const player1AbsolutePipCountFilter = filters.find(f => typeof f === 'string' && (f.startsWith('P>') || f.startsWith('P<') || f.startsWith('P')));
    const equityFilter = filters.find(f => typeof f === 'string' && (f.startsWith('e>') || f.startsWith('e<') || f.startsWith('e')));
    const dateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('T>') || f.startsWith('T<') || f.startsWith('T')));
    const movePatternMatch = command.match(/m["'][^"']*["']/);
    const movePatternFilter = movePatternMatch ? movePatternMatch[0] : '';
    const searchTextMatch = command.match(/t["'][^"']*["']/);
    const searchText = searchTextMatch ? searchTextMatch[0] : '';
    const player1OutfieldBlotFilter = filters.find(f => typeof f === 'string' && (f.startsWith('bo>') || f.startsWith('bo<') || f.startsWith('bo')));
    const player2OutfieldBlotFilter = filters.find(f => typeof f === 'string' && (f.startsWith('BO>') || f.startsWith('BO<') || f.startsWith('BO')));
    const player1JanBlotFilter = filters.find(f => typeof f === 'string' && (f.startsWith('bj>') || f.startsWith('bj<') || f.startsWith('bj')));
    const player2JanBlotFilter = filters.find(f => typeof f === 'string' && (f.startsWith('BJ>') || f.startsWith('BJ<') || f.startsWith('BJ')));
    const moveErrorFilter = filters.find(f => typeof f === 'string' && (f.startsWith('E>') || f.startsWith('E<') || (f.startsWith('E') && /^E\d/.test(f))));

    const matchIDTokens = filters.filter(f => typeof f === 'string' && /^ma\d/.test(f));
    let matchIDsFilter = '';
    if (matchIDTokens.length > 0) {
        const parts = matchIDTokens.map(token => token.slice(2));
        matchIDsFilter = parts.join(';');
    }

    const tournamentIDTokens = filters.filter(f => typeof f === 'string' && /^tn\d/.test(f));
    let tournamentIDsFilter = '';
    if (tournamentIDTokens.length > 0) {
        const parts = tournamentIDTokens.map(token => token.slice(2));
        tournamentIDsFilter = parts.join(';');
    }

    return {
        includeCube, includeScore, noContactFilter, decisionTypeFilter, diceRollFilter, mirrorPositionFilter,
        pipCountFilter, winRateFilter, gammonRateFilter, backgammonRateFilter,
        player2WinRateFilter, player2GammonRateFilter, player2BackgammonRateFilter,
        player1CheckerOffFilter, player2CheckerOffFilter,
        player1BackCheckerFilter, player2BackCheckerFilter,
        player1CheckerInZoneFilter, player2CheckerInZoneFilter,
        player1AbsolutePipCountFilter, equityFilter, dateFilter,
        movePatternFilter, searchText,
        player1OutfieldBlotFilter, player2OutfieldBlotFilter,
        player1JanBlotFilter, player2JanBlotFilter, moveErrorFilter,
        matchIDsFilter, tournamentIDsFilter
    };
}

function insertTags(tags) {
    commentTextStore.update(text => {
        const existingTags = new Set(text.match(/#[^\s#]+/g) || []);
        const newTags = tags.split(' ').filter(tag => !existingTags.has(tag));
        const updatedText = `${newTags.join(' ')}\n${text}`;
        setTimeout(() => {
            const textAreaEl = document.getElementById('commentTextArea');
            if (textAreaEl) {
                /** @type {HTMLTextAreaElement} */ (textAreaEl).setSelectionRange(updatedText.length, updatedText.length);
                textAreaEl.focus();
            }
        }, 0);
        SaveComment(get(currentPositionIndexStore), updatedText);
        return updatedText;
    });
}
