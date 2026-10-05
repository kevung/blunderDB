<!--
  DuelClocks — score, cube and clocks of the open Duel (ADR-0073), in one line. Mounted in the
  Duel tab and in the status bar, where it stays visible with the tab folded. Nothing is
  computed here that the Arbiter decides: the reserves come from its state, run down to now.
-->
<script>
    import { t } from '../i18n';
    import { duelNowStore } from '../stores/duelStore.js';
    import { clockView, formatClock, scoreline, humanSide } from '../services/duel.js';

    let {
        /** A `duel.State`. */
        duel = null,
        /** Compact: no player names (status bar). */
        compact = false
    } = $props();

    let line = $derived(scoreline(duel));
    let clocks = $derived(clockView(duel, $duelNowStore));
    let human = $derived(humanSide(duel));
    let names = $derived([duel?.header?.player1 || $t('duel.player1'), duel?.header?.player2 || $t('duel.player2')]);
    let cube = $derived(duel?.awaiting?.position?.cube?.value ?? 0);
</script>

{#if duel}
    <span class="duel-clocks" data-testid="duel-clocks">
        {#each [0, 1] as side (side)}
            <span class="side" class:running={clocks?.running === side} class:you={human === side}>
                {#if !compact}<span class="name">{names[side]}</span>{/if}
                <span class="points">{line.score[side]}</span>
                {#if clocks}
                    <span class="clock" class:out={clocks.reserve[side] <= 0}>{formatClock(clocks.reserve[side])}</span>
                {/if}
            </span>
            {#if side === 0}<span class="sep">·</span>{/if}
        {/each}
        <span class="length">{line.length ? $t('duel.lengthPoints', { n: line.length }) : $t('duel.moneySession')}</span>
        {#if cube > 1}<span class="cube">{$t('duel.cubeValue', { n: cube })}</span>{/if}
        {#if clocks && clocks.running >= 0 && clocks.delayLeft > 0}
            <span class="delay">{$t('duel.delayLeft', { s: Math.ceil(clocks.delayLeft / 1000) })}</span>
        {/if}
    </span>
{/if}

<style>
    .duel-clocks {
        display: inline-flex;
        align-items: center;
        gap: 0.5em;
        font-variant-numeric: tabular-nums;
    }

    .side {
        display: inline-flex;
        gap: 0.35em;
        align-items: baseline;
    }

    .side.you .name {
        font-weight: 600;
    }

    .points {
        font-weight: 600;
    }

    .running .clock {
        color: var(--color-primary);
        font-weight: 600;
    }

    .clock.out {
        color: var(--color-danger);
    }

    .sep,
    .length,
    .cube,
    .delay {
        color: var(--color-text-muted);
    }
</style>
