// V0.1 webview bench: Worker loading, rVFC, per-frame cost, seeks, feature detection.
// The test video carries its frame index as 12 binary blocks in its top 64 px (1080p).
const q = new URLSearchParams(location.search);
const VIDEO = q.get('v');
const RESULT = q.get('r');
const FPS = 30;
const report = { ua: navigator.userAgent, href: location.href, origin: location.origin, start: new Date().toISOString() };
const logEl = document.getElementById('log');
const log = (m) => { logEl.textContent += m + '\n'; };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const withTimeout = (p, ms) => Promise.race([p, sleep(ms).then(() => 'timeout ' + ms + ' ms')]);
const r1 = (x) => Math.round(x * 1000) / 1000;
function stats(a) {
  if (!a.length) return { n: 0 };
  const s = [...a].sort((x, y) => x - y);
  const p = (k) => r1(s[Math.min(s.length - 1, Math.floor(k * s.length))]);
  return { n: s.length, p50: p(0.5), p95: p(0.95), max: r1(s[s.length - 1]), mean: r1(s.reduce((x, y) => x + y, 0) / s.length) };
}
async function send(final) {
  try {
    await fetch(RESULT + (final ? '?final=1' : ''), { method: 'POST', body: JSON.stringify(report, null, 1) });
  } catch (e) { log('send failed: ' + e); }
}
async function section(name, fn) {
  log('== ' + name);
  try { report[name] = await fn(); } catch (e) { report[name] = { error: String(e && (e.name + ': ' + e.message + ' @ ' + e.stack) || e) }; }
  log(JSON.stringify(report[name]).slice(0, 400));
  await send(false);
}

// ---------- 1. Workers
function tryWorker(label, make) {
  return new Promise((resolve) => {
    let w;
    const t0 = performance.now();
    const done = (r) => {
      clearTimeout(to);
      try { w && w.terminate(); } catch { /* gone */ }
      resolve({ label, ms: r1(performance.now() - t0), ...r });
    };
    const to = setTimeout(() => done({ ok: false, err: 'timeout 5s' }), 5000);
    try { w = make(); } catch (e) { done({ ok: false, err: 'ctor: ' + e }); return; }
    w.onerror = (e) => { if (e.preventDefault) e.preventDefault(); done({ ok: false, err: 'error event: ' + (e.message || e.type) + ' ' + (e.filename || '') }); };
    w.onmessageerror = () => done({ ok: false, err: 'messageerror' });
    w.onmessage = (e) => done({ ok: !!(e.data && e.data.pong), data: e.data });
    w.postMessage({ ping: true });
  });
}
async function blobOf(path, type) {
  const txt = await (await fetch(path)).text();
  return URL.createObjectURL(new Blob([txt], { type: type || 'text/javascript' }));
}
async function workers() {
  const out = [];
  const head = await fetch('/bench/w-classic.js');
  out.push({ label: 'fetch /bench/w-classic.js', ok: head.ok, status: head.status, type: head.headers.get('content-type') });
  out.push(await tryWorker('classic absolute', () => new Worker('/bench/w-classic.js')));
  out.push(await tryWorker('classic relative', () => new Worker('w-classic.js')));
  out.push(await tryWorker('classic importScripts', () => new Worker('/bench/w-importscripts.js')));
  out.push(await tryWorker('module with static import', () => new Worker('/bench/w-module.js', { type: 'module' })));
  out.push(await tryWorker('module flat', () => new Worker('/bench/w-module-flat.js', { type: 'module' })));
  out.push(await tryWorker('module via new URL(import.meta.url)', () => new Worker(new URL('./w-module.js', import.meta.url), { type: 'module' })));
  const bc = await blobOf('/bench/w-classic.js');
  out.push(await tryWorker('blob classic', () => new Worker(bc)));
  const bm = await blobOf('/bench/w-module-flat.js');
  out.push(await tryWorker('blob module flat', () => new Worker(bm, { type: 'module' })));
  const absImport = `import { dep } from '${location.origin}/bench/w-dep.js';\nself.onmessage = () => self.postMessage({ pong: true, kind: 'blob-module-abs-import', dep });`;
  const ba = URL.createObjectURL(new Blob([absImport], { type: 'text/javascript' }));
  out.push(await tryWorker('blob module importing wails:// absolute', () => new Worker(ba, { type: 'module' })));
  return out;
}

