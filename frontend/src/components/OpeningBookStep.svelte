<script>
    // Un temps d'une question d'Ouverture (ADR-0094) : la consigne avant le coup,
    // le verdict du livre après. En cas d'erreur, chaque candidat contre le
    // meilleur : écart, IC 95 % de l'écart, « à égalité » — en équité normalisée
    // au score, la seule échelle qui sorte du moteur (ADR-0019).
    import { t } from '../i18n';

    let { question, verdict = null, revealed = false, outOfTime = false } = $props();

    // La clé du livre lit le score répondeur d'abord : « 4-2 » = le répondeur à
    // 4 points, l'ouvreur à 2. Le joueur au trait est l'ouvreur au premier temps.
    let parts = $derived(question.score === 'money' ? null : question.score.split('-'));
    let you = $derived(parts ? (question.step === 'opening' ? parts[1] : parts[0]) : '');
    let them = $derived(parts ? (question.step === 'opening' ? parts[0] : parts[1]) : '');
    let roll = $derived(question.step === 'opening' ? question.opening : question.reply);
    let showAnalysis = $derived(revealed && verdict && (outOfTime || !verdict.right));

    // Le plafond de demi-coups et la profondeur sont dans la signature du
    // rollout ; la table de bearoff, elle, est dite par le champ du livre (la
    // signature n'énonce que la politique).
    let ply = $derived(/(\d+-ply) plays/.exec(verdict?.signature ?? '')?.[1] ?? '');
    let truncation = $derived(/truncation (\d+) half-moves/.exec(verdict?.signature ?? '')?.[1]);

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
    {#if question.step === 'reply' && question.followsBest}
        <p class="note" data-testid="opening-follows-best">{$t('training.opening.followsBest', { move: question.openingPlay })}</p>
    {:else if question.step === 'reply'}
        <p class="note">{$t('training.opening.after', { move: question.openingPlay })}</p>
    {/if}

    {#if revealed && verdict}
        <p class="verdict" class:correct={!outOfTime && verdict.right} data-testid="training-verdict">
            {#if !outOfTime}
                {#if !verdict.legal}
                    {$t('training.illegal')}
                {:else if verdict.right}
                    {$t('training.opening.right', { move: verdict.notation })}
                {:else if verdict.played < 0}
                    {$t('training.opening.unranked', { move: verdict.notation })}
                {:else}
                    {$t('training.opening.behind', { move: verdict.notation, gap: fixed(verdict.ranked[verdict.played].gap) })}
                {/if}
            {/if}
        </p>
        {#if showAnalysis}
            <table class="ranked" data-testid="opening-ranked">
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
                        <tr class:played={i === verdict.played} class:tied={row.tied}>
                            <td>{row.move}</td>
                            <td class="num">{fixed(row.gap)}</td>
                            <td class="num">{i === 0 ? '—' : `±${fixed(row.ci95)}`}</td>
                            <td>{row.tied ? $t('training.opening.tied') : ''}</td>
                        </tr>
                    {/each}
                </tbody>
            </table>
            <p class="hint">{$t('training.opening.scale')}</p>
        {/if}
        <p class="precision" data-testid="opening-precision" title={verdict.signature}>
            {$t('training.opening.precision', {
                ply,
                truncation: truncation ? $t('training.opening.truncated', { n: truncation }) : $t('training.opening.untruncated'),
                games: verdict.games,
                bearoff: verdict.exact_bearoff ? $t('training.opening.withBearoff') : $t('training.opening.withoutBearoff')
            })}
        </p>
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
    .verdict,
    .hint,
    .precision {
        margin: 0;
    }

    .score,
    .note,
    .hint,
    .precision {
        color: var(--color-text-muted);
    }

    .precision {
        font-size: var(--font-size-small);
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
