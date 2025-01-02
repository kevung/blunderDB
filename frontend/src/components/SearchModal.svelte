<script>
    import { createEventDispatcher, onMount, onDestroy } from 'svelte';
    import { statusBarModeStore } from '../stores/uiStore';

    export let visible = false;
    export let onClose;
    export let onLoadPositionsByFilters;

    const dispatch = createEventDispatcher();

    let filters = [];
    let includeCube = false;
    let includeScore = false;
    let pipCountMin = -375;
    let pipCountMax = 375;
    let winRateFilter = '';
    let gammonRateFilter = '';
    let backgammonRateFilter = '';
    let player2WinRateFilter = '';
    let player2GammonRateFilter = '';
    let player2BackgammonRateFilter = '';
    let player1CheckerOffFilter = '';
    let player2CheckerOffFilter = '';
    let player1BackCheckerFilter = '';
    let player2BackCheckerFilter = '';
    let player1CheckerInZoneFilter = '';
    let player2CheckerInZoneFilter = '';
    let searchText = ''; // Update this line to handle multiple words
    let player1AbsolutePipCountFilter = '';
    let equityFilter = '';
    let decisionTypeFilter = false; // Rename this line
    let diceRollFilter = false; // Add this line

    let selectedFilter = '';
    let pipCountOption = 'min'; // Default option for pip count
    let pipCountRangeMin = -375; // Min value for pip count range
    let pipCountRangeMax = 375; // Max value for pip count range
    let winRateOption = 'min'; // Default option for win rate
    let winRateMin = 0; // Min value for win rate
    let winRateMax = 100; // Max value for win rate
    let winRateRangeMin = 0; // Min value for win rate range
    let winRateRangeMax = 100; // Max value for win rate range
    let gammonRateOption = 'min'; // Default option for gammon rate
    let gammonRateMin = 0; // Min value for gammon rate
    let gammonRateMax = 100; // Max value for gammon rate
    let gammonRateRangeMin = 0; // Min value for gammon rate range
    let gammonRateRangeMax = 100; // Max value for gammon rate range
    let backgammonRateOption = 'min'; // Default option for backgammon rate
    let backgammonRateMin = 0; // Min value for backgammon rate
    let backgammonRateMax = 100; // Max value for backgammon rate
    let backgammonRateRangeMin = 0; // Min value for backgammon rate range
    let backgammonRateRangeMax = 100; // Max value for backgammon rate range

    let player2WinRateOption = 'min'; // Default option for opponent win rate
    let player2WinRateMin = 0; // Min value for opponent win rate
    let player2WinRateMax = 100; // Max value for opponent win rate
    let player2WinRateRangeMin = 0; // Min value for opponent win rate range
    let player2WinRateRangeMax = 100; // Max value for opponent win rate range

    let player2GammonRateOption = 'min'; // Default option for opponent gammon rate
    let player2GammonRateMin = 0; // Min value for opponent gammon rate
    let player2GammonRateMax = 100; // Max value for opponent gammon rate
    let player2GammonRateRangeMin = 0; // Min value for opponent gammon rate range
    let player2GammonRateRangeMax = 100; // Max value for opponent gammon rate range

    let player2BackgammonRateOption = 'min'; // Default option for opponent backgammon rate
    let player2BackgammonRateMin = 0; // Min value for opponent backgammon rate
    let player2BackgammonRateMax = 100; // Max value for opponent backgammon rate
    let player2BackgammonRateRangeMin = 0; // Min value for opponent backgammon rate range
    let player2BackgammonRateRangeMax = 100; // Max value for opponent backgammon rate range

    let player1CheckerOffOption = 'min'; // Default option for player checker off
    let player1CheckerOffMin = 0; // Min value for player checker off
    let player1CheckerOffMax = 15; // Max value for player checker off
    let player1CheckerOffRangeMin = 0; // Min value for player checker off range
    let player1CheckerOffRangeMax = 15; // Max value for player checker off range

    let player2CheckerOffOption = 'min'; // Default option for opponent checker off
    let player2CheckerOffMin = 0; // Min value for opponent checker off
    let player2CheckerOffMax = 15; // Max value for opponent checker off
    let player2CheckerOffRangeMin = 0; // Min value for opponent checker off range
    let player2CheckerOffRangeMax = 15; // Max value for opponent checker off range

    let player1BackCheckerOption = 'min'; // Default option for player back checker
    let player1BackCheckerMin = 0; // Min value for player back checker
    let player1BackCheckerMax = 15; // Max value for player back checker
    let player1BackCheckerRangeMin = 0; // Min value for player back checker range
    let player1BackCheckerRangeMax = 15; // Max value for player back checker range

    let player2BackCheckerOption = 'min'; // Default option for opponent back checker
    let player2BackCheckerMin = 0; // Min value for opponent back checker
    let player2BackCheckerMax = 15; // Max value for opponent back checker
    let player2BackCheckerRangeMin = 0; // Min value for opponent back checker range
    let player2BackCheckerRangeMax = 15; // Max value for opponent back checker range

    let player1CheckerInZoneOption = 'min'; // Default option for player checker in zone
    let player1CheckerInZoneMin = 0; // Min value for player checker in zone
    let player1CheckerInZoneMax = 15; // Max value for player checker in zone
    let player1CheckerInZoneRangeMin = 0; // Min value for player checker in zone range
    let player1CheckerInZoneRangeMax = 15; // Max value for player checker in zone range

    let player2CheckerInZoneOption = 'min'; // Default option for opponent checker in zone
    let player2CheckerInZoneMin = 0; // Min value for opponent checker in zone
    let player2CheckerInZoneMax = 15; // Max value for opponent checker in zone
    let player2CheckerInZoneRangeMin = 0; // Min value for opponent checker in zone range
    let player2CheckerInZoneRangeMax = 15; // Max value for opponent checker in zone range

    let player1AbsolutePipCountOption = 'min'; // Default option for player absolute pip count
    let player1AbsolutePipCountMin = 0; // Min value for player absolute pip count
    let player1AbsolutePipCountMax = 375; // Max value for player absolute pip count
    let player1AbsolutePipCountRangeMin = 0; // Min value for player absolute pip count range
    let player1AbsolutePipCountRangeMax = 375; // Max value for player absolute pip count range

    let equityOption = 'min'; // Default option for equity
    let equityMin = -1000; // Min value for equity
    let equityMax = 1000; // Max value for equity
    let equityRangeMin = -1000; // Min value for equity range
    let equityRangeMax = 1000; // Max value for equity range

    let availableFilters = [
        'Include Cube',
        'Include Score',
        'Include Decision Type', // Rename this line
        'Include Dice Roll', // Add this line
        'Pipcount Difference',
        'Player Absolute Pipcount',
        'Equity (millipoints)',
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
        'Search Text'
    ];

    function addFilter() {
        if (selectedFilter && !filters.includes(selectedFilter)) {
            filters = [...filters, selectedFilter];
            selectedFilter = '';
        }
    }

    function removeFilter(filter) {
        filters = filters.filter(f => f !== filter);
    }

    function handleSearch() {
        const transformedFilters = filters.map(filter => {
            switch (filter) {
                case 'Include Cube':
                    return 'cube';
                case 'Include Score':
                    return 'score';
                case 'Pipcount Difference':
                    return pipCountOption === 'min' ? `p>${pipCountMin}` : pipCountOption === 'max' ? `p<${pipCountMax}` : `p${pipCountRangeMin},${pipCountRangeMax}`;
                case 'Player Absolute Pipcount':
                    return player1AbsolutePipCountOption === 'min' ? `P>${player1AbsolutePipCountMin}` : player1AbsolutePipCountOption === 'max' ? `P<${player1AbsolutePipCountMax}` : `P${player1AbsolutePipCountRangeMin},${player1AbsolutePipCountRangeMax}`;
                case 'Equity (millipoints)':
                    return equityOption === 'min' ? `e>${equityMin}` : equityOption === 'max' ? `e<${equityMax}` : `e${equityRangeMin},${equityRangeMax}`;
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
                case 'Search Text':
                    return searchText.split(';').map(text => text.trim()).join(' ');
                default:
                    return '';
            }
        });

        // Ensure 'Include Cube' filter is correctly handled
        const includeCube = filters.includes('Include Cube');
        const includeScore = filters.includes('Include Score');
        const pipCountFilter = transformedFilters.find(filter => filter.startsWith('p'));
        const winRateFilter = transformedFilters.find(filter => filter.startsWith('w'));
        const gammonRateFilter = transformedFilters.find(filter => filter.startsWith('g'));
        const backgammonRateFilter = transformedFilters.find(filter => filter.startsWith('b'));
        const player2WinRateFilter = transformedFilters.find(filter => filter.startsWith('W'));
        const player2GammonRateFilter = transformedFilters.find(filter => filter.startsWith('G'));
        const player2BackgammonRateFilter = transformedFilters.find(filter => filter.startsWith('B'));
        const player1CheckerOffFilter = transformedFilters.find(filter => filter.startsWith('o'));
        const player2CheckerOffFilter = transformedFilters.find(filter => filter.startsWith('O'));
        const player1BackCheckerFilter = transformedFilters.find(filter => filter.startsWith('k'));
        const player2BackCheckerFilter = transformedFilters.find(filter => filter.startsWith('K'));
        const player1CheckerInZoneFilter = transformedFilters.find(filter => filter.startsWith('z'));
        const player2CheckerInZoneFilter = transformedFilters.find(filter => filter.startsWith('Z'));
        const player1AbsolutePipCountFilter = transformedFilters.find(filter => filter.startsWith('P'));
        const equityFilter = transformedFilters.find(filter => filter.startsWith('e'));
        const searchTextFilter = searchText.split(';').map(text => text.trim()).join(' ');

        const decisionTypeFilter = filters.includes('Include Decision Type'); // Rename this line
        const diceRollFilter = filters.includes('Include Dice Roll'); // Add this line

        const searchTextArray = searchText; // Update this line to pass searchText as a single string

        statusBarModeStore.set('NORMAL');
        onLoadPositionsByFilters(transformedFilters, includeCube, includeScore, pipCountFilter, winRateFilter, gammonRateFilter, backgammonRateFilter, player2WinRateFilter, player2GammonRateFilter, player2BackgammonRateFilter, player1CheckerOffFilter, player2CheckerOffFilter, player1BackCheckerFilter, player2BackCheckerFilter, player1CheckerInZoneFilter, player2CheckerInZoneFilter, searchTextArray, player1AbsolutePipCountFilter, equityFilter, decisionTypeFilter, diceRollFilter); // Rename this line
        onClose();
    }

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            onClose();
        } else if (event.key === 'Enter') {
            handleSearch();
        }
    }

    function clearFilters() {
        filters = [];
        includeCube = false;
        includeScore = false;
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
        searchText = '';
    }

    onMount(() => {
        window.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        window.removeEventListener('keydown', handleKeyDown);
    });
