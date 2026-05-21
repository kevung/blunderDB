<script>
    import { logger } from '../utils/logger.js';
    import { onMount, onDestroy } from 'svelte';
    import { commentTextStore, currentPositionIndexStore, commandTextStore, statusBarModeStore, statusBarTextStore, activeModal, MODAL, openModal, closeModal } from '../stores/uiStore';
    import { SaveComment, Migrate_1_0_0_to_1_1_0, ClearCommandHistory } from '../../wailsjs/go/database/Database.js';
    import { positionsStore, positionStore } from '../stores/positionStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { commandHistoryStore } from '../stores/commandHistoryStore';
    import { searchHistoryStore } from '../stores/searchHistoryStore';
    import { LoadCommandHistory, SaveCommand, SaveSearchHistory } from '../../wailsjs/go/database/Database.js';
    import { Migrate_1_1_0_to_1_2_0, Migrate_1_2_0_to_1_3_0 } from '../../wailsjs/go/database/Database.js';

    let {
        onToggleHelp,
        onNewDatabase,
        onOpenDatabase,
        onImportDatabase,
        onExportDatabase,
        importPosition,
        onSavePosition,
        onUpdatePosition,
        onDeletePosition,
        onToggleAnalysis,
        onToggleComment,
        exitApp,
        onLoadPositionsByFilters,
        onLoadAllPositions,
        toggleFilterLibraryPanel,
        toggleSearchHistoryPanel,
        toggleMatchPanel,
        toggleCollectionPanel,
        toggleEPCMode,
        toggleMatchMode
    } = $props();
    let inputEl;

    let initialized = false;

    // Read-only mirrors of stores — always current when read in handlers/commands
    let positions = $derived($positionsStore);
    let databaseLoaded = $derived($databaseLoadedStore);
    let commandHistory = $derived($commandHistoryStore);

    let historyIndex = -1;

    // Focus + load history when command modal opens; remove click-outside listener when closed
    $effect(() => {
        const value = $activeModal;
        if (value === MODAL.COMMAND) {
            commandTextStore.set('');
            setTimeout(() => {
                inputEl?.focus();
            }, 0);
            window.addEventListener('click', handleClickOutside);
            LoadCommandHistory().then((history) => {
                commandHistoryStore.set((history || []).reverse());
                historyIndex = -1;
            });
        } else {
            window.removeEventListener('click', handleClickOutside);
        }
    });

    async function handleKeyDown(event) {
        event.stopPropagation();

        if ($activeModal === MODAL.COMMAND) {
            if (event.code === 'ArrowUp') {
                if (historyIndex < commandHistory.length - 1) {
                    historyIndex++;
                    commandTextStore.set(commandHistory[historyIndex]);
                    // Move cursor to the end without delay
                    requestAnimationFrame(() => {
                        inputEl.setSelectionRange(inputEl.value.length, inputEl.value.length);
                    });
                }
            } else if (event.code === 'ArrowDown') {
                if (historyIndex > 0) {
                    historyIndex--;
                    commandTextStore.set(commandHistory[historyIndex]);
                    // Move cursor to the end without delay
                    requestAnimationFrame(() => {
                        inputEl.setSelectionRange(inputEl.value.length, inputEl.value.length);
                    });
                } else {
                    historyIndex = -1;
                    commandTextStore.set('');
                }
            } else if (event.code === 'Backspace' && inputEl.value === '') {
                closeModal();
            } else if (event.code === 'Escape') {
                closeModal();
            } else if (event.code === 'Enter') {
                const command = inputEl.value.trim();
                logger.log('Command entered:', command); // Debugging log
                if (command) {
                    commandHistoryStore.update((history) => {
                        history = history || []; // Ensure history is an array
                        history.unshift(command); // Add the new command to the beginning
                        return history;
                    });
                    historyIndex = -1; // Reset the history index
                    SaveCommand(command); // Save the command to the database
                }
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
                } else if (command.startsWith('#')) {
                    const tags = command.slice(1).trim();
                    insertTags(tags);
                } else if (command === 'new' || command === 'ne' || command === 'n') {
                    onNewDatabase();
                } else if (command === 'open' || command === 'op' || command === 'o') {
                    onOpenDatabase();
                } else if (command === 'import_db' || command === 'idb') {
                    onImportDatabase();
                } else if (command === 'export_db' || command === 'edb') {
                    onExportDatabase();
                } else if (command === 'import' || command === 'i') {
                    importPosition();
                } else if (command === 'write' || command === 'wr' || command === 'w') {
                    onSavePosition();
                } else if (command === 'write!' || command === 'wr!' || command === 'w!') {
                    onUpdatePosition();
                } else if (command === 'delete' || command === 'del' || command === 'd') {
                    onDeletePosition();
                } else if (command === 'list' || command === 'l') {
                    onToggleAnalysis();
                } else if (command === 'comment' || command === 'co') {
                    logger.log('Toggling comment panel'); // Debugging log
                    onToggleComment();
                } else if (command === 'quit' || command === 'q') {
                    exitApp();
                } else if (command === 'help' || command === 'he' || command === 'h') {
                    onToggleHelp();
                } else if (command === 'e') {
                    onLoadAllPositions();
                } else if (command.startsWith('ss')) {
                    // Search in current results (from NORMAL mode after a prior search)
                    if ($statusBarModeStore === 'NORMAL' || $statusBarModeStore === 'EDIT') {
                        if (positions.length === 0) {
                            statusBarTextStore.set('No current results to search in.');
                        } else {
                            // Collect IDs of currently displayed positions
                            const currentIDs = positions
                                .map((p) => p.id)
                                .filter((id) => id != null)
                                .join(',');

                            // Save to search history
                            const searchHistoryEntry = {
                                command: command,
                                position: JSON.stringify($positionStore),
                                timestamp: Date.now()
                            };
                            searchHistoryStore.update((history) => {
                                const newHistory = [searchHistoryEntry, ...history].slice(0, 100);
                                return newHistory;
                            });
                            SaveSearchHistory(command, JSON.stringify($positionStore)).catch((err) => {
                                logger.error('Error saving search history:', err);
                            });

                            if (command === 'ss') {
                                // ss with no extra filters: just reload current results (no-op effectively)
                                onLoadPositionsByFilters(
                                    [],
                                    false,
                                    false,
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    false,
                                    false,
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    '',
                                    false,
                                    false,
                                    '',
                                    command,
                                    '',
                                    '',
                                    currentIDs
                                );
                            } else {
                                // Parse filters from the command after "ss"
                                const filters = command
                                    .slice(2)
                                    .trim()
                                    .split(' ')
                                    .map((filter) => filter.trim());
                                const includeCube = filters.includes('cube') || filters.includes('cu') || filters.includes('c') || filters.includes('cub');
                                const includeScore = filters.includes('score') || filters.includes('sco') || filters.includes('sc') || filters.includes('s');
                                const noContactFilter = filters.includes('nc');
                                const decisionTypeFilter = filters.includes('d');
                                const diceRollFilter = filters.includes('D');
                                const mirrorPositionFilter = filters.includes('M');
                                const pipCountFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('p>') || filter.startsWith('p<') || filter.startsWith('p')));
                                const winRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('w>') || filter.startsWith('w<') || filter.startsWith('w')));
                                const gammonRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('g>') || filter.startsWith('g<') || filter.startsWith('g')));
                                const backgammonRateFilter = filters.find(
                                    (filter) =>
                                        typeof filter === 'string' &&
                                        (filter.startsWith('b>') || filter.startsWith('b<') || (filter.startsWith('b') && !filter.startsWith('bo'))) &&
                                        !filter.startsWith('bj')
                                );
                                const player2WinRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('W>') || filter.startsWith('W<') || filter.startsWith('W')));
                                const player2GammonRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('G>') || filter.startsWith('G<') || filter.startsWith('G')));
                                const player2BackgammonRateFilter = filters.find(
                                    (filter) =>
                                        typeof filter === 'string' &&
                                        (filter.startsWith('B>') || filter.startsWith('B<') || (filter.startsWith('B') && !filter.startsWith('BO'))) &&
                                        !filter.startsWith('BJ')
                                );
                                let player1CheckerOffFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('o>') || filter.startsWith('o<') || filter.startsWith('o')));
                                if (player1CheckerOffFilter && !player1CheckerOffFilter.includes(',') && !player1CheckerOffFilter.includes('>') && !player1CheckerOffFilter.includes('<')) {
                                    player1CheckerOffFilter = `${player1CheckerOffFilter},${player1CheckerOffFilter.slice(1)}`;
                                }
                                let player2CheckerOffFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('O>') || filter.startsWith('O<') || filter.startsWith('O')));
                                if (player2CheckerOffFilter && !player2CheckerOffFilter.includes(',') && !player2CheckerOffFilter.includes('>') && !player2CheckerOffFilter.includes('<')) {
                                    player2CheckerOffFilter = `${player2CheckerOffFilter},${player2CheckerOffFilter.slice(1)}`;
                                }
                                let player1BackCheckerFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('k>') || filter.startsWith('k<') || filter.startsWith('k')));
                                if (player1BackCheckerFilter && !player1BackCheckerFilter.includes(',') && !player1BackCheckerFilter.includes('>') && !player1BackCheckerFilter.includes('<')) {
                                    player1BackCheckerFilter = `${player1BackCheckerFilter},${player1BackCheckerFilter.slice(1)}`;
                                }
                                let player2BackCheckerFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('K>') || filter.startsWith('K<') || filter.startsWith('K')));
                                if (player2BackCheckerFilter && !player2BackCheckerFilter.includes(',') && !player2BackCheckerFilter.includes('>') && !player2BackCheckerFilter.includes('<')) {
                                    player2BackCheckerFilter = `${player2BackCheckerFilter},${player2BackCheckerFilter.slice(1)}`;
                                }
                                let player1CheckerInZoneFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('z>') || filter.startsWith('z<') || filter.startsWith('z')));
                                if (player1CheckerInZoneFilter && !player1CheckerInZoneFilter.includes(',') && !player1CheckerInZoneFilter.includes('>') && !player1CheckerInZoneFilter.includes('<')) {
                                    player1CheckerInZoneFilter = `${player1CheckerInZoneFilter},${player1CheckerInZoneFilter.slice(1)}`;
                                }
                                let player2CheckerInZoneFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('Z>') || filter.startsWith('Z<') || filter.startsWith('Z')));
                                if (player2CheckerInZoneFilter && !player2CheckerInZoneFilter.includes(',') && !player2CheckerInZoneFilter.includes('>') && !player2CheckerInZoneFilter.includes('<')) {
                                    player2CheckerInZoneFilter = `${player2CheckerInZoneFilter},${player2CheckerInZoneFilter.slice(1)}`;
                                }
                                const player1AbsolutePipCountFilter = filters.find(
                                    (filter) => typeof filter === 'string' && (filter.startsWith('P>') || filter.startsWith('P<') || filter.startsWith('P'))
                                );
                                const equityFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('e>') || filter.startsWith('e<') || filter.startsWith('e')));
                                const dateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('T>') || filter.startsWith('T<') || filter.startsWith('T')));
                                const movePatternMatch = command.match(/m["'][^"']*["']/);
                                const movePatternFilter = movePatternMatch ? movePatternMatch[0] : '';
                                const searchTextMatch = command.match(/t["'][^"']*["']/);
                                const searchText = searchTextMatch ? searchTextMatch[0] : '';
                                const player1OutfieldBlotFilter = filters.find(
                                    (filter) => typeof filter === 'string' && (filter.startsWith('bo>') || filter.startsWith('bo<') || filter.startsWith('bo'))
                                );
                                const player2OutfieldBlotFilter = filters.find(
                                    (filter) => typeof filter === 'string' && (filter.startsWith('BO>') || filter.startsWith('BO<') || filter.startsWith('BO'))
                                );
                                const player1JanBlotFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('bj>') || filter.startsWith('bj<') || filter.startsWith('bj')));
                                const player2JanBlotFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('BJ>') || filter.startsWith('BJ<') || filter.startsWith('BJ')));
                                const moveErrorFilter = filters.find(
                                    (filter) => typeof filter === 'string' && (filter.startsWith('E>') || filter.startsWith('E<') || (filter.startsWith('E') && /^E\d/.test(filter)))
                                );

                                const matchIDTokens = filters.filter((f) => typeof f === 'string' && /^ma\d/.test(f));
                                let matchIDsFilter = '';
                                if (matchIDTokens.length > 0) {
                                    const parts = [];
                                    for (const token of matchIDTokens) {
                                        const val = token.slice(2);
                                        if (val.includes(',')) {
                                            parts.push(val);
                                        } else {
                                            parts.push(val);
                                        }
                                    }
                                    matchIDsFilter = parts.join(';');
                                }

                                const tournamentIDTokens = filters.filter((f) => typeof f === 'string' && /^tn\d/.test(f));
                                let tournamentIDsFilter = '';
                                if (tournamentIDTokens.length > 0) {
                                    const parts = [];
                                    for (const token of tournamentIDTokens) {
                                        const val = token.slice(2);
                                        if (val.includes(',')) {
                                            parts.push(val);
                                        } else {
                                            parts.push(val);
                                        }
                                    }
                                    tournamentIDsFilter = parts.join(';');
                                }

                                onLoadPositionsByFilters(
                                    filters,
                                    includeCube,
                                    includeScore,
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
                                    searchText,
                                    player1AbsolutePipCountFilter,
                                    equityFilter,
                                    decisionTypeFilter,
                                    diceRollFilter,
                                    movePatternFilter,
                                    dateFilter,
                                    player1OutfieldBlotFilter,
                                    player2OutfieldBlotFilter,
                                    player1JanBlotFilter,
                                    player2JanBlotFilter,
                                    noContactFilter,
                                    mirrorPositionFilter,
                                    moveErrorFilter,
                                    command,
                                    matchIDsFilter,
                                    tournamentIDsFilter,
                                    currentIDs
                                );
                            }
                        }
                    } else {
                        statusBarTextStore.set('Search in results is not available in current mode.');
                    }
                } else if (command.startsWith('s')) {
                    if ($statusBarModeStore === 'EDIT') {
                        // Save to search history for all search commands
                        const searchHistoryEntry = {
                            command: command,
                            position: JSON.stringify($positionStore),
                            timestamp: Date.now()
                        };

                        // Update search history store
                        searchHistoryStore.update((history) => {
                            const newHistory = [searchHistoryEntry, ...history].slice(0, 100); // Keep only last 100
                            return newHistory;
                        });

                        // Save to database
                        SaveSearchHistory(command, JSON.stringify($positionStore)).catch((err) => {
                            logger.error('Error saving search history:', err);
                        });

                        if (command === 's') {
                            onLoadPositionsByFilters(
                                [],
                                false,
                                false,
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                false,
                                false,
                                '',
                                '',
                                '',
                                '',
                                '',
                                '',
                                false,
                                false,
                                '',
                                command,
                                '',
                                ''
                            );
                        } else {
                            const filters = command
                                .slice(1)
                                .trim()
                                .split(' ')
                                .map((filter) => filter.trim());
                            const includeCube = filters.includes('cube') || filters.includes('cu') || filters.includes('c') || filters.includes('cub');
                            const includeScore = filters.includes('score') || filters.includes('sco') || filters.includes('sc') || filters.includes('s');
                            const noContactFilter = filters.includes('nc');
                            const decisionTypeFilter = filters.includes('d');
                            const diceRollFilter = filters.includes('D');
                            const mirrorPositionFilter = filters.includes('M');
                            const pipCountFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('p>') || filter.startsWith('p<') || filter.startsWith('p')));
                            const winRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('w>') || filter.startsWith('w<') || filter.startsWith('w')));
                            const gammonRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('g>') || filter.startsWith('g<') || filter.startsWith('g')));
                            const backgammonRateFilter = filters.find(
                                (filter) =>
                                    typeof filter === 'string' &&
                                    (filter.startsWith('b>') || filter.startsWith('b<') || (filter.startsWith('b') && !filter.startsWith('bo'))) &&
                                    !filter.startsWith('bj')
                            );
                            const player2WinRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('W>') || filter.startsWith('W<') || filter.startsWith('W')));
                            const player2GammonRateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('G>') || filter.startsWith('G<') || filter.startsWith('G')));
                            const player2BackgammonRateFilter = filters.find(
                                (filter) =>
                                    typeof filter === 'string' &&
                                    (filter.startsWith('B>') || filter.startsWith('B<') || (filter.startsWith('B') && !filter.startsWith('BO'))) &&
                                    !filter.startsWith('BJ')
                            );
                            let player1CheckerOffFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('o>') || filter.startsWith('o<') || filter.startsWith('o')));
                            if (player1CheckerOffFilter && !player1CheckerOffFilter.includes(',') && !player1CheckerOffFilter.includes('>') && !player1CheckerOffFilter.includes('<')) {
                                player1CheckerOffFilter = `${player1CheckerOffFilter},${player1CheckerOffFilter.slice(1)}`; // Handle case where 'ox' means 'ox,x'
                            }
                            let player2CheckerOffFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('O>') || filter.startsWith('O<') || filter.startsWith('O')));
                            if (player2CheckerOffFilter && !player2CheckerOffFilter.includes(',') && !player2CheckerOffFilter.includes('>') && !player2CheckerOffFilter.includes('<')) {
                                player2CheckerOffFilter = `${player2CheckerOffFilter},${player2CheckerOffFilter.slice(1)}`; // Handle case where 'Ox' means 'Ox,x'
                            }
                            let player1BackCheckerFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('k>') || filter.startsWith('k<') || filter.startsWith('k')));
                            if (player1BackCheckerFilter && !player1BackCheckerFilter.includes(',') && !player1BackCheckerFilter.includes('>') && !player1BackCheckerFilter.includes('<')) {
                                player1BackCheckerFilter = `${player1BackCheckerFilter},${player1BackCheckerFilter.slice(1)}`; // Handle case where 'kx' means 'kx,x'
                            }
                            let player2BackCheckerFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('K>') || filter.startsWith('K<') || filter.startsWith('K')));
                            if (player2BackCheckerFilter && !player2BackCheckerFilter.includes(',') && !player2BackCheckerFilter.includes('>') && !player2BackCheckerFilter.includes('<')) {
                                player2BackCheckerFilter = `${player2BackCheckerFilter},${player2BackCheckerFilter.slice(1)}`; // Handle case where 'Kx' means 'Kx,x'
                            }
                            let player1CheckerInZoneFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('z>') || filter.startsWith('z<') || filter.startsWith('z')));
                            if (player1CheckerInZoneFilter && !player1CheckerInZoneFilter.includes(',') && !player1CheckerInZoneFilter.includes('>') && !player1CheckerInZoneFilter.includes('<')) {
                                player1CheckerInZoneFilter = `${player1CheckerInZoneFilter},${player1CheckerInZoneFilter.slice(1)}`; // Handle case where 'zx' means 'zx,x'
                            }
                            let player2CheckerInZoneFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('Z>') || filter.startsWith('Z<') || filter.startsWith('Z')));
                            if (player2CheckerInZoneFilter && !player2CheckerInZoneFilter.includes(',') && !player2CheckerInZoneFilter.includes('>') && !player2CheckerInZoneFilter.includes('<')) {
                                player2CheckerInZoneFilter = `${player2CheckerInZoneFilter},${player2CheckerInZoneFilter.slice(1)}`; // Handle case where 'Zx' means 'Zx,x'
                            }
                            const player1AbsolutePipCountFilter = filters.find(
                                (filter) => typeof filter === 'string' && (filter.startsWith('P>') || filter.startsWith('P<') || filter.startsWith('P'))
                            );
                            const equityFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('e>') || filter.startsWith('e<') || filter.startsWith('e')));
                            const dateFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('T>') || filter.startsWith('T<') || filter.startsWith('T')));
                            const movePatternMatch = command.match(/m["'][^"']*["']/);
                            const movePatternFilter = movePatternMatch ? movePatternMatch[0] : '';
                            const searchTextMatch = command.match(/t["'][^"']*["']/);
                            const searchText = searchTextMatch ? searchTextMatch[0] : '';
                            const player1OutfieldBlotFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('bo>') || filter.startsWith('bo<') || filter.startsWith('bo')));
                            const player2OutfieldBlotFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('BO>') || filter.startsWith('BO<') || filter.startsWith('BO')));
                            const player1JanBlotFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('bj>') || filter.startsWith('bj<') || filter.startsWith('bj')));
                            const player2JanBlotFilter = filters.find((filter) => typeof filter === 'string' && (filter.startsWith('BJ>') || filter.startsWith('BJ<') || filter.startsWith('BJ')));
                            const moveErrorFilter = filters.find(
                                (filter) => typeof filter === 'string' && (filter.startsWith('E>') || filter.startsWith('E<') || (filter.startsWith('E') && /^E\d/.test(filter)))
                            );

                            // Match ID filter: ma23, ma2,4 (range), multiple ma tokens joined with ";"
                            const matchIDTokens = filters.filter((f) => typeof f === 'string' && /^ma\d/.test(f));
                            let matchIDsFilter = '';
                            if (matchIDTokens.length > 0) {
                                const parts = [];
                                for (const token of matchIDTokens) {
                                    const val = token.slice(2); // remove 'ma' prefix
                                    if (val.includes(',')) {
                                        // Range: pass as-is (e.g., "2,4" means IDs 2 to 4)
                                        parts.push(val);
                                    } else {
                                        parts.push(val);
                                    }
                                }
                                // Multiple individual IDs separated by ";", ranges kept with ","
                                matchIDsFilter = parts.join(';');
                            }

                            // Tournament ID filter: tn5, tn1,3 (range), multiple tn tokens joined with ";"
                            const tournamentIDTokens = filters.filter((f) => typeof f === 'string' && /^tn\d/.test(f));
                            let tournamentIDsFilter = '';
                            if (tournamentIDTokens.length > 0) {
                                const parts = [];
                                for (const token of tournamentIDTokens) {
                                    const val = token.slice(2); // remove 'tn' prefix
                                    if (val.includes(',')) {
                                        parts.push(val);
                                    } else {
                                        parts.push(val);
                                    }
                                }
                                tournamentIDsFilter = parts.join(';');
                            }

                            logger.log('Filters:', filters); // Add logging
                            logger.log('Search Text:', searchText); // Add logging
                            logger.log('Move Pattern Filter:', movePatternFilter); // Add logging

                            // display in console log all the filters
                            logger.log('includeCube:', includeCube);
                            logger.log('includeScore:', includeScore);
                            logger.log('pipCountFilter:', pipCountFilter);
                            logger.log('winRateFilter:', winRateFilter);
                            logger.log('gammonRateFilter:', gammonRateFilter);
                            logger.log('backgammonRateFilter:', backgammonRateFilter);
                            logger.log('player2WinRateFilter:', player2WinRateFilter);
                            logger.log('player2GammonRateFilter:', player2GammonRateFilter);
                            logger.log('player2BackgammonRateFilter:', player2BackgammonRateFilter);
                            logger.log('player1CheckerOffFilter:', player1CheckerOffFilter);
                            logger.log('player2CheckerOffFilter:', player2CheckerOffFilter);
                            logger.log('player1BackCheckerFilter:', player1BackCheckerFilter);
                            logger.log('player2BackCheckerFilter:', player2BackCheckerFilter);
                            logger.log('player1CheckerInZoneFilter:', player1CheckerInZoneFilter);
                            logger.log('player2CheckerInZoneFilter:', player2CheckerInZoneFilter);
                            logger.log('player1AbsolutePipCountFilter:', player1AbsolutePipCountFilter);
                            logger.log('equityFilter:', equityFilter);
                            logger.log('decisionTypeFilter:', decisionTypeFilter);
                            logger.log('diceRollFilter:', diceRollFilter);
                            logger.log('dateFilter:', dateFilter);
                            logger.log('player1OutfieldBlotFilter:', player1OutfieldBlotFilter);
                            logger.log('player2OutfieldBlotFilter:', player2OutfieldBlotFilter);
                            logger.log('player1JanBlotFilter:', player1JanBlotFilter);
                            logger.log('player2JanBlotFilter:', player2JanBlotFilter);
                            logger.log('noContactFilter:', noContactFilter);
                            logger.log('mirrorPositionFilter:', mirrorPositionFilter);
                            logger.log('moveErrorFilter:', moveErrorFilter);

                            onLoadPositionsByFilters(
                                filters,
                                includeCube,
                                includeScore,
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
                                searchText,
                                player1AbsolutePipCountFilter,
                                equityFilter,
                                decisionTypeFilter,
                                diceRollFilter,
                                movePatternFilter,
                                dateFilter,
                                player1OutfieldBlotFilter,
                                player2OutfieldBlotFilter,
                                player1JanBlotFilter,
                                player2JanBlotFilter,
                                noContactFilter,
                                mirrorPositionFilter,
                                moveErrorFilter,
                                command, // Pass the original search command for session tracking
                                matchIDsFilter,
                                tournamentIDsFilter
                            );
                        }
                    } else {
                        statusBarTextStore.set('Search is only available in edit mode.');
                    }
                } else if (command === 'filter' || command === 'fl') {
                    toggleFilterLibraryPanel();
                } else if (command === 'history' || command === 'hi') {
                    toggleSearchHistoryPanel();
                } else if (command === 'match' || command === 'ma') {
                    toggleMatchPanel();
                } else if (command === 'collection' || command === 'coll') {
                    toggleCollectionPanel();
                } else if (command === 'epc') {
                    closeModal();
                    toggleEPCMode();
                } else if (command === 'm') {
                    closeModal();
                    toggleMatchMode();
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
                        openModal(MODAL.METADATA);
                    } else {
                        statusBarTextStore.set('No database loaded.');
                    }
                } else if (command === 'tp2') {
                    openModal(MODAL.TAKE_POINT_2);
                } else if (command === 'tp4') {
                    openModal(MODAL.TAKE_POINT_4);
                } else if (command === 'migrate_from_1_0_to_1_1') {
                    try {
                        await Migrate_1_0_0_to_1_1_0();
                        statusBarTextStore.set('Database migrated to version 1.1.0 successfully.');
                    } catch (error) {
                        logger.error('Error migrating database:', error);
                        statusBarTextStore.set('Error migrating database.');
                    }
                } else if (command === 'migrate_from_1_1_to_1_2') {
                    try {
                        await Migrate_1_1_0_to_1_2_0();
                        statusBarTextStore.set('Database migrated to version 1.2.0 successfully.');
                    } catch (error) {
                        logger.error('Error migrating database:', error);
                        statusBarTextStore.set('Error migrating database.');
                    }
                } else if (command === 'migrate_from_1_2_to_1_3') {
                    try {
                        await Migrate_1_2_0_to_1_3_0();
                        statusBarTextStore.set('Database migrated to version 1.3.0 successfully.');
                    } catch (error) {
                        logger.error('Error migrating database:', error);
                        statusBarTextStore.set('Error migrating database.');
                    }
                } else if (command === 'cl' || command === 'clear') {
                    try {
                        await ClearCommandHistory();
                        commandHistoryStore.set([]);
                        statusBarTextStore.set('Command history cleared.');
                    } catch (error) {
                        logger.error('Error clearing command history:', error);
                        statusBarTextStore.set('Error clearing command history.');
                    }
                }
                closeModal(); // Hide the command line after processing the command
            } else if (event.ctrlKey && event.code === 'KeyH') {
                onToggleHelp();
            }
        }
    }

    function insertTags(tags) {
        commentTextStore.update((text) => {
            const existingTags = new Set(text.match(/#[^\s#]+/g) || []);
            const newTags = tags.split(' ').filter((tag) => !existingTags.has(tag));
            const updatedText = `${newTags.join(' ')}\n${text}`;
            setTimeout(() => {
                const textAreaEl = document.getElementById('commentTextArea');
                if (textAreaEl) {
                    textAreaEl.setSelectionRange(updatedText.length, updatedText.length);
                    textAreaEl.focus();
                }
            }, 0);
            // Save the updated comment to the database
            SaveComment($currentPositionIndexStore, updatedText);
            return updatedText;
        });
    }

    function handleClickOutside(event) {
        if ($activeModal === MODAL.COMMAND && !inputEl.contains(event.target)) {
            closeModal();
        }
    }

    function handleGlobalKeyDown(_event) {
        if (!initialized) {
            initialized = true;
            window.addEventListener('keydown', handleKeyDown);
        }
    }

    onMount(() => {
        window.addEventListener('keydown', handleGlobalKeyDown);
    });

    onDestroy(() => {
        window.removeEventListener('click', handleClickOutside);
        window.removeEventListener('keydown', handleGlobalKeyDown);
    });
</script>

{#if $activeModal === MODAL.COMMAND}
    <input type="text" bind:this={inputEl} bind:value={$commandTextStore} class="command-input" placeholder=" Type your command here. " onkeydown={handleKeyDown} />
{/if}

<style>
    input {
        position: fixed;
        top: 350px;
        left: 50%;
        transform: translateX(-50%);
        z-index: 1000;
        width: 70%;
        padding: 8px;
        border: 1px solid rgba(0, 0, 0, 0.3); /* Subtle border */
        border-radius: 1px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0);
        outline: none;
        background-color: white; /* Ensure background is opaque */
        font-size: 18px;
    }
</style>
