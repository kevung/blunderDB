<script>
    // The header strip every panel view opens with (ADR-0085, G1-G3):
    // [←] title · count · filters ⟶ secondary actions [primary action].
    // The title never stretches, so the right end is the same place in every panel and the eye
    // finds the primary action there without reading.
    import CountLink from './CountLink.svelte';

    /**
     * @type {{
     *   title?: string,
     *   titleHint?: string,
     *   count?: string | null,
     *   onCount?: (() => void) | null,
     *   countHint?: string,
     *   onBack?: (() => void) | null,
     *   backHint?: string,
     *   children?: import('svelte').Snippet,
     *   actions?: import('svelte').Snippet
     * }}
     */
    let { title = '', titleHint = undefined, count = null, onCount = null, countHint = undefined, onBack = null, backHint = undefined, children = undefined, actions = undefined } = $props();
</script>

<div class="panel-header" data-testid="panel-header">
    {#if onBack}
        <button type="button" class="back-btn" data-testid="panel-back" onclick={onBack} title={backHint} aria-label={backHint}>←</button>
    {/if}
    {#if title}
        <span class="panel-title" title={titleHint ?? title}>{title}</span>
    {/if}
    {#if count != null && count !== ''}
        {#if onCount}
            <CountLink label={count} onclick={onCount} title={countHint} />
        {:else}
            <span class="panel-count">{count}</span>
        {/if}
    {/if}
    {#if children}
        {@render children()}
    {/if}
    <span class="spacer"></span>
    {#if actions}
        <span class="panel-actions">{@render actions()}</span>
    {/if}
</div>

<style>
    .panel-header {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        min-height: 24px;
        padding: var(--space-1) var(--space-2);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        background: var(--color-surface-alt);
        border-bottom: 1px solid var(--color-border);
        flex-shrink: 0;
        text-align: start;
    }
    .back-btn {
        background: none;
        border: none;
        cursor: pointer;
        font-size: var(--font-size-title);
        color: var(--color-text-muted);
        padding: 0 var(--space-1);
        line-height: 1;
        /* A pointer target is at least 24 px square (WCAG 2.5.8). */
        min-width: 24px;
        min-height: 24px;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }
    .back-btn:hover {
        color: var(--color-text);
    }
    .panel-title {
        font-size: var(--font-size-base);
        font-weight: 600;
        color: var(--color-text);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        min-width: 0;
    }
    .panel-count {
        flex-shrink: 0;
        font-variant-numeric: tabular-nums;
    }
    .spacer {
        flex: 1;
    }
    /* The icon buttons a panel puts in its strip read as the table's (PanelTable). */
    .panel-header :global(.icon-btn) {
        background: none;
        border: none;
        cursor: pointer;
        font-size: var(--font-size-base);
        color: var(--color-text-muted);
        padding: 2px 4px;
        line-height: 1;
    }
    .panel-header :global(.icon-btn:hover:not(:disabled)) {
        color: var(--color-text);
    }
    .panel-actions {
        display: inline-flex;
        align-items: center;
        gap: var(--space-1);
        flex-shrink: 0;
    }
</style>
