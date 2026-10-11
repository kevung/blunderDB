import { describe, expect, it, vi } from 'vitest';
import fr from '../i18n/locales/fr.json';

vi.mock('../i18n', () => ({
    translate: (key, params = {}) => {
        const text = key.split('.').reduce((node, part) => node?.[part], fr) ?? key;
        return text.replace(/\{(\w+)\}/g, (_, name) => String(params[name]));
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
