<script>
    import { logger } from '../utils/logger.js';
    import Modal from './Modal.svelte';
    import CountLink from './panels/CountLink.svelte';
    import { tick } from 'svelte';
    import { SvelteSet } from 'svelte/reactivity';
    import { get } from 'svelte/store';
    import { GetAllPlayerNames, MergePlayers, ListAliases, RemoveAlias, MainPlayer, PlanMainPlayer, SetMainPlayer, ClearMainPlayer, PlayerMatchIDs } from '../../wailsjs/go/database/Database.js';
    import { statusBarTextStore } from '../stores/uiStore';
    import { openMatchListInPanel } from '../services/positionLoader.js';
    import { t, tMsg } from '../i18n';

    // Props
    let { onClose, onMerged } = $props();

    // All player names (canonical) with their match counts
    let allPlayers = $state(/** @type {any[]} */ ([]));
    // Canonical name → the other spellings merged into it.
    let aliasesOf = $state(/** @type {Record<string, string[]>} */ ({}));
    // The library's Main player, '' when none.
    let mainPlayer = $state('');
    // A designation waiting for confirmation: the name and what it swaps.
    let pending = $state(/** @type {{ name: string, swap: number[], refused: number[] } | null} */ (null));
    // Matches the Main player still has at seat 2: a merge that brings them a
    // new spelling, or names them under another one, swaps nothing by itself.
    let mainLeft = $state(0);
    // The star shown full: the name being confirmed, else the Main player.
    let shownMain = $derived(pending ? pending.name : mainPlayer);
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
    // Short feedback on the last merge, "A, B → C"; the window stays open for the next one.
    let lastMerge = $state('');
    /** @type {HTMLInputElement | undefined} */
    let filterInput = $state();

    // Filtered view of the player list: a name matches by itself or by one of its aliases.
    let filteredPlayers = $derived.by(() => {
        const q = filterText.trim().toLowerCase();
        const shown = q ? allPlayers.filter((p) => p.Name.toLowerCase().includes(q) || (aliasesOf[p.Name] ?? []).some((a) => a.toLowerCase().includes(q))) : allPlayers;
        // The Main player heads the list.
        const i = shown.findIndex((p) => p.Name === mainPlayer);
        return i > 0 ? [shown[i], ...shown.slice(0, i), ...shown.slice(i + 1)] : shown;
    });

    // Load player list on mount
    async function loadPlayers(silent = false) {
        if (!silent) loading = true;
        error = '';
        try {
            const [players, aliases, main] = await Promise.all([GetAllPlayerNames(), ListAliases('player'), MainPlayer()]);
            allPlayers = players || [];
            /** @type {Record<string, string[]>} */
            const by = {};
            for (const a of aliases || []) (by[a.canonical] ??= []).push(a.alias);
            aliasesOf = by;
            mainPlayer = main || '';
        } catch (e) {
            logger.error('ManagePlayersModal: failed to load players', e);
            error = get(t)('merge.errorLoad');
        } finally {
            loading = false;
        }
        try {
            await reviewMain();
        } catch (e) {
            logger.error('ManagePlayersModal: plan main player failed', e);
            error = String(e);
        }
    }

    loadPlayers();

    // What the Main player's designation would still swap; when something is
    // left, the confirmation band comes back.
    async function reviewMain() {
        mainLeft = 0;
        if (!mainPlayer) return;
        const plan = await PlanMainPlayer(mainPlayer);
        mainLeft = (plan.swap ?? []).length;
        if (mainLeft > 0) pending = { name: plan.name || mainPlayer, swap: plan.swap ?? [], refused: plan.refused ?? [] };
        else if (pending?.name === mainPlayer) pending = null;
    }

    async function clearMain() {
        saving = true;
        error = '';
        try {
            await ClearMainPlayer();
            mainPlayer = '';
            mainLeft = 0;
            pending = null;
            statusBarTextStore.set(tMsg('merge.mainCleared'));
        } catch (e) {
            logger.error('ManagePlayersModal: clear main player failed', e);
            error = String(e);
        } finally {
            saving = false;
        }
    }

    function toggleSelect(/** @type {string} */ name) {
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

    function useAsCanonical(/** @type {string} */ name) {
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
            lastMerge = `${namesToMerge.join(', ')} → ${target}`;
            selectedNames.clear();
            canonicalName = '';
            filterText = '';
            await onMerged();
            await loadPlayers(true);
            await tick();
            filterInput?.focus();
        } catch (e) {
            logger.error('ManagePlayersModal: merge failed', e);
            error = String(e);
        } finally {
            saving = false;
        }
    }

    // Undoing a merge: the alias goes, its matches read under their own name again.
    async function removeAlias(/** @type {string} */ alias) {
        saving = true;
        error = '';
        try {
            await RemoveAlias('player', alias);
            statusBarTextStore.set(tMsg('merge.aliasRemoved', { alias }));
            await onMerged();
            await loadPlayers(true);
        } catch (e) {
            logger.error('ManagePlayersModal: remove alias failed', e);
            error = String(e);
        } finally {
            saving = false;
        }
    }

    // The star: on the Main player with nothing left to swap it forgets them
    // (nothing is swapped back); with matches left, or on anyone else, it shows
    // what the designation swaps, to be confirmed.
    async function toggleMain(/** @type {string} */ name) {
        error = '';
        if (name === mainPlayer && mainLeft === 0) {
            await clearMain();
            return;
        }
        try {
            const plan = await PlanMainPlayer(name);
            pending = { name: plan.name || name, swap: plan.swap ?? [], refused: plan.refused ?? [] };
        } catch (e) {
            logger.error('ManagePlayersModal: plan main player failed', e);
            error = String(e);
        }
    }

    async function confirmMain() {
        if (!pending) return;
        saving = true;
        error = '';
        try {
            const plan = await SetMainPlayer(pending.name);
            mainPlayer = plan.name;
            pending = null;
            mainLeft = 0;
            statusBarTextStore.set(tMsg('merge.mainSet', { name: plan.name, n: (plan.swap ?? []).length }));
            await onMerged();
            await loadPlayers(true);
        } catch (e) {
            logger.error('ManagePlayersModal: set main player failed', e);
            error = String(e);
        } finally {
            saving = false;
        }
    }

    function openMatches(/** @type {number[]} */ ids) {
        openMatchListInPanel(ids);
        onClose();
    }

    async function openPlayerMatches(/** @type {string} */ name) {
        try {
            openMatches((await PlayerMatchIDs(name)) ?? []);
        } catch (e) {
            logger.error('ManagePlayersModal: player matches failed', e);
            error = String(e);
        }
    }
