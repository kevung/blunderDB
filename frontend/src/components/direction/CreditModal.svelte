<script>
    /*
     * Le crédit du moteur (issue #378, fonctionnel.md §10).
     *
     * Le moteur de tournoi n'est pas de blunderDB : c'est **Nicomaque, créé par Nicolas
     * Harmand**. Le crédit doit être visible là où quelqu'un se demande d'où vient ce que
     * l'écran affiche — d'où ce bouton dans la gestion d'un tournoi, en plus de la page À
     * propos, de l'aide et du pied des pages produites.
     *
     * La version affichée est celle réellement embarquée, lue sur la Direction ouverte, et non
     * une chaîne recopiée à la main qui vieillirait en silence.
     */
    import { t } from '../../i18n';
    import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';

    let { engineVersion = '', onClose = () => {} } = $props();

    const REPO = 'https://github.com/PileOfCells/backgammon-tournoi';
    const DOCS = 'https://pileofcells.github.io/backgammon-tournoi/';

    function open(url) {
        // Le navigateur du système, pas la vue web : une page de documentation n'a rien à
        // faire dans la fenêtre de l'application.
        BrowserOpenURL(url);
    }
</script>

<div class="credit">
    <header>
        <span class="title">{$t('direction.credit.title')}</span>
        <span class="grow"></span>
        <button type="button" class="close" onclick={onClose} title={$t('common.close')}>×</button>
    </header>

    <p class="who">{$t('direction.credit.by')}</p>
    <p class="what">{$t('direction.credit.what')}</p>

    <div class="links">
        <button type="button" onclick={() => open(REPO)}>{$t('direction.credit.repo')}</button>
        <button type="button" onclick={() => open(DOCS)}>{$t('direction.credit.docs')}</button>
    </div>

    {#if engineVersion}
        <p class="version">{$t('direction.credit.version', { version: engineVersion })}</p>
    {/if}
</div>

<style>
    .credit {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: var(--space-2);
        max-width: 32rem;
        border: 1px solid var(--color-primary);
        border-radius: var(--radius);
        background: var(--color-surface);
        text-align: left;
    }

    header {
        display: flex;
        align-items: center;
    }

    .title {
        font-weight: 600;
    }

    .grow {
        flex: 1;
    }

    .who {
        margin: 0;
        font-weight: 600;
    }

    .what {
        margin: 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .links {
        display: flex;
        gap: var(--space-1);
    }

    .version {
        margin: 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        font-variant-numeric: tabular-nums;
    }

    button {
        padding: 0.15rem 0.6rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
        font-size: var(--font-size-small);
    }

    .close {
        border-color: transparent;
        color: var(--color-text-muted);
    }
</style>
