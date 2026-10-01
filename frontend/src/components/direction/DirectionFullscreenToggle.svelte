<script>
    import { t } from '../../i18n';
    import { directionPageShownStore } from '../../stores/directionStore.js';
    import { directionFullscreenStore, toggleDirectionFullscreen } from '../../services/directionFullscreen.js';
</script>

<!-- Entrée et sortie à la souris du plein écran de la Direction ; le clavier a F11 et Échap. -->
{#if $directionPageShownStore}
    <button
        type="button"
        class="td-fs-toggle"
        data-testid="direction-fullscreen-toggle"
        aria-pressed={$directionFullscreenStore}
        title={$directionFullscreenStore ? $t('direction.fullscreen.exit') : $t('direction.fullscreen.enter')}
        aria-label={$directionFullscreenStore ? $t('direction.fullscreen.exit') : $t('direction.fullscreen.enter')}
        onclick={toggleDirectionFullscreen}
    >
        <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
            {#if $directionFullscreenStore}
                <path d="M9 3v4a2 2 0 0 1-2 2H3M21 9h-4a2 2 0 0 1-2-2V3M3 15h4a2 2 0 0 1 2 2v4M15 21v-4a2 2 0 0 1 2-2h4" />
            {:else}
                <path d="M3 9V5a2 2 0 0 1 2-2h4M15 3h4a2 2 0 0 1 2 2v4M21 15v4a2 2 0 0 1-2 2h-4M9 21H5a2 2 0 0 1-2-2v-4" />
            {/if}
        </svg>
    </button>
{/if}

<style>
    .td-fs-toggle {
        position: fixed;
        right: 10px;
        bottom: 10px;
        z-index: 20;
        width: 36px;
        height: 36px;
        display: flex;
        align-items: center;
        justify-content: center;
        border: 1px solid var(--border-color, #bbb);
        border-radius: 6px;
        background: var(--bg-color, #fff);
        color: inherit;
        opacity: 0.75;
        cursor: pointer;
    }
    .td-fs-toggle:hover,
    .td-fs-toggle:focus-visible {
        opacity: 1;
    }
</style>
