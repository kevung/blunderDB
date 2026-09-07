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
    import { isMoneyPosition } from '../utils/cubeDecision.js';
    import { PHASE, COMMAND, pressKey, applyCandidates, selectCandidate, cursorCommands, initialKeyState, enterDicePair, enterSingleDie, DICE_KINDS } from '../services/transcriptionKeys.js';
    import CandidateMovesTable from './CandidateMovesTable.svelte';
    import DiceTriangle from './DiceTriangle.svelte';
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
        setTranscription,
        clearTranscription,
        resetTranscriptionKeys,
        resetTranscriptionPointFilter
    } from '../stores/transcriptionStore.js';
    import { filterByPoints } from '../services/transcriptionFilter.js';
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
        if (inserting || !info?.has_position || (info.kind !== 'checker' && info.kind !== 'dance')) {
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
        applyMouseDice(enterDicePair(get(transcriptionKeyStore), high, low, { expects }));
    }

    /** Une case de la rangée des six : le dé d'un camp, à l'ouverture. */
    function pickDie(die) {
        if (!draft) return;
        applyMouseDice(enterSingleDie(get(transcriptionKeyStore), die, { expects }));
    }

    function chooseCandidate(index) {
        const next = selectCandidate(get(transcriptionKeyStore), index);
        transcriptionKeyStore.set(next.state);
        run(next.commands);
        panelEl?.focus({ preventScroll: true });
    }

    // ── the keyboard ─────────────────────────────────────────────────────
    //
    // The panel handles its own keys while it has focus (ux.md §3): the digits
    // are the dice and they mean something else elsewhere, so they cannot go
    // through the global dispatcher. panelKeyGuard states, once for every docked
    // panel, what a panel may never swallow — Ctrl combos, Space, '?', and
    // anything typed in a field.

    function handleKeyDown(event) {
        if (!draft) return;
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

        const result = pressKey(keys, event, { expects });
        if (!result.handled) return;
        event.preventDefault();
        event.stopPropagation();

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
        if ((kind === COMMAND.DELETE || kind === COMMAND.FLIP_SIDE) && !onAction) return;
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
        selectedMoveStore.set(null);
        // Le plateau lit ces deux magasins pour décider si un clic le concerne :
        // un panneau démonté ne doit plus rien filtrer.
        transcriptionCandidateStepsStore.set([]);
        resetTranscriptionPointFilter();
        // Le coup joué au plateau appartient au panneau : démonté, il ne doit
        // plus se dessiner ni prendre les clics du damier.
        quizPlayStore.set(null);
        boardPlayArmed = false;
    });

    // The panel takes the keyboard as soon as a draft is open: the whole point
    // of the design is that a match is typed without ever reaching for the
    // mouse, and a first keystroke that landed nowhere would break the count
    // before it starts.
    $effect(() => {
        if (draft) panelEl?.focus({ preventScroll: true });
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
    let lengthLabel = $derived(matchLength === 0 ? $t('transcription.money') : $t('transcription.points', { n: matchLength }));
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
    let cubeLabel = $derived.by(() => {
        const v = 1 << (cube.value ?? 0);
        if (awaitingAnswer) return $t('transcription.doubleOffered', { v });
        if (cube.owner !== 0 && cube.owner !== 1) return $t('transcription.cubeCentred', { v });
        return $t('transcription.cubeOwned', { v, player: playerName(cube.owner) });
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
    // Verrou d'enregistrement : il désarme le plateau pendant les quatre gestes
    // qui partent au moteur, sinon chaque réponse ré-armerait le coup — 21
    // appels à LegalMoves par geste, et un plateau qui se rejoue sous la souris.
    let recordingPlay = $state(false);
    let boardPlayArmed = false;
    let boardPlayGeneration = 0;

    let boardPlayOpen = $derived(!!draft && !matchOver && !awaitingAnswer && !recordingPlay && !freeMode && keys.phase === PHASE.DICE && expects === 'checker');

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

    let boardPlaySteps = $derived($quizPlayStore && !$quizPlayStore.free ? $quizPlayStore.steps.length : 0);
    let boardPlayAmbiguous = $derived((rollsAllowed?.size ?? 0) > 1);
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

    // ── the .mat text ────────────────────────────────────────────────────
    //
    // Rendered by the SAME renderer the library's matches go through
    // (TranscriptionMAT → transcript.MatchParts → ingest.RenderMAT): the pane
    // shows the file that would be written, not a second opinion about it.
    // It is fetched only while the pane is unfolded, and again at every change
    // of the document, so a folded pane costs no round trip per keystroke.
    let matOpen = $state(false);
    let matText = $state('');

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
                        <input id="transcriptionLength" class="length-input" type="text" inputmode="numeric" bind:value={formLength} />
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
                <td class="narrow-col">{row.match_id ? `#${row.match_id}` : $t('transcription.notSaved')}</td>
            {/snippet}
        </PanelTable>
    {:else}
        <div class="draft">
            <div class="draft-bar">
                <button class="new-btn" onclick={backToList}>{$t('transcription.backToList')}</button>
                <span class="badge">{lengthLabel}</span>
                {#if matchLength > 0}
                    <span class="badge">{$t('transcription.score', { a: score[0], b: score[1] })}</span>
                {/if}
                {#if annotated?.next?.crawford}
                    <span class="badge">{$t('transcription.crawford')}</span>
                {/if}
                <span class="badge">{$t('transcription.gameNumber', { n: gameNumber })}</span>
                <span class="badge">{cubeLabel}</span>
                <span class="badge on-roll">{playerName(sideOnRoll)}</span>
                <span class="badge save-state">{$t(saveState.key, saveState.params)}</span>
                <button class="new-btn" onclick={() => (metaOpen = !metaOpen)} title={$t('transcription.metadataTooltip')}>{$t('transcription.metadata')}</button>
                <button class="new-btn" onclick={handleSave} disabled={busy} title={$t('transcription.saveTooltip')}>{$t('transcription.save')}</button>
                <button class="new-btn" onclick={handleExport} disabled={busy} title={$t('transcription.exportMatTooltip')}>{$t('transcription.exportMat')}</button>
                <button class="new-btn" onclick={handleClose} disabled={busy} title={$t('transcription.closeDraftTooltip')}>{$t('transcription.closeDraft')}</button>
            </div>

            {#if metaOpen}
                <!-- L'en-tête du brouillon (T3.1) : les deux noms, l'événement,
                     le lieu, la ronde, la date, le transcripteur, le tournoi.
                     Rien n'y est exigé — un brouillon sans noms s'enregistre et
                     s'exporte, avec des en-têtes vides. -->
                <TranscriptionMetadata header={annotated?.document?.header ?? {}} apply={sendGesture} {busy} />
            {/if}

            {#if matchOver}
                <p class="hint">{$t('transcription.matchOver', { player: playerName(matchWinner), a: score[0], b: score[1] })}</p>
            {/if}

            <div class="edit-bar">
                <button class="edit-btn" onclick={() => runCommand(COMMAND.INSERT_BEFORE)} title={$t('transcription.insertBeforeTooltip')}>{$t('transcription.insertBefore')}</button>
                <button class="edit-btn" onclick={() => runCommand(COMMAND.INSERT_AFTER)} title={$t('transcription.insertAfterTooltip')}>{$t('transcription.insertAfter')}</button>
                <button class="edit-btn" onclick={() => runCommand(COMMAND.DELETE)} disabled={!onAction} title={$t('transcription.deleteTooltip')}>{$t('transcription.delete')}</button>
                <button class="edit-btn" onclick={() => runCommand(COMMAND.FLIP_SIDE)} disabled={!onAction} title={$t('transcription.flipSideTooltip')}>{$t('transcription.flipSide')}</button>
                <button class="edit-btn" onclick={() => runCommand(COMMAND.UNDO)} disabled={!history.canUndo} title={$t('transcription.undoTooltip')}>{$t('transcription.undo')}</button>
                <button class="edit-btn" onclick={() => runCommand(COMMAND.REDO)} disabled={!history.canRedo} title={$t('transcription.redoTooltip')}>{$t('transcription.redo')}</button>
            </div>

            <div class="draft-body">
                <div class="entry-col">
                    <div class="entry">
                        <span class="entry-label">
                            {#if keys.phase === PHASE.RESIGN}
                                {$t('transcription.resignPrompt', { player: playerName(sideOnRoll) })}
                            {:else if awaitingAnswer}
                                {$t('transcription.answerPrompt', { player: playerName(sideOnRoll) })}
                            {:else if expects === 'opening'}
                                {$t('transcription.openingPrompt')}
                            {:else}
                                {$t('transcription.rollPrompt', { player: playerName(sideOnRoll) })}
                            {/if}
                        </span>
                        {#if !awaitingAnswer && keys.phase !== PHASE.RESIGN}
                            <span class="die" class:filled={keys.dice[0] > 0}>{dieCells[0]}</span>
                            <span class="die" class:filled={keys.dice[1] > 0}>{dieCells[1]}</span>
                        {/if}
                    </div>

                    {#if diceEntryOpen}
                        <!-- La cible souris des dés (T2.1), SOUS les deux cases
                             du jet et jamais à leur place : le clavier reste
                             deux fois plus rapide (0,56 s contre 1,21 s) et les
                             deux entrées coexistent. Sous, et non à côté :
                             mesuré à 178 px, le triangle ne laisserait pas de
                             quoi écrire la phrase du camp au trait dans une
                             colonne de 288 px. -->
                        <DiceTriangle single={expects === 'opening'} allowed={rollsAllowed} onPick={pickDice} onDie={pickDie} />
                    {/if}

                    {#if handEntryOpen}
                        <!-- Le coup au plateau (T2.3) et ses deux replis (T2.4).
                             La bascule et la notation sont SOUS le triangle :
                             elles servent une fois par match, quand le triangle
                             sert à chaque tour. -->
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
                    {/if}

                    {#if freeMode}
                        <p class="hint">{diceEntered ? $t('transcription.freeMoveHint') : $t('transcription.freeMoveNeedsDice')}</p>
                    {:else if boardPlayAmbiguous}
                        <!-- Le plateau ne peut pas trancher : plusieurs jets
                             produisent ce coup (une sortie, un dé injouable).
                             Le triangle n'offre plus qu'eux, et rien n'est
                             enregistré avant le clic. -->
                        <p class="hint">{$t('transcription.rollAmbiguous', { rolls: [...(rollsAllowed ?? [])].join(', ') })}</p>
                    {:else if boardPlaySteps > 0}
                        <p class="hint">{$t('transcription.boardPlaySteps', { n: boardPlaySteps })}</p>
                    {:else if boardPlayOpen}
                        <p class="hint">{$t('transcription.boardPlayHint')}</p>
                    {/if}

                    {#if lastFlags.length}
                        <!-- Une Incohérence est MARQUÉE, jamais refusée (ADR-0044) :
                             l'Action est dans le document, et la phrase dit laquelle. -->
                        <p class="flag">{$t('transcription.inconsistencyPrefix')} {lastFlags.join(' · ')}</p>
                    {/if}

                    {#if underReview}
                        <!-- Le coup enregistré n'est pas un coup du jet corrigé :
                             le premier candidat est posé à sa place et rien n'est
                             écrit avant la validation (fonctionnel.md §2). -->
                        <p class="flag">{$t('transcription.reviewHint')}</p>
                    {:else if correcting}
                        <p class="hint">{$t('transcription.correcting')}</p>
                    {/if}

                    {#if keys.phase === PHASE.RESIGN}
                        <p class="hint">{$t('transcription.resignHint')}</p>
                    {:else if awaitingAnswer}
                        <p class="hint">{$t('transcription.answerHint')}</p>
                    {:else if keys.tie}
                        <p class="hint">{$t('transcription.tie')}</p>
                    {:else if expects === 'opening'}
                        <p class="hint">{$t('transcription.openingHint')}</p>
                    {:else if danced}
                        <p class="hint">{$t('transcription.dance')}</p>
                    {:else if keys.phase === PHASE.ROLL}
                        <!-- Les deux états d'ux.md §3 sont DITS, parce qu'un même écran
                             y répond de deux façons opposées au même chiffre : tant que
                             la liste n'a pas été touchée il recommence le jet, après il
                             valide. Une différence invisible serait un piège. -->
                        <p class="hint">{$t('transcription.correctable')}</p>
                    {:else if keys.phase === PHASE.CANDIDATE}
                        <p class="hint">{$t('transcription.chosen')}</p>
                    {/if}

                    {#if $transcriptionPointFilterStore.length}
                        <!-- Le filtre est un état d'AFFICHAGE (T2.2) : il est
                             dit à l'écran, jamais écrit dans le document. -->
                        <p class="hint">{$t('transcription.pointFilter', { points: $transcriptionPointFilterStore.join(', '), n: visible.length })}</p>
                    {/if}

                    {#if visible.length}
                        {#if unranked}
                            <p class="hint">{$t('transcription.unranked')}</p>
                            <ol class="plain-candidates">
                                {#each visible as row, index (row.gen)}
                                    <li>
                                        <button class="plain-candidate" class:selected={index === keys.selected} onclick={() => chooseCandidate(index)}>{row.move.move}</button>
                                    </li>
                                {/each}
                            </ol>
                        {:else}
                            <div class="candidates">
                                <CandidateMovesTable
                                    moves={rankedMoves}
                                    selectedMove={$selectedMoveStore}
                                    onRowClick={(move) => chooseCandidate(rankedMoves.indexOf(move))}
                                    showProvenance={false}
                                    baseline={null}
                                    {isMoney}
                                />
                            </div>
                        {/if}
                    {/if}
                </div>

                <div class="transcript-col">
                    <TranscriptView {annotated} cursor={annotated?.cursor ?? 0} players={[playerName(0), playerName(1)]} {matText} onSelect={selectAction} onMatToggle={(open) => (matOpen = open)} />
                </div>
            </div>
        </div>
    {/if}
    {#if error}
        <p class="error">{error}</p>
    {/if}
</section>

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

    .new-btn:disabled {
        color: var(--color-text-muted);
        cursor: default;
    }

    .hand-entry {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--space-1);
        margin-top: var(--space-1);
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

    .draft {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
        padding: var(--space-2);
        min-height: 0;
    }

    .draft-bar {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--space-2);
    }

    .draft-bar .new-btn {
        margin-left: 0;
    }

    .badge {
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        color: var(--color-text-muted);
    }

    .on-roll {
        color: var(--color-text);
        font-weight: 600;
    }

    .edit-bar {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-1);
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

    .entry {
        display: flex;
        align-items: center;
        gap: var(--space-2);
    }

    .die {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 24px;
        height: 24px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        color: var(--color-text-muted);
        font-variant-numeric: tabular-nums;
    }

    .die.filled {
        color: var(--color-text);
        font-weight: 600;
    }

    .candidates {
        min-height: 0;
        overflow: auto;
    }

    .plain-candidates {
        margin: 0;
        padding-left: var(--space-4);
        max-height: 12em;
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

    /* Saisie à gauche, Transcript à droite (ux.md §2). Chaque colonne défile
       dans sa propre boîte : la page, elle, ne défile jamais latéralement. */
    .draft-body {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-2);
        min-height: 0;
    }

    .entry-col,
    .transcript-col {
        display: flex;
        flex: 1 1 18em;
        flex-direction: column;
        gap: var(--space-2);
        min-width: 0;
        min-height: 0;
    }

    .hint {
        margin: 0;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .flag {
        margin: 0;
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
</style>
