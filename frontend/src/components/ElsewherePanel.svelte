<!--
  The « Ailleurs » view (ADR-0101): for the board on screen, what every other open library group
  knows of it — analysis judged with this library's thresholds, comments and their signer, the
  matches that played it, its nearest boards there. A result opens its group on that position;
  a pin chooses the board the swap key (x) shows.
-->
<script>
    import { elsewhereStore, elsewherePinStore } from '../stores/elsewhereStore.js';
    import { positionStore } from '../stores/positionStore.js';
    import { databasePathStore } from '../stores/databaseStore.js';
    import { groupIdsStore, activeGroupId, groupName, groupColorIndex } from '../stores/libraryGroupStore.js';
    import { loadElsewhere, openElsewhere, togglePin, analysisSummary, formatCost } from '../services/elsewhereService.js';
    import { renderPositionSVG } from '../services/diagramService.js';
    import { t } from '../i18n';

    const THUMB_WIDTH = 132;
    const THUMB_HEIGHT = 96;

    let others = $derived($groupIdsStore.filter((id) => id !== $activeGroupId));

    // Read again when the board, the library or the set of groups moves; the service keeps one
    // answer per board, so coming back to the tab reads nothing.
    // A short wait, so a board being edited or browsed fast reads once it settles.
    $effect(() => {
        void $databasePathStore;
        void others.length;
        // Emptied by a library change (libraryScope.js): read again.
        void ($elsewhereStore.key === '');
        const position = $positionStore;
        const timer = setTimeout(() => loadElsewhere(position), 150);
        return () => clearTimeout(timer);
    });

    let view = $derived($elsewhereStore.view);
    let settings = $derived(view?.judge?.settings);

    /** @type {Record<string, string>} The verdicts of storage.Judge, by their key. */
    const VERDICT_KEYS = {
        good: 'elsewhere.verdict.good',
        error: 'elsewhere.verdict.error',
        blunder: 'elsewhere.verdict.blunder',
        unscored: 'elsewhere.verdict.unscored',
        'other-met': 'elsewhere.verdict.otherMet'
    };
    /** @param {string} verdict */
    const verdictLabel = (verdict) => $t(VERDICT_KEYS[verdict] ?? VERDICT_KEYS.unscored);

    /** @param {number} group @param {number} positionId */
    const isPinned = (group, positionId) => $elsewherePinStore?.group === group && $elsewherePinStore?.positionId === positionId;

    /** @param {any} position */
    const thumb = (position) => renderPositionSVG(position, { width: THUMB_WIDTH, height: THUMB_HEIGHT, showPipcount: false });
</script>

