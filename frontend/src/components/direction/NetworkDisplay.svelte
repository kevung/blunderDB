<script>
    /**
     * L'affichage du tournoi sur le réseau du club (ADR-0097) : le bureau sert la page d'affichage
     * de CE tournoi, en lecture seule, et montre le QR code de son adresse. Éteint tant que le
     * directeur ne l'allume pas ; l'avertissement reste visible tant que la page est servie, parce
     * que personne n'est authentifié (ADR-0005).
     */
    import { t } from '../../i18n';
    import { StartTournamentDisplay, StopTournamentDisplay, GetTournamentDisplayStatus } from '../../../wailsjs/go/gui/App.js';
    import { BrowserOpenURL, EventsOn } from '../../../wailsjs/runtime/runtime.js';

    const DEFAULT_PORT = 8770;

    /** @type {{ tournamentId: number }} */
    let { tournamentId } = $props();

    /** @type {{ running: boolean, tournamentId: number, addresses: string[], publicAddresses: string[], address: string, url: string, localUrl: string, qr: string, error: string, errorPort: number }} */
    const STOPPED = { running: false, tournamentId: 0, addresses: [], publicAddresses: [], address: '', url: '', localUrl: '', qr: '', error: '', errorPort: 0 };
    let status = $state(STOPPED);

    // Un statut absent (liaison factice, événement sans charge) vaut « éteint » : tout le
    // panneau lit status.running sans garde.
    /** @param {any} s */
    const settle = (s) => (status = s ?? STOPPED);
    let port = $state(DEFAULT_PORT);
    let busy = $state(false);

    // L'erreur arrive sous forme de code : le panneau la dit en clair, la cause système reste
    // au journal.
    const errorText = $derived(
        status.error === 'port_in_use'
            ? $t('direction.network.errPortInUse', { port: status.errorPort })
            : status.error === 'no_tournament'
              ? $t('direction.network.errNoTournament')
              : status.error
                ? $t('direction.network.errFailed')
                : ''
    );
    const addressIsPublic = $derived(status.publicAddresses?.includes(status.address));

    const showsThis = $derived(status.running && status.tournamentId === tournamentId);

    $effect(() => {
        // Sans liaison Go (aperçu, tests), l'appel lève avant toute promesse : l'affichage
        // reste simplement éteint.
        Promise.resolve()
            .then(() => GetTournamentDisplayStatus())
            .then(settle)
            .catch(() => {});
    });

    // Le bureau arrête l'affichage quand le fichier ouvert change : le panneau l'apprend ici.
    $effect(() => {
        try {
            return EventsOn('direction:network-display', settle);
        } catch {
            return undefined;
        }
    });

    /** @param {() => Promise<any>} action */
    async function apply(action) {
        busy = true;
        try {
            settle(await action());
        } catch {
            status = { ...status, error: 'failed' };
        } finally {
            busy = false;
        }
    }

    /** @param {string} [address] */
    function start(address = status.address) {
        apply(() => StartTournamentDisplay({ tournamentId, port: Number(port) || DEFAULT_PORT, address: address || '' }));
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
            <button type="button" data-testid="direction-network-start" disabled={busy} onclick={() => start()}>{$t('direction.network.start')}</button>
        {/if}
        {#if status.running}
            <button type="button" data-testid="direction-network-stop" disabled={busy} onclick={stop}>{$t('direction.network.stop')}</button>
        {/if}
    </div>
    {#if status.error}
        <p class="error">{$t('direction.network.error', { error: errorText })}</p>
    {/if}
    {#if showsThis && !status.url}
        <p class="warning" data-testid="direction-network-nolan">{$t('direction.network.noLan')}</p>
        <button type="button" class="link" onclick={() => BrowserOpenURL(status.localUrl)}>{$t('direction.network.openHere')}</button>
    {/if}
    {#if showsThis && status.url}
        <div class="share">
            <img class="qr" src={status.qr} alt={$t('direction.network.qr')} width="192" height="192" />
            <div>
                <p class="facts">{$t('direction.network.url')}</p>
                <p class="facts path" data-testid="direction-network-url">{status.url}</p>
                {#if addressIsPublic}
                    <p class="warning" data-testid="direction-network-public">{$t('direction.network.publicAddress', { address: status.address })}</p>
                {/if}
                {#if status.addresses.length > 1}
                    <label class="facts">
                        {$t('direction.network.address')}
                        <select data-testid="direction-network-address" value={status.address} disabled={busy} onchange={(e) => start(e.currentTarget.value)}>
                            {#each status.addresses as a (a)}
                                <option value={a}>{a}{status.publicAddresses?.includes(a) ? ' ⚠' : ''}</option>
                            {/each}
                        </select>
                    </label>
                {/if}
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
