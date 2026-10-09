<script>
    // The at-a-glance summary of a match's review (storage.MatchReview,
    // ADR-0078): per player, the PR, the MWC the match cost, and whether the
    // dice or the play decided the result. One loss only, the match's own; the
    // intervals, the 7-point loss and the luck's components are in the folded
    // details (MatchReviewDetails), and in the tooltips here.
    import { t } from '../i18n';
    import { fmtPRInterval } from '../utils/interval.js';
    import { fmtLoss } from '../utils/decisionLoss.js';
    import { hasReview, luckVerdict, signedPct } from '../utils/matchReview.js';

    /** @type {{ review: any | null, player1: string, player2: string }} */
    let { review, player1, player2 } = $props();

    let names = $derived([player1, player2]);

    /** @param {any} p */
    const prHint = (p) => (fmtPRInterval(p.pr_interval) ? `${$t('matchReview.prInterval')} ${fmtPRInterval(p.pr_interval)}` : $t('matchReview.noInterval'));
    /** @param {any} luck */
    const luckHint = (luck) =>
        $t('matchReview.luckAdjusted', { adjusted: signedPct(luck.adjusted), result: signedPct(luck.result), luck: signedPct(luck.luck), balance: signedPct(luck.error_balance) });
</script>

{#if hasReview(review)}
    <section class="match-review" aria-label={$t('matchReview.title')} data-testid="match-review">
        {#each review.players as p, seat (seat)}
            {#if p.decisions > 0}
                <div class="player" data-testid="review-player-{seat}">
                    <b class="name"
                        ><svg class="swatch" viewBox="0 0 18 6" aria-hidden="true"><line class="line" class:player1={seat === 0} class:player2={seat === 1} x1="1" y1="3" x2="17" y2="3" /></svg>{names[
                            seat
                        ]}</b
                    >
                    <span class="figure" title={prHint(p)}><span class="label">PR</span> <span data-testid="review-pr-{seat}">{p.pr.toFixed(2)}</span></span>
                    <span class="figure" title={$t('matchReview.mwcLossHint')}
                        ><span class="label">{$t('match.lossTotal')}</span> <span data-testid="review-mwc-{seat}">{fmtLoss(p.mwc_loss) || '0'}</span></span
                    >
                    {#if p.luck?.available}
                        <span class="luck" title={luckHint(p.luck)} data-testid="review-luck-{seat}">
                            {$t(luckVerdict(p.luck))}<span class="adjusted">{$t('matchReview.adjustedShort', { adjusted: signedPct(p.luck.adjusted) })}</span>
                        </span>
                    {/if}
                </div>
            {/if}
        {/each}
    </section>
{/if}

<style>
    .match-review {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
        gap: 4px 16px;
        padding: 6px 12px;
        font-size: var(--font-size-small);
        text-align: left;
    }
    .player {
        display: flex;
        flex-wrap: wrap;
        align-items: baseline;
        gap: 2px 12px;
        min-width: 0;
    }
    .name {
        font-size: var(--font-size-base);
        color: var(--color-text);
    }
    /* The line each player has on the cumulative chart just below. */
    .swatch {
        width: 18px;
        height: 6px;
        margin-right: 6px;
        vertical-align: middle;
    }
    .line {
        stroke-width: 2;
        stroke-linecap: round;
    }
    .line.player1 {
        stroke: var(--player1-color, currentColor);
    }
    .line.player2 {
        stroke: var(--player2-color, currentColor);
        stroke-dasharray: 5 3;
    }
    .label {
        color: var(--color-text-muted);
    }
    .figure {
        white-space: nowrap;
        font-variant-numeric: tabular-nums;
    }
    .luck {
        font-style: italic;
    }
    .adjusted {
        margin-left: 6px;
        font-style: normal;
        color: var(--color-text-muted);
        font-variant-numeric: tabular-nums;
    }
</style>
