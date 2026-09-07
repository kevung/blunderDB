<script>
    /*
     * Les réglages d'une Direction (ADR-0047 §3, tasks/nicomaque/ux.md §2.1).
     *
     * Deux contraintes commandent le dessin de cet écran, et elles viennent du cadrage :
     * le coût d'entrée doit être bas — un directeur qui dirige un tournoi par an ne doit rien
     * lire — et la souris est première. D'où : des cartes de format cliquables plutôt qu'un
     * formulaire vide, un défaut qui tient pour un tournoi de club dans chaque champ, une
     * phrase d'infobulle par réglage, et aucun assistant en plusieurs écrans.
     *
     * En préparation tout est modifiable ; une fois le tournoi commencé les champs figés sont
     * GRISÉS AVEC LEUR RAISON plutôt qu'absents, pour que le directeur sache que ce n'est pas
     * lui qui a mal cherché.
     */
    import { t } from '../../i18n';
    import { namedConfigs } from '../../stores/directionStore';

    let { config = $bindable(), state = 'draft', tournamentName = '', onApply = () => {}, onStart = null, onDelete = null, entrantCount = 0 } = $props();

    const isDraft = $derived(state === 'draft');
    const frozenReason = $derived(isDraft ? '' : $t('direction.settings.frozen'));

    function pickNamed(named) {
        config = named.build(tournamentName || config?.name || '');
        onApply(config);
    }

    function phaseKindLabel(kind) {
        return $t(`direction.format.${kind}`);
    }

    /* Le nombre d'exemptions qu'un tableau devrait donner au premier tour avec cet effectif :
       une information que le directeur regarde pour choisir son format, pas un réglage. */
    const byesAtFirstRound = $derived.by(() => {
        if (!entrantCount) return 0;
        let size = 1;
        while (size < entrantCount) size *= 2;
        return size - entrantCount;
    });
</script>

<div class="settings">
    {#if isDraft}
        <section class="formats">
            <h3>{$t('direction.settings.format')}</h3>
            <div class="cards">
                {#each namedConfigs as named (named.id)}
                    <button type="button" class="card" class:recommended={named.recommended} onclick={() => pickNamed(named)}>
                        <span class="card-name">{$t(`direction.named.${named.id}`)}</span>
                        <span class="card-hint">{$t(`direction.named.${named.id}Hint`)}</span>
                        {#if named.recommended}
                            <span class="badge">{$t('direction.settings.recommended')}</span>
                        {/if}
                    </button>
                {/each}
            </div>
        </section>
    {/if}

    {#if config}
        <section>
            <h3>{$t('direction.settings.phases')}</h3>
            <ul class="phases">
                {#each config.phases || [] as phase, i (phase.kind + i)}
                    <li>
                        <span class="phase-index">{i + 1}</span>
                        <span class="phase-kind">{phaseKindLabel(phase.kind)}</span>
                        <label title={$t('direction.settings.lengthHint')}>
                            {$t('direction.settings.length')}
                            <input type="number" min="1" max="99" bind:value={phase.length} disabled={!isDraft} title={frozenReason} />
                        </label>
                        {#if phase.kind === 'swiss_lives'}
                            <label title={$t('direction.settings.livesHint')}>
                                {$t('direction.settings.lives')}
                                <input type="number" min="1" max="5" bind:value={phase.lives} disabled={!isDraft} title={frozenReason} />
                            </label>
                            <label title={$t('direction.settings.targetHint')}>
                                {$t('direction.settings.target')}
                                <input type="number" min="0" step="8" bind:value={phase.target} title={$t('direction.settings.targetHint')} />
                            </label>
                        {/if}
                        {#if phase.kind === 'bracket' || phase.kind === 'lives_bracket'}
                            <label title={$t('direction.settings.finalLengthHint')}>
                                {$t('direction.settings.finalLength')}
                                <input type="number" min="0" max="99" bind:value={phase.final_length} disabled={!isDraft} title={frozenReason} />
                            </label>
                        {/if}
                    </li>
                {/each}
            </ul>
        </section>

        <section>
            <h3>{$t('direction.settings.tables')}</h3>
            <label title={$t('direction.settings.tableCountHint')}>
                {$t('direction.settings.tableCount')}
                <input type="number" min="0" max="200" bind:value={config.tables.count} />
            </label>
            {#if entrantCount > 0}
                <p class="facts">
                    {$t('direction.settings.entrants', { n: entrantCount })}
                    {#if byesAtFirstRound > 0}
                        &middot;
                        {$t('direction.settings.byes', { n: byesAtFirstRound })}
                    {/if}
                </p>
            {/if}
        </section>

        <div class="actions">
            <button type="button" class="primary" onclick={() => onApply(config)}>
                {$t('direction.settings.apply')}
            </button>
            {#if isDraft && onStart}
                <button type="button" class="primary" onclick={onStart}>
                    {$t('direction.settings.start')}
                </button>
            {/if}
            {#if onDelete}
                <button type="button" class="danger" onclick={onDelete}>
                    {$t('direction.settings.delete')}
                </button>
            {/if}
        </div>
        {#if !isDraft}
            <p class="frozen-note">{frozenReason}</p>
        {/if}
    {/if}
</div>

<style>
    .settings {
        display: flex;
        flex-direction: column;
        gap: 1rem;
        padding: 0.75rem;
        overflow-y: auto;
    }

    h3 {
        font-size: var(--font-size-base);
        font-weight: 600;
        margin: 0 0 0.4rem;
        color: var(--color-text);
    }

    .cards {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
        gap: 0.5rem;
    }

    /* Une carte est une cible large : le premier geste d'un directeur occasionnel ne doit pas
       demander de viser. */
    .card {
        display: flex;
        flex-direction: column;
        gap: 0.2rem;
        text-align: left;
        padding: 0.6rem 0.7rem;
        min-height: 64px;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    .card:hover,
    .card:focus-visible {
        border-color: var(--color-primary);
    }

    .card.recommended {
        border-color: var(--color-primary);
    }

    .card-name {
        font-weight: 600;
    }

    .card-hint {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .badge {
        align-self: flex-start;
        font-size: var(--font-size-small);
        color: var(--color-primary);
    }

    .phases {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0.35rem;
    }

    .phases li {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 0.5rem;
    }

    .phase-index {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 1.4rem;
        height: 1.4rem;
        border-radius: 50%;
        background: var(--color-border);
        font-size: var(--font-size-small);
    }

    .phase-kind {
        font-weight: 600;
        min-width: 9rem;
    }

    label {
        display: inline-flex;
        align-items: center;
        gap: 0.3rem;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    /* Un contrôle n'hérite ni de la taille ni de la famille : sans `font: inherit` il
       retomberait dans la police du navigateur (invariant « une seule échelle de type »). */
    input {
        width: 4.5rem;
        padding: 0.15rem 0.3rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
    }

    input:disabled {
        opacity: 0.55;
        cursor: not-allowed;
    }

    .facts,
    .frozen-note {
        margin: 0.3rem 0 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .actions {
        display: flex;
        gap: 0.5rem;
        flex-wrap: wrap;
    }

    button.primary,
    button.danger {
        padding: 0.35rem 0.8rem;
        border-radius: var(--radius);
        border: 1px solid var(--color-border);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    button.primary {
        border-color: var(--color-primary);
    }

    button.danger {
        border-color: var(--color-danger);
    }
</style>
