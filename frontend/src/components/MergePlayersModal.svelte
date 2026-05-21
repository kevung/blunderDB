<script>
    import { logger } from '../utils/logger.js';
    import { SvelteSet } from 'svelte/reactivity';
    import { GetAllPlayerNames, MergePlayers } from '../../wailsjs/go/main/Database.js';
    import { statusBarTextStore } from '../stores/uiStore';

    // Props
    let { onClose, onMerged } = $props();

    // All player names with their match counts
    let allPlayers = $state([]);
    // Selected names to be merged
    let selectedNames = new SvelteSet();
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
            error = 'Failed to load player names.';
        } finally {
            loading = false;
        }
    }

    loadPlayers();

    function toggleSelect(name) {
        const next = new SvelteSet(selectedNames);
        if (next.has(name)) {
            next.delete(name);
        } else {
            next.add(name);
        }
        selectedNames = next;
        // Auto-populate the canonical name with the first selected name if it is empty
        if (canonicalName === '' && next.size > 0) {
            canonicalName = [...next][0] ?? '';
        }
    }

    function useAsCanonical(name) {
        canonicalName = name ?? '';
    }

    async function doMerge() {
        const namesToMerge = [...selectedNames];
        const target = canonicalName.trim();
        if (namesToMerge.length < 2) {
            error = 'Select at least 2 names to merge.';
            return;
        }
        if (!target) {
            error = 'Enter a canonical name.';
            return;
        }
        saving = true;
        error = '';
        try {
            await MergePlayers(namesToMerge, target);
            statusBarTextStore.set(`Merged ${namesToMerge.length} names → "${target}"`);
            onMerged();
            onClose();
        } catch (e) {
            logger.error('MergePlayersModal: merge failed', e);
            error = String(e);
        } finally {
            saving = false;
        }
    }

    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            event.stopPropagation();
            event.preventDefault();
            onClose();
        }
    }
</script>

