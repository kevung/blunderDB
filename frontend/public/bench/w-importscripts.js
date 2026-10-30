importScripts('w-dep-classic.js');
self.onmessage = () => self.postMessage({ pong: true, kind: 'importScripts', dep: self.depClassic, self: String(self.location) });
