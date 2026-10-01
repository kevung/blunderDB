// commandPalette.js — what the command palette offers, how it ranks it, and
// what choosing an entry does. It owns no list; it reads:
//   - the command-line commands (commandVocabulary.js, locked to
//     commandProcessor.js); an entry runs through processCommand as if typed;
//   - the tabs (tabCatalog.js);
//   - the saved filters (filterLibraryStore);
//   - the matches (GetAllMatches);
//   - the open Direction's room (RencontreSearchIndex / DirectionSearchIndex): its
//     players, tables, running matches and events, only while a Direction is open.
// Its own data is each command's description, palette.cmd.<name> in the nine
// locales — a test requires one per command.

import { get } from 'svelte/store';
import { COMMANDS } from '../commandVocabulary.js';
import { TABS } from './tabCatalog.js';
import { fuzzyMatch, fold } from '../utils/fuzzy.js';
import { formatDate } from '../utils/matchTable.js';
import { GetAllMatches, RencontreSearchIndex, DirectionSearchIndex } from '../../wailsjs/go/database/Database.js';
import { activeTabStore, commandTextStore, showCommandInputStore, commandPaletteOpenStore, commandPaletteScopeStore, matchOpenRequestStore } from '../stores/uiStore.js';
import { directionStore, openDirectionIdStore, requestDirectionJump } from '../stores/directionStore.js';
import { databaseLoadedStore } from '../stores/databaseStore.js';
import { loadFilterLibrary, runSavedFilter } from './filterLibraryService.js';
import { showTab } from './tabToggles.js';
import { toggleEvalMode } from './modeMachine.js';
import { processCommand } from '../commandProcessor.js';
import { logger } from '../utils/logger.js';

/** @typedef {'command' | 'tab' | 'filter' | 'match' | 'direction'} PaletteKind */

/**
 * @typedef {object} PaletteItem
 * @property {PaletteKind} kind
 * @property {string} id        unique across the palette ("command:new", "match:12")
 * @property {string} label     what the entry is called, in the interface language
 * @property {string} detail    the secondary line: command and aliases, shortcut, filter text…
 * @property {string[]} keywords other texts a query may match (command name, aliases…)
 * @property {string[]} exact   texts that, typed whole, put this entry first
 * @property {string | number | object} target    what runPaletteItem acts on
 * @property {number} [boost]   added to the score of a match: a player at a table before a free one
 * @property {string} [badge]   the kind's label when the entry says it itself (a player, a table…)
 */

/**
 * Commands needing an argument: choosing one opens the command line with it
 * written, cursor after the space.
 */
export const ARGUMENT_COMMANDS = new Set(['s', 'ss']);

/**
 * The i18n key of a command's palette description.
 * @param {string} name
 */
export function commandLabelKey(name) {
    return `palette.cmd.${name}`;
}

/**
 * Every entry the palette can offer, in its resting order: pinned filters,
 * tabs, commands, the other filters, then the matches, newest first.
 *
 * With `scope: 'direction'` the room's entries are all there is.
 *
 * @param {{ translate: (key: string, params?: Record<string, unknown> | null) => string, matches?: any[], filters?: Array<{ id: number, name: string, command: string, pinned?: boolean }>, direction?: any[], scope?: 'direction' | null }} sources
 * @returns {PaletteItem[]}
 */
export function buildPaletteItems({ translate, matches = [], filters = [], direction = [], scope = null }) {
    const room = directionItems(translate, direction || []);
    if (scope === 'direction') return room;

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

    return [...pinned, ...tabs, ...commands, ...otherFilters, ...room, ...matchItems];
}

/** Worth a few characters of length in the fuzzy score, never a better match. */
const PLAYING_BOOST = 3;

/**
 * The room's entries as the palette lists them. A table answers to its number typed whole ("4",
 * "t4"); a player at a table ranks a little above a free one of the same name.
 *
 * @param {(key: string, params?: Record<string, unknown> | null) => string} translate
 * @param {Array<{ kind: string, tournamentId: number, epreuve?: string, name: string, playerId?: string, club?: string, opponent?: string, state?: string, table?: number, matchId?: string }>} entries
 * @returns {PaletteItem[]}
 */
