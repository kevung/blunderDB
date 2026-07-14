<script>
    import { onMount, onDestroy } from 'svelte';
    import { SvelteMap, SvelteSet } from 'svelte/reactivity';
    import { trapFocus } from '../utils/focusTrap.js';
    import { t } from '../i18n';
    import { formatDate } from '../utils/matchTable.js';
    import { GetAllMatches, GetAllTournaments } from '../../wailsjs/go/database/Database.js';

    let { visible = false, matchIDsSelected = [], tournamentIDsSelected = [], onApply, onCancel } = $props();

    let matches = $state([]);
    let tournaments = $state([]);
    let matchFilterText = $state('');
    let tournamentFilterText = $state('');
    let localMatchIDs = $state([]);
    let localTournamentIDs = $state([]);

    // Fetch a fresh match/tournament list on every open, pruning any
    // previously selected ID that no longer exists (a deleted match/tournament).
    $effect(() => {
        if (!visible) return;
        (async () => {
            try {
                matches = (await GetAllMatches()) || [];
            } catch {
                matches = [];
            }
            try {
                tournaments = (await GetAllTournaments()) || [];
            } catch {
                tournaments = [];
            }
            const validMatchIDs = new Set(matches.map((m) => m.id));
            const validTournamentIDs = new Set(tournaments.map((t2) => t2.id));
            localMatchIDs = matchIDsSelected.filter((id) => validMatchIDs.has(id));
            localTournamentIDs = tournamentIDsSelected.filter((id) => validTournamentIDs.has(id));
        })();
    });

    let filteredMatches = $derived.by(() => {
        if (!matchFilterText) return matches;
        const needle = matchFilterText.toLowerCase();
        return matches.filter((m) => `${m.player1_name} ${m.player2_name} ${m.event || ''} ${m.tournament_name || ''}`.toLowerCase().includes(needle));
    });

    let filteredTournaments = $derived.by(() => {
        if (!tournamentFilterText) return tournaments;
        const needle = tournamentFilterText.toLowerCase();
        return tournaments.filter((t2) => `${t2.name} ${t2.location || ''}`.toLowerCase().includes(needle));
    });

    // A match belongs to a checked tournament -> it is implicitly selected
    // (and its checkbox is locked) until the tournament is unchecked.
    let matchIDsByTournament = $derived.by(() => {
        const map = new SvelteMap();
        for (const m of matches) {
            if (m.tournament_id != null) {
                if (!map.has(m.tournament_id)) map.set(m.tournament_id, new SvelteSet());
                map.get(m.tournament_id).add(m.id);
            }
        }
        return map;
    });

    let impliedMatchIDs = $derived.by(() => {
        const s = new SvelteSet();
        for (const tid of localTournamentIDs) {
            for (const mid of matchIDsByTournament.get(tid) ?? []) s.add(mid);
        }
        return s;
    });

    function isMatchChecked(matchId) {
        return localMatchIDs.includes(matchId) || impliedMatchIDs.has(matchId);
    }

    function isMatchDisabled(matchId) {
        return impliedMatchIDs.has(matchId) && !localMatchIDs.includes(matchId);
    }

    function toggleMatch(matchId) {
        if (isMatchDisabled(matchId)) return;
        localMatchIDs = localMatchIDs.includes(matchId) ? localMatchIDs.filter((id) => id !== matchId) : [...localMatchIDs, matchId];
    }

    function selectAllMatches() {
        localMatchIDs = [...new Set([...localMatchIDs, ...filteredMatches.map((m) => m.id)])];
    }

    function selectNoMatches() {
        const visibleIDs = new Set(filteredMatches.map((m) => m.id));
        localMatchIDs = localMatchIDs.filter((id) => !visibleIDs.has(id));
    }

    function toggleTournament(tournamentId) {
        localTournamentIDs = localTournamentIDs.includes(tournamentId) ? localTournamentIDs.filter((id) => id !== tournamentId) : [...localTournamentIDs, tournamentId];
    }

    function selectAllTournaments() {
        localTournamentIDs = [...new Set([...localTournamentIDs, ...filteredTournaments.map((t2) => t2.id)])];
    }

    function selectNoTournaments() {
        const visibleIDs = new Set(filteredTournaments.map((t2) => t2.id));
        localTournamentIDs = localTournamentIDs.filter((id) => !visibleIDs.has(id));
    }

    function handleApply() {
        onApply(localMatchIDs, localTournamentIDs);
    }

    function handleKeyDown(event) {
        if (event.key === 'Escape' && visible) {
            onCancel();
        }
    }

    onMount(() => {
        window.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        window.removeEventListener('keydown', handleKeyDown);
    });
</script>

