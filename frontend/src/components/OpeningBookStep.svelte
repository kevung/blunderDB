<script>
    // Une question d'Ouverture (ADR-0094) : la consigne avant le coup ; après,
    // le tableau du livre et une marque juste/faux, sans prose. Chaque candidat
    // contre le meilleur : écart, IC 95 % de l'écart, « à égalité » — en équité
    // normalisée au score, la seule échelle qui sorte du moteur (ADR-0019).
    import { t } from '../i18n';

    let { question, verdict = null, revealed = false, outOfTime = false } = $props();

    // La clé du livre lit le score répondeur d'abord : « 4-2 » = le répondeur à
    // 4 points, l'ouvreur à 2. Le joueur au trait est l'ouvreur au 1st roll.
    let parts = $derived(question.score === 'money' ? null : question.score.split('-'));
    let you = $derived(parts ? (question.step === 'opening' ? parts[1] : parts[0]) : '');
    let them = $derived(parts ? (question.step === 'opening' ? parts[0] : parts[1]) : '');
    let roll = $derived(question.step === 'opening' ? question.opening : question.reply);
    // Un coup légal que le livre ne classe pas : une ligne de plus, sans écart.
    let unranked = $derived(revealed && verdict && !outOfTime && verdict.legal && verdict.played < 0 ? verdict.notation : '');

    // La précision du rollout reste à portée, en infobulle du tableau.
    let ply = $derived(/(\d+-ply) plays/.exec(verdict?.signature ?? '')?.[1] ?? '');
    let truncation = $derived(/truncation (\d+) half-moves/.exec(verdict?.signature ?? '')?.[1]);
    let precision = $derived(
        verdict
            ? $t('training.opening.precision', {
                  ply,
                  truncation: truncation ? $t('training.opening.truncated', { n: truncation }) : $t('training.opening.untruncated'),
                  games: verdict.games,
                  bearoff: verdict.exact_bearoff ? $t('training.opening.withBearoff') : $t('training.opening.withoutBearoff')
              })
            : ''
    );

    /** @param {number} x */
    function fixed(x) {
        return (x ?? 0).toFixed(3);
    }
</script>

<div class="opening-step" data-testid="opening-step">
    <p class="prompt" data-testid="opening-prompt">
        {$t(question.step === 'opening' ? 'training.opening.promptOpening' : 'training.opening.promptReply', { roll })}
        <span class="score">
            {#if parts}
                {$t('training.opening.score', { you, them })}
            {:else}
                {$t('training.opening.money')}
            {/if}
        </span>
    </p>
    {#if question.step === 'reply'}
        <p class="note" data-testid="opening-after">{$t('training.opening.after', { move: question.openingPlay })}</p>
    {/if}

    {#if revealed && verdict}
        {#if !outOfTime}
            <p class="verdict" class:correct={verdict.right} data-testid="training-verdict">
                {#if !verdict.legal}
                    {$t('training.illegal')}
                {:else}
                    <span class="mark" aria-hidden="true">{verdict.right ? '✓' : '✗'}</span>
                    {$t(verdict.right ? 'training.opening.markRight' : 'training.opening.markWrong')}
                {/if}
            </p>
        {/if}
        <table class="ranked" data-testid="opening-ranked" title={precision}>
            <thead>
                <tr>
                    <th>{$t('training.opening.colPlay')}</th>
                    <th class="num">{$t('training.opening.colGap')}</th>
                    <th class="num">{$t('training.opening.colCi')}</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                {#each verdict.ranked as row, i (row.move)}
                    <tr class:played={!outOfTime && i === verdict.played} class:tied={row.tied}>
                        <td>{row.move}</td>
                        <td class="num">{fixed(row.gap)}</td>
                        <td class="num">{i === 0 ? '—' : `±${fixed(row.ci95)}`}</td>
                        <td>{row.tied ? $t('training.opening.tied') : ''}</td>
                    </tr>
                {/each}
                {#if unranked}
                    <tr class="played" data-testid="opening-unranked">
                        <td>{unranked}</td>
                        <td class="num">—</td>
                        <td class="num">—</td>
                        <td></td>
                    </tr>
                {/if}
            </tbody>
        </table>
    {/if}
</div>

<style>
    .opening-step {
        display: flex;
        flex-direction: column;
        gap: 0.4em;
    }

    .prompt,
    .note,
    .verdict {
        margin: 0;
    }

    .score,
    .note {
        color: var(--color-text-muted);
    }

    .verdict {
        color: var(--color-danger);
    }

    /* Pas de jeton « succès » (ADR-0031) : l'accent suffit. */
    .verdict.correct {
        color: var(--color-primary);
    }

    /* Colonnes au contenu, à gauche du panneau, comme le tableau des candidats. */
    .ranked {
        align-self: flex-start;
        width: auto;
        border-collapse: collapse;
        font-variant-numeric: tabular-nums;
    }

    .ranked th {
        text-align: left;
        font-weight: normal;
        color: var(--color-text-muted);
        padding: 2px 10px;
        white-space: nowrap;
    }

    .ranked td {
        padding: 2px 10px;
        white-space: nowrap;
    }

    .ranked th.num {
        text-align: right;
    }

    .ranked .num {
        text-align: right;
    }

    .ranked tr.tied td:first-child {
        font-weight: bold;
    }

    .ranked tr.played td {
        color: var(--color-primary);
    }
</style>