function directionItems(translate, entries) {
    /** @type {PaletteItem[]} */
    const out = [];
    for (const e of entries) {
        const tid = e.tournamentId;
        const tableText = e.table ? translate('direction.proposals.table', { n: e.table }) : '';
        const base = { kind: /** @type {PaletteKind} */ ('direction'), keywords: /** @type {string[]} */ ([]), exact: /** @type {string[]} */ ([]) };
        if (e.kind === 'epreuve') {
            out.push({ ...base, id: `direction:epreuve:${tid}`, badge: translate('palette.kindEpreuve'), label: e.name, detail: '', target: { kind: 'epreuve', tournamentId: tid } });
        } else if (e.kind === 'player') {
            const playing = e.state === 'playing' && e.table;
            const detail = [translate(`direction.players.states.${e.state || 'free'}`), playing ? tableText : '', e.opponent ? translate('palette.directionVs', { name: e.opponent }) : '', e.epreuve]
                .filter(Boolean)
                .join(' · ');
            out.push({
                ...base,
                id: `direction:player:${tid}:${e.playerId}`,
                badge: translate('palette.kindPlayer'),
                label: e.name,
                detail,
                boost: playing ? PLAYING_BOOST : 0,
                keywords: [e.club || '', e.epreuve || ''].filter(Boolean),
                target: playing ? { kind: 'table', tournamentId: tid, table: e.table, open: true } : { kind: 'player', tournamentId: tid, name: e.name }
            });
        } else if (e.kind === 'table') {
            const occupied = e.tournamentId > 0 && e.name;
            const detail = occupied ? [e.name, e.epreuve].filter(Boolean).join(' · ') : translate(e.state === 'unavailable' ? 'direction.table.unavailable' : 'direction.table.free');
            out.push({
                ...base,
                id: `direction:table:${e.table}`,
                badge: translate('palette.kindTable'),
                label: translate('palette.directionTable', { n: e.table }),
                detail,
                keywords: [`t${e.table}`, String(e.table), e.name],
                exact: [`t${e.table}`, String(e.table)],
                target: { kind: 'table', tournamentId: tid, table: e.table, open: !!occupied }
            });
        } else if (e.kind === 'match') {
            out.push({
                ...base,
                id: `direction:match:${tid}:${e.matchId}`,
                badge: translate('palette.kindRunning'),
                label: e.name,
                detail: [tableText, e.epreuve].filter(Boolean).join(' · '),
                target: { kind: 'table', tournamentId: tid, table: e.table, open: true }
            });
        }
    }
    return out;
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
        if (score !== -Infinity) score += item.boost || 0;
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

/**
 * What a Direction's room offers the palette: its Rencontre's index when it plays in one, its
 * own otherwise. Nothing without a Direction open.
 *
 * @returns {Promise<any[]>}
 */
export async function loadDirectionIndex() {
    const id = get(openDirectionIdStore);
    if (id === null) return [];
    const rencontreId = get(directionStore)?.rencontreId;
    try {
        return (rencontreId ? await RencontreSearchIndex(rencontreId) : await DirectionSearchIndex(id)) || [];
    } catch (error) {
        logger.error('Command palette: loading the room index failed:', error);
        return [];
    }
}

export function openCommandPalette() {
    commandPaletteScopeStore.set(null);
    commandPaletteOpenStore.set(true);
}

/** The `/` key of the Direction page: the palette on the room's entries alone. */
export function openDirectionSearch() {
    if (get(openDirectionIdStore) === null) return;
    commandPaletteScopeStore.set('direction');
    commandPaletteOpenStore.set(true);
}

export function closeCommandPalette() {
    commandPaletteOpenStore.set(false);
}

/** Ctrl+Maj+P: open the palette, or close it when it is already open. */
export function toggleCommandPalette() {
    if (!get(commandPaletteOpenStore)) commandPaletteScopeStore.set(null);
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
        case 'direction':
            // The request first: the page reads it when it mounts, or at once when it is shown.
            requestDirectionJump(/** @type {any} */ (item.target));
            activeTabStore.set('tournaments');
            return;
    }
}
