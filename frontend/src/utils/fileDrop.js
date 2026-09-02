/**
 * Svelte action: file drag & drop over the whole window.
 *
 *   <main use:fileDrop={{ onDrop, onOverlayChange }}>
 *
 * Wails delivers the dropped paths through its runtime (OnFileDrop — the
 * only way to get native paths; the browser drop event carries none), while
 * the drag-over highlight is driven by the browser's own dragover / dragleave
 * / drop events on `window`. Both are wired for the node's lifetime.
 *
 * On Linux the WebView must keep receiving the drop for OnFileDrop to fire
 * (DisableWebViewDrop stays false, see internal/gui/run.go); this action only
 * listens, it never cancels the drop.
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
