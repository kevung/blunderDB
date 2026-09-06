<script>
    import { onDestroy, tick } from 'svelte';
    import { statusBarTextStore, currentPositionIndexStore, commandTextStore, showCommandInputStore, dbMutationCounterStore, activeTabStore } from '../stores/uiStore';
    import { libraryCountsStore, refreshLibraryCounts } from '../stores/libraryCountsStore.js';
    import { databasePathStore } from '../stores/databaseStore';
    import { loadAllPositions } from '../services/positionService.js';
    import { watchImportNoticeStore } from '../stores/watchStore.js';
    import { showFileImportModalStore, fileImportModeStore } from '../stores/importModalStore.js';
    import { positionsStore, matchContextStore } from '../stores/positionStore';
    import { commandHistoryStore } from '../stores/commandHistoryStore';
    import { gammonNetBatchStore } from '../stores/gammonNetBatchStore';
    import { LoadCommandHistory, SaveCommand } from '../../wailsjs/go/database/Database.js';
    import { CancelGammonNetBatch } from '../../wailsjs/go/gui/App.js';
    import { EventsOn } from '../../wailsjs/runtime/runtime.js';
    import { t, tMsg, resolveStatusMessage } from '../i18n';
    import { getCommandSuggestions } from '../commandVocabulary.js';

    /** @type {function(string): void} */
    let { onCommand = (_cmd) => {} } = $props();

    let inputEl = $state();
    let showInput = $derived($showCommandInputStore);
    let commandHistory = $derived($commandHistoryStore);
    let historyIndex = -1;

    // MATCH-mode move/game counters. These used to be {@const} in the
    // template, recomputed with three O(n) passes (filter, slice+filter,
    // map+Math.max) on every navigated move of a match — 500 moves ×
    // however many re-renders a review pass triggers (D.8, #208). Now
    // computed once per movePositions change; the per-move counter is an O(1)
    // array lookup in the template.
    let checkerMoves = $derived($matchContextStore.movePositions.filter((p) => p.move_type === 'checker'));
    let checkerCountUpTo = $derived.by(() => {
        const moves = $matchContextStore.movePositions;
        const counts = new Array(moves.length);
        let running = 0;
        for (let i = 0; i < moves.length; i++) {
            if (moves[i].move_type === 'checker') running++;
            counts[i] = running;
        }
        return counts;
    });
    let lastGameNumber = $derived.by(() => {
        let max = 1;
        for (const p of $matchContextStore.movePositions) if (p.game_number > max) max = p.game_number;
        return max;
    });

    // --- Command autocompletion ------------------------------------------------
    // Suggestions for the typed command word. Tab / Shift-Tab cycle through them;
    // Escape dismisses the dropdown (a second Escape closes the command line).
    // ArrowUp/Down stay reserved for command history.
    let suggestionsDismissed = $state(false);
    let selectedSuggestion = $state(0);
    let suggestions = $derived(suggestionsDismissed ? [] : getCommandSuggestions($commandTextStore));

    $effect(() => {
        $commandTextStore; // track dependency so edits reset the dropdown
        suggestionsDismissed = false;
        selectedSuggestion = 0;
    });

    function applySuggestion(index) {
        const cmd = suggestions[index];
        if (!cmd) return;
        commandTextStore.set(cmd.name);
        requestAnimationFrame(() => {
            inputEl?.setSelectionRange(cmd.name.length, cmd.name.length);
            inputEl?.focus();
        });
    }

    function cycleSuggestion(step) {
        if (suggestions.length === 0) return;
        applySuggestion(selectedSuggestion);
        selectedSuggestion = (selectedSuggestion + step + suggestions.length) % suggestions.length;
    }

    // The status store may hold a plain string or a tMsg() descriptor
    // ({ i18nKey, i18nParams }). Resolving through $t here makes the displayed
    // message re-translate live whenever the language changes.
    let statusMessage = $derived(resolveStatusMessage($statusBarTextStore, $t));

    // Le compteur de bibliothèque (#287). Rafraîchi à l'ouverture d'une base et
    // après chaque mutation, jamais en boucle : trois COUNT ne coûtent rien une
    // fois et coûteraient tout à chaque frappe.
    $effect(() => {
        void $databasePathStore;
        void $dbMutationCounterStore;
        refreshLibraryCounts();
    });

    /**
     * Chaque nombre du compteur ouvre ce qu'il compte. « Blunders » passe par
     * la ligne de commande plutôt que par un chemin à part : le jeton `E>100`
     * est le même seuil que celui du compteur, et l'utilisateur le voit.
     * @param {'positions'|'blunders'|'matches'} what
     */
    async function showLibrary(what) {
        if (what === 'matches') {
            activeTabStore.set('matches');
            return;
        }
        if (what === 'blunders') {
            commandTextStore.set('s E>100');
            showCommandInputStore.set(true);
            await tick();
            inputEl?.focus();
            return;
        }
        await loadAllPositions();
    }

    $effect(() => {
        if ($showCommandInputStore) {
            loadHistory()
                .then(() => tick())
                .then(() => inputEl?.focus());
        }
    });

    async function loadHistory() {
        const history = await LoadCommandHistory();
        commandHistoryStore.set((history || []).reverse());
        historyIndex = -1;
    }

    // gammonNet batch (#129, ADR-0013): "a bounded, visible job — never a
    // resident background task". The status bar is always mounted, so it is
    // where the running total is noticed without a modal getting in the way;
    // cancelling here mirrors the config tab's bearoff download control.
    let gammonNetBatch = $derived($gammonNetBatchStore);

    const unsubGammonNetBatch = [
        EventsOn('gammonnet-batch:progress', (p) => gammonNetBatchStore.set(p)),
        // The evaluated/refused/failed split (#191) is the batch's own
        // end-of-run figure — refused is deliberate (a dance, a match score
        // beyond the MET) and never worth a warning; failed positions are
        // retried, unchanged, the next time this batch runs.
        EventsOn('gammonnet-batch:done', (summary) => {
            gammonNetBatchStore.set(null);
            statusBarTextStore.set(tMsg('eval.batchDone', summary ?? { evaluated: 0, refused: 0, failed: 0 }));
        }),
        EventsOn('gammonnet-batch:cancelled', () => gammonNetBatchStore.set(null)),
        EventsOn('gammonnet-batch:error', () => gammonNetBatchStore.set(null))
    ];
    onDestroy(() => unsubGammonNetBatch.forEach((off) => off && off()));

    function cancelGammonNetBatch() {
        CancelGammonNetBatch();
    }

    export function focusInput() {
        showCommandInputStore.set(true);
    }

    function hideInput() {
        showCommandInputStore.set(false);
        commandTextStore.set('');
        historyIndex = -1;
    }

    function handleKeyDown(event) {
        if (event.code === 'Tab') {
            // Tab / Shift-Tab cycle through autocompletion matches.
            event.stopPropagation();
            event.preventDefault();
            cycleSuggestion(event.shiftKey ? -1 : 1);
            return;
        }
        if (event.code === 'Escape' && suggestions.length > 0) {
            // Dismiss the dropdown first; a second Escape closes the command line.
            event.stopPropagation();
            event.preventDefault();
            suggestionsDismissed = true;
            return;
        }
        if (event.code === 'ArrowUp') {
            event.stopPropagation();
            event.preventDefault();
            if (historyIndex < commandHistory.length - 1) {
                historyIndex++;
                commandTextStore.set(commandHistory[historyIndex]);
                requestAnimationFrame(() => {
                    inputEl?.setSelectionRange(inputEl.value.length, inputEl.value.length);
                });
            }
        } else if (event.code === 'ArrowDown') {
            event.stopPropagation();
            event.preventDefault();
            if (historyIndex > 0) {
                historyIndex--;
                commandTextStore.set(commandHistory[historyIndex]);
                requestAnimationFrame(() => {
                    inputEl?.setSelectionRange(inputEl.value.length, inputEl.value.length);
                });
            } else {
                historyIndex = -1;
                commandTextStore.set('');
            }
        } else if (event.code === 'Escape') {
            event.stopPropagation();
            event.preventDefault();
            hideInput();
        } else if (event.code === 'Enter') {
            event.stopPropagation();
            event.preventDefault();
            const command = ($commandTextStore || '').trim();
            if (command) {
                commandHistoryStore.update((history) => {
                    history = history || [];
                    history.unshift(command);
                    return history;
                });
                historyIndex = -1;
                SaveCommand(command);
                onCommand(command);
            }
            hideInput();
        }
    }
