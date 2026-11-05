import { describe, it, expect } from 'vitest';
import { syncErrorKey, syncErrorIsVerbatim, syncSkipped, syncSkipReasonKey } from './syncService.js';
import en from '../i18n/locales/en.json';

describe('syncErrorKey', () => {
    it('words every kind the Go side reports, and falls back for an unforeseen one', () => {
        for (const kind of ['refused', 'unavailable', 'rate_limited', 'too_large', 'blocked', 'address', 'no_token', 'no_library', 'library_changed', 'tool_locked']) {
            const key = syncErrorKey(kind);
            expect(key).toBe(`sync.error.${kind}`);
            expect(/** @type {Record<string, string>} */ (en.sync.error)[kind]).toBeTruthy();
        }
        expect(syncErrorKey('something_new')).toBe('sync.error.other');
    });
});

describe('syncErrorIsVerbatim', () => {
    it('shows the server words as they are for an exceeded allowance only', () => {
        expect(syncErrorIsVerbatim('allowance')).toBe(true);
        expect(syncErrorIsVerbatim('refused')).toBe(false);
    });
});

describe('syncUnapplied', () => {
    it('counts what either side could not apply, so a passed-on record is never shown as done', async () => {
        const { syncUnapplied } = await import('./syncService.js');
        expect(syncUnapplied(null)).toBe(0);
        expect(
            syncUnapplied({
                sent: { records: 5, conflicts: 1, missing: 0, unsupported: 0 },
                received: { records: 4, conflicts: 0, missing: 2, unsupported: 3, unsynchronizable: 1 }
            })
        ).toBe(6);
    });
});

describe('the records set aside', () => {
    it('are listed from the state, and each reason has its wording in every language', async () => {
        expect(syncSkipped({ running: false, server: '', hasToken: false, lastSuccess: '' })).toEqual([]);
        const skipped = [{ direction: 'sent', family: 'match', uuid: 'u-1', reason: 'interrupted' }];
        expect(syncSkipped({ running: false, server: '', hasToken: false, lastSuccess: '', skipped })).toEqual(skipped);
        expect(syncSkipReasonKey('interrupted')).toBe('sync.skipped.interrupted');
        expect(syncSkipReasonKey('too_large')).toBe('sync.skipped.too_large');
        for (const lang of ['de', 'el', 'en', 'es', 'fi', 'fr', 'it', 'ja', 'ru']) {
            const sync = (await import(`../i18n/locales/${lang}.json`)).default.sync;
            for (const k of ['title', 'item', 'sent', 'received', 'too_large', 'interrupted']) expect(sync.skipped[k], `${lang} sync.skipped.${k}`).toBeTruthy();
            expect(sync.error.blocked, `${lang} sync.error.blocked`).toBeTruthy();
        }
    });
});
