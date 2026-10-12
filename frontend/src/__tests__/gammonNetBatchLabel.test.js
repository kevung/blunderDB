import { describe, expect, it } from 'vitest';
import fr from '../i18n/locales/fr.json';
import { gammonNetBatchLabel } from '../utils/gammonNetBatchLabel.js';

// A minimal t over the French catalogue: the label is built from the
// catalogue's keys, never from a hard-coded unit.
function t(key, params = {}) {
    const text = key.split('.').reduce((node, part) => node[part], fr);
    return text.replace(/\{(\w+)\}/g, (_, name) => String(params[name]));
}

describe('gammonNetBatchLabel', () => {
    it('names the depth through the catalogue', () => {
        expect(gammonNetBatchLabel(t, { done: 0, total: 252, ply: 2 })).toBe('gammonNet analyse 2-ply : 0/252');
    });

    it('falls back to the counts alone, without a double space, when no depth is sent', () => {
        const label = gammonNetBatchLabel(t, { done: 3, total: 10 });
        expect(label).toBe('gammonNet analyse : 3/10');
        expect(label).not.toMatch(/ {2}/);
    });

    it('names the job it waits behind while queued', () => {
        expect(gammonNetBatchLabel(t, { done: 0, total: 0, ply: 2, job: 'rollout' })).toBe('gammonNet en attente : un rollout en cours');
    });
});
