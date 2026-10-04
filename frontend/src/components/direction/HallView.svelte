<script>
    /*
     * Toutes les tables (ADR-0056 §5, ADR-0058) : une seule grille, une case par table de l'Événement quelle que
     * soit l'épreuve, marquée de son épreuve, puis les propositions de chaque épreuve. Rien n'y
     * est calculé : la fusion est RencontreTableGrid, côté Go, et chaque geste part vers
     * l'épreuve de sa case. La grille est celle d'une épreuve (TableGrid) : mêmes menus, même
     * clavier, même glisser-déposer.
     */
    import { t, tMsg } from '../../i18n';
    import TableGrid from './TableGrid.svelte';
    import { proposalLabel, actionKey, isRepair, tableTitle } from './labels.js';
    import { hallEnterResult, hallEnterForfeit, hallMoveMatch, hallCancelMatch, hallConfirmProposal } from '../../stores/directionStore';

    /** @typedef {import('../../../wailsjs/go/models').service.HallView} Hall */
    /** @typedef {import('../../../wailsjs/go/models').service.HallEvent} HallEvent */
    /** @typedef {import('../../stores/directionStore.js').ProposalAction} ProposalAction */
    /** @typedef {{ tournamentId?: number, table: number, matchId?: string }} HallCellLike */

    /**
     * @type {{
     *     hall?: Hall | null,
     *     error?: string,
     *     busy?: boolean,
     *     act: (fn: () => Promise<unknown>, key: string | ((e: any) => import('../../i18n').StatusMessage)) => Promise<boolean>,
     *     onHistory?: (name: string, tournamentId: number) => void,
     *     onOutOfService?: (table: number, out: boolean) => void
     * }}
     */
    let { hall = null, error = '', busy = false, act, onHistory, onOutOfService } = $props();

    /** @param {HallCellLike | undefined} c */
    const tidOf = (c) => c?.tournamentId || 0;

    /** @type {(m: string, w: string, a: number, b: number, note: string, c?: HallCellLike) => Promise<boolean>} */
    const onResult = (m, w, a, b, note, c) => act(() => hallEnterResult(tidOf(c), m, w, a, b, note), 'direction.result.error');
    /** @type {(m: string, w: string, note: string, c?: HallCellLike) => Promise<boolean>} */
    const onForfeit = (m, w, note, c) => act(() => hallEnterForfeit(tidOf(c), m, w, note), 'direction.result.error');
    /** @type {(m: string, table: number, c?: HallCellLike) => Promise<boolean>} */
    const onMove = (m, table, c) =>
        act(
            () => hallMoveMatch(tidOf(c), m, table),
            (e) => tMsg('direction.table.moveRefused', { reason: String(e?.message ?? e).replace(/^direction:\s*/, '') })
        );
    /** @type {(m: string, c?: HallCellLike) => Promise<boolean>} */
    const onCancel = (m, c) => act(() => hallCancelMatch(tidOf(c), m), 'direction.result.error');

    /** @param {number} tid @param {ProposalAction} a */
    const launch = (tid, a) => act(() => hallConfirmProposal(tid, a), 'direction.proposals.errorConfirm');

    /*
     * Une seule file, toutes épreuves mêlées : l'ordre est celui de RencontreTableGrid (les
     * joueurs qui attendent depuis le plus longtemps d'abord). Les matchs retenus faute de table
     * ou de joueur libre suivent, sans bouton : lancés, ils ne seraient sur aucune table.
     */
    const queue = $derived(hall?.queue || []);
    const held = $derived(hall?.held || []);
    /** @type {Record<number, HallEvent>} */
    const eventOf = $derived(Object.fromEntries((hall?.events || []).map((ev) => [ev.tournamentId, ev])));

    /** @param {HallEvent} ev @returns {(id: string | undefined) => string} */
    const namesOf = (ev) => (id) => (id && ev.names?.[id]) || id || '';
    /** Le nom des tables de l'Événement qui en portent un. */
    const tableNames = $derived(Object.fromEntries((hall?.cells || []).filter((c) => c.name).map((c) => [c.table, c.name])));
    /** @param {number} index */
    const evColor = (index) => `--ev: var(--td-event-${index % 6})`;
</script>