</script>

<div class="status-bar" data-testid="status-bar" data-tour="statusbar">
    {#if showInput}
        <div class="command-input-row">
            {#if suggestions.length > 0}
                <ul class="command-suggestions" role="listbox">
                    {#each suggestions as cmd, i (cmd.name)}
                        <li
                            role="option"
                            aria-selected={i === selectedSuggestion}
                            class:selected={i === selectedSuggestion}
                            onmousedown={(e) => {
                                e.preventDefault();
                                applySuggestion(i);
                            }}
                        >
                            <span class="cmd-name">{cmd.name}</span>
                            {#if cmd.aliases.length > 0}
                                <span class="cmd-aliases">{cmd.aliases.join(', ')}</span>
                            {/if}
                        </li>
                    {/each}
                </ul>
            {/if}
            <span class="prompt-char">&gt;</span>
            <input type="text" bind:this={inputEl} bind:value={$commandTextStore} class="command-input" placeholder={$t('statusBar.typeCommand')} onkeydown={handleKeyDown} onblur={hideInput} />
        </div>
    {:else}
        <!-- Scoped to this span, not the whole status bar (#204): the status
             bar also holds the command-line suggestions listbox, the
             gammonNet batch chip and the position counter, none of which
             are status announcements — a live region on the whole bar had a
             screen reader re-read all of it on every one of those changes,
             not just an actual status message. -->
        <span class="info-message" role="status" aria-live="polite" data-testid="status-bar-message" title={statusMessage}>{statusMessage}</span>
    {/if}
    <!-- Le dossier surveillé (#258) annonce ses imports ici, jamais par une
         fenêtre : l'utilisateur étudiait une position quand ses matchs sont
         arrivés. Le bandeau ouvre le compte rendu si on le lui demande, et
         disparaît dès qu'on l'écarte. -->
    {#if $watchImportNoticeStore}
        <span class="watch-import-chip">
            {$t('status.watchImportNotice', {
                succeeded: $watchImportNoticeStore.succeeded,
                skipped: $watchImportNoticeStore.skipped,
                failed: $watchImportNoticeStore.failed
            })}
            <button
                type="button"
                class="watch-import-action"
                onclick={() => {
                    fileImportModeStore.set('completed');
                    showFileImportModalStore.set(true);
                    watchImportNoticeStore.set(null);
                }}>{$t('status.watchImportReport')}</button
            >
            <button type="button" class="watch-import-action" onclick={() => watchImportNoticeStore.set(null)}>{$t('common.close')}</button>
        </span>
    {/if}
    {#if gammonNetBatch}
        <span class="gammonnet-batch-chip" title={$t('eval.batchProgress', { done: gammonNetBatch.done, total: gammonNetBatch.total })}>
            {$t('eval.batchProgress', { done: gammonNetBatch.done, total: gammonNetBatch.total })}
            <button type="button" class="gammonnet-batch-cancel" onclick={cancelGammonNetBatch}>{$t('eval.batchCancel')}</button>
        </span>
    {/if}
    {#if $matchContextStore.isMatchMode && $matchContextStore.movePositions.length > 0}
        <span class="position-info">{$t('statusBar.move')} {checkerCountUpTo[$matchContextStore.currentIndex] ?? 0}/{checkerMoves.length}</span>
        <span class="position-info"
            >{$t('statusBar.game')}
            {$matchContextStore.movePositions[$matchContextStore.currentIndex]?.game_number || 1}/{lastGameNumber}</span
        >
    {:else}
        <span class="position-info">{$positionsStore.length > 0 ? $currentPositionIndexStore + 1 : 0} / {$positionsStore.length}</span>
    {/if}
    <!-- Le compteur de bibliothèque (#287) : ce que la base contient, et
         chaque nombre ouvre ce qu'il compte. Un chiffre qu'on ne peut pas
         suivre est une décoration. -->
    {#if $libraryCountsStore}
        <span class="library-counts">
            <button type="button" class="count-link" onclick={() => showLibrary('positions')} title={$t('statusBar.countPositionsTitle')}>
                {$t('statusBar.countPositions', { n: $libraryCountsStore.positions })}
            </button>
            <span class="count-sep">·</span>
            <button type="button" class="count-link" onclick={() => showLibrary('blunders')} title={$t('statusBar.countBlundersTitle')}>
                {$t('statusBar.countBlunders', { n: $libraryCountsStore.blunders })}
            </button>
            <span class="count-sep">·</span>
            <button type="button" class="count-link" onclick={() => showLibrary('matches')} title={$t('statusBar.countMatchesTitle')}>
                {$t('statusBar.countMatches', { n: $libraryCountsStore.matches })}
            </button>
        </span>
    {/if}
</div>

<style>
    .watch-import-chip {
        display: inline-flex;
        align-items: center;
        gap: 0.4em;
        margin-left: 0.8em;
        white-space: nowrap;
    }

    .watch-import-action {
        cursor: pointer;
    }

    .library-counts {
        display: inline-flex;
        align-items: center;
        gap: 0.25em;
        margin-left: 0.8em;
        white-space: nowrap;
        color: var(--color-text-muted);
    }

    /* Pas de `font: inherit` ici : style.css le pose déjà sur les contrôles
       de formulaire (ADR-0008), et le répéter est ce que la garde de type
       interdit. */
    .count-link {
        background: none;
        border: none;
        padding: 0;
        color: inherit;
        cursor: pointer;
        text-decoration: underline dotted;
    }

    .count-link:hover {
        color: var(--color-primary);
    }

    .count-sep {
        opacity: 0.6;
    }

    .status-bar {
        display: flex;
        align-items: center;
        background: #f7f7f7;
        border-top: 1px solid #e0e0e0;
        padding: 2px 0;
        flex-shrink: 0;
        width: 100%;
        font-size: var(--font-size-base);
        font-family: var(--font-family-ui);
        gap: 0;
        user-select: none;
        height: 22px;
    }

    .info-message {
        flex: 1;
        padding: 0 10px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: #555;
        font-size: var(--font-size-base);
        line-height: 22px;
    }

    .gammonnet-batch-chip {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 0 10px;
        flex-shrink: 0;
        font-variant-numeric: tabular-nums;
        color: #555;
        font-size: var(--font-size-small);
        border-left: 1px solid #e0e0e0;
        line-height: 22px;
        white-space: nowrap;
    }

    .gammonnet-batch-cancel {
        background: none;
        border: none;
        padding: 0;
        color: var(--color-primary);
        text-decoration: underline;
        cursor: pointer;
        font-size: var(--font-size-small);
    }

    .position-info {
        padding: 0 10px;
        flex-shrink: 0;
        font-variant-numeric: tabular-nums;
        color: #555;
        font-size: var(--font-size-base);
        border-left: 1px solid #e0e0e0;
        line-height: 22px;
    }

    .command-input-row {
        position: relative;
        flex: 1;
        display: flex;
        align-items: center;
        padding: 0 6px;
        min-width: 0;
    }

    /* The status bar sits at the bottom, so the dropdown opens upwards. */
    .command-suggestions {
        position: absolute;
        bottom: 100%;
        left: 6px;
        margin: 0 0 2px 0;
        padding: 0;
        list-style: none;
        min-width: 220px;
        max-height: 220px;
        overflow-y: auto;
        background-color: white;
        border: 1px solid rgba(0, 0, 0, 0.25);
        border-radius: 2px;
        box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.15);
        z-index: 1100;
    }

    .command-suggestions li {
        display: flex;
        justify-content: space-between;
        align-items: baseline;
        gap: 12px;
        padding: 4px 10px;
        font-size: var(--font-size-base);
        cursor: pointer;
    }

    .command-suggestions li.selected {
        background-color: #e8f0fe;
    }

    .command-suggestions li:hover {
        background-color: #f0f0f0;
    }

    .cmd-name {
        font-family: var(--font-family-mono);
        font-weight: 600;
        color: var(--color-text);
    }

    .cmd-aliases {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .prompt-char {
        color: var(--color-primary);
        font-weight: bold;
        margin-right: 4px;
        font-family: var(--font-family-mono);
        font-size: var(--font-size-base);
        line-height: 22px;
    }

    .command-input {
        flex: 1;
        background: transparent;
        border: none;
        outline: none;
        color: var(--color-text);
        font-family: var(--font-family-mono);
        font-size: var(--font-size-base);
        padding: 0;
        line-height: 22px;
        height: 22px;
    }

    .command-input::placeholder {
        color: #aaa;
    }
</style>
