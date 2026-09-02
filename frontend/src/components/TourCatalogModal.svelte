<script>
    import Modal from './Modal.svelte';
    import { t } from '../i18n';
    import { TOURS, startTour } from '../services/tourService.js';
    import { loadDemoDatabase } from '../services/databaseService.js';

    let { visible = false, onClose } = $props();

    function loadDemo() {
        onClose();
        loadDemoDatabase();
    }

    function launch(id) {
        // startTour() closes the active modal itself, then drives the tour.
        startTour(id);
    }
</script>

<Modal open={visible} onclose={onClose} size="medium" align="center" closeOnOverlay>
    {#snippet title()}{$t('tour.catalogTitle')}{/snippet}
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

    <div class="demo-row">
        <span class="demo-hint">{$t('tour.demoHint')}</span>
        <button class="demo-button" onclick={loadDemo}>{$t('tour.loadDemo')}</button>
    </div>

    {#snippet footer()}
        <button class="primary-button" onclick={onClose}>{$t('common.close')}</button>
    {/snippet}
</Modal>

<style>
    .catalog-desc {
        margin: 0;
        color: #555;
        font-size: var(--font-size-title);
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
        font-size: var(--font-size-base);
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
        font-size: var(--font-size-base);
    }

    .start-button:hover {
        background-color: #5a6268;
    }

    .demo-row {
        padding-top: 12px;
        border-top: 1px solid #eee;
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        text-align: left;
    }

    .demo-hint {
        font-size: var(--font-size-base);
        color: #666;
    }

    .demo-button {
        flex: 0 0 auto;
        padding: 6px 14px;
        border: 1px solid #6c757d;
        border-radius: 4px;
        background-color: white;
        color: #6c757d;
        cursor: pointer;
        font-size: var(--font-size-base);
    }

    .demo-button:hover {
        background-color: #f0f0f0;
    }

    .primary-button {
        padding: 8px 14px;
        border: 1px solid #ccc;
        border-radius: 4px;
        background-color: #f5f5f5;
        cursor: pointer;
        font-size: var(--font-size-title);
    }

    .primary-button:hover {
        background-color: #e9e9e9;
    }
</style>
