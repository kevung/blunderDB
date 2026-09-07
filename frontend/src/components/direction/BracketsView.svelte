<script>
    /*
     * La vue Arbres (issue #373, fonctionnel.md §5.6).
     *
     * Le moteur dessine aussi les arbres — son paquet `render` produit le SVG de la page
     * d'affichage autonome — mais la vue DANS l'application est dessinée ici, et c'est un
     * partage volontaire : à l'intérieur, un arbre est traduit en neuf langues et cliquable, et
     * ces deux choses vivent au frontend.
     *
     * Ce qu'un directeur vient y chercher : où en est le tableau, et quel match cloche. Un
     * avertissement du moteur se voit donc SUR la place concernée, pas seulement dans une liste
     * loin de l'arbre.
     *
     * Une phase suisse n'a pas de graphe avant sa bascule : sa vue est le tableau des vies, avec
     * les adversaires déjà rencontrés — ce qu'on regarde avant d'apparier deux joueurs à la main.
     */
    import { t } from '../../i18n';
    import { renderLabel, renderSectionName } from './labels.js';

    let { phases = [], onOpenMatch = () => {} } = $props();

    /* La phase courante est dépliée ; les précédentes sont repliées mais consultables — un
       directeur relit le tableau des vies pendant que le tableau final tourne. */
    let openPhases = $state(null);

    const state = $derived.by(() => {
        if (openPhases) return openPhases;
        const s = {};
        for (const p of phases) s[p.index] = p.current;
        return s;
    });

    function toggle(i) {
        const next = { ...state };
        next[i] = !next[i];
        openPhases = next;
    }

    function phaseTitle(p) {
        return p.name || $t(`direction.format.${p.kind}`);
    }

    /* Les places d'une section, rangées par tour : c'est ainsi qu'un tableau se lit, en
       colonnes du premier tour à la finale. */
    function columns(section) {
        const cols = [];
        for (const m of section.matches) {
            (cols[m.round] ||= []).push(m);
        }
        return cols.map((c) => c || []);
    }

    function outcome(m) {
        if (m.skipped) return $t('direction.bracket.skipped');
        if (m.walkover) return $t('direction.bracket.walkover');
        if (m.done && m.scoreA + m.scoreB > 0) return `${m.scoreA}–${m.scoreB}`;
        if (m.running) return $t('direction.bracket.running');
        return '';
    }
</script>

<div class="brackets">
    {#each phases as p (p.index)}
        <section class="phase">
            <button type="button" class="phase-head" onclick={() => toggle(p.index)}>
                <span class="chevron">{state[p.index] ? '▾' : '▸'}</span>
                <span class="phase-name">{phaseTitle(p)}</span>
                {#if p.current}
                    <span class="badge">{$t('direction.bracket.current')}</span>
                {/if}
                {#if !p.drawn && p.sections.length === 0 && !p.lives}
                    <span class="muted">{$t('direction.bracket.notDrawn')}</span>
                {/if}
            </button>

            {#if state[p.index]}
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
                                    <!-- Un barrage n'a pas de graphe : ses appariements se
                                         tirent au fur et à mesure. Ce qui compte est qui reste
                                         et pour combien de places. -->
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
