<!--
  The split screen's second pane (ADR-0101): a read-only viewer of one library group, the same
  width as the live pane. It reads its group by number (ViewerSteps, ViewerShow, ViewerCatalogOf)
  and writes no store of the live group: its walk, its board and its analysis are its own state.
  Clicking the pane outside its controls activates its group; the panes then trade roles and each
  library keeps its side.
-->
<script>
    import { ViewerSteps, ViewerShow, ViewerCatalogOf } from '../../wailsjs/go/gui/App.js';
    import { splitStore } from '../stores/splitStore.js';
    import { groupIdsStore, activeGroupId, parkedGroupsStore, groupName, groupColorIndex } from '../stores/libraryGroupStore.js';
    import { databasePathStore } from '../stores/databaseStore.js';
    import { boardOrientationStore } from '../stores/boardOrientationStore.js';
    import { viewerGroup, followActivation, activateViewer, viewerStartOf, closeSplitIfNarrow, toggleSplit } from '../services/splitService.js';
    import { analysisSummary } from '../services/elsewhereService.js';
    import { renderPositionSVG } from '../services/diagramService.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';

    let group = $derived(viewerGroup($splitStore.group, $groupIdsStore, $activeGroupId));
    let path = $derived(group === $activeGroupId ? $databasePathStore : ($parkedGroupsStore.get(group)?.path ?? ''));
    let sameLibrary = $derived(group === $activeGroupId);

    // The viewer's group came on screen: it takes the group left, on the other side.
    let lastActive = $activeGroupId;
    $effect(() => {
        const to = $activeGroupId;
        const from = lastActive;
        lastActive = to;
        if (from === to) return;
        const next = followActivation($splitStore, from, to, $groupIdsStore);
        if (next !== $splitStore) splitStore.set(next);
    });

    /** @type {import('../services/splitService.js').ViewerSource} */
    let source = $state({ kind: 'library', id: 0 });
    /** @type {any[]} */
    let steps = $state([]);
    let index = $state(0);
    /** @type {any} */
    let item = $state(null);
    let error = $state('');
    /** @type {{ matches: any[], collections: any[] }} */
    let catalog = $state({ matches: [], collections: [] });

    // A read answered after the viewer moved on shows nothing.
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
            logger.error('Error reading the viewer group:', e);
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
        const start = viewerStartOf(g);
        walk(start.source, { positionId: start.positionId, index: start.index });
        ViewerCatalogOf(g)
            .then((c) => {
                if (shownGroup === g) catalog = c ?? { matches: [], collections: [] };
            })
            .catch((e) => logger.error('Error reading the viewer catalog:', e));
    });

    $effect(() => {
        const step = steps[index];
        const g = group;
        const mine = generation;
        if (!step) {
            item = null;
            return;
        }
        ViewerShow(g, step.positionId)
            .then((read) => {
                if (mine === generation && steps[index] === step) item = read;
            })
            .catch((e) => logger.error('Error reading a viewer position:', e));
    });

    let sourceValue = $derived(`${source.kind}:${source.id}`);

    /** @param {Event} event */
    function chooseSource(event) {
        const [chosen, id] = /** @type {HTMLSelectElement} */ (event.currentTarget).value.split(':');
        const kind = /** @type {'library' | 'match' | 'collection'} */ (chosen);
        walk({ kind, id: Number(id) });
    }

    /** @param {number} to */
    function go(to) {
        if (steps.length) index = Math.max(0, Math.min(to, steps.length - 1));
    }

    /** @param {MouseEvent} event */
    function onPaneClick(event) {
        if (/** @type {HTMLElement} */ (event.target).closest('button, select, a, .controls')) return;
        activateViewer();
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

    /** @param {number | null | undefined} x */
    const eq = (x) => (x == null ? '' : (x >= 0 ? '+' : '') + x.toFixed(3));
</script>

<svelte:window onresize={() => closeSplitIfNarrow()} />

<!-- The keyboard reaches the viewer's group by Ctrl+Shift+PageUp/PageDown. -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<section
    class="split-viewer"
    class:left={$splitStore.side === 'left'}
    style="--group-color: var(--group-color-{groupColorIndex(group)})"
    data-testid="split-viewer"
    aria-label={$t('split.viewerLabel', { name: groupName(path) || $t('libraryGroups.unnamed') })}
    onclick={onPaneClick}
>
    <header>
        <span class="chip"></span>
        <span class="name">{groupName(path) || $t('libraryGroups.unnamed')}</span>
        <span class="badge">{$t('split.readOnly')}</span>
        <span class="hint">{sameLibrary ? $t('split.sameLibrary') : $t('split.activateHint')}</span>
        <button class="close controls" title={$t('split.close')} aria-label={$t('split.close')} onclick={() => toggleSplit()}>×</button>
    </header>

    <div class="controls bar">
        <select value={sourceValue} onchange={chooseSource} aria-label={$t('split.source')} data-testid="split-source">
            <option value="library:0">{$t('split.library')}</option>
            {#if catalog.matches.length}
                <optgroup label={$t('split.matches')}>
                    {#each catalog.matches as m (m.id)}<option value="match:{m.id}">{m.label}</option>{/each}
                </optgroup>
            {/if}
            {#if catalog.collections.length}
                <optgroup label={$t('split.collections')}>
                    {#each catalog.collections as c (c.id)}<option value="collection:{c.id}">{c.label} ({c.count})</option>{/each}
                </optgroup>
            {/if}
        </select>
        <button onclick={() => go(0)} disabled={index === 0} aria-label={$t('split.first')} title={$t('split.first')}>⏮</button>
        <button onclick={() => go(index - 1)} disabled={index === 0} aria-label={$t('split.previous')} title={$t('split.previous')}>◀</button>
        <span class="count" data-testid="split-count">{steps.length ? `${index + 1} / ${steps.length}` : '0 / 0'}</span>
        <button onclick={() => go(index + 1)} disabled={index >= steps.length - 1} aria-label={$t('split.next')} title={$t('split.next')}>▶</button>
        <button onclick={() => go(steps.length - 1)} disabled={index >= steps.length - 1} aria-label={$t('split.last')} title={$t('split.last')}>⏭</button>
    </div>

    {#if source.kind === 'match' && step}
        <div class="played" data-testid="split-played">
            {$t('split.moveAt', { game: step.gameNumber, move: step.moveNumber })}{#if step.played}
                · {step.played}{/if}
        </div>
    {/if}

    <div class="board" bind:this={boardHost}>
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
        /* The other half of the window, at the interface scale (see .main-container in App). */
        zoom: var(--ui-scale, 1);
        position: fixed;
        top: 0;
        bottom: 0;
        right: 0;
        width: calc(50vw / var(--ui-scale, 1));
        box-sizing: border-box;
        display: flex;
        flex-direction: column;
        background: var(--color-surface);
        color: var(--color-text);
        border-left: 3px solid var(--group-color);
        overflow: hidden;
        cursor: pointer;
    }
    .split-viewer.left {
        right: auto;
        left: 0;
        border-left: none;
        border-right: 3px solid var(--group-color);
    }
    header,
    .bar,
    .played {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 8px;
        flex-shrink: 0;
    }
    header {
        border-bottom: 1px solid var(--color-border);
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
        font-size: var(--font-size-small);
        border: 1px solid var(--group-color);
        border-radius: 999px;
        padding: 0 6px;
        white-space: nowrap;
    }
    .hint {
        font-size: var(--font-size-small);
        opacity: 0.7;
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .controls {
        cursor: default;
    }
    select {
        flex: 1;
        min-width: 0;
    }
    .close {
        background: none;
        border: none;
        cursor: pointer;
    }
    .count {
        min-width: 5em;
        text-align: center;
        font-variant-numeric: tabular-nums;
    }
    .played {
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
        max-height: 35%;
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
