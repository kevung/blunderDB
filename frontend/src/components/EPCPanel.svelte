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

    // Défi mode: values are replaced by a placeholder until their zone is
    // revealed; clicking a masked row/table reveals it.
    let maskedBottom = $derived(challenge && !revealed.bottom);
    let maskedTop = $derived(challenge && !revealed.top);
    let maskedRace = $derived(challenge && !revealed.race);

    const HIDDEN = '···';
    const show = (masked, v) => (masked ? HIDDEN : v);
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
            <div class="tables-container">
                <!-- Players × quantities, Analysis-panel table style. The Δ row
                     absorbs the old comparison section. -->
                {#if data.bottomEPC || data.topEPC}
                    <table class="players-table">
                        <tbody>
                            <tr>
                                <th></th>
                                <th>{$t('epc.epc')}</th>
                                <th>{$t('epc.pipCount')}</th>
                                <th>{$t('epc.wastage')}</th>
                                <th>{$t('epc.avgRolls')}</th>
                                <th>{$t('epc.stdDev')}</th>
                            </tr>
                            {#if data.bottomEPC}
                                <tr class:masked={maskedBottom} onclick={() => maskedBottom && reveal('bottom')} title={maskedBottom ? $t('epc.clickToReveal') : undefined}>
                                    <td class="row-label"><span class="player-indicator bottom"></span>{$t('epc.bottomBlack')}</td>
                                    <td class="main-value">{show(maskedBottom, data.bottomEPC.epc.toFixed(2))}</td>
                                    <td>{show(maskedBottom, data.bottomEPC.pipCount)}</td>
                                    <td>{show(maskedBottom, data.bottomEPC.wastage.toFixed(2))}</td>
                                    <td>{show(maskedBottom, data.bottomEPC.meanRolls.toFixed(3))}</td>
                                    <td>{show(maskedBottom, data.bottomEPC.stdDev.toFixed(3))}</td>
                                </tr>
                            {/if}
                            {#if data.topEPC}
                                <tr class:masked={maskedTop} onclick={() => maskedTop && reveal('top')} title={maskedTop ? $t('epc.clickToReveal') : undefined}>
                                    <td class="row-label"><span class="player-indicator top"></span>{$t('epc.topWhite')}</td>
                                    <td class="main-value">{show(maskedTop, data.topEPC.epc.toFixed(2))}</td>
                                    <td>{show(maskedTop, data.topEPC.pipCount)}</td>
                                    <td>{show(maskedTop, data.topEPC.wastage.toFixed(2))}</td>
                                    <td>{show(maskedTop, data.topEPC.meanRolls.toFixed(3))}</td>
                                    <td>{show(maskedTop, data.topEPC.stdDev.toFixed(3))}</td>
                                </tr>
                            {/if}
                            {#if data.bottomEPC && data.topEPC}
                                {@const bothShown = !maskedBottom && !maskedTop}
                                <tr class="delta-row">
                                    <td class="row-label" title={$t('epc.comparison')}>Δ</td>
                                    <td class="main-value">{show(!bothShown, Math.abs(data.bottomEPC.epc - data.topEPC.epc).toFixed(2))}</td>
                                    <td>{show(!bothShown, Math.abs(data.bottomEPC.pipCount - data.topEPC.pipCount))}</td>
                                    <td>{show(!bothShown, Math.abs(data.bottomEPC.wastage - data.topEPC.wastage).toFixed(2))}</td>
                                    <td>—</td>
                                    <td>—</td>
                                </tr>
                            {/if}
                        </tbody>
                    </table>
                {/if}

                <!-- Race / cube table, mirroring the Analysis panel's cube
                     decision table (decision | equity, verdict as best action). -->
                {#if data.race}
                    <table class="race-table" class:masked={maskedRace} onclick={() => maskedRace && reveal('race')} title={maskedRace ? $t('epc.clickToReveal') : undefined}>
                        <tbody>
                            <tr>
                                <th class="race-title" colspan="2">
                                    <span class="player-indicator" class:bottom={data.race.on_roll === 0} class:top={data.race.on_roll === 1}></span>
                                    {$t('epc.race.title')}
                                    {#if data.race.regime === 'exact'}
                                        <span class="badge badge-exact" title={$t('epc.race.exactTooltip', { n: data.race.source_checkers })}>
                                            {$t('epc.race.exact')} · TS-06-{String(data.race.source_checkers).padStart(2, '0')}
                                        </span>
                                    {:else}
                                        <span class="badge badge-estimated" title={$t('epc.race.estimatedTooltip', { p99: pct(data.race.p99) })}>
                                            {$t('epc.race.estimated')} ± {pct(data.race.sigma)} %
                                        </span>
                                    {/if}
                                </th>
                            </tr>
                            <tr>
                                <td>{$t('epc.race.winProb')}</td>
                                <td class="main-value">{show(maskedRace, pct(data.race.win_prob) + ' %')}</td>
                            </tr>
                            {#if data.race.money}
                                <tr>
                                    <td>{$t('epc.race.cubeless')}</td>
                                    <td>{show(maskedRace, eq(data.race.money.cubeless))}</td>
                                </tr>
                                <tr>
                                    <td>{$t('epc.race.noDouble')}</td>
                                    <td>{show(maskedRace, eq(data.race.money.no_double))}</td>
                                </tr>
                                <tr>
                                    <td>{$t('epc.race.doubleTake')}</td>
                                    <td>{show(maskedRace, eq(data.race.money.double_take))}</td>
                                </tr>
                                <tr>
                                    <td>{$t('epc.race.doublePass')}</td>
                                    <td>{show(maskedRace, eq(data.race.money.double_pass))}</td>
                                </tr>
                                <tr class="best-action-row">
                                    <td>{$t('epc.race.verdictLabel')} ({$t('epc.race.cubeStates.' + data.race.money.cube_state)})</td>
                                    <td>
                                        {#if maskedRace}
                                            {HIDDEN}
                                        {:else if data.race.money.verdict}
                                            {$t('epc.race.verdicts.' + data.race.money.verdict)}
                                        {:else}
                                            {$t('epc.race.noDecision')}
                                        {/if}
                                    </td>
                                </tr>
                            {:else}
                                <tr>
                                    <td colspan="2" class="download-hint">
                                        {$t('epc.race.downloadHint')}
                                        <button class="link-button" onclick={() => openModal(MODAL.CONFIG)}>{$t('epc.race.openConfig')}</button>
                                    </td>
                                </tr>
                            {/if}
                        </tbody>
                    </table>
                {/if}

                <label class="challenge-toggle" title={$t('epc.challengeTooltip')}>
                    <input type="checkbox" checked={challenge} onchange={toggleChallenge} />
                    <span>{$t('epc.challenge')}</span>
                </label>
            </div>
        </div>
    {/if}
</div>

<style>
    .epc-panel {
        height: 100%;
        overflow-y: auto;
        overflow-x: hidden;
        padding: 4px 12px;
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
    }

    /* Same idiom as the Analysis panel: tables side by side, wrapping when
       narrow. The défi toggle rides in the same row, top-right. */
    .tables-container {
        display: flex;
        flex-wrap: wrap;
        gap: 4px 24px;
        align-items: flex-start;
    }

    table {
        border-collapse: collapse;
        font-size: var(--font-size-base);
    }

    .players-table {
        flex: 0 1 auto;
    }

    .race-table {
        flex: 0 1 auto;
        min-width: 230px;
    }

    th,
    td {
        padding: 1px 8px;
        text-align: center;
        white-space: nowrap;
    }

    th {
        font-size: var(--font-size-small);
        color: #777;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        font-weight: 600;
    }

    tbody tr + tr td {
        border-top: 1px solid #eee;
    }

    .row-label {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: var(--font-size-small);
        font-weight: 600;
        color: #444;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        text-align: left;
    }

    .race-table td:first-child {
        font-size: var(--font-size-small);
        color: #666;
        text-align: left;
    }

    .race-title {
        text-align: left;
    }

    .race-title .badge {
        margin-left: 8px;
    }

    .main-value {
        font-weight: 600;
        color: #1a56c4;
        font-variant-numeric: tabular-nums;
    }

    td {
        font-variant-numeric: tabular-nums;
        color: #222;
    }

    .delta-row td {
        color: #666;
    }

    .delta-row .main-value {
        color: #666;
    }

    .best-action-row td {
        font-weight: 600;
        color: #1a56c4;
        border-top: 1px solid #ddd;
    }

    tr.masked,
    table.masked {
        cursor: pointer;
    }

    tr.masked td:not(.row-label),
    table.masked td:not(:first-child) {
        color: #aaa;
        letter-spacing: 2px;
    }

    tr.masked:hover td,
    table.masked:hover td {
        background: #f5f5f5;
    }

    .player-indicator {
        display: inline-block;
        width: 10px;
        height: 10px;
        border-radius: 50%;
        flex-shrink: 0;
        vertical-align: middle;
    }

    .race-title .player-indicator {
        margin-right: 4px;
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
        padding: 0 8px;
        border-radius: 9px;
        font-size: var(--font-size-small);
        font-weight: 600;
        letter-spacing: 0.3px;
        text-transform: none;
        white-space: nowrap;
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

    .download-hint {
        font-size: var(--font-size-small);
        color: #8a6413;
        white-space: normal;
        max-width: 340px;
        text-align: left;
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

    .challenge-toggle {
        margin-left: auto;
        display: flex;
        align-items: center;
        gap: 5px;
        cursor: pointer;
        color: #555;
        font-size: var(--font-size-small);
        user-select: none;
        white-space: nowrap;
    }

    .challenge-toggle input {
        font: inherit;
        margin: 0;
    }
</style>
