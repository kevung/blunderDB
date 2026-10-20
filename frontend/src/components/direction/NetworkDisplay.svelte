<script>
    /**
     * L'affichage du tournoi sur le réseau du club (ADR-0097) : le bureau sert la page d'affichage
     * de CE tournoi, en lecture seule, et montre le QR code de son adresse. Éteint tant que le
     * directeur ne l'allume pas ; l'avertissement reste visible tant que la page est servie, parce
     * que personne n'est authentifié (ADR-0005).
     */
    import { t } from '../../i18n';
    import { StartTournamentDisplay, StopTournamentDisplay, GetTournamentDisplayStatus } from '../../../wailsjs/go/gui/App.js';
    import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime.js';

    const DEFAULT_PORT = 8770;

    /** @type {{ tournamentId: number }} */
    let { tournamentId } = $props();

    /** @type {{ running: boolean, tournamentId: number, url: string, localUrl: string, qr: string, error: string }} */
    let status = $state({ running: false, tournamentId: 0, url: '', localUrl: '', qr: '', error: '' });
    let port = $state(DEFAULT_PORT);
    let busy = $state(false);

    const showsThis = $derived(status.running && status.tournamentId === tournamentId);

    $effect(() => {
        // Sans liaison Go (aperçu, tests), l'appel lève avant toute promesse : l'affichage
        // reste simplement éteint.
        Promise.resolve()
            .then(() => GetTournamentDisplayStatus())
            .then((s) => (status = s))
            .catch(() => {});
    });

    /** @param {() => Promise<any>} action */
    async function apply(action) {
        busy = true;
        try {
            status = await action();
        } catch (e) {
            status = { ...status, error: String(e) };
        } finally {
            busy = false;
        }
    }

    function start() {
        apply(() => StartTournamentDisplay({ tournamentId, port: Number(port) || DEFAULT_PORT }));
    }

    function stop() {
        apply(() => StopTournamentDisplay());
    }
</script>

<details class="sec" data-testid="direction-network-display">
    <summary><h3>{$t('direction.network.title')}</h3></summary>
    <p class="facts">{$t('direction.network.hint')}</p>
    <p class="warning" role="note">{$t('direction.network.warning')}</p>
    {#if status.running && !showsThis}
        <p class="facts">{$t('direction.network.other', { id: status.tournamentId })}</p>
    {/if}
    <div class="actions">
        {#if !showsThis}
            <label>
                {$t('direction.network.port')}
                <input type="number" min="1" max="65535" bind:value={port} />
            </label>
            <button type="button" data-testid="direction-network-start" disabled={busy} onclick={start}>{$t('direction.network.start')}</button>
        {/if}
        {#if status.running}
            <button type="button" data-testid="direction-network-stop" disabled={busy} onclick={stop}>{$t('direction.network.stop')}</button>
        {/if}
    </div>
    {#if status.error}
        <p class="error">{$t('direction.network.error', { error: status.error })}</p>
    {/if}
    {#if showsThis}
        <div class="share">
            <img class="qr" src={status.qr} alt={$t('direction.network.qr')} width="192" height="192" />
            <div>
                <p class="facts">{$t('direction.network.url')}</p>
                <p class="facts path" data-testid="direction-network-url">{status.url}</p>
                <button type="button" class="link" onclick={() => BrowserOpenURL(status.localUrl)}>{$t('direction.network.openHere')}</button>
            </div>
        </div>
    {/if}
</details>

<style>
    .sec summary h3 {
        display: inline;
        font-size: var(--font-size-base);
    }

    .facts {
        color: var(--color-text-muted);
        margin: 0.3rem 0;
    }

    .path {
        font-family: var(--font-family-mono);
        overflow-wrap: anywhere;
        color: var(--color-text);
    }

    .warning {
        color: var(--color-danger);
        margin: 0.3rem 0;
    }

    .error {
        color: var(--color-danger);
    }

    .actions {
        display: flex;
        gap: 0.5rem;
        align-items: center;
        flex-wrap: wrap;
    }

    .actions input {
        width: 6rem;
    }

    .share {
        display: flex;
        gap: 1rem;
        align-items: flex-start;
        flex-wrap: wrap;
        margin-top: 0.5rem;
    }

    .qr {
        image-rendering: pixelated;
        /* Un lecteur de QR code veut des modules sombres sur fond clair, quel que soit le thème. */
        background: white;
    }

    .link {
        background: none;
        border: none;
        padding: 0;
        color: var(--color-primary);
        text-decoration: underline;
        cursor: pointer;
    }
</style>
