<script>
    /*
     * La bande d'horloge (fonctionnel.md §6, ux.md §2.4) : une ligne de texte, sans clignotement
     * ni son ; les avertissements sont un compteur cliquable. Affiche le jour et le temps de jeu
     * (direction.PlayingTime, les nuits exclues) et la fin estimée (sim.Forecast), datée si ce
     * n'est pas aujourd'hui. Absente d'une Direction close.
     */
    import { t, language } from '../../i18n';

    /**
     * @type {{
     *     clock?: import('../../../wailsjs/go/models').database.ClockView | null,
     *     warnings?: number,
     *     onWarnings?: () => void
     * }}
     */
    let { clock = null, warnings = 0, onWarnings = () => {} } = $props();

    /* L'heure avance sans qu'aucun événement ne soit écrit : sans ce battement, la bande
       resterait figée à la dernière décision. */
    let now = $state(new Date());
    $effect(() => {
        const timer = setInterval(() => (now = new Date()), 30000);
        return () => clearInterval(timer);
    });

    const hhmm = $derived(now.toLocaleTimeString($language, { hour: '2-digit', minute: '2-digit' }));

    /** @param {number | undefined} seconds */
    function duration(seconds) {
        if (!seconds || seconds < 0) return '';
        const h = Math.floor(seconds / 3600);
        const m = Math.floor((seconds % 3600) / 60);
        return h > 0 ? `${h} h ${String(m).padStart(2, '0')}` : `${m} min`;
    }

    /**
     * Une heure, précédée de son jour quand ce n'est pas aujourd'hui : « pause à 12:30 »,
     * « fin estimée : ven. 02:00 ».
     *
     * @param {string | undefined} iso
     */
    function at(iso) {
        if (!iso) return '';
        const d = new Date(iso);
        if (Number.isNaN(d.getTime())) return '';
        const time = d.toLocaleTimeString($language, { hour: '2-digit', minute: '2-digit' });
        if (d.toDateString() === now.toDateString()) return time;
        return `${d.toLocaleDateString($language, { weekday: 'short' })} ${time}`;
    }
</script>

{#if clock && !clock.finished}
    <div class="clock">
        <span class="time">{hhmm}</span>
        {#if clock.day > 1}
            <span data-testid="direction-clock-played">&middot; {$t('direction.clock.day', { n: clock.day })} &middot; {$t('direction.clock.playing', { d: duration(clock.playingSeconds) })}</span>
        {:else if clock.elapsedSeconds}
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
            <span data-testid="direction-clock-break">&middot; {$t('direction.clock.break', { at: at(clock.nextBreak) })}</span>
        {/if}
        {#if clock.estimatedEnd}
            <span data-testid="direction-clock-end" title={$t('direction.clock.endHint')}>&middot; {$t('direction.clock.end', { at: at(clock.estimatedEnd) })}</span>
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
