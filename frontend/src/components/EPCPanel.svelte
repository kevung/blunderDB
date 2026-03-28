<script>
   import { statusBarModeStore } from '../stores/uiStore';
   import { epcDataStore } from '../stores/epcStore';

   $: isActive = $statusBarModeStore === 'EPC';
   $: data = $epcDataStore;
</script>

<div class="epc-panel">
   {#if !isActive}
      <div class="epc-inactive">
         <div class="epc-inactive-message">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="inactive-icon">
               <path stroke-linecap="round" stroke-linejoin="round" d="M15.75 15.75V18m-7.5-6.75h.008v.008H8.25v-.008Zm0 2.25h.008v.008H8.25V13.5Zm0 2.25h.008v.008H8.25v-.008Zm0 2.25h.008v.008H8.25V18Zm2.498-6.75h.007v.008h-.007v-.008Zm0 2.25h.007v.008h-.007V13.5Zm0 2.25h.007v.008h-.007v-.008Zm0 2.25h.007v.008h-.007V18Zm2.504-6.75h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V13.5Zm0 2.25h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V18Zm2.498-6.75h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V13.5ZM8.25 6h7.5v2.25h-7.5V6ZM12 2.25c-1.892 0-3.758.11-5.593.322C5.307 2.7 4.5 3.65 4.5 4.757V19.5a2.25 2.25 0 0 0 2.25 2.25h10.5a2.25 2.25 0 0 0 2.25-2.25V4.757c0-1.108-.806-2.057-1.907-2.185A48.507 48.507 0 0 0 12 2.25Z" />
            </svg>
            <span>EPC calculator is inactive. Edit the board to compute EPC values.</span>
         </div>
      </div>
   {:else if data.error}
      <div class="epc-error">
         <span class="error-text">{data.error}</span>
      </div>
   {:else if !data.bottomEPC && !data.topEPC}
      <div class="epc-inactive">
         <div class="epc-inactive-message">
            <span>Place checkers on the home board to compute EPC.</span>
         </div>
      </div>
   {:else}
      <div class="epc-content">
         <!-- Bottom player (Black) -->
         {#if data.bottomEPC}
         <div class="epc-player-section">
            <div class="epc-player-header">
               <span class="player-indicator bottom"></span>
               <span class="player-label">Bottom (Black)</span>
            </div>
            <div class="epc-grid">
               <div class="epc-card epc-main">
                  <div class="epc-card-label">EPC</div>
                  <div class="epc-card-value">{data.bottomEPC.epc.toFixed(2)}</div>
               </div>
               <div class="epc-card">
                  <div class="epc-card-label">Pip Count</div>
                  <div class="epc-card-value">{data.bottomEPC.pipCount}</div>
               </div>
               <div class="epc-card">
                  <div class="epc-card-label">Wastage</div>
                  <div class="epc-card-value">{data.bottomEPC.wastage.toFixed(2)}</div>
               </div>
               <div class="epc-card">
                  <div class="epc-card-label">Avg Rolls</div>
                  <div class="epc-card-value">{data.bottomEPC.meanRolls.toFixed(3)}</div>
               </div>
               <div class="epc-card">
                  <div class="epc-card-label">Std Dev</div>
                  <div class="epc-card-value">{data.bottomEPC.stdDev.toFixed(3)}</div>
               </div>
            </div>
         </div>
         {/if}

         <!-- Top player (White) -->
         {#if data.topEPC}
         <div class="epc-player-section">
            <div class="epc-player-header">
               <span class="player-indicator top"></span>
               <span class="player-label">Top (White)</span>
            </div>
            <div class="epc-grid">
               <div class="epc-card epc-main">
                  <div class="epc-card-label">EPC</div>
                  <div class="epc-card-value">{data.topEPC.epc.toFixed(2)}</div>
               </div>
               <div class="epc-card">
                  <div class="epc-card-label">Pip Count</div>
                  <div class="epc-card-value">{data.topEPC.pipCount}</div>
               </div>
               <div class="epc-card">
                  <div class="epc-card-label">Wastage</div>
                  <div class="epc-card-value">{data.topEPC.wastage.toFixed(2)}</div>
               </div>
               <div class="epc-card">
                  <div class="epc-card-label">Avg Rolls</div>
                  <div class="epc-card-value">{data.topEPC.meanRolls.toFixed(3)}</div>
               </div>
               <div class="epc-card">
                  <div class="epc-card-label">Std Dev</div>
                  <div class="epc-card-value">{data.topEPC.stdDev.toFixed(3)}</div>
               </div>
            </div>
         </div>
         {/if}

         <!-- Comparison section when both players have data -->
         {#if data.bottomEPC && data.topEPC}
         <div class="epc-comparison">
            <div class="epc-comparison-header">Comparison</div>
            <div class="epc-comparison-grid">
               <div class="epc-comp-item">
                  <span class="comp-label">EPC Diff</span>
                  <span class="comp-value" class:advantage-bottom={data.bottomEPC.epc < data.topEPC.epc} class:advantage-top={data.topEPC.epc < data.bottomEPC.epc}>
                     {Math.abs(data.bottomEPC.epc - data.topEPC.epc).toFixed(2)}
                  </span>
               </div>
               <div class="epc-comp-item">
                  <span class="comp-label">Pip Diff</span>
                  <span class="comp-value" class:advantage-bottom={data.bottomEPC.pipCount < data.topEPC.pipCount} class:advantage-top={data.topEPC.pipCount < data.bottomEPC.pipCount}>
                     {Math.abs(data.bottomEPC.pipCount - data.topEPC.pipCount)}
                  </span>
               </div>
               <div class="epc-comp-item">
                  <span class="comp-label">Wastage Diff</span>
                  <span class="comp-value">
                     {Math.abs(data.bottomEPC.wastage - data.topEPC.wastage).toFixed(2)}
                  </span>
               </div>
            </div>
         </div>
         {/if}
      </div>
   {/if}
</div>

<style>
   .epc-panel {
      height: 100%;
      overflow-y: auto;
      padding: 8px 12px;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      font-size: 12px;
   }

   .epc-inactive {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      color: #888;
   }

   .epc-inactive-message {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
   }

   .inactive-icon {
      width: 18px;
      height: 18px;
      flex-shrink: 0;
   }

   .epc-error {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
   }

   .error-text {
      color: #c62828;
      font-size: 12px;
   }

   .epc-content {
      display: flex;
      flex-direction: column;
      gap: 10px;
   }

   .epc-player-section {
      display: flex;
      flex-direction: column;
      gap: 4px;
   }

   .epc-player-header {
      display: flex;
      align-items: center;
      gap: 6px;
      font-weight: 600;
      font-size: 11px;
      color: #444;
      text-transform: uppercase;
      letter-spacing: 0.5px;
   }

   .player-indicator {
      width: 10px;
      height: 10px;
      border-radius: 50%;
      flex-shrink: 0;
   }

   .player-indicator.bottom {
      background: #333;
      border: 1px solid #555;
   }

   .player-indicator.top {
      background: #fff;
      border: 1px solid #999;
   }

   .epc-grid {
      display: flex;
      gap: 6px;
      flex-wrap: wrap;
   }

   .epc-card {
      background: #f8f8f8;
      border: 1px solid #e0e0e0;
      border-radius: 4px;
      padding: 4px 10px;
      min-width: 70px;
      text-align: center;
   }

   .epc-card.epc-main {
      background: #e8f0fe;
      border-color: #c4d8f5;
   }

   .epc-card-label {
      font-size: 10px;
      color: #777;
      text-transform: uppercase;
      letter-spacing: 0.3px;
      margin-bottom: 1px;
   }

   .epc-card-value {
      font-size: 14px;
      font-weight: 600;
      color: #222;
      font-variant-numeric: tabular-nums;
   }

   .epc-main .epc-card-value {
      font-size: 16px;
      color: #1a56c4;
   }

   .epc-comparison {
      border-top: 1px solid #e0e0e0;
      padding-top: 6px;
   }

   .epc-comparison-header {
      font-size: 10px;
      font-weight: 600;
      color: #666;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      margin-bottom: 4px;
   }

   .epc-comparison-grid {
      display: flex;
      gap: 12px;
   }

   .epc-comp-item {
      display: flex;
      align-items: center;
      gap: 6px;
   }

   .comp-label {
      font-size: 11px;
      color: #777;
   }

   .comp-value {
      font-size: 13px;
      font-weight: 600;
      font-variant-numeric: tabular-nums;
      color: #444;
   }

   .comp-value.advantage-bottom {
      color: #333;
   }

   .comp-value.advantage-top {
      color: #888;
   }
</style>
