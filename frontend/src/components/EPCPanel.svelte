<script>
    import { onMount } from 'svelte';
    import { statusBarModeStore, MODAL, openModal } from '../stores/uiStore';
    import { epcDataStore, epcChallengeStore, epcRevealedStore, resetEpcReveal } from '../stores/epcStore';
    import { GetEpcChallenge, SaveEpcChallenge } from '../../wailsjs/go/main/Config.js';
    import { t } from '../i18n';

    let isActive = $derived($statusBarModeStore === 'EPC');
    let data = $derived($epcDataStore);
    let challenge = $derived($epcChallengeStore);
    let revealed = $derived($epcRevealedStore);

    onMount(() => {
        GetEpcChallenge()
            .then((v) => epcChallengeStore.set(!!v))
            .catch(() => {});
    });

    function toggleChallenge(e) {
        const on = e.target.checked;
        epcChallengeStore.set(on);
        resetEpcReveal();
        SaveEpcChallenge(on).catch(() => {});
    }

    function reveal(zone) {
        epcRevealedStore.update((r) => ({ ...r, [zone]: true }));
    }

    const pct = (x) => (100 * x).toFixed(2);
    const eq = (x) => (x >= 0 ? '+' : '') + x.toFixed(3);
</script>

<div class="epc-panel">
    {#if !isActive}
        <div class="epc-inactive">
            <div class="epc-inactive-message">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="inactive-icon">
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        d="M15.75 15.75V18m-7.5-6.75h.008v.008H8.25v-.008Zm0 2.25h.008v.008H8.25V13.5Zm0 2.25h.008v.008H8.25v-.008Zm0 2.25h.008v.008H8.25V18Zm2.498-6.75h.007v.008h-.007v-.008Zm0 2.25h.007v.008h-.007V13.5Zm0 2.25h.007v.008h-.007v-.008Zm0 2.25h.007v.008h-.007V18Zm2.504-6.75h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V13.5Zm0 2.25h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V18Zm2.498-6.75h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V13.5ZM8.25 6h7.5v2.25h-7.5V6ZM12 2.25c-1.892 0-3.758.11-5.593.322C5.307 2.7 4.5 3.65 4.5 4.757V19.5a2.25 2.25 0 0 0 2.25 2.25h10.5a2.25 2.25 0 0 0 2.25-2.25V4.757c0-1.108-.806-2.057-1.907-2.185A48.507 48.507 0 0 0 12 2.25Z"
                    />
                </svg>
                <span>{$t('epc.inactive')}</span>
            </div>
        </div>
    {:else if data.error}
        <div class="epc-error">
            <span class="error-text">{data.error}</span>
        </div>
    {:else if !data.bottomEPC && !data.topEPC && !data.race}
        <div class="epc-inactive">
            <div class="epc-inactive-message">
                <span>{$t('epc.placeCheckers')}</span>
            </div>
        </div>
    {:else}
        <div class="epc-content">
            <div class="epc-toolbar">
                <label class="challenge-toggle" title={$t('epc.challengeTooltip')}>
                    <input type="checkbox" checked={challenge} onchange={toggleChallenge} />
                    <span>{$t('epc.challenge')}</span>
                </label>
            </div>

            <!-- Bottom player (Black) -->
            {#if data.bottomEPC}
                <div class="epc-player-section maskable">
                    {#if challenge && !revealed.bottom}
                        <button class="mask-overlay" onclick={() => reveal('bottom')}>{$t('epc.clickToReveal')}</button>
                    {/if}
                    <div class="epc-player-header">
                        <span class="player-indicator bottom"></span>
                        <span class="player-label">{$t('epc.bottomBlack')}</span>
                    </div>
                    <div class="epc-grid">
                        <div class="epc-card epc-main">
                            <div class="epc-card-label">{$t('epc.epc')}</div>
                            <div class="epc-card-value">{data.bottomEPC.epc.toFixed(2)}</div>
                        </div>
                        <div class="epc-card">
                            <div class="epc-card-label">{$t('epc.pipCount')}</div>
                            <div class="epc-card-value">{data.bottomEPC.pipCount}</div>
                        </div>
                        <div class="epc-card">
                            <div class="epc-card-label">{$t('epc.wastage')}</div>
                            <div class="epc-card-value">{data.bottomEPC.wastage.toFixed(2)}</div>
                        </div>
                        <div class="epc-card">
                            <div class="epc-card-label">{$t('epc.avgRolls')}</div>
                            <div class="epc-card-value">{data.bottomEPC.meanRolls.toFixed(3)}</div>
                        </div>
                        <div class="epc-card">
                            <div class="epc-card-label">{$t('epc.stdDev')}</div>
                            <div class="epc-card-value">{data.bottomEPC.stdDev.toFixed(3)}</div>
                        </div>
                    </div>
                </div>
            {/if}

            <!-- Top player (White) -->
            {#if data.topEPC}
                <div class="epc-player-section maskable">
                    {#if challenge && !revealed.top}
                        <button class="mask-overlay" onclick={() => reveal('top')}>{$t('epc.clickToReveal')}</button>
                    {/if}
                    <div class="epc-player-header">
                        <span class="player-indicator top"></span>
                        <span class="player-label">{$t('epc.topWhite')}</span>
                    </div>
                    <div class="epc-grid">
                        <div class="epc-card epc-main">
                            <div class="epc-card-label">{$t('epc.epc')}</div>
                            <div class="epc-card-value">{data.topEPC.epc.toFixed(2)}</div>
                        </div>
                        <div class="epc-card">
                            <div class="epc-card-label">{$t('epc.pipCount')}</div>
                            <div class="epc-card-value">{data.topEPC.pipCount}</div>
                        </div>
                        <div class="epc-card">
                            <div class="epc-card-label">{$t('epc.wastage')}</div>
                            <div class="epc-card-value">{data.topEPC.wastage.toFixed(2)}</div>
                        </div>
                        <div class="epc-card">
                            <div class="epc-card-label">{$t('epc.avgRolls')}</div>
                            <div class="epc-card-value">{data.topEPC.meanRolls.toFixed(3)}</div>
                        </div>
                        <div class="epc-card">
                            <div class="epc-card-label">{$t('epc.stdDev')}</div>
                            <div class="epc-card-value">{data.topEPC.stdDev.toFixed(3)}</div>
                        </div>
                    </div>
                </div>
            {/if}

            <!-- Race zone: win probability + money cube (pure bearoff only) -->
            {#if data.race}
                <div class="epc-player-section epc-race maskable">
                    {#if challenge && !revealed.race}
                        <button class="mask-overlay" onclick={() => reveal('race')}>{$t('epc.clickToReveal')}</button>
                    {/if}
                    <div class="epc-player-header">
                        <span class="player-indicator" class:bottom={data.race.on_roll === 0} class:top={data.race.on_roll === 1}></span>
                        <span class="player-label">{$t('epc.race.title')}</span>
                        {#if data.race.regime === 'exact'}
                            <span class="badge badge-exact" title={$t('epc.race.exactTooltip', { n: data.race.source_checkers })}>
                                {$t('epc.race.exact')} · TS-06-{String(data.race.source_checkers).padStart(2, '0')}
                            </span>
                        {:else}
                            <span class="badge badge-estimated" title={$t('epc.race.estimatedTooltip', { p99: pct(data.race.p99) })}>
                                {$t('epc.race.estimated')} ± {pct(data.race.sigma)} %
                            </span>
                        {/if}
                    </div>
                    <div class="epc-grid">
                        <div class="epc-card epc-main">
                            <div class="epc-card-label">{$t('epc.race.winProb')}</div>
                            <div class="epc-card-value">{pct(data.race.win_prob)} %</div>
                        </div>
                        {#if data.race.money}
                            <div class="epc-card">
                                <div class="epc-card-label">{$t('epc.race.cubeless')}</div>
                                <div class="epc-card-value">{eq(data.race.money.cubeless)}</div>
                            </div>
                            <div class="epc-card">
                                <div class="epc-card-label">{$t('epc.race.noDouble')}</div>
                                <div class="epc-card-value">{eq(data.race.money.no_double)}</div>
                            </div>
                            <div class="epc-card">
                                <div class="epc-card-label">{$t('epc.race.doubleTake')}</div>
                                <div class="epc-card-value">{eq(data.race.money.double_take)}</div>
                            </div>
                            <div class="epc-card">
                                <div class="epc-card-label">{$t('epc.race.doublePass')}</div>
                                <div class="epc-card-value">{eq(data.race.money.double_pass)}</div>
                            </div>
                        {/if}
                    </div>
                    {#if data.race.money}
                        <div class="verdict-row">
                            <span class="verdict-label">{$t('epc.race.verdictLabel')} ({$t('epc.race.cubeStates.' + data.race.money.cube_state)}) :</span>
                            {#if data.race.money.verdict}
                                <span class="verdict-chip">{$t('epc.race.verdicts.' + data.race.money.verdict)}</span>
                            {:else}
                                <span class="verdict-chip verdict-none">{$t('epc.race.noDecision')}</span>
                            {/if}
                        </div>
                    {:else}
                        <div class="download-hint">
                            {$t('epc.race.downloadHint')}
                            <button class="link-button" onclick={() => openModal(MODAL.CONFIG)}>{$t('epc.race.openConfig')}</button>
                        </div>
                    {/if}
                </div>
            {/if}

            <!-- Comparison section when both players have data -->
            {#if data.bottomEPC && data.topEPC && (!challenge || (revealed.bottom && revealed.top))}
                <div class="epc-comparison">
                    <div class="epc-comparison-header">{$t('epc.comparison')}</div>
                    <div class="epc-comparison-grid">
                        <div class="epc-comp-item">
                            <span class="comp-label">{$t('epc.epcDiff')}</span>
                            <span class="comp-value" class:advantage-bottom={data.bottomEPC.epc < data.topEPC.epc} class:advantage-top={data.topEPC.epc < data.bottomEPC.epc}>
                                {Math.abs(data.bottomEPC.epc - data.topEPC.epc).toFixed(2)}
                            </span>
                        </div>
                        <div class="epc-comp-item">
                            <span class="comp-label">{$t('epc.pipDiff')}</span>
                            <span class="comp-value" class:advantage-bottom={data.bottomEPC.pipCount < data.topEPC.pipCount} class:advantage-top={data.topEPC.pipCount < data.bottomEPC.pipCount}>
                                {Math.abs(data.bottomEPC.pipCount - data.topEPC.pipCount)}
                            </span>
                        </div>
                        <div class="epc-comp-item">
                            <span class="comp-label">{$t('epc.wastageDiff')}</span>
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
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans JP', sans-serif;
        font-size: var(--font-size-base);
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
        font-size: var(--font-size-base);
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
        font-size: var(--font-size-base);
    }

    .epc-content {
        display: flex;
        flex-direction: column;
        gap: 10px;
    }

    .epc-toolbar {
        display: flex;
        justify-content: flex-end;
    }

    .challenge-toggle {
        display: flex;
        align-items: center;
        gap: 5px;
        cursor: pointer;
        color: #555;
        font-size: var(--font-size-small);
        user-select: none;
    }

    .challenge-toggle input {
        font: inherit;
        margin: 0;
    }

    .epc-player-section {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }

    .maskable {
        position: relative;
    }

    .mask-overlay {
        position: absolute;
        inset: 0;
        z-index: 2;
        display: flex;
        align-items: center;
        justify-content: center;
        background: #f2f2f2;
        border: 1px dashed #bbb;
        border-radius: 4px;
        color: #666;
        font: inherit;
        font-size: var(--font-size-base);
        cursor: pointer;
    }

    .mask-overlay:hover {
        background: #ececec;
        color: #333;
    }

    .epc-player-header {
        display: flex;
        align-items: center;
        gap: 6px;
        font-weight: 600;
        font-size: var(--font-size-small);
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

    .badge {
        margin-left: auto;
        padding: 1px 8px;
        border-radius: 9px;
        font-size: var(--font-size-small);
        font-weight: 600;
        letter-spacing: 0.3px;
        text-transform: none;
    }

    .badge-exact {
        background: #e5f3e8;
        border: 1px solid #bcdcc4;
        color: #1e6b34;
    }

    .badge-estimated {
        background: #fdf3e1;
        border: 1px solid #ecd7a8;
        color: #8a6413;
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
        font-size: var(--font-size-small);
        color: #777;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        margin-bottom: 1px;
    }

    .epc-card-value {
        font-size: var(--font-size-base);
        font-weight: 600;
        color: #222;
        font-variant-numeric: tabular-nums;
    }

    .epc-main .epc-card-value {
        font-size: var(--font-size-title);
        color: #1a56c4;
    }

    .verdict-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .verdict-label {
        font-size: var(--font-size-small);
        color: #666;
    }

    .verdict-chip {
        padding: 1px 10px;
        border-radius: 9px;
        background: #e8f0fe;
        border: 1px solid #c4d8f5;
        color: #1a56c4;
        font-size: var(--font-size-base);
        font-weight: 600;
    }

    .verdict-chip.verdict-none {
        background: #f2f2f2;
        border-color: #ddd;
        color: #777;
        font-weight: 400;
    }

    .download-hint {
        font-size: var(--font-size-small);
        color: #8a6413;
    }

    .link-button {
        background: none;
        border: none;
        padding: 0;
        font: inherit;
        color: #1a56c4;
        text-decoration: underline;
        cursor: pointer;
    }

    .epc-comparison {
        border-top: 1px solid #e0e0e0;
        padding-top: 6px;
    }

    .epc-comparison-header {
        font-size: var(--font-size-small);
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
        font-size: var(--font-size-small);
        color: #777;
    }

    .comp-value {
        font-size: var(--font-size-base);
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
