<script>
    // Le panneau Calculs : les tâches durables de toutes les bibliothèques, en cours, en
    // attente et terminées (registre de tâches, ADR-0115).
    import Modal from './Modal.svelte';
    import { t } from '../i18n';
    import { jobsStore, coresStore, runningJobs, waitingJobs, endedJobs, allPaused, pauseJob, resumeJob, cancelJob, removeJob, moveJob, toggleAllPaused } from '../stores/jobsStore.js';
    import { fileImportMinimizedStore, showFileImportModalStore } from '../stores/importModalStore.js';
    import { tabGroupsStore, groupColorIndex, activateGroup, activeGroupId } from '../stores/libraryGroupStore.js';

    let { visible = false, onClose } = $props();

    // Libellé court de chaque genre de tâche ; le registre envoie le code.
    /** @type {Record<string, string>} */
    const KIND_KEYS = {
        'gammonnet-missing': 'calculs.kind.gammonnet',
        'gammonnet-stale': 'calculs.kind.gammonnet',
        'gammonnet-match': 'calculs.kind.gammonnet',
        rollout: 'calculs.kind.rollout',
        'rollout-batch': 'calculs.kind.rolloutBatch',
        maintenance: 'calculs.kind.maintenance',
        'match-stats-fill': 'calculs.kind.maintenance',
        import: 'calculs.kind.import',
        repair: 'calculs.kind.repair',
        transfer: 'calculs.kind.transfer',
        bearoff: 'calculs.kind.bearoff',
        sync: 'calculs.kind.sync'
    };

    /** @param {import('../stores/jobsStore.js').JobInfo} job */
    const kindLabel = (job) => $t(KIND_KEYS[job.kind] ?? 'calculs.kind.other');

    // La maintenance nomme sa ligne par son code (`stats.matches`) ; le reste porte déjà son objet.
    /** @param {import('../stores/jobsStore.js').JobInfo} job */
    const objectLabel = (job) => (job.kind === 'maintenance' || job.kind === 'match-stats-fill' ? $t(`maintenance.line_${String(job.label).replace('.', '_')}`) : job.label);

    /** @param {number} id */
    const libraryName = (id) => $tabGroupsStore.find((g) => g.id === id)?.name || $t('libraryGroups.unnamed');

    /** @param {import('../stores/jobsStore.js').JobInfo} job */
    const percent = (job) => (job.total > 0 ? Math.min(100, Math.round((job.done / job.total) * 100)) : null);

    // Dernière position de la file d'une bibliothèque : on ne descend pas plus bas.
    /** @param {import('../stores/jobsStore.js').JobInfo} job */
    const isLast = (job) => !$waitingJobs.some((j) => j.library === job.library && j.position > job.position);

    // Le compteur mène à la bibliothèque qui porte la tâche.
    /** @param {number} library */
    async function showLibrary(library) {
        if (library > 0 && library !== $activeGroupId) await activateGroup(library);
        onClose?.();
    }
</script>

