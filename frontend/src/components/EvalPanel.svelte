<script>
    import { onMount, onDestroy, untrack } from 'svelte';
    import { statusBarModeStore, MODAL, openModal, configInitialTabStore } from '../stores/uiStore';
    import { epcDataStore, epcChallengeStore, epcRevealedStore, resetEpcReveal } from '../stores/epcStore';
    import { positionStore } from '../stores/positionStore';
    import { selectedMoveStore } from '../stores/analysisStore';
    import { databasePathStore } from '../stores/databaseStore';
    import { positionRefusal } from '../services/positionRefusal.js';
    import { saveScratchBoard } from '../services/scratchBoard.js';
    import { GetEpcChallenge, SaveEpcChallenge, GetGammonNetDisplayPly, GetGammonNetPruneK, GetGammonNetCandidates } from '../../wailsjs/go/main/Config.js';
    import { EvaluatePositionImmediate, StartEvaluationAtRest, CancelEvaluationAtRest } from '../../wailsjs/go/gui/App.js';
    import { EventsOn, BrowserOpenURL } from '../../wailsjs/runtime/runtime.js';
    import { logger } from '../utils/logger.js';
    import { isBareLetter } from '../utils/keys.js';
    import { onChange } from '../utils/onChange.js';
    import { t } from '../i18n';
    import { moverFactsToSides } from '../utils/positionFacts.js';
    import { cubeDecision, cubeTurnability, isMoneyPosition } from '../utils/cubeDecision.js';
    import CandidateMovesTable from './CandidateMovesTable.svelte';
    import CubeVerdictTable from './CubeVerdictTable.svelte';
    import PositionFactsTable from './PositionFactsTable.svelte';

    let isActive = $derived($statusBarModeStore === 'EVAL');

    // « Ajouter à la base » writes the position alone via saveScratchBoard(),
    // never the evaluation. Disabled, with the reason as tooltip, when there is
    // no database or the save would refuse. On a stored position the click
    // marks it individually imported.
    let addRefusal = $derived(!$databasePathStore ? 'eval.addPositionNoDatabase' : $positionStore?.board ? positionRefusal($positionStore) : null);
    let adding = $state(false);

    async function addPosition() {
        if (addRefusal || adding) return;
        adding = true;
        try {
            await saveScratchBoard(); // says the number, or the refusal, itself
        } finally {
            adding = false;
        }
    }
    let data = $derived($epcDataStore);
    let challenge = $derived($epcChallengeStore);
    let revealed = $derived($epcRevealedStore);

    // Dice set → checker decision, [0, 0] → cube (ADR-0017 rule 2, ADR-0013).
    let dice = $derived($positionStore?.dice ?? [0, 0]);
    let hasDiceSet = $derived(dice[0] > 0 && dice[1] > 0);
    let onRoll = $derived($positionStore?.player_on_roll ?? 0);
    // isMoneyPosition, not an inline `!== -1`: OR and AND forms disagree on a
    // malformed half-money score.
    let isMoney = $derived(isMoneyPosition($positionStore));
    let hasScore = $derived(!isMoney);
    let jacoby = $derived(isMoney && $positionStore?.has_jacoby === 1);
    let beaver = $derived(isMoney && $positionStore?.has_beaver === 1);
    // The cube ceiling applies at any score: read off the position.
    let maxCube = $derived($positionStore?.max_cube ?? 0);

    let evalMoves = $state([]);
    let evalCubeAnalysis = $state(null);
    // The verdict as a value (ADR-0020 rule 3): translatable and keeps "too
    // good", unlike the imported BestCubeAction string.
    let evalCubeVerdict = $state('');
    // The engine declined the position (score beyond the MET): a state the
    // panel names, not an error (ADR-0020 rule 4).
    let evalRefused = $state(false);
    // Whether gammonNet has answered for this position: pending vs declined
    // (cubeDecision's `settled`).
    let evalSettled = $state(false);
    // The engine never answered (Wails call rejected or `gammonnet-eval:error`),
    // as opposed to declining; otherwise the panel would stay pending forever.
    // Cleared when a new evaluation starts or succeeds (`applyEvalResult`).
    let evalFailed = $state(false);
    let evalFailedMessage = $state('');
    // Race "evaluated" regime (ADR-0012): gammonNet's async verdict, where the
    // synchronous exact/estimated path has none. Null unless the position is a
    // race outside the exact domain (same predicate as race.Evaluate).
    let evalRaceOverride = $state(null);
    // Pre-roll fact vector (ADR-0017), mover-relative; converted to bottom/top below.
    let evalPreRoll = $state(null);

    // Exact keeps its win probability (ADR-0012), but its table is money-only:
    // at a match score the equities and verdict come from the match-aware
    // evaluated regime, and the badge names both (ADR-0017 decision 4).
    let displayRace = $derived.by(() => {
        if (!data.race) return evalRaceOverride;
        if (data.race.regime !== 'exact') return evalRaceOverride ?? data.race;
        if (!hasScore) return data.race;
        if (!evalRaceOverride) return data.race; // evaluated hasn't landed yet
        return { ...evalRaceOverride, win_prob: data.race.win_prob, source_checkers: data.race.source_checkers, exactWin: true };
    });

    // 0-ply synchronously (~376µs, ADR-0011), then the display depth after
    // 500 ms of rest, cancelled by any newer gesture.
    const EVAL_REST_DELAY_MS = 500;
    let evalRestTimer = null;
    let evalGeneration = 0; // guards a late "done" against a position the user already left

    // The signature is the effect's only tracked dependency; everything else is
    // untrack()ed, since reading the $state it writes loops
    // (effect_update_depth_exceeded).
    let positionSignature = $derived(JSON.stringify($positionStore ?? null));

    $effect(() => {
        const signature = positionSignature; // tracked: the position
        const active = isActive; // tracked: is the Eval tab even shown
        if (!signature || !active) return;
        untrack(() => {
            runEvaluationEscalation();
        });
    });

    // Leaving the tab clears the selected-move arrow, as AnalysisPanel does.
    $effect(
        onChange(
            () => isActive,
            (v) => {
                if (!v) selectedMoveStore.set(null);
            },
            false
        )
    );

    function runEvaluationEscalation() {
        const pos = $positionStore;
        if (!isActive || !pos) return;

        // Clear the arrow before the new result lands: the old list belongs to
        // the position just left (ADR-0017 rule 3).
        selectedMoveStore.set(null);
        evalSettled = false;
        evalFailed = false;
        evalFailedMessage = '';

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
            .catch((error) => {
                logger.error('gammonNet 0-ply evaluation failed:', error);
                if (generation !== evalGeneration) return; // superseded while awaiting
                evalFailed = true;
                evalFailedMessage = String(error);
            });

        evalRestTimer = setTimeout(() => {
            if (generation !== evalGeneration) return;
            Promise.all([GetGammonNetDisplayPly(), GetGammonNetPruneK(), GetGammonNetCandidates()])
                .then(([ply, pruneK, candidates]) => {
                    if (generation !== evalGeneration) return;
                    StartEvaluationAtRest(pos, ply, pruneK, candidates).catch((error) => {
                        logger.error('gammonNet evaluation-at-rest failed to start:', error);
                        if (generation !== evalGeneration) return;
                        evalFailed = true;
                        evalFailedMessage = String(error);
                    });
                })
                .catch((error) => {
                    logger.error('gammonNet evaluation-at-rest settings failed:', error);
                    if (generation !== evalGeneration) return;
                    evalFailed = true;
                    evalFailedMessage = String(error);
                });
        }, EVAL_REST_DELAY_MS);
    }

    // The depth label names what actually produced the result; "cancelled"
    // leaves the 0-ply result as is. Moves and Cube never come together: touch
    // only the one present, or the other side flashes back to pending.
    function applyEvalResult(result) {
        evalSettled = true;
        evalFailed = false;
        evalFailedMessage = '';
        evalRefused = !!result?.refused;
        if (evalRefused) {
            // Nothing to say: drop both sides, never keep the previous position's.
            evalMoves = [];
            evalCubeAnalysis = null;
            evalCubeVerdict = '';
            evalRaceOverride = null;
            evalPreRoll = null;
            return;
        }
        if (result?.moves !== undefined) evalMoves = result.moves;
        if (result?.cube !== undefined) {
            evalCubeAnalysis = result.cube ?? null;
            evalCubeVerdict = result?.cubeVerdict ?? '';
        }
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
            EventsOn('gammonnet-eval:error', (e) => {
                logger.error('gammonNet evaluation-at-rest error:', e);
                evalFailed = true;
                evalFailedMessage = String(e);
            })
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

    // One-word attribution; full credit is in the help's Acknowledgements (ADR-0011).
    function openGammonNetRepo() {
        BrowserOpenURL('https://github.com/kevung/gammonNet');
    }

    // Focus target for the keyboard navigation below.
    /** @type {HTMLElement | undefined} */
    let panelEl;

    function handleMoveRowClick(move) {
        if ($selectedMoveStore === move.move) {
            selectedMoveStore.set(null);
        } else {
            selectedMoveStore.set(move.move);
        }
        // Rows are plain <tr>s: focus the panel explicitly (WebKit and Chromium
        // differ on walking up to a focusable ancestor).
        panelEl?.focus({ preventScroll: true });
    }

    // j/k and arrows walk the candidates, Escape drops the selection, as in the
    // analysis panel. Required: keyboardService withholds these keys app-wide
    // while selectedMoveStore is set.
    function handleKeyDown(event) {
        if (event.key === 'Escape') {
            if ($selectedMoveStore) selectedMoveStore.set(null);
            return;
        }

        if (!$selectedMoveStore || evalMoves.length === 0) return;
        const currentIndex = evalMoves.findIndex((m) => m.move === $selectedMoveStore);

        if (isBareLetter(event, 'j') || event.key === 'ArrowDown') {
            event.preventDefault();
            if (currentIndex >= 0 && currentIndex < evalMoves.length - 1) {
                selectedMoveStore.set(evalMoves[currentIndex + 1].move);
            }
        } else if (isBareLetter(event, 'k') || event.key === 'ArrowUp') {
            event.preventDefault();
            if (currentIndex > 0) {
                selectedMoveStore.set(evalMoves[currentIndex - 1].move);
            }
        }
    }

    // positionStore is the source of truth: any change re-runs updateEPC and re-masks Défi.

    // Défi: three zones (bottom row, top row, decision block), each revealed by a click.
    let maskedBottom = $derived(challenge && !revealed.bottom);
    let maskedTop = $derived(challenge && !revealed.top);
    let maskedDecision = $derived(challenge && !revealed.decision);

    const HIDDEN = '···';
    const pct = (x) => (100 * x).toFixed(2);

    // Position facts per board side, always pre-roll (ADR-0017 rule 1): from
    // displayRace on a race, else the generic PreRoll payload.
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
    // Cubeless equity arrives in the position's referential (ADR-0016): no conversion.
    let facts = $derived(data.race ? raceFacts : genericFacts);

    // Mover-relative, the frame of the move rows (ADR-0018 rule 2); only for
    // CandidateMovesTable, dice set.
    let baselineFacts = $derived.by(() => {
        if (data.race) {
            if (!displayRace) return null;
            return {
                cubelessEquity: displayRace.money?.cubeless ?? null,
                playerWinChance: displayRace.win_prob,
                playerGammonChance: displayRace.win_gammon ?? 0,
                playerBackgammonChance: displayRace.win_backgammon ?? 0,
                opponentWinChance: displayRace.win_prob != null ? 1 - displayRace.win_prob : null,
                opponentGammonChance: displayRace.lose_gammon ?? 0,
                opponentBackgammonChance: displayRace.lose_backgammon ?? 0
            };
        }
        return evalPreRoll;
    });

    // One cube Decision shape (ADR-0020) from race.Money or DoublingCubeAnalysis,
    // shown whenever there are no dice; its state cell says answer, none or refusal.
    let decision = $derived(
        cubeDecision({
            race: displayRace,
            isRace: !!data.race,
            cubeAnalysis: evalCubeAnalysis,
            verdictKey: evalCubeVerdict,
            refused: evalRefused,
            turnability: cubeTurnability($positionStore),
            settled: evalSettled
        })
    );
    let showDecision = $derived(!hasDiceSet);

    // PositionFactsTable (ADR-0018 rule 1) would be empty with dice off a race: not mounted.
    let showFactsTable = $derived(!!data.race || !hasDiceSet);

    // Off a race, depth is named once in the strip (ADR-0018 rule 4).
    let genericDepthLabel = $derived(data.race ? null : hasDiceSet ? (evalMoves[0]?.analysisDepth ?? null) : (evalCubeAnalysis?.analysisDepth ?? null));
</script>

<!-- Present in every state, errors included: a failed evaluation leaves the position savable. -->
{#snippet addPositionButton()}
    <button type="button" class="add-position" onclick={addPosition} disabled={!!addRefusal || adding} title={addRefusal ? $t(addRefusal) : $t('eval.addPositionTooltip')}>
        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v6m3-3H9m12 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
        <span>{$t('eval.addPosition')}</span>
    </button>
{/snippet}

<!-- A <section> taking focus for keyboard delegation (tabindex="-1"); no ARIA
     role is both a landmark and interactive, hence the ignore. -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<section class="eval-panel" bind:this={panelEl} aria-label={$t('eval.panelLabel')} tabindex="-1" onkeydown={handleKeyDown}>
    {#if !isActive}
        <div class="eval-inactive">
            <div class="eval-inactive-message">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="inactive-icon">
                    <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        d="M15.75 15.75V18m-7.5-6.75h.008v.008H8.25v-.008Zm0 2.25h.008v.008H8.25V13.5Zm0 2.25h.008v.008H8.25v-.008Zm0 2.25h.008v.008H8.25V18Zm2.498-6.75h.007v.008h-.007v-.008Zm0 2.25h.007v.008h-.007V13.5Zm0 2.25h.007v.008h-.007v-.008Zm0 2.25h.007v.008h-.007V18Zm2.504-6.75h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V13.5Zm0 2.25h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V18Zm2.498-6.75h.008v.008h-.008v-.008Zm0 2.25h.008v.008h-.008V13.5ZM8.25 6h7.5v2.25h-7.5V6ZM12 2.25c-1.892 0-3.758.11-5.593.322C5.307 2.7 4.5 3.65 4.5 4.757V19.5a2.25 2.25 0 0 0 2.25 2.25h10.5a2.25 2.25 0 0 0 2.25-2.25V4.757c0-1.108-.806-2.057-1.907-2.185A48.507 48.507 0 0 0 12 2.25Z"
                    />
                </svg>
                <span>{$t('eval.inactive')}</span>
            </div>
        </div>
    {:else if data.error}
        <div class="eval-content">
            <div class="badges-strip">{@render addPositionButton()}</div>
            <div class="eval-error">
                <span class="error-text">{data.error}</span>
            </div>
        </div>
    {:else if evalFailed}
        <div class="eval-content">
            <div class="badges-strip">{@render addPositionButton()}</div>
            <div class="eval-error">
                <span class="error-text">{$t('eval.failed', { error: evalFailedMessage })}</span>
            </div>
        </div>
    {:else}
        <div class="eval-content">
            <!-- The strip, a full-width line on top (ADR-0020 rule 8): a badge qualifies the numbers below. -->
            <div class="badges-strip">
                {@render addPositionButton()}
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
                {#if genericDepthLabel}
                    <span class="badge badge-evaluated" title={$t('analysis.analysisDepth')}>{genericDepthLabel}</span>
                {/if}
                <button class="eval-engine-badge" onclick={openGammonNetRepo} title={$t('eval.engineTooltip')} aria-label={$t('eval.engineTooltip')}>?</button>
                <label class="challenge-toggle" title={$t('epc.challengeTooltip')}>
                    <input type="checkbox" checked={challenge} onchange={toggleChallenge} />
                    <span>{$t('epc.challenge')}</span>
                </label>
            </div>

            <!-- Facts and the one decision block (ADR-0017 rule 2); facts stacked on
                 one grid (ADR-0021), flex-wrap only for a hand-narrowed panel. -->
            <div class="top-row">
                {#if showFactsTable}
                    <PositionFactsTable
                        bottom={facts.bottom}
                        top={facts.top}
                        bottomEPC={data.bottomEPC}
                        topEPC={data.topEPC}
                        bottomPoints={data.bottomPoints}
                        topPoints={data.topPoints}
                        {maskedBottom}
                        {maskedTop}
                        onRevealBottom={() => reveal('bottom')}
                        onRevealTop={() => reveal('top')}
                        showProbabilities={!hasDiceSet}
                    />
                {/if}

                {#if showDecision}
                    <!-- Défi masks in place: values and verdict become `···` (ADR-0020 rule 7). -->
                    <!-- svelte-ignore a11y_click_events_have_key_events -->
                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                    <div class="decision-cube" class:masked={maskedDecision} onclick={() => maskedDecision && reveal('decision')} title={maskedDecision ? $t('epc.clickToReveal') : undefined}>
                        <CubeVerdictTable {decision} cubeValue={$positionStore?.cube?.value ?? 0} showInfo={false} masked={maskedDecision} {isMoney} {jacoby} {beaver} {maxCube} />
                    </div>
                {/if}
            </div>

            <!-- The only scrolling region (ADR-0017). The Baseline row lives in this
                 table, so Défi masks it with the ranking (ADR-0018 rules 2, 6). -->
            {#if hasDiceSet}
                {#if maskedDecision}
                    <!-- A button for keyboard reveal; focus goes back to the panel, since the reveal removes it. -->
                    <button
                        type="button"
                        class="decision-cube-masked moves-masked"
                        onclick={() => {
                            reveal('decision');
                            panelEl?.focus({ preventScroll: true });
                        }}
                        title={$t('epc.clickToReveal')}>{HIDDEN}</button
                    >
                {:else}
                    <div class="moves-scroll">
                        <CandidateMovesTable moves={evalMoves} selectedMove={$selectedMoveStore} onRowClick={handleMoveRowClick} showProvenance={false} baseline={baselineFacts} {isMoney} />
                        {#if evalMoves.length === 0}
                            <div class="eval-placeholder">{evalRefused ? $t('cube.refused') : $t('eval.pending')}</div>
                        {/if}
                    </div>
                {/if}
            {/if}
        </div>
    {/if}
</section>

<style>
    .eval-panel {
        height: 100%;
        box-sizing: border-box;
        /* Only .moves-scroll scrolls (ADR-0017). */
        overflow: hidden;
        padding: 3px 14px;
        font-family: var(--font-family-ui);
        font-size: var(--font-size-base);
        /* For the child tables' @container rules. */
        container-type: inline-size;
        /* A focus target for j/k, not a control. */
        outline: none;
    }

    .eval-inactive {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: var(--color-text-muted);
    }

    .eval-inactive-message {
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

    .eval-error {
        display: flex;
        align-items: center;
        justify-content: center;
        height: 100%;
    }

    .error-text {
        color: var(--color-danger);
        font-size: var(--font-size-base);
    }

    .eval-content {
        height: 100%;
        display: flex;
        flex-direction: column;
        gap: 6px;
    }

    /* Under the strip, the error message takes what is left of the panel. */
    .eval-content > .eval-error {
        flex: 1 1 auto;
        height: auto;
    }

    /* The strip (ADR-0020 rule 8): its own right-aligned line. */
    .badges-strip {
        flex: 0 0 auto;
        display: flex;
        align-items: center;
        justify-content: flex-end;
        gap: 8px;
        flex-wrap: wrap;
    }

    /* Facts (≤ 561 px, ADR-0021) beside the decision block; wrap is a fallback. */
    .top-row {
        flex: 0 0 auto;
        display: flex;
        flex-wrap: wrap;
        align-items: flex-start;
        gap: 8px 20px;
    }

    .eval-placeholder {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .decision-cube {
        display: flex;
        font-size: var(--font-size-base);
    }

    .decision-cube.masked {
        cursor: pointer;
    }

    .decision-cube.masked:hover {
        background: var(--color-surface-alt);
    }

    .decision-cube-masked {
        padding: 0;
        border: none;
        background: none;
        display: flex;
        align-items: center;
        justify-content: center;
        min-width: 180px;
        align-self: stretch;
        color: var(--color-text-muted);
        letter-spacing: 2px;
        cursor: pointer;
        font-size: var(--font-size-base);
    }

    .decision-cube-masked:hover {
        background: var(--color-surface-alt);
    }

    /* Masked Baseline+list (ADR-0018 rule 6): fills the region so height holds. */
    .moves-masked {
        flex: 1 1 auto;
        min-height: 0;
        width: 100%;
    }

    /* Labelled, not an icon, for discoverability; shaped like the badges. */
    .add-position {
        display: inline-flex;
        align-items: center;
        gap: var(--space-1);
        padding: 0 var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: 9px;
        background: var(--color-surface);
        color: var(--color-primary);
        font-size: var(--font-size-small);
        font-weight: 600;
        white-space: nowrap;
        cursor: pointer;
    }

    .add-position:hover:not(:disabled) {
        border-color: var(--color-primary);
    }

    .add-position:disabled {
        color: var(--color-text-muted);
        cursor: default;
    }

    .add-position svg {
        width: 14px;
        height: 14px;
        flex-shrink: 0;
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

    /* Third regime (ADR-0012): blue-leaning, neither exact green nor estimated amber. */
    .badge-evaluated {
        background: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface));
        border: 1px solid color-mix(in srgb, var(--color-primary) 30%, var(--color-surface));
        color: color-mix(in srgb, var(--color-primary) 80%, var(--color-text));
    }

    /* Composite exact + evaluated (ADR-0017 decision 4). */
    .badge-composite {
        background: linear-gradient(90deg, #e5f3e8 0 50%, #e8f0fe 50% 100%);
        border: 1px solid #c4d8f5;
        color: var(--color-primary);
    }

    /* One-character engine link; full attribution in the help. */
    .eval-engine-badge {
        width: 14px;
        height: 14px;
        line-height: 14px;
        padding: 0;
        border: 1px solid var(--color-border);
        border-radius: 50%;
        background: transparent;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
        text-align: center;
        cursor: pointer;
    }

    .eval-engine-badge:hover {
        color: var(--color-primary);
        border-color: var(--color-primary);
    }

    .challenge-toggle {
        display: flex;
        align-items: center;
        gap: 5px;
        cursor: pointer;
        color: var(--color-text-muted);
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
</style>
