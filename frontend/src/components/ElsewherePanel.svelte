<!--
  The « Autres bases » panel (ADR-0101, ADR-0102): the exchange post between open bases. For the
  position on screen, what every other open base knows of it — present or not, its analysis
  judged with this base's thresholds, its comments and their signer, the matches that played it,
  its nearest boards there — and the two copies, to that base or from it. The lot gathers rows of
  any base, to copy at once into the one named. A result opens its base on that position; « Show
  with x » chooses the board the swap key shows.
-->
<script>
    import { elsewhereStore, elsewherePinStore } from '../stores/elsewhereStore.js';
    import { positionStore } from '../stores/positionStore.js';
    import { analysisStore } from '../stores/analysisStore.js';
    import { databasePathStore } from '../stores/databaseStore.js';
    import { groupIdsStore, activeGroupId, groupName, groupColorIndex, tabGroupsStore } from '../stores/libraryGroupStore.js';
    import { transferBasketStore, lockedGroupsStore } from '../stores/transferBasketStore.js';
    import { loadElsewhere, openElsewhere, togglePin, analysisSummary, formatCost } from '../services/elsewhereService.js';
    import { copyPositionToPane, fetchPositionFrom, pourBasket, removeFromBasket, emptyBasket } from '../services/transferService.js';
    import { openDatabaseInNewGroup } from '../services/databaseService.js';
    import { libraryDrag } from '../utils/libraryDrag.js';
    import { renderPositionSVG } from '../services/diagramService.js';
    import { t } from '../i18n';

    const THUMB_WIDTH = 132;
    const THUMB_HEIGHT = 96;

    let others = $derived($groupIdsStore.filter((id) => id !== $activeGroupId));
    let hasPosition = $derived(!!$databasePathStore && !!$positionStore);

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
    let here = $derived(analysisSummary($analysisStore?.positionId === $positionStore?.id ? $analysisStore : null));

    /** @param {number} id */
    const baseName = (id) => groupName($tabGroupsStore.find((g) => g.id === id)?.path ?? '') || $t('libraryGroups.unnamed');

    /**
     * What a base holds of the position: absent, present, or present with an analysis that is
     * not this base's — the one a copy keeps there.
     * @param {any} g
     */
    function status(g) {
        if (!g.positionId) return 'absent';
        const there = analysisSummary(g.analysis);
        return here.kind !== 'none' && there.kind !== 'none' && there.best !== here.best ? 'otherAnalysis' : 'present';
    }

    /** @type {Record<string, string>} What a base holds of the position, by status(). */
    const STATUS_KEYS = { present: 'elsewhere.status.present', absent: 'elsewhere.status.absent', otherAnalysis: 'elsewhere.status.otherAnalysis' };
    /** @type {Record<string, string>} The kinds of row a lot holds. */
    const KIND_KEYS = {
        position: 'transfer.kind.position',
        match: 'transfer.kind.match',
        tournament: 'transfer.kind.tournament',
        collection: 'transfer.kind.collection',
        lesson: 'transfer.kind.lesson'
    };

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

    /** @param {number} dst */
    async function copyTo(dst) {
        const id = $positionStore?.id;
        if (!id) return;
        await copyPositionToPane(dst, { group: $activeGroupId, kind: 'position', id, label: `#${id}` });
        loadElsewhere($positionStore, { force: true });
    }

    /** @param {number} src @param {number} id */
    async function fetchFrom(src, id) {
        await fetchPositionFrom(src, id);
        loadElsewhere($positionStore, { force: true });
    }

    // The base the lot goes into: chosen in the panel, another open base by default.
    let lotTarget = $state(0);
    let lotDst = $derived($groupIdsStore.includes(lotTarget) ? lotTarget : (others[0] ?? 0));
</script>