</script>

{#if visible}
    <div class="modal-backdrop" on:click={onClose}></div>
    <div class="modal" on:click|stopPropagation>
        <div class="modal-body">
            <div class="form-group">
                <select bind:value={selectedFilter} class="filter-dropdown">
                    <option value="" disabled>Select a filter</option>
                    {#each availableFilters as filter}
                        <option value={filter}>{filter}</option>
                    {/each}
                </select>
                <button class="add-button" on:click={addFilter}>+</button>
            </div>
            {#each filters as filter}
                <div class="form-group">
                    <div class="filter-label-container">
                        <label class="filter-label">{filter}</label>
                    </div>
                    <div class="filter-options-wrapper">
                        {#if filter === 'Include Cube' || filter === 'Include Score' || filter === 'Include Decision Type' || filter === 'Include Dice Roll'}
                            <!-- No input needed for these filters -->
                        {/if}
                        {#if filter === 'Pipcount Difference'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={pipCountOption} value="min" /> Min
                                        <input type="number" bind:value={pipCountMin} placeholder="Min" class="filter-input" disabled={pipCountOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={pipCountOption} value="max" /> Max
                                        <input type="number" bind:value={pipCountMax} placeholder="Max" class="filter-input" disabled={pipCountOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={pipCountOption} value="range" /> Range
                                        <input type="number" bind:value={pipCountRangeMin} placeholder="Min" class="filter-input" disabled={pipCountOption !== 'range'} />
                                        <input type="number" bind:value={pipCountRangeMax} placeholder="Max" class="filter-input" disabled={pipCountOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Player Absolute Pipcount'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1AbsolutePipCountOption} value="min" /> Min
                                        <input type="number" bind:value={player1AbsolutePipCountMin} placeholder="Min" class="filter-input" min="0" max="375" disabled={player1AbsolutePipCountOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1AbsolutePipCountOption} value="max" /> Max
                                        <input type="number" bind:value={player1AbsolutePipCountMax} placeholder="Max" class="filter-input" min="0" max="375" disabled={player1AbsolutePipCountOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1AbsolutePipCountOption} value="range" /> Range
                                        <input type="number" bind:value={player1AbsolutePipCountRangeMin} placeholder="Min" class="filter-input" min="0" max="375" disabled={player1AbsolutePipCountOption !== 'range'} />
                                        <input type="number" bind:value={player1AbsolutePipCountRangeMax} placeholder="Max" class="filter-input" min="0" max="375" disabled={player1AbsolutePipCountOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Equity (millipoints)'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={equityOption} value="min" /> Min
                                        <input type="number" bind:value={equityMin} placeholder="Min" class="filter-input" disabled={equityOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={equityOption} value="max" /> Max
                                        <input type="number" bind:value={equityMax} placeholder="Max" class="filter-input" disabled={equityOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={equityOption} value="range" /> Range
                                        <input type="number" bind:value={equityRangeMin} placeholder="Min" class="filter-input" disabled={equityOption !== 'range'} />
                                        <input type="number" bind:value={equityRangeMax} placeholder="Max" class="filter-input" disabled={equityOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Win Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={winRateOption} value="min" /> Min
                                        <input type="number" bind:value={winRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={winRateOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={winRateOption} value="max" /> Max
                                        <input type="number" bind:value={winRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={winRateOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={winRateOption} value="range" /> Range
                                        <input type="number" bind:value={winRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={winRateOption !== 'range'} />
                                        <input type="number" bind:value={winRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={winRateOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Gammon Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={gammonRateOption} value="min" /> Min
                                        <input type="number" bind:value={gammonRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={gammonRateOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={gammonRateOption} value="max" /> Max
                                        <input type="number" bind:value={gammonRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={gammonRateOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={gammonRateOption} value="range" /> Range
                                        <input type="number" bind:value={gammonRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={gammonRateOption !== 'range'} />
                                        <input type="number" bind:value={gammonRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={gammonRateOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Backgammon Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={backgammonRateOption} value="min" /> Min
                                        <input type="number" bind:value={backgammonRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={backgammonRateOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={backgammonRateOption} value="max" /> Max
                                        <input type="number" bind:value={backgammonRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={backgammonRateOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={backgammonRateOption} value="range" /> Range
                                        <input type="number" bind:value={backgammonRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={backgammonRateOption !== 'range'} />
                                        <input type="number" bind:value={backgammonRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={backgammonRateOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Win Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2WinRateOption} value="min" /> Min
                                        <input type="number" bind:value={player2WinRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2WinRateOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2WinRateOption} value="max" /> Max
                                        <input type="number" bind:value={player2WinRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2WinRateOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2WinRateOption} value="range" /> Range
                                        <input type="number" bind:value={player2WinRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2WinRateOption !== 'range'} />
                                        <input type="number" bind:value={player2WinRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2WinRateOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Gammon Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2GammonRateOption} value="min" /> Min
                                        <input type="number" bind:value={player2GammonRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2GammonRateOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2GammonRateOption} value="max" /> Max
                                        <input type="number" bind:value={player2GammonRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2GammonRateOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2GammonRateOption} value="range" /> Range
                                        <input type="number" bind:value={player2GammonRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2GammonRateOption !== 'range'} />
                                        <input type="number" bind:value={player2GammonRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2GammonRateOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Backgammon Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackgammonRateOption} value="min" /> Min
                                        <input type="number" bind:value={player2BackgammonRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2BackgammonRateOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackgammonRateOption} value="max" /> Max
                                        <input type="number" bind:value={player2BackgammonRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2BackgammonRateOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackgammonRateOption} value="range" /> Range
                                        <input type="number" bind:value={player2BackgammonRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2BackgammonRateOption !== 'range'} />
                                        <input type="number" bind:value={player2BackgammonRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} disabled={player2BackgammonRateOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Player Checker-Off'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerOffOption} value="min" /> Min
                                        <input type="number" bind:value={player1CheckerOffMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1CheckerOffOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerOffOption} value="max" /> Max
                                        <input type="number" bind:value={player1CheckerOffMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1CheckerOffOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerOffOption} value="range" /> Range
                                        <input type="number" bind:value={player1CheckerOffRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1CheckerOffOption !== 'range'} />
                                        <input type="number" bind:value={player1CheckerOffRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1CheckerOffOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Checker-Off'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerOffOption} value="min" /> Min
                                        <input type="number" bind:value={player2CheckerOffMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2CheckerOffOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerOffOption} value="max" /> Max
                                        <input type="number" bind:value={player2CheckerOffMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2CheckerOffOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerOffOption} value="range" /> Range
                                        <input type="number" bind:value={player2CheckerOffRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2CheckerOffOption !== 'range'} />
                                        <input type="number" bind:value={player2CheckerOffRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2CheckerOffOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Player Back Checker'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1BackCheckerOption} value="min" /> Min
                                        <input type="number" bind:value={player1BackCheckerMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1BackCheckerOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1BackCheckerOption} value="max" /> Max
                                        <input type="number" bind:value={player1BackCheckerMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1BackCheckerOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1BackCheckerOption} value="range" /> Range
                                        <input type="number" bind:value={player1BackCheckerRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1BackCheckerOption !== 'range'} />
                                        <input type="number" bind:value={player1BackCheckerRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1BackCheckerOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Back Checker'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackCheckerOption} value="min" /> Min
                                        <input type="number" bind:value={player2BackCheckerMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2BackCheckerOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackCheckerOption} value="max" /> Max
                                        <input type="number" bind:value={player2BackCheckerMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2BackCheckerOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackCheckerOption} value="range" /> Range
                                        <input type="number" bind:value={player2BackCheckerRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2BackCheckerOption !== 'range'} />
                                        <input type="number" bind:value={player2BackCheckerRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2BackCheckerOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Player Checker in the Zone'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerInZoneOption} value="min" /> Min
                                        <input type="number" bind:value={player1CheckerInZoneMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1CheckerInZoneOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerInZoneOption} value="max" /> Max
                                        <input type="number" bind:value={player1CheckerInZoneMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1CheckerInZoneOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerInZoneOption} value="range" /> Range
                                        <input type="number" bind:value={player1CheckerInZoneRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1CheckerInZoneOption !== 'range'} />
                                        <input type="number" bind:value={player1CheckerInZoneRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player1CheckerInZoneOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Checker in the Zone'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerInZoneOption} value="min" /> Min
                                        <input type="number" bind:value={player2CheckerInZoneMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2CheckerInZoneOption !== 'min'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerInZoneOption} value="max" /> Max
                                        <input type="number" bind:value={player2CheckerInZoneMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2CheckerInZoneOption !== 'max'} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerInZoneOption} value="range" /> Range
                                        <input type="number" bind:value={player2CheckerInZoneRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2CheckerInZoneOption !== 'range'} />
                                        <input type="number" bind:value={player2CheckerInZoneRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} disabled={player2CheckerInZoneOption !== 'range'} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Search Text'}
                            <div class="search-text-container">
                                <input type="text" bind:value={searchText} class="filter-input search-text-input" />
                            </div>
                        {/if}
                    </div>
                    <button class="remove-button" on:click={() => removeFilter(filter)}>−</button>
                </div>
            {/each}
            <div class="modal-buttons">
                <button class="primary-button" on:click={handleSearch}>Search</button>
                <button class="secondary-button" on:click={onClose}>Cancel</button>
                <button class="secondary-button" on:click={clearFilters}>Clear Filters</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.5);
        z-index: 999;
    }

    .modal {
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        background: white;
        padding: 1rem;
        border-radius: 8px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        z-index: 1000;
        width: 90%;
        max-width: 800px; /* Increase the max-width */
        max-height: 80vh; /* Limit the height of the modal */
        overflow-y: auto; /* Add vertical scrollbar if content exceeds max height */
        padding-left: 1rem; /* Add left padding to make it symmetric */
    }

    .modal-header {
        display: none; /* Remove the header */
    }

    .close-button {
        background: none;
        border: none;
        font-size: 1.5rem;
        cursor: pointer;
    }

    .modal-body {
        padding: 0; /* Remove padding to eliminate extra white space */
        display: flex;
        flex-direction: column;
        gap: 15px; /* Add space between filters */
    }

    .form-group {
        display: flex;
        align-items: center;
        justify-content: space-between; /* Align text of filters */
        margin-bottom: 10px;
    }

    .filter-label-container {
        width: 120px; /* Set a fixed width for the container */
        height: 50px; /* Set a fixed height for the container */
        display: flex;
        align-items: center;
    }

    .filter-label {
        flex: 1;
        margin-bottom: 5px;
        font-size: 18px; /* Set font size */
        text-align: left; /* Align text to the left */
        width: 250px; /* Set a fixed width for the filter label */
        padding-left: 20px; /* Increase left padding */
    }

    .filter-options-wrapper {
        flex: 2;
        margin-left: 20px; /* Increase gap between filter label and filter options */
    }

    input[type="text"], input[type="number"], select {
        flex: 2;
        margin-right: 10px;
        font-size: 18px; /* Set font size */
        width: 200px; /* Set a fixed width for input fields */
    }

    .filter-dropdown {
        font-size: 18px; /* Set font size to match the text description of the filter */
        width: 200px; /* Set a fixed width for dropdown */
    }

    .filter-input {
        font-size: 18px; /* Set font size */
        width: 50px; /* Set a fixed width for input fields */
    }

    .search-text-container {
        width: 100%;
        display: flex;
        justify-content: flex-start; /* Align to start */
    }

    .search-text-input {
        width: 100%; /* Make the search text input fill its parent container */
    }

    .add-button, .remove-button {
        background: none;
        border: none;
        font-size: 1.5rem;
        cursor: pointer;
        color: #6c757d;
        width: 50px; /* Set a fixed width for both add and remove buttons */
    }

    .modal-buttons {
        margin-top: 10px;
        display: flex;
        justify-content: center;
        gap: 10px; /* Add space between buttons */
    }

    .modal-buttons button {
        padding: 8px 14px; /* Slightly increase padding */
        border: none;
        border-radius: 4px;
        cursor: pointer;
        font-size: 15px; /* Slightly increase font size */
    }

    .primary-button {
        background-color: #6c757d; /* Sober grey color */
        color: white;
    }

    .secondary-button {
        background-color: #ccc;
    }

    .primary-button:hover {
        background-color: #5a6268; /* Slightly darker grey on hover */
    }

    .secondary-button:hover {
        background-color: #999;
    }

    .filter-input-container {
        display: flex;
        align-items: center;
        margin-top: 5px; /* Add margin to align below the dropdown */
    }

    .filter-options {
        display: flex;
        flex-direction: column;
        gap: 5px;
        align-items: flex-start; /* Align text of filters */
        width: 300px; /* Set a fixed width for filter options */
    }

    .filter-option {
        display: flex;
        align-items: center;
        gap: 10px; /* Add space between radio button and input */
        flex: 1;
        width: 100%; /* Ensure the filter option takes full width */
    }

    .filter-option label {
        width: 100px; /* Set a fixed width for the label */
        text-align: left; /* Align text to the left */
    }

    .filter-input {
        flex: 1; /* Ensure the input field takes the remaining space */
    }

    input[type="number"] {
        width: 100px; /* Set a fixed width for number input fields */
    }

    .filter-options-container {
        display: flex;
        justify-content: center;
        width: 100%;
    }

    .filter-options-container.expanded {
        width: 100%;
    }

    .filter-options.expanded {
        width: 100%;
    }

    .debug-border {
        border: 1px solid red; /* Add border for debugging */
    }

    input:disabled {
        background-color: #c0c0c0; /* A little more darker color for disabled input fields */
    }
</style>