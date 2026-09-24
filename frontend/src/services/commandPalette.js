// commandPalette.js — what the command palette offers, how it ranks it, and
// what choosing an entry does (#287).
//
// The palette owns no list of its own. It reads four that already exist:
//
//   - the commands of the command line, from commandVocabulary.js — the list
//     autocompletion and commandVocabulary.sync.test.js already hold to
//     commandProcessor.js, so a new command reaches the palette by being added
//     there, and a palette entry runs through processCommand exactly as if it
//     had been typed;
//   - the tabs of the tabbed panel, from tabCatalog.js;
//   - the saved filters, from filterLibraryStore (the search panel's library);
//   - the matches, from GetAllMatches — the call the match panel makes.
//
// Only the short description of each command is the palette's own: a command
// name alone ("cm", "gv2") says nothing to someone looking for "cube matrix".
// Those live in the nine locales under palette.cmd.<name>, and a test requires
// one for every command of the vocabulary.

import { get } from 'svelte/store';
import { COMMANDS } from '../commandVocabulary.js';
import { TABS } from './tabCatalog.js';
import { fuzzyMatch, fold } from '../utils/fuzzy.js';
import { formatDate } from '../utils/matchTable.js';
import { GetAllMatches } from '../../wailsjs/go/database/Database.js';
import { activeTabStore, commandTextStore, showCommandInputStore, commandPaletteOpenStore, matchOpenRequestStore } from '../stores/uiStore.js';
import { databaseLoadedStore } from '../stores/databaseStore.js';
import { loadFilterLibrary, runSavedFilter } from './filterLibraryService.js';
import { showTab } from './tabToggles.js';
import { toggleEvalMode } from './modeMachine.js';
import { processCommand } from '../commandProcessor.js';
import { logger } from '../utils/logger.js';

/** @typedef {'command' | 'tab' | 'filter' | 'match'} PaletteKind */

/**
 * @typedef {object} PaletteItem
 * @property {PaletteKind} kind
 * @property {string} id        unique across the palette ("command:new", "match:12")
 * @property {string} label     what the entry is called, in the interface language
 * @property {string} detail    the secondary line: command and aliases, shortcut, filter text…
 * @property {string[]} keywords other texts a query may match (command name, aliases…)
 * @property {string[]} exact   texts that, typed whole, put this entry first
 * @property {object} target    what runPaletteItem acts on
 */

/**
 * Commands that only make sense with an argument typed after them: choosing
 * them opens the command line with the command already written, cursor after
 * the space, rather than running them bare.
 */
export const ARGUMENT_COMMANDS = new Set(['s', 'ss']);

/** The i18n key of a command's palette description. */
export function commandLabelKey(name) {
    return `palette.cmd.${name}`;
}

/**
 * Every entry the palette can offer, in its resting order: pinned filters,
 * tabs, commands, the other filters, then the matches, newest first.
 *
 * @param {{ translate: (key: string, params?: object) => string, matches?: any[], filters?: Array<{ id: number, name: string, command: string, pinned?: boolean }> }} sources
 * @returns {PaletteItem[]}
 */
export function buildPaletteItems({ translate, matches = [], filters = [] }) {
    /** @type {PaletteItem[]} */
    const pinned = [];
    /** @type {PaletteItem[]} */
    const otherFilters = [];
    for (const f of filters || []) {
        const item = {
            kind: /** @type {PaletteKind} */ ('filter'),
            id: `filter:${f.id}`,
            label: f.name,
            detail: f.command,
            keywords: [f.command],
            exact: [],
            target: f
        };
        (f.pinned ? pinned : otherFilters).push(item);
    }

    const tabs = TABS.map((tab) => ({
        kind: /** @type {PaletteKind} */ ('tab'),
        id: `tab:${tab.id}`,
        label: translate(tab.labelKey),
        detail: tab.shortcut,
        keywords: [tab.id],
        exact: [],
        target: tab.id
    }));

    const commands = COMMANDS.map((cmd) => {
        const forms = [cmd.name, ...cmd.aliases];
        return {
            kind: /** @type {PaletteKind} */ ('command'),
            id: `command:${cmd.name}`,
            label: translate(commandLabelKey(cmd.name)),
            detail: forms.join(' · '),
            keywords: forms,
            exact: forms,
            target: cmd.name
        };
    });

    const byDate = [...(matches || [])].sort((a, b) => String(b.match_date || '').localeCompare(String(a.match_date || '')));
    const matchItems = byDate.map((m) => {
        const tournament = m.tournament_name || m.event || '';
        const date = m.match_date ? formatDate(m.match_date) : '';
        const parts = [translate('palette.matchLength', { n: m.match_length ?? 0 }), date, tournament].filter((p) => p && p !== '-');
        return {
            kind: /** @type {PaletteKind} */ ('match'),
            id: `match:${m.id}`,
            label: `${m.player1_name || '?'} – ${m.player2_name || '?'}`,
            detail: parts.join(' · '),
            keywords: [tournament, date].filter(Boolean),
            exact: [],
            target: m.id
        };
    });

    return [...pinned, ...tabs, ...commands, ...otherFilters, ...matchItems];
}

