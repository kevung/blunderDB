<script>
    // La fiche de score (ADR-0040 règle 4) : un score non ordonné, ses deux
    // faces côte à côte, et seulement les cases que les tables de référence
    // définissent.
    //
    // Les deux faces sont là parce qu'une décision de videau au score a besoin
    // des deux : le point de prise corrigé combine les valeurs de gammon des
    // deux joueurs, et c'est le point de prise de l'adversaire qui dit si le
    // double passe. À score égal il n'y a qu'une colonne — les deux liraient
    // les mêmes cases.
    //
    // Un composant, deux hôtes : l'exercice Scores aujourd'hui, la carte Anki
    // d'un paquet de scores demain (ADR-0042). D'où les propriétés, qui ne
    // parlent que de la fiche et jamais de la session.
    import { t } from '../i18n';
    import TrainingNumberCell from './TrainingNumberCell.svelte';
    import { numberTypeLabelKey } from '../services/trainingLabels.js';

    let { card, numbers = [], revealed = false, faults = [], locked = false, onToggle = () => {} } = $props();
</script>

<table class="score-card">
    <caption>{$t('training.scoreCaption', { a: card.awayYou, b: card.awayOpponent })}</caption>
    <thead>
        <tr>
            <th scope="col"></th>
            {#each card.faces as face (face.face)}
                <th scope="col">{face.face === 'you' ? $t('training.faceYou') : $t('training.faceOpponent')}</th>
            {/each}
        </tr>
    </thead>
    <tbody>
        {#each card.rows as row (row.type)}
            <tr>
                <th scope="row">{$t(numberTypeLabelKey(row.type))}</th>
                {#each row.cells as cell, i (i)}
                    <td>
                        {#if cell}
                            <TrainingNumberCell
                                value={cell.value}
                                precision={cell.precision}
                                {revealed}
                                {locked}
                                fault={faults[numbers.indexOf(cell)] === true}
                                label={$t(numberTypeLabelKey(row.type))}
                                onToggle={() => onToggle(numbers.indexOf(cell))}
                            />
                        {/if}
                    </td>
                {/each}
            </tr>
        {/each}
    </tbody>
</table>

<style>
    .score-card {
        border-collapse: collapse;
    }

    caption {
        text-align: left;
        padding-bottom: 0.3em;
        font-weight: 600;
    }

    th,
    td {
        padding: 0.15em 0.4em;
        text-align: right;
        font-size: var(--font-size-base);
        font-weight: normal;
    }

    thead th {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    tbody th {
        text-align: left;
        white-space: nowrap;
        color: var(--color-text-muted);
    }
</style>
