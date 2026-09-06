<script>
    // L'écran d'accueil (#284, fiche I.28).
    //
    // Sans base ouverte, blunderDB montrait un plateau vide et une barre
    // d'outils. Un plateau vide n'est pas une invitation : il ne dit ni ce que
    // l'outil sait faire, ni par où commencer. Quatre chemins, nommés par ce
    // qu'ils apportent plutôt que par la mécanique qu'ils déclenchent.
    //
    // « Importer mes matchs » est le chemin qui compte, et il enchaîne : une
    // base neuve, les fichiers choisis, le compte rendu (#257) — « voici votre
    // PR et vos pires erreurs » — puis la file d'étude (#259) si l'on veut.
    // C'est la promesse du produit tenue en deux minutes plutôt qu'expliquée.
    import { openModal, MODAL } from '../stores/uiStore';
    import { newDatabase, openDatabase, openDatabaseByPath, loadDemoDatabase } from '../services/databaseService.js';
    import { importPosition } from '../services/importService.js';
    import { GetLastDatabasePath } from '../../wailsjs/go/main/Config.js';
    import { PathExists } from '../../wailsjs/go/gui/App.js';
    import { logger } from '../utils/logger.js';
    import { t } from '../i18n';

    let { onDismiss } = $props();

    /** Le chemin de la dernière base, s'il pointe encore sur un fichier. */
    let lastPath = $state('');

    $effect(() => {
        void loadLastPath();
    });

    async function loadLastPath() {
        try {
            const path = await GetLastDatabasePath();
            // Une base déplacée ou effacée ne doit pas être proposée : un
            // bouton qui échoue au clic est pire que pas de bouton.
            lastPath = path && (await PathExists(path)) ? path : '';
        } catch (error) {
            logger.error('could not read the last database path:', error);
            lastPath = '';
        }
    }

    /** @param {string} path */
    function basename(path) {
        const parts = path.split(/[\\/]/);
        return parts[parts.length - 1] || path;
    }

    // Importer mes matchs : une base neuve puis le sélecteur de fichiers. Le
    // compte rendu s'ouvre tout seul à la fin de l'import, et propose la file.
    async function importMyMatches() {
        await newDatabase();
        await importPosition();
    }
</script>

<div class="home" data-tour="home">
    <h1>{$t('home.title')}</h1>
    <p class="lede">{$t('home.lede')}</p>

    <div class="choices">
        <button type="button" class="choice primary" onclick={importMyMatches}>
            <span class="choice-title">{$t('home.importTitle')}</span>
            <span class="choice-note">{$t('home.importNote')}</span>
        </button>

        <button type="button" class="choice" onclick={loadDemoDatabase}>
            <span class="choice-title">{$t('home.demoTitle')}</span>
            <span class="choice-note">{$t('home.demoNote')}</span>
        </button>

        <button type="button" class="choice" onclick={() => openModal(MODAL.TOUR)}>
            <span class="choice-title">{$t('home.tourTitle')}</span>
            <span class="choice-note">{$t('home.tourNote')}</span>
        </button>

        <button type="button" class="choice" onclick={openDatabase}>
            <span class="choice-title">{$t('home.openTitle')}</span>
            <span class="choice-note">{$t('home.openNote')}</span>
        </button>
    </div>

    {#if lastPath}
        <button type="button" class="resume" onclick={() => openDatabaseByPath(lastPath)}>
            {$t('home.resume', { name: basename(lastPath) })}
        </button>
    {/if}

    <button type="button" class="dismiss" data-testid="home-dismiss" onclick={() => onDismiss?.()}>{$t('home.dismiss')}</button>
</div>

<style>
    .home {
        position: absolute;
        inset: 0;
        z-index: 5;
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 0.8em;
        padding: 2em;
        background: var(--color-surface);
        overflow-y: auto;
    }

    h1 {
        margin: 0;
        font-size: var(--font-size-title);
    }

    .lede {
        margin: 0 0 0.6em 0;
        color: var(--color-text-muted);
        max-width: 34em;
        text-align: center;
    }

    .choices {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(15em, 1fr));
        gap: 0.6em;
        width: 100%;
        max-width: 44em;
    }

    .choice {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        gap: 0.25em;
        padding: 0.9em 1em;
        text-align: left;
        cursor: pointer;
        border: 1px solid var(--color-border);
        background: var(--color-surface);
    }

    .choice.primary {
        border-color: var(--color-primary);
    }

    .choice-title {
        font-weight: 600;
    }

    .choice-note {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .resume,
    .dismiss {
        cursor: pointer;
        border: none;
        background: none;
        color: var(--color-primary);
    }

    .dismiss {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }
</style>
