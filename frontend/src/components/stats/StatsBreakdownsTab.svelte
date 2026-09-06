<script>
    // Les trois ventilations de #266 (fiche I.10) : les mêmes décisions que
    // les chiffres globaux, découpées par phase de partie, par étiquette et
    // par score. Aucune d'elles ne redéfinit ce qui compte comme une décision
    // — ce serait un second PR sous le même nom.
    import { t } from '../../i18n';
    import { MIN_CELL_DECISIONS } from './gradeBands.js';

    let { result = null } = $props();

    let phases = $derived(result?.PerPhase ?? []);
    let tags = $derived(result?.PerTag ?? []);
    let cells = $derived(result?.PerScore ?? []);

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

    function scoreLabel(cell) {
        if (cell.MoverAway === 0 && cell.OpponentAway === 0) return $t('stats.scoreMoney');
        return `${cell.MoverAway}-${cell.OpponentAway}`;
    }
</script>

<div class="breakdowns">
    <section>
        <h3>{$t('stats.byPhase')}</h3>
        {#if phases.length === 0}
            <p class="empty">{$t('stats.noData')}</p>
        {:else}
            <table>
                <thead>
                    <tr>
                        <th>{$t('stats.phase')}</th>
                        <th class="num">{$t('stats.decisions')}</th>
                        <th class="num">{$t('stats.blunders')}</th>
                        <th class="num">PR</th>
                    </tr>
                </thead>
                <tbody>
                    {#each phases as p (p.Phase)}
                        <tr>
                            <td>{phaseLabel(p.Phase)}</td>
                            <td class="num">{p.NumDecisions}</td>
                            <td class="num">{p.BlunderCount}</td>
                            <td class="num">{p.PR.toFixed(2)}</td>
                        </tr>
                    {/each}
                </tbody>
            </table>
        {/if}
    </section>

    <section>
        <h3>{$t('stats.byTag')}</h3>
        {#if tags.length === 0}
            <p class="empty">{$t('stats.noTags')}</p>
        {:else}
            <table>
                <thead>
                    <tr>
                        <th>{$t('stats.tag')}</th>
                        <th class="num">{$t('stats.decisions')}</th>
                        <th class="num">{$t('stats.blunders')}</th>
                        <th class="num">PR</th>
                    </tr>
                </thead>
                <tbody>
                    {#each tags as tag (tag.Tag)}
                        <tr>
                            <td>{tag.Tag}</td>
                            <td class="num">{tag.NumDecisions}</td>
                            <td class="num">{tag.BlunderCount}</td>
                            <td class="num">{tag.PR.toFixed(2)}</td>
                        </tr>
                    {/each}
                </tbody>
            </table>
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
            <table>
                <thead>
                    <tr>
                        <th>{$t('stats.score')}</th>
                        <th class="num">{$t('stats.decisions')}</th>
                        <th class="num">{$t('stats.blunders')}</th>
                        <th class="num">PR</th>
                    </tr>
                </thead>
                <tbody>
                    {#each cells as cell (`${cell.MoverAway}-${cell.OpponentAway}`)}
                        <!-- Une cellule trop maigre est grisée, jamais cachée :
                             son effectif reste lisible, donc l'omission reste
                             vérifiable. -->
                        <tr class:thin={cell.NumDecisions < MIN_CELL_DECISIONS}>
                            <td>{scoreLabel(cell)}</td>
                            <td class="num">{cell.NumDecisions}</td>
                            <td class="num">{cell.BlunderCount}</td>
                            <td class="num">{cell.PR.toFixed(2)}</td>
                        </tr>
                    {/each}
                </tbody>
            </table>
            <p class="note">{$t('stats.thinCells', { n: MIN_CELL_DECISIONS })}</p>
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
    .empty,
    .note {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
        margin: var(--space-1) 0 0 0;
    }
</style>
