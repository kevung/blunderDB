/**
 * Un jour lu de "2025-03-02" est minuit UTC ; l'afficher en heure locale donne la veille à
 * l'ouest de Greenwich. TZ est posé avant tout `Date`.
 */
import { describe, test, expect, beforeAll } from 'vitest';

beforeAll(() => {
    process.env.TZ = 'America/New_York';
});

describe('jours UTC sous un fuseau négatif', () => {
    test('le constat : formater l’instant en heure locale glisse d’un jour', async () => {
        const { formatDate } = await import('../utils/format.js');
        const ms = new Date('2025-03-02').getTime();
        expect(formatDate(new Date(ms))).toBe(formatDate(new Date(2025, 2, 1)));
    });

    test('formatUtcDay et formatIsoDay donnent le même jour que la date lue', async () => {
        const { formatUtcDay, formatIsoDay, formatDate } = await import('../utils/format.js');
        const ms = new Date('2025-03-02').getTime();
        expect(formatUtcDay(ms)).toBe(formatDate(new Date(2025, 2, 2)));
        expect(formatUtcDay(ms)).toBe(formatIsoDay('2025-03-02'));
    });
});
