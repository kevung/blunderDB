<script>
    import { onDestroy, tick } from 'svelte';
    import { statusBarTextStore, currentPositionIndexStore, commandTextStore, showCommandInputStore, dbMutationCounterStore, activeTabStore } from '../stores/uiStore';
    import { libraryCountsStore, refreshLibraryCounts } from '../stores/libraryCountsStore.js';
    import { databasePathStore } from '../stores/databaseStore';
    import { loadAllPositions } from '../services/positionService.js';
    import { watchImportNoticeStore } from '../stores/watchStore.js';
    import { transcriptionResumeStore, refreshTranscriptionResume, resumeTranscriptionAnalysis, dismissTranscriptionResume } from '../services/transcriptionSave.js';
    import { transcriptionPromptStore, transcriptionNoticeStore } from '../stores/transcriptionStore.js';
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

    // MATCH-mode counters, computed once per movePositions change, not per render.
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

    // Command autocompletion: Tab/Shift-Tab cycle, Escape dismisses (a second one
    // closes the line); ArrowUp/Down stay for history.
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

    // A string or a tMsg() descriptor, resolved through $t so it re-translates live.
    let statusMessage = $derived(resolveStatusMessage($statusBarTextStore, $t));

    // Compteur de bibliothèque : rafraîchi à l'ouverture et après chaque mutation, jamais en boucle.
    $effect(() => {
        void $databasePathStore;
        void $dbMutationCounterStore;
        refreshLibraryCounts();
    });

    // Reprise de l'analyse d'une transcription (ADR-0045 §8), demandée à
    // l'ouverture seulement, pour qu'une proposition écartée ne revienne pas aussitôt.
    $effect(() => {
        const path = $databasePathStore;
        if (!path) {
            // Aucune base ouverte : il n'y a personne à qui poser la question,
            // et la proposition d'une base précédente n'a plus d'objet.
            dismissTranscriptionResume();
            return;
        }
        refreshTranscriptionResume();
    });

    /**
     * Chaque nombre ouvre ce qu'il compte ; « Blunders » passe par la ligne de
     * commande (`E>100`, le même seuil), visible de l'utilisateur.
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

    // gammonNet batch (ADR-0013): a bounded, visible job, shown and cancelled
    // here since the status bar is always mounted.
    let gammonNetBatch = $derived($gammonNetBatchStore);

    const unsubGammonNetBatch = [
        EventsOn('gammonnet-batch:progress', (p) => gammonNetBatchStore.set(p)),
        // Refused is deliberate (dance, score beyond the MET), never a warning;
        // failed positions are retried on the next run.
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
        <!-- Live region on this span only: the rest of the bar is not status announcements. -->
        <span class="info-message" role="status" aria-live="polite" data-testid="status-bar-message" title={statusMessage}>{statusMessage}</span>
    {/if}
    <!-- Imports du dossier surveillé : un bandeau, jamais une fenêtre. -->
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
    <!-- L'Action attendue (ADR-0048 décision 2), remplacée 1,5 s par la réponse
         d'un geste sans effet (décision 9). -->
    {#if $transcriptionNoticeStore}
        <span class="transcription-notice" data-testid="transcription-notice">{$t($transcriptionNoticeStore.key, $transcriptionNoticeStore.params)}</span>
    {:else if $transcriptionPromptStore}
        <span class="transcription-prompt" data-testid="transcription-prompt">{$t($transcriptionPromptStore.key, $transcriptionPromptStore.params)}</span>
    {/if}
    <!-- Lot d'analyse interrompu (ADR-0045 §8), recompté à l'ouverture ; écarter n'écrit rien. -->
    {#if $transcriptionResumeStore}
        <span class="transcription-resume-chip">
            {$t('transcription.resumeAnalysis', { n: $transcriptionResumeStore.to_analyze })}
            <button type="button" class="transcription-resume-action" onclick={resumeTranscriptionAnalysis}>{$t('transcription.resumeAnalysisFinish')}</button>
            <button type="button" class="transcription-resume-action" onclick={dismissTranscriptionResume}>{$t('common.close')}</button>
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
    <!-- Compteur de bibliothèque : chaque nombre ouvre ce qu'il compte. -->
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

    .transcription-prompt {
        color: var(--color-text);
    }

    /* Une réponse, pas une alerte : elle dit qu'un geste n'avait rien à faire,
       ce qui est une information ordinaire. */
    .transcription-notice {
        color: var(--color-text-muted);
        font-style: italic;
    }

    .transcription-resume-chip {
        display: inline-flex;
        align-items: center;
        gap: 0.4em;
        margin-left: 0.8em;
        white-space: nowrap;
    }

    .transcription-resume-action {
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

    /* Pas de `font: inherit` : style.css le pose déjà (ADR-0008), la garde interdit de le répéter. */
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
        background: var(--color-surface-alt);
        border-top: 1px solid var(--color-border);
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
        color: var(--color-text-muted);
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
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
        border-left: 1px solid var(--color-border);
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
        color: var(--color-text-muted);
        font-size: var(--font-size-base);
        border-left: 1px solid var(--color-border);
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
        background-color: var(--color-surface);
        border: 1px solid var(--color-border);
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
        background-color: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface));
    }

    .command-suggestions li:hover {
        background-color: var(--color-surface-alt);
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
        color: var(--color-text-muted);
    }
</style>
