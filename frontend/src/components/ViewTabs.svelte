<script>
    import { viewStore } from '../stores/viewStore';
    import { tabGroupsStore, activateGroup, closeGroup, groupActivityStore } from '../stores/libraryGroupStore.js';
    import { transferBasketStore, lockedGroupsStore } from '../stores/transferBasketStore.js';
    import { libraryDrag } from '../utils/libraryDrag.js';
    import { registerKeys } from '../services/keyDispatch.js';
    import { t } from '../i18n';
    import { splitStore } from '../stores/splitStore.js';
    import { toggleSplit } from '../services/splitService.js';

    const { views, activeViewId, switchTo, addView, closeView, renameView } = viewStore;

    let editingId = $state(null);
    let editingName = $state('');
    let editInput = $state();

    // Several libraries open: the bar shows every group, each behind a header in its colour.
    const grouped = $derived($tabGroupsStore.length > 1);
    // The basket's menu: pour it into a group, or empty it.
    let basketMenuOpen = $state(false);
    /** @param {number} id */
    async function pourInto(id) {
        basketMenuOpen = false;
        await (await transfer()).pourBasket(id);
    }
    // Loaded on first use, as databaseService: the bar stays light to mount.
    const transfer = () => import('../services/transferService.js');
    const tabCount = $derived($tabGroupsStore.reduce((n, g) => n + g.views.length, 0));

    // Too many tabs for the window: the row scrolls rather than squeezing each tab to a sliver.
    // The wheel scrolls it sideways, and the tab shown is kept in sight.
    /** @type {HTMLElement | undefined} */
    let tabList = $state();
    /** @param {WheelEvent} event */
    function scrollSideways(event) {
        if (!tabList || event.deltaX !== 0 || tabList.scrollWidth <= tabList.clientWidth) return;
        tabList.scrollLeft += event.deltaY;
        event.preventDefault();
    }
    $effect(() => {
        void $activeViewId;
        void $tabGroupsStore;
        tabList?.querySelector('.view-tab.active')?.scrollIntoView?.({ block: 'nearest', inline: 'nearest' });
    });

    /**
     * A tab of the group on screen switches view; a tab of another group brings its library and
     * that view in one gesture.
     * @param {number} groupId
     * @param {number} viewId
     * @param {boolean} activeGroup
     */
    function handleTabClick(groupId, viewId, activeGroup) {
        if (activeGroup) switchTo(viewId);
        else activateGroup(groupId, viewId);
    }

    // On demand: the database service pulls in the whole open flow, which the bar does not need
    // until the user asks for a second library.
    async function openInNewGroup() {
        (await import('../services/databaseService.js')).openDatabaseInNewGroup();
    }

    /** @param {{ id: number, name: string }} view */
    function handleDoubleClick(view) {
        editingId = view.id;
        editingName = view.name;
        setTimeout(() => editInput?.focus(), 0);
    }

    function finishRename() {
        if (editingId !== null && editingName.trim()) {
            renameView(editingId, editingName.trim());
        }
        editingId = null;
        editingName = '';
    }

    /** @param {KeyboardEvent} event */
    function handleRenameKeydown(event) {
        if (event.key === 'Enter') {
            event.stopPropagation();
            finishRename();
        } else if (event.key === 'Escape') {
            event.stopPropagation();
            editingId = null;
            editingName = '';
        }
    }

    // Enter on a focused tab switches to it. Only Enter: Space stays the global
    // command-line key and the arrows the board's, so a click on a tab (which
    // now focuses it) changes nothing that follows.
    /**
     * @param {KeyboardEvent} event
     * @param {number} groupId
     * @param {number} viewId
     * @param {boolean} activeGroup
     */
    function handleTabKeydown(event, groupId, viewId, activeGroup) {
        if (event.target !== event.currentTarget) return; // rename field and close button own their keys
        if (event.key !== 'Enter') return;
        event.preventDefault();
        handleTabClick(groupId, viewId, activeGroup);
    }

    /**
     * @param {MouseEvent} event
     * @param {number} viewId
     */
    function handleClose(event, viewId) {
        event.stopPropagation();
        closeView(viewId);
    }

    /**
     * @param {MouseEvent} event
     * @param {number} groupId
     */
    function handleCloseGroup(event, groupId) {
        event.stopPropagation();
        closeGroup(groupId);
    }

    /** @param {KeyboardEvent} event */
    function handleKeydown(event) {
        // Match the digit by physical key (event.code) so the labeled number keys
        // work without Shift on layouts where the top row is shifted (e.g. AZERTY,
        // where unshifted "1" produces "&").
        const m = event.code.match(/^(?:Digit|Numpad)([1-9])$/);
        if (event.ctrlKey && m) {
            const idx = parseInt(m[1], 10) - 1;
            const vs = $views;
            if (idx < vs.length) {
                event.preventDefault();
                switchTo(vs[idx].id);
            }
        }
    }

    $effect(() => registerKeys('viewTabs', handleKeydown));
