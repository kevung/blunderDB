<script>
    import { t } from '../i18n';
    import { logger } from '../utils/logger.js';
    import { AnalysisMETStatus } from '../../wailsjs/go/database/Database.js';

    // "MET différente" (ADR-0068): a match-play analysis valued with another
    // table than the library's current one, left out of the statistics. Read
    // each time the analysis changes; nothing is written.
    let { positionId = 0, analysis = null } = $props();
    let status = $state(null);

    $effect(() => {
        const id = positionId;
        if (!id || !analysis) {
            status = null;
            return;
        }
        let stale = false;
        // A rejected or missing binding hides the badge, never the panel.
        Promise.resolve()
            .then(() => AnalysisMETStatus(id))
            .then((s) => {
                if (!stale) status = s;
            })
            .catch((e) => {
                logger.error('METBadge: status failed', e);
                if (!stale) status = null;
            });
        return () => {
            stale = true;
        };
    });
</script>

{#if status?.different}
    <p class="met-different" title={$t('analysis.metDifferentTitle', { name: status.name, current: status.current })}>
        {$t('analysis.metDifferent', { name: status.name })}
    </p>
{/if}

<style>
    .met-different {
        margin: 4px 0;
        font-size: var(--font-size-small);
        color: var(--color-danger);
    }
</style>
