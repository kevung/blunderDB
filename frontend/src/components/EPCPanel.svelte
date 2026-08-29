<script>
    import { onMount, onDestroy, untrack } from 'svelte';
    import { statusBarModeStore, MODAL, openModal, configInitialTabStore } from '../stores/uiStore';
    import { epcDataStore, epcChallengeStore, epcRevealedStore, resetEpcReveal } from '../stores/epcStore';
    import { positionStore } from '../stores/positionStore';
    import { GetEpcChallenge, SaveEpcChallenge, GetGammonNetDisplayPly, GetGammonNetPruneK, GetGammonNetCandidates } from '../../wailsjs/go/main/Config.js';
    import { EvaluatePositionImmediate, StartEvaluationAtRest, CancelEvaluationAtRest } from '../../wailsjs/go/gui/App.js';
    import { EventsOn } from '../../wailsjs/runtime/runtime.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';
    import CandidateMovesTable from './CandidateMovesTable.svelte';
    import CubeVerdictTable from './CubeVerdictTable.svelte';

    let isActive = $derived($statusBarModeStore === 'EPC');
    let data = $derived($epcDataStore);
    let challenge = $derived($epcChallengeStore);
    let revealed = $derived($epcRevealedStore);

    // The evaluation volet always renders — the position on the board decides
    // what there is to compute (dice set → candidate moves, no dice → a cube
    // verdict), the same [0, 0]-means-no-dice convention AnalysisPanel uses
    // for its own cube position (#124/#125, ADR-0013).
    let dice = $derived($positionStore?.dice ?? [0, 0]);
    let hasDiceSet = $derived(dice[0] > 0 && dice[1] > 0);
    let evalMoves = $state([]);
    let evalCubeAnalysis = $state(null);
    // Race panel's "evaluated" regime (#126, ADR-0012): gammonNet's own
    // async result (same 0-ply-then-2-ply escalation as evalMoves/
    // evalCubeAnalysis above), carrying a verdict where the fast synchronous
    // path (updateEPC / epcDataStore, "exact"/"estimated" only) has none.
    // Null whenever the position is not a race outside the exact domain —
    // the Go side gates it on the same predicate race.Evaluate itself uses,
    // so this self-clears on the very next 0-ply call after any gesture.
    let evalRaceOverride = $state(null);
    // Exact never yields to it (ADR-0012: "it wins wherever it is
    // available, and nothing displaces it"); otherwise prefer the evaluated
    // regime once it has landed, falling back to the synchronous
    // exact/estimated payload while it is still in flight.
    let displayRace = $derived(data.race?.regime === 'exact' ? data.race : (evalRaceOverride ?? data.race));

    // Progressive escalation (#125): 0-ply synchronously at the gesture
    // (measured ~376µs, ADR-0011 — cheap enough for a plain round trip),
    // then the configured display depth (canonically 2-ply k=12) in the
    // background after 500ms of rest, cancelled by any newer gesture. No
    // 1-ply step: a state the user would never see pass.
    const EVAL_REST_DELAY_MS = 500;
    let evalRestTimer = null;
    let evalGeneration = 0; // guards a late "done" against a position the user already left

    // A stable signature is the effect's ONLY tracked dependency — never
    // evalMoves/evalCubeAnalysis, which this same effect writes. Reading a
    // $state an effect just wrote is exactly the fcde0243 regression
    // (effect_update_depth_exceeded, StatsFilterBar.svelte): the fix there,
    // reused here, is deriving from a local value and wrapping everything
    // else in untrack() so Svelte sees one dependency.
    let positionSignature = $derived(JSON.stringify($positionStore ?? null));

    $effect(() => {
        const signature = positionSignature; // tracked: the position
        const active = isActive; // tracked: is the Eval tab even shown
        if (!signature || !active) return;
        untrack(() => {
            runEvaluationEscalation();
        });
    });

    function runEvaluationEscalation() {
        const pos = $positionStore;
        if (!isActive || !pos) return;

        evalGeneration += 1;
        const generation = evalGeneration;

        if (evalRestTimer) {
            clearTimeout(evalRestTimer);
            evalRestTimer = null;
        }
        CancelEvaluationAtRest().catch(() => {});

        GetGammonNetPruneK()
            .then((pruneK) =>
                GetGammonNetCandidates().then((candidates) =>
                    EvaluatePositionImmediate(pos, pruneK, candidates).then((result) => {
                        if (generation !== evalGeneration) return; // superseded while awaiting
                        applyEvalResult(result);
                    })
                )
            )
            .catch((error) => logger.error('gammonNet 0-ply evaluation failed:', error));

        evalRestTimer = setTimeout(() => {
            if (generation !== evalGeneration) return;
            Promise.all([GetGammonNetDisplayPly(), GetGammonNetPruneK(), GetGammonNetCandidates()])
                .then(([ply, pruneK, candidates]) => {
                    if (generation !== evalGeneration) return;
                    StartEvaluationAtRest(pos, ply, pruneK, candidates).catch((error) => logger.error('gammonNet evaluation-at-rest failed to start:', error));
                })
                .catch((error) => logger.error('gammonNet evaluation-at-rest settings failed:', error));
        }, EVAL_REST_DELAY_MS);
    }

    // The depth label on the applied result always says what actually
    // produced it (0-ply or the display depth) — never a depth that was
    // requested but superseded before it ran. A "cancelled" event simply
    // leaves whatever 0-ply result is already showing untouched.
    function applyEvalResult(result) {
        evalMoves = result?.moves ?? [];
        evalCubeAnalysis = result?.cube ?? null;
        evalRaceOverride = result?.race ?? null;
    }

    let unsubEval = [];
    onMount(() => {
        GetEpcChallenge()
            .then((v) => epcChallengeStore.set(!!v))
            .catch(() => {});

        unsubEval = [
            EventsOn('gammonnet-eval:done', (result) => applyEvalResult(result)),
            EventsOn('gammonnet-eval:cancelled', () => {}),
            EventsOn('gammonnet-eval:error', (e) => logger.error('gammonNet evaluation-at-rest error:', e))
        ];
    });

    onDestroy(() => {
        if (evalRestTimer) clearTimeout(evalRestTimer);
        CancelEvaluationAtRest().catch(() => {});
        unsubEval.forEach((off) => off && off());
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

    function openBearoffSettings() {
        configInitialTabStore.set('bearoff');
        openModal(MODAL.CONFIG);
    }

    // The race analysis follows the position: the on-roll player is edited on
    // the board (click a player's bearoff/score rectangle, as in EDIT mode)
    // and the cube owner by clicking the cube on the board. The position
    // store is the single source of truth: any change re-triggers updateEPC
    // and re-masks the défi zones.

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

    // Decision-theoretic best equity: the roller picks double or not, the
    // opponent then picks the cheaper response.
    let bestEq = $derived(displayRace?.money ? Math.max(displayRace.money.no_double, Math.min(displayRace.money.double_take, displayRace.money.double_pass)) : 0);
    let bestVerdict = $derived(displayRace?.money?.verdict ?? '');
    // Gap to the best decision, shown under every non-best equity (XG style).
    const gap = (v) => '(' + sd(v - bestEq, 3) + ')';

    // Win probabilities per colour (the stored value is the on-roll player's).
    let winBlack = $derived(displayRace ? (displayRace.on_roll === 0 ? displayRace.win_prob : 1 - displayRace.win_prob) : 0);
    let winWhite = $derived(displayRace ? 1 - winBlack : 0);
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
    {:else}
        <label class="challenge-toggle" title={$t('epc.challengeTooltip')}>
            <input type="checkbox" checked={challenge} onchange={toggleChallenge} />
            <span>{$t('epc.challenge')}</span>
        </label>
        <div class="tables-container">
            <!-- Evaluation volet: always present, whatever the position — a
                 race or not, home-board bearoff or not. Structurally decided
                 by whether dice are set; both branches are placeholders until
                 the live gammonNet evaluation is wired in. -->
            <div class="eval-section">
                {#if hasDiceSet}
                    <CandidateMovesTable moves={evalMoves} />
                    {#if evalMoves.length === 0}
                        <div class="eval-placeholder">{$t('eval.pending')}</div>
                    {/if}
                {:else if evalCubeAnalysis}
                    <CubeVerdictTable cubeAnalysis={evalCubeAnalysis} cubeValue={$positionStore?.cube?.value ?? 0} />
                {:else}
                    <div class="eval-placeholder">{$t('eval.pending')}</div>
                {/if}
            </div>

            <!-- Race volet: reserves its place, but only a race position
                 fills it — a colour dot suffices as the row label; the Δ row
                 is signed (bottom − top: negative when Black leads) and
                 absorbs the old comparison section. -->
            <div class="race-section">
                {#if !data.bottomEPC && !data.topEPC && !data.race}
                    <div class="epc-race-hint">{$t('epc.placeCheckers')}</div>
                {/if}
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

                <!-- Race / cube table, transposed like the players table. Win
                 chances are given per colour (dots in the headers, no % sign
                 in the values); equities and the verdict are the on-roll
                 player's, as in the Analysis panel. -->
                {#if data.race}
                    <div class="race-wrap">
                        <table class="race-table" class:masked={maskedRace} onclick={() => maskedRace && reveal('race')} title={maskedRace ? $t('epc.clickToReveal') : undefined}>
                            <tbody>
                                {#if displayRace.money}
                                    <tr>
                                        <th><span class="player-indicator bottom"></span> {$t('epc.race.winPct')}</th>
                                        <th><span class="player-indicator top"></span> {$t('epc.race.winPct')}</th>
                                        <th>{$t('epc.race.cubeless')}</th>
                                        <th>{$t('epc.race.noDouble')}</th>
                                        <th>{$t('epc.race.doubleTake')}</th>
                                        <th>{$t('epc.race.doublePass')}</th>
                                    </tr>
                                    <tr>
                                        <td class="main-value">{show(maskedRace, pct(winBlack))}</td>
                                        <td class="main-value">{show(maskedRace, pct(winWhite))}</td>
                                        <td>{show(maskedRace, eq(displayRace.money.cubeless))}</td>
                                        <td>
                                            {show(maskedRace, eq(displayRace.money.no_double))}
                                            {#if !maskedRace && bestVerdict && bestVerdict !== 'no_double'}
                                                <div class="eq-gap">{gap(displayRace.money.no_double)}</div>
                                            {/if}
                                        </td>
                                        <td>
                                            {show(maskedRace, eq(displayRace.money.double_take))}
                                            {#if !maskedRace && bestVerdict && bestVerdict !== 'double_take'}
                                                <div class="eq-gap">{gap(displayRace.money.double_take)}</div>
                                            {/if}
                                        </td>
                                        <td>
                                            {show(maskedRace, eq(displayRace.money.double_pass))}
                                            {#if !maskedRace && bestVerdict && bestVerdict !== 'double_pass'}
                                                <div class="eq-gap">{gap(displayRace.money.double_pass)}</div>
                                            {/if}
                                        </td>
                                    </tr>
                                {:else}
                                    <tr>
                                        <th><span class="player-indicator bottom"></span> {$t('epc.race.winPct')}</th>
                                        <th><span class="player-indicator top"></span> {$t('epc.race.winPct')}</th>
                                    </tr>
                                    <tr>
                                        <td class="main-value">{show(maskedRace, pct(winBlack))}</td>
                                        <td class="main-value">{show(maskedRace, pct(winWhite))}</td>
                                    </tr>
                                {/if}
                            </tbody>
                        </table>
                        <div class="race-side">
                            {#if displayRace.money}
                                <span class="decision-chip" title={$t('epc.race.cubeStates.' + displayRace.money.cube_state)}>
                                    {#if maskedRace}
                                        {HIDDEN}
                                    {:else if displayRace.money.verdict}
                                        {$t('epc.race.verdicts.' + displayRace.money.verdict)}
                                    {:else}
                                        {$t('epc.race.noDecision')}
                                    {/if}
                                </span>
                            {/if}
                            {#if displayRace.regime === 'exact'}
                                <span class="badge badge-exact" title={$t('epc.race.exactTooltip', { n: displayRace.source_checkers })}>
                                    {$t('epc.race.exact')}
                                </span>
                            {:else if displayRace.regime === 'evaluated'}
                                <span class="badge badge-evaluated" title={$t('epc.race.evaluatedTooltip')}>
                                    {$t('epc.race.evaluated')} · {displayRace.depth}
                                </span>
                            {:else}
                                <span class="badge badge-estimated" title={$t('epc.race.estimatedTooltip', { p99: pct(displayRace.p99) })}>
                                    {$t('epc.race.estimated')} ± {pct(displayRace.sigma)} %
                                </span>
                                <span class="download-hint">
                                    {$t('epc.race.downloadHint')}
                                    <button class="link-button" onclick={openBearoffSettings}>{$t('epc.race.openConfig')}</button>
                                </span>
                            {/if}
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    {/if}
</div>

<style>
    .epc-panel {
        position: relative;
        height: 100%;
        /* height:100% + vertical padding overflowed the tab body by exactly
           2×padding in content-box sizing — the phantom scrollbar. */
        box-sizing: border-box;
        overflow-y: auto;
        overflow-x: hidden;
        padding: 3px 14px;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans JP', sans-serif;
        font-size: var(--font-size-base);
        /* Lets CandidateMovesTable/CubeVerdictTable's own @container rules
           (mounted in .eval-section below) stack on a narrow panel, exactly
           as they already do inside AnalysisPanel. */
        container-type: inline-size;
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
        top: 6px;
        right: 14px;
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
        margin: 0;
    }

    /* Evaluation volet on top, race volet right below it — a single airy
       column, all sizes on the app's normal type tokens. Right padding
       leaves room for the pinned défi toggle. */
    .tables-container {
        display: flex;
        flex-direction: column;
        gap: 8px;
        align-items: flex-start;
        width: 100%;
        padding-right: 80px;
        box-sizing: border-box;
    }

    .eval-section {
        width: 100%;
    }

    .eval-placeholder {
        color: #888;
        font-size: var(--font-size-small);
    }

    /* Reserves its place even on a non-race position, rather than the panel
       height jumping when the race volet has nothing to show. */
    .race-section {
        width: 100%;
        min-height: 1.6em;
    }

    .epc-race-hint {
        color: #888;
        font-size: var(--font-size-small);
    }

    /* Clear visual separation between the two tables; the regime badge sits
       to the right of the race table. */
    .race-wrap {
        display: flex;
        align-items: center;
        gap: 14px;
        border-top: 2px solid #e0e0e0;
        padding-top: 5px;
        margin-top: 0;
    }

    table {
        border-collapse: collapse;
        font-size: var(--font-size-base);
    }

    th,
    td {
        padding: 1px 18px;
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

    /* Third regime (#126, ADR-0012): a distinct blue-leaning tone — closer
       to .decision-chip than to either the green "exact" or the amber
       "estimated" (a played-out search, not a lookup and not a summary
       estimate). */
    .badge-evaluated {
        background: #e8f0fe;
        border: 1px solid #c4d8f5;
        color: #1a56c4;
    }

    .race-side {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: 4px;
        min-width: 0;
    }

    .decision-chip {
        padding: 1px 10px;
        border-radius: 9px;
        background: #e8f0fe;
        border: 1px solid #c4d8f5;
        color: #1a56c4;
        font-weight: 600;
        white-space: nowrap;
    }

    .eq-gap {
        font-size: var(--font-size-small);
        color: #999;
        font-weight: 400;
        letter-spacing: 0;
    }

    .download-hint {
        font-size: var(--font-size-small);
        color: #8a6413;
        white-space: normal;
        max-width: 420px;
        text-align: left;
    }

    .link-button {
        background: none;
        border: none;
        padding: 0;
        color: #1a56c4;
        text-decoration: underline;
        cursor: pointer;
    }
</style>
