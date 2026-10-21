<!--
  Every application-level modal, mounted once for the app's lifetime from the
  moment its chunk is loaded (lazyComponents.js). Each then stays in the tree
  and shows itself on its own `visible` prop; the keyboard
  handling (Escape, focus trap) lives in Modal.svelte, not here. App.svelte
  only places <ModalHost />; the state driving each modal comes from the stores
  and services imported below, never through props.
-->
<script>
    import { activeModal, MODAL, closeModal } from '../stores/uiStore.js';
    import {
        showImportProgressModalStore,
        importModalModeStore,
        importAnalysisStore,
        importResultStore,
        showFileImportModalStore,
        fileImportModeStore,
        fileImportTotalFilesStore,
        fileImportCurrentIndexStore,
        fileImportCurrentFileStore,
        fileImportResultsStore,
        fileImportReportStore,
        fileImportJournalStore,
        fileImportInterruptedStore,
        fileImportProgressStore
    } from '../stores/importModalStore.js';
    import { exportModalModeStore, exportPositionCountStore, exportMetadataStore, exportOptionsStore, exportMatchesStore, exportDecksStore, exportScopeStore } from '../stores/exportModalStore.js';
    import { closeWarningModal, warningMessageStore, protectedCopyPathStore, protectedCopyErrorStore, unlockProtectedCopy, cancelProtectedCopy } from '../services/databaseService.js';
    import {
        handleImportCommit,
        handleImportCancel,
        handleImportClose,
        handleFileImportCancel,
        handleFileImportMinimize,
        handleFileImportClose,
        openImportedPosition,
        resumeInterruptedImport,
        openJournalMatch,
        analyzeRemainingAfterImport
    } from '../services/importService.js';
    import { handleExportCommit, handleExportCancel, setExportScope, openExportPositions, openExportMatches, planExport, plannedIds } from '../services/exportService.js';
    import { toggleHelpModal } from '../services/keyboardService.js';
    import { confirmModalStore, resolveConfirm } from '../services/confirmService.js';

    import WarningModal from './WarningModal.svelte';
    import ProtectedCopyModal from './ProtectedCopyModal.svelte';
    import { beginStudyQueueFromReport } from '../services/studyQueueService.js';
    import { modals } from './lazyComponents.js';

    // Every modal but the two a database open can raise at once (warning, protected copy)
    // is its own chunk: fetched when first opened, or by the preload once the window is
    // usable. Once loaded it stays mounted, as a static one would.
    const loaded = modals.components;
    $effect(() => {
        const modal = $activeModal;
        if (modal) modals.load(modal);
        if ($showImportProgressModalStore) modals.load('importProgress');
        if ($showFileImportModalStore) modals.load('fileImportProgress');
    });
</script>

