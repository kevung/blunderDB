<script>
   import { activeTabStore } from '../stores/uiStore';
   
   import ConsolePanel from './ConsolePanel.svelte';
   import AnalysisPanel from './AnalysisPanel.svelte';
   import CommentPanel from './CommentPanel.svelte';
   import SearchPanel from './SearchPanel.svelte';
   import CollectionPanel from './CollectionPanel.svelte';
   import MetadataPanel from './MetadataPanel.svelte';
   import MatchPanel from './MatchPanel.svelte';
   import TournamentPanel from './TournamentPanel.svelte';
   import EPCPanel from './EPCPanel.svelte';
   import AnkiPanel from './AnkiPanel.svelte';

   // Props passed through to panels
   export let onLoadPositionsByFilters;
   export let onCloseAnalysis;
   export let onCloseComment;
   export let onOpenCollection;
   export let onAddToFilterLibrary;

   let tabs = [
      { id: 'analysis', label: 'Analysis', icon: 'analysis', shortcut: 'Ctrl+L' },
      { id: 'comments', label: 'Comments', icon: 'comments', shortcut: 'Ctrl+P' },
      { id: 'search', label: 'Search', icon: 'search', shortcut: 'Ctrl+F' },
      { id: 'collections', label: 'Collections', icon: 'collections', shortcut: 'Ctrl+B' },
      { id: 'matches', label: 'Matchs', icon: 'matches', shortcut: 'Ctrl+Tab' },
      { id: 'tournaments', label: 'Tournaments', icon: 'tournaments', shortcut: 'Ctrl+Y' },
      { id: 'epc', label: 'EPC', icon: 'epc', shortcut: 'Ctrl+E' },
      { id: 'anki', label: 'Anki', icon: 'anki', shortcut: 'Ctrl+K' },
      { id: 'metadata', label: 'Metadata', icon: 'metadata', shortcut: 'Ctrl+M' },
      { id: 'log', label: 'Log', icon: 'log', shortcut: '' },
   ];

   let draggedIndex = null;
   let dragOverIndex = null;
   let isDragging = false;
   let dragStartX = 0;
   let tabBarEl;

   function selectTab(tabId) {
      activeTabStore.set(tabId);
   }

   function getTabIndexAtX(clientX) {
      if (!tabBarEl) return null;
      const buttons = tabBarEl.children;
      for (let i = 0; i < buttons.length; i++) {
         const rect = buttons[i].getBoundingClientRect();
         if (clientX >= rect.left && clientX <= rect.right) {
            return i;
         }
      }
      return null;
   }

   function handleMouseDown(e, index) {
      e.preventDefault();
      draggedIndex = index;
      dragStartX = e.clientX;
      isDragging = false;

      function onMouseMove(ev) {
         ev.preventDefault();
         if (Math.abs(ev.clientX - dragStartX) > 4) {
            isDragging = true;
         }
         if (isDragging) {
            dragOverIndex = getTabIndexAtX(ev.clientX);
         }
      }

      function onMouseUp(ev) {
         window.removeEventListener('mousemove', onMouseMove);
         window.removeEventListener('mouseup', onMouseUp);

         if (isDragging && draggedIndex !== null && dragOverIndex !== null && draggedIndex !== dragOverIndex) {
            const reordered = [...tabs];
            const [moved] = reordered.splice(draggedIndex, 1);
            reordered.splice(dragOverIndex, 0, moved);
            tabs = reordered;
         } else if (!isDragging) {
            // Simple click - select the tab
            selectTab(tabs[index].id);
         }
         draggedIndex = null;
         dragOverIndex = null;
         isDragging = false;
      }

      window.addEventListener('mousemove', onMouseMove);
      window.addEventListener('mouseup', onMouseUp);
   }
</script>