<div class="elsewhere-panel" data-testid="elsewhere-panel">
    {#if others.length === 0}
        <p class="empty" data-testid="elsewhere-alone">{$t('elsewhere.alone')}</p>
    {:else}
        <div class="judge">
            {#if settings}
                <span>{$t('elsewhere.judgedWith', { error: formatCost(settings.errorThresholdMP), blunder: formatCost(settings.blunderThresholdMP) })}</span>
            {/if}
            <span class="hint">{$t('elsewhere.swapHint')}</span>
            <button type="button" class="refresh" onclick={() => loadElsewhere($positionStore, { force: true })} title={$t('elsewhere.refresh')}>⟳</button>
        </div>
        {#if $elsewhereStore.loading && !view}
            <p class="empty">{$t('common.loading')}…</p>
        {:else if $elsewhereStore.error}
            <p class="empty error">{$t('elsewhere.error')}</p>
        {/if}
        {#each view?.groups ?? [] as g (g.group)}
            {@const summary = analysisSummary(g.analysis)}
            <section class="group" style="--group-color: var(--group-color-{groupColorIndex(g.group)})" data-testid="elsewhere-group">
                <header>
                    <span class="dot" aria-hidden="true"></span>
                    <span class="name" title={g.path}>{groupName(g.path) || $t('libraryGroups.unnamed')}</span>
                    {#if g.positionId}
                        <button type="button" class="link" onclick={() => openElsewhere(g.group, g.positionId)} data-testid="elsewhere-open">{$t('elsewhere.open')}</button>
                    {/if}
                </header>
                {#if !g.positionId}
                    <p class="absent">{$t('elsewhere.absent')}</p>
                {:else}
                    <div class="line">
                        {#if summary.kind === 'none'}
                            <span class="muted">{$t('elsewhere.notAnalysed')}</span>
                        {:else}
                            <span>{$t(summary.kind === 'cube' ? 'elsewhere.bestCube' : 'elsewhere.bestMove')} <strong>{summary.best}</strong></span>
                            {#if summary.engine}<span class="muted">· {summary.engine}</span>{/if}
                        {/if}
                        {#if g.otherMet}
                            <span class="badge" data-testid="elsewhere-other-met" title={$t('elsewhere.otherMetTitle')}
                                >{$t('elsewhere.otherMet', { name: g.metName || $t('elsewhere.builtinMet') })}</span
                            >
                        {/if}
                    </div>
                    {#each g.comments as c (c.id)}
                        <div class="comment">
                            <div class="who">
                                {#if c.signer?.fingerprint}
                                    <span class:unverified={!c.signer.verified}
                                        >{c.signer.verified ? '✓' : '⚠'}
                                        {$t('comment.signedBy', { name: c.signer.name || c.signer.fingerprint })}{c.signer.verified ? '' : ' — ' + $t('comment.signerUnverified')}</span
                                    >
                                {:else if c.signer}
                                    <span class="unverified">⚠ {$t('comment.signatureUnreadable')}</span>
                                {:else if c.author}
                                    <span>{c.author}</span>
                                {/if}
                            </div>
                            <div class="text">{c.text}</div>
                        </div>
                    {/each}
                    {#if g.plays.length > 0}
                        <div class="plays-head">
                            {g.playsTruncated ? $t('elsewhere.playsTruncated', { shown: g.plays.length, total: g.matchesTotal }) : $t('elsewhere.plays', { count: g.matchesTotal })}
                        </div>
                        <ul class="plays">
                            {#each g.plays as p, i (i)}
                                <li>
                                    <button type="button" class="row" onclick={() => openElsewhere(g.group, g.positionId)} title={$t('elsewhere.open')}>
                                        <span class="match">{p.player1} – {p.player2}{p.event ? ' · ' + p.event : ''}{p.matchDate ? ' · ' + p.matchDate : ''}</span>
                                        <span class="move">#{p.moveNumber} {p.played || '—'}</span>
                                        <span class="verdict {p.verdict}">{verdictLabel(p.verdict)}{p.errorMP != null ? ' ' + formatCost(p.errorMP) : ''}</span>
                                    </button>
                                </li>
                            {/each}
                        </ul>
                    {/if}
                {/if}
                {#if g.neighbours.length > 0}
                    <div class="plays-head">{$t('elsewhere.neighbours')}</div>
                    <ul class="neighbours">
                        {#each g.neighbours as n (n.zobrist)}
                            <li class:pinned={isPinned(g.group, n.position.id)}>
                                <button type="button" class="thumb" onclick={() => openElsewhere(g.group, n.position.id)} title={$t('elsewhere.open')}>
                                    <!-- eslint-disable-next-line svelte/no-at-html-tags -- renderPositionSVG draws from the position's numbers only -->
                                    {@html thumb(n.position)}
                                </button>
                                <span class="dist">{$t('elsewhere.distance', { distance: n.distance })}</span>
                                <button
                                    type="button"
                                    class="link"
                                    data-testid="elsewhere-pin"
                                    aria-pressed={isPinned(g.group, n.position.id)}
                                    onclick={() => togglePin({ group: g.group, path: g.path, positionId: n.position.id, position: n.position, distance: n.distance })}
                                    >{isPinned(g.group, n.position.id) ? $t('elsewhere.unpin') : $t('elsewhere.pin')}</button
                                >
                            </li>
                        {/each}
                    </ul>
                {/if}
            </section>
        {/each}
    {/if}
</div>

<style>
    .elsewhere-panel {
        height: 100%;
        overflow-y: auto;
        padding: 6px 10px;
        box-sizing: border-box;
        font-size: var(--font-size-small);
        color: var(--color-text);
    }
    .judge {
        display: flex;
        flex-wrap: wrap;
        gap: 4px 12px;
        align-items: center;
        color: var(--color-text-muted);
        margin-bottom: 6px;
    }
    .hint {
        flex: 1;
    }
    .refresh,
    .link {
        background: none;
        border: none;
        color: var(--color-primary);
        cursor: pointer;
        padding: 0 4px;
        font-family: inherit;
        font-size: inherit;
    }
    .empty,
    .absent,
    .muted {
        color: var(--color-text-muted);
    }
    .error {
        color: var(--color-danger);
    }
    .group {
        border-left: 3px solid var(--group-color);
        padding: 4px 0 6px 8px;
        margin-bottom: 8px;
    }
    .group header {
        display: flex;
        align-items: center;
        gap: 6px;
        font-weight: 600;
    }
    .dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: var(--group-color);
    }
    .name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        min-width: 0;
    }
    .line {
        display: flex;
        flex-wrap: wrap;
        gap: 4px 8px;
        align-items: baseline;
        margin: 3px 0;
    }
    .badge {
        border: 1px solid var(--color-border);
        border-radius: 999px;
        padding: 0 6px;
        color: var(--color-text-muted);
    }
    .comment {
        margin: 3px 0;
        padding: 3px 6px;
        background: var(--color-surface-alt);
        border-radius: 4px;
    }
    .who {
        color: var(--color-text-muted);
    }
    .unverified {
        color: var(--color-danger);
    }
    .text {
        white-space: pre-wrap;
        overflow-wrap: anywhere;
    }
    .plays-head {
        margin-top: 4px;
        color: var(--color-text-muted);
    }
    .plays,
    .neighbours {
        list-style: none;
        margin: 2px 0;
        padding: 0;
    }
    .row {
        display: grid;
        grid-template-columns: minmax(0, 1fr) auto auto;
        gap: 8px;
        width: 100%;
        background: none;
        border: none;
        padding: 1px 0;
        text-align: left;
        font-family: inherit;
        font-size: inherit;
        color: inherit;
        cursor: pointer;
    }
    .row:hover {
        background: var(--color-surface-alt);
    }
    .match {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .verdict.error {
        color: var(--color-warning, var(--color-danger));
    }
    .verdict.blunder {
        color: var(--color-danger);
    }
    .verdict.other-met,
    .verdict.unscored {
        color: var(--color-text-muted);
    }
    .neighbours {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
    }
    .neighbours li {
        display: flex;
        flex-direction: column;
        align-items: center;
        border: 1px solid transparent;
        border-radius: 4px;
    }
    .neighbours li.pinned {
        border-color: var(--group-color);
    }
    .thumb {
        background: none;
        border: none;
        padding: 0;
        cursor: pointer;
        line-height: 0;
    }
    .dist {
        color: var(--color-text-muted);
    }
</style>
