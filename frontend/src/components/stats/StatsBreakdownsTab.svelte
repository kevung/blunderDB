<script>
    // Les ventilations, plus celle par plan de jeu : les mêmes décisions que
    // les chiffres globaux, découpées par phase de partie, par étiquette et
    // par score. Aucune d'elles ne redéfinit ce qui compte comme une décision
    // — ce serait un second PR sous le même nom.
    //
    // Les colonnes Positions et Bourdes ouvrent ce qu'elles comptent : chaque
    // chiffre est la longueur de la liste que charge la sélection « breakdown »
    // (GetStatsBreakdownPositionCounts), une position atteinte par plusieurs
    // décisions n'y comptant qu'une fois. Les décisions restent affichées :
    // c'est le dénominateur du PR.
    import { get } from 'svelte/store';
    import { t } from '../../i18n';
    import { fmtPRInterval, hasInterval } from '../../utils/interval.js';
    import { statsFilterStore, statsResultFilterStore } from '../../stores/statsStore.js';
    import { loadPositionsFromStatsSelection } from '../../services/positionLoader.js';
    import { GetStatsBreakdownPositionCounts } from '../../../wailsjs/go/database/Database.js';
    import PanelTable from '../panels/PanelTable.svelte';
    import CountLink from '../panels/CountLink.svelte';

    let { result = null } = $props();

    let phases = $derived(result?.PerPhase ?? []);
    let tags = $derived(result?.PerTag ?? []);
    let scoreCells = $derived(result?.PerScore ?? []);
    // Le plan de jeu. Les lignes « inconnu » sont écartées : sur une
    // base dont les plans n'ont jamais été calculés elles seraient TOUTE la
    // table, et elles ne diraient rien qu'un `blunderdb repair` ne règle.
    let gameTypes = $derived((result?.PerGameType ?? []).filter((/** @type {any} */ g) => g.GameType !== 'unknown'));

    /** @type {Record<string, Record<string, {Positions: number, Blunders: number}>> | null} */
    let counts = $state(null);
    /** Why the counts could not be read; shown on every cell instead of a wait. */
    let countsError = $state('');
    let countsFor = 0;

    // Les lignes viennent du résultat : on compte et on ouvre sous le filtre
    // qui l'a produit, pas sous celui de la barre, qui peut l'avoir devancé.
    function resultFilter() {
        return get(statsResultFilterStore) ?? get(statsFilterStore);
    }

    // Recompté à chaque résultat ; une réponse devancée par un résultat plus
    // récent est ignorée.
    $effect(() => {
        if (!result) return;
        const ticket = ++countsFor;
        counts = null;
        countsError = '';
        Promise.resolve()
            .then(() => GetStatsBreakdownPositionCounts(/** @type {any} */ (resultFilter())))
            .then((c) => {
                if (ticket === countsFor) counts = c ?? {};
            })
            .catch((/** @type {any} */ err) => {
                if (ticket === countsFor) countsError = err?.message ?? String(err);
            });
    });

    /**
     * The counts of one row: null while they are read, 'missing' when the
     * reply has no such row — the two reads disagree, which must show.
     * @param {string} dim @param {string} key
     */
    function countOf(dim, key) {
        if (counts == null) return null;
        return counts[dim]?.[key] ?? 'missing';
    }

    /** @param {string} dim @param {string} key @param {boolean} onlyBlunders */
    function open(dim, key, onlyBlunders) {
        loadPositionsFromStatsSelection(resultFilter(), { Kind: 'breakdown', Breakdown: dim, BreakdownKey: key, OnlyBlunders: onlyBlunders });
    }

    /**
     * One count of a row, or null while it is read or when it cannot be.
     * @param {string} dim
     * @param {string} key
     * @param {boolean} onlyBlunders
     */
    function countValue(dim, key, onlyBlunders) {
        const c = countOf(dim, key);
        return c == null || c === 'missing' ? null : onlyBlunders ? c.Blunders : c.Positions;
    }

    /**
     * The columns of one breakdown table; sorted and filtered by PanelTable.
     * @param {string} dim
     * @param {string} heading
     */
    function columnsFor(dim, heading) {
        return [
            { key: 'label', label: heading, sortable: true, filter: /** @type {const} */ ('text'), value: (/** @type {any} */ r) => r.label },
            {
                key: 'positions',
                label: $t('stats.positions'),
                sortable: true,
                defaultDir: 'desc',
                align: 'right',
                filter: /** @type {const} */ ('number'),
                value: (/** @type {any} */ r) => countValue(dim, r.key, false)
            },
            {
                key: 'blunders',
                label: $t('stats.blunders'),
                title: $t('stats.breakdownBlundersHint'),
                sortable: true,
                defaultDir: 'desc',
                align: 'right',
                filter: /** @type {const} */ ('number'),
                value: (/** @type {any} */ r) => countValue(dim, r.key, true)
            },
            {
                key: 'decisions',
                label: $t('stats.decisions'),
                sortable: true,
                defaultDir: 'desc',
                align: 'right',
                filter: /** @type {const} */ ('number'),
                value: (/** @type {any} */ r) => r.row.NumDecisions
            },
            { key: 'pr', label: 'PR', sortable: true, align: 'right', filter: /** @type {const} */ ('number'), value: (/** @type {any} */ r) => r.row.PR },
            { key: 'interval', label: $t('stats.interval95'), title: $t('stats.intervalHint'), align: 'right' }
        ];
    }

    /** @param {string} phase */
    function phaseLabel(phase) {
        switch (phase) {
            case 'opening':
                return $t('stats.phaseOpening');
            case 'middlegame':
                return $t('stats.phaseMiddlegame');
            case 'race':
                return $t('stats.phaseRace');
            case 'bearoff':
                return $t('stats.phaseBearoff');
            default:
                return $t('stats.phaseUnknown');
        }
    }

    /** @param {string} gameType */
    function gameTypeLabel(gameType) {
        const key = `stats.gameType_${gameType}`;
        const label = $t(key);
        return label === key ? gameType : label;
    }

    /** @param {any} cell */
    function scoreLabel(cell) {
        if (cell.Money) return $t('stats.scoreMoney');
        return `${cell.MoverAway}-${cell.OpponentAway}`;
    }

    /** The key a score cell is counted under (storage.BreakdownScore). @param {any} cell */
    function scoreKey(cell) {
        return cell.Money ? 'money' : `${cell.MoverAway}-${cell.OpponentAway}`;
    }
