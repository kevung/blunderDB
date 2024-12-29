<script>
    import { createEventDispatcher, onMount, onDestroy } from 'svelte';

    export let visible = false;
    export let onClose;

    const dispatch = createEventDispatcher();

    let filters = [];
    let includeCube = false;
    let includeScore = false;
    let pipCountMin = '';
    let pipCountMax = '';
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
    let searchText = '';
    let player1AbsolutePipCountFilter = '';
    let equityFilter = '';

    let selectedFilter = '';
    let pipCountOption = 'min'; // Default option for pip count
    let pipCountRangeMin = ''; // Min value for pip count range
    let pipCountRangeMax = ''; // Max value for pip count range
    let winRateOption = 'min'; // Default option for win rate
    let winRateMin = ''; // Min value for win rate
    let winRateMax = ''; // Max value for win rate
    let winRateRangeMin = ''; // Min value for win rate range
    let winRateRangeMax = ''; // Max value for win rate range
    let gammonRateOption = 'min'; // Default option for gammon rate
    let gammonRateMin = ''; // Min value for gammon rate
    let gammonRateMax = ''; // Max value for gammon rate
    let gammonRateRangeMin = ''; // Min value for gammon rate range
    let gammonRateRangeMax = ''; // Max value for gammon rate range
    let backgammonRateOption = 'min'; // Default option for backgammon rate
    let backgammonRateMin = ''; // Min value for backgammon rate
    let backgammonRateMax = ''; // Max value for backgammon rate
    let backgammonRateRangeMin = ''; // Min value for backgammon rate range
    let backgammonRateRangeMax = ''; // Max value for backgammon rate range

    let player2WinRateOption = 'min'; // Default option for opponent win rate
    let player2WinRateMin = ''; // Min value for opponent win rate
    let player2WinRateMax = ''; // Max value for opponent win rate
    let player2WinRateRangeMin = ''; // Min value for opponent win rate range
    let player2WinRateRangeMax = ''; // Max value for opponent win rate range

    let player2GammonRateOption = 'min'; // Default option for opponent gammon rate
    let player2GammonRateMin = ''; // Min value for opponent gammon rate
    let player2GammonRateMax = ''; // Max value for opponent gammon rate
    let player2GammonRateRangeMin = ''; // Min value for opponent gammon rate range
    let player2GammonRateRangeMax = ''; // Max value for opponent gammon rate range

    let player2BackgammonRateOption = 'min'; // Default option for opponent backgammon rate
    let player2BackgammonRateMin = ''; // Min value for opponent backgammon rate
    let player2BackgammonRateMax = ''; // Max value for opponent backgammon rate
    let player2BackgammonRateRangeMin = ''; // Min value for opponent backgammon rate range
    let player2BackgammonRateRangeMax = ''; // Max value for opponent backgammon rate range

    let player1CheckerOffOption = 'min'; // Default option for player checker off
    let player1CheckerOffMin = ''; // Min value for player checker off
    let player1CheckerOffMax = ''; // Max value for player checker off
    let player1CheckerOffRangeMin = ''; // Min value for player checker off range
    let player1CheckerOffRangeMax = ''; // Max value for player checker off range

    let player2CheckerOffOption = 'min'; // Default option for opponent checker off
    let player2CheckerOffMin = ''; // Min value for opponent checker off
    let player2CheckerOffMax = ''; // Max value for opponent checker off
    let player2CheckerOffRangeMin = ''; // Min value for opponent checker off range
    let player2CheckerOffRangeMax = ''; // Max value for opponent checker off range

    let player1BackCheckerOption = 'min'; // Default option for player back checker
    let player1BackCheckerMin = ''; // Min value for player back checker
    let player1BackCheckerMax = ''; // Max value for player back checker
    let player1BackCheckerRangeMin = ''; // Min value for player back checker range
    let player1BackCheckerRangeMax = ''; // Max value for player back checker range

    let player2BackCheckerOption = 'min'; // Default option for opponent back checker
    let player2BackCheckerMin = ''; // Min value for opponent back checker
    let player2BackCheckerMax = ''; // Max value for opponent back checker
    let player2BackCheckerRangeMin = ''; // Min value for opponent back checker range
    let player2BackCheckerRangeMax = ''; // Max value for opponent back checker range

    let player1CheckerInZoneOption = 'min'; // Default option for player checker in zone
    let player1CheckerInZoneMin = ''; // Min value for player checker in zone
    let player1CheckerInZoneMax = ''; // Max value for player checker in zone
    let player1CheckerInZoneRangeMin = ''; // Min value for player checker in zone range
    let player1CheckerInZoneRangeMax = ''; // Max value for player checker in zone range

    let player2CheckerInZoneOption = 'min'; // Default option for opponent checker in zone
    let player2CheckerInZoneMin = ''; // Min value for opponent checker in zone
    let player2CheckerInZoneMax = ''; // Max value for opponent checker in zone
    let player2CheckerInZoneRangeMin = ''; // Min value for opponent checker in zone range
    let player2CheckerInZoneRangeMax = ''; // Max value for opponent checker in zone range

    let player1AbsolutePipCountOption = 'min'; // Default option for player absolute pip count
    let player1AbsolutePipCountMin = ''; // Min value for player absolute pip count
    let player1AbsolutePipCountMax = ''; // Max value for player absolute pip count
    let player1AbsolutePipCountRangeMin = ''; // Min value for player absolute pip count range
    let player1AbsolutePipCountRangeMax = ''; // Max value for player absolute pip count range

    let equityOption = 'min'; // Default option for equity
    let equityMin = ''; // Min value for equity
    let equityMax = ''; // Max value for equity
    let equityRangeMin = ''; // Min value for equity range
    let equityRangeMax = ''; // Max value for equity range

    let availableFilters = [
        'Include Cube',
        'Include Score',
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
        dispatch('search', {
            filters,
            includeCube,
            includeScore,
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
            searchText
        });
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
        pipCountMin = '';
        pipCountMax = '';
        pipCountRangeMin = '';
        pipCountRangeMax = '';
        winRateOption = 'min';
        winRateMin = '';
        winRateMax = '';
        winRateRangeMin = '';
        winRateRangeMax = '';
        gammonRateOption = 'min';
        gammonRateMin = '';
        gammonRateMax = '';
        gammonRateRangeMin = '';
        gammonRateRangeMax = '';
        backgammonRateOption = 'min';
        backgammonRateMin = '';
        backgammonRateMax = '';
        backgammonRateRangeMin = '';
        backgammonRateRangeMax = '';
        player2WinRateOption = 'min';
        player2WinRateMin = '';
        player2WinRateMax = '';
        player2WinRateRangeMin = '';
        player2WinRateRangeMax = '';
        player2GammonRateOption = 'min';
        player2GammonRateMin = '';
        player2GammonRateMax = '';
        player2GammonRateRangeMin = '';
        player2GammonRateRangeMax = '';
        player2BackgammonRateOption = 'min';
        player2BackgammonRateMin = '';
        player2BackgammonRateMax = '';
        player2BackgammonRateRangeMin = '';
        player2BackgammonRateRangeMax = '';
        player1CheckerOffOption = 'min';
        player1CheckerOffMin = '';
        player1CheckerOffMax = '';
        player1CheckerOffRangeMin = '';
        player1CheckerOffRangeMax = '';
        player2CheckerOffOption = 'min';
        player2CheckerOffMin = '';
        player2CheckerOffMax = '';
        player2CheckerOffRangeMin = '';
        player2CheckerOffRangeMax = '';
        player1BackCheckerOption = 'min';
        player1BackCheckerMin = '';
        player1BackCheckerMax = '';
        player1BackCheckerRangeMin = '';
        player1BackCheckerRangeMax = '';
        player2BackCheckerOption = 'min';
        player2BackCheckerMin = '';
        player2BackCheckerMax = '';
        player2BackCheckerRangeMin = '';
        player2BackCheckerRangeMax = '';
        player1CheckerInZoneOption = 'min';
        player1CheckerInZoneMin = '';
        player1CheckerInZoneMax = '';
        player1CheckerInZoneRangeMin = '';
        player1CheckerInZoneRangeMax = '';
        player2CheckerInZoneOption = 'min';
        player2CheckerInZoneMin = '';
        player2CheckerInZoneMax = '';
        player2CheckerInZoneRangeMin = '';
        player2CheckerInZoneRangeMax = '';
        player1AbsolutePipCountOption = 'min';
        player1AbsolutePipCountMin = '';
        player1AbsolutePipCountMax = '';
        player1AbsolutePipCountRangeMin = '';
        player1AbsolutePipCountRangeMax = '';
        equityOption = 'min';
        equityMin = '';
        equityMax = '';
        equityRangeMin = '';
        equityRangeMax = '';
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
                        {#if filter === 'Include Cube' || filter === 'Include Score'}
                            <!-- No input needed for these filters -->
                        {/if}
                        {#if filter === 'Pipcount Difference'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={pipCountOption} value="min" /> Min
                                        <input type="number" bind:value={pipCountMin} placeholder="Min" class="filter-input" on:input={e => e.target.value = e.target.value.replace(/\D/g, '')} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={pipCountOption} value="max" /> Max
                                        <input type="number" bind:value={pipCountMax} placeholder="Max" class="filter-input" on:input={e => e.target.value = e.target.value.replace(/\D/g, '')} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={pipCountOption} value="range" /> Range
                                        <input type="number" bind:value={pipCountRangeMin} placeholder="Min" class="filter-input" on:input={e => e.target.value = e.target.value.replace(/\D/g, '')} />
                                        <input type="number" bind:value={pipCountRangeMax} placeholder="Max" class="filter-input" on:input={e => e.target.value = e.target.value.replace(/\D/g, '')} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Player Absolute Pipcount'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1AbsolutePipCountOption} value="min" /> Min
                                        <input type="text" bind:value={player1AbsolutePipCountMin} placeholder="Min" class="filter-input" on:input={e => e.target.value = Math.max(0, Math.min(375, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1AbsolutePipCountOption} value="max" /> Max
                                        <input type="text" bind:value={player1AbsolutePipCountMax} placeholder="Max" class="filter-input" on:input={e => e.target.value = Math.max(0, Math.min(375, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1AbsolutePipCountOption} value="range" /> Range
                                        <input type="text" bind:value={player1AbsolutePipCountRangeMin} placeholder="Min" class="filter-input" on:input={e => e.target.value = Math.max(0, Math.min(375, e.target.value.replace(/\D/g, '')))} />
                                        <input type="text" bind:value={player1AbsolutePipCountRangeMax} placeholder="Max" class="filter-input" on:input={e => e.target.value = Math.max(0, Math.min(375, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Equity (millipoints)'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={equityOption} value="min" /> Min
                                        <input type="text" bind:value={equityMin} placeholder="Min" class="filter-input" on:input={e => e.target.value = e.target.value.replace(/\D/g, '')} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={equityOption} value="max" /> Max
                                        <input type="text" bind:value={equityMax} placeholder="Max" class="filter-input" on:input={e => e.target.value = e.target.value.replace(/\D/g, '')} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={equityOption} value="range" /> Range
                                        <input type="text" bind:value={equityRangeMin} placeholder="Min" class="filter-input" on:input={e => e.target.value = e.target.value.replace(/\D/g, '')} />
                                        <input type="text" bind:value={equityRangeMax} placeholder="Max" class="filter-input" on:input={e => e.target.value = e.target.value.replace(/\D/g, '')} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Win Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={winRateOption} value="min" /> Min
                                        <input type="number" bind:value={winRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={winRateOption} value="max" /> Max
                                        <input type="number" bind:value={winRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={winRateOption} value="range" /> Range
                                        <input type="number" bind:value={winRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={winRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Gammon Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={gammonRateOption} value="min" /> Min
                                        <input type="number" bind:value={gammonRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={gammonRateOption} value="max" /> Max
                                        <input type="number" bind:value={gammonRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={gammonRateOption} value="range" /> Range
                                        <input type="number" bind:value={gammonRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={gammonRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Backgammon Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={backgammonRateOption} value="min" /> Min
                                        <input type="number" bind:value={backgammonRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={backgammonRateOption} value="max" /> Max
                                        <input type="number" bind:value={backgammonRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={backgammonRateOption} value="range" /> Range
                                        <input type="number" bind:value={backgammonRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={backgammonRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Win Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2WinRateOption} value="min" /> Min
                                        <input type="number" bind:value={player2WinRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2WinRateOption} value="max" /> Max
                                        <input type="number" bind:value={player2WinRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2WinRateOption} value="range" /> Range
                                        <input type="number" bind:value={player2WinRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={player2WinRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Gammon Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2GammonRateOption} value="min" /> Min
                                        <input type="number" bind:value={player2GammonRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2GammonRateOption} value="max" /> Max
                                        <input type="number" bind:value={player2GammonRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2GammonRateOption} value="range" /> Range
                                        <input type="number" bind:value={player2GammonRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={player2GammonRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Backgammon Rate'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackgammonRateOption} value="min" /> Min
                                        <input type="number" bind:value={player2BackgammonRateMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackgammonRateOption} value="max" /> Max
                                        <input type="number" bind:value={player2BackgammonRateMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackgammonRateOption} value="range" /> Range
                                        <input type="number" bind:value={player2BackgammonRateRangeMin} placeholder="Min" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={player2BackgammonRateRangeMax} placeholder="Max" class="filter-input" min="0" max="100" on:input={e => e.target.value = Math.max(0, Math.min(100, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Player Checker-Off'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerOffOption} value="min" /> Min
                                        <input type="number" bind:value={player1CheckerOffMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerOffOption} value="max" /> Max
                                        <input type="number" bind:value={player1CheckerOffMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerOffOption} value="range" /> Range
                                        <input type="number" bind:value={player1CheckerOffRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={player1CheckerOffRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Checker-Off'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerOffOption} value="min" /> Min
                                        <input type="number" bind:value={player2CheckerOffMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerOffOption} value="max" /> Max
                                        <input type="number" bind:value={player2CheckerOffMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerOffOption} value="range" /> Range
                                        <input type="number" bind:value={player2CheckerOffRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={player2CheckerOffRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Player Back Checker'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1BackCheckerOption} value="min" /> Min
                                        <input type="number" bind:value={player1BackCheckerMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1BackCheckerOption} value="max" /> Max
                                        <input type="number" bind:value={player1BackCheckerMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1BackCheckerOption} value="range" /> Range
                                        <input type="number" bind:value={player1BackCheckerRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={player1BackCheckerRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Back Checker'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackCheckerOption} value="min" /> Min
                                        <input type="number" bind:value={player2BackCheckerMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackCheckerOption} value="max" /> Max
                                        <input type="number" bind:value={player2BackCheckerMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2BackCheckerOption} value="range" /> Range
                                        <input type="number" bind:value={player2BackCheckerRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={player2BackCheckerRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Player Checker in the Zone'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerInZoneOption} value="min" /> Min
                                        <input type="number" bind:value={player1CheckerInZoneMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerInZoneOption} value="max" /> Max
                                        <input type="number" bind:value={player1CheckerInZoneMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player1CheckerInZoneOption} value="range" /> Range
                                        <input type="number" bind:value={player1CheckerInZoneRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={player1CheckerInZoneRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                </div>
                            </div>
                        {/if}
                        {#if filter === 'Opponent Checker in the Zone'}
                            <div class="filter-options-container expanded">
                                <div class="filter-options expanded">
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerInZoneOption} value="min" /> Min
                                        <input type="number" bind:value={player2CheckerInZoneMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerInZoneOption} value="max" /> Max
                                        <input type="number" bind:value={player2CheckerInZoneMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                    </label>
                                    <label class="filter-option">
                                        <input type="radio" bind:group={player2CheckerInZoneOption} value="range" /> Range
                                        <input type="number" bind:value={player2CheckerInZoneRangeMin} placeholder="Min" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
                                        <input type="number" bind:value={player2CheckerInZoneRangeMax} placeholder="Max" class="filter-input" min="0" max="15" on:input={e => e.target.value = Math.max(0, Math.min(15, e.target.value.replace(/\D/g, '')))} />
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
        width: 250px; /* Set a fixed width for the container */
        height: 50px; /* Set a fixed height for the container */
        display: flex;
        align-items: center;
    }

    .filter-label {
        flex: 1;
        margin-bottom: 5px;
        font-size: 18px; /* Set font size */
        text-align: left; /* Align text to the left */
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
        justify-content: center;
    }

    .search-text-input {
        width: 100%;
        max-width: 1200px; /* Increase the width of the filter input for "Search Text" */
    }

    .add-button, .remove-button {
        background: none;
        border: none;
        font-size: 1.5rem;
        cursor: pointer;
        color: #6c757d;
        width: 50px; /* Set a fixed width for remove button */
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
</style>