{#if visible}
    <div class="modal-overlay" role="dialog" aria-modal="true" aria-label={$t('search.pickerDialogLabel')} use:trapFocus>
        <div class="modal-content picker-modal-content">
            <h2>{$t('search.pickerTitle')}</h2>

            <section class="collections-section">
                <div class="collections-header">
                    <span>{$t('search.pickerMatchesHeader')}</span>
                    <input type="text" bind:value={matchFilterText} placeholder={$t('search.pickerFilterPlaceholder')} class="filter-input" />
                    <div class="collections-buttons">
                        <button type="button" class="small-btn" onclick={selectAllMatches}>{$t('export.all')}</button>
                        <button type="button" class="small-btn" onclick={selectNoMatches}>{$t('export.none')}</button>
                    </div>
                </div>
                <div class="collections-list">
                    {#each filteredMatches as match (match.id)}
                        <label class="collection-checkbox" class:disabled={isMatchDisabled(match.id)}>
                            <input type="checkbox" checked={isMatchChecked(match.id)} disabled={isMatchDisabled(match.id)} onchange={() => toggleMatch(match.id)} />
                            <span class="coll-name">{match.player1_name} vs {match.player2_name}</span>
                            <span class="coll-count">{match.tournament_name || ''} {match.match_date ? formatDate(match.match_date) : ''}</span>
                        </label>
                    {/each}
                </div>
            </section>

            <section class="collections-section">
                <div class="collections-header">
                    <span>{$t('search.pickerTournamentsHeader')}</span>
                    <input type="text" bind:value={tournamentFilterText} placeholder={$t('search.pickerFilterPlaceholder')} class="filter-input" />
                    <div class="collections-buttons">
                        <button type="button" class="small-btn" onclick={selectAllTournaments}>{$t('export.all')}</button>
                        <button type="button" class="small-btn" onclick={selectNoTournaments}>{$t('export.none')}</button>
                    </div>
                </div>
                <div class="collections-list">
                    {#each filteredTournaments as tournament (tournament.id)}
                        <label class="collection-checkbox">
                            <input type="checkbox" checked={localTournamentIDs.includes(tournament.id)} onchange={() => toggleTournament(tournament.id)} />
                            <span class="coll-name">{tournament.name}</span>
                            <span class="coll-count">({tournament.matchCount}) {tournament.date || ''} {tournament.location || ''}</span>
                        </label>
                    {/each}
                </div>
            </section>

            <div class="button-group">
                <button onclick={onCancel}>{$t('common.cancel')}</button>
                <button class="btn-export" onclick={handleApply}>{$t('common.apply')}</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.7);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 2000;
    }

    .modal-content {
        background-color: white;
        padding: 30px;
        border-radius: 8px;
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
        display: flex;
        flex-direction: column;
        gap: 20px;
    }

    .picker-modal-content {
        width: min(90vw, 900px);
        max-height: 85vh;
        overflow-y: auto;
    }

    h2 {
        margin: 0;
        font-size: 20px;
        color: #333;
    }

    .collections-section {
        border: 1px solid #ddd;
        border-radius: 4px;
        overflow: hidden;
    }

    .collections-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        background-color: #f9f9f9;
        border-bottom: 1px solid #ddd;
        font-size: 13px;
        font-weight: 500;
    }

    .filter-input {
        flex: 1;
        padding: 4px 8px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 13px;
        font-family: inherit;
        min-width: 0;
    }

    .collections-buttons {
        display: flex;
        gap: 4px;
    }

    .small-btn {
        font-size: 11px;
        padding: 2px 8px;
    }

    .collections-list {
        max-height: 260px;
        overflow-y: auto;
        padding: 4px;
    }

    .collection-checkbox {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 8px;
        cursor: pointer;
        font-size: 13px;
    }

    .collection-checkbox:hover {
        background-color: #f5f5f5;
    }

    .collection-checkbox.disabled {
        opacity: 0.6;
    }

    .collection-checkbox input[type='checkbox'] {
        width: 16px;
        height: 16px;
        cursor: pointer;
        accent-color: #333;
    }

    .collection-checkbox input[type='checkbox']:disabled {
        cursor: not-allowed;
    }

    .coll-name {
        flex: 1;
    }

    .coll-count {
        color: #888;
        font-size: 12px;
    }

    .button-group {
        display: flex;
        gap: 10px;
        justify-content: flex-end;
        margin-top: 10px;
    }

    button {
        padding: 10px 20px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s ease;
        background-color: white;
        color: #333;
    }

    button:hover:not(:disabled) {
        background-color: #f5f5f5;
        border-color: #999;
    }

    .btn-export {
        background-color: #333;
        color: white;
        border-color: #333;
    }

    .btn-export:hover:not(:disabled) {
        background-color: #555;
        border-color: #555;
    }
</style>
