/**
 * NetworkDisplay.svelte : l'affichage réseau d'un tournoi se monte éteint, avertit qu'il est
 * lisible par tout le réseau local, et montre le QR code et l'adresse une fois lancé.
 */
import { describe, test, expect, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const calls = vi.hoisted(() => ({ started: /** @type {any[]} */ ([]), stopped: 0 }));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({ BrowserOpenURL: () => {}, EventsOn: () => () => {}, EventsOff: () => {} }));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    GetTournamentDisplayStatus: () => Promise.resolve({ running: false, tournamentId: 0, url: '', localUrl: '', qr: '', error: '' }),
    StartTournamentDisplay: (/** @type {any} */ c) => {
        calls.started.push(c);
        if (c.tournamentId === 8) {
            return Promise.resolve({ running: false, tournamentId: 0, addresses: [], publicAddresses: [], address: '', url: '', localUrl: '', qr: '', error: 'port_in_use', errorPort: c.port });
        }
        if (c.tournamentId === 9) {
            return Promise.resolve({
                running: true,
                tournamentId: 9,
                addresses: ['125.40.2.124'],
                publicAddresses: ['125.40.2.124'],
                address: '125.40.2.124',
                url: 'http://125.40.2.124:8770/app/?tournoi=9',
                localUrl: 'http://127.0.0.1:8770/app/?tournoi=9',
                qr: 'data:image/png;base64,AAAA',
                error: '',
                errorPort: 0
            });
        }
        if (c.tournamentId === 7) {
            return Promise.resolve({ running: true, tournamentId: 7, addresses: [], address: '', url: '', localUrl: 'http://127.0.0.1:8770/app/?tournoi=7', qr: '', error: '' });
        }
        return Promise.resolve({
            running: true,
            tournamentId: c.tournamentId,
            addresses: ['192.168.1.10'],
            address: '192.168.1.10',
            url: `http://192.168.1.10:${c.port}/app/?tournoi=${c.tournamentId}`,
            localUrl: `http://127.0.0.1:${c.port}/app/?tournoi=${c.tournamentId}`,
            qr: 'data:image/png;base64,AAAA',
            error: ''
        });
    },
    StopTournamentDisplay: () => {
        calls.stopped++;
        return Promise.resolve({ running: false, tournamentId: 0, url: '', localUrl: '', qr: '', error: '' });
    }
}));

import NetworkDisplay from '../components/direction/NetworkDisplay.svelte';

afterEach(() => cleanup());

describe('NetworkDisplay', () => {
    test('off until asked, warns, then shows the QR code and the address', async () => {
        const { getByTestId, queryByTestId, container } = render(NetworkDisplay, { props: { tournamentId: 42 } });
        await tick();
        expect(queryByTestId('direction-network-stop')).toBeNull();
        expect(container.querySelector('img.qr')).toBeNull();
        expect(container.querySelector('.warning')?.textContent).toBeTruthy();

        await fireEvent.click(getByTestId('direction-network-start'));
        await tick();
        await tick();
        expect(calls.started).toEqual([{ tournamentId: 42, port: 8770, address: '' }]);
        expect(getByTestId('direction-network-url').textContent).toBe('http://192.168.1.10:8770/app/?tournoi=42');
        expect(container.querySelector('img.qr')?.getAttribute('src')).toBe('data:image/png;base64,AAAA');

        await fireEvent.click(getByTestId('direction-network-stop'));
        await tick();
        await tick();
        expect(calls.stopped).toBe(1);
        expect(container.querySelector('img.qr')).toBeNull();
    });

    test('without a local network address, says so instead of drawing a QR code to 127.0.0.1', async () => {
        const { getByTestId, queryByTestId, container } = render(NetworkDisplay, { props: { tournamentId: 7 } });
        await tick();
        await fireEvent.click(getByTestId('direction-network-start'));
        await tick();
        await tick();
        expect(getByTestId('direction-network-nolan').textContent).toBeTruthy();
        expect(container.querySelector('img.qr')).toBeNull();
        expect(queryByTestId('direction-network-url')).toBeNull();
    });

    test('a busy port is worded for the director, without the system message', async () => {
        const { getByTestId, container } = render(NetworkDisplay, { props: { tournamentId: 8 } });
        await tick();
        await fireEvent.click(getByTestId('direction-network-start'));
        await tick();
        await tick();
        const text = container.querySelector('.error')?.textContent || '';
        expect(text).toContain('8770');
        expect(text).not.toMatch(/listen tcp|bind:/);
    });

    test('warns when the proposed address is not a private one', async () => {
        const { getByTestId } = render(NetworkDisplay, { props: { tournamentId: 9 } });
        await tick();
        await fireEvent.click(getByTestId('direction-network-start'));
        await tick();
        await tick();
        expect(getByTestId('direction-network-public').textContent).toContain('125.40.2.124');
    });
});
