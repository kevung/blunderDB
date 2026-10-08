/**
 * shortcutMap.js — every keyboard scope and the keys it binds, declared once.
 *
 * keyDispatch.registerKeys() refuses an undeclared scope, so a handler cannot exist without
 * its line here, and shortcutMap.test.js reads this table to find duplicates: two scopes that
 * can be active together and bind the same chord at the same tier, or a scope that takes a
 * chord from a lower tier without saying so in `shadows`.
 *
 * Chords: modifiers first, in the order Ctrl, Alt, Shift, then the key — a lowercase letter
 * for a letter (Shift+J for the shifted one), a digit for the digit row, otherwise the
 * KeyboardEvent key name (Home, PageDown, Escape, Space, ?). Ctrl stands for Cmd too.
 *
 * `group`: scopes of one group are never active together (one docked tab at a time, the
 * Direction page hiding the board), so their chords cannot collide.
 * `shadows`: chords this scope deliberately takes over from a scope of a later tier.
 */

/** Dispatch order; a lower tier sees the key first. */
export const TIER = Object.freeze({
    OVERLAY: 0,
    CAPTURE: 1,
    MENU: 2,
    PANEL: 3,
    LOCAL: 4,
    GLOBAL: 5
});

/** The tiers dispatched in the capture phase, before any element handler. */
export const CAPTURE_TIERS = new Set([TIER.OVERLAY, TIER.CAPTURE]);

/** @param {string} prefix @param {number} from @param {number} to */
const digits = (prefix, from, to) => Array.from({ length: to - from + 1 }, (_, i) => `${prefix}${from + i}`);

/**
 * @typedef {{ tier: number, keys: string[], group?: string, shadows?: string[] }} ScopeDecl
 * @type {Record<string, ScopeDecl>}
 */
