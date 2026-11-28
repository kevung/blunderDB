/**
 * longOpService.test.js — une tâche longue lancée d'une boîte : l'issue qui
 * revient est celle de sa tâche, même arrivée avant l'id, jamais celle d'une
 * autre tâche du même genre.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

/** @type {Map<string, Set<(p: any) => void>>} */
const bus = new Map();
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: (/** @type {string} */ name, /** @type {(p: any) => void} */ cb) => {
        if (!bus.has(name)) bus.set(name, new Set());
        bus.get(name)?.add(cb);
        return () => bus.get(name)?.delete(cb);
    }
}));
const emit = (/** @type {string} */ name, /** @type {any} */ p) => [...(bus.get(name) ?? [])].forEach((cb) => cb(p));

const { runLongOp } = await import('../services/longOpService.js');

describe('runLongOp', () => {
    beforeEach(() => bus.clear());

    it('reports progress and resolves on its own task, ignoring another', async () => {
        const seen = /** @type {number[][]} */ ([]);
        const out = runLongOp('x', async () => 7, { onProgress: (d, t) => seen.push([d, t]) });
        await Promise.resolve();
        await Promise.resolve();
        emit('x:progress', { id: 7, done: 1, total: 3 });
        emit('x:done', { id: 8, result: 'other' });
        emit('x:done', { id: 7, result: 'mine' });
        expect(await out).toEqual({ status: 'done', result: 'mine' });
        expect(seen).toEqual([[1, 3]]);
        expect(bus.get('x:done')?.size).toBe(0);
    });

    it('keeps an outcome that came before the id', async () => {
        const out = runLongOp('x', async () => {
            emit('x:cancelled', { id: 4 });
            return 4;
        });
        expect(await out).toEqual({ status: 'cancelled' });
    });

    it('turns a refused start into an error', async () => {
        const out = runLongOp('x', async () => {
            throw new Error('no library');
        });
        expect(await out).toMatchObject({ status: 'error', message: expect.stringContaining('no library') });
    });
});
