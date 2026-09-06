<script>
    import { logger } from '../utils/logger.js';
    import MinMaxFilterRow from './MinMaxFilterRow.svelte';
    import MatchTournamentPickerModal from './MatchTournamentPickerModal.svelte';
    import Modal from './Modal.svelte';
    import { t, tMsg } from '../i18n';
    import { formatDateTime } from '../utils/format.js';
    import { onMount, onDestroy, tick, untrack } from 'svelte';
    import { statusBarTextStore, currentPositionIndexStore, activeTabStore } from '../stores/uiStore';
    import { positionStore, positionsStore, positionBeforeFilterLibraryStore, positionIndexBeforeFilterLibraryStore } from '../stores/positionStore';
    import { searchExcludePositionStore, searchStructureModeStore, searchOfferedCubeStore, emptySearchBoardPosition, boardHasCheckers } from '../stores/searchExcludePositionStore';
    import { searchHistoryStore, MAX_SEARCH_HISTORY } from '../stores/searchHistoryStore';
    import { buildFilterTokens, buildSearchCommand, parseFilterTokens, parseSearchCommand, filterTokenHint } from '../services/searchFilterService.js';
    import { NUMERIC_FILTERS, NUMERIC_FILTER_BY_LABEL, createFilterState, clear as clearNumeric, toStore as numericToStore, fromStore as numericFromStore } from '../services/filterModel.js';
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
    // Comment filter mode: 'contains' searches the text (t"…"), 'has'/'none'
    // only ask whether a comment is there at all (co / xco). Mutually exclusive
    // by construction — the backend treats presence and content as independent
    // AND clauses, but offering both at once would only allow redundant or
    // contradictory searches.
    let commentMode = $state('contains');
    let movePattern = $state('');
    let matchIDsSelected = $state([]);
    let tournamentIDsSelected = $state([]);
    let showPickerModal = $state(false);
    let playerName = $state('');

    // The 20 numeric min/max/range filters live in one reactive object keyed by
    // filter key — see services/filterModel.js, the table that declares them.
    // Each entry is { option, min, max, rangeMin, rangeMax }; the rows bind
    // straight into it, and clear/save/restore/tokenise are loops over the table.
    let numeric = $state(createFilterState());
    // The 20 numeric backend arguments (`${key}Filter`), read from a parse
    // result keyed by short name (parseFilterTokens → `${short}Filter`,
    // parseSearchCommand → `${short}`).
    const numericArgs = (get) => Object.fromEntries(NUMERIC_FILTERS.map((f) => [`${f.key}Filter`, get(f.short)]));
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

    // Command-line-only filters (#203): `xD65` (exclude a dice roll) and
    // `id5,10` (position id) have no panel checkbox — deliberately, not by
    // omission. Both are repeatable/list-shaped in a way the other checkbox
    // filters aren't (several `xD`/`id` tokens combine, `xD` also needs a
    // 21-roll picker), and both are already documented as typed-command-only
    // in doc/source/cmd_mode.rst. They still parse correctly wherever a
    // command reaches parseSearchTokens (typed, or a history/library replay —
    // see searchFilterService.js's doc comment for why that used to matter).
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
        'Comment',
        'Best Move or Cube Decision',
        'Creation Date',
        'Match IDs',
        'Tournament IDs',
        'Player',
        'Individually Imported',
        'Flagged'
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
        'Individually Imported': 'individuallyImported',
        Flagged: 'flagged',
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
        Comment: 'comment',
        'Best Move or Cube Decision': 'bestMoveOrCubeDecision',
        'Creation Date': 'creationDate',
        'Matches & Tournaments': 'matchesTournaments',
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
        { name: 'Text / Pattern', filters: ['Comment', 'Best Move or Cube Decision'] },
        { name: 'Other', filters: ['Creation Date', 'Matches & Tournaments', 'Player', 'Individually Imported', 'Flagged'] }
    ];

    // Which structure the main board is currently editing: 'include' (au moins)
    // or 'exclude' (sauf). While in 'exclude' mode the include board is stashed.
    // Declared before restoreSearchState() below, which assigns structureMode.
    let structureMode = $state('include');
    let includeBoardStash = $state(null);

    // Initialize all filters as disabled, then restore previous search state if available.
    // Both keys are set through the same forEach (rather than a lone top-level
    // `filterEnabled['Matches & Tournaments'] = false` statement, #205): that
    // stray assignment — every value it wrote was already a literal, not
    // derived from anything reactive, so there was nothing actually stale
    // about it — was the one line of this pair the compiler flagged as
    // "only captures the initial value", unlike the identical-shaped
    // assignment happening inside this very closure.
    [...availableFilters, 'Matches & Tournaments'].forEach((f) => (filterEnabled[f] = false));
    restoreSearchState();

    let activeFilterCount = $derived(availableFilters.filter((f) => filterEnabled[f]).length + (filterEnabled['Matches & Tournaments'] ? 1 : 0));
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

        const activeFilters = availableFilters.filter((f) => filterEnabled[f] || (filterEnabled['Matches & Tournaments'] && (f === 'Match IDs' || f === 'Tournament IDs')));
        const transformedFilters = buildFilterTokens(activeFilters, {
            ...numericToStore(numeric),
            diceRollOption,
            searchText,
            commentMode,
            movePattern,
            creationDateOption,
            creationDateMin,
            creationDateMax,
            creationDateRangeMin,
            creationDateRangeMax,
            matchIDsSelected,
            tournamentIDsSelected,
            playerName
        });

        // Cube sub-type: when the decision-type filter constrains to a cube
        // decision, narrow to double/no-double (`dd`) or take/pass (`dr`). The `d`
        // token already carries the cube decision_type read from the board.
        if (filterEnabled['Include Decision Type'] && decisionMode === 'cube' && cubeSubType !== 'all') {
            transformedFilters.push(cubeSubType === 'takepass' ? 'dr' : 'dd');
        }

        const parsed = parseFilterTokens(transformedFilters);

        const searchCommand = buildSearchCommand(excludeActive ? [...transformedFilters, 'x'] : transformedFilters);

        const excludePositionJSON = excludeActive ? JSON.stringify($searchExcludePositionStore) : '';
        const entry = { command: searchCommand, position: JSON.stringify($positionStore), excludePosition: excludePositionJSON, timestamp: Date.now() };
        searchHistoryStore.update((h) => [entry, ...h].slice(0, MAX_SEARCH_HISTORY));
        SaveSearchHistory(searchCommand, JSON.stringify($positionStore), excludePositionJSON).catch((err) => logger.error('Error saving search history:', err));

        let restrictToPositionIDs = '';
        if (searchInCurrentResults) {
            restrictToPositionIDs = ($positionsStore?.ids || []).filter((id) => id != null).join(',');
        }

        onLoadPositionsByFilters({
            filters: activeFilters.length > 0 ? transformedFilters : [],
            includeCube: parsed.incCube,
            includeScore: parsed.incScore,
            ...numericArgs((short) => parsed[`${short}Filter`]),
            // Only the 'contains' mode carries text. In 'has'/'none' the box is
            // disabled but may still hold what the user typed before switching,
            // and sending it would AND a content filter onto a presence one.
            searchText: commentMode === 'contains' && searchText ? `t"${searchText}"` : '',
            decisionTypeFilter: parsed.dtFilter,
            diceRollFilter: parsed.drFilter,
            movePatternFilter: movePattern ? `m"${movePattern}"` : '',
            dateFilter: parsed.cdFilter,
            noContactFilter: parsed.ncFilter,
            mirrorPositionFilter: parsed.mirFilter,
            individuallyImportedFilter: parsed.iiFilter,
            flaggedFilter: parsed.flFilter,
            searchCommand,
            matchIDsFilter: parsed.matchIDs,
            tournamentIDsFilter: parsed.tournamentIDs,
            restrictToPositionIDs,
            openInNewTab,
            diceRollMode: parsed.drMode,
            playerFilter: playerName ? `pl"${playerName}"` : ''
        });

        saveSearchState();
    }

    function clearFilters() {
        availableFilters.forEach((f) => (filterEnabled[f] = false));
        filterEnabled['Matches & Tournaments'] = false;
        filterEnabled = filterEnabled;
        clearNumeric(numeric);
        searchText = '';
        commentMode = 'contains';
        movePattern = '';
        diceRollOption = 'both';
        decisionMode = 'checker';
        cubeSubType = 'all';
        searchOfferedCubeStore.set(false);
        matchIDsSelected = [];
        tournamentIDsSelected = [];
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
            onLoadPositionsByFilters({
                filters: f.cmdFilters,
                includeCube: f.ic,
                includeScore: f.is,
                ...numericArgs((short) => f[short]),
                searchText: f.st,
                decisionTypeFilter: f.dt,
                diceRollFilter: f.dr,
                movePatternFilter: f.mpf,
                dateFilter: f.cd,
                noContactFilter: f.nc,
                mirrorPositionFilter: f.mp,
                individuallyImportedFilter: f.ii,
                flaggedFilter: f.fl,
                searchCommand: command,
                matchIDsFilter: f.matchIDs,
                tournamentIDsFilter: f.tournamentIDs,
                diceRollMode: f.drMode,
                playerFilter: f.plf,
                // Command-line-only tokens with no panel checkbox (#203): unlike
                // commentFilter/cubeResponseFilter, positionService does not
                // re-derive these from `filters`, so they must be forwarded
                // explicitly or a replayed `s D xD65`/`s id5,10` silently loses
                // the exclusion/restriction on double-click.
                exceptDiceFilter: f.xd,
                positionIDsFilter: f.posIds,
                gamePhaseFilter: f.ph,
                commentOriginFilter: f.coOrigin
            });
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
        return formatDateTime(timestamp);
    }

    // --- Saved filter (bookmarked search) functions ---
    async function selectSavedFilter(filter) {
        if (selectedSavedFilter && selectedSavedFilter.id === filter.id) {
            selectedSavedFilter = null;
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
        }
    }

    function handleKeyDown(event) {
        if ($activeTabStore !== 'search') return;
        if (event.target.matches('input, textarea, select')) {
            // Escape belongs to the global dispatcher (App.svelte listens on
            // `window`, one level above this `document` listener): it blurs
            // the field so the bare-key shortcuts work again. Stopping it here
            // left the user stuck in the field with no way out but the mouse.
            // Tab stays with the field on purpose, same as Escape: since #204,
            // the global dispatcher only hijacks a bare Tab into "open the
            // search tab" while focus sits on the board — but stopping it here
            // too means Tab moves between this form's fields even before that
            // guard is reached, and protects against a future change to it.
            if (event.key === 'Escape') return;
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
            commentMode,
            movePattern,
            matchIDsSelected,
            tournamentIDsSelected,
            playerName,
            ...numericToStore(numeric),
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
        commentMode = saved.commentMode ?? 'contains';
        movePattern = saved.movePattern;
        matchIDsSelected = Array.isArray(saved.matchIDsSelected) ? saved.matchIDsSelected : [];
        tournamentIDsSelected = Array.isArray(saved.tournamentIDsSelected) ? saved.tournamentIDsSelected : [];
        playerName = saved.playerName ?? '';
        numericFromStore(numeric, saved);
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
                                            {:else if NUMERIC_FILTER_BY_LABEL[filter]}
                                                {@const nf = NUMERIC_FILTER_BY_LABEL[filter]}
                                                <MinMaxFilterRow
                                                    bind:option={numeric[nf.key].option}
                                                    bind:minVal={numeric[nf.key].min}
                                                    bind:maxVal={numeric[nf.key].max}
                                                    bind:rangeMin={numeric[nf.key].rangeMin}
                                                    bind:rangeMax={numeric[nf.key].rangeMax}
                                                    min={nf.bounds.min}
                                                    max={nf.bounds.max}
                                                />
                                            {:else if filter === 'Comment'}
                                                <div class="minmax-controls">
                                                    <label
                                                        ><input type="radio" name="commentMode" value="contains" checked={commentMode === 'contains'} onchange={() => (commentMode = 'contains')} />
                                                        {$t('search.comment.contains')}</label
                                                    ><label
                                                        ><input type="radio" name="commentMode" value="has" checked={commentMode === 'has'} onchange={() => (commentMode = 'has')} />
                                                        {$t('search.comment.has')}</label
                                                    ><label
                                                        ><input type="radio" name="commentMode" value="none" checked={commentMode === 'none'} onchange={() => (commentMode = 'none')} />
                                                        {$t('search.comment.none')}</label
                                                    >
                                                </div>
                                                {#if commentMode === 'contains'}
                                                    <div class="text-control">
                                                        <span class="hint">{$t('search.searchTextHint')}</span><input type="text" bind:value={searchText} class="text-input" />
                                                    </div>
                                                {/if}
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
                                            {:else if filter === 'Matches & Tournaments'}
                                                <div class="text-control">
                                                    <span class="hint">
                                                        {$t('search.matchesTournamentsCount', { matches: matchIDsSelected.length, tournaments: tournamentIDsSelected.length })}
                                                    </span>
                                                    <button type="button" class="small-btn" onclick={() => (showPickerModal = true)}>{$t('search.openPicker')}</button>
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

<MatchTournamentPickerModal
    visible={showPickerModal}
    {matchIDsSelected}
    {tournamentIDsSelected}
    onApply={(matches, tournaments) => {
        matchIDsSelected = matches;
        tournamentIDsSelected = tournaments;
        showPickerModal = false;
    }}
    onCancel={() => (showPickerModal = false)}
/>

<Modal open={showSaveDialog} onclose={cancelSaveDialog} size="small" closeButton={false} closeOnOverlay={true} label={$t('search.saveSearch')}>
    <h3 class="dialog-title">{$t('search.saveSearch')}</h3>
    <p class="command-preview">{selectedSearch?.command || ''}</p>
    <div class="dialog-form">
        <label for="filterNameInput">{$t('search.name')}</label>
        <input type="text" id="filterNameInput" bind:value={filterName} placeholder={$t('search.enterName')} onkeydown={(e) => e.key === 'Enter' && saveToFilterLibrary()} />
    </div>
    {#snippet footer()}
        <button class="btn-search" onclick={saveToFilterLibrary}>{$t('common.save')}</button>
        <button class="btn-clear" onclick={cancelSaveDialog}>{$t('common.cancel')}</button>
    {/snippet}
</Modal>

<style>
    .search-panel {
        display: flex;
        height: 100%;
        background: white;
        overflow: hidden;
        font-size: var(--font-size-base);
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
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
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
        color: var(--color-text);
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
        font-size: var(--font-size-small);
        padding: 3px 10px;
        border: 1px solid var(--color-border);
        background: #fff;
        color: #555;
        border-radius: 3px;
        cursor: pointer;
    }
    .structure-btn:hover {
        background: #f0f0f0;
    }
    .structure-btn.active {
        color: var(--color-text);
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
        font-size: var(--font-size-small);
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
        font-size: var(--font-size-small);
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
        font-size: var(--font-size-base);
        color: var(--color-text);
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
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        margin-right: auto;
    }
    .search-in-results {
        display: flex;
        align-items: center;
        gap: 3px;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        cursor: pointer;
        user-select: none;
        -webkit-user-select: none;
    }
    .btn-search {
        padding: 4px 12px;
        border: none;
        border-radius: 3px;
        cursor: pointer;
        font-size: var(--font-size-base);
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
        font-size: var(--font-size-base);
        background: #ccc;
        color: var(--color-text);
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
        font-size: var(--font-size-small);
        padding: 3px 10px;
        border: 1px solid var(--color-border);
        background: #fff;
        color: #555;
        border-radius: 3px;
        cursor: pointer;
    }
    .decision-btn:hover {
        background: #f0f0f0;
    }
    .decision-btn.active {
        color: var(--color-text);
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
        font-size: var(--font-size-base);
        user-select: none;
        -webkit-user-select: none;
    }
    .num-input {
        width: 60px;
        font-size: var(--font-size-base);
        padding: 2px 3px;
    }
    .date-input {
        font-size: var(--font-size-base);
        padding: 2px 3px;
    }
    .text-control {
        display: flex;
        align-items: center;
        gap: 6px;
    }
    .hint {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        white-space: nowrap;
    }
    .text-input {
        flex: 1;
        font-size: var(--font-size-base);
        padding: 3px 4px;
        max-width: 200px;
    }
    .small-btn {
        padding: 3px 10px;
        border: none;
        border-radius: 3px;
        cursor: pointer;
        font-size: var(--font-size-small);
        background: #ccc;
        color: var(--color-text);
    }
    .small-btn:hover {
        background: #999;
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
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
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
        font-size: var(--font-size-small);
        border: 1px solid #ddd;
        user-select: none;
    }
    .history-table td {
        padding: 2px 4px;
        border: 1px solid #ddd;
        text-align: center;
        font-size: var(--font-size-small);
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
        font-family: var(--font-family-mono);
    }
    .actions-cell {
        width: 60px;
    }
    .action-btn {
        background: none;
        border: none;
        cursor: pointer;
        padding: 1px 3px;
        color: var(--color-text-muted);
        display: inline-flex;
        align-items: center;
    }
    .action-btn:hover {
        color: var(--color-text);
    }
    .action-btn.in-library {
        color: var(--color-text);
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
        font-size: var(--font-size-small);
    }
    .saved-cmd {
        flex: 1;
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: #555;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    input:disabled {
        background-color: #e0e0e0;
    }
    .dialog-title {
        margin: 0 0 12px;
        font-size: var(--font-size-base);
    }
    .command-preview {
        background: var(--color-surface-alt);
        padding: 8px;
        border-radius: var(--radius);
        font-family: var(--font-family-mono);
        font-size: var(--font-size-base);
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
        font-size: var(--font-size-base);
        user-select: none;
        -webkit-user-select: none;
    }
    .dialog-form input {
        width: 100%;
        padding: 6px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        font-size: var(--font-size-base);
        box-sizing: border-box;
    }
</style>
