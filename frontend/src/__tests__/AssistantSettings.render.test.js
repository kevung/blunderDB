/**
 * AssistantSettings.svelte : le serveur MCP local se monte éteint, son réglage s'enregistre et
 * s'applique, et l'adresse d'écoute s'affiche quand il tourne.
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/runtime/runtime.js', () => ({ EventsOn: () => () => {}, EventsOff: () => {} }));
const calls = vi.hoisted(() => ({ saved: /** @type {any[]} */ ([]), configured: /** @type {any[]} */ ([]) }));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetMCPHost: () => Promise.resolve({ on: false, port: 0, write: false }),
    GetAssistant: () => Promise.resolve({ on: false, preset: 'groq', baseURL: '', model: '', remoteAck: '' }),
    SaveAssistant: () => Promise.resolve(),
    SaveMCPHost: (s) => {
        calls.saved.push(s);
        return Promise.resolve();
    }
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    GetMCPHostStatus: () => Promise.resolve({ running: false, url: '', write: false, error: '' }),
    AssistantPresets: () =>
        Promise.resolve([
            { id: 'ollama', name: 'Ollama', baseURL: 'http://localhost:11434/v1', model: 'qwen2.5:7b', remote: false, needsKey: false },
            { id: 'groq', name: 'Groq', baseURL: 'https://api.groq.com/openai/v1', model: 'llama', remote: true, needsKey: true }
        ]),
    AssistantHasKey: () => Promise.resolve(true),
    AssistantSetKey: () => Promise.resolve(),
    ConfigureMCPHost: (c) => {
        calls.configured.push(c);
        return Promise.resolve({ running: c.enabled, url: c.enabled ? `http://127.0.0.1:${c.port}/mcp` : '', write: c.write, error: '' });
    }
}));

import AssistantSettings from '../components/AssistantSettings.svelte';

beforeEach(() => {
    calls.saved.length = 0;
    calls.configured.length = 0;
});
afterEach(() => cleanup());

describe('AssistantSettings', () => {
    test('mounts off, then enabling saves and starts the localhost server', async () => {
        const { container, findByTestId } = render(AssistantSettings);
        await tick();
        const on = /** @type {HTMLInputElement} */ (container.querySelector('#config-mcp-on'));
        expect(on.checked).toBe(false);
        expect(/** @type {HTMLInputElement} */ (container.querySelector('#config-mcp-port')).value).toBe('8765');
        await fireEvent.click(on);
        expect(calls.saved.at(-1)).toEqual({ on: true, port: 8765, write: false });
        expect(calls.configured.at(-1)).toEqual({ enabled: true, port: 8765, write: false });
        expect((await findByTestId('mcp-url')).textContent).toContain('http://127.0.0.1:8765/mcp');
    });

    test('a remote provider shows its privacy warning; the key is said to be in the keyring', async () => {
        const { findByTestId, container } = render(AssistantSettings);
        expect((await findByTestId('assistant-privacy')).textContent).toContain('Groq');
        expect((await findByTestId('assistant-key-state')).textContent).toBeTruthy();
        expect(/** @type {HTMLInputElement} */ (container.querySelector('#config-assistant-on')).checked).toBe(false);
    });
});
