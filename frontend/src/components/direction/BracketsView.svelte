<script>
    /*
     * La vue Arbres (fonctionnel.md §5.6), dessinée ici et non par le `render` du moteur :
     * traduite et cliquable. Un avertissement se voit sur la place concernée. Une phase
     * suisse avant bascule montre le tableau des vies et les adversaires rencontrés.
     */
    import { t } from '../../i18n';
    import { renderLabel, renderSectionName } from './labels.js';
    import { BOX_W, BOX_H, layoutSection, crossTable } from './bracketLayout.js';
    import ResultCard from './ResultCard.svelte';
    import CorrectionPanel from './CorrectionPanel.svelte';
    import ContextMenu from '../ContextMenu.svelte';
    import { matchMenu } from '../../services/directionMenus.js';
    import { menuRequest, isMenuKey } from '../../services/contextMenuTrigger.js';

    /* `cells` donne la table d'un match en cours ; les gestes sont ceux de la grille des
       tables : cliquer une place, c'est ouvrir la même fiche que sur la case. */
    /**
     * @type {{
     *     phases?: any[], cells?: any[], busy?: boolean,
     *     onHistory?: (name: string) => void,
     *     onResult?: (matchId: string, winner: string, scoreA: number, scoreB: number, note: string) => void,
     *     onForfeit?: (matchId: string, winner: string, note: string) => void,
     *     onMove?: (matchId: string, table: number) => void,
     *     onCancel?: (matchId: string) => void,
     *     onCorrect?: (matchId: string, winner: string, scoreA: number, scoreB: number) => void
     * }}
     */
    let { phases = [], cells = [], busy = false, onResult = () => {}, onForfeit = () => {}, onMove = () => {}, onCancel = () => {}, onCorrect = () => {}, onHistory = undefined } = $props();

    let menu = $state(/** @type {import('../../services/contextMenuTrigger.js').MenuRequest | null} */ (null));
    /** La fiche ouverte par le menu sur « changer de table » s'ouvre sur son champ. */
    let openInMove = $state(false);

    /** Ouvre la fiche sur la place, sans la refermer si elle l'est déjà. @param {string} sec @param {BracketMatch} m @param {boolean} move */
    function show(sec, m, move) {
        if (!canOpen(m)) return;
        openInMove = move;
        openKey = placeKey(sec, m);
    }

    /** @param {MouseEvent | KeyboardEvent} ev @param {string} sec @param {BracketMatch} m */
    function onPlaceMenu(ev, sec, m) {
        const req = menuRequest(ev, () =>
            matchMenu((k, p) => $t(k, p), m, {
                openResult: () => show(sec, m, false),
                openMove: () => show(sec, m, true),
                onForfeit,
                onCancel,
                onCorrect: () => show(sec, m, false),
                onHistory
            })
        );
        if (req) menu = req;
    }

    /** La place dont la fiche est ouverte, par clé de section et de place. */
    let openKey = $state('');

    /* La phase courante est dépliée ; les précédentes sont repliées mais consultables — un
       directeur relit le tableau des vies pendant que le tableau final tourne. */
    /** @typedef {import('../../../wailsjs/go/models').database.BracketPhase} BracketPhase */
    /** @typedef {import('../../../wailsjs/go/models').database.BracketSection} BracketSection */
    /** @typedef {import('../../../wailsjs/go/models').database.BracketMatch} BracketMatch */

    let openPhases = $state(/** @type {Record<number, boolean> | null} */ (null));

    // Pas `state` : svelte-check lirait alors la rune `$state` ci-dessus comme un abonnement au
    // store `state` — le piège qui avait fait planter DirectionSettings.
    const unfolded = $derived.by(() => {
        if (openPhases) return openPhases;
        /** @type {Record<number, boolean>} */
        const s = {};
        for (const p of phases) s[p.index] = p.current;
        return s;
    });

    /** @param {number} i */
    function toggle(i) {
        const next = { ...unfolded };
        next[i] = !next[i];
        openPhases = next;
    }

    /** @param {BracketPhase} p */
    function phaseTitle(p) {
        return p.name || $t(`direction.format.${p.kind}`);
    }

    /** @param {BracketSection} s */
    function isPool(s) {
        return s.kind === 'poule';
    }

    /** @param {string} sec @param {BracketMatch} m */
    const placeKey = (sec, m) => `${sec}/${m.key}`;

    /* Une fiche n'existe que pour un match en cours, ou fini entre deux joueurs : un forfait à un
       seul joueur n'a rien à corriger ici. */
    /** @param {BracketMatch} m */
    const canOpen = (m) => !!m.matchId && (m.running || (m.done && !!m.a && !!m.b));

    /** @param {string} sec @param {BracketMatch} m */
    function open(sec, m) {
        if (!canOpen(m)) return;
        openInMove = false;
        openKey = openKey === placeKey(sec, m) ? '' : placeKey(sec, m);
    }

    /** @param {KeyboardEvent} e @param {string} sec @param {BracketMatch} m */
    function onPlaceKey(e, sec, m) {
        if (isMenuKey(e)) {
            onPlaceMenu(e, sec, m);
            return;
        }
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            open(sec, m);
        }
    }

    /** @param {BracketMatch} m */
    function runningCell(m) {
        const c = cells.find((x) => x.matchId === m.matchId);
        return {
            table: c?.table,
            noTable: c ? !!c.noTable : true,
            length: m.length,
            matchId: /** @type {string} */ (m.matchId),
            a: m.a || '',
            b: m.b || '',
            aName: m.aName,
            bName: m.bName
        };
    }

    /** @param {string} name */
    function short(name) {
        return name.length > 22 ? `${name.slice(0, 21)}…` : name;
    }

    /** Le score d'un camp ; ailleurs que dans un score, l'issue (forfait, en cours). */
    /** @param {BracketMatch} m @param {number} side */
    function sideText(m, side) {
        if (m.skipped || m.walkover) return side === 0 ? outcome(m) : '';
        if (m.done && (m.scoreA || 0) + (m.scoreB || 0) > 0) return String(side === 0 ? m.scoreA || 0 : m.scoreB || 0);
        if (m.running && side === 0) return '●';
        return '';
    }

    /** @param {BracketMatch} m */
    function outcome(m) {
        if (m.skipped) return $t('direction.bracket.skipped');
        if (m.walkover) return $t('direction.bracket.walkover');
        // Un score nul est omis par le backend (`omitempty`) : un 7–0 arrive sans `scoreB`.
        if (m.done && (m.scoreA || 0) + (m.scoreB || 0) > 0) return `${m.scoreA || 0}–${m.scoreB || 0}`;
        if (m.running) return $t('direction.bracket.running');
        return '';
    }
