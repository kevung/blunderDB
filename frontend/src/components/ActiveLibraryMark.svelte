<!--
  Which library the panels and the status bar speak for, while the split screen shows two
  (ADR-0101 §8): the active group's colour chip and name. Split off, there is one board and
  nothing to tell apart.
-->
<script>
    import { splitStore } from '../stores/splitStore.js';
    import { activeGroupId, tabGroupsStore, groupColorIndex } from '../stores/libraryGroupStore.js';
    import { t } from '../i18n';

    /** @type {{ testid: string }} */
    let { testid } = $props();

    let name = $derived($tabGroupsStore.find((g) => g.id === $activeGroupId)?.name || $t('libraryGroups.unnamed'));
</script>

{#if $splitStore.on}
    <span class="active-library-mark" style="--group-color: var(--group-color-{groupColorIndex($activeGroupId)})" data-testid={testid} title={$t('split.activeLibrary', { name })}
        ><span class="chip"></span><span class="name">{name}</span></span
    >
{/if}

<style>
    .active-library-mark {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        flex-shrink: 0;
        max-width: 14em;
        padding: 0 6px;
        font-size: var(--font-size-small);
        font-weight: 600;
        white-space: nowrap;
    }
    .chip {
        width: 9px;
        height: 9px;
        border-radius: 50%;
        background: var(--group-color);
        flex-shrink: 0;
    }
    .name {
        overflow: hidden;
        text-overflow: ellipsis;
    }
</style>
