import { get } from 'svelte/store';
import { commentTextStore, currentPositionIndexStore, statusBarModeStore, statusBarTextStore, logEntriesStore, addLogEntry, openModal, MODAL } from './stores/uiStore';
import { positionsStore, positionStore } from './stores/positionStore';
import { databaseLoadedStore } from './stores/databaseStore';
import { commandHistoryStore } from './stores/commandHistoryStore';
import { searchHistoryStore } from './stores/searchHistoryStore';
import { excludePositionHistoryJSON } from './stores/searchExcludePositionStore';
import { SaveComment, Migrate_1_0_0_to_1_1_0, ClearCommandHistory } from '../wailsjs/go/database/Database.js';
import { SaveSearchHistory } from '../wailsjs/go/database/Database.js';
import { Migrate_1_1_0_to_1_2_0, Migrate_1_2_0_to_1_3_0 } from '../wailsjs/go/database/Database.js';
import { logger } from './utils/logger.js';
// NOTE: these UI messages are translated at emission time via the non-reactive
// `translate` helper. Already-displayed messages do not retranslate on language change.
import { translate, tMsg } from './i18n';

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
        addLogEntry(translate('commands.goToPosition', { n: index + 1 }), 'result');
    } else if (command.startsWith('#')) {
        const tags = command.slice(1).trim();
        insertTags(tags);
        addLogEntry(translate('commands.tagsAdded', { tags }), 'result');
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
    } else if (command === 'tutorial' || command === 'tour') {
        openModal(MODAL.TOUR);
    } else if (command === 'demo') {
        callbacks.onLoadDemo?.();
    } else if (command === 'e') {
        callbacks.onLoadAllPositions?.();
    } else if (command === 'stats' || command === 'st') {
        callbacks.onToggleStats?.();
    } else if (command === 'blunders' || command === 'bl' || command.startsWith('blunders ') || command.startsWith('bl ')) {
        // Optional count: `bl 50` loads the 50 worst; bare `bl` keeps the default.
        const n = parseInt(command.split(/\s+/)[1], 10);
        callbacks.onLoadBlunders?.(Number.isInteger(n) && n > 0 ? n : undefined);
    } else if (command.startsWith('ss')) {
        handleSubSearch(command, positions);
    } else if (command.startsWith('s')) {
        handleSearch(command);
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
        openModal(MODAL.MET);
    } else if (command === 'tp2_last') {
        openModal(MODAL.TAKE_POINT_2_LAST);
    } else if (command === 'tp2_live') {
        openModal(MODAL.TAKE_POINT_2_LIVE);
    } else if (command === 'tp4_last') {
        openModal(MODAL.TAKE_POINT_4_LAST);
    } else if (command === 'tp4_live') {
        openModal(MODAL.TAKE_POINT_4_LIVE);
    } else if (command === 'gv1') {
        openModal(MODAL.GAMMON_VALUE_1);
    } else if (command === 'gv2') {
        openModal(MODAL.GAMMON_VALUE_2);
    } else if (command === 'gv4') {
        openModal(MODAL.GAMMON_VALUE_4);
    } else if (command === 'meta') {
        if (databaseLoaded) {
            callbacks.toggleMetadataPanel?.();
        } else {
            statusBarTextStore.set(tMsg('commands.noDatabaseLoaded'));
        }
    } else if (command === 'tp2') {
        openModal(MODAL.TAKE_POINT_2);
    } else if (command === 'tp4') {
        openModal(MODAL.TAKE_POINT_4);
    } else if (command === 'migrate_from_1_0_to_1_1') {
        Migrate_1_0_0_to_1_1_0()
            .then(() => {
                statusBarTextStore.set(tMsg('commands.dbMigrated', { version: '1.1.0' }));
            })
            .catch((error) => {
                logger.error('Error migrating database:', error);
                statusBarTextStore.set(tMsg('commands.errorMigrating'));
            });
    } else if (command === 'migrate_from_1_1_to_1_2') {
        Migrate_1_1_0_to_1_2_0()
            .then(() => {
                statusBarTextStore.set(tMsg('commands.dbMigrated', { version: '1.2.0' }));
            })
            .catch((error) => {
                logger.error('Error migrating database:', error);
                statusBarTextStore.set(tMsg('commands.errorMigrating'));
            });
    } else if (command === 'migrate_from_1_2_to_1_3') {
        Migrate_1_2_0_to_1_3_0()
            .then(() => {
                statusBarTextStore.set(tMsg('commands.dbMigrated', { version: '1.3.0' }));
            })
            .catch((error) => {
                logger.error('Error migrating database:', error);
                statusBarTextStore.set(tMsg('commands.errorMigrating'));
            });
    } else if (command === 'cl' || command === 'clear') {
        ClearCommandHistory()
            .then(() => {
                commandHistoryStore.set([]);
                logEntriesStore.set([]);
                statusBarTextStore.set(tMsg('commands.commandHistoryCleared'));
            })
            .catch((error) => {
                logger.error('Error clearing command history:', error);
                statusBarTextStore.set(tMsg('commands.errorClearingHistory'));
            });
    }
}