<div class="tabbed-panel">
   <div class="tab-bar" bind:this={tabBarEl}>
      {#each tabs as tab, i}
         <button
            class="tab-button"
            class:active={$activeTabStore === tab.id}
            class:drag-over={dragOverIndex === i && draggedIndex !== i}
            class:dragging={draggedIndex === i && isDragging}
            title={tab.shortcut ? `${tab.label} (${tab.shortcut})` : tab.label}
            on:mousedown={(e) => handleMouseDown(e, i)}
         >
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="tab-icon">
               {#if tab.icon === 'log'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
               {:else if tab.icon === 'analysis'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M8.242 5.992h12m-12 6.003H20.24m-12 5.999h12M4.117 7.495v-3.75H2.99m1.125 3.75H2.99m1.125 0H5.24m-1.92 2.577a1.125 1.125 0 1 1 1.591 1.59l-1.83 1.83h2.16M2.99 15.745h1.125a1.125 1.125 0 0 1 0 2.25H3.74m0-.002h.375a1.125 1.125 0 0 1 0 2.25H2.99" />
               {:else if tab.icon === 'comments'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.76c0 1.6 1.123 2.994 2.707 3.227 1.087.16 2.185.283 3.293.369V21l4.076-4.076a1.526 1.526 0 0 1 1.037-.443 48.282 48.282 0 0 0 5.68-.494c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z" />
               {:else if tab.icon === 'search'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
               {:else if tab.icon === 'collections'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M17.593 3.322c1.1.128 1.907 1.077 1.907 2.185V21L12 17.25 4.5 21V5.507c0-1.108.806-2.057 1.907-2.185a48.507 48.507 0 0 1 11.186 0Z" />
               {:else if tab.icon === 'matches'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 0 1 0 3.75H5.625a1.875 1.875 0 0 1 0-3.75Z" />
               {:else if tab.icon === 'tournaments'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M16.5 18.75h-9m9 0a3 3 0 0 1 3 3h-15a3 3 0 0 1 3-3m9 0v-3.375c0-.621-.503-1.125-1.125-1.125h-.871M7.5 18.75v-3.375c0-.621.504-1.125 1.125-1.125h.872m5.007 0H9.497m5.007 0a7.454 7.454 0 0 1-.982-3.172M9.497 14.25a7.454 7.454 0 0 0 .981-3.172M5.25 4.236c-.982.143-1.954.317-2.916.52A6.003 6.003 0 0 0 7.73 9.728M5.25 4.236V4.5c0 2.108.966 3.99 2.48 5.228M5.25 4.236V2.721C7.456 2.41 9.71 2.25 12 2.25c2.291 0 4.545.16 6.75.47v1.516M7.73 9.728a6.726 6.726 0 0 0 2.748 1.35m8.272-6.842V4.5c0 2.108-.966 3.99-2.48 5.228m2.48-5.492a46.32 46.32 0 0 1 2.916.52 6.003 6.003 0 0 1-5.395 4.972m0 0a6.726 6.726 0 0 1-2.749 1.35m0 0a6.772 6.772 0 0 1-3.044 0" />
               {:else if tab.icon === 'epc'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 15.75V18m-7.5-6.75h.008v.008H8.25v-.008Zm0 2.25h.008v.008H8.25V13.5Zm0 2.25h.008v.008H8.25v-.008Zm0 2.25h.008v.008H8.25V18Zm2.498-6.75h.007v.008h-.007v-.008Zm0 2.25h.007v.008h-.007V13.5Zm0 2.25h.007v.008h-.007v-.008Zm0 2.25h.007v.008h-.007V18Zm2.504-6.75h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V13.5Zm0 2.25h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V18Zm2.498-6.75h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V13.5ZM8.25 6h7.5v2.25h-7.5V6ZM12 2.25c-1.892 0-3.758.11-5.593.322C5.307 2.7 4.5 3.65 4.5 4.757V19.5a2.25 2.25 0 0 0 2.25 2.25h10.5a2.25 2.25 0 0 0 2.25-2.25V4.757c0-1.108-.806-2.057-1.907-2.185A48.507 48.507 0 0 0 12 2.25Z" />
               {:else if tab.icon === 'anki'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="M4.26 10.147a60.438 60.438 0 0 0-.491 6.347A48.62 48.62 0 0 1 12 20.904a48.62 48.62 0 0 1 8.232-4.41 60.46 60.46 0 0 0-.491-6.347m-15.482 0a50.636 50.636 0 0 0-2.658-.813A59.906 59.906 0 0 1 12 3.493a59.903 59.903 0 0 1 10.399 5.84c-.896.248-1.783.52-2.658.814m-15.482 0A50.717 50.717 0 0 1 12 13.489a50.702 50.702 0 0 1 7.74-3.342M6.75 15a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Zm0 0v-3.675A55.378 55.378 0 0 1 12 8.443m-7.007 11.55A5.981 5.981 0 0 0 6.75 15.75v-1.5" />
               {:else if tab.icon === 'metadata'}
                  <path stroke-linecap="round" stroke-linejoin="round" d="m11.25 11.25.041-.02a.75.75 0 0 1 1.063.852l-.708 2.836a.75.75 0 0 0 1.063.853l.041-.021M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9-3.75h.008v.008H12V8.25Z" />
               {/if}
            </svg>
            <span class="tab-label">{tab.label}</span>
         </button>
      {/each}
   </div>

   <div class="tab-content">
      {#if $activeTabStore === 'analysis'}
         <AnalysisPanel
            visible={true}
            onClose={onCloseAnalysis}
         />
      {:else if $activeTabStore === 'comments'}
         <CommentPanel
            visible={true}
            onClose={onCloseComment}
         />
      {:else if $activeTabStore === 'search'}
         <SearchPanel
            onLoadPositionsByFilters={onLoadPositionsByFilters}
            onAddToFilterLibrary={onAddToFilterLibrary}
         />
      {:else if $activeTabStore === 'collections'}
         <CollectionPanel onOpenCollection={onOpenCollection} />
      {:else if $activeTabStore === 'anki'}
         <AnkiPanel />
      {:else if $activeTabStore === 'matches'}
         <MatchPanel />
      {:else if $activeTabStore === 'tournaments'}
         <TournamentPanel />
      {:else if $activeTabStore === 'epc'}
         <EPCPanel />
      {:else if $activeTabStore === 'metadata'}
         <MetadataPanel />
      {:else if $activeTabStore === 'log'}
         <ConsolePanel />
      {/if}
   </div>
</div>

<style>
   .tabbed-panel {
      display: flex;
      flex-direction: column;
      height: 100%;
      min-height: 0;
      border-top: 1px solid #ccc;
      background: #fff;
   }

   .tab-bar {
      display: flex;
      align-items: center;
      background: #f0f0f0;
      border-bottom: 1px solid #ccc;
      overflow-x: auto;
      flex-shrink: 0;
      height: 28px;
      user-select: none;
      -webkit-user-select: none;
   }

   .tab-button {
      display: flex;
      align-items: center;
      gap: 4px;
      padding: 4px 10px;
      border: none;
      border-bottom: 2px solid transparent;
      background: transparent;
      cursor: pointer;
      font-size: 12px;
      color: #555;
      white-space: nowrap;
      height: 100%;
      transition: background-color 0.15s, border-color 0.15s;
      user-select: none;
      -webkit-user-select: none;
   }

   .tab-button.drag-over {
      border-left: 2px solid #1a73e8;
   }

   .tab-button.dragging {
      opacity: 0.5;
   }

   .tab-button:hover {
      background: #e0e0e0;
   }

   .tab-button.active {
      color: #1a73e8;
      border-bottom-color: #1a73e8;
      background: #fff;
   }

   .tab-icon {
      width: 14px;
      height: 14px;
      flex-shrink: 0;
   }

   .tab-label {
      font-size: 11px;
   }

   .tab-content {
      flex: 1;
      min-height: 0;
      overflow-y: auto;
      overflow-x: hidden;
   }
</style>
