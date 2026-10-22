/**
 * actionCatalog.js — what a context menu offers on an object, declared once per kind of object
 * (position, match, tournament, collection). The board and the rows of the lists build their menu
 * here, so the same action carries the same label, group and shortcut wherever it is offered.
 *
 * The catalog decides nothing about how an action is done: the caller hands its handlers in
 * `ctx.do`, keyed by action id, and an action without a handler is not offered. An action also
 * states when it applies (`when`, read on the context); only the applicable ones are shown, and a
 * group left empty disappears.
 *
 * The frequent actions sit at the first level (`top`); the others go into a submenu per group.
 */

import { chordLabel } from '../utils/keyNames.js';

/**
 * @typedef {{ label: string, onClick?: () => void, shortcut?: string, disabled?: boolean, separatorBefore?: boolean, submenu?: MenuItem[] }} MenuItem
 *
 * @typedef {object} ActionContext
 * @property {string} [mode]                         statusBarModeStore's value
 * @property {Record<string, (arg?: any) => any>} do   the handlers, by action id
 * @property {number} [positionId]                   the stored position (0: a board not stored)
 * @property {boolean} [dbOpen]
 * @property {boolean} [studiedMark]                the position carries the studied mark (undefined: not read yet)
 * @property {boolean} [challengeOn]                 the analysis challenge is on
 * @property {{ id: number, name: string }[]} [ankiDecks]
 * @property {{ id: number, name: string }[]} [otherBases]  the open bases other than the one on screen
 * @property {{ id: number, name: string }[]} [collections] the collections of the base on screen
 *
 * @typedef {object} Action
 * @property {string} id
 * @property {string} label                          i18n key
 * @property {string | null} group                   a GROUPS key, null for the first level only
 * @property {boolean} [top]                         also at the first level
 * @property {string} [shortcut]                     the chord (shortcutMap.js notation), named beside the label
 * @property {(ctx: ActionContext) => boolean} [when]
 * @property {(ctx: ActionContext) => { label: string, args?: Record<string, any>, arg: any }[]} [each]
 *           one entry per element (an Anki deck): its handler receives `arg`
 */

/** The submenus, in their order; `send` copies the object to another open base. */
export const GROUPS = ['copy', 'annotate', 'navigate', 'analyze', 'train', 'manage', 'send'];

/** @type {Record<string, string>} */
export const GROUP_LABELS = {
    copy: 'actions.group.copy',
    annotate: 'actions.group.annotate',
    navigate: 'actions.group.navigate',
    analyze: 'actions.group.analyze',
    train: 'actions.group.train',
    manage: 'actions.group.manage',
    send: 'actions.group.send'
};

// A position studied from the library, as opposed to the Search and Eval scratch boards.
const STUDIED = ['NORMAL', 'MATCH', 'COLLECTION'];
const SCRATCH = ['EDIT', 'EVAL'];

/** @param {ActionContext} c */
const studied = (c) => STUDIED.includes(c.mode ?? '') && !!c.positionId && !!c.dbOpen;
/** @param {ActionContext} c */
const notSearch = (c) => c.mode !== 'EDIT';
/** @param {ActionContext} c */
const stored = (c) => !!c.positionId && !SCRATCH.includes(c.mode ?? '');

/**
 * The send group of a kind of object: one copy per other open base, and the lot.
 * @param {string} copyLabel
 * @param {(ctx: ActionContext) => boolean} [when]
 * @returns {Action[]}
 */
function send(copyLabel, when = () => true) {
    return [
        { id: 'copyTo', label: copyLabel, group: 'send', when, each: (c) => (c.otherBases ?? []).map((b) => ({ label: copyLabel, args: { base: b.name }, arg: b.id })) },
        { id: 'addToLot', label: 'actions.addToLot', group: 'send', when }
    ];
}

