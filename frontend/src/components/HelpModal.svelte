<!-- HelpModal.svelte -->
<script>
    import Modal from './Modal.svelte';
    import { logger } from '../utils/logger.js';
    import { isBareLetter } from '../utils/keys.js';
    import { onMount } from 'svelte';
    import { metaStore } from '../stores/metaStore'; // Import metaStore
    import { t, language } from '../i18n';
    import { help, loadHelpFor } from '../i18n/help/index.js';
    import { GetDatabaseVersion } from '../../wailsjs/go/database/Database'; // Correct import path

    let { visible = false, onClose } = $props();

    let activeTab = $state('manual'); // Default active tab
    const tabs = ['manual', 'shortcuts', 'commands', 'about'];
    let contentArea = $state();

    let databaseVersion = $state('');
    let applicationVersion = $derived($metaStore.applicationVersion);

    // appVersion/dbVersion are spliced into raw HTML below ({@html}); escape them even
    // though today's sources (metaStore, GetDatabaseVersion()) are trusted, so a future
    // caller can't turn this interpolation into an XSS hole without also touching this line.
    function escapeHtml(value) {
        return String(value).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);
    }

    let aboutHtml = $derived(($help.about || '').replace(/\{appVersion\}/g, escapeHtml(applicationVersion)).replace(/\{dbVersion\}/g, escapeHtml(databaseVersion)));

    // The active language's help bundle (~170 kB) is fetched only when the
    // modal is actually opened, not eagerly at startup (#207). Re-runs if the
    // language changes while the modal stays open; loadHelpFor() is a no-op
    // once a bundle is cached.
    $effect(() => {
        if (visible) loadHelpFor($language);
    });

    onMount(async () => {
        try {
            databaseVersion = await GetDatabaseVersion();
        } catch (error) {
            logger.error('Error fetching database version:', error);
        }
    });

    function switchTab(tab) {
        activeTab = tab;
    }

    // Every key pressed while the help is open belongs to it. Escape is Modal's; the
    // rest is handled here and stopped so the global dispatcher never sees it.
    function handleKeyDown(event) {
        event.preventDefault();
        event.stopImmediatePropagation();
        if (event.ctrlKey && event.code === 'KeyF') {
            onClose();
        } else if (!event.ctrlKey && event.key === '?') {
            onClose();
        } else if (!event.ctrlKey && event.key === 'ArrowRight') {
            navigateTabs(1); // Move to the next tab
        } else if (!event.ctrlKey && event.key === 'ArrowLeft') {
            navigateTabs(-1); // Move to the previous tab
        } else if (isBareLetter(event, 'l')) {
            navigateTabs(1); // Move to the next tab
        } else if (isBareLetter(event, 'h')) {
            navigateTabs(-1); // Move to the previous tab
        } else if (!event.ctrlKey && event.key === 'ArrowDown') {
            scrollContent(1); // Scroll down
        } else if (!event.ctrlKey && event.key === 'ArrowUp') {
            scrollContent(-1); // Scroll up
        } else if (isBareLetter(event, 'j')) {
            scrollContent(1); // Scroll down
        } else if (isBareLetter(event, 'k')) {
            scrollContent(-1); // Scroll up
        } else if (!event.ctrlKey && event.key === 'PageDown') {
            scrollContent('bottom'); // Go to the bottom of the page
        } else if (!event.ctrlKey && event.key === 'PageUp') {
            scrollContent('top'); // Go to the top of the page
        } else if (!event.ctrlKey && event.key === ' ') {
            // Space key
            scrollContent('page'); // Scroll down by the height of the content
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

            if (direction === 1) {
                // Arrow down
                contentArea.scrollTop += scrollAmount;
            } else if (direction === -1) {
                // Arrow up
                contentArea.scrollTop -= scrollAmount;
            } else if (direction === 'bottom') {
                // PageDown
                contentArea.scrollTop = contentArea.scrollHeight; // Go to bottom
            } else if (direction === 'top') {
                // PageUp
                contentArea.scrollTop = 0; // Go to top
            } else if (direction === 'page') {
                // Space key
                contentArea.scrollTop += contentArea.clientHeight; // Scroll down by the visible area height
            }
        }
    }
</script>

