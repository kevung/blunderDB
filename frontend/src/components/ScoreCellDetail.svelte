<script module>
    // A finished evaluation per (board, score, depth, search settings), kept for the session: a
    // return to a cell is instant. Bounded, the oldest going first.
    const MEMORY_MAX = 200;
    // A plain cache, read and written outside any rendering: nothing reacts to it.
    // eslint-disable-next-line svelte/prefer-svelte-reactivity
    const memory = /** @type {Map<string, any>} */ (new Map());
    /** @param {string} key @param {any} value */
    function remember(key, value) {
        memory.delete(key);
        memory.set(key, value);
        if (memory.size > MEMORY_MAX) memory.delete(memory.keys().next().value);
    }
</script>

<script>
    // The candidate plays of a checker-map cell, at that cell's score: the table of the Decision
    // view, filled by the same engine calls — the 0-ply answer at once, then the configured depth.
    // Changing cell cancels the search; a cell already evaluated at this depth answers from memory.
    import { onDestroy } from 'svelte';
    import { EventsOn } from '../../wailsjs/runtime/runtime.js';
    import { EvaluatePositionImmediate, StartEvaluationAtRest, CancelEvaluationAtRest } from '../../wailsjs/go/gui/App.js';
    import { GetGammonNetDisplayPly, GetGammonNetPruneK, GetGammonNetCandidates } from '../../wailsjs/go/main/Config.js';
    import { selectedMoveStore } from '../stores/analysisStore';
    import { positionAtScore } from '../utils/scoreMap.js';
    import { nextMapTag } from '../utils/mapTag.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';
    import CandidateMovesTable from './CandidateMovesTable.svelte';

    // The tag is offset so it never meets the Decision view's own search numbers.
    const TAG_BASE = 1_000_000_000;

    let { cell, position, onSettled = () => {} } = $props();

    let moves = $state(/** @type {Array<any>} */ ([]));
    let preRoll = $state(/** @type {any} */ (null));
    let settled = $state(false);
    let failed = $state('');
    let generation = 0;
    let tag = 0;

    let target = $derived(positionAtScore(position, cell));

    $effect(() => {
        const pos = target;
        run(pos);
        return () => {
            generation += 1;
            CancelEvaluationAtRest().catch(() => {});
            selectedMoveStore.set(null);
        };
    });

    onDestroy(() => {
        generation += 1;
    });

    const off = EventsOn('gammonnet-eval:done', (result) => {
        if (!tag || result?.tag !== tag) return;
        apply(result);
        settle();
    });
    onDestroy(() => off && off());

    /** @param {any} result */
    function apply(result) {
        if (result?.refused) {
            moves = [];
            return;
        }
        if (result?.moves !== undefined) moves = result.moves;
        if (result?.preRoll) preRoll = result.preRoll;
    }

    function settle() {
        settled = true;
        remember(memoryKey, { moves, preRoll });
        onSettled();
    }

    let memoryKey = '';

    /** @param {any} pos */
    async function run(pos) {
        const mine = ++generation;
        tag = 0;
        settled = false;
        failed = '';
        try {
            const [ply, pruneK, candidates] = await Promise.all([GetGammonNetDisplayPly(), GetGammonNetPruneK(), GetGammonNetCandidates()]);
            if (mine !== generation) return;
            memoryKey = JSON.stringify([pos, ply, pruneK, candidates]);
            const known = memory.get(memoryKey);
            if (known) {
                moves = known.moves;
                preRoll = known.preRoll;
                settled = true;
                return;
            }
            moves = [];
            preRoll = null;
            const first = await EvaluatePositionImmediate(pos, pruneK, candidates);
            if (mine !== generation) return;
            apply(first);
            if (ply <= 0 || first?.refused) {
                settle();
                return;
            }
            tag = TAG_BASE + nextMapTag();
            await StartEvaluationAtRest(pos, ply, pruneK, candidates, tag);
        } catch (e) {
            logger.error('score cell evaluation failed:', e);
            if (mine === generation) failed = String(e);
        }
    }

    /** @param {any} move */
    const hover = (move) => selectedMoveStore.set(move?.move ?? null);
</script>

<div class="cell-detail">
    {#if failed}
        <div class="failed" role="alert">{$t('eval.failed', { error: failed })}</div>
    {:else}
        <div class="moves-scroll">
            <CandidateMovesTable {moves} selectedMove={$selectedMoveStore} {position} onRowHover={hover} showProvenance={false} baseline={preRoll} isMoney={false} />
            {#if moves.length === 0}
                <div class="pending">{settled ? $t('cube.refused') : $t('eval.pending')}</div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .cell-detail {
        min-width: 0;
    }
    .moves-scroll {
        overflow-x: auto;
        max-width: 100%;
    }
    .pending,
    .failed {
        color: var(--color-text-muted);
        padding: var(--space-1) 0;
    }
</style>
