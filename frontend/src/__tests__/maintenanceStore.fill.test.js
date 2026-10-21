import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const fill = vi.fn();
vi.mock('../../wailsjs/go/database/Database.js', () => ({ FillMatchStatsInBackground: () => fill() }));

const { maintenanceStore, maintenanceEndedStore, startMatchStatsFill, dropMatchStatsFill, STATS_MATCHES } = await import('../stores/maintenanceStore.js');

describe('startMatchStatsFill', () => {
    beforeEach(() => {
        fill.mockReset();
        maintenanceStore.set(null);
    });

    it('shows the stats.matches line running once a fill starts', async () => {
        fill.mockResolvedValue(true);
        await startMatchStatsFill();
        expect(get(maintenanceStore)).toEqual({ kind: STATS_MATCHES, done: 0, total: 0 });
    });

    it('keeps a progress the fill already reported', async () => {
        fill.mockImplementation(async () => {
            maintenanceStore.set({ kind: STATS_MATCHES, done: 3, total: 9 });
            return true;
        });
        await startMatchStatsFill();
        expect(get(maintenanceStore)).toEqual({ kind: STATS_MATCHES, done: 3, total: 9 });
    });

    it('drops a line left by a fill the file change stopped', async () => {
        maintenanceStore.set({ kind: STATS_MATCHES, done: 4, total: 9 });
        fill.mockResolvedValue(false);
        await startMatchStatsFill();
        expect(get(maintenanceStore)).toBeNull();
    });

    it('leaves another line running alone, and survives a refusal', async () => {
        maintenanceStore.set({ kind: 'vacuum', done: 0, total: 0 });
        fill.mockRejectedValue(new Error('no database open'));
        await startMatchStatsFill();
        expect(get(maintenanceStore)).toEqual({ kind: 'vacuum', done: 0, total: 0 });
    });

    it('shows nothing once the fill ended before the call resolved', async () => {
        fill.mockImplementation(async () => {
            maintenanceStore.set({ kind: STATS_MATCHES, done: 9, total: 9 });
            maintenanceStore.set(null);
            maintenanceEndedStore.update((n) => n + 1);
            return true;
        });
        await startMatchStatsFill();
        expect(get(maintenanceStore)).toBeNull();
    });

    it('drops the line of a closed library, never another one', () => {
        maintenanceStore.set({ kind: STATS_MATCHES, done: 1, total: 9 });
        dropMatchStatsFill();
        expect(get(maintenanceStore)).toBeNull();
        maintenanceStore.set({ kind: 'vacuum', done: 0, total: 0 });
        dropMatchStatsFill();
        expect(get(maintenanceStore)).toEqual({ kind: 'vacuum', done: 0, total: 0 });
    });
});
