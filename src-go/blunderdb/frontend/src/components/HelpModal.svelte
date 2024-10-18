<!-- HelpModal.svelte -->
<script>

    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';

    export let visible = false;
    export let onClose;

    let activeTab = "manual"; // Default active tab
    const tabs = ['manual', 'shortcuts', 'commands', 'about'];
    let contentArea;

    function switchTab(tab) {
        activeTab = tab;
    }

    // Close on Esc key
    function handleKeyDown(event) {

        // Prevent default action and stop event propagation
        event.stopPropagation();

        if(visible) {
            event.preventDefault();
            if (event.key === 'Escape') {
                onClose();
            } else if (event.key === 'ArrowRight') {
                navigateTabs(1); // Move to the next tab
            } else if (event.key === 'ArrowLeft') {
                navigateTabs(-1); // Move to the previous tab
            } else if (event.key === 'l') {
                navigateTabs(1); // Move to the next tab
            } else if (event.key === 'h') {
                navigateTabs(-1); // Move to the previous tab
            } else if (event.key === 'ArrowDown') {
                scrollContent(1); // Scroll down
            } else if (event.key === 'ArrowUp') {
                scrollContent(-1); // Scroll up
            } else if (event.key === 'j') {
                scrollContent(1); // Scroll down
            } else if (event.key === 'k') {
                scrollContent(-1); // Scroll up
            } else if (event.key === 'PageDown') {
                scrollContent('bottom'); // Go to the bottom of the page
            } else if (event.key === 'PageUp') {
                scrollContent('top'); // Go to the top of the page
            } else if (event.key === ' ') { // Space key
                scrollContent('page'); // Scroll down by the height of the content
            }
        }
    }

    function navigateTabs(direction) {
        const currentIndex = tabs.indexOf(activeTab);
        const newIndex = (currentIndex + direction + tabs.length) % tabs.length;
        switchTab(tabs[newIndex]);
    }

    
    function scrollContent(direction) {
        if (contentArea) {
            const scrollAmount = 60; // Pixels to scroll per key press

            if (direction === 1) { // Arrow down
                contentArea.scrollTop += scrollAmount;
            } else if (direction === -1) { // Arrow up
                contentArea.scrollTop -= scrollAmount;
            } else if (direction === 'bottom') { // PageDown
                contentArea.scrollTop = contentArea.scrollHeight; // Go to bottom
            } else if (direction === 'top') { // PageUp
                contentArea.scrollTop = 0; // Go to top
            } else if (direction === 'page') { // Space key
                contentArea.scrollTop += contentArea.clientHeight; // Scroll down by the visible area height
            }
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
            <div class="tab-content" bind:this={contentArea}>
                {#if activeTab === 'manual'}
                    <h2>Manual</h2>
                    <p>Here is the user manual for blunderDB...</p>
                {/if}

                {#if activeTab === 'shortcuts'}
                    <h2>Shortcut Summary</h2>

                    <h3>Database</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Shortcut</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>

                            <tr>
                                <td>Ctrl + N</td>
                                <td>New Database</td>
                            </tr>

                            <tr>
                                <td>Ctrl + O</td>
                                <td>Open Database</td>
                            </tr>

                            <tr>
                                <td>Ctrl + Q</td>
                                <td>Exit blunderDB</td>
                            </tr>

                        </tbody>
                    </table>

                    <h3>Position</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Shortcut</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>

                            <tr>
                                <td>Ctrl + I</td>
                                <td>Import Position</td>
                            </tr>

                            <tr>
                                <td>Ctrl + C</td>
                                <td>Copy Position</td>
                            </tr>

                            <tr>
                                <td>Ctrl + V</td>
                                <td>Paste Position</td>
                            </tr>

                        </tbody>
                    </table>

                    <h3>Navigation</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Shortcut</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>

                            <tr>
                                <td>PageUp, h</td>
                                <td>First Position</td>
                            </tr>

                            <tr>
                                <td>Left, k</td>
                                <td>Previous Position</td>
                            </tr>

                            <tr>
                                <td>Right, j</td>
                                <td>Next Position</td>
                            </tr>

                            <tr>
                                <td>PageDown, l</td>
                                <td>Last Position</td>
                            </tr>

                            <tr>
                                <td>Ctrl-K</td>
                                <td>Go To Position</td>
                            </tr>

                        </tbody>
                    </table>

                    <h3>Modes</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Shortcut</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>

                            <tr>
                                <td>Tab</td>
                                <td>Toggle Edit Mode</td>
                            </tr>

                            <tr>
                                <td>Space</td>
                                <td>Switch To Command Mode</td>
                            </tr>

                        </tbody>
                    </table>

                    <h3>Tools</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Shortcut</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>

                            <tr>
                                <td>Ctrl + L</td>
                                <td>Show Analysis</td>
                            </tr>

                            <tr>
                                <td>Ctrl + P</td>
                                <td>Write Comments</td>
                            </tr>

                            <tr>
                                <td>Ctrl + F</td>
                                <td>Find Position</td>
                            </tr>

                            <tr>
                                <td>Ctrl + H, ?</td>
                                <td>Open Help</td>
                            </tr>

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
        border-radius: 4px;
        width: 80%;
        height: 70%; /* Fix height to 70% of the viewport */
        box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
        position: relative;
        display: flex;
        flex-direction: column;
    }

    .close-button {
        position: absolute;
        top: -8px;
        right: 4px;
        font-size: 24px;
        font-weight: bold;
        color: #666;
        cursor: pointer;
        z-index: 10;
        transition: background-color 0.3s ease, opacity 0.3s ease;
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
        margin: 0 auto;
        width: 80%;
        border-collapse: collapse;
    }

    th, td {
        padding: 12px;
        text-align: center;
        border-bottom: 1px solid #ddd;
        width: 50%;
    }

    th {
        background-color: #f4f4f4;
    }

    tr:hover {
        background-color: #f1f1f1;
    }
</style>

