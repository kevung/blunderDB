<script>
    /*
     * La vue Arbres (fonctionnel.md §5.6), dessinée ici et non par le `render` du moteur :
     * traduite et cliquable. Un avertissement se voit sur la place concernée. Une phase
     * suisse avant bascule montre le tableau des vies et les adversaires rencontrés.
     */
    import { t } from '../../i18n';
    import { renderLabel, renderSectionName } from './labels.js';

    let { phases = [], onOpenMatch = () => {} } = $props();

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

    /* Les places d'une section, rangées par tour : c'est ainsi qu'un tableau se lit, en
       colonnes du premier tour à la finale. */
    /** @param {BracketSection} section */
    function columns(section) {
        /** @type {BracketMatch[][]} */
        const cols = [];
        for (const m of section.matches) {
            (cols[m.round] ||= []).push(m);
        }
        return cols.map((c) => c || []);
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
                                {:else}
                                    <div class="cols">
                                        {#each columns(s) as col, ci (ci)}
                                            <div class="col">
                                                {#each col as m (m.key)}
                                                    <button
                                                        type="button"
                                                        class="place"
                                                        class:done={m.done}
                                                        class:running={m.running}
                                                        class:flagged={m.flagged}
                                                        class:skipped={m.skipped}
                                                        disabled={!m.matchId}
                                                        title={renderLabel($t, m.label)}
                                                        onclick={() => onOpenMatch(m)}
                                                    >
                                                        <span class="side" class:winner={m.done && m.winner === m.a}>{m.aName || $t('direction.bracket.pending')}</span>
                                                        <span class="side" class:winner={m.done && m.winner === m.b}>{m.bName || $t('direction.bracket.pending')}</span>
                                                        {#if outcome(m)}
                                                            <span class="outcome">{outcome(m)}</span>
                                                        {/if}
                                                    </button>
                                                {/each}
                                            </div>
                                        {/each}
                                    </div>
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

    .cols {
        display: flex;
        gap: var(--space-2);
        align-items: flex-start;
    }

    .col {
        display: flex;
        flex-direction: column;
        justify-content: space-around;
        gap: var(--space-1);
        min-width: 11rem;
    }

    .place {
        display: flex;
        flex-direction: column;
        gap: 1px;
        padding: 0.2rem 0.4rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        text-align: left;
        font-size: var(--font-size-small);
        cursor: pointer;
    }

    .place:disabled {
        cursor: default;
        color: var(--color-text-muted);
    }

    .place.running {
        border-color: var(--color-primary);
    }

    /* L'avertissement du moteur se voit SUR la place, pas seulement dans une liste loin de
       l'arbre : c'est là que le directeur regarde quand un tableau cloche. */
    .place.flagged {
        border-color: var(--color-danger);
        border-width: 2px;
    }

    .place.skipped {
        opacity: 0.5;
    }

    .side.winner {
        font-weight: 600;
    }

    .outcome {
        color: var(--color-text-muted);
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
