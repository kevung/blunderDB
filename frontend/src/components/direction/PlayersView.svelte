<script>
    /*
     * La vue Joueurs (issue #369, fonctionnel.md §4, ux.md §4.1 et §4.2).
     *
     * Le budget commande le dessin : inscrire vingt joueurs à la suite doit se faire au clavier
     * seul — nom, Entrée, nom, Entrée — sans jamais toucher la souris. D'où un champ qui garde
     * le focus et se vide après chaque inscription.
     *
     * L'autocomplétion propose les Players de la base. En choisir un fixe l'orthographe EXACTE
     * que portent ses matchs et pré-remplit la cote avec son PR : c'est le seul lien entre un
     * Participant et un Player, une égalité de nom décidée par le directeur, jamais une
     * inférence, et jamais une fiche « personne » — l'identité que le glossaire refuse.
     */
    import { t } from '../../i18n';
    import { renderLabel, renderSectionName } from './labels.js';

    let {
        rows = [],
        suggestions = [],
        busy = false,
        started = false,
        onAdd = () => {},
        onUpdate = () => {},
        onWithdraw = () => {},
        // Les places d'exemption encore libres et les inscrits qui n'ont pas encore de place
        // (issue #392). Vides en préparation : il n'y a pas de retardataire avant le tirage.
        slots = [],
        infos = [],
        onAddAtSlot = null
    } = $props();

    let name = $state('');
    let club = $state('');
    let rating = $state('');
    let filter = $state('');
    let editing = $state(null);
    let nameInput = $state(null);

    const shown = $derived(filter.trim() ? rows.filter((r) => (r.name + ' ' + (r.club || '')).toLowerCase().includes(filter.trim().toLowerCase())) : rows);

    /* Les Players proposés : ceux dont le nom commence par ce qui est tapé, les plus joués
       d'abord, et jamais ceux déjà inscrits. */
    const matches = $derived.by(() => {
        const q = name.trim().toLowerCase();
        if (q.length < 2) return [];
        const already = new Set(rows.map((r) => r.name.toLowerCase()));
        return suggestions.filter((s) => s.name.toLowerCase().includes(q) && !already.has(s.name.toLowerCase())).slice(0, 6);
    });

    function num(v) {
        const n = parseFloat(String(v).replace(',', '.'));
        return Number.isFinite(n) && n >= 0 ? n : 0;
    }

    /* Où le retardataire entrera. Par défaut la première place libre : c'est le cas de très
       loin le plus fréquent, et la destination est écrite à côté du champ avant de valider.
       « plus tard » reste à un geste, et le moteur ne refait jamais un tirage. */
    let slotKey = $state('');
    $effect(() => {
        const keys = slots.map((s) => s.key);
        if (!keys.includes(slotKey)) slotKey = keys[0] || '';
    });

    const chosenSlot = $derived(slots.find((s) => s.key === slotKey) || null);

    function slotLabel(s) {
        const where = renderSectionName($t, s.section);
        const what = renderLabel($t, s.label);
        return [where, what].filter(Boolean).join(' · ') || s.key;
    }

    async function add() {
        const n = name.trim();
        if (!n) return;
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
    async function pick(s) {
        name = s.name;
        rating = s.pr ? s.pr.toFixed(1) : '';
        await add();
    }

    function startEdit(r) {
        editing = { id: r.id, name: r.name, club: r.club || '', rating: r.rating || '' };
    }

    async function saveEdit() {
        if (!editing) return;
        await onUpdate(editing.id, editing.name.trim(), editing.club.trim(), num(editing.rating));
        editing = null;
    }
</script>

<div class="players">
    <form
        class="entry"
        onsubmit={(e) => {
            e.preventDefault();
            add();
        }}
    >
        <input bind:this={nameInput} bind:value={name} type="text" placeholder={$t('direction.players.name')} disabled={busy} autocomplete="off" />
        <input bind:value={club} type="text" placeholder={$t('direction.players.club')} disabled={busy} />
        <input bind:value={rating} type="text" inputmode="decimal" placeholder={$t('direction.players.rating')} title={$t('direction.players.ratingHint')} disabled={busy} />
        {#if slots.length}
            <!-- Ce que le directeur voit AVANT de valider : où ce joueur va entrer. -->
            <label class="slot" title={$t('direction.players.slotHint')}>
                {$t('direction.players.entersAt')}
                <select bind:value={slotKey}>
                    {#each slots as s (s.key)}
                        <option value={s.key}>{slotLabel(s)}</option>
                    {/each}
                    <option value="">{$t('direction.players.later')}</option>
                </select>
            </label>
        {/if}
        <button type="submit" class="primary" disabled={busy || !name.trim()}>{$t('direction.players.add')}</button>
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
        <input bind:value={filter} type="text" placeholder={$t('direction.players.filter')} class="filter" />
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
                <tr>
                    {#if editing && editing.id === r.id}
                        <td><input bind:value={editing.name} type="text" /></td>
                        <td><input bind:value={editing.club} type="text" /></td>
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
                            {#if r.table}
                                &middot; {$t('direction.proposals.table', { n: r.table })}
                            {/if}
                        </td>
                        <td class="actions">
                            <button type="button" disabled={busy} onclick={() => startEdit(r)}>{$t('direction.players.correct')}</button>
                            {#if r.state !== 'withdrawn'}
                                <button type="button" disabled={busy} onclick={() => onWithdraw(r.id, false)} title={$t('direction.players.withdrawNowHint')}
                                    >{$t('direction.players.withdrawNow')}</button
                                >
                                {#if r.state === 'playing'}
                                    <button type="button" disabled={busy} onclick={() => onWithdraw(r.id, true)} title={$t('direction.players.withdrawLaterHint')}
                                        >{$t('direction.players.withdrawLater')}</button
                                    >
                                {/if}
                            {/if}
                        </td>
                    {/if}
                </tr>
            {/each}
            {#if shown.length === 0}
                <tr><td colspan="8" class="muted">{$t('direction.players.none')}</td></tr>
            {/if}
        </tbody>
    </table>
</div>

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
</style>
