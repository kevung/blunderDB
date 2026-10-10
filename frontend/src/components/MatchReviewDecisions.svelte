<script>
    // What a match's review sends back to the board: per player, the decisions
    // most worth revisiting (one click shows the decision) and how the errors
    // split between hasty and deliberate play.
    import { t } from '../i18n';
    import { fmtLoss } from '../utils/decisionLoss.js';
    import { hasReview, paceHint } from '../utils/matchReview.js';

    /** @type {{ review: any | null, movePositions: any[], player1: string, player2: string, onselect?: (index: number) => void }} */
    let { review, movePositions, player1, player2, onselect = () => {} } = $props();

    let names = $derived([player1, player2]);

    /** @param {number} moveId */
    function select(moveId) {
        const i = movePositions.findIndex((mp) => mp.move_id === moveId);
        if (i >= 0) onselect(i);
    }
</script>

<div class="review-decisions" data-testid="match-review-decisions">
    {#if hasReview(review)}
        {#each review.players as p, seat (seat)}
            {#if p.decisions > 0}
                <div class="player">
                    <h5>{names[seat]}</h5>
                    {#if p.to_review?.length}
                        <div class="muted">{$t('matchReview.toReview')}</div>
                        <ol class="to-review">
                            {#each p.to_review as d (d.move_id)}
                                <li>
                                    <button class="link" data-testid="review-decision" onclick={() => select(d.move_id)} title={$t('matchReview.showDecision')}>
                                        {$t('matchReview.decision', { game: d.game_number, move: d.move_number, kind: $t(d.decision_type === 'cube' ? 'matchReview.cube' : 'matchReview.checker') })}
                                    </button>
                                    {$t('matchReview.lossAvoidable', { loss: fmtLoss(d.mwc_loss), avoidable: fmtLoss(d.avoidable_loss) })}
                                </li>
                            {/each}
                        </ol>
                    {/if}
                    {#if p.pace && p.pace.hasty + p.pace.deliberate + p.pace.unknown > 0}
                        <div class="pace" data-testid="error-pace">
                            {$t('matchReview.pace', {
                                hasty: p.pace.hasty,
                                hastyLoss: fmtLoss(p.pace.hasty_loss) || '0 %',
                                deliberate: p.pace.deliberate,
                                deliberateLoss: fmtLoss(p.pace.deliberate_loss) || '0 %',
                                unknown: p.pace.unknown
                            })}
                            {#if paceHint(p.pace)}<em>{$t(paceHint(p.pace))}</em>{/if}
                        </div>
                    {/if}
                    {#if !p.to_review?.length}
                        <div class="muted">{$t('matchReview.nothingToReview')}</div>
                    {/if}
                </div>
            {/if}
        {/each}
    {/if}
</div>

<style>
    .review-decisions {
        padding: 8px 12px;
        font-size: var(--font-size-base);
        text-align: left;
    }
    .player + .player {
        margin-top: 12px;
    }
    h5 {
        margin: 0 0 4px;
        font-size: inherit;
        color: var(--color-text);
    }
    .muted {
        color: var(--color-text-muted);
    }
    .to-review {
        margin: 4px 0;
        padding-left: 24px;
    }
    .to-review li {
        padding: 1px 0;
    }
    .link {
        background: none;
        border: none;
        padding: 0;
        color: inherit;
        cursor: pointer;
        text-decoration: underline;
    }
    .pace {
        margin-top: 4px;
        color: var(--color-text-muted);
    }
    em {
        margin-left: 4px;
    }
</style>
