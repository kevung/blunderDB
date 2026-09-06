import { get } from 'svelte/store';
import { commentTextStore, currentPositionIndexStore, statusBarModeStore, statusBarTextStore, openModal, MODAL } from './stores/uiStore';
import { positionsStore, positionStore } from './stores/positionStore';
import { databaseLoadedStore } from './stores/databaseStore';
import { commandHistoryStore } from './stores/commandHistoryStore';
import { searchHistoryStore, MAX_SEARCH_HISTORY } from './stores/searchHistoryStore';
import { excludePositionHistoryJSON } from './stores/searchExcludePositionStore';
import { SaveComment, ClearCommandHistory } from '../wailsjs/go/database/Database.js';
import { SaveSearchHistory } from '../wailsjs/go/database/Database.js';
import { logger } from './utils/logger.js';
// NOTE: status messages are emitted as tMsg() descriptors so the status bar can
// re-translate them live when the language changes.
import { tMsg } from './i18n';
// The search-token grammar (parseSearchTokens) and its quote-stripping helper
// live in searchFilterService.js, shared with the "retour" replay path
// (parseSearchCommand) — see that module's doc comment and #203. Re-exported
// here unchanged so existing importers (this file's own tests, ankiService.js)
// do not have to change their import path.
import { parseSearchTokens, stripQuotedTokens } from './services/searchFilterService.js';
export { stripQuotedTokens };

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
        statusBarTextStore.set(tMsg('commands.goToPosition', { n: index + 1 }));
    } else if (command.startsWith('#')) {
        const tags = command.slice(1).trim();
        insertTags(tags);
        statusBarTextStore.set(tMsg('commands.tagsAdded', { tags }));
    } else if (command === 'new' || command === 'ne' || command === 'n') {
        callbacks.onNewDatabase?.();
    } else if (command === 'open' || command === 'op' || command === 'o') {
        callbacks.onOpenDatabase?.();
    } else if (command === 'import_db' || command === 'idb') {
        callbacks.onImportDatabase?.();
    } else if (command === 'export_db' || command === 'edb') {
        callbacks.onExportDatabase?.();
    } else if (command.startsWith('import ')) {
        // `import <identifiant>` : le même verbe qu'`import` tout court, avec
        // un argument. Sans argument on choisit un fichier ; avec, on lit
        // l'identifiant — le cas où il arrive d'ailleurs que du
        // presse-papier : d'un message, d'un forum, d'un script (#262).
        // Testé AVANT la forme exacte, qui ne peut pas le capter.
        callbacks.onImportIdentifier?.(command.slice('import '.length));
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
    } else if (command === 'trash') {
        openModal(MODAL.TRASH);
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
    } else if (command === 'ss' || command.startsWith('ss ')) {
        handleSearchCommand(command, positions, { isSubSearch: true });
    } else if (command === 's' || command.startsWith('s ')) {
        handleSearchCommand(command, positions, { isSubSearch: false });
    } else if (command === 'history' || command === 'hi') {
        callbacks.focusSearchTab?.();
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
    } else if (command === 'cm') {
        openModal(MODAL.CUBE_MATRIX);
    } else if (command === 'tags') {
        openModal(MODAL.TAGS);
    } else if (command === 'log') {
        openModal(MODAL.LOG);
    } else if (command.startsWith('ask ')) {
        // `ask <phrase>` : la phrase est traduite en jetons VISIBLES, écrits
        // dans la barre de commande, que l'utilisateur relit et lance (#283).
        callbacks.onIntent?.(command.slice('ask '.length));
    } else if (command === 'ask') {
        callbacks.onIntent?.('');
    } else if (command.startsWith('like ')) {
        // `like <id>` : les positions les plus proches de celle-là (#293).
        // Testé AVANT la forme exacte, que la ligne suivante capte pour
        // prendre la position courante.
        callbacks.onSimilar?.(parseInt(command.slice('like '.length).trim(), 10));
    } else if (command === 'like') {
        callbacks.onSimilar?.(0);
    } else if (command.startsWith('train ')) {
        // `train <exercice>` : pips, epc, tp. Testé AVANT la forme exacte, que
        // la ligne suivante capte pour ouvrir le choix (#273).
        callbacks.onTraining?.(command.slice('train '.length).trim());
    } else if (command === 'train') {
        callbacks.onTraining?.('');
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
    } else if (command === 'cl' || command === 'clear') {
        ClearCommandHistory()
            .then(() => {
                commandHistoryStore.set([]);
                statusBarTextStore.set(tMsg('commands.commandHistoryCleared'));
            })
            .catch((error) => {
                logger.error('Error clearing command history:', error);
                statusBarTextStore.set(tMsg('commands.errorClearingHistory'));
            });
    } else {
        statusBarTextStore.set(tMsg('commands.unknown', { command }));
    }
}

