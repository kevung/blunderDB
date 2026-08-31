<script>
    import { onMount, onDestroy, untrack } from 'svelte';
    import { statusBarModeStore, MODAL, openModal, configInitialTabStore } from '../stores/uiStore';
    import { epcDataStore, epcChallengeStore, epcRevealedStore, resetEpcReveal } from '../stores/epcStore';
    import { positionStore } from '../stores/positionStore';
    import { selectedMoveStore } from '../stores/analysisStore';
    import { GetEpcChallenge, SaveEpcChallenge, GetGammonNetDisplayPly, GetGammonNetPruneK, GetGammonNetCandidates } from '../../wailsjs/go/main/Config.js';
    import { EvaluatePositionImmediate, StartEvaluationAtRest, CancelEvaluationAtRest } from '../../wailsjs/go/gui/App.js';
    import { EventsOn, BrowserOpenURL } from '../../wailsjs/runtime/runtime.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';
    import { moverFactsToSides } from '../utils/positionFacts.js';
    import CandidateMovesTable from './CandidateMovesTable.svelte';
    import CubeVerdictTable from './CubeVerdictTable.svelte';
    import PositionFactsTable from './PositionFactsTable.svelte';

    let isActive = $derived($statusBarModeStore === 'EPC');
    let data = $derived($epcDataStore);
    let challenge = $derived($epcChallengeStore);
    let revealed = $derived($epcRevealedStore);

    // The decision the board is asking for is decided structurally by
    // whether dice are set (ADR-0017 rule 2) — the same [0, 0]-means-no-dice
    // convention AnalysisPanel uses for its own cube position (#124/#125,
    // ADR-0013).
    let dice = $derived($positionStore?.dice ?? [0, 0]);
    let hasDiceSet = $derived(dice[0] > 0 && dice[1] > 0);
    let onRoll = $derived($positionStore?.player_on_roll ?? 0);
    let hasScore = $derived(($positionStore?.score?.[0] ?? -1) !== -1 || ($positionStore?.score?.[1] ?? -1) !== -1);

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
    // The position's fact vector (ADR-0017): win/gammon/backgammon chances
    // and the cubeless equity, before any roll — always mover-relative
    // (Player/Opponent), converted to bottom/top below. Free on the cube
    // branch, paid for on the moves branch (see gammonnet_eval.go).
    let evalPreRoll = $state(null);

    // Exact never yields ITS OWN win probability (ADR-0012: "it wins
    // wherever it is available, and nothing displaces it" — a real lookup,
    // referential-independent). But the exact table is money-referential
    // (MoneyFromEntry never reads the score): at a match score its equities
    // and verdict answer the wrong question, so the evaluated regime — which
    // IS match-aware via Decide's MatchState — supplies those instead, and
    // the badge names both sources (ADR-0017 decision 4). Off score, or
    // outside the exact domain, this is unchanged from before.
    let displayRace = $derived.by(() => {
        if (!data.race) return evalRaceOverride;
        if (data.race.regime !== 'exact') return evalRaceOverride ?? data.race;
        if (!hasScore) return data.race;
        if (!evalRaceOverride) return data.race; // evaluated hasn't landed yet
        return { ...evalRaceOverride, win_prob: data.race.win_prob, source_checkers: data.race.source_checkers, exactWin: true };
    });

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

    // Leaving the Eval tab clears the board's selected-move arrow — the same
    // visibility-driven clearing AnalysisPanel already does, so a move
    // picked here does not linger once a different panel is showing.
    let _prevActive = false;
    $effect(() => {
        const v = isActive;
        if (v !== _prevActive) {
            if (!v) selectedMoveStore.set(null);
            _prevActive = v;
        }
    });

    function runEvaluationEscalation() {
        const pos = $positionStore;
        if (!isActive || !pos) return;

        // Clear the selected-move arrow BEFORE the new result lands, not
        // after: while the escalation is in flight the old candidate list
        // (and its arrow) are for a position the user just left (ADR-0017
        // rule 3's "a stale value is never shown dimmed" — the gesture that
        // invalidates it is the gesture that triggers the recomputation).
        selectedMoveStore.set(null);

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
        evalPreRoll = result?.preRoll ?? null;
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
        selectedMoveStore.set(null);
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

    // #131: a discreet, single-word attribution — the engine's name is the
    // link itself, never a sentence. Full credit (Strehl for the network,
    // gammonNet for the search/MET/cube configuration around it, ADR-0011)
    // lives in the Acknowledgements section of the in-app help, not here.
    function openGammonNetRepo() {
        BrowserOpenURL('https://github.com/kevung/gammonNet');
    }

    function handleMoveRowClick(move) {
        if ($selectedMoveStore === move.move) {
            selectedMoveStore.set(null);
        } else {
            selectedMoveStore.set(move.move);
        }
    }

    // The race analysis follows the position: the on-roll player is edited on
    // the board (click a player's bearoff/score rectangle, as in EDIT mode)
    // and the cube owner by clicking the cube on the board. The position
    // store is the single source of truth: any change re-triggers updateEPC
    // and re-masks the défi zones.

    // Défi mode: three zones — the bottom row, the top row, and the one
    // decision block the board is asking for (ADR-0017's Q8 corollary).
    // Values are replaced by a placeholder until their zone is revealed;
    // clicking a masked row/block reveals it.
    let maskedBottom = $derived(challenge && !revealed.bottom);
    let maskedTop = $derived(challenge && !revealed.top);
    let maskedDecision = $derived(challenge && !revealed.decision);

    const HIDDEN = '···';
    const show = (masked, v) => (masked ? HIDDEN : v);
    const pct = (x) => (100 * x).toFixed(2);
    const eq = (x) => (x >= 0 ? '+' : '') + x.toFixed(3);
    const sd = (x, digits) => (x >= 0 ? '+' : '') + x.toFixed(digits);

    // Decision-theoretic best equity: the roller picks double or not, the
    // opponent then picks the cheaper response.
    let bestEq = $derived(displayRace?.money ? Math.max(displayRace.money.no_double, Math.min(displayRace.money.double_take, displayRace.money.double_pass)) : 0);
    let bestVerdict = $derived(displayRace?.money?.verdict ?? '');
    // Gap to the best decision, shown under every non-best equity (XG style).
    const gap = (v) => '(' + sd(v - bestEq, 3) + ')';

    // ADR-0017 rule 1/CONTEXT.md "Position fact": win/gammon/backgammon and
    // the cubeless equity, per board side — always pre-roll, whatever the
    // board is asking. A race position is authoritative from displayRace
    // (it already carries the regime badge and, off the exact table, the
    // same computation the generic cube path would otherwise duplicate);
    // any other position falls back to the generic PreRoll payload.
    let raceFacts = $derived(
        displayRace
            ? moverFactsToSides(
                  { win: displayRace.win_prob, gammon: displayRace.win_gammon ?? 0, backgammon: displayRace.win_backgammon ?? 0, cubeless: displayRace.money?.cubeless ?? null },
                  { win: 1 - displayRace.win_prob, gammon: displayRace.lose_gammon ?? 0, backgammon: displayRace.lose_backgammon ?? 0 },
                  displayRace.on_roll
              )
            : { bottom: null, top: null }
    );
    let genericFacts = $derived(
        evalPreRoll
            ? moverFactsToSides(
                  { win: evalPreRoll.playerWinChance, gammon: evalPreRoll.playerGammonChance, backgammon: evalPreRoll.playerBackgammonChance, cubeless: evalPreRoll.cubelessEquity },
                  { win: evalPreRoll.opponentWinChance, gammon: evalPreRoll.opponentGammonChance, backgammon: evalPreRoll.opponentBackgammonChance },
                  onRoll
              )
            : { bottom: null, top: null }
    );
    // The cubeless equity follows the position's own referential (money at
    // money play, 2×MWC−1 at a match score, ADR-0016): CubelessValue and
    // race.CubeVerdict.Cubeless are both already computed in that scale, so
    // no adjustment is needed here — ADR-0017's dependency on ADR-0016 for
    // this column is resolved.
    let facts = $derived(data.race ? raceFacts : genericFacts);

    // Whether a race decision (money ND/DT/DP + verdict) is on screen at
    // all — absent in the estimated regime (ADR-0009/0012: never
    // estimated), and always absent once dice are set (ADR-0017 rule 2: a
    // race with dice on it is asking about checkers, not the cube).
    let showRaceDecision = $derived(!hasDiceSet && !!data.race && !!displayRace?.money);
    let showGenericCube = $derived(!hasDiceSet && !data.race && !!evalCubeAnalysis);
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
        <div class="epc-content">
            <!-- Top row: the always-present facts table, the one decision
                 block the board asks for (never both a race verdict and a
                 checker decision — ADR-0017 rule 2), and the badge column.
                 flex-wrap so a narrow panel stacks it instead of squeezing
                 (ADR-0017's container-query layout). -->
            <div class="top-row">
                <PositionFactsTable
                    bottom={facts.bottom}
                    top={facts.top}
                    bottomEPC={data.bottomEPC}
                    topEPC={data.topEPC}
                    {maskedBottom}
                    {maskedTop}
                    onRevealBottom={() => reveal('bottom')}
                    onRevealTop={() => reveal('top')}
                    preRoll={hasDiceSet}
                />

                {#if showRaceDecision}
                    <table class="decision-race" class:masked={maskedDecision} onclick={() => maskedDecision && reveal('decision')} title={maskedDecision ? $t('epc.clickToReveal') : undefined}>
                        <tbody>
                            <tr>
                                <th>{$t('epc.race.cubeless')}</th>
                                <th>{$t('epc.race.noDouble')}</th>
                                <th>{$t('epc.race.doubleTake')}</th>
                                <th>{$t('epc.race.doublePass')}</th>
                            </tr>
                            <tr>
                                <td>{show(maskedDecision, eq(displayRace.money.cubeless))}</td>
                                <td>
                                    {show(maskedDecision, eq(displayRace.money.no_double))}
                                    {#if !maskedDecision && bestVerdict && bestVerdict !== 'no_double'}
                                        <div class="eq-gap">{gap(displayRace.money.no_double)}</div>
                                    {/if}
                                </td>
                                <td>
                                    {show(maskedDecision, eq(displayRace.money.double_take))}
                                    {#if !maskedDecision && bestVerdict && bestVerdict !== 'double_take'}
                                        <div class="eq-gap">{gap(displayRace.money.double_take)}</div>
                                    {/if}
                                </td>
                                <td>
                                    {show(maskedDecision, eq(displayRace.money.double_pass))}
                                    {#if !maskedDecision && bestVerdict && bestVerdict !== 'double_pass'}
                                        <div class="eq-gap">{gap(displayRace.money.double_pass)}</div>
                                    {/if}
                                </td>
                            </tr>
                        </tbody>
                    </table>
                    <div class="decision-chip-wrap">
                        <span class="decision-chip" title={$t('epc.race.cubeStates.' + displayRace.money.cube_state)}>
                            {#if maskedDecision}
                                {HIDDEN}
                            {:else if displayRace.money.verdict}
                                {$t('epc.race.verdicts.' + displayRace.money.verdict)}
                            {:else}
                                {$t('epc.race.noDecision')}
                            {/if}
                        </span>
                    </div>
                {:else if showGenericCube}
                    {#if maskedDecision}
                        <!-- CubeVerdictTable is a foreign component with its own
                             scoped CSS: a wrapper `.masked` class cannot grey
                             out values it never sees. Défi hides the whole
                             thing by not handing it real data until revealed —
                             the only way to guarantee nothing leaks. -->
                        <div class="decision-cube-masked" onclick={() => reveal('decision')} title={$t('epc.clickToReveal')}>{HIDDEN}</div>
                    {:else}
                        <div class="decision-cube">
                            <CubeVerdictTable cubeAnalysis={evalCubeAnalysis} cubeValue={$positionStore?.cube?.value ?? 0} />
                        </div>
                    {/if}
                {:else if !hasDiceSet}
                    <div class="decision-pending">{$t('eval.pending')}</div>
                {/if}

                <div class="badges-col">
                    {#if data.race}
                        {#if displayRace?.exactWin}
                            <span class="badge badge-composite" title={$t('epc.race.exactAndEvaluatedTooltip')}>
                                {$t('epc.race.exactAndEvaluated')}
                            </span>
                        {:else if displayRace?.regime === 'exact'}
                            <span class="badge badge-exact" title={$t('epc.race.exactTooltip', { n: displayRace.source_checkers })}>
                                {$t('epc.race.exact')}
                            </span>
                        {:else if displayRace?.regime === 'evaluated'}
                            <span class="badge badge-evaluated" title={$t('epc.race.evaluatedTooltip')}>
                                {$t('epc.race.evaluated')} · {displayRace.depth}
                            </span>
                        {:else if displayRace}
                            <button
                                class="badge badge-estimated badge-link"
                                onclick={openBearoffSettings}
                                title={$t('epc.race.estimatedTooltip', { p99: pct(displayRace.p99) }) + ' ' + $t('epc.race.downloadHint')}
                                aria-label={$t('epc.race.openConfig')}
                            >
                                {$t('epc.race.estimated')} ± {pct(displayRace.sigma)} %
                            </button>
                        {/if}
                    {/if}
                    <button class="eval-engine-badge" onclick={openGammonNetRepo} title={$t('eval.engineTooltip')} aria-label={$t('eval.engineTooltip')}>?</button>
                    <label class="challenge-toggle" title={$t('epc.challengeTooltip')}>
                        <input type="checkbox" checked={challenge} onchange={toggleChallenge} />
                        <span>{$t('epc.challenge')}</span>
                    </label>
                </div>
            </div>

            {#if !data.bottomEPC && !data.topEPC}
                <div class="epc-race-hint">{$t('epc.placeCheckers')}</div>
            {/if}

            <!-- The moves list is the only region that ever scrolls
                 (ADR-0017): its header stays sticky, and everything above
                 (facts, badges, and a race/cube decision when there is no
                 dice) stays on screen regardless of candidate count. -->
            {#if hasDiceSet}
                <div class="moves-scroll">
                    <CandidateMovesTable moves={evalMoves} selectedMove={$selectedMoveStore} onRowClick={handleMoveRowClick} />
                    {#if evalMoves.length === 0}
                        <div class="eval-placeholder">{$t('eval.pending')}</div>
                    {/if}
                </div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .epc-panel {
        height: 100%;
        box-sizing: border-box;
        /* Only .moves-scroll below ever scrolls (ADR-0017) — the panel
           itself never grows a scrollbar. */
        overflow: hidden;
        padding: 3px 14px;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Noto Sans JP', sans-serif;
        font-size: var(--font-size-base);
        /* Lets CandidateMovesTable/CubeVerdictTable/PositionFactsTable's own
           @container rules stack on a narrow panel (ADR-0017's layout). */
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

    .epc-content {
        height: 100%;
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    /* Facts + the one decision block + badges, on one row when the panel is
       wide enough (~1000px, ADR-0017's budget), wrapping to two lines and
       then a stacked column as it narrows — one flow, driven by the panel's
       own width, not by whether it is docked at the bottom or the side. */
    .top-row {
        flex: 0 0 auto;
        display: flex;
        flex-wrap: wrap;
        align-items: flex-start;
        gap: 8px 20px;
    }

    .decision-pending,
    .eval-placeholder,
    .epc-race-hint {
        color: #888;
        font-size: var(--font-size-small);
    }

    .epc-race-hint {
        flex: 0 0 auto;
    }

    .decision-race,
    .decision-cube {
        border-collapse: collapse;
        font-size: var(--font-size-base);
    }

    .decision-cube {
        display: flex;
    }

    .decision-cube-masked {
        display: flex;
        align-items: center;
        justify-content: center;
        min-width: 180px;
        align-self: stretch;
        color: #aaa;
        letter-spacing: 2px;
        cursor: pointer;
        font-size: var(--font-size-base);
    }

    .decision-cube-masked:hover {
        background: #f5f5f5;
    }

    .decision-race th,
    .decision-race td {
        padding: 2px 10px;
        text-align: center;
        white-space: nowrap;
        font-variant-numeric: tabular-nums;
    }

    .decision-race th {
        font-size: var(--font-size-small);
        color: #777;
        text-transform: uppercase;
        letter-spacing: 0.3px;
        font-weight: 600;
    }

    .decision-race td {
        font-weight: 600;
        color: #1a56c4;
    }

    .decision-race.masked {
        cursor: pointer;
    }

    .decision-race.masked td {
        color: #aaa;
        letter-spacing: 2px;
    }

    .decision-race.masked:hover td {
        background: #f5f5f5;
    }

    .decision-chip-wrap {
        display: flex;
        align-items: center;
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

    /* The badge column always sits last in the row: regime badge, engine
       attribution, and the défi toggle — no more absolute positioning or
       reserved padding (ADR-0017 removes both). */
    .badges-col {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-left: auto;
        flex-wrap: wrap;
    }

    .badge {
        padding: 0 8px;
        border-radius: 9px;
        font-size: var(--font-size-small);
        font-weight: 600;
        letter-spacing: 0.3px;
        white-space: nowrap;
    }

    .badge-link {
        border: none;
        cursor: pointer;
        font-family: inherit;
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

    /* ADR-0017 decision 4: exact's win probability, evaluated's equities and
       verdict — a composite the badge names outright rather than picking a
       single colour that would misrepresent one half of it. */
    .badge-composite {
        background: linear-gradient(90deg, #e5f3e8 0 50%, #e8f0fe 50% 100%);
        border: 1px solid #c4d8f5;
        color: #1a56c4;
    }

    /* #131: a discreet mention that gammonNet is the engine, one character
       and a link — never a sentence in the panel itself (full attribution
       lives in the in-app help's Acknowledgements). */
    .eval-engine-badge {
        width: 14px;
        height: 14px;
        line-height: 14px;
        padding: 0;
        border: 1px solid #ccc;
        border-radius: 50%;
        background: transparent;
        color: #aaa;
        font-size: var(--font-size-small);
        text-align: center;
        cursor: pointer;
    }

    .eval-engine-badge:hover {
        color: #1a56c4;
        border-color: #1a56c4;
    }

    .challenge-toggle {
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

    /* The only scrolling region in the panel (ADR-0017). */
    .moves-scroll {
        flex: 1 1 auto;
        min-height: 0;
        overflow-y: auto;
    }

    @container (max-width: 700px) {
        .badges-col {
            margin-left: 0;
        }
    }
</style>
