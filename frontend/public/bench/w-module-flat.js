self.onmessage = (e) => self.postMessage({ pong: true, kind: 'module-flat', self: String(self.location) });