// Handles both `s...` (search) and `ss...` (sub-search, restricted to the IDs of the
// currently displayed positions) — same body throughout, differing only in the command
// prefix length, the bare-command spelling, whether positions must be non-empty first, and
// which IDs (if any) get sent as restrictToPositionIDs.
function handleSearchCommand(command, positions, { isSubSearch }) {
    const mode = get(statusBarModeStore);
    if (mode !== 'NORMAL' && mode !== 'EDIT') {
        statusBarTextStore.set(tMsg(isSubSearch ? 'commands.subSearchModeUnavailable' : 'commands.searchRequiresMode'));
        return;
    }

    let currentIDs = '';
    if (isSubSearch) {
        if (positions.length === 0) {
            statusBarTextStore.set(tMsg('commands.noResultsToSearchIn'));
            return;
        }
        currentIDs = positions.ids.filter((id) => id != null).join(',');
    }

    const searchHistoryEntry = {
        command: command,
        position: JSON.stringify(get(positionStore)),
        timestamp: Date.now()
    };
    searchHistoryStore.update((history) => {
        const newHistory = [searchHistoryEntry, ...history].slice(0, MAX_SEARCH_HISTORY);
        return newHistory;
    });
    SaveSearchHistory(command, JSON.stringify(get(positionStore)), excludePositionHistoryJSON()).catch((err) => {
        logger.error('Error saving search history:', err);
    });

    const bareCommand = isSubSearch ? 'ss' : 's';
    if (command === bareCommand) {
        callbacks.onLoadPositionsByFilters?.({
            searchCommand: command,
            ...(isSubSearch ? { restrictToPositionIDs: currentIDs } : {})
        });
    } else {
        const prefixLength = isSubSearch ? 2 : 1;
        const filters = stripQuotedTokens(command.slice(prefixLength).trim())
            .split(' ')
            .map((filter) => filter.trim());
        const parsedFilters = parseFilters(filters, command);
        callbacks.onLoadPositionsByFilters?.({
            filters,
            ...parsedFilters,
            searchCommand: command,
            restrictToPositionIDs: isSubSearch ? currentIDs : ''
        });
    }
}

/**
 * Parse the `s …`/`ss …` filter tokens typed on the command line (the
 * "aller" path). A thin wrapper over {@link parseSearchTokens}
 * (searchFilterService.js) — see that function's doc comment for the shared
 * grammar and #203, the bug this split used to hide.
 *
 * @param {string[]} filters - filter tokens, already split and quote-stripped by the caller.
 * @param {string} command - the raw command, used to recover quoted values (`t"…"`, `m"…"`, `pl"…"`).
 * @returns {object} the parsed filter values, under their long (backend) field names.
 */
export function parseFilters(filters, command) {
    return parseSearchTokens(filters, command);
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
        const positionId = positionsStore.idAt(get(currentPositionIndexStore));
        if (positionId != null) {
            SaveComment(positionId, updatedText);
        }
        return updatedText;
    });
}
