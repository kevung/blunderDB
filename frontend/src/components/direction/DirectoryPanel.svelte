<script>
    /*
     * L'annuaire (issue #391, fonctionnel.md §4.2).
     *
     * Un directeur de club dirige les mêmes trente personnes tous les mois. Retaper leurs noms à
     * chaque tournoi est le premier abandon possible du logiciel : d'où « reprendre les inscrits
     * du mois dernier », qui est UN clic une fois le bloc ouvert.
     *
     * Le bloc est replié par défaut. Un directeur qui inscrit son premier tournoi n'a rien à
     * reprendre, et l'écran des joueurs doit rester ce qu'il est — un champ, une liste.
     */
    import { t } from '../../i18n';

    let { sources = [], entries = [], busy = false, onTake = () => {}, onExport = () => {}, onParse = async () => null, onImport = () => {} } = $props();

    let open = $state(false);
    let pasted = $state('');
    let preview = $state(null);

    async function read() {
        preview = await onParse(pasted);
    }

    function confirmImport() {
        const rows = preview?.rows || [];
        preview = null;
        pasted = '';
        open = true;
        onImport(rows);
    }
</script>

<section class="directory">
    <button type="button" class="head" onclick={() => (open = !open)} title={$t('direction.directory.hint')}>
        <span class="chevron">{open ? '▾' : '▸'}</span>
        {$t('direction.directory.title', { n: entries.length })}
    </button>

    {#if open}
        <div class="body">
            {#if sources.length}
                <h4>{$t('direction.directory.takeAgain')}</h4>
                <ul class="sources">
                    {#each sources as s (s.tournamentId)}
                        <li>
                            <span class="s-name">{s.name}</span>
                            <span class="s-meta">
                                {#if s.date}{s.date} &middot;{/if}
                                {$t('direction.directory.entrants', { n: s.entrants })}
                            </span>
                            <span class="grow"></span>
                            <button type="button" disabled={busy || s.entrants === 0} onclick={() => onTake(s.tournamentId)}>
                                {$t('direction.directory.take')}
                            </button>
                        </li>
                    {/each}
                </ul>
            {:else}
                <p class="muted">{$t('direction.directory.empty')}</p>
            {/if}

            <div class="csv">
                <button type="button" onclick={onExport} title={$t('direction.directory.exportHint')}>
                    {$t('direction.directory.export')}
                </button>
                <span class="muted">{$t('direction.directory.importHint')}</span>
            </div>
            <textarea bind:value={pasted} rows="3" placeholder={$t('direction.directory.paste')}></textarea>
            <div class="csv">
                <button type="button" disabled={!pasted.trim()} onclick={read}>{$t('direction.directory.read')}</button>
            </div>

            {#if preview}
                <!-- L'aperçu, avant que quoi que ce soit n'entre : lire n'écrit rien du tout. -->
                <div class="preview">
                    <p>{$t('direction.directory.willEnter', { n: preview.rows.length })}</p>
                    {#if (preview.skipped || []).length}
                        <p class="muted">
                            {#each preview.skipped as s (s.line)}
                                {$t('direction.directory.skipped', { line: s.line, text: s.text || '' })}
                            {/each}
                        </p>
                    {/if}
                    {#if preview.errors.length}
                        <ul class="errors">
                            {#each preview.errors as e (e.line + e.code)}
                                <li>{$t(`direction.directory.errors.${e.code}`, { line: e.line, text: e.text || '' })}</li>
                            {/each}
                        </ul>
                    {/if}
                    <div class="csv">
                        <button type="button" class="primary" disabled={busy || preview.rows.length === 0} onclick={confirmImport}>
                            {$t('direction.directory.confirm')}
                        </button>
                        <button type="button" onclick={() => (preview = null)}>{$t('common.cancel')}</button>
                    </div>
                </div>
            {/if}
        </div>
    {/if}
</section>

<style>
    .directory {
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        margin: var(--space-1) 0;
    }

    .head {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        width: 100%;
        padding: 0.35rem 0.6rem;
        border: none;
        background: none;
        color: var(--color-text);
        cursor: pointer;
        text-align: left;
    }

    .chevron {
        color: var(--color-text-muted);
    }

    .body {
        display: flex;
        flex-direction: column;
        gap: var(--space-1);
        padding: 0 0.6rem 0.6rem;
    }

    h4 {
        margin: 0;
        font-size: var(--font-size-small);
        font-weight: 600;
        color: var(--color-text-muted);
    }

    .sources {
        list-style: none;
        margin: 0;
        padding: 0;
        display: flex;
        flex-direction: column;
        gap: 0.25rem;
    }

    .sources li {
        display: flex;
        align-items: center;
        gap: var(--space-1);
    }

    .s-name {
        font-weight: 600;
    }

    .s-meta,
    .muted {
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .grow {
        flex: 1;
    }

    .csv {
        display: flex;
        align-items: center;
        gap: var(--space-1);
        flex-wrap: wrap;
    }

    textarea {
        width: 100%;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface-alt);
        color: var(--color-text);
        padding: 0.3rem;
        resize: vertical;
    }

    .preview {
        border-top: 1px solid var(--color-border);
        padding-top: var(--space-1);
    }

    .preview p {
        margin: 0 0 0.3rem;
    }

    .errors {
        margin: 0 0 0.4rem;
        padding-left: 1.1rem;
        color: var(--color-danger);
        font-size: var(--font-size-small);
    }

    button {
        padding: 0.2rem 0.6rem;
        border: 1px solid var(--color-border);
        border-radius: var(--radius);
        background: var(--color-surface);
        color: var(--color-text);
        cursor: pointer;
    }

    button.head {
        border: none;
        background: none;
    }

    button.primary {
        border-color: var(--color-primary);
    }
</style>
