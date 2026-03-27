<script>
    import { onDestroy } from 'svelte';
    import { LoadMetadata, SaveMetadata } from '../../wailsjs/go/main/Database.js';
    import { activeTabStore } from '../stores/uiStore';

    let user = '';
    let description = '';
    let dateOfCreation = '';
    let databaseVersion = '';
    let loaded = false;

    async function loadMetadata() {
        try {
            const metadata = await LoadMetadata();
            user = metadata.user || '';
            description = metadata.description || '';
            dateOfCreation = metadata.dateOfCreation || '';
            databaseVersion = metadata.database_version || '';
            loaded = true;
        } catch (error) {
            console.error('Error loading metadata:', error);
        }
    }

    async function saveMetadata() {
        if (!loaded) return;
        try {
            await SaveMetadata({ user, description, dateOfCreation });
        } catch (error) {
            console.error('Error saving metadata:', error);
        }
    }

    // Load when tab becomes active, save when leaving
    let wasActive = false;
    const unsubscribe = activeTabStore.subscribe(async value => {
        if (value === 'metadata') {
            await loadMetadata();
            wasActive = true;
        } else if (wasActive) {
            await saveMetadata();
            wasActive = false;
        }
    });

    onDestroy(() => {
        unsubscribe();
        if (wasActive) saveMetadata();
    });
</script>

<div class="metadata-panel">
    <div class="meta-row">
        <div class="form-group">
            <label for="meta-user">User</label>
            <input id="meta-user" type="text" bind:value={user} on:blur={saveMetadata} />
        </div>
        <div class="form-group">
            <label for="meta-date">Created</label>
            <input id="meta-date" type="date" bind:value={dateOfCreation} on:change={saveMetadata} />
        </div>
        <div class="form-group">
            <label for="meta-version">Version</label>
            <input id="meta-version" type="text" bind:value={databaseVersion} readonly />
        </div>
    </div>
    <div class="form-group desc-group">
        <label for="meta-description">Description</label>
        <textarea id="meta-description" bind:value={description} on:blur={saveMetadata} rows="2"></textarea>
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
    }

    input, textarea {
        padding: 3px 6px;
        border: 1px solid #ccc;
        border-radius: 3px;
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

    input:focus, textarea:focus {
        outline: none;
        border-color: #1a73e8;
    }
</style>
