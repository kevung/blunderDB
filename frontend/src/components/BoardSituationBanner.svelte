<script>
    // Bandeau posé sur le plateau quand il ne montre pas la position courante (services/boardSituation.js).
    import { t } from '../i18n';
    import { boardBannerKeyStore } from '../services/boardSituation.js';
</script>

{#if $boardBannerKeyStore}
    <div class="board-situation-banner" data-testid="board-situation-banner" role="status">{$t($boardBannerKeyStore)}</div>
{/if}

<style>
    .board-situation-banner {
        position: absolute;
        top: 6px;
        left: 50%;
        transform: translateX(-50%);
        z-index: 5;
        max-width: calc(100% - 120px);
        padding: 2px 10px;
        border-radius: 3px;
        background: var(--color-surface-alt);
        color: var(--color-text);
        border: 1px solid var(--color-border);
        font-size: var(--font-size-small);
        pointer-events: none;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }
</style>
