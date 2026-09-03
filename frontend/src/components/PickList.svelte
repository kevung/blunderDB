<!--
  A titled list of tick boxes with All/None, for picking a subset of matches, tournaments
  or collections (or anything else with an `id`). Extracted from ExportDatabaseModal, which
  used to declare this same markup as a local snippet — MatchTournamentPickerModal grew a
  second, slightly different copy of it independently (fiche D.9, #209). One component now,
  shared by both.

  `describe(item)` names an item and gives its trailing detail text; return
  `{ name, count, partial }` — `count` is rendered as-is (the caller writes its own
  parentheses where it wants them), `partial` marks a count short of the whole (used for a
  collection exported without every member position).
-->
<script>
    import { t } from '../i18n';

    let {
        header,
        items,
        isChecked,
        isDisabled = () => false,
        toggle,
        selectAll,
        selectNone,
        describe,
        // Optional inline text filter in the header (MatchTournamentPickerModal). Left
        // undefined, no input renders and the caller filters `items` itself either way.
        filterValue = $bindable(undefined),
        filterPlaceholder = ''
    } = $props();
</script>

<div class="pick-list-section">
    <div class="pick-list-header">
        <span>{header}</span>
        {#if filterValue !== undefined}
            <input type="text" bind:value={filterValue} placeholder={filterPlaceholder} class="pick-list-filter" />
        {/if}
        <div class="pick-list-buttons">
            <button type="button" class="small-btn" onclick={selectAll}>{$t('export.all')}</button>
            <button type="button" class="small-btn" onclick={selectNone}>{$t('export.none')}</button>
        </div>
    </div>
    <div class="pick-list-items">
        {#each items as item (item.id)}
            {@const d = describe(item)}
            <label class="pick-list-checkbox" class:disabled={isDisabled(item.id)}>
                <input type="checkbox" checked={isChecked(item.id)} disabled={isDisabled(item.id)} onchange={() => toggle(item.id)} />
                <span class="pick-list-name">{d.name}</span>
                <span class="pick-list-count" class:partial={d.partial}>{d.count}</span>
            </label>
        {/each}
    </div>
</div>

<style>
    .pick-list-section {
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        overflow: hidden;
    }

    .pick-list-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 8px 12px;
        background-color: var(--color-surface-alt);
        border-bottom: 1px solid var(--color-border);
        font-size: var(--font-size-base);
        font-weight: 500;
    }

    .pick-list-buttons {
        display: flex;
        gap: 4px;
    }

    .small-btn {
        padding: 2px 8px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        font-size: var(--font-size-small);
        font-weight: 500;
        cursor: pointer;
        background-color: var(--color-surface);
        color: var(--color-text);
        transition: all 0.2s ease;
    }

    .small-btn:hover {
        background-color: var(--color-surface-alt);
        border-color: #999;
    }

    .pick-list-items {
        max-height: 120px;
        overflow-y: auto;
        padding: 4px;
    }

    .pick-list-checkbox {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 8px;
        cursor: pointer;
        font-size: var(--font-size-base);
    }

    .pick-list-checkbox:hover {
        background-color: var(--color-surface-alt);
    }

    .pick-list-checkbox.disabled {
        opacity: 0.6;
    }

    .pick-list-checkbox input[type='checkbox'] {
        width: 16px;
        height: 16px;
        cursor: pointer;
        accent-color: var(--color-text);
    }

    .pick-list-checkbox input[type='checkbox']:disabled {
        cursor: not-allowed;
    }

    .pick-list-filter {
        flex: 1;
        padding: 4px 8px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        font-size: var(--font-size-base);
        font-family: inherit;
        min-width: 0;
    }

    .pick-list-name {
        flex: 1;
    }

    .pick-list-count {
        color: var(--color-text-muted);
        font-size: var(--font-size-base);
    }

    .pick-list-count.partial {
        color: var(--color-danger);
        font-weight: 600;
    }
</style>
