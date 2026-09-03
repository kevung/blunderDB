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
    import { isBareLetter } from '../utils/keys.js';
    import { t } from '../i18n';
    import { moverFactsToSides } from '../utils/positionFacts.js';
    import { cubeDecision, cubeTurnability, isMoneyPosition } from '../utils/cubeDecision.js';
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
    // isMoneyPosition, not a second inline `!= -1` predicate (#190/C.3 point
    // 2): this used to read `score[0] !== -1 || score[1] !== -1`, which
    // agrees with the AND form everywhere except the malformed case — one
    // side carrying the money sentinel, the other a real away score — where
    // the two silently disagreed on whether the position is money or match.
    let isMoney = $derived(isMoneyPosition($positionStore));
    let hasScore = $derived(!isMoney);
    let jacoby = $derived(isMoney && $positionStore?.has_jacoby === 1);
    let beaver = $derived(isMoney && $positionStore?.has_beaver === 1);

    let evalMoves = $state([]);
    let evalCubeAnalysis = $state(null);
    // The cube verdict as a VALUE (ADR-0020 rule 3), beside the analysis's own
    // BestCubeAction string: the string is what an importer stored, in its
    // engine's words; this is ours, so it is translated and it keeps "too
    // good", which cubeActionLabel has to flatten.
    let evalCubeVerdict = $state('');
    // The engine declined this position outright — a match score beyond the
    // MET's horizon. Data, not a rejected promise (ADR-0020 rule 4): a refusal
    // is a state the panel names, and it used to arrive as an error the
    // frontend logged and swallowed, leaving the previous position's numbers
    // on screen under a placeholder promising an evaluation was coming.
    let evalRefused = $state(false);
    // Whether gammonNet has answered for the position currently on the board.
    // Distinguishes "estimated, and the evaluated regime has not spoken yet"
    // (pending) from "estimated, and it declined" (no decision) — see
    // cubeDecision's `settled`.
    let evalSettled = $state(false);
    // The engine did not decline the position — it never answered at all: the
    // Wails call rejected, or a `gammonnet-eval:error` event arrived for the
    // in-flight evaluation-at-rest. Before this state existed, only
    // `logger.error` (invisible in production) marked the difference, and the
    // panel stayed on `eval.pending` forever — the residual debt from
    // ADR-0017 this fixes. `evalFailedMessage` carries the error text shown to
    // the user; both are cleared the moment a new evaluation starts or one
    // succeeds (`applyEvalResult`).
    let evalFailed = $state(false);
    let evalFailedMessage = $state('');
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

    // The depth label on the applied result always says what actually
    // produced it (0-ply or the display depth) — never a depth that was
    // requested but superseded before it ran. A "cancelled" event simply
    // leaves whatever 0-ply result is already showing untouched.
    //
    // Moves and Cube are never both present (gammonnet_eval.go's
    // GammonNetEvalResult: one or the other, `omitempty` elides the unused
    // one). Only touch the field the result actually carries — overwriting
    // the other with its own "nothing yet" value (`[]`/`null`) would flash
    // the pending placeholder every time a checker-play gesture follows a
    // cube gesture on the same position, even though that side's last real
    // evaluation is still perfectly valid.
    function applyEvalResult(result) {
        evalSettled = true;
        evalFailed = false;
        evalFailedMessage = '';
        evalRefused = !!result?.refused;
        if (evalRefused) {
            // Nothing this build can say about this position: drop both sides
            // rather than leave the previous position's answer standing under
            // a state that says there is none.
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

    // #131: a discreet, single-word attribution — the engine's name is the
    // link itself, never a sentence. Full credit (Strehl for the network,
    // gammonNet for the search/MET/cube configuration around it, ADR-0011)
    // lives in the Acknowledgements section of the in-app help, not here.
    function openGammonNetRepo() {
        BrowserOpenURL('https://github.com/kevung/gammonNet');
    }

    // The panel element itself: focus target for the keyboard navigation
    // below (a click on a row hands it the keyboard).
    let panelEl;

    function handleMoveRowClick(move) {
        if ($selectedMoveStore === move.move) {
            selectedMoveStore.set(null);
        } else {
            selectedMoveStore.set(move.move);
        }
        // The click is also what hands this panel the keyboard: the rows are
        // plain <tr>s, so focus would otherwise stay wherever it was and the
        // handler below would never see a key. Explicit rather than relying
        // on the browser walking up to the nearest focusable ancestor, which
        // WebKit and Chromium do not do alike.
        panelEl?.focus({ preventScroll: true });
    }

    // Walking the candidate list with the keyboard, exactly as the analysis
    // panel does it (doc/source/raccourcis.rst): once a move is selected,
    // j/BAS and k/HAUT move the selection — and with it the board's arrows —
    // one rank at a time, Escape drops it. The list here is the evaluation's
    // own ranking (never re-sorted, this panel offers no sort), so the order
    // walked is the order shown.
    //
    // This handler is not a convenience: keyboardService withholds
    // j/k/arrows app-wide while selectedMoveStore is set, so in a panel that
    // shows candidates and does not handle them itself, those keys did
    // nothing at all.
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
    const pct = (x) => (100 * x).toFixed(2);

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

    // The pre-roll vector in the axis of the list it heads (ADR-0018 rule
    // 2): mover-relative — the SAME frame as a move row's own fields
    // (domain.CheckerMove), never converted to bottom/top. Only ever handed
    // to CandidateMovesTable, and only while dice are set; PositionFactsTable
    // keeps the per-side reading (facts above) the rest of the time.
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

    // One cube Decision, one shape, whatever regime produced it (ADR-0020).
    // The two source shapes — race.Money on a bearoff, DoublingCubeAnalysis
    // everywhere else — become one object here, in the single place this panel
    // already composes the exact/evaluated merge above. The block is shown
    // whenever the board is asking a cube question at all, i.e. no dice
    // (ADR-0017 rule 2), and its own state cell says whether there is an
    // answer, none to be had, or a refusal.
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

    // PositionFactsTable now carries only per-side facts (ADR-0018 rule 1):
    // the race block always, the probability vector only when there is no
    // list to read it against. With dice on a non-race position neither
    // applies, and the table would be empty — so it is not mounted at all,
    // leaving the content row genuinely empty rather than holding a table with
    // nothing in it.
    let showFactsTable = $derived(!!data.race || !hasDiceSet);

    // Depth is already named by the race regime badge in the strip when the
    // position is a race (ADR-0012). Off a race, CubeVerdictTable's own
    // depth/engine footer is hidden in this panel (ADR-0018 rule 4) since
    // every row of a live evaluation shares one depth/engine — so it is
    // named once here instead.
    let genericDepthLabel = $derived(data.race ? null : hasDiceSet ? (evalMoves[0]?.analysisDepth ?? null) : (evalCubeAnalysis?.analysisDepth ?? null));
</script>

<!-- A <section> rather than a <div>: the panel takes focus and listens for
     keys (handleKeyDown), which is a landmark's business and a static
     element's a11y warning — the same shape AnalysisPanel already has. -->
<section class="epc-panel" bind:this={panelEl} role="region" aria-label={$t('eval.panelLabel')} tabindex="-1" onkeydown={handleKeyDown}>
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
    {:else if evalFailed}
        <div class="epc-error">
            <span class="error-text">{$t('eval.failed', { error: evalFailedMessage })}</span>
        </div>
    {:else}
        <div class="epc-content">
            <!-- The strip: regime badge, depth, engine link and the Défi
                 toggle, on their own full-width line (ADR-0018 rule 4, applied
                 as written by ADR-0020 rule 8). It was a third member of the
                 content row until then, held in the corner by a
                 `margin-left: auto` that manufactured a band of white across
                 the middle whenever the row had something in it. It stays at
                 the top: a badge qualifies the numbers below it. -->
            <div class="badges-strip">
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

            <!-- Content row: the facts stack and the one decision block the
                 board asks for (never both a cube verdict and a checker
                 decision — ADR-0017 rule 2). The facts are two blocks stacked
                 on one column grid rather than one line of ten columns
                 (ADR-0021), which is what keeps the decision ON this row at the
                 default window size in all nine languages; flex-wrap stays as
                 the safety net for a panel narrowed by hand. Nothing pushes
                 anything to a far edge, so no width can produce a void. -->
            <div class="top-row">
                {#if showFactsTable}
                    <PositionFactsTable
                        bottom={facts.bottom}
                        top={facts.top}
                        bottomEPC={data.bottomEPC}
                        topEPC={data.topEPC}
                        {maskedBottom}
                        {maskedTop}
                        onRevealBottom={() => reveal('bottom')}
                        onRevealTop={() => reveal('top')}
                        showProbabilities={!hasDiceSet}
                    />
                {/if}

                {#if showDecision}
                    <!-- Défi masks in place: the three rows stay, their values
                         and the verdict become `···`, and the best-row
                         emphasis goes with them (ADR-0020 rule 7). The opaque
                         stand-in this used to need is gone — its excuse was
                         that CubeVerdictTable was "a foreign component with
                         its own scoped CSS", which stopped being true when the
                         mask moved inside it. -->
                    <!-- svelte-ignore a11y_click_events_have_key_events -->
                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                    <div class="decision-cube" class:masked={maskedDecision} onclick={() => maskedDecision && reveal('decision')} title={maskedDecision ? $t('epc.clickToReveal') : undefined}>
                        <CubeVerdictTable {decision} cubeValue={$positionStore?.cube?.value ?? 0} showInfo={false} masked={maskedDecision} {isMoney} {jacoby} {beaver} />
                    </div>
                {/if}
            </div>

            <!-- The moves list is the only region that ever scrolls
                 (ADR-0017): its header stays sticky, and everything above
                 (facts, badges, and a race/cube decision when there is no
                 dice) stays on screen regardless of candidate count. With
                 dice set the Baseline row lives inside this same table
                 (ADR-0018 rule 2), so Défi masks the band and the ranking
                 together — the order IS the answer, there is no partial
                 reveal (ADR-0018 rule 6). -->
            {#if hasDiceSet}
                {#if maskedDecision}
                    <div class="decision-cube-masked moves-masked" onclick={() => reveal('decision')} title={$t('epc.clickToReveal')}>{HIDDEN}</div>
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
        /* The panel takes focus so the candidate list answers j/k (see
           handleKeyDown); it is a focus target, not a control. */
        outline: none;
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

    /* The strip (ADR-0020 rule 8): its own full-width line above the content,
       right-aligned, so nothing in the content row has to be pushed to a far
       edge to keep it in the corner — the rule that used to make the void. */
    .badges-strip {
        flex: 0 0 auto;
        display: flex;
        align-items: center;
        justify-content: flex-end;
        gap: 8px;
        flex-wrap: wrap;
    }

    /* Facts + the one decision block, side by side. Since ADR-0021 the facts
       are two blocks stacked on one column grid (561 px at worst) instead of a
       single line of ten columns (up to 880 px), so the pair fits the default
       panel in all nine languages and the decision no longer falls under the
       numbers it answers. The wrap is still there — one flow, driven by the
       panel's own width, not by whether it is docked at the bottom or the side
       — but it is now a fallback for a hand-narrowed panel, not the normal
       case. */
    .top-row {
        flex: 0 0 auto;
        display: flex;
        flex-wrap: wrap;
        align-items: flex-start;
        gap: 8px 20px;
    }

    .eval-placeholder {
        color: #888;
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
        background: #f5f5f5;
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

    /* The masked stand-in for the Baseline+list block (ADR-0018 rule 6):
       same look as the cube's mask, but filling the flexible region the
       real .moves-scroll would otherwise take, so masking never collapses
       the panel's height. */
    .moves-masked {
        flex: 1 1 auto;
        min-height: 0;
        width: 100%;
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
       to the neutral chips than to either the green "exact" or the amber
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
</style>