let probeWorker = null;
function workerCall(msg, transfer) {
  return new Promise((resolve) => {
    probeWorker.onmessage = (e) => resolve(e.data);
    probeWorker.postMessage(msg, transfer || []);
  });
}
async function transfer() {
  const times = [];
  let detachedOk = true;
  for (let i = 0; i < 300; i++) {
    const buf = new ArrayBuffer(320 * 180 * 4);
    const t0 = performance.now();
    const p = workerCall({ buf }, [buf]);
    if (buf.byteLength !== 0) detachedOk = false;
    const back = await p;
    times.push(performance.now() - t0);
    if (back.n !== 320 * 180 * 4) detachedOk = false;
  }
  const b0 = performance.now();
  for (let i = 0; i < 1000; i++) { const buf = new ArrayBuffer(320 * 180 * 4); await workerCall({ buf }, [buf]); }
  return { roundTripMs: stats(times.slice(20)), batchMeanRoundTripMs: r1((performance.now() - b0) / 1000), sourceDetachedAndSizeKept: detachedOk };
}

// ---------- video helpers
const canvas = document.getElementById('c');
let ctx = canvas.getContext('2d');
function decodeIndex(img) {
  let n = 0;
  for (let b = 0; b < 12; b++) {
    const x = Math.round((160 * b + 80) * 320 / 1920);
    if (img.data[(5 * 320 + x) * 4] > 128) n |= 1 << b;
  }
  return n;
}
function grab() {
  const t0 = performance.now();
  ctx.drawImage(video, 0, 0, 320, 180);
  const t1 = performance.now();
  const img = ctx.getImageData(0, 0, 320, 180);
  const t2 = performance.now();
  return { img, draw: t1 - t0, read: t2 - t1 };
}
const once = (el, ev, ms) => new Promise((resolve) => {
  const t = setTimeout(() => { el.removeEventListener(ev, h); resolve(null); }, ms);
  const h = () => { clearTimeout(t); resolve(performance.now()); };
  el.addEventListener(ev, h, { once: true });
});
const nextFrame = (ms) => new Promise((resolve) => {
  if (!video.requestVideoFrameCallback) { resolve(null); return; }
  const t = setTimeout(() => resolve(null), ms);
  video.requestVideoFrameCallback((now, md) => { clearTimeout(t); resolve({ at: performance.now(), md }); });
});
let video;
let rvfcSeen = 0;
async function setupVideo() {
  video = document.createElement('video');
  video.crossOrigin = 'anonymous';
  video.muted = true;
  video.preload = 'auto';
  video.playsInline = true;
  video.width = 480;
  document.body.insertBefore(video, canvas);
  const md = once(video, 'loadeddata', 15000);
  video.src = VIDEO;
  const t = await md;
  return { loadeddata: !!t, duration: video.duration, w: video.videoWidth, h: video.videoHeight, error: video.error && video.error.code };
}
async function seekTo(t) {
  const s = once(video, 'seeked', 5000);
  video.currentTime = t;
  return s;
}

