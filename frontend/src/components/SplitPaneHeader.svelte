<!--
  The header of a split-screen pane (ADR-0101 §8): the library's colour chip and name, "same
  library" when both panes hold it, and the pane's own "n / N" counter, which carries the position
  shown onto the other pane (libraryDrag).
-->
<script>
    import { libraryDrag } from '../utils/libraryDrag.js';
    import { toggleSplit } from '../services/splitService.js';
    import { t } from '../i18n';

    /**
     * @type {{
     *   name: string,
     *   count: string,
     *   same?: boolean,
     *   live?: boolean,
     *   drag: () => (import('../stores/transferBasketStore.js').TransferItem | null)
     * }}
     */
    let { name, count, same = false, live = false, drag } = $props();
</script>

<header class="split-pane-header">
    <span class="chip"></span>
    <span class="name" data-testid="split-name">{name || $t('libraryGroups.unnamed')}</span>
    {#if same}<span class="badge" data-testid="split-same">{$t('split.sameLibrary')}</span>{/if}
    <span class="hint">{live ? $t('split.liveHint') : $t('split.frozenHint')}</span>
    <span class="count" data-testid="split-count" title={$t('transfer.dragPosition')} use:libraryDrag={drag}>{count}</span>
    {#if !live}
        <button class="close" title={$t('split.close')} aria-label={$t('split.close')} onclick={() => toggleSplit()}>×</button>
    {/if}
</header>

<style>
    .split-pane-header {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 2px 8px;
        flex-shrink: 0;
        min-width: 0;
        border-bottom: 1px solid var(--color-border);
        background: var(--color-surface);
        font-size: var(--font-size-small);
    }
    .chip {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        background: var(--group-color);
        flex-shrink: 0;
    }
    .name {
        font-weight: 600;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .badge {
        border: 1px solid var(--group-color);
        border-radius: 999px;
        padding: 0 6px;
        white-space: nowrap;
    }
    .hint {
        opacity: 0.7;
        flex: 1;
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .count {
        min-width: 4em;
        text-align: center;
        font-variant-numeric: tabular-nums;
        cursor: grab;
    }
    .close {
        background: none;
        border: none;
        cursor: pointer;
        color: inherit;
    }
</style>
