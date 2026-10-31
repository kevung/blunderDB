<script>
    // Development only (`import.meta.env.DEV` is false in the published bundle, where vite drops
    // this branch): writes the vision events of the session for the bench.
    import { saveBenchEvents } from '../services/visionBenchExport.js';
    import { logger } from '../utils/logger.js';
    import { VISION_EVENTS_MAX } from '../stores/visionEventsStore.js';

    /** @type {{ events: readonly import('../vision/contract.js').VisionEvent[], calibration: string, source: string }} */
    let { events, calibration, source } = $props();

    async function save() {
        try {
            await saveBenchEvents(events, calibration, source);
        } catch (err) {
            logger.error('could not save the bench events:', err);
        }
    }
</script>

{#if import.meta.env.DEV && events.length > 0}
    <button type="button" class="bench-export" onclick={save}
        >Enregistrer les événements (banc) — {events.length}{events.length >= VISION_EVENTS_MAX ? ' : plein, les plus anciens sont perdus' : ''}</button
    >
{/if}