// ---------- 2. rVFC at 1x, with the full per-frame pipeline (3)
async function playAt(rate, seconds, from, pipeline) {
  await seekTo(from);
  video.playbackRate = rate;
  const q0 = video.getVideoPlaybackQuality ? video.getVideoPlaybackQuality() : null;
  const rows = [];
  let stop = false;
  const cb = (now, md) => {
    if (stop) return;
    rvfcSeen++;
    const ct = video.currentTime;
    const row = { at: performance.now(), mediaTime: md.mediaTime, ct, pf: md.presentedFrames };
    if (pipeline) {
      const t0 = performance.now();
      const g = grab();
      row.idx = decodeIndex(g.img);
      const buf = g.img.data.buffer;
      probeWorker.postMessage({ buf }, [buf]);
      row.draw = g.draw; row.read = g.read; row.total = performance.now() - t0;
    }
    rows.push(row);
    video.requestVideoFrameCallback(cb);
  };
  probeWorker.onmessage = () => {};
  video.requestVideoFrameCallback(cb);
  const wall0 = performance.now();
  const mt0 = video.currentTime;
  const raf = [];
  const rafLoop = () => {
    if (stop) return;
    if (!pipeline || rows.length === 0) {
      const ct = video.currentTime;
      const t0 = performance.now();
      const g = grab();
      const idx = decodeIndex(g.img);
      if (pipeline) { const buf = g.img.data.buffer; probeWorker.postMessage({ buf }, [buf]); }
      raf.push({ at: t0, ct, idx, total: performance.now() - t0 });
    }
    requestAnimationFrame(rafLoop);
  };
  const ended = once(video, 'ended', seconds * 1000 + 60000);
  video.addEventListener('playing', () => video.requestVideoFrameCallback(cb), { once: true });
  requestAnimationFrame(rafLoop);
  const tp = performance.now();
  let playResolvedMs = null;
  video.play().then(() => { playResolvedMs = r1(performance.now() - tp); }, (e) => { rows.err = String(e); });
  const t00 = performance.now();
  const byRaf = new Promise((res) => { const tick = () => { if (performance.now() - t00 >= seconds * 1000) res('raf'); else requestAnimationFrame(tick); }; requestAnimationFrame(tick); });
  const stoppedBy = await Promise.race([sleep(seconds * 1000).then(() => 'timer'), byRaf, ended.then((t) => (t === null ? 'ended-timeout' : 'ended'))]);
  const tEnd = stoppedBy === 'ended' ? 1 : null;
  stop = true;
  video.pause();
  const wall = (performance.now() - wall0) / 1000;
  const q1 = video.getVideoPlaybackQuality ? video.getVideoPlaybackQuality() : null;
  const gaps = [];
  let missed = 0, nonMono = 0, idxEq = 0, idxOff1 = 0, idxBad = 0, ctEq = 0, ctAhead = [];
  for (let i = 0; i < rows.length; i++) {
    const r = rows[i];
    if (i) {
      gaps.push(r.at - rows[i - 1].at);
      missed += Math.max(0, r.pf - rows[i - 1].pf - 1);
      if (r.mediaTime <= rows[i - 1].mediaTime) nonMono++;
    }
    const fromMt = Math.round(r.mediaTime * FPS);
    if (r.idx !== undefined) {
      if (r.idx === fromMt) idxEq++; else if (Math.abs(r.idx - fromMt) === 1) idxOff1++; else idxBad++;
    }
    if (Math.floor(r.ct * FPS + 1e-6) === fromMt) ctEq++;
    ctAhead.push((r.ct - r.mediaTime) * 1000);
  }
  const mtOffGrid = rows.filter((r) => Math.abs(r.mediaTime * FPS - Math.round(r.mediaTime * FPS)) > 0.01).length;
  return {
    rate, wallS: r1(wall), callbacks: rows.length, callbacksPerS: r1(rows.length / wall),
    mediaAdvanceS: r1(video.currentTime - mt0), effectiveRate: r1((video.currentTime - mt0) / wall),
    presentedFramesMissedBetweenCallbacks: missed, mediaTimeNonMonotonic: nonMono, mediaTimeOffFrameGrid: mtOffGrid,
    decodedIndexVsMediaTime: pipeline ? { equal: idxEq, off1: idxOff1, other: idxBad } : undefined,
    currentTimeFrameEqualsMediaTimeFrame: ctEq, currentTimeMinusMediaTimeMs: stats(ctAhead),
    callbackGapMs: stats(gaps),
    pipelineMs: pipeline ? { draw: stats(rows.map((r) => r.draw)), getImageData: stats(rows.map((r) => r.read)), total: stats(rows.map((r) => r.total)) } : undefined,
    droppedVideoFrames: q0 && q1 ? q1.droppedVideoFrames - q0.droppedVideoFrames : 'n/a',
    totalVideoFrames: q0 && q1 ? q1.totalVideoFrames - q0.totalVideoFrames : 'n/a',
    endedEarly: tEnd !== null, stoppedBy, playResolvedMs, canvasReadableAfter: (() => { try { grab(); return true; } catch (e) { return e.name; } })(),
    rafSampler: (() => {
      const uniq = new Set(raf.map((r) => r.idx));
      let ctEq = 0, ctOff1 = 0, ctOther = 0, back = 0;
      raf.forEach((r, i) => {
        const e = Math.floor(r.ct * FPS + 1e-6);
        if (e === r.idx) ctEq++; else if (Math.abs(e - r.idx) === 1) ctOff1++; else ctOther++;
        if (i && r.idx < raf[i - 1].idx) back++;
      });
      const span = raf.length ? raf[raf.length - 1].idx - raf[0].idx + 1 : 0;
      return { samples: raf.length, samplesPerS: r1(raf.length / wall), distinctFrames: uniq.size, framesInSpan: span,
        distinctFramesPerS: r1(uniq.size / wall), decodedVsCurrentTime: { equal: ctEq, off1: ctOff1, other: ctOther }, indexWentBack: back,
        pipelineTotalMs: stats(raf.map((r) => r.total)) };
    })(),
    err: rows.err
  };
}

