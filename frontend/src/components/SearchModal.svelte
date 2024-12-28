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

    let player2WinRateOption = 'min'; // Default option for player 2 win rate
    let player2WinRateMin = ''; // Min value for player 2 win rate
    let player2WinRateMax = ''; // Max value for player 2 win rate
    let player2WinRateRangeMin = ''; // Min value for player 2 win rate range
    let player2WinRateRangeMax = ''; // Max value for player 2 win rate range

    let player2GammonRateOption = 'min'; // Default option for player 2 gammon rate
    let player2GammonRateMin = ''; // Min value for player 2 gammon rate
    let player2GammonRateMax = ''; // Max value for player 2 gammon rate
    let player2GammonRateRangeMin = ''; // Min value for player 2 gammon rate range
    let player2GammonRateRangeMax = ''; // Max value for player 2 gammon rate range

    let player2BackgammonRateOption = 'min'; // Default option for player 2 backgammon rate
    let player2BackgammonRateMin = ''; // Min value for player 2 backgammon rate
    let player2BackgammonRateMax = ''; // Max value for player 2 backgammon rate
    let player2BackgammonRateRangeMin = ''; // Min value for player 2 backgammon rate range
    let player2BackgammonRateRangeMax = ''; // Max value for player 2 backgammon rate range

    let player1CheckerOffOption = 'min'; // Default option for player 1 checker off
    let player1CheckerOffMin = ''; // Min value for player 1 checker off
    let player1CheckerOffMax = ''; // Max value for player 1 checker off
    let player1CheckerOffRangeMin = ''; // Min value for player 1 checker off range
    let player1CheckerOffRangeMax = ''; // Max value for player 1 checker off range

    let player2CheckerOffOption = 'min'; // Default option for player 2 checker off
    let player2CheckerOffMin = ''; // Min value for player 2 checker off
    let player2CheckerOffMax = ''; // Max value for player 2 checker off
    let player2CheckerOffRangeMin = ''; // Min value for player 2 checker off range
    let player2CheckerOffRangeMax = ''; // Max value for player 2 checker off range

    let player1BackCheckerOption = 'min'; // Default option for player 1 back checker
    let player1BackCheckerMin = ''; // Min value for player 1 back checker
    let player1BackCheckerMax = ''; // Max value for player 1 back checker
    let player1BackCheckerRangeMin = ''; // Min value for player 1 back checker range
    let player1BackCheckerRangeMax = ''; // Max value for player 1 back checker range

    let player2BackCheckerOption = 'min'; // Default option for player 2 back checker
    let player2BackCheckerMin = ''; // Min value for player 2 back checker
    let player2BackCheckerMax = ''; // Max value for player 2 back checker
    let player2BackCheckerRangeMin = ''; // Min value for player 2 back checker range
    let player2BackCheckerRangeMax = ''; // Max value for player 2 back checker range

    let player1CheckerInZoneOption = 'min'; // Default option for player 1 checker in zone
    let player1CheckerInZoneMin = ''; // Min value for player 1 checker in zone
    let player1CheckerInZoneMax = ''; // Max value for player 1 checker in zone
    let player1CheckerInZoneRangeMin = ''; // Min value for player 1 checker in zone range
    let player1CheckerInZoneRangeMax = ''; // Max value for player 1 checker in zone range

    let player2CheckerInZoneOption = 'min'; // Default option for player 2 checker in zone
    let player2CheckerInZoneMin = ''; // Min value for player 2 checker in zone
    let player2CheckerInZoneMax = ''; // Max value for player 2 checker in zone
    let player2CheckerInZoneRangeMin = ''; // Min value for player 2 checker in zone range
    let player2CheckerInZoneRangeMax = ''; // Max value for player 2 checker in zone range

    let player1AbsolutePipCountOption = 'min'; // Default option for player 1 absolute pip count
    let player1AbsolutePipCountMin = ''; // Min value for player 1 absolute pip count
    let player1AbsolutePipCountMax = ''; // Max value for player 1 absolute pip count
    let player1AbsolutePipCountRangeMin = ''; // Min value for player 1 absolute pip count range
    let player1AbsolutePipCountRangeMax = ''; // Max value for player 1 absolute pip count range

    let equityOption = 'min'; // Default option for equity
    let equityMin = ''; // Min value for equity
    let equityMax = ''; // Max value for equity
    let equityRangeMin = ''; // Min value for equity range
    let equityRangeMax = ''; // Max value for equity range

    let availableFilters = [
        'Include Cube',
        'Include Score',
        'Pip Count',
        'Win Rate Filter',
        'Gammon Rate Filter',
        'Backgammon Rate Filter',
        'Player 2 Win Rate Filter',
        'Player 2 Gammon Rate Filter',
        'Player 2 Backgammon Rate Filter',
        'Player 1 Checker Off Filter',
        'Player 2 Checker Off Filter',
        'Player 1 Back Checker Filter',
        'Player 2 Back Checker Filter',
        'Player 1 Checker In Zone Filter',
        'Player 2 Checker In Zone Filter',
        'Search Text',
        'Player 1 Absolute Pip Count Filter',
        'Equity Filter'
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
        <div class="modal-header">
            <button class="close-button" on:click={onClose}>×</button>
        </div>
        <div class="modal-body">
            <div class="form-group">
                <label>Select Filter</label>
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
                    <label>{filter}</label>
                    {#if filter === 'Include Cube' || filter === 'Include Score'}
                        <!-- No input needed for these filters -->
                    {/if}
                    {#if filter === 'Pip Count'}
                        <div class="pip-count-options">
                            <label class="pip-count-option">
                                <input type="radio" bind:group={pipCountOption} value="min" /> Min
                                <input type="number" bind:value={pipCountMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="pip-count-option">
                                <input type="radio" bind:group={pipCountOption} value="max" /> Max
                                <input type="number" bind:value={pipCountMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="pip-count-option">
                                <input type="radio" bind:group={pipCountOption} value="range" /> Range
                                <input type="number" bind:value={pipCountRangeMin} placeholder="Min" class="filter-input" />
                                <input type="number" bind:value={pipCountRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Win Rate Filter'}
                        <div class="win-rate-options">
                            <label class="win-rate-option">
                                <input type="radio" bind:group={winRateOption} value="min" /> Min
                                <input type="text" bind:value={winRateMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="win-rate-option">
                                <input type="radio" bind:group={winRateOption} value="max" /> Max
                                <input type="text" bind:value={winRateMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="win-rate-option">
                                <input type="radio" bind:group={winRateOption} value="range" /> Range
                                <input type="text" bind:value={winRateRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={winRateRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Gammon Rate Filter'}
                        <div class="gammon-rate-options">
                            <label class="gammon-rate-option">
                                <input type="radio" bind:group={gammonRateOption} value="min" /> Min
                                <input type="text" bind:value={gammonRateMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="gammon-rate-option">
                                <input type="radio" bind:group={gammonRateOption} value="max" /> Max
                                <input type="text" bind:value={gammonRateMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="gammon-rate-option">
                                <input type="radio" bind:group={gammonRateOption} value="range" /> Range
                                <input type="text" bind:value={gammonRateRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={gammonRateRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Backgammon Rate Filter'}
                        <div class="backgammon-rate-options">
                            <label class="backgammon-rate-option">
                                <input type="radio" bind:group={backgammonRateOption} value="min" /> Min
                                <input type="text" bind:value={backgammonRateMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="backgammon-rate-option">
                                <input type="radio" bind:group={backgammonRateOption} value="max" /> Max
                                <input type="text" bind:value={backgammonRateMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="backgammon-rate-option">
                                <input type="radio" bind:group={backgammonRateOption} value="range" /> Range
                                <input type="text" bind:value={backgammonRateRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={backgammonRateRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Player 2 Win Rate Filter'}
                        <div class="player2-win-rate-options">
                            <label class="player2-win-rate-option">
                                <input type="radio" bind:group={player2WinRateOption} value="min" /> Min
                                <input type="text" bind:value={player2WinRateMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="player2-win-rate-option">
                                <input type="radio" bind:group={player2WinRateOption} value="max" /> Max
                                <input type="text" bind:value={player2WinRateMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="player2-win-rate-option">
                                <input type="radio" bind:group={player2WinRateOption} value="range" /> Range
                                <input type="text" bind:value={player2WinRateRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={player2WinRateRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Player 2 Gammon Rate Filter'}
                        <div class="player2-gammon-rate-options">
                            <label class="player2-gammon-rate-option">
                                <input type="radio" bind:group={player2GammonRateOption} value="min" /> Min
                                <input type="text" bind:value={player2GammonRateMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="player2-gammon-rate-option">
                                <input type="radio" bind:group={player2GammonRateOption} value="max" /> Max
                                <input type="text" bind:value={player2GammonRateMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="player2-gammon-rate-option">
                                <input type="radio" bind:group={player2GammonRateOption} value="range" /> Range
                                <input type="text" bind:value={player2GammonRateRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={player2GammonRateRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Player 2 Backgammon Rate Filter'}
                        <div class="player2-backgammon-rate-options">
                            <label class="player2-backgammon-rate-option">
                                <input type="radio" bind:group={player2BackgammonRateOption} value="min" /> Min
                                <input type="text" bind:value={player2BackgammonRateMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="player2-backgammon-rate-option">
                                <input type="radio" bind:group={player2BackgammonRateOption} value="max" /> Max
                                <input type="text" bind:value={player2BackgammonRateMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="player2-backgammon-rate-option">
                                <input type="radio" bind:group={player2BackgammonRateOption} value="range" /> Range
                                <input type="text" bind:value={player2BackgammonRateRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={player2BackgammonRateRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Player 1 Checker Off Filter'}
                        <div class="player1-checker-off-options">
                            <label class="player1-checker-off-option">
                                <input type="radio" bind:group={player1CheckerOffOption} value="min" /> Min
                                <input type="text" bind:value={player1CheckerOffMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="player1-checker-off-option">
                                <input type="radio" bind:group={player1CheckerOffOption} value="max" /> Max
                                <input type="text" bind:value={player1CheckerOffMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="player1-checker-off-option">
                                <input type="radio" bind:group={player1CheckerOffOption} value="range" /> Range
                                <input type="text" bind:value={player1CheckerOffRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={player1CheckerOffRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Player 2 Checker Off Filter'}
                        <div class="player2-checker-off-options">
                            <label class="player2-checker-off-option">
                                <input type="radio" bind:group={player2CheckerOffOption} value="min" /> Min
                                <input type="text" bind:value={player2CheckerOffMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="player2-checker-off-option">
                                <input type="radio" bind:group={player2CheckerOffOption} value="max" /> Max
                                <input type="text" bind:value={player2CheckerOffMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="player2-checker-off-option">
                                <input type="radio" bind:group={player2CheckerOffOption} value="range" /> Range
                                <input type="text" bind:value={player2CheckerOffRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={player2CheckerOffRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Player 1 Back Checker Filter'}
                        <div class="player1-back-checker-options">
                            <label class="player1-back-checker-option">
                                <input type="radio" bind:group={player1BackCheckerOption} value="min" /> Min
                                <input type="text" bind:value={player1BackCheckerMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="player1-back-checker-option">
                                <input type="radio" bind:group={player1BackCheckerOption} value="max" /> Max
                                <input type="text" bind:value={player1BackCheckerMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="player1-back-checker-option">
                                <input type="radio" bind:group={player1BackCheckerOption} value="range" /> Range
                                <input type="text" bind:value={player1BackCheckerRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={player1BackCheckerRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Player 2 Back Checker Filter'}
                        <div class="player2-back-checker-options">
                            <label class="player2-back-checker-option">
                                <input type="radio" bind:group={player2BackCheckerOption} value="min" /> Min
                                <input type="text" bind:value={player2BackCheckerMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="player2-back-checker-option">
                                <input type="radio" bind:group={player2BackCheckerOption} value="max" /> Max
                                <input type="text" bind:value={player2BackCheckerMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="player2-back-checker-option">
                                <input type="radio" bind:group={player2BackCheckerOption} value="range" /> Range
                                <input type="text" bind:value={player2BackCheckerRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={player2BackCheckerRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Player 1 Checker In Zone Filter'}
                        <div class="player1-checker-in-zone-options">
                            <label class="player1-checker-in-zone-option">
                                <input type="radio" bind:group={player1CheckerInZoneOption} value="min" /> Min
                                <input type="text" bind:value={player1CheckerInZoneMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="player1-checker-in-zone-option">
                                <input type="radio" bind:group={player1CheckerInZoneOption} value="max" /> Max
                                <input type="text" bind:value={player1CheckerInZoneMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="player1-checker-in-zone-option">
                                <input type="radio" bind:group={player1CheckerInZoneOption} value="range" /> Range
                                <input type="text" bind:value={player1CheckerInZoneRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={player1CheckerInZoneRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Player 2 Checker In Zone Filter'}
                        <div class="player2-checker-in-zone-options">
                            <label class="player2-checker-in-zone-option">
                                <input type="radio" bind:group={player2CheckerInZoneOption} value="min" /> Min
                                <input type="text" bind:value={player2CheckerInZoneMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="player2-checker-in-zone-option">
                                <input type="radio" bind:group={player2CheckerInZoneOption} value="max" /> Max
                                <input type="text" bind:value={player2CheckerInZoneMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="player2-checker-in-zone-option">
                                <input type="radio" bind:group={player2CheckerInZoneOption} value="range" /> Range
                                <input type="text" bind:value={player2CheckerInZoneRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={player2CheckerInZoneRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Search Text'}
                        <input type="text" bind:value={searchText} class="filter-input" />
                    {/if}
                    {#if filter === 'Player 1 Absolute Pip Count Filter'}
                        <div class="player1-absolute-pip-count-options">
                            <label class="player1-absolute-pip-count-option">
                                <input type="radio" bind:group={player1AbsolutePipCountOption} value="min" /> Min
                                <input type="text" bind:value={player1AbsolutePipCountMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="player1-absolute-pip-count-option">
                                <input type="radio" bind:group={player1AbsolutePipCountOption} value="max" /> Max
                                <input type="text" bind:value={player1AbsolutePipCountMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="player1-absolute-pip-count-option">
                                <input type="radio" bind:group={player1AbsolutePipCountOption} value="range" /> Range
                                <input type="text" bind:value={player1AbsolutePipCountRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={player1AbsolutePipCountRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
                    {#if filter === 'Equity Filter'}
                        <div class="equity-options">
                            <label class="equity-option">
                                <input type="radio" bind:group={equityOption} value="min" /> Min
                                <input type="text" bind:value={equityMin} placeholder="Min" class="filter-input" />
                            </label>
                            <label class="equity-option">
                                <input type="radio" bind:group={equityOption} value="max" /> Max
                                <input type="text" bind:value={equityMax} placeholder="Max" class="filter-input" />
                            </label>
                            <label class="equity-option">
                                <input type="radio" bind:group={equityOption} value="range" /> Range
                                <input type="text" bind:value={equityRangeMin} placeholder="Min" class="filter-input" />
                                <input type="text" bind:value={equityRangeMax} placeholder="Max" class="filter-input" />
                            </label>
                        </div>
                    {/if}
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
    }

    .modal-header {
        display: flex;
        justify-content: flex-end;
        align-items: center;
        border-bottom: 1px solid #ddd;
        padding-bottom: 0.5rem;
        margin-bottom: 1rem;
    }

    .close-button {
        background: none;
        border: none;
        font-size: 1.5rem;
        cursor: pointer;
    }

    .modal-body {
        padding: 1rem 0;
        display: flex;
        flex-direction: column;
        gap: 10px; /* Add space between filters */
    }

    .form-group {
        display: flex;
        align-items: center;
        margin-bottom: 10px;
    }

    label {
        flex: 1;
        margin-bottom: 5px;
    }

    input[type="text"], input[type="number"], select {
        flex: 2;
        margin-right: 10px;
        font-size: 14px; /* Set font size */
    }

    .filter-dropdown {
            font-size: 14px; /* Reduce font size */
    }

    .filter-input {
        font-size: 18px; /* Set font size */
    }

    .add-button, .remove-button {
        background: none;
        border: none;
        font-size: 1.5rem;
        cursor: pointer;
        color: #6c757d;
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

    .pip-count-options, .win-rate-options, .gammon-rate-options, .backgammon-rate-options, .player2-win-rate-options, .player2-gammon-rate-options, .player2-backgammon-rate-options, .player1-checker-off-options, .player2-checker-off-options, .player1-back-checker-options, .player2-back-checker-options, .player1-checker-in-zone-options, .player2-checker-in-zone-options, .player1-absolute-pip-count-options, .equity-options {
        display: flex;
        flex-direction: column;
        gap: 5px;
    }

    .pip-count-option, .win-rate-option, .gammon-rate-option, .backgammon-rate-option, .player2-win-rate-option, .player2-gammon-rate-option, .player2-backgammon-rate-option, .player1-checker-off-option, .player2-checker-off-option, .player1-back-checker-option, .player2-back-checker-option, .player1-checker-in-zone-option, .player2-checker-in-zone-option, .player1-absolute-pip-count-option, .equity-option {
        display: flex;
        align-items: center;
        gap: 10px; /* Add space between radio button and input */
    }
</style>