/**
 * @typedef {{ item: PaletteItem, score: number, labelIndices: number[] }} RankedItem
 */

/**
 * Rank the entries against a query. An empty query keeps the resting order.
 * A query typed whole as a command or one of its aliases puts that command
 * first — "st" is `stats`, not the best scattered match of s and t.
 *
 * @param {PaletteItem[]} items
 * @param {string} query
 * @param {number} [limit]
 * @returns {RankedItem[]}
 */
export function rankPaletteItems(items, query, limit = 60) {
    const q = (query || '').trim();
    if (q === '') return items.slice(0, limit).map((item) => ({ item, score: 0, labelIndices: [] }));
    const folded = fold(q);

    /** @type {Array<RankedItem & { order: number }>} */
    const ranked = [];
    items.forEach((item, order) => {
        const onLabel = fuzzyMatch(q, item.label);
        let score = onLabel ? onLabel.score : -Infinity;
        for (const keyword of item.keywords) {
            const m = fuzzyMatch(q, keyword);
            if (m && m.score * 0.9 > score) score = m.score * 0.9;
        }
        if (item.exact.some((form) => fold(form) === folded)) score += 1000;
        if (score === -Infinity) return;
        ranked.push({ item, score, labelIndices: onLabel ? onLabel.indices : [], order });
    });
    ranked.sort((a, b) => b.score - a.score || a.order - b.order);
    return ranked.slice(0, limit).map(({ item, score, labelIndices }) => ({ item, score, labelIndices }));
}

/**
 * Refresh what the palette reads from the database: the filter library (into
 * filterLibraryStore) and the matches, returned. Nothing without a database.
 *
 * @returns {Promise<any[]>} the matches
 */
export async function loadPaletteSources() {
    if (!get(databaseLoadedStore)) return [];
    const [, matches] = await Promise.all([
        loadFilterLibrary(),
        GetAllMatches().catch((error) => {
            logger.error('Command palette: loading the matches failed:', error);
            return [];
        })
    ]);
    return matches || [];
}

export function openCommandPalette() {
    commandPaletteOpenStore.set(true);
}

export function closeCommandPalette() {
    commandPaletteOpenStore.set(false);
}

/** Ctrl+Maj+P: open the palette, or close it when it is already open. */
export function toggleCommandPalette() {
    commandPaletteOpenStore.update((open) => !open);
}

/**
 * Do what the chosen entry names. The palette is closed first by its caller;
 * nothing here depends on it.
 *
 * @param {PaletteItem} item
 */
export async function runPaletteItem(item) {
    switch (item.kind) {
        case 'command': {
            const name = /** @type {string} */ (item.target);
            if (ARGUMENT_COMMANDS.has(name)) {
                commandTextStore.set(`${name} `);
                showCommandInputStore.set(true);
            } else {
                processCommand(name);
            }
            return;
        }
        case 'tab': {
            const id = /** @type {string} */ (item.target);
            // Eval is a mode as much as a tab: entering it goes through the
            // mode machine, like Ctrl+E — but never out of it from here.
            if (id === 'eval') {
                if (get(activeTabStore) !== 'eval') toggleEvalMode();
            } else {
                showTab(id);
            }
            return;
        }
        case 'filter':
            await runSavedFilter(/** @type {any} */ (item.target));
            return;
        case 'match':
            // The match panel opens it once its list is loaded, the way a
            // double-click on its row does (MatchPanel.svelte).
            matchOpenRequestStore.set(/** @type {number} */ (item.target));
            activeTabStore.set('matches');
            return;
    }
}
