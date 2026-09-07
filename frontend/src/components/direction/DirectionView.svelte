<script>
    /*
     * La vue tournoi : ce que la zone principale affiche à la place du plateau quand une
     * Direction est ouverte (ADR-0047, tasks/nicomaque/ux.md §2).
     *
     * C'est la première fois de l'application que la zone principale montre autre chose que le
     * plateau, et cela n'arrive QUE là : tout autre onglet ramène le plateau sans rien fermer.
     *
     * Ce lot pose le squelette et l'écran des réglages. La page « Direction » elle-même — file
     * des propositions, grille des tables, fiche de résultat — arrive avec les issues qui la
     * mesurent (#370, #371), et les autres vues avec les leurs. Les onglets absents ne sont pas
     * un oubli : ils sont vides tant que leur issue n'a pas livré, et le disent.
     */
    import { t } from '../../i18n';
    import { statusBarTextStore } from '../../stores/uiStore';
    import { tMsg } from '../../i18n';
    import { logger } from '../../utils/logger.js';
    import DirectionSettings from './DirectionSettings.svelte';
    import { directionStore, openDirectionIdStore, saveDirectionConfig, startDirection, deleteDirection, closeDirection } from '../../stores/directionStore';

    const view = $derived($directionStore);
    const state = $derived(view?.state || 'draft');

    /* En préparation la vue s'ouvre sur les Réglages, puisque c'est le seul geste possible ;
       en cours elle s'ouvrira sur la page Direction, où le directeur passe 95 % de son temps. */
    let tab = $state('settings');
    $effect(() => {
        if (state !== 'draft' && tab === 'settings') tab = 'direction';
    });

    let config = $state(null);
    $effect(() => {
        // La configuration éditée est une copie : tant qu'elle n'est pas enregistrée, elle ne
        // doit pas se confondre avec ce que la base contient.
        const c = view?.config;
        config = c ? JSON.parse(JSON.stringify(c)) : null;
    });

    const tabs = [
        { id: 'direction', labelKey: 'direction.tabs.direction' },
        { id: 'players', labelKey: 'direction.tabs.players' },
        { id: 'brackets', labelKey: 'direction.tabs.brackets' },
        { id: 'standings', labelKey: 'direction.tabs.standings' },
        { id: 'history', labelKey: 'direction.tabs.history' },
        { id: 'settings', labelKey: 'direction.tabs.settings' }
    ];

    async function apply(next) {
        try {
            await saveDirectionConfig($openDirectionIdStore, next);
            statusBarTextStore.set(tMsg('direction.settings.saved'));
        } catch (e) {
            logger.error('direction: saving configuration failed', e);
            statusBarTextStore.set(tMsg('direction.settings.errorSaving'));
        }
    }

    async function start() {
        try {
            await startDirection($openDirectionIdStore, []);
            statusBarTextStore.set(tMsg('direction.settings.started'));
        } catch (e) {
            logger.error('direction: start failed', e);
            statusBarTextStore.set(tMsg('direction.settings.errorStarting'));
        }
    }

    async function remove() {
        if (!window.confirm($t('direction.settings.deleteConfirm'))) return;
        try {
            await deleteDirection($openDirectionIdStore);
            statusBarTextStore.set(tMsg('direction.settings.deleted'));
        } catch (e) {
            logger.error('direction: delete failed', e);
        }
    }
</script>

<div class="direction-view">
    <header>
        <span class="name">{view?.config?.name || ''}</span>
        <span class="state">{$t(`direction.state.${state}`)}</span>
        <nav>
            {#each tabs as item (item.id)}
                <button type="button" class:active={tab === item.id} onclick={() => (tab = item.id)}>{$t(item.labelKey)}</button>
            {/each}
        </nav>
        <span class="spacer"></span>
        <button type="button" class="close" onclick={closeDirection}>{$t('direction.close')}</button>
    </header>

    <div class="body">
        {#if tab === 'settings'}
            <DirectionSettings
                bind:config
                {state}
                tournamentName={view?.config?.name || ''}
                entrantCount={view?.players?.length || 0}
                onApply={apply}
                onStart={state === 'draft' ? start : null}
                onDelete={remove}
            />
        {:else}
            <p class="placeholder">{$t('direction.tabs.notYet')}</p>
        {/if}
    </div>
</div>

<style>
    .direction-view {
        display: flex;
        flex-direction: column;
        height: 100%;
        min-height: 0;
        text-align: left;
        color: var(--color-text);
        background: var(--color-surface);
    }

    header {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        padding: var(--space-1) var(--space-2);
        border-bottom: 1px solid var(--color-border);
        flex-wrap: wrap;
    }

    .name {
        font-weight: 600;
    }

    .state {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    nav {
        display: flex;
        gap: var(--space-1);
        flex-wrap: wrap;
    }

    nav button,
    .close {
        font-size: var(--font-size-small);
        padding: 0.15rem 0.55rem;
        border: 1px solid transparent;
        border-radius: var(--radius);
        background: transparent;
        color: var(--color-text-muted);
        cursor: pointer;
    }

    nav button.active {
        border-color: var(--color-primary);
        color: var(--color-text);
    }

    .close {
        border-color: var(--color-border);
        color: var(--color-text);
    }

    .spacer {
        flex: 1;
    }

    .body {
        flex: 1;
        min-height: 0;
        overflow: auto;
    }

    .placeholder {
        margin: var(--space-4);
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
</style>
