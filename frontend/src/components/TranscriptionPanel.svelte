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

  What is not here yet: the resignation (T1.5), the Transcript column (T1.6),
  correcting through the Cursor (T1.7), saving (T1.9).

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
    import { PHASE, COMMAND, pressKey, applyCandidates, selectCandidate } from '../services/transcriptionKeys.js';
    import CandidateMovesTable from './CandidateMovesTable.svelte';
    import { transcriptionListStore, transcriptionStore, transcriptionKeyStore, setTranscription, clearTranscription, resetTranscriptionKeys } from '../stores/transcriptionStore.js';
    import { ListTranscriptions, CreateTranscription, OpenTranscription, ApplyTranscriptionGesture } from '../../wailsjs/go/database/Database.js';
    import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';
    import { GetGammonNetPruneK } from '../../wailsjs/go/main/Config.js';
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
    // side — closing it for good (deleting the row) is its own gesture, with a
    // confirmation, and belongs to T1.9.
    function backToList() {
        clearTranscription();
        refresh();
    }

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
            // Le camp d'une action de videau n'est PAS posé ici : sans `HasSide`
            // le moteur prend celui qu'il attend — le camp au trait pour un
            // double, le camp d'en face pour une réponse (transcript.cubeGesture).
            case COMMAND.DOUBLE:
                return { Kind: 'double' };
            case COMMAND.TAKE:
                return { Kind: 'take' };
            case COMMAND.PASS:
                return { Kind: 'pass' };
            case COMMAND.SELECT: {
                const entry = ranked[command.index];
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
        const gestures = commands.map(gestureOf).filter(Boolean);
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

    // ── the candidates ───────────────────────────────────────────────────
    //
    // Two calls, both of them the ones the Eval panel already makes: LegalMoves
    // for the plays and their steps, EvaluatePositionImmediate for the 0-ply
    // ranking (candidates = 0, so ALL of them come back — not the ten a stored
    // analysis keeps). Never StartEvaluationAtRest: a transcription must answer
    // between two keystrokes, and the display-depth tier costs a fifth of a
    // second.

    let candidateGeneration = 0;

    /** The Position the roll being entered is played from, dice and side set. */
    function rollPosition() {
        const state = get(transcriptionKeyStore);
        const ann = get(transcriptionStore)?.annotated;
        const base = ann?.next?.position;
        if (!base || !state.dice[0] || !state.dice[1]) return null;
        return {
            ...structuredClone(base),
            id: 0,
            dice: [state.dice[0], state.dice[1]],
            player_on_roll: ann.next.side,
            decision_type: 0
        };
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
            const list = moves.map((move) => ({ move, gen: byNotation[move.move] })).filter((row) => row.gen !== undefined);
            if (list.length) return { list, unranked: false };
        } catch (err) {
            logger.error('The 0-ply ranking of a transcription roll failed:', err);
        }
        // The generator's own order, which is an order and not a ranking.
        return { list: plays.map((play, index) => ({ move: { index, move: play.notation }, gen: index })), unranked: true };
    }

    /**
     * Answers the roll the machine is waiting on: no legal play is a dance, one
     * or more preselects the first. Superseded by a newer roll rather than
     * cancelled — a stale answer must never land on the roll that replaced it.
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
        if (panelKeyGuard(event)) return;
        if (!panelEl?.contains(document.activeElement)) return;

        const result = pressKey(keys, event, { expects });
        if (!result.handled) return;
        event.preventDefault();
        event.stopPropagation();

        // A roll that is starting again, or one that has just been validated,
        // leaves a list that belongs to nobody: it goes before the gestures do,
        // so no `select` can address it any more.
        if (result.state.phase === PHASE.DICE || result.state.phase === PHASE.DIE1) {
            ranked = [];
            unranked = false;
        }
        danced = false;

        transcriptionKeyStore.set(result.state);
        run(result.commands).then(settleCandidates);
    }

    onMount(() => {
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('keydown', handleKeyDown);
        selectedMoveStore.set(null);
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
        const rolled = !opening && dice[0] > 0 && dice[1] > 0 ? [dice[0], dice[1]] : [0, 0];
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
        selectedMoveStore.set(ranked[keys.selected]?.move?.move ?? null);
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

    // The two dice as they come in, "·" for a die not entered yet. An opening's
    // two dice belong to two different players, which is why they are shown
    // here and not drawn on the board.
    let dieCells = $derived([keys.dice[0] || '·', keys.dice[1] || '·']);

    let rankedMoves = $derived(ranked.map((row) => row.move));
    // Which referential the equity column is stated in (ADR-0016 point 6,
    // ADR-0019): money points at money play, normalised match equity at a score.
    let isMoney = $derived(isMoneyPosition(annotated?.next?.position));
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
            </div>

            {#if matchOver}
                <p class="hint">{$t('transcription.matchOver', { player: playerName(matchWinner), a: score[0], b: score[1] })}</p>
            {/if}

            <div class="entry">
                <span class="entry-label">
                    {#if awaitingAnswer}
                        {$t('transcription.answerPrompt', { player: playerName(sideOnRoll) })}
                    {:else if expects === 'opening'}
                        {$t('transcription.openingPrompt')}
                    {:else}
                        {$t('transcription.rollPrompt', { player: playerName(sideOnRoll) })}
                    {/if}
                </span>
                {#if !awaitingAnswer}
                    <span class="die" class:filled={keys.dice[0] > 0}>{dieCells[0]}</span>
                    <span class="die" class:filled={keys.dice[1] > 0}>{dieCells[1]}</span>
                {/if}
            </div>

            {#if lastFlags.length}
                <!-- Une Incohérence est MARQUÉE, jamais refusée (ADR-0044) :
                     l'Action est dans le document, et la phrase dit laquelle. -->
                <p class="flag">{$t('transcription.inconsistencyPrefix')} {lastFlags.join(' · ')}</p>
            {/if}

            {#if awaitingAnswer}
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

            {#if ranked.length && !awaitingAnswer}
                {#if unranked}
                    <p class="hint">{$t('transcription.unranked')}</p>
                    <ol class="plain-candidates">
                        {#each ranked as row, index (row.gen)}
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
