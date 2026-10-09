/**
 * pageStep.sync.test.js — the PageUp / PageDown steps are declared twice: config.go (what the
 * backend accepts and stores) and uiStore.js (what the Settings offer). A value offered but
 * not accepted would be saved and silently reset to the default on the next launch.
 */
import { must } from './helpers/must.js';
import { describe, test, expect } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { PAGE_STEPS, PAGE_STEP_DEFAULT } from '../stores/uiStore.js';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..', '..');
const configGo = fs.readFileSync(path.join(ROOT, 'config.go'), 'utf8');

describe('page steps mirror config.go', () => {
    test('PageSteps', () => {
        const m = /var PageSteps = \[\]string\{([^}]*)\}/.exec(configGo);
        expect(m).not.toBeNull();
        expect(JSON.parse(`[${must(m)[1]}]`)).toEqual([...PAGE_STEPS]);
    });

    test('DefaultPageStep', () => {
        const m = /\bDefaultPageStep\s*=\s*"([^"]*)"/.exec(configGo);
        expect(m?.[1]).toBe(PAGE_STEP_DEFAULT);
    });
});
