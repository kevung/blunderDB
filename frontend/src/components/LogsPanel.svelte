<script>
   import { logEntriesStore } from '../stores/uiStore';
   import { onMount, tick } from 'svelte';

   let logContainer;
   let entries = [];

   logEntriesStore.subscribe(async value => {
      entries = value;
      await tick();
      if (logContainer) {
         logContainer.scrollTop = logContainer.scrollHeight;
      }
   });
</script>

<div class="logs-panel">
   <div class="logs-output" bind:this={logContainer}>
      {#if entries.length === 0}
         <div class="empty-msg">No operations logged yet.</div>
      {:else}
         {#each entries as entry}
            <div class="log-line">
               <span class="log-time">{entry.timestamp.toLocaleTimeString('sv-SE', { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
               <span class="log-msg">{entry.message}</span>
            </div>
         {/each}
      {/if}
   </div>
</div>

<style>
   .logs-panel {
      display: flex;
      flex-direction: column;
      height: 100%;
      font-size: 13px;
      background: #fff;
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
      font-family: 'Consolas', 'Monaco', monospace;
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
