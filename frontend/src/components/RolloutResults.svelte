<script>
    // Les rollouts d'une position, à côté de l'analyse et jamais à sa place (ADR-0013, ADR-0060) :
    // une Configuration par bloc, ses candidats avec équité, IC 95 % et JSD.
    import { t } from '../i18n';

    /** @type {{ rollouts?: any[], live?: any[], liveGames?: number, liveMaxGames?: number, unsaved?: any }} */
    let { rollouts = [], live = [], liveGames = 0, liveMaxGames = 0, unsaved = null } = $props();

    const CUBE_KEYS = { 'No double': 'analysis.noDouble', 'Double/Take': 'analysis.doubleTake', 'Double/Pass': 'analysis.doublePass' };

    /** @param {string} move */
    function label(move) {
        const key = CUBE_KEYS[move];
        return key ? $t(key) : move;
    }
    /** @param {number} v */
    const eq = (v) => (Number.isFinite(v) ? v.toFixed(3) : '');
    /** @param {number} v */
    const jsd = (v) => (Number.isFinite(v) && v > 0 ? v.toFixed(1) : '');

    /** A live or unsaved result put in the shape of a stored one. */
    let unsavedBlock = $derived(
        unsaved
            ? {
                  analysisEngine: unsaved.engine_version,
                  analysisDepth: '',
                  signature: unsaved.signature,
                  kind: unsaved.kind,
                  settings: { jsdLimit: unsaved.settings?.jsd_limit ?? 0 },
                  games: unsaved.games,
                  stop: unsaved.stop,
                  cubefulBias: unsaved.cubeful_bias,
                  exactBearoff: unsaved.exact_bearoff,
                  candidates: (unsaved.candidates ?? []).map((/** @type {any} */ c) => ({ move: c.move, equity: c.equity, stdErr: c.std_err, ci95: c.ci95, games: c.games, jsd: c.jsd }))
              }
            : null
    );
    let blocks = $derived([...(unsavedBlock ? [unsavedBlock] : []), ...rollouts]);
</script>

{#snippet table(/** @type {any[]} */ candidates, /** @type {number} */ limit)}
    <table class="rollout-table">
        <thead>
            <tr>
                <th class="move">{$t('analysis.move')}</th>
                <th>{$t('analysis.equity')}</th>
                <th title={$t('rollout.ciHint')}>{$t('rollout.ci95')}</th>
                <th title={$t('rollout.jsdHint')}>{$t('rollout.jsd')}</th>
                <th>{$t('rollout.games')}</th>
            </tr>
        </thead>
        <tbody>
            {#each candidates as c, i (c.move)}
                <tr class:best={i === 0}>
                    <td class="move">{label(c.move)}</td>
                    <td class="num">{eq(c.equity)}</td>
                    <td class="num" title={$t('rollout.stdErr', { value: eq(c.stdErr) })}>±{eq(c.ci95)}</td>
                    <td class="num" class:decided={limit > 0 && c.jsd >= limit}>{jsd(c.jsd)}</td>
                    <td class="num">{c.games}</td>
                </tr>
            {/each}
        </tbody>
    </table>
{/snippet}

{#if live.length}
    <section class="rollout-block live" aria-label={$t('rollout.live')} data-testid="rollout-live">
        <h3>{$t('rollout.live')} <span class="muted">{liveGames}/{liveMaxGames}</span></h3>
        {@render table(live, 0)}
    </section>
{/if}

{#each blocks as r (r.signature + r.analysisDepth)}
    <section class="rollout-block" aria-label={$t('rollout.stored')} data-testid="rollout-block">
        <h3>
            {$t('rollout.title')}
            {r.analysisDepth || ''}
            {#if r === unsavedBlock}<span class="muted">· {$t('rollout.notStored')}</span>{/if}
        </h3>
        {@render table(r.candidates ?? [], r.settings?.jsdLimit ?? 0)}
        {#if r.kind === 'cube' && r.bestCubeAction}
            <p class="note" data-testid="rollout-cube-verdict">
                {$t('rollout.cubeVerdict', { action: r.bestCubeAction, double: jsd(r.jsdDouble) || '0', take: jsd(r.jsdTake) || '0' })}
            </p>
        {/if}
        <p class="note">{r.stop === 'jsd' ? $t('rollout.stopJsd') : $t('rollout.stopMax', { games: r.games })}</p>
        {#if r.cubefulBias}<p class="note" data-testid="rollout-bias">{$t('rollout.cubefulBias')}</p>{/if}
        {#if r.exactBearoff}<p class="note">{$t('rollout.exactBearoff')}</p>{/if}
        <details class="configuration">
            <summary>{$t('rollout.configuration')}</summary>
            <p class="engine">{r.analysisEngine}</p>
            <p class="signature">{r.signature}</p>
        </details>
    </section>
{/each}

<style>
    .rollout-block {
        margin-top: var(--space-3);
        padding-top: var(--space-2);
        border-top: 1px solid var(--color-border);
    }
    h3 {
        margin: 0 0 var(--space-1);
        font-size: var(--font-size-base);
        font-weight: 600;
    }
    .muted,
    .note,
    .configuration {
        color: var(--color-text-muted);
    }
    .note {
        margin: var(--space-1) 0 0;
        font-size: var(--font-size-small);
    }
    .rollout-table {
        width: 100%;
        border-collapse: collapse;
        font-size: var(--font-size-base);
        font-variant-numeric: tabular-nums;
    }
    th,
    td {
        padding: 2px var(--space-2);
        text-align: right;
        border-bottom: 1px solid var(--color-border);
    }
    th {
        font-weight: 600;
    }
    .move {
        text-align: left;
    }
    tr.best td {
        font-weight: 600;
    }
    .decided {
        color: var(--color-primary);
        font-weight: 600;
    }
    .configuration {
        margin-top: var(--space-1);
        font-size: var(--font-size-small);
    }
    .configuration p {
        margin: var(--space-1) 0 0;
    }
    .signature,
    .engine {
        font-family: var(--font-family-mono);
        overflow-wrap: anywhere;
    }
</style>
