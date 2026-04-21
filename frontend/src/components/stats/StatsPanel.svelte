<script>
    import { onMount, onDestroy } from 'svelte';
    import {
        statsFilterStore,
        statsResultStore,
        statsLoadingStore,
        statsMetricStore,
        refreshStats
    } from '../../stores/statsStore.js';
    import { activeTabStore } from '../../stores/uiStore.js';
    import StatsFilterBar from './StatsFilterBar.svelte';
    import StatsDashboardTab from './StatsDashboardTab.svelte';
    import StatsProgressionTab from './StatsProgressionTab.svelte';
    import StatsErrorsTab from './StatsErrorsTab.svelte';

    /** Currently active inner tab. */
    let activeTab = $state('dashboard');

    let unsubscribeFilter;

    onMount(() => {
        refreshStats($statsFilterStore);
        unsubscribeFilter = statsFilterStore.subscribe((filter) => {
            refreshStats(filter);
        });
    });

    onDestroy(() => {
        unsubscribeFilter?.();
    });

    function handleClose() {
        activeTabStore.set('analysis');
    }
</script>

<section class="stats-panel" role="region" aria-label="Stats">
        <header class="stats-header">
            <h2 class="stats-title">Stats</h2>
            <div class="metric-toggle" role="group" aria-label="Metric">
                <button
                    class="metric-btn"
                    class:active={$statsMetricStore === 'pr'}
                    onclick={() => statsMetricStore.set('pr')}
                    aria-pressed={$statsMetricStore === 'pr'}
                >PR</button>
                <button
                    class="metric-btn"
                    class:active={$statsMetricStore === 'mwc'}
                    onclick={() => statsMetricStore.set('mwc')}
                    aria-pressed={$statsMetricStore === 'mwc'}
                >MWC</button>
            </div>
            <button class="close-btn" onclick={handleClose} aria-label="Close stats panel">✕</button>
        </header>

        <StatsFilterBar />

        <nav class="tabs" role="tablist">
            <button
                class="tab-btn"
                class:active={activeTab === 'dashboard'}
                role="tab"
                aria-selected={activeTab === 'dashboard'}
                onclick={() => (activeTab = 'dashboard')}
            >Dashboard</button>
            <button
                class="tab-btn"
                class:active={activeTab === 'progression'}
                role="tab"
                aria-selected={activeTab === 'progression'}
                onclick={() => (activeTab = 'progression')}
            >Progression</button>
            <button
                class="tab-btn"
                class:active={activeTab === 'errors'}
                role="tab"
                aria-selected={activeTab === 'errors'}
                onclick={() => (activeTab = 'errors')}
            >Errors</button>
        </nav>

        <div class="tab-content" role="tabpanel">
            {#if $statsLoadingStore}
                <p class="loading-msg">Loading…</p>
            {:else if activeTab === 'dashboard'}
                <StatsDashboardTab result={$statsResultStore} metric={$statsMetricStore} />
            {:else if activeTab === 'progression'}
                <StatsProgressionTab result={$statsResultStore} metric={$statsMetricStore} />
            {:else if activeTab === 'errors'}
                <StatsErrorsTab result={$statsResultStore} metric={$statsMetricStore} />
            {/if}
        </div>
    </section>

<style>
    .stats-panel {
        display: flex;
        flex-direction: column;
        height: 100%;
        min-height: 0;
        background: #fff;
        font-size: 12px;
    }

    /* ── Header ── */
    .stats-header {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 4px 8px;
        border-bottom: 1px solid #e0e0e0;
        flex-shrink: 0;
        background: #fafafa;
    }

    .stats-title {
        margin: 0;
        font-size: 13px;
        font-weight: 600;
        color: #333;
        flex: 1;
    }

    /* ── PR / MWC toggle ── */
    .metric-toggle {
        display: flex;
        border: 1px solid #ccc;
        border-radius: 3px;
        overflow: hidden;
    }

    .metric-btn {
        background: none;
        border: none;
        padding: 2px 8px;
        font-size: 11px;
        cursor: pointer;
        color: #555;
        transition: background 0.1s;
    }

    .metric-btn:hover {
        background: #f0f0f0;
    }

    .metric-btn.active {
        background: #1976d2;
        color: #fff;
    }

    /* ── Close button ── */
    .close-btn {
        background: none;
        border: none;
        cursor: pointer;
        font-size: 13px;
        color: #999;
        padding: 2px 4px;
        line-height: 1;
        border-radius: 2px;
    }

    .close-btn:hover {
        color: #333;
        background: #f0f0f0;
    }

    /* ── Tab bar ── */
    .tabs {
        display: flex;
        border-bottom: 1px solid #e0e0e0;
        flex-shrink: 0;
        background: #fafafa;
    }

    .tab-btn {
        background: none;
        border: none;
        border-bottom: 2px solid transparent;
        padding: 5px 12px;
        font-size: 11px;
        cursor: pointer;
        color: #555;
        transition: border-color 0.1s, color 0.1s;
    }

    .tab-btn:hover {
        color: #1976d2;
    }

    .tab-btn.active {
        border-bottom-color: #1976d2;
        color: #1976d2;
        font-weight: 600;
    }

    /* ── Tab content ── */
    .tab-content {
        flex: 1;
        min-height: 0;
        overflow-y: auto;
    }

    .loading-msg {
        color: #999;
        font-size: 12px;
        text-align: center;
        padding: 24px;
    }
</style>
