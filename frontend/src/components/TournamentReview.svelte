<script>
    // One player's tournament review (storage.TournamentReview, ADR-0081): L7 and PR against
    // the player's usual level over the year before, then the same PR cut by round, by the
    // decision's rank in its match (fatigue), at pressure scores and by pace, and the
    // tournament's error families. A verdict shows only where an interval holds it.
    import { GetTournamentReview } from '../../wailsjs/go/database/Database.js';
    import { logger } from '../utils/logger.js';
    import { fmtPRInterval } from '../utils/interval.js';
    import { fmtMwc7Full, mwc7Tooltip } from '../utils/mwc7.js';
    import { t } from '../i18n';

    /** @type {{ tournamentId: number, players: string[] }} */
    let { tournamentId, players } = $props();

    let player = $state('');
    /** @type {any | null} */
    let review = $state(null);
    let loading = $state(false);

    $effect(() => {
        const id = tournamentId;
        const who = player;
        let stale = false;
        loading = true;
        GetTournamentReview(id, who)
            .then((r) => {
                if (!stale) review = r || null;
            })
            .catch((/** @type {unknown} */ e) => {
                logger.error('Error loading the tournament review:', e);
                if (!stale) review = null;
            })
            .finally(() => {
                if (!stale) loading = false;
            });
        return () => {
            stale = true;
        };
    });

    /** @param {number} pr @param {any} iv @param {number} n */
    function prBand(pr, iv, n) {
        if (!n) return '—';
        const band = fmtPRInterval(iv);
        return band ? `${pr.toFixed(2)} ${band}` : pr.toFixed(2);
    }

    /** @param {any} c @param {number} [scale] */
    function versus(c, scale = 1) {
        if (!c || c.verdict === 'insufficient') return $t('tournamentReview.verdict_insufficient');
        const d = (scale * c.delta).toFixed(2);
        return `${$t(`tournamentReview.verdict_${c.verdict}`)} (${c.delta >= 0 ? '+' : ''}${d})`;
    }

    /** @param {any} c @param {number} [scale] */
    function versusTitle(c, scale = 1) {
        if (!c || c.verdict === 'insufficient') return $t('tournamentReview.insufficientHint');
        return `Δ ${(scale * c.delta).toFixed(2)} [${(scale * c.low).toFixed(2)}, ${(scale * c.high).toFixed(2)}]`;
    }

    const CUBE_LABELS = /** @type {Record<string, string>} */ ({
        offer_missed: 'stats.cubeOfferMissed',
        offer_premature: 'stats.cubeOfferPremature',
        answer_wrong_pass: 'stats.cubeAnswerWrongPass',
        answer_wrong_take: 'stats.cubeAnswerWrongTake'
    });

    /** @param {string} key @param {string} fallback */
    function label(key, fallback) {
        const s = $t(key);
        return s === key ? fallback : s;
    }

    /** @param {any} f */
    function familyName(f) {
        const theme = f.Kind === 'cube' && CUBE_LABELS[f.Theme] ? CUBE_LABELS[f.Theme] : `stats.recurringTheme_${f.Theme}`;
        return `${label(`stats.gameType_${f.GameType}`, f.GameType)} · ${$t(`stats.recurringKind_${f.Kind}`)} · ${label(theme, f.Theme)}`;
    }

    /** @param {string} part @param {string} key */
    function cellLabel(part, key) {
        return part === 'rank' ? $t('tournamentReview.rankCell', { range: key }) : $t(`tournamentReview.${part}_${key}`);
    }

    let parts = $derived(
        review
            ? [
                  { id: 'rank', title: 'tournamentReview.byRank', cells: review.by_rank ?? [] },
                  { id: 'pressure', title: 'tournamentReview.byPressure', cells: review.by_pressure ?? [] },
                  { id: 'clock', title: 'tournamentReview.byClock', cells: review.by_clock ?? [] }
              ]
            : []
    );
</script>

