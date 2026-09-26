/**
 * Svelte action: file drag & drop over the whole window.
 *
 *   <main use:fileDrop={{ onDrop, onOverlayChange }}>
 *
 * Paths come from Wails (OnFileDrop — the browser drop event carries none); the highlight from the
 * browser's dragover/dragleave/drop on `window`. On Linux the WebView must keep receiving the drop
 * (DisableWebViewDrop false, internal/gui/run.go): this action only listens, never cancels.
 *
 * @param {HTMLElement} _node - unused: the listeners are window-wide
 * @param {Object} params
 * @param {Function} params.onDrop          - (x, y, paths) with the native paths, from Wails
 * @param {Function} params.onOverlayChange - (visible) when the drag-over highlight should show/hide
 */
import { OnFileDrop, OnFileDropOff } from '../../wailsjs/runtime/runtime.js';

export function fileDrop(_node, params) {
    let current = params;
    let overlayShown = false;
    let dragCounter = 0;

    function setOverlay(visible) {
        if (visible === overlayShown) return;
        overlayShown = visible;
        current.onOverlayChange(visible);
    }
    function onDragOver(e) {
        e.preventDefault();
        if (!overlayShown) {
            dragCounter++;
            setOverlay(true);
        }
    }
    function onDragLeave() {
        dragCounter--;
        if (dragCounter <= 0) {
            dragCounter = 0;
            setOverlay(false);
        }
    }
    function onDrop() {
        dragCounter = 0;
        setOverlay(false);
    }

    OnFileDrop((x, y, paths) => current.onDrop(x, y, paths), false);
    window.addEventListener('dragover', onDragOver);
    window.addEventListener('dragleave', onDragLeave);
    window.addEventListener('drop', onDrop);

    return {
        update(next) {
            current = next;
        },
        destroy() {
            window.removeEventListener('dragover', onDragOver);
            window.removeEventListener('dragleave', onDragLeave);
            window.removeEventListener('drop', onDrop);
            OnFileDropOff();
        }
    };
}
