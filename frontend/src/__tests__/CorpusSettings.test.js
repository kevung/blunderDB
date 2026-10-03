/**
 * CorpusSettings.svelte: the corpus tab of the settings. It mounts, hands the
 * skip-duplicates toggle to the Database, edits the aliases of the chosen kind
 * (player or event) through the bindings, applies a suggestion, and lists the
 * probable duplicates FindDuplicateMatches returns.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

const calls = {
    SetSkipDuplicates: vi.fn(() => Promise.resolve()),
    ListAliases: vi.fn(() => Promise.resolve([])),
    SetAlias: vi.fn(() => Promise.resolve()),
    RemoveAlias: vi.fn(() => Promise.resolve(true)),
    SuggestAliases: vi.fn(() => Promise.resolve([{ canonical: 'Alice', aliases: ['alice'] }])),
    FindDuplicateMatches: vi.fn(() =>
        Promise.resolve([
            { kind: 'same_dice', matchId: 7, otherId: 3, players: 'Durand B. – Martin A.', otherPlayers: 'Alice Martin – Bob Durand' },
            { kind: 'longer', matchId: 9, otherId: 4, players: 'Carol – Dave', otherPlayers: 'Carol – Dave' }
        ])
    )
};
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SetSkipDuplicates: (...a) => calls.SetSkipDuplicates(...a),
    ListAliases: (...a) => calls.ListAliases(...a),
    SetAlias: (...a) => calls.SetAlias(...a),
    RemoveAlias: (...a) => calls.RemoveAlias(...a),
    SuggestAliases: (...a) => calls.SuggestAliases(...a),
    FindDuplicateMatches: (...a) => calls.FindDuplicateMatches(...a)
}));

import CorpusSettings from '../components/CorpusSettings.svelte';
import { skipDuplicatesStore } from '../stores/corpusStore.js';

async function settle() {
    for (let i = 0; i < 4; i++) await tick();
}

beforeEach(() => {
    Object.values(calls).forEach((f) => f.mockClear());
    skipDuplicatesStore.set(false);
});
afterEach(() => cleanup());

describe('CorpusSettings', () => {
    test('mounts and lists the player aliases', async () => {
        calls.ListAliases.mockImplementationOnce(() => Promise.resolve([{ alias: 'Martin A.', canonical: 'Alice Martin' }]));
        const { getByText } = render(CorpusSettings);
        await settle();
        expect(calls.ListAliases).toHaveBeenCalledWith('player');
        expect(getByText('Martin A.')).toBeTruthy();
        expect(getByText('Alice Martin')).toBeTruthy();
    });

    test('the skip-duplicates toggle reaches the Database', async () => {
        const { container } = render(CorpusSettings);
        await settle();
        const box = container.querySelector('#config-skip-duplicates');
        await fireEvent.click(box);
        await settle();
        expect(calls.SetSkipDuplicates).toHaveBeenCalledWith(true);
        expect(get(skipDuplicatesStore)).toBe(true);
    });

    test('adds an event alias as typed, then a suggested player alias', async () => {
        const { container, getByTestId, findByText } = render(CorpusSettings);
        await settle();
        const kindButtons = container.querySelectorAll('.kind-switch button');
        await fireEvent.click(kindButtons[1]);
        await settle();
        expect(calls.ListAliases).toHaveBeenLastCalledWith('event');
        const [alias, canonical] = container.querySelectorAll('.alias-form input');
        await fireEvent.input(alias, { target: { value: 'Open 25' } });
        await fireEvent.input(canonical, { target: { value: 'Autumn Open 2025' } });
        await fireEvent.click(container.querySelector('.alias-form button'));
        await settle();
        expect(calls.SetAlias).toHaveBeenCalledWith('event', 'Open 25', 'Autumn Open 2025');

        await fireEvent.click(kindButtons[0]);
        await settle();
        await fireEvent.click(getByTestId('suggest-aliases'));
        await settle();
        expect(calls.SuggestAliases).toHaveBeenCalledWith('player');
        await findByText(/alice/);
        await fireEvent.click(container.querySelector('.suggestions button'));
        await settle();
        expect(calls.SetAlias).toHaveBeenLastCalledWith('player', 'alice', 'Alice');
    });

    test('lists the probable duplicates', async () => {
        const { getByTestId } = render(CorpusSettings);
        await settle();
        await fireEvent.click(getByTestId('find-duplicates'));
        await settle();
        expect(calls.FindDuplicateMatches).toHaveBeenCalled();
        const items = getByTestId('duplicate-suspects').querySelectorAll('li');
        expect(items).toHaveLength(2);
        expect(items[0].textContent).toContain('#3');
        expect(items[0].textContent).toContain('#7');
        expect(items[1].textContent).toContain('#9');
    });
});