</script>

{#snippet card(/** @type {BracketMatch} */ m)}
    <div class="card-layer" data-testid="bracket-card">
        {#if m.running}
            <ResultCard cell={runningCell(m)} {busy} startInMove={openInMove} onClose={() => (openKey = '')} {onResult} {onForfeit} {onMove} {onCancel} />
        {:else if m.done && m.a && m.b}
            <div class="correction">
                <div class="correction-head">
                    <span>{$t('direction.bracket.correct')}</span>
                    <button type="button" class="close" title={$t('common.close')} onclick={() => (openKey = '')}>×</button>
                </div>
                <CorrectionPanel
                    a={m.a}
                    b={m.b}
                    aName={m.aName || m.a}
                    bName={m.bName || m.b}
                    {busy}
                    testid="bracket-correct"
                    onPick={(w, sa, sb) => {
                        onCorrect(/** @type {string} */ (m.matchId), w, sa, sb);
                        openKey = '';
                    }}
                />
            </div>
        {/if}
    </div>
{/snippet}

{#snippet graph(/** @type {string} */ sec, /** @type {ReturnType<typeof layoutSection>} */ g, /** @type {boolean} */ ghost)}
    <div class="graph-wrap">
        <svg class="graph" class:ghost width={g.width} height={g.height} viewBox="0 0 {g.width} {g.height}" role="group" data-testid={ghost ? 'bracket-skeleton' : 'bracket-graph'}>
            {#each g.edges as e (e.key)}
                <path class="edge" class:loser={e.loser} d={e.d} fill="none" />
            {/each}
            {#each g.nodes as n (n.m.key)}
                {@const m = n.m}
                {#if ghost}
                    <rect class="box" x={n.x} y={n.y} width={BOX_W} height={BOX_H} rx="4" />
                {:else}
                    <g
                        class="place"
                        class:done={m.done}
                        class:running={m.running}
                        class:flagged={m.flagged}
                        class:skipped={m.skipped}
                        class:idle={!m.matchId}
                        role="button"
                        tabindex={m.matchId ? 0 : -1}
                        aria-disabled={!m.matchId}
                        aria-label={`${m.aName || $t('direction.bracket.pending')} – ${m.bName || $t('direction.bracket.pending')}`}
                        data-testid="bracket-place"
                        transform="translate({n.x},{n.y})"
                        onclick={() => open(sec, m)}
                        oncontextmenu={(e) => onPlaceMenu(e, sec, m)}
                        onkeydown={(e) => onPlaceKey(e, sec, m)}
                    >
                        <title>{renderLabel($t, m.label)}</title>
                        <rect class="box" width={BOX_W} height={BOX_H} rx="4" />
                        <line class="sep" x1="0" y1={BOX_H / 2} x2={BOX_W} y2={BOX_H / 2} />
                        <text class="side" class:winner={m.done && m.winner === m.a} class:pending={!m.aName} x="6" y={BOX_H / 4 + 4}>{short(m.aName || $t('direction.bracket.pending'))}</text>
                        <text class="side" class:winner={m.done && m.winner === m.b} class:pending={!m.bName} x="6" y={(3 * BOX_H) / 4 + 4}>{short(m.bName || $t('direction.bracket.pending'))}</text>
                        <text class="score" x={BOX_W - 6} y={BOX_H / 4 + 4} text-anchor="end">{sideText(m, 0)}</text>
                        <text class="score" x={BOX_W - 6} y={(3 * BOX_H) / 4 + 4} text-anchor="end">{sideText(m, 1)}</text>
                    </g>
                {/if}
            {/each}
        </svg>
        {#each g.nodes as n (n.m.key)}
            {#if !ghost && openKey === placeKey(sec, n.m)}
                <div class="card-anchor" style="left:{n.x}px; top:{n.y + BOX_H}px">{@render card(n.m)}</div>
            {/if}
        {/each}
    </div>
{/snippet}

<div class="brackets">
    {#each phases as p (p.index)}
        <section class="phase">
            <button type="button" class="phase-head" onclick={() => toggle(p.index)}>
                <span class="chevron">{unfolded[p.index] ? '▾' : '▸'}</span>
                <span class="phase-name">{phaseTitle(p)}</span>
                {#if p.current}
                    <span class="badge">{$t('direction.bracket.current')}</span>
                {/if}
                {#if !p.drawn && p.sections.length === 0 && !p.lives}
                    <span class="muted">{$t('direction.bracket.notDrawn')}</span>
                {/if}
            </button>

            {#if unfolded[p.index]}
                {#if p.lives && p.lives.length}
                    <table class="lives">
                        <thead>
                            <tr>
                                <th>{$t('direction.players.name')}</th>
                                <th>{$t('direction.players.lives')}</th>
                                <th>{$t('direction.players.record')}</th>
                                <th>{$t('direction.bracket.byes')}</th>
                                <th>{$t('direction.players.opponents')}</th>
                            </tr>
                        </thead>
                        <tbody>
                            {#each p.lives as r (r.id)}
                                <tr class:out={r.out}>
                                    <td class="name">{r.name}</td>
                                    <td>{r.lives}</td>
                                    <td>{r.wins}–{r.losses}</td>
                                    <td>{r.byes || ''}</td>
                                    <td class="opponents" title={(r.opponents || []).join(', ')}>{(r.opponents || []).join(', ')}</td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                {/if}

                {#if p.sections.length}
                    <div class="sections">
                        {#each p.sections as s (s.name)}
                            <div class="section">
                                <h4>{renderSectionName($t, s.name)}</h4>
                                {#if s.players && s.players.length}
                                    <!-- Barrage sans graphe : qui reste, pour combien de places. -->
                                    <p class="muted">
                                        {$t('direction.bracket.playoff', {
                                            players: s.players.join(', '),
                                            spots: s.spots
                                        })}
                                    </p>
                                {:else if isPool(s)}
                                    {@const ct = crossTable(s)}
                                    <table class="cross" data-testid="bracket-pool">
                                        <thead>
                                            <tr>
                                                <th></th>
                                                {#each ct.players as pl (pl.id)}<th class="vs" title={pl.name}>{short(pl.name)}</th>{/each}
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {#each ct.players as row, i (row.id)}
                                                <tr>
                                                    <th class="name" scope="row">{row.name}</th>
                                                    {#each ct.players as col, j (col.id)}
                                                        {@const c = ct.cell(i, j)}
                                                        <td class:self={i === j} class:done={c?.m.done} class:running={c?.m.running} class:flagged={c?.m.flagged}>
                                                            {#if c && i !== j}
                                                                <button
                                                                    type="button"
                                                                    class="pool-cell"
                                                                    disabled={!c.m.matchId}
                                                                    data-testid="bracket-place"
                                                                    title={renderLabel($t, c.m.label)}
                                                                    onclick={() => open(s.name, c.m)}
                                                                    oncontextmenu={(e) => onPlaceMenu(e, s.name, c.m)}
                                                                    onkeydown={(e) => onPlaceMenu(e, s.name, c.m)}
                                                                >
                                                                    {#if c.m.done}<span class:winner={c.m.winner === row.id}>{c.own}–{c.other}</span>{:else if c.m.running}●{:else}·{/if}
                                                                </button>
                                                                {#if openKey === placeKey(s.name, c.m)}
                                                                    <div class="card-anchor" style="left:0; top:100%">{@render card(c.m)}</div>
                                                                {/if}
                                                            {/if}
                                                        </td>
                                                    {/each}
                                                </tr>
                                            {/each}
                                        </tbody>
                                    </table>
                                {:else}
                                    {@render graph(s.name, layoutSection(s), !p.drawn)}
                                {/if}
                            </div>
                        {/each}
                    </div>
                {/if}
            {/if}
        </section>
    {/each}
    {#if phases.length === 0}
        <p class="muted">{$t('direction.bracket.none')}</p>
    {/if}
</div>

{#if menu}
    <ContextMenu x={menu.x} y={menu.y} items={menu.items} onClose={() => (menu = null)} />
{/if}

<style>
    .brackets {
        display: flex;
        flex-direction: column;
        gap: var(--space-2);
        padding: var(--space-2);
        min-width: 0;
    }

    .phase-head {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        width: 100%;
        text-align: left;
        padding: var(--space-1);
        border: none;
        border-bottom: 1px solid var(--color-border);
        background: transparent;
        color: var(--color-text);
        cursor: pointer;
    }

    .phase-name {
        font-weight: 600;
    }

    .chevron {
        color: var(--color-text-muted);
    }

    .badge {
        font-size: var(--font-size-small);
        color: var(--color-primary);
    }

    .muted {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    /* Le principal et la consolante côte à côte, et l'ensemble défile horizontalement plutôt
       que de déborder la page — un tableau de 64 est large. */
    .sections {
        display: flex;
        gap: var(--space-4);
        overflow-x: auto;
        padding: var(--space-1) 0;
    }

    .section h4 {
        margin: 0 0 var(--space-1);
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text-muted);
    }

    .graph-wrap {
        position: relative;
    }

    .graph {
        display: block;
    }

    .edge {
        stroke: var(--color-border);
        stroke-width: 1.5;
    }

    .edge.loser {
        stroke-dasharray: 4 3;
    }

    .box {
        fill: var(--color-surface);
        stroke: var(--color-border);
    }

    .ghost .box {
        fill: var(--color-surface-alt);
        stroke-dasharray: 3 3;
        opacity: 0.7;
    }

    .place {
        cursor: pointer;
        font-size: var(--font-size-small);
        color: var(--color-text);
    }

    .place.idle {
        cursor: default;
    }

    .place:focus-visible {
        outline: none;
    }

    .place:focus-visible .box {
        stroke: var(--color-primary);
        stroke-width: 2;
    }

    .place .sep {
        stroke: var(--color-border);
    }

    .place text {
        fill: var(--color-text);
        font-size: var(--font-size-small);
    }

    .place text.pending,
    .place text.score {
        fill: var(--color-text-muted);
    }

    .place.idle text {
        fill: var(--color-text-muted);
    }

    .place text.winner {
        font-weight: 600;
        fill: var(--color-text);
    }

    .place.running .box {
        stroke: var(--color-primary);
        stroke-width: 2;
    }

    .place.flagged .box {
        stroke: var(--color-danger);
        stroke-width: 2;
    }

    .place.skipped {
        opacity: 0.5;
    }

    .card-anchor {
        position: absolute;
        z-index: 20;
        width: 18rem;
    }

    .card-layer {
        position: relative;
    }

    .correction {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: var(--space-2);
        border: 1px solid var(--color-primary);
        border-radius: var(--radius);
        background: var(--color-surface);
        box-shadow: 0 6px 18px rgba(0, 0, 0, 0.18);
    }

    .correction-head {
        display: flex;
        justify-content: space-between;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .correction-head .close {
        border: none;
        background: transparent;
        color: var(--color-text-muted);
        cursor: pointer;
    }

    table.cross {
        border-collapse: collapse;
        font-size: var(--font-size-small);
    }

    .cross th,
    .cross td {
        border: 1px solid var(--color-border);
        padding: 0.15rem 0.4rem;
        text-align: center;
        position: relative;
    }

    .cross th.name {
        text-align: left;
        font-weight: 600;
    }

    .cross th.vs {
        font-weight: 400;
        color: var(--color-text-muted);
        max-width: 7rem;
    }

    .cross td.self {
        background: var(--color-surface-alt);
    }

    .cross td.running {
        outline: 2px solid var(--color-primary);
        outline-offset: -2px;
    }

    .cross td.flagged {
        outline: 2px solid var(--color-danger);
        outline-offset: -2px;
    }

    .pool-cell {
        border: none;
        background: transparent;
        color: var(--color-text);
        cursor: pointer;
        width: 100%;
    }

    .pool-cell:disabled {
        cursor: default;
        color: var(--color-text-muted);
    }

    .pool-cell .winner {
        font-weight: 600;
    }

    table.lives {
        border-collapse: collapse;
        width: 100%;
        font-size: var(--font-size-small);
    }

    .lives th {
        text-align: left;
        font-weight: 600;
        color: var(--color-text-muted);
        border-bottom: 1px solid var(--color-border);
        padding: 0.2rem 0.4rem;
    }

    .lives td {
        padding: 0.15rem 0.4rem;
        border-bottom: 1px solid var(--color-surface-alt);
    }

    .lives tr.out td {
        color: var(--color-text-muted);
        text-decoration: line-through;
    }

    .lives td.name {
        font-weight: 600;
    }

    td.opponents {
        max-width: 18rem;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: var(--color-text-muted);
    }
</style>