{#if $loaded.goToPosition}
    {@const Modal = $loaded.goToPosition}
    <Modal visible={$activeModal === MODAL.GO_TO_POSITION} onClose={() => closeModal()} />
{/if}

{#if $loaded.trash}
    {@const Modal = $loaded.trash}
    <Modal visible={$activeModal === MODAL.TRASH} onClose={() => closeModal()} />
{/if}
{#if $loaded.resumeBatch}
    {@const Modal = $loaded.resumeBatch}
    <Modal visible={$activeModal === MODAL.RESUME_BATCH} onClose={() => closeModal()} />
{/if}

{#if $loaded.cubeMatrix}
    {@const Modal = $loaded.cubeMatrix}
    <Modal visible={$activeModal === MODAL.CUBE_MATRIX} onClose={() => closeModal()} />
{/if}

{#if $loaded.tags}
    {@const Modal = $loaded.tags}
    <Modal visible={$activeModal === MODAL.TAGS} onClose={() => closeModal()} />
{/if}

{#if $loaded.lessonEditor}
    {@const Modal = $loaded.lessonEditor}
    <Modal visible={$activeModal === MODAL.LESSON_EDITOR} onClose={() => closeModal()} />
{/if}

{#if $loaded.log}
    {@const Modal = $loaded.log}
    <Modal visible={$activeModal === MODAL.LOG} onClose={() => closeModal()} />
{/if}
{#if $loaded.contactSheet}
    {@const Modal = $loaded.contactSheet}
    <Modal visible={$activeModal === MODAL.CONTACT_SHEET} onClose={() => closeModal()} />
{/if}

<WarningModal message={$warningMessageStore} visible={$activeModal === MODAL.WARNING} onClose={closeWarningModal} />
<WarningModal
    message={$confirmModalStore?.message || ''}
    visible={$confirmModalStore !== null}
    mode="confirm"
    confirmLabel={$confirmModalStore?.confirmLabel || ''}
    cancelLabel={$confirmModalStore?.cancelLabel || ''}
    choices={$confirmModalStore?.choices || []}
    tone={$confirmModalStore?.tone || 'danger'}
    onChoose={(v) => resolveConfirm(v)}
    onClose={() => resolveConfirm(false)}
    onConfirm={() => resolveConfirm(true)}
/>

<ProtectedCopyModal visible={$activeModal === MODAL.PROTECTED_COPY} fileName={$protectedCopyPathStore} error={$protectedCopyErrorStore} onSubmit={unlockProtectedCopy} onCancel={cancelProtectedCopy} />

{#if $loaded.importProgress}
    {@const Modal = $loaded.importProgress}
    <Modal
        visible={$showImportProgressModalStore}
        mode={$importModalModeStore}
        analysis={$importAnalysisStore}
        result={$importResultStore}
        onCancel={handleImportCancel}
        onCommit={handleImportCommit}
        onClose={handleImportClose}
    />
{/if}

{#if $loaded.fileImportProgress}
    {@const Modal = $loaded.fileImportProgress}
    <Modal
        visible={$showFileImportModalStore}
        mode={$fileImportModeStore}
        totalFiles={$fileImportTotalFilesStore}
        currentIndex={$fileImportCurrentIndexStore}
        currentFile={$fileImportCurrentFileStore}
        results={$fileImportResultsStore}
        report={$fileImportReportStore}
        journal={$fileImportJournalStore}
        interrupted={$fileImportInterruptedStore !== null}
        onResume={resumeInterruptedImport}
        onOpenMatch={openJournalMatch}
        progress={$fileImportProgressStore}
        onMinimize={handleFileImportMinimize}
        onCancel={handleFileImportCancel}
        onClose={handleFileImportClose}
        onOpenPosition={openImportedPosition}
        onAnalyzeRemaining={analyzeRemainingAfterImport}
        onStartStudyQueue={beginStudyQueueFromReport}
    />
{/if}

{#if $loaded.exportDatabase}
    {@const Modal = $loaded.exportDatabase}
    <Modal
        visible={$activeModal === MODAL.EXPORT_DATABASE}
        mode={$exportModalModeStore}
        positionCount={$exportPositionCountStore}
        matches={$exportMatchesStore}
        decks={$exportDecksStore}
        chooseScope={$exportScopeStore.chooseable}
        onScope={setExportScope}
        onOpenPositions={openExportPositions}
        onOpenMatches={openExportMatches}
        planFor={planExport}
        planIdsFor={plannedIds}
        bind:metadata={$exportMetadataStore}
        bind:exportOptions={$exportOptionsStore}
        onCancel={handleExportCancel}
        onExport={handleExportCommit}
    />
{/if}

{#if $loaded.help}
    {@const Modal = $loaded.help}
    <Modal visible={$activeModal === MODAL.HELP} onClose={toggleHelpModal} />
{/if}
{#if $loaded.config}
    {@const Modal = $loaded.config}
    <Modal visible={$activeModal === MODAL.CONFIG} onClose={() => closeModal()} />
{/if}

{#if $loaded.tour}
    {@const Modal = $loaded.tour}
    <Modal visible={$activeModal === MODAL.TOUR} onClose={() => closeModal()} />
{/if}
