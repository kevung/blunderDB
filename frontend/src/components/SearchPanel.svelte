<script>
    import { onMount, onDestroy, tick } from 'svelte';
    import { statusBarTextStore, currentPositionIndexStore, activeTabStore } from '../stores/uiStore';
    import { positionStore, positionsStore, positionBeforeFilterLibraryStore, positionIndexBeforeFilterLibraryStore } from '../stores/positionStore';
    import { searchHistoryStore } from '../stores/searchHistoryStore';
    import { filterLibraryStore } from '../stores/filterLibraryStore';
    import { SaveSearchHistory, LoadSearchHistory, DeleteSearchHistoryEntry, LoadFilters, SaveFilter, UpdateFilter, DeleteFilter, SaveEditPosition, LoadEditPosition } from '../../wailsjs/go/main/Database.js';

    export let onLoadPositionsByFilters;
    export let onAddToFilterLibrary;

    // Sub-tab state
    let activeSubTab = 'search'; // 'search', 'history', 'saved'

    // Filter state
    let filterEnabled = {};
    let searchInCurrentResults = false;

    let searchText = '';
    let movePattern = '';
    let matchIDsInput = '';
    let tournamentIDsInput = '';

    let pipCountOption = 'min'; let pipCountMin = -375; let pipCountMax = 375; let pipCountRangeMin = -375; let pipCountRangeMax = 375;
    let winRateOption = 'min'; let winRateMin = 0; let winRateMax = 100; let winRateRangeMin = 0; let winRateRangeMax = 100;
    let gammonRateOption = 'min'; let gammonRateMin = 0; let gammonRateMax = 100; let gammonRateRangeMin = 0; let gammonRateRangeMax = 100;
    let backgammonRateOption = 'min'; let backgammonRateMin = 0; let backgammonRateMax = 100; let backgammonRateRangeMin = 0; let backgammonRateRangeMax = 100;
    let player2WinRateOption = 'min'; let player2WinRateMin = 0; let player2WinRateMax = 100; let player2WinRateRangeMin = 0; let player2WinRateRangeMax = 100;
    let player2GammonRateOption = 'min'; let player2GammonRateMin = 0; let player2GammonRateMax = 100; let player2GammonRateRangeMin = 0; let player2GammonRateRangeMax = 100;
    let player2BackgammonRateOption = 'min'; let player2BackgammonRateMin = 0; let player2BackgammonRateMax = 100; let player2BackgammonRateRangeMin = 0; let player2BackgammonRateRangeMax = 100;
    let player1CheckerOffOption = 'min'; let player1CheckerOffMin = 0; let player1CheckerOffMax = 15; let player1CheckerOffRangeMin = 0; let player1CheckerOffRangeMax = 15;
    let player2CheckerOffOption = 'min'; let player2CheckerOffMin = 0; let player2CheckerOffMax = 15; let player2CheckerOffRangeMin = 0; let player2CheckerOffRangeMax = 15;
    let player1BackCheckerOption = 'min'; let player1BackCheckerMin = 0; let player1BackCheckerMax = 15; let player1BackCheckerRangeMin = 0; let player1BackCheckerRangeMax = 15;
    let player2BackCheckerOption = 'min'; let player2BackCheckerMin = 0; let player2BackCheckerMax = 15; let player2BackCheckerRangeMin = 0; let player2BackCheckerRangeMax = 15;
    let player1CheckerInZoneOption = 'min'; let player1CheckerInZoneMin = 0; let player1CheckerInZoneMax = 15; let player1CheckerInZoneRangeMin = 0; let player1CheckerInZoneRangeMax = 15;
    let player2CheckerInZoneOption = 'min'; let player2CheckerInZoneMin = 0; let player2CheckerInZoneMax = 15; let player2CheckerInZoneRangeMin = 0; let player2CheckerInZoneRangeMax = 15;
    let player1AbsolutePipCountOption = 'min'; let player1AbsolutePipCountMin = 0; let player1AbsolutePipCountMax = 375; let player1AbsolutePipCountRangeMin = 0; let player1AbsolutePipCountRangeMax = 375;
    let equityOption = 'min'; let equityMin = -1000; let equityMax = 1000; let equityRangeMin = -1000; let equityRangeMax = 1000;
    let moveErrorOption = 'min'; let moveErrorMin = 0; let moveErrorMax = 1000; let moveErrorRangeMin = 0; let moveErrorRangeMax = 1000;
    let player1OutfieldBlotOption = 'min'; let player1OutfieldBlotMin = 0; let player1OutfieldBlotMax = 15; let player1OutfieldBlotRangeMin = 0; let player1OutfieldBlotRangeMax = 15;
    let player2OutfieldBlotOption = 'min'; let player2OutfieldBlotMin = 0; let player2OutfieldBlotMax = 15; let player2OutfieldBlotRangeMin = 0; let player2OutfieldBlotRangeMax = 15;
    let player1JanBlotOption = 'min'; let player1JanBlotMin = 0; let player1JanBlotMax = 15; let player1JanBlotRangeMin = 0; let player1JanBlotRangeMax = 15;
    let player2JanBlotOption = 'min'; let player2JanBlotMin = 0; let player2JanBlotMax = 15; let player2JanBlotRangeMin = 0; let player2JanBlotRangeMax = 15;
    let creationDateOption = 'min'; let creationDateMin = ''; let creationDateMax = ''; let creationDateRangeMin = ''; let creationDateRangeMax = '';

    // History state
    let searchHistory = [];
    let selectedSearch = null;
    let showSaveDialog = false;
    let filterName = '';

    // Saved (filter library) state
    let savedFilters = [];
    let selectedSavedFilter = null;
    let savedFilterName = '';
    let savedFilterCommand = '';

    searchHistoryStore.subscribe(value => { searchHistory = value; });
    filterLibraryStore.subscribe(value => { savedFilters = value || []; });

    let availableFilters = [
        'Include Cube', 'Include Score', 'Include Decision Type', 'Include Dice Roll',
        'No Contact', 'Mirror Position', 'Pipcount Difference', 'Player Absolute Pipcount',
        'Equity (millipoints)', 'Move Error (millipoints, Player 1)',
        'Win Rate', 'Gammon Rate', 'Backgammon Rate',
        'Opponent Win Rate', 'Opponent Gammon Rate', 'Opponent Backgammon Rate',
        'Player Checker-Off', 'Opponent Checker-Off', 'Player Back Checker', 'Opponent Back Checker',
        'Player Checker in the Zone', 'Opponent Checker in the Zone',
        'Player Outfield Blot', 'Opponent Outfield Blot', 'Player Jan Blot', 'Opponent Jan Blot',
        'Search Text', 'Best Move or Cube Decision', 'Creation Date', 'Match IDs', 'Tournament IDs'
    ];

    let filterGroups = [
        { name: 'Display', filters: ['Include Cube', 'Include Score', 'Include Decision Type', 'Include Dice Roll'] },
        { name: 'Position', filters: ['No Contact', 'Mirror Position'] },
        { name: 'Pipcount', filters: ['Pipcount Difference', 'Player Absolute Pipcount'] },
        { name: 'Equity / Error', filters: ['Equity (millipoints)', 'Move Error (millipoints, Player 1)'] },
        { name: 'Player Rates', filters: ['Win Rate', 'Gammon Rate', 'Backgammon Rate'] },
        { name: 'Opponent Rates', filters: ['Opponent Win Rate', 'Opponent Gammon Rate', 'Opponent Backgammon Rate'] },
        { name: 'Checkers', filters: ['Player Checker-Off', 'Opponent Checker-Off', 'Player Back Checker', 'Opponent Back Checker', 'Player Checker in the Zone', 'Opponent Checker in the Zone'] },
        { name: 'Blots', filters: ['Player Outfield Blot', 'Opponent Outfield Blot', 'Player Jan Blot', 'Opponent Jan Blot'] },
        { name: 'Text / Pattern', filters: ['Search Text', 'Best Move or Cube Decision'] },
        { name: 'Other', filters: ['Creation Date', 'Match IDs', 'Tournament IDs'] }
    ];

    // Initialize all filters as disabled
    availableFilters.forEach(f => filterEnabled[f] = false);

    $: activeFilterCount = availableFilters.filter(f => filterEnabled[f]).length;

    activeTabStore.subscribe(async value => {
        if (value === 'search') {
            await loadHistory();
            await loadSavedFilters();
        }
    });

    async function loadHistory() {
        try {
            const history = await LoadSearchHistory();
            searchHistoryStore.set(history || []);
        } catch (error) {
            console.error('Error loading search history:', error);
        }
    }

    async function loadSavedFilters() {
        try {
            const lib = await LoadFilters();
            filterLibraryStore.set(lib || []);
        } catch (error) {
            filterLibraryStore.set([]);
        }
    }

    function isInFilterLibrary(search) {
        return savedFilters.some(f => f.command === search.command);
    }

    function handleSearch() {
        const activeFilters = availableFilters.filter(f => filterEnabled[f]);
        const transformedFilters = activeFilters.map(filter => {
            switch (filter) {
                case 'Include Cube': return 'cube';
                case 'Include Score': return 'score';
                case 'Include Decision Type': return 'd';
                case 'Include Dice Roll': return 'D';
                case 'No Contact': return 'nc';
                case 'Mirror Position': return 'M';
                case 'Pipcount Difference':
                    return pipCountOption === 'min' ? `p>${pipCountMin}` : pipCountOption === 'max' ? `p<${pipCountMax}` : `p${pipCountRangeMin},${pipCountRangeMax}`;
                case 'Player Absolute Pipcount':
                    return player1AbsolutePipCountOption === 'min' ? `P>${player1AbsolutePipCountMin}` : player1AbsolutePipCountOption === 'max' ? `P<${player1AbsolutePipCountMax}` : `P${player1AbsolutePipCountRangeMin},${player1AbsolutePipCountRangeMax}`;
                case 'Equity (millipoints)':
                    return equityOption === 'min' ? `e>${equityMin}` : equityOption === 'max' ? `e<${equityMax}` : `e${equityRangeMin},${equityRangeMax}`;
                case 'Move Error (millipoints, Player 1)':
                    return moveErrorOption === 'min' ? `E>${moveErrorMin}` : moveErrorOption === 'max' ? `E<${moveErrorMax}` : `E${moveErrorRangeMin},${moveErrorRangeMax}`;
                case 'Win Rate':
                    return winRateOption === 'min' ? `w>${winRateMin}` : winRateOption === 'max' ? `w<${winRateMax}` : `w${winRateRangeMin},${winRateRangeMax}`;
                case 'Gammon Rate':
                    return gammonRateOption === 'min' ? `g>${gammonRateMin}` : gammonRateOption === 'max' ? `g<${gammonRateMax}` : `g${gammonRateRangeMin},${gammonRateRangeMax}`;
                case 'Backgammon Rate':
                    return backgammonRateOption === 'min' ? `b>${backgammonRateMin}` : backgammonRateOption === 'max' ? `b<${backgammonRateMax}` : `b${backgammonRateRangeMin},${backgammonRateRangeMax}`;
                case 'Opponent Win Rate':
                    return player2WinRateOption === 'min' ? `W>${player2WinRateMin}` : player2WinRateOption === 'max' ? `W<${player2WinRateMax}` : `W${player2WinRateRangeMin},${player2WinRateRangeMax}`;
                case 'Opponent Gammon Rate':
                    return player2GammonRateOption === 'min' ? `G>${player2GammonRateMin}` : player2GammonRateOption === 'max' ? `G<${player2GammonRateMax}` : `G${player2GammonRateRangeMin},${player2GammonRateRangeMax}`;
                case 'Opponent Backgammon Rate':
                    return player2BackgammonRateOption === 'min' ? `B>${player2BackgammonRateMin}` : player2BackgammonRateOption === 'max' ? `B<${player2BackgammonRateMax}` : `B${player2BackgammonRateRangeMin},${player2BackgammonRateRangeMax}`;
                case 'Player Checker-Off':
                    return player1CheckerOffOption === 'min' ? `o>${player1CheckerOffMin}` : player1CheckerOffOption === 'max' ? `o<${player1CheckerOffMax}` : `o${player1CheckerOffRangeMin},${player1CheckerOffRangeMax}`;
                case 'Opponent Checker-Off':
                    return player2CheckerOffOption === 'min' ? `O>${player2CheckerOffMin}` : player2CheckerOffOption === 'max' ? `O<${player2CheckerOffMax}` : `O${player2CheckerOffRangeMin},${player2CheckerOffRangeMax}`;
                case 'Player Back Checker':
                    return player1BackCheckerOption === 'min' ? `k>${player1BackCheckerMin}` : player1BackCheckerOption === 'max' ? `k<${player1BackCheckerMax}` : `k${player1BackCheckerRangeMin},${player1BackCheckerRangeMax}`;
                case 'Opponent Back Checker':
                    return player2BackCheckerOption === 'min' ? `K>${player2BackCheckerMin}` : player2BackCheckerOption === 'max' ? `K<${player2BackCheckerMax}` : `K${player2BackCheckerRangeMin},${player2BackCheckerRangeMax}`;
                case 'Player Checker in the Zone':
                    return player1CheckerInZoneOption === 'min' ? `z>${player1CheckerInZoneMin}` : player1CheckerInZoneOption === 'max' ? `z<${player1CheckerInZoneMax}` : `z${player1CheckerInZoneRangeMin},${player1CheckerInZoneRangeMax}`;
                case 'Opponent Checker in the Zone':
                    return player2CheckerInZoneOption === 'min' ? `Z>${player2CheckerInZoneMin}` : player2CheckerInZoneOption === 'max' ? `Z<${player2CheckerInZoneMax}` : `Z${player2CheckerInZoneRangeMin},${player2CheckerInZoneRangeMax}`;
                case 'Search Text': return `t"${searchText}"`;
                case 'Best Move or Cube Decision': return `m"${movePattern}"`;
                case 'Creation Date':
                    const formatDate = date => date.replace(/-/g, '/');
                    return creationDateOption === 'min' ? `T>${formatDate(creationDateMin)}` : creationDateOption === 'max' ? `T<${formatDate(creationDateMax)}` : `T${formatDate(creationDateRangeMin)},${formatDate(creationDateRangeMax)}`;
                case 'Player Outfield Blot':
                    return player1OutfieldBlotOption === 'min' ? `bo>${player1OutfieldBlotMin}` : player1OutfieldBlotOption === 'max' ? `bo<${player1OutfieldBlotMax}` : `bo${player1OutfieldBlotRangeMin},${player1OutfieldBlotRangeMax}`;
                case 'Opponent Outfield Blot':
                    return player2OutfieldBlotOption === 'min' ? `BO>${player2OutfieldBlotMin}` : player2OutfieldBlotOption === 'max' ? `BO<${player2OutfieldBlotMax}` : `BO${player2OutfieldBlotRangeMin},${player2OutfieldBlotRangeMax}`;
                case 'Player Jan Blot':
                    return player1JanBlotOption === 'min' ? `bj>${player1JanBlotMin}` : player1JanBlotOption === 'max' ? `bj<${player1JanBlotMax}` : `bj${player1JanBlotRangeMin},${player1JanBlotRangeMax}`;
                case 'Opponent Jan Blot':
                    return player2JanBlotOption === 'min' ? `BJ>${player2JanBlotMin}` : player2JanBlotOption === 'max' ? `BJ<${player2JanBlotMax}` : `BJ${player2JanBlotRangeMin},${player2JanBlotRangeMax}`;
                case 'Match IDs': return matchIDsInput ? `ma${matchIDsInput}` : '';
                case 'Tournament IDs': return tournamentIDsInput ? `tn${tournamentIDsInput}` : '';
                default: return '';
            }
        });

        const incCube = transformedFilters.includes('cube');
        const incScore = transformedFilters.includes('score');
        const ncFilter = transformedFilters.includes('nc');
        const mirFilter = transformedFilters.includes('M');
        const pcFilter = transformedFilters.find(f => f.startsWith('p'));
        const wrFilter = transformedFilters.find(f => f.startsWith('w'));
        const grFilter = transformedFilters.find(f => f.startsWith('g'));
        const bgFilter = transformedFilters.find(f => f.startsWith('b') && !f.startsWith('bo') && !f.startsWith('bj'));
        const p2wrFilter = transformedFilters.find(f => f.startsWith('W'));
        const p2grFilter = transformedFilters.find(f => f.startsWith('G'));
        const p2bgFilter = transformedFilters.find(f => f.startsWith('B') && !f.startsWith('BO') && !f.startsWith('BJ'));
        const p1coFilter = transformedFilters.find(f => f.startsWith('o'));
        const p2coFilter = transformedFilters.find(f => f.startsWith('O'));
        const p1bcFilter = transformedFilters.find(f => f.startsWith('k'));
        const p2bcFilter = transformedFilters.find(f => f.startsWith('K'));
        const p1czFilter = transformedFilters.find(f => f.startsWith('z'));
        const p2czFilter = transformedFilters.find(f => f.startsWith('Z'));
        const p1apcFilter = transformedFilters.find(f => f.startsWith('P'));
        const eqFilter = transformedFilters.find(f => f.startsWith('e'));
        const meFilter = transformedFilters.find(f => f.startsWith('E'));
        const p1obFilter = transformedFilters.find(f => f.startsWith('bo'));
        const p2obFilter = transformedFilters.find(f => f.startsWith('BO'));
        const p1jbFilter = transformedFilters.find(f => f.startsWith('bj'));
        const p2jbFilter = transformedFilters.find(f => f.startsWith('BJ'));
        const matchIDToken = transformedFilters.find(f => f.startsWith('ma'));
        const matchIDs = matchIDToken ? matchIDToken.slice(2) : '';
        const tournamentIDToken = transformedFilters.find(f => f.startsWith('tn'));
        const tournamentIDs = tournamentIDToken ? tournamentIDToken.slice(2) : '';
        const dtFilter = transformedFilters.includes('d');
        const drFilter = transformedFilters.includes('D');
        const cdFilter = transformedFilters.find(f => f.startsWith('T'));

        const commandParts = ['s'];
        transformedFilters.forEach(f => {
            if (f !== 't""' && f !== 'm""') commandParts.push(f);
        });
        const searchCommand = commandParts.join(' ');

        const entry = { command: searchCommand, position: JSON.stringify($positionStore), timestamp: Date.now() };
        searchHistoryStore.update(h => [entry, ...h].slice(0, 100));
        SaveSearchHistory(searchCommand, JSON.stringify($positionStore)).catch(err => console.error('Error saving search history:', err));

        let restrictToPositionIDs = '';
        if (searchInCurrentResults) {
            const currentPositions = $positionsStore || [];
            restrictToPositionIDs = currentPositions.map(p => p.id).filter(id => id != null).join(',');
        }

        onLoadPositionsByFilters(activeFilters.length > 0 ? transformedFilters : [], incCube, incScore, pcFilter, wrFilter, grFilter, bgFilter,
            p2wrFilter, p2grFilter, p2bgFilter, p1coFilter, p2coFilter, p1bcFilter, p2bcFilter,
            p1czFilter, p2czFilter, searchText ? `t"${searchText}"` : '', p1apcFilter, eqFilter, dtFilter, drFilter, movePattern ? `m"${movePattern}"` : '',
            cdFilter, p1obFilter, p2obFilter, p1jbFilter, p2jbFilter, ncFilter, mirFilter, meFilter,
            searchCommand, matchIDs, tournamentIDs, restrictToPositionIDs);
    }

    function clearFilters() {
        availableFilters.forEach(f => filterEnabled[f] = false);
        filterEnabled = filterEnabled;
        pipCountOption = 'min'; pipCountMin = -375; pipCountMax = 375; pipCountRangeMin = -375; pipCountRangeMax = 375;
        winRateOption = 'min'; winRateMin = 0; winRateMax = 100; winRateRangeMin = 0; winRateRangeMax = 100;
        gammonRateOption = 'min'; gammonRateMin = 0; gammonRateMax = 100; gammonRateRangeMin = 0; gammonRateRangeMax = 100;
        backgammonRateOption = 'min'; backgammonRateMin = 0; backgammonRateMax = 100; backgammonRateRangeMin = 0; backgammonRateRangeMax = 100;
        player2WinRateOption = 'min'; player2WinRateMin = 0; player2WinRateMax = 100; player2WinRateRangeMin = 0; player2WinRateRangeMax = 100;
        player2GammonRateOption = 'min'; player2GammonRateMin = 0; player2GammonRateMax = 100; player2GammonRateRangeMin = 0; player2GammonRateRangeMax = 100;
        player2BackgammonRateOption = 'min'; player2BackgammonRateMin = 0; player2BackgammonRateMax = 100; player2BackgammonRateRangeMin = 0; player2BackgammonRateRangeMax = 100;
        player1CheckerOffOption = 'min'; player1CheckerOffMin = 0; player1CheckerOffMax = 15; player1CheckerOffRangeMin = 0; player1CheckerOffRangeMax = 15;
        player2CheckerOffOption = 'min'; player2CheckerOffMin = 0; player2CheckerOffMax = 15; player2CheckerOffRangeMin = 0; player2CheckerOffRangeMax = 15;
        player1BackCheckerOption = 'min'; player1BackCheckerMin = 0; player1BackCheckerMax = 15; player1BackCheckerRangeMin = 0; player1BackCheckerRangeMax = 15;
        player2BackCheckerOption = 'min'; player2BackCheckerMin = 0; player2BackCheckerMax = 15; player2BackCheckerRangeMin = 0; player2BackCheckerRangeMax = 15;
        player1CheckerInZoneOption = 'min'; player1CheckerInZoneMin = 0; player1CheckerInZoneMax = 15; player1CheckerInZoneRangeMin = 0; player1CheckerInZoneRangeMax = 15;
        player2CheckerInZoneOption = 'min'; player2CheckerInZoneMin = 0; player2CheckerInZoneMax = 15; player2CheckerInZoneRangeMin = 0; player2CheckerInZoneRangeMax = 15;
        player1AbsolutePipCountOption = 'min'; player1AbsolutePipCountMin = 0; player1AbsolutePipCountMax = 375; player1AbsolutePipCountRangeMin = 0; player1AbsolutePipCountRangeMax = 375;
        equityOption = 'min'; equityMin = -1000; equityMax = 1000; equityRangeMin = -1000; equityRangeMax = 1000;
        moveErrorOption = 'min'; moveErrorMin = 0; moveErrorMax = 1000; moveErrorRangeMin = 0; moveErrorRangeMax = 1000;
        searchText = ''; movePattern = '';
        player1OutfieldBlotOption = 'min'; player1OutfieldBlotMin = 0; player1OutfieldBlotMax = 15; player1OutfieldBlotRangeMin = 0; player1OutfieldBlotRangeMax = 15;
        player2OutfieldBlotOption = 'min'; player2OutfieldBlotMin = 0; player2OutfieldBlotMax = 15; player2OutfieldBlotRangeMin = 0; player2OutfieldBlotRangeMax = 15;
        player1JanBlotOption = 'min'; player1JanBlotMin = 0; player1JanBlotMax = 15; player1JanBlotRangeMin = 0; player1JanBlotRangeMax = 15;
        player2JanBlotOption = 'min'; player2JanBlotMin = 0; player2JanBlotMax = 15; player2JanBlotRangeMin = 0; player2JanBlotRangeMax = 15;
        matchIDsInput = ''; tournamentIDsInput = '';
        creationDateOption = 'min'; creationDateMin = ''; creationDateMax = ''; creationDateRangeMin = ''; creationDateRangeMax = '';
        searchInCurrentResults = false;
    }

    // History functions
    function selectSearch(search) {
        if (selectedSearch === search) {
            selectedSearch = null;
            if ($positionBeforeFilterLibraryStore) {
                positionStore.set($positionBeforeFilterLibraryStore);
            }
            if ($positionIndexBeforeFilterLibraryStore >= 0) {
                const savedIndex = $positionIndexBeforeFilterLibraryStore;
                currentPositionIndexStore.set(-1);
                currentPositionIndexStore.set(savedIndex);
            }
        } else {
            if (!selectedSearch && !$positionBeforeFilterLibraryStore) {
                positionBeforeFilterLibraryStore.set(JSON.parse(JSON.stringify($positionStore)));
                positionIndexBeforeFilterLibraryStore.set($currentPositionIndexStore);
            }
            selectedSearch = search;
            if (search.position) {
                positionStore.set(JSON.parse(search.position));
            }
            currentPositionIndexStore.set(-1);
        }
    }

    function executeSearch(search) {
        if (search.position) {
            positionStore.set(JSON.parse(search.position));
        }
        const command = search.command;
        if (command.startsWith('s ') || command === 's') {
            const cmdFilters = command === 's' ? [] : command.slice(2).trim().split(' ').map(f => f.trim());
            const ic = cmdFilters.includes('cube') || cmdFilters.includes('cu') || cmdFilters.includes('c') || cmdFilters.includes('cub');
            const is = cmdFilters.includes('score') || cmdFilters.includes('sco') || cmdFilters.includes('sc') || cmdFilters.includes('s');
            const nc = cmdFilters.includes('nc');
            const dt = cmdFilters.includes('d');
            const dr = cmdFilters.includes('D');
            const mp = cmdFilters.includes('M');
            const pc = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('p>') || f.startsWith('p<') || f.startsWith('p')));
            const wr = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('w>') || f.startsWith('w<') || f.startsWith('w')));
            const gr = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('g>') || f.startsWith('g<') || f.startsWith('g')));
            const bg = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('b>') || f.startsWith('b<') || (f.startsWith('b') && !f.startsWith('bo'))) && !f.startsWith('bj'));
            const p2wr = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('W>') || f.startsWith('W<') || f.startsWith('W')));
            const p2gr = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('G>') || f.startsWith('G<') || f.startsWith('G')));
            const p2bg = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('B>') || f.startsWith('B<') || f.startsWith('B') && !f.startsWith('BO')) && !f.startsWith('BJ'));
            let p1co = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('o>') || f.startsWith('o<') || f.startsWith('o')));
            if (p1co && !p1co.includes(',') && !p1co.includes('>') && !p1co.includes('<')) p1co = `${p1co},${p1co.slice(1)}`;
            let p2co = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('O>') || f.startsWith('O<') || f.startsWith('O')));
            if (p2co && !p2co.includes(',') && !p2co.includes('>') && !p2co.includes('<')) p2co = `${p2co},${p2co.slice(1)}`;
            let p1bc = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('k>') || f.startsWith('k<') || f.startsWith('k')));
            if (p1bc && !p1bc.includes(',') && !p1bc.includes('>') && !p1bc.includes('<')) p1bc = `${p1bc},${p1bc.slice(1)}`;
            let p2bc = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('K>') || f.startsWith('K<') || f.startsWith('K')));
            if (p2bc && !p2bc.includes(',') && !p2bc.includes('>') && !p2bc.includes('<')) p2bc = `${p2bc},${p2bc.slice(1)}`;
            let p1cz = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('z>') || f.startsWith('z<') || f.startsWith('z')));
            if (p1cz && !p1cz.includes(',') && !p1cz.includes('>') && !p1cz.includes('<')) p1cz = `${p1cz},${p1cz.slice(1)}`;
            let p2cz = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('Z>') || f.startsWith('Z<') || f.startsWith('Z')));
            if (p2cz && !p2cz.includes(',') && !p2cz.includes('>') && !p2cz.includes('<')) p2cz = `${p2cz},${p2cz.slice(1)}`;
            const p1apc = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('P>') || f.startsWith('P<') || f.startsWith('P')));
            const eq = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('e>') || f.startsWith('e<') || f.startsWith('e')));
            const cd = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('T>') || f.startsWith('T<') || f.startsWith('T')));
            const mpm = command.match(/m["'][^"']*["']/);
            const mpf = mpm ? mpm[0] : '';
            const stm = command.match(/t["'][^"']*["']/);
            const st = stm ? stm[0] : '';
            const p1ob = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('bo>') || f.startsWith('bo<') || f.startsWith('bo')));
            const p2ob = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('BO>') || f.startsWith('BO<') || f.startsWith('BO')));
            const p1jb = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('bj>') || f.startsWith('bj<') || f.startsWith('bj')));
            const p2jb = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('BJ>') || f.startsWith('BJ<') || f.startsWith('BJ')));
            const me = cmdFilters.find(f => typeof f === 'string' && (f.startsWith('E>') || f.startsWith('E<') || (f.startsWith('E') && /^E\d/.test(f))));
            const maTokens = cmdFilters.filter(f => typeof f === 'string' && /^ma\d/.test(f));
            const matchIDs = maTokens.length > 0 ? maTokens.map(t => t.slice(2)).join(';') : '';
            const tnTokens = cmdFilters.filter(f => typeof f === 'string' && /^tn\d/.test(f));
            const tournamentIDs = tnTokens.length > 0 ? tnTokens.map(t => t.slice(2)).join(';') : '';
            onLoadPositionsByFilters(cmdFilters, ic, is, pc, wr, gr, bg, p2wr, p2gr, p2bg, p1co, p2co,
                p1bc, p2bc, p1cz, p2cz, st, p1apc, eq, dt, dr, mpf, cd, p1ob, p2ob, p1jb, p2jb, nc, mp, me, command, matchIDs, tournamentIDs);
        }
    }

    function handleDoubleClick(search) {
        executeSearch(search);
    }

    function showAddToLibraryDialog(search) {
        selectedSearch = search;
        showSaveDialog = true;
        filterName = '';
    }

    function cancelSaveDialog() {
        showSaveDialog = false;
        filterName = '';
    }

    async function saveToFilterLibrary() {
        if (!filterName || !selectedSearch) {
            statusBarTextStore.set('Please enter a filter name');
            return;
        }
        if (onAddToFilterLibrary) {
            await onAddToFilterLibrary(filterName, selectedSearch.command, selectedSearch.position);
            await loadSavedFilters();
            statusBarTextStore.set('Filter saved to library');
        }
        cancelSaveDialog();
    }

    async function deleteSearch(search, event) {
        event.stopPropagation();
        try {
            await DeleteSearchHistoryEntry(search.timestamp);
            await loadHistory();
            statusBarTextStore.set('Search deleted from history');
        } catch (error) {
            statusBarTextStore.set('Error deleting search');
        }
    }

    function formatTimestamp(timestamp) {
        const date = new Date(timestamp);
        return date.toLocaleString();
    }

    // --- Saved filter (bookmarked search) functions ---
    async function selectSavedFilter(filter) {
        if (selectedSavedFilter && selectedSavedFilter.id === filter.id) {
            selectedSavedFilter = null;
            savedFilterName = '';
            savedFilterCommand = '';
            if ($positionBeforeFilterLibraryStore) {
                positionStore.set($positionBeforeFilterLibraryStore);
            }
            if ($positionIndexBeforeFilterLibraryStore >= 0) {
                const savedIndex = $positionIndexBeforeFilterLibraryStore;
                currentPositionIndexStore.set(-1);
                currentPositionIndexStore.set(savedIndex);
            }
            return;
        }
        if (!selectedSavedFilter && !$positionBeforeFilterLibraryStore) {
            positionBeforeFilterLibraryStore.set(JSON.parse(JSON.stringify($positionStore)));
            positionIndexBeforeFilterLibraryStore.set($currentPositionIndexStore);
        }
        selectedSavedFilter = filter;
        savedFilterName = filter.name;
        savedFilterCommand = filter.command;
        const editPosition = await LoadEditPosition(filter.name);
        if (editPosition) {
            positionStore.set(JSON.parse(editPosition));
        }
        currentPositionIndexStore.set(-1);
    }

    async function executeSavedFilter(filter) {
        const editPosition = await LoadEditPosition(filter.name);
        if (editPosition) {
            positionStore.set(JSON.parse(editPosition));
        }
        executeSearch({ command: filter.command, position: editPosition });
    }

    async function deleteSavedFilter() {
        if (selectedSavedFilter) {
            await DeleteFilter(selectedSavedFilter.id);
            await loadSavedFilters();
            selectedSavedFilter = null;
            savedFilterName = '';
            savedFilterCommand = '';
        }
    }

    function handleKeyDown(event) {
        if ($activeTabStore !== 'search') return;
        if (event.target.matches('input, textarea, select')) {
            event.stopPropagation();
            if (event.key === 'Enter') {
                handleSearch();
            }
            return;
        }
        // Allow all keys to propagate to the global handler for position navigation
    }

    onMount(() => {
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

<div class="search-panel">
    <!-- Left sub-tab sidebar -->
    <div class="sub-tab-sidebar">
        <button class="sub-tab-btn" class:active={activeSubTab === 'search'} on:click={() => activeSubTab = 'search'}>Search</button>
        <button class="sub-tab-btn" class:active={activeSubTab === 'history'} on:click={() => activeSubTab = 'history'}>History</button>
        <button class="sub-tab-btn" class:active={activeSubTab === 'saved'} on:click={() => activeSubTab = 'saved'}>Saved</button>
    </div>

    <!-- Content area -->
    <div class="sub-tab-content">
        {#if activeSubTab === 'search'}
            <!-- Filter Builder with checkboxes -->
            <div class="filter-section">
                <div class="action-bar top-action-bar">
                    <label class="search-in-results"><input type="checkbox" bind:checked={searchInCurrentResults} /> In results</label>
                    <span class="active-count">{activeFilterCount} active</span>
                    <button class="btn-search" on:click={handleSearch}>Search</button>
                    <button class="btn-clear" on:click={clearFilters}>Clear</button>
                </div>
                <div class="filter-groups">
                    {#each filterGroups as group}
                        <div class="filter-group">
                            <div class="group-header">{group.name}</div>
                            {#each group.filters as filter}
                                <div class="filter-item" class:active={filterEnabled[filter]}>
                                    <label class="filter-checkbox">
                                        <input type="checkbox" bind:checked={filterEnabled[filter]} />
                                        <span class="filter-label">{filter}</span>
                                    </label>
                                    {#if filterEnabled[filter]}
                                        <div class="filter-params">
                                            {#if filter === 'Pipcount Difference'}
                                                <div class="minmax-controls">
                                                    <label><input type="radio" bind:group={pipCountOption} value="min" /> Min <input type="number" bind:value={pipCountMin} class="num-input" disabled={pipCountOption !== 'min'} /></label>
                                                    <label><input type="radio" bind:group={pipCountOption} value="max" /> Max <input type="number" bind:value={pipCountMax} class="num-input" disabled={pipCountOption !== 'max'} /></label>
                                                    <label><input type="radio" bind:group={pipCountOption} value="range" /> Range <input type="number" bind:value={pipCountRangeMin} class="num-input" disabled={pipCountOption !== 'range'} /> <input type="number" bind:value={pipCountRangeMax} class="num-input" disabled={pipCountOption !== 'range'} /></label>
                                                </div>
                                            {:else if filter === 'Player Absolute Pipcount'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player1AbsolutePipCountOption} value="min" /> Min <input type="number" bind:value={player1AbsolutePipCountMin} class="num-input" min="0" max="375" disabled={player1AbsolutePipCountOption !== 'min'} /></label><label><input type="radio" bind:group={player1AbsolutePipCountOption} value="max" /> Max <input type="number" bind:value={player1AbsolutePipCountMax} class="num-input" min="0" max="375" disabled={player1AbsolutePipCountOption !== 'max'} /></label><label><input type="radio" bind:group={player1AbsolutePipCountOption} value="range" /> Range <input type="number" bind:value={player1AbsolutePipCountRangeMin} class="num-input" min="0" max="375" disabled={player1AbsolutePipCountOption !== 'range'} /> <input type="number" bind:value={player1AbsolutePipCountRangeMax} class="num-input" min="0" max="375" disabled={player1AbsolutePipCountOption !== 'range'} /></label></div>
                                            {:else if filter === 'Equity (millipoints)'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={equityOption} value="min" /> Min <input type="number" bind:value={equityMin} class="num-input" disabled={equityOption !== 'min'} /></label><label><input type="radio" bind:group={equityOption} value="max" /> Max <input type="number" bind:value={equityMax} class="num-input" disabled={equityOption !== 'max'} /></label><label><input type="radio" bind:group={equityOption} value="range" /> Range <input type="number" bind:value={equityRangeMin} class="num-input" disabled={equityOption !== 'range'} /> <input type="number" bind:value={equityRangeMax} class="num-input" disabled={equityOption !== 'range'} /></label></div>
                                            {:else if filter === 'Move Error (millipoints, Player 1)'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={moveErrorOption} value="min" /> Min <input type="number" bind:value={moveErrorMin} class="num-input" min="0" disabled={moveErrorOption !== 'min'} /></label><label><input type="radio" bind:group={moveErrorOption} value="max" /> Max <input type="number" bind:value={moveErrorMax} class="num-input" min="0" disabled={moveErrorOption !== 'max'} /></label><label><input type="radio" bind:group={moveErrorOption} value="range" /> Range <input type="number" bind:value={moveErrorRangeMin} class="num-input" min="0" disabled={moveErrorOption !== 'range'} /> <input type="number" bind:value={moveErrorRangeMax} class="num-input" min="0" disabled={moveErrorOption !== 'range'} /></label></div>
                                            {:else if filter === 'Win Rate'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={winRateOption} value="min" /> Min <input type="number" bind:value={winRateMin} class="num-input" min="0" max="100" disabled={winRateOption !== 'min'} /></label><label><input type="radio" bind:group={winRateOption} value="max" /> Max <input type="number" bind:value={winRateMax} class="num-input" min="0" max="100" disabled={winRateOption !== 'max'} /></label><label><input type="radio" bind:group={winRateOption} value="range" /> Range <input type="number" bind:value={winRateRangeMin} class="num-input" min="0" max="100" disabled={winRateOption !== 'range'} /> <input type="number" bind:value={winRateRangeMax} class="num-input" min="0" max="100" disabled={winRateOption !== 'range'} /></label></div>
                                            {:else if filter === 'Gammon Rate'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={gammonRateOption} value="min" /> Min <input type="number" bind:value={gammonRateMin} class="num-input" min="0" max="100" disabled={gammonRateOption !== 'min'} /></label><label><input type="radio" bind:group={gammonRateOption} value="max" /> Max <input type="number" bind:value={gammonRateMax} class="num-input" min="0" max="100" disabled={gammonRateOption !== 'max'} /></label><label><input type="radio" bind:group={gammonRateOption} value="range" /> Range <input type="number" bind:value={gammonRateRangeMin} class="num-input" min="0" max="100" disabled={gammonRateOption !== 'range'} /> <input type="number" bind:value={gammonRateRangeMax} class="num-input" min="0" max="100" disabled={gammonRateOption !== 'range'} /></label></div>
                                            {:else if filter === 'Backgammon Rate'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={backgammonRateOption} value="min" /> Min <input type="number" bind:value={backgammonRateMin} class="num-input" min="0" max="100" disabled={backgammonRateOption !== 'min'} /></label><label><input type="radio" bind:group={backgammonRateOption} value="max" /> Max <input type="number" bind:value={backgammonRateMax} class="num-input" min="0" max="100" disabled={backgammonRateOption !== 'max'} /></label><label><input type="radio" bind:group={backgammonRateOption} value="range" /> Range <input type="number" bind:value={backgammonRateRangeMin} class="num-input" min="0" max="100" disabled={backgammonRateOption !== 'range'} /> <input type="number" bind:value={backgammonRateRangeMax} class="num-input" min="0" max="100" disabled={backgammonRateOption !== 'range'} /></label></div>
                                            {:else if filter === 'Opponent Win Rate'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player2WinRateOption} value="min" /> Min <input type="number" bind:value={player2WinRateMin} class="num-input" min="0" max="100" disabled={player2WinRateOption !== 'min'} /></label><label><input type="radio" bind:group={player2WinRateOption} value="max" /> Max <input type="number" bind:value={player2WinRateMax} class="num-input" min="0" max="100" disabled={player2WinRateOption !== 'max'} /></label><label><input type="radio" bind:group={player2WinRateOption} value="range" /> Range <input type="number" bind:value={player2WinRateRangeMin} class="num-input" min="0" max="100" disabled={player2WinRateOption !== 'range'} /> <input type="number" bind:value={player2WinRateRangeMax} class="num-input" min="0" max="100" disabled={player2WinRateOption !== 'range'} /></label></div>
                                            {:else if filter === 'Opponent Gammon Rate'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player2GammonRateOption} value="min" /> Min <input type="number" bind:value={player2GammonRateMin} class="num-input" min="0" max="100" disabled={player2GammonRateOption !== 'min'} /></label><label><input type="radio" bind:group={player2GammonRateOption} value="max" /> Max <input type="number" bind:value={player2GammonRateMax} class="num-input" min="0" max="100" disabled={player2GammonRateOption !== 'max'} /></label><label><input type="radio" bind:group={player2GammonRateOption} value="range" /> Range <input type="number" bind:value={player2GammonRateRangeMin} class="num-input" min="0" max="100" disabled={player2GammonRateOption !== 'range'} /> <input type="number" bind:value={player2GammonRateRangeMax} class="num-input" min="0" max="100" disabled={player2GammonRateOption !== 'range'} /></label></div>
                                            {:else if filter === 'Opponent Backgammon Rate'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player2BackgammonRateOption} value="min" /> Min <input type="number" bind:value={player2BackgammonRateMin} class="num-input" min="0" max="100" disabled={player2BackgammonRateOption !== 'min'} /></label><label><input type="radio" bind:group={player2BackgammonRateOption} value="max" /> Max <input type="number" bind:value={player2BackgammonRateMax} class="num-input" min="0" max="100" disabled={player2BackgammonRateOption !== 'max'} /></label><label><input type="radio" bind:group={player2BackgammonRateOption} value="range" /> Range <input type="number" bind:value={player2BackgammonRateRangeMin} class="num-input" min="0" max="100" disabled={player2BackgammonRateOption !== 'range'} /> <input type="number" bind:value={player2BackgammonRateRangeMax} class="num-input" min="0" max="100" disabled={player2BackgammonRateOption !== 'range'} /></label></div>
                                            {:else if filter === 'Player Checker-Off'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player1CheckerOffOption} value="min" /> Min <input type="number" bind:value={player1CheckerOffMin} class="num-input" min="0" max="15" disabled={player1CheckerOffOption !== 'min'} /></label><label><input type="radio" bind:group={player1CheckerOffOption} value="max" /> Max <input type="number" bind:value={player1CheckerOffMax} class="num-input" min="0" max="15" disabled={player1CheckerOffOption !== 'max'} /></label><label><input type="radio" bind:group={player1CheckerOffOption} value="range" /> Range <input type="number" bind:value={player1CheckerOffRangeMin} class="num-input" min="0" max="15" disabled={player1CheckerOffOption !== 'range'} /> <input type="number" bind:value={player1CheckerOffRangeMax} class="num-input" min="0" max="15" disabled={player1CheckerOffOption !== 'range'} /></label></div>
                                            {:else if filter === 'Opponent Checker-Off'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player2CheckerOffOption} value="min" /> Min <input type="number" bind:value={player2CheckerOffMin} class="num-input" min="0" max="15" disabled={player2CheckerOffOption !== 'min'} /></label><label><input type="radio" bind:group={player2CheckerOffOption} value="max" /> Max <input type="number" bind:value={player2CheckerOffMax} class="num-input" min="0" max="15" disabled={player2CheckerOffOption !== 'max'} /></label><label><input type="radio" bind:group={player2CheckerOffOption} value="range" /> Range <input type="number" bind:value={player2CheckerOffRangeMin} class="num-input" min="0" max="15" disabled={player2CheckerOffOption !== 'range'} /> <input type="number" bind:value={player2CheckerOffRangeMax} class="num-input" min="0" max="15" disabled={player2CheckerOffOption !== 'range'} /></label></div>
                                            {:else if filter === 'Player Back Checker'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player1BackCheckerOption} value="min" /> Min <input type="number" bind:value={player1BackCheckerMin} class="num-input" min="0" max="15" disabled={player1BackCheckerOption !== 'min'} /></label><label><input type="radio" bind:group={player1BackCheckerOption} value="max" /> Max <input type="number" bind:value={player1BackCheckerMax} class="num-input" min="0" max="15" disabled={player1BackCheckerOption !== 'max'} /></label><label><input type="radio" bind:group={player1BackCheckerOption} value="range" /> Range <input type="number" bind:value={player1BackCheckerRangeMin} class="num-input" min="0" max="15" disabled={player1BackCheckerOption !== 'range'} /> <input type="number" bind:value={player1BackCheckerRangeMax} class="num-input" min="0" max="15" disabled={player1BackCheckerOption !== 'range'} /></label></div>
                                            {:else if filter === 'Opponent Back Checker'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player2BackCheckerOption} value="min" /> Min <input type="number" bind:value={player2BackCheckerMin} class="num-input" min="0" max="15" disabled={player2BackCheckerOption !== 'min'} /></label><label><input type="radio" bind:group={player2BackCheckerOption} value="max" /> Max <input type="number" bind:value={player2BackCheckerMax} class="num-input" min="0" max="15" disabled={player2BackCheckerOption !== 'max'} /></label><label><input type="radio" bind:group={player2BackCheckerOption} value="range" /> Range <input type="number" bind:value={player2BackCheckerRangeMin} class="num-input" min="0" max="15" disabled={player2BackCheckerOption !== 'range'} /> <input type="number" bind:value={player2BackCheckerRangeMax} class="num-input" min="0" max="15" disabled={player2BackCheckerOption !== 'range'} /></label></div>
                                            {:else if filter === 'Player Checker in the Zone'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player1CheckerInZoneOption} value="min" /> Min <input type="number" bind:value={player1CheckerInZoneMin} class="num-input" min="0" max="15" disabled={player1CheckerInZoneOption !== 'min'} /></label><label><input type="radio" bind:group={player1CheckerInZoneOption} value="max" /> Max <input type="number" bind:value={player1CheckerInZoneMax} class="num-input" min="0" max="15" disabled={player1CheckerInZoneOption !== 'max'} /></label><label><input type="radio" bind:group={player1CheckerInZoneOption} value="range" /> Range <input type="number" bind:value={player1CheckerInZoneRangeMin} class="num-input" min="0" max="15" disabled={player1CheckerInZoneOption !== 'range'} /> <input type="number" bind:value={player1CheckerInZoneRangeMax} class="num-input" min="0" max="15" disabled={player1CheckerInZoneOption !== 'range'} /></label></div>
                                            {:else if filter === 'Opponent Checker in the Zone'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player2CheckerInZoneOption} value="min" /> Min <input type="number" bind:value={player2CheckerInZoneMin} class="num-input" min="0" max="15" disabled={player2CheckerInZoneOption !== 'min'} /></label><label><input type="radio" bind:group={player2CheckerInZoneOption} value="max" /> Max <input type="number" bind:value={player2CheckerInZoneMax} class="num-input" min="0" max="15" disabled={player2CheckerInZoneOption !== 'max'} /></label><label><input type="radio" bind:group={player2CheckerInZoneOption} value="range" /> Range <input type="number" bind:value={player2CheckerInZoneRangeMin} class="num-input" min="0" max="15" disabled={player2CheckerInZoneOption !== 'range'} /> <input type="number" bind:value={player2CheckerInZoneRangeMax} class="num-input" min="0" max="15" disabled={player2CheckerInZoneOption !== 'range'} /></label></div>
                                            {:else if filter === 'Player Outfield Blot'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player1OutfieldBlotOption} value="min" /> Min <input type="number" bind:value={player1OutfieldBlotMin} class="num-input" min="0" max="15" disabled={player1OutfieldBlotOption !== 'min'} /></label><label><input type="radio" bind:group={player1OutfieldBlotOption} value="max" /> Max <input type="number" bind:value={player1OutfieldBlotMax} class="num-input" min="0" max="15" disabled={player1OutfieldBlotOption !== 'max'} /></label><label><input type="radio" bind:group={player1OutfieldBlotOption} value="range" /> Range <input type="number" bind:value={player1OutfieldBlotRangeMin} class="num-input" min="0" max="15" disabled={player1OutfieldBlotOption !== 'range'} /> <input type="number" bind:value={player1OutfieldBlotRangeMax} class="num-input" min="0" max="15" disabled={player1OutfieldBlotOption !== 'range'} /></label></div>
                                            {:else if filter === 'Opponent Outfield Blot'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player2OutfieldBlotOption} value="min" /> Min <input type="number" bind:value={player2OutfieldBlotMin} class="num-input" min="0" max="15" disabled={player2OutfieldBlotOption !== 'min'} /></label><label><input type="radio" bind:group={player2OutfieldBlotOption} value="max" /> Max <input type="number" bind:value={player2OutfieldBlotMax} class="num-input" min="0" max="15" disabled={player2OutfieldBlotOption !== 'max'} /></label><label><input type="radio" bind:group={player2OutfieldBlotOption} value="range" /> Range <input type="number" bind:value={player2OutfieldBlotRangeMin} class="num-input" min="0" max="15" disabled={player2OutfieldBlotOption !== 'range'} /> <input type="number" bind:value={player2OutfieldBlotRangeMax} class="num-input" min="0" max="15" disabled={player2OutfieldBlotOption !== 'range'} /></label></div>
                                            {:else if filter === 'Player Jan Blot'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player1JanBlotOption} value="min" /> Min <input type="number" bind:value={player1JanBlotMin} class="num-input" min="0" max="15" disabled={player1JanBlotOption !== 'min'} /></label><label><input type="radio" bind:group={player1JanBlotOption} value="max" /> Max <input type="number" bind:value={player1JanBlotMax} class="num-input" min="0" max="15" disabled={player1JanBlotOption !== 'max'} /></label><label><input type="radio" bind:group={player1JanBlotOption} value="range" /> Range <input type="number" bind:value={player1JanBlotRangeMin} class="num-input" min="0" max="15" disabled={player1JanBlotOption !== 'range'} /> <input type="number" bind:value={player1JanBlotRangeMax} class="num-input" min="0" max="15" disabled={player1JanBlotOption !== 'range'} /></label></div>
                                            {:else if filter === 'Opponent Jan Blot'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={player2JanBlotOption} value="min" /> Min <input type="number" bind:value={player2JanBlotMin} class="num-input" min="0" max="15" disabled={player2JanBlotOption !== 'min'} /></label><label><input type="radio" bind:group={player2JanBlotOption} value="max" /> Max <input type="number" bind:value={player2JanBlotMax} class="num-input" min="0" max="15" disabled={player2JanBlotOption !== 'max'} /></label><label><input type="radio" bind:group={player2JanBlotOption} value="range" /> Range <input type="number" bind:value={player2JanBlotRangeMin} class="num-input" min="0" max="15" disabled={player2JanBlotOption !== 'range'} /> <input type="number" bind:value={player2JanBlotRangeMax} class="num-input" min="0" max="15" disabled={player2JanBlotOption !== 'range'} /></label></div>
                                            {:else if filter === 'Search Text'}
                                                <div class="text-control"><span class="hint">(tag1;tag2;...)</span><input type="text" bind:value={searchText} class="text-input" /></div>
                                            {:else if filter === 'Best Move or Cube Decision'}
                                                <div class="text-control"><span class="hint">(pattern1;pattern2;...)</span><input type="text" bind:value={movePattern} class="text-input" /></div>
                                            {:else if filter === 'Creation Date'}
                                                <div class="minmax-controls"><label><input type="radio" bind:group={creationDateOption} value="min" /> Min <input type="date" bind:value={creationDateMin} class="date-input" disabled={creationDateOption !== 'min'} /></label><label><input type="radio" bind:group={creationDateOption} value="max" /> Max <input type="date" bind:value={creationDateMax} class="date-input" disabled={creationDateOption !== 'max'} /></label><label><input type="radio" bind:group={creationDateOption} value="range" /> Range <input type="date" bind:value={creationDateRangeMin} class="date-input" disabled={creationDateOption !== 'range'} /> <input type="date" bind:value={creationDateRangeMax} class="date-input" disabled={creationDateOption !== 'range'} /></label></div>
                                            {:else if filter === 'Match IDs'}
                                                <div class="text-control"><span class="hint">(e.g. 3 or 2,5 for range)</span><input type="text" bind:value={matchIDsInput} class="text-input" placeholder="ID or range" /></div>
                                            {:else if filter === 'Tournament IDs'}
                                                <div class="text-control"><span class="hint">(e.g. 1 or 1,3 for range)</span><input type="text" bind:value={tournamentIDsInput} class="text-input" placeholder="ID or range" /></div>
                                            {/if}
                                        </div>
                                    {/if}
                                </div>
                            {/each}
                        </div>
                    {/each}
                </div>
            </div>

        {:else if activeSubTab === 'history'}
            <div class="history-section">
                {#if searchHistory.length === 0}
                    <p class="empty-message">No search history yet.</p>
                {:else}
                    <div class="history-table-container">
                        <table class="history-table">
                            <thead><tr><th>Date</th><th>Command</th><th>Actions</th></tr></thead>
                            <tbody>
                                {#each searchHistory as search}
                                    <tr class:selected={selectedSearch === search} on:click={() => selectSearch(search)} on:dblclick={() => handleDoubleClick(search)}>
                                        <td class="date-cell">{formatTimestamp(search.timestamp)}</td>
                                        <td class="command-cell">{search.command}</td>
                                        <td class="actions-cell">
                                            <button class="action-btn" class:in-library={isInFilterLibrary(search)} on:click|stopPropagation={() => showAddToLibraryDialog(search)} title="Save to bookmarks">
                                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14"><path stroke-linecap="round" stroke-linejoin="round" d="M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0 1 11.186 0Z" /></svg>
                                            </button>
                                            <button class="action-btn delete-btn" on:click|stopPropagation={(e) => deleteSearch(search, e)} title="Delete">
                                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14"><path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" /></svg>
                                            </button>
                                        </td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                {/if}
            </div>

        {:else if activeSubTab === 'saved'}
            <div class="saved-section">
                {#if savedFilters.length === 0}
                    <p class="empty-message">No saved searches. Bookmark searches from History.</p>
                {:else}
                    <div class="saved-list">
                        {#each savedFilters as sf}
                            <div class="saved-item" class:selected={selectedSavedFilter && selectedSavedFilter.id === sf.id} on:click={() => selectSavedFilter(sf)} on:dblclick={() => executeSavedFilter(sf)}>
                                <span class="saved-name">{sf.name}</span>
                                <span class="saved-cmd">{sf.command}</span>
                                <button class="action-btn delete-btn" on:click|stopPropagation={() => { selectedSavedFilter = sf; deleteSavedFilter(); }} title="Remove">
                                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14"><path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" /></svg>
                                </button>
                            </div>
                        {/each}
                    </div>
                {/if}
            </div>
        {/if}
    </div>
</div>

{#if showSaveDialog}
    <div class="save-dialog-overlay" on:click|self={cancelSaveDialog}>
        <div class="save-dialog">
            <h3>Save Search</h3>
            <p class="command-preview">{selectedSearch?.command || ''}</p>
            <div class="dialog-form">
                <label for="filterNameInput">Name:</label>
                <input type="text" id="filterNameInput" bind:value={filterName} placeholder="Enter name" on:keydown={(e) => e.key === 'Enter' && saveToFilterLibrary()} />
            </div>
            <div class="dialog-actions">
                <button class="btn-search" on:click|stopPropagation={saveToFilterLibrary}>Save</button>
                <button class="btn-clear" on:click|stopPropagation={cancelSaveDialog}>Cancel</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .search-panel { display: flex; height: 100%; background: white; overflow: hidden; font-size: 12px; user-select: none; -webkit-user-select: none; }
    .search-panel input, .search-panel textarea { user-select: text; -webkit-user-select: text; }
    .sub-tab-sidebar { display: flex; flex-direction: column; width: 70px; flex-shrink: 0; background: #f5f5f5; border-right: 1px solid #ddd; }
    .sub-tab-btn { border: none; background: transparent; padding: 8px 4px; font-size: 11px; color: #666; cursor: pointer; border-left: 2px solid transparent; text-align: center; transition: background 0.15s; user-select: none; -webkit-user-select: none; }
    .sub-tab-btn:hover { background: #e8e8e8; }
    .sub-tab-btn.active { color: #333; font-weight: 600; background: #fff; border-left-color: #555; }
    .sub-tab-content { flex: 1; min-width: 0; overflow-y: auto; overflow-x: hidden; }
    .filter-section { display: flex; flex-direction: column; height: 100%; }
    .top-action-bar { position: sticky; top: 0; background: white; z-index: 2; border-bottom: 1px solid #ddd; padding: 6px 8px; }
    .filter-groups { flex: 1; overflow-y: auto; padding: 4px 8px 8px; }
    .filter-group { margin-bottom: 2px; }
    .group-header { font-size: 11px; font-weight: 700; color: #555; text-transform: uppercase; letter-spacing: 0.5px; padding: 6px 0 2px; border-bottom: 1px solid #eee; margin-bottom: 2px; user-select: none; -webkit-user-select: none; }
    .filter-item { padding: 2px 0 2px 4px; border-radius: 3px; }
    .filter-item.active { background: #f0f7ff; }
    .filter-checkbox { display: flex; align-items: center; gap: 6px; cursor: pointer; padding: 1px 0; }
    .filter-checkbox input[type="checkbox"] { margin: 0; cursor: pointer; accent-color: #6c757d; }
    .filter-label { font-size: 12px; color: #333; user-select: none; }
    .filter-item.active .filter-label { font-weight: 500; color: #1a1a1a; }
    .filter-params { margin: 2px 0 4px 22px; }
    .action-bar { display: flex; align-items: center; gap: 8px; }
    .active-count { font-size: 11px; color: #888; margin-right: auto; }
    .search-in-results { display: flex; align-items: center; gap: 3px; font-size: 11px; color: #666; cursor: pointer; user-select: none; -webkit-user-select: none; }
    .btn-search { padding: 4px 12px; border: none; border-radius: 3px; cursor: pointer; font-size: 12px; background: #6c757d; color: white; }
    .btn-search:hover { background: #5a6268; }
    .btn-clear { padding: 4px 12px; border: none; border-radius: 3px; cursor: pointer; font-size: 12px; background: #ccc; color: #333; }
    .btn-clear:hover { background: #999; }
    .minmax-controls { display: flex; flex-direction: column; gap: 2px; }
    .minmax-controls label { display: flex; align-items: center; gap: 4px; font-size: 12px; user-select: none; -webkit-user-select: none; }
    .num-input { width: 60px; font-size: 12px; padding: 2px 3px; }
    .date-input { font-size: 12px; padding: 2px 3px; }
    .text-control { display: flex; align-items: center; gap: 6px; }
    .hint { font-size: 11px; color: #888; white-space: nowrap; }
    .text-input { flex: 1; font-size: 12px; padding: 3px 4px; max-width: 200px; }

    .history-section { flex: 1; display: flex; flex-direction: column; overflow: hidden; padding: 4px; }
    .empty-message { text-align: center; color: #888; font-size: 11px; padding: 12px; }
    .history-table-container { flex: 1; overflow-y: auto; }
    .history-table { width: 100%; border-collapse: collapse; }
    .history-table thead { position: sticky; top: 0; background: #f2f2f2; z-index: 1; }
    .history-table th { padding: 2px 4px; text-align: center; font-weight: bold; font-size: 11px; border: 1px solid #ddd; user-select: none; }
    .history-table td { padding: 2px 4px; border: 1px solid #ddd; text-align: center; font-size: 11px; }
    .history-table tbody tr { cursor: pointer; }
    .history-table tbody tr:hover { background-color: #e6f2ff; }
    .history-table tbody tr.selected { background-color: #b3d9ff !important; }
    .date-cell { width: 140px; white-space: nowrap; }
    .command-cell { font-family: monospace; }
    .actions-cell { width: 60px; }
    .action-btn { background: none; border: none; cursor: pointer; padding: 1px 3px; color: #666; display: inline-flex; align-items: center; }
    .action-btn:hover { color: #333; }
    .action-btn.in-library { color: #333; }
    .delete-btn:hover { color: #c00; }

    .saved-section { padding: 4px; overflow-y: auto; height: 100%; }
    .saved-list { display: flex; flex-direction: column; gap: 1px; }
    .saved-item { display: flex; align-items: center; gap: 8px; padding: 4px 8px; cursor: pointer; border-bottom: 1px solid #f0f0f0; }
    .saved-item:hover { background: #e6f2ff; }
    .saved-item.selected { background: #b3d9ff; }
    .saved-name { font-weight: 600; min-width: 120px; font-size: 11px; }
    .saved-cmd { flex: 1; font-family: monospace; font-size: 11px; color: #555; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

    input:disabled { background-color: #e0e0e0; }
    .save-dialog-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.5); display: flex; justify-content: center; align-items: center; z-index: 1001; }
    .save-dialog { background: white; border-radius: 8px; padding: 24px; width: 90%; max-width: 400px; box-shadow: 0 4px 6px rgba(0,0,0,0.1); }
    .save-dialog h3 { margin: 0 0 12px; font-size: 14px; }
    .command-preview { background: #f5f5f5; padding: 8px; border-radius: 4px; font-family: monospace; font-size: 12px; margin-bottom: 12px; word-break: break-all; }
    .dialog-form { margin-bottom: 12px; }
    .dialog-form label { display: block; margin-bottom: 4px; font-weight: bold; font-size: 12px; user-select: none; -webkit-user-select: none; }
    .dialog-form input { width: 100%; padding: 6px; border: 1px solid #ddd; border-radius: 4px; font-size: 13px; box-sizing: border-box; }
    .dialog-actions { display: flex; justify-content: flex-end; gap: 8px; }
</style>
