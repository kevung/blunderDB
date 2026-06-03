import { describe, test, expect } from 'vitest';
import { getCommandSuggestions, COMMANDS } from '../commandVocabulary.js';

describe('getCommandSuggestions', () => {
    test('empty / whitespace input yields no suggestions', () => {
        expect(getCommandSuggestions('')).toEqual([]);
        expect(getCommandSuggestions('   ')).toEqual([]);
        expect(getCommandSuggestions(null)).toEqual([]);
    });

    test('only suggests for the command word (stops after a space)', () => {
        expect(getCommandSuggestions('s p>30')).toEqual([]);
        expect(getCommandSuggestions('write ')).toEqual([]);
    });

    test('does not suggest for position numbers or tags', () => {
        expect(getCommandSuggestions('12')).toEqual([]);
        expect(getCommandSuggestions('#blunder')).toEqual([]);
    });

    test('matches by canonical name prefix', () => {
        const names = getCommandSuggestions('imp').map((c) => c.name);
        expect(names).toContain('import');
        expect(names).toContain('import_db');
    });

    test('matches by alias prefix and returns the canonical name', () => {
        const names = getCommandSuggestions('wr').map((c) => c.name);
        // 'wr' is an alias of both write and write!
        expect(names).toContain('write');
        expect(names).toContain('write!');
    });

    test('is case-insensitive', () => {
        const lower = getCommandSuggestions('ep').map((c) => c.name);
        const upper = getCommandSuggestions('EP').map((c) => c.name);
        expect(lower).toEqual(upper);
        expect(lower).toContain('epc');
    });

    test('exact command still appears (so it can be confirmed)', () => {
        expect(getCommandSuggestions('epc').map((c) => c.name)).toContain('epc');
    });

    test('every command entry has a non-empty name', () => {
        for (const cmd of COMMANDS) {
            expect(typeof cmd.name).toBe('string');
            expect(cmd.name.length).toBeGreaterThan(0);
            expect(Array.isArray(cmd.aliases)).toBe(true);
        }
    });
});
