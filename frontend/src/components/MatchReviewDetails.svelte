<script>
    // The details of a match's review, what the summary keeps for a second
    // look. Per player, the PR and the 7-point loss with their interval over
    // the games, the luck-adjusted result and its components; then the table of
    // losses against difficulty. The decisions to revisit are MatchReviewDecisions.
    import { t } from '../i18n';
    import { fmtPRInterval } from '../utils/interval.js';
    import { fmtMwc7Full, mwc7Tooltip } from '../utils/mwc7.js';
    import { fmtExcess, fmtLoss, lossSeries, servedDifficulty } from '../utils/decisionLoss.js';
    import { hasReview, luckVerdict, signedPct } from '../utils/matchReview.js';

    /** @type {{ review: any | null, losses: any[] | null, movePositions: any[], player1: string, player2: string }} */
    let { review, losses, movePositions, player1, player2 } = $props();

    let names = $derived([player1, player2]);
    let series = $derived(lossSeries(movePositions, losses));
    // The difficulty summary is the review's, computed in Go: the excess and
    // the ratio are the ones `match --format summary` prints.
    let served = $derived(servedDifficulty(review));
</script>

<div class="review-details" data-testid="match-review-details">
    {#if hasReview(review)}
        {#each review.players as p, seat (seat)}
            {#if p.decisions > 0}
                <div class="player">
                    <h5>{names[seat]}</h5>
                    <div class="facts">
                        <span>
                            {$t('matchReview.prInterval')}
                            {#if fmtPRInterval(p.pr_interval)}
                                <span class="value" title={$t('stats.intervalHint')}>PR {p.pr.toFixed(2)} {fmtPRInterval(p.pr_interval)}</span>
                            {:else}
                                <span class="value" title={$t('matchReview.noInterval')}>PR {p.pr.toFixed(2)} —</span>
                            {/if}
                        </span>
                        {#if p.mwc7?.available}
                            <span title={`${$t('matchReview.mwc7Hint')}\n${mwc7Tooltip(p.mwc7, $t)}`} data-testid="review-mwc7-{seat}">
                                {$t('matchReview.mwc7Label')} <span class="value">{fmtMwc7Full(p.mwc7)}</span>
                            </span>
                        {/if}
                    </div>
                    {#if p.luck?.available}
                        <div class="luck" data-testid="luck-adjusted">
                            {$t('matchReview.luckAdjusted', {
                                adjusted: signedPct(p.luck.adjusted),
                                result: signedPct(p.luck.result),
                                luck: signedPct(p.luck.luck),
                                balance: signedPct(p.luck.error_balance)
                            })}
                            — <em>{$t(luckVerdict(p.luck))}</em>
                            {#if p.luck.rolls_measured < p.luck.rolls}
                                <span class="muted">{$t('matchReview.luckPartial', { measured: p.luck.rolls_measured, rolls: p.luck.rolls })}</span>
                            {/if}
                        </div>
                    {/if}
                </div>
            {/if}
        {/each}
    {/if}

    {#if series.any}
        <div class="table-wrap">
            <table class="loss-table">
                <thead>
                    <tr>
                        <th></th>
                        <th>{$t('match.lossTotal')}</th>
                        <th>{$t('match.lossScored')}</th>
                        {#if served.any}
                            <th title={$t('match.difficultyColTooltip')}>{$t('match.difficultyTotal')}</th>
                            <th title={$t('match.excessTooltip')}>{$t('match.excess')}</th>
                            <th title={$t('match.ratioTooltip')}>{$t('match.ratio')}</th>
                            <th title={$t('match.avoidableTooltip')}>{$t('match.avoidableCount')}</th>
                        {/if}
                    </tr>
                </thead>
                <tbody>
                    {#each [0, 1] as p (p)}
                        <tr>
                            <td class="loss-player">{names[p]}</td>
                            <td data-testid="loss-total-{p}">{fmtLoss(series.totals[p])}</td>
                            <td>{series.scored[p]}</td>
                            {#if served.any}
                                {@const s = served.difficulty[p]}
                                <td data-testid="difficulty-total-{p}">{s ? fmtLoss(s.difficulty) : ''}</td>
                                <td data-testid="excess-{p}">{s ? fmtExcess(s.excess) : ''}</td>
                                <td data-testid="ratio-{p}">{s?.ratio == null ? '—' : s.ratio.toFixed(2)}</td>
                                <td data-testid="avoidable-{p}">{s?.avoidable ?? ''}</td>
                            {/if}
                        </tr>
                    {/each}
                </tbody>
            </table>
        </div>
    {/if}
</div>

<style>
    .review-details {
        padding: 8px 12px;
        font-size: var(--font-size-small);
        text-align: left;
    }
    .player + .player,
    .table-wrap {
        margin-top: 8px;
    }
    h5 {
        margin: 0 0 2px;
        font-size: inherit;
        color: var(--color-text);
    }
    .facts {
        display: flex;
        flex-wrap: wrap;
        gap: 2px 16px;
    }
    .value {
        font-variant-numeric: tabular-nums;
        color: var(--color-text);
    }
    .facts,
    .muted {
        color: var(--color-text-muted);
    }
    em {
        margin-left: 4px;
    }
    .table-wrap {
        overflow-x: auto;
    }
    .loss-table {
        border-collapse: collapse;
    }
    .loss-table th,
    .loss-table td {
        padding: 2px 8px;
        text-align: right;
        font-variant-numeric: tabular-nums;
    }
    .loss-table th {
        font-weight: 500;
        color: var(--color-text-muted);
    }
    .loss-table .loss-player {
        text-align: left;
    }
</style>
