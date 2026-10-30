import 'fake-indexeddb/auto';
import { IDBFactory } from 'fake-indexeddb';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { createDescriber, replay } from '../vision/descriptors.js';
import { clearJournal, createMemoryJournal, openJournal } from '../vision/journal.js';
import { createWorkerCore } from '../vision/workerCore.js';
import { CALIBRATION, scene } from './fixtures/vision/frames.js';

const ROI = CALIBRATION.rois[0];

/** Six seconds at 2 img/s: the left dice ROI changes at 1 s, the picture is cut away at 3 s and back at 4 s. */
function syntheticFrames() {
    return Array.from({ length: 12 }, (_, i) => {
        const t = i / 2;
        const changed = t >= 3 && t < 4 ? [CALIBRATION.witness, ROI] : t >= 1 ? [ROI] : [];
        return { mediaTime: t, data: scene(changed, t >= 3 && t < 4 ? 5 : 1) };
    });
}

describe('descriptor journal (IndexedDB)', () => {
    it('keeps records in analysis order, per session, across batches', async () => {
        const idb = new IDBFactory();
        const j = await openJournal({ session: 'draft-1/plein', indexedDB: idb, batchSize: 3 });
        const other = await openJournal({ session: 'draft-2/plein', indexedDB: idb });
        for (let i = 0; i < 7; i++) j.append({ t: i, u: 0, w: null, r: {} });
        other.append({ t: 99, u: 0, w: null, r: {} });
        await other.flush();
        expect((await j.read()).map((r) => r.t)).toEqual([0, 1, 2, 3, 4, 5, 6]);
        await j.close();
        const reopened = await openJournal({ session: 'draft-1/plein', indexedDB: idb });
        expect((await reopened.read()).length).toBe(7);
        expect((await other.read()).map((r) => r.t)).toEqual([99]);
    });

    it('clears one session, or every session', async () => {
        const idb = new IDBFactory();
        const a = await openJournal({ session: 'a', indexedDB: idb });
        const b = await openJournal({ session: 'b', indexedDB: idb });
        a.append({ t: 1, u: 0, w: null, r: {} });
        b.append({ t: 2, u: 0, w: null, r: {} });
        await Promise.all([a.flush(), b.flush()]);
        await clearJournal({ session: 'a', indexedDB: idb });
        expect(await a.read()).toEqual([]);
        expect((await b.read()).length).toBe(1);
        // Every session, while connections are still open: must not block.
        await clearJournal({ indexedDB: idb });
        expect(await b.read()).toEqual([]);
    });

    it('drops the pending batch on clear, so a later flush writes nothing back', async () => {
        const idb = new IDBFactory();
        const j = await openJournal({ session: 's', indexedDB: idb, batchSize: 100 });
        j.append({ t: 1, u: 0, w: null, r: {} });
        await j.clear();
        await j.flush();
        expect(await j.read()).toEqual([]);
    });

    it('falls back to memory without IndexedDB', async () => {
        const j = await openJournal({ session: 's', indexedDB: /** @type {any} */ (null) });
        j.append({ t: 1, u: 0, w: null, r: {} });
        expect(await j.read()).toEqual([{ t: 1, u: 0, w: null, r: {} }]);
        expect(createMemoryJournal('x').session).toBe('x');
    });
});

describe('journal replay', () => {
    it('replays a stored journal into the same events as the live analysis, without video', async () => {
        const idb = new IDBFactory();
        /** @type {any[]} */
        const posted = [];
        const core = createWorkerCore({
            post: (m) => posted.push(m),
            openJournal: (o) => openJournal({ ...o, indexedDB: idb }),
            clearJournal: (o) => clearJournal({ ...o, indexedDB: idb })
        });
        await core.handle({ kind: 'calibrate', calibration: CALIBRATION });
        await core.handle({ kind: 'start', session: 'draft-1/plein' });
        for (const f of syntheticFrames()) await core.handle({ kind: 'frame', mediaTime: f.mediaTime, width: 320, height: 180, buffer: f.data.buffer });
        await core.handle({ kind: 'stop' });

        expect(posted[0]).toEqual({ kind: 'rate', fps: 2 });
        expect(posted.filter((m) => m.kind === 'frameDone').length).toBe(12);
        const live = posted.filter((m) => m.kind === 'event').map((m) => m.event);
        expect(live.map((e) => [e.type, e.mediaTime, e.state])).toEqual([
            ['frame.uncertain', 3, 'uncertain'],
            ['frame.uncertain', 4, 'uncertain']
        ]);

        const journal = await openJournal({ session: 'draft-1/plein', indexedDB: idb });
        const records = await journal.read();
        expect(records.length).toBe(12);
        expect(replay(records)).toEqual(live);

        // The records are the describer's, byte for byte: tuning can start from the journal.
        const d = createDescriber(CALIBRATION);
        expect(records).toEqual(syntheticFrames().map((f) => d.describe(f)));
    });

    it('clears the journal through the worker and acknowledges it', async () => {
        const idb = new IDBFactory();
        /** @type {any[]} */
        const posted = [];
        const core = createWorkerCore({
            post: (m) => posted.push(m),
            openJournal: (o) => openJournal({ ...o, indexedDB: idb }),
            clearJournal: (o) => clearJournal({ ...o, indexedDB: idb })
        });
        await core.handle({ kind: 'calibrate', calibration: CALIBRATION });
        await core.handle({ kind: 'start', session: 's' });
        const [f] = syntheticFrames();
        await core.handle({ kind: 'frame', mediaTime: 0, width: 320, height: 180, buffer: f.data.buffer });
        await core.handle({ kind: 'clearJournal', session: 's', id: 7 });
        await core.handle({ kind: 'stop' });
        expect(posted.at(-1)).toEqual({ kind: 'journalCleared', id: 7 });
        expect(await (await openJournal({ session: 's', indexedDB: idb })).read()).toEqual([]);
    });

    it('ignores frames before calibration or start', async () => {
        /** @type {any[]} */
        const posted = [];
        const core = createWorkerCore({ post: (m) => posted.push(m), openJournal: async (o) => createMemoryJournal(o.session) });
        const [f] = syntheticFrames();
        await core.handle({ kind: 'frame', mediaTime: 0, width: 320, height: 180, buffer: f.data.buffer });
        await core.handle({ kind: 'start', session: 's' });
        await core.handle({ kind: 'frame', mediaTime: 0, width: 320, height: 180, buffer: f.data.buffer });
        expect(posted.map((m) => m.kind)).toEqual(['frameDone', 'rate', 'frameDone']);
    });

    it('replays the journal fixtures to their expected events', () => {
        const dir = join(__dirname, 'fixtures', 'vision');
        const fixture = JSON.parse(readFileSync(join(dir, 'witness-cut.journal.json'), 'utf8'));
        const events = replay(fixture.records).map((e) => ({ type: e.type, mediaTime: e.mediaTime, state: e.state }));
        expect(events).toEqual(fixture.expected);
    });
});
