// Canonical command-mode commands and aliases for command-line autocompletion. Kept in sync with
// commandProcessor.js / CommandLine.svelte by commandVocabulary.sync.test.js: processCommand has
// no trailing else, so an unhandled command is a silent no-op. `name` is inserted on completion.
// Search filter tokens (p, w, ma, …) are excluded: they only follow the `s ` prefix.

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
    { name: 'trash', aliases: [] },
    { name: 'demo', aliases: [] },
    { name: 'e', aliases: [] },
    { name: 's', aliases: [] },
    { name: 'ss', aliases: [] },
    { name: 'stats', aliases: ['st'] },
    { name: 'blunders', aliases: ['bl'] },
    { name: 'history', aliases: ['hi'] },
    { name: 'match', aliases: ['ma'] },
    { name: 'collection', aliases: ['coll'] },
    { name: 'eval', aliases: ['epc'] },
    { name: 'transcribe', aliases: ['tr'] },
    { name: 'direct', aliases: [] },
    { name: 'm', aliases: [] },
    { name: 'met', aliases: [] },
    { name: 'cm', aliases: [] },
    { name: 'tags', aliases: [] },
    { name: 'train', aliases: [] },
    { name: 'log', aliases: [] },
    { name: 'grid', aliases: ['gr'] },
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
 * The entries whose name or an alias starts with the typed text. Only the first token is
 * completed, never position numbers (`12`) or tags (`#blunder`).
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
