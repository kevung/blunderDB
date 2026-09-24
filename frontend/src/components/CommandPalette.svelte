<!--
  CommandPalette — one field that finds anything by approximate name (#287):
  a command of the command line, a tab, a saved filter, a match. Ctrl+Maj+P
  opens and closes it (keyboardService.js); Ctrl+K stays the Anki tab's.

  What it lists and what choosing does live in services/commandPalette.js, which
  reads the existing lists (commandVocabulary, tabCatalog, filterLibraryStore,
  GetAllMatches) — the palette owns none of its own.

  Keys: ↑/↓ (or Tab / Maj+Tab) move, Page↑/Page↓ jump, Entrée runs, Échap closes
  (escapeService: before any panel or the global dispatcher sees it).
-->
<script>
    import { tick, untrack } from 'svelte';
    import { commandPaletteOpenStore } from '../stores/uiStore.js';
    import { filterLibraryStore } from '../stores/filterLibraryStore.js';
    import { closeOnEscape } from '../services/escapeService.js';
    import { buildPaletteItems, rankPaletteItems, runPaletteItem, loadPaletteSources, closeCommandPalette } from '../services/commandPalette.js';
    import { highlightSegments } from '../utils/fuzzy.js';
    import { t } from '../i18n';

    const KIND_LABEL_KEYS = {
        command: 'palette.kindCommand',
        tab: 'palette.kindTab',
        filter: 'palette.kindFilter',
        match: 'palette.kindMatch'
    };
    const PAGE = 8;

    let query = $state('');
    let active = $state(0);
    let matches = $state([]);
    /** @type {HTMLInputElement | undefined} */
    let inputEl = $state();
    /** @type {HTMLElement | undefined} */
    let listEl = $state();
    /** @type {Element | null} */
    let returnFocus = null;

    const open = $derived($commandPaletteOpenStore);
    const items = $derived(buildPaletteItems({ translate: $t, matches, filters: $filterLibraryStore }));
    const results = $derived(rankPaletteItems(items, query));

    $effect(() => {
        if (open) return closeOnEscape(close);
    });

    $effect(() => {
        if (open) untrack(onOpen);
    });

    async function onOpen() {
        returnFocus = document.activeElement;
        query = '';
        active = 0;
        await tick();
        inputEl?.focus();
        matches = await loadPaletteSources();
    }

    function close() {
        closeCommandPalette();
        const back = returnFocus;
        returnFocus = null;
        if (back instanceof HTMLElement && back.isConnected && back !== document.body) back.focus();
    }

    /** @param {import('../services/commandPalette.js').PaletteItem} item */
    async function choose(item) {
        close();
        await tick();
        await runPaletteItem(item);
    }

    /** @param {number} index */
    async function moveTo(index) {
        if (results.length === 0) return;
        active = Math.max(0, Math.min(results.length - 1, index));
        await tick();
        listEl?.querySelector('[aria-selected="true"]')?.scrollIntoView?.({ block: 'nearest' });
    }

    /** @param {KeyboardEvent} event */
    function handleKeyDown(event) {
        let handled = true;
        if (event.key === 'ArrowDown' || (event.key === 'Tab' && !event.shiftKey)) {
            moveTo(active + 1 >= results.length ? 0 : active + 1);
        } else if (event.key === 'ArrowUp' || (event.key === 'Tab' && event.shiftKey)) {
            moveTo(active - 1 < 0 ? results.length - 1 : active - 1);
        } else if (event.key === 'PageDown') {
            moveTo(active + PAGE);
        } else if (event.key === 'PageUp') {
            moveTo(active - PAGE);
        } else if (event.key === 'Enter') {
            const chosen = results[active];
            if (chosen) choose(chosen.item);
        } else {
            handled = false;
        }
        if (handled) {
            event.preventDefault();
            event.stopPropagation();
        }
    }

    function onInput() {
        active = 0;
        if (listEl) listEl.scrollTop = 0;
    }
</script>

{#if open}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
        class="palette-backdrop"
        data-testid="command-palette"
        onmousedown={(e) => {
            if (e.target === e.currentTarget) close();
        }}
    >
        <div class="palette" role="dialog" aria-modal="true" aria-label={$t('palette.title')}>
            <input
                bind:this={inputEl}
                bind:value={query}
                oninput={onInput}
                onkeydown={handleKeyDown}
                class="palette-input"
                type="text"
                role="combobox"
                aria-expanded="true"
                aria-controls="command-palette-list"
                aria-activedescendant={results[active] ? `palette-option-${active}` : undefined}
                aria-autocomplete="list"
                placeholder={$t('palette.placeholder')}
                spellcheck="false"
                autocomplete="off"
            />
            <ul id="command-palette-list" class="palette-list" role="listbox" aria-label={$t('palette.title')} bind:this={listEl}>
                {#each results as { item, labelIndices }, i (item.id)}
                    <li
                        id="palette-option-{i}"
                        role="option"
                        aria-selected={i === active}
                        class:active={i === active}
                        onmousedown={(e) => {
                            e.preventDefault();
                            choose(item);
                        }}
                        onmousemove={() => (active = i)}
                    >
                        <span class="kind kind-{item.kind}">{$t(KIND_LABEL_KEYS[item.kind])}</span>
                        <span class="label"
                            >{#each highlightSegments(item.label, labelIndices) as seg, k (k)}{#if seg.hit}<mark>{seg.text}</mark>{:else}{seg.text}{/if}{/each}</span
                        >
                        {#if item.detail}<span class="detail">{item.detail}</span>{/if}
                    </li>
                {/each}
                {#if results.length === 0}
                    <li class="empty" role="presentation">{$t('palette.noResults')}</li>
                {/if}
            </ul>
            <div class="palette-hint">{$t('palette.hint')}</div>
        </div>
    </div>
{/if}

<style>
    .palette-backdrop {
        position: fixed;
        inset: 0;
        z-index: 1500;
        display: flex;
        justify-content: center;
        align-items: flex-start;
        padding-top: 12vh;
        background-color: rgba(0, 0, 0, 0.35);
    }

    .palette {
        display: flex;
        flex-direction: column;
        width: min(600px, calc(100vw - 32px));
        max-height: 70vh;
        background-color: var(--color-surface);
        color: var(--color-text);
        border: 1px solid var(--color-border);
        border-radius: var(--radius, 6px);
        box-shadow: 0 8px 28px rgba(0, 0, 0, 0.3);
        overflow: hidden;
        /* The app's main container centres its text; a list reads from the left. */
        text-align: left;
    }

    .palette-input {
        padding: var(--space-3, 10px) var(--space-3, 12px);
        border: none;
        border-bottom: 1px solid var(--color-border);
        background-color: var(--color-surface);
        color: var(--color-text);
        outline: none;
    }

    .palette-list {
        list-style: none;
        margin: 0;
        padding: var(--space-1, 4px) 0;
        overflow-y: auto;
        min-height: 0;
    }

    .palette-list li {
        display: flex;
        align-items: baseline;
        gap: var(--space-2, 8px);
        padding: var(--space-1, 4px) var(--space-3, 12px);
        cursor: pointer;
    }

    .palette-list li.active {
        background-color: color-mix(in srgb, var(--color-primary) 16%, var(--color-surface));
    }

    .kind {
        flex: none;
        min-width: 6em;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .label {
        flex: 1;
        min-width: 0;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .label mark {
        background: none;
        color: var(--color-primary);
        font-weight: 600;
    }

    .detail {
        flex: none;
        max-width: 45%;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .empty {
        color: var(--color-text-muted);
        cursor: default;
    }

    .palette-hint {
        padding: var(--space-1, 4px) var(--space-3, 12px);
        border-top: 1px solid var(--color-border);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }
</style>
