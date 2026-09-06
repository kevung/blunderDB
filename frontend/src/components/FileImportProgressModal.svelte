<script>
    import Modal from './Modal.svelte';
    import { t } from '../i18n';

    let {
        visible = false,
        mode = 'idle',
        totalFiles = 0,
        currentIndex = 0,
        currentFile = '',
        results = { succeeded: 0, failed: 0, skipped: 0, errors: [] },
        report = null,
        onClose,
        onCancel,
        onOpenPosition,
        onAnalyzeRemaining,
        onStartStudyQueue
    } = $props();

    let progressPercent = $derived(totalFiles > 0 ? Math.round((currentIndex / totalFiles) * 100) : 0);
    // Escape only closes once the terminal state is reached (the "Fermer" button
    // is visible) — while importing is in progress, Escape must not silently
    // abandon it: the user has Annuler for that, deliberately.
    let closable = $derived(mode === 'completed');

    function basename(path) {
        if (!path) return '';
        return path.split('/').pop().split('\\').pop();
    }
</script>

<Modal open={visible} onclose={onClose} size="large" layer="top" closeButton={false} closeOnEscape={closable} label={$t('import.fileProgressTitle')}>
    {#if mode === 'importing'}
        <h2 class="modal-title">{$t('import.importingFiles')} <span class="spinner"></span></h2>
        <p class="status-text">{$t('import.importingFileN', { current: currentIndex, total: totalFiles })}</p>
        <p class="current-file" title={currentFile}>{basename(currentFile)}</p>

        <div class="progress-bar-container">
            <div class="progress-bar" style="width: {progressPercent}%"></div>
        </div>
        <p class="progress-text">{progressPercent}%</p>
    {:else if mode === 'completed'}
        <h2 class="modal-title">{$t('import.completedTitle')}</h2>

        <div class="summary">
            <p><strong>{$t('import.finished')}</strong> {$t('import.processedN', { processed: results.succeeded + results.failed + results.skipped, total: totalFiles })}</p>
        </div>

        <div class="stats">
            <div class="stat-item">
                <div class="stat-label">{$t('import.imported')}</div>
                <div class="stat-value">{results.succeeded}</div>
            </div>
            <div class="stat-item">
                <div class="stat-label">{$t('import.skipped')}</div>
                <div class="stat-value">{results.skipped}</div>
            </div>
            <div class="stat-item">
                <div class="stat-label">{$t('import.failed')}</div>
                <div class="stat-value" class:errors={results.failed > 0}>{results.failed}</div>
            </div>
        </div>

        <!-- The end-of-import report (#257): what just came in, rather than
             how many files were read. Absent when the batch could not be
             recorded, which must look like nothing rather than like a
             failure. -->
        {#if report}
            <div class="report">
                <div class="report-line">
                    <span class="report-label">{$t('import.reportNewPositions')}</span>
                    <span class="report-value">{report.positionsSaved}</span>
                </div>
                {#if report.positionsFlagged > 0}
                    <div class="report-line">
                        <span class="report-label">{$t('import.reportFlagged')}</span>
                        <span class="report-value">{report.positionsFlagged}</span>
                    </div>
                {/if}
                <div class="report-line">
                    <span class="report-label">{$t('import.reportPR')}</span>
                    <!-- A PR of zero over zero decisions is the absence of an
                         analysis, not a perfect game: the line says which. -->
                    <span class="report-value">
                        {#if report.decisions > 0}
                            {report.pr.toFixed(2)}
                            <span class="report-note">{$t('import.reportDecisions', { n: report.decisions, who: report.player || $t('import.reportBothPlayers') })}</span>
                        {:else}
                            <span class="report-note">{$t('import.reportNoAnalysis')}</span>
                        {/if}
                    </span>
                </div>
                {#if report.positionsWithoutAnalysis > 0}
                    <div class="report-line">
                        <span class="report-label">{$t('import.reportUnanalysed', { n: report.positionsWithoutAnalysis })}</span>
                        <button class="report-action" onclick={() => onAnalyzeRemaining?.()}>{$t('import.reportAnalyzeNow')}</button>
                    </div>
                {/if}

                <!-- La question qui suit le compte rendu (#259) : « qu'est-ce
                     que je regarde maintenant ? ». Le bouton n'apparaît que
                     s'il y a une réponse — un lot sans rien à revoir ne doit
                     pas proposer un parcours vide. -->
                {#if report.decisions > 0 || report.positionsFlagged > 0}
                    <div class="report-line">
                        <span class="report-label">{$t('studyQueue.offer')}</span>
                        <button class="report-action" onclick={() => onStartStudyQueue?.()}>{$t('studyQueue.start')}</button>
                    </div>
                {/if}

                {#if report.worstDecisions && report.worstDecisions.length > 0}
                    <div class="report-worst">
                        <div class="report-label">{$t('import.reportWorst')}</div>
                        {#each report.worstDecisions as d, i (d.positionId)}
                            <button class="worst-item" onclick={() => onOpenPosition?.(d.positionId)}>
                                <span class="worst-rank">{i + 1}.</span>
                                <span class="worst-kind">{d.isCube ? $t('import.reportCube') : $t('import.reportChecker')}</span>
                                <span class="worst-error">{(d.errorMp / 1000).toFixed(3)}</span>
                                <span class="worst-label">{d.label}</span>
                            </button>
                        {/each}
                    </div>
                {/if}
            </div>
        {/if}

        {#if results.errors.length > 0}
            <div class="error-list">
                {#each results.errors as err, i (i)}
                    <div class="error-item">
                        <span class="error-file">{basename(err.file)}</span>: {err.message}
                    </div>
                {/each}
            </div>
        {/if}
    {/if}
    {#snippet footer()}
        {#if mode === 'importing'}
            <button onclick={onCancel}>{$t('common.cancel')}</button>
        {:else if mode === 'completed'}
            <button onclick={onClose}>{$t('common.close')}</button>
        {/if}
    {/snippet}
</Modal>

<style>
    .report {
        margin-top: var(--space-3);
        border-top: 1px solid var(--color-border);
        padding-top: var(--space-2);
    }
    .report-line {
        display: flex;
        align-items: baseline;
        justify-content: space-between;
        gap: var(--space-2);
        padding: 2px 0;
    }
    .report-label {
        color: var(--color-text-muted);
    }
    .report-value {
        font-weight: 600;
    }
    .report-note {
        font-weight: 400;
        color: var(--color-text-muted);
    }
    .report-worst {
        margin-top: var(--space-2);
    }
    .worst-item {
        display: flex;
        align-items: baseline;
        gap: var(--space-2);
        width: 100%;
        text-align: left;
        background: none;
        border: none;
        border-radius: var(--radius);
        padding: 2px var(--space-1);
        cursor: pointer;
    }
    .worst-item:hover {
        background: var(--color-surface-alt);
    }
    .worst-rank {
        color: var(--color-text-muted);
        min-width: 1.5em;
    }
    .worst-kind {
        color: var(--color-text-muted);
        min-width: 5em;
    }
    .worst-error {
        font-variant-numeric: tabular-nums;
        min-width: 4em;
    }
    .worst-label {
        color: var(--color-text-muted);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }
    .current-file {
        color: var(--color-text-muted);
        font-size: var(--font-size-base);
        margin: 4px 0 0 0;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .progress-bar-container {
        width: 100%;
        background-color: #e0e0e0;
        border-radius: 4px;
        overflow: hidden;
        height: 8px;
    }

    .progress-bar {
        height: 100%;
        background-color: var(--color-text);
        transition: width 0.2s ease;
        border-radius: 4px;
    }

    .progress-text {
        font-size: var(--font-size-base);
        color: var(--color-text-muted);
        text-align: right;
        margin-top: 4px;
    }

    .stats {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 15px;
        margin-top: 10px;
    }

    .stat-item {
        text-align: center;
        padding: 15px;
        background-color: #f5f5f5;
        border-radius: 4px;
        border: 1px solid #ddd;
    }

    .stat-label {
        font-size: var(--font-size-base);
        color: var(--color-text-muted);
        text-transform: uppercase;
        margin-bottom: 5px;
    }

    .stat-value {
        font-size: var(--font-size-stat-figure);
        font-weight: bold;
        color: var(--color-text);
    }

    .stat-value.errors {
        color: #c33;
    }

    .error-list {
        max-height: 150px;
        overflow-y: auto;
        background-color: #fff5f5;
        border: 1px solid #e0c0c0;
        border-radius: 4px;
        padding: 10px;
    }

    .error-item {
        font-size: var(--font-size-base);
        color: #833;
        margin: 4px 0;
        word-break: break-all;
    }

    .error-item .error-file {
        font-weight: 600;
    }
</style>
