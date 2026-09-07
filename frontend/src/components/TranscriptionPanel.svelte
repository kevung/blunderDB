<!--
  TranscriptionPanel — the library's transcription drafts, and the draft open.

  A Transcription is a DRAFT of a match being typed in, distinct from the Match
  it produces (ADR-0045). This panel is both its front door — the drafts that
  exist, the form that starts one — and, once a draft is open, the surface the
  match is typed on.

  What T1.2 puts here: the creation form (fonctionnel.md §1.1 — the LENGTH is
  the only field asked for, `0` being a money session, which is what unfolds the
  Jacoby and beaver flags) and the opening (§6 flux 2 — the die of player 1, the
  die of player 2, the stronger one starts and plays BOTH dice as its first
  checker play; a tie reads "relance" and waits for another opening).

  What is not here yet: the candidate list and the rest of the turn (T1.3), the
  cube (T1.4), the resignation (T1.5), the Transcript column (T1.6), correcting
  through the Cursor (T1.7), saving (T1.9).

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
    import { panelKeyGuard } from '../services/keyboardService.js';
    import { COMMAND, pressKey } from '../services/transcriptionKeys.js';
    import { transcriptionListStore, transcriptionStore, transcriptionKeyStore, setTranscription, clearTranscription, resetTranscriptionKeys } from '../stores/transcriptionStore.js';
    import { ListTranscriptions, CreateTranscription, OpenTranscription, ApplyTranscriptionGesture } from '../../wailsjs/go/database/Database.js';

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

    function gestureOf(command) {
        switch (command.kind) {
            case COMMAND.DIE:
                return { Kind: 'enter_die', Die: command.value };
            case COMMAND.CLEAR:
                return { Kind: 'clear_dice' };
            case COMMAND.VALIDATE:
                return { Kind: 'validate' };
            default:
                return null;
        }
    }

    function run(commands) {
        if (!commands.length) return pending;
        const id = draft?.id;
        if (id == null) return pending;
        pending = pending
            .then(async () => {
                for (const command of commands) {
                    const gesture = gestureOf(command);
                    if (!gesture) continue;
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
        transcriptionKeyStore.set(result.state);
        run(result.commands);
    }

    onMount(() => {
        document.addEventListener('keydown', handleKeyDown);
    });

    onDestroy(() => {
        document.removeEventListener('keydown', handleKeyDown);
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
    function boardPosition(ann) {
        const actions = ann.actions ?? [];
        const at = ann.cursor ?? 0;
        const current = at >= 0 && at < actions.length ? actions[at] : null;
        const base = current?.has_position ? current.before : ann.next?.position;
        if (!base) return null;
        return { ...structuredClone(base), id: 0 };
    }

    $effect(() => {
        if (!annotated || $statusBarModeStore !== 'TRANSCRIBE') return;
        const pos = boardPosition(annotated);
        if (pos) positionStore.set(pos);
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

    // The two dice as they come in, "·" for a die not entered yet. An opening's
    // two dice belong to two different players, which is why they are shown
    // here and not drawn on the board.
    let dieCells = $derived([keys.dice[0] || '·', keys.dice[1] || '·']);
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
                <span class="badge on-roll">{playerName(sideOnRoll)}</span>
            </div>

            <div class="entry">
                <span class="entry-label">
                    {expects === 'opening' ? $t('transcription.openingPrompt') : $t('transcription.rollPrompt', { player: playerName(sideOnRoll) })}
                </span>
                <span class="die" class:filled={keys.dice[0] > 0}>{dieCells[0]}</span>
                <span class="die" class:filled={keys.dice[1] > 0}>{dieCells[1]}</span>
            </div>

            {#if keys.tie}
                <p class="hint">{$t('transcription.tie')}</p>
            {:else if expects === 'opening'}
                <p class="hint">{$t('transcription.openingHint')}</p>
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

    .hint {
        margin: 0;
        color: var(--color-text-muted);
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