<div class="elsewhere-panel" data-testid="elsewhere-panel">
    {#if others.length === 0}
        <div class="invite" data-testid="elsewhere-alone">
            <p>{$t('elsewhere.alone')}</p>
            <button type="button" class="action" data-testid="elsewhere-open-base" onclick={() => openDatabaseInNewGroup()}>{$t('elsewhere.openBase')}</button>
        </div>
    {:else}
        <p class="intro">{$t('elsewhere.intro')}</p>
        {#if !hasPosition}
            <p class="empty" data-testid="elsewhere-no-position">{$t('elsewhere.noPosition')}</p>
        {:else if $elsewhereStore.loading && !view}
            <p class="empty">{$t('common.loading')}…</p>
        {:else if $elsewhereStore.error}
            <p class="empty error">{$t('elsewhere.error')}</p>
        {/if}
        {#each hasPosition ? (view?.groups ?? []) : [] as g (g.group)}
            {@const summary = analysisSummary(g.analysis)}
            {@const st = status(g)}
            {@const name = groupName(g.path) || $t('libraryGroups.unnamed')}
            {@const locked = $lockedGroupsStore.has(g.group)}
            <section class="group" style="--group-color: var(--group-color-{groupColorIndex(g.group)})" data-testid="elsewhere-group">
                <header>
                    <span class="dot" aria-hidden="true"></span>
                    <span class="name" title={g.path}>{name}</span>
                    <span class="status {st}" data-testid="elsewhere-status">{$t(STATUS_KEYS[st])}</span>
                    {#if locked}<span class="muted" title={$t('transfer.lockGroup')}>🔒</span>{/if}
                    {#if g.positionId}
                        <button type="button" class="link" onclick={() => openElsewhere(g.group, g.positionId)} data-testid="elsewhere-open">{$t('elsewhere.open')}</button>
                    {/if}
                </header>
                <div class="exchange">
                    <button
                        type="button"
                        class="action"
                        data-testid="elsewhere-copy-to"
                        disabled={!$positionStore?.id || locked}
                        title={locked ? $t('transfer.locked') : !$positionStore?.id ? $t('transfer.basketDraft') : ''}
                        onclick={() => copyTo(g.group)}>{$t('elsewhere.copyTo', { base: name })}</button
                    >
                    <button
                        type="button"
                        class="action"
                        data-testid="elsewhere-fetch-from"
                        disabled={!g.positionId || $lockedGroupsStore.has($activeGroupId)}
                        title={$lockedGroupsStore.has($activeGroupId) ? $t('transfer.locked') : ''}
                        onclick={() => fetchFrom(g.group, g.positionId)}>{$t('elsewhere.fetchFrom', { base: name })}</button
                    >
                </div>
                {#if st === 'otherAnalysis'}
                    <p class="kept" data-testid="elsewhere-kept">{$t('elsewhere.keptThere')}</p>
                {/if}
                {#if g.positionId}
                    <details open>
                        <summary>{$t('elsewhere.block.analysis')}</summary>
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
                    </details>
                    {#if g.comments.length > 0}
                        <details open>
                            <summary>{$t('elsewhere.block.comments', { n: g.comments.length })}</summary>
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
                        </details>
                    {/if}
                    {#if g.plays.length > 0}
                        <details open>
                            <summary>{$t('elsewhere.block.matches', { n: g.matchesTotal })}</summary>
                            {#if g.playsTruncated}<div class="plays-head">{$t('elsewhere.playsTruncated', { shown: g.plays.length, total: g.matchesTotal })}</div>{/if}
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
                        </details>
                    {/if}
                {/if}
                {#if g.neighbours.length > 0}
                    <!-- Folded when the base holds the position itself, open when it does not. -->
                    <details open={!g.positionId} data-testid="elsewhere-neighbours">
                        <summary>{$t('elsewhere.block.neighbours', { n: g.neighbours.length })}</summary>
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
                    </details>
                {/if}
            </section>
        {/each}

        <!-- The lot: rows of any base, copied at once into the one named; a row dragged here joins it. -->
        <section class="lot lot-drop" data-testid="transfer-lot" data-library-drop="basket">
            <header>
                <span
                    class="name"
                    role="button"
                    tabindex="-1"
                    use:libraryDrag={() => ($transferBasketStore.length > 0 ? { kind: 'basket', label: $t('transfer.basketTitle', { n: $transferBasketStore.length }) } : null)}
                    >{$t('transfer.basketTitle', { n: $transferBasketStore.length })}</span
                >
                {#if $transferBasketStore.length > 0}
                    <button type="button" class="link" data-testid="transfer-lot-empty" onclick={() => emptyBasket()}>{$t('transfer.emptyBasket')}</button>
                {/if}
            </header>
            {#if $transferBasketStore.length === 0}
                <p class="muted">{$t('transfer.basketHint')}</p>
            {:else}
                <ul class="lot-items">
                    {#each $transferBasketStore as item (`${item.group}:${item.kind}:${item.id}`)}
                        <li data-testid="transfer-lot-item">
                            <span class="dot" style="--group-color: var(--group-color-{groupColorIndex(item.group)})" aria-hidden="true"></span>
                            <span class="what">{$t(KIND_KEYS[item.kind])} {item.label}</span>
                            <span class="muted">{baseName(item.group)}</span>
                            <button
                                type="button"
                                class="link"
                                data-testid="transfer-lot-remove"
                                title={$t('transfer.removeFromLot')}
                                aria-label={$t('transfer.removeFromLot')}
                                onclick={() => removeFromBasket(item)}>×</button
                            >
                        </li>
                    {/each}
                </ul>
                <div class="exchange">
                    <label class="muted" for="lot-dst">{$t('transfer.lotInto')}</label>
                    <select id="lot-dst" data-testid="transfer-lot-dst" value={lotDst} onchange={(e) => (lotTarget = Number(/** @type {HTMLSelectElement} */ (e.currentTarget).value))}>
                        {#each $tabGroupsStore as group (group.id)}
                            <option value={group.id}>{group.name || $t('libraryGroups.unnamed')}</option>
                        {/each}
                    </select>
                    <button type="button" class="action" data-testid="transfer-lot-copy" disabled={!lotDst || $lockedGroupsStore.has(lotDst)} onclick={() => pourBasket(lotDst)}
                        >{$t('transfer.pourInto', { name: baseName(lotDst) })}</button
                    >
                </div>
            {/if}
        </section>

        <footer class="judge">
            {#if settings}
                <span>{$t('elsewhere.judgedWith', { error: formatCost(settings.errorThresholdMP), blunder: formatCost(settings.blunderThresholdMP) })}</span>
            {/if}
            <span class="hint">{$t('elsewhere.swapHint')}</span>
            <button type="button" class="refresh" onclick={() => loadElsewhere($positionStore, { force: true })} title={$t('elsewhere.refresh')}>⟳ {$t('elsewhere.refresh')}</button>
        </footer>
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
    .intro {
        margin: 0 0 8px;
        color: var(--color-text-muted);
    }
    .invite {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: 8px;
    }
    .status {
        font-size: var(--font-size-small);
        padding: 0 6px;
        border-radius: 8px;
        border: 1px solid var(--color-border);
    }
    .status.present {
        color: var(--color-success, var(--color-text));
    }
    .status.otherAnalysis {
        color: var(--color-warning, var(--color-text));
    }
    .status.absent {
        color: var(--color-text-muted);
    }
    .exchange {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        align-items: center;
        margin: 4px 0;
    }
    .action {
        padding: 2px 8px;
        font-size: var(--font-size-small);
        cursor: pointer;
    }
    .action:disabled {
        cursor: default;
        opacity: 0.5;
    }
    .kept {
        margin: 2px 0;
        color: var(--color-warning, var(--color-text-muted));
    }
    details summary {
        cursor: pointer;
        color: var(--color-text-muted);
        margin-top: 4px;
    }
    .lot {
        margin-top: 10px;
        padding-top: 6px;
        border-top: 1px solid var(--color-border);
    }
    .lot header {
        display: flex;
        gap: 8px;
        align-items: baseline;
    }
    .lot-items {
        list-style: none;
        margin: 4px 0;
        padding: 0;
    }
    .lot-items li {
        display: flex;
        gap: 6px;
        align-items: center;
    }
    .lot-items .what {
        flex: 1;
    }
    footer.judge {
        margin-top: 10px;
    }
</style>
