<!--
  TranscriptionPanel — the library's transcription drafts and the draft open.

  A Transcription is a draft of a match being typed in, distinct from the Match
  it produces (ADR-0045). The panel is a CLIENT of the Go engine (ADR-0045
  rule 9): every gesture goes to ApplyTranscriptionGesture and comes back as a
  whole annotated document; it derives no score, Crawford or side on roll.

  Creation asks only the length (0 = money, fonctionnel.md §1.1). The candidate
  list is a 0-ply Evaluation (EvaluatePositionImmediate, EvalPanel's table),
  shown and never stored (ADR-0045 rule 8). Mouse triggers (dice triangle, cube
  row, board cube, Transcript context menu) go through the key machine and
  `applyResult`, like a keystroke. Corrections are marked by the Replay, never
  refused (ADR-0044); the undo stack lives in Go's transcript.Editor and a
  crash may lose it (ADR-0045 rule 1). Save/export/close logic lives in
  services/transcriptionSave.js.
-->
<script>
    import { onMount, onDestroy, untrack } from 'svelte';
    import PanelTable from './panels/PanelTable.svelte';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { databaseLoadedStore } from '../stores/databaseStore.js';
    import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
    import { positionStore } from '../stores/positionStore.js';
    import { selectedMoveStore } from '../stores/analysisStore.js';
    import { panelKeyGuard } from '../services/keyboardService.js';
    import { isBareLetter } from '../utils/keys.js';
    import { isMoneyPosition } from '../utils/cubeDecision.js';
    import {
        PHASE,
        COMMAND,
        pressKey,
        applyCandidates,
        selectCandidate,
        cursorCommands,
        cursorStop,
        currentStop,
        initialKeyState,
        enterDicePair,
        enterSingleDie,
        cubeGesture,
        selectionDelta,
        beginResign,
        resignWithLevel,
        cancelResign,
        menuCommands,
        dieOf,
        DICE_KINDS
    } from '../services/transcriptionKeys.js';
    import CandidateMovesTable from './CandidateMovesTable.svelte';
    import DiceTriangle from './DiceTriangle.svelte';
    import CubeActionRow from './CubeActionRow.svelte';
    import ContextMenu from './ContextMenu.svelte';
    import TranscriptView from './TranscriptView.svelte';
    import TranscriptionMetadata from './TranscriptionMetadata.svelte';
    import {
        transcriptionListStore,
        transcriptionStore,
        transcriptionKeyStore,
        transcriptionHistoryStore,
        transcriptionHistoryActionStore,
        transcriptionCubeRequestStore,
        transcriptionWheelStore,
        transcriptionInfoStore,
        transcriptionPromptStore,
        transcriptionBoardSwapStore,
        setTranscription,
        clearTranscription,
        resetTranscriptionKeys,
        noticeTranscription,
        clearTranscriptionNotice
    } from '../stores/transcriptionStore.js';
    import Modal from './Modal.svelte';
    import { writeTextToClipboard } from '../services/clipboardService.js';
    import { quizPlayStore } from '../stores/quizPlayStore.js';
    import { ROLLS, newBoardPlay, deducedDice, choosableRolls, undoBoardStep, stepsFromNotation, boardAfterSteps } from '../services/transcriptionPlay.js';
    import { containsSteps } from '../services/quizPlay.js';
    import { ListTranscriptions, CreateTranscription, OpenTranscription, ApplyTranscriptionGesture, TranscriptionMAT } from '../../wailsjs/go/database/Database.js';
    import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';
    import { GetGammonNetPruneK } from '../../wailsjs/go/main/Config.js';
    import { saveDraft, exportDraftMat, closeDraft, draftSaveState, savedMatchID, transcriptionSaveStore, resetTranscriptionSave } from '../services/transcriptionSave.js';
    import { get } from 'svelte/store';

    // Mirrors Go's transcript.DefaultMatchLength: the form shows it before any call.
    /** @typedef {import('../services/transcriptionKeys.js').KeyState} KeyState */
    /** @typedef {import('../services/transcriptionPlay.js').BoardPlayState} BoardPlayState */
    /**
     * Une commande de la machine à touches, ou celle d'un coup posé par ses pas.
     *
     * @typedef {import('../services/transcriptionKeys.js').KeyCommand & {steps?: {from: number, to: number}[], board?: any}} PanelCommand
     */
    /** @typedef {{move: any, gen: number, steps: any[]}} Candidate */

    const DEFAULT_MATCH_LENGTH = 7;

    let busy = $state(false);
    let error = $state('');
    let showForm = $state(false);
    // Text, not a number: a number-bound input turns "empty" into 0, i.e. money.
    let formLength = $state('');
    let formJacoby = $state(true);
    let formBeaver = $state(false);
    let panelEl = $state(/** @type {HTMLElement | null} */ (null));
    // Volet des métadonnées, replié par défaut ; l'en-tête vit dans le document.
    let metaOpen = $state(false);

    let draft = $derived($transcriptionStore);
    let annotated = $derived(draft?.annotated ?? null);
    let keys = $derived($transcriptionKeyStore);
    // `awaits` : ce que le document attend après sa dernière Action ;
    // `expects` : ce qu'attend la cellule sous le curseur (`entry.kind`), sinon
    // `awaits`. Confondre les deux ressaisissait une ouverture comme un coup de
    // pions, hors de la convention qui décide qui commence.
    let awaits = $derived(annotated?.next?.expects ?? '');
    let expects = $derived(annotated?.entry?.kind || awaits);
    // Discriminant de la touche chiffrée (ADR-0048 décision 1) : en bout de
    // document elle valide, sur une Action existante elle recommence le jet.
    let replacing = $derived(annotated?.entry?.replacing === true);
    // Cellule tenue par le curseur, aucun dé tapé : les gestes de videau y
    // corrigent au rang de l'Entry. Dès qu'un dé est tapé, `t`/`p` cessent
    // d'être des réponses (`p` redevient le compte de pips global).
    let editingSlot = $derived(!!annotated?.entry && !(annotated.entry.dice?.[0] > 0) && !(annotated.entry.dice?.[1] > 0));
    // Dernière Action relue : un jet retapé s'y valide au chiffre suivant (ADR-0051).
    let onLast = $derived(replacing && annotated?.entry?.at === (annotated?.actions?.length ?? 0) - 1);
    let keyContext = $derived({ expects, replacing, editing: editingSlot, last: onLast });

    // Length 0 = money, the only case asking for the session's rules (ADR-0028).
    let formIsMoney = $derived(formLength.trim() === '0');
    let formValid = $derived(/^\d+$/.test(formLength.trim()));

    // Length of the latest draft, else 7 (fonctionnel.md §1.1); an unreadable draft carries -1.
    function defaultLength() {
        const last = $transcriptionListStore.find((row) => row.match_length >= 0);
        return last ? String(last.match_length) : String(DEFAULT_MATCH_LENGTH);
    }

    async function refresh() {
        if (!$databaseLoadedStore) {
            transcriptionListStore.set([]);
            return;
        }
        try {
            transcriptionListStore.set((await ListTranscriptions()) ?? []);
            error = '';
        } catch (err) {
            logger.error('Failed to list the transcription drafts:', err);
            error = String(err);
        }
    }

    // Refresh on tab entry and library change: another file's drafts must not survive an open.
    $effect(() => {
        void $databaseLoadedStore;
        if ($activeTabStore === 'transcription') refresh();
    });

    function openForm() {
        formLength = defaultLength();
        formJacoby = true;
        formBeaver = false;
        showForm = true;
    }

    async function createDraft() {
        if (busy || !$databaseLoadedStore || !formValid) return;
        busy = true;
        const length = Number(formLength.trim());
        try {
            const state = await CreateTranscription(
                /** @type {any} */ ({
                    match_length: length,
                    jacoby: length === 0 ? formJacoby : false,
                    beaver: length === 0 ? formBeaver : false
                })
            );
            setTranscription(state);
            resetTranscriptionKeys();
            showForm = false;
            await refresh();
            error = '';
        } catch (err) {
            logger.error('Failed to create a transcription draft:', err);
            error = String(err);
        } finally {
            busy = false;
        }
    }

    /** @param {any} row */
    async function openDraft(row) {
        if (busy || !row) return;
        busy = true;
        try {
            setTranscription(await OpenTranscription(row.id));
            resetTranscriptionKeys();
            error = '';
        } catch (err) {
            logger.error('Failed to open a transcription draft:', err);
            error = String(err);
        } finally {
            busy = false;
        }
    }

    // Back to the list; the draft stays open Go-side (deleting it is handleClose).
    function backToList() {
        clearTranscription();
        refresh();
    }

    // Enregistrer, exporter, fermer. `busy` verrouille : un second Ctrl+Entrée
    // relancerait la réécriture du match entier.

    async function handleSave() {
        if (busy || !draft) return;
        busy = true;
        try {
            if (await saveDraft(draft)) {
                error = '';
                await refresh();
            }
        } finally {
            busy = false;
        }
    }

    async function handleExport() {
        if (busy || !draft) return;
        busy = true;
        try {
            await exportDraftMat(draft);
        } finally {
            busy = false;
        }
    }

    async function handleClose() {
        if (busy || !draft) return;
        busy = true;
        try {
            if (await closeDraft(draft)) {
                clearTranscription();
                resetTranscriptionSave();
                await refresh();
            }
        } finally {
            busy = false;
        }
    }

    // Horloge qui fait vieillir « enregistré il y a 3 min ».
    let nowTick = $state(Date.now());
    $effect(() => {
        if (!draft) return;
        const handle = setInterval(() => (nowTick = Date.now()), 30000);
        return () => clearInterval(handle);
    });

    let saveState = $derived(draftSaveState(draft, $transcriptionSaveStore, nowTick));

    // Le bouton nomme le Match, pas « Enregistrer » : le brouillon s'écrit à
    // chaque Action (ADR-0048 décision 12).
    let savedMatchId = $derived(savedMatchID(annotated) || ($transcriptionSaveStore && $transcriptionSaveStore.id === draft?.id ? $transcriptionSaveStore.matchId : 0));
    let saveLabel = $derived(savedMatchId ? $t('transcription.updateMatch', { id: savedMatchId }) : $t('transcription.createMatch'));

    // Each command becomes one gesture, sent in order through one promise chain
    // (never a bare `await` per handler) so keystrokes' round trips never interleave.

    let pending = Promise.resolve();

    // Candidates ranked by 0-ply; `gen` is the index in LegalMoves' order, which
    // `select_candidate` addresses. The join is exact: LegalMoves dedups by board.
    let ranked = $state(/** @type {Candidate[]} */ ([]));
    // Unranked (score beyond the MET, no weights): plays still listed in generator order.
    let unranked = $state(false);
    // The last roll allowed no play at all: the `dance` Action was recorded on
    // its own, without a keystroke.
    let danced = $state(false);

    /** @param {PanelCommand} command */
    function gestureOf(command) {
        switch (command.kind) {
            case COMMAND.DIE:
                return { Kind: 'enter_die', Die: command.value };
            case COMMAND.CLEAR:
                return { Kind: 'clear_dice' };
            case COMMAND.VALIDATE:
                return { Kind: 'validate' };
            case COMMAND.DANCE:
                return { Kind: 'dance' };
            // Coup posé par ses pas (ADR-0052). `BoardAfter` ne vient que des
            // chemins qui peuvent être illégaux ; le moteur le jette dès qu'un
            // coup légal atteint ce plateau (transcript.validate).
            case COMMAND.ENTER_PLAY:
                return {
                    Kind: 'enter_play',
                    Steps: (command.steps ?? []).map((step) => ({ from: step.from, to: step.to, hit: false })),
                    BoardAfter: command.board ?? null
                };
            // Pas de `HasSide` : le moteur prend le camp attendu (transcript.cubeGesture).
            case COMMAND.DOUBLE:
                return { Kind: 'double' };
            case COMMAND.TAKE:
                return { Kind: 'take' };
            case COMMAND.PASS:
                return { Kind: 'pass' };
            // Camp au trait par défaut ; `flip_side` le change ensuite (fonctionnel.md §1.2, §2).
            case COMMAND.RESIGN:
                return { Kind: 'resign', Level: command.value };
            case COMMAND.CURSOR_BACK:
                return { Kind: 'cursor_back' };
            case COMMAND.CURSOR_FORWARD:
                return { Kind: 'cursor_forward' };
            // Gestes de correction, sans camp : le moteur le propose, seul
            // `flip_side` le change (ADR-0045 §4).
            case COMMAND.INSERT_BEFORE:
                return { Kind: 'insert_before' };
            case COMMAND.INSERT_AFTER:
                return { Kind: 'insert_after' };
            case COMMAND.DELETE:
                return { Kind: 'delete' };
            case COMMAND.FLIP_SIDE:
                return { Kind: 'flip_side' };
            // Pile en mémoire dans transcript.Editor, hors de transcript.Apply (pure).
            case COMMAND.UNDO:
                return { Kind: 'undo' };
            case COMMAND.REDO:
                return { Kind: 'redo' };
            case COMMAND.SELECT: {
                // Rang dans la liste montrée, réduite par les pas joués (ADR-0052).
                const entry = visible[/** @type {number} */ (command.index)];
                return entry ? { Kind: 'select_candidate', Candidate: entry.gen } : null;
            }
            default:
                return null;
        }
    }

    // Converted synchronously: a `select` rank refers to the list at keystroke time.
    /** @param {PanelCommand[]} commands */
    function run(commands) {
        return queue(commands.map(gestureOf).filter(Boolean));
    }

    /** @param {any[]} gestures sent in order, one round trip each */
    function queue(gestures) {
        if (!gestures.length) return pending;
        const id = draft?.id;
        if (id == null) return pending;
        pending = pending
            .then(async () => {
                for (const gesture of gestures) {
                    setTranscription(await ApplyTranscriptionGesture(id, gesture));
                }
                error = '';
            })
            .catch((err) => {
                logger.error('A transcription gesture failed:', err);
                error = String(err);
            });
        return pending;
    }

    /**
     * Geste des métadonnées, par la même file que les touches pour ne pas doubler un jet en vol.
     *
     * @param {any} gesture
     */
    function sendGesture(gesture) {
        return gesture ? queue([gesture]) : pending;
    }

    // Candidates: LegalMoves + EvaluatePositionImmediate at 0-ply (candidates = 0
    // returns all). Never StartEvaluationAtRest: too slow between two keystrokes.

    let candidateGeneration = 0;

    /**
     * The Position the roll being entered is played from, dice and side set:
     * `annotated.entry` (transcript.EntryInfo) for a correction or insertion,
     * never `next.position`, which is the end of the document.
     */
    function entryPosition() {
        const ann = get(transcriptionStore)?.annotated;
        if (!ann) return null;
        const entry = ann.entry;
        const at = entry?.at ?? ann.cursor ?? 0;
        const base = positionAt(ann, at);
        const side = entry ? entry.side : ann.next?.side;
        if (!base) return null;
        return { ...structuredClone(base), id: 0, dice: [0, 0], player_on_roll: side, decision_type: 0 };
    }

    /**
     * La Position d'où part l'Action de rang `at`, et celle que le match a
     * atteinte en bout de document.
     *
     * `has_position` faux (ouverture, abandon) ne veut pas dire « rien à
     * montrer » : le Replay remplit `before` pour toutes (transcript/replay.go).
     *
     * @param {any} ann
     * @param {number} at
     */
    function positionAt(ann, at) {
        const actions = ann?.actions ?? [];
        const info = at >= 0 && at < actions.length ? actions[at] : null;
        return info?.before ?? ann?.next?.position ?? null;
    }

    function rollPosition() {
        const state = get(transcriptionKeyStore);
        const pos = entryPosition();
        if (!pos || !state.dice[0] || !state.dice[1]) return null;
        return { ...pos, dice: [state.dice[0], state.dice[1]] };
    }

    /**
     * The plays of the roll, best first, each with its index in LegalMoves.
     *
     * @param {any} pos
     * @returns {Promise<{list: Candidate[], unranked: boolean}>}
     */
    async function computeCandidates(pos) {
        const plays = (await LegalMoves(pos)) ?? [];
        if (!plays.length) return { list: [], unranked: false };

        // Plain lookup, not a Map: local to this call, never read reactively.
        const byNotation = Object.create(null);
        plays.forEach((play, index) => {
            if (byNotation[play.notation] === undefined) byNotation[play.notation] = index;
        });

        try {
            const pruneK = await GetGammonNetPruneK();
            const result = await EvaluatePositionImmediate(pos, pruneK, 0);
            const moves = result?.refused ? [] : (result?.moves ?? []);
            const list = moves
                .map((move) => ({ move, gen: byNotation[move.move] }))
                .filter((row) => row.gen !== undefined)
                // Réduits par les pas joués au plateau (ADR-0052).
                .map((row) => ({ ...row, steps: plays[row.gen]?.steps ?? [] }));
            if (list.length) return { list, unranked: false };
        } catch (err) {
            logger.error('The 0-ply ranking of a transcription roll failed:', err);
        }
        // The generator's own order, which is an order and not a ranking.
        return { list: plays.map((play, index) => ({ move: { index, move: play.notation }, gen: index, steps: play.steps ?? [] })), unranked: true };
    }

    /**
     * Answers the awaited roll: no legal play is a dance, else the first is
     * preselected. A stale answer is dropped, never applied to a newer roll.
     */
    async function settleCandidates() {
        if (!get(transcriptionKeyStore).awaitingCandidates) return;
        const pos = rollPosition();
        if (!pos) return;

        const generation = ++candidateGeneration;
        let answer;
        try {
            answer = await computeCandidates(pos);
        } catch (err) {
            logger.error('The legal plays of a transcription roll failed:', err);
            error = String(err);
            return;
        }
        if (generation !== candidateGeneration) return;

        ranked = answer.list;
        unranked = answer.unranked;
        danced = answer.list.length === 0;

        const next = applyCandidates(get(transcriptionKeyStore), answer.list.length);
        transcriptionKeyStore.set(next.state);
        await run(next.commands);
    }

    // The engine moves the Cursor; the panel re-asks the candidates, an
    // Evaluation never stored (ADR-0045 rule 8).

    /**
     * Re-arms the panel on the Cursor's Action, recorded play selected; any
     * non-checker-play cell leaves the keyboard machine at rest.
     */
    async function settleCursor() {
        const ann = get(transcriptionStore)?.annotated;
        if (!ann) return;
        const info = (ann.actions ?? [])[ann.cursor ?? 0];
        ranked = [];
        unranked = false;
        danced = false;

        // Une insertion ne charge pas le voisin qu'elle repousse ; `entry.replacing` tranche.
        const inserting = ann.entry != null && ann.entry.replacing === false;
        // `unrecorded` : jet connu, coup à renseigner — candidats listés, aucun présélectionné.
        const playable = info?.kind === 'checker' || info?.kind === 'dance' || info?.kind === 'unrecorded';
        if (inserting || !info?.before || !playable) {
            resetTranscriptionKeys();
            return;
        }

        const pos = { ...structuredClone(info.before), id: 0 };
        const generation = ++candidateGeneration;
        let answer;
        try {
            answer = await computeCandidates(pos);
        } catch (err) {
            logger.error('The legal plays of the Action under the cursor failed:', err);
            error = String(err);
            return;
        }
        if (generation !== candidateGeneration) return;

        ranked = answer.list;
        unranked = answer.unranked;
        danced = answer.list.length === 0;

        // Recorded play found by notation; if illegal, the first candidate stands in.
        const played = ranked.findIndex((row) => row.move.move === info.notation);
        transcriptionKeyStore.set({
            ...initialKeyState(),
            phase: PHASE.ROLL,
            dice: [info.before.dice?.[0] ?? 0, info.before.dice?.[1] ?? 0],
            candidateCount: ranked.length,
            selected: played >= 0 ? played : 0
        });
    }

    /**
     * A click on a cell of the Transcript: the Cursor walks to that Action.
     *
     * @param {number} index
     */
    function selectAction(index, hole = false) {
        const ann = get(transcriptionStore)?.annotated;
        if (!ann) return;
        const from = currentStop(ann);
        const to = cursorStop(ann, index, hole);
        // Le second clic d'un double-clic (ADR-0052) arrive souvent avant le
        // retour du premier : ne pas refaire le chemin deux fois.
        if (to === from || to === walkingTo) return;
        walkingTo = to;
        run(cursorCommands(from, to))
            .then(settleCursor)
            .finally(() => {
                if (walkingTo === to) walkingTo = null;
            });
        panelEl?.focus({ preventScroll: true });
    }

    /**
     * A click on the hole of a double turn: the Cursor walks onto it, where the
     * missing turn is typed (ADR-0054).
     *
     * @param {number} index - the Action the hole stands before
     */
    function selectHole(index) {
        selectAction(index, true);
    }

    /** @type {number | null} L'arrêt ([cursorStop]) vers lequel le Cursor est en chemin. */
    let walkingTo = null;

    // Triangle des jets : le clic passe par la même machine que les touches
    // (`enterDicePair` = deux frappes), puis le focus revient au panneau.

    /** @param {{state: KeyState, commands: PanelCommand[]}} next */
    function applyMouseDice(next) {
        ranked = [];
        unranked = false;
        danced = false;
        transcriptionKeyStore.set(next.state);
        run(next.commands).then(settleCandidates);
        panelEl?.focus({ preventScroll: true });
    }

    /**
     * Une case du triangle : le jet entier, dé fort d'abord.
     *
     * @param {number} high
     * @param {number} low
     */
    function pickDice(high, low) {
        if (!draft) return;
        // Coup déjà joué au plateau : la case choisit parmi les jets possibles.
        const play = /** @type {BoardPlayState | null} */ (get(quizPlayStore));
        if (play && !play.free && play.steps.length) {
            sendPlay([high, low], play.steps, null);
            panelEl?.focus({ preventScroll: true });
            return;
        }
        applyMouseDice(enterDicePair(get(transcriptionKeyStore), high, low, keyContext));
    }

    /**
     * Une case de la rangée des six : le dé d'un camp, à l'ouverture.
     *
     * @param {number} die
     */
    function pickDie(die) {
        if (!draft) return;
        applyMouseDice(enterSingleDie(get(transcriptionKeyStore), die, keyContext));
    }

    /** @param {number} index */
    function chooseCandidate(index) {
        const next = selectCandidate(get(transcriptionKeyStore), index);
        transcriptionKeyStore.set(next.state);
        run(next.commands);
        panelEl?.focus({ preventScroll: true });
    }

    // Chaque geste est atteignable à la souris (ADR-0048 décision 11).

    /**
     * Un cran de molette = `j`/`k`, sur la liste et sur le plateau (App.svelte
     * la laisse passer en mode TRANSCRIBE) : on reconnaît le coup par ses flèches.
     *
     * @param {number} delta
     */
    function stepCandidate(delta) {
        const state = get(transcriptionKeyStore);
        if (state.candidateCount <= 0) return false;
        const next = selectCandidate(state, state.selected + delta);
        if (next.state.selected === state.selected && state.phase === next.state.phase) return true;
        transcriptionKeyStore.set(next.state);
        run(next.commands);
        return true;
    }

    // Molette sur le plateau, relayée par un magasin (le panneau ne reçoit pas ses événements).
    $effect(() => {
        const wanted = $transcriptionWheelStore;
        if (!wanted) return;
        transcriptionWheelStore.set(null);
        if (!draft || $activeTabStore !== 'transcription') return;
        stepCandidate(wanted.delta);
    });

    /** @param {WheelEvent} event */
    function wheelCandidates(event) {
        if (!draft || !visible.length) return;
        const delta = event.deltaY > 0 ? 1 : event.deltaY < 0 ? -1 : 0;
        if (delta === 0) return;
        event.preventDefault();
        stepCandidate(delta);
    }

    /**
     * Double-clic : valider (= `Entrée`) ; seul chemin souris pour le dernier
     * coup d'une partie, sans jet suivant pour le valider.
     *
     * @param {number} index
     */
    function commitCandidate(index) {
        if (!draft) return;
        const next = selectCandidate(get(transcriptionKeyStore), index);
        transcriptionKeyStore.set(initialKeyState());
        ranked = [];
        unranked = false;
        danced = false;
        run([...next.commands, { kind: COMMAND.VALIDATE }]).then(settleCandidates);
        panelEl?.focus({ preventScroll: true });
    }

    /** Un clic sur les cases du jet les efface : l'équivalent de Retour arrière. */
    function clearDice() {
        if (!draft) return;
        const state = get(transcriptionKeyStore);
        if (state.phase === PHASE.DICE) {
            noticeTranscription('transcription.notice.noDice');
            return;
        }
        applyResult({ state: { ...initialKeyState(), tie: state.tie }, commands: [{ kind: COMMAND.CLEAR }] });
        panelEl?.focus({ preventScroll: true });
    }

    // Videau à la souris (rangée [D][T][P][R] et videau du plateau) : mêmes
    // `cubeGesture`/`beginResign` que `d`/`t`/`p`/`r`, puis `applyResult`.

    /**
     * Un des trois gestes de videau : double, prise, passe.
     *
     * @param {string} kind
     */
    function sendCube(kind) {
        if (!draft) return;
        applyResult(cubeGesture(get(transcriptionKeyStore), kind));
        panelEl?.focus({ preventScroll: true });
    }

    /** `[R]` : la résignation annoncée, son niveau attendu. Rien n'est écrit. */
    function startResign() {
        if (!draft) return;
        applyResult(beginResign(get(transcriptionKeyStore)));
        panelEl?.focus({ preventScroll: true });
    }

    /**
     * Le niveau donné au clic — le second des deux gestes d'une résignation.
     *
     * @param {number} level
     */
    function pickResignLevel(level) {
        if (!draft) return;
        applyResult(resignWithLevel(get(transcriptionKeyStore), level));
        panelEl?.focus({ preventScroll: true });
    }

    /** La résignation abandonnée : le bouton qui double `Échap`. */
    function abortResign() {
        if (!draft) return;
        applyResult(cancelResign(get(transcriptionKeyStore)));
        panelEl?.focus({ preventScroll: true });
    }

    // Videau cliqué sur le plateau (utils/boardInteractions.js) : jugé ici,
    // jeté devant une offre, où la réponse revient au camp d'en face.
    $effect(() => {
        const wanted = $transcriptionCubeRequestStore;
        if (!wanted) return;
        transcriptionCubeRequestStore.set(null);
        if (!draft || !canCubeAct) return;
        sendCube(COMMAND.DOUBLE);
    });

    // Menu contextuel : `i`/`a`/`x`/`s` agissent au Cursor, donc le menu y mène
    // d'abord le Cursor (`menuCommands`).

    let transcriptMenu = $state(/** @type {{index: number, x: number, y: number} | null} */ (null));

    /**
     * @param {number} index
     * @param {{x: number, y: number}} at
     */
    function openTranscriptMenu(index, at) {
        if (!draft) return;
        transcriptMenu = { index, x: at.x, y: at.y };
    }

    /**
     * Une entrée du menu : le Cursor mené à la cellule, puis la correction.
     *
     * @param {number} index
     * @param {string} kind
     */
    function menuCommand(index, kind) {
        if (!draft) return;
        const ann = get(transcriptionStore)?.annotated;
        if (!ann) return;
        resetTranscriptionKeys();
        ranked = [];
        unranked = false;
        danced = false;
        run(menuCommands(currentStop(ann), cursorStop(ann, index), kind)).then(settleCursor);
        panelEl?.focus({ preventScroll: true });
    }

    let transcriptMenuItems = $derived.by(() => {
        if (!transcriptMenu) return [];
        const at = transcriptMenu.index;
        return [
            { label: $t('transcription.insertBefore'), onClick: () => menuCommand(at, COMMAND.INSERT_BEFORE) },
            { label: $t('transcription.insertAfter'), onClick: () => menuCommand(at, COMMAND.INSERT_AFTER) },
            { label: $t('transcription.delete'), onClick: () => menuCommand(at, COMMAND.DELETE) },
            { label: $t('transcription.flipSide'), onClick: () => menuCommand(at, COMMAND.FLIP_SIDE) }
        ];
    });

    // The panel owns its keys while focused (digits are dice, ux.md §3);
    // panelKeyGuard says what it may never swallow.

    /** @param {KeyboardEvent} event */
    function handleKeyDown(event) {
        if (!draft) return handleListKeyDown(event);
        // Ctrl+Entrée (Ctrl+S est pris globalement) : lu avant panelKeyGuard,
        // qui laisse passer tout combo Ctrl, puis arrêté.
        if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
            if (!panelEl?.contains(document.activeElement)) return;
            event.preventDefault();
            event.stopPropagation();
            handleSave();
            return;
        }
        if (panelKeyGuard(event)) return;
        if (!panelEl?.contains(document.activeElement)) return;

        // Coup en cours au plateau : Retour arrière défait le dernier pas.
        if (event.key === 'Backspace' && ($quizPlayStore?.steps?.length ?? 0) > 0) {
            event.preventDefault();
            event.stopPropagation();
            undoBoardPlayStep();
            return;
        }
        // Coup hors des règles : Entrée l'enregistre avec son plateau (ADR-0052).
        if (event.key === 'Enter' && $quizPlayStore?.free && $quizPlayStore.steps.length) {
            event.preventDefault();
            event.stopPropagation();
            commitFreePlay();
            return;
        }
        // Un coup libre est enregistré avant que le chiffre ne commence le jet
        // suivant, sinon le candidat sélectionné le remplacerait.
        if (dieOf(event) > 0 && $quizPlayStore?.free && $quizPlayStore.steps.length) {
            event.preventDefault();
            event.stopPropagation();
            commitFreePlay()?.then(() => applyResult(pressKey(get(transcriptionKeyStore), event, keyContext)));
            return;
        }

        const result = pressKey(keys, event, keyContext);
        // Retour arrière sur un jet vide : pris, sans effet.
        if (result.handled && event.key === 'Backspace' && !result.commands.length && keys.phase === PHASE.DICE) {
            event.preventDefault();
            event.stopPropagation();
            noticeTranscription('transcription.notice.noDice');
            return;
        }
        if (!result.handled) return;
        event.preventDefault();
        event.stopPropagation();
        applyResult(result);
    }

    // Liste des brouillons au clavier (ADR-0048 décision 10). Triée du plus
    // récent au plus ancien : `Ctrl+Maj+T` `Entrée` rouvre le brouillon en cours.

    let listIndex = $state(0);

    // La liste change : le surlignage revient en haut.
    $effect(() => {
        const rows = $transcriptionListStore;
        if (listIndex >= rows.length) listIndex = 0;
    });

    /** @param {KeyboardEvent} event */
    function handleListKeyDown(event) {
        if ($activeTabStore !== 'transcription') return;
        if (panelKeyGuard(event)) return;
        if (!panelEl?.contains(document.activeElement)) return;

        const rows = $transcriptionListStore;
        const delta = selectionDelta(event);
        if (delta !== 0 && rows.length) {
            event.preventDefault();
            event.stopPropagation();
            listIndex = Math.min(Math.max(listIndex + delta, 0), rows.length - 1);
            return;
        }
        if (event.key === 'Enter' && !showForm && rows[listIndex]) {
            event.preventDefault();
            event.stopPropagation();
            openDraft(rows[listIndex]);
            return;
        }
        if (isBareLetter(event, 'n') && !showForm) {
            event.preventDefault();
            event.stopPropagation();
            openForm();
        }
    }

    /**
     * La suite de tout geste (touche, triangle, videau), écrite une seule fois
     * pour qu'un bouton et sa touche ne divergent jamais.
     *
     * @param {{state: KeyState, commands: PanelCommand[]}} result
     */
    function applyResult(result) {
        // A restarted or validated roll drops its list before any gesture can `select` it.
        if (result.state.phase === PHASE.DICE || result.state.phase === PHASE.DIE1) {
            ranked = [];
            unranked = false;
        }
        danced = false;

        transcriptionKeyStore.set(result.state);
        // Moving/editing re-arms on the Cursor (ADR-0045 rule 8); filling a die
        // waits for the roll. A correction key takes the button's path.
        if (result.commands.length === 1 && EDITS.has(result.commands[0].kind)) {
            runCommand(result.commands[0].kind);
            return;
        }

        const rearms = result.commands.some((c) => REARMING.has(c.kind));
        run(result.commands).then(rearms ? settleCursor : settleCandidates);
    }

    // Document-changing commands a button also offers: both go through runCommand.
    /** @type {Set<string>} */
    const EDITS = new Set([COMMAND.INSERT_BEFORE, COMMAND.INSERT_AFTER, COMMAND.DELETE, COMMAND.FLIP_SIDE, COMMAND.UNDO, COMMAND.REDO]);
    // Commands after which the panel re-reads the Cursor the engine chose:
    // walking, every edit, and the cube gestures (which may correct in place).
    /** @type {Set<string>} */
    const REARMING = new Set([COMMAND.CURSOR_BACK, COMMAND.CURSOR_FORWARD, COMMAND.DOUBLE, COMMAND.TAKE, COMMAND.PASS, COMMAND.RESIGN, ...EDITS]);

    /**
     * Runs one command from a key, a button or the global dispatcher.
     *
     * At the end of the document a camp change has no Action and does nothing;
     * delete steps back onto the last Action (ADR-0050).
     *
     * @param {string} kind
     */
    function runCommand(kind) {
        if (!draft) return;
        // Un geste sans effet répond dans la barre d'état (ADR-0048 décision 9).
        const nothingToDelete = kind === COMMAND.DELETE && !(annotated?.actions?.length ?? 0) && !annotated?.entry;
        if ((kind === COMMAND.FLIP_SIDE && !onAction) || nothingToDelete) {
            noticeTranscription('transcription.notice.noAction');
            return;
        }
        if (kind === COMMAND.UNDO && !history.canUndo) {
            noticeTranscription('transcription.notice.nothingToUndo');
            return;
        }
        if (kind === COMMAND.REDO && !history.canRedo) {
            noticeTranscription('transcription.notice.nothingToRedo');
            return;
        }
        resetTranscriptionKeys();
        ranked = [];
        unranked = false;
        danced = false;
        run([{ kind }]).then(settleCursor);
        panelEl?.focus({ preventScroll: true });
    }

    // Ctrl combos are always global (isAlwaysGlobal): undo/redo arrive through a store.
    $effect(() => {
        const wanted = $transcriptionHistoryActionStore;
        if (!wanted) return;
        transcriptionHistoryActionStore.set(null);
        if ($activeTabStore !== 'transcription') return;
        runCommand(wanted === 'redo' ? COMMAND.REDO : COMMAND.UNDO);
    });

    onMount(() => {
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('keydown', handleKeyDown);
        clearTimeout(matCopyTimer);
        clearTranscriptionNotice();
        selectedMoveStore.set(null);
        // Démonté, le panneau retire son coup du plateau.
        quizPlayStore.set(null);
        boardPlayArmed = false;
        // Pas de clic de videau périmé au remontage.
        transcriptionCubeRequestStore.set(null);
    });

    // Focus on tab entry so the first keystroke lands (ADR-0048 décision 10),
    // unless a Transcript cell is being typed in (ADR-0052).
    $effect(() => {
        void draft;
        if ($activeTabStore !== 'transcription') return;
        const active = document.activeElement;
        if (active && panelEl?.contains(active) && active.matches('input, textarea, select')) return;
        panelEl?.focus({ preventScroll: true });
    });

    // Le panneau pose l'état (barre de match) et l'Action attendue (barre
    // d'état) sans les dessiner (ADR-0048 décision 2, ux.md §5).

    $effect(() => {
        if (!draft || !annotated) {
            transcriptionInfoStore.set(null);
            return;
        }
        transcriptionInfoStore.set({
            lengthKey: matchLength === 0 ? 'transcription.money' : 'transcription.points',
            lengthParams: { n: matchLength },
            score: matchLength > 0 ? [score[0], score[1]] : null,
            crawford: annotated?.next?.crawford === true,
            gameNumber,
            cubeKey: cubeLabelKey.key,
            cubeParams: cubeLabelKey.params,
            onRoll: playerName(sideOnRoll),
            player1: playerName(0),
            player2: playerName(1)
        });
    });

    /**
     * L'Action attendue en un mot, toujours une tant qu'un brouillon est ouvert.
     * Pas d'instruction permanente : sa place est `raccourcis.rst` (décision 8).
     */
    let promptState = $derived.by(() => {
        if (!draft) return null;
        if (matchOver) return { key: 'transcription.matchOver', params: { player: playerName(matchWinner), a: score[0], b: score[1] } };
        if (keys.phase === PHASE.RESIGN) return { key: 'transcription.resignPrompt', params: { player: playerName(sideOnRoll) } };
        if (awaitingAnswer) return { key: 'transcription.answerPrompt', params: { player: playerName(sideOnRoll) } };
        if (underReview) return { key: 'transcription.reviewHint', params: {} };
        if (correcting) return { key: 'transcription.correcting', params: {} };
        if (keys.tie) return { key: 'transcription.tie', params: {} };
        if (danced) return { key: 'transcription.dance', params: {} };
        if (expects === 'opening') return { key: 'transcription.openingPrompt', params: {} };
        return { key: 'transcription.rollPrompt', params: { player: playerName(sideOnRoll) } };
    });

    $effect(() => {
        transcriptionPromptStore.set(promptState);
    });

    // ── the board ────────────────────────────────────────────────────────

    // The board shows the Cursor's Action's starting Position (the match's
    // current one at the end), with the dice as typed — except an opening's,
    // which belong to two players and stay in the panel.
    /**
     * @param {any} ann
     * @param {number[]} dice
     * @param {boolean} opening
     */
    function boardPosition(ann, dice, opening) {
        const actions = ann.actions ?? [];
        const at = ann.cursor ?? 0;
        const current = at >= 0 && at < actions.length ? actions[at] : null;
        const base = positionAt(ann, at);
        if (!base) return null;
        const rolled = (!opening || current) && dice[0] > 0 && dice[1] > 0 ? [dice[0], dice[1]] : [0, 0];
        const pos = { ...structuredClone(base), id: 0, dice: rolled };
        // Dés du côté du camp de l'Action saisie, pas du voisin qu'une insertion repousse.
        if (ann.entry && ann.entry.at === at && typeof ann.entry.side === 'number') pos.player_on_roll = ann.entry.side;
        return pos;
    }

    $effect(() => {
        if (!annotated || $statusBarModeStore !== 'TRANSCRIBE') return;
        const pos = boardPosition(annotated, keys.dice, expects === 'opening');
        if (pos) positionStore.set(pos);
    });

    // Arrows of the selected candidate: display only, never written to the library.
    $effect(() => {
        if (!draft || !ranked.length || keys.phase === PHASE.DICE || keys.phase === PHASE.DIE1) {
            selectedMoveStore.set(null);
            return;
        }
        selectedMoveStore.set(visible[keys.selected]?.move?.move ?? null);
    });

    // ── what the panel reads ─────────────────────────────────────────────

    const columns = $derived([
        { key: 'updated', label: $t('transcription.updated') },
        { key: 'players', label: $t('transcription.players') },
        { key: 'length', label: $t('transcription.length'), narrow: true, align: 'right' },
        { key: 'actions', label: $t('transcription.actionCount'), narrow: true, align: 'right' },
        { key: 'match', label: $t('transcription.match'), narrow: true }
    ]);

    // Timestamps are text ("YYYY-MM-DD hh:mm:ss"); seconds dropped.
    /** @param {string | null | undefined} stamp */
    function shortStamp(stamp) {
        return stamp && stamp.length >= 16 ? stamp.slice(0, 16) : (stamp ?? '');
    }

    /** @param {any} row */
    function playersOf(row) {
        if (row.label) return row.label;
        const pair = [row.player1, row.player2].filter(Boolean);
        return pair.length ? pair.join(' — ') : $t('transcription.unnamed');
    }

    // 0 = money; -1 = unreadable draft, still listed so it can be deleted.
    /** @param {any} row */
    function lengthOf(row) {
        if (row.match_length < 0) return '—';
        return row.match_length === 0 ? $t('transcription.money') : $t('transcription.points', { n: row.match_length });
    }

    let matchLength = $derived(annotated?.document?.header?.match_length ?? 0);
    let score = $derived(annotated?.score ?? [0, 0]);
    let sideOnRoll = $derived(annotated?.next?.side ?? 0);

    /** @param {number} side */
    function playerName(side) {
        const header = annotated?.document?.header ?? {};
        const named = side === 0 ? header.player1 : header.player2;
        return named || (side === 0 ? $t('transcription.player1') : $t('transcription.player2'));
    }

    // Cube, game and match end are read off the Replay, never derived (ADR-0045 rule 9).

    // `value` is the log2 exponent; `owner` -1 (domain.None) = centred.
    let cube = $derived(annotated?.next?.position?.cube ?? { owner: -1, value: 0 });
    // A pending double shows the offered level (fonctionnel.md §1.2); it is a
    // fact of the match, read at the document's end, not at the Cursor.
    let awaitingAnswer = $derived(awaits === 'take');
    // La clé, non la phrase : seule la barre de match traduit.
    let cubeLabelKey = $derived.by(() => {
        const v = 1 << (cube.value ?? 0);
        if (awaitingAnswer) return { key: 'transcription.doubleOffered', params: { v } };
        if (cube.owner !== 0 && cube.owner !== 1) return { key: 'transcription.cubeCentred', params: { v } };
        return { key: 'transcription.cubeOwned', params: { v, player: playerName(cube.owner) } };
    });

    let matchOver = $derived(annotated?.finished === true);
    let matchWinner = $derived(annotated?.winner ?? -1);
    let gameNumber = $derived(annotated?.next?.game_number ?? 1);

    // Inconsistencies are shown, never refused (ADR-0044); the text comes from the
    // kind, never the engine's English detail.
    let lastFlags = $derived.by(() => {
        const actions = annotated?.actions ?? [];
        const last = actions[actions.length - 1];
        return (last?.inconsistencies ?? []).map((/** @type {{kind: string}} */ i) => $t(`transcription.inconsistency.${i.kind}`));
    });

    // Boutons de correction : chacun double une touche (ux.md §3), pour la découverte.

    let history = $derived($transcriptionHistoryStore);
    // Insérer, supprimer, changer de camp n'ont de sens que sur une Action.
    let onAction = $derived((annotated?.actions?.length ?? 0) > 0 && (annotated?.cursor ?? 0) < (annotated?.actions?.length ?? 0));
    // « À revoir » : le coup enregistré n'est pas du nouveau jet — dit par le moteur (fonctionnel.md §2).
    let underReview = $derived(annotated?.entry?.review === true);
    let correcting = $derived(annotated?.entry?.replacing === true);

    // The dice as typed, "·" for a missing one.
    let dieCells = $derived([keys.dice[0] || '·', keys.dice[1] || '·']);

    // Triangle seulement quand un jet est attendu : pas de cible qui ne répond à rien.
    let diceEntryOpen = $derived(!!draft && !matchOver && !awaitingAnswer && keys.phase !== PHASE.RESIGN && DICE_KINDS.has(expects));

    // Rangée [D][T][P][R] : annonce du camp au trait ou réponse d'en face, jamais
    // les quatre ; masquée match fini. Le clavier, lui, accepte tout (ADR-0044).
    let resigning = $derived(keys.phase === PHASE.RESIGN);
    let cubeRowOpen = $derived(!!draft && !matchOver);
    let canCubeAct = $derived(cubeRowOpen && !awaitingAnswer && !resigning);
    // [T]/[P] répondent à une offre ou corrigent la cellule tenue par le curseur.
    let canCubeAnswer = $derived(cubeRowOpen && (awaitingAnswer || editingSlot) && !resigning);

    // Chaque pas joué ne garde que les candidats qui le contiennent (règle
    // d'`alivePlays`, ADR-0052). `ranked` reste complet ; `visible` est ce que
    // tout rang désigne (`j`/`k`, clic, flèches).
    let visible = $derived.by(() => {
        const play = /** @type {BoardPlayState | null} */ ($quizPlayStore);
        if (!play?.rolled) return ranked;
        if (play.free) return [];
        if (!play.steps.length) return ranked;
        return ranked.filter((row) => containsSteps(row.steps ?? [], play.steps));
    });
    let playFree = $derived(/** @type {BoardPlayState | null} */ ($quizPlayStore)?.free === true);

    // Pas changés : présélectionner le premier restant. Gardé par la clé des pas,
    // pour ne pas écraser la présélection posée par `settleCursor`.
    let stepsKey = '';
    $effect(() => {
        const play = /** @type {BoardPlayState | null} */ ($quizPlayStore);
        const list = visible;
        const key = !play?.rolled ? '' : play.free ? 'free' : play.steps.map((s) => `${s.from}>${s.to}`).join(' ');
        if (key === stepsKey) return;
        stepsKey = key;
        if (!play?.rolled) return;

        const state = get(transcriptionKeyStore);
        if (state.phase !== PHASE.ROLL && state.phase !== PHASE.CANDIDATE) return;
        transcriptionKeyStore.set({ ...state, candidateCount: list.length, selected: 0 });
        if (list.length) run([{ kind: COMMAND.SELECT, index: 0 }]);
    });

    let rankedMoves = $derived(visible.map((row) => row.move));
    // Which referential the equity column is stated in (ADR-0016 point 6,
    // ADR-0019): money points at money play, normalised match equity at a score.
    let isMoney = $derived(isMoneyPosition(annotated?.next?.position));

    // Coup joué au plateau (ADR-0052), sans geste d'armement. Dés non saisis :
    // union des coups des 21 jets, dés déduits des pas. Dés saisis : coups de ce
    // jet ; un coup légal achevé part seul, un glissé hors règles attend Entrée.
    // Le coup vit dans `quizPlayStore` (services/quizPlay.js) ; ici : armement,
    // déduction, envoi.

    // Désarme le plateau pendant l'envoi, sinon chaque réponse réarmerait le coup.
    let recordingPlay = $state(false);
    let boardPlayArmed = false;
    let boardPlayGeneration = 0;

    let diceEntered = $derived(keys.dice[0] > 0 && keys.dice[1] > 0);
    // Le jet saisi (ou chargé par `settleCursor` sur une Action relue).
    let boardRoll = $derived(diceEntered && (keys.phase === PHASE.ROLL || keys.phase === PHASE.CANDIDATE) ? [keys.dice[0], keys.dice[1]] : null);
    let boardPlayOpen = $derived(!!draft && !matchOver && !awaitingAnswer && !recordingPlay && expects === 'checker' && (keys.phase === PHASE.DICE || boardRoll !== null));

    // Clé d'armement (place, camp, position, jet) — pas le document, qui change à chaque pas.
    let boardPlayKey = $derived.by(() => {
        if (!boardPlayOpen || !annotated) return '';
        const at = annotated.entry?.at ?? annotated.cursor ?? 0;
        const side = annotated.entry ? annotated.entry.side : annotated.next?.side;
        return JSON.stringify([at, side, boardRoll, annotated.actions?.length ?? 0, positionAt(annotated, at)?.board ?? null]);
    });

    /**
     * Coups légaux du jet saisi, ou union des 21 jets en une salve.
     *
     * @param {any} pos
     * @param {number} generation
     * @param {number[] | null} rolled
     */
    async function loadBoardPlay(pos, generation, rolled) {
        const rolls = rolled ? [rolled] : ROLLS;
        const answers = await Promise.all(rolls.map(([high, low]) => LegalMoves({ ...pos, dice: [high, low] }).catch(() => [])));
        if (generation !== boardPlayGeneration) return;
        const byRoll = rolls.map((dice, index) => ({ dice, plays: answers[index] ?? [] })).filter((entry) => entry.plays.length);
        stepsKey = '';
        quizPlayStore.set(newBoardPlay(pos, byRoll, { rolled }));
        boardPlayArmed = true;
    }

    $effect(() => {
        const key = boardPlayKey;
        const rolled = untrack(() => boardRoll);
        const generation = ++boardPlayGeneration;
        if (!key) {
            if (boardPlayArmed) {
                stepsKey = '';
                quizPlayStore.set(null);
                boardPlayArmed = false;
            }
            return;
        }
        const pos = entryPosition();
        if (!pos) return;
        loadBoardPlay(pos, generation, rolled);
    });

    /**
     * Coup achevé et un seul jet possible : l'Action part seule. Plusieurs jets :
     * `deducedDice` rend `null` et le triangle tranche.
     */
    $effect(() => {
        const play = /** @type {BoardPlayState | null} */ ($quizPlayStore);
        if (!play || play.free || recordingPlay) return;
        const dice = deducedDice(play);
        if (dice) sendPlay(play.rolled ?? dice, play.steps, null);
    });

    /** Les jets que l'utilisateur peut encore désigner, ou `null` pour tous. */
    let rollsAllowed = $derived.by(() => {
        const play = /** @type {BoardPlayState | null} */ ($quizPlayStore);
        if (!play || play.free || play.rolled || play.steps.length === 0) return null;
        return new Set(choosableRolls(play));
    });

    /**
     * L'Action : dés, pas, et plateau si besoin. Les dés sont toujours renvoyés
     * (`enter_die` sur une Entry pleine recommence le jet à l'identique) ;
     * `lead` mène d'abord le Cursor à la cellule.
     *
     * @param {number[]} dice
     * @param {{from: number, to: number}[]} steps
     * @param {any} board
     * @param {PanelCommand[]} [lead]
     */
    async function sendPlay(dice, steps, board, lead = []) {
        if (recordingPlay || !draft) return;
        recordingPlay = true;
        quizPlayStore.set(null);
        boardPlayArmed = false;
        stepsKey = '';
        ranked = [];
        unranked = false;
        danced = false;
        try {
            await run([...lead, { kind: COMMAND.DIE, value: dice[0] }, { kind: COMMAND.DIE, value: dice[1] }, { kind: COMMAND.ENTER_PLAY, steps, board }, { kind: COMMAND.VALIDATE }]);
        } finally {
            recordingPlay = false;
        }
        resetTranscriptionKeys();
    }

    /**
     * Entrée sur un coup hors des règles (ADR-0052) : le plateau part avec les
     * pas et tranche quand ils ne suffisent pas. Jamais envoyé seul.
     */
    function commitFreePlay() {
        const play = /** @type {BoardPlayState | null} */ (get(quizPlayStore));
        if (!play?.free || !play.steps.length || !play.rolled) return null;
        return sendPlay(play.rolled, play.steps, play.board);
    }

    // Cellules dont le coup se tape (ADR-0052) : celles que `settleCursor` rejoue.
    const TYPABLE_KINDS = new Set(['checker', 'dance', 'unrecorded']);

    /**
     * Coup tapé dans une cellule du Transcript (`13/7 8/7*`, ADR-0052), dés de
     * la cellule, parsé par `parseMoveNotation`. Rien n'est jugé : un illégal
     * est enregistré et marqué (ADR-0044).
     *
     * @param {number} index l'Action de la cellule
     * @param {string} text
     * @param {boolean} pending la cellule en pointillés de la saisie en cours
     * @returns {boolean} vrai quand le coup part — le champ se ferme alors
     */
    function commitNotation(index, text, pending) {
        const ann = get(transcriptionStore)?.annotated;
        if (!draft || !ann || recordingPlay) return false;
        /** @type {any} */
        let pos;
        /** @type {number[]} */
        let dice;
        /** @type {PanelCommand[]} */
        let lead = [];
        if (pending) {
            if (!ann.entry || ann.entry.kind === 'opening') return false;
            pos = entryPosition();
            dice = ann.entry.dice ?? [0, 0];
        } else {
            const info = (ann.actions ?? [])[index];
            if (!info?.before || !TYPABLE_KINDS.has(info.kind)) return false;
            pos = { ...structuredClone(info.before), player_on_roll: typeof info.side === 'number' ? info.side : info.before.player_on_roll };
            dice = info.before.dice ?? [0, 0];
            // Cursor pas encore sur la cellule : le chemin précède le coup, même file.
            lead = cursorCommands(currentStop(ann), cursorStop(ann, index));
        }
        if (!pos || !(dice[0] > 0) || !(dice[1] > 0)) return false;
        const mover = pos.player_on_roll;
        const steps = stepsFromNotation(text, mover);
        if (!steps.length) return false;
        sendPlay([dice[0], dice[1]], steps, boardAfterSteps(pos.board, steps, mover), lead);
        panelEl?.focus({ preventScroll: true });
        return true;
    }

    /**
     * Score annoncé d'une partie (ADR-0053) : `[p1, p2]`, ou `null` pour le
     * score déduit. Même file que les touches.
     *
     * @param {number} opening
     * @param {[number, number] | null} score
     */
    function declareScore(opening, score) {
        if (!draft) return false;
        sendGesture({ Kind: 'set_score', At: opening, Score: score });
        panelEl?.focus({ preventScroll: true });
        return true;
    }

    /** Retour arrière pendant un coup au plateau : le dernier pas est défait. */
    function undoBoardPlayStep() {
        quizPlayStore.update((play) => undoBoardStep(play));
    }

    // Le .mat en modale (ADR-0048 décision 6), rendu par ingest.RenderMAT via
    // TranscriptionMAT, demandé seulement modale ouverte. Une modale parce que
    // ses colonnes alignées demandent ~409 px (62 car.).
    let matOpen = $state(false);
    let matText = $state('');
    let matCopied = $state(false);
    /** @type {ReturnType<typeof setTimeout> | undefined} */
    let matCopyTimer = undefined;

    async function copyMat() {
        await writeTextToClipboard(matText);
        matCopied = true;
        clearTimeout(matCopyTimer);
        matCopyTimer = setTimeout(() => (matCopied = false), 1500);
    }

    $effect(() => {
        const id = draft?.id;
        void annotated;
        if (!matOpen || id == null) return;
        let live = true;
        TranscriptionMAT(id)
            .then((text) => {
                if (live) matText = text ?? '';
            })
            .catch((err) => {
                logger.error('The .mat text of a transcription draft failed:', err);
            });
        return () => {
            live = false;
        };
    });
