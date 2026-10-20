<!--
  The in-place swap (ADR-0101): while the swap key shows it, the other library's board is drawn
  over this one, at the board's own place and size, under that library's colour and name. Nothing
  of the board on screen changes: no store is written, the group and the view stay; moving to
  another position, losing the window or releasing a held key puts this library back.
-->
<script>
    import { get } from 'svelte/store';
    import { elsewherePeekStore } from '../stores/elsewhereStore.js';
    import { positionStore } from '../stores/positionStore.js';
    import { boardOrientationStore } from '../stores/boardOrientationStore.js';
    import { groupName, groupColorIndex } from '../stores/libraryGroupStore.js';
    import { endSwap, swapKeyUp, analysisSummary, boardKey } from '../services/elsewhereService.js';
    import { renderPositionSVG } from '../services/diagramService.js';
    import { isLetter } from '../utils/keys.js';
    import { t } from '../i18n';

    let peek = $derived($elsewherePeekStore);

    // The board's own box, read when the swap shows: the drawing covers exactly it.
    let box = $state({ left: 0, top: 0, width: 0, height: 0 });
    $effect(() => {
        if (!peek) return;
        const board = document.getElementById('backgammon-board');
        const host = board?.closest('.canvas-container');
        if (!board || !host) return;
        const b = board.getBoundingClientRect();
        const h = host.getBoundingClientRect();
        box = { left: b.left - h.left, top: b.top - h.top, width: b.width, height: b.height };
    });

    let svg = $derived(peek && box.width > 0 ? renderPositionSVG(peek.position, { width: Math.round(box.width), height: Math.round(box.height), orientation: $boardOrientationStore }) : '');
    let summary = $derived(analysisSummary(peek?.report?.analysis));

    // Another board on screen ends the swap: it showed the other library's view of the last one.
    // The same board set again (a panel reloading it) does not.
    let shownFor = '';
    $effect(() => {
        const key = boardKey($positionStore);
        if (shownFor && key !== shownFor) endSwap();
        shownFor = peek ? key : '';
    });

    /** @param {KeyboardEvent} event */
    function onKeyUp(event) {
        if (isLetter(event, 'x') && get(elsewherePeekStore)) swapKeyUp();
    }
</script>

<svelte:window onkeyup={onKeyUp} onblur={endSwap} />

{#if peek && svg}
    <div
        class="swap"
        style="left: {box.left}px; top: {box.top}px; width: {box.width}px; height: {box.height}px; --group-color: var(--group-color-{groupColorIndex(peek.group)})"
        data-testid="elsewhere-swap"
        role="img"
        aria-label={$t('elsewhere.swapLabel', { name: groupName(peek.path) })}
    >
        <!-- eslint-disable-next-line svelte/no-at-html-tags -- renderPositionSVG draws from the position's numbers only -->
        {@html svg}
        <div class="tag">
            <span class="name">{groupName(peek.path) || $t('libraryGroups.unnamed')}</span>
            {#if peek.pinned}<span>· {$t('elsewhere.pinned')}</span>{/if}
            {#if summary.kind !== 'none'}<span>· {summary.best}</span>{/if}
            {#if peek.report?.comments?.length}<span>· {$t('elsewhere.commentCount', { count: peek.report.comments.length })}</span>{/if}
            {#if peek.report?.otherMet}<span>· {$t('elsewhere.otherMet', { name: peek.report.metName || $t('elsewhere.builtinMet') })}</span>{/if}
        </div>
    </div>
{/if}

<style>
    .swap {
        position: absolute;
        z-index: 5;
        box-sizing: border-box;
        outline: 3px solid var(--group-color);
        outline-offset: -3px;
        pointer-events: none;
        line-height: 0;
    }
    .tag {
        position: absolute;
        top: 6px;
        left: 50%;
        transform: translateX(-50%);
        max-width: 90%;
        display: flex;
        gap: 4px;
        padding: 2px 8px;
        border-radius: 999px;
        background: var(--color-surface);
        border: 2px solid var(--group-color);
        color: var(--color-text);
        font-size: var(--font-size-small);
        line-height: 1.4;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
    .name {
        font-weight: 600;
    }
</style>
