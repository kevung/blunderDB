<script>
    import { onMount } from 'svelte';
    import { statusBarModeStore, MODAL, openModal, configInitialTabStore } from '../stores/uiStore';
    import { epcDataStore, epcChallengeStore, epcRevealedStore, resetEpcReveal } from '../stores/epcStore';
    import { positionStore } from '../stores/positionStore';
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

    // The race analysis is computed for the position's on-roll player and
    // cube owner; both are editable straight from the race table header. The
    // position store is the single source of truth: changing it re-triggers
    // updateEPC (and re-masks the défi zones, consistently with board edits).
    let onRoll = $derived($positionStore?.player_on_roll ?? 0);
    let cubeState = $derived.by(() => {
        const owner = $positionStore?.cube?.owner ?? -1;
        if (owner === -1) return 'centered';
        return owner === onRoll ? 'owned' : 'against';
    });

    function setOnRoll(p) {
        positionStore.update((pos) => {
            // Keep the cube on the same side relative to the new on-roll player
            // is NOT what we want: the owner is absolute; just flip the roller.
            pos.player_on_roll = p;
            return pos;
        });
    }

    function setCubeState(state) {
        positionStore.update((pos) => {
            const or = pos.player_on_roll ?? 0;
            pos.cube.owner = state === 'centered' ? -1 : state === 'owned' ? or : 1 - or;
            return pos;
        });
    }

    function openBearoffSettings() {
        configInitialTabStore.set('bearoff');
        openModal(MODAL.CONFIG);
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
    // Signed difference bottom − top: negative when Black leads the race.
    const sd = (x, digits) => (x >= 0 ? '+' : '') + x.toFixed(digits);
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
        <label class="challenge-toggle" title={$t('epc.challengeTooltip')}>
            <input type="checkbox" checked={challenge} onchange={toggleChallenge} />
            <span>{$t('epc.challenge')}</span>
        </label>
        <div class="tables-container">
            <!-- Players × quantities, Analysis-panel table style. Black is
                 always the bottom player, so a colour dot suffices as the row
                 label; the Δ row is signed (bottom − top: negative when Black
                 leads) and absorbs the old comparison section. -->
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
                            <tr class:masked={maskedBottom} onclick={() => maskedBottom && reveal('bottom')} title={maskedBottom ? $t('epc.clickToReveal') : $t('epc.bottomBlack')}>
                                <td class="row-label"><span class="player-indicator bottom"></span></td>
                                <td class="main-value">{show(maskedBottom, data.bottomEPC.epc.toFixed(2))}</td>
                                <td>{show(maskedBottom, data.bottomEPC.pipCount)}</td>
                                <td>{show(maskedBottom, data.bottomEPC.wastage.toFixed(2))}</td>
                                <td>{show(maskedBottom, data.bottomEPC.meanRolls.toFixed(3))}</td>
                                <td>{show(maskedBottom, data.bottomEPC.stdDev.toFixed(3))}</td>
                            </tr>
                        {/if}
                        {#if data.topEPC}
                            <tr class:masked={maskedTop} onclick={() => maskedTop && reveal('top')} title={maskedTop ? $t('epc.clickToReveal') : $t('epc.topWhite')}>
                                <td class="row-label"><span class="player-indicator top"></span></td>
                                <td class="main-value">{show(maskedTop, data.topEPC.epc.toFixed(2))}</td>
                                <td>{show(maskedTop, data.topEPC.pipCount)}</td>
                                <td>{show(maskedTop, data.topEPC.wastage.toFixed(2))}</td>
                                <td>{show(maskedTop, data.topEPC.meanRolls.toFixed(3))}</td>
                                <td>{show(maskedTop, data.topEPC.stdDev.toFixed(3))}</td>
                            </tr>
                        {/if}
                        {#if data.bottomEPC && data.topEPC}
                            {@const bothShown = !maskedBottom && !maskedTop}
                            <tr class="delta-row" title={$t('epc.comparison')}>
                                <td class="row-label">Δ</td>
                                <td class="main-value">{show(!bothShown, sd(data.bottomEPC.epc - data.topEPC.epc, 2))}</td>
                                <td>{show(!bothShown, sd(data.bottomEPC.pipCount - data.topEPC.pipCount, 0))}</td>
                                <td>{show(!bothShown, sd(data.bottomEPC.wastage - data.topEPC.wastage, 2))}</td>
                                <td>—</td>
                                <td>—</td>
                            </tr>
                        {/if}
                    </tbody>
                </table>
            {/if}

            <!-- Race / cube table, transposed like the players table so both
                 tables have the same (minimal) height: quantities in columns,
                 verdict as the Analysis panel's best-action row. -->
            {#if data.race}
                <table class="race-table" class:masked={maskedRace} onclick={() => maskedRace && reveal('race')} title={maskedRace ? $t('epc.clickToReveal') : undefined}>
                    <tbody>
                        <tr>
                            <th class="race-title" colspan={data.race.money ? 5 : 2}>
                                <span class="race-title-inner">
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
                                    <span class="race-controls" role="group" aria-label={$t('epc.race.title')} onclick={(e) => e.stopPropagation()}>
                                        <span class="ctl-label">{$t('epc.race.onRoll')}</span>
                                        <button class="dot-btn" class:active={onRoll === 0} onclick={() => setOnRoll(0)} title={$t('epc.bottomBlack')} aria-pressed={onRoll === 0}>
                                            <span class="player-indicator bottom"></span>
                                        </button>
                                        <button class="dot-btn" class:active={onRoll === 1} onclick={() => setOnRoll(1)} title={$t('epc.topWhite')} aria-pressed={onRoll === 1}>
                                            <span class="player-indicator top"></span>
                                        </button>
                                        <span class="ctl-label">{$t('epc.race.cube')}</span>
                                        <select class="cube-select" value={cubeState} aria-label={$t('epc.race.cube')} onchange={(e) => setCubeState(e.target.value)}>
                                            <option value="centered">{$t('epc.race.cubeStates.centered')}</option>
                                            <option value="owned">{$t('epc.race.cubeStates.owned')}</option>
                                            <option value="against">{$t('epc.race.cubeStates.against')}</option>
                                        </select>
                                    </span>
                                </span>
                            </th>
                        </tr>
                        {#if data.race.money}
                            <tr>
                                <th>{$t('epc.race.winProb')}</th>
                                <th>{$t('epc.race.cubeless')}</th>
                                <th>{$t('epc.race.noDouble')}</th>
                                <th>{$t('epc.race.doubleTake')}</th>
                                <th>{$t('epc.race.doublePass')}</th>
                            </tr>
                            <tr>
                                <td class="main-value">{show(maskedRace, pct(data.race.win_prob) + ' %')}</td>
                                <td>{show(maskedRace, eq(data.race.money.cubeless))}</td>
                                <td>{show(maskedRace, eq(data.race.money.no_double))}</td>
                                <td>{show(maskedRace, eq(data.race.money.double_take))}</td>
                                <td>{show(maskedRace, eq(data.race.money.double_pass))}</td>
                            </tr>
                            <tr class="best-action-row">
                                <td colspan="5">
                                    {$t('epc.race.verdictLabel')} ({$t('epc.race.cubeStates.' + data.race.money.cube_state)}) :
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
                                <th>{$t('epc.race.winProb')}</th>
                                <td class="main-value">{show(maskedRace, pct(data.race.win_prob) + ' %')}</td>
                            </tr>
                            <tr>
                                <td colspan="2" class="download-hint">
                                    {$t('epc.race.downloadHint')}
                                    <button class="link-button" onclick={openBearoffSettings}>{$t('epc.race.openConfig')}</button>
                                </td>
                            </tr>
                        {/if}
                    </tbody>
                </table>
            {/if}
        </div>
    {/if}
</div>

<style>
    .epc-panel {
        position: relative;
        height: 100%;
        overflow-y: auto;
        overflow-x: hidden;
        padding: 2px 12px;
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

    /* The défi toggle stays pinned to the panel's top-right corner. */
    .challenge-toggle {
        position: absolute;
        top: 4px;
        right: 12px;
        z-index: 3;
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

    /* Players table on top, race table right below it — a single compact
       column, all sizes on the app's normal type tokens. Right padding
       leaves room for the pinned défi toggle. */
    .tables-container {
        display: flex;
        flex-direction: column;
        gap: 3px;
        align-items: flex-start;
        padding-right: 80px;
    }

    table {
        border-collapse: collapse;
        font-size: var(--font-size-base);
    }

    th,
    td {
        padding: 0 8px;
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
        font-size: var(--font-size-small);
        font-weight: 600;
        color: #444;
    }

    .race-title {
        text-align: left;
    }

    .race-title-inner {
        display: flex;
        align-items: center;
        gap: 6px;
        flex-wrap: wrap;
    }

    .race-controls {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        margin-left: 12px;
    }

    .ctl-label {
        font-size: var(--font-size-small);
        color: #777;
        text-transform: uppercase;
        letter-spacing: 0.3px;
    }

    .dot-btn {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        padding: 1px 4px;
        background: none;
        border: 1px solid transparent;
        border-radius: 4px;
        cursor: pointer;
    }

    .dot-btn.active {
        border-color: #1a56c4;
        background: #e8f0fe;
    }

    .cube-select {
        font: inherit;
        font-size: var(--font-size-small);
        padding: 0 2px;
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
        text-align: left;
    }

    tr.masked,
    table.masked {
        cursor: pointer;
    }

    tr.masked td:not(.row-label),
    table.masked td {
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
</style>
