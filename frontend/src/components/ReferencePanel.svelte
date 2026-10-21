<script>
    // Les tables du score, consultées sans base : MET Kazaross-XG2, points de prise,
    // valeurs du gammon, leur vue agrégée et les nombres de Neil. Les données viennent de
    // services/referenceTables.js et stores/metTable ; l'onglet affiché, de referenceStore.
    import { t } from '../i18n';
    import PanelHeader from './panels/PanelHeader.svelte';
    import { REFERENCE_TABLES, referenceValue } from '../services/referenceTables.js';
    import { tableData as metTable } from '../stores/metTable';
    import { NEIL_ANCHORS, neilEstimate, formatNeilNumber } from '../services/neilNumbers.js';
    import { REFERENCE_SECTIONS, referenceSectionStore } from '../stores/referenceStore.js';

    const sectionLabels = $derived({
        met: $t('reference.met'),
        takePoints: $t('reference.takePoints'),
        gammonValues: $t('reference.gammonValues'),
        aggregated: $t('reference.aggregated'),
        neil: $t('reference.neil')
    });

    const NEIL_SOURCE = 'https://bkgm.com/articles/GOL/demo/equity.htm';
    const NEIL_MEMO = '8 is 6, 11 is 5, 15 is 4';

    const takePointGroups = [
        { cube: 2, live: 'tp2.live', last: 'tp2.last' },
        { cube: 4, live: 'tp4.live', last: 'tp4.last' }
    ];
    const gammonValueTables = [
        { cube: 1, id: 'gv1' },
        { cube: 2, id: 'gv2' },
        { cube: 4, id: 'gv4' }
    ];
    const aggregatedAway = [2, 3, 4, 5, 6, 7, 8, 9];
    const neilAway = Array.from({ length: 14 }, (_, i) => i + 2);

    /** @param {string} id @param {number} you @param {number} opp */
    function look(id, you, opp) {
        const value = referenceValue(id, you, opp);
        return value == null ? '—' : value.toFixed(REFERENCE_TABLES[id].precision);
    }

    /** @param {string} id */
    function rowsOf(id) {
        const table = REFERENCE_TABLES[id];
        return table.data.map((_, r) => r + table.rowOffset);
    }

    /** @param {string} id */
    function colsOf(id) {
        const table = REFERENCE_TABLES[id];
        return Array.from({ length: table.colCount }, (_, c) => c + table.colOffset);
    }

    /** @param {number} gap */
    function formatGap(gap) {
        const text = Math.abs(gap).toFixed(1);
        if (text === '0.0') return '0.0';
        return `${gap > 0 ? '+' : '−'}${text}`;
    }
</script>

