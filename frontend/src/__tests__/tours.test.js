import { describe, test, expect } from 'vitest';
import { TOURS, getTourById } from '../tours.js';
import en from '../i18n/locales/en.json';

function resolve(key) {
    return key.split('.').reduce((o, k) => (o == null ? undefined : o[k]), en);
}

describe('tours data', () => {
    test('has at least the general tour', () => {
        expect(getTourById('general')).toBeTruthy();
    });

    test('tour ids are unique', () => {
        const ids = TOURS.map((t) => t.id);
        expect(new Set(ids).size).toBe(ids.length);
    });

    test('every tour and step references i18n keys that exist in en.json', () => {
        for (const tour of TOURS) {
            expect(typeof resolve(tour.titleKey)).toBe('string');
            expect(typeof resolve(tour.descKey)).toBe('string');
            expect(tour.steps.length).toBeGreaterThan(0);
            for (const step of tour.steps) {
                expect(typeof resolve(step.titleKey)).toBe('string');
                expect(typeof resolve(step.bodyKey)).toBe('string');
                // element, when present, must be a non-empty selector string
                if (step.element !== undefined) {
                    expect(typeof step.element).toBe('string');
                    expect(step.element.length).toBeGreaterThan(0);
                }
            }
        }
    });

    test('navigation button labels exist', () => {
        for (const key of ['tour.next', 'tour.prev', 'tour.done', 'tour.start', 'tour.catalogTitle', 'tour.catalogDesc']) {
            expect(typeof resolve(key)).toBe('string');
        }
    });
});
