<script>
    import { tick } from 'svelte';
    import { statusBarTextStore, currentPositionIndexStore, commandTextStore, showCommandInputStore, addLogEntry } from '../stores/uiStore';
    import { positionsStore, matchContextStore } from '../stores/positionStore';
    import { analysisStore } from '../stores/analysisStore';
    import { tableData as metTable } from '../stores/metTable';
    import { takePoint2LiveTable } from '../stores/takePoint2LiveTable';
    import { takePoint2LastTable } from '../stores/takePoint2LastTable';
    import { gammonValue1Table } from '../stores/gammonValue1Table';
    import { gammonValue2Table } from '../stores/gammonValue2Table';
    import { gammonValue4Table } from '../stores/gammonValue4Table';
    import { takePoint4LiveTable } from '../stores/takePoint4LiveTable';
    import { takePoint4LastTable } from '../stores/takePoint4LastTable';
    import { commandHistoryStore } from '../stores/commandHistoryStore';
    import { LoadCommandHistory, SaveCommand } from '../../wailsjs/go/database/Database.js';
    import { get } from 'svelte/store';
    import { t, tMsg, resolveStatusMessage } from '../i18n';
    import { getCommandSuggestions } from '../commandVocabulary.js';

    /** @type {function(string): void} */
    let { onCommand = (_cmd) => {} } = $props();

    let inputEl = $state();
    let showInput = $derived($showCommandInputStore);
    let commandHistory = $derived($commandHistoryStore);
    let historyIndex = -1;

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
                addLogEntry(`> ${command}`, 'command');
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

    function showDatesAndMetadata() {
        const analysis = get(analysisStore);
        const positions = get(positionsStore);
        const currentIndex = get(currentPositionIndexStore);

        const tr = get(t);

        if (!analysis || !analysis.creationDate || !analysis.lastModifiedDate) {
            statusBarTextStore.set(tMsg('statusBar.noDatabaseOpened'));
            return;
        }

        const _options = { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' };
        const formatDate = (date) => {
            const [year, month, day] = date.toLocaleDateString('sv-SE').split('-');
            const time = date.toLocaleTimeString('sv-SE', { hour: '2-digit', minute: '2-digit' });
            return `${year}/${month}/${day} ${time}`;
        };
        const creationDate = formatDate(new Date(analysis.creationDate));
        const lastModifiedDate = formatDate(new Date(analysis.lastModifiedDate));
        let statusText = tr('statusBar.createdModified', { created: creationDate, modified: lastModifiedDate });

        if (positions.length === 0 || currentIndex < 0 || currentIndex >= positions.length) {
            statusText += ` | ${tr('statusBar.noPositionData')}`;
        } else {
            const position = positions[currentIndex];
            const cubeValue = position.cube.value;
            let metValue = 'N/A';
            let tp2LiveValue = 'N/A';
            let tp2LastValue = 'N/A';
            let gv1Value = 'N/A';
            let gv2Value = 'N/A';
            let gv4Value = 'N/A';
            let tp4LiveValue = 'N/A';
            let tp4LastValue = 'N/A';

            if (position.score[0] - 1 >= 0 && position.score[0] - 1 < metTable.length && position.score[1] - 1 >= 0 && position.score[1] - 1 < metTable[0].length) {
                metValue = metTable[position.score[0] - 1][position.score[1] - 1].toFixed(1);
            }

            if (position.score[0] - 2 >= 0 && position.score[0] - 2 < takePoint2LiveTable.length && position.score[1] - 2 >= 0 && position.score[1] - 2 < takePoint2LiveTable[0].length) {
                tp2LiveValue = takePoint2LiveTable[position.score[0] - 2][position.score[1] - 2].toFixed(1);
            }

            if (position.score[0] - 2 >= 0 && position.score[0] - 2 < takePoint2LastTable.length && position.score[1] - 2 >= 0 && position.score[1] - 2 < takePoint2LastTable[0].length) {
                tp2LastValue = takePoint2LastTable[position.score[0] - 2][position.score[1] - 2].toFixed(1);
            }

            if (position.score[0] - 2 >= 0 && position.score[0] - 2 < gammonValue1Table.length && position.score[1] - 2 >= 0 && position.score[1] - 2 < gammonValue1Table[0].length) {
                gv1Value = gammonValue1Table[position.score[0] - 2][position.score[1] - 2].toFixed(2);
            }

            if (position.score[0] - 3 >= 0 && position.score[0] - 3 < gammonValue2Table.length && position.score[1] - 2 >= 0 && position.score[1] - 2 < gammonValue2Table[0].length) {
                gv2Value = gammonValue2Table[position.score[0] - 3][position.score[1] - 2].toFixed(2);
            }

            if (position.score[0] - 5 >= 0 && position.score[0] - 5 < gammonValue4Table.length && position.score[1] - 2 >= 0 && position.score[1] - 2 < gammonValue4Table[0].length) {
                gv4Value = gammonValue4Table[position.score[0] - 5][position.score[1] - 2].toFixed(2);
            }

            if (position.score[0] - 3 >= 0 && position.score[0] - 3 < takePoint4LiveTable.length && position.score[1] - 3 >= 0 && position.score[1] - 3 < takePoint4LiveTable[0].length) {
                tp4LiveValue = takePoint4LiveTable[position.score[0] - 3][position.score[1] - 3].toFixed(0);
            }

            if (position.score[0] - 3 >= 0 && position.score[0] - 3 < takePoint4LastTable.length && position.score[1] - 3 >= 0 && position.score[1] - 3 < takePoint4LastTable[0].length) {
                tp4LastValue = takePoint4LastTable[position.score[0] - 3][position.score[1] - 3].toFixed(0);
            }

            let metadata = `met: ${metValue}`;
            if (cubeValue === 0) {
                metadata += ` | tp2_live: ${tp2LiveValue} | tp2_last: ${tp2LastValue} | gv1: ${gv1Value} | gv2: ${gv2Value}`;
            } else if (cubeValue === 1) {
                metadata += ` | tp4_live: ${tp4LiveValue} | tp4_last: ${tp4LastValue} | gv2: ${gv2Value} | gv4: ${gv4Value}`;
            } else if (cubeValue === 2) {
                metadata += ` | gv4: ${gv4Value}`;
            }
            statusText += ` | ${metadata}`;
        }

        statusBarTextStore.set(statusText);
    }

    window.addEventListener('keydown', (event) => {
        if (event.ctrlKey && event.key === 'g') {
            showDatesAndMetadata();
        }
    });
</script>

<div class="status-bar" role="status" aria-live="polite" data-testid="status-bar" data-tour="statusbar">
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
        <span class="info-message" data-testid="status-bar-message" title={statusMessage}>{statusMessage}</span>
    {/if}
    {#if $matchContextStore.isMatchMode && $matchContextStore.movePositions.length > 0}
        {@const checkerMoves = $matchContextStore.movePositions.filter((p) => p.move_type === 'checker')}
        {@const currentCheckerIndex = $matchContextStore.movePositions.slice(0, $matchContextStore.currentIndex + 1).filter((p) => p.move_type === 'checker').length}
        <span class="position-info">{$t('statusBar.move')} {currentCheckerIndex}/{checkerMoves.length}</span>
        <span class="position-info"
            >{$t('statusBar.game')}
            {$matchContextStore.movePositions[$matchContextStore.currentIndex]?.game_number || 1}/{Math.max(...$matchContextStore.movePositions.map((p) => p.game_number))}</span
        >
    {:else}
        <span class="position-info">{$positionsStore.length > 0 ? $currentPositionIndexStore + 1 : 0} / {$positionsStore.length}</span>
    {/if}
</div>

<style>
    .status-bar {
        display: flex;
        align-items: center;
        background: #f7f7f7;
        border-top: 1px solid #e0e0e0;
        padding: 2px 0;
        flex-shrink: 0;
        width: 100%;
        font-size: 12px;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
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
        font-size: 12px;
        line-height: 22px;
    }

    .position-info {
        padding: 0 10px;
        flex-shrink: 0;
        font-variant-numeric: tabular-nums;
        color: #555;
        font-size: 12px;
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
        font-size: 13px;
        cursor: pointer;
    }

    .command-suggestions li.selected {
        background-color: #e8f0fe;
    }

    .command-suggestions li:hover {
        background-color: #f0f0f0;
    }

    .cmd-name {
        font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
        font-weight: 600;
        color: #333;
    }

    .cmd-aliases {
        font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
        font-size: 11px;
        color: #888;
    }

    .prompt-char {
        color: #1a73e8;
        font-weight: bold;
        margin-right: 4px;
        font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
        font-size: 12px;
        line-height: 22px;
    }

    .command-input {
        flex: 1;
        background: transparent;
        border: none;
        outline: none;
        color: #333;
        font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
        font-size: 12px;
        padding: 0;
        line-height: 22px;
        height: 22px;
    }

    .command-input::placeholder {
        color: #aaa;
    }
</style>
