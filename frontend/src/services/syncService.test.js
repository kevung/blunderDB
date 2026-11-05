import { describe, it, expect } from 'vitest';
import { syncErrorKey, syncErrorIsVerbatim, syncErrorWords, syncSkipped, syncSkipReasonKey, syncStatusMessage } from './syncService.js';
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

describe('syncErrorWords', () => {
    it('words an exceeded allowance from its figures, once, in every language', async () => {
        const w = syncErrorWords({ kind: 'allowance', message: 'Plan limit reached', allowance: { axis: 'positions', allowed: 150, held: 140, new: 20 } });
        expect(w).toMatchObject({ key: 'sync.error.allowance', verbatim: '', detail: '' });
        expect(w.params).toMatchObject({ axis: 'sync.allowance.axis.positions', allowed: 150, held: 140, new: 20 });
        expect(syncErrorWords({ kind: 'allowance', message: 'x', allowance: { axis: 'unknown', allowed: 1, held: 1, new: 1 } }).params.axis).toBe('sync.allowance.axis.other');
        expect(syncErrorWords({ kind: 'allowance', message: 'The server said so' })).toMatchObject({ key: '', verbatim: 'The server said so', detail: '' });
        expect(syncErrorWords({ kind: 'server_refusal', message: 'no journal' })).toMatchObject({ key: '', verbatim: 'no journal' });
        expect(syncErrorWords({ kind: 'unavailable', message: 'dial tcp' })).toMatchObject({ key: 'sync.error.unavailable', detail: 'dial tcp' });
        expect(syncErrorWords({ kind: 'invalid_records', message: 'x' }).key).toBe('sync.error.invalid_records');
        for (const lang of ['de', 'el', 'en', 'es', 'fi', 'fr', 'it', 'ja', 'ru']) {
            const sync = (await import(`../i18n/locales/${lang}.json`)).default.sync;
            expect(sync.error.allowance, lang).toContain('{allowed}');
            expect(sync.error.allowance, lang).toContain('{new}');
            expect(sync.error.invalid_records, lang).toBeTruthy();
            for (const a of ['positions', 'matches', 'storage', 'collections', 'other']) expect(sync.allowance.axis[a], `${lang} ${a}`).toBeTruthy();
            for (const r of ['invalid', 'reference']) expect(sync.skipped[r], `${lang} ${r}`).toBeTruthy();
        }
    });
});

describe('syncStatusMessage', () => {
    it('tells on the status bar what a run left behind, since the tab may stay closed', async () => {
        const base = { running: false, server: '', hasToken: true, lastSuccess: '' };
        expect(syncStatusMessage(base)).toBeNull();
        expect(syncStatusMessage({ ...base, running: true, skipped: [{ direction: 'sent', family: 'match', uuid: 'u', reason: 'too_large' }] })).toBeNull();
        const skipped = [{ direction: 'sent', family: 'match', uuid: 'u', reason: 'too_large' }];
        const result = { sent: { records: 3, conflicts: 0, missing: 0, unsupported: 1 }, received: { records: 0, conflicts: 0, missing: 0, unsupported: 0 } };
        expect(syncStatusMessage({ ...base, skipped, result })).toEqual({ i18nKey: 'sync.skipped.status', i18nParams: { count: 2 } });
        expect(syncStatusMessage({ ...base, error: { kind: 'refused', message: 'HTTP 401' } })).toEqual({ i18nKey: 'sync.error.refused', i18nParams: {} });
        const allowance = syncStatusMessage({ ...base, error: { kind: 'allowance', message: 'Limite atteinte', allowance: { axis: 'matches', allowed: 10, held: 10, new: 1 } } });
        expect(allowance).toMatchObject({ i18nKey: 'sync.error.allowance', i18nParams: { axis: { i18nKey: 'sync.allowance.axis.matches' }, allowed: 10 } });
        for (const lang of ['de', 'el', 'en', 'es', 'fi', 'fr', 'it', 'ja', 'ru']) {
            const sync = (await import(`../i18n/locales/${lang}.json`)).default.sync;
            expect(sync.skipped.status, lang).toContain('{count}');
        }
    });
});
