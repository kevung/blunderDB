<script>
    // Vues de corpus : face-à-face de deux joueurs, PR par fenêtre
    // calendaire, classement. Chaque vue se calcule à la demande — ce sont
    // des agrégats sur toute la base, trop chers pour suivre chaque frappe.
    import { HeadToHead, PRByWindow, PlayerRanking, GetAllPlayerNames } from '../../../wailsjs/go/database/Database.js';
    import { t } from '../../i18n/index.js';

    /** @type {{ filter: object }} */
    let { filter } = $props();

    let players = $state([]);
    let playerA = $state('');
    let playerB = $state('');
    let h2h = $state(null);
    let h2hError = $state('');

    let months = $state(3);
    let windows = $state(null);
    let windowsError = $state('');

    let minDecisions = $state(500);
    let ranking = $state(null);
    let rankingError = $state('');

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
                windows = (await PRByWindow(filter, Number(months))) ?? [];
            } catch (err) {
                windows = null;
                windowsError = err?.message ?? String(err);
            }
        });
    }

    function loadRanking() {
        return run('ranking', async () => {
            rankingError = '';
            try {
                ranking = (await PlayerRanking(filter, Math.max(0, Number(minDecisions) || 0))) ?? [];
            } catch (err) {
                ranking = null;
                rankingError = err?.message ?? String(err);
            }
        });
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
            <input id="corpus-a" list="corpus-players" bind:value={playerA} />
            <label for="corpus-b">{$t('stats.corpus.playerB')}</label>
            <input id="corpus-b" list="corpus-players" bind:value={playerB} />
            <datalist id="corpus-players">
                {#each players as name (name)}
                    <option value={name}></option>
                {/each}
            </datalist>
            <button onclick={loadH2H} disabled={!playerA || !playerB || busy !== ''}>{$t('stats.corpus.compute')}</button>
        </div>
        {#if h2hError}
            <p class="corpus-error">{h2hError}</p>
        {:else if h2h}
            <p class="corpus-summary">
                {$t('stats.corpus.h2hSummary', { a: h2h.player_a, b: h2h.player_b, winsA: h2h.wins_a, winsB: h2h.wins_b, prA: fmt(h2h.pr_a), prB: fmt(h2h.pr_b) })}
            </p>
            {#if (h2h.matches ?? []).length === 0}
                <p class="corpus-empty">{$t('stats.corpus.h2hNone')}</p>
            {:else}
                <table class="corpus-table">
                    <thead>
                        <tr>
                            <th>{$t('stats.corpus.date')}</th>
                            <th>{$t('stats.corpus.length')}</th>
                            <th>{$t('stats.corpus.winner')}</th>
                            <th>{$t('stats.corpus.prOf', { name: h2h.player_a })}</th>
                            <th>{$t('stats.corpus.prOf', { name: h2h.player_b })}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each h2h.matches as m (m.id)}
                            <tr>
                                <td>{m.date}</td>
                                <td>{m.match_length}</td>
                                <td>{outcomeLabel(m.outcome)}</td>
                                <td>{fmt(m.pr_a)}</td>
                                <td>{fmt(m.pr_b)}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
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
                <table class="corpus-table">
                    <thead>
                        <tr>
                            <th>{$t('stats.corpus.from')}</th>
                            <th>{$t('stats.corpus.to')}</th>
                            <th>{$t('stats.corpus.matches')}</th>
                            <th>{$t('stats.corpus.decisions')}</th>
                            <th>PR</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each windows as w (w.from)}
                            <tr>
                                <td>{w.from}</td>
                                <td>{w.to}</td>
                                <td>{w.num_matches}</td>
                                <td>{w.num_decisions}</td>
                                <td>{fmt(w.pr)}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            {/if}
        {/if}
    </section>

    <section class="corpus-section" aria-labelledby="corpus-ranking-title">
        <h3 id="corpus-ranking-title">{$t('stats.corpus.rankingTitle')}</h3>
        <div class="corpus-controls">
            <label for="corpus-min">{$t('stats.corpus.minDecisions')}</label>
            <input id="corpus-min" type="number" min="0" bind:value={minDecisions} />
            <button onclick={loadRanking} disabled={busy !== ''}>{$t('stats.corpus.compute')}</button>
        </div>
        {#if rankingError}
            <p class="corpus-error">{rankingError}</p>
        {:else if ranking}
            {#if ranking.length === 0}
                <p class="corpus-empty">{$t('stats.corpus.noData')}</p>
            {:else}
                <table class="corpus-table">
                    <thead>
                        <tr>
                            <th>#</th>
                            <th>{$t('stats.corpus.player')}</th>
                            <th>{$t('stats.corpus.matches')}</th>
                            <th>{$t('stats.corpus.record')}</th>
                            <th>{$t('stats.corpus.decisions')}</th>
                            <th>PR</th>
                            <th>{$t('stats.corpus.prChecker')}</th>
                            <th>{$t('stats.corpus.prCube')}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each ranking as r (r.name)}
                            <tr>
                                <td>{r.rank}</td>
                                <td>{r.name}</td>
                                <td>{r.matches}</td>
                                <td>{r.wins}–{r.losses}</td>
                                <td>{r.decisions}</td>
                                <td>{fmt(r.pr)}</td>
                                <td>{fmt(r.pr_checker)}</td>
                                <td>{fmt(r.pr_cube)}</td>
                            </tr>
                        {/each}
                    </tbody>
                </table>
            {/if}
        {/if}
    </section>
    {#if busy}
        <p class="corpus-busy" role="status">{$t('stats.loading')}</p>
    {/if}
</div>

<style>
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
    .corpus-controls {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 4px 8px;
        margin-bottom: 4px;
    }
    .corpus-controls input[type='number'] {
        width: 6em;
    }
    .corpus-table {
        border-collapse: collapse;
        font-size: var(--font-size-small);
    }
    .corpus-table th,
    .corpus-table td {
        padding: 2px 8px;
        border-bottom: 1px solid var(--color-border);
        text-align: right;
    }
    .corpus-table th:nth-child(2),
    .corpus-table td:nth-child(2) {
        text-align: left;
    }
    .corpus-error {
        color: var(--color-error, #b00020);
    }
    .corpus-empty,
    .corpus-busy {
        color: var(--color-text-muted, inherit);
    }
</style>
