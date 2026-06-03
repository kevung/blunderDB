<script>
    import { trapFocus } from '../utils/focusTrap.js';
    import { t } from '../i18n';
    import { TOURS, startTour } from '../services/tourService.js';

    let { visible = false, onClose } = $props();

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            onClose();
        }
    }

    function launch(id) {
        // startTour() closes the active modal itself, then drives the tour.
        startTour(id);
    }
</script>

{#if visible}
    <div class="modal-overlay" onclick={onClose} onkeydown={handleKeyDown} role="dialog" aria-modal="true" aria-label={$t('tour.catalogTitle')} use:trapFocus>
        <div class="modal-content" onclick={(e) => e.stopPropagation()}>
            <div class="close-button" onclick={onClose}>×</div>
            <h2>{$t('tour.catalogTitle')}</h2>
            <p class="catalog-desc">{$t('tour.catalogDesc')}</p>

            <ul class="tour-list">
                {#each TOURS as tour (tour.id)}
                    <li>
                        <div class="tour-text">
                            <span class="tour-title">{$t(tour.titleKey)}</span>
                            <span class="tour-desc">{$t(tour.descKey)}</span>
                        </div>
                        <button class="start-button" onclick={() => launch(tour.id)}>{$t('tour.start')}</button>
                    </li>
                {/each}
            </ul>

            <div class="modal-buttons">
                <button class="primary-button" onclick={onClose}>{$t('common.close')}</button>
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
    }

    .modal-content {
        background-color: white;
        padding: 1rem;
        border-radius: 4px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        z-index: 1000;
        width: 90%;
        max-width: 460px;
        max-height: 80vh;
        overflow-y: auto;
        position: relative;
        display: flex;
        flex-direction: column;
        text-align: center;
    }

    h2 {
        margin: 0 0 0.5rem;
        font-size: 1.25rem;
    }

    .catalog-desc {
        margin: 0 0 1rem;
        color: #555;
        font-size: 0.95rem;
    }

    .close-button {
        position: absolute;
        top: 8px;
        right: 8px;
        font-size: 1.5rem;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        z-index: 10;
    }

    .tour-list {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .tour-list li {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        padding: 10px 12px;
        border: 1px solid #e0e0e0;
        border-radius: 6px;
        text-align: left;
    }

    .tour-text {
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .tour-title {
        font-weight: 600;
    }

    .tour-desc {
        font-size: 0.85rem;
        color: #666;
    }

    .start-button {
        flex: 0 0 auto;
        padding: 6px 14px;
        border: none;
        border-radius: 4px;
        background-color: #6c757d;
        color: white;
        cursor: pointer;
        font-size: 14px;
    }

    .start-button:hover {
        background-color: #5a6268;
    }

    .modal-buttons {
        margin-top: 14px;
        display: flex;
        justify-content: center;
    }

    .primary-button {
        padding: 8px 14px;
        border: 1px solid #ccc;
        border-radius: 4px;
        background-color: #f5f5f5;
        cursor: pointer;
        font-size: 15px;
    }

    .primary-button:hover {
        background-color: #e9e9e9;
    }
</style>