/** @type {Record<string, Action[]>} */
export const CATALOG = {
    position: [
        // At the first level and again in their group, where the rest of their family is.
        { id: 'evaluate', label: 'board.menu.evaluate', group: 'analyze', top: true, when: notSearch },
        { id: 'copyImage', label: 'actions.copyImage', group: 'copy', top: true, shortcut: 'Ctrl+X' },

        { id: 'copyXGID', label: 'actions.copyXGID', group: 'copy' },
        { id: 'copyPositionText', label: 'actions.copyPositionText', group: 'copy', shortcut: 'Ctrl+C' },
        { id: 'copyImageWithAnalysis', label: 'board.menu.copyImageWithAnalysis', group: 'copy', when: notSearch },
        { id: 'saveImageSVG', label: 'board.menu.saveImageSVG', group: 'copy' },
        { id: 'saveImagePNG', label: 'board.menu.saveImagePNG', group: 'copy' },

        { id: 'comment', label: 'actions.comment', group: 'annotate', when: studied },
        { id: 'addTag', label: 'actions.addTag', group: 'annotate', when: studied },
        { id: 'markStudied', label: 'actions.markStudied', group: 'annotate', when: (c) => studied(c) && c.studiedMark !== true },
        { id: 'unmarkStudied', label: 'actions.unmarkStudied', group: 'annotate', when: (c) => studied(c) && c.studiedMark !== false },

        { id: 'neighbours', label: 'board.menu.neighbours', group: 'navigate', when: stored },
        { id: 'sameStructure', label: 'actions.sameStructure', group: 'navigate', when: (c) => c.mode === 'NORMAL' && !!c.dbOpen },
        { id: 'newView', label: 'board.menu.newView', group: 'navigate', when: (c) => !SCRATCH.includes(c.mode ?? '') },

        { id: 'evaluateMirror', label: 'board.menu.evaluateMirror', group: 'analyze', when: notSearch },
        { id: 'evaluateInNewView', label: 'board.menu.evaluateInNewView', group: 'analyze', when: (c) => STUDIED.includes(c.mode ?? '') },
        { id: 'rolloutFast', label: 'actions.rolloutFast', group: 'analyze', when: notSearch },
        { id: 'rolloutStandard', label: 'actions.rolloutStandard', group: 'analyze', when: notSearch },

        {
            id: 'addToAnkiDeck',
            label: 'board.menu.addToAnkiDeck',
            group: 'train',
            when: stored,
            each: (c) => (c.ankiDecks ?? []).map((deck) => ({ label: 'board.menu.addToAnkiDeck', args: { deck: deck.name }, arg: deck }))
        },
        { id: 'challenge', label: 'actions.challenge', group: 'train', shortcut: 'm', when: (c) => STUDIED.includes(c.mode ?? '') && !c.challengeOn },

        { id: 'save', label: 'actions.save', group: 'manage', shortcut: 'Ctrl+S', when: (c) => SCRATCH.includes(c.mode ?? '') && !!c.dbOpen },
        { id: 'trash', label: 'actions.trash', group: 'manage', shortcut: 'Delete', when: studied },
        ...send('actions.copyPositionTo', stored),
        {
            id: 'addToCollection',
            label: 'actions.addToCollection',
            group: 'send',
            when: stored,
            each: (c) => (c.collections ?? []).map((k) => ({ label: 'actions.addToCollection', args: { name: k.name }, arg: k }))
        }
    ],
    match: [
        { id: 'open', label: 'actions.openMatch', group: null, top: true },
        { id: 'exportMat', label: 'match.exportMat', group: 'copy' },
        { id: 'editTranscription', label: 'match.editTranscription', group: 'manage' },
        { id: 'swapPlayers', label: 'match.swapPlayers', group: 'manage' },
        { id: 'edit', label: 'common.edit', group: 'manage' },
        { id: 'trash', label: 'actions.trash', group: 'manage' },
        ...send('actions.copyMatchTo')
    ],
    tournament: [
        { id: 'open', label: 'actions.openTournament', group: null, top: true },
        { id: 'edit', label: 'common.edit', group: 'manage' },
        // A tournament has no trash: it is deleted, after the panel's confirmation.
        { id: 'delete', label: 'common.delete', group: 'manage' },
        ...send('actions.copyTournamentTo')
    ],
    collection: [
        { id: 'open', label: 'actions.openCollection', group: null, top: true },
        { id: 'edit', label: 'common.edit', group: 'manage' },
        { id: 'trash', label: 'actions.trash', group: 'manage' }
    ]
};

/**
 * The menu entries of one action in this context: none when it does not apply.
 *
 * @param {Action} action
 * @param {ActionContext} ctx
 * @param {(key: string, args?: Record<string, any>) => string} t
 * @returns {MenuItem[]}
 */
function entries(action, ctx, t) {
    const run = ctx.do?.[action.id];
    if (typeof run !== 'function' || (action.when && !action.when(ctx))) return [];
    if (action.each) return action.each(ctx).map((e) => ({ label: t(e.label, e.args), onClick: () => run(e.arg) }));
    return [{ label: t(action.label), onClick: () => run(), ...(action.shortcut ? { shortcut: chordLabel(action.shortcut, t) } : {}) }];
}

/**
 * The menu of an object of this kind: the first-level actions, then one submenu per group that
 * holds an applicable action.
 *
 * @param {keyof typeof CATALOG} kind
 * @param {ActionContext} ctx
 * @param {(key: string, args?: Record<string, any>) => string} t
 * @returns {MenuItem[]}
 */
export function menuFor(kind, ctx, t) {
    const actions = CATALOG[kind];
    if (!actions) throw new Error(`actionCatalog: unknown kind "${kind}"`);
    /** @type {MenuItem[]} */
    const items = [];
    const seen = new Set();
    for (const action of actions) {
        if ((action.top || action.group === null) && !seen.has(action.id)) {
            seen.add(action.id);
            items.push(...entries(action, ctx, t));
        }
    }
    let first = true;
    for (const group of GROUPS) {
        const submenu = actions.filter((a) => a.group === group).flatMap((a) => entries(a, ctx, t));
        if (submenu.length === 0) continue;
        items.push({ label: t(GROUP_LABELS[group]), submenu, ...(first && items.length > 0 ? { separatorBefore: true } : {}) });
        first = false;
    }
    return items;
}
