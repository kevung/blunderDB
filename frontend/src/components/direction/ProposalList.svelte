<script>
    /*
     * La file des propositions (issue #370, tasks/nicomaque/ux.md §2.3 et §4.1).
     *
     * C'est là que le directeur passe 95 % de son temps, et le budget de gestes le dit :
     * confirmer une proposition est UN clic, tout lancer en fait DEUX quel que soit le nombre.
     * La souris est première ; `j`/`k`/Entrée accélèrent pour qui les connaît, et l'infobulle
     * les enseigne — un directeur occasionnel ne doit rien avoir à retenir.
     *
     * Ce que la file ne fait pas : refuser. Le moteur propose, le directeur décide (ADR-0047).
     * D'où « apparier à la main », toujours offert, et « ignorer pour l'instant », qui n'écrit
     * rien du tout — la proposition revient identique au prochain appel, puisqu'elle est
     * déterministe.
     */
    import { t } from '../../i18n';
    import { SvelteSet } from 'svelte/reactivity';
    import { proposalLabel, actionKey, renderWarning, isRepair } from './labels.js';

    let { proposals = [], players = [], busy = false, onConfirm = () => {}, onConfirmAll = () => {}, onManual = () => {} } = $props();

    /* Le compte à rebours d'une micro-ronde (issue #388). L'échéance vient du moteur ; ce qui
       se compte ici, c'est le temps qui reste, et il faut donc un battement de seconde — aucun
       événement ne sera écrit d'ici là. */
    let tick = $state(Date.now());
    $effect(() => {
        const timer = setInterval(() => (tick = Date.now()), 1000);
        return () => clearInterval(timer);
    });

    function remaining(until) {
        if (!until) return '';
        const at = new Date(until).getTime();
        if (!Number.isFinite(at) || at < 946684800000) return '';
        const left = Math.max(0, Math.round((at - tick) / 1000));
        const m = Math.floor(left / 60);
        return `${m}:${String(left % 60).padStart(2, '0')}`;
    }

    /* « Attendre » n'est pas une proposition : c'est le moteur qui dit qu'il n'y a rien à
       faire, et sa raison intéresse le directeur. On la montre, sans bouton. */
    const actionable = $derived(proposals.filter((a) => a.kind !== 'wait'));
    const waiting = $derived(proposals.filter((a) => a.kind === 'wait'));

    let selected = $state(0);
    const ignored = new SvelteSet();
    let confirming = $state(false);
    let manualOpen = $state(false);
    let manualA = $state('');
    let manualB = $state('');
    let manualLength = $state(0);
    let manualTable = $state(0);

    const shown = $derived(actionable.filter((a) => !ignored.has(actionKey(a))));

    $effect(() => {
        if (selected >= shown.length) selected = Math.max(0, shown.length - 1);
    });

    function playerName(id) {
        const p = players.find((x) => x.id === id);
        return p ? p.name : id;
    }

    function ignore(a) {
        ignored.add(actionKey(a));
    }

    /* Tout lancer demande une confirmation avec la liste : c'est le seul geste de la page qui
       agit sur plusieurs matchs d'un coup, et le seul qu'on ne peut pas défaire d'un clic. */
    function askConfirmAll() {
        confirming = true;
    }

    async function doConfirmAll() {
        confirming = false;
        ignored.clear();
        await onConfirmAll();
    }

    async function startManual() {
        if (!manualA || !manualB || manualA === manualB) return;
        await onManual(manualA, manualB, manualLength, manualTable);
        manualA = '';
        manualB = '';
        manualOpen = false;
    }

    function onKey(e) {
        if (e.target && ['INPUT', 'SELECT', 'TEXTAREA'].includes(e.target.tagName)) return;
        if (e.key === 'j' || e.key === 'ArrowDown') {
            selected = Math.min(selected + 1, shown.length - 1);
            e.preventDefault();
        } else if (e.key === 'k' || e.key === 'ArrowUp') {
            selected = Math.max(selected - 1, 0);
            e.preventDefault();
        } else if (e.key === 'Enter' && shown[selected]) {
            onConfirm(shown[selected]);
            e.preventDefault();
        }
    }
</script>

<svelte:window onkeydown={onKey} />