{#snippet grid(/** @type {string} */ id)}
    <table class="ref-grid">
        <thead>
            <tr>
                <th></th>
                {#each colsOf(id) as opp (opp)}
                    <th>{opp}</th>
                {/each}
            </tr>
        </thead>
        <tbody>
            {#each rowsOf(id) as you (you)}
                <tr>
                    <th>{you}</th>
                    {#each colsOf(id) as opp (opp)}
                        <td>{look(id, you, opp)}</td>
                    {/each}
                </tr>
            {/each}
        </tbody>
    </table>
{/snippet}

<div class="reference-panel" data-testid="reference-panel">
    <PanelHeader title={$t('reference.title')}>
        <div class="segmented" role="tablist" aria-label={$t('reference.title')}>
            {#each REFERENCE_SECTIONS as section (section)}
                <button
                    type="button"
                    role="tab"
                    class="sub-tab-btn"
                    class:active={$referenceSectionStore === section}
                    aria-selected={$referenceSectionStore === section}
                    data-testid="reference-tab-{section}"
                    onclick={() => referenceSectionStore.set(section)}>{sectionLabels[section]}</button
                >
            {/each}
        </div>
    </PanelHeader>

    <div class="reference-body" data-testid="reference-section-{$referenceSectionStore}">
        {#if $referenceSectionStore === 'met'}
            <p class="hint">{$t('reference.metLegend')} · {$t('reference.axisHint')}</p>
            <div class="scroll">
                <table class="ref-grid">
                    <thead>
                        <tr>
                            <th></th>
                            {#each metTable[0] as _, c (c)}
                                <th>{c + 1}</th>
                            {/each}
                        </tr>
                    </thead>
                    <tbody>
                        {#each metTable as row, r (r)}
                            <tr>
                                <th>{r + 1}</th>
                                {#each row as cell, c (c)}
                                    <td>{cell.toFixed(1)}</td>
                                {/each}
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {:else if $referenceSectionStore === 'takePoints'}
            <p class="hint">{$t('reference.axisHint')}</p>
            {#each takePointGroups as group (group.cube)}
                <h3>{$t('reference.cube', { n: group.cube })}</h3>
                <div class="scroll">
                    <div class="side-by-side">
                        <div>
                            <h4>{$t('reference.longRaces')}</h4>
                            {@render grid(group.live)}
                        </div>
                        <div>
                            <h4>{$t('reference.lastRoll')}</h4>
                            {@render grid(group.last)}
                        </div>
                    </div>
                </div>
            {/each}
        {:else if $referenceSectionStore === 'gammonValues'}
            <p class="hint">{$t('reference.axisHint')}</p>
            {#each gammonValueTables as table (table.id)}
                <h3>{$t('reference.cube', { n: table.cube })}</h3>
                <div class="scroll">{@render grid(table.id)}</div>
            {/each}
        {:else if $referenceSectionStore === 'aggregated'}
            <p class="hint">{$t('reference.axisHint')}</p>
            <div class="scroll">
                <table class="ref-grid aggregated">
                    <thead>
                        <tr>
                            <th></th>
                            {#each aggregatedAway as opp (opp)}
                                <th>{opp}</th>
                            {/each}
                        </tr>
                    </thead>
                    <tbody>
                        {#each aggregatedAway as you (you)}
                            <tr>
                                <th>{you}</th>
                                {#each aggregatedAway as opp (opp)}
                                    <td data-testid="aggregated-{you}-{opp}">
                                        <div>{look('tp2.live', you, opp)} / {look('tp2.last', you, opp)}</div>
                                        <div>{look('tp4.live', you, opp)} / {look('tp4.last', you, opp)}</div>
                                        <div class="muted">{look('gv1', you, opp)} · {look('gv2', you, opp)} · {look('gv4', you, opp)}</div>
                                    </td>
                                {/each}
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
        {:else}
            <p class="hint" title={NEIL_MEMO}>{$t('reference.neilFormula')}</p>
            <div class="scroll">
                <table class="ref-grid">
                    <tbody>
                        <tr>
                            <th>n</th>
                            {#each NEIL_ANCHORS as [away] (away)}
                                <th>{away}</th>
                            {/each}
                        </tr>
                        <tr>
                            <th>Neil</th>
                            {#each NEIL_ANCHORS as [away, value] (away)}
                                <td data-testid="neil-number-{away}">{formatNeilNumber(value)}</td>
                            {/each}
                        </tr>
                    </tbody>
                </table>
            </div>
            <p class="hint">{$t('reference.neilAxis')}</p>
            <div class="scroll">
                <table class="ref-grid neil">
                    <thead>
                        <tr>
                            <th></th>
                            {#each neilAway as trailer (trailer)}
                                <th>{trailer}</th>
                            {/each}
                        </tr>
                    </thead>
                    <tbody>
                        {#each neilAway as leader (leader)}
                            <tr>
                                <th>{leader}</th>
                                {#each neilAway as trailer (trailer)}
                                    {@const cell = neilEstimate(leader, trailer)}
                                    {#if cell}
                                        <td class:muted={cell.fallback} title={cell.fallback ? $t('reference.neilFallback') : undefined} data-testid="neil-{leader}-{trailer}">
                                            <div>{cell.estimate.toFixed(1)}</div>
                                            {#if cell.gap != null}
                                                <div class="gap" data-testid="neil-gap-{leader}-{trailer}">{formatGap(cell.gap)}</div>
                                            {/if}
                                        </td>
                                    {:else}
                                        <td></td>
                                    {/if}
                                {/each}
                            </tr>
                        {/each}
                    </tbody>
                </table>
            </div>
            <p class="source">{$t('reference.source')} <a href={NEIL_SOURCE} target="_blank" rel="noreferrer">Kit Woolsey, GammOnLine 1999</a></p>
        {/if}
    </div>
</div>

<style>
    .reference-panel {
        display: flex;
        flex-direction: column;
        height: 100%;
        min-height: 0;
        font-size: var(--font-size-small);
        color: var(--color-text);
    }
    .reference-body {
        flex: 1;
        min-height: 0;
        overflow-y: auto;
        padding: 6px 10px;
        box-sizing: border-box;
    }
    .segmented {
        display: inline-flex;
        flex-shrink: 0;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        overflow: hidden;
    }
    .sub-tab-btn {
        border: none;
        background: var(--color-surface);
        padding: 2px var(--space-2);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        cursor: pointer;
        white-space: nowrap;
    }
    .sub-tab-btn + .sub-tab-btn {
        border-left: 1px solid var(--color-border);
    }
    .sub-tab-btn:hover {
        color: var(--color-text);
    }
    .sub-tab-btn.active {
        color: var(--color-text);
        font-weight: 600;
        background: color-mix(in srgb, var(--color-text) 10%, var(--color-surface));
    }
    .hint,
    .source {
        margin: 0 0 4px;
        color: var(--color-text-muted);
    }
    .source {
        margin-top: 6px;
    }
    h3 {
        margin: 8px 0 4px;
        font-size: var(--font-size-base);
        font-weight: 600;
    }
    h4 {
        margin: 0 0 2px;
        font-weight: 500;
        color: var(--color-text-muted);
    }
    /* Natural width, so a narrow panel scrolls instead of squeezing digits onto each other. */
    .scroll {
        overflow-x: auto;
    }
    .side-by-side {
        display: grid;
        grid-template-columns: max-content max-content;
        gap: 12px;
        width: max-content;
    }
    .ref-grid {
        width: max-content;
        border-collapse: collapse;
        font-variant-numeric: tabular-nums;
    }
    .ref-grid th,
    .ref-grid td {
        border: 1px solid var(--color-border);
        padding: 2px 5px;
        text-align: center;
        white-space: nowrap;
    }
    .ref-grid th {
        font-weight: 600;
    }
    .ref-grid tbody tr:nth-child(even) {
        background: var(--color-surface-alt);
    }
    .aggregated td,
    .neil td {
        line-height: 1.2;
    }
    .muted,
    .gap {
        color: var(--color-text-muted);
    }
</style>
