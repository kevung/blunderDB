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
    import { statsFilterStore } from '../../stores/statsStore.js';
    import { loadPositionsFromStatsSelection } from '../../services/positionLoader.js';
    import { GetStatsBreakdownPositionCounts } from '../../../wailsjs/go/database/Database.js';

    let { result = null } = $props();

    let phases = $derived(result?.PerPhase ?? []);
    let tags = $derived(result?.PerTag ?? []);
    let cells = $derived(result?.PerScore ?? []);
    // Le plan de jeu. Les lignes « inconnu » sont écartées : sur une
    // base dont les plans n'ont jamais été calculés elles seraient TOUTE la
    // table, et elles ne diraient rien qu'un `blunderdb repair` ne règle.
    let gameTypes = $derived((result?.PerGameType ?? []).filter((/** @type {any} */ g) => g.GameType !== 'unknown'));

    /** @type {Record<string, Record<string, {Positions: number, Blunders: number}>> | null} */
    let counts = $state(null);
    let countsFor = 0;

    // Recompté à chaque résultat, avec le filtre qui l'a produit ; une réponse
    // devancée par un résultat plus récent est ignorée.
    $effect(() => {
        if (!result) return;
        const ticket = ++countsFor;
        counts = null;
        Promise.resolve(GetStatsBreakdownPositionCounts(/** @type {any} */ (get(statsFilterStore))))
            .then((c) => {
                if (ticket === countsFor) counts = c ?? {};
            })
            .catch(() => {
                if (ticket === countsFor) counts = {};
            });
    });

    /** @param {string} dim @param {string} key */
    function countOf(dim, key) {
        return counts?.[dim]?.[key] ?? null;
    }

    /** @param {string} dim @param {string} key @param {boolean} onlyBlunders */
    function open(dim, key, onlyBlunders) {
        loadPositionsFromStatsSelection(get(statsFilterStore), { Kind: 'breakdown', Breakdown: dim, BreakdownKey: key, OnlyBlunders: onlyBlunders });
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
        {@const n = c == null ? null : onlyBlunders ? c.Blunders : c.Positions}
        <td class="num">
            {#if n == null}
                <span class="pending">…</span>
            {:else if n === 0}
                <span class="zero">0</span>
            {:else}
                <button
                    type="button"
                    class="count-link"
                    data-testid="breakdown-{onlyBlunders ? 'blunders' : 'positions'}-{dim}-{key}"
                    onclick={() => open(dim, key, onlyBlunders)}
                    title={onlyBlunders ? $t('stats.breakdownOpenBlunders', { n, decisions }) : $t('stats.breakdownOpenPositions', { n, decisions })}>{n}</button
                >
            {/if}
        </td>
    {/snippet}

    {#snippet table(/** @type {string} */ dim, /** @type {string} */ heading, /** @type {any[]} */ rows)}
        <table>
            <thead>
                <tr>
                    <th>{heading}</th>
                    <th class="num">{$t('stats.positions')}</th>
                    <th class="num" title={$t('stats.breakdownBlundersHint')}>{$t('stats.blunders')}</th>
                    <th class="num">{$t('stats.decisions')}</th>
                    <th class="num">PR</th>
                    <th class="num" title={$t('stats.intervalHint')}>{$t('stats.interval95')}</th>
                </tr>
            </thead>
            <tbody>
                {#each rows as r (r.key)}
                    <!-- Une ligne trop maigre est grisée, jamais cachée : son
                         effectif reste lisible, donc l'omission reste vérifiable. -->
                    <tr class:thin={!hasInterval(r.row.PRInterval)}>
                        <td>{r.label}</td>
                        {@render countCell(dim, r.key, false, r.row.NumDecisions)}
                        {@render countCell(dim, r.key, true, r.row.BlunderCount)}
                        <td class="num decisions">{r.row.NumDecisions}</td>
                        <td class="num">{r.row.PR.toFixed(2)}</td>
                        <td class="num ci">{fmtPRInterval(r.row.PRInterval)}</td>
                    </tr>
                {/each}
            </tbody>
        </table>
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
        {#if cells.length === 0}
            <p class="empty">{$t('stats.noData')}</p>
        {:else}
            {@render table(
                'score',
                $t('stats.score'),
                cells.map((/** @type {any} */ cell) => ({ key: scoreKey(cell), label: scoreLabel(cell), row: cell }))
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
    table {
        width: 100%;
        border-collapse: collapse;
    }
    th,
    td {
        text-align: left;
        padding: 2px var(--space-2);
        border-bottom: 1px solid var(--color-border);
    }
    th {
        color: var(--color-text-muted);
        font-weight: 400;
    }
    .num {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }
    .thin td {
        color: var(--color-text-muted);
    }
    .decisions,
    .pending,
    .zero {
        color: var(--color-text-muted);
    }
    /* Le chiffre est le lien : même convention que les compteurs de la barre d'état. */
    .count-link {
        background: none;
        border: none;
        padding: 0;
        color: inherit;
        cursor: pointer;
        font-variant-numeric: tabular-nums;
        text-decoration: underline dotted;
    }
    .count-link:hover,
    .count-link:focus-visible {
        color: var(--color-primary);
    }
    .empty,
    .note {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
        margin: var(--space-1) 0 0 0;
    }
</style>
