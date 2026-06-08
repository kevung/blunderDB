<script>
    // Compact header strip shown directly above the board while reviewing a
    // match. It is the match-mode counterpart to the (now removed) "P1 vs P2"
    // status-bar text: in general/database position review it renders nothing,
    // so the board stays uncluttered. Full match metadata (event, round, date,
    // length) is fetched by id via GetMatchByID — the matchContextStore only
    // carries the player names, and matchID is always set in match mode.
    import { matchContextStore } from '../stores/positionStore';
    import { GetMatchByID } from '../../wailsjs/go/database/Database.js';
    import { boardColorsStore } from '../stores/boardColorsStore';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';

    let match = $state(null);
    let loadedMatchID = $state(null);

    // Fetch (and cache) the full match whenever the active match id changes.
    $effect(() => {
        const ctx = $matchContextStore;
        if (!ctx.isMatchMode || !ctx.matchID) {
            match = null;
            loadedMatchID = null;
            return;
        }
        if (ctx.matchID === loadedMatchID) return;
        const wantID = ctx.matchID;
        GetMatchByID(wantID)
            .then((m) => {
                // Ignore a stale response if the user navigated on in the meantime.
                if ($matchContextStore.matchID === wantID) {
                    match = m;
                    loadedMatchID = wantID;
                }
            })
            .catch((e) => {
                logger.error('MatchInfoBar: failed to load match', e);
                match = null;
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

    // Optional metadata fields, in display order, empties omitted.
    let metaParts = $derived(
        [match?.event, match?.location, match?.round, formatDate(match?.match_date), match?.match_length > 0 ? `${match.match_length}${$t('matchInfo.points')}` : '']
            .map((s) => (s == null ? '' : String(s).trim()))
            .filter((s) => s.length > 0)
    );

    let visible = $derived($matchContextStore.isMatchMode && !!$matchContextStore.matchID);
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
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
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
</style>
