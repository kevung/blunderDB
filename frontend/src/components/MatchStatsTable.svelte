<script>
    // A match's statistics per player (GetMatchDetailStats), in the match
    // sheet's folded "Stats" section: loaded when the section opens.
    import { t } from '../i18n';
    import { MATCH_STAT_ROWS } from '../utils/matchTable.js';

    /** @type {{ stats: any | null, loading: boolean, player1: string, player2: string }} */
    let { stats, loading, player1, player2 } = $props();
</script>

{#if loading}
    <div class="loading-state">{$t('match.loadingStats')}</div>
{:else if !stats}
    <div class="empty-state">{$t('match.noAnalysedPositions')}</div>
{:else}
    {@const p1 = stats.player1}
    {@const p2 = stats.player2}
    {@const p1Name = player1 || $t('match.player1')}
    {@const p2Name = player2 || $t('match.player2')}
    <table class="stats-table">
        <thead>
            <tr>
                <th class="stats-label"></th>
                <th class="stats-player">{p1Name}</th>
                <th class="stats-player">{p2Name}</th>
            </tr>
        </thead>
        <tbody>
            {#each MATCH_STAT_ROWS as row, i (i)}
                {#if 'section' in row}
                    <tr class="stats-section-header"><td colspan="3">{$t(row.section)}</td></tr>
                {:else}
                    <tr>
                        <td class="stats-label{row.sub ? ' sub-label' : ''}">{row.bullet ? '• ' : ''}{$t(row.label ?? '')}</td>
                        <td class="stats-val{row.valClass ? ' ' + row.valClass : ''}{row.sub ? ' sub-val' : ''}" title={row.title?.(p1)}>{row.fmt(p1)}</td>
                        <td class="stats-val{row.valClass ? ' ' + row.valClass : ''}{row.sub ? ' sub-val' : ''}" title={row.title?.(p2)}>{row.fmt(p2)}</td>
                    </tr>
                {/if}
            {/each}
        </tbody>
    </table>
{/if}

<style>
    .loading-state,
    .empty-state {
        padding: 8px 0;
        color: var(--color-text-muted);
    }

    .stats-table {
        width: 100%;
        border-collapse: collapse;
        font-size: var(--font-size-base);
    }

    .stats-table th {
        text-align: left;
        padding: 4px 8px;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        border-bottom: 2px solid var(--color-border);
        font-weight: 600;
    }

    .stats-table th.stats-player {
        text-align: right;
        min-width: 80px;
    }

    .stats-section-header td {
        background: var(--color-surface-alt);
        padding: 5px 8px;
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text);
        text-transform: uppercase;
        letter-spacing: 0.03em;
        border-top: 1px solid var(--color-border);
    }

    .stats-label {
        padding: 3px 8px;
        color: var(--color-text-muted);
        font-size: var(--font-size-base);
    }

    .sub-label {
        padding-left: 20px;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .stats-val {
        text-align: right;
        padding: 3px 8px;
        font-variant-numeric: tabular-nums;
        color: var(--color-text);
        min-width: 80px;
    }

    .sub-val {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .pr-val {
        font-weight: 600;
        color: color-mix(in srgb, var(--color-primary) 80%, var(--color-text));
    }
</style>
