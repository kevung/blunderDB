<!--
  Every application-level modal, mounted once for the app's lifetime. Each is
  always in the tree and shows itself on its own `visible` prop; the keyboard
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
        fileImportResultsStore
    } from '../stores/importModalStore.js';
    import { exportModalModeStore, exportPositionCountStore, exportMetadataStore, exportOptionsStore, exportMatchesStore } from '../stores/exportModalStore.js';
    import { closeWarningModal, warningMessageStore, protectedCopyPathStore, protectedCopyErrorStore, unlockProtectedCopy, cancelProtectedCopy } from '../services/databaseService.js';
    import { handleImportCommit, handleImportCancel, handleImportClose, handleFileImportCancel, handleFileImportClose } from '../services/importService.js';
    import { handleExportCommit, handleExportCancel } from '../services/exportService.js';
    import { toggleHelpModal } from '../services/keyboardService.js';
    import { confirmModalStore, resolveConfirm } from '../services/confirmService.js';
    import { MODAL_TABLES } from './modalTables.js';

    import HelpModal from './HelpModal.svelte';
    import ConfigModal from './ConfigModal.svelte';
    import TourCatalogModal from './TourCatalogModal.svelte';
    import GoToPositionModal from './GoToPositionModal.svelte';
    import MetModal from './MetModal.svelte';
    import DataTableModal from './DataTableModal.svelte';
    import WarningModal from './WarningModal.svelte';
    import ProtectedCopyModal from './ProtectedCopyModal.svelte';
    import ImportProgressModal from './ImportProgressModal.svelte';
    import FileImportProgressModal from './FileImportProgressModal.svelte';
    import ExportDatabaseModal from './ExportDatabaseModal.svelte';
</script>

<GoToPositionModal visible={$activeModal === MODAL.GO_TO_POSITION} onClose={() => closeModal()} />

<MetModal visible={$activeModal === MODAL.MET} onClose={() => closeModal()} />
{#each Object.entries(MODAL_TABLES) as [modal, tables] (modal)}
    <DataTableModal visible={$activeModal === modal} onClose={() => closeModal()} {tables} />
{/each}

<WarningModal message={$warningMessageStore} visible={$activeModal === MODAL.WARNING} onClose={closeWarningModal} />
<WarningModal
    message={$confirmModalStore?.message || ''}
    visible={$confirmModalStore !== null}
    mode="confirm"
    confirmLabel={$confirmModalStore?.confirmLabel || ''}
    cancelLabel={$confirmModalStore?.cancelLabel || ''}
    onClose={() => resolveConfirm(false)}
    onConfirm={() => resolveConfirm(true)}
/>

<ProtectedCopyModal visible={$activeModal === MODAL.PROTECTED_COPY} fileName={$protectedCopyPathStore} error={$protectedCopyErrorStore} onSubmit={unlockProtectedCopy} onCancel={cancelProtectedCopy} />

<ImportProgressModal
    visible={$showImportProgressModalStore}
    mode={$importModalModeStore}
    analysis={$importAnalysisStore}
    result={$importResultStore}
    onCancel={handleImportCancel}
    onCommit={handleImportCommit}
    onClose={handleImportClose}
/>

<FileImportProgressModal
    visible={$showFileImportModalStore}
    mode={$fileImportModeStore}
    totalFiles={$fileImportTotalFilesStore}
    currentIndex={$fileImportCurrentIndexStore}
    currentFile={$fileImportCurrentFileStore}
    results={$fileImportResultsStore}
    onCancel={handleFileImportCancel}
    onClose={handleFileImportClose}
/>

<ExportDatabaseModal
    visible={$activeModal === MODAL.EXPORT_DATABASE}
    mode={$exportModalModeStore}
    positionCount={$exportPositionCountStore}
    matches={$exportMatchesStore}
    bind:metadata={$exportMetadataStore}
    bind:exportOptions={$exportOptionsStore}
    onCancel={handleExportCancel}
    onExport={handleExportCommit}
/>

<HelpModal visible={$activeModal === MODAL.HELP} onClose={toggleHelpModal} />
<ConfigModal visible={$activeModal === MODAL.CONFIG} onClose={() => closeModal()} />

<TourCatalogModal visible={$activeModal === MODAL.TOUR} onClose={() => closeModal()} />
