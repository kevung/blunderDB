<script>
    import { logger } from '../utils/logger.js';
    import { onDestroy } from 'svelte';
    import { LoadMetadata, SaveMetadata } from '../../wailsjs/go/database/Database.js';
    import { activeTabStore } from '../stores/uiStore';
    import { t } from '../i18n';

    let user = $state('');
    let description = $state('');
    let dateOfCreation = $state('');
    let databaseVersion = $state('');
    let loaded = $state(false);

    async function loadMetadata() {
        try {
            const metadata = await LoadMetadata();
            user = metadata.user || '';
            description = metadata.description || '';
            dateOfCreation = metadata.dateOfCreation || '';
            databaseVersion = metadata.database_version || '';
            loaded = true;
        } catch (error) {
            logger.error('Error loading metadata:', error);
        }
    }

    async function saveMetadata() {
        if (!loaded) return;
        try {
            await SaveMetadata({ user, description, dateOfCreation });
        } catch (error) {
            logger.error('Error saving metadata:', error);
        }
    }

    // Load when tab becomes active, save when leaving
    let wasActive = false;
    $effect(() => {
        const value = $activeTabStore;
        if (value === 'metadata') {
            loadMetadata().then(() => {
                wasActive = true;
            });
        } else if (wasActive) {
            saveMetadata();
            wasActive = false;
        }
    });

    onDestroy(() => {
        if (wasActive) saveMetadata();
    });
</script>

<div class="metadata-panel">
    <div class="meta-row">
        <div class="form-group">
            <label for="meta-user">{$t('metadata.user')}</label>
            <input id="meta-user" type="text" bind:value={user} onblur={saveMetadata} />
        </div>
        <div class="form-group">
            <label for="meta-date">{$t('metadata.created')}</label>
            <input id="meta-date" type="date" bind:value={dateOfCreation} onchange={saveMetadata} />
        </div>
        <div class="form-group">
            <label for="meta-version">{$t('metadata.version')}</label>
            <input id="meta-version" type="text" bind:value={databaseVersion} readonly />
        </div>
    </div>
    <div class="form-group desc-group">
        <label for="meta-description">{$t('metadata.description')}</label>
        <textarea id="meta-description" bind:value={description} onblur={saveMetadata} rows="2"></textarea>
    </div>
</div>

<style>
    .metadata-panel {
        padding: 6px 10px;
        display: flex;
        flex-direction: column;
        gap: 4px;
        height: 100%;
        overflow-y: auto;
        background: white;
        box-sizing: border-box;
    }

    .meta-row {
        display: flex;
        gap: 12px;
        align-items: flex-end;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 1px;
    }

    .desc-group {
        flex: 1;
        min-height: 0;
    }

    label {
        font-size: 10px;
        font-weight: 600;
        color: #888;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        user-select: none;
        -webkit-user-select: none;
        font-size: 12px;
        font-family: inherit;
    }

    textarea {
        flex: 1;
        min-height: 30px;
        resize: vertical;
    }

    input:read-only {
        background: #f5f5f5;
        color: #888;
    }

    input:focus,
    textarea:focus {
        outline: none;
        border-color: #1a73e8;
    }
</style>
