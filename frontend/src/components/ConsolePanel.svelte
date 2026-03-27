<script>
   import { onMount, tick } from 'svelte';
   import { commentTextStore, currentPositionIndexStore, commandTextStore, statusBarModeStore, showCommandStore, statusBarTextStore, activeTabStore, logEntriesStore } from '../stores/uiStore';
   import { SaveComment, Migrate_1_0_0_to_1_1_0, ClearCommandHistory } from '../../wailsjs/go/main/Database.js';
   import { positionsStore, positionStore } from '../stores/positionStore';
   import { showMetModalStore, showTakePoint2LastModalStore, showTakePoint2LiveModalStore, showTakePoint4LastModalStore, showTakePoint4LiveModalStore, showGammonValue1ModalStore, showGammonValue2ModalStore, showGammonValue4ModalStore, showMetadataModalStore, showTakePoint2ModalStore, showTakePoint4ModalStore } from '../stores/uiStore';
   import { databaseLoadedStore } from '../stores/databaseStore';
   import { commandHistoryStore } from '../stores/commandHistoryStore';
   import { searchHistoryStore } from '../stores/searchHistoryStore';
   import { LoadCommandHistory, SaveCommand, SaveSearchHistory } from '../../wailsjs/go/main/Database.js';
   import { Migrate_1_1_0_to_1_2_0, Migrate_1_2_0_to_1_3_0 } from '../../wailsjs/go/main/Database.js';

   export let onToggleHelp;
   export let onNewDatabase;
   export let onOpenDatabase;
   export let onImportDatabase;
   export let onExportDatabase;
   export let importPosition;
   export let onSavePosition;
   export let onUpdatePosition;
   export let onDeletePosition;
   export let onToggleAnalysis;
   export let onToggleComment;
   export let exitApp;
   export let onLoadPositionsByFilters;
   export let onLoadAllPositions;
   export let toggleFilterLibraryPanel;
   export let toggleSearchHistoryPanel;
   export let toggleMatchPanel;
   export let toggleCollectionPanel;
   export let toggleEPCMode;
   export let toggleMatchMode;

   let inputEl;
   let outputEl;
   let logContainer;

   let positions = [];
   positionsStore.subscribe(value => positions = value);

   let databaseLoaded = false;
   databaseLoadedStore.subscribe(value => databaseLoaded = value);

   let commandHistory = [];
   let historyIndex = -1;
   let consoleOutput = [];
   let logEntries = [];

   commandHistoryStore.subscribe(value => commandHistory = value);

   logEntriesStore.subscribe(async value => {
      logEntries = value;
      await tick();
      if (logContainer) {
         logContainer.scrollTop = logContainer.scrollHeight;
      }
   });

   // Focus the input when the console tab becomes active
   activeTabStore.subscribe(async value => {
      if (value === 'console') {
         await loadHistory();
         await tick();
         inputEl?.focus();
      }
   });

   async function loadHistory() {
      const history = await LoadCommandHistory();
      commandHistoryStore.set((history || []).reverse());
      historyIndex = -1;
   }

   export function focusInput() {
      inputEl?.focus();
   }

   async function scrollToBottom() {
      await tick();
      if (outputEl) {
         outputEl.scrollTop = outputEl.scrollHeight;
      }
   }

   function handleKeyDown(event) {
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
         inputEl?.blur();
      } else if (event.code === 'Enter') {
         event.stopPropagation();
         const command = inputEl.value.trim();
         if (command) {
            consoleOutput = [...consoleOutput, { type: 'input', text: command }];
            commandHistoryStore.update(history => {
               history = history || [];
               history.unshift(command);
               return history;
            });
            historyIndex = -1;
            SaveCommand(command);
            processCommand(command);
         }
         commandTextStore.set('');
         scrollToBottom();
      } else if (event.ctrlKey && event.code === 'KeyH') {
         event.stopPropagation();
         onToggleHelp();
      }
      // Other keys (including Ctrl shortcuts) propagate to global handler
   }

   function processCommand(command) {
      const match = command.match(/^(\d+)$/);
      if (match) {
         const positionNumber = parseInt(match[1], 10);
         let index;
         if (positionNumber < 1) {
            index = 0;
         } else if (positionNumber > positions.length) {
            index = positions.length - 1;
         } else {
            index = positionNumber - 1;
         }
         currentPositionIndexStore.set(index);
         consoleOutput = [...consoleOutput, { type: 'output', text: `Go to position ${index + 1}` }];
      } else if (command.startsWith('#')) {
         const tags = command.slice(1).trim();
         insertTags(tags);
         consoleOutput = [...consoleOutput, { type: 'output', text: `Tags added: ${tags}` }];
      } else if (command === 'new' || command === 'ne' || command === 'n') {
         onNewDatabase();
      } else if (command === 'open' || command === 'op' || command === 'o') {
         onOpenDatabase();
      } else if (command === 'import_db' || command === 'idb') {
         onImportDatabase();
      } else if (command === 'export_db' || command === 'edb') {
         onExportDatabase();
      } else if (command === 'import' || command === 'i') {
         importPosition();
      } else if (command === 'write' || command === 'wr' || command === 'w') {
         onSavePosition();
      } else if (command === 'write!' || command === 'wr!' || command === 'w!') {
         onUpdatePosition();
      } else if (command === 'delete' || command === 'del' || command === 'd') {
         onDeletePosition();
      } else if (command === 'list' || command === 'l') {
         onToggleAnalysis();
      } else if (command === 'comment' || command === 'co') {
         onToggleComment();
      } else if (command === 'quit' || command === 'q') {
         exitApp();
      } else if (command === 'help' || command === 'he' || command === 'h') {
         onToggleHelp();
      } else if (command === 'e') {
         onLoadAllPositions();
      } else if (command.startsWith('ss')) {
         handleSubSearch(command);
      } else if (command.startsWith('s')) {
         handleSearch(command);
      } else if (command === 'filter' || command === 'fl') {
         toggleFilterLibraryPanel();
      } else if (command === 'history' || command === 'hi') {
         toggleSearchHistoryPanel();
      } else if (command === 'match' || command === 'ma') {
         toggleMatchPanel();
      } else if (command === 'collection' || command === 'coll') {
         toggleCollectionPanel();
      } else if (command === 'epc') {
         toggleEPCMode();
      } else if (command === 'm') {
         toggleMatchMode();
      } else if (command === 'met') {
         showMetModalStore.set(true);
      } else if (command === 'tp2_last') {
         showTakePoint2LastModalStore.set(true);
      } else if (command === 'tp2_live') {
         showTakePoint2LiveModalStore.set(true);
      } else if (command === 'tp4_last') {
         showTakePoint4LastModalStore.set(true);
      } else if (command === 'tp4_live') {
         showTakePoint4LiveModalStore.set(true);
      } else if (command === 'gv1') {
         showGammonValue1ModalStore.set(true);
      } else if (command === 'gv2') {
         showGammonValue2ModalStore.set(true);
      } else if (command === 'gv4') {
         showGammonValue4ModalStore.set(true);
      } else if (command === 'meta') {
         if (databaseLoaded) {
            showMetadataModalStore.set(true);
         } else {
            statusBarTextStore.set('No database loaded.');
         }
      } else if (command === 'tp2') {
         showTakePoint2ModalStore.set(true);
      } else if (command === 'tp4') {
         showTakePoint4ModalStore.set(true);
      } else if (command === 'migrate_from_1_0_to_1_1') {
         Migrate_1_0_0_to_1_1_0().then(() => {
            statusBarTextStore.set('Database migrated to version 1.1.0 successfully.');
         }).catch(error => {
            console.error('Error migrating database:', error);
            statusBarTextStore.set('Error migrating database.');
         });
      } else if (command === 'migrate_from_1_1_to_1_2') {
         Migrate_1_1_0_to_1_2_0().then(() => {
            statusBarTextStore.set('Database migrated to version 1.2.0 successfully.');
         }).catch(error => {
            console.error('Error migrating database:', error);
            statusBarTextStore.set('Error migrating database.');
         });
      } else if (command === 'migrate_from_1_2_to_1_3') {
         Migrate_1_2_0_to_1_3_0().then(() => {
            statusBarTextStore.set('Database migrated to version 1.3.0 successfully.');
         }).catch(error => {
            console.error('Error migrating database:', error);
            statusBarTextStore.set('Error migrating database.');
         });
      } else if (command === 'cl' || command === 'clear') {
         ClearCommandHistory().then(() => {
            commandHistoryStore.set([]);
            consoleOutput = [];
            statusBarTextStore.set('Command history cleared.');
         }).catch(error => {
            console.error('Error clearing command history:', error);
            statusBarTextStore.set('Error clearing command history.');
         });
      }
   }

   function handleSubSearch(command) {
      if ($statusBarModeStore === 'NORMAL' || $statusBarModeStore === 'EDIT') {
         if (positions.length === 0) {
            statusBarTextStore.set('No current results to search in.');
            return;
         }
         const currentIDs = positions.map(p => p.id).filter(id => id != null).join(',');
         const searchHistoryEntry = {
            command: command,
            position: JSON.stringify($positionStore),
            timestamp: Date.now()
         };
         searchHistoryStore.update(history => {
            const newHistory = [searchHistoryEntry, ...history].slice(0, 100);
            return newHistory;
         });
         SaveSearchHistory(command, JSON.stringify($positionStore)).catch(err => {
            console.error('Error saving search history:', err);
         });

         if (command === 'ss') {
            onLoadPositionsByFilters([], false, false, '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', false, false, '', '', '', '', '', '', false, false, '', command, '', '', currentIDs);
         } else {
            const filters = command.slice(2).trim().split(' ').map(filter => filter.trim());
            const parsedFilters = parseFilters(filters, command);
            onLoadPositionsByFilters(
               filters, parsedFilters.includeCube, parsedFilters.includeScore,
               parsedFilters.pipCountFilter, parsedFilters.winRateFilter, parsedFilters.gammonRateFilter,
               parsedFilters.backgammonRateFilter, parsedFilters.player2WinRateFilter, parsedFilters.player2GammonRateFilter,
               parsedFilters.player2BackgammonRateFilter, parsedFilters.player1CheckerOffFilter, parsedFilters.player2CheckerOffFilter,
               parsedFilters.player1BackCheckerFilter, parsedFilters.player2BackCheckerFilter,
               parsedFilters.player1CheckerInZoneFilter, parsedFilters.player2CheckerInZoneFilter,
               parsedFilters.searchText, parsedFilters.player1AbsolutePipCountFilter, parsedFilters.equityFilter,
               parsedFilters.decisionTypeFilter, parsedFilters.diceRollFilter, parsedFilters.movePatternFilter,
               parsedFilters.dateFilter, parsedFilters.player1OutfieldBlotFilter, parsedFilters.player2OutfieldBlotFilter,
               parsedFilters.player1JanBlotFilter, parsedFilters.player2JanBlotFilter,
               parsedFilters.noContactFilter, parsedFilters.mirrorPositionFilter, parsedFilters.moveErrorFilter,
               command, parsedFilters.matchIDsFilter, parsedFilters.tournamentIDsFilter, currentIDs
            );
         }
      } else {
         statusBarTextStore.set('Search in results is not available in current mode.');
      }
   }

   function handleSearch(command) {
      if ($statusBarModeStore === 'NORMAL' || $statusBarModeStore === 'EDIT') {
         const searchHistoryEntry = {
            command: command,
            position: JSON.stringify($positionStore),
            timestamp: Date.now()
         };
         searchHistoryStore.update(history => {
            const newHistory = [searchHistoryEntry, ...history].slice(0, 100);
            return newHistory;
         });
         SaveSearchHistory(command, JSON.stringify($positionStore)).catch(err => {
            console.error('Error saving search history:', err);
         });

         if (command === 's') {
            onLoadPositionsByFilters([], false, false, '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', '', false, false, '', '', '', '', '', '', false, false, '', command, '', '');
         } else {
            const filters = command.slice(1).trim().split(' ').map(filter => filter.trim());
            const parsedFilters = parseFilters(filters, command);
            onLoadPositionsByFilters(
               filters, parsedFilters.includeCube, parsedFilters.includeScore,
               parsedFilters.pipCountFilter, parsedFilters.winRateFilter, parsedFilters.gammonRateFilter,
               parsedFilters.backgammonRateFilter, parsedFilters.player2WinRateFilter, parsedFilters.player2GammonRateFilter,
               parsedFilters.player2BackgammonRateFilter, parsedFilters.player1CheckerOffFilter, parsedFilters.player2CheckerOffFilter,
               parsedFilters.player1BackCheckerFilter, parsedFilters.player2BackCheckerFilter,
               parsedFilters.player1CheckerInZoneFilter, parsedFilters.player2CheckerInZoneFilter,
               parsedFilters.searchText, parsedFilters.player1AbsolutePipCountFilter, parsedFilters.equityFilter,
               parsedFilters.decisionTypeFilter, parsedFilters.diceRollFilter, parsedFilters.movePatternFilter,
               parsedFilters.dateFilter, parsedFilters.player1OutfieldBlotFilter, parsedFilters.player2OutfieldBlotFilter,
               parsedFilters.player1JanBlotFilter, parsedFilters.player2JanBlotFilter,
               parsedFilters.noContactFilter, parsedFilters.mirrorPositionFilter, parsedFilters.moveErrorFilter,
               command, parsedFilters.matchIDsFilter, parsedFilters.tournamentIDsFilter
            );
         }
      } else {
         statusBarTextStore.set('Search requires NORMAL or EDIT mode.');
      }
   }

   function parseFilters(filters, command) {
      const includeCube = filters.includes('cube') || filters.includes('cu') || filters.includes('c') || filters.includes('cub');
      const includeScore = filters.includes('score') || filters.includes('sco') || filters.includes('sc') || filters.includes('s');
      const noContactFilter = filters.includes('nc');
      const decisionTypeFilter = filters.includes('d');
      const diceRollFilter = filters.includes('D');
      const mirrorPositionFilter = filters.includes('M');
      const pipCountFilter = filters.find(f => typeof f === 'string' && (f.startsWith('p>') || f.startsWith('p<') || f.startsWith('p')));
      const winRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('w>') || f.startsWith('w<') || f.startsWith('w')));
      const gammonRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('g>') || f.startsWith('g<') || f.startsWith('g')));
      const backgammonRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('b>') || f.startsWith('b<') || (f.startsWith('b') && !f.startsWith('bo'))) && !f.startsWith('bj'));
      const player2WinRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('W>') || f.startsWith('W<') || f.startsWith('W')));
      const player2GammonRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('G>') || f.startsWith('G<') || f.startsWith('G')));
      const player2BackgammonRateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('B>') || f.startsWith('B<') || f.startsWith('B') && !f.startsWith('BO')) && !f.startsWith('BJ'));
      let player1CheckerOffFilter = filters.find(f => typeof f === 'string' && (f.startsWith('o>') || f.startsWith('o<') || f.startsWith('o')));
      if (player1CheckerOffFilter && !player1CheckerOffFilter.includes(',') && !player1CheckerOffFilter.includes('>') && !player1CheckerOffFilter.includes('<')) {
         player1CheckerOffFilter = `${player1CheckerOffFilter},${player1CheckerOffFilter.slice(1)}`;
      }
      let player2CheckerOffFilter = filters.find(f => typeof f === 'string' && (f.startsWith('O>') || f.startsWith('O<') || f.startsWith('O')));
      if (player2CheckerOffFilter && !player2CheckerOffFilter.includes(',') && !player2CheckerOffFilter.includes('>') && !player2CheckerOffFilter.includes('<')) {
         player2CheckerOffFilter = `${player2CheckerOffFilter},${player2CheckerOffFilter.slice(1)}`;
      }
      let player1BackCheckerFilter = filters.find(f => typeof f === 'string' && (f.startsWith('k>') || f.startsWith('k<') || f.startsWith('k')));
      if (player1BackCheckerFilter && !player1BackCheckerFilter.includes(',') && !player1BackCheckerFilter.includes('>') && !player1BackCheckerFilter.includes('<')) {
         player1BackCheckerFilter = `${player1BackCheckerFilter},${player1BackCheckerFilter.slice(1)}`;
      }
      let player2BackCheckerFilter = filters.find(f => typeof f === 'string' && (f.startsWith('K>') || f.startsWith('K<') || f.startsWith('K')));
      if (player2BackCheckerFilter && !player2BackCheckerFilter.includes(',') && !player2BackCheckerFilter.includes('>') && !player2BackCheckerFilter.includes('<')) {
         player2BackCheckerFilter = `${player2BackCheckerFilter},${player2BackCheckerFilter.slice(1)}`;
      }
      let player1CheckerInZoneFilter = filters.find(f => typeof f === 'string' && (f.startsWith('z>') || f.startsWith('z<') || f.startsWith('z')));
      if (player1CheckerInZoneFilter && !player1CheckerInZoneFilter.includes(',') && !player1CheckerInZoneFilter.includes('>') && !player1CheckerInZoneFilter.includes('<')) {
         player1CheckerInZoneFilter = `${player1CheckerInZoneFilter},${player1CheckerInZoneFilter.slice(1)}`;
      }
      let player2CheckerInZoneFilter = filters.find(f => typeof f === 'string' && (f.startsWith('Z>') || f.startsWith('Z<') || f.startsWith('Z')));
      if (player2CheckerInZoneFilter && !player2CheckerInZoneFilter.includes(',') && !player2CheckerInZoneFilter.includes('>') && !player2CheckerInZoneFilter.includes('<')) {
         player2CheckerInZoneFilter = `${player2CheckerInZoneFilter},${player2CheckerInZoneFilter.slice(1)}`;
      }
      const player1AbsolutePipCountFilter = filters.find(f => typeof f === 'string' && (f.startsWith('P>') || f.startsWith('P<') || f.startsWith('P')));
      const equityFilter = filters.find(f => typeof f === 'string' && (f.startsWith('e>') || f.startsWith('e<') || f.startsWith('e')));
      const dateFilter = filters.find(f => typeof f === 'string' && (f.startsWith('T>') || f.startsWith('T<') || f.startsWith('T')));
      const movePatternMatch = command.match(/m["'][^"']*["']/);
      const movePatternFilter = movePatternMatch ? movePatternMatch[0] : '';
      const searchTextMatch = command.match(/t["'][^"']*["']/);
      const searchText = searchTextMatch ? searchTextMatch[0] : '';
      const player1OutfieldBlotFilter = filters.find(f => typeof f === 'string' && (f.startsWith('bo>') || f.startsWith('bo<') || f.startsWith('bo')));
      const player2OutfieldBlotFilter = filters.find(f => typeof f === 'string' && (f.startsWith('BO>') || f.startsWith('BO<') || f.startsWith('BO')));
      const player1JanBlotFilter = filters.find(f => typeof f === 'string' && (f.startsWith('bj>') || f.startsWith('bj<') || f.startsWith('bj')));
      const player2JanBlotFilter = filters.find(f => typeof f === 'string' && (f.startsWith('BJ>') || f.startsWith('BJ<') || f.startsWith('BJ')));
      const moveErrorFilter = filters.find(f => typeof f === 'string' && (f.startsWith('E>') || f.startsWith('E<') || (f.startsWith('E') && /^E\d/.test(f))));

      const matchIDTokens = filters.filter(f => typeof f === 'string' && /^ma\d/.test(f));
      let matchIDsFilter = '';
      if (matchIDTokens.length > 0) {
         const parts = matchIDTokens.map(token => token.slice(2));
         matchIDsFilter = parts.join(';');
      }

      const tournamentIDTokens = filters.filter(f => typeof f === 'string' && /^tn\d/.test(f));
      let tournamentIDsFilter = '';
      if (tournamentIDTokens.length > 0) {
         const parts = tournamentIDTokens.map(token => token.slice(2));
         tournamentIDsFilter = parts.join(';');
      }

      return {
         includeCube, includeScore, noContactFilter, decisionTypeFilter, diceRollFilter, mirrorPositionFilter,
         pipCountFilter, winRateFilter, gammonRateFilter, backgammonRateFilter,
         player2WinRateFilter, player2GammonRateFilter, player2BackgammonRateFilter,
         player1CheckerOffFilter, player2CheckerOffFilter,
         player1BackCheckerFilter, player2BackCheckerFilter,
         player1CheckerInZoneFilter, player2CheckerInZoneFilter,
         player1AbsolutePipCountFilter, equityFilter, dateFilter,
         movePatternFilter, searchText,
         player1OutfieldBlotFilter, player2OutfieldBlotFilter,
         player1JanBlotFilter, player2JanBlotFilter, moveErrorFilter,
         matchIDsFilter, tournamentIDsFilter
      };
   }

   function insertTags(tags) {
      commentTextStore.update(text => {
         const existingTags = new Set(text.match(/#[^\s#]+/g) || []);
         const newTags = tags.split(' ').filter(tag => !existingTags.has(tag));
         const updatedText = `${newTags.join(' ')}\n${text}`;
         setTimeout(() => {
            const textAreaEl = document.getElementById('commentTextArea');
            if (textAreaEl) {
               textAreaEl.setSelectionRange(updatedText.length, updatedText.length);
               textAreaEl.focus();
            }
         }, 0);
         SaveComment($currentPositionIndexStore, updatedText);
         return updatedText;
      });
   }

   onMount(async () => {
      await loadHistory();
   });
</script>

<div class="console-panel">
   <!-- Left: Command Prompt -->
   <div class="console-left">
      <div class="console-output" bind:this={outputEl}>
         {#each consoleOutput as entry}
            <div class="console-line {entry.type}">
               {#if entry.type === 'input'}
                  <span class="prompt">&gt;</span> <span class="cmd">{entry.text}</span>
               {:else}
                  <span class="result">{entry.text}</span>
               {/if}
            </div>
         {/each}
      </div>
      <div class="console-input-row">
         <span class="prompt-char">&gt;</span>
         <input
            type="text"
            bind:this={inputEl}
            bind:value={$commandTextStore}
            class="console-input"
            placeholder="Type command..."
            on:keydown={handleKeyDown}
         />
      </div>
   </div>

   <div class="console-divider"></div>

   <!-- Right: Logs -->
   <div class="console-right">
      <div class="logs-output" bind:this={logContainer}>
         {#if logEntries.length === 0}
            <div class="empty-msg">No operations logged yet.</div>
         {:else}
            {#each logEntries as entry}
               <div class="log-line">
                  <span class="log-time">{entry.timestamp.toLocaleTimeString('sv-SE', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
                  <span class="log-msg">{entry.message}</span>
               </div>
            {/each}
         {/if}
      </div>
   </div>
</div>

<style>
   .console-panel {
      display: flex;
      flex-direction: row;
      height: 100%;
      background: #fff;
      color: #333;
      font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
      font-size: 13px;
   }

   .console-left {
      flex: 1;
      display: flex;
      flex-direction: column;
      min-width: 0;
   }

   .console-divider {
      width: 1px;
      background: #ddd;
      flex-shrink: 0;
   }

   .console-right {
      flex: 1;
      display: flex;
      flex-direction: column;
      min-width: 0;
   }

   .console-output {
      flex: 1;
      overflow-y: auto;
      padding: 4px 8px;
      min-height: 0;
   }

   .console-line {
      line-height: 1.5;
      white-space: pre-wrap;
      word-break: break-all;
   }

   .console-line.input .prompt {
      color: #555;
      font-weight: bold;
   }

   .console-line.input .cmd {
      color: #555;
   }

   .console-line.output .result {
      color: #2e7d32;
   }

   .console-input-row {
      display: flex;
      align-items: center;
      border-top: 1px solid #ddd;
      padding: 2px 8px;
      background: #fafafa;
   }

   .prompt-char {
      color: #555;
      font-weight: bold;
      margin-right: 6px;
   }

   .console-input {
      flex: 1;
      background: transparent;
      border: none;
      outline: none;
      color: #333;
      font-family: inherit;
      font-size: inherit;
      padding: 4px 0;
   }

   .console-input::placeholder {
      color: #aaa;
   }

   .logs-output {
      flex: 1;
      overflow-y: auto;
      padding: 4px 8px;
      min-height: 0;
   }

   .log-line {
      display: flex;
      gap: 10px;
      line-height: 1.6;
      border-bottom: 1px solid #f0f0f0;
      padding: 1px 0;
   }

   .log-time {
      color: #888;
      white-space: nowrap;
      min-width: 64px;
   }

   .log-msg {
      color: #333;
   }

   .empty-msg {
      color: #999;
      text-align: center;
      padding: 20px;
   }
</style>
