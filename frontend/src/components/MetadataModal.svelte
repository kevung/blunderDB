<script>
    import { onMount, onDestroy } from 'svelte';
    import { LoadMetadata, SaveMetadata } from '../../wailsjs/go/main/Database.js'; // Import functions

    export let visible = false;
    export let onClose;

    let user = '';
    let description = '';
    let dateOfCreation = '';

    async function loadMetadata() {
        try {
            const metadata = await LoadMetadata();
            user = metadata.user || '';
            description = metadata.description || '';
            dateOfCreation = metadata.dateOfCreation || '';
        } catch (error) {
            console.error('Error loading metadata:', error);
        }
    }

    async function saveMetadata() {
        try {
            await SaveMetadata({ user, description, dateOfCreation });
        } catch (error) {
            console.error('Error saving metadata:', error);
        }
    }

    function handleClose() {
        saveMetadata();
        onClose();
    }

    $: if (visible) {
        loadMetadata();
    }

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            handleClose();
        }
    }

    function handleClickOutside(event) {
        if (event.target.classList.contains('modal-overlay')) {
            handleClose();
        }
    }

    onMount(() => {
        window.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        window.removeEventListener('keydown', handleKeyDown);
    });
</script>

{#if visible}
    <div class="modal-overlay" on:click={handleClickOutside}>
        <div class="modal-content">
            <!-- Metadata title removed -->
            <div class="form-group">
                <label for="user">User:</label>
                <input id="user" type="text" bind:value={user} />
            </div>
            <div class="form-group">
                <label for="description">Description:</label>
                <textarea id="description" bind:value={description} style="height: 150px;"></textarea>
            </div>
            <div class="form-group">
                <label for="dateOfCreation">Date of Creation:</label>
                <input id="dateOfCreation" type="date" bind:value={dateOfCreation} />
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
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        justify-content: center;
        align-items: center;
        z-index: 1000;
    }

    .modal-content {
        background: white;
        padding: 1rem;
        border-radius: 8px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        z-index: 1000;
        width: 90%;
        max-width: 800px;
        max-height: 80vh;
        overflow-y: auto;
        padding-left: 1rem;
        display: flex;
        flex-direction: column;
        gap: 15px;
    }

    .form-group {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    label {
        margin-bottom: 5px;
        font-size: 18px;
    }

    input, textarea {
        padding: 8px;
        border: 1px solid #ccc;
        border-radius: 4px;
        font-size: 18px;
    }
</style>
