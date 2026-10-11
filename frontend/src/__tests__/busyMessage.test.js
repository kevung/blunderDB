import { describe, expect, it, vi } from 'vitest';
import fr from '../i18n/locales/fr.json';

vi.mock('../i18n', () => ({
    /** @param {string} key @param {Record<string, unknown>} [params] */
    translate: (key, params = {}) => {
        /** @type {any} */
        const text = key.split('.').reduce((/** @type {any} */ node, part) => node?.[part], fr) ?? key;
        return String(text).replace(/\{(\w+)\}/g, (/** @type {string} */ _, /** @type {string} */ name) => String(params[name]));
    }
}));

const { busyMessage } = await import('../utils/busyMessage.js');

describe('busyMessage', () => {
    it('names each job code Go refuses with', () => {
        expect(busyMessage('database is busy: rollout')).toBe('Base occupée : un rollout en cours');
        expect(busyMessage('rollout: database is busy: gammonnet')).toBe('Base occupée : une analyse gammonNet en cours');
        expect(busyMessage('database is busy: maintenance')).toBe('Base occupée : une opération de maintenance en cours');
    });

    it('leaves any other message to its caller', () => {
        expect(busyMessage('database is read-only')).toBeNull();
        expect(busyMessage(undefined)).toBeNull();
    });
});
