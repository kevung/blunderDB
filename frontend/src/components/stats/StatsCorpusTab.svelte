<script>
    // Vues de corpus : face-à-face de deux joueurs, PR par fenêtre
    // calendaire. Le classement est l'onglet Joueurs, trié par PR et filtré
    // par colonne sur les décisions. Chaque vue se calcule à la demande — ce sont
    // des agrégats sur toute la base, trop chers pour suivre chaque frappe.
    import { HeadToHead, PRByWindow, GetAllPlayerNames, StatsMatchIDs } from '../../../wailsjs/go/database/Database.js';
    import { untrack } from 'svelte';
    import { t } from '../../i18n/index.js';
    import { loadPositionsFromStatsSelection, openMatchListInPanel, openMatchInPanel } from '../../services/positionLoader.js';
    import EntityAutocomplete from '../EntityAutocomplete.svelte';
    import { narrowToMonths } from '../../utils/statsWindow.js';
    import CountLink from '../panels/CountLink.svelte';
    import PanelTable from '../panels/PanelTable.svelte';
    import { readOfLibrary } from '../../stores/groupActivityStore.js';

    /** @type {{ filter: object }} */
    let { filter } = $props();

    let players = $state([]);
    // The filter's player is the obvious first side of a head-to-head.
    let playerA = $state(untrack(() => /** @type {any} */ (filter)?.playerName ?? ''));
    let playerB = $state('');
    let h2h = $state(/** @type {any} */ (null));
    let h2hError = $state('');

    let months = $state(3);
    let windows = $state(null);
    let windowsError = $state('');

    let busy = $state('');

    $effect(() => {
        GetAllPlayerNames()
            .then((list) => (players = (list ?? []).map((p) => p.Name)))
            .catch(() => (players = []));
    });

    const fmt = (v) => (Number.isFinite(v) ? v.toFixed(2) : '—');

    async function run(kind, fn) {
        busy = kind;
        try {
            await fn();
        } finally {
            busy = '';
        }
    }

    function loadH2H() {
        if (!playerA || !playerB) return;
        return run('h2h', async () => {
            h2hError = '';
            try {
                h2h = await HeadToHead(playerA, playerB, filter);
            } catch (err) {
                h2h = null;
                h2hError = err?.message ?? String(err);
            }
        });
    }

    function loadWindows() {
        return run('windows', async () => {
            windowsError = '';
            try {
                windows = (await readOfLibrary(() => PRByWindow(filter, Number(months)))) ?? [];
            } catch (err) {
                windows = null;
                windowsError = err?.message ?? String(err);
            }
        });
    }

    // A row opens what it sums up: a match row or a player's record opens
    // matches in the Match panel, a window's decisions open their positions.
    // A count inside a row opens the other half and stops there.

    /** The filter narrowed to a window's months (both ends inclusive). */
    const windowFilter = (/** @type {any} */ w) => narrowToMonths(/** @type {any} */ (filter), w);

    const openPositions = (/** @type {object} */ f) => loadPositionsFromStatsSelection(f, { Kind: 'all', OnlyWithError: false });

    async function openCountedMatches(/** @type {object} */ f) {
        await openMatchListInPanel((await StatsMatchIDs(/** @type {any} */ (f))) ?? []);
    }

    /** Enter or Space on a focused row does what a click does. */
    function onRowKey(/** @type {KeyboardEvent} */ e, /** @type {() => void} */ open) {
        if (e.target !== e.currentTarget || (e.key !== 'Enter' && e.key !== ' ')) return;
        e.preventDefault();
        open();
    }

    /** A count inside a clickable row: its own target, not the row's. */
    function countClick(/** @type {Event} */ e, /** @type {() => void} */ open) {
        e.stopPropagation();
        open();
    }

    const finite = (/** @type {any} */ v) => (Number.isFinite(v) ? v : null);

    const h2hColumns = $derived([
        { key: 'date', label: $t('stats.corpus.date'), sortable: true, filter: /** @type {const} */ ('text'), value: (/** @type {any} */ m) => m.date },
        { key: 'length', label: $t('stats.corpus.length'), sortable: true, align: 'right', filter: /** @type {const} */ ('number'), value: (/** @type {any} */ m) => m.match_length },
        { key: 'winner', label: $t('stats.corpus.winner'), sortable: true, filter: /** @type {const} */ ('text'), value: (/** @type {any} */ m) => outcomeLabel(m.outcome) },
        {
            key: 'pr_a',
            label: $t('stats.corpus.prOf', { name: h2h?.player_a ?? '' }),
            sortable: true,
            align: 'right',
            filter: /** @type {const} */ ('number'),
            value: (/** @type {any} */ m) => finite(m.pr_a)
        },
        {
            key: 'pr_b',
            label: $t('stats.corpus.prOf', { name: h2h?.player_b ?? '' }),
            sortable: true,
            align: 'right',
            filter: /** @type {const} */ ('number'),
            value: (/** @type {any} */ m) => finite(m.pr_b)
        }
    ]);

    const windowColumns = $derived([
        { key: 'from', label: $t('stats.corpus.from'), sortable: true, filter: /** @type {const} */ ('text'), value: (/** @type {any} */ w) => w.from },
        { key: 'to', label: $t('stats.corpus.to'), sortable: true, filter: /** @type {const} */ ('text'), value: (/** @type {any} */ w) => w.to },
        {
            key: 'matches',
            label: $t('stats.corpus.matches'),
            sortable: true,
            defaultDir: 'desc',
            align: 'right',
            filter: /** @type {const} */ ('number'),
            value: (/** @type {any} */ w) => w.num_matches
        },
        {
            key: 'decisions',
            label: $t('stats.corpus.decisions'),
            sortable: true,
            defaultDir: 'desc',
            align: 'right',
            filter: /** @type {const} */ ('number'),
            value: (/** @type {any} */ w) => w.num_decisions
        },
        { key: 'pr', label: 'PR', sortable: true, align: 'right', filter: /** @type {const} */ ('number'), value: (/** @type {any} */ w) => finite(w.pr) }
    ]);

    /**
     * The attributes that make a row a keyboard-reachable control.
     * @param {string} testid
     * @param {string} title
     * @param {() => void} open
     */
    function rowControl(testid, title, open) {
        return { 'data-testid': testid, role: 'button', tabindex: 0, title, onkeydown: (/** @type {KeyboardEvent} */ e) => onRowKey(e, open) };
    }

    function outcomeLabel(o) {
        if (o > 0) return playerA;
        if (o < 0) return playerB;
        return '—';
    }
