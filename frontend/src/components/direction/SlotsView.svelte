<script>
    /*
     * Les emplacements (issues #381-#384, ADR-0047 §7).
     *
     * C'est tout l'objet de la décision : un tournoi dirigé crée ses Matchs avant qu'ils soient
     * joués, et chaque match lancé est un emplacement qu'une transcription ou un import vient
     * remplir. Un tournoi de club sans enregistrement les laisse vides ; un tournoi à
     * transcription obligatoire les remplit tous, et le directeur obtient ce que personne n'a
     * aujourd'hui : les matchs d'un tournoi rangés par ronde et par joueur, analysés.
     *
     * Une règle commande tout l'écran : RIEN NE SE RATTACHE PAR DÉDUCTION. Une coïncidence de
     * noms est une suggestion que le directeur accepte, jamais une décision du logiciel. D'où un
     * bouton par ligne, et jamais de rattachement en masse automatique.
     */
    import { t } from '../../i18n';
    import { renderLabel } from './labels.js';

    let { slots = [], unattached = [], busy = false, onTranscribe = () => {}, onAttach = () => {}, onDetach = () => {}, onOpenMatch = () => {} } = $props();

    let bannerDismissed = $state(false);

    const suggested = $derived(unattached.filter((u) => u.suggestSlot));
    const showBanner = $derived(!bannerDismissed && unattached.length > 0);

    const filled = $derived(slots.filter((s) => s.matchId).length);
</script>

<div class="slots">
    {#if showBanner}
        <!-- Le cas réel : trois jours après le tournoi, un joueur envoie douze fichiers. Le
             bandeau les montre avec leur emplacement suggéré, un geste par ligne. -->
        <div class="banner">
            <span>{$t('direction.slots.unattached', { n: unattached.length })}</span>
            <span class="grow"></span>
            <button type="button" onclick={() => (bannerDismissed = true)}>{$t('direction.slots.dismiss')}</button>
        </div>
        <ul class="unattached">
            {#each unattached as u (u.matchId)}
                <li>
                    <span class="names">{u.player1} – {u.player2}</span>
                    {#if u.length}
                        <span class="muted">{$t('direction.proposals.points', { n: u.length })}</span>
                    {/if}
                    {#if u.date}
                        <span class="muted">{u.date}</span>
                    {/if}
                    <span class="grow"></span>
                    {#if u.suggestSlot}
                        <span class="muted">{u.slotLabel}</span>
                        <button type="button" class="primary" disabled={busy} onclick={() => onAttach(u.suggestSlot, u.matchId)}>{$t('direction.slots.attach')}</button>
                    {:else}
                        <span class="muted">{$t('direction.slots.noSuggestion')}</span>
                    {/if}
                </li>
            {/each}
        </ul>
        {#if suggested.length === 0}
            <p class="muted">{$t('direction.slots.noneSuggested')}</p>
        {/if}
    {/if}

    <h3>{$t('direction.slots.title', { filled, total: slots.length })}</h3>

    <table>
        <thead>
            <tr>
                <th>{$t('direction.slots.where')}</th>
                <th>{$t('direction.slots.players')}</th>
                <th>{$t('direction.standings.note')}</th>
                <th>{$t('direction.slots.match')}</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            {#each slots as s (s.slotId)}
                <tr class:flagged={s.disagreement}>
                    <td class="muted">{renderLabel($t, s.label)}</td>
                    <td class="names">{s.aName} – {s.bName}</td>
                    <td class="muted">
                        {#if s.done}
                            {s.winnerName}
                            {#if s.scoreA || s.scoreB}
                                {s.scoreA}–{s.scoreB}
                            {/if}
                        {/if}
                    </td>
                    <td>
                        {#if s.matchId}
                            <button type="button" class="link" onclick={() => onOpenMatch(s.matchId)}>{$t('direction.slots.open')}</button>
                            {#if s.disagreement}
                                <!-- L'écart est MONTRÉ, jamais résolu : pendant un tournoi
                                     c'est la parole du directeur qui fait foi. -->
                                <span class="warn" title={s.disagreement}>⚠ {$t('direction.slots.disagrees')}</span>
                            {/if}
                        {:else if s.draftId}
                            <span class="muted">{$t('direction.slots.draft')}</span>
                        {:else}
                            <span class="muted">—</span>
                        {/if}
                    </td>
                    <td class="actions">
                        {#if s.matchId}
                            <button type="button" disabled={busy} onclick={() => onDetach(s.slotId)}>{$t('direction.slots.detach')}</button>
                        {:else if !s.draftId}
                            <button type="button" disabled={busy} onclick={() => onTranscribe(s.slotId)} title={$t('direction.slots.transcribeHint')}>{$t('direction.slots.transcribe')}</button>
                        {/if}
                    </td>
                </tr>
            {/each}
            {#if slots.length === 0}
                <tr><td colspan="5" class="muted">{$t('direction.slots.none')}</td></tr>
            {/if}
        </tbody>
    </table>
</div>

<style>
    .slots {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: var(--space-2);
        min-width: 0;
    }

    .banner {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        padding: var(--space-1) var(--space-2);
        border: 1px solid var(--color-primary);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        font-size: var(--font-size-small);
    }

    .unattached {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 1px;
        max-height: 14rem;
        overflow-y: auto;
    }

    .unattached li {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        padding: 0.15rem var(--space-2);
        font-size: var(--font-size-small);
    }

    h3 {
        margin: var(--space-1) 0 0;
        font-size: var(--font-size-base);
        font-weight: 600;
        color: var(--color-text);
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

    tr.flagged td {
        border-bottom-color: var(--color-danger);
    }

    .names {
        font-weight: 600;
    }

    .muted {
        color: var(--color-text-muted);
    }

    .warn {
        color: var(--color-danger);
        margin-left: 0.3rem;
    }

    .grow {
        flex: 1;
    }

    .actions {
        text-align: right;
    }

    button {
        padding: 0.05rem 0.5rem;
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

    button.link {
        border-color: transparent;
        color: var(--color-primary);
        padding-left: 0;
    }

    button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
</style>
