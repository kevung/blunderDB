<script>
    // Bandeau posé sur le plateau quand il ne montre pas la position courante (services/boardSituation.js).
    import { t } from '../i18n';
    import { boardBannerKeyStore } from '../services/boardSituation.js';

    // Board.svelte measures only on 'resize': refit once the layout has reflowed.
    $effect(() => {
        void $boardBannerKeyStore;
        requestAnimationFrame(() => window.dispatchEvent(new Event('resize')));
    });
</script>

{#if $boardBannerKeyStore}
    <div class="board-situation-banner" data-testid="board-situation-banner" role="status">{$t($boardBannerKeyStore)}</div>
{/if}

<style>
    /* In the flow, above the board area: the board refits under it. */
    .board-situation-banner {
        padding: 2px 10px;
        background: var(--color-surface-alt);
        color: var(--color-text);
        border-bottom: 1px solid var(--color-border);
        font-size: var(--font-size-small);
        text-align: center;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
</style>
