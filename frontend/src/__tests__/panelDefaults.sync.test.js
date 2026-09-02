/**
 * panelDefaults.sync.test.js
 *
 * The tabbed panel's default size is declared twice: config.go (what the
 * backend answers on a first launch, and what it clamps a missing value to)
 * and panelLayoutStore.js (what App.svelte draws with until that answer
 * arrives). The two drifted — 380/520 against 250/420 — and the panel
 * jumped on every first launch (#201). Same pattern as fontScale.sync.test.js:
 * read the source of truth and compare.
 */

import { describe, test, expect, vi } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetPanelPosition: vi.fn(),
    SavePanelPosition: vi.fn(),
    GetPanelHeight: vi.fn(),
    SavePanelHeight: vi.fn(),
    GetPanelWidth: vi.fn(),
    SavePanelWidth: vi.fn()
}));

import { DEFAULT_PANEL_HEIGHT, DEFAULT_PANEL_WIDTH, DEFAULT_PANEL_POSITION } from '../stores/panelLayoutStore.js';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..', '..');
const configGo = fs.readFileSync(path.join(ROOT, 'config.go'), 'utf8');

function goConst(name) {
    const m = new RegExp(`\\b${name}\\s*=\\s*([^\\n]+)`).exec(configGo);
    if (!m) throw new Error(`${name} not found in config.go`);
    return m[1].trim();
}

describe('panel defaults mirror config.go', () => {
    test('DefaultPanelHeight', () => {
        expect(DEFAULT_PANEL_HEIGHT).toBe(Number(goConst('DefaultPanelHeight')));
    });

    test('DefaultPanelWidth', () => {
        expect(DEFAULT_PANEL_WIDTH).toBe(Number(goConst('DefaultPanelWidth')));
    });

    test('DefaultPanelPosition', () => {
        // `DefaultPanelPosition = PanelPositionBottom`: resolve the named constant.
        const named = goConst('DefaultPanelPosition');
        const literal = /^"(.*)"$/.exec(goConst(named))?.[1];
        expect(DEFAULT_PANEL_POSITION).toBe(literal);
    });
});
