<!--
  TranscriptionPanel — the library's transcription drafts, and the draft open.

  A Transcription is a DRAFT of a match being typed in, distinct from the Match
  it produces (ADR-0045). This panel is both its front door — the drafts that
  exist, the form that starts one — and, once a draft is open, the surface the
  match is typed on.

  What is here: the creation form (fonctionnel.md §1.1 — the LENGTH is the only
  field asked for, `0` being a money session, which is what unfolds the Jacoby
  and beaver flags), the opening (§6 flux 2 — the die of player 1, the die of
  player 2, the stronger one starts and plays BOTH dice as its first checker
  play; a tie reads "relance" and waits for another opening), and the turn of
  checkers: two dice, every legal play ranked by a 0-ply Evaluation, the first
  preselected, `j`/`k` to walk them, a digit to validate and open the next roll.

  The candidate list is a 0-ply EVALUATION in the glossary's sense: it is shown,
  it is never written to the library (ADR-0045 rule 8). The ranking is the Eval
  panel's own — EvaluatePositionImmediate — and the table is the Eval panel's own
  component, mounted here exactly as EPCPanel mounts it.

  The cube is three keys: `d` doubles or redoubles, `t` takes, `p` passes, each
  of them validating a play left selected before it (fonctionnel.md §6 flux 5-7).
  Les quatre gestes de videau ont un second déclencheur, à la souris (T2.5) : la
  rangée [D] [T] [P] [R] sous le triangle des jets, et le videau dessiné sur le
  plateau, qui propose un double. Ce ne sont que des déclencheurs — ils passent
  par la machine des touches puis par `applyResult`, la suite d'une frappe —, et
  un clic droit sur une cellule du Transcript ouvre de même les quatre gestes de
  correction, le Cursor mené d'abord jusqu'à la cellule.
  A game ends by a pass, by a bear-off or past the cube — the score advances, the
  Crawford game is derived, and the next opening is expected, all of it read off
  the Replay and none of it computed here.

  A resignation is `r` then `1`/`2`/`3`, `Escape` between the two cancelling it:
  the game is given up by the side on roll, at that level times the cube, and it
  adds no Move at all to the saved match — only the winner and the points of the
  Game (ADR-0045 §6).

  The right half is the Transcript (T1.6): TranscriptView draws the two
  columns of the score sheet, the Cursor cell is framed, `h`/`l` and the left
  and right arrows walk it from cell to cell — from camp to camp — and a click
  on a cell does the same with the mouse. Moving it reloads the board on the
  Action aimed at and lists its candidates with the play that was recorded
  selected.

  The correction is the other half of the Transcript (T1.7). Walking back to a
  cell and typing again REPLACES the Action there, and validating gives the
  Cursor back where it came from; `i` and `a` insert beside it, `x` and Del
  delete it, `s` gives it to the other camp, `Ctrl+Z` and `Ctrl+Maj+Z` walk the
  session's undo stack — which lives in Go, in the draft's transcript.Editor,
  and which a crash is allowed to lose (ADR-0045 rule 1). None of them judges:
  an insertion beside its own camp makes a double turn, a deletion makes
  another, a changed camp can make the plays that follow illegal — all of them
  MARKED by the Replay and none of them refused (ADR-0044). After every Replay
  the Cursor lands on the first Inconsistency the gesture left behind.

  La barre du brouillon porte enfin les trois gestes qui le font sortir de
  lui-même (T1.9) : « Enregistrer » (Ctrl+Entrée) en fait un Match — créé la
  première fois, remplacé ensuite, même `id` — puis lance l'analyse ciblée
  dont la barre d'état montre la progression ; « Exporter .mat » écrit le
  fichier tel que le match a été tapé ; « Fermer le brouillon » supprime la
  ligne. Un avertissement précède les deux premiers quand le document porte
  des incohérences ou un coup illégal, et n'oppose jamais de refus (ADR-0044).
  Le calcul de tout cela est dans services/transcriptionSave.js ; ici il n'y a
  que des boutons.

  The panel is a CLIENT of the Go engine (ADR-0045 rule 9): every gesture goes
  to ApplyTranscriptionGesture and comes back as a whole annotated document. It
  derives no score, no Crawford and no side on roll of its own.
