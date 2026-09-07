<script>
    /*
     * Le dernier geste du directeur, repris sur place (issue #372, ux.md flux F28).
     *
     * Un directeur est interrompu toutes les deux minutes : il cliquera le mauvais nom, et il
     * s'en apercevra dans la seconde. Ce qui rend cela supportable, c'est que l'erreur se
     * défasse LÀ, sous la file, en deux clics — et que défaire ne soit jamais effacer.
     *
     * `Ctrl+Z` ouvre la même correction. Il n'efface rien : la correction est un événement de
     * plus, et le premier résultat reste dans l'historique là où il a eu lieu.
     */
    import { t } from '../../i18n';

    let { last = null, busy = false, onCorrect = () => {}, onCancelMatch = () => {} } = $props();

    let open = $state(false);
    let scoreA = $state('');
    let scoreB = $state('');

    /* Ctrl+Z ouvre la reprise du dernier geste. Il n'annule rien tout seul : ce serait défaire
       sans montrer quoi, et le directeur doit voir ce qu'il reprend. */
    function onKey(e) {
        if (!last) return;
        if ((e.ctrlKey || e.metaKey) && e.key === 'z') {
            e.preventDefault();
            open = true;
        } else if (e.key === 'Escape' && open) {
            open = false;
        }
    }

    function num(v) {
        const n = parseInt(v, 10);
        return Number.isFinite(n) && n >= 0 ? n : 0;
    }

    function correct(winner) {
        onCorrect(last.matchId, winner, num(scoreA), num(scoreB));
        open = false;
        scoreA = '';
        scoreB = '';
    }
</script>

<svelte:window onkeydown={onKey} />

{#if last}
    <div class="last">
        <span class="what">
            {#if last.correctable}
                {$t('direction.last.result', {
                    winner: last.winnerName,
                    loser: last.winnerName === last.aName ? last.bName : last.aName
                })}
            {:else}
                {$t('direction.last.started', { a: last.aName, b: last.bName })}
            {/if}
        </span>
        <span class="grow"></span>
        {#if last.correctable}
            <button type="button" disabled={busy} onclick={() => (open = !open)}>{$t('direction.last.correct')}</button>
        {:else if last.cancellable}
            <button type="button" class="danger" disabled={busy} onclick={() => onCancelMatch(last.matchId)}>{$t('direction.last.cancel')}</button>
        {/if}
    </div>

    {#if open && last.correctable}
        <div class="correct">
            <span class="hint">{$t('direction.last.whoWon')}</span>
            <button type="button" class="winner" disabled={busy} onclick={() => correct(last.a)}>{last.aName}</button>
            <button type="button" class="winner" disabled={busy} onclick={() => correct(last.b)}>{last.bName}</button>
            <input type="number" min="0" max="99" bind:value={scoreA} />
            <span>–</span>
            <input type="number" min="0" max="99" bind:value={scoreB} />
        </div>
    {/if}
{/if}

<style>
    .last,
    .correct {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        padding: var(--space-1) var(--space-2);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .correct {
        border-top: 1px solid var(--color-border);
    }

    .grow {
        flex: 1;
    }

    button {
        padding: 0.1rem 0.5rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
        font-size: var(--font-size-small);
    }

    button.winner {
        border-color: var(--color-primary);
        font-weight: 600;
        min-height: 30px;
    }

    button.danger {
        border-color: var(--color-danger);
        color: var(--color-danger);
    }

    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    input {
        width: 3.2rem;
        padding: 0.1rem 0.3rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
    }
</style>