<section class="tournament-review" data-testid="tournament-review">
    <div class="review-head">
        <h4>{$t('tournamentReview.title')}</h4>
        <label>
            {$t('tournamentReview.player')}
            <select bind:value={player} data-testid="tournament-review-player">
                <option value="">{$t('tournamentReview.playerDefault')}</option>
                {#each players as name (name)}
                    <option value={name}>{name}</option>
                {/each}
            </select>
        </label>
        {#if loading}<span class="muted">{$t('common.loading')}…</span>{/if}
    </div>
    {#if review && review.decisions > 0}
        <p class="headline">
            {$t('tournamentReview.summary', { player: review.player, matches: review.matches, decisions: review.decisions })}
            —
            {#if review.usual.available}
                {$t('tournamentReview.usualWindow', { from: review.usual.from, to: review.usual.to, matches: review.usual.matches })}
            {:else}
                <span class="muted">{$t('tournamentReview.usualUnknown', { matches: review.usual.matches, min: 5 })}</span>
            {/if}
        </p>
        <table class="review-table">
            <thead>
                <tr><th></th><th>{$t('tournamentReview.tournament')}</th><th>{$t('tournamentReview.usual')}</th><th>{$t('tournamentReview.versus')}</th></tr>
            </thead>
            <tbody>
                <tr>
                    <td>PR</td>
                    <td>{prBand(review.pr, review.pr_interval, review.decisions)}</td>
                    <td>{prBand(review.usual.pr, review.usual.pr_interval, review.usual.decisions)}</td>
                    <td class="verdict {review.pr_versus.verdict}" title={versusTitle(review.pr_versus)}>{versus(review.pr_versus)}</td>
                </tr>
                <tr>
                    <td>L₇</td>
                    <td title={mwc7Tooltip(review.mwc7, $t)}>{fmtMwc7Full(review.mwc7)}</td>
                    <td title={mwc7Tooltip(review.usual.mwc7, $t)}>{fmtMwc7Full(review.usual.mwc7)}</td>
                    <td class="verdict {review.mwc7_versus.verdict}" title={versusTitle(review.mwc7_versus, 100)}>{versus(review.mwc7_versus, 100)}</td>
                </tr>
            </tbody>
        </table>

        <h5>{$t('tournamentReview.byRound')}</h5>
        <table class="review-table" data-testid="tournament-review-rounds">
            <thead>
                <tr><th>#</th><th>{$t('tournamentReview.opponent')}</th><th>{$t('tournamentReview.decisions')}</th><th>PR</th><th>L₇</th><th>{$t('tournamentReview.versusUsualPR')}</th></tr>
            </thead>
            <tbody>
                {#each review.rounds as r (r.match_id)}
                    <tr>
                        <td>{r.round}</td>
                        <td>{r.opponent}</td>
                        <td>{r.decisions}</td>
                        <td>{prBand(r.pr, r.pr_interval, r.decisions)}</td>
                        <td title={mwc7Tooltip(r.mwc7, $t)}>{fmtMwc7Full(r.mwc7)}</td>
                        <td class="verdict {r.versus.verdict}" title={versusTitle(r.versus)}>{versus(r.versus)}</td>
                    </tr>
                {/each}
            </tbody>
        </table>

        {#each parts as part (part.id)}
            <h5>{$t(part.title)}</h5>
            <table class="review-table">
                <thead>
                    <tr
                        ><th></th><th>{$t('tournamentReview.decisions')}</th><th>{$t('tournamentReview.tournament')}</th><th>{$t('tournamentReview.usual')}</th><th>{$t('tournamentReview.versus')}</th
                        ></tr
                    >
                </thead>
                <tbody>
                    {#each part.cells as c (c.key)}
                        <tr class:empty={c.decisions === 0}>
                            <td>{cellLabel(part.id, c.key)}</td>
                            <td>{c.decisions}</td>
                            <td>{prBand(c.pr, c.pr_interval, c.decisions)}</td>
                            <td>{prBand(c.usual_pr, c.usual_interval, c.usual_decisions)}</td>
                            <td class="verdict {c.versus.verdict}" title={versusTitle(c.versus)}>{versus(c.versus)}</td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        {/each}
        <p class="muted note">{$t('tournamentReview.clockNote')}</p>

        <h5>{$t('tournamentReview.families')}</h5>
        {#if review.families.length === 0}
            <p class="muted">{$t('tournamentReview.noFamily', { tentative: review.tentative })}</p>
        {:else}
            <ol class="families">
                {#each review.families as f (`${f.GameType}|${f.Kind}|${f.Theme}`)}
                    <li>
                        {familyName(f)} —
                        {$t('tournamentReview.family', {
                            errors: f.Errors,
                            recoverable: (100 * f.Recoverable).toFixed(2),
                            low: (100 * f.Low).toFixed(2),
                            high: (100 * f.High).toFixed(2)
                        })}
                    </li>
                {/each}
            </ol>
            {#if review.tentative > 0}<p class="muted">{$t('tournamentReview.tentative', { tentative: review.tentative })}</p>{/if}
        {/if}
    {:else if review && !loading}
        <p class="muted">{$t('tournamentReview.noDecision')}</p>
    {/if}
</section>

<style>
    .tournament-review {
        margin: 6px 0;
        padding: 6px 8px;
        border: 1px solid var(--color-border, currentColor);
        border-radius: 4px;
        font-size: var(--font-size-small);
        overflow-x: auto;
    }
    .review-head {
        display: flex;
        align-items: center;
        gap: 12px;
        flex-wrap: wrap;
    }
    h4,
    h5 {
        margin: 6px 0 2px;
    }
    .headline {
        margin: 2px 0 6px;
    }
    .review-table {
        border-collapse: collapse;
    }
    .review-table th,
    .review-table td {
        padding: 1px 8px;
        text-align: left;
        white-space: nowrap;
    }
    .muted,
    tr.empty {
        color: var(--color-text-muted, inherit);
    }
    .verdict.worse {
        color: var(--color-danger);
    }
    .verdict.better {
        color: var(--color-primary);
    }
    .verdict.insufficient {
        color: var(--color-text-muted, inherit);
    }
    .families {
        margin: 2px 0;
        padding-left: 20px;
    }
    .note {
        margin: 4px 0;
    }
</style>
