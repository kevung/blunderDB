<script>
    /*
     * La vue Historique (fonctionnel.md §5.8) : la trace lisible des décisions, filtrable par
     * joueur. Une correction s'offre sur la ligne (le match fini n'est plus sur la grille),
     * avec le panneau de la dernière décision.
     */
    import { t } from '../../i18n';
    import { closeOnEscape } from '../../services/escapeService.js';
    import { renderLabel } from './labels.js';
    import CorrectionPanel from './CorrectionPanel.svelte';

    /**
     * @type {{
     *     entries?: import('../../../wailsjs/go/models').database.HistoryEntry[],
     *     busy?: boolean,
     *     onCorrect?: (matchId: string, winner: string, scoreA: number, scoreB: number) => void,
     *     onCancel?: (matchId: string) => void,
     *     onNote?: (text: string) => void
     * }}
     */
    let { entries = [], busy = false, onCorrect = () => {}, onCancel = () => {}, onNote = () => {} } = $props();

    let filter = $state('');
    let note = $state('');
    /* La ligne dont la correction est ouverte : une seule à la fois, comme la fiche d'une table. */
    let correcting = $state(/** @type {number | null} */ (null));

    /* Échap referme la correction avant tout geste global. */
    $effect(() => {
        if (correcting !== null) return closeOnEscape(() => (correcting = null));
    });

    /**
     * @param {import('../../../wailsjs/go/models').database.HistoryEntry} e
     * @param {string} winner
     * @param {number} scoreA
     * @param {number} scoreB
     */
    function correct(e, winner, scoreA, scoreB) {
        onCorrect(e.matchId || '', winner, scoreA, scoreB);
        correcting = null;
    }

    const shown = $derived(
        filter.trim()
            ? entries.filter((e) => {
                  const q = filter.trim().toLowerCase();
                  return [e.playerName, e.aName, e.bName, e.winnerName, e.matchId, e.text].filter(Boolean).some((s) => s.toLowerCase().includes(q));
              })
            : entries
    );

    /** @param {string} iso */
    function when(iso) {
        if (!iso) return '';
        const d = new Date(iso);
        return Number.isNaN(d.getTime()) ? '' : d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
    }

    /* Ce que la ligne raconte. Chaque sorte d'événement a sa phrase, dans la langue de
       l'utilisateur : le moteur, lui, n'en écrit aucune. */
    /** @param {import('../../../wailsjs/go/models').database.HistoryEntry} e */
    function what(e) {
        switch (e.kind) {
            case 'created':
                return $t('direction.history.created');
            case 'player_added':
                return $t('direction.history.entered', { player: e.playerName || e.player });
            case 'player_withdrawn':
                return $t('direction.history.withdrawn', { player: e.playerName || e.player });
            case 'match_started':
                return $t('direction.history.started', {
                    a: e.aName,
                    b: e.bName,
                    where: renderLabel($t, e.label)
                });
            case 'result':
                if (e.forfeit) return $t('direction.history.forfeit', { winner: e.winnerName });
                /* `omitempty` : un côté absent vaut zéro ; les deux absents = aucun score saisi. */
                if (e.scoreA == null && e.scoreB == null) return $t('direction.history.resultNoScore', { winner: e.winnerName });
                return $t('direction.history.result', {
                    winner: e.winnerName,
                    scoreA: e.scoreA ?? 0,
                    scoreB: e.scoreB ?? 0
                });
            case 'result_corrected':
                return $t('direction.history.corrected', { winner: e.winnerName });
            case 'match_cancelled':
                return $t('direction.history.cancelled', { match: e.matchId });
            case 'bye':
                return $t('direction.history.bye', { player: e.playerName || e.player });
            case 'draw':
                return $t('direction.history.draw', { where: renderLabel($t, e.label) });
            case 'next_phase':
                return $t('direction.history.nextPhase');
            case 'config_changed':
                return $t('direction.history.configChanged');
            case 'table_changed':
                return $t('direction.history.tableChanged', { n: e.table });
            case 'length_changed':
                return $t('direction.history.lengthChanged', { n: e.length });
            case 'finished':
                return $t('direction.history.finished');
            case 'reopened':
                return $t('direction.history.reopened');
            case 'note':
                return e.text;
            default:
                return e.kind;
        }
    }
</script>

<div class="history">
    <header>
        <input bind:value={filter} type="text" placeholder={$t('direction.history.filter')} class="filter" data-testid="direction-history-filter" />
        <span class="count">{$t('direction.history.count', { n: shown.length })}</span>
    </header>

    <form
        class="note"
        onsubmit={(e) => {
            e.preventDefault();
            if (note.trim()) {
                onNote(note.trim());
                note = '';
            }
        }}
    >
        <input bind:value={note} type="text" placeholder={$t('direction.history.notePlaceholder')} />
        <button type="submit" disabled={busy || !note.trim()}>{$t('direction.history.addNote')}</button>
    </form>

    <ol>
        {#each shown as e (e.seq)}
            <li data-testid="direction-history-{e.seq}" class:note-line={e.kind === 'note'}>
                <span class="time">{when(e.time)}</span>
                <span class="what">{what(e)}</span>
                {#if e.text && e.kind !== 'note'}
                    <span class="remark">— {e.text}</span>
                {/if}
                <span class="grow"></span>
                {#if e.correctable}
                    <button type="button" data-testid="direction-history-correct" disabled={busy} onclick={() => (correcting = correcting === e.seq ? null : e.seq)}
                        >{$t('direction.last.correct')}</button
                    >
                {:else if e.cancellable}
                    <button type="button" disabled={busy} onclick={() => onCancel(e.matchId)}>{$t('direction.last.cancel')}</button>
                {/if}
            </li>
            {#if correcting === e.seq && e.correctable}
                <li class="correcting">
                    <CorrectionPanel a={e.a || ''} b={e.b || ''} aName={e.aName || ''} bName={e.bName || ''} {busy} testid="direction-history" onPick={(w, a, b) => correct(e, w, a, b)} />
                </li>
            {/if}
        {/each}
        {#if shown.length === 0}
            <li class="empty">{$t('direction.history.none')}</li>
        {/if}
    </ol>
</div>

<style>
    .history {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: var(--space-2);
        min-width: 0;
    }

    header,
    .note {
        display: flex;
        gap: var(--space-1);
        align-items: center;
    }

    .filter {
        min-width: 12rem;
    }

    .note input {
        flex: 1;
    }

    input {
        padding: 0.2rem 0.4rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
    }

    .count {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    ol {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
    }

    li {
        display: flex;
        align-items: baseline;
        gap: var(--space-1);
        padding: 0.15rem 0.2rem;
        border-bottom: 1px solid var(--color-surface-alt);
        font-size: var(--font-size-small);
    }

    /* Une annotation du directeur est ses propres mots : elle se distingue du reste, qui est
       le compte rendu de ses gestes. */
    li.note-line .what {
        font-style: italic;
    }

    .time {
        color: var(--color-text-muted);
        min-width: 3rem;
        font-variant-numeric: tabular-nums;
    }

    .remark {
        color: var(--color-text-muted);
    }

    .grow {
        flex: 1;
    }

    li.correcting {
        display: block;
        padding: 0;
    }

    .empty {
        color: var(--color-text-muted);
    }

    button {
        padding: 0.05rem 0.4rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
        font-size: var(--font-size-small);
    }

    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>
