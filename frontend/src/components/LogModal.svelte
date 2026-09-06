<script>
    // Le journal d'activité (#287, fiche I.31).
    //
    // L'interface attrape des erreurs à quelque soixante-quinze endroits, et
    // jusqu'ici chacune finissait sur une ligne de barre d'état qui
    // disparaissait. Le détail était dans le journal depuis toujours ; ce qui
    // manquait était de pouvoir le lire sans quitter l'application, et de
    // pouvoir le COPIER pour le joindre à un rapport.
    //
    // Le journal n'est pas filtré ni reformaté ici : ce qui est montré est ce
    // que le fichier contient. Un journal qu'on embellit est un journal qu'on
    // ne peut plus citer.
    import Modal from './Modal.svelte';
    import { ReadLogTail, OpenLogsFolder } from '../../wailsjs/go/gui/App.js';
    import { ClipboardSetText } from '../../wailsjs/runtime/runtime.js';
    import { statusBarTextStore } from '../stores/uiStore.js';
    import { logger } from '../utils/logger.js';
    import { t, tMsg } from '../i18n';

    let { visible = false, onClose } = $props();

    /** Le nombre de lignes lues. Deux cents : de quoi couvrir une session de
     *  travail sans faire d'une fenêtre un fichier. */
    const LINES = 200;

    /** @type {string[]} */
    let lines = $state([]);
    let error = $state('');

    $effect(() => {
        if (!visible) return;
        error = '';
        let cancelled = false;
        // `await` rather than `.then` : le pont Wails est remplacé par un
        // bouchon dans les tests de composants, et un bouchon n'a pas de
        // `.then` — l'attendre marche, l'enchaîner casse le rendu.
        (async () => {
            try {
                const result = await ReadLogTail(LINES);
                if (!cancelled) lines = Array.isArray(result) ? result : [];
            } catch (e) {
                if (cancelled) return;
                logger.error('could not read the log:', e);
                error = String(e);
            }
        })();
        return () => {
            cancelled = true;
        };
    });

    async function copyReport() {
        try {
            await ClipboardSetText(lines.join('\n'));
            statusBarTextStore.set(tMsg('log.copied', { n: lines.length }));
        } catch (e) {
            logger.error('could not copy the log:', e);
            statusBarTextStore.set(tMsg('log.copyFailed'));
        }
    }
</script>

<Modal open={visible} onclose={onClose} size="wide" closeOnOverlay label={$t('log.title')}>
    <h2 class="log-title">{$t('log.title')}</h2>
    <div class="log-actions">
        <button type="button" onclick={copyReport} disabled={lines.length === 0}>{$t('log.copy')}</button>
        <button type="button" onclick={() => OpenLogsFolder()}>{$t('log.openFolder')}</button>
    </div>
    {#if error}
        <p class="log-error">{error}</p>
    {:else if lines.length === 0}
        <p class="log-empty">{$t('log.empty')}</p>
    {:else}
        <pre class="log-body">{lines.join('\n')}</pre>
    {/if}
</Modal>

<style>
    .log-title {
        margin: 0 0 0.5em;
        font-size: var(--font-size-title);
        font-weight: 600;
    }

    .log-actions {
        display: flex;
        gap: 0.4em;
        margin-bottom: 0.5em;
    }

    .log-actions button {
        cursor: pointer;
    }

    .log-body {
        max-height: 60vh;
        overflow: auto;
        margin: 0;
        padding: 0.4em 0.6em;
        background: var(--color-surface-alt);
        border: 1px solid var(--color-border);
        font-family: var(--font-family-mono);
        font-size: var(--font-size-small);
        white-space: pre;
    }

    .log-empty,
    .log-error {
        margin: 0;
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .log-error {
        color: var(--color-danger);
    }
</style>
