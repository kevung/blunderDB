<script>
    // Compact header strip shown directly above the board. While reviewing a
    // match it shows that match (players, event, round, date, length + tournament).
    // Outside match mode — search results, collection, go-to — it shows the
    // *provenance* of the position being studied: the match(es) it came from,
    // resolved via GetPositionProvenance. A position no match references (e.g. one
    // imported on its own) shows nothing, so the board stays uncluttered. Because
    // positions dedupe across imports, provenance can be one-to-many: the first
    // match is shown and a "+N" badge lists the rest.
    import { matchContextStore, positionStore } from '../stores/positionStore';
    import { GetMatchByID, GetPositionProvenance } from '../../wailsjs/go/database/Database.js';
    import { boardColorsStore } from '../stores/boardColorsStore';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';

    let match = $state(null);
    // Other matches this position also came from (provenance is one-to-many).
    let otherMatches = $state([]);
    // Cache key: 'm<id>' in match mode, 'p<id>' for a studied position's provenance.
    let loadedKey = $state(null);

    // In match mode: fetch the reviewed match by id. Outside: fetch the studied
    // position's provenance. Cached by key so navigating within a match (matchID
    // stable) or redrawing doesn't refetch.
    $effect(() => {
        const ctx = $matchContextStore;
        if (ctx.isMatchMode && ctx.matchID) {
            const key = 'm' + ctx.matchID;
            if (loadedKey === key) return;
            const wantID = ctx.matchID;
            GetMatchByID(wantID)
                .then((m) => {
                    if ($matchContextStore.matchID === wantID) {
                        match = m;
                        otherMatches = [];
                        loadedKey = key;
                    }
                })
                .catch((e) => {
                    logger.error('MatchInfoBar: failed to load match', e);
                    match = null;
                });
            return;
        }

        // Provenance of the studied position (only real, saved positions).
        const posID = $positionStore?.id;
        if (!posID || posID <= 0) {
            match = null;
            otherMatches = [];
            loadedKey = null;
            return;
        }
        const key = 'p' + posID;
        if (loadedKey === key) return;
        const wantID = posID;
        GetPositionProvenance(wantID)
            .then((matches) => {
                // Ignore a stale response if the user navigated on, or entered a match.
                if ($positionStore?.id !== wantID || $matchContextStore.isMatchMode) return;
                match = matches && matches.length > 0 ? matches[0] : null;
                otherMatches = matches && matches.length > 1 ? matches.slice(1) : [];
                loadedKey = key;
            })
            .catch((e) => {
                logger.error('MatchInfoBar: failed to load provenance', e);
                match = null;
                otherMatches = [];
            });
    });

    // Prefer the live names from the context store (always present) and fall
    // back to the fetched match.
    let player1Name = $derived($matchContextStore.player1Name || match?.player1_name || '');
    let player2Name = $derived($matchContextStore.player2Name || match?.player2_name || '');

    function formatDate(value) {
        if (!value) return '';
        const d = new Date(value);
        // Guard against Go's zero time (year 0001) and invalid dates.
        if (isNaN(d.getTime()) || d.getFullYear() < 1900) return '';
        const [year, month, day] = d.toLocaleDateString('sv-SE').split('-');
        return `${year}/${month}/${day}`;
    }

    // Optional metadata fields, in display order, empties omitted. Tournament
    // name leads (it is the broadest provenance), then event/location/round/date/length.
    let metaParts = $derived(
        [
            match?.tournament_name,
            match?.event,
            match?.location,
            match?.round,
            formatDate(match?.match_date),
            match?.match_length > 0 ? `${match.match_length}${$t('matchInfo.points')}` : ''
        ]
            .map((s) => (s == null ? '' : String(s).trim()))
            .filter((s) => s.length > 0)
    );

    // Tooltip listing the other matches the (deduplicated) position came from.
    // Data only — player names — so it needs no translation.
    let otherMatchesTitle = $derived(otherMatches.map((m) => `${m.player1_name} ${$t('matchInfo.vs')} ${m.player2_name}`).join('\n'));

    // Visible in match mode, or whenever a studied position resolves to a match.
    let visible = $derived(($matchContextStore.isMatchMode && !!$matchContextStore.matchID) || !!match);
</script>

{#if visible}
    <div class="match-info-bar" data-testid="match-info-bar">
        <span class="player">
            <span class="disc" style="background:{$boardColorsStore.checker1};"></span>
            <span class="name">{player1Name}</span>
        </span>
        <span class="vs">{$t('matchInfo.vs')}</span>
        <span class="player">
            <span class="disc" style="background:{$boardColorsStore.checker2};"></span>
            <span class="name">{player2Name}</span>
        </span>
        {#if metaParts.length > 0}
            <span class="sep">·</span>
            <span class="meta">{metaParts.join(' · ')}</span>
        {/if}
        {#if otherMatches.length > 0}
            <span class="more" title={otherMatchesTitle}>+{otherMatches.length}</span>
        {/if}
    </div>
{/if}

<style>
    .match-info-bar {
        display: flex;
        align-items: center;
        gap: 6px;
        width: 100%;
        box-sizing: border-box;
        padding: 2px 10px;
        height: 22px;
        flex-shrink: 0;
        background: #f7f7f7;
        border-bottom: 1px solid #e0e0e0;
        font-size: 12px;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans JP', sans-serif;
        color: #555;
        user-select: none;
        overflow: hidden;
        white-space: nowrap;
    }

    .player {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        flex-shrink: 0;
    }

    .disc {
        width: 9px;
        height: 9px;
        border-radius: 50%;
        border: 1px solid #999;
        flex-shrink: 0;
    }

    .name {
        font-weight: 600;
        color: #333;
    }

    .vs {
        color: #999;
        flex-shrink: 0;
    }

    .sep {
        color: #bbb;
        flex-shrink: 0;
    }

    .meta {
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .more {
        flex-shrink: 0;
        padding: 0 5px;
        border-radius: 8px;
        background: #e0e0e0;
        color: #555;
        font-size: 11px;
        font-weight: 600;
        cursor: default;
    }
</style>