<Modal open={visible} onclose={onClose} size="panel" closeOnOverlay label="Help" onkeydown={handleKeyDown}>
    <!-- Tabs -->
    <div class="tab-header">
        <button class={activeTab === 'manual' ? 'active' : ''} onclick={() => switchTab('manual')}>{$t('help.tabManual')}</button>
        <button class={activeTab === 'shortcuts' ? 'active' : ''} onclick={() => switchTab('shortcuts')}>{$t('help.tabShortcuts')}</button>
        <button class={activeTab === 'commands' ? 'active' : ''} onclick={() => switchTab('commands')}>{$t('help.tabCommands')}</button>
        <button class={activeTab === 'about' ? 'active' : ''} onclick={() => switchTab('about')}>{$t('help.tabAbout')}</button>
    </div>

    <!-- Tab Content -->
    <div class="tab-content" bind:this={contentArea}>
        {#if activeTab === 'manual'}
            <!-- eslint-disable-next-line svelte/no-at-html-tags -- generated help corpus (cmd/help-gen escapes every string it renders; guarded by help.safety.test.js), no runtime interpolation -->
            {@html $help.manual}
        {/if}

        {#if activeTab === 'shortcuts'}
            <!-- eslint-disable-next-line svelte/no-at-html-tags -- generated help corpus (cmd/help-gen escapes every string it renders; guarded by help.safety.test.js), no runtime interpolation -->
            {@html $help.shortcuts}
        {/if}

        {#if activeTab === 'commands'}
            <!-- eslint-disable-next-line svelte/no-at-html-tags -- generated help corpus (cmd/help-gen escapes every string it renders; guarded by help.safety.test.js), no runtime interpolation -->
            {@html $help.commands}
        {/if}

        {#if activeTab === 'about'}
            <!-- eslint-disable-next-line svelte/no-at-html-tags -- the only interpolated placeholders (appVersion/dbVersion) are HTML-escaped in aboutHtml above -->
            {@html aboutHtml}
        {/if}
    </div>
</Modal>

<style>
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
        font-size: var(--font-size-title);
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

    /* Help tab content is injected via {@html}, so Svelte's scoped-CSS hash is
       not applied to those elements. Use :global() nested under .tab-content so
       the styling targets the injected HTML without leaking to the rest of the app.
       The vocabulary below is the whole of what cmd/help-gen emits (h3, p, ul/li,
       table, code, div.admonition, blockquote, pre.math) plus what the
       hand-written About fragment uses. */
    .tab-content :global(p),
    .tab-content :global(ul),
    .tab-content :global(h2),
    .tab-content :global(h3),
    .tab-content :global(h4),
    .tab-content :global(h5) {
        margin: 0 20px 20px 20px; /* Add bottom margin for spacing */
        text-align: justify;
    }

    /* The manual nests three deep. Hierarchy comes from weight and colour,
       not from a ladder of absolute sizes (ADR-0008): h3 keeps the tab's
       title size, and the two below it step down in emphasis only. */
    .tab-content :global(h4),
    .tab-content :global(h5) {
        font-size: var(--font-size-base);
        margin-top: var(--space-4);
    }

    .tab-content :global(h5) {
        font-weight: 500;
        color: var(--color-text-muted);
    }

    .tab-content :global(table) {
        margin: 0 auto;
        width: 80%;
        border-collapse: collapse;
    }

    .tab-content :global(th),
    .tab-content :global(td) {
        padding: 12px;
        text-align: center;
        border-bottom: 1px solid #ddd;
        width: 50%;
    }

    .tab-content :global(th) {
        background-color: #f4f4f4;
    }

    .tab-content :global(tr:hover) {
        background-color: #f1f1f1;
    }

    /* A term's indented definition, and the manual is full of them: the
       vertical rule is what tells it apart from the paragraph above. */
    .tab-content :global(blockquote) {
        margin: 0 20px 20px 40px;
        padding-left: var(--space-3);
        border-left: 2px solid var(--color-border);
    }

    /* The one .. math:: in the documentation, shown as it is written: the
       modal carries no formula renderer. */
    .tab-content :global(pre.math) {
        margin: 0 20px 20px 40px;
        padding: var(--space-2);
        overflow-x: auto;
        font-family: var(--font-family-mono);
        background-color: var(--color-surface-alt);
        border-radius: var(--radius);
    }

    /* A .. note:: / .. tip:: / .. warning:: / .. important:: / .. caution::
       carried over from the documentation. */
    .tab-content :global(.admonition) {
        margin: 0 20px 20px 20px;
        padding: var(--space-2) var(--space-3);
        border-left: 3px solid var(--color-border);
        background-color: var(--color-surface-alt);
        border-radius: 0 var(--radius) var(--radius) 0;
    }

    .tab-content :global(.admonition.warning),
    .tab-content :global(.admonition.caution),
    .tab-content :global(.admonition.important) {
        border-left-color: var(--color-danger);
    }

    .tab-content :global(.admonition p) {
        margin: 0 0 var(--space-2) 0;
    }

    .tab-content :global(.admonition p:last-child) {
        margin-bottom: 0;
    }

    /* ``literal`` from the .rst sources: command names, filter tokens, keys. */
    .tab-content :global(code) {
        font-family: var(--font-family-mono);
        background-color: var(--color-surface-alt);
        padding: 0 3px;
        border-radius: var(--radius);
    }
</style>
