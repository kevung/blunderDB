/**
 * Le résultat d'un match dans l'Historique de la direction (#408, question 12).
 *
 * Le moteur encode les scores avec `omitempty` : un 7–0 arrive sans le zéro, et un résultat
 * saisi sans score arrive sans aucun des deux. La ligne affichait « 7–undefined » et
 * « undefined–undefined ». Un côté absent est un zéro omis ; les deux absents disent
 * qu'aucun score n'a été saisi, et la ligne n'en invente pas un (« 0–0 » serait faux).
 */

import { describe, test, expect, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';

import HistoryView from '../components/direction/HistoryView.svelte';

afterEach(cleanup);

/** @param {Record<string, any>} fields */
function renderResult(fields) {
    const entries = [{ kind: 'result', winnerName: 'Alice', matchId: 'M1', ...fields }];
    return render(HistoryView, { props: { entries } }).container.textContent || '';
}

describe("le résultat d'un match dans l'historique", () => {
    test('un côté absent est un zéro omis : 7–0', () => {
        const text = renderResult({ scoreA: 7 });
        expect(text).not.toContain('undefined');
        expect(text).toContain('7–0');
    });

    test("l'autre côté absent aussi : 0–7", () => {
        const text = renderResult({ scoreB: 7 });
        expect(text).not.toContain('undefined');
        expect(text).toContain('0–7');
    });

    test('sans aucun score saisi, la ligne nomme le vainqueur sans inventer de score', () => {
        const text = renderResult({});
        expect(text).toContain('Alice');
        expect(text).not.toContain('undefined');
        expect(text).not.toContain('0–0');
    });

    test('un score complet reste tel quel', () => {
        const text = renderResult({ scoreA: 7, scoreB: 5 });
        expect(text).toContain('7–5');
    });

    test('un forfait garde sa phrase, sans score', () => {
        const text = renderResult({ forfeit: true });
        expect(text).toContain('Alice');
        expect(text).not.toContain('undefined');
        expect(text).not.toContain('–0');
    });
});
