<script>
    // An empty list that says what to do next: import something, or go back to the welcome
    // screen when no database is open. One component so every panel offers the same two ways out.
    import { t } from '../../i18n/index.js';
    import { databasePathStore } from '../../stores/databaseStore';
    import { homeDismissedStore } from '../../stores/uiStore';
    import { importPosition } from '../../services/importService';

    /** @type {{ text: string, actions?: boolean }} */
    let { text, actions = true } = $props();
</script>

<div class="empty-state" data-testid="empty-state">
    <span class="empty-text">{text}</span>
    {#if actions}
        <span class="empty-actions">
            <button type="button" class="empty-btn primary" onclick={() => importPosition()}>{$t('emptyState.import')}</button>
            {#if !$databasePathStore}
                <button type="button" class="empty-btn" onclick={() => homeDismissedStore.set(false)}>{$t('emptyState.backHome')}</button>
            {/if}
        </span>
    {/if}
</div>

<style>
    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 10px;
        text-align: center;
        color: var(--color-text-muted);
        padding: 24px;
        font-size: var(--font-size-base);
    }
    .empty-actions {
        display: flex;
        gap: 8px;
        flex-wrap: wrap;
        justify-content: center;
    }
    .empty-btn {
        padding: 4px 12px;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        cursor: pointer;
        font-size: var(--font-size-base);
        background: var(--color-surface);
        color: var(--color-text);
    }
    .empty-btn.primary {
        background: var(--color-primary);
        border-color: var(--color-primary);
        color: white;
    }
</style>
