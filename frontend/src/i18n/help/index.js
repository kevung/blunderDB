import { language } from '../index.js';
import { derived } from 'svelte/store';

import en from './en.js';
import fi from './fi.js';
import el from './el.js';
import ja from './ja.js';
import de from './de.js';
import es from './es.js';
import fr from './fr.js';
import it from './it.js';
import ru from './ru.js';

const maps = { en, fi, el, ja, de, es, fr, it, ru };

const FALLBACK = en;

// Reactive help content for the active language. Falls back to English per-tab,
// so a partially-translated locale still renders English for any missing tab.
export const help = derived(language, ($lang) => {
    const m = maps[$lang] || {};
    return {
        manual: m.manual ?? FALLBACK.manual,
        shortcuts: m.shortcuts ?? FALLBACK.shortcuts,
        commands: m.commands ?? FALLBACK.commands,
        about: m.about ?? FALLBACK.about
    };
});
