<script>
    /** @typedef {import('../../wailsjs/go/models').domain.Match} Match */
    import { confirmAction } from '../services/confirmService.js';
    import { trashMatchEntry } from '../services/matchTrash.js';
    import { loadTournaments } from '../services/tournamentLoader.js';
    import { logger } from '../utils/logger.js';
    import { repairCounts, repairDiffers, repairDoneKey, repairMatchErrorKey } from '../utils/repairMatch.js';
    import { analysisChallengeStore, analysisChallengeHiddenStore } from '../stores/analysisChallengeStore.js';
    import { formatDate, formatDiceShort, GRADE_MARKS, indexMoveGrades, countGrades, fmtGradeCost } from '../utils/matchTable.js';
    import { createInlineEdit } from '../utils/inlineEdit.svelte.js';
    import { untrack, tick } from 'svelte';
    import { get } from 'svelte/store';
    import { SvelteSet } from 'svelte/reactivity';
    import {
        GetMatchByID,
        UpdateMatchComment,
        GetMatchMovePositions,
        GetGamesByMatch,
        LoadAnalysis,
        SetMatchTournamentByName,
        SetMatchVideoSource,
        RepairMatch,
        SaveLastVisitedPosition,
        GetMatchDetailStats,
        GetMatchMoveGrades,
        GetMatchDecisionLosses,
        GetMatchReview,
        GetMatchTimeSummary,
        GetMatchOrigin
    } from '../../wailsjs/go/database/Database.js';
    import { VideoSourceKind, OpenVideoExternally, PickTranscriptionVideo } from '../../wailsjs/go/gui/App.js';
    import VideoDock from './VideoDock.svelte';
    import { rateKeyDirection } from '../utils/videoRate.js';
    import { isBareLetter } from '../utils/keys.js';
    import EntityAutocomplete from './EntityAutocomplete.svelte';
    import CountLink from './panels/CountLink.svelte';
    import { loadPositionsFromSelection } from '../services/positionLoader.js';
    import { exportMatchMat } from '../services/exportService.js';
    import MatchCharts from './MatchCharts.svelte';
    import { fmtLoss, sortByLoss } from '../utils/decisionLoss.js';
    import MatchReview from './MatchReview.svelte';
    import MatchReviewDetails from './MatchReviewDetails.svelte';
    import MatchDetailHeader from './MatchDetailHeader.svelte';
    import MatchReviewDecisions from './MatchReviewDecisions.svelte';
    import { MATCH_TABS, rememberTab, rememberedTab } from '../utils/matchSheet.js';
    import MatchStatsTable from './MatchStatsTable.svelte';
    import { hasReview } from '../utils/matchReview.js';
    import MatchOrigin from './MatchOrigin.svelte';
    import { cumulativeClocks, fmtClock, fmtDuration, hasAnyDuration, sortByDuration } from '../utils/decisionTime.js';
    import { editMatchTranscription } from '../services/transcriptionSave.js';
    import { t, tMsg } from '../i18n';
    import { positionStore, matchContextStore, lastVisitedMatchStore } from '../stores/positionStore';
    import { statusBarModeStore, dbMutationCounterStore, statusBarTextStore, activeTabStore, commentTextStore } from '../stores/uiStore';
    import { analysisStore, selectedMoveStore } from '../stores/analysisStore';
    import { tournamentsStore } from '../stores/tournamentStore';
    import { matchListStore } from '../stores/matchListStore.js';
    import { libraryCountsStore } from '../stores/libraryCountsStore.js';
    import { readOfLibrary, isLibraryLeft, libraryMark, writingOnLibrary } from '../stores/groupActivityStore.js';

    /**
     * The sheet of one match: header, study summary, video and the Transcript | Charts | Review | Details | Info |
     * Stats tabs. It reads the match it shows and registers nothing global (no key listener, no shared list,
     * no shared sort): a host that wants keys forwards them to `handleSheetKey`.
     *
     * `matchId`: the match to show (null shows nothing).
     * `onDeleted(match)`: called once the sheet trashed its match, before the dependent stores are bumped.
     */
    let { matchId = null, onDeleted = null } = $props();
    // DOM ids are per sheet: two sheets (Matches, Tournaments) may be on screen together.
    const uid = $props.id();

    let lastVisitedMatch = $derived($lastVisitedMatchStore);
    let tournaments = $derived($tournamentsStore || []);

    /** @type {any} */
    let detailMatch = $state(untrack(() => get(matchListStore).rows.find((/** @type {any} */ m) => m.id === matchId) ?? null)); // Match currently shown, from the rows at once
    /** @type {any[]} */
    let detailMovePositions = $state([]); // MatchMovePosition[] for the detail match
    /** @type {any[]} */
    let detailGames = $state([]); // Game[] for the detail match
    /** @type {any[]} */
    let detailGrades = $state([]); // MoveGrade[] for the detail match
    /** @type {import('../../wailsjs/go/models').storage.MatchTimeSummary | null} */
    let detailTimes = $state(null);
    /** @type {any[] | null} the MWC loss of each decision */
    let detailLosses = $state(null);
    let detailReview = $state(/** @type {import('../../wailsjs/go/models').storage.MatchReview | null} */ (null));
    /** @type {import('../../wailsjs/go/models').duel.Origin | null} the Origin of a match played here */
    let detailOrigin = $state(null);
    // The decision under the pointer or the keyboard on either chart, by its Move.
    /** @type {number | null} */
    let hoveredMove = $state(null);
    /** @type {'' | 'asc' | 'desc'} the transcript's order by decision time */
    let timeSort = $state('');
    /** @type {'' | 'asc' | 'desc'} */
    let lossSort = $state('');
    let loadingDetail = $state(false);
    /** @type {any} */
    let detailStats = $state(null); // MatchDetailStats for the detail match
    let loadingStats = $state(false);

    // Inline tournament editing (autocomplete over the known tournaments)
    const tournamentEdit = createInlineEdit({
        onSave: async (id, value, mark) => {
            const name = value.trim();
            try {
                if (!(await writingOnLibrary(mark, async () => (await SetMatchTournamentByName(id, name), true)))) return;
                await matchListStore.refreshRow(id);
                await loadTournaments();
                statusBarTextStore.set(name ? tMsg('match.tournamentSet', { name }) : tMsg('match.tournamentCleared'));
            } catch (error) {
                logger.error('Error setting tournament:', error);
                statusBarTextStore.set(tMsg('match.errorSettingTournament'));
            }
        }
    });

    function startEditTournament(/** @type {Match} */ match, /** @type {Event} */ event) {
        event.stopPropagation();
        tournamentEdit.start(match.id, match.tournament_name || match.event || '');
    }

    async function selectTournamentOption(/** @type {{ name: string }} */ tournament) {
        tournamentEdit.draft = tournament.name;
        await tournamentEdit.save();
    }

    /** Whether the tournament field of the Info tab is being edited. */
    export function tournamentEditing() {
        return tournamentEdit.editingId !== null;
    }

    export function cancelTournamentEdit() {
        tournamentEdit.cancel();
    }

    // The match's video (ADR-0082): a file plays in a pane of this sheet, a web source opens
    // in the browser at the timestamped link.
    /** @type {'' | 'file' | 'youtube' | 'url'} */
    let videoKind = $state('');
    let videoOpen = $state(false);
    let videoStartMs = $state(0);
    /** @type {any} */
    let videoPane = $state(null);
    let videoOnBoard = $state(false);
    let videoDraft = $state('');
    const videoSource = $derived(detailMatch?.video_source || '');

    // The pane belongs to the match shown: another match closes it, a new source for the same
    // match (a relocated file) keeps it open.
    $effect(() => {
        void detailMatch?.id;
        videoOpen = false;
    });

    $effect(() => {
        const source = videoSource;
        videoDraft = source;
        if (!source) {
            videoKind = '';
            return;
        }
        let cancelled = false;
        VideoSourceKind(source)
            .then((k) => {
                if (!cancelled) videoKind = k || '';
            })
            .catch(() => {
                if (!cancelled) videoKind = '';
            });
        return () => {
            cancelled = true;
        };
    });

    // A Move's Repère: the dice fell, else the action was done. Absent is unknown, never zero.
    function moveTickMs(/** @type {Partial<import("../../wailsjs/go/models").domain.MatchMovePosition> | null | undefined} */ mp) {
        return mp?.roll_tick_ms ?? mp?.tick_ms ?? null;
    }

    // Open the video at the decision, one second before the dice fell.
    async function viewInVideo(/** @type {Partial<import("../../wailsjs/go/models").domain.MatchMovePosition> | null | undefined} */ mp) {
        const tick = moveTickMs(mp);
        if (!videoSource || tick === null) return;
        await openVideoAt(Math.max(0, tick - 1000));
    }

    async function openVideoAt(/** @type {number} */ ms) {
        if (videoKind === 'file') {
            videoStartMs = ms;
            if (videoOpen && videoPane) videoPane.seek(ms);
            videoOpen = true;
        } else if (videoKind === 'youtube' || videoKind === 'url') {
            try {
                await OpenVideoExternally(videoSource, ms);
            } catch (error) {
                logger.error('Error opening the video:', error);
                statusBarTextStore.set(tMsg('match.errorVideo'));
            }
        }
    }

    function toggleVideo() {
        if (videoKind === 'file' && videoOpen) videoOpen = false;
        else openVideoAt(0);
    }

    async function saveVideoSource(/** @type {string} */ source) {
        if (!detailMatch) return;
        const id = detailMatch.id;
        const value = source.trim();
        try {
            await SetMatchVideoSource(id, value);
            const video_source = value || undefined;
            Object.assign(detailMatch, { video_source });
            matchListStore.patchRow(id, { video_source });
            statusBarTextStore.set(tMsg(value ? 'match.videoSet' : 'match.videoDetached'));
        } catch (error) {
            logger.error('Error setting the video source:', error);
            statusBarTextStore.set(tMsg('match.errorVideo'));
        }
    }

    async function pickVideoFile() {
        try {
            const path = await PickTranscriptionVideo();
            if (path) await saveVideoSource(path);
        } catch (error) {
            logger.error('Error choosing the video:', error);
        }
    }

    // Inline match comment editing (detail pane, metadata view)
    const commentEdit = createInlineEdit({
        onSave: async (matchId, text, mark) => {
            try {
                if (!(await writingOnLibrary(mark, async () => (await UpdateMatchComment(matchId, text), true)))) return;
                // The backend signs the edit with « Votre nom »; read back who it names.
                const comment_author = (await GetMatchByID(matchId))?.comment_author || '';
                if (detailMatch && detailMatch.id === matchId) Object.assign(detailMatch, { comment: text, comment_author });
                matchListStore.patchRow(matchId, { comment: text, comment_author });
                statusBarTextStore.set(tMsg('match.commentUpdated'));
            } catch (error) {
                logger.error('Error updating comment:', error);
                statusBarTextStore.set(tMsg('match.errorUpdatingComment'));
            }
        }
    });

    /** "1855 (205) – 1600 (0)": each seat's rating and, when stated, its experience, player 1 first. */
    function formatRatings(/** @type {Match} */ m) {
        const seat = (/** @type {number | null | undefined} */ elo, /** @type {number | string | null | undefined} */ exp) =>
            elo == null ? '—' : `${Math.round(elo)}${exp == null ? '' : ` (${exp})`}`;
        return `${seat(m.player1_elo, m.player1_experience)} – ${seat(m.player2_elo, m.player2_experience)}`;
    }

    /** The optional rules a money session was played under: "Jacoby, Beaver", or none. */
    function formatSessionRules(/** @type {Match} */ m) {
        const rules = [m.has_jacoby && 'Jacoby', m.has_beaver && 'Beaver'].filter(Boolean);
        return rules.length ? rules.join(', ') : $t('match.rulesNone');
    }

    /** A match by id: from the loaded rows, else read on its own (it may lie beyond the first page). */
    async function findMatch(/** @type {number} */ id) {
        try {
            return (await GetMatchByID(id)) || null;
        } catch {
            return null;
        }
    }

    /** @type {Promise<void>} */
    let loadPromise = Promise.resolve();

    // The sheet follows the match its host designates.
    $effect(() => {
        const id = matchId;
        untrack(() => reload(id));
    });

    /** Reads the match shown again (after a repair or a swap of its players, which changed its rows). */
    export function reload(/** @type {number | null} */ id = matchId) {
        loadPromise = show(id);
        return loadPromise;
    }

    /** Resolves once the match designated by the host is on the sheet with its decisions. */
    export async function whenLoaded() {
        await tick();
        await loadPromise;
    }

    async function show(/** @type {number | null} */ id) {
        if (id == null) {
            detailMatch = null;
            return;
        }
        const loaded = get(matchListStore).rows.find((/** @type {any} */ m) => m.id === id);
        const match = loaded ?? (await findMatch(id));
        if (matchId !== id) return;
        if (match) await loadMatchDetail(match);
        else detailMatch = null;
    }

    async function loadMatchDetail(/** @type {Match} */ match) {
        if (!match) return;
        loadingDetail = true;
        detailMatch = match;
        detailStats = null; // reset stats when switching match
        try {
            const [movePositions, games, grades, times, origin, losses, review] = await Promise.all([
                GetMatchMovePositions(match.id),
                GetGamesByMatch(match.id),
                loadMoveGrades(match.id),
                loadTimeSummary(match.id),
                loadOrigin(match.id),
                loadDecisionLosses(match.id),
                loadReview(match.id)
            ]);
            if (detailMatch?.id !== match.id) return;
            detailMovePositions = movePositions || [];
            detailGames = games || [];
            detailGrades = grades;
            detailTimes = times;
            detailLosses = losses;
            detailReview = review;
            detailOrigin = origin;
        } catch (error) {
            logger.error('Error loading match detail:', error);
            detailMovePositions = [];
            detailGames = [];
            detailGrades = [];
            detailTimes = null;
            detailLosses = null;
            detailReview = null;
            detailOrigin = null;
        }
        loadingDetail = false;
    }

    // Marks are optional: on failure the transcript shows unmarked.
    /** @param {number} matchID */
    async function loadMoveGrades(matchID) {
        try {
            return (await GetMatchMoveGrades(matchID)) || [];
        } catch (error) {
            logger.error('Error loading move grades:', error);
            return [];
        }
    }

    // The time summary is optional too: without it the transcript shows no summary.
    /** @param {number} matchID */
    async function loadTimeSummary(matchID) {
        try {
            return (await GetMatchTimeSummary(matchID)) || null;
        } catch (error) {
            logger.error('Error loading time summary:', error);
            return null;
        }
    }

    // The per-decision MWC loss is optional too: without it no loss chart is drawn.
    /** @param {number} matchID */
    async function loadDecisionLosses(matchID) {
        try {
            return (await GetMatchDecisionLosses(matchID)) || null;
        } catch (error) {
            logger.error('Error loading decision losses:', error);
            return null;
        }
    }

    // The study summary is optional too: without it no review box is shown.
    /** @param {number} matchID */
    async function loadReview(matchID) {
        try {
            return (await GetMatchReview(matchID)) || null;
        } catch (error) {
            logger.error('Error loading match review:', error);
            return null;
        }
    }

    // The origin is optional too: a match not played here has none.
    /** @param {number} matchID */
    async function loadOrigin(matchID) {
        try {
            return (await GetMatchOrigin(matchID)) || null;
        } catch (error) {
            logger.error('Error loading match origin:', error);
            return null;
        }
    }

    function cycleTimeSort() {
        lossSort = '';
        timeSort = timeSort === '' ? 'desc' : timeSort === 'desc' ? 'asc' : '';
    }
    function cycleLossSort() {
        timeSort = '';
        lossSort = lossSort === '' ? 'desc' : lossSort === 'desc' ? 'asc' : '';
    }
    // One order at a time: by the time a decision took, by what it cost, or the match's own.
    /** @param {{ mp: any, globalIdx: number, grade: any, loss: number | null, difficulty: number | null, avoidable: boolean }[]} moves */
    const ordered = (moves) => (lossSort ? sortByLoss(moves, lossSort) : sortByDuration(moves, timeSort));

    // Grades follow the library thresholds: re-read when the library counter changes.
    $effect(() => {
        void $libraryCountsStore; // tracked dep: re-grade on a threshold change or an import
        untrack(() => {
            const match = detailMatch;
            if (!match) return;
            loadMoveGrades(match.id).then((grades) => {
                if (detailMatch && detailMatch.id === match.id) detailGrades = grades;
            });
        });
    });

    async function loadMatchStats(/** @type {Match} */ match) {
        if (!match || loadingStats) return;
        loadingStats = true;
        try {
            detailStats = await GetMatchDetailStats(match.id);
        } catch (error) {
            logger.error('Error loading match stats:', error);
            detailStats = null;
        }
        loadingStats = false;
    }

    // Called by the Stats section while it is open, again when the match
    // changes under it; only the match is tracked, not the loading state.
    function openStats() {
        const match = detailMatch;
        untrack(() => {
            if (match && !detailStats && !loadingStats) loadMatchStats(match);
        });
    }

    // The Time column appears once the match recorded any duration.
    let showTimes = $derived(hasAnyDuration(detailMovePositions.map((mp) => ({ mp }))));
    // Computed in match order, whatever the transcript's sort, indexed by globalIdx.
    let clocks = $derived(cumulativeClocks(detailMovePositions));
    // A time cell is never blank: no decision, or one the Arbiter played alone, reads as a dash.
    let showLosses = $derived(/** @type {any[]} */ (detailLosses ?? []).some((d) => d.mwc_loss != null));

    // The sheet's tabs: a tab with nothing to show is left out, and the sheet
    // falls back to the transcript without forgetting the tab the user chose.
    let chosenTab = $state(rememberedTab());
    let tabs = $derived(
        MATCH_TABS.filter((id) => {
            if (id === 'charts') return showTimes || showLosses;
            if (id === 'review') return hasReview(detailReview);
            if (id === 'details') return hasReview(detailReview) || showLosses;
            return true;
        })
    );
    let currentTab = $derived(tabs.includes(chosenTab) ? chosenTab : 'transcript');
    /** @type {Record<string, string>} */
    const TAB_LABELS = { transcript: 'match.tabTranscript', charts: 'match.tabCharts', review: 'match.tabReview', details: 'match.tabDetails', info: 'match.info', stats: 'match.tabStats' };

    /** @param {string} id */
    function selectTab(id) {
        chosenTab = id;
        rememberTab(id);
    }

    /** @param {KeyboardEvent} e */
    function onTabKey(e) {
        const i = tabs.indexOf(currentTab);
        const next =
            e.key === 'ArrowRight'
                ? tabs[(i + 1) % tabs.length]
                : e.key === 'ArrowLeft'
                  ? tabs[(i - 1 + tabs.length) % tabs.length]
                  : e.key === 'Home'
                    ? tabs[0]
                    : e.key === 'End'
                      ? tabs[tabs.length - 1]
                      : null;
        if (!next) return;
        // The arrows walk the tabs here, not the match.
        e.preventDefault();
        e.stopPropagation();
        selectTab(next);
        tick().then(() => document.getElementById(`${uid}-tab-${next}`)?.focus());
    }

    // The statistics are loaded when their tab is shown.
    $effect(() => {
        if (currentTab === 'stats') openStats();
    });
    const orDash = (/** @type {string} */ text) => text || '—';
    // Under a Cadence the clock column shows the reserve left, replayed by the backend with the
    // Arbiter's own arithmetic; without one, the time used so far.
    let remaining = $derived(detailOrigin?.clock?.remaining ?? null);
    let cadenceInfo = $derived.by(() => {
        const c = detailOrigin?.cadence_settings;
        if (!c) return null;
        const bankMS = detailOrigin?.clock?.start?.[0] ?? (c.reserve ? c.reserve * 1000 : null);
        return {
            name: c.name || '',
            delay: c.delay || 0,
            perPoint: c.reservePerPoint || 0,
            bank: bankMS === null ? '' : fmtClock(bankMS),
            timeOut: c.timeOut === 'lose_match' ? $t('match.timeOutLoseMatch') : $t('match.timeOutContinue')
        };
    });

    // Moves grouped by game, each with its precomputed globalIdx (an indexOf per
    // row would be quadratic).
    let transcriptGames = $derived.by(() => {
        if (!detailMovePositions.length) return [];
        const gradeByMove = indexMoveGrades(detailGrades);
        const lossByMove = new Map(/** @type {any[]} */ (detailLosses ?? []).map((d) => [d.move_id, d]));
        /** @type {Map<number, { mp: any, globalIdx: number, grade: any, loss: number | null, difficulty: number | null, avoidable: boolean }[]>} */
        // eslint-disable-next-line svelte/prefer-svelte-reactivity -- local temp inside $derived
        const gameMap = new Map();
        detailMovePositions.forEach((/** @type {any} */ mp, globalIdx) => {
            let moves = gameMap.get(mp.game_number);
            if (!moves) gameMap.set(mp.game_number, (moves = []));
            const d = lossByMove.get(mp.move_id);
            moves.push({ mp, globalIdx, grade: gradeByMove.get(mp.move_id), loss: d?.mwc_loss ?? null, difficulty: d?.difficulty ?? null, avoidable: !!d?.avoidable });
        });
        const result = [];
        for (const [gameNum, moves] of gameMap) {
            const gameInfo = detailGames.find((g) => g.game_number === gameNum);
            result.push({ gameNumber: gameNum, moves, gameInfo, marks: countGrades(moves) });
        }
        return result;
    });

    // A game's count of errors opens those positions (ADR-0085, G7). The link sits in the
    // game's <summary>: the click must not also fold or unfold the game.
    /**
     * @param {MouseEvent} event
     * @param {{ mp: any, grade: any }[]} moves
     * @param {'blunder' | 'error'} grade
     */
    function openGraded(event, moves, grade) {
        event.preventDefault();
        event.stopPropagation();
        const ids = moves.filter((m) => m.grade?.grade === grade && m.mp.position?.id).map((m) => m.mp.position.id);
        loadPositionsFromSelection([...new Set(ids)]);
    }

    // Games whose move table is mounted (collapsed ones render only <summary>).
    // A SvelteSet mutated in place; reseeded to the current game on match change.
    const openGames = new SvelteSet();

    $effect(() => {
        const moves = detailMovePositions; // tracked dep: reseed on a new match's moves
        untrack(() => {
            openGames.clear();
            if (!moves.length) return;
            const ctx = get(matchContextStore);
            let targetIndex = null;
            if (ctx.isMatchMode && detailMatch && ctx.matchID === detailMatch.id) {
                targetIndex = ctx.currentIndex;
            } else if (lastVisitedMatch && detailMatch && lastVisitedMatch.matchID === detailMatch.id) {
                targetIndex = lastVisitedMatch.currentIndex;
            }
            const targetMove = targetIndex != null ? moves[targetIndex] : null;
            const defaultGame = targetMove ? targetMove.game_number : moves[moves.length - 1].game_number;
            openGames.add(defaultGame);
        });
    });

    function setGameOpen(/** @type {number} */ gameNumber, /** @type {boolean} */ isOpen) {
        if (isOpen) openGames.add(gameNumber);
        else openGames.delete(gameNumber);
    }

    // Last game stepped into: reopening reacts to crossing into a game, so the
    // user can still collapse the current one.
    /** @type {number | null} */
    let lastCrossedGame = null;

    // In MATCH mode, crossing into a collapsed game reopens it (never closes any).
    $effect(() => {
        const ctx = $matchContextStore;
        if (!ctx.isMatchMode || !detailMatch || ctx.matchID !== detailMatch.id) {
            lastCrossedGame = null;
            return;
        }
        const move = detailMovePositions[ctx.currentIndex];
        if (!move) return;
        if (move.game_number === untrack(() => lastCrossedGame)) return; // still in the same game: leave the user's fold alone
        lastCrossedGame = move.game_number;
        untrack(() => {
            if (!openGames.has(move.game_number)) setGameOpen(move.game_number, true);
        });
    });

    // The decision a chart points at: shown as the transcript shows a row clicked, then brought into view.
    /** @param {number} index */
    async function jumpToMove(index) {
        // The crossing effect leaves a game the review is already in folded, so a
        // jump within it would scroll to a row that is not rendered.
        const game = detailMovePositions[index]?.game_number;
        if (game != null && !openGames.has(game)) setGameOpen(game, true);
        await navigateToMove(index);
        await tick();
        document.querySelector(`[data-move-idx="${index}"]`)?.scrollIntoView?.({ block: 'nearest' });
    }

    // The row the review is on, when it is in this match.
    let selectedIdx = $derived($matchContextStore.isMatchMode && detailMatch && $matchContextStore.matchID === detailMatch.id ? $matchContextStore.currentIndex : -1);
    // Défi actif : marques et pertes de la liste sont masquées, sauf celle du coup affiché une fois
    // son analyse dévoilée (même clé que le panneau Analyse) ; sinon la liste trahit la réponse.
    /** @param {number} idx */
    const gradeHidden = (idx) => $analysisChallengeStore && (idx !== selectedIdx || $analysisChallengeHiddenStore);

    async function navigateToMove(/** @type {number} */ moveIndex) {
        if (!detailMatch || !detailMovePositions.length) return;
        // Enter match mode and navigate to the clicked move
        const movePositions = detailMovePositions;
        const match = detailMatch;

        matchContextStore.set({
            isMatchMode: true,
            matchID: match.id,
            movePositions: movePositions,
            currentIndex: moveIndex,
            player1Name: match.player1_name,
            player2Name: match.player2_name
        });

        const movePos = movePositions[moveIndex];
        positionStore.set(movePos.position);

        let analysis = null;
        try {
            analysis = await readOfLibrary(() => LoadAnalysis(movePos.position.id));
        } catch (_error) {
            if (isLibraryLeft(_error)) return;
            /* ignored */
        }

        const currentPlayedMove = movePos.checker_move || '';
        const currentPlayedCubeAction = movePos.cube_action || '';

        analysisStore.set({
            positionId: analysis?.positionId || null,
            xgid: analysis?.xgid || '',
            player1: analysis?.player1 || '',
            player2: analysis?.player2 || '',
            analysisType: analysis?.analysisType || '',
            analysisEngineVersion: analysis?.analysisEngineVersion || '',
            checkerAnalysis: analysis?.checkerAnalysis || { moves: [] },
            doublingCubeAnalysis: analysis?.doublingCubeAnalysis || {
                analysisDepth: '',
                playerWinChances: 0,
                playerGammonChances: 0,
                playerBackgammonChances: 0,
                opponentWinChances: 0,
                opponentGammonChances: 0,
                opponentBackgammonChances: 0,
                cubelessNoDoubleEquity: 0,
                cubelessDoubleEquity: 0,
                cubefulNoDoubleEquity: 0,
                cubefulNoDoubleError: 0,
                cubefulDoubleTakeEquity: 0,
                cubefulDoubleTakeError: 0,
                cubefulDoublePassEquity: 0,
                cubefulDoublePassError: 0,
                bestCubeAction: '',
                wrongPassPercentage: 0,
                wrongTakePercentage: 0
            },
            playedMove: currentPlayedMove,
            playedCubeAction: currentPlayedCubeAction,
            playedMoves: analysis?.playedMoves || [],
            playedCubeActions: analysis?.playedCubeActions || [],
            creationDate: analysis?.creationDate || '',
            lastModifiedDate: analysis?.lastModifiedDate || ''
        });

        commentTextStore.set('');
        selectedMoveStore.set(null);
        statusBarModeStore.set('MATCH');
        // Player names are in MatchInfoBar.svelte, not the status bar.

        lastVisitedMatchStore.set({
            matchID: match.id,
            currentIndex: moveIndex,
            gameNumber: movePos.game_number
        });

        SaveLastVisitedPosition(match.id, moveIndex).catch((e) => {
            logger.error('Error persisting last visited position:', e);
        });

        // Switch to analysis tab so user sees the analysis
        activeTabStore.set('analysis');
    }

    /** Enters match mode on the last visited decision of the match shown (the first one by default). */
    export async function enterMatchMode() {
        await whenLoaded();
        const match = detailMatch;
        if (!match) return;
        if (!detailMovePositions.length) {
            // Load if not already loaded
            await loadMatchDetail(match);
        }
        if (!detailMovePositions.length) {
            statusBarTextStore.set(tMsg('match.noMovesFound'));
            return;
        }

        let startIndex = 0;
        if (lastVisitedMatch && lastVisitedMatch.matchID === match.id) {
            if (lastVisitedMatch.currentIndex >= 0 && lastVisitedMatch.currentIndex < detailMovePositions.length) {
                startIndex = lastVisitedMatch.currentIndex;
            }
        } else if (match.last_visited_position >= 0 && match.last_visited_position < detailMovePositions.length) {
            startIndex = match.last_visited_position;
        }

        await navigateToMove(startIndex);
    }

    /** Shows a decision by its Move id; false when the match has no such decision any more. */
    export async function jumpToMoveId(/** @type {number} */ moveId) {
        const index = detailMovePositions.findIndex((mp) => mp.move_id === moveId);
        if (index < 0) return false;
        await jumpToMove(index);
        return true;
    }

    // The dry run first: nothing to confirm when the match is its file.
    async function repairFromSource(/** @type {Match} */ match) {
        const mark = libraryMark();
        try {
            const plan = await RepairMatch(match.id, true);
            if (!repairDiffers(plan)) {
                statusBarTextStore.set(tMsg('match.repairUpToDate'));
                return;
            }
            const question = get(t)('match.confirmRepair', repairCounts(plan));
            if (!(await confirmAction(question, { confirmLabel: get(t)('match.repair') }))) return;
            const done = await writingOnLibrary(mark, () => RepairMatch(match.id, false));
            if (!done) return;
            await matchListStore.refreshRow(match.id);
            if (detailMatch && detailMatch.id === match.id) {
                detailMatch = get(matchListStore).rows.find((/** @type {any} */ m) => m.id === match.id) || detailMatch;
                await loadMatchDetail(detailMatch);
            }
            dbMutationCounterStore.update((n) => n + 1);
            statusBarTextStore.set(tMsg(repairDoneKey(done), repairCounts(done)));
        } catch (error) {
            logger.error('Error repairing match:', error);
            statusBarTextStore.set(tMsg(repairMatchErrorKey(error)));
        }
    }

    function getPlayerName(/** @type {{ player_on_roll: number, player1_name?: string, player2_name?: string }} */ mp) {
        return mp.player_on_roll === 0 ? mp.player1_name || $t('match.player1') : mp.player2_name || $t('match.player2');
    }

    /**
     * The keys of the sheet itself, for its host: video speed ([ ]) and `v`.
     * Returns true when the key was claimed.
     */
    export function handleSheetKey(/** @type {KeyboardEvent} */ event) {
        const typing = event.target instanceof Element && event.target.matches('input, textarea, select, [contenteditable]');
        if (typing) return false;
        const rateStep = rateKeyDirection(event);
        if (rateStep !== 0 && videoOpen && videoPane) {
            event.stopPropagation();
            event.preventDefault();
            videoPane.stepRate(rateStep);
            return true;
        }
        if (isBareLetter(event, 'v') && videoSource && selectedIdx >= 0) {
            const mp = detailMovePositions[selectedIdx];
            if (moveTickMs(mp) !== null) {
                event.stopPropagation();
                event.preventDefault();
                viewInVideo(mp);
                return true;
            }
        }
        return false;
    }

    async function deleteMatchEntry(/** @type {Match} */ match, /** @type {Event | undefined} */ event) {
        event?.stopPropagation();
        await trashMatchEntry(match, async () => {
            await matchListStore.reload().catch((/** @type {unknown} */ error) => logger.error('Error loading matches:', error));
            await onDeleted?.(match);
        });
    }
