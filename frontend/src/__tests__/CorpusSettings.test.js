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
            {
                kind: 'same_dice',
                matchId: 7,
                otherId: 3,
                players: 'Durand B. – Martin A.',
                otherPlayers: 'Alice Martin – Bob Durand',
                pairings: [
                    {
                        aliases: [
                            { alias: 'Durand B.', canonical: 'Alice Martin' },
                            { alias: 'Martin A.', canonical: 'Bob Durand' }
                        ]
                    },
                    {
                        aliases: [
                            { alias: 'Durand B.', canonical: 'Bob Durand' },
                            { alias: 'Martin A.', canonical: 'Alice Martin' }
                        ]
                    }
                ]
            },
            { kind: 'longer', matchId: 9, otherId: 4, players: 'Carol – Dave', otherPlayers: 'Carol – Dave' }
        ])
    )
};
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SetSkipDuplicates: (/** @type {any[]} */ ...a) => calls.SetSkipDuplicates(...a),
    ListAliases: (/** @type {any[]} */ ...a) => calls.ListAliases(...a),
    SetAlias: (/** @type {any[]} */ ...a) => calls.SetAlias(...a),
    RemoveAlias: (/** @type {any[]} */ ...a) => calls.RemoveAlias(...a),
    SuggestAliases: (/** @type {any[]} */ ...a) => calls.SuggestAliases(...a),
    FindDuplicateMatches: (/** @type {any[]} */ ...a) => calls.FindDuplicateMatches(...a)
}));

import CorpusSettings from '../components/CorpusSettings.svelte';
import { skipDuplicatesStore } from '../stores/corpusStore.js';
import { must } from './helpers/must.js';

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
        await fireEvent.click(must(box));
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
        await fireEvent.click(must(container.querySelector('.alias-form button')));
        await settle();
        expect(calls.SetAlias).toHaveBeenCalledWith('event', 'Open 25', 'Autumn Open 2025');

        await fireEvent.click(kindButtons[0]);
        await settle();
        await fireEvent.click(getByTestId('suggest-aliases'));
        await settle();
        expect(calls.SuggestAliases).toHaveBeenCalledWith('player');
        await findByText(/alice/);
        await fireEvent.click(must(container.querySelector('.suggestions button')));
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

    test('records the aliases of the reading chosen, in one click', async () => {
        const { getByTestId, getAllByTestId, queryAllByTestId } = render(CorpusSettings);
        await settle();
        await fireEvent.click(getByTestId('find-duplicates'));
        await settle();
        const buttons = getAllByTestId('apply-pairing');
        expect(buttons).toHaveLength(2);
        await fireEvent.click(buttons[1]);
        await settle();
        expect(calls.SetAlias).toHaveBeenCalledTimes(2);
        expect(calls.SetAlias).toHaveBeenNthCalledWith(1, 'player', 'Durand B.', 'Bob Durand');
        expect(calls.SetAlias).toHaveBeenNthCalledWith(2, 'player', 'Martin A.', 'Alice Martin');
        expect(queryAllByTestId('apply-pairing')).toHaveLength(0);
        expect(getByTestId('pairing-applied')).toBeTruthy();
    });
});
