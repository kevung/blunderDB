<script>
    import { viewStore } from '../stores/viewStore';

    const { views, activeViewId, switchTo, addView, closeView, renameView } = viewStore;

    let editingId = null;
    let editingName = '';
    let editInput;

    function handleTabClick(viewId) {
        switchTo(viewId);
    }

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

    function handleRenameKeydown(event) {
        if (event.key === 'Enter') {
            event.stopPropagation();
            finishRename();
        } else if (event.key === 'Escape') {
            event.stopPropagation();
            editingId = null; editingName = '';
        }
    }

    function handleClose(event, viewId) {
        event.stopPropagation();
        closeView(viewId);
    }

    function handleKeydown(event) {
        if (event.ctrlKey && event.key >= '1' && event.key <= '9') {
            const idx = parseInt(event.key) - 1;
            const vs = $views;
            if (idx < vs.length) {
                event.preventDefault();
                switchTo(vs[idx].id);
            }
        }
    }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="view-tabs">
    {#each $views as view, i (view.id)}
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div
            class="view-tab"
            class:active={$activeViewId === view.id}
            on:click={() => handleTabClick(view.id)}
            on:dblclick={() => handleDoubleClick(view)}
        >
            {#if editingId === view.id}
                <input
                    bind:this={editInput}
                    bind:value={editingName}
                    class="rename-input"
                    on:blur={finishRename}
                    on:keydown={handleRenameKeydown}
                />
            {:else}
                <span class="tab-name">{view.name}</span>
            {/if}
            {#if $views.length > 1}
                <button class="close-btn" on:click={(e) => handleClose(e, view.id)} title="Close view">&times;</button>
            {/if}
        </div>
    {/each}
    <button class="add-btn" on:click={addView} title="New view">+</button>
</div>

<style>
    .view-tabs {
        display: flex;
        align-items: stretch;
        background: #f0f0f0;
        border-bottom: 1px solid #ddd;
        flex-shrink: 0;
        height: 26px;
        overflow-x: auto;
        overflow-y: hidden;
        gap: 0;
        padding-left: 4px;
    }

    .view-tab {
        display: flex;
        align-items: center;
        gap: 4px;
        padding: 0 8px;
        font-size: 11px;
        color: #666;
        cursor: pointer;
        border-right: 1px solid #e0e0e0;
        white-space: nowrap;
        user-select: none;
        transition: background 0.1s;
        min-width: 0;
    }

    .view-tab:hover {
        background: #e8e8e8;
    }

    .view-tab.active {
        background: #fff;
        color: #333;
        font-weight: 600;
        border-bottom: 1px solid #fff;
        margin-bottom: -1px;
    }

    .tab-name {
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .rename-input {
        width: 60px;
        font-size: 11px;
        border: 1px solid #aaa;
        padding: 1px 3px;
        outline: none;
        background: #fff;
    }

    .close-btn {
        border: none;
        background: none;
        font-size: 14px;
        color: #999;
        cursor: pointer;
        padding: 0 2px;
        line-height: 1;
        display: flex;
        align-items: center;
    }

    .close-btn:hover {
        color: #c00;
    }

    .add-btn {
        border: none;
        background: none;
        font-size: 16px;
        color: #888;
        cursor: pointer;
        padding: 0 8px;
        display: flex;
        align-items: center;
    }

    .add-btn:hover {
        color: #333;
    }
</style>