<div class="hall" data-testid="direction-hall">
    {#if error}
        <p class="error" role="alert" data-testid="hall-error">{$t('direction.hall.error', { reason: error })}</p>
    {/if}
    {#if hall}
        {#each hall.events.filter((ev) => ev.error) as ev (ev.tournamentId)}
            <p class="error" role="alert" data-testid="hall-event-error-{ev.tournamentId}">{$t('direction.hall.eventError', { event: ev.name, reason: ev.error })}</p>
        {/each}
        <ul class="legend" aria-label={$t('direction.hall.legend')}>
            {#each hall.events as ev (ev.tournamentId)}
                <li class="chip" style={evColor(ev.index)}>{ev.name}</li>
            {/each}
        </ul>
        <TableGrid cells={hall.cells || []} {busy} {onResult} {onForfeit} {onMove} {onCancel} onHistory={onHistory ? (name, c) => onHistory(name, tidOf(c)) : undefined} {onOutOfService} />
        <section class="proposals" data-testid="hall-proposals">
            <h3>{$t('direction.hall.proposals', { n: queue.length })}</h3>
            {#if queue.length}
                <ul data-testid="hall-queue">
                    {#each queue as p (p.tournamentId + ':' + actionKey(p.action))}
                        {@const a = p.action}
                        <li style={evColor(p.eventIndex)} data-testid="hall-proposal-{p.tournamentId}">
                            <span class="chip">{eventOf[p.tournamentId]?.name ?? ''}</span>
                            <span class="what">{proposalLabel($t, a, namesOf(eventOf[p.tournamentId]))}</span>
                            {#if a.length}<span class="meta">{$t('direction.proposals.points', { n: a.length })}</span>{/if}
                            {#if a.table}<span class="meta">{$t('direction.proposals.table', { n: tableTitle(a.table, tableNames[a.table]) })}</span>{/if}
                            <span class="grow"></span>
                            <button type="button" class="go" disabled={busy} onclick={() => launch(p.tournamentId, a)}>
                                {isRepair(a) ? $t('direction.proposals.cancelMatch') : $t('direction.proposals.launch')}
                            </button>
                        </li>
                    {/each}
                </ul>
            {/if}
            {#if held.length}
                <ul class="held" data-testid="hall-held">
                    {#each held as p (p.tournamentId + ':' + actionKey(p.action))}
                        {@const a = p.action}
                        <li style={evColor(p.eventIndex)} data-testid="hall-held-{p.tournamentId}">
                            <span class="chip">{eventOf[p.tournamentId]?.name ?? ''}</span>
                            <span class="what">{proposalLabel($t, a, namesOf(eventOf[p.tournamentId]))}</span>
                            {#if a.length}<span class="meta">{$t('direction.proposals.points', { n: a.length })}</span>{/if}
                            <span class="meta warn">{$t(`direction.reason.${a.reason}`)}</span>
                        </li>
                    {/each}
                </ul>
            {/if}
            {#if queue.length === 0 && held.length === 0}
                <p class="empty">{$t('direction.proposals.none')}</p>
            {/if}
        </section>
    {:else if !error}
        <p class="empty">{$t('direction.hall.loading')}</p>
    {/if}
</div>

<style>
    .hall {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
        min-width: 0;
    }

    .legend {
        display: flex;
        flex-wrap: wrap;
        gap: var(--space-1);
        margin: 0;
        padding: var(--space-2) var(--space-2) 0;
        list-style: none;
    }

    .chip {
        padding: 0 var(--space-1);
        border-left: 3px solid var(--ev);
        border-radius: 3px;
        background: color-mix(in srgb, var(--ev) 15%, transparent);
        color: var(--color-text);
        font-size: var(--font-size-small);
        font-weight: 600;
    }

    .proposals {
        padding: 0 var(--space-2) var(--space-2);
    }

    h3 {
        margin: 0 0 var(--space-1);
        font-size: var(--font-size-base);
        font-weight: 600;
        color: var(--color-text);
    }

    ul {
        margin: 0;
        padding: 0;
        list-style: none;
    }

    .proposals li {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: var(--space-2);
        min-height: var(--td-target, auto);
        padding: 2px var(--space-1);
        border-left: 3px solid var(--ev);
        border-bottom: 1px solid var(--color-border);
    }

    .meta {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .meta.warn {
        color: var(--color-danger);
    }

    .held {
        margin-top: var(--space-2);
    }

    .grow {
        flex: 1;
    }

    .go {
        min-height: var(--td-target, auto);
        min-width: var(--td-target, auto);
    }

    .error {
        margin: var(--space-2) var(--space-2) 0;
        color: var(--color-danger);
    }

    .empty {
        color: var(--color-text-muted);
        padding: var(--space-2);
    }
</style>
