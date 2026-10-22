// Vite plugin: `locales/<lang>.json?merged` is that locale with the English keys it lacks filled
// in, so the app loads one dictionary instead of the locale plus English for the fallback.
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';

/** @param {any} base @param {any} own @returns {any} */
export function merge(base, own) {
    if (own === undefined) return base;
    if (base === undefined || base === null || typeof base !== 'object' || typeof own !== 'object' || own === null) return own;
    const out = { ...base };
    for (const [k, v] of Object.entries(own)) out[k] = merge(base[k], v);
    return out;
}

export function mergedLocales() {
    const suffix = '.json?merged';
    return {
        name: 'blunderdb-merged-locales',
        enforce: /** @type {'pre'} */ ('pre'),
        /**
         * @this {{ addWatchFile: (id: string) => void }}
         * @param {string} id
         */
        load(id) {
            if (!id.includes('/i18n/locales/') || !id.endsWith(suffix)) return null;
            const file = id.slice(0, -'?merged'.length);
            const en = resolve(dirname(file), 'en.json');
            this.addWatchFile(file);
            this.addWatchFile(en);
            const merged = merge(JSON.parse(readFileSync(en, 'utf8')), JSON.parse(readFileSync(file, 'utf8')));
            return `export default ${JSON.stringify(merged)};`;
        }
    };
}
