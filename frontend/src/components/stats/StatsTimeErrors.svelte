<script>
    // Time taken over a decision against the error it cost, per player: the
    // decisions of each duration band, their mean error and blunder rate.
    // Only decisions with a recorded time are counted; the section is absent
    // when none is.
    import { onMount } from 'svelte';
    import { t } from '../../i18n/index.js';
    import { GetTimeErrors } from '../../../wailsjs/go/database/Database.js';
    import { logger } from '../../utils/logger.js';
    import PanelTable from '../panels/PanelTable.svelte';
    import CountLink from '../panels/CountLink.svelte';
    import { loadPositionsFromStatsSelection } from '../../services/positionLoader.js';

    /** @type {{ player: string, bucket: number, decisions: number, scored: number, mean_error_mp: number, blunders: number, positions: number }[]} */
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
    /** @typedef {(typeof rows)[number]} Row */
    const mean = (/** @type {Row} */ r) => (r.scored > 0 ? r.mean_error_mp.toFixed(1) : '');
    const rateValue = (/** @type {Row} */ r) => (r.scored > 0 ? (100 * r.blunders) / r.scored : null);
    const rate = (/** @type {Row} */ r) => (r.scored > 0 ? /** @type {number} */ (rateValue(r)).toFixed(1) + ' %' : '');

    // The table reads every timed decision whatever the filter, and so does its selection.
    const open = (/** @type {Row} */ r) => loadPositionsFromStatsSelection({}, { Kind: 'time_band', Player: r.player, TimeBucket: r.bucket });

    const columns = $derived([
        { key: 'player', label: $t('stats.timePlayer'), sortable: true, filter: /** @type {const} */ ('text'), value: (/** @type {Row} */ r) => r.player },
        { key: 'band', label: $t('stats.timeBand'), sortable: true, value: (/** @type {Row} */ r) => r.bucket },
        {
            key: 'decisions',
            label: $t('stats.timeDecisions'),
            sortable: true,
            defaultDir: 'desc',
            align: 'right',
            filter: /** @type {const} */ ('number'),
            value: (/** @type {Row} */ r) => r.positions
        },
        {
            key: 'mean',
            label: $t('stats.timeMeanError'),
            sortable: true,
            align: 'right',
            filter: /** @type {const} */ ('number'),
            value: (/** @type {Row} */ r) => (r.scored > 0 ? r.mean_error_mp : null)
        },
        { key: 'rate', label: $t('stats.timeBlunderRate'), sortable: true, align: 'right', filter: /** @type {const} */ ('number'), value: rateValue }
    ]);
</script>

{#if rows.length > 0}
    <section class="chart-section" data-testid="time-errors">
        <h3 class="section-title">{$t('stats.timeErrorsTitle')}</h3>
        <p class="hint">{$t('stats.timeErrorsHint')}</p>
        <PanelTable viewId="stats.timeErrors" {rows} rowKey={(r) => r.player + ':' + r.bucket} {columns}>
            {#snippet cells(/** @type {Row} */ r)}
                <td>{r.player}</td>
                <td>{$t(BAND_KEYS[r.bucket])}</td>
                <td class="align-right"
                    >{#if r.positions > 0}<CountLink
                            label={r.positions}
                            testid="time-band-{r.bucket}"
                            onclick={() => open(r)}
                            title={$t('stats.breakdownOpenPositions', { n: r.positions, decisions: r.decisions })}
                        />{:else}0{/if}</td
                >
                <td class="align-right">{mean(r)}</td>
                <td class="align-right">{rate(r)}</td>
            {/snippet}
        </PanelTable>
    </section>
{/if}

<style>
    .hint {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
</style>
