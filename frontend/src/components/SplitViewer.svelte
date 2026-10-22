<!--
  The frozen pane of the split screen (ADR-0101 §8): one library group, on its fixed side, shown
  where that group was left — its board, the analysis and the comment — dimmed. It reads its group
  by number (ViewerSteps, ViewerShow) and writes no store of the active group. A click activates
  its group (activatePane); nothing else moves.
-->
<script>
    import { ViewerSteps, ViewerShow } from '../../wailsjs/go/gui/App.js';
    import { splitShakeStore, splitFrozenStore } from '../stores/splitStore.js';
    import { groupColorIndex } from '../stores/libraryGroupStore.js';
    import { boardOrientationStore } from '../stores/boardOrientationStore.js';
    import { activatePane, viewerStartOf } from '../services/splitService.js';
    import { analysisSummary } from '../services/elsewhereService.js';
    import { renderPositionSVG } from '../services/diagramService.js';
    import { libraryDrag } from '../utils/libraryDrag.js';
    import SplitPaneHeader from './SplitPaneHeader.svelte';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';

    /** @type {{ group: number, side: 'left' | 'right', name: string, same?: boolean }} */
    let { group, side, name, same = false } = $props();

    /** @type {import('../services/splitService.js').ViewerSource} */
    let source = $state({ kind: 'library', id: 0 });
    /** @type {any[]} */
    let steps = $state([]);
    let index = $state(0);
    /** @type {any} */
    let item = $state(null);
    let error = $state('');

    // A read answered after the pane moved on shows nothing.
    let generation = 0;

    /** @param {import('../services/splitService.js').ViewerSource} src @param {{ positionId?: number | null, index?: number }} [at] */
    async function walk(src, at = {}) {
        const mine = ++generation;
        const g = group;
        source = src;
        error = '';
        try {
            const read = (await ViewerSteps(g, src)) ?? [];
            if (mine !== generation) return;
            steps = read;
            let i = at.index ?? 0;
            if (at.positionId != null) {
                const found = read.findIndex((s) => s.positionId === at.positionId);
                if (found >= 0) i = found;
            }
            index = Math.max(0, Math.min(i, read.length - 1));
        } catch (e) {
            if (mine !== generation) return;
            logger.error('Error reading the frozen pane:', e);
            steps = [];
            error = String(e);
        }
    }

    // A new group in the pane starts where that group was left.
    let shownGroup = -1;
    $effect(() => {
        const g = group;
        if (g === shownGroup) return;
        shownGroup = g;
        item = null;
        steps = [];
        const start = viewerStartOf(g);
        walk(start.source, { positionId: start.positionId, index: start.index });
    });

    // Two views of one library traded places: this pane freezes on the position just left.
    let refrozen = 0;
    $effect(() => {
        const ask = $splitFrozenStore.refreeze;
        if (!ask || ask.n === refrozen) return;
        refrozen = ask.n;
        walk({ kind: 'library', id: 0 }, { positionId: ask.positionId });
    });

    $effect(() => {
        const step = steps[index];
        const g = group;
        const mine = generation;
        splitFrozenStore.update((f) => ({ ...f, positionId: step?.positionId ?? null }));
        if (!step) {
            item = null;
            return;
        }
        ViewerShow(g, step.positionId)
            .then((read) => {
                if (mine === generation && steps[index] === step && g === group) item = read;
            })
            .catch((e) => logger.error('Error reading a frozen position:', e));
    });

    let shaking = $state(false);
    let shook = $splitShakeStore.n;
    $effect(() => {
        const s = $splitShakeStore;
        if (s.n === shook) return;
        shook = s.n;
        if (s.side !== side) return;
        shaking = false;
        requestAnimationFrame(() => (shaking = true));
        const timer = setTimeout(() => (shaking = false), 450);
        return () => clearTimeout(timer);
    });

    /** @param {MouseEvent} event */
    function onPaneClick(event) {
        if (/** @type {HTMLElement} */ (event.target).closest('button, a, .count')) return;
        activatePane(side);
    }

    let boardHost = $state(/** @type {HTMLElement | null} */ (null));
    let box = $state({ width: 0, height: 0 });
    $effect(() => {
        if (!boardHost) return;
        const observer = new ResizeObserver(([entry]) => {
            const { width, height } = entry.contentRect;
            // The report diagram's proportions, fitted in the box.
            const w = Math.min(width, (height * 520) / 380);
            box = { width: Math.floor(w), height: Math.floor((w * 380) / 520) };
        });
        observer.observe(boardHost);
        return () => observer.disconnect();
    });

    let svg = $derived(item && box.width > 40 ? renderPositionSVG(item.position, { width: box.width, height: box.height, orientation: $boardOrientationStore }) : '');
    let summary = $derived(analysisSummary(item?.analysis));
    let moves = $derived((item?.analysis?.checkerAnalysis?.moves ?? []).slice(0, 5));
    let cube = $derived(item?.analysis?.doublingCubeAnalysis ?? null);
    let step = $derived(steps[index] ?? null);
    let count = $derived(steps.length ? `${index + 1} / ${steps.length}` : '0 / 0');

    /** The position shown, carried onto the other pane or a group's header. */
    const dragged = () => (step?.positionId ? { group, kind: /** @type {const} */ ('position'), id: step.positionId, label: `#${step.positionId}` } : null);

    /** @param {number | null | undefined} x */
    const eq = (x) => (x == null ? '' : (x >= 0 ? '+' : '') + x.toFixed(3));
