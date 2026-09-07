<!--
  TranscriptionPanel — the library's transcription drafts.

  A Transcription is a DRAFT of a match being typed in, distinct from the Match
  it produces (ADR-0045). This panel is its front door: the drafts that exist,
  and the button that starts one.

  What it is not, yet: the typing itself. The creation form (the length, the two
  players, the event) is T1.2 and the button below creates a draft of the
  default length until it lands; the saisie, the Transcript and the candidate
  list are T1.3 and after. The panel is deliberately the whole of the tab
  meanwhile, so the tab is never a blank screen.
-->
<script>
    import PanelTable from './panels/PanelTable.svelte';
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { databaseLoadedStore } from '../stores/databaseStore.js';
    import { activeTabStore } from '../stores/uiStore.js';
    import { transcriptionListStore, setTranscription } from '../stores/transcriptionStore.js';
    import { ListTranscriptions, CreateTranscription } from '../../wailsjs/go/database/Database.js';

    // The default length a bare "new transcription" opens on. It is Go's
    // transcript.DefaultMatchLength; stated here because the button sends a
    // header and T1.2's form will send the user's answer instead.
    const DEFAULT_MATCH_LENGTH = 7;

    let busy = $state(false);
    let error = $state('');

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

    async function createDraft() {
        if (busy || !$databaseLoadedStore) return;
        busy = true;
        try {
            const state = await CreateTranscription({ match_length: DEFAULT_MATCH_LENGTH });
            setTranscription(state);
            await refresh();
            error = '';
        } catch (err) {
            logger.error('Failed to create a transcription draft:', err);
            error = String(err);
        } finally {
            busy = false;
        }
    }

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
</script>

<div class="transcription-panel">
    <PanelTable rows={$transcriptionListStore} {columns} emptyText={$t('transcription.empty')}>
        {#snippet header()}
            <span class="detail-title">{$t('transcription.title')}</span>
            <button class="new-btn" onclick={createDraft} disabled={busy || !$databaseLoadedStore} title={$t('transcription.newTooltip')}>
                {$t('transcription.new')}
            </button>
        {/snippet}
        {#snippet cells(row)}
            <td class="stamp-cell">{shortStamp(row.updated_at || row.created_at)}</td>
            <td>{playersOf(row)}</td>
            <td class="narrow-col align-right">{lengthOf(row)}</td>
            <td class="narrow-col align-right">{row.action_count < 0 ? '—' : row.action_count}</td>
            <td class="narrow-col">{row.match_id ? `#${row.match_id}` : $t('transcription.notSaved')}</td>
        {/snippet}
    </PanelTable>
    {#if error}
        <p class="error">{error}</p>
    {/if}
</div>

<style>
    .transcription-panel {
        display: flex;
        flex-direction: column;
        height: 100%;
        min-height: 0;
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

    .stamp-cell {
        white-space: nowrap;
        color: var(--color-text-muted);
    }

    .error {
        margin: var(--space-2);
        color: var(--color-danger);
    }
</style>
