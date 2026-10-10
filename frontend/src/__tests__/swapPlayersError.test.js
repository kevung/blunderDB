/**
 * swapPlayersError.test.js — a refused swap of a match's players says why,
 * in the user's language; any other error stays the panel's failure message.
 */
import { describe, test, expect } from 'vitest';
import { swapPlayersErrorKey } from '../utils/swapPlayersError.js';
import fr from '../i18n/locales/fr.json';
import en from '../i18n/locales/en.json';
import de from '../i18n/locales/de.json';
import el from '../i18n/locales/el.json';
import es from '../i18n/locales/es.json';
import fi from '../i18n/locales/fi.json';
import it from '../i18n/locales/it.json';
import ja from '../i18n/locales/ja.json';
import ru from '../i18n/locales/ru.json';

describe('swapPlayersErrorKey', () => {
    test('names each refusal of the backend', () => {
        expect(swapPlayersErrorKey('swap players: match 7: storage: invalid argument: a transcription draft is open on the match', 'match.errorSwapping')).toBe('match.swapDraftOpen');
        expect(swapPlayersErrorKey(new Error('match 7: storage: invalid argument: the match was played here, under arbitration'), 'tournament.errorSwapping')).toBe('match.swapPlayedHere');
    });

    test('leaves any other error to the caller', () => {
        expect(swapPlayersErrorKey('no database is currently open', 'tournament.errorSwapping')).toBe('tournament.errorSwapping');
    });

    test('every language says both refusals', () => {
        for (const locale of [fr, en, de, el, es, fi, it, ja, ru]) {
            expect(locale.match.swapDraftOpen).toBeTruthy();
            expect(locale.match.swapPlayedHere).toBeTruthy();
        }
    });
});
