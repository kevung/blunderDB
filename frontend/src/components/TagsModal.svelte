<script>
    // Le vocabulaire de tags (#265, fiche I.9).
    //
    // Un tag est un `#mot` dans un commentaire. Rien ne le déclare, aucune
    // table ne le porte, et c'est voulu : le vocabulaire est la prose de
    // l'utilisateur, et exiger une déclaration avant de pouvoir taguer
    // transformerait une habitude en paperasse. Ce qui manquait, c'est
    // l'autre moitié — VOIR le vocabulaire qu'on s'est construit, et cliquer
    // un tag plutôt que se rappeler comment on l'écrivait.
    //
    // Une fenêtre plutôt qu'un onglet (la fiche disait « panneau ») : on
    // consulte un vocabulaire puis on le referme, et cliquer un tag lance une
    // recherche, qui bascule de toute façon sur les résultats. Une fenêtre
    // qui se ferme au clic est exactement ce comportement-là.
    import Modal from './Modal.svelte';
    import { Tags, RecommendedTags } from '../../wailsjs/go/database/Database.js';
    import { loadPositionsByFilters } from '../services/positionService.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';

    let { visible = false, onClose } = $props();

    /** @type {{tag: string, count: number}[]} */
    let tags = $state([]);
    /** @type {string[]} */
    let recommended = $state([]);
    let loading = $state(false);

    $effect(() => {
        if (visible) void load();
    });

    async function load() {
        loading = true;
        try {
            const [used, suggested] = await Promise.all([Tags(), RecommendedTags()]);
            tags = used || [];
            recommended = suggested || [];
        } catch (error) {
            logger.error('could not read the tags:', error);
            tags = [];
        } finally {
            loading = false;
        }
    }

    /** Les tags suggérés que cette base n'utilise pas encore. */
    let unused = $derived.by(() => {
        /** @type {Record<string, boolean>} */
        const used = {};
        for (const entry of tags) used[entry.tag] = true;
        return recommended.filter((r) => !used[r]);
    });

    /** @param {string} tag */
    function search(tag) {
        onClose?.();
        loadPositionsByFilters({ tagFilter: tag, searchCommand: `s ${tag}` });
    }
</script>

<Modal open={visible} onclose={onClose} size="auto" closeOnOverlay label={$t('tags.title')}>
    <div class="tags">
        <p class="note">{$t('tags.intro')}</p>

        {#if loading}
            <p class="note">{$t('common.loading')}</p>
        {:else if tags.length === 0}
            <p class="note">{$t('tags.none')}</p>
        {:else}
            <ul class="tag-list">
                {#each tags as entry (entry.tag)}
                    <li>
                        <button type="button" class="tag" onclick={() => search(entry.tag)} title={$t('tags.searchHint', { tag: entry.tag })}>
                            {entry.tag}
                        </button>
                        <span class="count">{$t('tags.positions', { n: entry.count })}</span>
                    </li>
                {/each}
            </ul>
        {/if}

        {#if unused.length > 0}
            <p class="note suggest">{$t('tags.suggested')}</p>
            <ul class="tag-list suggested">
                {#each unused as tag (tag)}
                    <li><span class="tag inert">{tag}</span></li>
                {/each}
            </ul>
        {/if}
    </div>
</Modal>

<style>
    .tags {
        display: flex;
        flex-direction: column;
        gap: 0.6em;
        min-width: 20em;
        max-width: 32em;
    }

    .note {
        margin: 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .suggest {
        margin-top: 0.4em;
        border-top: 1px solid var(--color-border);
        padding-top: 0.6em;
    }

    .tag-list {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0.15em;
        max-height: 18em;
        overflow-y: auto;
    }

    .tag-list.suggested {
        flex-direction: row;
        flex-wrap: wrap;
        gap: 0.4em;
    }

    .tag-list li {
        display: flex;
        align-items: baseline;
        gap: 0.6em;
    }

    .tag {
        cursor: pointer;
        border: none;
        background: none;
        padding: 0;
        color: var(--color-primary);
        text-align: left;
    }

    .tag.inert {
        cursor: default;
        color: var(--color-text-muted);
    }

    .count {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
</style>
