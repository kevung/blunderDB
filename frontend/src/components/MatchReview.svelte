<script>
    // A match's study summary (storage.MatchReview, ADR-0078), per player: the PR and the
    // 7-point loss with their interval over the games, whether the dice or the play decided the
    // result, the three errors most worth revisiting (one click shows the decision), and whether
    // the errors came from haste or from a gap in knowledge.
    import { t } from '../i18n';
    import { fmtPRInterval } from '../utils/interval.js';
    import { fmtMwc7Full, mwc7Tooltip } from '../utils/mwc7.js';
    import { fmtLoss } from '../utils/decisionLoss.js';

    /** @type {{ review: any | null, movePositions: any[], player1: string, player2: string, onselect?: (index: number) => void }} */
    let { review, movePositions, player1, player2, onselect = () => {} } = $props();

    let names = $derived([player1, player2]);

    /** @param {number} v a signed MWC fraction */
    const signed = (v) => (v >= 0 ? '+' : '−') + Math.abs(v * 100).toFixed(1) + ' %';

    /** Which of dice and play decided the result, from the player's side. */
    function verdict(/** @type {any} */ luck) {
        const won = luck.result > 0;
        const skill = luck.adjusted > 0;
        if (won) return skill ? 'matchReview.wonByPlay' : 'matchReview.wonByDice';
        return skill ? 'matchReview.lostByDice' : 'matchReview.lostByPlay';
    }

    /** What the split of the errors by time calls for. */
    function paceHint(/** @type {any} */ pace) {
        if (pace.hasty > pace.deliberate) return 'matchReview.paceDiscipline';
        if (pace.deliberate > pace.hasty) return 'matchReview.paceKnowledge';
        return '';
    }

    /** @param {number} moveId */
    function select(moveId) {
        const i = movePositions.findIndex((mp) => mp.move_id === moveId);
        if (i >= 0) onselect(i);
    }
</script>

{#if review && review.players?.some((/** @type {any} */ p) => p.decisions > 0)}
    <section class="match-review" data-testid="match-review">
        <h4>{$t('matchReview.title')}</h4>
        {#each review.players as p, seat (seat)}
            {#if p.decisions > 0}
                <div class="player">
                    <div class="headline">
                        <b>{names[seat]}</b>
                        <span>PR {p.pr.toFixed(2)}</span>
                        {#if fmtPRInterval(p.pr_interval)}
                            <span class="ci" title={$t('stats.intervalHint')}>{fmtPRInterval(p.pr_interval)}</span>
                        {:else}
                            <span class="ci" title={$t('matchReview.oneGame')}>—</span>
                        {/if}
                        {#if p.mwc7?.available}
                            <span title={mwc7Tooltip(p.mwc7, $t)}>{$t('mwc7.short')} {fmtMwc7Full(p.mwc7)}</span>
                        {/if}
                    </div>
                    {#if p.luck?.available}
                        <div class="luck" data-testid="luck-adjusted">
                            {$t('matchReview.luckAdjusted', { adjusted: signed(p.luck.adjusted), result: signed(p.luck.result), luck: signed(p.luck.luck), balance: signed(p.luck.error_balance) })}
                            — <em>{$t(verdict(p.luck))}</em>
                            {#if p.luck.rolls_measured < p.luck.rolls}
                                <span class="partial">{$t('matchReview.luckPartial', { measured: p.luck.rolls_measured, rolls: p.luck.rolls })}</span>
                            {/if}
                        </div>
                    {/if}
                    {#if p.to_review?.length}
                        <ol class="to-review">
                            {#each p.to_review as d (d.move_id)}
                                <li>
                                    <button class="link" onclick={() => select(d.move_id)} title={$t('matchReview.showDecision')}>
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
                </div>
            {/if}
        {/each}
    </section>
{/if}

<style>
    .match-review {
        margin: 0 0 8px;
        padding: 6px 8px;
        border: 1px solid var(--color-border, currentColor);
        border-radius: 4px;
        font-size: var(--font-size-small);
    }
    h4 {
        margin: 0 0 4px;
        font-size: inherit;
    }
    .player + .player {
        margin-top: 6px;
    }
    .headline {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
    }
    .ci,
    .partial {
        color: var(--color-text-muted, inherit);
    }
    .to-review {
        margin: 2px 0;
        padding-left: 20px;
    }
    .link {
        background: none;
        border: none;
        padding: 0;
        color: inherit;
        cursor: pointer;
        text-decoration: underline;
        font-size: inherit;
        font-family: inherit;
    }
    em {
        margin-left: 4px;
    }
</style>