</script>

{#if detailMatch}
    <div class="detail-pane">
        <MatchDetailHeader
            match={detailMatch}
            games={detailGames}
            cadence={cadenceInfo}
            hasVideo={!!videoSource}
            {videoOpen}
            ontogglevideo={toggleVideo}
            onreview={() => enterMatchMode(detailMatch)}
            onexport={() => exportMatchMat(detailMatch)}
            onedit={() => editMatchTranscription(detailMatch.id)}
            ondelete={(e) => deleteMatchEntry(detailMatch, e)}
        />

        {#if !loadingDetail}
            <MatchReview review={detailReview} player1={detailMatch.player1_name} player2={detailMatch.player2_name} />
        {/if}

        {#if videoOpen && videoKind === 'file'}
            <!-- Folded to nothing while the video sits beside the board. -->
            <div class="match-video-slot" class:empty={videoOnBoard}>
                <VideoDock bind:this={videoPane} bind:onBoard={videoOnBoard} owner="match" source={videoSource} startMs={videoStartMs} onrelocate={saveVideoSource} />
            </div>
        {/if}

        <div class="match-tabs" role="tablist" tabindex="-1" aria-label={$t('match.tabsLabel')} onkeydown={onTabKey}>
            {#each tabs as id (id)}
                <button
                    class="detail-tab"
                    class:active={id === currentTab}
                    role="tab"
                    id="{uid}-tab-{id}"
                    aria-selected={id === currentTab}
                    aria-controls="{uid}-tabpanel-{id}"
                    tabindex={id === currentTab ? 0 : -1}
                    data-testid="match-tab-{id}"
                    onclick={() => selectTab(id)}>{$t(TAB_LABELS[id])}</button
                >
            {/each}
        </div>

        <!-- Every tab stays mounted: a tab shown again keeps its scroll and its open games. -->
        <div class="tab-panel match-charts" role="tabpanel" id="{uid}-tabpanel-charts" aria-labelledby="{uid}-tab-charts" data-testid="match-tabpanel-charts" hidden={currentTab !== 'charts'}>
            {#if !loadingDetail}
                <MatchCharts
                    matchId={detailMatch.id}
                    times={detailTimes}
                    losses={detailLosses}
                    movePositions={detailMovePositions}
                    player1={detailMatch.player1_name}
                    player2={detailMatch.player2_name}
                    hovered={hoveredMove}
                    onhover={(id) => (hoveredMove = id)}
                    onselect={jumpToMove}
                />
            {/if}
        </div>

        <div
            class="tab-panel transcript-container"
            role="tabpanel"
            id="{uid}-tabpanel-transcript"
            aria-labelledby="{uid}-tab-transcript"
            data-testid="match-tabpanel-transcript"
            hidden={currentTab !== 'transcript'}
        >
            {#if loadingDetail}
                <div class="loading-state">{$t('common.loading')}</div>
            {:else if transcriptGames.length === 0}
                <div class="empty-state">{$t('match.noMovesRecorded')}</div>
            {:else}
                {#each transcriptGames as game (game.gameNumber)}
                    {@const isOpen = openGames.has(game.gameNumber)}
                    <details class="game-section" open={isOpen} ontoggle={(e) => setGameOpen(game.gameNumber, e.currentTarget.open)}>
                        <summary class="game-header">
                            <span class="game-title">{$t('match.game', { n: game.gameNumber })}</span>
                            {#if game.gameInfo}
                                <span class="game-score">{$t('match.score')}: {game.gameInfo.initial_score[0]}–{game.gameInfo.initial_score[1]}</span>
                                <!-- game.winner: 1 = player 1, -1 = player 2, 0 = unfinished (domain.WinnerPlayer1) -->
                                {#if game.gameInfo.winner === 1 || game.gameInfo.winner === -1}
                                    <span class="game-result"
                                        >{$t('match.wonBy', {
                                            player: game.gameInfo.winner === 1 ? detailMatch.player1_name : detailMatch.player2_name,
                                            points: game.gameInfo.points_won
                                        })}</span
                                    >
                                {/if}
                            {/if}
                            {#if game.marks.blunders > 0}
                                <span class="game-marks grade-blunder"
                                    ><CountLink
                                        label="{game.marks.blunders} {GRADE_MARKS.blunder}"
                                        title={$t('match.gameBlunderCount', { n: game.marks.blunders })}
                                        onclick={(e) => openGraded(e, game.moves, 'blunder')}
                                    /></span
                                >
                            {/if}
                            {#if game.marks.errors > 0}
                                <span class="game-marks grade-error"
                                    ><CountLink
                                        label="{game.marks.errors} {GRADE_MARKS.error}"
                                        title={$t('match.gameErrorCount', { n: game.marks.errors })}
                                        onclick={(e) => openGraded(e, game.moves, 'error')}
                                    /></span
                                >
                            {/if}
                        </summary>
                        {#if isOpen}
                            <table class="transcript-table">
                                <thead>
                                    <tr>
                                        <th class="transcript-num">#</th>
                                        <th class="transcript-player">{$t('match.player')}</th>
                                        <th class="transcript-dice">{$t('match.dice')}</th>
                                        <th class="transcript-move">{$t('match.move')}</th>
                                        {#if showTimes}
                                            <th class="transcript-time" title={$t('match.timeCubeTooltip')}>{$t('match.timeCubeCol')}</th>
                                            <th class="transcript-time"
                                                ><button class="time-sort" onclick={cycleTimeSort} title={$t('match.timeSortTooltip')}
                                                    >{$t('match.timePlayCol')}{timeSort === 'desc' ? ' ▼' : timeSort === 'asc' ? ' ▲' : ''}</button
                                                ></th
                                            >
                                            <th class="transcript-time" title={$t(remaining ? 'match.timeRemainTooltip' : 'match.timeClockTooltip')}
                                                >{$t(remaining ? 'match.timeRemainCol' : 'match.timeClockCol')}</th
                                            >
                                        {/if}
                                        {#if showLosses}
                                            <th class="transcript-time" title={$t('match.lossColTooltip')}
                                                ><button class="time-sort" data-testid="loss-sort" onclick={cycleLossSort} title={$t('match.lossSortTooltip')}
                                                    >{$t('match.lossCol')}{lossSort === 'desc' ? ' ▼' : lossSort === 'asc' ? ' ▲' : ''}</button
                                                ></th
                                            >
                                            <th class="transcript-time" title={$t('match.difficultyColTooltip')}>{$t('match.difficultyCol')}</th>
                                        {/if}
                                    </tr>
                                </thead>
                                <tbody>
                                    {#each ordered(game.moves) as { mp, globalIdx, grade, loss, difficulty, avoidable }, mi (globalIdx)}
                                        <tr
                                            class="transcript-row"
                                            class:cube-row={mp.move_type === 'cube'}
                                            class:graded-error={grade?.grade === 'error' && !gradeHidden(globalIdx)}
                                            class:graded-blunder={grade?.grade === 'blunder' && !gradeHidden(globalIdx)}
                                            class:current-move={selectedIdx === globalIdx}
                                            class:linked={hoveredMove === mp.move_id}
                                            data-move-idx={globalIdx}
                                            onclick={() => navigateToMove(globalIdx)}
                                            title={$t('match.clickToReview')}
                                        >
                                            <td class="transcript-num">
                                                {mi + 1}
                                                {#if videoSource && moveTickMs(mp) !== null}
                                                    <button
                                                        class="row-video"
                                                        data-testid="view-in-video"
                                                        title="{$t('match.viewInVideo')} (v)"
                                                        onclick={(e) => {
                                                            e.stopPropagation();
                                                            viewInVideo(mp);
                                                        }}>🎞</button
                                                    >
                                                {/if}
                                            </td>
                                            <td class="transcript-player" class:player1={mp.player_on_roll === 0} class:player2={mp.player_on_roll === 1}>
                                                {getPlayerName(mp)}
                                            </td>
                                            <td class="transcript-dice">
                                                {#if mp.move_type === 'checker'}
                                                    {formatDiceShort(mp.position.dice)}
                                                {/if}
                                            </td>
                                            <td class="transcript-move">
                                                {#if mp.move_type === 'cube'}
                                                    <span class="cube-action">{mp.cube_action || $t('match.cube')}</span>
                                                {:else}
                                                    {mp.checker_move || '—'}
                                                {/if}
                                                {#if grade && !gradeHidden(globalIdx)}
                                                    <span
                                                        class="grade-mark grade-{grade.grade}"
                                                        title={$t(grade.grade === 'blunder' ? 'match.gradeBlunder' : 'match.gradeError', { cost: fmtGradeCost(grade.error_mp) })}
                                                        >{GRADE_MARKS[grade.grade]}</span
                                                    >
                                                {/if}
                                            </td>
                                            {#if showTimes}
                                                <td class="transcript-time" data-testid="move-time-cube">{orDash(fmtDuration(mp.move_type === 'cube' ? mp.decision_ms : mp.cube_decision_ms))}</td>
                                                <td class="transcript-time" data-testid="move-time-play">{orDash(mp.move_type === 'cube' ? '' : fmtDuration(mp.decision_ms))}</td>
                                                <td class="transcript-time" data-testid="move-time-clock">{orDash(remaining ? fmtClock(remaining[globalIdx]) : fmtClock(clocks[globalIdx]))}</td>
                                            {/if}
                                            {#if showLosses}
                                                <td
                                                    class="transcript-time transcript-loss"
                                                    class:grade-error={grade?.grade === 'error' && !gradeHidden(globalIdx)}
                                                    class:grade-blunder={grade?.grade === 'blunder' && !gradeHidden(globalIdx)}
                                                    data-testid="move-loss"
                                                    >{gradeHidden(globalIdx) ? '···' : loss === null ? '—' : loss > 0 ? fmtLoss(loss) : '0'}{#if avoidable && !gradeHidden(globalIdx)}<span
                                                            class="avoidable-mark"
                                                            title={$t('match.avoidableTooltip')}
                                                            aria-label={$t('match.avoidable')}
                                                            data-testid="move-avoidable">!</span
                                                        >{/if}</td
                                                >
                                                <td class="transcript-time" data-testid="move-difficulty"
                                                    >{gradeHidden(globalIdx) ? '···' : difficulty === null ? '—' : difficulty > 0 ? fmtLoss(difficulty) : '0'}</td
                                                >
                                            {/if}
                                        </tr>
                                    {/each}
                                </tbody>
                            </table>
                        {/if}
                    </details>
                {/each}
            {/if}
        </div>

        <div class="tab-panel" role="tabpanel" id="{uid}-tabpanel-review" aria-labelledby="{uid}-tab-review" data-testid="match-tabpanel-review" hidden={currentTab !== 'review'}>
            <MatchReviewDecisions review={detailReview} movePositions={detailMovePositions} player1={detailMatch.player1_name} player2={detailMatch.player2_name} onselect={jumpToMove} />
        </div>

        <div class="tab-panel" role="tabpanel" id="{uid}-tabpanel-details" aria-labelledby="{uid}-tab-details" data-testid="match-tabpanel-details" hidden={currentTab !== 'details'}>
            <MatchReviewDetails review={detailReview} losses={detailLosses} movePositions={detailMovePositions} player1={detailMatch.player1_name} player2={detailMatch.player2_name} />
        </div>

        <div class="tab-panel" role="tabpanel" id="{uid}-tabpanel-info" aria-labelledby="{uid}-tab-info" data-testid="match-tabpanel-info" hidden={currentTab !== 'info'}>
            <div class="metadata-container">
                <table class="metadata-table">
                    <tbody>
                        <tr><td class="meta-label">{$t('match.player1')}</td><td class="meta-value">{detailMatch.player1_name || '—'}</td></tr>
                        <tr><td class="meta-label">{$t('match.player2')}</td><td class="meta-value">{detailMatch.player2_name || '—'}</td></tr>
                        <tr
                            ><td class="meta-label">{$t('match.matchLength')}</td><td class="meta-value"
                                >{detailMatch.match_length > 1 ? $t('match.points', { n: detailMatch.match_length }) : $t('match.point', { n: detailMatch.match_length })}</td
                            ></tr
                        >
                        {#if cadenceInfo}
                            <tr data-testid="meta-cadence"
                                ><td class="meta-label">{$t('match.cadenceRow')}</td><td class="meta-value"
                                    >{[cadenceInfo.name, $t('match.originDelay', { s: cadenceInfo.delay }), $t('match.timeOutRow', { what: cadenceInfo.timeOut })].filter(Boolean).join(', ')}</td
                                ></tr
                            >
                            <tr data-testid="meta-bank"
                                ><td class="meta-label">{$t('match.bankRow')}</td><td class="meta-value"
                                    >{cadenceInfo.bank ? $t('match.bankEach', { time: cadenceInfo.bank }) : '—'}{cadenceInfo.perPoint
                                        ? ' (' + $t('match.bankPerPoint', { s: cadenceInfo.perPoint }) + ')'
                                        : ''}</td
                                ></tr
                            >
                        {/if}
                        <tr><td class="meta-label">{$t('match.games')}</td><td class="meta-value">{detailMatch.game_count || detailGames.length || '—'}</td></tr>
                        <tr><td class="meta-label">{$t('match.date')}</td><td class="meta-value">{formatDate(detailMatch.match_date)}</td></tr>
                        <tr data-testid="meta-video">
                            <td class="meta-label">{$t('match.video')}</td>
                            <td class="meta-value">
                                <input
                                    type="text"
                                    class="match-video-input"
                                    bind:value={videoDraft}
                                    placeholder={$t('match.videoPlaceholder')}
                                    onkeydown={(e) => {
                                        // Enter commits through the change event, once.
                                        if (e.key === 'Enter') {
                                            e.stopPropagation();
                                        } else if (e.key === 'Escape') {
                                            e.stopPropagation();
                                            videoDraft = videoSource;
                                        }
                                    }}
                                    onchange={() => saveVideoSource(videoDraft)}
                                />
                                <button class="detail-tab" onclick={pickVideoFile}>{$t('match.videoChoose')}</button>
                                {#if videoSource}<button class="detail-tab" data-testid="video-detach" onclick={() => saveVideoSource('')}>{$t('match.videoDetach')}</button>{/if}
                            </td>
                        </tr>
                        <tr>
                            <td class="meta-label">{$t('match.comment')}</td>
                            <td class="meta-value">
                                {#if commentEdit.isEditing(detailMatch.id)}
                                    <input type="text" class="match-comment-input" bind:value={commentEdit.draft} onkeydown={commentEdit.onKeyDown} onblur={commentEdit.onBlur} />
                                {:else}
                                    <!-- svelte-ignore a11y_click_events_have_key_events -->
                                    <!-- svelte-ignore a11y_no_static_element_interactions -->
                                    <span class="match-comment-display" onclick={() => commentEdit.start(detailMatch.id, detailMatch.comment || '')} title={$t('match.clickToAddComment')}>
                                        {detailMatch.comment || $t('match.addComment')}
                                    </span>
                                    {#if detailMatch.comment && detailMatch.comment_author}
                                        <span class="match-comment-author">{detailMatch.comment_author}</span>
                                    {/if}
                                {/if}
                            </td>
                        </tr>
                        <tr>
                            <td class="meta-label">{$t('match.tournament')}</td>
                            <td
                                class="meta-value tournament-meta-cell"
                                onclick={(e) => {
                                    e.stopPropagation();
                                    ((e) => startEditTournament(detailMatch, e))(e);
                                }}
                            >
                                {#if tournamentEdit.isEditing(detailMatch.id)}
                                    <div class="tournament-cell-edit">
                                        <EntityAutocomplete
                                            bind:value={tournamentEdit.draft}
                                            items={tournaments}
                                            autofocus
                                            blurDelay={200}
                                            placeholder={$t('match.tournamentNamePlaceholder')}
                                            onSelect={selectTournamentOption}
                                            onSubmit={() => tournamentEdit.save()}
                                            onCancel={tournamentEdit.cancel}
                                            onDismiss={tournamentEdit.cancel}
                                        />
                                    </div>
                                {:else}
                                    <span class="tournament-display" title={$t('match.clickToEdit')}>{detailMatch.tournament_name || detailMatch.event || '—'}</span>
                                {/if}
                            </td>
                        </tr>
                        <tr><td class="meta-label">{$t('match.event')}</td><td class="meta-value">{detailMatch.event || '—'}</td></tr>
                        <tr><td class="meta-label">{$t('match.location')}</td><td class="meta-value">{detailMatch.location || '—'}</td></tr>
                        <tr><td class="meta-label">{$t('match.round')}</td><td class="meta-value">{detailMatch.round || '—'}</td></tr>
                        {#if detailMatch.player1_elo != null || detailMatch.player2_elo != null}
                            <tr><td class="meta-label">{$t('match.ratings')}</td><td class="meta-value">{formatRatings(detailMatch)}</td></tr>
                        {/if}
                        {#if detailMatch.transcriber}
                            <tr><td class="meta-label">{$t('match.transcriber')}</td><td class="meta-value">{detailMatch.transcriber}</td></tr>
                        {/if}
                        {#if detailMatch.match_length === 0 && (detailMatch.has_jacoby != null || detailMatch.has_beaver != null)}
                            <tr><td class="meta-label">{$t('match.sessionRules')}</td><td class="meta-value">{formatSessionRules(detailMatch)}</td></tr>
                        {/if}
                        {#if detailMatch.engine_version}
                            <tr><td class="meta-label">{$t('match.writtenBy')}</td><td class="meta-value">{detailMatch.engine_version}</td></tr>
                        {/if}
                        <tr
                            ><td class="meta-label">{$t('match.sourceFile')}</td><td class="meta-value source-file"
                                >{detailMatch.file_path || '—'}
                                {#if detailMatch.file_path}
                                    <button class="icon-btn" data-testid="match-repair" onclick={() => repairFromSource(detailMatch)} title={$t('match.repairTooltip')} aria-label={$t('match.repair')}
                                        >⟲</button
                                    >
                                {/if}</td
                            ></tr
                        >
                        <tr><td class="meta-label">{$t('match.importDate')}</td><td class="meta-value">{formatDate(detailMatch.import_date)}</td></tr>
                        <tr><td class="meta-label">{$t('match.matchId')}</td><td class="meta-value id-value">{detailMatch.id}</td></tr>
                    </tbody>
                </table>
            </div>
            {#if detailOrigin}
                <h4 class="tab-heading">{$t('match.origin')}</h4>
                <MatchOrigin origin={detailOrigin} player1={detailMatch.player1_name} player2={detailMatch.player2_name} />
            {/if}
        </div>

        <div class="tab-panel" role="tabpanel" id="{uid}-tabpanel-stats" aria-labelledby="{uid}-tab-stats" data-testid="match-tabpanel-stats" hidden={currentTab !== 'stats'}>
            <MatchStatsTable stats={detailStats} loading={loadingStats} player1={detailMatch.player1_name} player2={detailMatch.player2_name} />
        </div>
    </div>
{/if}

<style>
    .match-video-slot {
        width: 100%;
        aspect-ratio: 16 / 9;
        max-height: calc(40vh / var(--ui-scale, 1));
        flex: 0 0 auto;
    }
    .match-video-slot.empty {
        aspect-ratio: auto;
        height: 0;
    }
    /* The tabs take what the header and the summary leave; only a tab scrolls. */
    .match-tabs {
        flex: 0 0 auto;
        display: flex;
        flex-wrap: wrap;
        padding: 0 8px;
        border-top: 1px solid var(--color-border);
        border-bottom: 1px solid var(--color-border);
    }
    .match-tabs .detail-tab {
        padding: 6px 12px;
        font-size: var(--font-size-base);
    }
    .tab-panel {
        flex: 1 1 auto;
        min-height: 0;
        overflow-y: auto;
        text-align: left;
    }
    /* The charts share the tab's height out; under their least height, the tab scrolls. */
    .match-charts:not([hidden]) {
        display: flex;
        flex-direction: column;
    }
    .tab-heading {
        margin: 12px 8px 4px;
        font-size: var(--font-size-base);
        color: var(--color-text);
    }

    /* Detail pane placeholders (the list's own empty state is PanelTable's). */
    .empty-state,
    .loading-state {
        text-align: center;
        color: var(--color-text-muted);
        padding: 24px;
        font-size: var(--font-size-base);
    }

    .row-video {
        background: none;
        border: 0;
        padding: 0 2px;
        cursor: pointer;
        opacity: 0.6;
    }
    .row-video:hover {
        opacity: 1;
    }

    .detail-tab {
        background: none;
        border: none;
        border-bottom: 2px solid transparent;
        padding: 4px 12px;
        cursor: pointer;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        transition:
            color 0.15s,
            border-color 0.15s;
    }

    .detail-tab:hover {
        color: var(--color-text);
    }

    .detail-tab.active {
        color: color-mix(in srgb, var(--color-primary) 80%, var(--color-text));
        border-bottom-color: var(--color-primary);
    }

    /* --- Transcript --- */
    .transcript-container {
        /* An Error is the Blunder's red, faded: one hue whose strength is the
           gravity, from the palette's own danger token (ADR-0031). */
        --grade-error-color: color-mix(in srgb, var(--color-danger) 45%, var(--color-surface));
        flex: 1;
        overflow-y: auto;
        padding: 0;
    }

    .game-section {
        margin-bottom: 2px;
    }

    .game-header {
        position: sticky;
        top: 0;
        display: flex;
        align-items: center;
        gap: 10px;
        padding: 4px 12px;
        background: var(--color-surface-alt);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        border-bottom: 1px solid var(--color-border);
        z-index: 1;
        cursor: pointer;
        list-style: none;
    }

    /* .game-header is a <summary>; the native marker is replaced by a triangle flipping with [open]. */
    .game-header::-webkit-details-marker {
        display: none;
    }

    .game-header::before {
        content: '▸';
        color: var(--color-text-muted);
    }

    .game-section[open] > .game-header::before {
        content: '▾';
    }

    .game-title {
        font-weight: 600;
        color: var(--color-text);
    }

    .game-score {
        color: var(--color-text-muted);
    }

    .game-result {
        color: color-mix(in srgb, #2e7d32 60%, var(--color-text));
        font-style: italic;
    }

    .transcript-table {
        width: 100%;
        border-collapse: collapse;
        font-size: var(--font-size-small);
    }

    .transcript-table thead th {
        padding: 2px 8px;
        text-align: left;
        font-weight: 600;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        border-bottom: 1px solid var(--color-border);
        background: var(--color-surface-alt);
    }

    /* Text at the start of the cell, numbers at its end (ADR-0085, G4), headers over their values. */
    .transcript-table thead th.transcript-num,
    .transcript-table thead th.transcript-time {
        text-align: right;
    }

    .transcript-player,
    .transcript-move {
        text-align: start;
    }

    .transcript-table tbody td {
        padding: 2px 8px;
        border-bottom: 1px solid var(--color-border);
    }

    .transcript-row {
        cursor: pointer;
        transition: background-color 0.1s;
    }

    .transcript-row:hover {
        background-color: color-mix(in srgb, var(--color-primary) 8%, var(--color-surface));
    }

    .transcript-num {
        width: 28px;
        text-align: right;
        font-variant-numeric: tabular-nums;
        color: var(--color-text-muted);
    }

    .transcript-player {
        width: 100px;
        max-width: 100px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .transcript-player.player1 {
        color: var(--color-text);
    }

    .transcript-player.player2 {
        color: var(--color-text-muted);
    }

    .transcript-dice {
        width: 32px;
        text-align: center;
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .transcript-move {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: var(--color-text);
    }

    .cube-row {
        background-color: color-mix(in srgb, #ffc107 12%, var(--color-surface));
    }

    .cube-row:hover {
        background-color: color-mix(in srgb, #ffc107 20%, var(--color-surface));
    }

    /* Transcript marks: a left rule and a glyph, so the grade reads without colour. */
    .transcript-row.graded-error > td:first-child {
        box-shadow: inset 3px 0 0 var(--grade-error-color);
    }

    .transcript-row.graded-blunder > td:first-child {
        box-shadow: inset 3px 0 0 var(--color-danger);
    }

    .transcript-row.graded-blunder {
        background-color: color-mix(in srgb, var(--color-danger) 8%, var(--color-surface));
    }

    .transcript-row.graded-blunder:hover {
        background-color: color-mix(in srgb, var(--color-danger) 14%, var(--color-surface));
    }

    .transcript-time {
        text-align: right;
        white-space: nowrap;
        color: var(--text-muted, inherit);
        font-size: var(--font-size-small);
        font-variant-numeric: tabular-nums;
    }
    .transcript-row.linked {
        background-color: var(--color-surface-alt);
    }
    .transcript-row.current-move {
        box-shadow: inset 0 0 0 2px var(--color-primary);
    }
    .avoidable-mark {
        margin-left: 3px;
        font-weight: bold;
        color: var(--color-danger);
    }
    .transcript-loss.grade-error,
    .transcript-loss.grade-blunder {
        font-weight: 700;
    }
    .time-sort {
        background: none;
        border: none;
        padding: 0;
        color: inherit;
        cursor: pointer;
    }
    .grade-mark,
    .game-marks {
        font-weight: 700;
        font-family: var(--font-family-ui);
    }

    .grade-mark {
        margin-left: 6px;
    }

    .grade-error {
        color: color-mix(in srgb, var(--color-danger) 70%, var(--color-text-muted));
    }

    .grade-blunder {
        color: var(--color-danger);
    }

    .cube-action {
        color: color-mix(in srgb, #e65100 65%, var(--color-text));
        font-weight: 500;
        font-family: inherit;
    }

    /* --- Metadata view --- */
    .metadata-container {
        padding: 0;
    }

    .metadata-table {
        width: 100%;
        border-collapse: collapse;
        font-size: var(--font-size-base);
    }

    .metadata-table td {
        padding: 4px 8px;
        border-bottom: 1px solid var(--color-border);
        vertical-align: top;
    }

    .meta-label {
        width: 100px;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .meta-value {
        color: var(--color-text);
    }

    .source-file {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        word-break: break-all;
    }

    .id-value {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .tournament-meta-cell {
        cursor: pointer;
    }

    .tournament-cell-edit {
        position: relative;
    }

    .tournament-display {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .tournament-display:hover {
        color: var(--color-primary);
    }

    .match-comment-display {
        cursor: pointer;
        padding: 1px 3px;
        border-radius: 3px;
        min-width: 40px;
        display: inline-block;
        color: var(--color-text-muted);
        font-style: italic;
    }

    .match-comment-display:hover {
        background: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface));
    }

    .match-comment-author {
        margin-left: 6px;
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text);
    }

    .match-video-input,
    .match-comment-input {
        width: 100%;
        padding: 1px 3px;
        font-size: var(--font-size-small);
        border: 1px solid var(--color-primary);
        border-radius: 3px;
        outline: none;
        box-sizing: border-box;
    }

    .detail-pane {
        width: 100%;
        height: 100%;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        user-select: none;
        -webkit-user-select: none;
    }

    .tournament-meta-cell {
        cursor: pointer;
    }

    .tournament-cell-edit {
        position: relative;
    }

    .tournament-display {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .tournament-display:hover {
        color: var(--color-primary);
    }
</style>
