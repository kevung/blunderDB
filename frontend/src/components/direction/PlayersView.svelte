<script>
    import { confirmAction } from '../../services/confirmService.js';
    /*
     * La vue Joueurs (fonctionnel.md §4, ux.md §4.1, §4.2) : inscription au clavier seul, le
     * champ garde le focus. Choisir un Player fixe l'orthographe et pré-remplit la cote : le
     * seul lien Participant–Player, une égalité de nom décidée par le directeur.
     */
    import { t } from '../../i18n';
    import { renderLabel, renderSectionName } from './labels.js';
    import ContextMenu from '../ContextMenu.svelte';
    import { playerMenu } from '../../services/directionMenus.js';
    import { menuRequest } from '../../services/contextMenuTrigger.js';

    let {
        rows = [],
        suggestions = [],
        busy = false,
        started = false,
        onAdd = () => {},
        onUpdate = () => {},
        onWithdraw = () => {},
        // Le retour d'un retiré est un geste nommé : corriger sa fiche ne le réinscrit plus.
        onReinstate = (/** @type {string} */ _id) => {},
        // Absenter (D7.1) : jusqu'à une heure (until, ISO 8601) ou jusqu'à une ronde (round),
        // jamais les deux — le moteur refuse sinon. Le retour est un second geste, à un clic.
        onAbsent = (/** @type {string} */ _id, /** @type {string} */ _until, /** @type {number} */ _round) => {},
        onReturn = (/** @type {string} */ _id) => {},
        // Les gestes qui sortent de la vue : la table du joueur, son historique.
        onGoTable = /** @type {((table: number, open: boolean) => void) | undefined} */ (undefined),
        onHistory = /** @type {((name: string) => void) | undefined} */ (undefined),
        onManual = /** @type {((id: string) => void) | undefined} */ (undefined),
        onGoEpreuve = /** @type {((event: string) => void) | undefined} */ (undefined),
        // Où le joueur joue en même temps, dans une autre épreuve de la Rencontre.
        elsewhere = /** @type {Record<string, { event: string, table?: number }>} */ ({}),
        // "à la ronde N" n'a de sens qu'au suisse par rondes en cours ; sinon la case ne doit
        // même pas être proposée.
        roundsMode = false,
        // Les places d'exemption encore libres et les inscrits qui n'ont pas encore de place.
        // Vides en préparation : il n'y a pas de retardataire avant le tirage.
        slots = [],
        infos = [],
        onAddAtSlot = null,
        // Les doubles : une paire est deux personnes, un seul Participant « A / B » (ADR-0056).
        pairs = /** @type {Record<string, {name: string, club?: string, rating?: number}[]>} */ ({}),
        onAddPair = (/** @type {{name: string, club: string, rating: number}[]} */ _m, /** @type {number} */ _r) => {},
        onUpdatePair = (/** @type {string} */ _i, /** @type {{name: string, club: string, rating: number}[]} */ _m, /** @type {number} */ _r) => {},
        // La recherche rapide mène à la fiche d'un joueur : le filtre s'applique une fois par
        // demande, puis redevient libre.
        filterRequest = /** @type {{ text: string, seq: number } | null} */ (null)
    } = $props();

    /* Une épreuve en doubles se reconnaît à ses paires ; la case reste libre pour la première. */
    let pairMode = $state(false);
    $effect(() => {
        if (Object.keys(pairs).length) pairMode = true;
    });
    let name2 = $state('');
    let club2 = $state('');
    let rating2 = $state('');
    let pairRating = $state('');
    /* La cote d'entrée d'une paire : la moyenne des cotes connues, corrigeable. */
    const meanRating = $derived.by(() => {
        const known = [num(rating), num(rating2)].filter((x) => x > 0);
        return known.length ? known.reduce((a, b) => a + b, 0) / known.length : 0;
    });

    let name = $state('');
    let club = $state('');
    let rating = $state('');
    let filter = $state('');
    let lastFilterSeq = 0;
    $effect(() => {
        if (filterRequest && filterRequest.seq !== lastFilterSeq) {
            lastFilterSeq = filterRequest.seq;
            filter = filterRequest.text;
        }
    });
    let editing = $state(/** @type {{ id: string, name: string, club: string, rating: number | string } | null} */ (null));
    let nameInput = $state(/** @type {HTMLInputElement | null} */ (null));

    const shown = $derived(filter.trim() ? rows.filter((r) => (r.name + ' ' + (r.club || '')).toLowerCase().includes(filter.trim().toLowerCase())) : rows);

    /* Les Players proposés : ceux dont le nom commence par ce qui est tapé, les plus joués
       d'abord, et jamais ceux déjà inscrits. */
    const matches = $derived.by(() => {
        const q = name.trim().toLowerCase();
        if (q.length < 2) return [];
        const already = new Set(rows.map((r) => r.name.toLowerCase()));
        return suggestions.filter((s) => s.name.toLowerCase().includes(q) && !already.has(s.name.toLowerCase())).slice(0, 6);
    });

    /** @param {number | string} v */
    function num(v) {
        const n = parseFloat(String(v).replace(',', '.'));
        return Number.isFinite(n) && n >= 0 ? n : 0;
    }

    /* Place du retardataire : la première libre par défaut, affichée avant de valider. */
    let slotKey = $state('');
    $effect(() => {
        const keys = slots.map((s) => s.key);
        if (!keys.includes(slotKey)) slotKey = keys[0] || '';
    });

    const chosenSlot = $derived(slots.find((s) => s.key === slotKey) || null);

    /** @param {import('../../../wailsjs/go/models').service.FreeSlot} s */
    function slotLabel(s) {
        const where = renderSectionName($t, s.section);
        const what = renderLabel($t, s.label);
        return [where, what].filter(Boolean).join(' · ') || s.key;
    }

    async function add() {
        const n = name.trim();
        if (!n) return;
        if (pairMode) {
            if (!name2.trim()) return;
            await onAddPair(
                [
                    { name: n, club: club.trim(), rating: num(rating) },
                    { name: name2.trim(), club: club2.trim(), rating: num(rating2) }
                ],
                num(pairRating)
            );
            name = name2 = rating = rating2 = pairRating = '';
            nameInput?.focus();
            return;
        }
        if (chosenSlot && onAddAtSlot) {
            await onAddAtSlot(n, club.trim(), num(rating), chosenSlot.section, chosenSlot.key);
        } else {
            await onAdd(n, club.trim(), num(rating));
        }
        name = '';
        rating = '';
        // Le club reste : dans un tournoi de club, il est le même vingt fois de suite.
        nameInput?.focus();
    }

    /* Choisir un Player fixe l'orthographe et la cote, puis inscrit : un seul geste. */
    /** @param {import('../../../wailsjs/go/models').service.EntrySuggestion} s */
    async function pick(s) {
        name = s.name;
        rating = s.pr ? s.pr.toFixed(1) : '';
        await add();
    }

    /**
     * Un retrait sort le joueur de tous les appariements à venir : on le confirme, la
     * réintégration restant possible ensuite.
     *
     * @param {{ id: string, name: string }} r
     * @param {boolean} after
     */
    async function withdraw(r, after) {
        // Réversible, rien n'est supprimé : un bouton à son nom, pas le « Supprimer » rouge.
        const message = $t(after ? 'direction.players.withdrawLaterConfirm' : 'direction.players.withdrawNowConfirm', { name: r.name });
        if (!(await confirmAction(message, { confirmLabel: $t('direction.players.withdrawDo'), tone: 'primary' }))) return;
        onWithdraw(r.id, after);
    }

    let menu = $state(/** @type {import('../../services/contextMenuTrigger.js').MenuRequest | null} */ (null));

    /** @param {MouseEvent | KeyboardEvent} ev @param {import('../../../wailsjs/go/models').service.ParticipantRow} r */
    function onRowMenu(ev, r) {
        if (editing && editing.id === r.id) return;
        const req = menuRequest(ev, () =>
            playerMenu(
                (k, p) => $t(k, p),
                { ...r, elsewhere: elsewhere[r.id] },
                {
                    busy,
                    onManual,
                    onGoElsewhere: onGoEpreuve,
                    onGoTable,
                    onHistory,
                    onAbsent: () => openAbsent(r),
                    onReturn: (id) => onReturn(id),
                    onWithdraw: (id, after) => onWithdraw(id, after),
                    onReinstate: (id) => onReinstate(id),
                    onEdit: () => startEdit(r)
                }
            )
        );
        if (req) menu = req;
    }

    /** @param {import('../../../wailsjs/go/models').service.ParticipantRow} r */
    function startEdit(r) {
        editing = { id: r.id, name: r.name, club: r.club || '', rating: r.rating || '' };
    }

    async function saveEdit() {
        if (!editing) return;
        const members = pairs[editing.id];
        if (members) {
            // Une paire garde ses personnes ; seule sa cote d'entrée se corrige ici.
            await onUpdatePair(
                editing.id,
                members.map((m) => ({ name: m.name, club: m.club || '', rating: m.rating || 0 })),
                num(editing.rating)
            );
            editing = null;
            return;
        }
        await onUpdate(editing.id, editing.name.trim(), editing.club.trim(), num(editing.rating));
        editing = null;
    }

    /* Le formulaire d'absence : une ligne à la fois, sous le nom du joueur, plutôt qu'une
       fenêtre — la place se lit à côté de la ligne qu'elle concerne (fonctionnel.md §4). */
    let absentEditing = $state(/** @type {string | null} */ (null));
    let absentTime = $state('');
    let absentRound = $state('');

    /* Par défaut, dans une demi-heure : la director n'a rien à taper pour le cas courant. */
    function defaultAbsentTime() {
        const d = new Date(Date.now() + 30 * 60000);
        return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
    }

    /** @param {import('../../../wailsjs/go/models').service.ParticipantRow} r */
    function openAbsent(r) {
        absentEditing = r.id;
        absentTime = defaultAbsentTime();
        absentRound = '';
    }

    /* Une heure du jour devient un instant : aujourd'hui, ou demain si l'heure est déjà passée.
       Des Date jetables, jamais mutées (svelte/prefer-svelte-reactivity) : seul le constructeur
       et des lecteurs (getFullYear…) les touchent. */
    /** @param {string} hhmm */
    function untilFromTime(hhmm) {
        const [h, m] = hhmm.split(':').map(Number);
        const today = new Date();
        const candidate = new Date(today.getFullYear(), today.getMonth(), today.getDate(), h, m, 0, 0).getTime();
        const chosen = candidate <= Date.now() ? candidate + 86400000 : candidate;
        return new Date(chosen).toISOString();
    }

    /** @param {string} id */
    async function confirmAbsentTime(id) {
        if (!absentTime) return;
        await onAbsent(id, untilFromTime(absentTime), 0);
        absentEditing = null;
    }

    /** @param {string} id */
    async function confirmAbsentRound(id) {
        const n = parseInt(absentRound, 10);
        if (!Number.isFinite(n) || n < 1) return;
        await onAbsent(id, '', n);
        absentEditing = null;
    }

    /** @param {import('../../../wailsjs/go/models').service.ParticipantRow} r */
    function absentLabel(r) {
        if (r.absentRound) return $t('direction.players.absentUntilRoundLabel', { n: r.absentRound });
        if (r.absentUntil) {
            const d = new Date(r.absentUntil);
            const hhmm = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
            return $t('direction.players.absentUntilTimeLabel', { time: hhmm });
        }
        return '';
    }
