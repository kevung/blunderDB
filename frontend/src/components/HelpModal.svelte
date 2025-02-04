<!-- HelpModal.svelte -->
<script>

    import { onMount, onDestroy } from 'svelte';
    import { fade } from 'svelte/transition';
    import { metaStore } from '../stores/metaStore'; // Import metaStore
    import { GetDatabaseVersion } from '../../wailsjs/go/main/Database'; // Correct import path

    export let visible = false;
    export let onClose;
    export let handleGlobalKeydown;

    let activeTab = "manual"; // Default active tab
    const tabs = ['manual', 'shortcuts', 'commands', 'about'];
    let contentArea;

    let databaseVersion = '';
    let applicationVersion = '';

    // Subscribe to the metaStore
    metaStore.subscribe(value => {
        applicationVersion = value.applicationVersion;
    });

    onMount(async () => {
        try {
            databaseVersion = await GetDatabaseVersion();
        } catch (error) {
            console.error('Error fetching database version:', error);
        }
    });

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
    <div class="modal-overlay" id="helpModal" tabindex="0" transition:fade={{ duration: 30 }}>
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
                    <h3>Introduction</h3>
                    <p>blunderDB is a software for creating backgammon position databases. Its main strength is to provide a single place to aggregate positions that a player has encountered (online, in tournaments) and to be able to re-study these positions by filtering them according to various arbitrarily combinable filters. blunderDB can also be used to create catalogs of reference positions.</p>
                    <p>Positions are stored in a database represented by a .db file.</p>
                    
                    <h3>Main Interactions</h3>
                    <p>The main interactions possible with blunderDB are:</p>
                    <ul>
                        <li>adding a new position,</li>
                        <li>modifying an existing position,</li>
                        <li>deleting an existing position,</li>
                        <li>searching for one or more positions.</li>
                    </ul>
                    
                    <h3>Modes</h3>
                    <p>To do this, the user switches to dedicated modes for:</p>
                    <ul>
                        <li>navigating and viewing positions (NORMAL mode),</li>
                        <li>editing positions (EDIT mode),</li>
                        <li>editing a query to filter positions (COMMAND mode or search window).</li>
                    </ul>
                    <p>The user can freely tag positions and annotate them with comments.</p>
                    <p>The rest of the manual describes the graphical interface and the main modes of blunderDB.</p>
                    
                    <h3>Description of the GUI</h3>
                    <p>The blunderDB GUI is structured from top to bottom as follows:</p>
                    <ul>
                        <li>[at the top] the menu bar, which gathers all the main operations that can be performed on the database,</li>
                        <li>[in the middle] the main display area, which allows displaying or editing backgammon positions,</li>
                        <li>[at the bottom] the status bar, which presents various information about the database or the current position.</li>
                    </ul>
                    <p>Panels can be displayed to:</p>
                    <ul>
                        <li>display analysis data associated with the current position from eXtreme Gammon (XG),</li>
                        <li>display, add, or modify comments</li>
                    </ul>
                    <p>Modal windows can be displayed to:</p>
                    <ul>
                        <li>[EDIT mode only] set search filters,</li>
                        <li>display the blunderDB help.</li>
                    </ul>
                    <p>The main display area provides the user with:</p>
                    <ul>
                        <li>a board to display or edit a backgammon position,</li>
                        <li>the level and owner of the cube,</li>
                        <li>the race count of each player,</li>
                        <li>the score of each player,</li>
                        <li>the dice to be played. If no value is displayed on the dice, the position of the dice indicates which player has the turn and that the position is a cube decision.</li>
                    </ul>
                    <p>The status bar is structured from left to right with the following information:</p>
                    <ul>
                        <li>the current mode (NORMAL, EDIT, COMMAND),</li>
                        <li>an information message related to an operation performed by the user,</li>
                        <li>the index of the current position, followed by the number of positions in the current library.</li>
                    </ul>
                    <p>In the case of positions resulting from a user search, the number of positions indicated in the status bar corresponds to the number of filtered positions.</p>
                    
                    <h3>NORMAL Mode</h3>
                    <p>NORMAL mode is the default mode of blunderDB. It is used to:</p>
                    <ul>
                        <li>scroll through the different positions in the current library,</li>
                        <li>display analysis information associated with a position,</li>
                        <li>display, add, and modify comments on a position.</li>
                    </ul>
                    
                    <h3>EDIT Mode</h3>
                    <p>EDIT mode allows editing a position to add it to the database or to define the type of position to search for. EDIT mode is activated by pressing the *TAB* key. The distribution of checkers, the cube, the score, and the turn can be modified using the mouse.</p>
                    
                    <h3>COMMAND Mode</h3>
                    <p>COMMAND mode allows performing all the functionalities of blunderDB available in the graphical interface: general operations on the database, position navigation, displaying analysis and/or comments, searching for positions according to filters... After getting familiar with the interface, it is recommended to gradually use this mode, which allows powerful and smooth use of blunderDB, especially for position search functionalities.</p>
                    <p>To switch to COMMAND mode from any other mode, press the *SPACE* key. To send a query and exit COMMAND mode, press the *ENTER* key.</p>
                    <p>blunderDB executes the queries sent by the user provided they are valid and immediately modifies the state of the database if necessary. There are no explicit save actions required by the user.</p>
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

                            <tr>
                                <td>Ctrl + M</td>
                                <td>Edit Metadata</td>
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

                            <tr>
                                <td>Backspace</td>
                                <td>Reset board, cube, score, and dice</td>
                            </tr>

                            <tr>
                                <td>Ctrl + G</td>
                                <td>Show Position Metadata</td>
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
                                <td>Ctrl + R</td>
                                <td>Reload All Positions</td>
                            </tr>

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
                            <tr>
                                <td>meta</td>
                                <td>Edit Metadata</td>
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
                                <td>met</td>
                                <td>Open The Kazaross-XG2 Match Equity Table</td>
                            </tr>
                            <tr>
                                <td>tp2</td>
                                <td>2-Cube Take Point (Live and Last)</td>
                            </tr>
                            <tr>
                                <td>tp2_live</td>
                                <td>2-Cube Take Point in Long Races</td>
                            </tr>
                            <tr>
                                <td>tp2_last</td>
                                <td>2-Cube Take Point in Last Roll Positions</td>
                            </tr>
                            <tr>
                                <td>tp4</td>
                                <td>4-Cube Take Point (Live and Last)</td>
                            </tr>
                            <tr>
                                <td>tp4_live</td>
                                <td>4-Cube Take Point in Long Races</td>
                            </tr>
                            <tr>
                                <td>tp4_last</td>
                                <td>4-Cube Take Point in Last Roll Positions</td>
                            </tr>
                            <tr>
                                <td>gv1</td>
                                <td>1-Cube Gammon Values</td>
                            </tr>
                            <tr>
                                <td>gv2</td>
                                <td>2-Cube Gammon Values</td>
                            </tr>
                            <tr>
                                <td>gv4</td>
                                <td>4-Cube Gammon Values</td>
                            </tr>
                            <tr>
                                <td>#tag1 tag2 ...</td>
                                <td>Tag Position</td>
                            </tr>
                            <tr>
                                <td>s</td>
                                <td>Search Positions with Filters</td>
                            </tr>
                            <tr>
                                <td>e</td>
                                <td>Reload All Positions</td>
                            </tr>
                        </tbody>
                    </table>
                    <h3>Filters</h3>
                    <table>
                        <thead>
                            <tr>
                                <th>Filter</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr>
                                <td>cube, cub, cu, c</td>
                                <td>Include cube</td>
                            </tr>
                            <tr>
                                <td>score, sco, sc, s</td>
                                <td>Include score</td>
                            </tr>
                            <tr>
                                <td>d</td>
                                <td>Include decision type</td>
                            </tr>
                            <tr>
                                <td>D</td>
                                <td>Include dice roll</td>
                            </tr>
                            <tr>
                                <td>nc</td>
                                <td>Include non-contact positions</td>
                            </tr>
                            <tr>
                                <td>p>x</td>
                                <td>Pip count &gt; x</td>
                            </tr>
                            <tr>
                                <td>p&lt;x</td>
                                <td>Pip count &lt; x</td>
                            </tr>
                            <tr>
                                <td>px,y</td>
                                <td>Pip count between x and y</td>
                            </tr>
                            <tr>
                                <td>P>x</td>
                                <td>Player absolute pip count &gt; x</td>
                            </tr>
                            <tr>
                                <td>P&lt;x</td>
                                <td>Player absolute pip count &lt; x</td>
                            </tr>
                            <tr>
                                <td>Px,y</td>
                                <td>Player absolute pip count between x and y</td>
                            </tr>
                            <tr>
                                <td>e>x</td>
                                <td>Equity &gt; x (in millipoints)</td>
                            </tr>
                            <tr>
                                <td>e&lt;x</td>
                                <td>Equity &lt; x (in millipoints)</td>
                            </tr>
                            <tr>
                                <td>ex,y</td>
                                <td>Equity between x and y (in millipoints)</td>
                            </tr>
                            <tr>
                                <td>w>x</td>
                                <td>Win rate &gt; x</td>
                            </tr>
                            <tr>
                                <td>w&lt;x</td>
                                <td>Win rate &lt; x</td>
                            </tr>
                            <tr>
                                <td>wx,y</td>
                                <td>Win rate between x and y</td>
                            </tr>
                            <tr>
                                <td>g>x</td>
                                <td>Gammon rate &gt; x</td>
                            </tr>
                            <tr>
                                <td>g&lt;x</td>
                                <td>Gammon rate &lt; x</td>
                            </tr>
                            <tr>
                                <td>gx,y</td>
                                <td>Gammon rate between x and y</td>
                            </tr>
                            <tr>
                                <td>b>x</td>
                                <td>Backgammon rate &gt; x</td>
                            </tr>
                            <tr>
                                <td>b&lt;x</td>
                                <td>Backgammon rate &lt; x</td>
                            </tr>
                            <tr>
                                <td>bx,y</td>
                                <td>Backgammon rate between x and y</td>
                            </tr>
                            <tr>
                                <td>W>x</td>
                                <td>Opponent win rate &gt; x</td>
                            </tr>
                            <tr>
                                <td>W&lt;x</td>
                                <td>Opponent win rate &lt; x</td>
                            </tr>
                            <tr>
                                <td>Wx,y</td>
                                <td>Opponent win rate between x and y</td>
                            </tr>
                            <tr>
                                <td>G>x</td>
                                <td>Opponent gammon rate &gt; x</td>
                            </tr>
                            <tr>
                                <td>G&lt;x</td>
                                <td>Opponent gammon rate &lt; x</td>
                            </tr>
                            <tr>
                                <td>Gx,y</td>
                                <td>Opponent gammon rate between x and y</td>
                            </tr>
                            <tr>
                                <td>B>x</td>
                                <td>Opponent backgammon rate &gt; x</td>
                            </tr>
                            <tr>
                                <td>B&lt;x</td>
                                <td>Opponent backgammon rate &lt; x</td>
                            </tr>
                            <tr>
                                <td>Bx,y</td>
                                <td>Opponent backgammon rate between x and y</td>
                            </tr>
                            <tr>
                                <td>o>x</td>
                                <td>Player checkers off &gt; x</td>
                            </tr>
                            <tr>
                                <td>o&lt;x</td>
                                <td>Player checkers off &lt; x</td>
                            </tr>
                            <tr>
                                <td>ox,y</td>
                                <td>Player checkers off between x and y</td>
                            </tr>
                            <tr>
                                <td>O>x</td>
                                <td>Opponent checkers off &gt; x</td>
                            </tr>
                            <tr>
                                <td>O&lt;x</td>
                                <td>Opponent checkers off &lt; x</td>
                            </tr>
                            <tr>
                                <td>Ox,y</td>
                                <td>Opponent checkers off between x and y</td>
                            </tr>
                            <tr>
                                <td>k>x</td>
                                <td>Player back checkers &gt; x</td>
                            </tr>
                            <tr>
                                <td>k&lt;x</td>
                                <td>Player back checkers &lt; x</td>
                            </tr>
                            <tr>
                                <td>kx,y</td>
                                <td>Player back checkers between x and y</td>
                            </tr>
                            <tr>
                                <td>K>x</td>
                                <td>Opponent back checkers &gt; x</td>
                            </tr>
                            <tr>
                                <td>K&lt;x</td>
                                <td>Opponent back checkers &lt; x</td>
                            </tr>
                            <tr>
                                <td>Kx,y</td>
                                <td>Opponent back checkers between x and y</td>
                            </tr>
                            <tr>
                                <td>z>x</td>
                                <td>Player checkers in zone &gt; x</td>
                            </tr>
                            <tr>
                                <td>z&lt;x</td>
                                <td>Player checkers in zone &lt; x</td>
                            </tr>
                            <tr>
                                <td>zx,y</td>
                                <td>Player checkers in zone between x and y</td>
                            </tr>
                            <tr>
                                <td>Z>x</td>
                                <td>Opponent checkers in zone &gt; x</td>
                            </tr>
                            <tr>
                                <td>Z&lt;x</td>
                                <td>Opponent checkers in zone &lt; x</td>
                            </tr>
                            <tr>
                                <td>Zx,y</td>
                                <td>Opponent checkers in zone between x and y</td>
                            </tr>
                            <tr>
                                <td>bo>x</td>
                                <td>Player outfield blot &gt; x</td>
                            </tr>
                            <tr>
                                <td>bo&lt;x</td>
                                <td>Player outfield blot &lt; x</td>
                            </tr>
                            <tr>
                                <td>box,y</td>
                                <td>Player outfield blot between x and y</td>
                            </tr>
                            <tr>
                                <td>BO>x</td>
                                <td>Opponent outfield blot &gt; x</td>
                            </tr>
                            <tr>
                                <td>BO&lt;x</td>
                                <td>Opponent outfield blot &lt; x</td>
                            </tr>
                            <tr>
                                <td>BOx,y</td>
                                <td>Opponent outfield blot between x and y</td>
                            </tr>
                            <tr>
                                <td>bj&gt;x</td>
                                <td>Player Jan blot &gt; x</td>
                            </tr>
                            <tr>
                                <td>bj&lt;x</td>
                                <td>Player Jan blot &lt; x</td>
                            </tr>
                            <tr>
                                <td>bjx,y</td>
                                <td>Player Jan blot between x and y</td>
                            </tr>
                            <tr>
                                <td>BJ&gt;x</td>
                                <td>Opponent Jan blot &gt; x</td>
                            </tr>
                            <tr>
                                <td>BJ&lt;x</td>
                                <td>Opponent Jan blot &lt; x</td>
                            </tr>
                            <tr>
                                <td>BJx,y</td>
                                <td>Opponent Jan blot between x and y</td>
                            </tr>

                            <tr>
                                <td>t"word1;word2;..."</td>
                                <td>Search text</td>
                            </tr>
                            <tr>
                                <td>m"pattern1;pattern2;..."</td>
                                <td>Best moves containing at least one of the given patterns</td>
                            </tr>
                            <tr>
                                <td>m"ND;DT;DP;..."</td>
                                <td>Best cube decision of No Double/Take, Double/Take, Double/Pass</td>
                            </tr>
                            <tr>
                                <td>T&gt;x</td>
                                <td>Creation date &gt; x (year/month/day)</td>
                            </tr>
                            <tr>
                                <td>T&lt;x</td>
                                <td>Creation date &lt; x (year/month/day)</td>
                            </tr>
                            <tr>
                                <td>Tx,y</td>
                                <td>Creation date between x and y</td>
                            </tr>
                        </tbody>
                    </table>
                {/if}

                {#if activeTab === 'about'}
                    
                    <h3>Version</h3>
                    <p>Application version: {applicationVersion}</p>
                    <p>Database version: {databaseVersion}</p>
                    
                    <h3>Author</h3>
                    <p><strong>Kévin Unger &lt;blunderdb@proton.me&gt;</strong></p>
                    <p>You can also find me on Heroes under the nickname <strong>postmanpat</strong>.</p>
                    <p>I developed blunderDB initially for my personal use to detect patterns in my mistakes. But it is very pleasant to have feedback, especially when a lot of hours have been spent on design, coding, debugging... So feel free to write to me to share your feedback.</p>
                    <p>Here are several ways to reach out:</p>
                    <ul>
                        <li>Discuss with me if we meet in a tournament,</li>
                        <li>Send me an email,</li>
                    </ul>
                    <h3>License</h3>
                    <p>blunderDB is licensed under the MIT License. This means you are free to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the software, provided that the original copyright notice and this permission notice are included in all copies or substantial portions of the software.</p>
                    <h3>Acknowledgements</h3>
                    <p>I dedicate this small software to my partner <strong>Anne-Claire</strong> and our dear daughter <strong>Perrine</strong>. I would like to especially thank some friends:</p>
                    <ul>
                        <li><strong>Tristan Remille</strong>, for introducing me to backgammon with joy and kindness; for showing the Way in understanding this wonderful game; for continuing to encourage me with kindness and patience in the face of my poor attempts to play better.</li>
                        <li><strong>Nicolas Harmand</strong>, a joyful companion for over a decade in great adventures, and a fantastic game partner since he caught the backgammon bug.</li>
                    </ul>
                    <p>The Kazaross-XG2 Match Equity Table (MET) is credited to <strong>Neil Kazaross</strong>.</p>
                    <p>The take point and gammon values tables are taken from <strong>Dirk Schiemann</strong>'s book <em>The Theory of Backgammon</em>.</p>
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

    .tab-content p, .tab-content ul, .tab-content h2, .tab-content h3 {
        margin: 0 20px 20px 20px; /* Add bottom margin for spacing */
        text-align: justify;
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

