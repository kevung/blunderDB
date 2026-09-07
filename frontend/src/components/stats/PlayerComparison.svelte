<script>
    // Deux joueurs côte à côte (#282, fiche I.26).
    //
    // Le bloc ne calcule rien : `playerComparison.js` décide de ce qui se
    // compare, et lui seul. Ici il n'y a que la mise en page et le repère
    // visuel du meilleur des deux — jamais sur une ligne de contexte.
    import { t } from '../../i18n/index.js';
    import { compareRows } from '../../services/playerComparison.js';

    /**
     * @type {{ a: any, b: any, onClear?: () => void }}
     */
    let { a, b, onClear = undefined } = $props();

    let lines = $derived(compareRows(a, b));
</script>

<section class="comparison" aria-label={$t('stats.compareTitle', { a: a?.name ?? '', b: b?.name ?? '' })}>
    <header>
        <span class="who">{a?.name ?? ''} <span class="vs">/</span> {b?.name ?? ''}</span>
        <button type="button" onclick={() => onClear?.()}>{$t('stats.compareClear')}</button>
    </header>
    <table>
        <tbody>
            {#each lines as line (line.key)}
                <tr class:context={line.kind === 'context'}>
                    <th scope="row">{$t(line.labelKey)}</th>
                    <td class="value" class:better={line.better === 'a'}>{line.a}</td>
                    <td class="value" class:better={line.better === 'b'}>{line.b}</td>
                </tr>
            {/each}
        </tbody>
    </table>
</section>

<style>
    .comparison {
        border: 1px solid var(--color-border);
        border-radius: 4px;
        padding: 0.4rem 0.5rem;
    }

    header {
        display: flex;
        align-items: baseline;
        justify-content: space-between;
        gap: 0.5rem;
        margin-bottom: 0.3rem;
    }

    .who {
        font-weight: 600;
    }

    /* La barre oblique sépare deux noms sans les opposer : le bloc compare des
       mesures, il n'organise pas un duel. */
    .vs {
        opacity: 0.5;
        padding: 0 0.15rem;
    }

    table {
        border-collapse: collapse;
        width: 100%;
    }

    th {
        text-align: left;
        font-weight: 400;
        opacity: 0.85;
        white-space: nowrap;
    }

    td.value {
        text-align: right;
        font-variant-numeric: tabular-nums;
        width: 6.5rem;
        white-space: nowrap;
    }

    /* Le meilleur des deux est mis en avant par le POIDS, pas par une couleur :
       une comparaison lue en noir et blanc, ou par un daltonien, doit dire la
       même chose (ADR-0031). */
    td.better {
        font-weight: 700;
    }

    /* Une ligne de contexte situe les taux au-dessus ; elle ne se lit pas
       comme un score, et son gris le dit. */
    tr.context th,
    tr.context td {
        opacity: 0.7;
    }
</style>
