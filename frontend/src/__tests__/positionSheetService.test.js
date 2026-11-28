import { describe, it, expect, vi, beforeEach } from 'vitest';

const StartPositionSheet = vi.fn(async (/** @type {any[]} */ ..._a) => 1);
vi.mock('../../wailsjs/go/database/Database.js', () => ({ CountCollectionPositions: vi.fn(async () => 7) }));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ StartPositionSheet }));
// The task starts once the file is chosen: here, at once.
const printSheet = vi.fn(async (/** @type {string} */ _name, /** @type {any} */ _keys, /** @type {(p: string) => Promise<number>} */ start) => {
    await start('/tmp/x.pdf');
    return '/tmp/x.pdf';
});
vi.mock('../services/printService.js', () => ({ printSheet }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
const listed = Array.from({ length: 1200 }, (_, i) => i + 1);
vi.mock('../stores/positionStore.js', () => ({ listedIds: vi.fn(async () => listed), browsingLibrary: () => false }));

const { get } = await import('svelte/store');
const svc = await import('../services/positionSheetService.js');

describe('positionSheetService', () => {
    beforeEach(() => {
        StartPositionSheet.mockClear();
        printSheet.mockClear();
        svc.positionPrintRequest.set(null);
    });

    it('opens on the whole browsed list, not the loaded rows', async () => {
        await svc.openBrowsedListPrint();
        const req = get(svc.positionPrintRequest);
        expect(req?.count).toBe(1200);
        expect(req?.source.ids).toHaveLength(1200);
    });

    it('prints a collection through the engine, which resolves it', async () => {
        await svc.openCollectionPrint({ id: 3, name: 'Révisions' });
        const req = /** @type {any} */ (get(svc.positionPrintRequest));
        expect(req).toMatchObject({ source: { collection: 3 }, title: 'Révisions', count: 7 });
        await svc.printPositionSheet(req.source, { ...svc.DEFAULT_POSITION_SHEET_OPTIONS, title: req.title });
        expect(StartPositionSheet).toHaveBeenCalledWith({ collection: 3 }, expect.objectContaining({ analysis: true, verso: false, title: 'Révisions' }), '/tmp/x.pdf');
    });

    it('refuses beyond the cap rather than print a part', async () => {
        const ids = Array.from({ length: svc.MAX_SHEET_POSITIONS + 1 }, (_, i) => i + 1);
        expect(await svc.printPositionSheet({ ids }, svc.DEFAULT_POSITION_SHEET_OPTIONS)).toBeNull();
        expect(StartPositionSheet).not.toHaveBeenCalled();
    });
});
