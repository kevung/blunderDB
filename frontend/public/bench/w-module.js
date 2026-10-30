import { dep } from './w-dep.js';
self.onmessage = (e) => self.postMessage({ pong: true, kind: 'module', dep, self: String(self.location), echo: e.data });