</script>

<!-- The keyboard reaches this pane by Ctrl+Alt+←/→. -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<section
    class="split-viewer"
    class:shake={shaking}
    style="--group-color: var(--group-color-{groupColorIndex(group)})"
    data-testid="split-viewer"
    data-side={side}
    data-group={group}
    data-library-drop={group}
    data-library-pane
    aria-label={$t('split.paneLabel', { name: name || $t('libraryGroups.unnamed') })}
    onclick={onPaneClick}
>
    <SplitPaneHeader {name} {count} {same} drag={dragged} />

    {#if source.kind === 'match' && step}
        <div class="played" data-testid="split-played">
            {$t('split.moveAt', { game: step.gameNumber, move: step.moveNumber })}{#if step.played}
                · {step.played}{/if}
        </div>
    {/if}

    <div class="board" bind:this={boardHost} use:libraryDrag={dragged}>
        {#if error}
            <p class="empty">{$t('split.error')}</p>
        {:else if !steps.length}
            <p class="empty">{$t('split.empty')}</p>
        {:else if svg}
            <!-- eslint-disable-next-line svelte/no-at-html-tags -- renderPositionSVG draws from the position's numbers only -->
            {@html svg}
        {/if}
    </div>

    <div class="info" data-testid="split-info">
        {#if item}
            {#if moves.length}
                <table>
                    <tbody>
                        {#each moves as m, i (i)}
                            <tr class:best={i === 0}>
                                <td>{i + 1}.</td>
                                <td class="move">{m.move}</td>
                                <td class="num">{eq(m.equity)}</td>
                                <td class="num">{i === 0 ? '' : eq(m.equityError != null ? -Math.abs(m.equityError) : m.equity - moves[0].equity)}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            {:else if cube?.bestCubeAction}
                <p><strong>{$t('elsewhere.bestCube')}</strong> {cube.bestCubeAction}</p>
                <table>
                    <tbody>
                        <tr><td>{$t('split.noDouble')}</td><td class="num">{eq(cube.cubefulNoDoubleEquity)}</td></tr>
                        <tr><td>{$t('split.doubleTake')}</td><td class="num">{eq(cube.cubefulDoubleTakeEquity)}</td></tr>
                        <tr><td>{$t('split.doublePass')}</td><td class="num">{eq(cube.cubefulDoublePassEquity)}</td></tr>
                    </tbody>
                </table>
            {:else}
                <p class="muted">{$t('elsewhere.notAnalysed')}</p>
            {/if}
            {#if summary.engine}<p class="muted">{summary.engine}</p>{/if}
            {#if item.comment}<p class="comment">{item.comment}</p>{/if}
        {/if}
    </div>
</section>

<style>
    .split-viewer {
        flex: 1 1 0;
        min-width: 0;
        min-height: 0;
        box-sizing: border-box;
        display: flex;
        flex-direction: column;
        background: var(--color-surface);
        color: var(--color-text);
        border: 3px solid transparent;
        overflow: hidden;
        cursor: pointer;
        opacity: 0.6;
        transition: opacity 0.15s;
    }
    .split-viewer:hover {
        opacity: 0.8;
    }
    .split-viewer.shake {
        animation: split-shake 0.4s;
    }
    @keyframes split-shake {
        0%,
        100% {
            transform: translateX(0);
        }
        20%,
        60% {
            transform: translateX(-6px);
        }
        40%,
        80% {
            transform: translateX(6px);
        }
    }
    @media (prefers-reduced-motion: reduce) {
        .split-viewer.shake {
            animation: none;
            outline: 2px dashed var(--group-color);
        }
    }
    .played {
        display: flex;
        align-items: center;
        padding: 2px 8px;
        flex-shrink: 0;
        font-weight: 600;
    }
    .board {
        flex: 1;
        min-height: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        line-height: 0;
    }
    .empty,
    .muted {
        opacity: 0.7;
        line-height: 1.4;
    }
    .info {
        flex-shrink: 0;
        max-height: 30%;
        overflow: auto;
        padding: 4px 12px 8px;
        border-top: 1px solid var(--color-border);
    }
    .info p {
        margin: 4px 0;
    }
    table {
        border-collapse: collapse;
        width: 100%;
    }
    td {
        padding: 1px 6px;
    }
    tr.best {
        font-weight: 600;
    }
    .move {
        width: 100%;
    }
    .num {
        text-align: right;
        font-variant-numeric: tabular-nums;
        white-space: nowrap;
    }
    .comment {
        white-space: pre-wrap;
    }
</style>
