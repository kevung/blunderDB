<script>
    import { logger } from '../utils/logger.js';
    import { t, tMsg } from '../i18n';
    import { onMount, onDestroy, tick, untrack } from 'svelte';
    import { statusBarTextStore, currentPositionIndexStore, activeTabStore } from '../stores/uiStore';
    import { positionStore, positionsStore, positionBeforeFilterLibraryStore, positionIndexBeforeFilterLibraryStore } from '../stores/positionStore';
    import { searchExcludePositionStore, searchStructureModeStore, searchOfferedCubeStore, emptySearchBoardPosition, boardHasCheckers } from '../stores/searchExcludePositionStore';
    import { searchHistoryStore } from '../stores/searchHistoryStore';
    import { buildFilterTokens, buildSearchCommand, parseFilterTokens, parseSearchCommand, filterTokenHint } from '../services/searchFilterService.js';
    import { filterLibraryStore } from '../stores/filterLibraryStore';
    import { searchParamsStore } from '../stores/searchParamsStore';
    import { databaseLoadedStore } from '../stores/databaseStore';
    import { SaveSearchHistory, LoadSearchHistory, DeleteSearchHistoryEntry, LoadFilters, DeleteFilter, LoadEditPosition, LoadExcludePosition } from '../../wailsjs/go/database/Database.js';

    let { onLoadPositionsByFilters, onAddToFilterLibrary } = $props();

    // Sub-tab state
    let activeSubTab = $state('search'); // 'search', 'history', 'saved'

    // Filter state
    let filterEnabled = $state({});
    let searchInCurrentResults = $state(false);
    let openInNewTab = $state(false);

    let searchText = $state('');
    let movePattern = $state('');
    let matchIDsInput = $state('');
    let tournamentIDsInput = $state('');
    let playerName = $state('');

    let pipCountOption = $state('min');
    let pipCountMin = $state(-375);
    let pipCountMax = $state(375);
    let pipCountRangeMin = $state(-375);
    let pipCountRangeMax = $state(375);
    let winRateOption = $state('min');
    let winRateMin = $state(0);
    let winRateMax = $state(100);
    let winRateRangeMin = $state(0);
    let winRateRangeMax = $state(100);
    let gammonRateOption = $state('min');
    let gammonRateMin = $state(0);
    let gammonRateMax = $state(100);
    let gammonRateRangeMin = $state(0);
    let gammonRateRangeMax = $state(100);
    let backgammonRateOption = $state('min');
    let backgammonRateMin = $state(0);
    let backgammonRateMax = $state(100);
    let backgammonRateRangeMin = $state(0);
    let backgammonRateRangeMax = $state(100);
    let player2WinRateOption = $state('min');
    let player2WinRateMin = $state(0);
    let player2WinRateMax = $state(100);
    let player2WinRateRangeMin = $state(0);
    let player2WinRateRangeMax = $state(100);
    let player2GammonRateOption = $state('min');
    let player2GammonRateMin = $state(0);
    let player2GammonRateMax = $state(100);
    let player2GammonRateRangeMin = $state(0);
    let player2GammonRateRangeMax = $state(100);
    let player2BackgammonRateOption = $state('min');
    let player2BackgammonRateMin = $state(0);
    let player2BackgammonRateMax = $state(100);
    let player2BackgammonRateRangeMin = $state(0);
    let player2BackgammonRateRangeMax = $state(100);
    let player1CheckerOffOption = $state('min');
    let player1CheckerOffMin = $state(0);
    let player1CheckerOffMax = $state(15);
    let player1CheckerOffRangeMin = $state(0);
    let player1CheckerOffRangeMax = $state(15);
    let player2CheckerOffOption = $state('min');
    let player2CheckerOffMin = $state(0);
    let player2CheckerOffMax = $state(15);
    let player2CheckerOffRangeMin = $state(0);
    let player2CheckerOffRangeMax = $state(15);
    let player1BackCheckerOption = $state('min');
    let player1BackCheckerMin = $state(0);
    let player1BackCheckerMax = $state(15);
    let player1BackCheckerRangeMin = $state(0);
    let player1BackCheckerRangeMax = $state(15);
    let player2BackCheckerOption = $state('min');
    let player2BackCheckerMin = $state(0);
    let player2BackCheckerMax = $state(15);
    let player2BackCheckerRangeMin = $state(0);
    let player2BackCheckerRangeMax = $state(15);
    let player1CheckerInZoneOption = $state('min');
    let player1CheckerInZoneMin = $state(0);
    let player1CheckerInZoneMax = $state(15);
    let player1CheckerInZoneRangeMin = $state(0);
    let player1CheckerInZoneRangeMax = $state(15);
    let player2CheckerInZoneOption = $state('min');
    let player2CheckerInZoneMin = $state(0);
    let player2CheckerInZoneMax = $state(15);
    let player2CheckerInZoneRangeMin = $state(0);
    let player2CheckerInZoneRangeMax = $state(15);
    let player1AbsolutePipCountOption = $state('min');
    let player1AbsolutePipCountMin = $state(0);
    let player1AbsolutePipCountMax = $state(375);
    let player1AbsolutePipCountRangeMin = $state(0);
    let player1AbsolutePipCountRangeMax = $state(375);
    let equityOption = $state('min');
    let equityMin = $state(-1000);
    let equityMax = $state(1000);
    let equityRangeMin = $state(-1000);
    let equityRangeMax = $state(1000);
    let moveErrorOption = $state('min');
    let moveErrorMin = $state(0);
    let moveErrorMax = $state(1000);
    let moveErrorRangeMin = $state(0);
    let moveErrorRangeMax = $state(1000);
    let player1OutfieldBlotOption = $state('min');
    let player1OutfieldBlotMin = $state(0);
    let player1OutfieldBlotMax = $state(15);
    let player1OutfieldBlotRangeMin = $state(0);
    let player1OutfieldBlotRangeMax = $state(15);
    let player2OutfieldBlotOption = $state('min');
    let player2OutfieldBlotMin = $state(0);
    let player2OutfieldBlotMax = $state(15);
    let player2OutfieldBlotRangeMin = $state(0);
    let player2OutfieldBlotRangeMax = $state(15);
    let player1JanBlotOption = $state('min');
    let player1JanBlotMin = $state(0);
    let player1JanBlotMax = $state(15);
    let player1JanBlotRangeMin = $state(0);
    let player1JanBlotRangeMax = $state(15);
    let player2JanBlotOption = $state('min');
    let player2JanBlotMin = $state(0);
    let player2JanBlotMax = $state(15);
    let player2JanBlotRangeMin = $state(0);
    let player2JanBlotRangeMax = $state(15);
    let diceRollOption = $state('both'); // 'both' | 'first'
    // Decision-type filter (Display group). The "Include Decision Type" checkbox is
    // the on/off ("Indifférent") gate; when on, decisionMode reflects the board
    // (Pions = checker / Cube = cube decision) and cubeSubType refines a cube
    // decision into all / double-no-double / take-pass.
    let decisionMode = $state('checker'); // 'checker' | 'cube' — meaningful while the filter is enabled
    let cubeSubType = $state('all'); // 'all' | 'double' | 'takepass'
    let lastCheckerDice = $state([3, 1]); // remembers the roll when toggling Pions ⇄ Cube from the panel
    let creationDateOption = $state('min');
    let creationDateMin = $state('');
    let creationDateMax = $state('');
    let creationDateRangeMin = $state('');
    let creationDateRangeMax = $state('');

    // History state — mirrors of stores, always current
    let searchHistory = $derived($searchHistoryStore);
    let selectedSearch = $state(null);
    let showSaveDialog = $state(false);
    let filterName = $state('');

    // Saved (filter library) state
    let savedFilters = $derived($filterLibraryStore || []);
    let selectedSavedFilter = $state(null);
    let _savedFilterName = '';
    let _savedFilterCommand = '';

    let availableFilters = [
        'Include Cube',
        'Include Score',
        'Include Decision Type',
        'Include Dice Roll',
        'No Contact',
        'Mirror Position',
        'Pipcount Difference',
        'Player Absolute Pipcount',
        'Equity (millipoints)',
        'Move Error (millipoints, Player 1)',
        'Win Rate',
        'Gammon Rate',
        'Backgammon Rate',
        'Opponent Win Rate',
        'Opponent Gammon Rate',
        'Opponent Backgammon Rate',
        'Player Checker-Off',
        'Opponent Checker-Off',
        'Player Back Checker',
        'Opponent Back Checker',
        'Player Checker in the Zone',
        'Opponent Checker in the Zone',
        'Player Outfield Blot',
        'Opponent Outfield Blot',
        'Player Jan Blot',
        'Opponent Jan Blot',
        'Search Text',
        'Best Move or Cube Decision',
        'Creation Date',
        'Match IDs',
        'Tournament IDs',
        'Player'
    ];

    // Canonical filter/group names stay in English because they double as logic
    // keys (object keys for filterEnabled/params and `{#if filter === '...'}`
    // branches). These maps yield the i18n key slug for the *displayed* label
    // only. The filter→token mapping lives in services/searchFilterService.js.
    const filterKeySlug = {
        'Include Cube': 'includeCube',
        'Include Score': 'includeScore',
        'Include Decision Type': 'includeDecisionType',
        'Include Dice Roll': 'includeDiceRoll',
        'No Contact': 'noContact',
        'Mirror Position': 'mirrorPosition',
        'Pipcount Difference': 'pipcountDifference',
        'Player Absolute Pipcount': 'playerAbsolutePipcount',
        'Equity (millipoints)': 'equity',
        'Move Error (millipoints, Player 1)': 'moveError',
        'Win Rate': 'winRate',
        'Gammon Rate': 'gammonRate',
        'Backgammon Rate': 'backgammonRate',
        'Opponent Win Rate': 'opponentWinRate',
        'Opponent Gammon Rate': 'opponentGammonRate',
        'Opponent Backgammon Rate': 'opponentBackgammonRate',
        'Player Checker-Off': 'playerCheckerOff',
        'Opponent Checker-Off': 'opponentCheckerOff',
        'Player Back Checker': 'playerBackChecker',
        'Opponent Back Checker': 'opponentBackChecker',
        'Player Checker in the Zone': 'playerCheckerInZone',
        'Opponent Checker in the Zone': 'opponentCheckerInZone',
        'Player Outfield Blot': 'playerOutfieldBlot',
        'Opponent Outfield Blot': 'opponentOutfieldBlot',
        'Player Jan Blot': 'playerJanBlot',
        'Opponent Jan Blot': 'opponentJanBlot',
        'Search Text': 'searchText',
        'Best Move or Cube Decision': 'bestMoveOrCubeDecision',
        'Creation Date': 'creationDate',
        'Match IDs': 'matchIDs',
        'Tournament IDs': 'tournamentIDs',
        Player: 'player'
    };
    const groupKeySlug = {
        Display: 'display',
        Position: 'position',
        Pipcount: 'pipcount',
        'Equity / Error': 'equityError',
        'Player Rates': 'playerRates',
        'Opponent Rates': 'opponentRates',
        Checkers: 'checkers',
        Blots: 'blots',
        'Text / Pattern': 'textPattern',
        Other: 'other'
    };
    function filterLabel(filter) {
        const name = typeof filter === 'string' ? filter : (filter?.name ?? '');
        return filterKeySlug[name] ? $t('search.filters.' + filterKeySlug[name]) : name;
    }
    function groupLabel(group) {
        const name = String(group ?? '');
        return groupKeySlug[name] ? $t('search.filterGroups.' + groupKeySlug[name]) : name;
    }

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
        { name: 'Other', filters: ['Creation Date', 'Match IDs', 'Tournament IDs', 'Player'] }
    ];

    // Which structure the main board is currently editing: 'include' (au moins)
    // or 'exclude' (sauf). While in 'exclude' mode the include board is stashed.
    // Declared before restoreSearchState() below, which assigns structureMode.
    let structureMode = $state('include');
    let includeBoardStash = $state(null);

    // Initialize all filters as disabled, then restore previous search state if available
    availableFilters.forEach((f) => (filterEnabled[f] = false));
    restoreSearchState();

    let activeFilterCount = $derived(availableFilters.filter((f) => filterEnabled[f]).length);
    // Track board position only while the search tab is active.
    // When the user switches away, App.svelte's exitEditMode() fires synchronously
    // and updates positionStore to a DB position before onDestroy runs.
    // This reactive block stops updating once $activeTabStore !== 'search',
    // so savedSearchPosition always holds the last board the user saw on this panel.
    let savedSearchPosition = $state(null);
    $effect(() => {
        // Only the include structure is tracked here; while editing the exclude
        // structure the main board holds the "Sauf" pattern, not the include one.
        if ($activeTabStore === 'search' && structureMode === 'include') {
            savedSearchPosition = JSON.parse(JSON.stringify($positionStore));
        }
    });

    // Board → panel: editing the board (placing dice → checker, clicking a player
    // rectangle → cube) drives the Pions/Cube choice. Only tracks the include board.
    $effect(() => {
        const dt = $positionStore?.decision_type === 1 ? 'cube' : 'checker';
        if ($activeTabStore === 'search' && structureMode === 'include') {
            untrack(() => {
                if (decisionMode !== dt) decisionMode = dt;
            });
        }
    });

    // A cube decision has no dice, so the Dice Roll filter is meaningless when the
    // decision-type filter constrains to a cube decision.
    $effect(() => {
        if (filterEnabled['Include Decision Type'] && decisionMode === 'cube') {
            untrack(() => {
                filterEnabled['Include Dice Roll'] = false;
            });
        }
    });

    // Panel → board: choosing Pions/Cube edits the include board (and remembers the
    // last real roll so toggling back to Pions restores it).
    function selectDecisionMode(mode) {
        if (structureMode !== 'include') return;
        positionStore.update((p) => {
            if (mode === 'cube') {
                if (p.decision_type !== 1) {
                    if (Array.isArray(p.dice) && (p.dice[0] || p.dice[1])) {
                        lastCheckerDice = [p.dice[0], p.dice[1]];
                    }
                    p.decision_type = 1;
                    p.dice = [0, 0];
                }
            } else {
                p.decision_type = 0;
                if (!Array.isArray(p.dice) || (!p.dice[0] && !p.dice[1])) {
                    p.dice = [lastCheckerDice[0], lastCheckerDice[1]];
                }
            }
            return p;
        });
        decisionMode = mode;
        if (mode === 'cube' && cubeSubType === 'takepass') applyOfferedCube();
    }

    // applyOfferedCube turns the board cube into a centered "offered" cube (owner
    // -1), matching how take/pass positions are stored (the board can't otherwise
    // build a centered value>1 cube). An offered cube is at least a double.
    function applyOfferedCube() {
        if (structureMode !== 'include') return;
        positionStore.update((p) => {
            p.decision_type = 1;
            p.dice = [0, 0];
            p.cube.owner = -1;
            if (!p.cube.value || p.cube.value < 1) p.cube.value = 1;
            return p;
        });
    }

    // Panel → cube sub-type. Take/pass needs the board cube rendered/edited as a
    // centered offered cube; double/all use the normal owner-based cube.
    function selectCubeSubType(value) {
        cubeSubType = value;
        if (value === 'takepass') {
            applyOfferedCube();
        } else if (structureMode === 'include') {
            // Leaving take/pass: reset the offered cube (centered, value > 1) back
            // to the initial centered 1-cube. An owned cube set in double mode
            // (owner 0/1) is preserved.
            positionStore.update((p) => {
                if (p.cube.owner === -1 && p.cube.value >= 1) {
                    p.cube.value = 0;
                }
                return p;
            });
        }
    }

    // Drive the offered-cube flag the board reads: on only while building a
    // take/pass query on the include board of the search tab.
    $effect(() => {
        const offered = $activeTabStore === 'search' && structureMode === 'include' && !!filterEnabled['Include Decision Type'] && decisionMode === 'cube' && cubeSubType === 'takepass';
        untrack(() => searchOfferedCubeStore.set(offered));
    });

    // restoreExcludeStructure resets the structure editing state to 'include' and
    // loads the exclude ("Sauf") board from a replayed history/saved entry (or an
    // empty board when the entry has none).
    function restoreExcludeStructure(excludePositionJSON) {
        structureMode = 'include';
        searchStructureModeStore.set('include');
        includeBoardStash = null;
        if (excludePositionJSON) {
            try {
                searchExcludePositionStore.set(JSON.parse(excludePositionJSON));
                return;
            } catch (_e) {
                /* fall through to empty */
            }
        }
        searchExcludePositionStore.set(emptySearchBoardPosition());
    }

    // switchStructureMode swaps which checker structure the main board edits.
    function switchStructureMode(mode) {
        if (mode === structureMode) return;
        if (mode === 'exclude') {
            includeBoardStash = JSON.parse(JSON.stringify($positionStore));
            structureMode = 'exclude';
            searchStructureModeStore.set('exclude');
            positionStore.set(JSON.parse(JSON.stringify($searchExcludePositionStore)));
        } else {
            searchExcludePositionStore.set(JSON.parse(JSON.stringify($positionStore)));
            positionStore.set(includeBoardStash ? JSON.parse(JSON.stringify(includeBoardStash)) : emptySearchBoardPosition());
            includeBoardStash = null;
            structureMode = 'include';
            searchStructureModeStore.set('include');
        }
    }
    $effect(() => {
        if ($activeTabStore === 'search' && $databaseLoadedStore) {
            loadHistory();
            loadSavedFilters();
        }
    });

    async function loadHistory() {
        try {
            const history = await LoadSearchHistory();
            searchHistoryStore.set(history || []);
        } catch (error) {
            logger.error('Error loading search history:', error);
        }
    }

    async function loadSavedFilters() {
        try {
            const lib = await LoadFilters();
            filterLibraryStore.set(lib || []);
        } catch (_error) {
            filterLibraryStore.set([]);
        }
    }

    function isInFilterLibrary(search) {
        return savedFilters.some((f) => f.command === search.command);
    }

    function handleSearch() {
        // The backend reads the include structure from positionStore, so make sure
        // the main board holds the include board (syncing the exclude board to its
        // store) before searching.
        if (structureMode === 'exclude') switchStructureMode('include');
        const excludeActive = boardHasCheckers($searchExcludePositionStore);

        const activeFilters = availableFilters.filter((f) => filterEnabled[f]);
        const transformedFilters = buildFilterTokens(activeFilters, {
            diceRollOption,
            pipCountOption,
            pipCountMin,
            pipCountMax,
            pipCountRangeMin,
            pipCountRangeMax,
            player1AbsolutePipCountOption,
            player1AbsolutePipCountMin,
            player1AbsolutePipCountMax,
            player1AbsolutePipCountRangeMin,
            player1AbsolutePipCountRangeMax,
            equityOption,
            equityMin,
            equityMax,
            equityRangeMin,
            equityRangeMax,
            moveErrorOption,
            moveErrorMin,
            moveErrorMax,
            moveErrorRangeMin,
            moveErrorRangeMax,
            winRateOption,
            winRateMin,
            winRateMax,
            winRateRangeMin,
            winRateRangeMax,
            gammonRateOption,
            gammonRateMin,
            gammonRateMax,
            gammonRateRangeMin,
            gammonRateRangeMax,
            backgammonRateOption,
            backgammonRateMin,
            backgammonRateMax,
            backgammonRateRangeMin,
            backgammonRateRangeMax,
            player2WinRateOption,
            player2WinRateMin,
            player2WinRateMax,
            player2WinRateRangeMin,
            player2WinRateRangeMax,
            player2GammonRateOption,
            player2GammonRateMin,
            player2GammonRateMax,
            player2GammonRateRangeMin,
            player2GammonRateRangeMax,
            player2BackgammonRateOption,
            player2BackgammonRateMin,
            player2BackgammonRateMax,
            player2BackgammonRateRangeMin,
            player2BackgammonRateRangeMax,
            player1CheckerOffOption,
            player1CheckerOffMin,
            player1CheckerOffMax,
            player1CheckerOffRangeMin,
            player1CheckerOffRangeMax,
            player2CheckerOffOption,
            player2CheckerOffMin,
            player2CheckerOffMax,
            player2CheckerOffRangeMin,
            player2CheckerOffRangeMax,
            player1BackCheckerOption,
            player1BackCheckerMin,
            player1BackCheckerMax,
            player1BackCheckerRangeMin,
            player1BackCheckerRangeMax,
            player2BackCheckerOption,
            player2BackCheckerMin,
            player2BackCheckerMax,
            player2BackCheckerRangeMin,
            player2BackCheckerRangeMax,
            player1CheckerInZoneOption,
            player1CheckerInZoneMin,
            player1CheckerInZoneMax,
            player1CheckerInZoneRangeMin,
            player1CheckerInZoneRangeMax,
            player2CheckerInZoneOption,
            player2CheckerInZoneMin,
            player2CheckerInZoneMax,
            player2CheckerInZoneRangeMin,
            player2CheckerInZoneRangeMax,
            searchText,
            movePattern,
            creationDateOption,
            creationDateMin,
            creationDateMax,
            creationDateRangeMin,
            creationDateRangeMax,
            player1OutfieldBlotOption,
            player1OutfieldBlotMin,
            player1OutfieldBlotMax,
            player1OutfieldBlotRangeMin,
            player1OutfieldBlotRangeMax,
            player2OutfieldBlotOption,
            player2OutfieldBlotMin,
            player2OutfieldBlotMax,
            player2OutfieldBlotRangeMin,
            player2OutfieldBlotRangeMax,
            player1JanBlotOption,
            player1JanBlotMin,
            player1JanBlotMax,
            player1JanBlotRangeMin,
            player1JanBlotRangeMax,
            player2JanBlotOption,
            player2JanBlotMin,
            player2JanBlotMax,
            player2JanBlotRangeMin,
            player2JanBlotRangeMax,
            matchIDsInput,
            tournamentIDsInput,
            playerName
        });

        // Cube sub-type: when the decision-type filter constrains to a cube
        // decision, narrow to double/no-double (`dd`) or take/pass (`dr`). The `d`
        // token already carries the cube decision_type read from the board.
        if (filterEnabled['Include Decision Type'] && decisionMode === 'cube' && cubeSubType !== 'all') {
            transformedFilters.push(cubeSubType === 'takepass' ? 'dr' : 'dd');
        }

        const {
            incCube,
            incScore,
            ncFilter,
            mirFilter,
            pcFilter,
            wrFilter,
            grFilter,
            bgFilter,
            p2wrFilter,
            p2grFilter,
            p2bgFilter,
            p1coFilter,
            p2coFilter,
            p1bcFilter,
            p2bcFilter,
            p1czFilter,
            p2czFilter,
            p1apcFilter,
            eqFilter,
            meFilter,
            p1obFilter,
            p2obFilter,
            p1jbFilter,
            p2jbFilter,
            matchIDs,
            tournamentIDs,
            dtFilter,
            drFilter,
            drMode,
            cdFilter
        } = parseFilterTokens(transformedFilters);

        const searchCommand = buildSearchCommand(excludeActive ? [...transformedFilters, 'x'] : transformedFilters);

        const excludePositionJSON = excludeActive ? JSON.stringify($searchExcludePositionStore) : '';
        const entry = { command: searchCommand, position: JSON.stringify($positionStore), excludePosition: excludePositionJSON, timestamp: Date.now() };
        searchHistoryStore.update((h) => [entry, ...h].slice(0, 100));
        SaveSearchHistory(searchCommand, JSON.stringify($positionStore), excludePositionJSON).catch((err) => logger.error('Error saving search history:', err));

        let restrictToPositionIDs = '';
        if (searchInCurrentResults) {
            const currentPositions = $positionsStore || [];
            restrictToPositionIDs = currentPositions
                .map((p) => p.id)
                .filter((id) => id != null)
                .join(',');
        }

        onLoadPositionsByFilters(
            activeFilters.length > 0 ? transformedFilters : [],
            incCube,
            incScore,
            pcFilter,
            wrFilter,
            grFilter,
            bgFilter,
            p2wrFilter,
            p2grFilter,
            p2bgFilter,
            p1coFilter,
            p2coFilter,
            p1bcFilter,
            p2bcFilter,
            p1czFilter,
            p2czFilter,
            searchText ? `t"${searchText}"` : '',
            p1apcFilter,
            eqFilter,
            dtFilter,
            drFilter,
            movePattern ? `m"${movePattern}"` : '',
            cdFilter,
            p1obFilter,
            p2obFilter,
            p1jbFilter,
            p2jbFilter,
            ncFilter,
            mirFilter,
            meFilter,
            searchCommand,
            matchIDs,
            tournamentIDs,
            restrictToPositionIDs,
            openInNewTab,
            drMode,
            '',
            playerName ? `pl"${playerName}"` : ''
        );

        saveSearchState();
    }

    function clearFilters() {
        availableFilters.forEach((f) => (filterEnabled[f] = false));
        filterEnabled = filterEnabled;
        pipCountOption = 'min';
        pipCountMin = -375;
        pipCountMax = 375;
        pipCountRangeMin = -375;
        pipCountRangeMax = 375;
        winRateOption = 'min';
        winRateMin = 0;
        winRateMax = 100;
        winRateRangeMin = 0;
        winRateRangeMax = 100;
        gammonRateOption = 'min';
        gammonRateMin = 0;
        gammonRateMax = 100;
        gammonRateRangeMin = 0;
        gammonRateRangeMax = 100;
        backgammonRateOption = 'min';
        backgammonRateMin = 0;
        backgammonRateMax = 100;
        backgammonRateRangeMin = 0;
        backgammonRateRangeMax = 100;
        player2WinRateOption = 'min';
        player2WinRateMin = 0;
        player2WinRateMax = 100;
        player2WinRateRangeMin = 0;
        player2WinRateRangeMax = 100;
        player2GammonRateOption = 'min';
        player2GammonRateMin = 0;
        player2GammonRateMax = 100;
        player2GammonRateRangeMin = 0;
        player2GammonRateRangeMax = 100;
        player2BackgammonRateOption = 'min';
        player2BackgammonRateMin = 0;
        player2BackgammonRateMax = 100;
        player2BackgammonRateRangeMin = 0;
        player2BackgammonRateRangeMax = 100;
        player1CheckerOffOption = 'min';
        player1CheckerOffMin = 0;
        player1CheckerOffMax = 15;
        player1CheckerOffRangeMin = 0;
        player1CheckerOffRangeMax = 15;
        player2CheckerOffOption = 'min';
        player2CheckerOffMin = 0;
        player2CheckerOffMax = 15;
        player2CheckerOffRangeMin = 0;
        player2CheckerOffRangeMax = 15;
        player1BackCheckerOption = 'min';
        player1BackCheckerMin = 0;
        player1BackCheckerMax = 15;
        player1BackCheckerRangeMin = 0;
        player1BackCheckerRangeMax = 15;
        player2BackCheckerOption = 'min';
        player2BackCheckerMin = 0;
        player2BackCheckerMax = 15;
        player2BackCheckerRangeMin = 0;
        player2BackCheckerRangeMax = 15;
        player1CheckerInZoneOption = 'min';
        player1CheckerInZoneMin = 0;
        player1CheckerInZoneMax = 15;
        player1CheckerInZoneRangeMin = 0;
        player1CheckerInZoneRangeMax = 15;
        player2CheckerInZoneOption = 'min';
        player2CheckerInZoneMin = 0;
        player2CheckerInZoneMax = 15;
        player2CheckerInZoneRangeMin = 0;
        player2CheckerInZoneRangeMax = 15;
        player1AbsolutePipCountOption = 'min';
        player1AbsolutePipCountMin = 0;
        player1AbsolutePipCountMax = 375;
        player1AbsolutePipCountRangeMin = 0;
        player1AbsolutePipCountRangeMax = 375;
        equityOption = 'min';
        equityMin = -1000;
        equityMax = 1000;
        equityRangeMin = -1000;
        equityRangeMax = 1000;
        moveErrorOption = 'min';
        moveErrorMin = 0;
        moveErrorMax = 1000;
        moveErrorRangeMin = 0;
        moveErrorRangeMax = 1000;
        searchText = '';
        movePattern = '';
        player1OutfieldBlotOption = 'min';
        player1OutfieldBlotMin = 0;
        player1OutfieldBlotMax = 15;
        player1OutfieldBlotRangeMin = 0;
        player1OutfieldBlotRangeMax = 15;
        player2OutfieldBlotOption = 'min';
        player2OutfieldBlotMin = 0;
        player2OutfieldBlotMax = 15;
        player2OutfieldBlotRangeMin = 0;
        player2OutfieldBlotRangeMax = 15;
        player1JanBlotOption = 'min';
        player1JanBlotMin = 0;
        player1JanBlotMax = 15;
        player1JanBlotRangeMin = 0;
        player1JanBlotRangeMax = 15;
        player2JanBlotOption = 'min';
        player2JanBlotMin = 0;
        player2JanBlotMax = 15;
        player2JanBlotRangeMin = 0;
        player2JanBlotRangeMax = 15;
        diceRollOption = 'both';
        decisionMode = 'checker';
        cubeSubType = 'all';
        searchOfferedCubeStore.set(false);
        matchIDsInput = '';
        tournamentIDsInput = '';
        playerName = '';
        creationDateOption = 'min';
        creationDateMin = '';
        creationDateMax = '';
        creationDateRangeMin = '';
        creationDateRangeMax = '';
        searchInCurrentResults = false;
        // Reset the (hidden) exclude structure and return to include editing mode.
        structureMode = 'include';
        searchStructureModeStore.set('include');
        includeBoardStash = null;
        searchExcludePositionStore.set(emptySearchBoardPosition());
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
            restoreExcludeStructure(search.excludePosition);
            currentPositionIndexStore.set(-1);
        }
    }

    function executeSearch(search) {
        if (search.position) {
            positionStore.set(JSON.parse(search.position));
        }
        restoreExcludeStructure(search.excludePosition);
        const command = search.command;
        if (command.startsWith('s ') || command === 's') {
            const f = parseSearchCommand(command);
            onLoadPositionsByFilters(
                f.cmdFilters,
                f.ic,
                f.is,
                f.pc,
                f.wr,
                f.gr,
                f.bg,
                f.p2wr,
                f.p2gr,
                f.p2bg,
                f.p1co,
                f.p2co,
                f.p1bc,
                f.p2bc,
                f.p1cz,
                f.p2cz,
                f.st,
                f.p1apc,
                f.eq,
                f.dt,
                f.dr,
                f.mpf,
                f.cd,
                f.p1ob,
                f.p2ob,
                f.p1jb,
                f.p2jb,
                f.nc,
                f.mp,
                f.me,
                command,
                f.matchIDs,
                f.tournamentIDs,
                '',
                false,
                f.drMode,
                '',
                f.plf
            );
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
            statusBarTextStore.set(tMsg('searchHistory.enterFilterName'));
            return;
        }
        if (onAddToFilterLibrary) {
            await onAddToFilterLibrary(filterName, selectedSearch.command, selectedSearch.position, selectedSearch.excludePosition);
            await loadSavedFilters();
            statusBarTextStore.set(tMsg('searchHistory.filterSaved'));
        }
        cancelSaveDialog();
    }

    async function deleteSearch(search, event) {
        event.stopPropagation();
        try {
            await DeleteSearchHistoryEntry(search.timestamp);
            await loadHistory();
            statusBarTextStore.set(tMsg('searchHistory.searchDeleted'));
        } catch (_error) {
            statusBarTextStore.set(tMsg('searchHistory.errorDeleting'));
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
            _savedFilterName = '';
            _savedFilterCommand = '';
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
        _savedFilterName = filter.name;
        _savedFilterCommand = filter.command;
        const editPosition = await LoadEditPosition(filter.name);
        if (editPosition) {
            positionStore.set(JSON.parse(editPosition));
        }
        const excludePosition = await LoadExcludePosition(filter.name);
        restoreExcludeStructure(excludePosition);
        currentPositionIndexStore.set(-1);
    }

    async function executeSavedFilter(filter) {
        const editPosition = await LoadEditPosition(filter.name);
        if (editPosition) {
            positionStore.set(JSON.parse(editPosition));
        }
        const excludePosition = await LoadExcludePosition(filter.name);
        executeSearch({ command: filter.command, position: editPosition, excludePosition });
    }

    async function deleteSavedFilter() {
        if (selectedSavedFilter) {
            await DeleteFilter(selectedSavedFilter.id);
            await loadSavedFilters();
            selectedSavedFilter = null;
            _savedFilterName = '';
            _savedFilterCommand = '';
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

    function saveSearchState() {
        // Sync the exclude board from the live board when it is the one being edited.
        const excludePosition = structureMode === 'exclude' ? JSON.parse(JSON.stringify($positionStore)) : JSON.parse(JSON.stringify($searchExcludePositionStore));
        searchParamsStore.set({
            position: savedSearchPosition,
            excludePosition,
            structureMode,
            filterEnabled: { ...filterEnabled },
            searchInCurrentResults,
            searchText,
            movePattern,
            matchIDsInput,
            tournamentIDsInput,
            playerName,
            pipCountOption,
            pipCountMin,
            pipCountMax,
            pipCountRangeMin,
            pipCountRangeMax,
            winRateOption,
            winRateMin,
            winRateMax,
            winRateRangeMin,
            winRateRangeMax,
            gammonRateOption,
            gammonRateMin,
            gammonRateMax,
            gammonRateRangeMin,
            gammonRateRangeMax,
            backgammonRateOption,
            backgammonRateMin,
            backgammonRateMax,
            backgammonRateRangeMin,
            backgammonRateRangeMax,
            player2WinRateOption,
            player2WinRateMin,
            player2WinRateMax,
            player2WinRateRangeMin,
            player2WinRateRangeMax,
            player2GammonRateOption,
            player2GammonRateMin,
            player2GammonRateMax,
            player2GammonRateRangeMin,
            player2GammonRateRangeMax,
            player2BackgammonRateOption,
            player2BackgammonRateMin,
            player2BackgammonRateMax,
            player2BackgammonRateRangeMin,
            player2BackgammonRateRangeMax,
            player1CheckerOffOption,
            player1CheckerOffMin,
            player1CheckerOffMax,
            player1CheckerOffRangeMin,
            player1CheckerOffRangeMax,
            player2CheckerOffOption,
            player2CheckerOffMin,
            player2CheckerOffMax,
            player2CheckerOffRangeMin,
            player2CheckerOffRangeMax,
            player1BackCheckerOption,
            player1BackCheckerMin,
            player1BackCheckerMax,
            player1BackCheckerRangeMin,
            player1BackCheckerRangeMax,
            player2BackCheckerOption,
            player2BackCheckerMin,
            player2BackCheckerMax,
            player2BackCheckerRangeMin,
            player2BackCheckerRangeMax,
            player1CheckerInZoneOption,
            player1CheckerInZoneMin,
            player1CheckerInZoneMax,
            player1CheckerInZoneRangeMin,
            player1CheckerInZoneRangeMax,
            player2CheckerInZoneOption,
            player2CheckerInZoneMin,
            player2CheckerInZoneMax,
            player2CheckerInZoneRangeMin,
            player2CheckerInZoneRangeMax,
            player1AbsolutePipCountOption,
            player1AbsolutePipCountMin,
            player1AbsolutePipCountMax,
            player1AbsolutePipCountRangeMin,
            player1AbsolutePipCountRangeMax,
            equityOption,
            equityMin,
            equityMax,
            equityRangeMin,
            equityRangeMax,
            moveErrorOption,
            moveErrorMin,
            moveErrorMax,
            moveErrorRangeMin,
            moveErrorRangeMax,
            player1OutfieldBlotOption,
            player1OutfieldBlotMin,
            player1OutfieldBlotMax,
            player1OutfieldBlotRangeMin,
            player1OutfieldBlotRangeMax,
            player2OutfieldBlotOption,
            player2OutfieldBlotMin,
            player2OutfieldBlotMax,
            player2OutfieldBlotRangeMin,
            player2OutfieldBlotRangeMax,
            player1JanBlotOption,
            player1JanBlotMin,
            player1JanBlotMax,
            player1JanBlotRangeMin,
            player1JanBlotRangeMax,
            player2JanBlotOption,
            player2JanBlotMin,
            player2JanBlotMax,
            player2JanBlotRangeMin,
            player2JanBlotRangeMax,
            diceRollOption,
            cubeSubType,
            creationDateOption,
            creationDateMin,
            creationDateMax,
            creationDateRangeMin,
            creationDateRangeMax
        });
    }

    // restoreSearchBoard restores the include board + exclude structure from the
    // saved search params. It is invoked from onMount after tick() so it runs after
    // App.svelte's enterEditMode() (which clears the board on entering the search
    // tab) — otherwise enterEditMode would clobber the restored board.
    function restoreSearchBoard() {
        const saved = $searchParamsStore;
        structureMode = 'include';
        searchStructureModeStore.set('include');
        includeBoardStash = null;
        if (!saved) {
            searchExcludePositionStore.set(emptySearchBoardPosition());
            return;
        }
        if (saved.position) {
            positionStore.set(JSON.parse(JSON.stringify(saved.position)));
        }
        searchExcludePositionStore.set(saved.excludePosition ? JSON.parse(JSON.stringify(saved.excludePosition)) : emptySearchBoardPosition());
    }

    function restoreSearchState() {
        const saved = $searchParamsStore;
        if (!saved) return;
        filterEnabled = { ...saved.filterEnabled };
        searchInCurrentResults = saved.searchInCurrentResults;
        searchText = saved.searchText;
        movePattern = saved.movePattern;
        matchIDsInput = saved.matchIDsInput;
        tournamentIDsInput = saved.tournamentIDsInput;
        playerName = saved.playerName ?? '';
        pipCountOption = saved.pipCountOption;
        pipCountMin = saved.pipCountMin;
        pipCountMax = saved.pipCountMax;
        pipCountRangeMin = saved.pipCountRangeMin;
        pipCountRangeMax = saved.pipCountRangeMax;
        winRateOption = saved.winRateOption;
        winRateMin = saved.winRateMin;
        winRateMax = saved.winRateMax;
        winRateRangeMin = saved.winRateRangeMin;
        winRateRangeMax = saved.winRateRangeMax;
        gammonRateOption = saved.gammonRateOption;
        gammonRateMin = saved.gammonRateMin;
        gammonRateMax = saved.gammonRateMax;
        gammonRateRangeMin = saved.gammonRateRangeMin;
        gammonRateRangeMax = saved.gammonRateRangeMax;
        backgammonRateOption = saved.backgammonRateOption;
        backgammonRateMin = saved.backgammonRateMin;
        backgammonRateMax = saved.backgammonRateMax;
        backgammonRateRangeMin = saved.backgammonRateRangeMin;
        backgammonRateRangeMax = saved.backgammonRateRangeMax;
        player2WinRateOption = saved.player2WinRateOption;
        player2WinRateMin = saved.player2WinRateMin;
        player2WinRateMax = saved.player2WinRateMax;
        player2WinRateRangeMin = saved.player2WinRateRangeMin;
        player2WinRateRangeMax = saved.player2WinRateRangeMax;
        player2GammonRateOption = saved.player2GammonRateOption;
        player2GammonRateMin = saved.player2GammonRateMin;
        player2GammonRateMax = saved.player2GammonRateMax;
        player2GammonRateRangeMin = saved.player2GammonRateRangeMin;
        player2GammonRateRangeMax = saved.player2GammonRateRangeMax;
        player2BackgammonRateOption = saved.player2BackgammonRateOption;
        player2BackgammonRateMin = saved.player2BackgammonRateMin;
        player2BackgammonRateMax = saved.player2BackgammonRateMax;
        player2BackgammonRateRangeMin = saved.player2BackgammonRateRangeMin;
        player2BackgammonRateRangeMax = saved.player2BackgammonRateRangeMax;
        player1CheckerOffOption = saved.player1CheckerOffOption;
        player1CheckerOffMin = saved.player1CheckerOffMin;
        player1CheckerOffMax = saved.player1CheckerOffMax;
        player1CheckerOffRangeMin = saved.player1CheckerOffRangeMin;
        player1CheckerOffRangeMax = saved.player1CheckerOffRangeMax;
        player2CheckerOffOption = saved.player2CheckerOffOption;
        player2CheckerOffMin = saved.player2CheckerOffMin;
        player2CheckerOffMax = saved.player2CheckerOffMax;
        player2CheckerOffRangeMin = saved.player2CheckerOffRangeMin;
        player2CheckerOffRangeMax = saved.player2CheckerOffRangeMax;
        player1BackCheckerOption = saved.player1BackCheckerOption;
        player1BackCheckerMin = saved.player1BackCheckerMin;
        player1BackCheckerMax = saved.player1BackCheckerMax;
        player1BackCheckerRangeMin = saved.player1BackCheckerRangeMin;
        player1BackCheckerRangeMax = saved.player1BackCheckerRangeMax;
        player2BackCheckerOption = saved.player2BackCheckerOption;
        player2BackCheckerMin = saved.player2BackCheckerMin;
        player2BackCheckerMax = saved.player2BackCheckerMax;
        player2BackCheckerRangeMin = saved.player2BackCheckerRangeMin;
        player2BackCheckerRangeMax = saved.player2BackCheckerRangeMax;
        player1CheckerInZoneOption = saved.player1CheckerInZoneOption;
        player1CheckerInZoneMin = saved.player1CheckerInZoneMin;
        player1CheckerInZoneMax = saved.player1CheckerInZoneMax;
        player1CheckerInZoneRangeMin = saved.player1CheckerInZoneRangeMin;
        player1CheckerInZoneRangeMax = saved.player1CheckerInZoneRangeMax;
        player2CheckerInZoneOption = saved.player2CheckerInZoneOption;
        player2CheckerInZoneMin = saved.player2CheckerInZoneMin;
        player2CheckerInZoneMax = saved.player2CheckerInZoneMax;
        player2CheckerInZoneRangeMin = saved.player2CheckerInZoneRangeMin;
        player2CheckerInZoneRangeMax = saved.player2CheckerInZoneRangeMax;
        player1AbsolutePipCountOption = saved.player1AbsolutePipCountOption;
        player1AbsolutePipCountMin = saved.player1AbsolutePipCountMin;
        player1AbsolutePipCountMax = saved.player1AbsolutePipCountMax;
        player1AbsolutePipCountRangeMin = saved.player1AbsolutePipCountRangeMin;
        player1AbsolutePipCountRangeMax = saved.player1AbsolutePipCountRangeMax;
        equityOption = saved.equityOption;
        equityMin = saved.equityMin;
        equityMax = saved.equityMax;
        equityRangeMin = saved.equityRangeMin;
        equityRangeMax = saved.equityRangeMax;
        moveErrorOption = saved.moveErrorOption;
        moveErrorMin = saved.moveErrorMin;
        moveErrorMax = saved.moveErrorMax;
        moveErrorRangeMin = saved.moveErrorRangeMin;
        moveErrorRangeMax = saved.moveErrorRangeMax;
        player1OutfieldBlotOption = saved.player1OutfieldBlotOption;
        player1OutfieldBlotMin = saved.player1OutfieldBlotMin;
        player1OutfieldBlotMax = saved.player1OutfieldBlotMax;
        player1OutfieldBlotRangeMin = saved.player1OutfieldBlotRangeMin;
        player1OutfieldBlotRangeMax = saved.player1OutfieldBlotRangeMax;
        player2OutfieldBlotOption = saved.player2OutfieldBlotOption;
        player2OutfieldBlotMin = saved.player2OutfieldBlotMin;
        player2OutfieldBlotMax = saved.player2OutfieldBlotMax;
        player2OutfieldBlotRangeMin = saved.player2OutfieldBlotRangeMin;
        player2OutfieldBlotRangeMax = saved.player2OutfieldBlotRangeMax;
        player1JanBlotOption = saved.player1JanBlotOption;
        player1JanBlotMin = saved.player1JanBlotMin;
        player1JanBlotMax = saved.player1JanBlotMax;
        player1JanBlotRangeMin = saved.player1JanBlotRangeMin;
        player1JanBlotRangeMax = saved.player1JanBlotRangeMax;
        player2JanBlotOption = saved.player2JanBlotOption;
        player2JanBlotMin = saved.player2JanBlotMin;
        player2JanBlotMax = saved.player2JanBlotMax;
        player2JanBlotRangeMin = saved.player2JanBlotRangeMin;
        player2JanBlotRangeMax = saved.player2JanBlotRangeMax;
        if (saved.diceRollOption) diceRollOption = saved.diceRollOption;
        if (saved.cubeSubType) cubeSubType = saved.cubeSubType;
        creationDateOption = saved.creationDateOption;
        creationDateMin = saved.creationDateMin;
        creationDateMax = saved.creationDateMax;
        creationDateRangeMin = saved.creationDateRangeMin;
        creationDateRangeMax = saved.creationDateRangeMax;
    }

    onMount(async () => {
        document.addEventListener('keydown', handleKeyDown);
        // Restore the board after the initial flush so App.svelte's enterEditMode()
        // (which clears the board on tab entry) has already run.
        await tick();
        restoreSearchBoard();
    });

    onDestroy(() => {
        saveSearchState();
        // Don't leak the offered-cube flag into normal position editing.
        searchOfferedCubeStore.set(false);
        document.removeEventListener('keydown', handleKeyDown);
    });
</script>

<div class="search-panel">
    <!-- Left sub-tab sidebar -->
    <div class="sub-tab-sidebar">
        <button class="sub-tab-btn" class:active={activeSubTab === 'search'} onclick={() => (activeSubTab = 'search')}>{$t('common.search')}</button>
        <button class="sub-tab-btn" class:active={activeSubTab === 'history'} onclick={() => (activeSubTab = 'history')}>{$t('search.historyTab')}</button>
        <button class="sub-tab-btn" class:active={activeSubTab === 'saved'} onclick={() => (activeSubTab = 'saved')}>{$t('search.savedTab')}</button>
    </div>

    <!-- Content area -->
    <div class="sub-tab-content">
        {#if activeSubTab === 'search'}
            <!-- Filter Builder with checkboxes -->
            <div class="filter-section">
                <div class="structure-toggle" class:exclude-active={structureMode === 'exclude'}>
                    <button class="structure-btn" class:active={structureMode === 'include'} onclick={() => switchStructureMode('include')} title={$t('search.atLeastTooltip')}
                        >{$t('search.atLeast')}</button
                    >
                    <button class="structure-btn exclude" class:active={structureMode === 'exclude'} onclick={() => switchStructureMode('exclude')} title={$t('search.exceptTooltip')}
                        >{$t('search.except')}</button
                    >
                    {#if boardHasCheckers($searchExcludePositionStore) || structureMode === 'exclude'}
                        <span class="structure-hint">{structureMode === 'exclude' ? $t('search.editingExcluded') : $t('search.exclusionSet')}</span>
                    {/if}
                </div>
                <div class="action-bar top-action-bar">
                    <label class="search-in-results"><input type="checkbox" bind:checked={searchInCurrentResults} /> {$t('search.inResults')}</label>
                    <label class="search-in-results"><input type="checkbox" bind:checked={openInNewTab} /> {$t('search.newTab')}</label>
                    <span class="active-count">{$t('search.activeCount', { n: activeFilterCount })}</span>
                    <button class="btn-search" onclick={handleSearch}>{$t('common.search')}</button>
                    <button class="btn-clear" onclick={clearFilters}>{$t('common.clear')}</button>
                </div>
                <div class="filter-groups">
                    {#each filterGroups as group (group.name)}
                        <div class="filter-group">
                            <div class="group-header">{groupLabel(group.name)}</div>
                            {#each group.filters as filter (filter)}
                                <div class="filter-item" class:active={filterEnabled[filter]}>
                                    <label class="filter-checkbox">
                                        <input
                                            type="checkbox"
                                            bind:checked={filterEnabled[filter]}
                                            disabled={filter === 'Include Dice Roll' && filterEnabled['Include Decision Type'] && decisionMode === 'cube'}
                                        />
                                        <span
                                            class="filter-label"
                                            class:label-disabled={filter === 'Include Dice Roll' && filterEnabled['Include Decision Type'] && decisionMode === 'cube'}
                                            title={filterTokenHint(filter)}>{filterLabel(filter)}</span
                                        >
                                    </label>
                                    {#if filterEnabled[filter]}
                                        <div class="filter-params">
                                            {#if filter === 'Include Decision Type'}
                                                <div class="decision-mode-controls">
                                                    <div class="decision-segment">
                                                        <button type="button" class="decision-btn" class:active={decisionMode === 'checker'} onclick={() => selectDecisionMode('checker')}
                                                            >{$t('search.decision.checker')}</button
                                                        >
                                                        <button type="button" class="decision-btn" class:active={decisionMode === 'cube'} onclick={() => selectDecisionMode('cube')}
                                                            >{$t('search.decision.cube')}</button
                                                        >
                                                    </div>
                                                    {#if decisionMode === 'cube'}
                                                        <div class="minmax-controls">
                                                            <label
                                                                ><input type="radio" name="cubeSubType" value="all" checked={cubeSubType === 'all'} onchange={() => selectCubeSubType('all')} />
                                                                {$t('search.decision.cubeAll')}</label
                                                            >
                                                            <label
                                                                ><input
                                                                    type="radio"
                                                                    name="cubeSubType"
                                                                    value="double"
                                                                    checked={cubeSubType === 'double'}
                                                                    onchange={() => selectCubeSubType('double')}
                                                                />
                                                                {$t('search.decision.cubeDouble')}</label
                                                            >
                                                            <label
                                                                ><input
                                                                    type="radio"
                                                                    name="cubeSubType"
                                                                    value="takepass"
                                                                    checked={cubeSubType === 'takepass'}
                                                                    onchange={() => selectCubeSubType('takepass')}
                                                                />
                                                                {$t('search.decision.cubeTakePass')}</label
                                                            >
                                                        </div>
                                                    {/if}
                                                </div>
                                            {:else if filter === 'Include Dice Roll'}
                                                <div class="minmax-controls">
                                                    <label><input type="radio" bind:group={diceRollOption} value="both" /> {$t('search.bothDice')}</label>
                                                    <label><input type="radio" bind:group={diceRollOption} value="first" /> {$t('search.firstDieOnly')}</label>
                                                </div>
                                            {:else if filter === 'Pipcount Difference'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={pipCountOption} value="min" />
                                                        {$t('common.min')} <input type="number" bind:value={pipCountMin} class="num-input" disabled={pipCountOption !== 'min'} /></label
                                                    >
                                                    <label
                                                        ><input type="radio" bind:group={pipCountOption} value="max" />
                                                        {$t('common.max')} <input type="number" bind:value={pipCountMax} class="num-input" disabled={pipCountOption !== 'max'} /></label
                                                    >
                                                    <label
                                                        ><input type="radio" bind:group={pipCountOption} value="range" />
                                                        {$t('common.range')} <input type="number" bind:value={pipCountRangeMin} class="num-input" disabled={pipCountOption !== 'range'} />
                                                        <input type="number" bind:value={pipCountRangeMax} class="num-input" disabled={pipCountOption !== 'range'} /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Player Absolute Pipcount'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player1AbsolutePipCountOption} value="min" />
                                                        {$t('common.min')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1AbsolutePipCountMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="375"
                                                            disabled={player1AbsolutePipCountOption !== 'min'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1AbsolutePipCountOption} value="max" />
                                                        {$t('common.max')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1AbsolutePipCountMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="375"
                                                            disabled={player1AbsolutePipCountOption !== 'max'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1AbsolutePipCountOption} value="range" />
                                                        {$t('common.range')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1AbsolutePipCountRangeMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="375"
                                                            disabled={player1AbsolutePipCountOption !== 'range'}
                                                        />
                                                        <input
                                                            type="number"
                                                            bind:value={player1AbsolutePipCountRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="375"
                                                            disabled={player1AbsolutePipCountOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Equity (millipoints)'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={equityOption} value="min" />
                                                        {$t('common.min')} <input type="number" bind:value={equityMin} class="num-input" disabled={equityOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={equityOption} value="max" />
                                                        {$t('common.max')} <input type="number" bind:value={equityMax} class="num-input" disabled={equityOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={equityOption} value="range" />
                                                        {$t('common.range')} <input type="number" bind:value={equityRangeMin} class="num-input" disabled={equityOption !== 'range'} />
                                                        <input type="number" bind:value={equityRangeMax} class="num-input" disabled={equityOption !== 'range'} /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Move Error (millipoints, Player 1)'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={moveErrorOption} value="min" />
                                                        {$t('common.min')} <input type="number" bind:value={moveErrorMin} class="num-input" min="0" disabled={moveErrorOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={moveErrorOption} value="max" />
                                                        {$t('common.max')} <input type="number" bind:value={moveErrorMax} class="num-input" min="0" disabled={moveErrorOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={moveErrorOption} value="range" />
                                                        {$t('common.range')} <input type="number" bind:value={moveErrorRangeMin} class="num-input" min="0" disabled={moveErrorOption !== 'range'} />
                                                        <input type="number" bind:value={moveErrorRangeMax} class="num-input" min="0" disabled={moveErrorOption !== 'range'} /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Win Rate'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={winRateOption} value="min" />
                                                        {$t('common.min')} <input type="number" bind:value={winRateMin} class="num-input" min="0" max="100" disabled={winRateOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={winRateOption} value="max" />
                                                        {$t('common.max')} <input type="number" bind:value={winRateMax} class="num-input" min="0" max="100" disabled={winRateOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={winRateOption} value="range" />
                                                        {$t('common.range')}
                                                        <input type="number" bind:value={winRateRangeMin} class="num-input" min="0" max="100" disabled={winRateOption !== 'range'} />
                                                        <input type="number" bind:value={winRateRangeMax} class="num-input" min="0" max="100" disabled={winRateOption !== 'range'} /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Gammon Rate'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={gammonRateOption} value="min" />
                                                        {$t('common.min')}
                                                        <input type="number" bind:value={gammonRateMin} class="num-input" min="0" max="100" disabled={gammonRateOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={gammonRateOption} value="max" />
                                                        {$t('common.max')}
                                                        <input type="number" bind:value={gammonRateMax} class="num-input" min="0" max="100" disabled={gammonRateOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={gammonRateOption} value="range" />
                                                        {$t('common.range')}
                                                        <input type="number" bind:value={gammonRateRangeMin} class="num-input" min="0" max="100" disabled={gammonRateOption !== 'range'} />
                                                        <input type="number" bind:value={gammonRateRangeMax} class="num-input" min="0" max="100" disabled={gammonRateOption !== 'range'} /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Backgammon Rate'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={backgammonRateOption} value="min" />
                                                        {$t('common.min')}
                                                        <input type="number" bind:value={backgammonRateMin} class="num-input" min="0" max="100" disabled={backgammonRateOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={backgammonRateOption} value="max" />
                                                        {$t('common.max')}
                                                        <input type="number" bind:value={backgammonRateMax} class="num-input" min="0" max="100" disabled={backgammonRateOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={backgammonRateOption} value="range" />
                                                        {$t('common.range')}
                                                        <input type="number" bind:value={backgammonRateRangeMin} class="num-input" min="0" max="100" disabled={backgammonRateOption !== 'range'} />
                                                        <input
                                                            type="number"
                                                            bind:value={backgammonRateRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="100"
                                                            disabled={backgammonRateOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Opponent Win Rate'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player2WinRateOption} value="min" />
                                                        {$t('common.min')}
                                                        <input type="number" bind:value={player2WinRateMin} class="num-input" min="0" max="100" disabled={player2WinRateOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2WinRateOption} value="max" />
                                                        {$t('common.max')}
                                                        <input type="number" bind:value={player2WinRateMax} class="num-input" min="0" max="100" disabled={player2WinRateOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2WinRateOption} value="range" />
                                                        {$t('common.range')}
                                                        <input type="number" bind:value={player2WinRateRangeMin} class="num-input" min="0" max="100" disabled={player2WinRateOption !== 'range'} />
                                                        <input
                                                            type="number"
                                                            bind:value={player2WinRateRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="100"
                                                            disabled={player2WinRateOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Opponent Gammon Rate'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player2GammonRateOption} value="min" />
                                                        {$t('common.min')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2GammonRateMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="100"
                                                            disabled={player2GammonRateOption !== 'min'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2GammonRateOption} value="max" />
                                                        {$t('common.max')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2GammonRateMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="100"
                                                            disabled={player2GammonRateOption !== 'max'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2GammonRateOption} value="range" />
                                                        {$t('common.range')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2GammonRateRangeMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="100"
                                                            disabled={player2GammonRateOption !== 'range'}
                                                        />
                                                        <input
                                                            type="number"
                                                            bind:value={player2GammonRateRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="100"
                                                            disabled={player2GammonRateOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Opponent Backgammon Rate'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player2BackgammonRateOption} value="min" />
                                                        {$t('common.min')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2BackgammonRateMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="100"
                                                            disabled={player2BackgammonRateOption !== 'min'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2BackgammonRateOption} value="max" />
                                                        {$t('common.max')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2BackgammonRateMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="100"
                                                            disabled={player2BackgammonRateOption !== 'max'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2BackgammonRateOption} value="range" />
                                                        {$t('common.range')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2BackgammonRateRangeMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="100"
                                                            disabled={player2BackgammonRateOption !== 'range'}
                                                        />
                                                        <input
                                                            type="number"
                                                            bind:value={player2BackgammonRateRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="100"
                                                            disabled={player2BackgammonRateOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Player Checker-Off'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player1CheckerOffOption} value="min" />
                                                        {$t('common.min')}
                                                        <input type="number" bind:value={player1CheckerOffMin} class="num-input" min="0" max="15" disabled={player1CheckerOffOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1CheckerOffOption} value="max" />
                                                        {$t('common.max')}
                                                        <input type="number" bind:value={player1CheckerOffMax} class="num-input" min="0" max="15" disabled={player1CheckerOffOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1CheckerOffOption} value="range" />
                                                        {$t('common.range')}
                                                        <input type="number" bind:value={player1CheckerOffRangeMin} class="num-input" min="0" max="15" disabled={player1CheckerOffOption !== 'range'} />
                                                        <input
                                                            type="number"
                                                            bind:value={player1CheckerOffRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1CheckerOffOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Opponent Checker-Off'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player2CheckerOffOption} value="min" />
                                                        {$t('common.min')}
                                                        <input type="number" bind:value={player2CheckerOffMin} class="num-input" min="0" max="15" disabled={player2CheckerOffOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2CheckerOffOption} value="max" />
                                                        {$t('common.max')}
                                                        <input type="number" bind:value={player2CheckerOffMax} class="num-input" min="0" max="15" disabled={player2CheckerOffOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2CheckerOffOption} value="range" />
                                                        {$t('common.range')}
                                                        <input type="number" bind:value={player2CheckerOffRangeMin} class="num-input" min="0" max="15" disabled={player2CheckerOffOption !== 'range'} />
                                                        <input
                                                            type="number"
                                                            bind:value={player2CheckerOffRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2CheckerOffOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Player Back Checker'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player1BackCheckerOption} value="min" />
                                                        {$t('common.min')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1BackCheckerMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1BackCheckerOption !== 'min'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1BackCheckerOption} value="max" />
                                                        {$t('common.max')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1BackCheckerMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1BackCheckerOption !== 'max'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1BackCheckerOption} value="range" />
                                                        {$t('common.range')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1BackCheckerRangeMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1BackCheckerOption !== 'range'}
                                                        />
                                                        <input
                                                            type="number"
                                                            bind:value={player1BackCheckerRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1BackCheckerOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Opponent Back Checker'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player2BackCheckerOption} value="min" />
                                                        {$t('common.min')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2BackCheckerMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2BackCheckerOption !== 'min'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2BackCheckerOption} value="max" />
                                                        {$t('common.max')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2BackCheckerMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2BackCheckerOption !== 'max'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2BackCheckerOption} value="range" />
                                                        {$t('common.range')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2BackCheckerRangeMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2BackCheckerOption !== 'range'}
                                                        />
                                                        <input
                                                            type="number"
                                                            bind:value={player2BackCheckerRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2BackCheckerOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Player Checker in the Zone'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player1CheckerInZoneOption} value="min" />
                                                        {$t('common.min')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1CheckerInZoneMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1CheckerInZoneOption !== 'min'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1CheckerInZoneOption} value="max" />
                                                        {$t('common.max')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1CheckerInZoneMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1CheckerInZoneOption !== 'max'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1CheckerInZoneOption} value="range" />
                                                        {$t('common.range')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1CheckerInZoneRangeMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1CheckerInZoneOption !== 'range'}
                                                        />
                                                        <input
                                                            type="number"
                                                            bind:value={player1CheckerInZoneRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1CheckerInZoneOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Opponent Checker in the Zone'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player2CheckerInZoneOption} value="min" />
                                                        {$t('common.min')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2CheckerInZoneMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2CheckerInZoneOption !== 'min'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2CheckerInZoneOption} value="max" />
                                                        {$t('common.max')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2CheckerInZoneMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2CheckerInZoneOption !== 'max'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2CheckerInZoneOption} value="range" />
                                                        {$t('common.range')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2CheckerInZoneRangeMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2CheckerInZoneOption !== 'range'}
                                                        />
                                                        <input
                                                            type="number"
                                                            bind:value={player2CheckerInZoneRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2CheckerInZoneOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Player Outfield Blot'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player1OutfieldBlotOption} value="min" />
                                                        {$t('common.min')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1OutfieldBlotMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1OutfieldBlotOption !== 'min'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1OutfieldBlotOption} value="max" />
                                                        {$t('common.max')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1OutfieldBlotMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1OutfieldBlotOption !== 'max'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1OutfieldBlotOption} value="range" />
                                                        {$t('common.range')}
                                                        <input
                                                            type="number"
                                                            bind:value={player1OutfieldBlotRangeMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1OutfieldBlotOption !== 'range'}
                                                        />
                                                        <input
                                                            type="number"
                                                            bind:value={player1OutfieldBlotRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1OutfieldBlotOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Opponent Outfield Blot'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player2OutfieldBlotOption} value="min" />
                                                        {$t('common.min')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2OutfieldBlotMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2OutfieldBlotOption !== 'min'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2OutfieldBlotOption} value="max" />
                                                        {$t('common.max')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2OutfieldBlotMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2OutfieldBlotOption !== 'max'}
                                                        /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2OutfieldBlotOption} value="range" />
                                                        {$t('common.range')}
                                                        <input
                                                            type="number"
                                                            bind:value={player2OutfieldBlotRangeMin}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2OutfieldBlotOption !== 'range'}
                                                        />
                                                        <input
                                                            type="number"
                                                            bind:value={player2OutfieldBlotRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2OutfieldBlotOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Player Jan Blot'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player1JanBlotOption} value="min" />
                                                        {$t('common.min')}
                                                        <input type="number" bind:value={player1JanBlotMin} class="num-input" min="0" max="15" disabled={player1JanBlotOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1JanBlotOption} value="max" />
                                                        {$t('common.max')}
                                                        <input type="number" bind:value={player1JanBlotMax} class="num-input" min="0" max="15" disabled={player1JanBlotOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player1JanBlotOption} value="range" />
                                                        {$t('common.range')}
                                                        <input type="number" bind:value={player1JanBlotRangeMin} class="num-input" min="0" max="15" disabled={player1JanBlotOption !== 'range'} />
                                                        <input
                                                            type="number"
                                                            bind:value={player1JanBlotRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player1JanBlotOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Opponent Jan Blot'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={player2JanBlotOption} value="min" />
                                                        {$t('common.min')}
                                                        <input type="number" bind:value={player2JanBlotMin} class="num-input" min="0" max="15" disabled={player2JanBlotOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2JanBlotOption} value="max" />
                                                        {$t('common.max')}
                                                        <input type="number" bind:value={player2JanBlotMax} class="num-input" min="0" max="15" disabled={player2JanBlotOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={player2JanBlotOption} value="range" />
                                                        {$t('common.range')}
                                                        <input type="number" bind:value={player2JanBlotRangeMin} class="num-input" min="0" max="15" disabled={player2JanBlotOption !== 'range'} />
                                                        <input
                                                            type="number"
                                                            bind:value={player2JanBlotRangeMax}
                                                            class="num-input"
                                                            min="0"
                                                            max="15"
                                                            disabled={player2JanBlotOption !== 'range'}
                                                        /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Search Text'}
                                                <div class="text-control"><span class="hint">{$t('search.searchTextHint')}</span><input type="text" bind:value={searchText} class="text-input" /></div>
                                            {:else if filter === 'Best Move or Cube Decision'}
                                                <div class="text-control">
                                                    <span class="hint">{$t('search.movePatternHint')}</span><input type="text" bind:value={movePattern} class="text-input" />
                                                </div>
                                            {:else if filter === 'Creation Date'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" bind:group={creationDateOption} value="min" />
                                                        {$t('common.min')} <input type="date" bind:value={creationDateMin} class="date-input" disabled={creationDateOption !== 'min'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={creationDateOption} value="max" />
                                                        {$t('common.max')} <input type="date" bind:value={creationDateMax} class="date-input" disabled={creationDateOption !== 'max'} /></label
                                                    ><label
                                                        ><input type="radio" bind:group={creationDateOption} value="range" />
                                                        {$t('common.range')} <input type="date" bind:value={creationDateRangeMin} class="date-input" disabled={creationDateOption !== 'range'} />
                                                        <input type="date" bind:value={creationDateRangeMax} class="date-input" disabled={creationDateOption !== 'range'} /></label
                                                    >
                                                </div>
                                            {:else if filter === 'Match IDs'}
                                                <div class="text-control">
                                                    <span class="hint">{$t('search.matchIdsHint')}</span><input
                                                        type="text"
                                                        bind:value={matchIDsInput}
                                                        class="text-input"
                                                        placeholder={$t('search.idOrRange')}
                                                    />
                                                </div>
                                            {:else if filter === 'Tournament IDs'}
                                                <div class="text-control">
                                                    <span class="hint">{$t('search.tournamentIdsHint')}</span><input
                                                        type="text"
                                                        bind:value={tournamentIDsInput}
                                                        class="text-input"
                                                        placeholder={$t('search.idOrRange')}
                                                    />
                                                </div>
                                            {:else if filter === 'Player'}
                                                <div class="text-control">
                                                    <span class="hint">{$t('search.playerHint')}</span><input type="text" bind:value={playerName} class="text-input" />
                                                </div>
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
                    <p class="empty-message">{$t('search.noHistory')}</p>
                {:else}
                    <div class="history-table-container">
                        <table class="history-table">
                            <thead><tr><th>{$t('search.date')}</th><th>{$t('search.command')}</th><th>{$t('search.actions')}</th></tr></thead>
                            <tbody>
                                {#each searchHistory as search (search.timestamp)}
                                    <tr class:selected={selectedSearch === search} onclick={() => selectSearch(search)} ondblclick={() => handleDoubleClick(search)}>
                                        <td class="date-cell">{formatTimestamp(search.timestamp)}</td>
                                        <td class="command-cell">{search.command}</td>
                                        <td class="actions-cell">
                                            <button
                                                class="action-btn"
                                                class:in-library={isInFilterLibrary(search)}
                                                onclick={(e) => {
                                                    e.stopPropagation();
                                                    (() => showAddToLibraryDialog(search))();
                                                }}
                                                title={$t('search.saveToBookmarks')}
                                            >
                                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14"
                                                    ><path
                                                        stroke-linecap="round"
                                                        stroke-linejoin="round"
                                                        d="M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0 1 11.186 0Z"
                                                    /></svg
                                                >
                                            </button>
                                            <button
                                                class="action-btn delete-btn"
                                                onclick={(e) => {
                                                    e.stopPropagation();
                                                    ((e) => deleteSearch(search, e))(e);
                                                }}
                                                title={$t('common.delete')}
                                            >
                                                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14"
                                                    ><path
                                                        stroke-linecap="round"
                                                        stroke-linejoin="round"
                                                        d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0"
                                                    /></svg
                                                >
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
                    <p class="empty-message">{$t('search.noSaved')}</p>
                {:else}
                    <div class="saved-list">
                        {#each savedFilters as sf (sf.id)}
                            <div
                                class="saved-item"
                                class:selected={selectedSavedFilter && selectedSavedFilter.id === sf.id}
                                onclick={() => selectSavedFilter(sf)}
                                ondblclick={() => executeSavedFilter(sf)}
                            >
                                <span class="saved-name">{sf.name}</span>
                                <span class="saved-cmd">{sf.command}</span>
                                <button
                                    class="action-btn delete-btn"
                                    onclick={(e) => {
                                        e.stopPropagation();
                                        selectedSavedFilter = sf;
                                        deleteSavedFilter();
                                    }}
                                    title={$t('search.remove')}
                                >
                                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14"
                                        ><path
                                            stroke-linecap="round"
                                            stroke-linejoin="round"
                                            d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0"
                                        /></svg
                                    >
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
    <div
        class="save-dialog-overlay"
        onclick={(e) => {
            if (e.target === e.currentTarget) cancelSaveDialog(e);
        }}
    >
        <div class="save-dialog">
            <h3>{$t('search.saveSearch')}</h3>
            <p class="command-preview">{selectedSearch?.command || ''}</p>
            <div class="dialog-form">
                <label for="filterNameInput">{$t('search.name')}</label>
                <input type="text" id="filterNameInput" bind:value={filterName} placeholder={$t('search.enterName')} onkeydown={(e) => e.key === 'Enter' && saveToFilterLibrary()} />
            </div>
            <div class="dialog-actions">
                <button
                    class="btn-search"
                    onclick={(e) => {
                        e.stopPropagation();
                        saveToFilterLibrary(e);
                    }}>{$t('common.save')}</button
                >
                <button
                    class="btn-clear"
                    onclick={(e) => {
                        e.stopPropagation();
                        cancelSaveDialog(e);
                    }}>{$t('common.cancel')}</button
                >
            </div>
        </div>
    </div>
{/if}

<style>
    .search-panel {
        display: flex;
        height: 100%;
        background: white;
        overflow: hidden;
        font-size: 12px;
        user-select: none;
        -webkit-user-select: none;
    }
    .search-panel input,
    .search-panel textarea {
        user-select: text;
        -webkit-user-select: text;
    }
    .sub-tab-sidebar {
        display: flex;
        flex-direction: column;
        width: 70px;
        flex-shrink: 0;
        background: #f5f5f5;
        border-right: 1px solid #ddd;
    }
    .sub-tab-btn {
        border: none;
        background: transparent;
        padding: 8px 4px;
        font-size: 11px;
        color: #666;
        cursor: pointer;
        border-left: 2px solid transparent;
        text-align: center;
        transition: background 0.15s;
        user-select: none;
        -webkit-user-select: none;
    }
    .sub-tab-btn:hover {
        background: #e8e8e8;
    }
    .sub-tab-btn.active {
        color: #333;
        font-weight: 600;
        background: #fff;
        border-left-color: #555;
    }
    .sub-tab-content {
        flex: 1;
        min-width: 0;
        overflow-y: auto;
        overflow-x: hidden;
    }
    .filter-section {
        display: flex;
        flex-direction: column;
        height: 100%;
    }
    .structure-toggle {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 6px 8px;
        border-bottom: 1px solid #ddd;
        background: #fafafa;
        position: sticky;
        top: 0;
        z-index: 3;
    }
    .structure-toggle.exclude-active {
        background: #fdecea;
        border-bottom-color: #e0b4b0;
    }
    .structure-btn {
        font-size: 11px;
        padding: 3px 10px;
        border: 1px solid #ccc;
        background: #fff;
        color: #555;
        border-radius: 3px;
        cursor: pointer;
    }
    .structure-btn:hover {
        background: #f0f0f0;
    }
    .structure-btn.active {
        color: #333;
        font-weight: 600;
        border-color: #555;
        background: #fff;
    }
    .structure-btn.exclude.active {
        color: #fff;
        background: #c0392b;
        border-color: #c0392b;
    }
    .structure-hint {
        margin-left: auto;
        font-size: 10px;
        color: #c0392b;
        font-style: italic;
    }
    .top-action-bar {
        position: sticky;
        top: 0;
        background: white;
        z-index: 2;
        border-bottom: 1px solid #ddd;
        padding: 6px 8px;
    }
    .filter-groups {
        flex: 1;
        overflow-y: auto;
        padding: 4px 8px 8px;
    }
    .filter-group {
        margin-bottom: 2px;
    }
    .group-header {
        font-size: 11px;
        font-weight: 700;
        color: #555;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        padding: 6px 0 2px;
        border-bottom: 1px solid #eee;
        margin-bottom: 2px;
        user-select: none;
        -webkit-user-select: none;
    }
    .filter-item {
        padding: 2px 0 2px 4px;
        border-radius: 3px;
    }
    .filter-item.active {
        background: #f0f7ff;
    }
    .filter-checkbox {
        display: flex;
        align-items: center;
        gap: 6px;
        cursor: pointer;
        padding: 1px 0;
    }
    .filter-checkbox input[type='checkbox'] {
        margin: 0;
        cursor: pointer;
        accent-color: #6c757d;
    }
    .filter-label {
        font-size: 12px;
        color: #333;
        user-select: none;
    }
    .filter-item.active .filter-label {
        font-weight: 500;
        color: #1a1a1a;
    }
    .filter-params {
        margin: 2px 0 4px 22px;
    }
    .action-bar {
        display: flex;
        align-items: center;
        gap: 8px;
    }
    .active-count {
        font-size: 11px;
        color: #888;
        margin-right: auto;
    }
    .search-in-results {
        display: flex;
        align-items: center;
        gap: 3px;
        font-size: 11px;
        color: #666;
        cursor: pointer;
        user-select: none;
        -webkit-user-select: none;
    }
    .btn-search {
        padding: 4px 12px;
        border: none;
        border-radius: 3px;
        cursor: pointer;
        font-size: 12px;
        background: #6c757d;
        color: white;
    }
    .btn-search:hover {
        background: #5a6268;
    }
    .btn-clear {
        padding: 4px 12px;
        border: none;
        border-radius: 3px;
        cursor: pointer;
        font-size: 12px;
        background: #ccc;
        color: #333;
    }
    .btn-clear:hover {
        background: #999;
    }
    .decision-mode-controls {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    .decision-segment {
        display: flex;
        gap: 4px;
    }
    .decision-btn {
        font-size: 11px;
        padding: 3px 10px;
        border: 1px solid #ccc;
        background: #fff;
        color: #555;
        border-radius: 3px;
        cursor: pointer;
    }
    .decision-btn:hover {
        background: #f0f0f0;
    }
    .decision-btn.active {
        color: #333;
        font-weight: 600;
        border-color: #555;
        background: #fff;
    }
    .label-disabled {
        opacity: 0.45;
    }
    .minmax-controls {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }
    .minmax-controls label {
        display: flex;
        align-items: center;
        gap: 4px;
        font-size: 12px;
        user-select: none;
        -webkit-user-select: none;
    }
    .num-input {
        width: 60px;
        font-size: 12px;
        padding: 2px 3px;
    }
    .date-input {
        font-size: 12px;
        padding: 2px 3px;
    }
    .text-control {
        display: flex;
        align-items: center;
        gap: 6px;
    }
    .hint {
        font-size: 11px;
        color: #888;
        white-space: nowrap;
    }
    .text-input {
        flex: 1;
        font-size: 12px;
        padding: 3px 4px;
        max-width: 200px;
    }

    .history-section {
        flex: 1;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        padding: 4px;
    }
    .empty-message {
        text-align: center;
        color: #888;
        font-size: 11px;
        padding: 12px;
    }
    .history-table-container {
        flex: 1;
        overflow-y: auto;
    }
    .history-table {
        width: 100%;
        border-collapse: collapse;
    }
    .history-table thead {
        position: sticky;
        top: 0;
        background: #f2f2f2;
        z-index: 1;
    }
    .history-table th {
        padding: 2px 4px;
        text-align: center;
        font-weight: bold;
        font-size: 11px;
        border: 1px solid #ddd;
        user-select: none;
    }
    .history-table td {
        padding: 2px 4px;
        border: 1px solid #ddd;
        text-align: center;
        font-size: 11px;
    }
    .history-table tbody tr {
        cursor: pointer;
    }
    .history-table tbody tr:hover {
        background-color: #e6f2ff;
    }
    .history-table tbody tr.selected {
        background-color: #b3d9ff !important;
    }
    .date-cell {
        width: 140px;
        white-space: nowrap;
    }
    .command-cell {
        font-family: monospace;
    }
    .actions-cell {
        width: 60px;
    }
    .action-btn {
        background: none;
        border: none;
        cursor: pointer;
        padding: 1px 3px;
        color: #666;
        display: inline-flex;
        align-items: center;
    }
    .action-btn:hover {
        color: #333;
    }
    .action-btn.in-library {
        color: #333;
    }
    .delete-btn:hover {
        color: #c00;
    }

    .saved-section {
        padding: 4px;
        overflow-y: auto;
        height: 100%;
    }
    .saved-list {
        display: flex;
        flex-direction: column;
        gap: 1px;
    }
    .saved-item {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 4px 8px;
        cursor: pointer;
        border-bottom: 1px solid #f0f0f0;
    }
    .saved-item:hover {
        background: #e6f2ff;
    }
    .saved-item.selected {
        background: #b3d9ff;
    }
    .saved-name {
        font-weight: 600;
        min-width: 120px;
        font-size: 11px;
    }
    .saved-cmd {
        flex: 1;
        font-family: monospace;
        font-size: 11px;
        color: #555;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    input:disabled {
        background-color: #e0e0e0;
    }
    .save-dialog-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1001;
    }
    .save-dialog {
        background: white;
        border-radius: 8px;
        padding: 24px;
        width: 90%;
        max-width: 400px;
        box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
    }
    .save-dialog h3 {
        margin: 0 0 12px;
        font-size: 14px;
    }
    .command-preview {
        background: #f5f5f5;
        padding: 8px;
        border-radius: 4px;
        font-family: monospace;
        font-size: 12px;
        margin-bottom: 12px;
        word-break: break-all;
    }
    .dialog-form {
        margin-bottom: 12px;
    }
    .dialog-form label {
        display: block;
        margin-bottom: 4px;
        font-weight: bold;
        font-size: 12px;
        user-select: none;
        -webkit-user-select: none;
    }
    .dialog-form input {
        width: 100%;
        padding: 6px;
        border: 1px solid #ddd;
        border-radius: 4px;
        font-size: 13px;
        box-sizing: border-box;
    }
    .dialog-actions {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
    }
</style>