-->
<script>
    import { onMount, onDestroy } from 'svelte';
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
        initialKeyState,
        enterDicePair,
        enterSingleDie,
        cubeGesture,
        selectionDelta,
        beginResign,
        resignWithLevel,
        cancelResign,
        menuCommands,
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
        transcriptionPointFilterStore,
        transcriptionCandidateStepsStore,
        transcriptionCubeRequestStore,
        transcriptionWheelStore,
        transcriptionInfoStore,
        transcriptionPromptStore,
        setTranscription,
        clearTranscription,
        resetTranscriptionKeys,
        resetTranscriptionPointFilter,
        noticeTranscription,
        clearTranscriptionNotice
    } from '../stores/transcriptionStore.js';
    import Modal from './Modal.svelte';
    import { filterByPoints } from '../services/transcriptionFilter.js';
    import { writeTextToClipboard } from '../services/clipboardService.js';
    import { quizPlayStore } from '../stores/quizPlayStore.js';
    import { ROLLS, newBoardPlay, newFreePlay, deducedDice, choosableRolls, undoBoardStep, stepsFromNotation, boardAfterSteps } from '../services/transcriptionPlay.js';
    import { ListTranscriptions, CreateTranscription, OpenTranscription, ApplyTranscriptionGesture, TranscriptionMAT } from '../../wailsjs/go/database/Database.js';
    import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';
    import { GetGammonNetPruneK } from '../../wailsjs/go/main/Config.js';
    import { saveDraft, exportDraftMat, closeDraft, draftSaveState, transcriptionSaveStore, resetTranscriptionSave } from '../services/transcriptionSave.js';
    import { get } from 'svelte/store';

    // The length a first draft is offered when the library holds none. It is
    // Go's transcript.DefaultMatchLength; stated here because the form has to
    // show a value before any call is made (fonctionnel.md §1.1).
    const DEFAULT_MATCH_LENGTH = 7;

    let busy = $state(false);
    let error = $state('');
    let showForm = $state(false);
    // The length is text, not a number: an empty field is a state of its own
    // ("nothing said yet"), and an <input type="number"> bound to a number turns
    // it into 0 — which here means a money session, the one answer the user did
    // not give.
    let formLength = $state('');
    let formJacoby = $state(true);
    let formBeaver = $state(false);
    let panelEl = $state(null);
    // Le volet des métadonnées (T3.1) : replié par défaut, ouvrable à tout
    // moment, et il survit au va-et-vient entre les onglets comme le reste du
    // brouillon puisqu'il ne tient rien — l'en-tête est dans le document.
    let metaOpen = $state(false);

    let draft = $derived($transcriptionStore);
    let annotated = $derived(draft?.annotated ?? null);
    let keys = $derived($transcriptionKeyStore);
    let expects = $derived(annotated?.next?.expects ?? '');
    // Le Cursor est-il sur une Action existante que la saisie remplacerait ?
    // C'est le discriminant de la touche chiffrée (ADR-0048 décision 1) : en
    // bout de document elle valide, ici elle recommence le jet sur place.
    let replacing = $derived(annotated?.entry?.replacing === true);
    let keyContext = $derived({ expects, replacing });

    // A length of 0 is a money session, and money is the only case where the
    // session's rules are asked for (ADR-0028: rules of the session, posted on
    // every Position, never an Action).
    let formIsMoney = $derived(formLength.trim() === '0');
    let formValid = $derived(/^\d+$/.test(formLength.trim()));

    // The default the form opens on: the length of the most recently updated
    // draft, and 7 when the library holds none (fonctionnel.md §1.1). A draft
    // whose document could not be read carries -1 and says nothing.
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

    // Refresh when the tab is entered and when the library changes: a draft
    // belongs to the library it names two players of, so the list of another
    // file must never survive an open.
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
            const state = await CreateTranscription({
                match_length: length,
                jacoby: length === 0 ? formJacoby : false,
                beaver: length === 0 ? formBeaver : false
            });
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

    // Back to the list. The draft stays in the library and stays open on the Go
    // side — closing it for good (deleting the row) is its own gesture, with
    // its confirmation, and it is handleClose below.
    function backToList() {
        clearTranscription();
        refresh();
    }

    // ── enregistrer, exporter, fermer (T1.9) ─────────────────────────────
    //
    // Trois boutons, trois appels au service. `busy` sert de verrou : un
    // enregistrement en cours ne doit pas être relancé par un second Ctrl+Entrée
    // — le remplacement réécrit le match entier.

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

    // « enregistré il y a 3 min » vieillit tout seul : sans cette horloge la
    // phrase resterait celle de l'enregistrement.
    let nowTick = $state(Date.now());
    $effect(() => {
        if (!draft) return;
        const handle = setInterval(() => (nowTick = Date.now()), 30000);
        return () => clearInterval(handle);
    });

    let saveState = $derived(draftSaveState(draft, $transcriptionSaveStore, nowTick));

    // ADR-0048 décision 12 : le bouton NOMME le Match. « Enregistrer » portait
    // deux concepts en un mot — le brouillon est écrit après chaque Action, le
    // Match est matérialisé par ce bouton — et l'interface montrait l'alarmant
    // (« jamais enregistré »), ce qui fait matérialiser le Match et lancer un lot
    // d'analyse 2-ply par prudence, contre un risque inexistant. L'info-bulle
    // disait déjà juste, à l'endroit que personne ne lit.
    let savedMatchId = $derived(annotated?.document?.match_id || draft?.match_id || 0);
    let saveLabel = $derived(savedMatchId ? $t('transcription.updateMatch', { id: savedMatchId }) : $t('transcription.createMatch'));

    // ── the gestures ─────────────────────────────────────────────────────
    //
    // Every command the keyboard machine emits becomes one gesture, and the
    // gestures of one keystroke are sent IN ORDER: entering the second die and
    // validating the opening are two round trips that must not interleave with
    // the next keystroke's. One promise chain, therefore, and never a bare
    // `await` per handler.

    let pending = Promise.resolve();

    // The candidates of the roll being entered, RANKED by the 0-ply evaluation.
    // `gen` is the same play's index in LegalMoves' own order, which is what
    // `select_candidate` addresses: the engine ranks, the generator numbers, and
    // the two orders are joined here rather than in Go — LegalMoves deduplicates
    // by resulting board, so one notation is one play and the join is exact.
    let ranked = $state([]);
    // The evaluation could not rank this roll (a score beyond the MET's horizon,
    // a build without weights). The plays are still listed, in the generator's
    // order, because a transcription that cannot be typed is worse than one typed
    // without a ranking.
    let unranked = $state(false);
    // The last roll allowed no play at all: the `dance` Action was recorded on
    // its own, without a keystroke.
    let danced = $state(false);

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
            // Le coup posé par ses PAS (T2.3, T2.4). `BoardAfter` n'est envoyé
            // que par les deux chemins qui peuvent produire un coup illégal —
            // le déplacement libre et la notation tapée — et le moteur le jette
            // de lui-même dès qu'un coup légal atteint ce plateau
            // (transcript.validate) : rien ne se marque illégal par accident.
            case COMMAND.ENTER_PLAY:
                return {
                    Kind: 'enter_play',
                    Steps: (command.steps ?? []).map((step) => ({ from: step.from, to: step.to, hit: false })),
                    BoardAfter: command.board ?? null
                };
            // Le camp d'une action de videau n'est PAS posé ici : sans `HasSide`
            // le moteur prend celui qu'il attend — le camp au trait pour un
            // double, le camp d'en face pour une réponse (transcript.cubeGesture).
            case COMMAND.DOUBLE:
                return { Kind: 'double' };
            case COMMAND.TAKE:
                return { Kind: 'take' };
            case COMMAND.PASS:
                return { Kind: 'pass' };
            // Le camp qui abandonne est celui au trait, « sauf indication
            // contraire » : sans `HasSide` le moteur prend celui qu'il attend,
            // et l'indication contraire est le geste « changer de camp » sur
            // l'Action une fois créée (fonctionnel.md §1.2, §2).
            case COMMAND.RESIGN:
                return { Kind: 'resign', Level: command.value };
            case COMMAND.CURSOR_BACK:
                return { Kind: 'cursor_back' };
            case COMMAND.CURSOR_FORWARD:
                return { Kind: 'cursor_forward' };
            // Les quatre gestes de correction. Aucun ne porte de camp : celui
            // d'une insertion est PROPOSÉ par le moteur (celui qui rend la suite
            // cohérente), et celui d'une Action existante lui appartient — seul
            // `flip_side` le change (ADR-0045 §4).
            case COMMAND.INSERT_BEFORE:
                return { Kind: 'insert_before' };
            case COMMAND.INSERT_AFTER:
                return { Kind: 'insert_after' };
            case COMMAND.DELETE:
                return { Kind: 'delete' };
            case COMMAND.FLIP_SIDE:
                return { Kind: 'flip_side' };
            // La pile vit dans le transcript.Editor de la session, en mémoire :
            // ces deux-là ne passent pas par transcript.Apply, qui est pure.
            case COMMAND.UNDO:
                return { Kind: 'undo' };
            case COMMAND.REDO:
                return { Kind: 'redo' };
            case COMMAND.SELECT: {
                // Le rang est celui de la liste MONTRÉE : filtrée, c'est elle
                // que `j`/`k` et le clic parcourent (T2.2).
                const entry = visible[command.index];
                return entry ? { Kind: 'select_candidate', Candidate: entry.gen } : null;
            }
            default:
                return null;
        }
    }

    // The commands are turned into gestures HERE, synchronously, and not inside
    // the chain below: a `select` names a rank in the list as it stands at the
    // keystroke, and the list is replaced as soon as the next roll comes in.
    function run(commands) {
        return queue(commands.map(gestureOf).filter(Boolean));
    }

    /** La file elle-même : les gestes partent dans l'ordre, un par aller-retour. */
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
     * Un geste qui ne vient pas du clavier : le volet des métadonnées (T3.1).
     *
     * Il passe par la MÊME file que les touches, et pour la même raison : un
     * nom validé pendant qu'un jet part encore ne doit pas doubler le tour
     * d'aller-retour de ce jet.
     */
    function sendGesture(gesture) {
        return gesture ? queue([gesture]) : pending;
    }

    // ── the candidates ───────────────────────────────────────────────────
    //
    // Two calls, both of them the ones the Eval panel already makes: LegalMoves
    // for the plays and their steps, EvaluatePositionImmediate for the 0-ply
    // ranking (candidates = 0, so ALL of them come back — not the ten a stored
    // analysis keeps). Never StartEvaluationAtRest: a transcription must answer
    // between two keystrokes, and the display-depth tier costs a fifth of a
    // second.

    let candidateGeneration = 0;

    /**
     * The Position the roll being entered is played from, dice and side set.
     *
     * A correction in place, and an insertion, are played from the Action they
     * land ON — not from the position the match has reached. Reading it off
     * `next.position` listed the candidates of the END of the document while
     * the user was correcting the middle of it. `annotated.entry` is the engine's
     * own account of the Action being typed (transcript.EntryInfo): where it
     * lands, for which camp, and whether it replaces.
     */
    function entryPosition() {
        const ann = get(transcriptionStore)?.annotated;
        if (!ann) return null;
        const entry = ann.entry;
        const at = entry?.at ?? ann.cursor ?? 0;
        const info = (ann.actions ?? [])[at];
        const base = info?.has_position ? info.before : ann.next?.position;
        const side = entry ? entry.side : ann.next?.side;
        if (!base) return null;
        return { ...structuredClone(base), id: 0, dice: [0, 0], player_on_roll: side, decision_type: 0 };
    }

    function rollPosition() {
        const state = get(transcriptionKeyStore);
        const pos = entryPosition();
        if (!pos || !state.dice[0] || !state.dice[1]) return null;
        return { ...pos, dice: [state.dice[0], state.dice[1]] };
    }

    /** The plays of the roll, best first, each with its index in LegalMoves. */
    async function computeCandidates(pos) {
        const plays = (await LegalMoves(pos)) ?? [];
        if (!plays.length) return { list: [], unranked: false };

        // A plain lookup, not a Map: it is built and thrown away inside this
        // call, and nothing reactive ever reads it.
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
                // Les pas du coup, que le filtre par point de départ lit (T2.2).
                .map((row) => ({ ...row, steps: plays[row.gen]?.steps ?? [] }));
            if (list.length) return { list, unranked: false };
        } catch (err) {
            logger.error('The 0-ply ranking of a transcription roll failed:', err);
        }
        // The generator's own order, which is an order and not a ranking.
        return { list: plays.map((play, index) => ({ move: { index, move: play.notation }, gen: index, steps: play.steps ?? [] })), unranked: true };
    }

    /**
     * Answers the roll the machine is waiting on: no legal play is a dance, one
     * or more preselects the first. Superseded by a newer roll rather than
     * cancelled — a stale answer must never land on the roll that replaced it.
     */
    async function settleCandidates() {
        if (!get(transcriptionKeyStore).awaitingCandidates) return;
        // Un NOUVEAU jet : le filtre par point appartenait au précédent. Il est
        // levé ici et non à chaque frappe — `j`/`k` doivent parcourir la liste
        // réduite, c'est de là que vient le gain d'ux.md §4.1.
        resetTranscriptionPointFilter();
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

    // ── the Cursor ───────────────────────────────────────────────────────
    //
    // Moving it is the engine's business (`cursor_back`/`cursor_forward` reload
    // the Action's dice and play into the Entry); what the panel owes is the
    // rest of the screen — the board, which follows annotated.cursor on its
    // own, and the candidate list, which has to be asked for again because it
    // is an Evaluation and is never stored (ADR-0045 rule 8).

    /**
     * Re-arms the panel on the Action the Cursor landed on: its candidates,
     * with the play that was RECORDED selected. A cell that is not a checker
     * play — an opening, a cube action, the end of the document — leaves the
     * keyboard machine at rest.
     */
    async function settleCursor() {
        const ann = get(transcriptionStore)?.annotated;
        if (!ann) return;
        const info = (ann.actions ?? [])[ann.cursor ?? 0];
        resetTranscriptionPointFilter();
        ranked = [];
        unranked = false;
        danced = false;

        // Une INSERTION attend une Action neuve à cet endroit : les dés et le
        // coup de l'Action qui s'y trouve sont ceux du voisin qu'elle repousse,
        // et les charger ferait taper par-dessus lui. Le moteur dit lequel des
        // deux c'est (`entry.replacing`), le panneau ne le devine pas.
        const inserting = ann.entry != null && ann.entry.replacing === false;
        // `unrecorded` est là exprès : le jet est connu, le coup ne l'est pas, et
        // poser le curseur dessus est justement le moment où l'utilisateur peut le
        // renseigner. Les candidats de ce jet sont donc listés comme pour tout
        // autre coup ; aucun n'est présélectionné puisque rien n'a été consigné.
        const playable = info?.kind === 'checker' || info?.kind === 'dance' || info?.kind === 'unrecorded';
        if (inserting || !info?.has_position || !playable) {
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

        // The play as it was written down, found by its notation: the ranking
        // is a list of the SAME plays, so the recorded one is in it — unless
        // the play is illegal, and then the first candidate stands in.
        const played = ranked.findIndex((row) => row.move.move === info.notation);
        transcriptionKeyStore.set({
            ...initialKeyState(),
            phase: PHASE.ROLL,
            dice: [info.before.dice?.[0] ?? 0, info.before.dice?.[1] ?? 0],
            candidateCount: ranked.length,
            selected: played >= 0 ? played : 0
        });
    }

    /** A click on a cell of the Transcript: the Cursor walks to that Action. */
    function selectAction(index) {
        const ann = get(transcriptionStore)?.annotated;
        if (!ann) return;
        const from = ann.cursor ?? 0;
        if (index === from) return;
        run(cursorCommands(from, index)).then(settleCursor);
        panelEl?.focus({ preventScroll: true });
    }

    // ── le triangle des jets (T2.1) ──────────────────────────────────────
    //
    // Le clic passe par la MÊME machine que les touches — `enterDicePair`
    // applique les deux dés l'un après l'autre, comme deux frappes — et le
    // panneau en fait exactement ce qu'il fait d'une frappe : la liste du jet
    // précédent est jetée avant que les gestes ne partent, puis les candidats
    // du nouveau jet sont demandés. Le focus revient au panneau : la souris
    // donne le jet, le clavier finit le tour, et rien ne se perd entre les deux.

    function applyMouseDice(next) {
        resetTranscriptionPointFilter();
        ranked = [];
        unranked = false;
        danced = false;
        transcriptionKeyStore.set(next.state);
        run(next.commands).then(settleCandidates);
        panelEl?.focus({ preventScroll: true });
    }

    /** Une case du triangle : le jet entier, dé fort d'abord. */
    function pickDice(high, low) {
        if (!draft) return;
        // Un coup est joué au plateau et plusieurs jets le produisent (T2.3) :
        // la case ne SAISIT alors pas un jet, elle dit lequel des jets encore
        // possibles a été lancé, et l'Action part avec les pas déjà joués.
        const play = get(quizPlayStore);
        if (play && !play.free && play.steps.length) {
            sendPlay([high, low], play.steps, null);
            panelEl?.focus({ preventScroll: true });
            return;
        }
        applyMouseDice(enterDicePair(get(transcriptionKeyStore), high, low, keyContext));
    }

    /** Une case de la rangée des six : le dé d'un camp, à l'ouverture. */
    function pickDie(die) {
        if (!draft) return;
        applyMouseDice(enterSingleDie(get(transcriptionKeyStore), die, keyContext));
    }

    function chooseCandidate(index) {
        const next = selectCandidate(get(transcriptionKeyStore), index);
        transcriptionKeyStore.set(next.state);
        run(next.commands);
        panelEl?.focus({ preventScroll: true });
    }

    // ── R3 : le chemin souris complet (ADR-0048 décision 11) ─────────────
    //
    // Le clavier reste la voie par défaut — les dés y sont mesurés deux fois
    // plus rapides — mais chaque geste doit être ATTEIGNABLE à la souris. Il
    // manquait : parcourir la liste par pas (il fallait cliquer une ligne
    // précise, donc l'avoir lue), valider le dernier coup d'une partie, annuler,
    // et effacer le jet en cours.

    /**
     * Un cran de molette = un pas dans la liste, l'exact équivalent de `j`/`k`.
     *
     * Posé sur la liste ET sur le plateau (App.svelte laisse passer la molette
     * en mode TRANSCRIBE au lieu de faire naviguer les positions) : l'œil reste
     * sur le plateau, les flèches du candidat défilent, et l'on reconnaît le
     * coup vu sur la vidéo par son IMAGE au lieu de traduire « 13/10 13/11 » de
     * tête. La liste suit la sélection ; elle ne défile pas d'elle-même.
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

    // Le cran donné au-dessus du PLATEAU, que le dispatcher de App.svelte pose
    // dans un magasin — le plateau ne connaît pas la liste, et le panneau ne
    // reçoit pas ses événements. Même chemin que `Ctrl+Z` et que le videau
    // cliqué, pour la même raison.
    $effect(() => {
        const wanted = $transcriptionWheelStore;
        if (!wanted) return;
        transcriptionWheelStore.set(null);
        if (!draft || $activeTabStore !== 'transcription') return;
        stepCandidate(wanted.delta);
    });

    function wheelCandidates(event) {
        if (!draft || !visible.length) return;
        const delta = event.deltaY > 0 ? 1 : event.deltaY < 0 ? -1 : 0;
        if (delta === 0) return;
        event.preventDefault();
        stepCandidate(delta);
    }

    /**
     * Double-clic : valider. Le simple clic sélectionne — les flèches du plateau
     * suivent, et c'est là que l'on reconnaît le coup. C'est le seul chemin
     * souris qui couvre le dernier coup d'une partie, qui n'a pas de jet suivant
     * pour porter sa validation ; au clavier, c'est `Entrée`.
     */
    function commitCandidate(index) {
        if (!draft) return;
        const next = selectCandidate(get(transcriptionKeyStore), index);
        transcriptionKeyStore.set(initialKeyState());
        ranked = [];
        unranked = false;
        danced = false;
        resetTranscriptionPointFilter();
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

    // ── le videau à la souris (T2.5) ─────────────────────────────────────
    //
    // Deux cibles pour les mêmes quatre gestes : la rangée [D] [T] [P] [R] du
    // panneau, et le videau DESSINÉ sur le plateau, qui propose un double.
    // Aucune des deux n'écrit de règle : elles passent par `cubeGesture` et
    // `beginResign`, qui sont le corps des touches `d`/`t`/`p`/`r`, puis par
    // `applyResult`, qui est la suite d'une frappe. Le budget d'ux.md §4.2 —
    // double puis prise à la souris, 3,4 s — se lit donc en clics : deux, un
    // par Action, et rien entre les deux.

    /** Un des trois gestes de videau : double, prise, passe. */
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

    /** Le niveau donné au clic — le second des deux gestes d'une résignation. */
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

    // Le videau cliqué sur le plateau. Le plateau POSE la demande et ne juge
    // rien (utils/boardInteractions.js) ; c'est ici qu'on sait ce que le
    // document attend, et une demande qui tombe devant une offre — où la
    // réponse appartient au camp d'en face, pas au videau — est simplement
    // jetée. Même chemin que `Ctrl+Z`, pour la même raison.
    $effect(() => {
        const wanted = $transcriptionCubeRequestStore;
        if (!wanted) return;
        transcriptionCubeRequestStore.set(null);
        if (!draft || !canCubeAct) return;
        sendCube(COMMAND.DOUBLE);
    });

    // ── le menu contextuel du Transcript (T2.5) ──────────────────────────
    //
    // Clic droit sur une cellule : insérer avant, insérer après, supprimer,
    // changer de camp. Ce sont `i`, `a`, `x` et `s`, et ils agissent sur
    // l'Action AU CURSOR — le menu commence donc par y mener le Cursor
    // (`menuCommands`), ce qui est le `h`×k de la relecture au clavier, en un
    // clic. Le clic droit place aussi le Cursor sans rien corriger : c'est la
    // correction en place, qui n'a pas d'entrée au menu parce qu'elle n'est
    // pas un geste — c'est retaper.

    let transcriptMenu = $state(null);

    function openTranscriptMenu(index, at) {
        if (!draft) return;
        transcriptMenu = { index, x: at.x, y: at.y };
    }

    /** Une entrée du menu : le Cursor mené à la cellule, puis la correction. */
    function menuCommand(index, kind) {
        if (!draft) return;
        const ann = get(transcriptionStore)?.annotated;
        if (!ann) return;
        resetTranscriptionKeys();
        ranked = [];
        unranked = false;
        danced = false;
        run(menuCommands(ann.cursor ?? 0, index, kind)).then(settleCursor);
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

    // ── the keyboard ─────────────────────────────────────────────────────
    //
    // The panel handles its own keys while it has focus (ux.md §3): the digits
    // are the dice and they mean something else elsewhere, so they cannot go
    // through the global dispatcher. panelKeyGuard states, once for every docked
    // panel, what a panel may never swallow — Ctrl combos, Space, '?', and
    // anything typed in a field.

    function handleKeyDown(event) {
        if (!draft) return handleListKeyDown(event);
        // Ctrl+Entrée enregistre. Ctrl+S ne peut pas : le dispatcher global le
        // tient pour « sauver la position » (keyboardService.js), et
        // isAlwaysGlobal renvoie vrai pour tout combo Ctrl — ce qui veut aussi
        // dire que panelKeyGuard laisserait passer celui-ci. Il est donc pris
        // AVANT la garde, et arrêté net pour que le dispatcher ne le revoie pas.
        if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
            if (!panelEl?.contains(document.activeElement)) return;
            event.preventDefault();
            event.stopPropagation();
            handleSave();
            return;
        }
        if (panelKeyGuard(event)) return;
        if (!panelEl?.contains(document.activeElement)) return;

        // Un coup se joue au plateau : Retour arrière défait le dernier pas
        // plutôt que d'effacer des dés que personne n'a tapés (T2.3). C'est la
        // correction d'un clic manqué, et elle ne coûte pas le coup entier.
        if (event.key === 'Backspace' && ($quizPlayStore?.steps?.length ?? 0) > 0) {
            event.preventDefault();
            event.stopPropagation();
            undoBoardPlayStep();
            return;
        }

        const result = pressKey(keys, event, keyContext);
        // Retour arrière sur un jet vide est PRIS (le plateau appartient au
        // brouillon) mais ne fait rien : il répond, comme toute la famille.
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

    // ── la porte d'entrée, au clavier (ADR-0048 décision 10) ─────────────
    //
    // `handleKeyDown` sortait sur `if (!draft) return;` : la liste ne traitait
    // AUCUNE touche, alors que le docstring du panneau pose l'objectif inverse —
    // « a match is typed without ever reaching for the mouse, and a first
    // keystroke that landed nowhere would break the count before it starts ».
    // Reprendre le brouillon de la veille, geste de chaque session après la
    // première, n'avait aucun chemin clavier du tout.
    //
    // `ListTranscriptions` rend les brouillons du plus récemment modifié au plus
    // ancien : la première ligne surlignée est donc celui en cours, et
    // `Ctrl+Maj+T` `Entrée` le rouvre en deux touches.

    let listIndex = $state(0);

    // La liste change (autre bibliothèque, brouillon fermé) : le surlignage
    // retombe sur la ligne du haut plutôt que sur un rang qui n'existe plus.
    $effect(() => {
        const rows = $transcriptionListStore;
        if (listIndex >= rows.length) listIndex = 0;
    });

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
     * La suite d'un geste, quelle que soit la main qui l'a donné : une touche,
     * une case du triangle, un bouton de la rangée du videau, le videau du
     * plateau.
     *
     * Elle est écrite UNE fois, ici, parce que le second déclencheur de T2.5
     * n'est qu'un déclencheur : la machine (services/transcriptionKeys.js) rend
     * le même couple `{state, commands}` d'où qu'il vienne, et tout ce qui suit
     * — la liste jetée, l'ordre des allers-retours, le réarmement du Cursor —
     * doit être le même, sans quoi un bouton et sa touche finiraient par
     * diverger sur un état que personne ne teste.
     *
     * @param {{state: object, commands: {kind: string}[]}} result
     */
    function applyResult(result) {
        // A roll that is starting again, or one that has just been validated,
        // leaves a list that belongs to nobody: it goes before the gestures do,
        // so no `select` can address it any more.
        if (result.state.phase === PHASE.DICE || result.state.phase === PHASE.DIE1) {
            resetTranscriptionPointFilter();
            ranked = [];
            unranked = false;
        }
        danced = false;

        transcriptionKeyStore.set(result.state);
        // A gesture that MOVES or EDITS lands the Cursor on an Action, and the
        // panel has to be re-armed on it: its candidates are an Evaluation and
        // are never stored, so they are asked for again (ADR-0045 rule 8). A
        // gesture that fills a die waits for the roll instead.
        // Une touche de correction est UNE commande, et elle passe par le même
        // chemin que le bouton — d'où son refus silencieux là où il n'y a rien
        // à corriger.
        if (result.commands.length === 1 && EDITS.has(result.commands[0].kind)) {
            runCommand(result.commands[0].kind);
            return;
        }

        const rearms = result.commands.some((c) => REARMING.has(c.kind));
        run(result.commands).then(rearms ? settleCursor : settleCandidates);
    }

    // The commands that CHANGE the document without typing anything into it, and
    // which a button offers as well as a key — so both go through runCommand.
    const EDITS = new Set([COMMAND.INSERT_BEFORE, COMMAND.INSERT_AFTER, COMMAND.DELETE, COMMAND.FLIP_SIDE, COMMAND.UNDO, COMMAND.REDO]);
    // The commands after which the panel re-reads the Cursor rather than the
    // roll being typed. Walking is one of them, and so is every edit: each
    // leaves the Cursor somewhere the ENGINE chose — on the first Inconsistency
    // when the gesture made one — and the candidates there have to be asked for
    // again, since an Evaluation is never stored (ADR-0045 rule 8).
    const REARMING = new Set([COMMAND.CURSOR_BACK, COMMAND.CURSOR_FORWARD, ...EDITS]);

    /**
     * Runs one command from a key, a button or the global dispatcher.
     *
     * Deleting and changing a camp need an Action to act on, and at the end of
     * the document — where the Cursor spends most of its time — there is none.
     * The keystroke then does NOTHING, like `Ctrl+Z` on an empty stack: sending
     * the gesture anyway would answer the most ordinary `x` in the world with
     * the engine's English "the cursor is not on an action".
     */
    function runCommand(kind) {
        if (!draft) return;
        // ADR-0048 décision 9 : un geste sans effet RÉPOND, une fois, dans la
        // barre d'état. Il se taisait, et le seul de la famille qui disait
        // quelque chose était un bouton grisé — celui que la décision 3 retire.
        if ((kind === COMMAND.DELETE || kind === COMMAND.FLIP_SIDE) && !onAction) {
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

    // `Ctrl+Z` / `Ctrl+Maj+Z` reach the panel through a store, because a Ctrl
    // combo is always global (keyboardService.js's isAlwaysGlobal) and the
    // dispatcher is where it is bound. Same shape as the Anki review keys.
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
        // Le plateau lit ces deux magasins pour décider si un clic le concerne :
        // un panneau démonté ne doit plus rien filtrer.
        transcriptionCandidateStepsStore.set([]);
        resetTranscriptionPointFilter();
        // Le coup joué au plateau appartient au panneau : démonté, il ne doit
        // plus se dessiner ni prendre les clics du damier.
        quizPlayStore.set(null);
        boardPlayArmed = false;
        // Un clic sur le videau resté sans réponse ne doit pas être servi au
        // remontage du panneau (T2.5).
        transcriptionCubeRequestStore.set(null);
    });

    // The panel takes the keyboard as soon as the TAB is entered — draft open or
    // draft list — because the whole point of the design is that a match is
    // typed without ever reaching for the mouse, and a first keystroke that
    // landed nowhere would break the count before it starts. Until ADR-0048 the
    // list was exactly such a keystroke (decision 10).
    $effect(() => {
        void draft;
        if ($activeTabStore === 'transcription') panelEl?.focus({ preventScroll: true });
    });

    // ── ce que les DEUX BARRES disent (ADR-0048 décision 2) ──────────────
    //
    // Le panneau POSE ces deux faits et ne les dessine plus. `ux.md` §5 les
    // plaçait déjà dehors — l'état dans la barre de match, au-dessus du plateau
    // donc lu au moment où l'œil y est ; l'Action attendue dans la barre d'état
    // — et ce câblage manquait. Les sept pastilles qui les remplaçaient
    // coûtaient 66 des 354 px qui repoussaient la liste des candidats hors de
    // l'écran.

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
     * L'Action attendue en un mot. Un ÉTAT : il y en a toujours exactement un
     * tant qu'un brouillon est ouvert.
     *
     * Les six lignes d'instruction permanentes qui l'accompagnaient sous les dés
     * ne sont plus nulle part dans le panneau (décision 8) : une instruction
     * permanente est l'aveu qu'un geste ne se devine pas, et sa place est
     * `raccourcis.rst` et l'aide qui en est engendrée, pas 13 px lus 250 fois.
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

    // The Position the board shows: the one the Cursor's Action was played
    // from, and the one the match has reached when the Cursor sits at the end.
    // An opening produces no Position (fonctionnel.md §1.2), hence has_position.
    //
    // The dice of the roll being entered go on it as they come in — an opening's
    // two dice excepted, which belong to two different players and are shown in
    // the panel instead.
    function boardPosition(ann, dice, opening) {
        const actions = ann.actions ?? [];
        const at = ann.cursor ?? 0;
        const current = at >= 0 && at < actions.length ? actions[at] : null;
        const base = current?.has_position ? current.before : ann.next?.position;
        if (!base) return null;
        const rolled = (!opening || current) && dice[0] > 0 && dice[1] > 0 ? [dice[0], dice[1]] : [0, 0];
        return { ...structuredClone(base), id: 0, dice: rolled };
    }

    $effect(() => {
        if (!annotated || $statusBarModeStore !== 'TRANSCRIBE') return;
        const pos = boardPosition(annotated, keys.dice, expects === 'opening');
        if (pos) positionStore.set(pos);
    });

    // The arrows of the selected candidate, through the store Board.svelte
    // already reads. It is a display of the Evaluation and nothing else: no
    // candidate is ever written to the library.
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

    // The backends store the timestamps as text ("2026-09-07 01:23:45"); the
    // seconds say nothing here.
    function shortStamp(stamp) {
        return stamp && stamp.length >= 16 ? stamp.slice(0, 16) : (stamp ?? '');
    }

    function playersOf(row) {
        if (row.label) return row.label;
        const pair = [row.player1, row.player2].filter(Boolean);
        return pair.length ? pair.join(' — ') : $t('transcription.unnamed');
    }

    // A length of 0 is a money session, and -1 is the sentinel summarize()
    // leaves on a draft whose document could not be read — the row still shows,
    // because a draft the panel cannot list is a draft nobody can delete.
    function lengthOf(row) {
        if (row.match_length < 0) return '—';
        return row.match_length === 0 ? $t('transcription.money') : $t('transcription.points', { n: row.match_length });
    }

    let matchLength = $derived(annotated?.document?.header?.match_length ?? 0);
    let score = $derived(annotated?.score ?? [0, 0]);
    let sideOnRoll = $derived(annotated?.next?.side ?? 0);

    function playerName(side) {
        const header = annotated?.document?.header ?? {};
        const named = side === 0 ? header.player1 : header.player2;
        return named || (side === 0 ? $t('transcription.player1') : $t('transcription.player2'));
    }

    // ── the cube, the end of a game and the end of the match ─────────────
    //
    // All of it is READ off the Replay: the panel derives no score, no Crawford
    // and no cube of its own (ADR-0045 rule 9).

    // `value` is the log2 exponent everywhere in the code base (XGID contract),
    // and `owner` is -1 — domain.None — while the cube sits in the middle.
    let cube = $derived(annotated?.next?.position?.cube ?? { owner: -1, value: 0 });
    // While a double waits for its answer, next.position carries the cube AT THE
    // LEVEL OFFERED, which is the one the answerer weighs (fonctionnel.md §1.2).
    let awaitingAnswer = $derived(expects === 'take');
    // La CLÉ et ses paramètres, non la phrase : c'est la barre de match qui la
    // traduit, et deux traductions du même fait finiraient par diverger.
    let cubeLabelKey = $derived.by(() => {
        const v = 1 << (cube.value ?? 0);
        if (awaitingAnswer) return { key: 'transcription.doubleOffered', params: { v } };
        if (cube.owner !== 0 && cube.owner !== 1) return { key: 'transcription.cubeCentred', params: { v } };
        return { key: 'transcription.cubeOwned', params: { v, player: playerName(cube.owner) } };
    });

    let matchOver = $derived(annotated?.finished === true);
    let matchWinner = $derived(annotated?.winner ?? -1);
    let gameNumber = $derived(annotated?.next?.game_number ?? 1);

    // The Inconsistencies of the Action just recorded — a double by a side that
    // does not hold the cube, an answer with no offer, an Action past the end of
    // the match. They are SHOWN, never a refusal (ADR-0044): the sentence comes
    // from the kind, so the engine's English detail never reaches the screen.
    let lastFlags = $derived.by(() => {
        const actions = annotated?.actions ?? [];
        const last = actions[actions.length - 1];
        return (last?.inconsistencies ?? []).map((i) => $t(`transcription.inconsistency.${i.kind}`));
    });

    // ── la correction ────────────────────────────────────────────────────
    //
    // Tout ce que ces boutons font, une touche le fait (ux.md §3) ; ils sont là
    // pour que les gestes soient DÉCOUVRABLES, et parce que la souris est un
    // chemin de plein droit — un clic sur une cellule du Transcript remplace
    // déjà `h`×k (ux.md §4.3).

    let history = $derived($transcriptionHistoryStore);
    // Le Cursor est-il sur une Action ? Insérer, supprimer et changer de camp
    // n'ont de sens que là ; en bout de document il n'y a rien à corriger.
    let onAction = $derived((annotated?.actions?.length ?? 0) > 0 && (annotated?.cursor ?? 0) < (annotated?.actions?.length ?? 0));
    // Le moteur a présélectionné un candidat parce que le coup enregistré n'est
    // pas un coup du nouveau jet : marqué « à revoir » jusqu'à validation
    // (fonctionnel.md §2). C'est le moteur qui le dit, jamais le panneau.
    let underReview = $derived(annotated?.entry?.review === true);
    let correcting = $derived(annotated?.entry?.replacing === true);

    // The two dice as they come in, "·" for a die not entered yet. An opening's
    // two dice belong to two different players, which is why they are shown
    // here and not drawn on the board.
    let dieCells = $derived([keys.dice[0] || '·', keys.dice[1] || '·']);

    // Le triangle n'est là que lorsqu'un jet est attendu : devant une réponse au
    // videau, pendant une résignation ou une fois le match fini, il n'y a pas de
    // dé à donner et une cible qui ne répond à rien vaut moins que pas de cible.
    let diceEntryOpen = $derived(!!draft && !matchOver && !awaitingAnswer && keys.phase !== PHASE.RESIGN && DICE_KINDS.has(expects));

    // La rangée [D] [T] [P] [R] (T2.5). Elle dit de qui est le tour : le camp
    // au trait annonce (double, résignation), ou le camp d'en face répond
    // (prise, passe) — jamais les quatre à la fois. Un bouton éteint n'est pas
    // un refus : le clavier prend `d` partout, et l'Incohérence qui en sort est
    // marquée (ADR-0044) ; la rangée, elle, est une cible, et une cible qui ne
    // répondrait à rien vaut moins que pas de cible — la même règle que celle
    // qui retire le triangle des jets devant une réponse au videau.
    //
    // Match fini : la rangée disparaît avec le triangle. Il n'y a plus de
    // partie à doubler ni à abandonner, et le clavier reste là pour qui veut
    // transcrire une Action au-delà de la fin.
    let resigning = $derived(keys.phase === PHASE.RESIGN);
    let cubeRowOpen = $derived(!!draft && !matchOver);
    let canCubeAct = $derived(cubeRowOpen && !awaitingAnswer && !resigning);
    let canCubeAnswer = $derived(cubeRowOpen && awaitingAnswer);

    // ── le filtre par point de départ (T2.2) ─────────────────────────────
    //
    // Le plateau POSE le filtre (utils/boardInteractions.js), le panneau montre
    // la liste réduite : c'est un état d'affichage, il ne crée aucune Action et
    // ne rappelle pas le moteur. `ranked` reste le classement complet — c'est
    // lui qu'on retrouve quand le filtre tombe — et `visible` est ce qui est
    // montré, donc ce que tout rang désigne : `j`/`k`, le clic sur une ligne,
    // les flèches du plateau.
    let visible = $derived(filterByPoints(ranked, $transcriptionPointFilterStore));

    // Ce que le plateau a besoin de savoir des candidats, et rien de plus : les
    // pas, pour reconnaître un point de départ.
    $effect(() => {
        transcriptionCandidateStepsStore.set(ranked.map((row) => ({ steps: row.steps ?? [] })));
    });

    // Le filtre a changé : la liste montrée n'a plus la même longueur, et le
    // premier de la liste réduite est présélectionné — sans quoi `j` partirait
    // d'un rang qui ne veut plus rien dire. Gardé par la clé du filtre : la
    // même liste ne doit pas se re-sélectionner à chaque changement de `ranked`.
    let lastFilterKey = '';
    $effect(() => {
        const points = $transcriptionPointFilterStore;
        const filterKey = points.join(',');
        if (filterKey === lastFilterKey) return;
        lastFilterKey = filterKey;

        const state = get(transcriptionKeyStore);
        if (state.phase !== PHASE.ROLL && state.phase !== PHASE.CANDIDATE) return;
        const list = filterByPoints(ranked, points);
        if (!list.length) return;
        transcriptionKeyStore.set({ ...state, candidateCount: list.length, selected: 0 });
        run([{ kind: COMMAND.SELECT, index: 0 }]);
    });

    let rankedMoves = $derived(visible.map((row) => row.move));
    // Which referential the equity column is stated in (ADR-0016 point 6,
    // ADR-0019): money points at money play, normalised match equity at a score.
    let isMoney = $derived(isMoneyPosition(annotated?.next?.position));

    // ── le coup joué au plateau (T2.3), le coup illégal (T2.4) ───────────
    //
    // Trois entrées pour un même endroit du document, et une seule règle pour
    // savoir laquelle est active : **ce que les dés valent**.
    //
    //   dés non saisis  → le plateau JOUE (T2.3). Le clic contraint le pion aux
    //                     coups légaux, et les dés se déduisent des pas.
    //   dés saisis      → la liste des candidats existe, et le clic sur un point
    //                     la FILTRE (T2.2, déjà en place).
    //   mode libre      → le clic déplace le pion sans rien vérifier (T2.4), et
    //                     le plateau obtenu devient `board_after`.
    //
    // Les deux premiers ne se chevauchent donc jamais, et cela ne coûte aucun
    // geste d'armement : le budget d'ux.md §4.1 (quatre pas à la souris ≤ 6 s)
    // ne paie que les pas. Taper un chiffre pendant un coup au plateau abandonne
    // le coup et rend la main à la saisie par les dés — c'est la sortie, et elle
    // est dite dans raccourcis.rst.
    //
    // Le coup lui-même n'est pas joué ici : `quizPlayStore` le porte, le
    // réducteur de `services/quizPlay.js` le fait avancer et `boardInteractions`
    // lui passe les clics. Ce qui est ici est l'armement (l'union des coups
    // légaux des 21 jets), la déduction lue sur l'état, et l'envoi de l'Action.

    let freeMode = $state(false);
    let notationText = $state('');
    // Le secours de saisie à la main est REPLIÉ par défaut (ADR-0048 décision 7).
    // `handEntryOpen` était vrai à chaque tour de pions — 250 fois par match pour
    // un usage attendu d'une fois — et ses 29 px de haut faisaient partie des
    // 49 px qui empêchaient la palette de tenir dans un dock de 280 px.
    let handOpen = $state(false);

    // Replier le secours rend le plateau à son mode contraint : laisser le
    // déplacement libre armé derrière un volet fermé serait un piège.
    $effect(() => {
        if (!handOpen && freeMode) freeMode = false;
    });
    // Verrou d'enregistrement : il désarme le plateau pendant les quatre gestes
    // qui partent au moteur, sinon chaque réponse ré-armerait le coup — 21
    // appels à LegalMoves par geste, et un plateau qui se rejoue sous la souris.
    let recordingPlay = $state(false);
    let boardPlayArmed = false;
    let boardPlayGeneration = 0;

    let boardPlayOpen = $derived(!!draft && !matchOver && !awaitingAnswer && !recordingPlay && !freeMode && keys.phase === PHASE.DICE && expects === 'checker');
    // Le coup joué au plateau reste possible volet fermé : c'est le PLATEAU qui
    // le porte, pas ce volet, qui n'offre que la bascule libre et la notation.

    /** L'union des coups légaux des 21 jets, demandée en une salve. */
    async function loadBoardPlay(pos, generation) {
        const answers = await Promise.all(ROLLS.map(([high, low]) => LegalMoves({ ...pos, dice: [high, low] }).catch(() => [])));
        if (generation !== boardPlayGeneration) return;
        const byRoll = ROLLS.map((dice, index) => ({ dice, plays: answers[index] ?? [] })).filter((entry) => entry.plays.length);
        quizPlayStore.set(newBoardPlay(pos, byRoll));
        boardPlayArmed = true;
    }

    // Un tour nouveau, un coup neuf. L'effet dépend d'`annotated` : chaque
    // Action enregistrée change la position de départ, donc l'union des coups.
    $effect(() => {
        const open = boardPlayOpen;
        const free = freeMode;
        void annotated;
        const generation = ++boardPlayGeneration;
        if (!open && !free) {
            if (boardPlayArmed) {
                quizPlayStore.set(null);
                boardPlayArmed = false;
            }
            return;
        }
        const pos = entryPosition();
        if (!pos) return;
        if (free) {
            quizPlayStore.set(newFreePlay(pos));
            boardPlayArmed = true;
            return;
        }
        loadBoardPlay(pos, generation);
    });

    /**
     * Le coup est achevé et un seul jet le produit : l'Action part sans qu'un
     * chiffre ait été tapé. C'est la promesse de T2.3, et elle ne se tient que
     * si personne ne devine — `deducedDice` rend `null` dès qu'il reste deux
     * jets, et le triangle prend alors le relais.
     */
    $effect(() => {
        const play = $quizPlayStore;
        if (!play || play.free || recordingPlay) return;
        const dice = deducedDice(play);
        if (dice) sendPlay(dice, play.steps, null);
    });

    /** Les jets que l'utilisateur peut encore désigner, ou `null` pour tous. */
    let rollsAllowed = $derived.by(() => {
        const play = $quizPlayStore;
        if (!play || play.free || play.steps.length === 0) return null;
        return new Set(choosableRolls(play));
    });

    // Ce que « n pas joués » et « plusieurs jets produisent ce coup » disaient
    // en prose est déjà DIT par les deux objets concernés (ADR-0048 décision 8) :
    // les pions déplacés sur le plateau, et le triangle, qui n'allume que les
    // jets encore possibles (`rollsAllowed`). Deux phrases pour ce qu'on voit.
    let freeSteps = $derived($quizPlayStore?.free ? $quizPlayStore.steps.length : 0);
    let diceEntered = $derived(keys.dice[0] > 0 && keys.dice[1] > 0);
    // Les deux chemins du coup illégal ne s'ouvrent que là où un coup de pions
    // s'écrit : jamais devant une réponse au videau, une résignation ou un
    // match fini, où il n'y aurait pas d'Action à porter le plateau.
    let handEntryOpen = $derived(!!draft && !matchOver && !awaitingAnswer && keys.phase !== PHASE.RESIGN && expects === 'checker');

    /**
     * L'Action : les deux dés, les pas, et le plateau quand il faut le dire.
     *
     * Les dés sont TOUJOURS renvoyés, même lorsqu'ils ont déjà été tapés : sur
     * une Entry pleine, `enter_die` recommence le jet, si bien que deux dés de
     * plus laissent exactement les deux mêmes (transcript.Apply). Un seul chemin
     * vaut mieux qu'une branche « les dés y sont-ils déjà ? ».
     */
    async function sendPlay(dice, steps, board) {
        if (recordingPlay || !draft) return;
        recordingPlay = true;
        quizPlayStore.set(null);
        boardPlayArmed = false;
        freeMode = false;
        notationText = '';
        ranked = [];
        unranked = false;
        danced = false;
        resetTranscriptionPointFilter();
        try {
            await run([{ kind: COMMAND.DIE, value: dice[0] }, { kind: COMMAND.DIE, value: dice[1] }, { kind: COMMAND.ENTER_PLAY, steps, board }, { kind: COMMAND.VALIDATE }]);
        } finally {
            recordingPlay = false;
        }
        resetTranscriptionKeys();
    }

    /**
     * « Ce plateau est le coup joué » (T2.4). Le plateau part avec les pas :
     * les deux disent la même chose tant que les pas suffisent à la décrire,
     * et le plateau tranche quand ils n'y suffisent pas — un pion posé sur un
     * point tenu par l'adversaire, par exemple, que le moteur laisserait
     * ailleurs s'il ne rejouait que les pas.
     */
    function validateFreePlay() {
        const play = get(quizPlayStore);
        if (!play?.free || !play.steps.length || !diceEntered) return;
        sendPlay([keys.dice[0], keys.dice[1]], play.steps, play.board);
        panelEl?.focus({ preventScroll: true });
    }

    /**
     * La notation tapée (T2.4) : `13/7 8/7*`, `bar/22`, `6/off`. Le parseur est
     * celui des flèches du plateau (`parseMoveNotation`), et rien n'est jugé —
     * un coup légal saisi par ce chemin reste un coup ordinaire, parce que le
     * moteur compare par PLATEAU RÉSULTANT et non par la provenance du geste.
     */
    function validateNotation() {
        const pos = entryPosition();
        if (!draft || !pos || !diceEntered) return;
        const steps = stepsFromNotation(notationText, pos.player_on_roll);
        if (!steps.length) return;
        sendPlay([keys.dice[0], keys.dice[1]], steps, boardAfterSteps(pos.board, steps, pos.player_on_roll));
        panelEl?.focus({ preventScroll: true });
    }

    /** Le mode libre, et le retour au plateau contraint. */
    function toggleFreeMode() {
        freeMode = !freeMode;
        panelEl?.focus({ preventScroll: true });
    }

    /** Retour arrière pendant un coup au plateau : le dernier pas est défait. */
    function undoBoardPlayStep() {
        quizPlayStore.update((play) => undoBoardStep(play));
    }

    // ── le texte .mat, en MODALE (ADR-0048 décision 6) ───────────────────
    //
    // Rendu par le MÊME moteur que les matchs de la bibliothèque
    // (TranscriptionMAT → transcript.MatchParts → ingest.RenderMAT) : la modale
    // montre le fichier qui serait écrit, non un second avis sur lui. Il n'est
    // demandé que pendant qu'elle est ouverte, et à chaque changement du
    // document : fermée, elle ne coûte aucun aller-retour par frappe.
    //
    // Pourquoi une modale et non le volet qu'elle remplace, sous le Transcript :
    // le `.mat` est de l'ASCII ALIGNÉ EN COLONNES — c'est sa seule raison d'être
    // regardé — et sa ligne la plus longue fait 62 caractères, ~409 px en
    // monospace 11 px, là où la colonne offrait 320. Un `.mat` désaligné n'est
    // pas une version dégradée du `.mat`, c'est autre chose. Le volet vivait de
    // surcroît dans `TranscriptView`, dont le docstring promet qu'il ne tient
    // rien : il y portait un presse-papiers, un minuteur et un état propres au
    // brouillon, ce qui l'empêchait d'être monté un jour sur un match stocké.
    let matOpen = $state(false);
    let matText = $state('');
    let matCopied = $state(false);
    let matCopyTimer = null;

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
            {#snippet cells(row)}
                <td class="stamp-cell">{shortStamp(row.updated_at || row.created_at)}</td>
                <td>{playersOf(row)}</td>
                <td class="narrow-col align-right">{lengthOf(row)}</td>
                <td class="narrow-col align-right">{row.action_count < 0 ? '—' : row.action_count}</td>
                <td class="narrow-col">{row.match_id ? `#${row.match_id}` : $t('transcription.noMatch')}</td>
            {/snippet}
        </PanelTable>
    {:else}
        <div class="draft">
            <!-- La barre du brouillon : les gestes qui le font SORTIR de lui-même,
                 et rien d'autre (ADR-0048 décisions 2, 3 et 11). Les sept pastilles
                 d'état sont parties dans la barre de match, les six boutons de
                 correction dans le clic droit du Transcript ; il reste ce qui n'a
                 pas d'autre domicile, plus l'état DU MATCH — le seul qui réponde à
                 une question qu'on se pose en regardant ce bouton — et les deux
                 flèches, qui sont le chemin souris vers l'annulation (R3). -->
            <div class="draft-bar">
                <button class="new-btn" onclick={backToList}>{$t('transcription.backToList')}</button>
                <span class="save-state">{$t(saveState.key, saveState.params)}</span>
                <span class="bar-gap"></span>
                <button class="icon-btn" onclick={() => runCommand(COMMAND.UNDO)} title={$t('transcription.undoTooltip')} aria-label={$t('transcription.undo')}>↶</button>
                <button class="icon-btn" onclick={() => runCommand(COMMAND.REDO)} title={$t('transcription.redoTooltip')} aria-label={$t('transcription.redo')}>↷</button>
                <button class="new-btn" onclick={() => (metaOpen = !metaOpen)} title={$t('transcription.metadataTooltip')}>{$t('transcription.metadata')}</button>
                <button class="new-btn" onclick={() => (matOpen = true)} title={$t('transcription.matModalTooltip')}>{$t('transcription.matModal')}</button>
                <button class="new-btn" onclick={handleExport} disabled={busy} title={$t('transcription.exportMatTooltip')}>{$t('transcription.exportMat')}</button>
                <button class="primary-btn" onclick={handleSave} disabled={busy} title={$t('transcription.saveTooltip')}>{saveLabel}</button>
                <button class="danger-btn" onclick={handleClose} disabled={busy} title={$t('transcription.closeDraftTooltip')}>{$t('transcription.closeDraft')}</button>
            </div>

            {#if metaOpen}
                <!-- L'en-tête du brouillon (T3.1) : les deux noms, l'événement,
                     le lieu, la ronde, la date, le transcripteur, le tournoi.
                     Rien n'y est exigé — un brouillon sans noms s'enregistre et
                     s'exporte, avec des en-têtes vides. -->
                <TranscriptionMetadata header={annotated?.document?.header ?? {}} apply={sendGesture} {busy} />
            {/if}

            <!-- Trois régions, et l'ordre du DOM est celui de la boîte ÉTROITE :
                 candidats, palette, Transcript. En boîte large la palette passe
                 première par `order` (ADR-0048 décision 5). L'invariant tient
                 dans les deux : rien ne s'intercale entre les cases du jet et la
                 première ligne de candidats. -->
            <div class="draft-body">
                <div class="candidates-col">
                    {#if unranked && visible.length}
                        <!-- R2 : ce message parle de LA LISTE — le classement 0-ply
                             n'a pas abouti, l'ordre est celui du générateur — donc
                             il habite son en-tête, et non la pile de prose sous les
                             dés d'où il vient. -->
                        <div class="list-head">
                            <span class="list-note">{$t('transcription.unranked')}</span>
                        </div>
                    {/if}
                    {#if $transcriptionPointFilterStore.length}
                        <!-- Le filtre est un état d'AFFICHAGE (T2.2), jamais écrit
                             dans le document. En PUCE et non en phrase : elle
                             annonçait le filtre sans dire comment en sortir. -->
                        <div class="list-head">
                            <button class="chip" onclick={resetTranscriptionPointFilter} title={$t('transcription.pointFilterClear')}>
                                {$transcriptionPointFilterStore.join(', ')} · {visible.length} ✕
                            </button>
                        </div>
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
                            <!-- La molette fait un pas dans la liste : l'exact
                                 équivalent de `j`/`k` (R3, décision 11). -->
                            <div class="candidates" data-testid="transcription-candidates" role="listbox" tabindex="-1" aria-label={$t('transcription.candidatesLabel')} onwheel={wheelCandidates}>
                                <CandidateMovesTable
                                    moves={rankedMoves}
                                    selectedMove={$selectedMoveStore}
                                    onRowClick={(move) => chooseCandidate(rankedMoves.indexOf(move))}
                                    onRowDblClick={(move) => commitCandidate(rankedMoves.indexOf(move))}
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
                    <!-- Les cinq réponses possibles à une seule question — « qu'a
                         fait le camp au trait ? » — sur une ligne : les deux dés,
                         et les quatre gestes de videau. Puis le triangle dessous,
                         sous les dés qu'il double (décision 7). -->
                    <div class="entry-row" data-testid="transcription-dice">
                        {#if !awaitingAnswer && keys.phase !== PHASE.RESIGN}
                            <!-- Cliquables : l'équivalent souris de Retour arrière (R3). -->
                            <button class="die" class:filled={keys.dice[0] > 0} onclick={clearDice} title={$t('transcription.clearDice')} aria-label={$t('transcription.clearDice')}
                                >{dieCells[0]}</button
                            >
                            <button class="die" class:filled={keys.dice[1] > 0} onclick={clearDice} title={$t('transcription.clearDice')} aria-label={$t('transcription.clearDice')}
                                >{dieCells[1]}</button
                            >
                        {/if}
                        {#if handEntryOpen}
                            <button
                                class="icon-btn"
                                class:active={handOpen}
                                onclick={() => (handOpen = !handOpen)}
                                title={$t('transcription.handEntryTooltip')}
                                aria-label={$t('transcription.handEntry')}>✎</button
                            >
                        {/if}
                    </div>

                    {#if cubeRowOpen}
                        <CubeActionRow canAct={canCubeAct} canAnswer={canCubeAnswer} {resigning} onGesture={sendCube} onResign={startResign} onLevel={pickResignLevel} onCancelResign={abortResign} />
                    {/if}

                    {#if handOpen && handEntryOpen}
                        <!-- Le secours (T2.4) : il prend la PLACE du triangle, les
                             deux ne servant jamais en même temps. Il était visible
                             250 fois par match pour un usage attendu d'une fois. -->
                        <div class="hand-entry">
                            <button class="edit-btn" class:active={freeMode} onclick={toggleFreeMode} title={$t('transcription.freeMoveTooltip')}>{$t('transcription.freeMove')}</button>
                            {#if freeMode}
                                <button class="edit-btn" onclick={validateFreePlay} disabled={!diceEntered || freeSteps === 0} title={$t('transcription.boardIsPlayTooltip')}
                                    >{$t('transcription.boardIsPlay')}</button
                                >
                            {/if}
                            <input
                                class="notation-input"
                                type="text"
                                bind:value={notationText}
                                placeholder={$t('transcription.notationPlaceholder')}
                                aria-label={$t('transcription.notationLabel')}
                                onkeydown={(event) => {
                                    if (event.key === 'Enter') {
                                        event.preventDefault();
                                        event.stopPropagation();
                                        validateNotation();
                                    }
                                }}
                            />
                            <button class="edit-btn" onclick={validateNotation} disabled={!diceEntered || !notationText.trim()} title={$t('transcription.notationTooltip')}
                                >{$t('transcription.notationApply')}</button
                            >
                        </div>
                    {:else if diceEntryOpen}
                        <!-- La cible souris des dés (T2.1), sous les deux cases du
                             jet et jamais à leur place : le clavier reste deux fois
                             plus rapide (0,56 s contre 1,21 s) et les deux entrées
                             coexistent. -->
                        <DiceTriangle single={expects === 'opening'} allowed={rollsAllowed} onPick={pickDice} onDie={pickDie} />
                    {/if}
                </div>

                <div class="transcript-col">
                    {#if lastFlags.length}
                        <!-- L'alerte habite là où est la cellule fautive (R2) :
                             une Incohérence est MARQUÉE, jamais refusée (ADR-0044),
                             et « il y en a une » doit se voir sans chercher un ⚠
                             dans une colonne. -->
                        <p class="flag">{$t('transcription.inconsistencyPrefix')} {lastFlags.join(' · ')}</p>
                    {/if}
                    <TranscriptView {annotated} cursor={annotated?.cursor ?? 0} players={[playerName(0), playerName(1)]} onSelect={selectAction} onMenu={openTranscriptMenu} />
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

<!-- La largeur `wide` est le point de la décision 6 : 62 caractères de monospace
     demandent ~409 px, et l'alignement en colonnes EST l'information. -->
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

    /* HIÉRARCHIE (ADR-0048 décision 3). Onze contrôles portaient la même bordure,
       le même fond et la même taille : une pastille en lecture seule et un bouton
       qui réécrit le match se ressemblaient trait pour trait. L'action principale
       est le bouton qui matérialise le Match ; celui qui supprime la ligne se
       distingue sans crier. */
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

    .hand-entry {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--space-1);
    }

    /* Ni taille ni famille ici : la règle globale de style.css met déjà
       `font: inherit` sur les contrôles de formulaire (ADR-0008). */
    .notation-input {
        flex: 1 1 8rem;
        min-width: 6rem;
        padding: var(--space-1);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
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

    /* LA CHAÎNE DE HAUTEUR (ADR-0048 décision 5). Sans `flex: 1; min-height: 0`
       jusqu'en bas, les `overflow: auto` intérieurs n'ont rien à faire déborder :
       `.tab-content` devient le seul conteneur qui défile, le panneau défile en
       bloc, et le `scrollIntoView` du Cursor le fait sauter à chaque Action
       validée. `container-type` sert le point de rupture ci-dessous. */
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

    /* Pousse les gestes vers la droite : l'état reste à gauche, collé au bouton
       qu'il concerne, et les gestes forment un groupe. */
    .bar-gap {
        flex: 1 1 auto;
    }

    .save-state {
        color: var(--color-text-muted);
    }

    /* Trois régions. L'ordre du DOM est celui de la boîte ÉTROITE — candidats,
       palette, Transcript — et la palette passe première en boîte large. Un seul
       point de rupture, et il n'est pas porteur : le contrat (les deux cases du
       jet et cinq lignes de candidats sans défilement) tient des deux côtés. */
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
    }

    /* Sous le plancher — un dock que l'utilisateur écrase quand même — c'est la
       PALETTE qui cède, dans sa propre boîte, et non le panneau qui se met à
       défiler en bloc. La ligne du jet est en tête, donc le contrat tient
       encore : ce qui disparaît est le bas du triangle, pas les dés. */
    .palette-col {
        flex: 0 0 auto;
        overflow: auto;
    }

    .transcript-col {
        flex: 1 1 auto;
    }

    @container (min-width: 900px) {
        .draft-body {
            flex-direction: row;
        }

        .palette-col {
            order: -1;
            flex: 0 0 220px;
        }

        .candidates-col {
            flex: 1 1 280px;
        }

        .transcript-col {
            flex: 1 1 320px;
        }
    }

    /* Les cinq réponses possibles à « qu'a fait le camp au trait ? » sur une
       ligne : les deux dés, les quatre gestes de videau, et le volet de secours
       (ADR-0048 décision 7). */
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

    /* Le filtre par point : une PUCE et non une phrase. Elle disait qu'il était
       posé sans dire comment en sortir ; le ✕ est le geste (décision 8). */
    .list-note {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .chip {
        padding: 0 var(--space-2);
        border: 1px solid var(--color-primary);
        border-radius: 999px;
        background: var(--color-surface);
        color: var(--color-primary);
        cursor: pointer;
    }

    .chip:hover {
        background: var(--color-surface-alt);
    }

    /* C'est ICI que le défilement a lieu, et nulle part ailleurs. */
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

    .edit-btn {
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    .edit-btn:hover:not(:disabled) {
        background: var(--color-surface-alt);
    }

    .edit-btn:disabled {
        color: var(--color-text-muted);
        cursor: default;
    }

    /* La bascule enfoncée : le plateau ne joue plus les coups légaux, il
       déplace librement, et cela doit se lire sans ouvrir la documentation. */
    .edit-btn.active {
        border-color: var(--color-primary);
        color: var(--color-primary);
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

    /* 62 caractères, la ligne la plus longue d'un `.mat` : la modale leur donne
       la largeur que la colonne du Transcript n'avait pas. */
    .mat-text {
        margin: 0;
        max-height: 60vh;
        overflow: auto;
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
    }
</style>