// ---------- 3. paused per-frame cost
async function cost() {
  await seekTo(10 + 1 / 120);
  await sleep(300);
  const out = {};
  for (const wrf of [false, true]) {
    const c = document.createElement('canvas');
    c.width = 320; c.height = 180;
    ctx = c.getContext('2d', { willReadFrequently: wrf });
    const d = [], r = [], t = [];
    for (let i = 0; i < 2000; i++) {
      const t0 = performance.now();
      const g = grab();
      d.push(g.draw); r.push(g.read); t.push(performance.now() - t0);
    }
    const b0 = performance.now();
    for (let i = 0; i < 2000; i++) { ctx.drawImage(video, 0, 0, 320, 180); ctx.getImageData(0, 0, 320, 180); }
    const batchMean = (performance.now() - b0) / 2000;
    const b1 = performance.now();
    for (let i = 0; i < 2000; i++) ctx.drawImage(video, 0, 0, 320, 180);
    const drawOnlyMean = (performance.now() - b1) / 2000;
    const g = grab();
    out[wrf ? 'willReadFrequently' : 'default'] = { draw: stats(d.slice(50)), getImageData: stats(r.slice(50)), total: stats(t.slice(50)), decodedIndex: decodeIndex(g.img), expected: 300, batchMeanMs: r1(batchMean), drawOnlyBatchMeanMs: r1(drawOnlyMean) };
  }
  ctx = canvas.getContext('2d');
  return out;
}

// ---------- 4. seeks
async function seekBench(label, frames) {
  video.pause();
  const toSeeked = [], toFrame = [];
  let rvfcMissing = 0, ok = 0, off = [], seekedMissing = 0, mtOk = 0;
  for (const n of frames) {
    const t = n / FPS + 1 / (4 * FPS);
    const f = nextFrame(rvfcSeen ? 1500 : 150);
    const s = once(video, 'seeked', 5000);
    const t0 = performance.now();
    video.currentTime = t;
    const ts = await s;
    const fr = await f;
    if (ts === null) seekedMissing++; else toSeeked.push(ts - t0);
    if (fr === null) rvfcMissing++;
    else { toFrame.push(fr.at - t0); if (Math.round(fr.md.mediaTime * FPS) === n) mtOk++; }
    let idx;
    try { idx = decodeIndex(grab().img); } catch (e) { idx = e.name; }
    if (idx === n) ok++; else off.push(typeof idx === 'number' ? idx - n : idx);
  }
  return {
    label, seeks: frames.length, seekedMs: stats(toSeeked), rvfcAfterSeekMs: stats(toFrame), rvfcMissing, seekedMissing,
    decodedFrameExact: ok, decodedFrameErrors: off.slice(0, 20), rvfcMediaTimeExact: mtOk
  };
}
function shuffled(a) {
  let s = 12345;
  const r = () => (s = (s * 1103515245 + 12345) & 0x7fffffff) / 0x7fffffff;
  return a.map((x) => [r(), x]).sort((x, y) => x[0] - y[0]).map((x) => x[1]);
}

// ---------- 5. detection
async function detect() {
  const simd = [0, 97, 115, 109, 1, 0, 0, 0, 1, 5, 1, 96, 0, 1, 123, 3, 2, 1, 0, 10, 10, 1, 8, 0, 65, 0, 253, 15, 253, 98, 11];
  await send(false);
  const out = {
    VideoFrame: typeof VideoFrame, VideoDecoder: typeof VideoDecoder, OffscreenCanvas: typeof OffscreenCanvas,
    crossOriginIsolated: self.crossOriginIsolated, SharedArrayBuffer: typeof SharedArrayBuffer,
    wasmBaseline: WebAssembly.validate(new Uint8Array([0, 97, 115, 109, 1, 0, 0, 0])),
    wasmSimd: WebAssembly.validate(new Uint8Array(simd)),
    rVFC: typeof HTMLVideoElement.prototype.requestVideoFrameCallback, fastSeek: typeof HTMLVideoElement.prototype.fastSeek,
    getVideoPlaybackQuality: typeof HTMLVideoElement.prototype.getVideoPlaybackQuality, createImageBitmap: typeof createImageBitmap
  };
  report.p5_partial = out;
  const step = async (n) => { report.p5_step = n; await send(false); };
  await step('worker probe');
  out.worker = await withTimeout(workerCall({ probe: true, simd }), 3000);
  return out;
}

