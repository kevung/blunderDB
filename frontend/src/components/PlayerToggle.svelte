<script>
    // A player's entry in a chart legend: the key and the name, ticked or not.
    // The last player ticked stays so, so the chart always has decisions to answer to.
    import { t } from '../i18n';

    /** @type {{ pressed: boolean, locked: boolean, ontoggle: () => void, children: import('svelte').Snippet }} */
    let { pressed, locked, ontoggle, children } = $props();
</script>

<button
    type="button"
    class="player-toggle"
    class:off={!pressed}
    aria-pressed={pressed}
    aria-disabled={locked ? 'true' : undefined}
    title={$t('match.legendToggle')}
    onclick={() => !locked && ontoggle()}
>
    {@render children()}
</button>

<style>
    .player-toggle {
        display: inline-flex;
        align-items: center;
        padding: 0 2px;
        border: none;
        background: none;
        color: inherit;
        cursor: pointer;
    }
    .player-toggle.off {
        opacity: 0.45;
        text-decoration: line-through;
    }
    .player-toggle[aria-disabled='true'] {
        cursor: default;
    }
</style>
