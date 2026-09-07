<script>
    /*
     * La bande d'horloge (issue #377, fonctionnel.md §6, ux.md §2.4).
     *
     * Un directeur se demande toute la soirée s'il finira avant minuit, et regarde les tables
     * qui traînent. Le moteur sait répondre ; ce qui manquait était l'endroit où le lire.
     *
     * C'est une LIGNE DE TEXTE, pas un tableau de bord. Rien ne clignote, il n'y a pas de son,
     * et les avertissements sont un compteur cliquable qui amène à l'objet concerné plutôt
     * qu'une fenêtre qui interrompt.
     */
    import { t } from '../../i18n';

    let { clock = null, warnings = 0, onWarnings = () => {} } = $props();

    /* L'heure avance sans qu'aucun événement ne soit écrit : sans ce battement, la bande
       resterait figée à la dernière décision. */
    let now = $state(new Date());
    $effect(() => {
        const timer = setInterval(() => (now = new Date()), 30000);
        return () => clearInterval(timer);
    });

    const hhmm = $derived(now.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }));

    function duration(seconds) {
        if (!seconds || seconds < 0) return '';
        const h = Math.floor(seconds / 3600);
        const m = Math.floor((seconds % 3600) / 60);
        return h > 0 ? `${h} h ${String(m).padStart(2, '0')}` : `${m} min`;
    }

    function at(iso) {
        if (!iso) return '';
        const d = new Date(iso);
        return Number.isNaN(d.getTime()) ? '' : d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
    }
</script>

{#if clock}
    <div class="clock">
        <span class="time">{hhmm}</span>
        {#if clock.elapsedSeconds}
            <span>&middot; {duration(clock.elapsedSeconds)}</span>
        {/if}
        <span
            >&middot; {$t('direction.clock.matches', {
                played: clock.played,
                running: clock.running
            })}</span
        >
        {#if clock.minutesPerPoint}
            <span
                >&middot; {$t('direction.clock.pace', {
                    observed: clock.minutesPerPoint.toFixed(1),
                    planned: clock.plannedPerPoint.toFixed(0)
                })}</span
            >
        {/if}
        {#if clock.slowMatches}
            <span class="warn">&middot; {$t('direction.clock.slow', { n: clock.slowMatches })}</span>
        {/if}
        {#if clock.nextBreak}
            <span>&middot; {$t('direction.clock.break', { at: at(clock.nextBreak) })}</span>
        {/if}
        <span class="grow"></span>
        {#if warnings > 0}
            <button type="button" class="warn-btn" onclick={onWarnings}>⚠ {$t('direction.clock.warnings', { n: warnings })}</button>
        {/if}
    </div>
{/if}

<style>
    /* Une ligne, pas un tableau de bord : le directeur la balaye du regard sans s'y arrêter. */
    .clock {
        display: flex;
        align-items: center;
        gap: 0.3rem;
        flex-wrap: wrap;
        padding: 0.2rem var(--space-2);
        border-bottom: 1px solid var(--color-border);
        font-size: var(--font-size-small);
        color: var(--color-text-muted);
    }

    .time {
        font-weight: 600;
        color: var(--color-text);
        font-variant-numeric: tabular-nums;
    }

    .warn {
        color: var(--color-danger);
    }

    .grow {
        flex: 1;
    }

    /* Le compteur amène à l'objet concerné : il n'interrompt pas, il conduit. */
    .warn-btn {
        padding: 0 0.4rem;
        border: 1px solid var(--color-danger);
        border-radius: var(--radius);
        background: transparent;
        color: var(--color-danger);
        cursor: pointer;
        font-size: var(--font-size-small);
    }
</style>
