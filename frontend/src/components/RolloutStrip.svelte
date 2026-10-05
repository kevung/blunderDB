<script>
    // Le rollout en cours, en une barre fine sous la table des coups : parties jouées, Annuler.
    // Elle n'existe que pendant le rollout ; un refus ou un échec s'y lit quelques secondes.
    import { t } from '../i18n';
    import { rolloutStore } from '../stores/rolloutStore.js';
    import { cancelRollout } from '../services/rolloutService.js';

    /** @type {{ positionId?: number, errorMs?: number }} */
    let { positionId = 0, errorMs = 6000 } = $props();

    let rollout = $derived($rolloutStore);
    let percent = $derived(rollout.maxGames > 0 ? Math.min(100, Math.round((100 * rollout.games) / rollout.maxGames)) : 0);
    let elsewhere = $derived(rollout.running && rollout.kind === 'position' && rollout.positionId !== positionId);
    let failed = $derived(rollout.running ? '' : rollout.error || (rollout.outcome?.type === 'error' || rollout.outcome?.type === 'batch-error' ? rollout.outcome.message || ' ' : ''));

    // A failure is read, then goes: the store forgets it, so it does not come back on remount.
    $effect(() => {
        if (!failed) return;
        const timer = setTimeout(() => rolloutStore.update((s) => ({ ...s, error: '', outcome: s.outcome?.type?.endsWith('error') ? null : s.outcome })), errorMs);
        return () => clearTimeout(timer);
    });
</script>

{#if rollout.running}
    <div class="rollout-strip" role="status" data-testid="rollout-progress">
        <progress max="100" value={percent} aria-label={$t('rollout.title')}></progress>
        <span class="count">
            {#if rollout.kind === 'batch'}{$t('rollout.batchProgress', { done: rollout.done, total: rollout.total })} ·{/if}
            {#if elsewhere}{$t('rollout.elsewhere')} ·{/if}
            {$t('rollout.gamesCount', { games: rollout.games, max: rollout.maxGames })}
        </span>
        <button type="button" onclick={cancelRollout} title={$t('rollout.cancelHint')} data-testid="rollout-cancel">{$t('rollout.cancel')}</button>
    </div>
{:else if failed}
    <p class="rollout-strip error" role="alert" data-testid="rollout-error">{$t('rollout.error', { message: failed })}</p>
{/if}

<style>
    .rollout-strip {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        margin: var(--space-1) 0 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }
    progress {
        flex: 1;
        min-width: 6em;
        height: 4px;
    }
    .count {
        font-variant-numeric: tabular-nums;
        white-space: nowrap;
    }
    button {
        padding: 0 var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }
    .error {
        color: var(--color-danger);
    }
</style>
