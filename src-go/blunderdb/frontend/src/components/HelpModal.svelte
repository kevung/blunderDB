<!-- HelpModal.svelte -->
<script>

    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';

    export let visible = false;
    export let onClose;
    export let handleGlobalKeydown;

    let activeTab = "manual"; // Default active tab
    const tabs = ['manual', 'shortcuts', 'commands', 'about'];
    let contentArea;

    function switchTab(tab) {
        activeTab = tab;
    }

    function activateGlobalShortcuts() {
        window.addEventListener('keydown', handleGlobalKeydown);
    }

    function deactivateGlobalShortcuts() {
        window.removeEventListener('keydown', handleGlobalKeydown);
    }


    // Close on Esc key
    function handleKeyDown(event) {

        // Prevent default action and stop event propagation
        event.stopPropagation();

        if(visible) {
            event.preventDefault();
            if (event.key === 'Escape') {
                onClose();
            } else if (event.ctrlKey && event.code === 'KeyH') {
                onClose();
            } else if (!event.ctrlKey && event.key === '?') {
                onClose();
            } else if (!event.ctrlKey && event.key === 'ArrowRight') {
                navigateTabs(1); // Move to the next tab
            } else if (!event.ctrlKey && event.key === 'ArrowLeft') {
                navigateTabs(-1); // Move to the previous tab
            } else if (!event.ctrlKey && event.key === 'l') {
                navigateTabs(1); // Move to the next tab
            } else if (!event.ctrlKey && event.key === 'h') {
                navigateTabs(-1); // Move to the previous tab
            } else if (!event.ctrlKey && event.key === 'ArrowDown') {
                scrollContent(1); // Scroll down
            } else if (!event.ctrlKey && event.key === 'ArrowUp') {
                scrollContent(-1); // Scroll up
            } else if (!event.ctrlKey && event.key === 'j') {
                scrollContent(1); // Scroll down
            } else if (!event.ctrlKey && event.key === 'k') {
                scrollContent(-1); // Scroll up
            } else if (!event.ctrlKey && event.key === 'PageDown') {
                scrollContent('bottom'); // Go to the bottom of the page
            } else if (!event.ctrlKey && event.key === 'PageUp') {
                scrollContent('top'); // Go to the top of the page
            } else if (!event.ctrlKey && event.key === ' ') { // Space key
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
        if (modalContent && !modalContent.contains(event.target)) {
            onClose(); // Close the help modal if the click is outside of it
        }
    }

    onMount(() => {
        window.addEventListener('keydown', handleKeyDown);
        window.addEventListener('click', handleClickOutside); // Add click event listener
        deactivateGlobalShortcuts();
    });

    onDestroy(() => {
        window.removeEventListener('keydown', handleKeyDown);
        window.removeEventListener('click', handleClickOutside); // Remove click event listener
        activateGlobalShortcuts();
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
        window.addEventListener('click', handleClickOutside); // Add click event listener
        deactivateGlobalShortcuts();
    } else {
        window.removeEventListener('keydown', handleKeyDown);
        window.removeEventListener('click', handleClickOutside); // Remove click event listener
        activateGlobalShortcuts();
    }

</script>

{#if visible}
    <div class="modal-overlay" id="helpModal" transition:fade={{ duration: 30 }}>
        <div class="modal-content" id="modalContent">
            <div class="close-button" on:click={onClose} on:keydown={handleKeyDown}>×</div>

            <!-- Tabs -->
            <div class="tab-header">
                <button class="{activeTab === 'manual' ? 'active' : ''}" on:click={() => switchTab('manual')}>Manual</button>
                <button class="{activeTab === 'shortcuts' ? 'active' : ''}" on:click={() => switchTab('shortcuts')}>Shortcut</button>
                <button class="{activeTab === 'commands' ? 'active' : ''}" on:click={() => switchTab('commands')}>Command Line</button>
                <button class="{activeTab === 'about' ? 'active' : ''}" on:click={() => switchTab('about')}>About</button>
            </div>

            <!-- Tab Content -->
            <div class="tab-content" bind:this={contentArea}>
                {#if activeTab === 'manual'}
                    <h2>Manual</h2>
                    <p>Here is the user manual for blunderDB...</p>
                {/if}

                {#if activeTab === 'shortcuts'}
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

                            <tr>
                                <td>Ctrl + S</td>
                                <td>Save Position</td>
                            </tr>

                            <tr>
                                <td>Ctrl + U</td>
                                <td>Update Position</td>
                            </tr>

                            <tr>
                                <td>Del</td>
                                <td>Delete Position</td>
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

                    <h3>Display</h3>

                    <table>
                        <thead>
                            <tr>
                                <th>Shortcut</th>
                                <th>Description</th>
                            </tr>
                        </thead>

                        <tbody>

                            <tr>
                                <td>Ctrl + ArrowLeft</td>
                                <td>Set Board Orientation to Left</td>
                            </tr>

                            <tr>
                                <td>Ctrl + ArrowRight</td>
                                <td>Set Board Orientation to Right</td>
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
                                <td>Edit Mode</td>
                            </tr>

                            <tr>
                                <td>Space</td>
                                <td>Command Mode</td>
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
                    <h3>Database</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Command</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>new, ne, n</td>
                                <td>Create a new database</td>
                            </tr>
                            <tr>
                                <td>open, op, o</td>
                                <td>Open an existing database</td>
                            </tr>
                            <tr>
                                <td>quit, q</td>
                                <td>Exit blunderDB</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Position</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Command</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>import, i</td>
                                <td>Import a position</td>
                            </tr>
                            <tr>
                                <td>write, wr, w</td>
                                <td>Save a position</td>
                            </tr>
                            <tr>
                                <td>write!, wr!, w!</td>
                                <td>Update a position</td>
                            </tr>
                            <tr>
                                <td>delete, del, d</td>
                                <td>Delete a position</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Navigation</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Command</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>[number]</td>
                                <td>Go to a specific position by index</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Tools</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Command</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>list, l</td>
                                <td>Show Analysis</td>
                            </tr>
                            <tr>
                                <td>comment, co</td>
                                <td>Write Comments</td>
                            </tr>
                            <tr>
                                <td>help, he, h</td>
                                <td>Open Help</td>
                            </tr>
                            <tr>
                                <td>#tag1 tag2 ...</td>
                                <td>Tag Position</td>
                            </tr>
                        </tbody>
                    </table>
                    <!-- Add more categories as needed -->
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
        padding: 0; /* Remove padding */
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
        margin-bottom: 0; /* Remove bottom margin */
        height: auto; /* Adjust height */
        padding: 0; /* Remove padding */
    }

    .tab-header button {
        flex: 1;
        padding: 0; /* Remove padding */
        background-color: #eee;
        border: none;
        cursor: pointer;
        font-size: 16px;
        outline: none;
        display: flex; /* Use flexbox */
        justify-content: center; /* Center horizontally */
        align-items: center; /* Center vertically */
        text-align: center; /* Center text */
        line-height: 35px; /* Ensure text is centered vertically */
        height: 35px; /* Set a fixed height */
        border-radius: 4px 4px 0 0; /* Add rounded corners to the top */
    }

    .tab-header button.active {
        background-color: #ccc;
        font-weight: bold;
    }

    .tab-content {
        flex-grow: 1;
        overflow-y: auto;
        border-top: 1px solid #ddd;
        padding: 0; /* Remove padding */
        box-sizing: border-box;
        height: calc(100% - 50px); /* Adjust height to ensure uniform tab size */
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