function handleSubSearch(command, positions) {
    const mode = get(statusBarModeStore);
    if (mode === 'NORMAL' || mode === 'EDIT') {
        if (positions.length === 0) {
            statusBarTextStore.set(tMsg('commands.noResultsToSearchIn'));
            return;
        }
        const currentIDs = positions
            .map((p) => p.id)
            .filter((id) => id != null)
            .join(',');
        const searchHistoryEntry = {
            command: command,
            position: JSON.stringify(get(positionStore)),
            timestamp: Date.now()
        };
        searchHistoryStore.update((history) => {
            const newHistory = [searchHistoryEntry, ...history].slice(0, 100);
            return newHistory;
        });
        SaveSearchHistory(command, JSON.stringify(get(positionStore)), excludePositionHistoryJSON()).catch((err) => {
            logger.error('Error saving search history:', err);
        });

        if (command === 'ss') {
            callbacks.onLoadPositionsByFilters?.({
                searchCommand: command,
                restrictToPositionIDs: currentIDs
            });
        } else {
            const filters = stripQuotedTokens(command.slice(2).trim())
                .split(' ')
                .map((filter) => filter.trim());
            const parsedFilters = parseFilters(filters, command);
            callbacks.onLoadPositionsByFilters?.({
                filters,
                ...parsedFilters,
                searchCommand: command,
                restrictToPositionIDs: currentIDs
            });
        }
    } else {
        statusBarTextStore.set(tMsg('commands.subSearchModeUnavailable'));
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
        searchHistoryStore.update((history) => {
            const newHistory = [searchHistoryEntry, ...history].slice(0, 100);
            return newHistory;
        });
        SaveSearchHistory(command, JSON.stringify(get(positionStore)), excludePositionHistoryJSON()).catch((err) => {
            logger.error('Error saving search history:', err);
        });

        if (command === 's') {
            callbacks.onLoadPositionsByFilters?.({ searchCommand: command });
        } else {
            const filters = stripQuotedTokens(command.slice(1).trim())
                .split(' ')
                .map((filter) => filter.trim());
            const parsedFilters = parseFilters(filters, command);
            callbacks.onLoadPositionsByFilters?.({
                filters,
                ...parsedFilters,
                searchCommand: command,
                restrictToPositionIDs: ''
            });
        }
    } else {
        statusBarTextStore.set(tMsg('commands.searchRequiresMode'));
    }
}

// Quoted filter values — pl"…" (player), m"…" (move pattern) and t"…" (search
// text / comment) — may contain spaces. Their values are recovered later via
// regexes on the raw command, but the naive `.split(' ')` that builds the token
// array would tear a multi-word value into loose words (`t"big win"` → `t"big`,
// `win"`; `t"a b c"` → `t"a`, `b`, `c"`) and those words get misclassified as
// range filters (`win"` → winRateFilter, the bare `b` → backgammonRateFilter),
// silently zeroing the result set. Strip the whole quoted region before splitting
// so no interior word survives. Both quote styles are supported.
export function stripQuotedTokens(str) {
    return str.replace(/(?:pl|m|t)["'][^"']*["']/g, ' ');
}

