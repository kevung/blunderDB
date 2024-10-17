<!-- HelpModal.svelte -->
<script>

    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';

    export let visible = false;
    export let onClose;

    let activeTab = "manual"; // Default active tab

    function switchTab(tab) {
        activeTab = tab;
    }

    // Close on Esc key
    function handleKeyDown(event) {
        if (event.key === 'Escape' && visible) {
            onClose();
        }
    }

    // Close the modal when clicking outside of it
    function handleClickOutside(event) {
        const modalContent = document.getElementById('modalContent');
        if (!modalContent.contains(event.target)) {
            onClose(); // Close the help modal if the click is outside of it
        }
    }

    onMount(() => {
        window.addEventListener('keydown', handleKeyDown);
        window.addEventListener('click', handleClickOutside);
    });

    onDestroy(() => {
        window.removeEventListener('keydown', handleKeyDown);
        window.removeEventListener('click', handleClickOutside);
    });

    // Focus modal content when visible and listen for Esc key
    $: if (visible) {
        setTimeout(() => {
            const helpModal = document.getElementById('helpModal');
            if (helpModal) {
                helpModal.focus();
            }
        }, 0);
        window.addEventListener('keydown', handleKeyDown);
        window.addEventListener('click', handleClickOutside);
    } else {
        window.removeEventListener('keydown', handleKeyDown);
        window.removeEventListener('click', handleClickOutside);
    }

</script>

{#if visible}
    <div class="modal-overlay" tabindex="0" id="helpModal" transition:fade={{ duration: 30 }}>
        <div class="modal-content" id="modalContent">
            <div class="close-button" on:click={onClose}>×</div>

            <!-- Tabs -->
            <div class="tab-header">
                <button class="{activeTab === 'manual' ? 'active' : ''}" on:click={() => switchTab('manual')}>Manual</button>
                <button class="{activeTab === 'shortcuts' ? 'active' : ''}" on:click={() => switchTab('shortcuts')}>Shortcut Summary</button>
                <button class="{activeTab === 'commands' ? 'active' : ''}" on:click={() => switchTab('commands')}>Command Line Summary</button>
                <button class="{activeTab === 'about' ? 'active' : ''}" on:click={() => switchTab('about')}>About</button>
            </div>

            <!-- Tab Content -->
            <div class="tab-content">
                {#if activeTab === 'manual'}
                    <h2>Manual</h2>
                    <p>Here is the user manual for blunderDB...</p>
                {/if}

                {#if activeTab === 'shortcuts'}
                    <h2>Shortcut Summary</h2>
                    <table>
                        <thead>
                            <tr>
                                <th>Shortcut</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>Ctrl + H</td>
                                <td>Toggle help modal</td>
                            </tr>
                            <tr>
                                <td>Ctrl + L</td>
                                <td>Toggle analysis panel</td>
                            </tr>
                            <tr>
                                <td>Ctrl + P</td>
                                <td>Toggle comment zone</td>
                            </tr>
                            <tr>
                                <td>Space</td>
                                <td>Open command line</td>
                            </tr>
                            <tr>
                                <td>Escape</td>
                                <td>Close command line or help modal</td>
                            </tr>
                            <!-- Add more shortcuts as needed -->
                        </tbody>
                    </table>
                {/if}

                {#if activeTab === 'commands'}
                    <h2>Command Line Summary</h2>
                    <p>Summary of all command line commands...</p>
                {/if}

                {#if activeTab === 'about'}
                    <h2>About blunderDB</h2>
                    <p>Information about the blunderDB project...</p>
                {/if}
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
        padding: 20px;
        border-radius: 8px;
        width: 80%;
        height: 70%; /* Fix height to 70% of the viewport */
        box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
        position: relative;
        display: flex;
        flex-direction: column;
    }

    .close-button {
        position: absolute;
        top: -5px;
        right: 5px;
        font-size: 24px;
        cursor: pointer;
        z-index: 10;
    }

    .tab-header {
        display: flex;
        margin-bottom: 20px;
        height: 50px;
    }

    .tab-header button {
        flex: 1;
        padding: 10px;
        background-color: #eee;
        border: none;
        cursor: pointer;
        font-size: 16px;
        outline: none;
    }

    .tab-header button.active {
        background-color: #ccc;
        font-weight: bold;
    }

    .tab-content {
        flex-grow: 1;
        overflow-y: auto;
        border-top: 1px solid #ddd;
        padding: 20px;
        box-sizing: border-box;
        min-height: 200px;
    }

    table {
        width: 100%;
        border-collapse: collapse;
    }

    th, td {
        padding: 12px;
        text-align: left;
        border-bottom: 1px solid #ddd;
    }

    th {
        background-color: #f4f4f4;
    }

    tr:hover {
        background-color: #f1f1f1;
    }
</style>

