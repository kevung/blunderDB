<script>
    import { SvelteMap, SvelteSet } from 'svelte/reactivity';
    import Modal from './Modal.svelte';
    import PickList from './PickList.svelte';
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
</script>

<Modal open={visible} onclose={onCancel} size="wide" layer="top" closeButton={false}>
    {#snippet title()}{$t('search.pickerTitle')}{/snippet}
    <PickList
        header={$t('search.pickerMatchesHeader')}
        items={filteredMatches}
        isChecked={isMatchChecked}
        isDisabled={isMatchDisabled}
        toggle={toggleMatch}
        selectAll={selectAllMatches}
        selectNone={selectNoMatches}
        describe={(match) => ({ name: `${match.player1_name} vs ${match.player2_name}`, count: `${match.tournament_name || ''} ${match.match_date ? formatDate(match.match_date) : ''}`.trim() })}
        bind:filterValue={matchFilterText}
        filterPlaceholder={$t('search.pickerFilterPlaceholder')}
    />

    <PickList
        header={$t('search.pickerTournamentsHeader')}
        items={filteredTournaments}
        isChecked={(id) => localTournamentIDs.includes(id)}
        toggle={toggleTournament}
        selectAll={selectAllTournaments}
        selectNone={selectNoTournaments}
        describe={(tournament) => ({ name: tournament.name, count: `(${tournament.matchCount}) ${tournament.date || ''} ${tournament.location || ''}`.trim() })}
        bind:filterValue={tournamentFilterText}
        filterPlaceholder={$t('search.pickerFilterPlaceholder')}
    />
    {#snippet footer()}
        <button onclick={onCancel}>{$t('common.cancel')}</button>
        <button class="primary" onclick={handleApply}>{$t('common.apply')}</button>
    {/snippet}
</Modal>
