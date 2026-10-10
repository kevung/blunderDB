<script>
    // An empty view that says what to do next (ADR-0085, G5): the panel's own primary action
    // when it has one, else import something, or go back to the welcome screen when no database
    // is open. The only centred block of a panel.
    import { t } from '../../i18n/index.js';
    import { databasePathStore } from '../../stores/databaseStore';
    import { homeDismissedStore } from '../../stores/uiStore';
    import { importPosition } from '../../services/importService';
    import { openDatabase } from '../../services/databaseService';

    // `clear` replaces the import gesture when the list is empty because of a filter, not
    // because nothing was ever imported. `action` replaces it when the panel's content is made
    // here rather than imported (a new collection, a new deck).
    /** @typedef {{ label: string, onClick: () => void }} EmptyAction */
    /** @type {{ text: string, actions?: boolean, clear?: EmptyAction | null, action?: EmptyAction | null }} */
    let { text, actions = true, clear = null, action = null } = $props();
</script>

<div class="empty-state" data-testid="empty-state">
    <span class="empty-text">{text}</span>
    {#if actions}
        <span class="empty-actions">
            {#if clear}
                <button type="button" class="empty-btn primary" data-testid="empty-clear" onclick={clear.onClick}>{clear.label}</button>
            {:else if action}
                <button type="button" class="empty-btn primary" data-testid="empty-action" onclick={action.onClick}>{action.label}</button>
            {:else if $databasePathStore}
                <button type="button" class="empty-btn primary" onclick={() => importPosition()}>{$t('emptyState.import')}</button>
            {:else}
                <button type="button" class="empty-btn primary" onclick={() => openDatabase()}>{$t('emptyState.openDatabase')}</button>
            {/if}
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
