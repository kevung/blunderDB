<script>
   import { onMount, onDestroy } from 'svelte';
   import { commentTextStore, currentPositionIndexStore, commandTextStore, previousModeStore, statusBarModeStore, showCommandStore, statusBarTextStore } from '../stores/uiStore';
   import { SaveComment } from '../../wailsjs/go/main/Database.js';
   import { positionsStore } from '../stores/positionStore';
   import { showMetModalStore, showTakePoint2LastModalStore, showTakePoint2LiveModalStore, showTakePoint4LastModalStore, showTakePoint4LiveModalStore, showGammonValue1ModalStore, showGammonValue2ModalStore, showGammonValue4ModalStore, showMetadataModalStore, showTakePoint2ModalStore, showTakePoint4ModalStore } from '../stores/uiStore';
   import { databaseLoadedStore } from '../stores/databaseStore'; // Ensure the import path is correct

   export let onToggleHelp;
   export let onNewDatabase;
   export let onOpenDatabase;
   export let importPosition;
   export let onSavePosition;
   export let onUpdatePosition;
   export let onDeletePosition;
   export let onToggleAnalysis;
   export let onToggleComment;
   export let exitApp;
   export let onLoadPositionsByFilters;
   export let onLoadAllPositions;
   let inputEl;

   let initialized = false;

   // Subscribe to the stores
   let positions = [];
   positionsStore.subscribe(value => positions = value);

   let databaseLoaded = false;
   databaseLoadedStore.subscribe(value => databaseLoaded = value);

   showCommandStore.subscribe(value => {
      if (value) {
         previousModeStore.set($statusBarModeStore);
         statusBarModeStore.set('COMMAND');
         commandTextStore.set('');
         setTimeout(() => {
            inputEl?.focus();
         }, 0);
         window.addEventListener('click', handleClickOutside);
      } else {
         statusBarModeStore.set($previousModeStore); // Restore the previous mode
         window.removeEventListener('click', handleClickOutside);
      }
   });

   function handleKeyDown(event) {
      event.stopPropagation();

      if ($showCommandStore) {
         if (event.code === 'Backspace' && inputEl.value === '') {
            showCommandStore.set(false);
         } else if (event.code === 'Escape') {
            showCommandStore.set(false);
         } else if (event.code === 'Enter') {
            const command = inputEl.value.trim();
            console.log('Command entered:', command); // Debugging log
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
            } else if (command.startsWith('#')) {
               const tags = command.slice(1).trim();
               insertTags(tags);
            } else if (command === 'new' || command === 'ne' || command === 'n') {
               onNewDatabase();
            } else if (command === 'open' || command === 'op' || command === 'o') {
               onOpenDatabase();
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
               console.log('Toggling comment panel'); // Debugging log
               onToggleComment();
            } else if (command === 'quit' || command === 'q') {
               exitApp();
            } else if (command === 'help' || command === 'he' || command === 'h') {
               onToggleHelp();
            } else if (command === 'e') {
               onLoadAllPositions();
            } else if (command.startsWith('s')) {
               if ($previousModeStore === 'EDIT') {
                  if (command === 's') {
                     onLoadPositionsByFilters([]);
                  } else {
                     const filters = command.slice(1).trim().split(' ').map(filter => filter.trim());
                     const includeCube = filters.includes('cube') || filters.includes('cu') || filters.includes('c') || filters.includes('cub');
                     const includeScore = filters.includes('score') || filters.includes('sco') || filters.includes('sc') || filters.includes('s');
                     const decisionTypeFilter = filters.includes('d');
                     const diceRollFilter = filters.includes('D');
                     const pipCountFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('p>') || filter.startsWith('p<') || filter.startsWith('p')));
                     const winRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('w>') || filter.startsWith('w<') || filter.startsWith('w')));
                     const gammonRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('g>') || filter.startsWith('g<') || filter.startsWith('g')));
                     const backgammonRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('b>') || filter.startsWith('b<') || filter.startsWith('b')));
                     const player2WinRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('W>') || filter.startsWith('W<') || filter.startsWith('W')));
                     const player2GammonRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('G>') || filter.startsWith('G<') || filter.startsWith('G')));
                     const player2BackgammonRateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('B>') || filter.startsWith('B<') || filter.startsWith('B')));
                     let player1CheckerOffFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('o>') || filter.startsWith('o<') || filter.startsWith('o')));
                     if (player1CheckerOffFilter && !player1CheckerOffFilter.includes(',') && !player1CheckerOffFilter.includes('>') && !player1CheckerOffFilter.includes('<')) {
                        player1CheckerOffFilter = `${player1CheckerOffFilter},${player1CheckerOffFilter.slice(1)}`; // Handle case where 'ox' means 'ox,x'
                     }
                     let player2CheckerOffFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('O>') || filter.startsWith('O<') || filter.startsWith('O')));
                     if (player2CheckerOffFilter && !player2CheckerOffFilter.includes(',') && !player2CheckerOffFilter.includes('>') && !player2CheckerOffFilter.includes('<')) {
                        player2CheckerOffFilter = `${player2CheckerOffFilter},${player2CheckerOffFilter.slice(1)}`; // Handle case where 'Ox' means 'Ox,x'
                     }
                     let player1BackCheckerFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('k>') || filter.startsWith('k<') || filter.startsWith('k')));
                     if (player1BackCheckerFilter && !player1BackCheckerFilter.includes(',') && !player1BackCheckerFilter.includes('>') && !player1BackCheckerFilter.includes('<')) {
                        player1BackCheckerFilter = `${player1BackCheckerFilter},${player1BackCheckerFilter.slice(1)}`; // Handle case where 'kx' means 'kx,x'
                     }
                     let player2BackCheckerFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('K>') || filter.startsWith('K<') || filter.startsWith('K')));
                     if (player2BackCheckerFilter && !player2BackCheckerFilter.includes(',') && !player2BackCheckerFilter.includes('>') && !player2BackCheckerFilter.includes('<')) {
                        player2BackCheckerFilter = `${player2BackCheckerFilter},${player2BackCheckerFilter.slice(1)}`; // Handle case where 'Kx' means 'Kx,x'
                     }
                     let player1CheckerInZoneFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('z>') || filter.startsWith('z<') || filter.startsWith('z')));
                     if (player1CheckerInZoneFilter && !player1CheckerInZoneFilter.includes(',') && !player1CheckerInZoneFilter.includes('>') && !player1CheckerInZoneFilter.includes('<')) {
                        player1CheckerInZoneFilter = `${player1CheckerInZoneFilter},${player1CheckerInZoneFilter.slice(1)}`; // Handle case where 'zx' means 'zx,x'
                     }
                     let player2CheckerInZoneFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('Z>') || filter.startsWith('Z<') || filter.startsWith('Z')));
                     if (player2CheckerInZoneFilter && !player2CheckerInZoneFilter.includes(',') && !player2CheckerInZoneFilter.includes('>') && !player2CheckerInZoneFilter.includes('<')) {
                        player2CheckerInZoneFilter = `${player2CheckerInZoneFilter},${player2CheckerInZoneFilter.slice(1)}`; // Handle case where 'Zx' means 'Zx,x'
                     }
                     const player1AbsolutePipCountFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('P>') || filter.startsWith('P<') || filter.startsWith('P')));
                     const equityFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('e>') || filter.startsWith('e<') || filter.startsWith('e')));
                     const dateFilter = filters.find(filter => typeof filter === 'string' && (filter.startsWith('T>') || filter.startsWith('T<') || filter.startsWith('T')));
                     const movePatternMatch = command.match(/m["'][^"']*["']/);
                     const movePatternFilter = movePatternMatch ? movePatternMatch[0] : '';
                     const searchTextMatch = command.match(/t["'][^"']*["']/);
                     const searchText = searchTextMatch ? searchTextMatch[0] : '';
                     console.log('Filters:', filters); // Add logging
                     console.log('Search Text:', searchText); // Add logging
                     console.log('Move Pattern Filter:', movePatternFilter); // Add logging

                     // display in console log all the filters
                     console.log('includeCube:', includeCube);
                     console.log('includeScore:', includeScore);
                     console.log('pipCountFilter:', pipCountFilter);
                     console.log('winRateFilter:', winRateFilter);
                     console.log('gammonRateFilter:', gammonRateFilter);
                     console.log('backgammonRateFilter:', backgammonRateFilter);
                     console.log('player2WinRateFilter:', player2WinRateFilter);
                     console.log('player2GammonRateFilter:', player2GammonRateFilter);
                     console.log('player2BackgammonRateFilter:', player2BackgammonRateFilter);
                     console.log('player1CheckerOffFilter:', player1CheckerOffFilter);
                     console.log('player2CheckerOffFilter:', player2CheckerOffFilter);
                     console.log('player1BackCheckerFilter:', player1BackCheckerFilter);
                     console.log('player2BackCheckerFilter:', player2BackCheckerFilter);
                     console.log('player1CheckerInZoneFilter:', player1CheckerInZoneFilter);
                     console.log('player2CheckerInZoneFilter:', player2CheckerInZoneFilter);
                     console.log('player1AbsolutePipCountFilter:', player1AbsolutePipCountFilter);
                     console.log('equityFilter:', equityFilter);
                     console.log('decisionTypeFilter:', decisionTypeFilter);
                     console.log('diceRollFilter:', diceRollFilter);
                     console.log('dateFilter:', dateFilter);
                     
                     onLoadPositionsByFilters(filters, includeCube, includeScore, pipCountFilter, winRateFilter, gammonRateFilter, backgammonRateFilter, player2WinRateFilter, player2GammonRateFilter, player2BackgammonRateFilter, player1CheckerOffFilter, player2CheckerOffFilter, player1BackCheckerFilter, player2BackCheckerFilter, player1CheckerInZoneFilter, player2CheckerInZoneFilter, searchText, player1AbsolutePipCountFilter, equityFilter, decisionTypeFilter, diceRollFilter, movePatternFilter, dateFilter);
                  }
               } else {
                  statusBarTextStore.set('Search is only available in edit mode.');
               }
            } else if (command === 'met') {
               showMetModalStore.set(true); // Show MET modal
            } else if (command === 'tp2_last') {
               showTakePoint2LastModalStore.set(true); // Show TakePoint2Last modal
            } else if (command === 'tp2_live') {
               showTakePoint2LiveModalStore.set(true); // Show TakePoint2Live modal
            } else if (command === 'tp4_last') {
               showTakePoint4LastModalStore.set(true); // Show TakePoint4Last modal
            } else if (command === 'tp4_live') {
               showTakePoint4LiveModalStore.set(true); // Show TakePoint4Live modal
            } else if (command === 'gv1') {
               showGammonValue1ModalStore.set(true); // Show GammonValue1 modal
            } else if (command === 'gv2') {
               showGammonValue2ModalStore.set(true); // Show GammonValue2 modal
            } else if (command === 'gv4') {
               showGammonValue4ModalStore.set(true); // Show GammonValue4 modal
            } else if (command === 'meta') {
               if (databaseLoaded) {
                  showMetadataModalStore.set(true); // Show Metadata modal
               } else {
                  statusBarTextStore.set('No database loaded.'); // Display message in status bar
               }
            } else if (command === 'tp2') {
               showTakePoint2ModalStore.set(true); // Show TakePoint2 modal
            } else if (command === 'tp4') {
               showTakePoint4ModalStore.set(true); // Show TakePoint4 modal
            }
            showCommandStore.set(false);
         } else if (event.ctrlKey && event.code === 'KeyH') {
            onToggleHelp();
         }
      }
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
         // Save the updated comment to the database
         SaveComment($currentPositionIndexStore, updatedText);
         return updatedText;
      });
   }

   function handleClickOutside(event) {
      if ($showCommandStore && !inputEl.contains(event.target)) {
         showCommandStore.set(false);
      }
   }

   onDestroy(() => {
      window.removeEventListener('click', handleClickOutside);
   });
</script>

{#if $showCommandStore}
   <input
         type="text"
         bind:this={inputEl}
         bind:value={$commandTextStore}
         class="command-input"
         placeholder=" Type your command here. "
         on:keydown={handleKeyDown}
         />
{/if}

<style>
    input {
        position: fixed;
        top: 350px;
        left: 50%;
        transform: translateX(-50%);
        z-index: 1000;
        width: 70%;
        padding: 8px;
        border: 1px solid rgba(0, 0, 0, 0.3); /* Subtle border */
        border-radius: 1px;
        box-shadow: 0 2px 10px rgba(0, 0, 0, 0);
        outline: none;
        background-color: white; /* Ensure background is opaque */
        font-size: 18px;
    }
</style>