</script>

<section class="transcription-panel" id="transcriptionPanel" aria-label={$t('transcription.title')} tabindex="-1" bind:this={panelEl}>
    {#if !draft}
        <PanelTable rows={$transcriptionListStore} {columns} emptyText={$t('transcription.empty')} pointerRows onSelect={openDraft}>
            {#snippet header()}
                <span class="detail-title">{$t('transcription.title')}</span>
                <button class="new-btn" onclick={openForm} disabled={busy || !$databaseLoadedStore} title={$t('transcription.newTooltip')}>
                    {$t('transcription.new')}
                </button>
            {/snippet}
            {#snippet subheader()}
                {#if showForm}
                    <form
                        class="create-form"
                        onsubmit={(e) => {
                            e.preventDefault();
                            createDraft();
                        }}
                    >
                        <label for="transcriptionLength">{$t('transcription.matchLength')}</label>
                        <!-- svelte-ignore a11y_autofocus -->
                        <input id="transcriptionLength" class="length-input" type="text" inputmode="numeric" autofocus bind:value={formLength} />
                        <span class="hint">{$t('transcription.moneyHint')}</span>
                        {#if formIsMoney}
                            <label class="rule"><input type="checkbox" bind:checked={formJacoby} /> {$t('transcription.jacoby')}</label>
                            <label class="rule"><input type="checkbox" bind:checked={formBeaver} /> {$t('transcription.beaver')}</label>
                        {/if}
                        <button class="new-btn" type="submit" disabled={busy || !formValid}>{$t('transcription.create')}</button>
                        <button class="new-btn" type="button" onclick={() => (showForm = false)}>{$t('transcription.cancel')}</button>
                    </form>
                {/if}
            {/snippet}
            {#snippet cells(/** @type {any} */ row)}
                <td class="stamp-cell">{shortStamp(row.updated_at || row.created_at)}</td>
                <td>{playersOf(row)}</td>
                <td class="narrow-col align-right">{lengthOf(row)}</td>
                <td class="narrow-col align-right">{row.action_count < 0 ? '—' : row.action_count}</td>
                <td class="narrow-col">{row.match_id ? `#${row.match_id}` : $t('transcription.noMatch')}</td>
            {/snippet}
        </PanelTable>
    {:else}
        <div class="draft">
            <!-- Barre du brouillon : les gestes qui le font sortir, l'état du Match
                 et l'annulation à la souris (ADR-0048 décisions 2, 3 et 11). -->
            <div class="draft-bar">
                <button class="new-btn" onclick={backToList}>{$t('transcription.backToList')}</button>
                <span class="save-state">{$t(saveState.key, saveState.params)}</span>
                <span class="bar-gap"></span>
                <button
                    class="icon-btn"
                    class:active={$transcriptionBoardSwapStore}
                    onclick={() => transcriptionBoardSwapStore.update((v) => !v)}
                    title={$t('transcription.boardSwapTooltip')}
                    aria-label={$t('transcription.boardSwap')}
                    aria-pressed={$transcriptionBoardSwapStore}>⇅</button
                >
                <button class="icon-btn" onclick={() => runCommand(COMMAND.UNDO)} title={$t('transcription.undoTooltip')} aria-label={$t('transcription.undo')}>↶</button>
                <button class="icon-btn" onclick={() => runCommand(COMMAND.REDO)} title={$t('transcription.redoTooltip')} aria-label={$t('transcription.redo')}>↷</button>
                <button class="new-btn" onclick={() => (metaOpen = !metaOpen)} title={$t('transcription.metadataTooltip')}>{$t('transcription.metadata')}</button>
                <button class="new-btn" onclick={() => (matOpen = true)} title={$t('transcription.matModalTooltip')}>{$t('transcription.matModal')}</button>
                <button class="new-btn" onclick={handleExport} disabled={busy} title={$t('transcription.exportMatTooltip')}>{$t('transcription.exportMat')}</button>
                <button class="primary-btn" onclick={handleSave} disabled={busy} title={$t('transcription.saveTooltip')}>{saveLabel}</button>
                <button class="danger-btn" onclick={handleClose} disabled={busy} title={$t('transcription.closeDraftTooltip')}>{$t('transcription.closeDraft')}</button>
            </div>

            {#if metaOpen}
                <!-- En-tête du brouillon : rien n'y est exigé. -->
                <TranscriptionMetadata header={annotated?.document?.header ?? {}} apply={sendGesture} {busy} />
            {/if}

            <!-- Ordre du DOM = boîte étroite ; en large, `order` (ADR-0048 décision 5).
                 Rien ne s'intercale entre les cases du jet et le premier candidat. -->
            <div class="draft-body">
                <div class="candidates-col">
                    {#if unranked && visible.length}
                        <!-- Message sur la liste, donc dans son en-tête. -->
                        <div class="list-head">
                            <span class="list-note">{$t('transcription.unranked')}</span>
                        </div>
                    {/if}
                    {#if playFree}
                        <!-- Coup hors des règles : la touche qui l'enregistre (ADR-0052). -->
                        <p class="list-note" data-testid="transcription-free-hint">{$t('transcription.freePlayHint')}</p>
                    {/if}
                    {#if visible.length}
                        {#if unranked}
                            <ol class="plain-candidates" data-testid="transcription-candidates">
                                {#each visible as row, index (row.gen)}
                                    <li>
                                        <button class="plain-candidate" class:selected={index === keys.selected} onclick={() => chooseCandidate(index)} ondblclick={() => commitCandidate(index)}
                                            >{row.move.move}</button
                                        >
                                    </li>
                                {/each}
                            </ol>
                        {:else}
                            <!-- Molette = `j`/`k` (décision 11). -->
                            <div class="candidates" data-testid="transcription-candidates" role="listbox" tabindex="-1" aria-label={$t('transcription.candidatesLabel')} onwheel={wheelCandidates}>
                                <CandidateMovesTable
                                    moves={rankedMoves}
                                    selectedMove={$selectedMoveStore}
                                    onRowClick={(/** @type {any} */ move) => chooseCandidate(rankedMoves.indexOf(move))}
                                    onRowDblClick={(/** @type {any} */ move) => commitCandidate(rankedMoves.indexOf(move))}
                                    showProvenance={false}
                                    baseline={null}
                                    projection="identify"
                                    {isMoney}
                                />
                            </div>
                        {/if}
                    {/if}
                </div>

                <div class="palette-col" data-testid="transcription-palette">
                    <!-- Dés et gestes de videau sur une ligne, triangle dessous (décision 7). -->
                    <div class="entry-row" data-testid="transcription-dice">
                        {#if !awaitingAnswer && keys.phase !== PHASE.RESIGN}
                            <!-- Cliquables : Retour arrière à la souris. -->
                            <button class="die" class:filled={keys.dice[0] > 0} onclick={clearDice} title={$t('transcription.clearDice')} aria-label={$t('transcription.clearDice')}
                                >{dieCells[0]}</button
                            >
                            <button class="die" class:filled={keys.dice[1] > 0} onclick={clearDice} title={$t('transcription.clearDice')} aria-label={$t('transcription.clearDice')}
                                >{dieCells[1]}</button
                            >
                        {/if}
                    </div>

                    {#if cubeRowOpen}
                        <CubeActionRow canAct={canCubeAct} canAnswer={canCubeAnswer} {resigning} onGesture={sendCube} onResign={startResign} onLevel={pickResignLevel} onCancelResign={abortResign} />
                    {/if}

                    {#if diceEntryOpen}
                        <!-- Sous les cases du jet, jamais à leur place : le clavier reste deux fois plus rapide. -->
                        <DiceTriangle single={expects === 'opening'} allowed={rollsAllowed} onPick={pickDice} onDie={pickDie} />
                    {/if}
                </div>

                <div class="transcript-col">
                    {#if lastFlags.length}
                        <!-- Incohérence marquée, jamais refusée (ADR-0044), visible sans chercher. -->
                        <p class="flag">{$t('transcription.inconsistencyPrefix')} {lastFlags.join(' · ')}</p>
                    {/if}
                    <TranscriptView
                        {annotated}
                        cursor={annotated?.cursor ?? 0}
                        players={[playerName(0), playerName(1)]}
                        onSelect={selectAction}
                        onHole={selectHole}
                        onMenu={openTranscriptMenu}
                        onEditMove={commitNotation}
                        onEditScore={declareScore}
                    />
                </div>
            </div>
        </div>
    {/if}
    {#if error}
        <p class="error">{error}</p>
    {/if}
    {#if transcriptMenu}
        <ContextMenu x={transcriptMenu.x} y={transcriptMenu.y} items={transcriptMenuItems} onClose={() => (transcriptMenu = null)} />
    {/if}
</section>

<!-- `wide` : 62 caractères alignés demandent ~409 px (décision 6). -->
<Modal open={matOpen} onclose={() => (matOpen = false)} size="wide" closeOnOverlay label={$t('transcription.matModal')}>
    <h2 class="mat-title">{$t('transcription.matModal')}</h2>
    <div class="mat-body">
        <button class="new-btn" type="button" onclick={copyMat}>{matCopied ? $t('transcript.copied') : $t('transcript.copy')}</button>
        <pre class="mat-text">{matText}</pre>
    </div>
</Modal>

<style>
    .transcription-panel {
        display: flex;
        flex-direction: column;
        height: 100%;
        min-height: 0;
        outline: none;
    }

    .detail-title {
        font-weight: 600;
    }

    .new-btn {
        margin-left: auto;
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    .new-btn:hover:not(:disabled) {
        background: var(--color-surface-alt);
    }

    .new-btn:disabled,
    .primary-btn:disabled,
    .danger-btn:disabled {
        color: var(--color-text-muted);
        cursor: default;
    }

    /* Hiérarchie (ADR-0048 décision 3) : action principale = matérialiser le Match. */
    .primary-btn,
    .danger-btn,
    .icon-btn {
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    .primary-btn {
        border-color: var(--color-primary);
        color: var(--color-primary);
        font-weight: 600;
    }

    .primary-btn:hover:not(:disabled),
    .danger-btn:hover:not(:disabled),
    .icon-btn:hover:not(:disabled) {
        background: var(--color-surface-alt);
    }

    .danger-btn {
        color: var(--color-danger);
    }

    .icon-btn {
        min-width: 1.9em;
        text-align: center;
    }

    .icon-btn.active {
        border-color: var(--color-primary);
        color: var(--color-primary);
        font-weight: 600;
    }

    .create-form {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--space-2);
        padding: var(--space-2);
        border-bottom: 1px solid var(--color-border);
    }

    .create-form .new-btn {
        margin-left: 0;
    }

    .length-input {
        width: 4em;
        padding: var(--space-1);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
    }

    .rule {
        display: inline-flex;
        align-items: center;
        gap: var(--space-1);
    }

    /* Chaîne de hauteur (ADR-0048 décision 5) : `flex: 1; min-height: 0` jusqu'en
       bas, sinon le panneau défile en bloc et saute à chaque Action. */
    .draft {
        display: flex;
        flex: 1;
        flex-direction: column;
        gap: var(--space-2);
        padding: var(--space-2);
        min-height: 0;
        container-type: inline-size;
    }

    .draft-bar {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--space-1);
        flex: 0 0 auto;
    }

    .draft-bar .new-btn {
        margin-left: 0;
    }

    /* Gestes à droite, l'état à gauche près de son bouton. */
    .bar-gap {
        flex: 1 1 auto;
    }

    .save-state {
        color: var(--color-text-muted);
    }

    /* Étroit : les trois régions se suivent ; les points de rupture les posent
       sur une grille nommée. Contrat : les deux cases du jet et cinq candidats
       sans défilement. Le Transcript partage la rangée du triangle dès 485 px
       (triangle 211 px + Transcript 250 px) ; en dessous il serait rogné. */
    .draft-body {
        display: flex;
        flex: 1;
        flex-direction: column;
        gap: var(--space-2);
        min-height: 0;
    }

    .candidates-col,
    .palette-col,
    .transcript-col {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        min-width: 0;
        min-height: 0;
    }

    .candidates-col {
        flex: 1 1 auto;
        grid-area: candidates;
    }

    /* Sous le plancher, la palette défile seule ; la ligne du jet reste visible. */
    .palette-col {
        flex: 0 0 auto;
        overflow: auto;
        grid-area: palette;
    }

    .transcript-col {
        flex: 1 1 auto;
        grid-area: transcript;
    }

    /* Le Transcript à côté du triangle, dans le blanc qu'il laissait. */
    @container (min-width: 500px) {
        .draft-body {
            display: grid;
            grid-template-columns: max-content minmax(0, 1fr);
            grid-template-rows: minmax(0, 1fr) auto;
            grid-template-areas:
                'candidates candidates'
                'palette    transcript';
        }
    }

    /* Trois de front, palette sous les dés (ADR-0048 décision 5). */
    @container (min-width: 900px) {
        .draft-body {
            grid-template-columns: 220px minmax(0, 1fr) minmax(0, 1.14fr);
            grid-template-rows: minmax(0, 1fr);
            grid-template-areas: 'palette candidates transcript';
        }
    }

    /* Les deux dés et les quatre gestes de videau sur une ligne. */
    .entry-row {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--space-1);
        flex: 0 0 auto;
    }

    .die {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 24px;
        height: 24px;
        padding: 0;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text-muted);
        font-variant-numeric: tabular-nums;
        cursor: pointer;
    }

    .die:hover {
        background: var(--color-surface-alt);
    }

    .die.filled {
        color: var(--color-text);
        font-weight: 600;
    }

    .list-head {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        flex: 0 0 auto;
    }

    /* Note sur la liste : ordre du générateur, ou coup hors des règles. */
    .list-note {
        margin: 0;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    /* Le seul conteneur qui défile. */
    .candidates {
        flex: 1;
        min-height: 0;
        overflow: auto;
        container-type: inline-size;
    }

    .plain-candidates {
        flex: 1;
        margin: 0;
        padding-left: var(--space-4);
        min-height: 0;
        overflow: auto;
    }

    .plain-candidate {
        padding: 0;
        border: none;
        background: none;
        color: var(--color-text);
        cursor: pointer;
    }

    .plain-candidate.selected {
        font-weight: 600;
    }

    .hint {
        margin: 0;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .flag {
        margin: 0;
        flex: 0 0 auto;
        color: var(--color-danger);
        font-size: var(--font-size-small);
    }

    .stamp-cell {
        white-space: nowrap;
        color: var(--color-text-muted);
    }

    .error {
        margin: var(--space-2);
        color: var(--color-danger);
    }

    .mat-title {
        margin: 0 0 var(--space-2);
        font-size: var(--font-size-title);
    }

    .mat-body {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
    }

    /* 62 caractères, la plus longue ligne d'un `.mat`. */
    .mat-text {
        margin: 0;
        max-height: 60vh;
        overflow: auto;
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
    }
</style>
