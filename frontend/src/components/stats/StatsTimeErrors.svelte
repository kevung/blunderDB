<script>
    // Time taken over a decision against the error it cost, per player: the
    // decisions of each duration band, their mean error and blunder rate.
    // Only decisions with a recorded time are counted; the section is absent
    // when none is.
    import { onMount } from 'svelte';
    import { t } from '../../i18n/index.js';
    import { GetTimeErrors } from '../../../wailsjs/go/database/Database.js';
    import { logger } from '../../utils/logger.js';

    /** @type {{ player: string, bucket: number, decisions: number, scored: number, mean_error_mp: number, blunders: number }[]} */
    let rows = $state([]);

    const BAND_KEYS = ['stats.timeBandUnder5', 'stats.timeBand5to15', 'stats.timeBand15to30', 'stats.timeBandOver30'];

    onMount(async () => {
        try {
            rows = (await GetTimeErrors()) || [];
        } catch (error) {
            logger.error('Error loading time/error table:', error);
            rows = [];
        }
    });

    // An unscored band reads empty: there is no error to average, not a zero one.
    const mean = (r) => (r.scored > 0 ? r.mean_error_mp.toFixed(1) : '');
    const rate = (r) => (r.scored > 0 ? ((100 * r.blunders) / r.scored).toFixed(1) + ' %' : '');
</script>

{#if rows.length > 0}
    <section class="chart-section" data-testid="time-errors">
        <h3 class="section-title">{$t('stats.timeErrorsTitle')}</h3>
        <p class="hint">{$t('stats.timeErrorsHint')}</p>
        <table class="time-errors">
            <thead>
                <tr>
                    <th>{$t('stats.timePlayer')}</th>
                    <th>{$t('stats.timeBand')}</th>
                    <th>{$t('stats.timeDecisions')}</th>
                    <th>{$t('stats.timeMeanError')}</th>
                    <th>{$t('stats.timeBlunderRate')}</th>
                </tr>
            </thead>
            <tbody>
                {#each rows as r (r.player + ':' + r.bucket)}
                    <tr>
                        <td>{r.player}</td>
                        <td>{$t(BAND_KEYS[r.bucket])}</td>
                        <td>{r.decisions}</td>
                        <td>{mean(r)}</td>
                        <td>{rate(r)}</td>
                    </tr>
                {/each}
            </tbody>
        </table>
    </section>
{/if}

<style>
    .time-errors {
        border-collapse: collapse;
        font-size: var(--font-size-small);
    }
    .time-errors th,
    .time-errors td {
        padding: 2px 10px;
        text-align: right;
    }
    .time-errors th:first-child,
    .time-errors td:first-child {
        text-align: left;
    }
    .hint {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
</style>
