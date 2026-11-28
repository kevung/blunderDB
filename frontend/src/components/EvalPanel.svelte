<script>
    import { onMount, onDestroy, untrack } from 'svelte';
    import { statusBarModeStore, MODAL, openModal, configInitialTabStore } from '../stores/uiStore';
    import { epcDataStore, epcChallengeStore, epcRevealedStore, resetEpcReveal } from '../stores/epcStore';
    import { positionStore } from '../stores/positionStore';
    import { resetEvalBoard, startingBoard } from '../services/boardReset.js';
    import { selectedMoveStore, evalAnalysisStore, evalVerdictStore } from '../stores/analysisStore';
    import { databasePathStore } from '../stores/databaseStore';
    import { positionRefusal } from '../services/positionRefusal.js';
    import { saveScratchBoard } from '../services/scratchBoard.js';
    import { GetEpcChallenge, SaveEpcChallenge, GetGammonNetDisplayPly, GetGammonNetPruneK, GetGammonNetCandidates } from '../../wailsjs/go/main/Config.js';
    import { EvaluatePositionImmediate, StartEvaluationAtRest, StartPlaysEvaluationAtRest, CancelEvaluationAtRest } from '../../wailsjs/go/gui/App.js';
    import { evalCache, evalCacheKey, equityDeltas, mergeMoves, plyOf } from '../services/evalCache.js';
    import { EventsOn, BrowserOpenURL } from '../../wailsjs/runtime/runtime.js';
    import { logger } from '../utils/logger.js';
    import { isBareLetter, isRolloutKey } from '../utils/keys.js';
    import { onChange } from '../utils/onChange.js';
    import { t } from '../i18n';
    import { moverFactsToSides } from '../utils/positionFacts.js';
    import { cubeDecision, cubeTurnability, isMoneyPosition, DECISION_STATE } from '../utils/cubeDecision.js';
    import CandidateMovesTable from './CandidateMovesTable.svelte';
    import CubeVerdictTable from './CubeVerdictTable.svelte';
    import PositionFactsTable from './PositionFactsTable.svelte';
    import RolloutStrip from './RolloutStrip.svelte';
    import RolloutResults from './RolloutResults.svelte';
    import ContextMenu from './ContextMenu.svelte';
    import PanelHeader from './panels/PanelHeader.svelte';
    import EmptyState from './panels/EmptyState.svelte';
    import { rolloutStore } from '../stores/rolloutStore.js';
    import { toggleRollout, cancelRollout, ensureRolloutEvents, syncRolloutStatus, rolloutOnBoard } from '../services/rolloutService.js';
    import { analysisMenuItems } from '../services/analysisMenu.js';
    import { evalViewStore, rollTrailStore, EVAL_VIEW } from '../stores/mapStore.js';
    import { seedSignature, trailPosition } from '../services/mapService.js';
    import { VIEW_LABELS } from '../utils/scoreMap.js';
    import RollMapView from './RollMapView.svelte';
    import ScoreMapView from './ScoreMapView.svelte';
    import { exportBoardImage } from '../services/clipboardService.js';
    import { rolloutsByMove, cubeRollouts } from '../utils/rolloutRows.js';
    import { clickSelection, contextSelection, selectedPlays } from '../utils/moveSelection.js';

    let isActive = $derived($statusBarModeStore === 'EVAL');

    // « Ajouter à la base » writes the position via saveScratchBoard(), with
    // the verdict on screen (evalVerdictStore). Disabled, with the reason as tooltip, when there is
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

    let evalMoves = $state(/** @type {any[]} */ ([]));
    let evalCubeAnalysis = $state(/** @type {any} */ (null));
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
    let evalRaceOverride = $state(/** @type {any} */ (null));
    // Pre-roll fact vector (ADR-0017), mover-relative; converted to bottom/top below.
    let evalPreRoll = $state(/** @type {any} */ (null));
    // The board the result on screen was computed for, and the match equity
    // table that valued it: CTRL-S stores the verdict only on that board and
    // under that table (evalVerdictStore).
    let evalSignature = $state('');
    let evalMetId = $state(0);

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
    /** @type {ReturnType<typeof setTimeout> | null} */
    let evalRestTimer = null;
    let evalGeneration = 0; // guards a late "done" against a position the user already left
    let escalationSignature = ''; // the board of the escalation evalGeneration names
    // What the search evalGeneration names is for: the at-rest escalation ('auto'), a re-run of
    // the position at the panel's depth ('local'), or of the selected plays only ('plays'); key
    // is its evalCache entry, before the rows on screen when it started.
    /** @type {{ kind: 'auto' | 'local' | 'plays', key: string, before: any[] }} */
    let pendingRun = { kind: 'auto', key: '', before: [] };

    const PLY_OPTIONS = [0, 1, 2, 3, 4];
    // The configured display depth, and the panel's own while the position stays: a new
    // position goes back to the configuration, so a deep check never slows the next gesture.
    let globalPly = $state(2);
    let localPly = $state(/** @type {number | null} */ (null));
    let panelPly = $derived(localPly ?? globalPly);
    let rerunning = $state(false);
    // The equity change of each play the last re-run touched (the Δ column), null before one.
    let evalDeltas = $state(/** @type {Map<string, number> | null} */ (null));

    /** The cache entry as an answer applyEvalResult takes. @param {any} entry @param {number} metId */
    function entryResult(entry, metId) {
        return { metId, moves: entry.moves, cube: entry.cube, cubeVerdict: entry.cubeVerdict, race: entry.race, preRoll: entry.preRoll };
    }

    // The signature is the effect's only tracked dependency; everything else is
    // untrack()ed, since reading the $state it writes loops
    // (effect_update_depth_exceeded).
    let positionSignature = $derived(JSON.stringify($positionStore ?? null));

    $effect(() => {
        const signature = positionSignature; // tracked: the position
        const active = isActive; // tracked: is the Eval tab even shown
        const view = $evalViewStore; // tracked: only the view on screen computes
        if (!signature || !active) return;
        untrack(() => {
            if (view === EVAL_VIEW.DECISION) runEvaluationEscalation();
            else stopEvaluation();
        });
    });

    // The roll map's trail leads the board: a change of position it did not make ends it. Armed
    // once the board has reached the trail's end, so the move that gets it there is not a change.
    let trailArmedFor = '';
    $effect(() => {
        const signature = positionSignature;
        const trail = $rollTrailStore;
        untrack(() => {
            if (!trail.steps.length) {
                trailArmedFor = '';
                return;
            }
            const expected = seedSignature(trailPosition(trail));
            if (signature === expected) trailArmedFor = expected;
            else if (trailArmedFor === expected) {
                trailArmedFor = '';
                rollTrailStore.set({ root: null, steps: [] });
            }
        });
    });

    // The maps take the search slot the at-rest evaluation holds: leave it, and drop its answers.
    function stopEvaluation() {
        evalGeneration += 1;
        if (evalRestTimer) {
            clearTimeout(evalRestTimer);
            evalRestTimer = null;
        }
        CancelEvaluationAtRest().catch(() => {});
    }

    // Leaving the tab clears the selected-move arrow, as AnalysisPanel does.
    $effect(
        onChange(
            () => isActive,
            (/** @type {boolean} */ v) => {
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
        pickedMoves = [];
        anchor = null;
        localPly = null;
        evalDeltas = null;
        rerunning = false;
        evalSettled = false;
        evalFailed = false;
        evalFailedMessage = '';

        evalGeneration += 1;
        const generation = evalGeneration;
        const signature = positionSignature;
        escalationSignature = signature;

        if (evalRestTimer) {
            clearTimeout(evalRestTimer);
            evalRestTimer = null;
        }
        CancelEvaluationAtRest().catch(() => {});

        GetGammonNetPruneK()
            .then((pruneK) =>
                GetGammonNetCandidates().then((candidates) =>
                    EvaluatePositionImmediate(/** @type {any} */ (pos), pruneK, candidates).then((result) => {
                        if (generation !== evalGeneration) return; // superseded while awaiting
                        // A deeper answer already computed for this position shows at once.
                        const entry = result?.refused ? null : evalCache.put(evalCacheKey(pos, { pruneK, candidates }), result);
                        applyEvalResult(entry ? entryResult(entry, result.metId ?? 0) : result, signature);
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
                    globalPly = ply;
                    const key = evalCacheKey(pos, { pruneK, candidates });
                    // Already as deep in the cache (shown by the 0-ply tier): no search.
                    if ((evalCache.get(key, evalMetId)?.ply ?? -1) >= ply) return;
                    pendingRun = { kind: 'auto', key, before: [] };
                    StartEvaluationAtRest(/** @type {any} */ (pos), ply, pruneK, candidates, generation).catch((error) => {
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

    // Re-runs at the panel's depth, without touching the configuration: the selected plays only
    // when there are (the others keep their rows and depths), else the whole position.
    /** @param {number} ply */
    function rerunAt(ply) {
        const pos = $positionStore;
        if (!isActive || !pos) return;
        localPly = ply;
        evalGeneration += 1;
        const generation = evalGeneration;
        escalationSignature = positionSignature;
        if (evalRestTimer) {
            clearTimeout(evalRestTimer);
            evalRestTimer = null;
        }
        const plays = rolloutSelection();
        rerunning = true;
        evalFailed = false;
        Promise.all([GetGammonNetPruneK(), GetGammonNetCandidates()])
            .then(([pruneK, candidates]) => {
                if (generation !== evalGeneration) return;
                const key = evalCacheKey(pos, { pruneK, candidates });
                pendingRun = { kind: plays.length ? 'plays' : 'local', key, before: evalMoves };
                const started = plays.length
                    ? StartPlaysEvaluationAtRest(/** @type {any} */ (pos), ply, pruneK, plays, generation)
                    : StartEvaluationAtRest(/** @type {any} */ (pos), ply, pruneK, candidates, generation);
                return started;
            })
            .catch((error) => {
                logger.error('gammonNet re-run failed to start:', error);
                if (generation !== evalGeneration) return;
                rerunning = false;
                evalFailed = true;
                evalFailedMessage = String(error);
            });
    }

    /** An answer of the search evalGeneration names, folded into the cache, then shown. @param {any} result */
    function applyRunResult(result) {
        const run = pendingRun;
        rerunning = false;
        const entry = result?.refused ? null : evalCache.put(run.key, result);
        if (run.kind === 'auto' || !entry) {
            applyEvalResult(entry ? entryResult(entry, result.metId ?? 0) : result, escalationSignature);
            return;
        }
        // A re-run shows what it computed, deeper or not, and how far each play moved.
        evalDeltas = equityDeltas(run.before, result.moves);
        if (run.kind === 'plays') {
            applyEvalResult({ metId: result.metId ?? 0, moves: mergeMoves(evalMoves, result.moves ?? []), race: evalRaceOverride, preRoll: evalPreRoll }, escalationSignature);
        } else {
            applyEvalResult(result, escalationSignature);
        }
    }

    // The depth label names what actually produced the result; "cancelled"
    // leaves the 0-ply result as is. Moves and Cube never come together: touch
    // only the one present, or the other side flashes back to pending.
    /** @param {any} result @param {string} signature the board it was computed for */
    function applyEvalResult(result, signature) {
        evalSignature = signature;
        evalMetId = result?.metId ?? 0;
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

    /** @type {Array<() => void>} */
    let unsubEval = [];
    onMount(() => {
        // A rollout outlives the panel: ask what runs, then listen.
        ensureRolloutEvents();
        syncRolloutStatus();
        GetEpcChallenge()
            .then((v) => epcChallengeStore.set(!!v))
            .catch(() => {});

        unsubEval = [
            // A search a newer gesture superseded can still answer: its tag
            // names the escalation it belongs to.
            EventsOn('gammonnet-eval:done', (result) => {
                if (result?.tag !== evalGeneration) return;
                applyRunResult(result);
            }),
            EventsOn('gammonnet-eval:cancelled', (e) => {
                if (e?.tag === evalGeneration) rerunning = false;
            }),
            EventsOn('gammonnet-eval:error', (e) => {
                if (e?.tag !== evalGeneration) return;
                rerunning = false;
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
        evalAnalysisStore.set(null);
        evalVerdictStore.set(null);
        unsubEval.forEach((off) => off && off());
    });

    function toggleChallenge(/** @type {Event} */ ev) {
        const e = /** @type {Event & { target: HTMLInputElement }} */ (ev);
        const on = e.target.checked;
        epcChallengeStore.set(on);
        resetEpcReveal();
        SaveEpcChallenge(on).catch(() => {});
    }

    function reveal(/** @type {string} */ zone) {
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

    // Rollouts of the scratch board (ADR-0060): it is never stored, so its rollout is run in
    // memory — no database, no write access needed — and shown while that board is on screen.
    // The selection and the menu are the Analysis panel's (utils/moveSelection.js, analysisMenu.js).
    let pickedMoves = $state(/** @type {string[]} */ ([]));
    let anchor = $state(/** @type {string | null} */ (null));
    let panelMenu = $state(/** @type {{ x: number, y: number, items: any[] } | null} */ (null));
    let rollout = $derived($rolloutStore);
    let onBoard = $derived(rolloutOnBoard(rollout, $positionStore, { unsaved: true }));
    let rolloutRows = $derived(rolloutsByMove({ unsaved: onBoard.unsaved, live: onBoard.live, position: $positionStore }));
    let cubeRolled = $derived(cubeRollouts([], onBoard.unsaved));
    const UNSAVED = { unsaved: true };

    function currentSelection() {
        return { picked: pickedMoves, anchor, selected: $selectedMoveStore };
    }

    /** @param {import('../utils/moveSelection.js').MoveSelection} sel */
    function applySelection(sel) {
        pickedMoves = sel.picked;
        anchor = sel.anchor;
        selectedMoveStore.set(sel.selected);
    }

    /** The plays a rollout started now would roll out; none is the whole position (its cube decision without dice). */
    function rolloutSelection() {
        return hasDiceSet ? selectedPlays(currentSelection()) : [];
    }

    /** @param {MouseEvent} event */
    function openPanelMenu(event) {
        event.preventDefault();
        // The board's own menu is off in EVAL (its right button edits), so the image files are offered here.
        const files = [
            { label: $t('board.menu.saveImageSVG'), onClick: () => exportBoardImage('svg') },
            { label: $t('board.menu.saveImagePNG'), onClick: () => exportBoardImage('png') }
        ];
        panelMenu = { x: event.clientX, y: event.clientY, items: [...analysisMenuItems(rolloutSelection(), UNSAVED), ...files] };
    }

    /** @param {{ move: string }} move @param {MouseEvent} event */
    function handleRowContextMenu(move, event) {
        applySelection(contextSelection(currentSelection(), move.move));
        openPanelMenu(event);
    }

    /** Right-click elsewhere in the panel: the selection, or the whole position. Défi keeps its answer hidden. */
    /** @param {MouseEvent} event */
    function handleContentContextMenu(event) {
        if (event.defaultPrevented || maskedDecision || $evalViewStore !== EVAL_VIEW.DECISION) return;
        openPanelMenu(event);
    }

    /** @param {{ move: string }} move @param {MouseEvent} [event] */
    function handleMoveRowClick(move, event) {
        applySelection(
            clickSelection(
                currentSelection(),
                move.move,
                event,
                evalMoves.map((m) => m.move)
            )
        );
        // Rows are plain <tr>s: focus the panel explicitly (WebKit and Chromium
        // differ on walking up to a focusable ancestor).
        panelEl?.focus({ preventScroll: true });
    }

    // j/k and arrows walk the candidates, Escape drops the selection, as in the
    // analysis panel. Required: keyboardService withholds these keys app-wide
    // while selectedMoveStore is set.
    function handleKeyDown(/** @type {KeyboardEvent} */ event) {
        if (/** @type {Element | null} */ (event.target)?.matches?.('input, select, textarea')) return;
        if (event.key === 'Escape') {
            // A running rollout is stopped first, then the selection cleared.
            if (rollout.running) {
                event.preventDefault();
                cancelRollout();
            } else if ($selectedMoveStore || pickedMoves.length) {
                event.preventDefault();
                pickedMoves = [];
                selectedMoveStore.set(null);
            }
            return;
        }

        // Shift+R rolls out the selected plays (the position when none is), or stops the rollout running.
        if (isRolloutKey(event) && isActive) {
            event.preventDefault();
            toggleRollout(rolloutSelection(), UNSAVED);
            return;
        }

        if (!$selectedMoveStore || evalMoves.length === 0) return;
        const currentIndex = evalMoves.findIndex((m) => m.move === $selectedMoveStore);

        if (isBareLetter(event, 'j') || event.key === 'ArrowDown') {
            event.preventDefault();
            if (currentIndex >= 0 && currentIndex < evalMoves.length - 1) {
                pickedMoves = [];
                selectedMoveStore.set(evalMoves[currentIndex + 1].move);
            }
        } else if (isBareLetter(event, 'k') || event.key === 'ArrowUp') {
            event.preventDefault();
            if (currentIndex > 0) {
                pickedMoves = [];
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
    const pct = (/** @type {number} */ x) => (100 * x).toFixed(2);

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

    // Nothing the engine said yet: one line saying so, not a grid of dashes. A refusal is an
    // answer, and the race tables answer before the engine does. Only while the engine is still
    // out (or the board is bare): a settled empty answer shows its table, never an endless wait.
    let boardEmpty = $derived(!($positionStore?.board?.points ?? []).some((p) => p?.checkers > 0));
    let noResult = $derived(!data.race && !evalRefused && !evalPreRoll && !evalCubeAnalysis && evalMoves.length === 0);
    let nothingYet = $derived(noResult && (boardEmpty || !evalSettled));

    // PositionFactsTable (ADR-0018 rule 1) would be empty with dice off a race: not mounted.
    let showFactsTable = $derived(!!data.race || !hasDiceSet);

    // What the panel shows, handed to the image copy (C-X C-X). The facts come from
    // the pre-roll payload (fractions) when the cube record has none, as on a race.
    $effect(() => {
        evalAnalysisStore.set(isActive ? evalAnalysisSnapshot() : null);
    });

    // What CTRL-S stores beside the board: the engine's own output, never the
    // panel's display shapes (a race table, facts folded into the cube block).
    $effect(() => {
        evalVerdictStore.set(isActive && evalSettled ? evalVerdictSnapshot() : null);
    });

    function evalVerdictSnapshot() {
        if (evalRefused) return null;
        const signature = evalSignature;
        const metId = evalMetId;
        if (hasDiceSet) {
            return evalMoves.length ? { signature, metId, analysis: { analysisType: 'CheckerMove', checkerAnalysis: { moves: evalMoves } } } : null;
        }
        return evalCubeAnalysis ? { signature, metId, analysis: { analysisType: 'DoublingCube', doublingCubeAnalysis: evalCubeAnalysis } } : null;
    }

    function evalAnalysisSnapshot() {
        if (evalRefused) return null;
        if (hasDiceSet) {
            return evalMoves.length ? { analysisType: 'CheckerMove', analysisEngineVersion: 'gammonNet', checkerAnalysis: { moves: evalMoves } } : null;
        }
        if (decision.state === DECISION_STATE.PENDING) return null;
        const f = baselineFacts;
        const pctOf = (/** @type {number | null | undefined} */ x) => (x == null ? undefined : 100 * x);
        const fromFacts = f
            ? {
                  playerWinChances: pctOf(f.playerWinChance),
                  playerGammonChances: pctOf(f.playerGammonChance),
                  playerBackgammonChances: pctOf(f.playerBackgammonChance),
                  opponentWinChances: pctOf(f.opponentWinChance),
                  opponentGammonChances: pctOf(f.opponentGammonChance),
                  opponentBackgammonChances: pctOf(f.opponentBackgammonChance),
                  cubelessNoDoubleEquity: f.cubelessEquity ?? undefined
              }
            : {};
        return {
            analysisType: 'DoublingCube',
            analysisEngineVersion: 'gammonNet',
            checkerAnalysis: { moves: [] },
            doublingCubeAnalysis: { ...fromFacts, ...(evalCubeAnalysis ?? {}) },
            decision
        };
    }

    // Off a race, depth is named once in the strip (ADR-0018 rule 4).
    // Rows from several depths (a targeted re-run) each name theirs, and the strip the span.
    let moveDepths = $derived([...new Set(evalMoves.map((m) => plyOf(m.analysisDepth)))].filter((p) => p >= 0).sort((a, b) => a - b));
    let mixedDepths = $derived(moveDepths.length > 1);
    let genericDepthLabel = $derived(
        data.race ? null : hasDiceSet ? (mixedDepths ? `${moveDepths[0]}–${moveDepths.at(-1)}-ply` : (evalMoves[0]?.analysisDepth ?? null)) : (evalCubeAnalysis?.analysisDepth ?? null)
    );
    let rerunPlays = $derived(hasDiceSet ? selectedPlays({ picked: pickedMoves, anchor, selected: $selectedMoveStore }).length : 0);
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

<!-- The band (ADR-0085): what the numbers are, then Défi, then keeping the position — the gesture
     that closes a check. -->
{#snippet evalHeader(/** @type {boolean} */ withBadges)}
    <PanelHeader title={$t('tabbedPanel.eval')}>
        {#if withBadges}
            <!-- In the strip, not on a line of its own: the panel must not scroll at its default size. -->
            <div class="view-switch" role="group" aria-label={$t('maps.view.label')}>
                {#each [EVAL_VIEW.DECISION, EVAL_VIEW.SCORES, EVAL_VIEW.ROLLS] as view (view)}
                    <button type="button" aria-pressed={$evalViewStore === view} onclick={() => evalViewStore.set(view)}>{$t(VIEW_LABELS[view])}</button>
                {/each}
            </div>
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
            {#if $evalViewStore === EVAL_VIEW.DECISION}
                <span class="depth-rerun">
                    <select
                        data-testid="eval-depth"
                        value={panelPly}
                        title={$t('eval.depthSelect')}
                        aria-label={$t('eval.depthSelect')}
                        onchange={(e) => rerunAt(Number(/** @type {HTMLSelectElement} */ (e.currentTarget).value))}
                    >
                        {#each PLY_OPTIONS as ply (ply)}
                            <option value={ply}>{$t('config.gammonnetPly', { n: ply })}</option>
                        {/each}
                    </select>
                    <button
                        type="button"
                        class="board-reset"
                        class:running={rerunning}
                        data-testid="eval-rerun"
                        onclick={() => rerunAt(panelPly)}
                        title={rerunPlays ? $t('eval.rerunPlays', { n: rerunPlays }) : $t('eval.rerun')}
                        aria-label={rerunPlays ? $t('eval.rerunPlays', { n: rerunPlays }) : $t('eval.rerun')}
                    >
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
                            <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 0 1 0 1.972l-11.54 6.347a1.125 1.125 0 0 1-1.667-.986V5.653Z"
                            />
                        </svg>
                        {#if rerunPlays}<span class="rerun-count">{rerunPlays}</span>{/if}
                    </button>
                </span>
            {/if}
        {/if}
        <button class="eval-engine-badge" onclick={openGammonNetRepo} title={$t('eval.engineTooltip')} aria-label={$t('eval.engineTooltip')}>?</button>
        {#snippet actions()}
            <label class="challenge-toggle" title={$t('epc.challengeTooltip')}>
                <input type="checkbox" checked={challenge} onchange={toggleChallenge} />
                <span>{$t('epc.challenge')}</span>
            </label>
            <span class="board-resets">
                <button
                    type="button"
                    class="board-reset"
                    data-testid="eval-start-position"
                    onclick={() => startingBoard('EVAL')}
                    title={$t('board.menu.startingPosition')}
                    aria-label={$t('board.menu.startingPosition')}
                >
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99"
                        />
                    </svg>
                </button>
                <button type="button" class="board-reset" data-testid="eval-clear-board" onclick={resetEvalBoard} title={$t('board.menu.clearPosition')} aria-label={$t('board.menu.clearPosition')}>
                    <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" aria-hidden="true">
                        <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0"
                        />
                    </svg>
                </button>
            </span>
            {@render addPositionButton()}
        {/snippet}
    </PanelHeader>
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
            {@render evalHeader(false)}
            <div class="eval-error">
                <span class="error-text">{data.error}</span>
            </div>
        </div>
    {:else if evalFailed}
        <div class="eval-content">
            {@render evalHeader(false)}
            <div class="eval-error">
                <span class="error-text">{$t('eval.failed', { error: evalFailedMessage })}</span>
            </div>
        </div>
    {:else}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="eval-content" oncontextmenu={handleContentContextMenu}>
            <!-- The strip, a full-width line on top (ADR-0020 rule 8): a badge qualifies the numbers below. -->
            {@render evalHeader(true)}

            {#if $evalViewStore === EVAL_VIEW.SCORES}
                <ScoreMapView active={isActive} />
            {:else if $evalViewStore === EVAL_VIEW.ROLLS}
                <RollMapView active={isActive} />
            {:else if nothingYet}
                <!-- The race facts (EPC, pips) do not wait for the engine: only its part is empty. -->
                {#if data.bottomEPC || data.topEPC}
                    <div class="top-row">
                        <PositionFactsTable
                            bottomEPC={data.bottomEPC}
                            topEPC={data.topEPC}
                            bottomPoints={data.bottomPoints}
                            topPoints={data.topPoints}
                            {maskedBottom}
                            {maskedTop}
                            onRevealBottom={() => reveal('bottom')}
                            onRevealTop={() => reveal('top')}
                            showProbabilities={false}
                        />
                    </div>
                {/if}
                <EmptyState text={boardEmpty ? $t('eval.emptyBoard') : $t('eval.evaluating')} actions={false} />
            {:else}
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

                {#if showDecision && !maskedDecision && (cubeRolled.length || onBoard.live.length)}
                    <!-- The cube decision has no play row: its rollout is read below it. -->
                    <RolloutResults rollouts={cubeRolled} live={onBoard.live} liveGames={rollout.games} liveMaxGames={rollout.maxGames} {isMoney} />
                {/if}

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
                            <CandidateMovesTable
                                moves={evalMoves}
                                selectedMove={$selectedMoveStore}
                                selectedMoves={pickedMoves}
                                rollouts={rolloutRows}
                                position={$positionStore}
                                onRowClick={handleMoveRowClick}
                                onRowContextMenu={handleRowContextMenu}
                                showProvenance={false}
                                showDepth={mixedDepths}
                                deltas={evalDeltas}
                                baseline={baselineFacts}
                                {isMoney}
                            />
                            {#if evalMoves.length === 0}
                                <div class="eval-placeholder">{evalRefused ? $t('cube.refused') : $t('eval.pending')}</div>
                            {/if}
                        </div>
                    {/if}
                {/if}
            {/if}
            <RolloutStrip here={onBoard.here} />
        </div>
    {/if}
    {#if panelMenu}
        <ContextMenu x={panelMenu.x} y={panelMenu.y} items={panelMenu.items} onClose={() => (panelMenu = null)} />
    {/if}
</section>

<style>
    .view-switch {
        display: flex;
        gap: 0;
    }
    .view-switch button {
        padding: 2px var(--space-3);
        border: 1px solid var(--color-border);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }
    .view-switch button:first-child {
        border-radius: var(--radius) 0 0 var(--radius);
    }
    .view-switch button:last-child {
        border-radius: 0 var(--radius) var(--radius) 0;
    }
    .view-switch button[aria-pressed='true'] {
        background: var(--color-primary);
        border-color: var(--color-primary);
        color: var(--color-surface);
        font-weight: 700;
    }
    .eval-panel {
        /* Interface chrome is not text to copy; fields below opt back in. */
        user-select: none;
        -webkit-user-select: none;
        height: 100%;
        box-sizing: border-box;
        /* Only .moves-scroll scrolls (ADR-0017). */
        overflow: hidden;
        padding: 0;
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
        box-sizing: border-box;
        display: flex;
        flex-direction: column;
        gap: 6px;
        padding: 0 var(--space-2) var(--space-1);
    }

    /* The band runs edge to edge over the padded content. */
    .eval-content > :global(.panel-header) {
        margin: 0 calc(-1 * var(--space-2));
    }

    /* Under the strip, the error message takes what is left of the panel. */
    .eval-content > .eval-error {
        flex: 1 1 auto;
        height: auto;
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
        padding: 1px var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-primary);
        font-size: var(--font-size-small);
        font-weight: 600;
        white-space: nowrap;
        cursor: pointer;
    }

    .board-resets {
        display: inline-flex;
        gap: 2px;
    }

    .board-reset {
        display: inline-flex;
        align-items: center;
        padding: 1px 4px;
        border: 1px solid transparent;
        border-radius: var(--radius);
        background: transparent;
        color: var(--color-text-muted);
        cursor: pointer;
    }

    .board-reset:hover {
        color: var(--color-primary);
        border-color: var(--color-border);
    }

    .board-reset svg {
        width: 14px;
        height: 14px;
    }

    .depth-rerun {
        display: inline-flex;
        align-items: center;
        gap: 2px;
    }

    .depth-rerun select {
        font-size: var(--font-size-small);
    }

    .board-reset.running {
        color: var(--color-primary);
        animation: rerun-pulse 1s ease-in-out infinite;
    }

    @keyframes rerun-pulse {
        50% {
            opacity: 0.4;
        }
    }

    .rerun-count {
        font-size: var(--font-size-small);
        margin-left: 1px;
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

    .eval-panel input {
        user-select: text;
        -webkit-user-select: text;
    }
</style>