</script>

<div class="breakdowns">
    {#snippet countCell(/** @type {string} */ dim, /** @type {string} */ key, /** @type {boolean} */ onlyBlunders, /** @type {number} */ decisions)}
        {@const c = countOf(dim, key)}
        {@const n = c == null || c === 'missing' ? null : onlyBlunders ? c.Blunders : c.Positions}
        <td class="align-right">
            {#if countsError || c === 'missing'}
                <span
                    class="count-error"
                    data-testid="breakdown-error-{onlyBlunders ? 'blunders' : 'positions'}-{dim}-{key}"
                    title={countsError ? $t('stats.breakdownCountsError', { error: countsError }) : $t('stats.breakdownCountsMissing')}>?</span
                >
            {:else if n == null}
                <span class="pending">…</span>
            {:else if n === 0}
                <span class="zero">0</span>
            {:else}
                <CountLink
                    label={n}
                    testid="breakdown-{onlyBlunders ? 'blunders' : 'positions'}-{dim}-{key}"
                    onclick={() => open(dim, key, onlyBlunders)}
                    title={onlyBlunders ? $t('stats.breakdownOpenBlunders', { n, decisions }) : $t('stats.breakdownOpenPositions', { n, decisions })}
                />
            {/if}
        </td>
    {/snippet}

    {#snippet table(/** @type {string} */ dim, /** @type {string} */ heading, /** @type {any[]} */ rows)}
        <PanelTable viewId="stats.breakdown.{dim}" {rows} rowKey={(r) => r.key} columns={columnsFor(dim, heading)} rowClass={(r) => (hasInterval(r.row.PRInterval) ? '' : 'thin')}>
            <!-- Une ligne trop maigre est grisée, jamais cachée : son
                 effectif reste lisible, donc l'omission reste vérifiable. -->
            {#snippet cells(/** @type {any} */ r)}
                <td>{r.label}</td>
                {@render countCell(dim, r.key, false, r.row.NumDecisions)}
                {@render countCell(dim, r.key, true, r.row.BlunderCount)}
                <td class="align-right decisions">{r.row.NumDecisions}</td>
                <td class="align-right">{r.row.PR.toFixed(2)}</td>
                <td class="align-right ci">{fmtPRInterval(r.row.PRInterval)}</td>
            {/snippet}
        </PanelTable>
    {/snippet}

    <section>
        <h3>{$t('stats.byPhase')}</h3>
        {#if phases.length === 0}
            <p class="empty">{$t('stats.noData')}</p>
        {:else}
            {@render table(
                'phase',
                $t('stats.phase'),
                phases.map((/** @type {any} */ p) => ({ key: p.Phase, label: phaseLabel(p.Phase), row: p }))
            )}
        {/if}
    </section>

    <section>
        <h3>{$t('stats.byGameType')}</h3>
        {#if gameTypes.length === 0}
            <p class="empty">{$t('stats.noGameTypes')}</p>
        {:else}
            {@render table(
                'game_type',
                $t('stats.gameType'),
                gameTypes.map((/** @type {any} */ g) => ({ key: g.GameType, label: gameTypeLabel(g.GameType), row: g }))
            )}
        {/if}
    </section>

    <section>
        <h3>{$t('stats.byTag')}</h3>
        {#if tags.length === 0}
            <p class="empty">{$t('stats.noTags')}</p>
        {:else}
            {@render table(
                'tag',
                $t('stats.tag'),
                tags.map((/** @type {any} */ tag) => ({ key: tag.Tag, label: tag.Tag, row: tag }))
            )}
            <!-- Une étiquette qualifie, elle ne partitionne pas : le dire est
                 la différence entre un lecteur qui fait confiance à la colonne
                 et un lecteur qui l'additionne et conclut que l'outil ment. -->
            <p class="note">{$t('stats.tagsDoNotSum')}</p>
        {/if}
    </section>

    <section>
        <h3>{$t('stats.byScore')}</h3>
        {#if scoreCells.length === 0}
            <p class="empty">{$t('stats.noData')}</p>
        {:else}
            {@render table(
                'score',
                $t('stats.score'),
                scoreCells.map((/** @type {any} */ cell) => ({ key: scoreKey(cell), label: scoreLabel(cell), row: cell }))
            )}
            <p class="note">{$t('stats.thinCells')}</p>
        {/if}
    </section>
</div>

<style>
    .breakdowns {
        display: flex;
        flex-direction: column;
        gap: var(--space-4);
        overflow-y: auto;
    }
    h3 {
        font-size: var(--font-size-base);
        margin: 0 0 var(--space-1) 0;
    }
    .breakdowns :global(.thin td) {
        color: var(--color-text-muted);
    }
    .decisions,
    .pending,
    .zero {
        color: var(--color-text-muted);
    }
    .count-error {
        color: var(--color-danger, #c0392b);
        cursor: help;
    }
    .empty,
    .note {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
        margin: var(--space-1) 0 0 0;
    }
</style>
