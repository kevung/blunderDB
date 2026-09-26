<script>
    // Header strip above the board: the reviewed match in MATCH mode, otherwise
    // the position's provenance (GetPositionProvenance) — nothing when no match
    // references it, the first match plus a "+N" badge when several do (dedup).
    import { matchContextStore, positionStore } from '../stores/positionStore';
    import { transcriptionInfoStore } from '../stores/transcriptionStore.js';
    import { GetMatchByID, GetPositionProvenance } from '../../wailsjs/go/database/Database.js';
    import { boardColorsStore } from '../stores/boardColorsStore';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { formatDate as formatDateI18n } from '../utils/format.js';
    import { SvelteSet } from 'svelte/reactivity';

    let match = $state(null);
    // Other matches this position also came from (provenance is one-to-many).
    let otherMatches = $state([]);
    // Cache key: 'm<id>' in match mode, 'p<id>' for a studied position's provenance.
    let loadedKey = $state(null);

    // Cached by key: navigating within a match or redrawing does not refetch.
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
    let player1Name = $derived($matchContextStore.player1Name || match?.player1_name || $transcriptionInfoStore?.player1 || '');
    let player2Name = $derived($matchContextStore.player2Name || match?.player2_name || $transcriptionInfoStore?.player2 || '');

    function formatDate(value) {
        if (!value) return '';
        const d = new Date(value);
        // Guard against Go's zero time (year 0001) and invalid dates.
        if (isNaN(d.getTime()) || d.getFullYear() < 1900) return '';
        return formatDateI18n(d);
    }

    // Metadata in display order, empties omitted, tournament first; deduped
    // case-insensitively since tournament_name and event are often equal.
    let metaParts = $derived.by(() => {
        const raw = [
            match?.tournament_name,
            match?.event,
            match?.location,
            match?.round,
            formatDate(match?.match_date),
            match?.match_length > 0 ? `${match.match_length}${$t('matchInfo.points')}` : ''
        ];
        const seen = new SvelteSet();
        const out = [];
        for (const v of raw) {
            const s = v == null ? '' : String(v).trim();
            if (!s) continue;
            const key = s.toLowerCase();
            if (seen.has(key)) continue;
            seen.add(key);
            out.push(s);
        }
        return out;
    });

    // Tooltip listing the other matches the (deduplicated) position came from.
    // Data only — player names — so it needs no translation.
    let otherMatchesTitle = $derived(otherMatches.map((m) => `${m.player1_name} ${$t('matchInfo.vs')} ${m.player2_name}`).join('\n'));

    // Troisième contexte : un brouillon en cours de frappe (ux.md §5, ADR-0048
    // décision 2). Le panneau pose ces faits ; rien n'est dérivé ici.
    let draftInfo = $derived($transcriptionInfoStore);

    let draftParts = $derived.by(() => {
        const info = draftInfo;
        if (!info) return [];
        const out = [$t(info.lengthKey, info.lengthParams)];
        if (info.score) out.push($t('transcription.score', { a: info.score[0], b: info.score[1] }));
        if (info.crawford) out.push($t('transcription.crawford'));
        out.push($t('transcription.gameNumber', { n: info.gameNumber }));
        out.push($t(info.cubeKey, info.cubeParams));
        return out;
    });

    // Visible in match mode, whenever a studied position resolves to a match, or
    // while a transcription draft is open.
    let visible = $derived(($matchContextStore.isMatchMode && !!$matchContextStore.matchID) || !!match || !!draftInfo);

    // Board.svelte measures only on 'resize': dispatch one (after a rAF, once the
    // layout has reflowed) when the bar appears or goes.
    $effect(() => {
        visible; // tracked dep
        requestAnimationFrame(() => window.dispatchEvent(new Event('resize')));
    });
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
        {#if draftInfo}
            <span class="sep">·</span>
            <span class="meta" data-testid="match-info-draft">{draftParts.join(' · ')}</span>
            <span class="on-roll">{draftInfo.onRoll}</span>
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
        background: var(--color-surface-alt);
        border-bottom: 1px solid var(--color-border);
        font-size: var(--font-size-base);
        font-family: var(--font-family-ui);
        color: var(--color-text-muted);
        user-select: none;
        overflow: hidden;
        white-space: nowrap;
    }

    /* Le camp au trait est le seul fait de la barre qu'on relit à chaque tour :
       il porte le poids que les six autres n'ont pas. */
    .on-roll {
        flex-shrink: 0;
        color: var(--color-text);
        font-weight: 600;
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
        color: var(--color-text);
    }

    .vs {
        color: var(--color-text-muted);
        flex-shrink: 0;
    }

    .sep {
        color: var(--color-text-muted);
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
        background: color-mix(in srgb, var(--color-text) 10%, var(--color-surface-alt));
        color: var(--color-text);
        font-size: var(--font-size-small);
        font-weight: 600;
        cursor: default;
    }
</style>
