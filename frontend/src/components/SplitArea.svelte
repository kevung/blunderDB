<!--
  The board area, cut in two while the split screen is on (ADR-0101 §8). Each side holds one
  library group and keeps it: the active group's pane is live (the board given as children, never
  remounted, outlined in its group's colour); the other is frozen (SplitViewer). Activating a
  group moves the outline, not the libraries. Split off, the area is the board alone.
-->
<script>
    import { untrack } from 'svelte';
    import { splitStore } from '../stores/splitStore.js';
    import { groupIdsStore, activeGroupId, tabGroupsStore, groupColorIndex } from '../stores/libraryGroupStore.js';
    import { positionStore, positionsStore, matchContextStore } from '../stores/positionStore.js';
    import { currentPositionIndexStore } from '../stores/uiStore.js';
    import { paneGroups, liveSide, followActivation, closeSplitIfNarrow, rememberSplit, splitRemembering } from '../services/splitService.js';
    import SplitViewer from './SplitViewer.svelte';
    import SplitPaneHeader from './SplitPaneHeader.svelte';

    /** @type {{ children: import('svelte').Snippet }} */
    let { children } = $props();

    let on = $derived($splitStore.on);
    let groups = $derived(on ? paneGroups($splitStore, $groupIdsStore, $activeGroupId) : null);
    let live = $derived(groups ? liveSide($splitStore, groups, $activeGroupId) : 'left');
    let frozenSide = $derived(/** @type {'left' | 'right'} */ (live === 'left' ? 'right' : 'left'));
    let same = $derived(!!groups && groups.left === groups.right);

    /** @param {number | null | undefined} id */
    const nameOf = (id) => $tabGroupsStore.find((g) => g.id === id)?.name ?? '';
    /** @param {number | null | undefined} id */
    const pathOf = (id) => $tabGroupsStore.find((g) => g.id === id)?.path ?? '';

    // A group neither pane holds came on screen (its tab, a library just opened): it takes the
    // live side; a group a pane holds leaves both sides as they were.
    let lastActive = $activeGroupId;
    $effect(() => {
        const to = $activeGroupId;
        const from = lastActive;
        lastActive = to;
        if (from === to) return;
        untrack(() => {
            const next = followActivation($splitStore, from, to, $groupIdsStore);
            if (next !== $splitStore) splitStore.set(next);
        });
    });

    // The split and the file on each side, kept for the next session.
    $effect(() => {
        const state = $splitStore;
        const g = groups;
        void $tabGroupsStore;
        if (!splitRemembering()) return;
        untrack(() => rememberSplit(g ? { ...state, ...g } : state, pathOf));
    });

    let liveCount = $derived(
        $matchContextStore.isMatchMode && $matchContextStore.movePositions.length > 0
            ? `${$matchContextStore.currentIndex + 1} / ${$matchContextStore.movePositions.length}`
            : `${$positionsStore.length > 0 ? $currentPositionIndexStore + 1 : 0} / ${$positionsStore.length}`
    );
    const liveDrag = () => ($positionStore?.id ? { group: $activeGroupId, kind: /** @type {const} */ ('position'), id: $positionStore.id, label: `#${$positionStore.id}` } : null);
</script>

<svelte:window onresize={() => closeSplitIfNarrow()} />

<div class="split-area" class:on>
    {#if on && groups}
        <SplitViewer group={groups[frozenSide]} side={frozenSide} name={nameOf(groups[frozenSide])} {same} />
    {/if}
    <div
        class="live-pane"
        class:split={on}
        style="--group-color: var(--group-color-{groupColorIndex($activeGroupId)})"
        data-testid={on ? 'split-live' : undefined}
        data-side={on ? live : undefined}
        data-group={on ? $activeGroupId : undefined}
        data-library-drop={on ? $activeGroupId : undefined}
        data-library-pane={on ? '' : undefined}
    >
        {#if on}
            <SplitPaneHeader name={nameOf($activeGroupId)} count={liveCount} {same} live drag={liveDrag} />
        {/if}
        <div class="live-board">
            {@render children()}
        </div>
    </div>
</div>

<style>
    .split-area {
        flex: 1;
        align-self: stretch;
        min-width: 0;
        min-height: 0;
        display: flex;
        flex-direction: row;
    }
    .split-area.on {
        gap: 4px;
    }
    /* Each pane sits on its side whatever the DOM order: the live board is never moved. */
    .live-pane[data-side='left'],
    .split-area :global(.split-viewer[data-side='left']) {
        order: 0;
    }
    .live-pane[data-side='right'],
    .split-area :global(.split-viewer[data-side='right']) {
        order: 2;
    }
    .live-pane {
        order: 1;
        flex: 1 1 0;
        min-width: 0;
        min-height: 0;
        display: flex;
        flex-direction: column;
        box-sizing: border-box;
    }
    .live-pane.split {
        border: 3px solid var(--group-color);
    }
    .live-board {
        flex: 1;
        min-height: 0;
        min-width: 0;
        display: flex;
        justify-content: center;
        align-items: center;
        position: relative;
    }
</style>
