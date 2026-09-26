// tabCatalog.js — the tabs of the tabbed panel, stated once for TabbedPanel
// and the command palette. `labelKey` is an i18n key, `shortcut` the key shown
// in the tooltip.

/** @typedef {{ id: string, labelKey: string, icon: string, shortcut: string }} TabEntry */

/** @type {ReadonlyArray<TabEntry>} */
export const TABS = Object.freeze([
    { id: 'matches', labelKey: 'tabbedPanel.matches', icon: 'matches', shortcut: 'Ctrl+Tab' },
    { id: 'tournaments', labelKey: 'tabbedPanel.tournaments', icon: 'tournaments', shortcut: 'Ctrl+Y' },
    { id: 'collections', labelKey: 'tabbedPanel.collections', icon: 'collections', shortcut: 'Ctrl+B' },
    { id: 'search', labelKey: 'tabbedPanel.search', icon: 'search', shortcut: 'Ctrl+F' },
    { id: 'analysis', labelKey: 'tabbedPanel.analysis', icon: 'analysis', shortcut: 'Ctrl+L' },
    { id: 'comments', labelKey: 'tabbedPanel.comments', icon: 'comments', shortcut: 'Ctrl+P' },
    { id: 'eval', labelKey: 'tabbedPanel.eval', icon: 'eval', shortcut: 'Ctrl+E' },
    // Entraînement entre Eval et Anki : l'ordre est celui de « calculer /
    // retenir » (ADR-0040 règle 1). Anki fait réviser ce qui se RETIENT,
    // l'Entraînement fait travailler ce qui se CALCULE.
    { id: 'training', labelKey: 'tabbedPanel.training', icon: 'training', shortcut: 'Ctrl+J' },
    { id: 'anki', labelKey: 'tabbedPanel.anki', icon: 'anki', shortcut: 'Ctrl+K' },
    { id: 'stats', labelKey: 'tabbedPanel.stats', icon: 'stats', shortcut: 'Ctrl+D' },
    { id: 'transcription', labelKey: 'tabbedPanel.transcription', icon: 'transcription', shortcut: 'Ctrl+Maj+T' },
    { id: 'metadata', labelKey: 'tabbedPanel.metadata', icon: 'metadata', shortcut: 'Ctrl+M' }
]);
