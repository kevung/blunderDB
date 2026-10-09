<script>
    // The one line atop a match's sheet: who played, how it ended, its length
    // and date; then the actions, the video and the review in reach, the rest
    // behind a "⋯" menu. Where it was played and everything else the file says
    // are in the sheet's folded sections.
    import { t } from '../i18n';
    import { formatDate } from '../utils/matchTable.js';
    import { finalScore } from '../utils/matchSheet.js';
    import ContextMenu from './ContextMenu.svelte';

    /**
     * @type {{
     *   match: any, games: any[], cadence?: { bank: string, delay: number } | null,
     *   hasVideo?: boolean, videoOpen?: boolean,
     *   ontogglevideo?: () => void, onreview: () => void, onexport: () => void, onedit: () => void, ondelete: (e: Event) => void
     * }}
     */
    let { match, games, cadence = null, hasVideo = false, videoOpen = false, ontogglevideo = () => {}, onreview, onexport, onedit, ondelete } = $props();

    let result = $derived(finalScore(games, match.match_length));
    let names = $derived([match.player1_name, match.player2_name]);
    let date = $derived(match.match_date && formatDate(match.match_date) !== '-' ? formatDate(match.match_date) : '');
    let where = $derived([match.tournament_name || match.event, match.round ? `R${match.round}` : '', match.location].filter(Boolean).join(' · '));

    /** @type {{ x: number, y: number } | null} */
    let menu = $state(null);

    /** @param {MouseEvent} e */
    function openMenu(e) {
        if (menu) {
            menu = null;
            return;
        }
        const r = /** @type {HTMLElement} */ (e.currentTarget).getBoundingClientRect();
        // After this click has run its course: the menu closes on any click outside it.
        setTimeout(() => (menu = { x: Math.max(0, r.right - 220), y: r.bottom + 2 }));
    }

    let items = $derived([
        { label: $t('match.exportMat'), onClick: onexport },
        { label: $t('match.editTranscription'), onClick: onedit },
        { label: $t('match.deleteMatch'), onClick: () => ondelete(new Event('click')) }
    ]);
</script>

<div class="detail-header" data-testid="match-detail-header">
    <div class="title" title={where || undefined}>
        <span class="player" class:winner={result?.winner === 0}>{names[0]}</span>
        {#if result}
            <span class="score" data-testid="header-score" title={result.winner === null ? $t('match.finalScore') : $t('match.matchWonBy', { player: names[result.winner] })}
                >{result.score[0]}–{result.score[1]}</span
            >
        {:else}
            <span class="vs">{$t('match.vs')}</span>
        {/if}
        <span class="player" class:winner={result?.winner === 1}>{names[1]}</span>
        <span class="badge">{match.match_length} pt</span>
        {#if cadence}<span class="badge" data-testid="header-cadence" title={$t('match.cadenceRow')}>{$t('match.cadenceBadge', { bank: cadence.bank || '—', s: cadence.delay })}</span>{/if}
        {#if date}<span class="date" title={$t('match.date')}>{date}</span>{/if}
    </div>
    <div class="actions">
        {#if hasVideo}
            <button class="action video-btn" class:active={videoOpen} data-testid="match-video" onclick={ontogglevideo} title={$t('match.videoOpen')}>🎞</button>
        {/if}
        <button class="action enter-match-btn" onclick={onreview} title="{$t('match.enterMatchMode')} (↵)">▶ {$t('match.review')}</button>
        <button class="action more" data-testid="match-more" aria-haspopup="menu" aria-expanded={!!menu} onclick={openMenu} title={$t('match.moreActions')} aria-label={$t('match.moreActions')}
            >⋯</button
        >
    </div>
</div>
{#if menu}
    <ContextMenu x={menu.x} y={menu.y} {items} onClose={() => (menu = null)} />
{/if}

<style>
    .detail-header {
        flex-shrink: 0;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 4px 12px;
        padding: 6px 12px;
        border-bottom: 1px solid var(--color-border);
        background: var(--color-surface-alt);
    }
    .title {
        display: flex;
        flex-wrap: wrap;
        align-items: baseline;
        gap: 2px 6px;
        min-width: 0;
        flex: 1 1 auto;
        color: var(--color-text);
    }
    .player {
        font-weight: 500;
    }
    .player.winner {
        font-weight: 700;
    }
    .score {
        font-weight: 700;
        font-variant-numeric: tabular-nums;
    }
    .vs,
    .date {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
    .badge {
        /* Text and fill come from a token pair the theme contrast test covers, so the badge stays AA in every theme. */
        background: var(--color-surface-alt);
        color: var(--color-text);
        border: 1px solid var(--color-primary);
        font-size: var(--font-size-small);
        font-weight: 600;
        padding: 0 6px;
        border-radius: 8px;
    }
    .actions {
        display: flex;
        align-items: center;
        gap: 2px;
        margin-left: auto;
    }
    .action {
        background: none;
        border: 1px solid transparent;
        border-radius: 4px;
        padding: 2px 8px;
        cursor: pointer;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }
    .action:hover,
    .action.active {
        color: var(--color-text);
        border-color: var(--color-border);
    }
    .enter-match-btn {
        color: color-mix(in srgb, var(--color-primary) 80%, var(--color-text));
        font-weight: 600;
    }
    .more {
        font-weight: 700;
    }
</style>
