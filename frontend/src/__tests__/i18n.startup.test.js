import { describe, test, expect, vi } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/main/Config.js', () => ({ SaveLanguage: vi.fn(() => Promise.resolve()) }));

describe('startupLanguage', () => {
    test('GetLanguage en échec : l’anglais est chargé, aucune clé brute', async () => {
        vi.resetModules();
        const { startupLanguage, t } = await import('../i18n');
        await startupLanguage(() => Promise.reject(new Error('boom')));
        expect(get(t)('common.back')).toBe('Back');
    });

    test('GetLanguage qui lève de façon synchrone : même repli', async () => {
        vi.resetModules();
        const { startupLanguage, t } = await import('../i18n');
        await startupLanguage(() => {
            throw new Error('missing binding');
        });
        expect(get(t)('common.back')).toBe('Back');
    });

    test('délai dépassé : l’anglais est chargé avant le montage', async () => {
        vi.resetModules();
        const { startupLanguage, t } = await import('../i18n');
        await startupLanguage(() => new Promise(() => {}), 10);
        expect(get(t)('common.back')).toBe('Back');
    });

    test('la langue persistée est chargée quand la lecture réussit', async () => {
        vi.resetModules();
        const { startupLanguage, t } = await import('../i18n');
        await startupLanguage(() => Promise.resolve('fr'));
        expect(get(t)('common.back')).toBe('Retour');
    });
});
