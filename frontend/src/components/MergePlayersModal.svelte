<script>
    import { logger } from '../utils/logger.js';
    import Modal from './Modal.svelte';
    import { SvelteSet } from 'svelte/reactivity';
    import { get } from 'svelte/store';
    import { GetAllPlayerNames, MergePlayers } from '../../wailsjs/go/database/Database.js';
    import { statusBarTextStore } from '../stores/uiStore';
    import { t, tMsg } from '../i18n';

    // Props
    let { onClose, onMerged } = $props();

    // All player names with their match counts
    let allPlayers = $state([]);
    // Selected names to be merged. A SvelteSet is mutated in place: the
    // template tracks *this* instance, a re-assigned copy would leave it
    // watching the old one.
    const selectedNames = new SvelteSet();
    // Canonical name the user wants to keep
    let canonicalName = $state('');
    // Filter text for the player list
    let filterText = $state('');
    // Loading / saving flags
    let loading = $state(true);
    let saving = $state(false);
    let error = $state('');

    // Filtered view of the player list
    let filteredPlayers = $derived.by(() => {
        const q = filterText.trim().toLowerCase();
        if (!q) return allPlayers;
        return allPlayers.filter((p) => p.Name.toLowerCase().includes(q));
    });

    // Load player list on mount
    async function loadPlayers() {
        loading = true;
        error = '';
        try {
            const result = await GetAllPlayerNames();
            allPlayers = result || [];
        } catch (e) {
            logger.error('MergePlayersModal: failed to load players', e);
            error = get(t)('merge.errorLoad');
        } finally {
            loading = false;
        }
    }

    loadPlayers();

    function toggleSelect(name) {
        if (selectedNames.has(name)) {
            selectedNames.delete(name);
        } else {
            selectedNames.add(name);
        }
        // Auto-populate the canonical name with the first selected name if it is empty
        if (canonicalName === '' && selectedNames.size > 0) {
            canonicalName = [...selectedNames][0] ?? '';
        }
    }

    function useAsCanonical(name) {
        canonicalName = name ?? '';
    }

    async function doMerge() {
        const namesToMerge = [...selectedNames];
        const target = canonicalName.trim();
        if (namesToMerge.length < 2) {
            error = get(t)('merge.errorSelect');
            return;
        }
        if (!target) {
            error = get(t)('merge.errorCanonical');
            return;
        }
        saving = true;
        error = '';
        try {
            await MergePlayers(namesToMerge, target);
            statusBarTextStore.set(tMsg('merge.merged', { n: namesToMerge.length, target }));
            onMerged();
            onClose();
        } catch (e) {
            logger.error('MergePlayersModal: merge failed', e);
            error = String(e);
        } finally {
            saving = false;
        }
    }
</script>

