/**
 * AssistantPanel.svelte : éteint, il renvoie aux paramètres ; allumé, une phrase part, le texte
 * du modèle est marqué comme texte libre, et une écriture préparée attend la confirmation.
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const state = vi.hoisted(() => ({ on: false, asked: /** @type {any[]} */ ([]), confirmed: /** @type {boolean[]} */ ([]) }));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetAssistant: () => Promise.resolve({ on: state.on, preset: 'ollama', baseURL: '', model: '', write: true, remoteAck: '' }),
    SaveAssistant: () => Promise.resolve(),
    // The MCP server's write switch is off: the assistant's own setting decides.
    GetMCPHost: () => Promise.resolve({ on: false, port: 0, write: false })
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    AssistantPresets: () => Promise.resolve([{ id: 'ollama', name: 'Ollama', baseURL: 'http://localhost:11434/v1', model: 'm', remote: false, needsKey: false }]),
    AssistantAsk: (req) => {
        state.asked.push(req);
        return Promise.resolve({
            entries: [
                { kind: 'user', text: req.text },
                { kind: 'tool', tool: 'search_positions', args: '{"query":"cube"}' },
                { kind: 'model', text: 'Voici.', free: true }
            ],
            pending: { tool: 'create_collection', title: 'Create a collection', args: '{"name":"Videau"}' }
        });
    },
    AssistantConfirm: (accept) => {
        state.confirmed.push(accept);
        return Promise.resolve({ entries: [{ kind: 'error', tool: 'create_collection', text: 'declined' }] });
    },
    AssistantCancel: () => {},
    AssistantReset: () => Promise.resolve()
}));

import AssistantPanel from '../components/AssistantPanel.svelte';
import { assistantEntriesStore, assistantPendingStore } from '../services/assistantService.js';

beforeEach(() => {
    state.asked.length = 0;
    state.confirmed.length = 0;
    assistantEntriesStore.set([]);
    assistantPendingStore.set(null);
});
afterEach(() => cleanup());

const settle = async () => {
    for (let i = 0; i < 5; i++) await tick();
};

describe('AssistantPanel', () => {
    test('off: points to the settings, offers no input', async () => {
        state.on = false;
        const { container, getByTestId } = render(AssistantPanel);
        await settle();
        expect(getByTestId('assistant-panel').querySelector('input')).toBeNull();
        expect(container.textContent).toBeTruthy();
    });

    test('on: a sentence is sent, free text is marked, a write waits for confirmation', async () => {
        state.on = true;
        const { container } = render(AssistantPanel);
        await settle();
        const input = /** @type {HTMLInputElement} */ (container.querySelector('.ask-input'));
        await fireEvent.input(input, { target: { value: 'mes erreurs de videau' } });
        await fireEvent.keyDown(input, { key: 'Enter' });
        await settle();
        expect(state.asked[0]).toMatchObject({ preset: 'ollama', write: true, text: 'mes erreurs de videau' });
        expect(container.querySelector('.entry.model .badge')).not.toBeNull();
        const pending = container.querySelector('.pending');
        expect(pending).not.toBeNull();
        const decline = /** @type {HTMLButtonElement} */ (pending?.querySelectorAll('button')[1]);
        await fireEvent.click(decline);
        await settle();
        expect(state.confirmed).toEqual([false]);
        expect(container.querySelector('.pending')).toBeNull();
    });
});
