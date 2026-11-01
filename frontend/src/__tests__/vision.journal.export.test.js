import 'fake-indexeddb/auto';
import { IDBFactory } from 'fake-indexeddb';
import { execFileSync } from 'node:child_process';
import { readFileSync, writeFileSync, mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { describe, expect, it, vi, afterEach } from 'vitest';

vi.mock('../../wailsjs/go/gui/App.js', () => ({ SaveBoardImageDialog: vi.fn(), SaveBoardSVG: vi.fn() }));

import { createDetector, replay } from '../vision/descriptors.js';
import { listJournalSessions, openJournal, readJournal } from '../vision/journal.js';
import { journalFile } from '../services/visionBenchExport.js';

const FIXTURE = join(__dirname, 'fixtures/vision/two-turns.journal.json');
const REPLAY = join(__dirname, '../../../tools/vision/replay.mjs');
/** @param {number} t @returns {import('../vision/descriptors.js').DescriptorRecord} */
const rec = (t) => ({ t, u: /** @type {0} */ (0), w: null, r: {} });

afterEach(() => localStorage.clear());

describe('journal reading', () => {
    it('lists the sessions and reads one in writing order, without erasing', async () => {
        const idb = new IDBFactory();
        const a = await openJournal({ session: '7:A', indexedDB: idb, batchSize: 2 });
        const b = await openJournal({ session: '7:B', indexedDB: idb });
        for (const t of [3, 1, 2]) a.append(rec(t));
        b.append(rec(9));
        await a.close();
        await b.close();
        expect(await listJournalSessions({ indexedDB: idb })).toEqual(['7:A', '7:B']);
        expect((await readJournal({ session: '7:A', indexedDB: idb })).map((r) => r.t)).toEqual([3, 1, 2]);
        expect((await readJournal({ session: '7:A', indexedDB: idb })).length).toBe(3);
        expect(await readJournal({ session: 'none', indexedDB: idb })).toEqual([]);
    });

    it('exports every session of the draft, and only those, with their calibrations', async () => {
        const idb = new IDBFactory();
        for (const [s, t] of /** @type {[string, number][]} */ ([
            ['7:Calibration', 1],
            ['7:Calibration 2', 2],
            ['70:Calibration', 3]
        ])) {
            const j = await openJournal({ session: s, indexedDB: idb });
            j.append(rec(t));
            await j.close();
        }
        const cal = {
            name: 'Calibration 2',
            clicks: [
                [0, 0],
                [1, 0],
                [1, 1],
                [0, 1]
            ],
            modelToImage: [1, 0, 0, 0, 1, 0, 0, 0, 1]
        };
        localStorage.setItem('blunderdb.transcription.videoCalibrations.7', JSON.stringify({ items: [cal], active: 'Calibration 2' }));
        const file = await journalFile(7, { indexedDB: idb });
        expect(file.draftId).toBe('7');
        expect(Object.keys(file.sessions)).toEqual(['7:Calibration', '7:Calibration 2']);
        expect(file.sessions['7:Calibration 2'].map((r) => r.t)).toEqual([2]);
        expect(Object.keys(file.calibrations)).toEqual(['7:Calibration 2']);
    });
});

describe('tools/vision/replay.mjs', () => {
    it('gives the events replay() gives, in the bench format', () => {
        const fixture = JSON.parse(readFileSync(FIXTURE, 'utf8'));
        const dir = mkdtempSync(join(tmpdir(), 'replay-'));
        const file = join(dir, 'journal.json');
        writeFileSync(file, JSON.stringify({ draftId: '1', sessions: { '1:C': fixture.records } }));
        const out = JSON.parse(execFileSync('node', [REPLAY, file], { encoding: 'utf8' }));
        const expected = replay(fixture.records, createDetector()).map((e) => ({ type: e.type, mediaTime: e.mediaTime, state: e.state }));
        expect(expected.length).toBeGreaterThan(0);
        expect(out).toEqual({ events: expected });
        const same = JSON.parse(execFileSync('node', [REPLAY, file, '--session', '1:C', '--overrides', '{}'], { encoding: 'utf8' }));
        expect(same).toEqual(out);
    });
});