<Modal open onclose={onClose} size="medium">
    {#snippet title()}{$t('merge.title')}{/snippet}
    <p class="hint">{$t('merge.hint')}</p>

    <!-- Filter input. Escape here clears the filter first; only an empty one lets
         it through to the dialog, which then closes. -->
    <input
        class="filter-input"
        type="text"
        placeholder={$t('merge.filterPlaceholder')}
        bind:value={filterText}
        onkeydown={(e) => {
            if (e.key === 'Escape' && filterText) {
                e.stopPropagation();
                filterText = '';
            }
        }}
    />

    <!-- Player list -->
    {#if loading}
        <div class="state-msg">{$t('common.loading')}</div>
    {:else if allPlayers.length === 0}
        <div class="state-msg">{$t('merge.noPlayers')}</div>
    {:else}
        <div class="player-list">
            {#each filteredPlayers as p (p.Name)}
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div class="player-row" class:selected={selectedNames.has(p.Name)} onclick={() => toggleSelect(p.Name)}>
                    <input type="checkbox" checked={selectedNames.has(p.Name)} tabindex="-1" onclick={(e) => e.stopPropagation()} onchange={() => toggleSelect(p.Name)} />
                    <span class="player-name">{p.Name}</span>
                    <span class="player-count" title={$t('merge.matchCountTitle')}>{p.Count}</span>
                    <button
                        class="use-btn"
                        title={$t('merge.useAsCanonical')}
                        onclick={(e) => {
                            e.stopPropagation();
                            useAsCanonical(p.Name);
                        }}>✓ {$t('merge.use')}</button
                    >
                </div>
            {/each}
        </div>
    {/if}

    <!-- Canonical name -->
    <div class="canonical-row">
        <label class="canonical-label" for="canonical-input">{$t('merge.canonicalName')}</label>
        <input
            id="canonical-input"
            class="canonical-input"
            type="text"
            bind:value={canonicalName}
            placeholder={$t('merge.nameToKeepPlaceholder')}
            onkeydown={(e) => {
                if (e.key === 'Enter') doMerge();
            }}
        />
    </div>

    {#if error}
        <div class="error-msg">{error}</div>
    {/if}

    <!-- Summary of selection -->
    {#if selectedNames.size > 0}
        <div class="selection-summary">
            {selectedNames.size === 1 ? $t('merge.selectedSummary', { n: selectedNames.size }) : $t('merge.selectedSummaryPlural', { n: selectedNames.size })}
            {[...selectedNames].join(', ')}
        </div>
    {/if}

    {#snippet footer()}
        <button onclick={onClose} disabled={saving}>{$t('common.cancel')}</button>
        <button class="btn-merge primary" onclick={doMerge} disabled={saving || selectedNames.size < 2 || !canonicalName.trim()}>
            {saving ? $t('merge.merging') : $t('merge.mergeButton', { n: selectedNames.size > 0 ? selectedNames.size : '' })}
        </button>
    {/snippet}
</Modal>

<style>
    .hint {
        font-size: var(--font-size-small);
        color: #666;
        margin: 0;
        line-height: 1.5;
    }

    .filter-input {
        width: 100%;
        padding: 4px 8px;
        font-size: var(--font-size-base);
        border: 1px solid #ccc;
        border-radius: 3px;
        box-sizing: border-box;
        outline: none;
    }

    .filter-input:focus {
        border-color: #1976d2;
    }

    .player-list {
        border: 1px solid #e0e0e0;
        border-radius: 3px;
        overflow-y: auto;
        max-height: 240px;
        flex-shrink: 0;
    }

    .player-row {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 8px;
        cursor: pointer;
        font-size: var(--font-size-base);
        border-bottom: 1px solid #f0f0f0;
        transition: background-color 0.1s;
    }

    .player-row:last-child {
        border-bottom: none;
    }

    .player-row:hover {
        background: #f5f5f5;
    }

    .player-row.selected {
        background: #e3f2fd;
    }

    .player-row.selected:hover {
        background: #bbdefb;
    }

    .player-name {
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .player-count {
        font-size: var(--font-size-small);
        color: #999;
        white-space: nowrap;
    }

    .use-btn {
        background: none;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: var(--font-size-small);
        color: #555;
        cursor: pointer;
        padding: 1px 5px;
        white-space: nowrap;
        flex-shrink: 0;
    }

    .use-btn:hover {
        background: #e8f5e9;
        border-color: #4caf50;
        color: #2e7d32;
    }

    .state-msg {
        text-align: center;
        color: #999;
        padding: 16px;
        font-size: var(--font-size-base);
    }

    .canonical-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .canonical-label {
        font-size: var(--font-size-base);
        color: #444;
        white-space: nowrap;
        flex-shrink: 0;
    }

    .canonical-input {
        flex: 1;
        padding: 4px 8px;
        font-size: var(--font-size-base);
        border: 1px solid #1976d2;
        border-radius: 3px;
        outline: none;
        box-sizing: border-box;
    }

    .error-msg {
        font-size: var(--font-size-small);
        color: #c62828;
        background: #ffebee;
        padding: 4px 8px;
        border-radius: 3px;
    }

    .selection-summary {
        font-size: var(--font-size-small);
        color: #555;
        font-style: italic;
        word-break: break-word;
    }
</style>
