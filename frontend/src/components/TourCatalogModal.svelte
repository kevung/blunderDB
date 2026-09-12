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
        <button onclick={onClose}>{$t('common.close')}</button>
    {/snippet}
</Modal>

<style>
    .catalog-desc {
        margin: 0;
        color: var(--color-text-muted);
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
        border: 1px solid var(--color-border);
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
        color: var(--color-text-muted);
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
        border-top: 1px solid var(--color-border);
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 12px;
        text-align: left;
    }

    .demo-hint {
        font-size: var(--font-size-base);
        color: var(--color-text-muted);
    }

    .demo-button {
        flex: 0 0 auto;
        padding: 6px 14px;
        border: 1px solid var(--color-text-muted);
        border-radius: 4px;
        background-color: var(--color-surface);
        color: var(--color-text-muted);
        cursor: pointer;
        font-size: var(--font-size-base);
    }

    .demo-button:hover {
        background-color: var(--color-surface-alt);
    }
</style>
