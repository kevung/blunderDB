<script>
   import { onMount, onDestroy } from 'svelte';
   import { commentTextStore, currentPositionIndexStore } from '../stores/uiStore'; // Import commentTextStore and currentPositionIndexStore
   import { SaveComment, LoadAllPositions, LoadPositionsByCheckerPosition } from '../../wailsjs/go/main/Database.js';
   import { positionsStore } from '../stores/positionStore'; // Import positionsStore

   export let visible = false;
   export let onClose;
   export let onToggleHelp;
   export let text = '';
   export let onNewDatabase;
   export let onOpenDatabase;
   export let importPosition;
   export let onSavePosition;
   export let onUpdatePosition;
   export let onDeletePosition;
   export let onToggleAnalysis; // Add the new attribute
   export let onToggleComment; // Add the new attribute
   export let exitApp;
   export let currentPositionId; // Add the current position ID
   export let onLoadPositionsByFilters; // Add the new attribute
   let inputEl;

   let initialized = false;

   // Subscribe to the stores
   let positions = [];
   positionsStore.subscribe(value => positions = value);

   $: if (visible && !initialized) {
      text = '';
      initialized = true;

      setTimeout(() => {
         inputEl?.focus();
      }, 0);

      window.addEventListener('click', handleClickOutside);
   }

   $: if (!visible) {
      initialized = false;
      window.removeEventListener('click', handleClickOutside);
   }

   function handleKeyDown(event) {
      event.stopPropagation();

      if(visible) {
         if(event.code === 'Backspace' && inputEl.value === '') {
            onClose();
         } else if (event.code === 'Escape') {
            onClose();
         } else if (event.code === 'Enter') {
            const command = inputEl.value.trim().toLowerCase();
            console.log('Command entered:', command); // Debugging log
            const match = command.match(/^(\d+)$/);
            if (match) {
               const positionNumber = parseInt(match[1], 10);
               onClose().then(() => {
                  currentPositionIndexStore.set(positionNumber - 1);
               });
            } else if (command === 'new' || command === 'ne' || command === 'n') {
               onClose().then(() => {
                  onNewDatabase();
               });
            } else if (command === 'open' || command === 'op' || command === 'o') {
               onClose().then(() => {
                  onOpenDatabase();
               });
            } else if (command === 'import' || command === 'i') {
               onClose().then(() => {
                  importPosition();
               });
            } else if (command === 'write' || command === 'wr' || command === 'w') {
               onClose().then(() => {
                  onSavePosition();
               });
            } else if (command === 'write!' || command === 'wr!' || command === 'w!') {
               onClose().then(() => {
                  onUpdatePosition();
               });
            } else if (command === 'delete' || command === 'del' || command === 'd') {
               onClose().then(() => {
                  onDeletePosition();
               });
            } else if (command === 'list' || command === 'l') {
               onClose().then(() => {
                  onToggleAnalysis();
               });
            } else if (command === 'comment' || command === 'co') {
               console.log('Toggling comment panel'); // Debugging log
               onClose().then(() => {
                  onToggleComment();
               });
            } else if (command === 'quit' || command === 'q') {
               onClose().then(() => {
                  exitApp();
               });
            } else if (command === 'help' || command === 'he' || command === 'h') {
               onClose().then(() => {
                  onToggleHelp();
               });
            } else if (command === 's') {
               onClose().then(() => {
                  onLoadPositionsByFilters([]);
               });
            } else if (command.startsWith('filter ')) {
               const filters = command.slice(7).split(' ').map(filter => filter.trim());
               onClose().then(() => {
                  onLoadPositionsByFilters(filters);
               });
            } else if (command === 'e') {
               onClose().then(async () => {
                  positionsStore.set(await LoadAllPositions());
                  if (positions.length > 0) {
                     currentPositionIndexStore.set(positions.length - 1);
                  }
               });
            } else if (command.startsWith('#')) {
               const tags = Array.from(new Set(command.split(' ').map((tag, index) => index === 0 ? tag : `#${tag}`))).join(' ');
               onClose().then(() => {
                  insertTags(tags);
               });
            } else if (command.startsWith('s')) {
               const filters = command.slice(1).trim().split(' ').map(filter => filter.trim());
               const includeCube = filters.includes('cube') || filters.includes('cu') || filters.includes('c') || filters.includes('cub');
               const includeScore = filters.includes('score') || filters.includes('sco') || filters.includes('sc') || filters.includes('s');
               const pipCountFilter = filters.find(filter => filter.startsWith('p>') || filter.startsWith('p<') || filter.startsWith('p'));
               const winRateFilter = filters.find(filter => filter.startsWith('w>') || filter.startsWith('w<') || filter.startsWith('w'));
               onClose().then(() => {
                  onLoadPositionsByFilters(filters, includeCube, includeScore, pipCountFilter, winRateFilter);
               });
            } else {
               onClose();
            }
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
         SaveComment(parseInt(currentPositionId), updatedText);
         return updatedText;
      });
   }

   function handleClickOutside(event) {
      if (visible && !inputEl.contains(event.target)) {
         onClose();
      }
   }

   onDestroy(() => {
      window.removeEventListener('click', handleClickOutside);
   });
</script>

{#if visible}
   <input
         type="text"
         bind:this={inputEl}
         bind:value={text}
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
