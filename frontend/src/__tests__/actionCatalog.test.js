/**
 * actionCatalog.test.js — the menus built from the shared catalog: what a mode, a position and
 * the handlers at hand let through, the frequent actions at the first level, one submenu per
 * non-empty group in a fixed order, and a label in every language.
 */

import { describe, test, expect, vi } from 'vitest';
import { CATALOG, GROUPS, GROUP_LABELS, menuFor } from '../services/actionCatalog.js';
import en from '../i18n/locales/en.json';
import fr from '../i18n/locales/fr.json';

/** @param {string} key @param {Record<string, any>} [args] */
const t = (key, args) => (args ? `${key}${JSON.stringify(args)}` : key);

/** A handler for every action of a kind. @param {string} kind */
function allHandlers(kind) {
    return Object.fromEntries(CATALOG[kind].map((a) => [a.id, vi.fn()]));
}

/** @param {any[]} items */
const labels = (items) => items.map((i) => i.label);
/** @param {any[]} items @param {string} group */
const submenu = (items, group) => items.find((i) => i.label === `actions.group.${group}`)?.submenu;

describe('actionCatalog', () => {
    test('a library position: the frequent actions first, then the groups in their order', () => {
        const items = menuFor('position', { mode: 'NORMAL', positionId: 42, dbOpen: true, ankiDecks: [], do: allHandlers('position') }, t);
        expect(labels(items)).toEqual([
            'board.menu.evaluate',
            'actions.copyImage',
            'actions.group.copy',
            'actions.group.annotate',
            'actions.group.navigate',
            'actions.group.analyze',
            'actions.group.train',
            'actions.group.manage'
        ]);
        expect(items[2].separatorBefore).toBe(true);
        expect(labels(submenu(items, 'annotate'))).toEqual(['actions.comment', 'actions.addTag', 'actions.markStudied', 'actions.unmarkStudied']);
        expect(labels(submenu(items, 'navigate'))).toEqual(['board.menu.neighbours', 'actions.sameStructure', 'board.menu.newView']);
        expect(labels(submenu(items, 'manage'))).toEqual(['actions.trash']);
        // Évaluer stays at the first level and leads its group.
        expect(labels(submenu(items, 'analyze'))[0]).toBe('board.menu.evaluate');
    });

    test('the studied mark: only the action that applies to the state is offered', () => {
        const ctx = { mode: 'NORMAL', positionId: 42, dbOpen: true, do: allHandlers('position') };
        const marks = (/** @type {boolean} */ studiedMark) => labels(submenu(menuFor('position', { ...ctx, studiedMark }, t), 'annotate')).filter((l) => /Studied$/.test(l));
        expect(marks(false)).toEqual(['actions.markStudied']);
        expect(marks(true)).toEqual(['actions.unmarkStudied']);
    });

    test('an action without a handler is not offered, and an emptied group disappears', () => {
        const handlers = allHandlers('position');
        for (const id of ['comment', 'addTag', 'markStudied', 'unmarkStudied']) delete handlers[id];
        const items = menuFor('position', { mode: 'NORMAL', positionId: 42, dbOpen: true, do: handlers }, t);
        expect(submenu(items, 'annotate')).toBeUndefined();
    });

    test('the Eval scratch board: saved with Ctrl-S, neither annotated nor trashed', () => {
        const items = menuFor('position', { mode: 'EVAL', positionId: 0, dbOpen: true, do: allHandlers('position') }, t);
        expect(submenu(items, 'annotate')).toBeUndefined();
        expect(submenu(items, 'manage')).toEqual([expect.objectContaining({ label: 'actions.save', shortcut: 'keyNames.ctrl+S' })]);
        expect(labels(submenu(items, 'navigate') ?? [])).not.toContain('board.menu.neighbours');
    });

    test('the Search board is not evaluated; without a library nothing is saved', () => {
        const items = menuFor('position', { mode: 'EDIT', positionId: 0, dbOpen: false, do: allHandlers('position') }, t);
        expect(labels(items)).not.toContain('board.menu.evaluate');
        expect(submenu(items, 'analyze')).toBeUndefined();
        expect(submenu(items, 'manage')).toBeUndefined();
    });

    test('one Anki entry per deck, each handing its deck to the handler', () => {
        const handlers = allHandlers('position');
        const decks = [
            { id: 7, name: 'Openings' },
            { id: 8, name: 'Bearoff' }
        ];
        const items = menuFor('position', { mode: 'NORMAL', positionId: 42, dbOpen: true, ankiDecks: decks, do: handlers }, t);
        const train = submenu(items, 'train');
        expect(labels(train)).toEqual(['board.menu.addToAnkiDeck{"deck":"Openings"}', 'board.menu.addToAnkiDeck{"deck":"Bearoff"}', 'actions.challenge']);
        train[1].onClick?.();
        expect(handlers.addToAnkiDeck).toHaveBeenCalledWith(decks[1]);
    });

    test('the challenge is offered while it is off, with its key', () => {
        const on = menuFor('position', { mode: 'NORMAL', positionId: 42, dbOpen: true, challengeOn: true, do: allHandlers('position') }, t);
        expect(submenu(on, 'train')).toBeUndefined();
        const off = menuFor('position', { mode: 'NORMAL', positionId: 42, dbOpen: true, challengeOn: false, do: allHandlers('position') }, t);
        expect(submenu(off, 'train')).toEqual([expect.objectContaining({ label: 'actions.challenge', shortcut: 'm' })]);
    });

    test('a click runs the handler of its action', () => {
        const handlers = allHandlers('position');
        const items = menuFor('position', { mode: 'NORMAL', positionId: 42, dbOpen: true, do: handlers }, t);
        items[0].onClick?.();
        expect(handlers.evaluate).toHaveBeenCalledTimes(1);
        submenu(items, 'manage')[0].onClick?.();
        expect(handlers.trash).toHaveBeenCalledTimes(1);
    });

    test.each(['match', 'tournament', 'collection'])('a %s row: open first, then the groups its handlers fill', (kind) => {
        const items = menuFor(kind, { do: allHandlers(kind) }, t);
        expect(items[0].label).toMatch(/^actions\.open/);
        expect(labels(submenu(items, 'manage'))).toContain(kind === 'tournament' ? 'common.delete' : 'actions.trash');
        expect(menuFor(kind, { do: {} }, t)).toEqual([]);
    });

    test('an unknown kind is a programming error', () => {
        expect(() => menuFor('player', { do: {} }, t)).toThrow(/unknown kind/);
    });

    test('every label and group exists in English and in French', () => {
        /** @param {any} dict @param {string} key */
        const has = (dict, key) => key.split('.').reduce((o, k) => o?.[k], dict) !== undefined;
        const keys = [...Object.values(CATALOG).flatMap((actions) => actions.map((a) => a.label)), ...GROUPS.map((g) => GROUP_LABELS[g])];
        for (const key of keys) {
            expect(has(en, key), `en: ${key}`).toBe(true);
            expect(has(fr, key), `fr: ${key}`).toBe(true);
        }
    });
});
