// Classic worker: pong, ArrayBuffer echo (transferred back), feature probe.
self.onmessage = (e) => {
  const d = e.data;
  if (d && d.buf) { self.postMessage({ buf: d.buf, n: d.buf.byteLength }, [d.buf]); return; }
  if (d && d.probe) {
    let simd = null;
    try { simd = WebAssembly.validate(new Uint8Array(d.simd)); } catch (err) { simd = 'throw ' + err; }
    self.postMessage({ probe: true, OffscreenCanvas: typeof OffscreenCanvas, VideoDecoder: typeof VideoDecoder,
      VideoFrame: typeof VideoFrame, crossOriginIsolated: self.crossOriginIsolated, SharedArrayBuffer: typeof SharedArrayBuffer, wasmSimd: simd,
      offscreen2d: (() => { try { return !!new OffscreenCanvas(4, 4).getContext('2d'); } catch (err) { return 'throw ' + err; } })() });
    return;
  }
  self.postMessage({ pong: true, kind: 'classic', self: String(self.location) });
};
