// Canonical list of command-mode commands and their aliases, used to power
// command-line autocompletion (see CommandLine.svelte). Keep this in sync with
// the command branches in commandProcessor.js / CommandLine.svelte —
// commandVocabulary.sync.test.js enforces it, since processCommand's if/else
// chain has no trailing else and an unhandled command is a silent no-op.
//
// `name` is the canonical form inserted on completion; `aliases` are the
// accepted shorthands. Search filter tokens (p, w, ma, id, …) are intentionally
// excluded — they only apply after the `s ` search prefix and are documented in
// the manual.

export const COMMANDS = [
    { name: 'new', aliases: ['ne', 'n'] },
    { name: 'open', aliases: ['op', 'o'] },
    { name: 'import', aliases: ['i'] },
    { name: 'import_db', aliases: ['idb'] },
    { name: 'export_db', aliases: ['edb'] },
    { name: 'write', aliases: ['wr', 'w'] },
    { name: 'write!', aliases: ['wr!', 'w!'] },
    { name: 'delete', aliases: ['del', 'd'] },
    { name: 'list', aliases: ['l'] },
    { name: 'comment', aliases: ['co'] },
    { name: 'quit', aliases: ['q'] },
    { name: 'help', aliases: ['he', 'h'] },
    { name: 'tutorial', aliases: ['tour'] },
    { name: 'demo', aliases: [] },
    { name: 'e', aliases: [] },
    { name: 's', aliases: [] },
    { name: 'ss', aliases: [] },
    { name: 'stats', aliases: ['st'] },
    { name: 'blunders', aliases: ['bl'] },
    { name: 'history', aliases: ['hi'] },
    { name: 'match', aliases: ['ma'] },
    { name: 'collection', aliases: ['coll'] },
    { name: 'epc', aliases: [] },
    { name: 'm', aliases: [] },
    { name: 'met', aliases: [] },
    { name: 'meta', aliases: [] },
    { name: 'tp2', aliases: [] },
    { name: 'tp2_last', aliases: [] },
    { name: 'tp2_live', aliases: [] },
    { name: 'tp4', aliases: [] },
    { name: 'tp4_last', aliases: [] },
    { name: 'tp4_live', aliases: [] },
    { name: 'gv1', aliases: [] },
    { name: 'gv2', aliases: [] },
    { name: 'gv4', aliases: [] },
    { name: 'clear', aliases: ['cl'] }
];

/**
 * Returns the command entries whose canonical name or one of its aliases starts
 * with the typed text. Suggestions are only offered for the command word — i.e.
 * the first token, before any space — and never for position-number navigation
 * (`12`) or tag insertion (`#blunder`).
 *
 * @param {string} text the current command-line input
 * @returns {Array<{name: string, aliases: string[]}>}
 */
export function getCommandSuggestions(text) {
    if (typeof text !== 'string') return [];
    // Past the command word (a space was typed) → no command suggestions.
    if (/\s/.test(text)) return [];
    const token = text.trim();
    if (token === '') return [];
    // Position-number navigation or tag insertion are not commands.
    if (/^[#\d]/.test(token)) return [];
    const lower = token.toLowerCase();
    return COMMANDS.filter((cmd) => [cmd.name, ...cmd.aliases].some((form) => form.toLowerCase().startsWith(lower)));
}