</script>

<div class="players">
    <form
        class="entry"
        data-testid="direction-player-entry"
        onsubmit={(e) => {
            e.preventDefault();
            add();
        }}
    >
        <input bind:this={nameInput} bind:value={name} type="text" placeholder={$t('direction.players.name')} disabled={busy} autocomplete="off" />
        <input bind:value={club} type="text" placeholder={$t('direction.players.club')} disabled={busy} />
        <input bind:value={rating} type="text" inputmode="decimal" placeholder={$t('direction.players.rating')} title={$t('direction.players.ratingHint')} disabled={busy} />
        <label class="pair-toggle" title={$t('direction.players.pairHint')}>
            <input type="checkbox" data-testid="direction-player-pair" bind:checked={pairMode} disabled={busy} />
            {$t('direction.players.pair')}
        </label>
        {#if pairMode}
            <input bind:value={name2} data-testid="direction-player-name2" type="text" placeholder={$t('direction.players.name2')} disabled={busy} autocomplete="off" />
            <input bind:value={club2} type="text" placeholder={$t('direction.players.club')} disabled={busy} />
            <input bind:value={rating2} type="text" inputmode="decimal" placeholder={$t('direction.players.rating')} disabled={busy} />
            <input
                bind:value={pairRating}
                data-testid="direction-player-pair-rating"
                type="text"
                inputmode="decimal"
                placeholder={meanRating ? meanRating.toFixed(2) : $t('direction.players.pairRating')}
                title={$t('direction.players.pairRatingHint')}
                disabled={busy}
            />
        {/if}
        {#if slots.length}
            <!-- Ce que le directeur voit AVANT de valider : où ce joueur va entrer. -->
            <label class="slot" title={$t('direction.players.slotHint')}>
                {$t('direction.players.entersAt')}
                <select data-testid="direction-player-slot" bind:value={slotKey}>
                    {#each slots as s (s.key)}
                        <option value={s.key}>{slotLabel(s)}</option>
                    {/each}
                    <option value="">{$t('direction.players.later')}</option>
                </select>
            </label>
        {/if}
        <button type="submit" class="primary" disabled={busy || !name.trim() || (pairMode && !name2.trim())}>{$t('direction.players.add')}</button>
    </form>

    {#if infos.length}
        <!-- Les inscrits qui ne jouent encore nulle part. La liste est DÉRIVÉE : celui à qui le
             directeur finit par donner une place en disparaît au même instant. -->
        <ul class="infos">
            {#each infos as i (i.player + i.code)}
                <li>
                    {#if i.code === 'enters_at'}
                        {$t('direction.players.willEnter', {
                            player: (rows.find((r) => r.id === i.player) || {}).name || i.player,
                            where: [renderSectionName($t, i.section), renderLabel($t, i.label)].filter(Boolean).join(' · ') || $t('direction.players.phaseN', { n: i.phase + 1 })
                        })}
                    {:else}
                        {$t('direction.players.noEntry', { player: (rows.find((r) => r.id === i.player) || {}).name || i.player })}
                    {/if}
                </li>
            {/each}
        </ul>
    {/if}

    {#if matches.length}
        <ul class="suggestions">
            {#each matches as s (s.name)}
                <li>
                    <button type="button" onclick={() => pick(s)}>
                        <span class="s-name">{s.name}</span>
                        <span class="s-meta">
                            {$t('direction.players.knownFor', { n: s.matches })}
                            {#if s.pr}
                                &middot; PR {s.pr.toFixed(1)}
                            {/if}
                        </span>
                    </button>
                </li>
            {/each}
        </ul>
    {/if}

    <div class="head">
        <input bind:value={filter} type="text" placeholder={$t('direction.players.filter')} class="filter" data-testid="direction-player-filter" />
        <span class="count">{$t('direction.players.count', { n: rows.length })}</span>
    </div>

    <table>
        <thead>
            <tr>
                <th>{$t('direction.players.name')}</th>
                <th>{$t('direction.players.club')}</th>
                <th>{$t('direction.players.rating')}</th>
                {#if started}
                    <th>{$t('direction.players.record')}</th>
                    <th>{$t('direction.players.lives')}</th>
                    <th>{$t('direction.players.opponents')}</th>
                {/if}
                <th>{$t('direction.players.state')}</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            {#each shown as r (r.id)}
                <tr data-testid="direction-player-{r.id}" tabindex="0" oncontextmenu={(ev) => onRowMenu(ev, r)} onkeydown={(ev) => onRowMenu(ev, r)}>
                    {#if editing && editing.id === r.id}
                        {#if pairs[editing.id]}
                            <td>{editing.name}</td>
                            <td>{editing.club}</td>
                        {:else}
                            <td><input bind:value={editing.name} type="text" /></td>
                            <td><input bind:value={editing.club} type="text" /></td>
                        {/if}
                        <td><input bind:value={editing.rating} type="text" inputmode="decimal" /></td>
                        {#if started}
                            <td colspan="3"></td>
                        {/if}
                        <td></td>
                        <td class="actions">
                            <button type="button" class="primary" onclick={saveEdit}>{$t('direction.players.save')}</button>
                            <button type="button" onclick={() => (editing = null)}>{$t('common.cancel')}</button>
                        </td>
                    {:else}
                        <td class="name">{r.name}</td>
                        <td class="muted">{r.club || ''}</td>
                        <td class="muted">{r.rating ? r.rating.toFixed(1) : ''}</td>
                        {#if started}
                            <td class="muted">{r.wins}–{r.losses}</td>
                            <td class="muted">{r.lives}</td>
                            <td class="muted opponents" title={(r.opponents || []).join(', ')}>{(r.opponents || []).join(', ')}</td>
                        {/if}
                        <td class="muted">
                            {$t(`direction.players.states.${r.state}`)}
                            {#if r.state === 'absent'}
                                &middot; {absentLabel(r)}
                            {/if}
                            {#if r.table}
                                &middot; {$t('direction.proposals.table', { n: r.table })}
                            {/if}
                        </td>
                        <td class="actions">
                            <button type="button" data-testid="direction-player-correct" disabled={busy} onclick={() => startEdit(r)}>{$t('direction.players.correct')}</button>
                            {#if r.state === 'withdrawn'}
                                <button type="button" data-testid="direction-player-reinstate" disabled={busy} onclick={() => onReinstate(r.id)} title={$t('direction.players.reinstateHint')}
                                    >{$t('direction.players.reinstate')}</button
                                >
                            {:else if r.state === 'absent'}
                                <button type="button" class="primary" data-testid="direction-player-return" disabled={busy} onclick={() => onReturn(r.id)} title={$t('direction.players.returnHint')}
                                    >{$t('direction.players.return')}</button
                                >
                                <button type="button" data-testid="direction-player-withdraw-now" disabled={busy} onclick={() => withdraw(r, false)} title={$t('direction.players.withdrawNowHint')}
                                    >{$t('direction.players.withdrawNow')}</button
                                >
                            {:else}
                                {#if r.state === 'free'}
                                    <button type="button" data-testid="direction-player-absent" disabled={busy} onclick={() => openAbsent(r)} title={$t('direction.players.absentHint')}
                                        >{$t('direction.players.absent')}</button
                                    >
                                {/if}
                                <button type="button" data-testid="direction-player-withdraw-now" disabled={busy} onclick={() => withdraw(r, false)} title={$t('direction.players.withdrawNowHint')}
                                    >{$t('direction.players.withdrawNow')}</button
                                >
                                {#if r.state === 'playing'}
                                    <button
                                        type="button"
                                        data-testid="direction-player-withdraw-later"
                                        disabled={busy}
                                        onclick={() => withdraw(r, true)}
                                        title={$t('direction.players.withdrawLaterHint')}>{$t('direction.players.withdrawLater')}</button
                                    >
                                {/if}
                            {/if}
                        </td>
                    {/if}
                </tr>
                {#if absentEditing === r.id}
                    <tr class="absent-form" data-testid="direction-player-absent-form">
                        <td colspan="8">
                            <span class="absent-label">{$t('direction.players.absentUntilTime')}</span>
                            <input type="time" bind:value={absentTime} data-testid="direction-player-absent-time" />
                            <button type="button" class="primary" disabled={busy || !absentTime} onclick={() => confirmAbsentTime(r.id)} data-testid="direction-player-absent-confirm-time"
                                >{$t('direction.players.absentConfirm')}</button
                            >
                            {#if roundsMode}
                                <span class="absent-label">{$t('direction.players.absentUntilRound')}</span>
                                <input type="number" min="1" bind:value={absentRound} data-testid="direction-player-absent-round" />
                                <button type="button" disabled={busy || !absentRound} onclick={() => confirmAbsentRound(r.id)} data-testid="direction-player-absent-confirm-round"
                                    >{$t('direction.players.absentConfirm')}</button
                                >
                            {/if}
                            <button type="button" onclick={() => (absentEditing = null)}>{$t('common.cancel')}</button>
                        </td>
                    </tr>
                {/if}
            {/each}
            {#if shown.length === 0}
                <tr><td colspan="8" class="muted">{$t('direction.players.none')}</td></tr>
            {/if}
        </tbody>
    </table>
</div>

{#if menu}
    <ContextMenu x={menu.x} y={menu.y} items={menu.items} onClose={() => (menu = null)} />
{/if}

<style>
    .players {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: var(--space-2);
        min-width: 0;
    }

    .entry,
    .head {
        display: flex;
        gap: var(--space-1);
        align-items: center;
        flex-wrap: wrap;
    }

    input {
        padding: 0.2rem 0.4rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
    }

    .entry input:first-child {
        min-width: 14rem;
    }

    .filter {
        min-width: 10rem;
    }

    .count {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .suggestions {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 1px;
        max-width: 30rem;
    }

    .suggestions button {
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        width: 100%;
        text-align: left;
        padding: 0.2rem 0.4rem;
        border: 1px solid transparent;
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
        cursor: pointer;
    }

    .suggestions button:hover {
        border-color: var(--color-primary);
    }

    .s-name {
        font-weight: 600;
    }

    .s-meta,
    .muted {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    table {
        border-collapse: collapse;
        width: 100%;
        font-size: var(--font-size-small);
    }

    th {
        text-align: left;
        font-weight: 600;
        color: var(--color-text-muted);
        border-bottom: 1px solid var(--color-border);
        padding: 0.2rem 0.4rem;
    }

    td {
        padding: 0.15rem 0.4rem;
        border-bottom: 1px solid var(--color-surface-alt);
    }

    td.name {
        font-weight: 600;
        color: var(--color-text);
    }

    /* Les adversaires rencontrés peuvent être nombreux : la colonne se tronque et l'infobulle
       les donne tous, plutôt que d'étirer la table hors de l'écran. */
    td.opponents {
        max-width: 16rem;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .actions {
        display: flex;
        gap: 0.2rem;
        justify-content: flex-end;
    }

    button {
        padding: 0.1rem 0.5rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
        font-size: var(--font-size-small);
        white-space: nowrap;
    }

    button.primary {
        border-color: var(--color-primary);
    }

    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    /* La destination d'un retardataire se lit à côté du champ, pas dans une fenêtre : le
       directeur la voit en tapant le nom. */
    .slot {
        display: inline-flex;
        align-items: center;
        gap: 0.3rem;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .slot select {
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
        padding: 0.15rem 0.3rem;
    }

    .infos {
        list-style: none;
        margin: 0.2rem 0;
        padding: 0;
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    /* Le formulaire d'absence : une ligne sous celle du joueur, pas une fenêtre — la place se
       lit à côté de la ligne qu'elle concerne. */
    .absent-form td {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        flex-wrap: wrap;
        padding: 0.3rem 0.4rem;
        background: var(--color-surface-alt);
    }

    .absent-label {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .absent-form input[type='time'],
    .absent-form input[type='number'] {
        padding: 0.15rem 0.3rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
    }
</style>
