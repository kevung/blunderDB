<script>
    // A player's name is a way to that player's statistics (ADR-0085): the Stats panel opens with
    // the shared filter on them.
    import { t } from '../i18n';
    import { openStatsFor } from '../services/positionLoader.js';

    /** @type {{ name: string, class?: string }} */
    let { name, class: className = '' } = $props();
</script>

{#if name}
    <button type="button" class="player-stats-link {className}" data-testid="player-stats-link" title={$t('stats.openPlayerStats', { name })} onclick={() => openStatsFor({ playerName: name })}
        >{name}</button
    >
{/if}

<style>
    /* No `font: inherit`: style.css already sets it (ADR-0008). */
    .player-stats-link {
        background: none;
        border: none;
        padding: 0;
        color: inherit;
        font-weight: inherit;
        cursor: pointer;
    }
    .player-stats-link:hover {
        color: var(--color-primary);
        text-decoration: underline dotted;
    }
</style>