</script>

<div class="corpus-tab">
    <section class="corpus-section" aria-labelledby="corpus-h2h-title">
        <h3 id="corpus-h2h-title">{$t('stats.corpus.h2hTitle')}</h3>
        <div class="corpus-controls">
            <label for="corpus-a">{$t('stats.corpus.playerA')}</label>
            <div class="corpus-player"><EntityAutocomplete id="corpus-a" variant="field" bind:value={playerA} items={players} label={(/** @type {string} */ n) => n} /></div>
            <label for="corpus-b">{$t('stats.corpus.playerB')}</label>
            <div class="corpus-player"><EntityAutocomplete id="corpus-b" variant="field" bind:value={playerB} items={players} label={(/** @type {string} */ n) => n} /></div>
            <button onclick={loadH2H} disabled={!playerA || !playerB || busy !== ''}>{$t('stats.corpus.compute')}</button>
        </div>
        {#if h2hError}
            <p class="corpus-error">{h2hError}</p>
        {:else if h2h}
            <p class="corpus-summary">
                {#if (h2h.matches ?? []).length > 0}
                    <button
                        type="button"
                        class="corpus-summary-link"
                        data-testid="corpus-h2h-summary"
                        title={$t('stats.corpus.openMatches')}
                        onclick={() => openMatchListInPanel(h2h.matches.map((m) => m.id))}
                        >{$t('stats.corpus.h2hSummary', { a: h2h.player_a, b: h2h.player_b, winsA: h2h.wins_a, winsB: h2h.wins_b, prA: fmt(h2h.pr_a), prB: fmt(h2h.pr_b) })}</button
                    >
                {:else}
                    {$t('stats.corpus.h2hSummary', { a: h2h.player_a, b: h2h.player_b, winsA: h2h.wins_a, winsB: h2h.wins_b, prA: fmt(h2h.pr_a), prB: fmt(h2h.pr_b) })}
                {/if}
            </p>
            {#if (h2h.matches ?? []).length === 0}
                <p class="corpus-empty">{$t('stats.corpus.h2hNone')}</p>
            {:else}
                <PanelTable
                    viewId="stats.corpus.h2h"
                    rows={h2h.matches}
                    columns={h2hColumns}
                    pointerRows
                    rowAttrs={(m) => rowControl('corpus-h2h-row', $t('stats.openMatch'), () => openMatchInPanel(m.id))}
                    onSelect={(m) => openMatchInPanel(m.id)}
                >
                    {#snippet cells(/** @type {any} */ m)}
                        <td>{m.date}</td>
                        <td class="align-right">{m.match_length}</td>
                        <td>{outcomeLabel(m.outcome)}</td>
                        <td class="align-right">{fmt(m.pr_a)}</td>
                        <td class="align-right">{fmt(m.pr_b)}</td>
                    {/snippet}
                </PanelTable>
            {/if}
        {/if}
    </section>

    <section class="corpus-section" aria-labelledby="corpus-windows-title">
        <h3 id="corpus-windows-title">{$t('stats.corpus.windowsTitle')}</h3>
        <div class="corpus-controls">
            <label for="corpus-months">{$t('stats.corpus.windowMonths')}</label>
            <select id="corpus-months" bind:value={months}>
                {#each [1, 3, 6, 12] as n (n)}
                    <option value={n}>{n}</option>
                {/each}
            </select>
            <button onclick={loadWindows} disabled={busy !== ''}>{$t('stats.corpus.compute')}</button>
        </div>
        {#if windowsError}
            <p class="corpus-error">{windowsError}</p>
        {:else if windows}
            {#if windows.length === 0}
                <p class="corpus-empty">{$t('stats.corpus.noData')}</p>
            {:else}
                <PanelTable
                    viewId="stats.corpus.windows"
                    rows={windows}
                    rowKey={(w) => w.from}
                    columns={windowColumns}
                    pointerRows
                    rowAttrs={(w) => rowControl('corpus-window-row', $t('stats.corpus.openPositions'), () => openPositions(windowFilter(w)))}
                    onSelect={(w) => openPositions(windowFilter(w))}
                >
                    {#snippet cells(/** @type {any} */ w)}
                        <td>{w.from}</td>
                        <td>{w.to}</td>
                        <td class="align-right"
                            >{#if w.num_matches > 0}<CountLink
                                    label={String(w.num_matches)}
                                    title={$t('stats.corpus.openMatches')}
                                    onclick={(/** @type {Event} */ e) => countClick(e, () => openCountedMatches(windowFilter(w)))}
                                />{:else}0{/if}</td
                        >
                        <td class="align-right"
                            ><CountLink
                                label={String(w.num_decisions)}
                                title={$t('stats.corpus.openPositions')}
                                onclick={(/** @type {Event} */ e) => countClick(e, () => openPositions(windowFilter(w)))}
                            /></td
                        >
                        <td class="align-right">{fmt(w.pr)}</td>
                    {/snippet}
                </PanelTable>
            {/if}
        {/if}
    </section>

    {#if busy}
        <p class="corpus-busy" role="status">{$t('stats.loading')}</p>
    {/if}
</div>

<style>
    .corpus-summary-link {
        all: unset;
        cursor: pointer;
        text-decoration: underline dotted;
    }
    .corpus-summary-link:focus-visible {
        outline: 2px solid var(--color-accent, currentColor);
    }
    .corpus-tab {
        display: flex;
        flex-direction: column;
        gap: 12px;
        padding: 8px;
    }
    .corpus-section h3 {
        margin: 0 0 4px;
        font-size: var(--font-size-base);
        font-weight: 600;
    }
    .corpus-player {
        width: 12em;
    }
    .corpus-controls {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 4px 8px;
        margin-bottom: 4px;
    }
    .corpus-error {
        color: var(--color-error, #b00020);
    }
    .corpus-empty,
    .corpus-busy {
        color: var(--color-text-muted, inherit);
    }
</style>