export const SHORTCUTS = {
    // escapeService.closeOnEscape: the last opened overlay closes first.
    overlay: { tier: TIER.OVERLAY, keys: ['Escape'], shadows: ['Escape'] },

    // Direction page (components/direction/): the proposal queue and the table-number digits.
    directionQueue: {
        tier: TIER.CAPTURE,
        group: 'direction',
        keys: ['j', 'k', 'ArrowDown', 'ArrowUp', 'Enter'],
        shadows: ['j', 'k', 'ArrowDown', 'ArrowUp', 'Enter']
    },
    directionGrid: { tier: TIER.CAPTURE, group: 'direction', keys: digits('', 0, 9), shadows: digits('', 1, 6) },

    // An open context menu owns its arrows and traps Tab; it runs before the docked panels, which
    // therefore never see a key the menu claims.
    contextMenu: {
        tier: TIER.MENU,
        keys: ['ArrowDown', 'ArrowUp', 'Home', 'End', 'Tab', 'Shift+Tab'],
        shadows: ['ArrowDown', 'ArrowUp', 'Home', 'End', 'Tab']
    },

    // Docked panels, each mounted while its tab is the active one. Each swallows the keys it does
    // not leave to the global handler (keyboardService.panelKeyGuard).
    search: { tier: TIER.PANEL, group: 'tab', keys: ['Enter', 'Tab'], shadows: ['Tab'] },
    matchPanel: {
        tier: TIER.PANEL,
        group: 'tab',
        // [ and ] set the speed of the open video.
        keys: ['Escape', '/', 'j', 'k', 'v', '[', ']', 'ArrowDown', 'ArrowUp', 'Enter', 'Delete'],
        shadows: ['Escape', '/', 'j', 'k', 'Delete']
    },
    collectionPanel: { tier: TIER.PANEL, group: 'tab', keys: ['Escape', 'Delete'], shadows: ['Escape', 'Delete'] },
    tournamentPanel: {
        tier: TIER.PANEL,
        group: 'tab',
        keys: ['Escape', 'j', 'k', 'ArrowDown', 'ArrowUp'],
        shadows: ['Escape', 'j', 'k']
    },
    transcription: {
        tier: TIER.PANEL,
        group: 'tab',
        keys: [
            ...digits('', 1, 6),
            'h',
            'l',
            'ArrowLeft',
            'ArrowRight',
            'j',
            'k',
            'ArrowDown',
            'ArrowUp',
            'a',
            'd',
            'i',
            'n',
            'p',
            'r',
            's',
            't',
            'x',
            'Backspace',
            'Delete',
            'Enter',
            'Ctrl+Enter',
            'Escape',
            // The video's keys, bound only while a source is attached; Space stays global
            // (the command line) without one.
            'Space',
            'Shift+ArrowLeft',
            'Shift+ArrowRight',
            'Ctrl+Shift+ArrowLeft',
            'Ctrl+Shift+ArrowRight',
            'v',
            'Shift+V',
            '[',
            ']'
        ],
        shadows: [...digits('', 1, 4), 'h', 'l', 'ArrowLeft', 'ArrowRight', 'j', 'k', 'p', 'r', 'Backspace', 'Delete', 'Escape', 'Space']
    },

    // Always mounted beside the board.
    boardOrientation: { tier: TIER.LOCAL, keys: ['Ctrl+ArrowLeft', 'Ctrl+ArrowRight'] },
    boardEdit: { tier: TIER.LOCAL, keys: ['Backspace'] },
    viewTabs: { tier: TIER.LOCAL, keys: digits('Ctrl+', 1, 9) },
    directionUndo: { tier: TIER.LOCAL, group: 'direction', keys: ['Ctrl+z'], shadows: ['Ctrl+z'] },

    // The Anki review keys (1-4, Space, Escape, p) are handled inside the global dispatcher
    // (keyboardService.handleKeyDown): no scope of their own, 1-4 are bound under `global`.

    // keyboardService.handleKeyDown.
    global: {
        tier: TIER.GLOBAL,
        keys: [
            'Escape',
            ...digits('', 1, 4),
            ...digits('Alt+', 1, 9),
            'Ctrl+n',
            'Ctrl+o',
            'Ctrl+q',
            'Ctrl+Shift+I',
            'Ctrl+Shift+F',
            'Ctrl+i',
            'Ctrl+c',
            'Ctrl+x',
            'Ctrl+v',
            'Ctrl+Shift+S',
            'Ctrl+s',
            'Ctrl+u',
            'Delete',
            'Home',
            'End',
            'PageUp',
            'PageDown',
            'h',
            'l',
            'ArrowLeft',
            'ArrowRight',
            'j',
            'k',
            'Ctrl+b',
            'Ctrl+r',
            'Ctrl+Tab',
            'Tab',
            'Space',
            'Ctrl+Shift+L',
            'Ctrl+l',
            'Ctrl+Shift+P',
            'Ctrl+p',
            'Ctrl+f',
            '?',
            'Ctrl+m',
            'Ctrl+j',
            'Ctrl+k',
            'Ctrl+Shift+Z',
            'Ctrl+z',
            'Ctrl+Shift+T',
            'Ctrl+t',
            'Ctrl+w',
            'Ctrl+y',
            'Ctrl+d',
            'Ctrl+h',
            'Ctrl+e',
            'Ctrl+g',
            'Ctrl+PageUp',
            'Ctrl+PageDown',
            'Shift+J',
            'Shift+K',
            'b',
            'p',
            'r',
            'F11',
            '/'
        ]
    }
};

/**
 * The chord(s) each toolbar button announces in its tooltip, in the chord notation above. The
 * tooltip names the keys through `utils/keyNames.js`, so a key is spelled once per language.
 * @type {Record<string, string[]>}
 */
export const TOOLBAR_CHORDS = {
    newDatabase: ['Ctrl+N'],
    openDatabase: ['Ctrl+O'],
    exportDatabase: ['Ctrl+Shift+S'],
    importPositionTip: ['Ctrl+I'],
    importFolder: ['Ctrl+Shift+F'],
    savePosition: ['Ctrl+S'],
    updatePosition: ['Ctrl+U'],
    deletePosition: ['Delete'],
    togglePile: ['b'],
    loadAllPositions: ['Ctrl+R'],
    firstPosition: ['PageUp', 'h'],
    previousPosition: ['ArrowLeft', 'k'],
    nextPosition: ['ArrowRight', 'j'],
    lastPosition: ['PageDown', 'l'],
    togglePipcount: ['p'],
    randomPosition: ['r'],
    training: ['Ctrl+J'],
    duel: ['Ctrl+H'],
    help: ['?'],
    copyPosition: ['Ctrl+C'],
    pastePosition: ['Ctrl+V'],
    copyBoardImage: ['Ctrl+X'],
    exit: ['Ctrl+Q'],
    importDatabaseTip: ['Ctrl+Shift+I']
};
