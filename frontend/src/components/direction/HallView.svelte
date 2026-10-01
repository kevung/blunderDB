<script>
    /*
     * La Salle (ADR-0056 §5) : une seule grille, une case par table de la Rencontre quelle que
     * soit l'épreuve, marquée de son épreuve, puis les propositions de chaque épreuve. Rien n'y
     * est calculé : la fusion est RencontreTableGrid, côté Go, et chaque geste part vers
     * l'épreuve de sa case. La grille est celle d'une épreuve (TableGrid) : mêmes menus, même
     * clavier, même glisser-déposer.
     */
    import { t, tMsg } from '../../i18n';
    import TableGrid from './TableGrid.svelte';
    import { proposalLabel, actionKey, isRepair } from './labels.js';
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

    /** Les propositions qu'on peut lancer, épreuve par épreuve ; « attendre » n'en est pas une. */
    const groups = $derived(
        (hall?.events || []).map((ev) => ({
            ev,
            actions: /** @type {ProposalAction[]} */ ((ev.proposals || []).filter((a) => a.kind !== 'wait'))
        }))
    );
    const pendingTotal = $derived(groups.reduce((n, g) => n + g.actions.length, 0));

    /** @param {HallEvent} ev @returns {(id: string | undefined) => string} */
    const namesOf = (ev) => (id) => (id && ev.names?.[id]) || id || '';
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
            <h3>{$t('direction.hall.proposals', { n: pendingTotal })}</h3>
            {#each groups as g (g.ev.tournamentId)}
                {#if g.actions.length}
                    <div class="group" style={evColor(g.ev.index)} data-testid="hall-proposals-{g.ev.tournamentId}">
                        <h4><span class="chip">{g.ev.name}</span></h4>
                        <ul>
                            {#each g.actions as a (actionKey(a))}
                                <li>
                                    <span class="what">{proposalLabel($t, a, namesOf(g.ev))}</span>
                                    {#if a.length}<span class="meta">{$t('direction.proposals.points', { n: a.length })}</span>{/if}
                                    {#if a.table}<span class="meta">{$t('direction.proposals.table', { n: a.table })}</span>{/if}
                                    <span class="grow"></span>
                                    <button type="button" class="go" disabled={busy} onclick={() => launch(g.ev.tournamentId, a)}>
                                        {isRepair(a) ? $t('direction.proposals.cancelMatch') : $t('direction.proposals.launch')}
                                    </button>
                                </li>
                            {/each}
                        </ul>
                    </div>
                {/if}
            {/each}
            {#if pendingTotal === 0}
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

    h4 {
        margin: var(--space-2) 0 var(--space-1);
        font-size: var(--font-size-base);
        font-weight: 600;
    }

    ul {
        margin: 0;
        padding: 0;
        list-style: none;
    }

    .group li {
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
