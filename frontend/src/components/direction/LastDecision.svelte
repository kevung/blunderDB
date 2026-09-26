<script>
    /*
     * Le dernier geste du directeur, corrigeable sur place en deux clics (ux.md flux F28), ou
     * par `Ctrl+Z`. Corriger n'efface rien : c'est un événement de plus.
     */
    import { t } from '../../i18n';
    import { closeOnEscape } from '../../services/escapeService.js';
    import { isLetter } from '../../utils/keys.js';
    import CorrectionPanel from './CorrectionPanel.svelte';

    let { last = null, busy = false, onCorrect = () => {}, onCancelMatch = () => {} } = $props();

    let open = $state(false);

    /* Ctrl+Z ouvre la reprise du dernier geste. Il n'annule rien tout seul : ce serait défaire
       sans montrer quoi, et le directeur doit voir ce qu'il reprend. */
    /** @param {KeyboardEvent} e */
    function onKey(e) {
        if (!last) return;
        if ((e.ctrlKey || e.metaKey) && !e.shiftKey && isLetter(e, 'z')) {
            e.preventDefault();
            open = true;
        }
    }

    /* Échap ferme la reprise avant tout geste global, où que soit le focus. */
    $effect(() => {
        if (open) return closeOnEscape(() => (open = false));
    });

    /** @type {(winner: string, scoreA: number, scoreB: number) => void} */
    function correct(winner, scoreA, scoreB) {
        onCorrect(last.matchId, winner, scoreA, scoreB);
        open = false;
    }
</script>

<svelte:window onkeydown={onKey} />

{#if last}
    <div class="last" data-testid="direction-last">
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
            <button type="button" data-testid="direction-last-correct" disabled={busy} onclick={() => (open = !open)}>{$t('direction.last.correct')}</button>
        {:else if last.cancellable}
            <button type="button" class="danger" data-testid="direction-last-cancel" disabled={busy} onclick={() => onCancelMatch(last.matchId)}>{$t('direction.last.cancel')}</button>
        {/if}
    </div>

    {#if open && last.correctable}
        <CorrectionPanel a={last.a} b={last.b} aName={last.aName} bName={last.bName} {busy} testid="direction-last" onPick={correct} />
    {/if}
{/if}

<style>
    .last {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        padding: var(--space-1) var(--space-2);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
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

    button.danger {
        border-color: var(--color-danger);
        color: var(--color-danger);
    }

    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>
