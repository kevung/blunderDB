import { describe, it, expect } from 'vitest';
import { parseVersion, isNewerVersion } from '../utils/semver.js';

describe('parseVersion', () => {
    it('parses a plain X.Y.Z string', () => {
        expect(parseVersion('0.36.0')).toEqual([0, 36, 0]);
        expect(parseVersion('12.3.45')).toEqual([12, 3, 45]);
    });

    it('returns null for anything else', () => {
        expect(parseVersion('v0.36.0')).toBeNull();
        expect(parseVersion('0.36')).toBeNull();
        expect(parseVersion('0.36.0-rc1')).toBeNull();
        expect(parseVersion('')).toBeNull();
        expect(parseVersion(undefined)).toBeNull();
        expect(parseVersion(null)).toBeNull();
    });
});

describe('isNewerVersion', () => {
    it('reports a newer patch/minor/major as newer', () => {
        expect(isNewerVersion('0.36.1', '0.36.0')).toBe(true);
        expect(isNewerVersion('0.37.0', '0.36.9')).toBe(true);
        expect(isNewerVersion('1.0.0', '0.99.99')).toBe(true);
    });

    it('reports the same or an older version as not newer', () => {
        expect(isNewerVersion('0.36.0', '0.36.0')).toBe(false);
        expect(isNewerVersion('0.35.9', '0.36.0')).toBe(false);
        expect(isNewerVersion('0.36.0', '0.36.1')).toBe(false);
    });

    it('never throws on an unparsable version, and reports false', () => {
        expect(isNewerVersion('not-a-version', '0.36.0')).toBe(false);
        expect(isNewerVersion('0.36.0', 'not-a-version')).toBe(false);
        expect(isNewerVersion(undefined, '0.36.0')).toBe(false);
    });
});
