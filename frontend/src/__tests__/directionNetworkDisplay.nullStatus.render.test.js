/**
 * NetworkDisplay.svelte : un statut absent (liaison qui répond null, événement sans charge) vaut
 * « affichage éteint ». Chaque cas part d'un affichage lancé : sans cette règle, la lecture de
 * .running sur null interrompt le rendu et le bouton « Arrêter » reste à l'écran.
 */
import { test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const io = vi.hoisted(() => ({
    push: /** @type {(s: any) => void} */ (() => {}),
    resolveLoad: /** @type {(s: any) => void} */ (() => {})
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    BrowserOpenURL: () => {},
    EventsOn: (/** @type {string} */ _name, /** @type {(s: any) => void} */ cb) => {
        io.push = cb;
        return () => {};
    },
    EventsOff: () => {}
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    GetTournamentDisplayStatus: () => new Promise((resolve) => (io.resolveLoad = resolve)),
    StartTournamentDisplay: () => Promise.resolve(null),
    StopTournamentDisplay: () => Promise.resolve(null)
}));

import NetworkDisplay from '../components/direction/NetworkDisplay.svelte';

const RUNNING = { running: true, tournamentId: 3, addresses: [], publicAddresses: [], address: '', url: '', localUrl: 'http://127.0.0.1:8770/app/?tournoi=3', qr: '', error: '', errorPort: 0 };

afterEach(() => cleanup());

/** Monte le panneau, l'affichage lancé par un événement. */
async function mountRunning() {
    const view = render(NetworkDisplay, { props: { tournamentId: 3 } });
    await tick();
    io.push(RUNNING);
    await tick();
    expect(view.getByTestId('direction-network-stop')).toBeTruthy();
    return view;
}

/** @param {ReturnType<typeof render>} view */
async function expectStopped(view) {
    await tick();
    await tick();
    expect(view.getByTestId('direction-network-start')).toBeTruthy();
    expect(view.queryByTestId('direction-network-stop')).toBeNull();
}

test('a null status at load reads as stopped', async () => {
    const view = await mountRunning();
    io.resolveLoad(null);
    await expectStopped(view);
});

test('a null status event reads as stopped', async () => {
    const view = await mountRunning();
    io.push(null);
    await expectStopped(view);
});

test('a null answer to stop reads as stopped', async () => {
    const view = await mountRunning();
    await fireEvent.click(view.getByTestId('direction-network-stop'));
    await expectStopped(view);
});
