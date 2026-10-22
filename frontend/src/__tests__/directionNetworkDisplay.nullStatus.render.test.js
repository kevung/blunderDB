/**
 * NetworkDisplay.svelte : un statut absent (liaison qui répond null, événement sans charge) vaut
 * « affichage éteint » ; le panneau lit .running et ne doit pas planter.
 */
import { test, expect, afterEach, vi } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

const events = vi.hoisted(() => ({ push: /** @type {((s: any) => void) | null} */ (null) }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    BrowserOpenURL: () => {},
    EventsOn: (/** @type {string} */ _name, /** @type {(s: any) => void} */ cb) => {
        events.push = cb;
        return () => {};
    },
    EventsOff: () => {}
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    GetTournamentDisplayStatus: () => Promise.resolve(null),
    StartTournamentDisplay: () => Promise.resolve(null),
    StopTournamentDisplay: () => Promise.resolve(null)
}));

import NetworkDisplay from '../components/direction/NetworkDisplay.svelte';

afterEach(() => cleanup());

test('a null status renders as stopped', async () => {
    const errors = /** @type {unknown[]} */ ([]);
    const onError = (/** @type {any} */ e) => errors.push(e?.reason ?? e?.error ?? e);
    window.addEventListener('error', onError);
    window.addEventListener('unhandledrejection', onError);
    try {
        const { getByTestId, queryByTestId } = render(NetworkDisplay, { props: { tournamentId: 3 } });
        await tick();
        await tick();
        events.push?.(null);
        await tick();
        expect(errors).toEqual([]);
        expect(getByTestId('direction-network-start')).toBeTruthy();
        expect(queryByTestId('direction-network-stop')).toBeNull();
    } finally {
        window.removeEventListener('error', onError);
        window.removeEventListener('unhandledrejection', onError);
    }
});
