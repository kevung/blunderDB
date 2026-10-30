/**
 * Point d'entrée du Worker de Z-vision. Chargé en module ES par
 * `new Worker(new URL('./worker.js', import.meta.url), { type: 'module' })`, la forme que Vite
 * empaquette et que WebKitGTK sert depuis `wails://` (banc V0.1) ; aucun repli Blob.
 */

import { createWorkerCore } from './workerCore.js';

const core = createWorkerCore({ post: (message) => self.postMessage(message) });
self.onmessage = (e) => void core.handle(e.data);