<section class="proposals">
    <header>
        <h3>{$t('direction.proposals.title', { n: shown.length })}</h3>
        {#if shown.length > 1}
            <button type="button" class="all" disabled={busy} title={$t('direction.proposals.allHint')} onclick={askConfirmAll}>{$t('direction.proposals.all')}</button>
        {/if}
    </header>

    {#if confirming}
        <div class="confirm">
            <p>{$t('direction.proposals.allConfirm', { n: shown.length })}</p>
            <ul>
                {#each shown as a (actionKey(a))}
                    <li class:repair={isRepair(a)}>
                        {#if isRepair(a)}
                            <span class="tag">{$t('direction.proposals.repairTag')}</span>
                        {/if}
                        {proposalLabel($t, a, playerName)}
                    </li>
                {/each}
            </ul>
            <div class="confirm-actions">
                <button type="button" class="primary" onclick={doConfirmAll}>{$t('direction.proposals.confirm')}</button>
                <button type="button" onclick={() => (confirming = false)}>{$t('common.cancel')}</button>
            </div>
        </div>
    {/if}

    <ul class="queue">
        {#each shown as a, i (actionKey(a))}
            <li class:selected={i === selected} class:repair={isRepair(a)} onmouseenter={() => (selected = i)}>
                {#if isRepair(a)}
                    <span class="tag">{$t('direction.proposals.repairTag')}</span>
                {/if}
                <span class="what">{proposalLabel($t, a, playerName)}</span>
                {#if a.length}
                    <span class="meta">{$t('direction.proposals.points', { n: a.length })}</span>
                {/if}
                {#if a.table}
                    <span class="meta">{$t('direction.proposals.table', { n: a.table })}</span>
                {:else if a.reason === 'waiting_table'}
                    <span class="meta warn">{$t('direction.reason.waiting_table')}</span>
                {/if}
                {#if a.warn}
                    <!-- Le moteur a remarqué quelque chose sur CETTE proposition et ne bloque
                         rien : le bouton reste là, et le directeur décide. -->
                    <span class="meta warn">{renderWarning($t, { code: a.warn, length: a.length }, playerName)}</span>
                {/if}
                <span class="grow"></span>
                <button type="button" class="go" disabled={busy} title={isRepair(a) ? $t('direction.proposals.repairHint') : $t('direction.proposals.launchHint')} onclick={() => onConfirm(a)}>
                    {isRepair(a) ? $t('direction.proposals.cancelMatch') : $t('direction.proposals.launch')}
                </button>
                <button type="button" class="more" title={$t('direction.proposals.ignore')} onclick={() => ignore(a)}>⋯</button>
            </li>
        {/each}
        {#if shown.length === 0}
            <li class="empty">
                {#if waiting.length}
                    {$t(`direction.reason.${waiting[0].reason || 'no_pairing'}`)}
                    {#if remaining(waiting[0].until)}
                        <span class="countdown">{remaining(waiting[0].until)}</span>
                    {/if}
                {:else}
                    {$t('direction.proposals.none')}
                {/if}
            </li>
        {/if}
    </ul>

    <div class="manual">
        {#if manualOpen}
            <select bind:value={manualA}>
                <option value="">{$t('direction.proposals.playerA')}</option>
                {#each players as p (p.id)}
                    <option value={p.id}>{p.name}</option>
                {/each}
            </select>
            <select bind:value={manualB}>
                <option value="">{$t('direction.proposals.playerB')}</option>
                {#each players as p (p.id)}
                    <option value={p.id}>{p.name}</option>
                {/each}
            </select>
            <input type="number" min="0" max="99" bind:value={manualLength} title={$t('direction.proposals.lengthHint')} />
            <input type="number" min="0" max="200" bind:value={manualTable} title={$t('direction.proposals.tableHint')} />
            <button type="button" class="primary" disabled={busy} onclick={startManual}>{$t('direction.proposals.launch')}</button>
            <button type="button" onclick={() => (manualOpen = false)}>{$t('common.cancel')}</button>
        {:else}
            <button type="button" class="link" onclick={() => (manualOpen = true)}>{$t('direction.proposals.manual')}</button>
        {/if}
    </div>
</section>

<style>
    .proposals {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: var(--space-2);
        min-width: 0;
    }

    header {
        display: flex;
        align-items: center;
        gap: var(--space-2);
    }

    h3 {
        margin: 0;
        font-size: var(--font-size-base);
        font-weight: 600;
        color: var(--color-text);
    }

    .queue {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 2px;
    }

    .queue li {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        padding: var(--space-1) var(--space-2);
        border: 1px solid transparent;
        border-radius: var(--radius);
    }

    /* La ligne visée se distingue par une bordure, pas par un fond : le directeur balaye la
       file du regard, et un aplat par ligne survolée ferait clignoter la page. */
    .queue li.selected {
        border-color: var(--color-primary);
    }

    .queue li.empty {
        color: var(--color-text-muted);
        font-size: var(--font-size-small);
    }

    .what {
        font-weight: 600;
    }

    .meta {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
        white-space: nowrap;
    }

    .meta.warn {
        color: var(--color-danger);
    }

    .grow {
        flex: 1;
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

    button.go,
    button.primary,
    button.all {
        border-color: var(--color-primary);
    }

    button.more {
        border-color: transparent;
        color: var(--color-text-muted);
    }

    button.link {
        border-color: transparent;
        color: var(--color-primary);
        padding-left: 0;
    }

    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .confirm {
        border: 1px solid var(--color-primary);
        border-radius: var(--radius);
        padding: var(--space-2);
        background: var(--color-surface-alt);
    }

    .confirm p {
        margin: 0 0 var(--space-1);
        font-size: var(--font-size-small);
    }

    .confirm ul {
        margin: 0 0 var(--space-2);
        padding-left: var(--space-4);
        font-size: var(--font-size-small);
        max-height: 12rem;
        overflow: auto;
    }

    .confirm-actions {
        display: flex;
        gap: var(--space-1);
    }

    .manual {
        display: flex;
        gap: var(--space-1);
        align-items: center;
        flex-wrap: wrap;
    }

    .manual input {
        width: 4rem;
    }

    select,
    input {
        padding: 0.15rem 0.3rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
        font-size: var(--font-size-small);
    }

    /* Le compte à rebours est un chiffre, pas une alarme : il informe, il ne presse pas. */
    .countdown {
        margin-left: var(--space-1);
        font-variant-numeric: tabular-nums;
        color: var(--color-text-muted);
    }

    /* Une réparation n'est pas une proposition ordinaire : elle DÉFAIT. Elle se distingue donc
       à l'œil, sans devenir une alerte — le directeur peut très bien préférer laisser le
       tableau tel qu'il a été joué et le noter à la main. */
    .repair {
        border-left: 3px solid var(--color-danger);
        padding-left: var(--space-1);
    }

    .tag {
        font-size: var(--font-size-small);
        color: var(--color-danger);
        text-transform: lowercase;
    }
</style>