{#snippet group(/** @type {string} */ id, /** @type {string} */ title, /** @type {import('../stores/jobsStore.js').JobInfo[]} */ list)}
    {#if list.length > 0}
        <section data-testid="jobs-{id}">
            <h3>{title} <span class="n">{list.length}</span></h3>
            <ul class="list">
                {#each list as job (job.id)}
                    {@const pct = percent(job)}
                    <li class="row" data-testid="job-row" data-job-id={job.id} data-state={job.state}>
                        <span class="kind"
                            >{kindLabel(job)}{#if job.state === 'running' && job.yielded}
                                <span class="yielded" data-testid="job-yielded">{$t('calculs.yielded')}</span>{/if}</span
                        >
                        <span class="object" title={objectLabel(job)}>{objectLabel(job)}</span>
                        {#if job.library > 0}
                            <span class="library" style="--group-color: var(--group-color-{groupColorIndex(job.library)})" title={libraryName(job.library)} data-testid="job-library">
                                <span class="chip"></span><span class="name">{libraryName(job.library)}</span>
                            </span>
                        {:else}
                            <span></span>
                        {/if}
                        <progress class="bar" value={pct ?? undefined} max="100" aria-label={$t('calculs.progress')}></progress>
                        {#if job.total > 0}
                            <button type="button" class="count" data-testid="job-count" onclick={() => showLibrary(job.library)}>{job.done} / {job.total}</button>
                        {:else}
                            <span class="count">{job.done > 0 ? job.done : ''}</span>
                        {/if}
                        <span class="actions">
                            {#if id === 'running'}
                                {#if job.state === 'paused'}
                                    <button type="button" data-testid="job-resume" onclick={() => resumeJob(job.id)}>{$t('calculs.resume')}</button>
                                {:else if job.pausable}
                                    <button type="button" data-testid="job-pause" onclick={() => pauseJob(job.id)}>{$t('calculs.pause')}</button>
                                {/if}
                                {#if job.kind === 'import' && $fileImportMinimizedStore}
                                    <button
                                        type="button"
                                        data-testid="job-show"
                                        onclick={() => {
                                            fileImportMinimizedStore.set(false);
                                            showFileImportModalStore.set(true);
                                            onClose?.();
                                        }}>{$t('calculs.show')}</button
                                    >
                                {/if}
                                {#if job.cancelable}
                                    <button type="button" data-testid="job-cancel" onclick={() => cancelJob(job.id)}>{$t('calculs.cancel')}</button>
                                {/if}
                            {:else if id === 'waiting'}
                                {#if job.state === 'paused'}
                                    <span class="end" data-testid="job-paused">{$t('calculs.pause')}</span>
                                    <button type="button" data-testid="job-resume" onclick={() => resumeJob(job.id)}>{$t('calculs.resume')}</button>
                                {/if}
                                <button type="button" data-testid="job-up" disabled={job.position <= 1} onclick={() => moveJob(job.id, job.position - 2)}>{$t('calculs.moveUp')}</button>
                                <button type="button" data-testid="job-down" disabled={isLast(job)} onclick={() => moveJob(job.id, job.position)}>{$t('calculs.moveDown')}</button>
                                <button type="button" data-testid="job-remove" onclick={() => removeJob(job.id)}>{$t('calculs.remove')}</button>
                            {:else}
                                <span class="end" data-testid="job-end">{$t(`calculs.end_${job.state}`)}</span>
                                <button type="button" data-testid="job-remove" onclick={() => removeJob(job.id)}>{$t('calculs.remove')}</button>
                            {/if}
                        </span>
                    </li>
                {/each}
            </ul>
        </section>
    {/if}
{/snippet}

<Modal open={visible} onclose={onClose} size="large" label={$t('calculs.title')}>
    <h2 class="modal-title">{$t('calculs.title')}</h2>
    <div class="head" data-testid="jobs-head">
        {#if $coresStore}
            <span class="cores" data-testid="jobs-cores"><span class="n">{$t('calculs.cores')}</span> {$coresStore.used} / {$coresStore.total}</span>
        {/if}
        <button type="button" data-testid="jobs-pause-all" disabled={$runningJobs.length + $waitingJobs.length === 0} onclick={() => toggleAllPaused()}>
            {$allPaused ? $t('calculs.resumeAll') : $t('calculs.pauseAll')}
        </button>
    </div>

    {#if $jobsStore.length === 0}
        <p class="empty" data-testid="jobs-empty">{$t('calculs.empty')}</p>
    {/if}
    {@render group('running', $t('calculs.running'), $runningJobs)}
    {@render group('waiting', $t('calculs.waiting'), $waitingJobs)}
    {@render group('ended', $t('calculs.ended'), $endedJobs)}
</Modal>

<style>
    .head {
        display: flex;
        align-items: center;
        gap: var(--space-2);
        margin-bottom: var(--space-2);
    }
    .cores {
        font-variant-numeric: tabular-nums;
    }
    .yielded {
        display: block;
        font-size: var(--font-size-small);
        font-style: italic;
    }
    h3 {
        margin: var(--space-2) 0 var(--space-1);
        font-weight: 600;
    }
    .n,
    .empty,
    .end,
    .kind {
        color: var(--color-text-muted);
    }
    .list {
        list-style: none;
        margin: 0;
        padding: 0;
        max-height: calc(40vh / var(--ui-scale, 1));
        overflow-y: auto;
    }
    .row {
        display: grid;
        grid-template-columns: 7em minmax(0, 1fr) 9em 8em 6em auto;
        align-items: center;
        gap: var(--space-2);
        padding: var(--space-1) 0;
        border-bottom: 1px solid var(--color-border);
    }
    .object,
    .name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .library {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        min-width: 0;
    }
    .chip {
        width: 9px;
        height: 9px;
        border-radius: 50%;
        background: var(--group-color);
        flex-shrink: 0;
    }
    .bar {
        width: 100%;
    }
    .count {
        text-align: right;
        font-variant-numeric: tabular-nums;
    }
    button.count {
        background: none;
        border: none;
        padding: 0;
        color: inherit;
        cursor: pointer;
        text-decoration: underline dotted;
    }
    .actions {
        display: inline-flex;
        gap: var(--space-1);
        justify-content: flex-end;
    }
</style>