export function parseFilters(filters, command) {
    const includeCube = filters.includes('cube') || filters.includes('cu') || filters.includes('c') || filters.includes('cub');
    const includeScore = filters.includes('score') || filters.includes('sco') || filters.includes('sc') || filters.includes('s');
    const noContactFilter = filters.includes('nc');
    const decisionTypeFilter = filters.includes('d');
    const diceRollFilter = filters.includes('D') || filters.includes('D1');
    const diceRollMode = filters.includes('D1') ? 'first' : 'both';
    // `xD65` excludes the 6-5 roll (order-insensitive); repeatable (`xD65 xD54`).
    // Unlike `D`, the value is inline in the token, not read from the board.
    // Joined into a ";"-separated string for the backend (ExceptDiceFilter).
    const exceptDiceFilter = filters
        .filter((f) => typeof f === 'string' && /^xD[1-6][1-6]$/.test(f))
        .map((f) => f.slice(2))
        .join(';');
    const mirrorPositionFilter = filters.includes('M');
    // Positions the user imported on their own rather than inside a match.
    // An exact match, so it does not collide with the id<ids> token.
    const individuallyImportedFilter = filters.includes('i');
    // 'x' marks that an exclusion ("Sauf") structure is active. The structure
    // itself is carried by the exclude board (store), like the include structure.
    const excludeStructure = filters.includes('x');
    // Exclude `pl"…"` (player filter) — it starts with 'p' but is not a pipcount.
    const pipCountFilter = filters.find((f) => typeof f === 'string' && !f.startsWith('pl') && (f.startsWith('p>') || f.startsWith('p<') || f.startsWith('p')));
    const winRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('w>') || f.startsWith('w<') || f.startsWith('w')));
    const gammonRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('g>') || f.startsWith('g<') || f.startsWith('g')));
    const backgammonRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('b>') || f.startsWith('b<') || (f.startsWith('b') && !f.startsWith('bo'))) && !f.startsWith('bj'));
    const player2WinRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('W>') || f.startsWith('W<') || f.startsWith('W')));
    const player2GammonRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('G>') || f.startsWith('G<') || f.startsWith('G')));
    const player2BackgammonRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('B>') || f.startsWith('B<') || (f.startsWith('B') && !f.startsWith('BO'))) && !f.startsWith('BJ'));
    let player1CheckerOffFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('o>') || f.startsWith('o<') || f.startsWith('o')));
    if (player1CheckerOffFilter && !player1CheckerOffFilter.includes(',') && !player1CheckerOffFilter.includes('>') && !player1CheckerOffFilter.includes('<')) {
        player1CheckerOffFilter = `${player1CheckerOffFilter},${player1CheckerOffFilter.slice(1)}`;
    }
    let player2CheckerOffFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('O>') || f.startsWith('O<') || f.startsWith('O')));
    if (player2CheckerOffFilter && !player2CheckerOffFilter.includes(',') && !player2CheckerOffFilter.includes('>') && !player2CheckerOffFilter.includes('<')) {
        player2CheckerOffFilter = `${player2CheckerOffFilter},${player2CheckerOffFilter.slice(1)}`;
    }
    let player1BackCheckerFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('k>') || f.startsWith('k<') || f.startsWith('k')));
    if (player1BackCheckerFilter && !player1BackCheckerFilter.includes(',') && !player1BackCheckerFilter.includes('>') && !player1BackCheckerFilter.includes('<')) {
        player1BackCheckerFilter = `${player1BackCheckerFilter},${player1BackCheckerFilter.slice(1)}`;
    }
    let player2BackCheckerFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('K>') || f.startsWith('K<') || f.startsWith('K')));
    if (player2BackCheckerFilter && !player2BackCheckerFilter.includes(',') && !player2BackCheckerFilter.includes('>') && !player2BackCheckerFilter.includes('<')) {
        player2BackCheckerFilter = `${player2BackCheckerFilter},${player2BackCheckerFilter.slice(1)}`;
    }
    let player1CheckerInZoneFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('z>') || f.startsWith('z<') || f.startsWith('z')));
    if (player1CheckerInZoneFilter && !player1CheckerInZoneFilter.includes(',') && !player1CheckerInZoneFilter.includes('>') && !player1CheckerInZoneFilter.includes('<')) {
        player1CheckerInZoneFilter = `${player1CheckerInZoneFilter},${player1CheckerInZoneFilter.slice(1)}`;
    }
    let player2CheckerInZoneFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('Z>') || f.startsWith('Z<') || f.startsWith('Z')));
    if (player2CheckerInZoneFilter && !player2CheckerInZoneFilter.includes(',') && !player2CheckerInZoneFilter.includes('>') && !player2CheckerInZoneFilter.includes('<')) {
        player2CheckerInZoneFilter = `${player2CheckerInZoneFilter},${player2CheckerInZoneFilter.slice(1)}`;
    }
    const player1AbsolutePipCountFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('P>') || f.startsWith('P<') || f.startsWith('P')));
    const equityFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('e>') || f.startsWith('e<') || f.startsWith('e')));
    const dateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('T>') || f.startsWith('T<') || f.startsWith('T')));
    const movePatternMatch = command.match(/m["'][^"']*["']/);
    const movePatternFilter = movePatternMatch ? movePatternMatch[0] : '';
    const searchTextMatch = command.match(/t["'][^"']*["']/);
    const searchText = searchTextMatch ? searchTextMatch[0] : '';
    // Player filter `pl"Name"` — matched on the raw command so names with spaces
    // survive (the space-split `filters` array would break them).
    const playerMatch = command.match(/pl["'][^"']*["']/);
    const playerFilter = playerMatch ? playerMatch[0] : '';
    const player1OutfieldBlotFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('bo>') || f.startsWith('bo<') || f.startsWith('bo')));
    const player2OutfieldBlotFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('BO>') || f.startsWith('BO<') || f.startsWith('BO')));
    const player1JanBlotFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('bj>') || f.startsWith('bj<') || f.startsWith('bj')));
    const player2JanBlotFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('BJ>') || f.startsWith('BJ<') || f.startsWith('BJ')));
    const moveErrorFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('E>') || f.startsWith('E<') || (f.startsWith('E') && /^E\d/.test(f))));

    const matchIDTokens = filters.filter((f) => typeof f === 'string' && /^ma\d/.test(f));
    let matchIDsFilter = '';
    if (matchIDTokens.length > 0) {
        const parts = matchIDTokens.map((token) => token.slice(2));
        matchIDsFilter = parts.join(';');
    }

    const tournamentIDTokens = filters.filter((f) => typeof f === 'string' && /^tn\d/.test(f));
    let tournamentIDsFilter = '';
    if (tournamentIDTokens.length > 0) {
        const parts = tournamentIDTokens.map((token) => token.slice(2));
        tournamentIDsFilter = parts.join(';');
    }

    // Position-id filter: `id12`, `id5,10` (range 5..10), or several `id` tokens
    // joined as an explicit list (e.g. `id5 id10`). Mirrors the ma/tn convention.
    const positionIDTokens = filters.filter((f) => typeof f === 'string' && /^id\d/.test(f));
    let positionIDsFilter = '';
    if (positionIDTokens.length > 0) {
        const parts = positionIDTokens.map((token) => token.slice(2));
        positionIDsFilter = parts.join(';');
    }

    return {
        includeCube,
        includeScore,
        noContactFilter,
        decisionTypeFilter,
        diceRollFilter,
        diceRollMode,
        exceptDiceFilter,
        mirrorPositionFilter,
        individuallyImportedFilter,
        excludeStructure,
        pipCountFilter,
        winRateFilter,
        gammonRateFilter,
        backgammonRateFilter,
        player2WinRateFilter,
        player2GammonRateFilter,
        player2BackgammonRateFilter,
        player1CheckerOffFilter,
        player2CheckerOffFilter,
        player1BackCheckerFilter,
        player2BackCheckerFilter,
        player1CheckerInZoneFilter,
        player2CheckerInZoneFilter,
        player1AbsolutePipCountFilter,
        equityFilter,
        dateFilter,
        movePatternFilter,
        searchText,
        player1OutfieldBlotFilter,
        player2OutfieldBlotFilter,
        player1JanBlotFilter,
        player2JanBlotFilter,
        moveErrorFilter,
        matchIDsFilter,
        tournamentIDsFilter,
        playerFilter,
        positionIDsFilter
    };
}

function insertTags(tags) {
    commentTextStore.update((text) => {
        const existingTags = new Set(text.match(/#[^\s#]+/g) || []);
        const newTags = tags.split(' ').filter((tag) => !existingTags.has(tag));
        const updatedText = `${newTags.join(' ')}\n${text}`;
        setTimeout(() => {
            const textAreaEl = document.getElementById('commentTextArea');
            if (textAreaEl) {
                /** @type {HTMLTextAreaElement} */ (textAreaEl).setSelectionRange(updatedText.length, updatedText.length);
                textAreaEl.focus();
            }
        }, 0);
        // NOTE: SaveComment expects the position *id*, not the array index.
        const positionId = get(positionsStore)[get(currentPositionIndexStore)]?.id;
        if (positionId != null) {
            SaveComment(positionId, updatedText);
        }
        return updatedText;
    });
}
