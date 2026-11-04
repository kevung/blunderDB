import { describe, it, expect } from 'vitest';
import { syncErrorKey } from './syncService.js';
import en from '../i18n/locales/en.json';

describe('syncErrorKey', () => {
    it('words every kind the Go side reports, and falls back for an unforeseen one', () => {
        for (const kind of ['refused', 'unavailable', 'rate_limited', 'too_large', 'address', 'no_token', 'no_library']) {
            const key = syncErrorKey(kind);
            expect(key).toBe(`sync.error.${kind}`);
            expect(/** @type {Record<string, string>} */ (en.sync.error)[kind]).toBeTruthy();
        }
        expect(syncErrorKey('something_new')).toBe('sync.error.other');
    });
});
