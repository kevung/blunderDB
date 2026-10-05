<script>
    // The origin of a Match played here: one line, opened onto the revealed
    // seed and its fingerprint so anyone can recompute the rolls without
    // trusting the Arbiter. Nothing is drawn for a Match imported or transcribed.
    import { t } from '../i18n';

    /** @type {{ origin: any, player1: string, player2: string }} */
    let { origin, player1, player2 } = $props();

    let cadence = $derived.by(() => {
        const c = origin?.cadence_settings;
        if (!c) return '';
        const reserve = c.reservePerPoint ? $t('match.originReservePerPoint', { s: c.reservePerPoint }) : $t('match.originReserve', { s: c.reserve || 0 });
        return [c.name, reserve, $t('match.originDelay', { s: c.delay || 0 })].filter(Boolean).join(', ');
    });
    let overTime = $derived(origin?.over_time === 1 ? player1 : origin?.over_time === 2 ? player2 : '');
</script>

{#if origin}
    <details class="match-origin" data-testid="match-origin">
        <summary>
            <span class="origin-played">{$t('match.originPlayedHere')}</span>
            {#if origin.stopped_early}<span data-testid="origin-stopped"> · {$t('match.originStopped')}</span>{/if}
            {#if cadence}<span> · {$t('match.originCadence', { cadence })}</span>{/if}
            {#if overTime}<span> · {$t('match.originOverTime', { name: overTime })}</span>{/if}
            {#if origin.bot_level}<span data-testid="origin-bot">
                    · {$t('match.originBot', { level: origin.bot_level })}{#if origin.bot_engine}
                        ({origin.bot_engine}){/if}</span
                >{/if}
        </summary>
        <dl>
            <dt>{$t('match.originStart')}</dt>
            <dd>
                {#if origin.start}<code>{origin.start}</code>{:else}{$t('match.originOpening')}{/if}
            </dd>
            <dt>{$t('match.originSeed')}</dt>
            <dd><code data-testid="origin-seed">{origin.dice_seed}</code></dd>
            <dt>{$t('match.originFingerprint')}</dt>
            <dd><code data-testid="origin-fingerprint">{origin.fingerprint}</code></dd>
        </dl>
    </details>
{/if}

<style>
    .match-origin {
        margin: 4px 0 8px;
        font-size: var(--font-size-small);
        color: var(--text-muted, inherit);
    }
    .match-origin summary {
        cursor: pointer;
    }
    .origin-played {
        font-weight: 600;
    }
    dl {
        display: grid;
        grid-template-columns: max-content 1fr;
        gap: 2px 8px;
        margin: 4px 0 0 14px;
    }
    dd {
        margin: 0;
        min-width: 0;
    }
    code {
        user-select: all;
        overflow-wrap: anywhere;
    }
</style>