</script>

<Modal open onclose={onClose} size="medium">
    {#snippet title()}{$t('merge.title')}{/snippet}

    <!-- Filter input. Escape here clears the filter first; only an empty one lets
         it through to the dialog, which then closes. -->
    <input
        class="filter-input"
        type="text"
        bind:this={filterInput}
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
                    <button
                        type="button"
                        class="main-btn"
                        class:main={p.Name === shownMain}
                        data-testid="main-player-star"
                        aria-pressed={p.Name === shownMain}
                        title={p.Name !== mainPlayer ? $t('merge.setMain') : mainLeft > 0 ? $t('merge.reapplyMain') : $t('merge.unsetMain')}
                        disabled={saving}
                        onclick={(e) => {
                            e.stopPropagation();
                            toggleMain(p.Name);
                        }}>{p.Name === shownMain ? '★' : '☆'}</button
                    >
                    <input type="checkbox" checked={selectedNames.has(p.Name)} tabindex="-1" onclick={(e) => e.stopPropagation()} onchange={() => toggleSelect(p.Name)} />
                    <span class="player-name">{p.Name}</span>
                    {#each aliasesOf[p.Name] ?? [] as alias (alias)}
                        <button
                            type="button"
                            class="alias-chip"
                            data-testid="alias-chip"
                            title={$t('merge.removeAlias')}
                            disabled={saving}
                            onclick={(e) => {
                                e.stopPropagation();
                                removeAlias(alias);
                            }}>{alias} ×</button
                        >
                    {/each}
                    <span class="player-count" role="presentation" onclick={(e) => e.stopPropagation()}>
                        <CountLink label={p.Count} title={$t('merge.matchCountTitle')} testid="player-match-count" onclick={() => openPlayerMatches(p.Name)} />
                    </span>
                    <button
                        type="button"
                        class="use-btn"
                        data-testid="keep-name"
                        title={$t('merge.useAsCanonical')}
                        aria-label={$t('merge.useAsCanonical')}
                        onclick={(e) => {
                            e.stopPropagation();
                            useAsCanonical(p.Name);
                        }}>✓</button
                    >
                </div>
            {/each}
        </div>
    {/if}

    {#if pending}
        <div class="main-confirm" role="status" data-testid="main-player-confirm">
            <span class="main-confirm-name">★ {pending.name}</span>
            <CountLink label={pending.swap.length} title={$t('merge.toSwapTitle')} testid="main-player-swap-count" onclick={() => pending && openMatches(pending.swap)} />
            <span>{$t('merge.toSwap')}</span>
            {#if pending.refused.length > 0}
                <span class="main-confirm-refused" title={$t('merge.refusedTitle')}>
                    · <CountLink label={pending.refused.length} title={$t('merge.refusedTitle')} testid="main-player-refused-count" onclick={() => pending && openMatches(pending.refused)} />
                    {$t('merge.refused')}
                </span>
            {/if}
            <span class="main-confirm-actions">
                {#if pending.name === mainPlayer}
                    <button type="button" data-testid="main-player-clear" onclick={clearMain} disabled={saving}>{$t('merge.clearMain')}</button>
                {/if}
                <button type="button" data-testid="main-player-cancel" onclick={() => (pending = null)} disabled={saving}>{$t('common.cancel')}</button>
                <button type="button" class="primary" data-testid="main-player-apply" onclick={confirmMain} disabled={saving}>{$t('merge.applyMain')}</button>
            </span>
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
    {:else if lastMerge}
        <div class="merge-done" role="status">{lastMerge}</div>
    {/if}

    <!-- Summary of selection -->
    {#if selectedNames.size > 0}
        <div class="selection-summary">
            {selectedNames.size === 1 ? $t('merge.selectedSummary', { n: selectedNames.size }) : $t('merge.selectedSummaryPlural', { n: selectedNames.size })}
            {[...selectedNames].join(', ')}
        </div>
    {/if}

    {#snippet footer()}
        <button class="btn-close" onclick={onClose} disabled={saving}>{$t('common.close')}</button>
        <button class="btn-merge primary" onclick={doMerge} disabled={saving || selectedNames.size < 2 || !canonicalName.trim()}>
            {saving ? $t('merge.merging') : $t('merge.mergeButton', { n: selectedNames.size > 0 ? selectedNames.size : '' })}
        </button>
    {/snippet}
</Modal>

<style>
    .filter-input {
        width: 100%;
        padding: 4px 8px;
        font-size: var(--font-size-base);
        border: 1px solid var(--color-border);
        border-radius: 3px;
        box-sizing: border-box;
        outline: none;
    }

    .filter-input:focus {
        border-color: var(--color-primary);
    }

    .player-list {
        border: 1px solid var(--color-border);
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
        border-bottom: 1px solid var(--color-border);
        transition: background-color 0.1s;
    }

    .player-row:last-child {
        border-bottom: none;
    }

    .player-row:hover {
        background: var(--color-surface-alt);
    }

    .player-row.selected {
        background: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface));
    }

    .player-row.selected:hover {
        background: color-mix(in srgb, var(--color-primary) 28%, var(--color-surface));
    }

    .player-name {
        flex: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .player-count {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        white-space: nowrap;
    }

    .main-btn {
        background: none;
        border: none;
        padding: 0 2px;
        cursor: pointer;
        color: var(--color-text-muted);
        font-size: var(--font-size-base);
        flex-shrink: 0;
    }

    .main-btn.main {
        color: var(--color-primary);
    }

    .main-btn:hover:not(:disabled) {
        color: var(--color-primary);
    }

    .alias-chip {
        background: var(--color-surface-alt);
        border: 1px solid var(--color-border);
        border-radius: 3px;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        cursor: pointer;
        padding: 0 4px;
        white-space: nowrap;
        flex-shrink: 0;
    }

    .alias-chip:hover:not(:disabled) {
        border-color: var(--color-danger);
        color: var(--color-danger);
    }

    .main-confirm {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--space-2);
        font-size: var(--font-size-small);
        padding: 4px 8px;
        border-radius: 3px;
        background: var(--color-surface-alt);
    }

    .main-confirm-name {
        font-weight: 600;
    }

    .main-confirm-refused {
        color: var(--color-text-muted);
    }

    .main-confirm-actions {
        margin-left: auto;
        display: flex;
        gap: 6px;
    }

    .use-btn {
        background: none;
        border: 1px solid var(--color-border);
        border-radius: 3px;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
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
        color: var(--color-text-muted);
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
        color: var(--color-text);
        white-space: nowrap;
        flex-shrink: 0;
    }

    .canonical-input {
        flex: 1;
        padding: 4px 8px;
        font-size: var(--font-size-base);
        border: 1px solid var(--color-primary);
        border-radius: 3px;
        outline: none;
        box-sizing: border-box;
    }

    .error-msg {
        font-size: var(--font-size-small);
        color: var(--color-danger);
        background: #ffebee;
        padding: 4px 8px;
        border-radius: 3px;
    }

    .merge-done {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        padding: 4px 8px;
        border-radius: 3px;
        background: var(--color-surface-alt);
        word-break: break-word;
    }

    .selection-summary {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        font-style: italic;
        word-break: break-word;
    }
</style>
