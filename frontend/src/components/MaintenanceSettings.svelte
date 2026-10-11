<script>
    // The library's state of health (ADR-0088): one line per operation, its count, its last
    // run and its action. Nothing here runs on its own; a count that costs a scan waits for
    // its "Count" button, and every count of positions or matches opens what it counts.
    import { get } from 'svelte/store';
    import { onDestroy } from 'svelte';
    import PanelHeader from './panels/PanelHeader.svelte';
    import PanelTable from './panels/PanelTable.svelte';
    import CountLink from './panels/CountLink.svelte';
    import EmptyState from './panels/EmptyState.svelte';
    import { t, tMsg } from '../i18n';
    import { databasePathStore } from '../stores/databaseStore.js';
    import { statusBarTextStore, MODAL, openModal } from '../stores/uiStore';
    import { maintenanceStore, maintenanceEndedStore } from '../stores/maintenanceStore.js';
    import { gammonNetBatchStore } from '../stores/gammonNetBatchStore';
    import { confirmAction } from '../services/confirmService.js';
    import { openPositionIDs, openMatchListInPanel } from '../services/positionLoader.js';
    import { busyMessage } from '../utils/busyMessage.js';
    import { formatDateTime } from '../utils/format.js';
    import { logger } from '../utils/logger.js';
    import { MaintenanceStatus, CountMaintenance, MaintenanceIDs, StaleGammonNetIDs, IsReadOnly, Vacuum } from '../../wailsjs/go/database/Database.js';
    import { StartMaintenance, StartGammonNetBatch, StartGammonNetStaleBatch } from '../../wailsjs/go/gui/App.js';

    /**
     * @typedef {{ kind: string, tier: string, count?: number, unit?: string, listable: boolean, runnable: boolean, applicable: boolean, last?: { finishedAt: number, changed: number, outcome: string } }} Line
     */

    /** @type {{ ply?: number, pruneK?: number, candidates?: number, onLeave?: () => void }} */
    let { ply = 0, pruneK = 0, candidates = 0, onLeave = () => {} } = $props();

    // The stale gammonNet verdicts are no maintenance kind of the backend: they depend on the
    // ply set here, and the scan that counts them is the one that lists them.
    const STALE = 'analysis.stale';

    let lines = $state(/** @type {Line[]} */ ([]));
    let readOnly = $state(false);
    /** Counts asked for with "Count", by kind; dropped whenever a pass may have changed them. */
    let counted = $state(/** @type {Record<string, number>} */ ({}));
    /** Kinds whose count is being computed. */
    let counting = $state(/** @type {Record<string, boolean>} */ ({}));
    let staleIDs = /** @type {number[]} */ ([]);
    let vacuumBusy = $state(false);

    let running = $derived($maintenanceStore);
    let busy = $derived(running != null || $gammonNetBatchStore != null || vacuumBusy);

    /** @param {string} kind */
    const labelKey = (kind) => kind.replace('.', '_');

    async function refresh() {
        if (!get(databasePathStore)) {
            lines = [];
            return;
        }
        try {
            const [status, ro] = await Promise.all([MaintenanceStatus(), IsReadOnly().catch(() => false)]);
            /** @type {Line[]} */
            const next = [];
            for (const line of /** @type {Line[]} */ (status ?? [])) {
                if (!line.applicable) continue;
                next.push(line);
                if (line.kind === 'analysis.missing') {
                    next.push({ kind: STALE, tier: 'scan', unit: 'positions', listable: true, runnable: false, applicable: true });
                }
            }
            lines = next;
            readOnly = !!ro;
            counted = {};
        } catch (error) {
            logger.error('Error reading the maintenance status:', error);
            lines = [];
        }
    }

    $effect(() => {
        $databasePathStore;
        $maintenanceEndedStore;
        refresh();
    });

    // A gammonNet batch ending changes the analysis counts.
    let batchWasRunning = false;
    const unsubBatch = gammonNetBatchStore.subscribe((b) => {
        if (batchWasRunning && b == null) refresh();
        batchWasRunning = b != null;
    });
    onDestroy(unsubBatch);

    /** @param {Line} line */
    function countOf(line) {
        return line.count ?? counted[line.kind];
    }

    /** @param {Line} line */
    function countable(line) {
        return line.tier !== 'none' && countOf(line) == null;
    }

    /** @param {Line} line */
    async function count(line) {
        counting = { ...counting, [line.kind]: true };
        try {
            if (line.kind === STALE) {
                staleIDs = (await StaleGammonNetIDs(ply)) ?? [];
                counted = { ...counted, [STALE]: staleIDs.length };
            } else {
                counted = { ...counted, [line.kind]: await CountMaintenance(line.kind) };
            }
        } catch (error) {
            statusBarTextStore.set(tMsg('maintenance.failed', { op: get(t)(`maintenance.line_${labelKey(line.kind)}`), error: String(error) }));
        } finally {
            counting = { ...counting, [line.kind]: false };
        }
    }

    /** @param {Line} line */
    async function open(line) {
        if (line.kind === 'trash') {
            openModal(MODAL.TRASH);
            return;
        }
        const ids = line.kind === STALE ? staleIDs : ((await MaintenanceIDs(line.kind)) ?? []);
        onLeave();
        if (line.unit === 'matches') await openMatchListInPanel(ids);
        else openPositionIDs(ids);
    }

    /** @param {Line} line */
    function clickable(line) {
        return line.listable || line.kind === 'trash';
    }

    /**
     * The label of a line's action, or '' when it has none here.
     * @param {Line} line
     */
    function actionKey(line) {
        if (line.kind === 'analysis.missing' || line.kind === STALE) return 'maintenance.analyze';
        if (line.kind === 'vacuum') return 'maintenance.compact';
        if (line.runnable) return 'maintenance.run';
        return '';
    }

    /** @param {Line} line */
    function actionDisabled(line) {
        if (readOnly || busy) return true;
        const n = countOf(line);
        return n === 0;
    }

    /** @param {Line} line */
    async function act(line) {
        if (line.kind === 'analysis.missing') StartGammonNetBatch(ply, pruneK, candidates);
        else if (line.kind === STALE) StartGammonNetStaleBatch(ply, pruneK, candidates);
        else if (line.kind === 'vacuum') await vacuum();
        else {
            // Shown at once: the status bar's listeners replace it with the real progress.
            maintenanceStore.set({ kind: line.kind, done: 0, total: 0 });
            StartMaintenance(line.kind);
        }
    }

    async function vacuum() {
        if (!(await confirmAction(get(t)('config.vacuumConfirm'), { confirmLabel: get(t)('config.vacuumConfirmButton') }))) return;
        vacuumBusy = true;
        try {
            const result = await Vacuum();
            const reclaimed = Math.max(0, (result?.SizeBefore ?? 0) - (result?.SizeAfter ?? 0));
            statusBarTextStore.set(reclaimed > 0 ? tMsg('config.vacuumDone', { mb: (reclaimed / 1e6).toFixed(1) }) : tMsg('config.vacuumNothing'));
        } catch (error) {
            const message = String(error);
            statusBarTextStore.set(tMsg('config.vacuumError', { error: busyMessage(message) ?? message }));
        } finally {
            vacuumBusy = false;
            refresh();
        }
    }

    /**
     * @param {Line} line
     * @param {number} n
     */
    function formatCountOf(line, n) {
        if (line.unit === 'bytes') return get(t)('maintenance.megabytes', { mb: (n / 1e6).toFixed(1) });
        return n.toLocaleString();
    }

    /** @param {NonNullable<Line['last']>} last */
    function lastLabel(last) {
        return `${formatDateTime(last.finishedAt * 1000)} · ${get(t)(`maintenance.outcome_${last.outcome}`)}`;
    }

    let columns = $derived([
        { key: 'op', label: $t('maintenance.colOperation'), elastic: true },
        { key: 'count', label: $t('maintenance.colCount'), narrow: true, align: 'right' },
        { key: 'last', label: $t('maintenance.colLast'), narrow: true },
        { key: 'action', label: '', actions: true }
    ]);