<div class="modal-backdrop" role="dialog" aria-modal="true" aria-label="Merge players" onkeydown={handleKeyDown}>
    <div class="modal-box">
        <div class="modal-header">
            <span class="modal-title">Merge player names</span>
            <button class="close-btn" onclick={onClose} title="Close">×</button>
        </div>

        <div class="modal-body">
            <p class="hint">Select the names that represent the same person, then choose or type the canonical name to keep. All matches will be updated.</p>

            <!-- Filter input -->
            <input
                class="filter-input"
                type="text"
                placeholder="Filter players…"
                bind:value={filterText}
                onkeydown={(e) => {
                    if (e.key === 'Escape') {
                        e.stopPropagation();
                        if (filterText) {
                            filterText = '';
                        } else {
                            onClose();
                        }
                    }
                }}
            />

            <!-- Player list -->
            {#if loading}
                <div class="state-msg">Loading…</div>
            {:else if allPlayers.length === 0}
                <div class="state-msg">No players found.</div>
            {:else}
                <div class="player-list">
                    {#each filteredPlayers as p (p.Name)}
                        <!-- svelte-ignore a11y_click_events_have_key_events -->
                        <!-- svelte-ignore a11y_no_static_element_interactions -->
                        <div class="player-row" class:selected={selectedNames.has(p.Name)} onclick={() => toggleSelect(p.Name)}>
                            <input type="checkbox" checked={selectedNames.has(p.Name)} tabindex="-1" onclick={(e) => e.stopPropagation()} onchange={() => toggleSelect(p.Name)} />
                            <span class="player-name">{p.Name}</span>
                            <span class="player-count" title="Number of matches">{p.Count}</span>
                            <button
                                class="use-btn"
                                title="Use as canonical name"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    useAsCanonical(p.Name);
                                }}>✓ use</button
                            >
                        </div>
                    {/each}
                </div>
            {/if}

            <!-- Canonical name -->
            <div class="canonical-row">
                <label class="canonical-label" for="canonical-input">Canonical name:</label>
                <input
                    id="canonical-input"
                    class="canonical-input"
                    type="text"
                    bind:value={canonicalName}
                    placeholder="Name to keep"
                    onkeydown={(e) => {
                        if (e.key === 'Enter') doMerge();
                        if (e.key === 'Escape') {
                            e.stopPropagation();
                            onClose();
                        }
                    }}
                />
            </div>

            {#if error}
                <div class="error-msg">{error}</div>
            {/if}

            <!-- Summary of selection -->
            {#if selectedNames.size > 0}
                <div class="selection-summary">
                    {selectedNames.size} name{selectedNames.size > 1 ? 's' : ''} selected:
                    {[...selectedNames].join(', ')}
                </div>
            {/if}
        </div>

        <div class="modal-footer">
            <button class="btn-cancel" onclick={onClose} disabled={saving}>Cancel</button>
            <button class="btn-merge" onclick={doMerge} disabled={saving || selectedNames.size < 2 || !canonicalName.trim()}>
                {saving ? 'Merging…' : `Merge ${selectedNames.size > 0 ? selectedNames.size : ''} names`}
            </button>
        </div>
    </div>
</div>

<style>
    .modal-backdrop {
        position: fixed;
        inset: 0;
        background: rgba(0, 0, 0, 0.45);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    }

    .modal-box {
        background: #fff;
        border-radius: 6px;
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.22);
        width: 480px;
        max-width: 95vw;
        max-height: 80vh;
        display: flex;
        flex-direction: column;
        overflow: hidden;
    }

    .modal-header {
        display: flex;
        align-items: center;
        padding: 10px 14px;
        border-bottom: 1px solid #e0e0e0;
        background: #fafafa;
        flex-shrink: 0;
    }

    .modal-title {
        font-size: 13px;
        font-weight: 600;
        color: #222;
        flex: 1;
    }

    .close-btn {
        background: none;
        border: none;
        font-size: 18px;
        color: #888;
        cursor: pointer;
        line-height: 1;
        padding: 0 2px;
    }

    .close-btn:hover {
        color: #333;
    }

    .modal-body {
        padding: 12px 14px;
        overflow-y: auto;
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 8px;
    }

    .hint {
        font-size: 11px;
        color: #666;
        margin: 0;
        line-height: 1.5;
    }

    .filter-input {
        width: 100%;
        padding: 4px 8px;
        font-size: 12px;
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
        font-size: 12px;
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
        font-size: 10px;
        color: #999;
        white-space: nowrap;
    }

    .use-btn {
        background: none;
        border: 1px solid #ccc;
        border-radius: 3px;
        font-size: 10px;
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
        font-size: 12px;
    }

    .canonical-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .canonical-label {
        font-size: 12px;
        color: #444;
        white-space: nowrap;
        flex-shrink: 0;
    }

    .canonical-input {
        flex: 1;
        padding: 4px 8px;
        font-size: 12px;
        border: 1px solid #1976d2;
        border-radius: 3px;
        outline: none;
        box-sizing: border-box;
    }

    .error-msg {
        font-size: 11px;
        color: #c62828;
        background: #ffebee;
        padding: 4px 8px;
        border-radius: 3px;
    }

    .selection-summary {
        font-size: 11px;
        color: #555;
        font-style: italic;
        word-break: break-word;
    }

    .modal-footer {
        display: flex;
        justify-content: flex-end;
        gap: 8px;
        padding: 10px 14px;
        border-top: 1px solid #e0e0e0;
        background: #fafafa;
        flex-shrink: 0;
    }

    .btn-cancel,
    .btn-merge {
        font-size: 12px;
        padding: 5px 14px;
        border-radius: 3px;
        cursor: pointer;
        border: 1px solid transparent;
    }

    .btn-cancel {
        background: #fff;
        border-color: #ccc;
        color: #444;
    }

    .btn-cancel:hover:not(:disabled) {
        background: #f5f5f5;
    }

    .btn-merge {
        background: #1976d2;
        color: #fff;
        border-color: #1565c0;
        font-weight: 600;
    }

    .btn-merge:hover:not(:disabled) {
        background: #1565c0;
    }

    .btn-merge:disabled,
    .btn-cancel:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>