</script>

<!-- A single view of a single library collapses the bar to the bare "+"; it expands with a second
     view or a second library. -->
<div class="view-tabs" class:collapsed={tabCount <= 1}>
    {#if tabCount > 1}
        <div class="view-tab-list" role="tablist" bind:this={tabList} onwheel={scrollSideways}>
            {#each $tabGroupsStore as group (group.id)}
                {#if grouped}
                    <div
                        class="group-header group-color-{group.color}"
                        class:active={group.active}
                        role="button"
                        tabindex="0"
                        title={group.path}
                        data-testid="library-group"
                        data-group-id={group.id}
                        data-library-drop={group.id}
                        class:locked={$lockedGroupsStore.has(group.id)}
                        onclick={() => activateGroup(group.id)}
                        onkeydown={(e) => e.target === e.currentTarget && e.key === 'Enter' && activateGroup(group.id)}
                    >
                        <span class="group-name">{group.name || $t('libraryGroups.unnamed')}</span>
                        <!-- A job of this library ran or finished while it was off screen. -->
                        {#if !group.active && $groupActivityStore.get(group.id)}
                            {@const done = $groupActivityStore.get(group.id)?.state === 'done'}
                            <span
                                class="group-badge"
                                class:done
                                data-testid="library-group-badge"
                                data-state={done ? 'done' : 'running'}
                                title={$t(done ? 'libraryGroups.jobDone' : 'libraryGroups.jobRunning')}
                                aria-label={$t(done ? 'libraryGroups.jobDone' : 'libraryGroups.jobRunning')}
                            ></span>
                        {/if}
                        <button
                            class="lock-btn"
                            class:on={$lockedGroupsStore.has(group.id)}
                            data-testid="library-group-lock"
                            aria-pressed={$lockedGroupsStore.has(group.id)}
                            onclick={(e) => {
                                e.stopPropagation();
                                transfer().then((m) => m.toggleGroupLock(group.id));
                            }}
                            title={$t($lockedGroupsStore.has(group.id) ? 'transfer.unlockGroup' : 'transfer.lockGroup')}
                            aria-label={$t($lockedGroupsStore.has(group.id) ? 'transfer.unlockGroup' : 'transfer.lockGroup')}
                        >
                            <svg viewBox="0 0 16 16" width="11" height="11" aria-hidden="true"
                                ><rect x="3" y="7" width="10" height="7" rx="1" fill="currentColor" /><path
                                    d={$lockedGroupsStore.has(group.id) ? 'M5 7V5a3 3 0 0 1 6 0v2' : 'M5 7V5a3 3 0 0 1 6 0'}
                                    fill="none"
                                    stroke="currentColor"
                                    stroke-width="1.6"
                                /></svg
                            >
                        </button>
                        <button class="close-btn" onclick={(e) => handleCloseGroup(e, group.id)} title={$t('libraryGroups.close')}>&times;</button>
                    </div>
                {/if}
                {#each group.views as view (view.id)}
                    {@const selected = group.active && $activeViewId === view.id}
                    <div
                        class="view-tab group-color-{group.color}"
                        class:grouped
                        class:active={selected}
                        class:resting={!group.active && group.activeViewId === view.id}
                        role="tab"
                        aria-selected={selected}
                        tabindex="0"
                        data-group-id={group.id}
                        onclick={() => handleTabClick(group.id, view.id, group.active)}
                        ondblclick={() => group.active && handleDoubleClick(view)}
                        onkeydown={(e) => handleTabKeydown(e, group.id, view.id, group.active)}
                    >
                        {#if group.active && editingId === view.id}
                            <input bind:this={editInput} bind:value={editingName} class="rename-input" onblur={finishRename} onkeydown={handleRenameKeydown} />
                        {:else}
                            <span class="tab-name">{view.name}</span>
                        {/if}
                        {#if group.active && group.views.length > 1}
                            <button class="close-btn" onclick={(e) => handleClose(e, view.id)} title={$t('viewTabs.closeView')}>&times;</button>
                        {/if}
                    </div>
                {/each}
            {/each}
        </div>
    {/if}
    <button class="add-btn" onclick={() => addView()} title={$t('viewTabs.newView')}
        >+{#if tabCount <= 1}<span class="add-label">{$t('viewTabs.newView')}</span>{/if}</button
    >
    <button class="group-btn" data-testid="open-library-group" onclick={openInNewGroup} title={$t('libraryGroups.openInNewGroup')} aria-label={$t('libraryGroups.openInNewGroup')}
        >&#x29C9;{#if !grouped}<span class="add-label">{$t('libraryGroups.openLabel')}</span>{/if}</button
    >
    <button
        class="group-btn"
        class:on={$splitStore.on}
        data-testid="toggle-split"
        onclick={() => toggleSplit()}
        title={$t('split.toggle')}
        aria-label={$t('split.toggle')}
        aria-pressed={$splitStore.on}>&#x25EB;</button
    >
    {#if grouped || $transferBasketStore.length > 0}
        <!-- The basket: rows of any group, poured at once into one; dragged onto a header, it pours there. -->
        <div class="basket">
            <span
                class="basket-btn"
                class:filled={$transferBasketStore.length > 0}
                role="button"
                tabindex="0"
                data-testid="transfer-basket"
                data-library-drop="basket"
                title={$t('transfer.basketTitle', { n: $transferBasketStore.length })}
                aria-label={$t('transfer.basketTitle', { n: $transferBasketStore.length })}
                aria-expanded={basketMenuOpen}
                onclick={() => (basketMenuOpen = !basketMenuOpen)}
                onkeydown={(e) => e.key === 'Enter' && (basketMenuOpen = !basketMenuOpen)}
                use:libraryDrag={() => ($transferBasketStore.length > 0 ? { kind: 'basket', label: $t('transfer.basketTitle', { n: $transferBasketStore.length }) } : null)}
                >{$t('transfer.basket')}<span class="basket-count">{$transferBasketStore.length}</span></span
            >
            {#if basketMenuOpen}
                <div class="basket-menu" role="menu" data-testid="transfer-basket-menu">
                    {#each $tabGroupsStore as group (group.id)}
                        <button
                            role="menuitem"
                            data-testid="transfer-basket-pour"
                            data-group-id={group.id}
                            disabled={$transferBasketStore.length === 0 || $lockedGroupsStore.has(group.id)}
                            onclick={() => pourInto(group.id)}>{$t('transfer.pourInto', { name: group.name || $t('libraryGroups.unnamed') })}</button
                        >
                    {/each}
                    <button
                        role="menuitem"
                        data-testid="transfer-basket-empty"
                        disabled={$transferBasketStore.length === 0}
                        onclick={() => {
                            basketMenuOpen = false;
                            transfer().then((m) => m.emptyBasket());
                        }}>{$t('transfer.emptyBasket')}</button
                    >
                </div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .view-tabs {
        display: flex;
        align-items: stretch;
        background: var(--color-surface-alt);
        border-bottom: 1px solid var(--color-border);
        flex-shrink: 0;
        height: 26px;
        overflow-x: auto;
        overflow-y: hidden;
        gap: 0;
        padding-left: 4px;
    }

    .view-tabs.collapsed {
        height: 20px;
        padding-left: 0;
    }

    .add-label {
        margin-left: 4px;
    }

    .view-tabs.collapsed .add-btn {
        font-size: var(--font-size-small);
        padding: 0 6px;
    }

    /* The tabs' own row inside the bar, which also holds the "+" button (not a tab). */
    .view-tab-list {
        display: flex;
        align-items: stretch;
        min-width: 0;
        overflow-x: auto;
        scrollbar-width: none;
    }

    .view-tab-list::-webkit-scrollbar {
        display: none;
    }

    .add-btn,
    .group-btn {
        flex-shrink: 0;
    }

    .view-tab {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 0 8px;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        cursor: pointer;
        border-right: 1px solid var(--color-border);
        white-space: nowrap;
        user-select: none;
        transition: background 0.1s;
        flex-shrink: 0;
    }

    .view-tab:hover {
        background: color-mix(in srgb, var(--color-text) 6%, var(--color-surface-alt));
    }

    .view-tab.active {
        background: var(--color-surface);
        color: var(--color-text);
        font-weight: 600;
        border-bottom: 1px solid var(--color-surface);
        margin-bottom: -1px;
    }

    .tab-name {
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .rename-input {
        width: 60px;
        font-size: var(--font-size-small);
        border: 1px solid var(--color-border);
        padding: 1px 3px;
        outline: none;
        background: var(--color-surface);
    }

    .close-btn {
        border: none;
        background: none;
        font-size: var(--font-size-base);
        color: var(--color-text-muted);
        cursor: pointer;
        padding: 0 2px;
        line-height: 1;
        display: flex;
        align-items: center;
    }

    .close-btn:hover {
        color: var(--color-danger);
    }

    .add-btn {
        border: none;
        background: none;
        font-size: var(--font-size-title);
        color: var(--color-text-muted);
        cursor: pointer;
        padding: 0 8px;
        display: flex;
        align-items: center;
    }

    .add-btn:hover {
        color: var(--color-text);
    }

    .group-btn.on {
        color: var(--color-primary);
    }
    .group-btn {
        border: none;
        background: none;
        font-size: var(--font-size-base);
        color: var(--color-text-muted);
        cursor: pointer;
        padding: 0 6px;
        display: flex;
        align-items: center;
    }

    .group-btn:hover {
        color: var(--color-text);
    }

    /* A group: its header carries the library's name on its colour; its tabs a top edge of it. */
    .group-header {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 0 6px 0 8px;
        margin-left: 6px;
        font-size: var(--font-size-small);
        /* A group waiting: the theme's own text on the bar, its colour as a top band. The text
           never sits on the hue with an opacity, which no theme keeps legible. */
        color: var(--color-text);
        box-shadow: inset 0 3px 0 var(--group-color);
        cursor: pointer;
        white-space: nowrap;
        user-select: none;
        flex-shrink: 0;
        border-radius: var(--radius) var(--radius) 0 0;
    }

    .group-header:first-child {
        margin-left: 0;
    }

    /* The group on screen: white on its hue, every hue dark enough for 4.5:1 (tokens.css). */
    .group-header.active {
        color: var(--group-text);
        background: var(--group-color);
        box-shadow: none;
        font-weight: 600;
    }

    .group-header .close-btn {
        color: inherit;
    }

    .group-badge {
        display: inline-block;
        width: 8px;
        height: 8px;
        margin-left: 4px;
        border-radius: 50%;
        background: var(--color-text-muted);
        flex-shrink: 0;
        animation: group-badge-pulse 1.2s ease-in-out infinite;
    }
    .group-badge.done {
        background: var(--color-primary);
        animation: none;
    }
    @keyframes group-badge-pulse {
        50% {
            opacity: 0.35;
        }
    }
    @media (prefers-reduced-motion: reduce) {
        .group-badge {
            animation: none;
        }
    }
    .lock-btn {
        display: inline-flex;
        align-items: center;
        padding: 0 2px;
        border: none;
        background: none;
        color: inherit;
        opacity: 0.45;
        cursor: pointer;
    }
    .lock-btn.on,
    .lock-btn:hover {
        opacity: 1;
    }
    /* Where a dragged row would land; a locked header says it takes nothing. */
    :global(.group-header.library-drop-over),
    :global(.basket-btn.library-drop-over) {
        outline: 2px dashed var(--color-primary);
        outline-offset: -2px;
    }
    :global(.group-header.locked.library-drop-over) {
        outline-color: var(--color-danger);
        outline-style: solid;
    }
    .basket {
        position: relative;
        flex-shrink: 0;
    }
    .basket-btn {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 2px 8px;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        cursor: pointer;
        user-select: none;
    }
    .basket-btn.filled {
        color: var(--color-text);
        cursor: grab;
    }
    .basket-count {
        min-width: 14px;
        padding: 0 4px;
        border-radius: 7px;
        text-align: center;
        background: var(--color-surface-alt);
        border: 1px solid var(--color-border);
    }
    .basket-menu {
        position: absolute;
        right: 0;
        top: 100%;
        z-index: 20;
        display: flex;
        flex-direction: column;
        min-width: 180px;
        padding: 4px 0;
        background: var(--color-surface);
        border: 1px solid var(--color-border);
        box-shadow: 0 2px 8px rgb(0 0 0 / 20%);
    }
    .basket-menu button {
        padding: 4px 12px;
        text-align: left;
        border: none;
        background: none;
        color: var(--color-text);
        font-size: var(--font-size-small);
        cursor: pointer;
    }
    .basket-menu button:hover:not(:disabled) {
        background: var(--color-surface-alt);
    }
    .basket-menu button:disabled {
        opacity: 0.5;
        cursor: default;
    }
    .group-name {
        max-width: 14em;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .view-tab.grouped {
        box-shadow: inset 0 2px 0 var(--group-color);
    }

    .view-tab.resting {
        color: var(--color-text);
    }

    .group-color-1 {
        --group-color: var(--group-color-1);
    }
    .group-color-2 {
        --group-color: var(--group-color-2);
    }
    .group-color-3 {
        --group-color: var(--group-color-3);
    }
    .group-color-4 {
        --group-color: var(--group-color-4);
    }
    .group-color-5 {
        --group-color: var(--group-color-5);
    }
    .group-color-6 {
        --group-color: var(--group-color-6);
    }
</style>