</script>

<div class="maintenance" data-testid="maintenance-tab">
    <PanelHeader title={$t('maintenance.title')}>
        {#snippet actions()}
            <button type="button" class="secondary-button" onclick={refresh} title={$t('maintenance.refresh')} aria-label={$t('maintenance.refresh')}>↻</button>
        {/snippet}
    </PanelHeader>
    {#if !$databasePathStore}
        <EmptyState text={$t('maintenance.noDatabase')} actions={false} />
    {:else}
        <PanelTable rows={lines} rowKey={(/** @type {Line} */ l) => l.kind} {columns}>
            {#snippet cells(/** @type {Line} */ line)}
                {@const n = countOf(line)}
                <td title={$t(`maintenance.hint_${labelKey(line.kind)}`)}>{$t(`maintenance.line_${labelKey(line.kind)}`)}</td>
                <td class="align-right nowrap" data-testid={`maintenance-count-${line.kind}`}>
                    {#if n != null}
                        {#if clickable(line) && n > 0}
                            <CountLink label={formatCountOf(line, n)} onclick={() => open(line)} testid={`maintenance-open-${line.kind}`} />
                        {:else}
                            {formatCountOf(line, n)}
                        {/if}
                    {:else if countable(line)}
                        <button type="button" class="link-button" disabled={counting[line.kind]} onclick={() => count(line)} data-testid={`maintenance-countbtn-${line.kind}`}>
                            {counting[line.kind] ? '…' : $t('maintenance.count')}
                        </button>
                    {:else}
                        <span class="muted">—</span>
                    {/if}
                </td>
                <td class="muted">
                    {#if running?.kind === line.kind}
                        {running.total > 0 ? `${running.done}/${running.total}` : $t('maintenance.running')}
                    {:else if line.last}
                        <span title={$t('maintenance.changed', { n: line.last.changed })}>{lastLabel(line.last)}</span>
                    {/if}
                </td>
                <td class="actions-col">
                    {#if actionKey(line)}
                        <button
                            type="button"
                            class="secondary-button"
                            disabled={actionDisabled(line)}
                            title={readOnly ? $t('maintenance.readOnly') : undefined}
                            onclick={() => act(line)}
                            data-testid={`maintenance-run-${line.kind}`}>{$t(actionKey(line))}</button
                        >
                    {/if}
                </td>
            {/snippet}
        </PanelTable>
    {/if}
</div>

<style>
    .maintenance {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
    }
    .nowrap {
        white-space: nowrap;
    }
    .muted {
        color: var(--color-text-muted);
        white-space: nowrap;
    }
    .secondary-button {
        padding: var(--space-1) 10px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background-color: var(--color-surface-alt);
        color: var(--color-text);
        cursor: pointer;
        white-space: nowrap;
    }
    .secondary-button:disabled {
        cursor: default;
        opacity: 0.6;
    }
    .link-button {
        background: none;
        border: none;
        padding: 0;
        color: var(--color-primary);
        text-decoration: underline;
        cursor: pointer;
    }
    .link-button:disabled {
        cursor: default;
        opacity: 0.6;
    }
</style>