async function webcodecs() {
  const out = {};
  report.p5_webcodecs_worker = await withTimeout(new Promise((resolve) => {
    probeWorker.onmessage = (e) => { if (e.data.codecs) resolve(e.data.codecs); else { report.p5_webcodecs_worker_partial = e.data.partial; send(false); } };
    probeWorker.postMessage({ codecs: ['vp09.00.10.08', 'av01.0.04M.08', 'avc1.42E01E', 'avc1.640028'] });
  }), 20000);
  await send(false);
  report.p5_webcodecs = out;
  const step = async (n) => { report.p5_step = n; await send(false); };
  if (typeof VideoDecoder !== 'undefined') {
    for (const codec of ['vp09.00.10.08', 'av01.0.04M.08', 'avc1.42E01E', 'avc1.640028']) {
      await step('isConfigSupported ' + codec);
      try { out['isConfigSupported ' + codec] = await withTimeout(VideoDecoder.isConfigSupported({ codec, codedWidth: 1920, codedHeight: 1080 }).then((r) => r.supported), 3000); }
      catch (e) { out['isConfigSupported ' + codec] = 'throw ' + e; }
    }
  }
  if (typeof VideoFrame !== 'undefined' && video) {
    await step('new VideoFrame(video)');
    try { const vf = new VideoFrame(video, { timestamp: 0 }); out.videoFrameFromVideo = vf.codedWidth + 'x' + vf.codedHeight; vf.close(); }
    catch (e) { out.videoFrameFromVideo = 'throw ' + e; }
  }
  return out;
}

async function tainted() {
  const v2 = document.createElement('video');
  v2.muted = true;
  const l = once(v2, 'loadeddata', 10000);
  v2.src = VIDEO + '?nocors';
  await l;
  const c = document.createElement('canvas').getContext('2d');
  c.drawImage(v2, 0, 0, 32, 18);
  try { c.getImageData(0, 0, 1, 1); return { withoutCrossOriginAttr: 'readable (not tainted)' }; }
  catch (e) { return { withoutCrossOriginAttr: 'throws ' + e.name }; }
}

async function main() {
  log('video=' + VIDEO);
  await section('p1_workers', workers);
  const firstOk = report.p1_workers.find((r) => r.ok && r.data && r.data.kind === 'classic');
  probeWorker = firstOk ? new Worker('/bench/w-classic.js') : new Worker(await blobOf('/bench/w-classic.js'));
  await section('p1_transfer', transfer);
  await section('video', setupVideo);
  await section('p5_detect', () => withTimeout(detect(), 15000));
  await section('p2_tainted', tainted);
  await section('p3_cost_paused', cost);
  await section('p2p3_play_1x', () => playAt(1, 12, 5, true));
  await section('p2_play_1x_no_pipeline', () => playAt(1, 6, 30, false));
  await section('p4_rate_2x', () => playAt(2, 6, 40, true));
  await section('p4_rate_4x', () => playAt(4, 6, 60, true));
  await section('p4_seek_keyframe', () => seekBench('keyframe', shuffled(Array.from({ length: 50 }, (_, k) => 60 * (k + 4) % 3540))));
  await section('p4_seek_midgop', () => seekBench('mid-GOP (+31 frames)', shuffled(Array.from({ length: 50 }, (_, k) => (60 * (k + 4) + 31) % 3540))));
  await section('p4_seek_endgop', () => seekBench('end of GOP (+59 frames)', shuffled(Array.from({ length: 50 }, (_, k) => (60 * (k + 4) + 59) % 3540))));
  await section('p4_seek_sequential_0_5s', () => seekBench('sequential 0.5 s', Array.from({ length: 100 }, (_, j) => 15 * (j + 1))));
  report.clockResolutionMs = (() => { const a = performance.now(); let b; do { b = performance.now(); } while (b === a); return r1(b - a); })();
  report.end = new Date().toISOString();
  await send(false);
  await section('p5_webcodecs', webcodecs);
  log('done');
  await send(true);
}
main();
